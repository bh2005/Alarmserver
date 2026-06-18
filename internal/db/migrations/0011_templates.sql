CREATE TABLE message_templates (
    channel     VARCHAR(50) PRIMARY KEY,
    template    TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
