package api

import (
	"context"
	"net/http"

	"github.com/bh2005/alarmserver/internal/db"
)

type DashboardAuditStore interface {
	List(ctx context.Context, limit int) ([]db.EventLogEntry, error)
	ListOpen(ctx context.Context) ([]db.EventLogEntry, error)
}

type DashboardMaintenanceStore interface {
	Active(ctx context.Context) ([]db.MaintenanceWindow, error)
}

type DashboardHeartbeatStore interface {
	List(ctx context.Context) ([]db.Heartbeat, error)
}

type DashboardServer struct {
	Audit       DashboardAuditStore
	Maintenance DashboardMaintenanceStore
	Heartbeats  DashboardHeartbeatStore
}

func NewDashboardServer(audit DashboardAuditStore, maint DashboardMaintenanceStore, hb DashboardHeartbeatStore) *DashboardServer {
	return &DashboardServer{Audit: audit, Maintenance: maint, Heartbeats: hb}
}

func (s *DashboardServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	recent, err := s.Audit.List(ctx, 20)
	if err != nil {
		http.Error(w, "could not load events", http.StatusInternalServerError)
		return
	}
	open, err := s.Audit.ListOpen(ctx)
	if err != nil {
		http.Error(w, "could not load open events", http.StatusInternalServerError)
		return
	}
	maintenance, err := s.Maintenance.Active(ctx)
	if err != nil {
		http.Error(w, "could not load maintenance", http.StatusInternalServerError)
		return
	}
	heartbeats, err := s.Heartbeats.List(ctx)
	if err != nil {
		http.Error(w, "could not load heartbeats", http.StatusInternalServerError)
		return
	}

	// Ensure non-nil slices for clean JSON output.
	if recent == nil {
		recent = []db.EventLogEntry{}
	}
	if open == nil {
		open = []db.EventLogEntry{}
	}
	if maintenance == nil {
		maintenance = []db.MaintenanceWindow{}
	}
	if heartbeats == nil {
		heartbeats = []db.Heartbeat{}
	}

	writeJSON(w, map[string]any{
		"recent_events":      recent,
		"open_events":        open,
		"active_maintenance": maintenance,
		"heartbeats":         heartbeats,
	})
}
