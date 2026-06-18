package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

// AuditStore implements core.AuditWriter against the event_log table.
type AuditStore struct {
	pool *Pool
}

func NewAuditStore(pool *Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) WriteAuditLog(ctx context.Context, event core.Event, rule *core.Rule, status string, errorDetail string) (int, error) {
	var ruleID *int
	var targetGroup *string
	var channels []string
	if rule != nil {
		ruleID = &rule.ID
		targetGroup = &rule.TargetGroup
		channels = rule.Channels
	}

	var id int
	err := s.pool.QueryRow(ctx, `
		INSERT INTO event_log
			(received_at, source, keyword, raw_payload, rule_id, target_group, channels, status, error_detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, event.ReceivedAt, event.Source, event.Keyword, event.RawPayload,
		ruleID, targetGroup, channels, status, errorDetail).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write audit log: %w", err)
	}
	return id, nil
}

// EventLogEntry is one row of event_log, as shown by the admin UI.
type EventLogEntry struct {
	ID           int             `json:"id"`
	ReceivedAt   time.Time       `json:"received_at"`
	Source       string          `json:"source"`
	Keyword      string          `json:"keyword"`
	RawPayload   json.RawMessage `json:"raw_payload,omitempty"`
	RuleID       *int            `json:"rule_id"`
	TargetGroup  string          `json:"target_group"`
	Channels     []string        `json:"channels"`
	Status       string          `json:"status"`
	ErrorDetail  string          `json:"error_detail"`
	Acknowledged bool            `json:"acknowledged"`
}

// List returns the most recent event_log rows, newest first.
func (s *AuditStore) List(ctx context.Context, limit int) ([]EventLogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, received_at, source, keyword, raw_payload, rule_id,
		       COALESCE(target_group, ''), channels, status, COALESCE(error_detail, ''),
		       EXISTS(SELECT 1 FROM acknowledgements WHERE event_log_id = event_log.id)
		FROM event_log
		ORDER BY received_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list event log: %w", err)
	}
	defer rows.Close()

	var out []EventLogEntry
	for rows.Next() {
		var e EventLogEntry
		if err := rows.Scan(&e.ID, &e.ReceivedAt, &e.Source, &e.Keyword, &e.RawPayload,
			&e.RuleID, &e.TargetGroup, &e.Channels, &e.Status, &e.ErrorDetail, &e.Acknowledged); err != nil {
			return nil, fmt.Errorf("scan event log entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListOpen returns dispatched events from the last 24 h that have no ACK yet.
func (s *AuditStore) ListOpen(ctx context.Context) ([]EventLogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.received_at, e.source, e.keyword, e.raw_payload, e.rule_id,
		       COALESCE(e.target_group, ''), e.channels, e.status, COALESCE(e.error_detail, ''),
		       FALSE
		FROM event_log e
		WHERE e.status = 'DISPATCHED'
		  AND e.received_at >= NOW() - INTERVAL '24 hours'
		  AND NOT EXISTS (SELECT 1 FROM acknowledgements WHERE event_log_id = e.id)
		ORDER BY e.received_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list open events: %w", err)
	}
	defer rows.Close()

	var out []EventLogEntry
	for rows.Next() {
		var e EventLogEntry
		if err := rows.Scan(&e.ID, &e.ReceivedAt, &e.Source, &e.Keyword, &e.RawPayload,
			&e.RuleID, &e.TargetGroup, &e.Channels, &e.Status, &e.ErrorDetail, &e.Acknowledged); err != nil {
			return nil, fmt.Errorf("scan open event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
