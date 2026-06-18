package auth

import (
	"context"
	"testing"
	"time"
)

type fakeUserStore struct {
	users map[string]*UserRecord
}

func (f *fakeUserStore) FindByUsername(ctx context.Context, username string) (*UserRecord, error) {
	return f.users[username], nil
}

func newFakeStore(t *testing.T, username, password, role string, active bool) *fakeUserStore {
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return &fakeUserStore{users: map[string]*UserRecord{
		username: {User: User{ID: 1, Username: username, Role: role, Active: active}, PasswordHash: hash},
	}}
}

func TestLogin_Success(t *testing.T) {
	store := newFakeStore(t, "alice", "correct-password", RoleAdmin, true)
	svc := NewService(store, "test-secret", time.Hour)

	token, err := svc.Login(context.Background(), "alice", "correct-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if claims.Username != "alice" || claims.Role != RoleAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newFakeStore(t, "alice", "correct-password", RoleAdmin, true)
	svc := NewService(store, "test-secret", time.Hour)

	_, err := svc.Login(context.Background(), "alice", "wrong-password")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	store := &fakeUserStore{users: map[string]*UserRecord{}}
	svc := NewService(store, "test-secret", time.Hour)

	_, err := svc.Login(context.Background(), "ghost", "whatever")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	store := newFakeStore(t, "alice", "correct-password", RoleAdmin, false)
	svc := NewService(store, "test-secret", time.Hour)

	_, err := svc.Login(context.Background(), "alice", "correct-password")
	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for inactive user, got %v", err)
	}
}

func TestVerify_RejectsTokenSignedWithDifferentSecret(t *testing.T) {
	store := newFakeStore(t, "alice", "correct-password", RoleAdmin, true)
	svc1 := NewService(store, "secret-one", time.Hour)
	svc2 := NewService(store, "secret-two", time.Hour)

	token, err := svc1.Login(context.Background(), "alice", "correct-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc2.Verify(token); err == nil {
		t.Fatal("expected verification to fail with a different secret")
	}
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	store := newFakeStore(t, "alice", "correct-password", RoleAdmin, true)
	svc := NewService(store, "test-secret", -time.Hour) // already expired

	token, err := svc.Login(context.Background(), "alice", "correct-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.Verify(token); err == nil {
		t.Fatal("expected verification to fail for an expired token")
	}
}

func TestCheckPassword_RejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("right")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to be rejected")
	}
	if !CheckPassword(hash, "right") {
		t.Fatal("expected right password to be accepted")
	}
}
