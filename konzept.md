# Alarm-Core: Konzept eines mandantenfähigen, kanalunabhängigen Alarmierungssystems

**Version:** 2.0
**Stand:** Juni 2026
**Hinweis:** Dieses Dokument ersetzt Version 1.0. In Version 2.0 wurden Wartungsfenster, Nachrichten-Templates, Wiederhol-Benachrichtigung, ein Alarm-Dashboard (NOC View), Heartbeat-Überwachung, Statistiken und eine vollständige OpenAPI-Dokumentation ergänzt.

---

## 1. Zielsetzung

Alarm-Core empfängt Events aus einem Eingangskanal, matched sie gegen ein admin-konfigurierbares Regelwerk (Zeitfenster, Wochentag, Priorität) und alarmiert die passende Zielgruppe über einen oder mehrere austauschbare Notification-Kanäle (Telefonanruf, SMS, E-Mail, Microsoft Teams). Wartungsfenster unterdrücken planbar Alarme ohne Regeländerung. Heartbeats überwachen externe Systeme nach dem Dead-Man's-Switch-Prinzip. Alle Vorgänge werden vollständig auditiert. Kein Stichwort/keine Regel ist im Code hartkodiert — alles läuft über PostgreSQL-Tabellen.

---

## 2. Architekturübersicht

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Eingangskanäle                              │
│   POST /api/events (HTTP-Webhook)  │  Heartbeat-Checker (intern)   │
└────────────────────┬────────────────────────────┬───────────────────┘
                     │ core.Event                  │ core.Event (HB-Ausfall)
                     ▼                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      alarm-core (Go, ein Binary)                    │
│                                                                     │
│  ┌─────────────┐  ┌───────────────────────────────────────────────┐ │
│  │ RuleEngine  │  │                  Processor                    │ │
│  │ (in-memory, │──│  1. Wartungsfenster-Check (MaintenanceChecker)│ │
│  │ Reload 60s) │  │  2. Rule-Match (RuleEngine)                   │ │
│  └─────────────┘  │  3. Template auflösen (TemplateLoader)        │ │
│                   │  4. dispatch() → Notifier                     │ │
│                   │  5. repeatNotify() Goroutine (opt.)           │ │
│                   └───────────────────────────────────────────────┘ │
│                                          │                          │
│  ┌─────────────┐  ┌──────────────┐       │ core.Notifier           │
│  │  AuditWriter│  │ HeartbeatChk │       ▼                         │
│  │ (event_log, │  │ Goroutine    │  ┌─────────────────────────────┐│
│  │ acks)       │  │ (Ticker 60s) │  │  asterisk  sms  email  teams││
│  └─────────────┘  └──────────────┘  └─────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │   PostgreSQL        │
                    │  (alle Tabellen)    │
                    └────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │  Vue 3 Frontend     │
                    │  (Tailwind, Pinia)  │
                    └────────────────────┘
