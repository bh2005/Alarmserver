package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/bh2005/alarmserver/internal/db"
)

type EventLogStore interface {
	List(ctx context.Context, limit int) ([]db.EventLogEntry, error)
}

type EventLogServer struct {
	Store EventLogStore
}

func NewEventLogServer(store EventLogStore) *EventLogServer {
	return &EventLogServer{Store: store}
}

func (s *EventLogServer) HandleList(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}
	entries, err := s.Store.List(r.Context(), limit)
	if err != nil {
		http.Error(w, "could not list event log", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []db.EventLogEntry{}
	}
	writeJSON(w, map[string][]db.EventLogEntry{"events": entries})
}
