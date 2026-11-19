# Alarm-Core

Mandantenfähiges, kanalunabhängiges Alarmierungssystem der Example AG.

Empfängt Events via HTTP-Webhook, matched sie gegen ein konfigurierbares Regelwerk und alarmiert Zielgruppen über Telefon (Asterisk), SMS (SMSEAGLE), E-Mail oder Microsoft Teams. Vollständig auditiert, mit Wartungsfenstern, Heartbeat-Überwachung, Wiederhol-Benachrichtigung und einer Vue-3-Management-Oberfläche.

---

## Schnellstart (lokale Entwicklung mit Docker)

```bash
# PostgreSQL + Frontend-Dev-Server
docker compose up -d db
cd frontend && npm install && npm run dev

# Backend (Go)
go run ./cmd/alarm-core
```

Umgebungsvariablen aus `deploy/alarm-core.env.example` kopieren und anpassen:

```bash
cp deploy/alarm-core.env.example .env
# .env bearbeiten, dann:
export $(cat .env | xargs)
go run ./cmd/alarm-core
```

Das Frontend läuft auf Port 5173 (Vite Dev), das Backend auf Port 8080.

---

## Architektur

```
HTTP-Webhook  ──►  Processor  ──►  Notifier (voice / sms / email / teams)
                       │
              Wartungsfenster-Check
              Template-Auflösung
              Wiederhol-Benachrichtigung
                       │
                  PostgreSQL
                       │
              Vue 3 Frontend (SPA)
```

Weitere Details: [konzept.md](konzept.md)

---

## Komponenten

| Schicht | Technologie |
|---|---|
| Backend | Go 1.26, `pgx/v5`, `net/http` |
| Datenbank | PostgreSQL 15+, Migrationen in `internal/db/migrations/` |
| Frontend | Vue 3, Vite, Tailwind CSS, Pinia |
| Sprachausgabe | Asterisk AMI + espeak-ng + sox (offline TTS) |
| SMS | SMSEAGLE HTTP-API |
| E-Mail | `net/smtp` gegen internen Smarthost |
| Teams | Incoming Webhook |

---

## Konfiguration (Umgebungsvariablen)

| Variable | Pflicht | Beschreibung |
|---|---|---|
| `DATABASE_URL` | ✓ | PostgreSQL DSN, z.B. `postgres://alarm:pass@localhost/alarmcore` |
| `ALARM_API_KEY` | — | API-Key für Einlieferer; fehlt → unauthentifiziert (Warnung im Log) |
| `JWT_SECRET` | — | Geheimnis für JWT-Session-Tokens; wird beim Start autogeneriert falls leer |
| `LISTEN_ADDR` | — | Bind-Adresse, Default `:8080` |
| `ASTERISK_HOST` | — | AMI-Adresse, z.B. `127.0.0.1:5038` |
| `ASTERISK_USER` | — | AMI-Login |
| `ASTERISK_PASS` | — | AMI-Passwort |
| `ASTERISK_CHANNEL_TPL` | — | Channel-Template, Default `PJSIP/{number}` |
| `TTS_ENABLED` | — | `true` aktiviert espeak-ng TTS |
| `SMSEAGLE_BASE_URL` | — | z.B. `http://smseagle.intern` |
| `SMSEAGLE_USER` | — | |
| `SMSEAGLE_PASS` | — | |
| `SMTP_HOST` | — | z.B. `mailrelay.intern:25` |
| `SMTP_FROM` | — | Absenderadresse |
| `TEAMS_WEBHOOK_URL` | — | Incoming-Webhook-URL |

Kanäle werden nur registriert wenn ihre Haupt-Variable gesetzt ist. Unbekannte Kanäle in Regeln erzeugen eine Log-Warnung, keinen Absturz.

---

## Features (v2.0)

- **Regelwerk** — Source + Keyword + Zeitfenster + Wochentag + Priorität; hot-reload alle 60s
- **Wartungsfenster** — unterdrücken Alarme zeitgesteuert ohne Regeländerung; optionaler Source/Keyword-Filter
- **Nachrichten-Templates** — kanalspezifische Texte mit Platzhaltern `{text}` `{source}` `{keyword}` `{time}` `{group}`
- **Wiederhol-Benachrichtigung** — re-alarmiert bis zur Quittierung, konfigurierbares Intervall und Maximum
- **Heartbeat-Überwachung** — Dead-Man's-Switch; externe Systeme pingen `/api/heartbeat/{name}`, Ausfall löst Alarm aus
- **Alarm-Dashboard** — NOC-View mit offenen Alarmen, aktiven Wartungsfenstern und Heartbeat-Status
- **Statistiken** — Events/Tag (30 Tage), Top-Keywords/-Sources, Kanal-Nutzung
- **Vollständige OpenAPI-Dokumentation** — Swagger UI unter `/api/docs`, Spec unter `/api/openapi.yaml`
- **JWT-Auth + API-Keys** — Admin/Viewer-Rollen; API-Keys für maschinelle Einlieferer (bcrypt, Plaintext nur bei Erstellung)
- **Dark Mode** — Tailwind `dark:`-Varianten, per Button umschaltbar, persistiert in localStorage
- **Kontexthilfe + Handbuch** — F1 öffnet seitenspezifische Hilfe; vollständiges Handbuch unter `/handbuch`

