package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKeyAuthenticator checks a plaintext key — either against a static
// env-var key (legacy) or against the DB-stored hashed keys.
type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, plaintextKey string) (bool, error)
}

// staticKeyAuth wraps a single plaintext env-var key for backwards compat.
type staticKeyAuth struct{ key string }

func (s staticKeyAuth) Authenticate(_ context.Context, provided string) (bool, error) {
	ok := subtle.ConstantTimeCompare([]byte(provided), []byte(s.key)) == 1
	return ok, nil
}

// StaticKeyAuthenticator wraps a fixed key (ALARM_API_KEY env var).
func StaticKeyAuthenticator(key string) APIKeyAuthenticator { return staticKeyAuth{key: key} }

// multiAuth checks a static key first (cheap), then falls back to the DB.
type multiAuth struct {
	static string
	db     APIKeyAuthenticator
}

func (m multiAuth) Authenticate(ctx context.Context, provided string) (bool, error) {
	if m.static != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(m.static)) == 1 {
		return true, nil
	}
	return m.db.Authenticate(ctx, provided)
}

// CombinedAuthenticator checks the static env-var key first, then the DB.
// Pass staticKey="" to skip the static check.
func CombinedAuthenticator(staticKey string, db APIKeyAuthenticator) APIKeyAuthenticator {
	return multiAuth{static: staticKey, db: db}
}

// RequireAPIKey wraps next with authentication via the provided authenticator.
// The key must be supplied as "Authorization: Bearer <key>" or "X-API-Key: <key>".
func RequireAPIKey(auth APIKeyAuthenticator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ok, err := auth.Authenticate(r.Context(), key)
		if err != nil || !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func extractAPIKey(r *http.Request) string {
	if v := r.Header.Get("X-API-Key"); v != "" {
		return v
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
