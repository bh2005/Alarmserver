package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

// APIKeyManager is implemented by db.APIKeyStore.
type APIKeyManager interface {
	List(ctx context.Context) ([]db.APIKey, error)
	Create(ctx context.Context, name, plaintextKey string) (db.APIKey, error)
	SetActive(ctx context.Context, id int, active bool) error
	Delete(ctx context.Context, id int) error
}

// APIKeysServer exposes API key management endpoints (admin only).
type APIKeysServer struct {
	Store APIKeyManager
}

func NewAPIKeysServer(store APIKeyManager) *APIKeysServer {
	return &APIKeysServer{Store: store}
}

// HandleCollection serves GET (list) and POST (create) on /api/apikeys.
func (s *APIKeysServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list api keys", http.StatusInternalServerError)
			return
		}
		if keys == nil {
			keys = []db.APIKey{}
		}
		writeJSON(w, map[string][]db.APIKey{"api_keys": keys})

	case http.MethodPost:
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		plaintext := randomHex32()
		key, err := s.Store.Create(r.Context(), payload.Name, plaintext)
		if err != nil {
			http.Error(w, "could not create api key (name may already exist)", http.StatusConflict)
			return
		}
		// Return the plaintext key exactly once — it is never retrievable again.
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{
			"id":         key.ID,
			"name":       key.Name,
			"active":     key.Active,
			"created_at": key.CreatedAt,
			"key":        plaintext,
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleItem serves POST /{id}/active and DELETE /{id} on /api/apikeys/.
func (s *APIKeysServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/active") {
		s.handleSetActive(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		s.handleDelete(w, r)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *APIKeysServer) handleSetActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseIDFromPath(r.URL.Path, "/api/apikeys/", "/active")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	var payload struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.Store.SetActive(r.Context(), id, payload.Active); err != nil {
		http.Error(w, "could not update api key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *APIKeysServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/apikeys/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if err := s.Store.Delete(r.Context(), id); err != nil {
		http.Error(w, "could not delete api key", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
