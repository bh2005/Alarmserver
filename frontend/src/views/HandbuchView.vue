<template>
  <div class="max-w-4xl mx-auto py-8 px-6">

    <!-- TOC + Content layout -->
    <div class="flex gap-8 items-start">

      <!-- Sticky TOC (hidden on small screens) -->
      <aside class="hidden xl:block w-56 flex-shrink-0 sticky top-6">
        <div class="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
          <div class="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-3">Inhalt</div>
          <nav class="space-y-1">
            <a v-for="s in sections" :key="s.id" :href="'#' + s.id"
              class="block text-xs text-slate-600 dark:text-slate-400 hover:text-ks-600 dark:hover:text-ks-400 py-0.5 transition-colors truncate"
              :class="s.level === 2 ? 'pl-3 text-[11px]' : 'font-medium'">
              {{ s.title }}
            </a>
          </nav>
        </div>
        <div class="mt-3 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 space-y-2">
          <a href="/api/docs" target="_blank"
            class="flex items-center gap-2 text-xs text-ks-600 dark:text-ks-400 hover:underline">
            <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
            </svg>
            Swagger / API Docs
          </a>
          <a href="/api/openapi.yaml" target="_blank"
            class="flex items-center gap-2 text-xs text-ks-600 dark:text-ks-400 hover:underline">
            <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            OpenAPI YAML
          </a>
        </div>
      </aside>

      <!-- Main content -->
      <div class="flex-1 min-w-0 handbook-content" v-html="renderedManual" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { marked } from 'marked'

const renderedManual = ref('')

const sections = [
  { id: 'uebersicht',          title: 'Systemübersicht',          level: 1 },
  { id: 'architektur',         title: 'Architektur',              level: 2 },
  { id: 'quickstart',          title: 'Quickstart',               level: 1 },
  { id: 'regeln',              title: 'Regeln',                   level: 1 },
  { id: 'zeitplan',            title: 'Zeitgesteuerter Plan',     level: 2 },
  { id: 'eskalation',          title: 'Eskalation',               level: 2 },
  { id: 'wiederholung',        title: 'Wiederhol-Benachrichtigung', level: 2 },
  { id: 'gruppen',             title: 'Gruppen & Empfänger',      level: 1 },
  { id: 'selbstverwaltung',    title: 'Selbstverwaltung',         level: 2 },
  { id: 'kanaele',             title: 'Kanäle',                   level: 1 },
  { id: 'voice',               title: 'Voice (Asterisk)',         level: 2 },
  { id: 'smseagle',            title: 'SMS / SMSEagle',           level: 2 },
  { id: 'email',               title: 'E-Mail (SMTP)',            level: 2 },
  { id: 'teams',               title: 'MS Teams',                 level: 2 },
  { id: 'ldap',                title: 'LDAP / Active Directory',  level: 2 },
  { id: 'wartungsfenster',     title: 'Wartungsfenster',          level: 1 },
  { id: 'heartbeats',          title: 'Heartbeat-Überwachung',    level: 1 },
  { id: 'templates',           title: 'Nachrichten-Templates',    level: 1 },
  { id: 'dashboard',           title: 'Dashboard / NOC',          level: 1 },
  { id: 'statistiken',         title: 'Statistiken',              level: 1 },
  { id: 'apikeys',             title: 'API Keys',                 level: 1 },
  { id: 'benutzer',            title: 'Benutzer & Rollen',        level: 1 },
  { id: 'api',                 title: 'API-Referenz',             level: 1 },
  { id: 'webhook',             title: 'Webhook-Ingestion',        level: 2 },
  { id: 'quittierung',         title: 'Quittierung',              level: 2 },
  { id: 'heartbeat-ping',      title: 'Heartbeat Ping',           level: 2 },
  { id: 'faq',                 title: 'FAQ',                      level: 1 },
]

