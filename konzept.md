# Alarm-Core: Konzept für ein mandantenfähiges Alarmierungssystem

**Version:** 0.3  
**Stand:** Juni 2026  
**Zielgruppe:** Freiwillige Feuerwehren, BOS-Einheiten

---

## 1. Zielsetzung

Dieses Dokument beschreibt die Architektur eines modularen Alarmierungssystems für Freiwillige Feuerwehren. Das System soll:

- Alarme aus verschiedenen Eingangskanälen (Leitstelle, DME, manuell) empfangen und normalisieren
- Einsatzkräfte anhand konfigurierbarer Regeln und Zeitpläne alarmieren
- Tages- und Nachtschleifen sowie Wochenend-Logik abbilden
- Ausfallsicher und ohne manuelle Code-Änderungen administrierbar sein

---

## 2. Architekturübersicht

```
┌─────────────────────────────────────────────────────────────┐
│                        Eingangskanäle                       │
│   REST-API │  IMAP/E-Mail  │  DME/Seriell (/dev/ttyUSB0)   │
└──────────────────────┬──────────────────────────────────────┘
                       │ normalisierter Alarm-Event
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                       Alarm-Core (Go)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Regel-Engine │  │  Scheduler   │  │   Audit-Logger   │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────────────┘  │
│         └────────┬─────────┘                                │
└──────────────────┼──────────────────────────────────────────┘
                   │ aufgelöste Zielgruppe + Kanäle
                   ▼
┌─────────────────────────────────────────────────────────────┐
│                      Ausgangskanäle                         │
│        Asterisk (SIP/VoIP)  │  SMSEAGLE (SMS)              │
└─────────────────────────────────────────────────────────────┘
```

**Kernprinzip:** Kein Eingangskanal kennt die Ausgangslogik. Kein Ausgangskanal kennt die Regellogik. Der Alarm-Core ist der einzige Ort, an dem beides zusammenkommt. Der Logging-Layer ist vollständig vom Core entkoppelt und bedient zwei getrennte Zielgruppen: Administratoren (technisch) und Vorgesetzte (fachlich).

---

## 3. Datenmodell (PostgreSQL)

### 3.1 Einsatzkräfte

```sql
CREATE TABLE einsatzkraefte (
    id          SERIAL PRIMARY KEY,
    vorname     VARCHAR(100) NOT NULL,
    nachname    VARCHAR(100) NOT NULL,
    rufnummer   VARCHAR(30),           -- für SIP-Anruf
    mobilnummer VARCHAR(30),           -- für SMS
    aktiv       BOOLEAN NOT NULL DEFAULT TRUE,
    erstellt_am TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 3.2 Einsatzgruppen

Eine Person kann mehreren Gruppen angehören (z. B. *Atemschutz* und *Führung*).

```sql
CREATE TABLE einsatzgruppen (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE, -- z.B. "Loeschzug_Tag", "AGT", "Wehrfuehrung"
    beschreibung TEXT
);

-- M:N-Verknüpfung Personen ↔ Gruppen
CREATE TABLE gruppen_mitglieder (
    gruppe_id INT NOT NULL REFERENCES einsatzgruppen(id) ON DELETE CASCADE,
    person_id INT NOT NULL REFERENCES einsatzkraefte(id) ON DELETE CASCADE,
    PRIMARY KEY (gruppe_id, person_id)
);
```

### 3.3 Regelwerk (Routing-Tabelle)

Das Herzstück des Systems. Jede Zeile definiert eine "Wenn–Dann"-Bedingung. Keine Regel = kein Code-Deployment.

```sql
CREATE TABLE alarm_regeln (
    id             SERIAL PRIMARY KEY,
    trigger_quelle VARCHAR(50)  NOT NULL, -- z.B. "LEITSTELLEN_API", "DME_SCANNER", "MANUELL"
    stichwort      VARCHAR(100) NOT NULL, -- z.B. "F 2", "H 1", "B MITTEL"
    -- Zeitfenster: NULL = gilt immer
    start_stunde   SMALLINT,              -- 0–23, NULL = kein Limit
    end_stunde     SMALLINT,              -- 0–23, NULL = kein Limit
    nur_werktags   BOOLEAN NOT NULL DEFAULT FALSE,
    ziel_gruppe_id INT NOT NULL REFERENCES einsatzgruppen(id),
    kanaele        TEXT[] NOT NULL,       -- z.B. ARRAY['SIP', 'SMS']
    prioritaet     SMALLINT NOT NULL DEFAULT 10, -- niedrigere Zahl = höhere Prio
    aktiv          BOOLEAN NOT NULL DEFAULT TRUE
);

