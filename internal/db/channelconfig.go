package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ChannelConfig holds the persisted configuration for one notification channel.
type ChannelConfig struct {
	Channel   string            `json:"channel"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ChannelConfigStore persists per-channel configuration in PostgreSQL.
type ChannelConfigStore struct {
	pool *Pool
}

func NewChannelConfigStore(pool *Pool) *ChannelConfigStore {
	return &ChannelConfigStore{pool: pool}
}

func (s *ChannelConfigStore) LoadAll(ctx context.Context) ([]ChannelConfig, error) {
	rows, err := s.pool.Query(ctx, `SELECT channel, enabled, config, updated_at FROM channel_configs ORDER BY channel`)
	if err != nil {
		return nil, fmt.Errorf("list channel configs: %w", err)
	}
	defer rows.Close()

	var out []ChannelConfig
	for rows.Next() {
		var cc ChannelConfig
		var raw []byte
		if err := rows.Scan(&cc.Channel, &cc.Enabled, &raw, &cc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan channel config: %w", err)
		}
		if err := json.Unmarshal(raw, &cc.Config); err != nil {
			cc.Config = map[string]string{}
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

func (s *ChannelConfigStore) Load(ctx context.Context, channel string) (*ChannelConfig, error) {
	var cc ChannelConfig
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT channel, enabled, config, updated_at FROM channel_configs WHERE channel = $1`, channel).
		Scan(&cc.Channel, &cc.Enabled, &raw, &cc.UpdatedAt)
	if err != nil {
		return nil, nil // not found
	}
	if err := json.Unmarshal(raw, &cc.Config); err != nil {
		cc.Config = map[string]string{}
	}
	return &cc, nil
}

func (s *ChannelConfigStore) Save(ctx context.Context, channel string, enabled bool, cfg map[string]string) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO channel_configs (channel, enabled, config, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (channel) DO UPDATE
		  SET enabled = EXCLUDED.enabled,
		      config  = EXCLUDED.config,
		      updated_at = NOW()
	`, channel, enabled, raw)
	if err != nil {
		return fmt.Errorf("save channel config: %w", err)
	}
	return nil
}
