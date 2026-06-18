package db

import (
	"context"
	"fmt"
)

type Group struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SelfService bool   `json:"self_service"`
	MemberCount int    `json:"member_count"`
}

type GroupStore struct {
	pool *Pool
}

func NewGroupStore(pool *Pool) *GroupStore {
	return &GroupStore{pool: pool}
}

func (s *GroupStore) List(ctx context.Context) ([]Group, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT g.id, g.name, COALESCE(g.description, ''), g.self_service, COUNT(gm.recipient_id)
		FROM groups g
		LEFT JOIN group_members gm ON gm.group_id = g.id
		GROUP BY g.id, g.name, g.description, g.self_service
		ORDER BY g.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.SelfService, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *GroupStore) Create(ctx context.Context, name, description string) (Group, error) {
	var g Group
	err := s.pool.QueryRow(ctx, `
		INSERT INTO groups (name, description) VALUES ($1, $2)
		RETURNING id, name, COALESCE(description, ''), self_service, 0
	`, name, description).Scan(&g.ID, &g.Name, &g.Description, &g.SelfService, &g.MemberCount)
	if err != nil {
		return Group{}, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

func (s *GroupStore) Update(ctx context.Context, id int, name, description string, selfService bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE groups SET name = $1, description = $2, self_service = $3 WHERE id = $4`,
		name, description, selfService, id)
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	return nil
}

func (s *GroupStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	return nil
}

// Member is a recipient as seen from a group's membership list.
type Member struct {
	RecipientID int    `json:"recipient_id"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
}

func (s *GroupStore) ListMembers(ctx context.Context, groupID int) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.first_name || ' ' || r.last_name, gm.active
		FROM group_members gm
		JOIN recipients r ON r.id = gm.recipient_id
		WHERE gm.group_id = $1
		ORDER BY r.last_name, r.first_name
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.RecipientID, &m.Name, &m.Active); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *GroupStore) AddMember(ctx context.Context, groupID, recipientID int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO group_members (group_id, recipient_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, groupID, recipientID)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (s *GroupStore) RemoveMember(ctx context.Context, groupID, recipientID int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM group_members WHERE group_id = $1 AND recipient_id = $2`, groupID, recipientID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}