const MANUAL = `# Alarm-Core — Benutzerhandbuch

> Version 1.0 · Stand Juni 2026 · Sprache: Deutsch

---

## Systemübersicht {#uebersicht}

**Alarm-Core** ist ein flexibles Alarmierungs- und Benachrichtigungssystem für technische Betriebsumgebungen. Es empfängt Ereignisse (Events) von beliebigen Quellen über eine REST-API, ordnet sie einem Regelwerk zu und leitet sie über konfigurierte Kanäle (Sprache, SMS, E-Mail, Teams) an die zuständigen Personengruppen weiter.

Das System ist bewusst **quellenagnostisch**: Nagios, Zabbix, Grafana, eigene Skripte oder manuelle Auslösung — jede Integration, die einen HTTP-POST senden kann, ist kompatibel.

### Kernfunktionen

- **Regelbasiertes Routing** — Quelle + Stichwort → Gruppe + Kanäle
- **Zeitgesteuerter Gruppenplan** — Tag- und Nachtschicht, Bereitschaft je Wochentag
- **Eskalation** — automatischer Rückfall auf andere Kanäle bei Nicht-Antwort
- **Wiederhol-Benachrichtigung** — erneuter Alarm bis Quittierung
- **Wartungsfenster** — geplante Unterdrückung
- **Heartbeat-Monitoring** — Ausfallerkennung externer Systeme
- **Nachrichten-Templates** — kanalspezifische Formatierung
- **Dashboard / NOC-Ansicht** — Echtzeit-Überblick
- **Statistiken** — Auswertung der letzten 30 Tage
- **OpenAPI / Swagger** — vollständig dokumentierte REST-API

### Architektur {#architektur}

\`\`\`
Externe Quelle             Alarm-Core Backend (Go)          Benachrichtigungskanäle
──────────────             ──────────────────────────        ──────────────────────
Nagios/Zabbix ──┐          ┌──────────────────────┐         ┌─ Asterisk (Voice)
eigenes Script──┤ HTTP POST │   Webhook-Endpoint   │         ├─ SMSEagle (SMS/Voice)
Grafana Alert ──┤ /api/events│   ↓ RuleEngine       │  Notify ├─ SMTP (E-Mail)
Heartbeat-Cron─┘           │   ↓ Processor        │────────►├─ Teams Webhook
                           │   ↓ Notifiers        │         └─ (weitere)
                           │   ↓ AuditStore       │
                           │     PostgreSQL        │
                           └──────────────────────┘
                                    ↑
                           Vue 3 Frontend (Admin-UI)
\`\`\`

---

## Quickstart {#quickstart}

### Erster Start

1. **Admin-Account**: Beim ersten Start wird automatisch ein Admin aus den Umgebungsvariablen \`ALARM_ADMIN_USERNAME\` / \`ALARM_ADMIN_PASSWORD\` angelegt. Standardwerte: \`admin\` / \`admin123\` — **sofort ändern!**
2. **Kanal konfigurieren**: Admin → Kanäle → gewünschten Kanal aktivieren und Zugangsdaten eingeben.
3. **Empfänger anlegen**: Admin → Gruppen → Empfänger → Neue Person mit Rufnummer/E-Mail.
4. **Gruppe erstellen**: Admin → Gruppen → Person als Mitglied hinzufügen.
5. **Regel anlegen**: Admin → Regeln → Quelle + Stichwort + Zielgruppe + Kanäle.
6. **Test-Event senden**:
\`\`\`bash
curl -X POST http://alarm-core:8080/api/events \\
  -H "X-API-Key: <api-key>" \\
  -H "Content-Type: application/json" \\
  -d '{"source":"TEST","keyword":"PING","text":"Testmeldung"}'
\`\`\`
7. **Event-Log prüfen**: Hat die Regel gematcht? Status \`DISPATCHED\` bedeutet Erfolg.

---

## Regeln {#regeln}

Regeln sind das Herzstück des Systems. Jede Regel definiert:

- **Wann** ein Event passt (Quelle, Stichwort, Zeitfenster, Wochentag)
- **Wen** zu alarmieren ist (Zielgruppe, ggf. zeitgesteuert)
- **Wie** alarmiert wird (Kanäle)
- **Was** bei Nicht-Antwort passiert (Eskalation, Wiederholung)

### Matching

Ein eingehendes Event passt genau dann zu einer Regel, wenn:
1. \`event.source == rule.source\` (Groß-/Kleinschreibung beachten)
2. \`event.keyword == rule.keyword\`
3. Die aktuelle Uhrzeit im Zeitfenster liegt (falls konfiguriert)
4. Es sich um einen Werktag handelt (falls "Nur Mo–Fr" aktiv)

Passen mehrere Regeln, gewinnt die mit dem **niedrigsten Prioritätswert** (z.B. 1 vor 10).

Regeln werden alle **60 Sekunden** automatisch neu geladen — keine Neustart nötig.

### Zeitgesteuerter Gruppenplan {#zeitplan}

Mit dem Zeitplan können verschiedene Gruppen je nach Tageszeit und Wochentag alarmiert werden:

\`\`\`
Mo–Fr  06:00–18:00 → Gruppe "Tagschicht"
Mo–Fr  18:00–06:00 → Gruppe "Nachtschicht"
Sa–So  ganztags    → Gruppe "Bereitschaft"
Fallback            → Gruppe "Leitung"
\`\`\`

Konfiguration in der Regelansicht unter **Zeitgesteuerter Gruppenplan** → mehrere Einträge mit Wochentagen, Von-Bis und Zielgruppe. Passt kein Eintrag, wird die Standardzielgruppe der Regel verwendet.

**Mitternacht-Wrap**: Fenster von 18:00 bis 06:00 funktioniert korrekt (Von > Bis = Nacht-Wrap).

### Eskalation {#eskalation}

Nur für den \`voice\`-Kanal (Sprachanruf über Asterisk): Wenn Empfänger nicht abheben, können nach einer Wartezeit weitere Kanäle ausgelöst werden.

Konfiguration in der Regel:
- **Eskalationskanäle**: z.B. \`sms\` (alle nicht-antwortenden bekommen eine SMS)
- **Wartezeit**: Sekunden, nach denen die Eskalation startet (0 = sofort nach Abschluss der Anrufe)

### Wiederhol-Benachrichtigung {#wiederholung}

Wenn **Quittierung erwarten** aktiv ist, kann eine automatische Wiederholung konfiguriert werden:

- **Interval** (Sekunden): Wie lange nach dem ersten Alarm vor der Wiederholung gewartet wird
- **Max. Wiederholungen**: Wie oft maximal wiederholt wird (1–20)

Wird zwischen zwei Wiederholungen quittiert, stoppt der Prozess sofort. Die Wiederholung läuft als Hintergrundprozess — ein Neustart des Servers beendet laufende Wiederholungen.

---

## Gruppen & Empfänger {#gruppen}

### Empfänger
Ein Empfänger entspricht einer realen Person mit Kontaktdaten:

| Feld | Verwendung |
|---|---|
| **Name** | Anzeige im UI |
| **Voice** | SIP-Rufnummer oder interne Durchwahl für Asterisk |
| **Mobile** | Mobilnummer für SMSEagle (Format: +4915...) |
| **E-Mail** | Adresse für SMTP-Kanal |
| **Aktiv** | Inaktive Empfänger werden nicht alarmiert |

Nur die Felder befüllen, die für die genutzten Kanäle benötigt werden.

### Gruppen
Eine Gruppe ist ein logisches Konstrukt ohne eigene Kontaktdaten. Empfänger werden als Mitglieder hinzugefügt.

**Mitgliedschaft**: Empfänger können in mehreren Gruppen gleichzeitig Mitglied sein.

### Selbstverwaltung {#selbstverwaltung}

Gruppen können als **Selbstverwaltung erlaubt** konfiguriert werden. Dann können Mitglieder im Bereich "Mein Einsatz" ihren eigenen Aktiv-Status setzen — z.B. für Bereitschafts-Rotationen ohne Admin-Eingriff.

Voraussetzung: Der Benutzer-Account muss mit einem Empfänger verknüpft sein (Admin → Benutzer → Empfänger-Dropdown).

---

## Kanäle {#kanaele}

Kanäle werden unter **Admin → Kanäle** konfiguriert. Nicht aktivierte Kanäle erzeugen eine Warnung im Log, wenn eine Regel sie referenziert.

### Voice — Asterisk {#voice}

Verbindung zur Asterisk-PBX über das AMI-Protokoll (Asterisk Manager Interface).

| Parameter | Beschreibung |
|---|---|
| Host | Hostname/IP der Asterisk-Instanz |
| Port | AMI-Port (Standard: 5038) |
| AMI-User | Benutzername aus \`/etc/asterisk/manager.conf\` |
| Passwort | AMI-Passwort |
| Announcement | Name der Sound-Datei (ohne Erweiterung), z.B. \`alarm-ansage\` |
| Channel Template | Asterisk-Channel-Format, z.B. \`SIP/{mobile}@provider\` |

**TTS**: Wenn serverseitig \`TTS_ENABLED=true\` gesetzt ist, wird der Event-Text per \`espeak-ng\` synthetisiert — kein fester Sound nötig. Spracheinstellung: \`TTS_VOICE\` (Standard: \`de\`).

### SMS / Voice — SMSEagle {#smseagle}

SMSEagle-Gerät (Hardware-SMS-Gateway).

| Parameter | Beschreibung |
|---|---|
| Base URL | HTTP-Adresse des Geräts, z.B. \`http://192.168.1.50\` |
| Login | API-Benutzername (wenn kein API-Key) |
| Passwort | API-Passwort |
| API Key | \`access_token\`-basierte Authentifizierung (Alternative zu Login/Passwort) |

SMSEagle bietet zwei Kanäle: \`sms\` (Textnachrichten) und \`voice-smseagle\` (Sprachanruf über Mobilnetz).

### E-Mail — SMTP {#email}

Standard-SMTP mit optionalem STARTTLS.

| Parameter | Beschreibung |
|---|---|
| Host | SMTP-Server-Adresse |
| Port | Standard: 587 (STARTTLS) oder 465 (SSL) |
| Username / Passwort | SMTP-Authentifizierung |
| From | Absenderadresse, z.B. \`alarmserver@example.com\` |

### MS Teams {#teams}

Über einen **Incoming Webhook** werden Nachrichten in einen Teams-Kanal gesendet.

Webhook in Teams einrichten: Kanal → ··· → Connectors → Incoming Webhook → URL kopieren.

### LDAP / Active Directory {#ldap}

Ermöglicht die Anmeldung mit AD-Credentials zusätzlich zu lokalen Konten.

| Parameter | Beschreibung |
|---|---|
| Host | LDAP-Server (Domain Controller) |
| Port | 389 (LDAP) oder 636 (LDAPS) |
| TLS | LDAPS aktivieren |
| Bind-DN | Service-Account für die Suche, z.B. \`CN=svc-alarm,OU=Service,DC=example,DC=com\` |
| Bind-Passwort | Passwort des Service-Accounts |
| User Base | OU, in der nach Benutzern gesucht wird, z.B. \`OU=Users,DC=example,DC=com\` |
| User Filter | LDAP-Filter für den Benutzernamen, z.B. \`(sAMAccountName=%s)\` |
| Standardrolle | Rolle, die LDAP-Benutzern zugeteilt wird: \`admin\` oder \`viewer\` |

**Login-Ablauf**: Lokaler Account → Fehler → LDAP-Bind. Lokale Admin-Accounts funktionieren immer, unabhängig von LDAP-Verfügbarkeit.

---

## Wartungsfenster {#wartungsfenster}

Wartungsfenster unterdrücken Alarme für definierte Zeiträume — z.B. während geplanter Server-Downtime.

### Wirkungsweise

1. Event kommt an
2. Processor prüft: gibt es ein aktives Wartungsfenster für Quelle+Stichwort?
3. Ja → Event wird **nicht** weitergeleitet, Status \`MAINTENANCE\` im Log
4. Nein → normales Regelwerk

### Filterung

| Feld | Leer | Befüllt |
|---|---|---|
| Quelle | Gilt für alle Quellen | Gilt nur für genau diese Quelle |
| Stichwort | Gilt für alle Stichworte | Gilt nur für dieses Stichwort |

Beide leer = alle Events in diesem Zeitraum werden unterdrückt.

### Planung

Wartungsfenster können **im Voraus** angelegt werden. Ein Fenster von 02:00 bis 06:00 kann heute um 09:00 erstellt werden — es wird automatisch aktiv, wenn die Zeit kommt.

---

## Heartbeat-Überwachung {#heartbeats}

Das Heartbeat-System erkennt den Ausfall externer Systeme, die sich nicht selbst melden können.

### Konzept ("Dead Man's Switch")

Externe Systeme senden regelmäßig einen **Ping**. Bleibt der Ping aus, wird ein Alarm erzeugt — das überwachte System meldet also aktiv, dass es noch läuft, statt beim Ausfall etwas zu senden.

### Ping senden

\`\`\`bash
# Mit API-Key
curl -X POST http://alarm-core:8080/api/heartbeat/mein-server \\
  -H "X-API-Key: <api-key>"

# → 204 No Content  (OK)
# → 404 Not Found   (Monitor nicht vorhanden oder deaktiviert)
\`\`\`

**Cronjob-Beispiel** (alle 5 Minuten):
\`\`\`cron
*/5 * * * * curl -sf -X POST http://alarm-core:8080/api/heartbeat/mein-server -H "X-API-Key: <key>" || true
\`\`\`

### Konfiguration

| Parameter | Beschreibung |
|---|---|
| Name | Eindeutiger Bezeichner (wird in der Ping-URL verwendet) |
| Interval (Sek.) | Erwarteter Abstand zwischen Pings, z.B. 300 (5 Min.) |
| Toleranz (Sek.) | Puffer vor Alarmauslösung, z.B. 60 |
| Quelle | \`source\`-Feld des erzeugten Events, z.B. \`heartbeat\` |
| Stichwort | \`keyword\` des Events — muss zu einer Regel passen! |

**Alarm-Zeitpunkt**: \`letzter Ping + Interval + Toleranz < jetzt\`

### Alarmspam-Schutz

Nach einem ausgelösten Alarm wird frühestens nach **1 Stunde** erneut alarmiert, auch wenn der Checker alle 60 Sekunden läuft. Ein eingehender Ping setzt den Timer zurück.

### Statusanzeige

| Status | Bedeutung |
|---|---|
| **ok** | Ping innerhalb der erwarteten Zeit |
| **dead** | Ausfall — Alarm bereits ausgelöst |
| **unknown** | Noch kein einziger Ping empfangen |
| **disabled** | Monitor deaktiviert |

---

## Nachrichten-Templates {#templates}

Templates ermöglichen die kanalspezifische Formatierung von Alarmnachrichten.

### Platzhalter

| Platzhalter | Bedeutung | Beispiel |
|---|---|---|
| \`{text}\` | Originaltext des Events | \`Überdruck Tank 3: 4.2 bar\` |
| \`{source}\` | Quelle des Events | \`ZABBIX\` |
| \`{keyword}\` | Stichwort | \`F2\` |
| \`{time}\` | Empfangszeitpunkt | \`17.06.2026 14:23:11\` |
| \`{group}\` | Alarmierte Gruppe | \`Bereitschaft\` |

### Beispiele

**SMS — kurz und prägnant:**
\`\`\`
[{keyword}] {source} {time}: {text}
\`\`\`
→ \`[F2] ZABBIX 17.06.2026 14:23: Überdruck Tank 3\`

**E-Mail — ausführlich:**
\`\`\`
Alarm-Core Benachrichtigung

Zeitpunkt: {time}
Quelle:    {source}
Stichwort: {keyword}
Gruppe:    {group}

Meldung:
{text}
\`\`\`

**Teams:**
\`\`\`
🚨 **{keyword}** von {source} — {text} _(Gruppe: {group}, {time})_
\`\`\`

### Verhalten ohne Template

Ist kein Template für einen Kanal hinterlegt, wird der **Rohtext** des Events (\`event.text\`) direkt gesendet.

---

## Dashboard / NOC-Ansicht {#dashboard}

Das Dashboard bietet eine Echtzeit-Übersicht über den Alarmzustand.

### Bereiche

**Wartungshinweis**: Gelbes Banner, wenn gerade Wartungsfenster aktiv sind.

**Heartbeat-Statusleiste**: Kompakte Kacheln für alle konfigurierten Monitore.

**Offene Alarme**: Events der letzten 24 Stunden mit Status \`DISPATCHED\` und ohne Quittierung. Das sind die Events, die Aufmerksamkeit erfordern.

**Letzte 20 Ereignisse**: Vollständige Timeline aller Events inkl. Status und Quittierungskennung.

### Aktualisierung

Das Dashboard lädt bei jedem Aufruf frisch — es gibt keine automatische Aktualisierung. Für einen permanenten NOC-Bildschirm empfiehlt sich ein Browser-Auto-Reload-Plugin oder ein Cron-gesteuerter Tab-Refresh.

---

## Statistiken {#statistiken}

Aggregierte Auswertung der letzten **30 Tage** (Zeitzone: Europe/Berlin).

| Kennzahl | Beschreibung |
|---|---|
| Alarme gesamt | Events mit Status \`DISPATCHED\` oder \`MAINTENANCE\` |
| Quittiert | Events mit mindestens einer Quittierung |
| Quittierungsrate | Anteil in Prozent |
| Events pro Tag | Tagesgenaues Balkendiagramm |
| Top Stichworte | Häufigste Keywords (max. 10) |
| Top Quellen | Häufigste Quellen (max. 10) |
| Kanalnutzung | Häufigste Kanäle (max. 10) |

\`NO_MATCH\`-Events (kein Regelwerk gefunden) werden in der Statistik **nicht** gezählt.

---

## API Keys {#apikeys}

API Keys ermöglichen den Zugriff auf Machine-to-Machine-Endpoints ohne Session-Cookie.

### Erstellen

1. Admin → API Keys → **+ Neu**
2. Name eingeben (z.B. \`Zabbix-Integration\`)
3. Den angezeigten Key sofort kopieren — er wird **nie wieder angezeigt**

### Verwenden

\`\`\`bash
# Header
curl -H "X-API-Key: alc_xxxxxxxxxxxx" http://alarm-core:8080/api/events ...

# Query-Parameter
curl "http://alarm-core:8080/api/heartbeat/srv01?api_key=alc_xxxxxxxxxxxx" -X POST
\`\`\`

### Widerruf

Deaktivierung: Key bleibt vorhanden aber wird abgewiesen → gut für temporäre Sperrung.
Löschen: unwiderruflich — zuerst neuen Key in Clients hinterlegen.

---

## Benutzer & Rollen {#benutzer}

### Rollen

| Rolle | Bereiche |
|---|---|
| **admin** | Alles: Regeln, Gruppen, Empfänger, Kanäle, API Keys, Benutzer, Wartung, Heartbeats, Templates, Statistiken |
| **viewer** | Event-Log lesen, eigene Bereitschaft setzen (Mein Einsatz) |

### Empfänger-Verknüpfung

Damit ein Benutzer unter "Mein Einsatz" seine Bereitschaft selbst setzen kann, muss sein Account mit einem **Empfänger** verknüpft sein.

Einrichten: Admin → Benutzer → Empfänger-Dropdown rechts neben dem Benutzernamen → Empfänger wählen.

### LDAP-Benutzer

Beim ersten LDAP-Login wird kein lokaler Account erstellt — die Authentifizierung läuft direkt gegen AD. Die Rolle wird aus dem Feld **Standardrolle** in der LDAP-Konfiguration übernommen.

---

## API-Referenz {#api}

Die vollständige API-Dokumentation ist interaktiv erreichbar unter:

- **Swagger UI**: [/api/docs](/api/docs) (im Browser öffnen)
- **OpenAPI YAML**: [/api/openapi.yaml](/api/openapi.yaml)

### Webhook-Ingestion {#webhook}

\`\`\`
POST /api/events
X-API-Key: <key>
Content-Type: application/json

{
  "source":  "ZABBIX",
  "keyword": "F2",
  "text":    "Host server01 ist nicht erreichbar"
}
\`\`\`

| Feld | Pflicht | Beschreibung |
|---|---|---|
| \`source\` | Ja | Quelle — muss exakt zur Regel passen |
| \`keyword\` | Ja | Stichwort — muss exakt zur Regel passen |
| \`text\` | Nein | Freitext für Ansage und Notifications |

Antwort: \`202 Accepted\` (asynchrone Verarbeitung)

### Quittierung {#quittierung}

\`\`\`
POST /api/acknowledgements
X-API-Key: <key>
Content-Type: application/json

{
  "event_log_id": 42,
  "recipient_id": 7,
  "response":     "KOMME",
  "channel":      "voice"
}
\`\`\`

Wird typischerweise vom Asterisk-AGI-Skript oder einem DTMF-Handler aufgerufen, wenn ein Angerufener \`1\` (Komme) oder \`2\` (Komme nicht) drückt.

### Heartbeat Ping {#heartbeat-ping}

\`\`\`
POST /api/heartbeat/{name}
X-API-Key: <key>

→ 204  Ping erfolgreich registriert
→ 404  Monitor nicht gefunden oder inaktiv
\`\`\`

---

## FAQ {#faq}

**Regel matcht nicht — warum?**
- Groß-/Kleinschreibung von Quelle und Stichwort prüfen (case-sensitive)
- Zeitfenster und Wochentag-Einschränkung prüfen
- Event-Log zeigt \`NO_MATCH\` und den konkreten Fehler (kein Regelwerk für source=X keyword=Y)
- Regeln werden erst nach bis zu 60 Sekunden aktiv

**Kanal wird nicht ausgelöst — warum?**
- Kanal in Admin → Kanäle aktiviert und konfiguriert?
- Log des Backends prüfen (\`go run ./cmd/alarm-core/ 2>&1\`): \`no_notifier_for_channel\` → Kanal fehlt in der Konfiguration

**Heartbeat-Alarm kommt, obwohl System läuft?**
- Läuft der Cron-Job auf dem überwachten Server?
- Ist der API-Key gültig und aktiv?
- Ist der Monitor-Name im Ping-URL korrekt (exakt gleich wie konfiguriert)?
- Netzwerk: kann der Server den Alarm-Core erreichen?

**LDAP-Login schlägt fehl?**
- Bind-DN und Passwort per \`ldapwhoami\` manuell testen
- User-Filter mit \`ldapsearch\` validieren: enthält er \`%s\`?
- TLS: Zertifikat des LDAP-Servers vertrauenswürdig?

**Wiederholung stoppt nicht nach Quittierung?**
- Quittierung muss über \`POST /api/acknowledgements\` mit der korrekten \`event_log_id\` gesendet werden
- ID ist im Event-Log sichtbar

**Wo sehe ich Backend-Fehler?**
Backend schreibt strukturiertes JSON-Log auf stdout. Im Dev-Betrieb:
\`\`\`bash
go run ./cmd/alarm-core/ 2>&1 | tee alarm.log
\`\`\`
Im Docker-Betrieb:
\`\`\`bash
docker logs -f alarm-core
\`\`\`
`

