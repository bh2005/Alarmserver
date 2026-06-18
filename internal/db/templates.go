package db

import (
	"context"
	"fmt"
	"time"
)

type MessageTemplate struct {
	Channel   string    `json:"channel"`
	Template  string    `json:"template"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TemplateStore struct {
	pool *Pool
}

func NewTemplateStore(pool *Pool) *TemplateStore {
	return &TemplateStore{pool: pool}
}

func (s *TemplateStore) GetTemplate(ctx context.Context, channel string) (string, error) {
	var tmpl string
	err := s.pool.QueryRow(ctx, `SELECT template FROM message_templates WHERE channel=$1`, channel).Scan(&tmpl)
	if err != nil {
		return "", nil // no template = use raw event text
	}
	return tmpl, nil
}

func (s *TemplateStore) List(ctx context.Context) ([]MessageTemplate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT channel, template, updated_at FROM message_templates ORDER BY channel
	`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	var out []MessageTemplate
	for rows.Next() {
		var t MessageTemplate
		if err := rows.Scan(&t.Channel, &t.Template, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TemplateStore) Upsert(ctx context.Context, channel, template string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO message_templates (channel, template, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (channel) DO UPDATE SET template = EXCLUDED.template, updated_at = NOW()
	`, channel, template)
	if err != nil {
		return fmt.Errorf("upsert template: %w", err)
	}
	return nil
}

func (s *TemplateStore) Delete(ctx context.Context, channel string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM message_templates WHERE channel=$1`, channel)
	return err
}
