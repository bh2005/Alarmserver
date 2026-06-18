CREATE TABLE maintenance_windows (
    id          SERIAL PRIMARY KEY,
    label       VARCHAR(200) NOT NULL,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    source      VARCHAR(100),
    keyword     VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maintenance_active ON maintenance_windows (starts_at, ends_at);
