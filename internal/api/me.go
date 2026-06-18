package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bh2005/alarmserver/internal/auth"
	"github.com/bh2005/alarmserver/internal/db"
)

// MeUserStore is the subset of db.UserStore used by MeServer.
type MeUserStore interface {
	FindByUsername(ctx context.Context, username string) (*auth.UserRecord, error)
}

// MeSelfServiceStore is implemented by db.SelfServiceStore.
type MeSelfServiceStore interface {
	GroupsForRecipient(ctx context.Context, recipientID int) ([]db.SelfServiceGroup, error)
	SetActive(ctx context.Context, groupID, recipientID int, active bool) error
}

// MeServer exposes self-service group endpoints for any logged-in user.
type MeServer struct {
	Users  MeUserStore
	Groups MeSelfServiceStore
}

func NewMeServer(users MeUserStore, groups MeSelfServiceStore) *MeServer {
	return &MeServer{Users: users, Groups: groups}
}

// HandleGroups serves GET /api/me/groups — the self-service groups the
// current user's linked recipient belongs to, with their active status.
func (s *MeServer) HandleGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	recipientID, ok := s.recipientID(w, r)
	if !ok {
		return
	}
	groups, err := s.Groups.GroupsForRecipient(r.Context(), recipientID)
	if err != nil {
		http.Error(w, "could not load groups", http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []db.SelfServiceGroup{}
	}
	writeJSON(w, map[string][]db.SelfServiceGroup{"groups": groups})
}

// HandleGroupActive serves POST /api/me/groups/{id}/active — toggle own
// active status in a self-service group.
func (s *MeServer) HandleGroupActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	recipientID, ok := s.recipientID(w, r)
	if !ok {
		return
	}
	groupID, ok := parseIDFromPath(r.URL.Path, "/api/me/groups/", "/active")
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
	if err := s.Groups.SetActive(r.Context(), groupID, recipientID, payload.Active); err != nil {
		http.Error(w, "group not found or not self-service", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recipientID extracts the current user's linked recipient ID from the session.
// Returns (0, false) and writes an error response if the user has no linked recipient.
func (s *MeServer) recipientID(w http.ResponseWriter, r *http.Request) (int, bool) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	user, err := s.Users.FindByUsername(r.Context(), claims.Username)
	if err != nil || user == nil || user.RecipientID == nil {
		http.Error(w, "no recipient linked to this account — ask an admin to link your account to a recipient", http.StatusForbidden)
		return 0, false
	}
	return *user.RecipientID, true
}
