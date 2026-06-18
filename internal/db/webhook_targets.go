package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type WebhookTargetRecord struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Label        string            `json:"label"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	BodyTemplate string            `json:"body_template"`
	TimeoutSec   int               `json:"timeout_sec"`
	Active       bool              `json:"active"`
}

type WebhookTargetStore struct {
	pool *Pool
}

func NewWebhookTargetStore(pool *Pool) *WebhookTargetStore {
	return &WebhookTargetStore{pool: pool}
}

func (s *WebhookTargetStore) List(ctx context.Context) ([]WebhookTargetRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, label, url, method, headers, COALESCE(body_template,''), timeout_sec, active
		FROM webhook_targets ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list webhook targets: %w", err)
	}
	defer rows.Close()

	var out []WebhookTargetRecord
	for rows.Next() {
		var r WebhookTargetRecord
		var hdrsRaw []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Label, &r.URL, &r.Method,
			&hdrsRaw, &r.BodyTemplate, &r.TimeoutSec, &r.Active); err != nil {
			return nil, err
		}
		r.Headers = map[string]string{}
		if len(hdrsRaw) > 0 {
			_ = json.Unmarshal(hdrsRaw, &r.Headers)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *WebhookTargetStore) ListActive(ctx context.Context) ([]WebhookTargetRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, label, url, method, headers, COALESCE(body_template,''), timeout_sec, active
		FROM webhook_targets WHERE active=TRUE ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active webhook targets: %w", err)
	}
	defer rows.Close()

	var out []WebhookTargetRecord
	for rows.Next() {
		var r WebhookTargetRecord
		var hdrsRaw []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Label, &r.URL, &r.Method,
			&hdrsRaw, &r.BodyTemplate, &r.TimeoutSec, &r.Active); err != nil {
			return nil, err
		}
		r.Headers = map[string]string{}
		if len(hdrsRaw) > 0 {
			_ = json.Unmarshal(hdrsRaw, &r.Headers)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *WebhookTargetStore) Create(ctx context.Context, r WebhookTargetRecord) (WebhookTargetRecord, error) {
	hdrs, _ := json.Marshal(r.Headers)
	var bt interface{}
	if r.BodyTemplate != "" {
		bt = r.BodyTemplate
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_targets (name, label, url, method, headers, body_template, timeout_sec, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		r.Name, r.Label, r.URL, r.Method, hdrs, bt, r.TimeoutSec, r.Active,
	).Scan(&r.ID)
	if err != nil {
		return WebhookTargetRecord{}, fmt.Errorf("create webhook target: %w", err)
	}
	return r, nil
}

func (s *WebhookTargetStore) Update(ctx context.Context, r WebhookTargetRecord) error {
	hdrs, _ := json.Marshal(r.Headers)
	var bt interface{}
	if r.BodyTemplate != "" {
		bt = r.BodyTemplate
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_targets
		SET name=$1, label=$2, url=$3, method=$4, headers=$5, body_template=$6, timeout_sec=$7, active=$8
		WHERE id=$9`,
		r.Name, r.Label, r.URL, r.Method, hdrs, bt, r.TimeoutSec, r.Active, r.ID)
	if err != nil {
		return fmt.Errorf("update webhook target: %w", err)
	}
	return nil
}

func (s *WebhookTargetStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM webhook_targets WHERE id=$1`, id)
	return err
}
