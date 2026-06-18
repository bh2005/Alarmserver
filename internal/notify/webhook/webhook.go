// Package webhook implements core.Notifier for generic outgoing HTTP webhooks.
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

const defaultBodyTemplate = `{"keyword":"{keyword}","source":"{source}","text":"{message}","group":"{group}","time":"{time}"}`

// Config holds all parameters for a single webhook target.
type Config struct {
	Name         string
	URL          string
	Method       string // default POST
	Headers      map[string]string
	BodyTemplate string // placeholders: {message} {keyword} {source} {group} {time}
	TimeoutSec   int
}

// Notifier sends an HTTP request to the configured URL on each Notify call.
// The recipients list is ignored — webhooks address a system, not a person.
type Notifier struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Notifier {
	if cfg.Method == "" {
		cfg.Method = "POST"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 10
	}
	if cfg.BodyTemplate == "" {
		cfg.BodyTemplate = defaultBodyTemplate
	}
	return &Notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second},
	}
}

func (n *Notifier) Channel() string { return n.cfg.Name }

// Notify sends the webhook. The message parameter is the already-resolved
// text (after template substitution by the Processor). Event metadata is not
// available here, so the body template only supports {message} in addition to
// the Processor-level placeholders already baked into message.
func (n *Notifier) Notify(ctx context.Context, _ []core.Recipient, message string) error {
	body := strings.NewReplacer("{message}", message).Replace(n.cfg.BodyTemplate)

	req, err := http.NewRequestWithContext(ctx, n.cfg.Method, n.cfg.URL, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("webhook %s: build request: %w", n.cfg.Name, err)
	}

	if _, ok := n.cfg.Headers["Content-Type"]; !ok {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook %s: request failed: %w", n.cfg.Name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook %s: server returned %d", n.cfg.Name, resp.StatusCode)
	}
	return nil
}
