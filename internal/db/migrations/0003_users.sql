-- UI login accounts. Separate from ALARM_API_KEY, which authenticates
-- machine-to-machine calls (webhook, announcements upload, ack recording).

CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'viewer', -- 'admin' | 'viewer'
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
