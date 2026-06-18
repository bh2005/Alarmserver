package api

import (
	"context"
	"encoding/json"
	"net/http"
)

// HealthChecker reports the status of one subsystem (e.g. "ok" or an error
// message). Implementations are pluggable per subsystem (db, voice, sms, ...).
type HealthChecker func(ctx context.Context) string

// HealthHandler builds an http.HandlerFunc for GET /health that reports the
// status of each registered subsystem.
func HealthHandler(checks map[string]HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := make(map[string]string, len(checks))
		for name, check := range checks {
			status[name] = check(r.Context())
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}
