// Package auth implements the UI's user login (username/password + JWT
// session token). This is separate from the machine-to-machine API key in
// internal/api/auth.go — different audience, different threat model.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

// LDAPAuthenticator is implemented by the ldap package and used as a fallback
// when local DB auth fails. Returning a role string on success.
type LDAPAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (role string, err error)
}

// User is the public, password-free view of a login account.
type User struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
	RecipientID *int   `json:"recipient_id"` // optional link to a recipients row for self-service
}

// UserRecord additionally carries the password hash; only the store layer
// and HashPassword/CheckPassword should ever see it.
type UserRecord struct {
	User
	PasswordHash string
}

// UserStore looks up accounts by username for login.
type UserStore interface {
	FindByUsername(ctx context.Context, username string) (*UserRecord, error)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Claims is the JWT payload issued on login.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Service issues and verifies session tokens.
type Service struct {
	Users    UserStore
	Secret   []byte
	TTL      time.Duration
	LDAPAuth LDAPAuthenticator // optional; nil = no LDAP fallback
}

func NewService(users UserStore, secret string, ttl time.Duration) *Service {
	if ttl == 0 {
		ttl = 12 * time.Hour
	}
	return &Service{Users: users, Secret: []byte(secret), TTL: ttl}
}

// Login verifies username/password against the local DB first, then LDAP.
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	// Local DB auth.
	record, err := s.Users.FindByUsername(ctx, username)
	if err == nil && record != nil && record.Active && CheckPassword(record.PasswordHash, password) {
		return s.issueToken(record.Username, record.Role)
	}

	// LDAP fallback.
	if s.LDAPAuth != nil {
		role, err := s.LDAPAuth.Authenticate(ctx, username, password)
		if err == nil {
			return s.issueToken(username, role)
		}
	}

	return "", ErrInvalidCredentials
}

func (s *Service) issueToken(username, role string) (string, error) {
	claims := Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.TTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.Secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning its claims.
func (s *Service) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return s.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return claims, nil
}
