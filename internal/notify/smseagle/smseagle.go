// Package smseagle implements core.Notifier for the "sms" channel via the
// SMSEAGLE HTTP API (http_api/send_sms).
package smseagle

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

type Config struct {
	BaseURL  string
	Login    string
	Password string
	APIKey   string // alternative to Login+Password
	Timeout  time.Duration
}

// authParams returns either access_token=… (API key) or login=…&pass=… query params.
func (c Config) authParams() string {
	if c.APIKey != "" {
		return "access_token=" + url.QueryEscape(c.APIKey)
	}
	return "login=" + url.QueryEscape(c.Login) + "&pass=" + url.QueryEscape(c.Password)
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

func (c *Client) Channel() string { return "sms" }

// Notify sends one SMS per recipient with a non-empty Mobile number. It
// always attempts every recipient and returns an error summarizing any
// failures.
func (c *Client) Notify(ctx context.Context, recipients []core.Recipient, message string) error {
	var failed []string
	attempted := 0
	for _, r := range recipients {
		if r.Mobile == "" {
			continue
		}
		attempted++
		if err := c.send(ctx, r.Mobile, message); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", r.Mobile, err))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("smseagle: %d/%d messages failed: %s", len(failed), attempted, strings.Join(failed, "; "))
	}
	return nil
}

// VoiceClient implements core.Notifier for the "voice-smseagle" channel via
// the SMSEagle voice-call API. Used as escalation backup when the primary
// Asterisk voice call goes unanswered.
type VoiceClient struct {
	cfg        Config
	httpClient *http.Client
}

func NewVoice(cfg Config) *VoiceClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &VoiceClient{cfg: cfg, httpClient: &http.Client{Timeout: cfg.Timeout}}
}

func (c *VoiceClient) Channel() string { return "voice-smseagle" }

// Notify places one voice call per recipient with a non-empty Voice number.
// The SMSEagle device reads the message text via TTS.
func (c *VoiceClient) Notify(ctx context.Context, recipients []core.Recipient, message string) error {
	var failed []string
	attempted := 0
	for _, r := range recipients {
		if r.Voice == "" {
			continue
		}
		attempted++
		if err := c.call(ctx, r.Voice, message); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", r.Voice, err))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("smseagle voice: %d/%d calls failed: %s", len(failed), attempted, strings.Join(failed, "; "))
	}
	return nil
}

func (c *VoiceClient) call(ctx context.Context, number, text string) error {
	endpoint := fmt.Sprintf("%s/index.php/http_api/send_voice?%s&to=%s&message=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"),
		c.cfg.authParams(),
		url.QueryEscape(number),
		url.QueryEscape(text),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) send(ctx context.Context, number, text string) error {
	endpoint := fmt.Sprintf("%s/index.php/http_api/send_sms?%s&to=%s&message=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"),
		c.cfg.authParams(),
		url.QueryEscape(number),
		url.QueryEscape(text),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}
