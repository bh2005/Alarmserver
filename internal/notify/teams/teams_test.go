package teams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotify_PostsMessageAsJSON(t *testing.T) {
	var gotBody, gotContentType, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		gotContentType = r.Header.Get("Content-Type")
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{WebhookURL: srv.URL})
	if err := c.Notify(context.Background(), nil, "deployment failed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected application/json, got %s", gotContentType)
	}
	if !strings.Contains(gotBody, "deployment failed") {
		t.Fatalf("body missing message: %q", gotBody)
	}
}

func TestNotify_ReportsNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(Config{WebhookURL: srv.URL})
	err := c.Notify(context.Background(), nil, "msg")
	if err == nil {
		t.Fatal("expected error for non-2xx status")
	}
}
