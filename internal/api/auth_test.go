package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAPIKey_ValidBearerToken(t *testing.T) {
	called := false
	h := RequireAPIKey(StaticKeyAuthenticator("secret123"), func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("expected request to pass through, got code=%d called=%v", rec.Code, called)
	}
}

func TestRequireAPIKey_ValidXAPIKeyHeader(t *testing.T) {
	h := RequireAPIKey(StaticKeyAuthenticator("secret123"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "secret123")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKey_WrongKey(t *testing.T) {
	called := false
	h := RequireAPIKey(StaticKeyAuthenticator("secret123"), func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "wrong")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if called {
		t.Fatal("handler should not be called for wrong key")
	}
}

func TestRequireAPIKey_MissingHeader(t *testing.T) {
	h := RequireAPIKey(StaticKeyAuthenticator("secret123"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAPIKey_EmptyConfiguredKeyNeverMatches(t *testing.T) {
	// Defensive: even an empty "provided" key must not match an empty
	// configured key (callers should never wrap with an empty key, but
	// the middleware itself must not silently allow it).
	h := RequireAPIKey(StaticKeyAuthenticator(""), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
