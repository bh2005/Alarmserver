package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bh2005/alarmserver/internal/core"
)

func TestHandleWebhook_ValidPayload(t *testing.T) {
	events := make(chan core.Event, 1)
	s := NewServer(events)

	req := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"keyword":"F2","text":"fire"}`))
	rec := httptest.NewRecorder()
	s.HandleWebhook(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	select {
	case e := <-events:
		if e.Keyword != "F2" || e.Source != "WEBHOOK" {
			t.Fatalf("unexpected event: %+v", e)
		}
	default:
		t.Fatal("expected event to be queued")
	}
}

func TestHandleWebhook_MissingKeyword(t *testing.T) {
	events := make(chan core.Event, 1)
	s := NewServer(events)

	req := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"text":"fire"}`))
	rec := httptest.NewRecorder()
	s.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	events := make(chan core.Event, 1)
	s := NewServer(events)

	req := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()
	s.HandleWebhook(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWebhook_WrongMethod(t *testing.T) {
	events := make(chan core.Event, 1)
	s := NewServer(events)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	s.HandleWebhook(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWebhook_QueueFull(t *testing.T) {
	events := make(chan core.Event) // unbuffered -> immediately full
	s := NewServer(events)

	req := httptest.NewRequest(http.MethodPost, "/api/events", strings.NewReader(`{"keyword":"F2"}`))
	rec := httptest.NewRecorder()
	s.HandleWebhook(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
