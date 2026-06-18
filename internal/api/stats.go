package api

import (
	"context"
	"net/http"

	"github.com/bh2005/alarmserver/internal/db"
)

type StatsStoreIface interface {
	Get(ctx context.Context) (db.Stats, error)
}

type StatsServer struct {
	Store StatsStoreIface
}

func NewStatsServer(store StatsStoreIface) *StatsServer {
	return &StatsServer{Store: store}
}

func (s *StatsServer) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats, err := s.Store.Get(r.Context())
	if err != nil {
		http.Error(w, "could not load stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}
