// Package imap polls an IMAP mailbox and injects matching e-mails as
// alarm events into the processing pipeline.
//
// Configuration (stored as a channel config row with channel="imap"):
//
//	host       IMAP server hostname
//	port       993 (IMAPS) or 143
//	tls        true/false
//	username   mailbox login
//	password   mailbox password
//	folder     mailbox folder to watch, default "INBOX"
//	poll_sec   polling interval in seconds, default 60
//	source     value injected as event.Source, default "imap"
package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/bh2005/alarmserver/internal/core"
)

// Config holds everything needed to connect to and poll the mailbox.
type Config struct {
	Host     string
	Port     int
	TLS      bool
	Username string
	Password string
	Folder   string
	PollSec  int
	Source   string
}

// Poller runs a background loop and sends core.Event values to the
// provided channel for every unseen e-mail it finds.
type Poller struct {
	cfg    Config
	events chan<- core.Event
	logger *slog.Logger
}

func New(cfg Config, events chan<- core.Event, logger *slog.Logger) *Poller {
	if cfg.Folder == "" {
		cfg.Folder = "INBOX"
	}
	if cfg.PollSec <= 0 {
		cfg.PollSec = 60
	}
	if cfg.Source == "" {
		cfg.Source = "imap"
	}
	if cfg.Port == 0 {
		if cfg.TLS {
			cfg.Port = 993
		} else {
			cfg.Port = 143
		}
	}
	return &Poller{cfg: cfg, events: events, logger: logger}
}

// Run polls the mailbox every PollSec seconds until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.cfg.PollSec) * time.Second)
	defer ticker.Stop()
	p.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	addr := fmt.Sprintf("%s:%d", p.cfg.Host, p.cfg.Port)

	var c *imapclient.Client
	if p.cfg.TLS {
		var err error
		c, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: p.cfg.Host},
		})
		if err != nil {
			p.logger.Error("imap_connect_failed", "err", err)
			return
		}
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			p.logger.Error("imap_connect_failed", "err", err)
			return
		}
		c = imapclient.New(conn, nil)
	}
	defer c.Close()

	if err := c.Login(p.cfg.Username, p.cfg.Password).Wait(); err != nil {
		p.logger.Error("imap_login_failed", "err", err)
		return
	}
	defer c.Logout().Wait()

	if _, err := c.Select(p.cfg.Folder, nil).Wait(); err != nil {
		p.logger.Error("imap_select_failed", "folder", p.cfg.Folder, "err", err)
		return
	}

	// Search for UNSEEN messages.
	data, err := c.Search(&imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}, nil).Wait()
	if err != nil {
		p.logger.Error("imap_search_failed", "err", err)
		return
	}
	seqNums := data.AllSeqNums()
	if len(seqNums) == 0 {
		return
	}

	seqSet := imap.SeqSetNum(seqNums...)
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	msgs, err := c.Fetch(seqSet, fetchOptions).Collect()
	if err != nil {
		p.logger.Error("imap_fetch_failed", "err", err)
		return
	}

	for _, msg := range msgs {
		subject := ""
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
		}
		body := ""
		for _, section := range msg.BodySection {
			body = string(section.Bytes)
			break
		}
		keyword := extractKeyword(subject, body)
		event := core.Event{
			Source:  p.cfg.Source,
			Keyword: keyword,
			Text:    strings.TrimSpace(subject + "\n" + body),
		}
		select {
		case p.events <- event:
			p.logger.Info("imap_event_injected", "subject", subject, "keyword", keyword)
		case <-ctx.Done():
			return
		}
	}

	// Mark fetched messages as seen.
	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagSeen},
	}
	c.Store(seqSet, storeFlags, nil).Close()
}

// extractKeyword picks the first ALL-CAPS word (≥3 chars) from the subject
// as the alarm keyword; falls back to the first word of the subject.
func extractKeyword(subject, _ string) string {
	for _, word := range strings.Fields(subject) {
		w := strings.Trim(word, `[]()"'.,:;!?`)
		if len(w) >= 3 && w == strings.ToUpper(w) && !isAllDigits(w) {
			return w
		}
	}
	fields := strings.Fields(subject)
	if len(fields) > 0 {
		return strings.Trim(fields[0], `[]()"'.,:;!?`)
	}
	return "MAIL"
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ConfigFromMap converts a string map (from db.ChannelConfig) to Config.
func ConfigFromMap(m map[string]string) (Config, error) {
	cfg := Config{
		Host:     m["host"],
		TLS:      m["tls"] == "true",
		Username: m["username"],
		Password: m["password"],
		Folder:   m["folder"],
		Source:   m["source"],
	}
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return Config{}, fmt.Errorf("imap: host, username and password are required")
	}
	if p := m["port"]; p != "" {
		cfg.Port, _ = strconv.Atoi(p)
	}
	if s := m["poll_sec"]; s != "" {
		cfg.PollSec, _ = strconv.Atoi(s)
	}
	return cfg, nil
}
