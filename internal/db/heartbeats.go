package db

import (
	"context"
	"fmt"
	"time"
)

type Heartbeat struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	IntervalSec int        `json:"interval_sec"`
	GraceSec    int        `json:"grace_sec"`
	Source      string     `json:"source"`
	Keyword     string     `json:"keyword"`
	LastSeen    *time.Time `json:"last_seen"`
	AlertedAt   *time.Time `json:"alerted_at"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	Status      string     `json:"status"` // "ok", "dead", "unknown", "disabled"
}

type HeartbeatStore struct {
	pool *Pool
}

func NewHeartbeatStore(pool *Pool) *HeartbeatStore {
	return &HeartbeatStore{pool: pool}
}

func (s *HeartbeatStore) Touch(ctx context.Context, name string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE heartbeats SET last_seen = NOW(), alerted_at = NULL WHERE name = $1 AND active = TRUE
	`, name)
	if err != nil {
		return false, fmt.Errorf("touch heartbeat: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *HeartbeatStore) List(ctx context.Context) ([]Heartbeat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, COALESCE(description,''), interval_sec, grace_sec, source, keyword,
		       last_seen, alerted_at, active, created_at
		FROM heartbeats ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list heartbeats: %w", err)
	}
	defer rows.Close()
	var out []Heartbeat
	for rows.Next() {
		var h Heartbeat
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.IntervalSec, &h.GraceSec,
			&h.Source, &h.Keyword, &h.LastSeen, &h.AlertedAt, &h.Active, &h.CreatedAt); err != nil {
			return nil, err
		}
		h.Status = heartbeatStatus(h)
		out = append(out, h)
	}
	return out, rows.Err()
}

func heartbeatStatus(h Heartbeat) string {
	if !h.Active {
		return "disabled"
	}
	if h.LastSeen == nil {
		return "unknown"
	}
	deadline := h.LastSeen.Add(time.Duration(h.IntervalSec+h.GraceSec) * time.Second)
	if time.Now().Before(deadline) {
		return "ok"
	}
	return "dead"
}

func (s *HeartbeatStore) Create(ctx context.Context, h Heartbeat) (Heartbeat, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO heartbeats (name, description, interval_sec, grace_sec, source, keyword, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, h.Name, h.Description, h.IntervalSec, h.GraceSec, h.Source, h.Keyword, h.Active).Scan(&h.ID, &h.CreatedAt)
	if err != nil {
		return Heartbeat{}, fmt.Errorf("create heartbeat: %w", err)
	}
	h.Status = heartbeatStatus(h)
	return h, nil
}

func (s *HeartbeatStore) Update(ctx context.Context, h Heartbeat) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE heartbeats
		SET description=$1, interval_sec=$2, grace_sec=$3, source=$4, keyword=$5, active=$6
		WHERE id=$7
	`, h.Description, h.IntervalSec, h.GraceSec, h.Source, h.Keyword, h.Active, h.ID)
	return err
}

func (s *HeartbeatStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM heartbeats WHERE id=$1`, id)
	return err
}

// Expired returns active heartbeats that are overdue and haven't been alerted
// within the last hour (to avoid spam).
func (s *HeartbeatStore) Expired(ctx context.Context) ([]Heartbeat, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, COALESCE(description,''), interval_sec, grace_sec, source, keyword,
		       last_seen, alerted_at, active, created_at
		FROM heartbeats
		WHERE active = TRUE
		  AND (last_seen IS NULL OR last_seen + make_interval(secs => interval_sec + grace_sec) < NOW())
		  AND (alerted_at IS NULL OR alerted_at + INTERVAL '1 hour' < NOW())
	`)
	if err != nil {
		return nil, fmt.Errorf("expired heartbeats: %w", err)
	}
	defer rows.Close()
	var out []Heartbeat
	for rows.Next() {
		var h Heartbeat
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.IntervalSec, &h.GraceSec,
			&h.Source, &h.Keyword, &h.LastSeen, &h.AlertedAt, &h.Active, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *HeartbeatStore) MarkAlerted(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `UPDATE heartbeats SET alerted_at = NOW() WHERE id=$1`, id)
	return err
}