onMounted(() => {
  // Pre-process markdown: replace  ## Title {#id}  with actual HTML heading tags
  // so anchor links from the TOC work correctly.
  const withIds = MANUAL.replace(
    /^(#{1,6})\s+(.*?)\s*\{#([\w-]+)\}\s*$/gm,
    (_, hashes, title, id) => {
      const level = hashes.length
      return `<h${level} id="${id}">${title}</h${level}>`
    },
  )
  renderedManual.value = marked.parse(withIds)
})
</script>

<style>
.handbook-content h1 {
  font-size: 1.5rem;
  font-weight: 800;
  color: #0f2557;
  margin-top: 2.5rem;
  margin-bottom: 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid #e2e8f0;
  scroll-margin-top: 1.5rem;
}
.handbook-content h1:first-child { margin-top: 0; }
.handbook-content h2 {
  font-size: 1.125rem;
  font-weight: 700;
  color: #173b7a;
  margin-top: 2rem;
  margin-bottom: 0.5rem;
  scroll-margin-top: 1.5rem;
}
.handbook-content h3 {
  font-size: 0.9375rem;
  font-weight: 600;
  color: #1e40af;
  margin-top: 1.5rem;
  margin-bottom: 0.375rem;
  scroll-margin-top: 1.5rem;
}
.handbook-content p  { font-size: 0.875rem; color: #374151; margin-bottom: 0.75rem; line-height: 1.7; }
.handbook-content ul,
.handbook-content ol { margin-left: 1.25rem; margin-bottom: 0.75rem; }
.handbook-content ul { list-style-type: disc; }
.handbook-content ol { list-style-type: decimal; }
.handbook-content li { font-size: 0.875rem; color: #374151; margin-bottom: 0.3rem; line-height: 1.6; }
.handbook-content strong { font-weight: 600; color: #111827; }
.handbook-content em { font-style: italic; color: #4b5563; }
.handbook-content a { color: #1d4ed8; text-decoration: underline; }
.handbook-content a:hover { color: #1e40af; }
.handbook-content code {
  background: #f1f5f9;
  color: #0f2557;
  padding: 0.15em 0.4em;
  border-radius: 0.25rem;
  font-size: 0.8125rem;
  font-family: ui-monospace, 'Cascadia Code', Menlo, monospace;
}
.handbook-content pre {
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 0.625rem;
  padding: 1rem;
  overflow-x: auto;
  margin-bottom: 1rem;
  font-size: 0.8125rem;
  line-height: 1.6;
}
.handbook-content pre code {
  background: transparent;
  color: inherit;
  padding: 0;
  font-size: inherit;
}
.handbook-content table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1rem;
  font-size: 0.875rem;
}
.handbook-content th {
  background: #f8fafc;
  padding: 0.5rem 0.75rem;
  text-align: left;
  font-weight: 600;
  color: #1e293b;
  border-bottom: 2px solid #e2e8f0;
  border-top: 1px solid #e2e8f0;
}
.handbook-content td {
  padding: 0.5rem 0.75rem;
  color: #374151;
  border-bottom: 1px solid #f1f5f9;
  vertical-align: top;
}
.handbook-content tr:hover td { background: #f8fafc; }
.handbook-content hr {
  border: none;
  border-top: 1px solid #e2e8f0;
  margin: 2rem 0;
}
.handbook-content blockquote {
  border-left: 3px solid #3b82f6;
  margin: 0 0 1rem;
  padding: 0.5rem 1rem;
  background: #eff6ff;
  border-radius: 0 0.375rem 0.375rem 0;
}
.handbook-content blockquote p { color: #1e40af; margin: 0; font-size: 0.8125rem; }

/* Dark mode */
.dark .handbook-content h1 { color: #bfdbfe; border-bottom-color: #1e293b; }
.dark .handbook-content h2 { color: #93c5fd; }
.dark .handbook-content h3 { color: #60a5fa; }
.dark .handbook-content p  { color: #94a3b8; }
.dark .handbook-content li { color: #94a3b8; }
.dark .handbook-content strong { color: #e2e8f0; }
.dark .handbook-content em { color: #94a3b8; }
.dark .handbook-content a  { color: #60a5fa; }
.dark .handbook-content code { background: #1e293b; color: #93c5fd; }
.dark .handbook-content pre  { background: #020617; }
.dark .handbook-content pre code { color: #e2e8f0; }
.dark .handbook-content th { background: #1e293b; color: #cbd5e1; border-color: #334155; }
.dark .handbook-content td { color: #94a3b8; border-bottom-color: #1e293b; }
.dark .handbook-content tr:hover td { background: #1e293b; }
.dark .handbook-content hr { border-top-color: #1e293b; }
.dark .handbook-content blockquote { background: #1e3a5f; border-left-color: #60a5fa; }
.dark .handbook-content blockquote p { color: #93c5fd; }
</style>
