package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

type GroupsStore interface {
	List(ctx context.Context) ([]db.Group, error)
	Create(ctx context.Context, name, description string) (db.Group, error)
	Update(ctx context.Context, id int, name, description string, selfService bool) error
	Delete(ctx context.Context, id int) error
	ListMembers(ctx context.Context, groupID int) ([]db.Member, error)
	AddMember(ctx context.Context, groupID, recipientID int) error
	RemoveMember(ctx context.Context, groupID, recipientID int) error
}

type GroupsServer struct {
	Store GroupsStore
}

func NewGroupsServer(store GroupsStore) *GroupsServer {
	return &GroupsServer{Store: store}
}

// HandleCollection serves GET (list) and POST (create) on /api/groups.
func (s *GroupsServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		groups, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "could not list groups", http.StatusInternalServerError)
			return
		}
		if groups == nil {
			groups = []db.Group{}
		}
		writeJSON(w, map[string][]db.Group{"groups": groups})
	case http.MethodPost:
		var payload struct{ Name, Description string }
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		g, err := s.Store.Create(r.Context(), payload.Name, payload.Description)
		if err != nil {
			http.Error(w, "could not create group (name may already exist)", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, g)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleItem serves PUT (update) and DELETE on /api/groups/{id}.
func (s *GroupsServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/groups/", "")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var payload struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			SelfService bool   `json:"self_service"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if err := s.Store.Update(r.Context(), id, payload.Name, payload.Description, payload.SelfService); err != nil {
			http.Error(w, "could not update group", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), id); err != nil {
			http.Error(w, "could not delete group", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleMembers serves GET (list) and POST (add {recipient_id}) on
// /api/groups/{id}/members.
func (s *GroupsServer) HandleMembers(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/groups/", "/members")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		members, err := s.Store.ListMembers(r.Context(), id)
		if err != nil {
			http.Error(w, "could not list members", http.StatusInternalServerError)
			return
		}
		if members == nil {
			members = []db.Member{}
		}
		writeJSON(w, map[string][]db.Member{"members": members})
	case http.MethodPost:
		var payload struct{ RecipientID int `json:"recipient_id"` }
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if err := s.Store.AddMember(r.Context(), id, payload.RecipientID); err != nil {
			http.Error(w, "could not add member", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleMemberItem serves DELETE on /api/groups/{id}/members/{recipientID}.
func (s *GroupsServer) HandleMemberItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	groupID, recipientID, ok := parseGroupMemberPath(r.URL.Path)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if err := s.Store.RemoveMember(r.Context(), groupID, recipientID); err != nil {
		http.Error(w, "could not remove member", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseGroupMemberPath parses "/api/groups/{groupID}/members/{recipientID}".
func parseGroupMemberPath(path string) (groupID, recipientID int, ok bool) {
	rest := strings.TrimPrefix(path, "/api/groups/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 3 || parts[1] != "members" {
		return 0, 0, false
	}
	gid, err1 := strconv.Atoi(parts[0])
	rid, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return gid, rid, true
}
