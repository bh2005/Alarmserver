package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bh2005/alarmserver/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "session_claims"

// SessionServer exposes login/me and the session middleware used to
// protect the UI-facing CRUD endpoints (rules, groups, recipients, ...).
// This is independent of the machine API key in auth.go.
type SessionServer struct {
	Auth *auth.Service
}

func NewSessionServer(svc *auth.Service) *SessionServer {
	return &SessionServer{Auth: svc}
}

func (s *SessionServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	token, err := s.Auth.Login(r.Context(), payload.Username, payload.Password)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func (s *SessionServer) HandleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{"username": claims.Username, "role": claims.Role})
}

// RequireSession accepts any valid, non-expired session token.
func (s *SessionServer) RequireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := s.Auth.Verify(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// HasValidSession reports whether the request carries a valid session
// token, without writing any response. Used to build endpoints reachable
// by both the UI (session token) and machine clients (API key) — e.g.
// announcements and acknowledgements.
func (s *SessionServer) HasValidSession(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		return false
	}
	_, err := s.Auth.Verify(token)
	return err == nil
}

// RequireAdmin additionally requires the admin role.
func (s *SessionServer) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.RequireSession(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := claimsFromContext(r.Context())
		if claims.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func claimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}
