package db

import (
	"context"
	"fmt"
)

// SelfServiceGroup is what GET /api/me/groups returns per group.
type SelfServiceGroup struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"` // this recipient's current active status in the group
}

// SelfServiceStore provides self-service read/write access for individual
// recipients. All methods verify group.self_service = TRUE.
type SelfServiceStore struct {
	pool *Pool
}

func NewSelfServiceStore(pool *Pool) *SelfServiceStore {
	return &SelfServiceStore{pool: pool}
}

// GroupsForRecipient returns all self-service groups the recipient belongs to,
// with the recipient's current active status in each group.
func (s *SelfServiceStore) GroupsForRecipient(ctx context.Context, recipientID int) ([]SelfServiceGroup, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT g.id, g.name, COALESCE(g.description, ''), gm.active
		FROM group_members gm
		JOIN groups g ON g.id = gm.group_id
		WHERE gm.recipient_id = $1 AND g.self_service = TRUE
		ORDER BY g.name
	`, recipientID)
	if err != nil {
		return nil, fmt.Errorf("self-service groups: %w", err)
	}
	defer rows.Close()

	var out []SelfServiceGroup
	for rows.Next() {
		var g SelfServiceGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Active); err != nil {
			return nil, fmt.Errorf("scan self-service group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// SetActive sets the recipient's active flag in a self-service group.
// Returns an error if the group is not self-service or the recipient is not a member.
func (s *SelfServiceStore) SetActive(ctx context.Context, groupID, recipientID int, active bool) error {
	res, err := s.pool.Exec(ctx, `
		UPDATE group_members gm
		SET active = $1
		FROM groups g
		WHERE gm.group_id = g.id
		  AND gm.group_id = $2
		  AND gm.recipient_id = $3
		  AND g.self_service = TRUE
	`, active, groupID, recipientID)
	if err != nil {
		return fmt.Errorf("set self-service active: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("group not found or not self-service")
	}
	return nil
}
