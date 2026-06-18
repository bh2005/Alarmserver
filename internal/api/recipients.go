package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bh2005/alarmserver/internal/db"
)

type RecipientsStore interface {
	List(ctx context.Context) ([]db.RecipientRecord, error)
	Create(ctx context.Context, r db.RecipientRecord) (db.RecipientRecord, error)
	Update(ctx context.Context, r db.RecipientRecord) error
	Delete(ctx context.Context, id int) error
}

type RecipientsServer struct {
	Store RecipientsStore
}

func NewRecipientsServer(store RecipientsStore) *RecipientsServer {
	return &RecipientsServer{Store: store}
}

func (s *RecipientsServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		recipients, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list recipients", http.StatusInternalServerError)
			return
		}
		if recipients == nil {
			recipients = []db.RecipientRecord{}
		}
		writeJSON(w, map[string][]db.RecipientRecord{"recipients": recipients})
	case http.MethodPost:
		var rec db.RecipientRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil || rec.FirstName == "" || rec.LastName == "" {
			http.Error(w, "first_name and last_name are required", http.StatusBadRequest)
			return
		}
		created, err := s.Store.Create(r.Context(), rec)
		if err != nil {
			http.Error(w, "could not create recipient", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RecipientsServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/recipients/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var rec db.RecipientRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		rec.ID = id
		if err := s.Store.Update(r.Context(), rec); err != nil {
			http.Error(w, "could not update recipient", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "could not delete recipient", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