```

**Kernprinzip:** Der Core kennt keine kanalspezifischen Details. `core.Notifier` ist das einzige Interface, über das Ausgangskanäle angesprochen werden. Ein Ausfall eines Kanals blockiert die anderen nicht.

---

## 3. Datenmodell (PostgreSQL)

Migrationen liegen unter `internal/db/migrations/`, getrackt über eine `schema_migrations`-Tabelle (idempotent, jede Datei einzeln).

### 3.1 `recipients`, `groups`, `group_members`

```sql
CREATE TABLE recipients (
    id          SERIAL PRIMARY KEY,
    first_name  VARCHAR(100) NOT NULL,
    last_name   VARCHAR(100) NOT NULL,
    voice       VARCHAR(30),
    mobile      VARCHAR(30),
    email       VARCHAR(255),
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE groups (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE group_members (
    group_id     INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    recipient_id INT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, recipient_id)
);
```

### 3.2 `rules` — das Regelwerk

```sql
CREATE TABLE rules (
    id                  SERIAL PRIMARY KEY,
    source              VARCHAR(50)  NOT NULL,
    keyword             VARCHAR(100) NOT NULL,
    start_hour          SMALLINT,
    end_hour            SMALLINT,
    weekdays_only       BOOLEAN NOT NULL DEFAULT FALSE,
    target_group_id     INT NOT NULL REFERENCES groups(id),
    channels            TEXT[] NOT NULL,
    priority            SMALLINT NOT NULL DEFAULT 10,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    require_ack         BOOLEAN NOT NULL DEFAULT FALSE,
    -- Wiederhol-Benachrichtigung (v2.0)
    repeat_interval_sec INT NOT NULL DEFAULT 0,  -- 0 = deaktiviert
    repeat_max          INT NOT NULL DEFAULT 3
);
```

### 3.3 `event_log` — Audit-Log

```sql
CREATE TABLE event_log (
    id            SERIAL PRIMARY KEY,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source        VARCHAR(50)  NOT NULL,
    keyword       VARCHAR(100) NOT NULL,
    raw_payload   JSONB,
    rule_id       INT REFERENCES rules(id),
    target_group  VARCHAR(100),
    channels      TEXT[],
    status        VARCHAR(20) NOT NULL DEFAULT 'DISPATCHED',
    -- Status: DISPATCHED | ERROR | NO_MATCH | MAINTENANCE
    error_detail  TEXT
);
```

### 3.4 `acknowledgements` — Quittierung

```sql
CREATE TABLE acknowledgements (
    id              SERIAL PRIMARY KEY,
    event_log_id    INT NOT NULL REFERENCES event_log(id) ON DELETE CASCADE,
    recipient_id    INT NOT NULL REFERENCES recipients(id),
    response        VARCHAR(12) NOT NULL,  -- 'COMING' | 'NOT_COMING' | 'LATER'
    channel         VARCHAR(10),
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_log_id, recipient_id)
);
```

### 3.5 `maintenance_windows` — Wartungsfenster (v2.0)

```sql
CREATE TABLE maintenance_windows (
    id         SERIAL PRIMARY KEY,
    label      VARCHAR(200) NOT NULL,
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ NOT NULL,
    source     VARCHAR(50),   -- NULL = alle Sources
    keyword    VARCHAR(100),  -- NULL = alle Keywords
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Während eines aktiven Wartungsfensters werden matchende Events mit Status `MAINTENANCE` ins Audit-Log geschrieben und **nicht** alarmiert. `source`/`keyword` NULL bedeutet "trifft alle" (globales Fenster).

### 3.6 `heartbeats` — Heartbeat-Überwachung (v2.0)

```sql
CREATE TABLE heartbeats (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    interval_sec INT NOT NULL DEFAULT 300,   -- erwartetes Ping-Intervall
    grace_sec    INT NOT NULL DEFAULT 60,    -- Toleranz nach Ablauf
    source       VARCHAR(50)  NOT NULL,      -- für die Alarm-Regel
    keyword      VARCHAR(100) NOT NULL,
    last_seen    TIMESTAMPTZ,
    alerted_at   TIMESTAMPTZ,               -- wann zuletzt alarmiert
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Externe Systeme pingen `POST /api/heartbeat/{name}`. Eine Goroutine prüft alle 60s, ob Heartbeats überfällig sind und schreibt ggf. einen `core.Event` in den Event-Channel (löst normale Regelverarbeitung aus).

### 3.7 `message_templates` — Nachrichten-Templates (v2.0)

```sql
CREATE TABLE message_templates (
    channel    VARCHAR(50) PRIMARY KEY,  -- 'voice', 'sms', 'email', 'teams'
    template   TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Unterstützte Platzhalter: `{text}`, `{source}`, `{keyword}`, `{time}`, `{group}`. Fehlt ein Template für einen Kanal, wird `event.Text` unverändert verwendet (Rückwärtskompatibilität).

---

## 4. Eingangskanal

```go
type Event struct {
    Source     string
    Keyword    string
    Text       string
    ReceivedAt time.Time
    RawPayload json.RawMessage
}
```

`POST /api/events` mit `{"keyword": "...", "text": "..."}` normalisiert auf `Source = "WEBHOOK"` und legt das Event in einen gepufferten Channel (Kapazität 100). Bei vollem Puffer: `503`.

### 4.1 Authentifizierung

`/api/events`, `/api/announcements`, `/api/acknowledgements` und `/api/heartbeat/{name}` verlangen einen API-Key als `Authorization: Bearer <key>` oder `X-API-Key: <key>` Header. Vergleich constant-time (`crypto/subtle`). Alle übrigen `/api/...`-Endpoints verlangen einen gültigen JWT-Session-Token (Login via `/api/login`). `/health` bleibt offen.

---

## 5. Processor (Kernlogik)

Der `Processor` implementiert drei pluggable Interfaces:

```go
type MaintenanceChecker interface {
    IsUnderMaintenance(ctx, source, keyword string, at time.Time) (bool, error)
}
type TemplateLoader interface {
    GetTemplate(ctx, channel string) (string, error)
}
type AckStatusChecker interface {
    IsAcknowledged(ctx, eventLogID int) (bool, error)
}
```

**Verarbeitungsreihenfolge pro Event:**

1. `MaintenanceChecker.IsUnderMaintenance()` — trifft ein Fenster: `MAINTENANCE`-Log, return
2. `RuleEngine.FindMatchingRule()` — kein Match: `NO_MATCH`-Log, return
3. `AuditWriter.LogEvent()` — schreibt `event_log`-Eintrag
4. Pro Kanal: `TemplateLoader.GetTemplate()` → Platzhalter ersetzen → `Notifier.Notify()`
5. Bei `repeat_interval_sec > 0`: `repeatNotify()` Goroutine starten

### 5.1 Wiederhol-Benachrichtigung

```go
func (p *Processor) repeatNotify(ctx, eventLogID, rule, event) {
    for i := 0; i < min(rule.RepeatMax, 10); i++ {
        time.Sleep(rule.RepeatIntervalSec)
        if acked, _ := p.AckChecker.IsAcknowledged(ctx, eventLogID); acked {
            return  // quittiert — Stop
        }
        p.dispatch(ctx, rule, event)
    }
}
```

Maximale Wiederholungen: `min(RepeatMax, 10)` (Schutz vor Endlosschleifen).

---

## 6. Regel-Engine

```go
func (e *RuleEngine) FindMatchingRule(event Event) (*Rule, error) {
    // source + keyword exakt
    // weekdays_only: Mo-Fr per time.Weekday()
    // [start_hour, end_hour): halboffenes Intervall
    // Bei mehreren Treffern: niedrigster Priority-Wert
}
```

**Bekannte Lücke:** Zeitfenster über Mitternacht (`start=17, end=6`) matcht nie — zwei Regeln als Workaround.

**Ausfallsicherheit:** `Reload()` alle 60s; DB-Fehler → letzter bekannter Regelsatz bleibt unverändert.

---

## 7. Notifier (Ausgangskanäle)

```go
type Notifier interface {
    Channel() string  // "voice" | "sms" | "email" | "teams"
    Notify(ctx context.Context, recipients []Recipient, message string) error
}
```

Registrierung in `main.go` ist Env-Var-getrieben: fehlt die Haupt-Variable, wird der Notifier nicht angelegt. Unbekannte Kanäle in `rules.channels` erzeugen nur eine Warnung.

| Kanal | Impl | Details |
|---|---|---|
| `voice` | Asterisk AMI | `Originate` pro Empfänger; TTS via espeak-ng+sox (offline, `TTS_ENABLED=true`); WAV-Cleanup nach 90s verzögert |
| `sms` | SMSEAGLE HTTP-API | Ein Request pro Empfänger |
| `email` | `net/smtp` | Ein Mail an alle Empfänger; typisch gegen internen Smarthost |
| `teams` | Incoming Webhook | `{"text":"..."}` an Kanal; `recipients`-Liste wird ignoriert |

---

## 8. Quittierung

```
POST /api/acknowledgements  {"event_id", "recipient_id", "response", "channel"}
GET  /api/acknowledgements?event_id=N
```

Kanalunabhängig. Upsert per `(event_log_id, recipient_id)`. `response`: `COMING` | `NOT_COMING` | `LATER`.

---

## 9. Wartungsfenster

```
GET    /api/maintenance        → Liste aller Fenster
POST   /api/maintenance        → Anlegen (label, starts_at, ends_at, source?, keyword?)
PUT    /api/maintenance/{id}   → Bearbeiten
DELETE /api/maintenance/{id}   → Löschen
```

Timestamps als RFC3339. Aktive Fenster (jetzt zwischen starts_at und ends_at) unterdrücken Alarme sofort ohne Regeländerung. Globale Fenster (ohne source/keyword) greifen für alle Events.

---

## 10. Heartbeat-Überwachung

```
GET    /api/heartbeats         → Liste mit berechnetem Status (OK/LATE/UNKNOWN)
POST   /api/heartbeats         → Heartbeat anlegen
PUT    /api/heartbeats/{id}    → Bearbeiten
DELETE /api/heartbeats/{id}    → Löschen
POST   /api/heartbeat/{name}   → Ping (API-Key-Auth, für externe Systeme)
```

Status-Logik:
- `OK` — last_seen innerhalb `interval_sec + grace_sec`
- `LATE` — überfällig
- `UNKNOWN` — noch nie gepingt

Bei Ausfall wird ein `core.Event` mit `Source=heartbeat.Source`, `Keyword=heartbeat.Keyword` erzeugt und normal durch die Regelverarbeitung geleitet — Wartungsfenster, Templates und Wiederholungen greifen also auch für Heartbeat-Alarme.

---

## 11. Nachrichten-Templates

```
GET    /api/templates          → Liste aller Templates
PUT    /api/templates/{channel}→ Setzen/Aktualisieren
DELETE /api/templates/{channel}→ Löschen (Fallback auf Rohtext)
```

Template-Beispiel für `voice`:
```
Alarm: {keyword} von {source} — {text} — Uhrzeit {time} — Gruppe {group}
```

Platzhalter werden via `strings.NewReplacer` ersetzt. Fehlt ein Template, wird `event.Text` verwendet.

---

## 12. Alarm-Dashboard (NOC View)

```
GET /api/dashboard
```

Liefert in einem Aufruf:
- `recent_events` — letzte 20 Einträge aus `event_log`
- `open_events` — unquittierte Events der letzten 24h (rot hervorgehoben im Frontend)
- `active_maintenance` — derzeit aktive Wartungsfenster
- `heartbeats` — alle Heartbeats mit berechnetem Status

Das Frontend zeigt diese Daten auf einer Echtzeit-aktualisierbaren Übersichtsseite (30s Auto-Refresh) — primär für NOC-Displays gedacht.

---

## 13. Statistiken

```
GET /api/stats
```

Rückgabe:
- `total_events` — Gesamtanzahl Events
- `acked_events` — davon quittiert
- `events_per_day` — Array der letzten 30 Tage (für Balkendiagramm)
- `top_keywords` — Top-10 Keywords nach Häufigkeit
- `top_sources` — Top-10 Sources
- `channel_usage` — Nutzungszähler pro Kanal (via `UNNEST(channels)`)

---

## 14. REST-API & OpenAPI

Vollständige OpenAPI-3.0-Spezifikation unter `GET /api/openapi.yaml`. Swagger UI unter `GET /api/docs` (Swagger UI 5 via CDN).

Alle Admin-Endpoints (`/api/rules`, `/api/groups`, `/api/recipients`, `/api/users`, `/api/channel-configs`, `/api/apikeys`, `/api/maintenance`, `/api/heartbeats`, `/api/templates`, `/api/stats`, `/api/dashboard`) erfordern JWT-Auth mit `role=admin`. Viewer-Rolle kann `/api/events` (Liste), `/api/acknowledgements`, `/api/announcements` lesen.

API-Keys (gehasht mit bcrypt in `api_keys`-Tabelle) sind für maschinelle Einlieferer (`POST /api/events`, `POST /api/heartbeat/{name}`). Plaintext wird nur bei Erstellung einmalig angezeigt.

---

## 15. Frontend (Vue 3 + Tailwind)

Einzelne SPA unter `frontend/`, gebaut mit Vite + Tailwind CSS + Pinia.

| Route | Seite | Rolle |
|---|---|---|
| `/dashboard` | Alarm-Dashboard (NOC View) | alle |
| `/` | Regeln (CRUD) | admin |
| `/groups` | Gruppen & Empfänger | admin |
| `/events` | Event-Log (Filter, Ack) | alle |
| `/announcements` | Sound-Uploads | admin |
| `/maintenance` | Wartungsfenster | admin |
| `/heartbeats` | Heartbeat-Überwachung | admin |
| `/templates` | Nachrichten-Templates | admin |
| `/stats` | Statistiken | admin |
| `/channels` | Kanal-Konfiguration | admin |
| `/apikeys` | API-Key-Verwaltung | admin |
| `/users` | Benutzerverwaltung | admin |
| `/me/groups` | Eigene Gruppenaktivierung | alle |
| `/handbuch` | Vollständiges Handbuch | alle |

Dark Mode mit Tailwind `dark:` Klassen, persistiert in localStorage. Kontextsensitive Hilfe via F1. Vollständiges Handbuch (`/handbuch`) mit Sidebar-Navigation.

---

## 16. Deployment (Debian 13, nativ — kein Docker für Asterisk)

Alle Scripts unter `deploy/`.

| Script | Zweck |
|---|---|
| `setup.sh` | Interaktiver Assistent, schreibt `/etc/alarm-core/alarm-core.env` |
| `install-postgres.sh` | Paket, Rolle/DB, Migrationen, täglicher pg_dump (14 Tage) |
| `install-asterisk.sh` | Build aus Quellcode (kein Debian-13-Paket), AMI, PJSIP, TTS |
| `install-alarm-core.sh` | Service-User, Binary, systemd-Unit |

**Stolpersteine:**
1. `Type=notify` im Asterisk-Unit braucht `libsystemd-dev` beim Build.
2. Postgres-Postinst startet Cluster nicht zuverlässig — Script ruft `systemctl enable --now postgresql` + `pg_isready` explizit auf.
3. TTS-WAV-Schreibrechte: `alarm-core` muss Gruppe `asterisk` sein + `ReadWritePaths=/var/lib/asterisk/sounds/custom` in systemd-Unit.

**Proxy/Mailrelay:** `setup.sh` fragt optional HTTP/HTTPS-Proxy (Go respektiert `HTTP_PROXY`/`HTTPS_PROXY` via `http.ProxyFromEnvironment`) und Postfix-Satellite-Setup ab.

---

## 17. Offene Punkte

| Punkt | Stand |
|---|---|
| Inbound-Quittierungskanal | `/api/acknowledgements` existiert, aber kein automatischer Aufrufer (SMS-Inbound, DTMF) |
| Weitere Eingangskanäle | IMAP/E-Mail-Polling, seriell/DME — `core.Event` ist vorbereitet |
| Mitternacht-Zeitfenster | Bewusst nicht gefixt, Workaround: zwei Regeln |
| Check_MK-Plugin | `/health` liefert DB-Status; Agent-Plugin nicht gebaut |
| LDAP-Authentifizierung | Benutzer aktuell nur lokal in `users`-Tabelle |
| SMSEAGLE API-Key-Login | SMSEAGLE unterstützt auch API-Key-Auth — noch nicht implementiert |
| Eigene Gruppenaktivierung durch User | UI vorhanden (`/me/groups`), zeitgesteuerte Slots (z.B. nur 8–17 Uhr aktiv) noch offen |

---

## 18. Repo-Struktur

```
alarmserver/
├── cmd/alarm-core/main.go            — Wiring, Env-Var-Konfiguration, HTTP-Server
├── internal/
│   ├── core/                         — Event, Rule, RuleEngine, Processor,
│   │                                    Notifier/AuditWriter/MaintenanceChecker/
│   │                                    TemplateLoader/AckStatusChecker-Interfaces
│   ├── db/                           — PostgreSQL-Implementierungen:
│   │   ├── rules.go, groups.go       — CRUD
│   │   ├── audit.go, acknowledgements.go
│   │   ├── maintenance.go            — Wartungsfenster (v2.0)
│   │   ├── heartbeats.go             — Heartbeat-Store (v2.0)
│   │   ├── templates.go              — Nachrichten-Templates (v2.0)
│   │   ├── stats.go                  — Statistiken (v2.0)
│   │   └── migrations/               — 0001…0011_*.sql
│   ├── api/                          — HTTP-Handler:
│   │   ├── events.go, announcements.go, acknowledgements.go
│   │   ├── maintenance.go            — v2.0
│   │   ├── heartbeats.go             — v2.0
│   │   ├── templates.go              — v2.0
│   │   ├── dashboard.go              — v2.0
│   │   ├── stats.go                  — v2.0
│   │   └── openapi.go                — Swagger UI + YAML (v2.0)
│   ├── notify/{asterisk,smseagle,email,teams}/
│   └── tts/espeak/
├── frontend/                         — Vue 3 + Vite + Tailwind
│   └── src/
│       ├── views/                    — 14 Views inkl. Dashboard, Handbook
│       ├── stores/                   — auth, theme, sidebar (Pinia)
│       ├── components/               — AppSidebar, HelpPanel, ...
│       └── content/helpTexts.js      — Kontexthilfe für alle Routen
└── deploy/                           — setup.sh + install-*.sh + systemd-Unit + Env-Template
```

## 19. Repo

```
https://github.com/bh2005/alarmserver
```
