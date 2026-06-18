ALTER TABLE rules ADD COLUMN escalation_channels TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE rules ADD COLUMN escalation_delay_seconds SMALLINT NOT NULL DEFAULT 120;
