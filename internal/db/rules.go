package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bh2005/alarmserver/internal/core"
)

// RuleStore implements core.RuleLoader against the PostgreSQL schema
// defined in migrations/0001_init.sql.
type RuleStore struct {
	pool *Pool
}

func NewRuleStore(pool *Pool) *RuleStore {
	return &RuleStore{pool: pool}
}

func (s *RuleStore) LoadRules(ctx context.Context) ([]core.Rule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.source, r.keyword, r.start_hour, r.end_hour,
		       r.weekdays_only, g.name, r.channels, r.priority, r.require_ack,
		       r.escalation_channels, r.escalation_delay_seconds, r.schedule,
		       r.repeat_interval_sec, r.repeat_max, r.escalation_policy_id
		FROM rules r
		JOIN groups g ON g.id = r.target_group_id
		WHERE r.active = TRUE
	`)
	if err != nil {
		return nil, fmt.Errorf("load rules: %w", err)
	}
	defer rows.Close()

	var rules []core.Rule
	for rows.Next() {
		var r core.Rule
		var scheduleRaw []byte
		if err := rows.Scan(&r.ID, &r.Source, &r.Keyword, &r.StartHour, &r.EndHour,
			&r.WeekdaysOnly, &r.TargetGroup, &r.Channels, &r.Priority, &r.RequireAck,
			&r.EscalationChannels, &r.EscalationDelaySec, &scheduleRaw,
			&r.RepeatIntervalSec, &r.RepeatMax, &r.EscalationPolicyID); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		if len(scheduleRaw) > 0 {
			_ = json.Unmarshal(scheduleRaw, &r.Schedule)
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rules: %w", err)
	}
	return rules, nil
}

// RuleRecord is the full row shape used by the admin UI — unlike
// core.Rule, it carries TargetGroupID (for editing) alongside the
// resolved TargetGroup name (for display), and includes inactive rules.
type RuleRecord struct {
	ID                   int                   `json:"id"`
	Source               string                `json:"source"`
	Keyword              string                `json:"keyword"`
	StartHour            *int                  `json:"start_hour"`
	EndHour              *int                  `json:"end_hour"`
	WeekdaysOnly         bool                  `json:"weekdays_only"`
	TargetGroupID        int                   `json:"target_group_id"`
	TargetGroup          string                `json:"target_group"`
	Channels             []string              `json:"channels"`
	Priority             int                   `json:"priority"`
	Active               bool                  `json:"active"`
	RequireAck           bool                  `json:"require_ack"`
	EscalationChannels   []string              `json:"escalation_channels"`
	EscalationDelaySec   int                   `json:"escalation_delay_seconds"`
	Schedule             []core.ScheduleEntry  `json:"schedule"`
	RepeatIntervalSec    int                   `json:"repeat_interval_sec"`
	RepeatMax            int                   `json:"repeat_max"`
	EscalationPolicyID   *int                  `json:"escalation_policy_id"`
}

func (s *RuleStore) List(ctx context.Context) ([]RuleRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.source, r.keyword, r.start_hour, r.end_hour, r.weekdays_only,
		       r.target_group_id, g.name, r.channels, r.priority, r.active, r.require_ack,
		       r.escalation_channels, r.escalation_delay_seconds, r.schedule,
		       r.repeat_interval_sec, r.repeat_max, r.escalation_policy_id
		FROM rules r
		JOIN groups g ON g.id = r.target_group_id
		ORDER BY r.priority, r.id
	`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var out []RuleRecord
	for rows.Next() {
		var r RuleRecord
		var scheduleRaw []byte
		if err := rows.Scan(&r.ID, &r.Source, &r.Keyword, &r.StartHour, &r.EndHour, &r.WeekdaysOnly,
			&r.TargetGroupID, &r.TargetGroup, &r.Channels, &r.Priority, &r.Active, &r.RequireAck,
			&r.EscalationChannels, &r.EscalationDelaySec, &scheduleRaw,
			&r.RepeatIntervalSec, &r.RepeatMax, &r.EscalationPolicyID); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		if len(scheduleRaw) > 0 {
			_ = json.Unmarshal(scheduleRaw, &r.Schedule)
		}
		if r.Schedule == nil {
			r.Schedule = []core.ScheduleEntry{}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RuleStore) Create(ctx context.Context, r RuleRecord) (RuleRecord, error) {
	scheduleJSON, _ := json.Marshal(r.Schedule)
	err := s.pool.QueryRow(ctx, `
		INSERT INTO rules (source, keyword, start_hour, end_hour, weekdays_only,
		                    target_group_id, channels, priority, active, require_ack,
		                    escalation_channels, escalation_delay_seconds, schedule,
		                    repeat_interval_sec, repeat_max, escalation_policy_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id
	`, r.Source, r.Keyword, r.StartHour, r.EndHour, r.WeekdaysOnly,
		r.TargetGroupID, r.Channels, r.Priority, r.Active, r.RequireAck,
		r.EscalationChannels, r.EscalationDelaySec, scheduleJSON,
		r.RepeatIntervalSec, r.RepeatMax, r.EscalationPolicyID).Scan(&r.ID)
	if err != nil {
		return RuleRecord{}, fmt.Errorf("create rule: %w", err)
	}
	return r, nil
}

func (s *RuleStore) Update(ctx context.Context, r RuleRecord) error {
	scheduleJSON, _ := json.Marshal(r.Schedule)
	_, err := s.pool.Exec(ctx, `
		UPDATE rules SET source=$1, keyword=$2, start_hour=$3, end_hour=$4, weekdays_only=$5,
		                 target_group_id=$6, channels=$7, priority=$8, active=$9, require_ack=$10,
		                 escalation_channels=$11, escalation_delay_seconds=$12, schedule=$13,
		                 repeat_interval_sec=$14, repeat_max=$15, escalation_policy_id=$16
		WHERE id = $17
	`, r.Source, r.Keyword, r.StartHour, r.EndHour, r.WeekdaysOnly,
		r.TargetGroupID, r.Channels, r.Priority, r.Active, r.RequireAck,
		r.EscalationChannels, r.EscalationDelaySec, scheduleJSON,
		r.RepeatIntervalSec, r.RepeatMax, r.EscalationPolicyID, r.ID)
	if err != nil {
		return fmt.Errorf("update rule: %w", err)
	}
	return nil
}

func (s *RuleStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	return nil
}
