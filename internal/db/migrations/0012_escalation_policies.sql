CREATE TABLE escalation_policies (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Each step defines which group to alert after how many seconds without ACK.
-- delay_sec is absolute (from event dispatch), not relative to previous step.
-- channels: empty array = inherit channels from the triggering rule.
CREATE TABLE escalation_steps (
    id          SERIAL PRIMARY KEY,
    policy_id   INT NOT NULL REFERENCES escalation_policies(id) ON DELETE CASCADE,
    step        SMALLINT NOT NULL,
    group_id    INT NOT NULL REFERENCES groups(id),
    delay_sec   INT NOT NULL DEFAULT 600,
    channels    TEXT[] NOT NULL DEFAULT '{}',
    UNIQUE (policy_id, step)
);

ALTER TABLE rules ADD COLUMN escalation_policy_id INT REFERENCES escalation_policies(id);
