package api

import (
	"net/http"
)

// HandleOpenAPISpec serves the OpenAPI 3.0 specification as YAML.
func HandleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(openAPIYAML))
}

// HandleSwaggerUI serves a minimal Swagger UI page that loads the spec from
// the same host.
func HandleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Alarm-Core API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: "/api/openapi.yaml",
  dom_id: '#swagger-ui',
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
  layout: "BaseLayout",
  deepLinking: true,
  tryItOutEnabled: true
})
</script>
</body>
</html>`

const openAPIYAML = `openapi: "3.0.3"
info:
  title: Alarm-Core API
  description: |
    REST API for the Alarm-Core notification routing server.
    Endpoints require either a session token (Cookie: session=...) from /auth/login
    or an API key (X-API-Key header or ?api_key= query param).
  version: "1.0.0"

servers:
  - url: /

tags:
  - name: Auth
  - name: Events
  - name: Rules
  - name: Groups
  - name: Recipients
  - name: Channels
  - name: APIKeys
  - name: Maintenance
  - name: Heartbeats
  - name: Templates
  - name: Dashboard
  - name: Stats
  - name: Users

paths:
  /auth/login:
    post:
      tags: [Auth]
      summary: Obtain a session token
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [username, password]
              properties:
                username: {type: string}
                password: {type: string}
      responses:
        "200":
          description: Login successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  token: {type: string}
                  role:  {type: string, enum: [admin, viewer]}
        "401":
          description: Invalid credentials

  /auth/me:
    get:
      tags: [Auth]
      summary: Return the currently authenticated user
      security: [{session: []}]
      responses:
        "200":
          description: Current user info

  /api/events:
    post:
      tags: [Events]
      summary: Ingest an alarm event (machine-to-machine)
      security: [{apiKey: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [source, keyword]
              properties:
                source:  {type: string}
                keyword: {type: string}
                text:    {type: string}
      responses:
        "202": {description: Accepted}
        "400": {description: Bad request}

  /api/events/log:
    get:
      tags: [Events]
      summary: List recent events with acknowledgement status
      security: [{session: []}]
      parameters:
        - name: limit
          in: query
          schema: {type: integer, default: 100}
      responses:
        "200":
          description: Event log entries
          content:
            application/json:
              schema:
                type: object
                properties:
                  events:
                    type: array
                    items: {$ref: '#/components/schemas/EventLogEntry'}

  /api/dashboard:
    get:
      tags: [Dashboard]
      summary: NOC dashboard — recent events, open alarms, maintenance, heartbeats
      security: [{session: []}]
      responses:
        "200":
          description: Dashboard data
          content:
            application/json:
              schema:
                type: object
                properties:
                  recent_events:      {type: array, items: {$ref: '#/components/schemas/EventLogEntry'}}
                  open_events:        {type: array, items: {$ref: '#/components/schemas/EventLogEntry'}}
                  active_maintenance: {type: array, items: {$ref: '#/components/schemas/MaintenanceWindow'}}
                  heartbeats:         {type: array, items: {$ref: '#/components/schemas/Heartbeat'}}

  /api/stats:
    get:
      tags: [Stats]
      summary: Aggregated statistics for the last 30 days
      security: [{session: []}]
      responses:
        "200":
          description: Statistics
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Stats'}

  /api/rules:
    get:
      tags: [Rules]
      summary: List all rules
      security: [{session: []}]
      responses:
        "200":
          description: Rule list
    post:
      tags: [Rules]
      summary: Create a rule
      security: [{session: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/RuleRecord'}
      responses:
        "201": {description: Created}

  /api/rules/{id}:
    put:
      tags: [Rules]
      summary: Update a rule
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/RuleRecord'}
      responses:
        "204": {description: Updated}
    delete:
      tags: [Rules]
      summary: Delete a rule
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      responses:
        "204": {description: Deleted}

  /api/maintenance:
    get:
      tags: [Maintenance]
      summary: List all maintenance windows
      security: [{session: []}]
      responses:
        "200":
          description: Maintenance windows
    post:
      tags: [Maintenance]
      summary: Create a maintenance window
      security: [{session: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/MaintenanceWindow'}
      responses:
        "201": {description: Created}

  /api/maintenance/{id}:
    put:
      tags: [Maintenance]
      summary: Update a maintenance window
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/MaintenanceWindow'}
      responses:
        "204": {description: Updated}
    delete:
      tags: [Maintenance]
      summary: Delete a maintenance window
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      responses:
        "204": {description: Deleted}

  /api/heartbeats:
    get:
      tags: [Heartbeats]
      summary: List all heartbeat monitors
      security: [{session: []}]
      responses:
        "200":
          description: Heartbeat list
    post:
      tags: [Heartbeats]
      summary: Create a heartbeat monitor
      security: [{session: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Heartbeat'}
      responses:
        "201": {description: Created}

  /api/heartbeats/{id}:
    put:
      tags: [Heartbeats]
      summary: Update a heartbeat monitor
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Heartbeat'}
      responses:
        "204": {description: Updated}
    delete:
      tags: [Heartbeats]
      summary: Delete a heartbeat monitor
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      responses:
        "204": {description: Deleted}

  /api/heartbeat/{name}:
    post:
      tags: [Heartbeats]
      summary: Report a heartbeat ping (machine-to-machine)
      security: [{apiKey: []}]
      parameters:
        - name: name
          in: path
          required: true
          schema: {type: string}
      responses:
        "204": {description: Ping recorded}
        "404": {description: Heartbeat not found}

  /api/templates:
    get:
      tags: [Templates]
      summary: List all message templates
      security: [{session: []}]
      responses:
        "200":
          description: Template list

  /api/templates/{channel}:
    put:
      tags: [Templates]
      summary: Create or update a message template for a channel
      description: |
        Placeholders: {text}, {source}, {keyword}, {time}, {group}
      security: [{session: []}]
      parameters:
        - name: channel
          in: path
          required: true
          schema: {type: string, example: sms}
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [template]
              properties:
                template:
                  type: string
                  example: "ALARM [{keyword}] von {source} um {time}: {text}"
      responses:
        "204": {description: Saved}
    delete:
      tags: [Templates]
      summary: Remove a channel template (revert to raw text)
      security: [{session: []}]
      parameters:
        - name: channel
          in: path
          required: true
          schema: {type: string}
      responses:
        "204": {description: Deleted}

  /api/acknowledgements:
    get:
      tags: [Events]
      summary: List acknowledgements
      security: [{session: []}]
      parameters:
        - name: event_log_id
          in: query
          schema: {type: integer}
      responses:
        "200":
          description: Acknowledgement list
    post:
      tags: [Events]
      summary: Record an acknowledgement
      security: [{apiKey: []}]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [event_log_id, recipient_id]
              properties:
                event_log_id:  {type: integer}
                recipient_id:  {type: integer}
                response:      {type: string}
                channel:       {type: string}
      responses:
        "204": {description: Recorded}

  /api/groups:
    get:
      tags: [Groups]
      summary: List all groups
      security: [{session: []}]
      responses:
        "200":
          description: Group list
    post:
      tags: [Groups]
      summary: Create a group
      security: [{session: []}]
      responses:
        "201": {description: Created}

  /api/recipients:
    get:
      tags: [Recipients]
      summary: List all recipients
      security: [{session: []}]
      responses:
        "200":
          description: Recipient list
    post:
      tags: [Recipients]
      summary: Create a recipient
      security: [{session: []}]
      responses:
        "201": {description: Created}

  /api/config/channels:
    get:
      tags: [Channels]
      summary: List available channel types and their current config
      security: [{session: []}]
      responses:
        "200":
          description: Channel config list

  /api/config/channels/{id}:
    put:
      tags: [Channels]
      summary: Update channel configuration
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: string}
      responses:
        "204": {description: Saved}

  /api/apikeys:
    get:
      tags: [APIKeys]
      summary: List API keys
      security: [{session: []}]
      responses:
        "200":
          description: API key list (secret never returned)
    post:
      tags: [APIKeys]
      summary: Create an API key (secret returned once)
      security: [{session: []}]
      responses:
        "201":
          description: Created — plaintext key in response, store it now

  /api/users:
    get:
      tags: [Users]
      summary: List users
      security: [{session: []}]
      responses:
        "200":
          description: User list

  /api/me/groups:
    get:
      tags: [Users]
      summary: List self-service groups for the current user
      security: [{session: []}]
      responses:
        "200":
          description: Group list with active flag

  /api/me/groups/{id}/active:
    post:
      tags: [Users]
      summary: Toggle own active status in a self-service group
      security: [{session: []}]
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: integer}
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [active]
              properties:
                active: {type: boolean}
      responses:
        "204": {description: Updated}

  /health:
    get:
      summary: Health check
      responses:
        "200":
          description: Service is healthy

components:
  securitySchemes:
    session:
      type: apiKey
      in: cookie
      name: session
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key

  schemas:
    EventLogEntry:
      type: object
      properties:
        id:           {type: integer}
        received_at:  {type: string, format: date-time}
        source:       {type: string}
        keyword:      {type: string}
        rule_id:      {type: integer, nullable: true}
        target_group: {type: string}
        channels:     {type: array, items: {type: string}}
        status:       {type: string, enum: [DISPATCHED, NO_MATCH, ERROR, MAINTENANCE]}
        error_detail: {type: string}
        acknowledged: {type: boolean}

    RuleRecord:
      type: object
      properties:
        id:                     {type: integer, readOnly: true}
        source:                 {type: string}
        keyword:                {type: string}
        start_hour:             {type: integer, nullable: true}
        end_hour:               {type: integer, nullable: true}
        weekdays_only:          {type: boolean}
        target_group_id:        {type: integer}
        channels:               {type: array, items: {type: string}}
        priority:               {type: integer}
        active:                 {type: boolean}
        require_ack:            {type: boolean}
        escalation_channels:    {type: array, items: {type: string}}
        escalation_delay_seconds: {type: integer}
        repeat_interval_sec:    {type: integer, description: "0 = no repeat"}
        repeat_max:             {type: integer, description: "Max repeat attempts, 0 = unlimited until ACK"}
        schedule:
          type: array
          items:
            type: object
            properties:
              days:         {type: array, items: {type: integer, description: "0=Mon … 6=Sun"}}
              from:         {type: string, example: "08:00"}
              to:           {type: string, example: "17:00"}
              target_group: {type: string}

    MaintenanceWindow:
      type: object
      properties:
        id:         {type: integer, readOnly: true}
        label:      {type: string}
        starts_at:  {type: string, format: date-time}
        ends_at:    {type: string, format: date-time}
        source:     {type: string, nullable: true, description: "null = any source"}
        keyword:    {type: string, nullable: true, description: "null = any keyword"}
        created_at: {type: string, format: date-time, readOnly: true}

    Heartbeat:
      type: object
      properties:
        id:           {type: integer, readOnly: true}
        name:         {type: string}
        description:  {type: string}
        interval_sec: {type: integer, default: 300}
        grace_sec:    {type: integer, default: 60}
        source:       {type: string, default: heartbeat}
        keyword:      {type: string}
        last_seen:    {type: string, format: date-time, nullable: true, readOnly: true}
        active:       {type: boolean}
        status:       {type: string, enum: [ok, dead, unknown, disabled], readOnly: true}
        created_at:   {type: string, format: date-time, readOnly: true}

    Stats:
      type: object
      properties:
        total_events:   {type: integer}
        acked_events:   {type: integer}
        events_per_day:
          type: array
          items:
            type: object
            properties:
              date:  {type: string, format: date}
              count: {type: integer}
        top_keywords:
          type: array
          items:
            type: object
            properties:
              name:  {type: string}
              count: {type: integer}
        top_sources:
          type: array
          items:
            type: object
            properties:
              name:  {type: string}
              count: {type: integer}
        channel_usage:
          type: array
          items:
            type: object
            properties:
              name:  {type: string}
              count: {type: integer}
`
