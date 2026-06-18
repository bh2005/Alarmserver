package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bh2005/alarmserver/internal/auth"
)

// UsersStore is implemented by db.UserStore.
type UsersStore interface {
	List(ctx context.Context) ([]auth.User, error)
	Create(ctx context.Context, username, passwordHash, role string) (auth.User, error)
	SetActive(ctx context.Context, id int, active bool) error
	SetRecipientID(ctx context.Context, id int, recipientID *int) error
	Delete(ctx context.Context, id int) error
}

// UsersServer exposes admin-only user management. Kept separate from
// SessionServer so login/me (used by every authenticated request) and
// administration (admin-only) have distinct, easily auditable handlers.
type UsersServer struct {
	Store UsersStore
}

func NewUsersServer(store UsersStore) *UsersServer {
	return &UsersServer{Store: store}
}

func (s *UsersServer) HandleList(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.List(r.Context())
	if err != nil {
		http.Error(w, "could not list users", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []auth.User{}
	}
	writeJSON(w, map[string][]auth.User{"users": users})
}

func (s *UsersServer) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}
	if payload.Role != auth.RoleAdmin && payload.Role != auth.RoleViewer {
		payload.Role = auth.RoleViewer
	}

	hash, err := auth.HashPassword(payload.Password)
	if err != nil {
		http.Error(w, "could not hash password", http.StatusInternalServerError)
		return
	}
	user, err := s.Store.Create(r.Context(), payload.Username, hash, payload.Role)
	if err != nil {
		http.Error(w, "could not create user (username may already exist)", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, user)
}

// HandleSetActive and HandleDelete operate on /api/users/{id}/... paths,
// parsed manually since main.go's mux predates Go's pattern wildcards here.
func (s *UsersServer) HandleSetActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseIDFromPath(r.URL.Path, "/api/users/", "/active")
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
		http.Error(w, "could not update user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *UsersServer) HandleSetRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseIDFromPath(r.URL.Path, "/api/users/", "/recipient")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	var payload struct {
		RecipientID *int `json:"recipient_id"` // null to unlink
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.Store.SetRecipientID(r.Context(), id, payload.RecipientID); err != nil {
		http.Error(w, "could not update user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *UsersServer) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseIDFromPath(r.URL.Path, "/api/users/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if err := s.Store.Delete(r.Context(), id); err != nil {
		http.Error(w, "could not delete user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseIDFromPath extracts the numeric ID from a path like
// "/api/users/{id}" or "/api/users/{id}/active" (suffix stripped first).
func parseIDFromPath(path, prefix, suffix string) (int, bool) {
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	rest := strings.TrimPrefix(path, prefix)
	if suffix != "" {
		if !strings.HasSuffix(rest, suffix) {
			return 0, false
		}
		rest = strings.TrimSuffix(rest, suffix)
	}
	rest = strings.Trim(rest, "/")
	id, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return id, true
}
