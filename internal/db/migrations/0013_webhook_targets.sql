-- Generic outgoing HTTP webhook channel.
-- The "name" field becomes the channel name used in rules (e.g. "slack", "servicenow").
CREATE TABLE webhook_targets (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(50)  NOT NULL UNIQUE,
    label           VARCHAR(100) NOT NULL,
    url             TEXT NOT NULL,
    method          VARCHAR(10)  NOT NULL DEFAULT 'POST',
    headers         JSONB        NOT NULL DEFAULT '{}',
    body_template   TEXT,   -- placeholders: {message} {keyword} {source} {group} {time}
    timeout_sec     INT          NOT NULL DEFAULT 10,
    active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