-- Index für schnelles Matching beim Alarm-Eingang
CREATE INDEX idx_regeln_lookup ON alarm_regeln (trigger_quelle, stichwort, aktiv);
```

**Beispiel-Datensätze:**

| Quelle | Stichwort | Zeitfenster | Nur Werktags | Ziel-Gruppe | Kanäle |
|---|---|---|---|---|---|
| `LEITSTELLEN_API` | `F 2` | 06:00–17:00 | Ja | `Zug_Mitte` + `FF_Nachbarort` | SIP, SMS |
| `LEITSTELLEN_API` | `F 2` | 17:00–06:00 / WE | Nein | `Zug_Mitte_Vollalarm` | SIP, SMS |
| `DME_SCANNER` | `H 1` | — | Nein | `Kleinschleife_Technik` | SMS |
| `MANUELL` | `INTERN` | — | Nein | `Wehrfuehrung` | SIP |

### 3.4 Audit-Log

Jeder Alarm wird vollständig protokolliert – für Nachbesprechungen und rechtliche Absicherung.

```sql
CREATE TABLE alarm_log (
    id             SERIAL PRIMARY KEY,
    eingang_um     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trigger_quelle VARCHAR(50)  NOT NULL,
    stichwort      VARCHAR(100) NOT NULL,
    roh_payload    JSONB,                -- Original-Daten des Eingangskanals
    regel_id       INT REFERENCES alarm_regeln(id),
    ziel_gruppe    VARCHAR(100),
    kanaele        TEXT[],
    status         VARCHAR(20) NOT NULL DEFAULT 'AUSGELOEST', -- AUSGELOEST | FEHLER | KEIN_TREFFER
    fehler_detail  TEXT
);
```

---

## 4. Trigger-Engine (Eingangskanäle)

Jeder Eingangskanal ist ein eigenständiger **Worker** (Go-Goroutine), der seinen Input auf einen einheitlichen `AlarmEvent`-Typ normalisiert und an den Core weitergibt.

```go
// Gemeinsamer Typ für alle Eingangskanäle
type AlarmEvent struct {
    Quelle    string          // "LEITSTELLEN_API", "DME_SCANNER", "MANUELL"
    Stichwort string          // "F 2", "H 1", ...
    Klartext  string          // Freitext des Alarms (für Anzeige/Log)
    Eingang   time.Time
    RohDaten  json.RawMessage // Original-Payload für den Audit-Log
}
```

### 4.1 HTTP REST API

Für moderne Einsatzleitrechner (ELR) oder Systeme wie Divera/GroupAlarm.

```go
func (s *Server) handleAlarmWebhook(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Stichwort string `json:"stichwort"`
        Klartext  string `json:"klartext"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "ungültiger Body", http.StatusBadRequest)
        return
    }
    event := AlarmEvent{
        Quelle:    "LEITSTELLEN_API",
        Stichwort: payload.Stichwort,
        Klartext:  payload.Klartext,
        Eingang:   time.Now(),
    }
    s.alarmChan <- event
    w.WriteHeader(http.StatusAccepted)
}
```

### 4.2 IMAP / E-Mail

Viele Leitstellen senden Alarme parallel als strukturierte E-Mail. Ein Hintergrunddienst pollt das Postfach und parst den Text per RegEx.

```go
func (w *ImapWorker) Run(alarmChan chan<- AlarmEvent) {
    ticker := time.NewTicker(15 * time.Second)
    for range ticker.C {
        mails, err := w.fetchUnread()
        if err != nil {
            log.Printf("[IMAP] Fehler beim Abruf: %v", err)
            continue
        }
        for _, mail := range mails {
            stichwort := parseStichwort(mail.Body) // RegEx auf Leitstellen-Format
            if stichwort == "" {
                continue
            }
            alarmChan <- AlarmEvent{
                Quelle:    "IMAP",
                Stichwort: stichwort,
                Klartext:  mail.Body,
                Eingang:   time.Now(),
            }
        }
    }
}
```

### 4.3 DME / Seriell

Ein USB-Funkscanner oder Pager-Cradle (z. B. Boss 915/935) gibt Textdaten über `/dev/ttyUSB0` aus.

```go
func (w *SerialWorker) Run(alarmChan chan<- AlarmEvent) {
    port, err := serial.Open("/dev/ttyUSB0", &serial.Mode{BaudRate: 9600})
    if err != nil {
        log.Fatalf("[SERIAL] Port nicht verfügbar: %v", err)
    }
    scanner := bufio.NewScanner(port)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if stichwort := parsePagerZeile(line); stichwort != "" {
            alarmChan <- AlarmEvent{
                Quelle:    "DME_SCANNER",
                Stichwort: stichwort,
                Klartext:  line,
                Eingang:   time.Now(),
            }
        }
    }
}
```

---

## 5. Regel-Engine (Routing-Logik)

Die Regel-Engine lädt die `alarm_regeln`-Tabelle beim Start in den Arbeitsspeicher und hält sie per periodischem Reload aktuell. **Keine Stichwörter sind im Code hartkodiert.**

```go
type AlarmRegel struct {
    ID           int
    TriggerQuelle string
    Stichwort    string
    StartStunde  *int    // nil = kein Limit
    EndStunde    *int
    NurWerktags  bool
    ZielGruppe   string
    Kanaele      []string
    Prioritaet   int
}

func (e *RegelEngine) FindePassendeRegel(event AlarmEvent) (*AlarmRegel, error) {
    jetzt := event.Eingang
    stunde := jetzt.Hour()
    istWerktag := jetzt.Weekday() >= time.Monday && jetzt.Weekday() <= time.Friday

    var treffer []AlarmRegel
    for _, regel := range e.regeln {
        if regel.TriggerQuelle != event.Quelle || regel.Stichwort != event.Stichwort {
            continue
        }
        if regel.NurWerktags && !istWerktag {
            continue
        }
        if regel.StartStunde != nil && stunde < *regel.StartStunde {
            continue
        }
        if regel.EndStunde != nil && stunde >= *regel.EndStunde {
            continue
        }
        treffer = append(treffer, regel)
    }

    if len(treffer) == 0 {
        return nil, fmt.Errorf("keine Regel für Quelle=%s Stichwort=%s", event.Quelle, event.Stichwort)
    }

    // Höchste Priorität gewinnt (niedrigster Wert)
    sort.Slice(treffer, func(i, j int) bool {
        return treffer[i].Prioritaet < treffer[j].Prioritaet
    })
    return &treffer[0], nil
}
```

> **Hinweis zur Zeitlogik:** Das Zeitfenster wird als `[StartStunde, EndStunde)` behandelt – d. h. `start=6, end=17` gilt von 06:00:00 bis 16:59:59. Stunde 17 ist bereits Nacht.

---

## 6. Ausgangskanäle (Telefonie & SMS)

### 6.1 Asterisk (SIP/VoIP) via AMI

Der Alarm-Core spricht mit Asterisk über das **Asterisk Manager Interface (AMI)**. Dabei wird pro Zielperson ein `Originate`-Kommando abgesetzt, das einen ausgehenden Anruf auslöst und bei Verbindung eine Sprachdurchsage (`.wav`/Text-to-Speech) abspielt.

```go
func (a *AsteriskClient) RufeAn(rufnummer string, ansage string) error {
    cmd := fmt.Sprintf(
        "Action: Originate\r\n"+
        "Channel: SIP/%s\r\n"+
        "Application: Playback\r\n"+
        "Data: %s\r\n"+
        "Timeout: 30000\r\n\r\n",
        rufnummer, ansage,
    )
    _, err := a.conn.Write([]byte(cmd))
    return err
}
```

### 6.2 SMSEAGLE (SMS)

SMSEAGLE bietet eine REST-API für den SMS-Versand.

```go
func (s *SmseagleClient) SendeSMS(nummer string, text string) error {
    url := fmt.Sprintf(
        "%s/index.php/http_api/send_sms?login=%s&pass=%s&to=%s&message=%s",
        s.BaseURL, s.Login, s.Passwort,
        url.QueryEscape(nummer),
        url.QueryEscape(text),
    )
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("SMSEAGLE Fehler: HTTP %d", resp.StatusCode)
    }
    return nil
}
```

---

## 7. Ausfallsicherheit

Ein Alarmierungssystem muss im schlimmsten Moment funktionieren.

### 7.1 Datenbankausfall

Regeln werden beim Start in den Speicher geladen. Ist die DB beim Alarm-Eingang nicht erreichbar, arbeitet der Core mit dem zuletzt bekannten Stand weiter.

```go
func (e *RegelEngine) StarteHintergrundReload(intervall time.Duration) {
    go func() {
        for range time.NewTicker(intervall).C {
            neueRegeln, err := e.db.LadeAlleRegeln()
            if err != nil {
                log.Printf("[RegelEngine] Reload fehlgeschlagen, behalte alten Stand: %v", err)
                continue // KEIN Überschreiben bei Fehler
            }
            e.mu.Lock()
            e.regeln = neueRegeln
            e.mu.Unlock()
        }
    }()
}
```

### 7.2 Watchdog / Health-Check

Ein HTTP-Endpunkt `/health` meldet den Status aller Subsysteme. Externe Überwachung (z. B. Prometheus/Alertmanager oder ein einfaches Bash-Cron-Script) kann darauf aufsetzt.

```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    status := map[string]string{
        "db":        s.db.Ping(),       // "ok" oder Fehlermeldung
        "asterisk":  s.ami.Ping(),
        "smseagle":  s.sms.Ping(),
        "serial":    s.serial.Status(),
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}
```

---

## 8. Logging

Das System unterscheidet strikt zwischen zwei Log-Schichten mit unterschiedlichen Zielgruppen, Formaten und Aufbewahrungsfristen.

```
┌─────────────────────────────────────────────────────────────┐
│                       Alarm-Core                            │
│                          │                                  │
│          ┌───────────────┴───────────────┐                  │
│          ▼                               ▼                  │
│   System-Logger                   Fach-Logger               │
│   (structured JSON                (PostgreSQL-              │
│    → journald)                     Tabellen)                │
└──────────┬────────────────────────────┬──────────────────── ┘
           │                            │
           ▼                            ▼
     Admin-Sicht                  Führungs-Sicht
  (Check_MK, Logfile)          (Auswertungs-Queries,
                                 späteres Web-UI)
```

---

### 8.1 System-Logging (Admin-Sicht)

Das System-Logging erfasst den **technischen Betriebszustand**: Worker-Status, Verbindungsfehler, Latenzen, Regel-Reload. Es schreibt strukturiertes JSON an `stdout` / `stderr`, wird von `systemd` via `journald` erfasst und von Check_MK ausgewertet (siehe Kapitel 9).

**Log-Level-Konzept:**

| Level | Wann | Beispiel |
|---|---|---|
| `DEBUG` | Nur im Entwicklungsbetrieb | Eingehende Zeile am seriellen Port |
| `INFO` | Normalbetrieb, jeder Alarm-Eingang | Alarm ausgelöst, Regel getroffen |
| `WARN` | Degradierter Betrieb, aber funktionsfähig | DB-Reload fehlgeschlagen, arbeite mit Cache |
| `ERROR` | Funktionsverlust an einem Kanal | Asterisk AMI nicht erreichbar |
| `FATAL` | System nicht startbar | Kein DB-Connect beim Start |

**Go-Implementierung (strukturiertes JSON-Logging mit `log/slog`):**

```go
import "log/slog"

// Initialisierung: JSON-Handler für journald-kompatible Ausgabe
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)

// Beispiele im Betrieb:
slog.Info("alarm_eingegangen",
    "quelle",    event.Quelle,
    "stichwort", event.Stichwort,
    "eingang",   event.Eingang.Format(time.RFC3339),
)

slog.Error("asterisk_nicht_erreichbar",
    "host",    a.Host,
    "fehler",  err.Error(),
    "versuch", versuch,
)

slog.Warn("regel_reload_fehlgeschlagen",
    "fehler",       err.Error(),
    "cache_alter_s", time.Since(e.letzterReload).Seconds(),
)
```

**Beispiel-Ausgabe (journald):**
```json
{"time":"2026-06-12T02:14:55Z","level":"INFO","msg":"alarm_eingegangen","quelle":"DME_SCANNER","stichwort":"F 2","eingang":"2026-06-12T02:14:55Z"}
{"time":"2026-06-12T02:14:55Z","level":"INFO","msg":"regel_getroffen","regel_id":3,"ziel_gruppe":"Zug_Mitte_Vollalarm","kanaele":["SIP","SMS"]}
{"time":"2026-06-12T02:14:56Z","level":"INFO","msg":"sip_anruf_gestartet","rufnummer":"+49561...","person_id":12}
{"time":"2026-06-12T02:14:58Z","level":"ERROR","msg":"sms_versand_fehlgeschlagen","person_id":7,"fehler":"HTTP 503"}
```

**Log-Rotation (logrotate):**
```
/var/log/alarm-core/*.log {
    daily
    rotate 30
    compress
    missingok
    notifempty
    postrotate
        systemctl kill -s HUP alarm-core.service
    endscript
}
```

---

### 8.2 Fachliches Logging / Auswertung (Führungs-Sicht)

Vorgesetzte benötigen keine Systemmetriken, sondern Antworten auf operative Fragen: Wer wurde wann alarmiert? Wie viele sind gekommen? Welche Stichwörter dominieren?

Das fachliche Logging schreibt ausschließlich in die PostgreSQL-Datenbank.

#### Datenmodell-Erweiterungen

```sql
-- Detaillierte Protokollierung pro alarmierter Person
CREATE TABLE alarm_zustellungen (
    id             SERIAL PRIMARY KEY,
    alarm_log_id   INT NOT NULL REFERENCES alarm_log(id) ON DELETE CASCADE,
    person_id      INT NOT NULL REFERENCES einsatzkraefte(id),
    kanal          VARCHAR(10) NOT NULL,  -- 'SIP' oder 'SMS'
    gesendet_um    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status         VARCHAR(20) NOT NULL,  -- 'GESENDET' | 'FEHLGESCHLAGEN' | 'KEINE_NUMMER'
    fehler_detail  TEXT
);

-- Quittierungen (Rückmeldungen der Kameraden)
CREATE TABLE quittierungen (
    id           SERIAL PRIMARY KEY,
    alarm_log_id INT NOT NULL REFERENCES alarm_log(id) ON DELETE CASCADE,
    person_id    INT NOT NULL REFERENCES einsatzkraefte(id),
    antwort      VARCHAR(10) NOT NULL,  -- 'KOMME' | 'KOMME_NICHT' | 'SPAETER'
    quittiert_um TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    kanal        VARCHAR(10),           -- über welchen Kanal quittiert wurde
    UNIQUE (alarm_log_id, person_id)
);

-- Indizes für schnelle Auswertungsabfragen
CREATE INDEX idx_zustellungen_alarm  ON alarm_zustellungen (alarm_log_id);
CREATE INDEX idx_quittierungen_alarm ON quittierungen (alarm_log_id);
CREATE INDEX idx_alarm_log_eingang   ON alarm_log (eingang_um DESC);
```

#### Auswertungs-Queries für Vorgesetzte

**Alarmaufkommen der letzten 12 Monate nach Stichwort:**
```sql
SELECT
    stichwort,
    COUNT(*)                                        AS anzahl_alarme,
    ROUND(AVG(EXTRACT(HOUR FROM eingang_um)), 1)    AS mittl_uhrzeit
FROM alarm_log
WHERE eingang_um >= NOW() - INTERVAL '12 months'
  AND status = 'AUSGELOEST'
GROUP BY stichwort
ORDER BY anzahl_alarme DESC;
```

**Verfügbarkeitsquote pro Kamerad (laufendes Jahr):**
```sql
SELECT
    e.nachname || ', ' || e.vorname              AS name,
    COUNT(DISTINCT q.alarm_log_id)               AS quittierungen_gesamt,
    COUNT(*) FILTER (WHERE q.antwort = 'KOMME')  AS zugesagt,
    ROUND(
        100.0 * COUNT(*) FILTER (WHERE q.antwort = 'KOMME')
        / NULLIF(COUNT(DISTINCT q.alarm_log_id), 0), 1
    )                                            AS verfuegbarkeit_pct
FROM einsatzkraefte e
LEFT JOIN quittierungen q ON q.person_id = e.id
LEFT JOIN alarm_log     a ON a.id = q.alarm_log_id
    AND a.eingang_um >= DATE_TRUNC('year', NOW())
WHERE e.aktiv = TRUE
GROUP BY e.id, e.nachname, e.vorname
ORDER BY verfuegbarkeit_pct DESC NULLS LAST;
```

**Alarmdetail-Protokoll (Nachbesprechung eines einzelnen Einsatzes):**
```sql
SELECT
    e.nachname || ', ' || e.vorname  AS kamerad,
    z.kanal,
    z.gesendet_um,
    z.status                         AS zustellung,
    q.antwort                        AS quittierung,
    q.quittiert_um
FROM alarm_zustellungen z
JOIN einsatzkraefte e ON e.id = z.person_id
LEFT JOIN quittierungen q
    ON q.person_id = z.person_id AND q.alarm_log_id = z.alarm_log_id
WHERE z.alarm_log_id = :alarm_id   -- Parameter: konkrete Alarm-ID
ORDER BY e.nachname, e.vorname;
```

---

## 9. Monitoring via Check_MK

Check_MK überwacht den technischen Betriebszustand des Alarm-Core. **Es greift ausschließlich auf den System-Log und den `/health`-Endpunkt zu – nie auf fachliche Daten.**

### 9.1 Lokaler Check_MK-Agent

Der Check_MK-Agent läuft als systemd-Service auf dem Alarm-Core-Host. Zusätzlich zur Standardüberwachung (CPU, RAM, Disk, Prozesse) werden zwei **Custom Checks** eingebunden.

**Agent-Plugin-Verzeichnis:** `/usr/lib/check_mk_agent/plugins/`

---

### 9.2 Custom Check: Subsystem-Health

Fragt den `/health`-Endpunkt des Alarm-Core ab und meldet den Status jedes Subsystems einzeln als Check_MK-Service.

```bash
#!/bin/bash
# /usr/lib/check_mk_agent/plugins/alarm_core_health

HEALTH_URL="http://localhost:8080/health"
RESPONSE=$(curl -sf --max-time 5 "$HEALTH_URL" 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "<<<alarm_core_health>>>"
    echo "2 Alarm-Core_Erreichbarkeit - CRIT: /health-Endpunkt nicht erreichbar"
    exit 0
fi

echo "<<<alarm_core_health>>>"

# Jeden Subsystem-Status einzeln auslesen und als eigenen Check ausgeben
for KEY in db asterisk smseagle serial; do
    VALUE=$(echo "$RESPONSE" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d.get('$KEY', 'unbekannt'))
" 2>/dev/null)

    if [ "$VALUE" = "ok" ]; then
        STATE=0
    else
        STATE=2
    fi

    LABEL=$(echo "$KEY" | tr '[:lower:]' '[:upper:]')
    echo "$STATE Alarm-Core_${LABEL} - $( [ $STATE -eq 0 ] && echo 'OK' || echo "CRIT: $VALUE" )"
done
```

**Check_MK-Sektions-Parser** (`~/local/share/check_mk/checks/alarm_core_health`):

```python
def inventory_alarm_core_health(info):
    for line in info:
        yield line[1], None  # Service-Name aus Zeile

def check_alarm_core_health(item, params, info):
    for line in info:
        if line[1] == item:
            state = int(line[0])
            detail = " ".join(line[2:])
            return state, detail
    return 3, "Service nicht gefunden (UNKNOWN)"

check_info["alarm_core_health"] = {
    "inventory_function": inventory_alarm_core_health,
    "check_function":     check_alarm_core_health,
    "service_description": "%s",
}
```

---

### 9.3 Custom Check: Letzter Alarm-Eingang

Erkennt, wenn seit längerer Zeit kein Alarm-Event mehr verarbeitet wurde – was auf einen hängenden Worker (z. B. IMAP-Poller, DME-Listener) hinweisen kann.

> **Hinweis:** In Schwachlastzeiten (Nacht, Wochentage ohne Einsatz) ist kein Alarm normal. Der Check ist daher als **passiver Frühindikator** gedacht, nicht als harter Alarm.

```bash
#!/bin/bash
# /usr/lib/check_mk_agent/plugins/alarm_core_last_event

# Letzten Alarm-Eingang aus der DB abfragen
LAST_EVENT=$(psql -U alarmcore -d alarmcore -At -c \
    "SELECT EXTRACT(EPOCH FROM (NOW() - MAX(eingang_um)))::INT
     FROM alarm_log
     WHERE status = 'AUSGELOEST';" 2>/dev/null)

echo "<<<alarm_core_last_event>>>"

if [ -z "$LAST_EVENT" ]; then
    echo "1 Alarm-Core_LetzterAlarm sekunden=U Kein Alarm-Eintrag in der Datenbank (WARN)"
    exit 0
fi

# Schwellwerte: WARN ab 24h, CRIT ab 72h (konfigurierbar)
WARN_SEC=86400
CRIT_SEC=259200

if   [ "$LAST_EVENT" -ge "$CRIT_SEC" ]; then STATE=2; LABEL="CRIT"
elif [ "$LAST_EVENT" -ge "$WARN_SEC" ]; then STATE=1; LABEL="WARN"
else                                          STATE=0; LABEL="OK"
fi

STUNDEN=$(( LAST_EVENT / 3600 ))
echo "$STATE Alarm-Core_LetzterAlarm sekunden=$LAST_EVENT;$WARN_SEC;$CRIT_SEC \
$LABEL: Letzter Alarm vor ${STUNDEN}h"
```

---

### 9.4 Journald-Log-Überwachung via Check_MK

Check_MK kann `journald`-Logs direkt überwachen. Dazu wird in der WATO-Konfiguration eine **Log-Watch-Regel** für die Unit `alarm-core.service` angelegt.

**`/etc/check_mk/logwatch.cfg` auf dem Host:**
```ini
[/var/log/alarm-core/alarm-core.log]
 C "level":"ERROR"
 C "level":"FATAL"
 W "level":"WARN"
 I "level":"INFO"
```

Alternativ direkt über den journald-Plugin-Mechanismus:
```bash
# /etc/check_mk/mk_logwatch.cfg
[journal]
 unit alarm-core.service
 C "level":"ERROR"
 C "level":"FATAL"
 W "level":"WARN"
```

Check_MK sammelt dann alle ERROR/FATAL-Einträge als **CRIT**-Events im Event-Log, WARN-Einträge als **WARN** – ohne dass ein Admin permanent in den Logs suchen muss.

---

### 9.5 Übersicht: Check_MK-Services nach dieser Konfiguration

| Service-Name | Quelle | WARN wenn | CRIT wenn |
|---|---|---|---|
| `Alarm-Core_Erreichbarkeit` | `/health`-Endpunkt | — | Endpunkt nicht antwortend |
| `Alarm-Core_DB` | `/health` → `db` | — | DB-Status nicht `ok` |
| `Alarm-Core_ASTERISK` | `/health` → `asterisk` | — | AMI nicht erreichbar |
| `Alarm-Core_SMSEAGLE` | `/health` → `smseagle` | — | SMS-API nicht erreichbar |
| `Alarm-Core_SERIAL` | `/health` → `serial` | — | Serieller Port inaktiv |
| `Alarm-Core_LetzterAlarm` | PostgreSQL | > 24h kein Alarm | > 72h kein Alarm |
| `Alarm-Core_Log` | journald logwatch | `WARN`-Einträge | `ERROR`/`FATAL`-Einträge |

---

## 10. Erweiterungen (Roadmap)

| Feature | Beschreibung | Priorität |
|---|---|---|
| **Quittierung** | Kameraden bestätigen per SMS-Antwort oder App: "Komme / Komme nicht" | Hoch |
| **Eskalation** | Wenn nach X Minuten < N Quittierungen eingegangen: automatisch nächste Gruppe alarmieren | Hoch |
| **Web-UI** | Administrationsoberfläche für Regeln, Gruppen, Personenverwaltung | Mittel |
| **Push-Notification** | Zusätzlicher Kanal über Firebase (Android) / APNs (iOS) | Mittel |
| **Einsatz-Protokoll** | Übersicht aller Alarme mit Quittierungsquote pro Einsatz, Export als PDF | Niedrig |

---

## 11. Offene Punkte

- **Authentifizierung am REST-Endpoint:** Mindestens API-Key-Validierung, besser mTLS, um Fremdalarmierung zu verhindern.
- **TTS vs. Sprachdateien:** Asterisk kann Text-to-Speech-Module (z. B. `Festival`) nutzen oder feste `.wav`-Ansagedateien abspielen – Entscheidung steht aus.
- **Datenschutz:** Rufnummern und Personendaten in der DB müssen verschlüsselt oder hinter Datenbankzugriffsrollen abgesichert werden (DSGVO).
- **Deployment:** Ob als systemd-Service, Docker-Container oder direkt auf dem Leitstellen-PC – noch nicht definiert.
- **Auswertungs-UI für Vorgesetzte:** Die SQL-Queries aus Kapitel 8.2 sind Grundlage; ein einfaches schreibgeschütztes Web-Dashboard (z. B. Grafana mit PostgreSQL-Datasource) wäre der nächste Schritt.