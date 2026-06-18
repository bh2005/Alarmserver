package db

import (
	"context"
	"fmt"

	"github.com/bh2005/alarmserver/internal/core"
)

// AcknowledgementStore implements core.AcknowledgementWriter/Reader against
// the acknowledgements table.
type AcknowledgementStore struct {
	pool *Pool
}

func NewAcknowledgementStore(pool *Pool) *AcknowledgementStore {
	return &AcknowledgementStore{pool: pool}
}

// RecordAcknowledgement is idempotent per (event, recipient): a repeated
// reply from the same recipient for the same event replaces the previous
// one rather than erroring.
func (s *AcknowledgementStore) RecordAcknowledgement(ctx context.Context, ack core.Acknowledgement) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO acknowledgements (event_log_id, recipient_id, response, channel)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_log_id, recipient_id)
		DO UPDATE SET response = EXCLUDED.response, channel = EXCLUDED.channel, acknowledged_at = NOW()
	`, ack.EventLogID, ack.RecipientID, ack.Response, ack.Channel)
	if err != nil {
		return fmt.Errorf("record acknowledgement: %w", err)
	}
	return nil
}

// IsAcknowledged reports whether at least one acknowledgement exists for the event.
func (s *AcknowledgementStore) IsAcknowledged(ctx context.Context, eventLogID int) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM acknowledgements WHERE event_log_id=$1`, eventLogID).Scan(&count)
	return count > 0, err
}

func (s *AcknowledgementStore) LoadAcknowledgements(ctx context.Context, eventLogID int) ([]core.Acknowledgement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT event_log_id, recipient_id, response, channel, acknowledged_at
		FROM acknowledgements
		WHERE event_log_id = $1
		ORDER BY acknowledged_at
	`, eventLogID)
	if err != nil {
		return nil, fmt.Errorf("load acknowledgements: %w", err)
	}
	defer rows.Close()

	var acks []core.Acknowledgement
	for rows.Next() {
		var a core.Acknowledgement
		var channel *string
		if err := rows.Scan(&a.EventLogID, &a.RecipientID, &a.Response, &channel, &a.AcknowledgedAt); err != nil {
			return nil, fmt.Errorf("scan acknowledgement: %w", err)
		}
		if channel != nil {
			a.Channel = *channel
		}
		acks = append(acks, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate acknowledgements: %w", err)
	}
	return acks, nil
}
