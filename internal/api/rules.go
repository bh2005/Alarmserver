package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bh2005/alarmserver/internal/db"
)

type RulesStore interface {
	List(ctx context.Context) ([]db.RuleRecord, error)
	Create(ctx context.Context, r db.RuleRecord) (db.RuleRecord, error)
	Update(ctx context.Context, r db.RuleRecord) error
	Delete(ctx context.Context, id int) error
}

// RulesServer serves the rules CRUD API. Reload of the live RuleEngine
// happens on its own 60s timer (core.RuleEngine.StartBackgroundReload) —
// edits made here become effective within that window, not instantly.
type RulesServer struct {
	Store RulesStore
}

func NewRulesServer(store RulesStore) *RulesServer {
	return &RulesServer{Store: store}
}

func (s *RulesServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list rules", http.StatusInternalServerError)
			return
		}
		if rules == nil {
			rules = []db.RuleRecord{}
		}
		writeJSON(w, map[string][]db.RuleRecord{"rules": rules})
	case http.MethodPost:
		var rule db.RuleRecord
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil || rule.Source == "" || rule.Keyword == "" {
			http.Error(w, "source and keyword are required", http.StatusBadRequest)
			return
		}
		created, err := s.Store.Create(r.Context(), rule)
		if err != nil {
			http.Error(w, "could not create rule", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RulesServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/rules/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var rule db.RuleRecord
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		rule.ID = id
		if err := s.Store.Update(r.Context(), rule); err != nil {
			http.Error(w, "could not update rule", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "could not delete rule", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
