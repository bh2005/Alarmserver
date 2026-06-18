CREATE TABLE heartbeats (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    interval_sec INT NOT NULL DEFAULT 300,
    grace_sec    INT NOT NULL DEFAULT 60,
    source       VARCHAR(100) NOT NULL DEFAULT 'heartbeat',
    keyword      VARCHAR(100) NOT NULL,
    last_seen    TIMESTAMPTZ,
    alerted_at   TIMESTAMPTZ,
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE rules ADD COLUMN repeat_interval_sec INT NOT NULL DEFAULT 0;
ALTER TABLE rules ADD COLUMN repeat_max          INT NOT NULL DEFAULT 3;
