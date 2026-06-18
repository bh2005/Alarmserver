// Package email implements core.Notifier for the "email" channel via SMTP.
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/bh2005/alarmserver/internal/core"
)

type Config struct {
	Host     string
	Port     int
	Username string // empty = no SMTP auth (e.g. local relay)
	Password string
	From     string
	Subject  string // default "Alert notification" if empty
}

// sendMailFunc matches smtp.SendMail's signature so tests can inject a fake.
type sendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

type Client struct {
	cfg      Config
	sendMail sendMailFunc
}

func New(cfg Config) *Client {
	if cfg.Subject == "" {
		cfg.Subject = "Alert notification"
	}
	return &Client{cfg: cfg, sendMail: smtp.SendMail}
}

func (c *Client) Channel() string { return "email" }

// Notify sends one e-mail addressed to all recipients with a non-empty
// Email address. SMTP delivery happens in a goroutine so ctx cancellation
// can be honored even though net/smtp has no native context support.
func (c *Client) Notify(ctx context.Context, recipients []core.Recipient, message string) error {
	to := recipientAddresses(recipients)
	if len(to) == 0 {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var auth smtp.Auth
	if c.cfg.Username != "" {
		auth = smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
	}
	body := buildMessage(c.cfg.From, to, c.cfg.Subject, message)

	done := make(chan error, 1)
	go func() {
		done <- c.sendMail(addr, auth, c.cfg.From, to, body)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("email: send: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func recipientAddresses(recipients []core.Recipient) []string {
	var to []string
	for _, r := range recipients {
		if r.Email != "" {
			to = append(to, r.Email)
		}
	}
	return to
}

func buildMessage(from string, to []string, subject, body string) []byte {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Subject: %s\r\n", subject)
	fmt.Fprintf(&sb, "From: %s\r\n", from)
	fmt.Fprintf(&sb, "To: %s\r\n", strings.Join(to, ", "))
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")
	return []byte(sb.String())
}
