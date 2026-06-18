// Package teams implements core.Notifier for the "teams" channel via a
// Microsoft Teams incoming webhook (or an Office 365 Connector / Power
// Automate workflow exposing the same simple JSON contract).
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

type Config struct {
	WebhookURL string
	Timeout    time.Duration
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: cfg.Timeout}}
}

func (c *Client) Channel() string { return "teams" }

// Notify posts message to the configured Teams webhook. Teams webhooks
// target a channel rather than individual people, so the recipients
// argument from core.Notifier is intentionally unused here — every call
// posts exactly one message regardless of how many recipients are passed.
func (c *Client) Notify(ctx context.Context, _ []core.Recipient, message string) error {
	payload := map[string]string{"text": message}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("teams: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("teams: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("teams: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("teams: http %d", resp.StatusCode)
	}
	return nil
}
