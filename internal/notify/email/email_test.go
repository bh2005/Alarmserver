package email

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

func TestNotify_SendsToAllRecipientsWithEmail(t *testing.T) {
	var gotTo []string
	var gotBody string

	c := New(Config{Host: "smtp.example.org", Port: 587, From: "alerts@example.org"})
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		gotTo = to
		gotBody = string(msg)
		return nil
	}

	err := c.Notify(context.Background(), []core.Recipient{
		{Email: "a@example.org"},
		{Email: "b@example.org"},
		{Email: ""}, // skipped
	}, "something happened")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotTo) != 2 {
		t.Fatalf("expected 2 recipients, got %v", gotTo)
	}
	if !strings.Contains(gotBody, "something happened") {
		t.Fatalf("body missing message: %q", gotBody)
	}
	if !strings.Contains(gotBody, "From: alerts@example.org") {
		t.Fatalf("body missing From header: %q", gotBody)
	}
}

func TestNotify_NoRecipientsIsNoOp(t *testing.T) {
	called := false
	c := New(Config{Host: "smtp.example.org", Port: 587, From: "alerts@example.org"})
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		return nil
	}

	err := c.Notify(context.Background(), []core.Recipient{{Email: ""}}, "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("sendMail should not be called when no recipient has an email address")
	}
}

func TestNotify_PropagatesSendError(t *testing.T) {
	c := New(Config{Host: "smtp.example.org", Port: 587, From: "alerts@example.org"})
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return errors.New("connection refused")
	}

	err := c.Notify(context.Background(), []core.Recipient{{Email: "a@example.org"}}, "msg")
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected wrapped send error, got: %v", err)
	}
}

func TestNotify_RespectsContextCancellation(t *testing.T) {
	c := New(Config{Host: "smtp.example.org", Port: 587, From: "alerts@example.org"})
	block := make(chan struct{})
	c.sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		<-block
		return nil
	}
	defer close(block)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := c.Notify(ctx, []core.Recipient{{Email: "a@example.org"}}, "msg")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}
