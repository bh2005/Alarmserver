-- Adds per-rule, optional acknowledgement ("coming / not coming") tracking
-- (concept doc ch. 8.2/10: recipients confirm via SMS reply, app, or DTMF).

ALTER TABLE rules ADD COLUMN require_ack BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE acknowledgements (
    id              SERIAL PRIMARY KEY,
    event_log_id    INT NOT NULL REFERENCES event_log(id) ON DELETE CASCADE,
    recipient_id    INT NOT NULL REFERENCES recipients(id),
    response        VARCHAR(12) NOT NULL, -- 'COMING' | 'NOT_COMING' | 'LATER'
    channel         VARCHAR(10),          -- channel the acknowledgement came in on
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_log_id, recipient_id)
);

CREATE INDEX idx_acknowledgements_event ON acknowledgements (event_log_id);
