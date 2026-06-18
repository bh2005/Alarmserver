package db

import (
	"context"
	"fmt"

	"github.com/bh2005/alarmserver/internal/auth"
)

// UserStore implements auth.UserStore plus the CRUD operations the UI's
// user management page needs.
type UserStore struct {
	pool *Pool
}

func NewUserStore(pool *Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) FindByUsername(ctx context.Context, username string) (*auth.UserRecord, error) {
	var r auth.UserRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, role, active, recipient_id
		FROM users WHERE username = $1
	`, username).Scan(&r.ID, &r.Username, &r.PasswordHash, &r.Role, &r.Active, &r.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &r, nil
}

func (s *UserStore) Count(ctx context.Context) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (s *UserStore) Create(ctx context.Context, username, passwordHash, role string) (auth.User, error) {
	var u auth.User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)
		RETURNING id, username, role, active, recipient_id
	`, username, passwordHash, role).Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.RecipientID)
	if err != nil {
		return auth.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (s *UserStore) List(ctx context.Context) ([]auth.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, role, active, recipient_id FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []auth.User
	for rows.Next() {
		var u auth.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Active, &u.RecipientID); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserStore) SetActive(ctx context.Context, id int, active bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET active = $1 WHERE id = $2`, active, id)
	if err != nil {
		return fmt.Errorf("set user active: %w", err)
	}
	return nil
}

func (s *UserStore) SetRecipientID(ctx context.Context, id int, recipientID *int) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET recipient_id = $1 WHERE id = $2`, recipientID, id)
	if err != nil {
		return fmt.Errorf("set user recipient_id: %w", err)
	}
	return nil
}

func (s *UserStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
