package smseagle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/bh2005/alarmserver/internal/core"
)

func TestNotify_SendsOnePerRecipient(t *testing.T) {
	var mu sync.Mutex
	var requested []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.URL.Query().Get("to"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Login: "user", Password: "pw"})
	err := c.Notify(context.Background(), []core.Recipient{
		{Mobile: "0151111"},
		{Mobile: "0152222"},
		{Mobile: ""}, // skipped
	}, "test message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requested) != 2 {
		t.Fatalf("expected 2 requests, got %d: %v", len(requested), requested)
	}
}

func TestNotify_ReportsHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Login: "user", Password: "pw"})
	err := c.Notify(context.Background(), []core.Recipient{{Mobile: "0151111"}}, "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "0151111") {
		t.Fatalf("expected error to mention the failed number, got: %v", err)
	}
}

func TestNotify_EscapesMessageAndCredentials(t *testing.T) {
	var gotURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Login: "a b", Password: "p&w"})
	if err := c.Notify(context.Background(), []core.Recipient{{Mobile: "+49 151"}}, "hello world & more"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotURL.Query().Get("message") != "hello world & more" {
		t.Fatalf("message not round-tripped correctly: %q", gotURL.Query().Get("message"))
	}
	if gotURL.Query().Get("to") != "+49 151" {
		t.Fatalf("number not round-tripped correctly: %q", gotURL.Query().Get("to"))
	}
}
