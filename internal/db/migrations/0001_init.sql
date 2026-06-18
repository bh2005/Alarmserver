-- Initial schema for the alarm-core engine (generic, channel-agnostic).

CREATE TABLE recipients (
    id          SERIAL PRIMARY KEY,
    first_name  VARCHAR(100) NOT NULL,
    last_name   VARCHAR(100) NOT NULL,
    voice       VARCHAR(30),           -- phone number for voice/SIP channel
    mobile      VARCHAR(30),           -- phone number for SMS channel
    email       VARCHAR(255),          -- address for the email channel
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A recipient can belong to multiple groups (e.g. "on-call" and "leadership").
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

-- Routing table: each row is one "if-then" condition. No rule = no behavior
-- to deploy in code; everything is admin-configurable.
CREATE TABLE rules (
    id              SERIAL PRIMARY KEY,
    source          VARCHAR(50)  NOT NULL, -- e.g. "WEBHOOK", "EMAIL", "SERIAL", "MANUAL"
    keyword         VARCHAR(100) NOT NULL,
    -- time window: NULL = always applies
    start_hour      SMALLINT,              -- 0-23, NULL = no limit
    end_hour        SMALLINT,              -- 0-23, NULL = no limit
    weekdays_only   BOOLEAN NOT NULL DEFAULT FALSE,
    target_group_id INT NOT NULL REFERENCES groups(id),
    channels        TEXT[] NOT NULL,       -- e.g. ARRAY['voice', 'sms']
    priority        SMALLINT NOT NULL DEFAULT 10, -- lower value = higher priority
    active          BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_rules_lookup ON rules (source, keyword, active);

-- Audit log: every incoming event is recorded in full, regardless of outcome.
CREATE TABLE event_log (
    id            SERIAL PRIMARY KEY,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source        VARCHAR(50)  NOT NULL,
    keyword       VARCHAR(100) NOT NULL,
    raw_payload   JSONB,                -- original payload of the input channel
    rule_id       INT REFERENCES rules(id),
    target_group  VARCHAR(100),
    channels      TEXT[],
    status        VARCHAR(20) NOT NULL DEFAULT 'DISPATCHED', -- DISPATCHED | ERROR | NO_MATCH
    error_detail  TEXT
);

CREATE INDEX idx_event_log_received ON event_log (received_at DESC);