---

## API (Kurzübersicht)

```
# Events
POST   /api/events                    # Alarm einliefern (API-Key)
GET    /api/events                    # Log abrufen (JWT)

# Quittierung
POST   /api/acknowledgements          # Quittieren
GET    /api/acknowledgements?event_id=N

# Ansagen
GET    /api/announcements             # Sound-Dateien auflisten
POST   /api/announcements             # Upload (multipart)
DELETE /api/announcements/{name}

# Wartungsfenster
GET    /api/maintenance
POST   /api/maintenance
PUT    /api/maintenance/{id}
DELETE /api/maintenance/{id}

# Heartbeats
GET    /api/heartbeats
POST   /api/heartbeats
PUT    /api/heartbeats/{id}
DELETE /api/heartbeats/{id}
POST   /api/heartbeat/{name}          # Ping (API-Key)

# Templates
GET    /api/templates
PUT    /api/templates/{channel}
DELETE /api/templates/{channel}

# Dashboard & Statistiken
GET    /api/dashboard
GET    /api/stats

# Regeln, Gruppen, Empfänger, Benutzer, Kanäle, API-Keys
# → vollständig unter /api/docs (Swagger UI)

# Authentifizierung
POST   /api/login
POST   /api/logout

# Health
GET    /health
```

---

## Deployment (Produktion, Debian 13)

### Voraussetzungen und Build

- Debian 13, Root-Zugriff und Internetzugang (oder ein HTTP-Proxy; `setup.sh` fragt danach).
- Das Binary `alarm-core` ist **nicht** im Repository enthalten. `setup.sh` prüft das gleich am Anfang und baut es auf Wunsch
  aus dem Quellcode (installiert dafür bei Bedarf `golang-go`). Es wird gebaut, bevor PostgreSQL und Asterisk installiert werden.
- `go.mod` verlangt Go 1.26. Debian 13 liefert eine ältere Version; die Go-Toolchain lädt die passende Version automatisch nach
  (`GOTOOLCHAIN=auto`, benötigt Internet oder den Proxy). Alternativ ein aktuelles Go von <https://go.dev/dl/> installieren.

Manuell bauen (auf dem Zielrechner, im Repo-Verzeichnis):

```bash
go build -o deploy/alarm-core ./cmd/alarm-core
# auf einem anderen Rechner bauen: GOOS=linux GOARCH=amd64 go build -o deploy/alarm-core ./cmd/alarm-core
```

*English:* the `alarm-core` binary is not shipped in the repository. `setup.sh` checks for it first and can build it from source
(it installs `golang-go` if needed). `go.mod` requires Go 1.26; the Go toolchain downloads it automatically if the packaged Go is older.

### Installation

```bash
cd deploy
sudo bash setup.sh
```

`setup.sh` ist ein interaktiver Assistent der alle Optionen abfragt und die drei Install-Scripts (`install-postgres.sh`, `install-asterisk.sh`, `install-alarm-core.sh`) aufruft.

**Wichtig:** Asterisk wird aus dem Quellcode gebaut — das Debian-13-Paket existiert in main nicht mehr.

---

## Repo-Struktur

```
alarmserver/
├── cmd/alarm-core/main.go          — Wiring + HTTP-Server
├── internal/
│   ├── core/                       — Interfaces, RuleEngine, Processor
│   ├── db/                         — PostgreSQL-Stores + migrations/
│   ├── api/                        — HTTP-Handler
│   ├── notify/                     — asterisk / smseagle / email / teams
│   └── tts/espeak/                 — Offline-TTS
├── frontend/                       — Vue 3 + Vite + Tailwind
├── deploy/                         — Installations-Scripts + systemd-Unit
└── konzept.md                      — Vollständige technische Dokumentation
```

---

## Kontakt / Intern

Projekt: `github.com/bh2005/alarmserver`
Ansprechpartner: noreply@example.com
