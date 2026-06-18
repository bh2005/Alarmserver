CREATE TABLE channel_configs (
    channel     VARCHAR(50) PRIMARY KEY,
    config      JSONB NOT NULL DEFAULT '{}',
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
