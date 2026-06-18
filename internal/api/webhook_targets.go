package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bh2005/alarmserver/internal/db"
)

type WebhookTargetStoreIface interface {
	List(ctx context.Context) ([]db.WebhookTargetRecord, error)
	Create(ctx context.Context, r db.WebhookTargetRecord) (db.WebhookTargetRecord, error)
	Update(ctx context.Context, r db.WebhookTargetRecord) error
	Delete(ctx context.Context, id int) error
}

type WebhookTargetsServer struct{ Store WebhookTargetStoreIface }

func NewWebhookTargetsServer(store WebhookTargetStoreIface) *WebhookTargetsServer {
	return &WebhookTargetsServer{Store: store}
}

// HandleCollection — GET /api/webhook-targets  POST /api/webhook-targets
func (s *WebhookTargetsServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []db.WebhookTargetRecord{}
		}
		writeJSON(w, list)

	case http.MethodPost:
		var rec db.WebhookTargetRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if rec.Name == "" || rec.URL == "" {
			http.Error(w, "name and url required", http.StatusBadRequest)
			return
		}
		if rec.Method == "" {
			rec.Method = "POST"
		}
		if rec.TimeoutSec == 0 {
			rec.TimeoutSec = 10
		}
		if rec.Headers == nil {
			rec.Headers = map[string]string{}
		}
		created, err := s.Store.Create(r.Context(), rec)
		if err != nil {
			http.Error(w, "create failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleItem — PUT/DELETE /api/webhook-targets/{id}
func (s *WebhookTargetsServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/webhook-targets/", "")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var rec db.WebhookTargetRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if rec.Name == "" || rec.URL == "" {
			http.Error(w, "name and url required", http.StatusBadRequest)
			return
		}
		rec.ID = id
		if rec.Method == "" {
			rec.Method = "POST"
		}
		if rec.Headers == nil {
			rec.Headers = map[string]string{}
		}
		if err := s.Store.Update(r.Context(), rec); err != nil {
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
