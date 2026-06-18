export const helpTexts = {

  '/dashboard': `
## Dashboard

Die NOC-Übersicht zeigt auf einen Blick den aktuellen Alarmzustand.

### Bereiche
- **Aktive Wartungsfenster**: Gelbes Banner, wenn gerade ein Wartungsfenster aktiv ist. Alarme der betroffenen Quelle/Stichwort-Kombination werden in dieser Zeit unterdrückt.
- **Heartbeat-Status**: Kacheln für jeden konfigurierten Monitor — grün (OK), rot (Ausfall), grau (noch nie ein Ping).
- **Offene Alarme**: Events der letzten 24 Stunden, für die noch keine Quittierung eingegangen ist. Diese sollten bearbeitet werden.
- **Letzte 20 Ereignisse**: Vollständige Timeline mit Status und Quittierungshinweis.

### Aktualisierung
Das Dashboard aktualisiert sich nicht automatisch — Schaltfläche **Aktualisieren** oben rechts oder Seite neu laden. Für automatisches Polling kann der Browser-Refresh-Intervall genutzt werden.
`,

  '/': `
## Regeln

Das Regelwerk entscheidet, wer bei welchem Event alarmiert wird. Jede Regel ist eine "Wenn-Dann"-Bedingung.

### Felder
- **Quelle / Stichwort**: muss exakt zum eingehenden Event passen (\`source\`/\`keyword\` aus \`POST /api/events\`)
- **Zeitfenster**: Start-/End-Stunde (0–23), leer = kein Limit. Fenster über Mitternacht nicht möglich — zwei Regeln anlegen (z.B. 17–24 und 0–6)
- **Nur werktags**: schränkt auf Mo–Fr ein
- **Zielgruppe**: Standardgruppe, die alarmiert wird
- **Kanäle**: \`voice\`, \`sms\`, \`email\`, \`teams\` — nur konfigurierte Kanäle wirken
- **Priorität**: bei mehreren passenden Regeln gewinnt der niedrigste Wert
- **Quittierung erwarten**: markiert, dass Empfänger antworten sollen

### Zeitgesteuerter Gruppenplan
Ermöglicht unterschiedliche Zielgruppen je nach Wochentag/Uhrzeit (z.B. Tagschicht Mo–Fr 6–18 Uhr, Bereitschaft sonst). Fallback ist immer die Standardzielgruppe.

### Eskalation
Wenn ein Sprachanruf nicht angenommen wird, können Eskalationskanäle (z.B. SMS an alle) nach einer Wartezeit ausgelöst werden.

### Wiederhol-Benachrichtigung
Wenn **Quittierung erwarten** aktiv ist, kann ein Wiederholintervall (Sekunden) konfiguriert werden. Der Alarm wird erneut ausgelöst, bis jemand quittiert oder die maximale Anzahl erreicht ist.

### Wirksamkeit
Änderungen werden vom laufenden Service alle **60 Sekunden** automatisch neu geladen.
`,

  '/groups': `
## Gruppen & Empfänger

Verwalte Zielgruppen und die Personen darin.

### Gruppen
Eine Gruppe ist ein Name + Beschreibung. Mitglieder werden separat zugeordnet.

**Selbstverwaltung**: Wenn eine Gruppe als "Selbstverwaltung erlaubt" markiert ist, können Mitglieder sich über **Mein Einsatz** selbst als aktiv/inaktiv setzen (z.B. für Bereitschaft).

### Empfänger
Jede Person kann Rufnummer (\`voice\`), Mobilnummer (\`sms\`) und E-Mail (\`email\`) haben — nur die für die geplanten Kanäle nötigen Felder befüllen. **Inaktive** Empfänger werden bei Alarmierung übersprungen.

### Datenschutz
Rufnummern und Namen sind personenbezogene Daten — dieser Bereich ist nur für Admins sichtbar.
`,

  '/events': `
## Event-Log

Vollständiger Verlauf aller eingegangenen Events — unabhängig davon, ob eine Regel gegriffen hat.

### Status
| Status | Bedeutung |
|---|---|
| \`DISPATCHED\` | Regel gefunden, Kanäle wurden angestoßen |
| \`NO_MATCH\` | Keine passende Regel gefunden |
| \`MAINTENANCE\` | Alarm wurde durch aktives Wartungsfenster unterdrückt |
| \`ERROR\` | Interner Fehler beim Verarbeiten |

### Quittierungen
Das Häkchen in der Spalte **Quittiert** zeigt, ob mindestens eine Quittierung eingegangen ist. Quittierungen laufen kanalunabhängig über \`POST /api/acknowledgements\`.
`,

  '/announcements': `
## Ansagen

Verwaltung der Sound-Dateien für den \`voice\`-Kanal.

### Statische Ansage
Hochgeladene \`.wav\`-Dateien (8 kHz Mono empfohlen) landen direkt im Asterisk-Sound-Verzeichnis.

### TTS (Text-to-Speech)
Ist TTS serverseitig aktiviert (\`TTS_ENABLED=true\`), wird der Alarmtext per \`espeak-ng\` gesprochen — komplett offline, keine Cloud-API.
`,

  '/users': `
## Benutzerverwaltung

Nur für Admins. Zwei Rollen:

| Rolle | Rechte |
|---|---|
| **admin** | Alles — Regeln, Gruppen, Empfänger, Kanäle, Benutzer, API-Keys verwalten |
| **viewer** | Nur Event-Log ansehen |

### Empfänger-Verknüpfung
Jeder Benutzer kann mit einem Empfänger verknüpft werden. Das ist die Voraussetzung für die Selbstverwaltungs-Funktion ("Mein Einsatz") — nur verknüpfte Benutzer können ihre Bereitschaft selbst setzen.

Der erste Admin-Account wird beim ersten Start automatisch aus \`ALARM_ADMIN_USERNAME\`/\`ALARM_ADMIN_PASSWORD\` angelegt.
`,

  '/channels': `
## Kanäle

Konfiguration der Benachrichtigungskanäle. Nicht aktivierte Kanäle werden beim Dispatch ignoriert (Warnung im Log).

### Verfügbare Kanäle
| Kanal | Protokoll | Pflichtfelder |
|---|---|---|
| **voice** | Asterisk AMI | Host, Port, AMI-User, Passwort |
| **sms** / **voice-smseagle** | SMSEagle HTTP | Base-URL + Login/Passwort **oder** API-Key |
| **email** | SMTP | Host, Port, Absender |
| **teams** | Incoming Webhook | Webhook-URL |
| **ldap** | LDAP/AD | Host, Bind-DN, Passwort, User-Base, Filter |

### Sensible Felder
Passwörter und Webhook-URLs werden **nie** im Klartext zurückgegeben — das API liefert \`***\`. Leer lassen beim Speichern erhält den bestehenden Wert.

### LDAP
Wenn LDAP konfiguriert und aktiviert ist, wird bei der Anmeldung zuerst der lokale Benutzer geprüft. Schlägt das fehl, erfolgt ein Fallback auf LDAP/AD. Lokale Admins können sich immer anmelden.
`,

  '/apikeys': `
## API Keys

Maschinenkonten für die Webhook-Ingestion und andere automatisierte Integrationen.

### Wichtig
Der Klartext-Key wird **einmalig** bei der Erstellung angezeigt und danach nie wieder abgerufen — sofort in der Zielanwendung hinterlegen.

### Verwendung
\`\`\`
POST /api/events
X-API-Key: <key>
\`\`\`
oder als Query-Parameter:
\`\`\`
POST /api/events?api_key=<key>
\`\`\`

Deaktivierte Keys werden sofort abgewiesen. Löschen ist unwiderruflich — ggf. neuen Key erstellen und zuerst in den Clients hinterlegen.
`,

  '/maintenance': `
## Wartungsfenster

Unterdrückt Alarme für definierte Zeiträume, z.B. während geplanter Wartungsarbeiten.

### Wirkungsweise
Trifft ein Event in ein aktives Wartungsfenster (und passen Quelle/Stichwort), wird es **nicht** weitergeleitet. Im Event-Log erscheint der Status \`MAINTENANCE\`.

### Filterung
- **Quelle leer**: gilt für alle Quellen
- **Stichwort leer**: gilt für alle Stichworte
- Beide leer: alle Events dieser Zeit werden unterdrückt

### Tipp
Wartungsfenster können auch vorausplanend angelegt werden (z.B. für kommende Nacht) und laufen automatisch ab — kein manuelles Eingreifen nötig.
`,

  '/heartbeats': `
## Heartbeat-Überwachung

Überwacht externe Systeme, die regelmäßig ein Lebenszeichen senden müssen ("Dead Man's Switch").

### Funktionsweise
1. Externes System sendet \`POST /api/heartbeat/{name}\` mit gültigem API-Key
2. Bleibt der nächste Ping nach **Interval + Toleranz** aus, erzeugt Alarm-Core ein Event mit der konfigurierten Quelle und dem Stichwort
3. Dieses Event durchläuft das normale Regelwerk → passende Regel muss existieren!

### Ping-Endpunkt
\`\`\`
POST /api/heartbeat/{name}
X-API-Key: <key>
→ 204 No Content  (OK)
→ 404 Not Found   (Monitor unbekannt oder inaktiv)
\`\`\`

### Status
| Status | Bedeutung |
|---|---|
| **ok** | Letzter Ping innerhalb Interval + Toleranz |
| **dead** | Kein Ping — Alarm bereits ausgelöst |
| **unknown** | Noch kein einziger Ping eingegangen |
| **disabled** | Monitor ist deaktiviert |

### Alarmspam-Schutz
Nach einem Alarm wird frühestens nach **1 Stunde** erneut alarmiert — selbst wenn die Prüfung jede Minute läuft.
`,

  '/templates': `
## Nachrichten-Templates

Kanalspezifische Formatierung der Alarmnachrichten. Ohne Template wird der Rohtext des Events gesendet.

### Platzhalter
| Platzhalter | Bedeutung |
|---|---|
| \`{text}\` | Originaltext des Events |
| \`{source}\` | Quelle (z.B. \`WEBHOOK\`) |
| \`{keyword}\` | Stichwort (z.B. \`F2\`) |
| \`{time}\` | Empfangszeitpunkt (TT.MM.JJJJ HH:MM:SS) |
| \`{group}\` | Name der alarmieren Gruppe |

### Beispiel
**SMS-Template:**
\`\`\`
ALARM [{keyword}] von {source} um {time}: {text} → Gruppe {group}
\`\`\`

**Ergebnis:**
\`\`\`
ALARM [F2] von WEBHOOK um 17.06.2026 14:23:11: Überdruck Tank 3 → Gruppe Bereitschaft
\`\`\`

### Gültige Kanalnamen
\`voice\`, \`voice-smseagle\`, \`sms\`, \`email\`, \`teams\`
`,

  '/stats': `
## Statistiken

Aggregierte Auswertung der letzten **30 Tage**. Nur für Admins.

### Kennzahlen
- **Alarme gesamt**: alle Events mit Status \`DISPATCHED\` oder \`MAINTENANCE\` (kein \`NO_MATCH\`)
- **Quittiert**: Events mit mindestens einer Quittierung
- **Quittierungsrate**: Anteil quittierter Alarme in Prozent

### Diagramme
- **Events pro Tag**: Balkendiagramm — Maus über Balken für genaues Datum/Anzahl
- **Top Stichworte / Quellen**: Häufigkeitsliste mit relativen Balken
- **Kanalnutzung**: welche Kanäle wie oft genutzt wurden

### Nutzung
Statistiken werden bei jedem Aufruf frisch aus der Datenbank berechnet — kein Cache.
`,

  '/me/groups': `
## Mein Einsatz

Hier kannst du deine eigene Bereitschaft in freigegebenen Gruppen selbst setzen, ohne auf einen Admin warten zu müssen.

### Voraussetzungen
1. Dein Benutzerkonto muss mit einem **Empfänger** verknüpft sein (Admin → Benutzer → Empfänger zuweisen)
2. Die Gruppe muss als **Selbstverwaltung erlaubt** konfiguriert sein (Admin → Gruppen)

### Wirkungsweise
- **Aktiv**: Du bist in der Gruppe eingeschaltet und wirst bei Alarmen dieser Gruppe alarmiert
- **Inaktiv**: Du bist im System als Mitglied eingetragen, wirst aber nicht alarmiert

### Hinweis
Inaktive Mitglieder werden beim Dispatch übersprungen — du musst dich selbst rechtzeitig auf **Aktiv** setzen, bevor deine Bereitschaft beginnt.
`,

  '/escalation-policies': `
## Eskalationspolitiken

Definiert, welche Gruppen **automatisch nachbenachrichtigt** werden, wenn ein Alarm innerhalb einer konfigurierten Zeit nicht quittiert wurde.

### Funktionsweise
1. Eine Regel löst einen Alarm aus und benachrichtigt die Primärgruppe.
2. Bleibt die Quittierung aus, startet die Eskalationspolitik im Hintergrund.
3. Jeder Schritt gibt an: **nach wie vielen Sekunden** (absolut ab Event-Eingang) welche **Gruppe** alarmiert wird.
4. Sobald jemand quittiert, werden alle ausstehenden Schritte abgebrochen.

### Schritte konfigurieren
- **Gruppe**: Die Gruppe, die bei diesem Schritt benachrichtigt wird.
- **Nach (Minuten)**: Absoluter Zeitversatz ab Event-Eingang — nicht relativ zum vorherigen Schritt.
- **Kanäle (optional)**: Überschreibt die Kanäle der auslösenden Regel. Leer lassen = Kanäle aus Regel verwenden.

### Zuweisung
Eine Eskalationspolitik wird **pro Regel** zugewiesen (im Formular der Regel ganz unten). Eine Regel ohne zugewiesene Politik eskaliert nicht automatisch.

### Hinweis
Webhook-Kanäle müssen nach dem Anlegen eines neuen Webhook-Ziels durch einen Backend-Neustart registriert werden, bevor sie hier als Kanal nutzbar sind.
`,

  '/webhook-targets': `
## Webhook-Ziele

Outbound-Webhooks ermöglichen die Integration mit externen Systemen wie **Slack**, **Microsoft Teams** (alternativ zum eingebauten Teams-Kanal), **ServiceNow**, **PagerDuty** oder beliebigen REST-APIs.

### Konfiguration
- **Name**: Kanalname — erscheint in Regel- und Eskalations-Konfiguration (z.B. \`slack\`). Nur Kleinbuchstaben, Zahlen, \`_\`, \`-\`.
- **Anzeigename**: Lesbarer Name für die Oberfläche.
- **URL**: Ziel-URL des Webhooks.
- **HTTP-Methode**: Meist \`POST\`. Manche APIs erwarten \`PUT\`.
- **Body-Template**: JSON-Vorlage mit \`{message}\` als Platzhalter für den Alarmtext. Leer = Standard-JSON mit allen Feldern.
- **Zusatz-Header**: z.B. \`{"Authorization": "Bearer token"}\` für gesicherte Endpoints.

### Platzhalter im Body-Template
\`{message}\`, \`{keyword}\`, \`{source}\`, \`{group}\`, \`{time}\`

### Wichtig: Backend-Neustart
Webhook-Ziele werden **einmalig beim Start** des Backends geladen. Nach dem Anlegen oder Ändern eines Ziels ist ein **Neustart des alarm-core** notwendig, damit der neue Kanal aktiv wird.
`,
}

export function getHelp(path) {
  return helpTexts[path] ?? `## ${path}\n\nFür diese Seite gibt es noch keinen speziellen Hilfetext.\n\nDas vollständige **[Handbuch](/handbuch)** enthält alle Informationen.`
}
