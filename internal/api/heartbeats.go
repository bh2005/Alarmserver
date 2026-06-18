package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

type HeartbeatStoreIface interface {
	List(ctx context.Context) ([]db.Heartbeat, error)
	Create(ctx context.Context, h db.Heartbeat) (db.Heartbeat, error)
	Update(ctx context.Context, h db.Heartbeat) error
	Delete(ctx context.Context, id int) error
	Touch(ctx context.Context, name string) (bool, error)
}

type HeartbeatsServer struct {
	Store HeartbeatStoreIface
}

func NewHeartbeatsServer(store HeartbeatStoreIface) *HeartbeatsServer {
	return &HeartbeatsServer{Store: store}
}

func (s *HeartbeatsServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list heartbeats", http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []db.Heartbeat{}
		}
		writeJSON(w, map[string]any{"heartbeats": list})
	case http.MethodPost:
		var h db.Heartbeat
		if err := json.NewDecoder(r.Body).Decode(&h); err != nil || h.Name == "" || h.Keyword == "" {
			http.Error(w, "name and keyword are required", http.StatusBadRequest)
			return
		}
		if h.IntervalSec <= 0 {
			h.IntervalSec = 300
		}
		if h.GraceSec <= 0 {
			h.GraceSec = 60
		}
		if h.Source == "" {
			h.Source = "heartbeat"
		}
		h.Active = true
		created, err := s.Store.Create(r.Context(), h)
		if err != nil {
			http.Error(w, "could not create heartbeat", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HeartbeatsServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/heartbeats/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var h db.Heartbeat
		if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		h.ID = id
		if err := s.Store.Update(r.Context(), h); err != nil {
			http.Error(w, "could not update heartbeat", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "could not delete heartbeat", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandlePing handles the public-facing heartbeat ping endpoint:
// POST /api/heartbeat/{name}
func (s *HeartbeatsServer) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/heartbeat/")
	name = strings.TrimSuffix(name, "/")
	if name == "" {
		http.Error(w, "heartbeat name required", http.StatusBadRequest)
		return
	}
	found, err := s.Store.Touch(r.Context(), name)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "heartbeat not found or inactive", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
