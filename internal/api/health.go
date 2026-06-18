package api

import (
	"context"
	"encoding/json"
	"net/http"
)

// HealthChecker reports the status of one subsystem (e.g. "ok" or an error
// message). Implementations are pluggable per subsystem (db, voice, sms, ...).
type HealthChecker func(ctx context.Context) string

// HealthHandler builds an http.HandlerFunc for GET /health.
// Returns HTTP 200 when all checks pass, HTTP 503 when any check fails.
// Check_MK and other monitors can rely on the HTTP status code alone.
func HealthHandler(checks map[string]HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := make(map[string]string, len(checks))
		allOK := true
		for name, check := range checks {
			result := check(r.Context())
			status[name] = result
			if result != "ok" {
				allOK = false
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if !allOK {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(status)
	}
}
