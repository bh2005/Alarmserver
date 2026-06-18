package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

type EscalationStore interface {
	List(ctx context.Context) ([]db.EscalationPolicyRecord, error)
	Get(ctx context.Context, id int) (*db.EscalationPolicyRecord, error)
	Create(ctx context.Context, name, description string) (db.EscalationPolicyRecord, error)
	Update(ctx context.Context, id int, name, description string) error
	Delete(ctx context.Context, id int) error
	AddStep(ctx context.Context, policyID, step, groupID, delaySec int, channels []string) (db.EscalationStepRecord, error)
	UpdateStep(ctx context.Context, stepID, groupID, delaySec int, channels []string) error
	DeleteStep(ctx context.Context, stepID int) error
}

type EscalationServer struct{ Store EscalationStore }

func NewEscalationServer(store EscalationStore) *EscalationServer {
	return &EscalationServer{Store: store}
}

// HandleCollection — GET /api/escalation-policies  POST /api/escalation-policies
func (s *EscalationServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.Store.List(r.Context())
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []db.EscalationPolicyRecord{}
		}
		writeJSON(w, list)

	case http.MethodPost:
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		rec, err := s.Store.Create(r.Context(), body.Name, body.Description)
		if err != nil {
			http.Error(w, "create failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, rec)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleItem — GET/PUT/DELETE /api/escalation-policies/{id}
func (s *EscalationServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDFromPath(r.URL.Path, "/api/escalation-policies/", "")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rec, err := s.Store.Get(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, rec)

	case http.MethodPut:
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		if err := s.Store.Update(r.Context(), id, body.Name, body.Description); err != nil {
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

// HandleSteps — POST /api/escalation-policies/{id}/steps
func (s *EscalationServer) HandleSteps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	policyID, ok := parseIDFromPath(r.URL.Path, "/api/escalation-policies/", "/steps")
	if !ok {
		http.Error(w, "invalid policy id", http.StatusBadRequest)
		return
	}
	var body struct {
		Step     int      `json:"step"`
		GroupID  int      `json:"group_id"`
		DelaySec int      `json:"delay_sec"`
		Channels []string `json:"channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GroupID == 0 {
		http.Error(w, "group_id required", http.StatusBadRequest)
		return
	}
	rec, err := s.Store.AddStep(r.Context(), policyID, body.Step, body.GroupID, body.DelaySec, body.Channels)
	if err != nil {
		http.Error(w, "add step failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, rec)
}

// HandleStepItem — PUT/DELETE /api/escalation-policies/{pid}/steps/{sid}
func (s *EscalationServer) HandleStepItem(w http.ResponseWriter, r *http.Request) {
	stepID, ok := escalationStepIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "invalid step id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var body struct {
			GroupID  int      `json:"group_id"`
			DelaySec int      `json:"delay_sec"`
			Channels []string `json:"channels"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GroupID == 0 {
			http.Error(w, "group_id required", http.StatusBadRequest)
			return
		}
		if err := s.Store.UpdateStep(r.Context(), stepID, body.GroupID, body.DelaySec, body.Channels); err != nil {
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := s.Store.DeleteStep(r.Context(), stepID); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// escalationStepIDFromPath parses the last segment of paths like
// /api/escalation-policies/42/steps/7  →  7
func escalationStepIDFromPath(path string) (int, bool) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return 0, false
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	return id, err == nil
}
