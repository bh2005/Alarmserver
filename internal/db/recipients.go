package db

import (
	"context"
	"fmt"

	"github.com/bh2005/alarmserver/internal/core"
)

// RecipientStore implements core.RecipientLoader against the recipients/
// groups/group_members tables.
type RecipientStore struct {
	pool *Pool
}

func NewRecipientStore(pool *Pool) *RecipientStore {
	return &RecipientStore{pool: pool}
}

func (s *RecipientStore) LoadGroupMembers(ctx context.Context, groupName string) ([]core.Recipient, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.first_name || ' ' || r.last_name, r.voice, r.mobile, r.email
		FROM recipients r
		JOIN group_members gm ON gm.recipient_id = r.id
		JOIN groups g ON g.id = gm.group_id
		WHERE g.name = $1 AND r.active = TRUE AND gm.active = TRUE
	`, groupName)
	if err != nil {
		return nil, fmt.Errorf("load group members: %w", err)
	}
	defer rows.Close()

	var recipients []core.Recipient
	for rows.Next() {
		var r core.Recipient
		var voice, mobile, email *string
		if err := rows.Scan(&r.ID, &r.Name, &voice, &mobile, &email); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		if voice != nil {
			r.Voice = *voice
		}
		if mobile != nil {
			r.Mobile = *mobile
		}
		if email != nil {
			r.Email = *email
		}
		recipients = append(recipients, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipients: %w", err)
	}
	return recipients, nil
}

// RecipientRecord is the full row shape used by the admin UI (unlike
// core.Recipient, which only carries what the Notifiers need).
type RecipientRecord struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Voice     string `json:"voice"`
	Mobile    string `json:"mobile"`
	Email     string `json:"email"`
	Active    bool   `json:"active"`
}

func (s *RecipientStore) List(ctx context.Context) ([]RecipientRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, first_name, last_name, COALESCE(voice,''), COALESCE(mobile,''), COALESCE(email,''), active
		FROM recipients ORDER BY last_name, first_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}
	defer rows.Close()

	var out []RecipientRecord
	for rows.Next() {
		var r RecipientRecord
		if err := rows.Scan(&r.ID, &r.FirstName, &r.LastName, &r.Voice, &r.Mobile, &r.Email, &r.Active); err != nil {
			return nil, fmt.Errorf("scan recipient: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RecipientStore) Create(ctx context.Context, r RecipientRecord) (RecipientRecord, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO recipients (first_name, last_name, voice, mobile, email, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, r.FirstName, r.LastName, r.Voice, r.Mobile, r.Email, r.Active).Scan(&r.ID)
	if err != nil {
		return RecipientRecord{}, fmt.Errorf("create recipient: %w", err)
	}
	return r, nil
}

func (s *RecipientStore) Update(ctx context.Context, r RecipientRecord) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE recipients SET first_name=$1, last_name=$2, voice=$3, mobile=$4, email=$5, active=$6
		WHERE id = $7
	`, r.FirstName, r.LastName, r.Voice, r.Mobile, r.Email, r.Active, r.ID)
	if err != nil {
		return fmt.Errorf("update recipient: %w", err)
	}
	return nil
}

func (s *RecipientStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM recipients WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recipient: %w", err)
	}
	return nil
}

// FindByMobile returns the recipient whose mobile number matches after
// stripping spaces and normalizing leading zeros / country code.
// Returns nil, nil if not found.
func (s *RecipientStore) FindByMobile(ctx context.Context, mobile string) (*RecipientRecord, error) {
	norm := normalizeMobile(mobile)
	rows, err := s.pool.Query(ctx, `
		SELECT id, first_name, last_name, COALESCE(voice,''), COALESCE(mobile,''), COALESCE(email,''), active
		FROM recipients WHERE active = TRUE AND mobile IS NOT NULL AND mobile != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("find by mobile: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r RecipientRecord
		if err := rows.Scan(&r.ID, &r.FirstName, &r.LastName, &r.Voice, &r.Mobile, &r.Email, &r.Active); err != nil {
			return nil, err
		}
		if normalizeMobile(r.Mobile) == norm {
			return &r, nil
		}
	}
	return nil, rows.Err()
}

// normalizeMobile strips whitespace/dashes and maps leading 00 → +.
func normalizeMobile(s string) string {
	var b []byte
	for _, c := range s {
		if c >= '0' && c <= '9' || c == '+' {
			b = append(b, byte(c))
		}
	}
	r := string(b)
	if len(r) > 2 && r[:2] == "00" {
		r = "+" + r[2:]
	}
	return r
}

// LatestUnackedEventForRecipient returns the event_log_id of the most
// recent event that was dispatched to the recipient and has no ACK yet.
// Returns 0, nil if there is no such event.
func (s *RecipientStore) LatestUnackedEventForRecipient(ctx context.Context, recipientID int) (int, error) {
	var id int
	err := s.pool.QueryRow(ctx, `
		SELECT el.id
		FROM event_log el
		WHERE el.status NOT IN ('MAINTENANCE')
		  AND NOT EXISTS (
			SELECT 1 FROM acknowledgements a WHERE a.event_log_id = el.id
		  )
		ORDER BY el.dispatched_at DESC
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		return 0, nil // no unacked event
	}
	return id, nil
}
