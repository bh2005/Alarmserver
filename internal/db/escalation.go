package db

import (
	"context"
	"fmt"

	"github.com/bh2005/alarmserver/internal/core"
)

type EscalationStore struct {
	pool *Pool
}

func NewEscalationStore(pool *Pool) *EscalationStore {
	return &EscalationStore{pool: pool}
}

// EscalationPolicyRecord is the admin-API view of a policy (includes steps with group names).
type EscalationPolicyRecord struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Steps       []EscalationStepRecord `json:"steps"`
}

type EscalationStepRecord struct {
	ID       int      `json:"id"`
	PolicyID int      `json:"policy_id"`
	Step     int      `json:"step"`
	GroupID  int      `json:"group_id"`
	Group    string   `json:"group"`
	DelaySec int      `json:"delay_sec"`
	Channels []string `json:"channels"`
}

// GetPolicy implements core.EscalationPolicyLoader — used by the Processor.
func (s *EscalationStore) GetPolicy(ctx context.Context, id int) (*core.EscalationPolicy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT es.step, g.name, es.delay_sec, es.channels
		FROM escalation_steps es
		JOIN groups g ON g.id = es.group_id
		WHERE es.policy_id = $1
		ORDER BY es.step
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get escalation policy steps: %w", err)
	}
	defer rows.Close()

	policy := &core.EscalationPolicy{ID: id}
	for rows.Next() {
		var step core.EscalationStep
		if err := rows.Scan(&step.Step, &step.GroupName, &step.DelaySec, &step.Channels); err != nil {
			return nil, fmt.Errorf("scan escalation step: %w", err)
		}
		policy.Steps = append(policy.Steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// fetch name
	_ = s.pool.QueryRow(ctx, `SELECT name FROM escalation_policies WHERE id=$1`, id).Scan(&policy.Name)
	return policy, nil
}

func (s *EscalationStore) List(ctx context.Context) ([]EscalationPolicyRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, COALESCE(description,'') FROM escalation_policies ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list escalation policies: %w", err)
	}
	defer rows.Close()

	var out []EscalationPolicyRecord
	for rows.Next() {
		var r EscalationPolicyRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		r.Steps = []EscalationStepRecord{}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load steps for each policy.
	for i := range out {
		steps, err := s.listSteps(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Steps = steps
	}
	return out, nil
}

func (s *EscalationStore) Get(ctx context.Context, id int) (*EscalationPolicyRecord, error) {
	var r EscalationPolicyRecord
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(description,'') FROM escalation_policies WHERE id=$1`, id,
	).Scan(&r.ID, &r.Name, &r.Description)
	if err != nil {
		return nil, fmt.Errorf("get escalation policy %d: %w", id, err)
	}
	steps, err := s.listSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	r.Steps = steps
	return &r, nil
}

func (s *EscalationStore) Create(ctx context.Context, name, description string) (EscalationPolicyRecord, error) {
	var r EscalationPolicyRecord
	r.Name = name
	r.Description = description
	err := s.pool.QueryRow(ctx,
		`INSERT INTO escalation_policies (name, description) VALUES ($1,$2) RETURNING id`,
		name, nullStr(description),
	).Scan(&r.ID)
	if err != nil {
		return EscalationPolicyRecord{}, fmt.Errorf("create escalation policy: %w", err)
	}
	r.Steps = []EscalationStepRecord{}
	return r, nil
}

func (s *EscalationStore) Update(ctx context.Context, id int, name, description string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE escalation_policies SET name=$1, description=$2 WHERE id=$3`,
		name, nullStr(description), id)
	if err != nil {
		return fmt.Errorf("update escalation policy: %w", err)
	}
	return nil
}

func (s *EscalationStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM escalation_policies WHERE id=$1`, id)
	return err
}

func (s *EscalationStore) AddStep(ctx context.Context, policyID, step, groupID, delaySec int, channels []string) (EscalationStepRecord, error) {
	if channels == nil {
		channels = []string{}
	}
	var r EscalationStepRecord
	r.PolicyID = policyID
	r.Step = step
	r.GroupID = groupID
	r.DelaySec = delaySec
	r.Channels = channels
	err := s.pool.QueryRow(ctx, `
		INSERT INTO escalation_steps (policy_id, step, group_id, delay_sec, channels)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		policyID, step, groupID, delaySec, channels,
	).Scan(&r.ID)
	if err != nil {
		return EscalationStepRecord{}, fmt.Errorf("add escalation step: %w", err)
	}
	_ = s.pool.QueryRow(ctx, `SELECT name FROM groups WHERE id=$1`, groupID).Scan(&r.Group)
	return r, nil
}

func (s *EscalationStore) UpdateStep(ctx context.Context, stepID, groupID, delaySec int, channels []string) error {
	if channels == nil {
		channels = []string{}
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE escalation_steps SET group_id=$1, delay_sec=$2, channels=$3 WHERE id=$4`,
		groupID, delaySec, channels, stepID)
	return err
}

func (s *EscalationStore) DeleteStep(ctx context.Context, stepID int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM escalation_steps WHERE id=$1`, stepID)
	return err
}

func (s *EscalationStore) listSteps(ctx context.Context, policyID int) ([]EscalationStepRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT es.id, es.policy_id, es.step, es.group_id, g.name, es.delay_sec, es.channels
		FROM escalation_steps es
		JOIN groups g ON g.id = es.group_id
		WHERE es.policy_id = $1
		ORDER BY es.step
	`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list escalation steps: %w", err)
	}
	defer rows.Close()

	var steps []EscalationStepRecord
	for rows.Next() {
		var r EscalationStepRecord
		if err := rows.Scan(&r.ID, &r.PolicyID, &r.Step, &r.GroupID, &r.Group, &r.DelaySec, &r.Channels); err != nil {
			return nil, err
		}
		steps = append(steps, r)
	}
	if steps == nil {
		steps = []EscalationStepRecord{}
	}
	return steps, rows.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
