package db

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// APIKey is the public view of a key (no hash).
type APIKey struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyStore manages API keys in PostgreSQL.
type APIKeyStore struct {
	pool *Pool
}

func NewAPIKeyStore(pool *Pool) *APIKeyStore {
	return &APIKeyStore{pool: pool}
}

func (s *APIKeyStore) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, active, created_at FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Active, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Create hashes the plaintext key and stores it. Returns the created APIKey.
func (s *APIKeyStore) Create(ctx context.Context, name, plaintextKey string) (APIKey, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextKey), bcrypt.DefaultCost)
	if err != nil {
		return APIKey{}, fmt.Errorf("hash api key: %w", err)
	}
	var k APIKey
	err = s.pool.QueryRow(ctx, `
		INSERT INTO api_keys (name, key_hash) VALUES ($1, $2)
		RETURNING id, name, active, created_at
	`, name, string(hash)).Scan(&k.ID, &k.Name, &k.Active, &k.CreatedAt)
	if err != nil {
		return APIKey{}, fmt.Errorf("create api key: %w", err)
	}
	return k, nil
}

func (s *APIKeyStore) SetActive(ctx context.Context, id int, active bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE api_keys SET active = $1 WHERE id = $2`, active, id)
	if err != nil {
		return fmt.Errorf("set api key active: %w", err)
	}
	return nil
}

func (s *APIKeyStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	return nil
}

// Authenticate checks a plaintext key against all active hashes.
// Returns true on match. bcrypt is slow by design; results are not cached
// because alarm ingestion calls are rare and latency is not critical.
func (s *APIKeyStore) Authenticate(ctx context.Context, plaintextKey string) (bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT key_hash FROM api_keys WHERE active = TRUE`)
	if err != nil {
		return false, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintextKey)) == nil {
			return true, nil
		}
	}
	return false, rows.Err()
}
