package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bh2005/alarmserver/internal/core"
)

type fakeAckStore struct {
	recorded []core.Acknowledgement
	loaded   []core.Acknowledgement
	loadErr  error
}

func (f *fakeAckStore) RecordAcknowledgement(ctx context.Context, ack core.Acknowledgement) error {
	f.recorded = append(f.recorded, ack)
	return nil
}

func (f *fakeAckStore) LoadAcknowledgements(ctx context.Context, eventLogID int) ([]core.Acknowledgement, error) {
	return f.loaded, f.loadErr
}

func TestHandleRecord_ValidPayload(t *testing.T) {
	store := &fakeAckStore{}
	s := NewAcknowledgementServer(store, store)

	body := `{"event_id":1,"recipient_id":2,"response":"COMING","channel":"sms"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acknowledgements", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.HandleRecord(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.recorded) != 1 {
		t.Fatalf("expected 1 recorded ack, got %d", len(store.recorded))
	}
	if store.recorded[0].Response != "COMING" {
		t.Fatalf("unexpected response: %+v", store.recorded[0])
	}
}

func TestHandleRecord_InvalidResponse(t *testing.T) {
	store := &fakeAckStore{}
	s := NewAcknowledgementServer(store, store)

	body := `{"event_id":1,"recipient_id":2,"response":"MAYBE","channel":"sms"}`
	req := httptest.NewRequest(http.MethodPost, "/api/acknowledgements", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.HandleRecord(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if len(store.recorded) != 0 {
		t.Fatal("expected nothing to be recorded for an invalid response")
	}
}

func TestHandleRecord_WrongMethod(t *testing.T) {
	store := &fakeAckStore{}
	s := NewAcknowledgementServer(store, store)
	req := httptest.NewRequest(http.MethodGet, "/api/acknowledgements", nil)
	rec := httptest.NewRecorder()
	s.HandleRecord(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleList_ReturnsAcknowledgements(t *testing.T) {
	store := &fakeAckStore{loaded: []core.Acknowledgement{
		{EventLogID: 1, RecipientID: 2, Response: "COMING", Channel: "sms"},
	}}
	s := NewAcknowledgementServer(store, store)

	req := httptest.NewRequest(http.MethodGet, "/api/acknowledgements?event_id=1", nil)
	rec := httptest.NewRecorder()
	s.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got struct {
		Acknowledgements []core.Acknowledgement `json:"acknowledgements"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(got.Acknowledgements) != 1 || got.Acknowledgements[0].Response != "COMING" {
		t.Fatalf("unexpected result: %+v", got.Acknowledgements)
	}
}

func TestHandleList_MissingEventID(t *testing.T) {
	store := &fakeAckStore{}
	s := NewAcknowledgementServer(store, store)
	req := httptest.NewRequest(http.MethodGet, "/api/acknowledgements", nil)
	rec := httptest.NewRecorder()
	s.HandleList(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
