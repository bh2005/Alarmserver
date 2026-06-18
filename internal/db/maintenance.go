package db

import (
	"context"
	"fmt"
	"time"
)

type MaintenanceWindow struct {
	ID        int       `json:"id"`
	Label     string    `json:"label"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Source    *string   `json:"source"`
	Keyword   *string   `json:"keyword"`
	CreatedAt time.Time `json:"created_at"`
}

type MaintenanceStore struct {
	pool *Pool
}

func NewMaintenanceStore(pool *Pool) *MaintenanceStore {
	return &MaintenanceStore{pool: pool}
}

func (s *MaintenanceStore) IsUnderMaintenance(ctx context.Context, source, keyword string, at time.Time) (bool, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM maintenance_windows
		WHERE starts_at <= $1 AND ends_at > $1
		  AND (source IS NULL OR source = $2)
		  AND (keyword IS NULL OR keyword = $3)
	`, at, source, keyword).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check maintenance: %w", err)
	}
	return count > 0, nil
}

func (s *MaintenanceStore) List(ctx context.Context) ([]MaintenanceWindow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, label, starts_at, ends_at, source, keyword, created_at
		FROM maintenance_windows ORDER BY starts_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list maintenance: %w", err)
	}
	defer rows.Close()
	var out []MaintenanceWindow
	for rows.Next() {
		var m MaintenanceWindow
		if err := rows.Scan(&m.ID, &m.Label, &m.StartsAt, &m.EndsAt, &m.Source, &m.Keyword, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *MaintenanceStore) Create(ctx context.Context, m MaintenanceWindow) (MaintenanceWindow, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO maintenance_windows (label, starts_at, ends_at, source, keyword)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, m.Label, m.StartsAt, m.EndsAt, m.Source, m.Keyword).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return MaintenanceWindow{}, fmt.Errorf("create maintenance window: %w", err)
	}
	return m, nil
}

func (s *MaintenanceStore) Update(ctx context.Context, m MaintenanceWindow) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE maintenance_windows
		SET label=$1, starts_at=$2, ends_at=$3, source=$4, keyword=$5
		WHERE id=$6
	`, m.Label, m.StartsAt, m.EndsAt, m.Source, m.Keyword, m.ID)
	return err
}

func (s *MaintenanceStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM maintenance_windows WHERE id=$1`, id)
	return err
}

func (s *MaintenanceStore) Active(ctx context.Context) ([]MaintenanceWindow, error) {
	now := time.Now()
	rows, err := s.pool.Query(ctx, `
		SELECT id, label, starts_at, ends_at, source, keyword, created_at
		FROM maintenance_windows WHERE starts_at <= $1 AND ends_at > $1 ORDER BY starts_at
	`, now)
	if err != nil {
		return nil, fmt.Errorf("active maintenance: %w", err)
	}
	defer rows.Close()
	var out []MaintenanceWindow
	for rows.Next() {
		var m MaintenanceWindow
		if err := rows.Scan(&m.ID, &m.Label, &m.StartsAt, &m.EndsAt, &m.Source, &m.Keyword, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
