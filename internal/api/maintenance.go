package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bh2005/alarmserver/internal/db"
)

type MaintenanceStoreIface interface {
	List(ctx context.Context) ([]db.MaintenanceWindow, error)
	Create(ctx context.Context, m db.MaintenanceWindow) (db.MaintenanceWindow, error)
	Update(ctx context.Context, m db.MaintenanceWindow) error
	Delete(ctx context.Context, id int) error
	Active(ctx context.Context) ([]db.MaintenanceWindow, error)
}

type MaintenanceServer struct {
	Store MaintenanceStoreIface
}

func NewMaintenanceServer(store MaintenanceStoreIface) *MaintenanceServer {
	return &MaintenanceServer{Store: store}
}

func (s *MaintenanceServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		windows, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list maintenance windows", http.StatusInternalServerError)
			return
		}
		if windows == nil {
			windows = []db.MaintenanceWindow{}
		}
		writeJSON(w, map[string]any{"windows": windows})
	case http.MethodPost:
		var payload struct {
			Label    string  `json:"label"`
			StartsAt string  `json:"starts_at"`
			EndsAt   string  `json:"ends_at"`
			Source   *string `json:"source"`
			Keyword  *string `json:"keyword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Label == "" {
			http.Error(w, "label, starts_at and ends_at are required", http.StatusBadRequest)
			return
		}
		starts, err1 := time.Parse(time.RFC3339, payload.StartsAt)
		ends, err2 := time.Parse(time.RFC3339, payload.EndsAt)
		if err1 != nil || err2 != nil || !ends.After(starts) {
			http.Error(w, "starts_at and ends_at must be valid RFC3339 timestamps and ends_at must be after starts_at", http.StatusBadRequest)
			return
		}
		m := db.MaintenanceWindow{
			Label:    payload.Label,
			StartsAt: starts,
			EndsAt:   ends,
			Source:   payload.Source,
			Keyword:  payload.Keyword,
		}
		created, err := s.Store.Create(r.Context(), m)
		if err != nil {
			http.Error(w, "could not create maintenance window", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *MaintenanceServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/maintenance/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var payload struct {
			Label    string  `json:"label"`
			StartsAt string  `json:"starts_at"`
			EndsAt   string  `json:"ends_at"`
			Source   *string `json:"source"`
			Keyword  *string `json:"keyword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Label == "" {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		starts, err1 := time.Parse(time.RFC3339, payload.StartsAt)
		ends, err2 := time.Parse(time.RFC3339, payload.EndsAt)
		if err1 != nil || err2 != nil {
			http.Error(w, "invalid timestamps", http.StatusBadRequest)
			return
		}
		if err := s.Store.Update(r.Context(), db.MaintenanceWindow{
			ID:       id,
			Label:    payload.Label,
			StartsAt: starts,
			EndsAt:   ends,
			Source:   payload.Source,
			Keyword:  payload.Keyword,
		}); err != nil {
			http.Error(w, "could not update maintenance window", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "could not delete maintenance window", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
