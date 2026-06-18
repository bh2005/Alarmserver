package core

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// ScheduleEntry defines which group to alarm within a specific time window.
// Days: 0=Monday … 6=Sunday. From/To in "HH:MM". From > To wraps midnight.
type ScheduleEntry struct {
	Days        []int  `json:"days"`
	From        string `json:"from"`
	To          string `json:"to"`
	TargetGroup string `json:"target_group"`
}

// Rule is a single "if-then" routing entry: which events go to which group
// over which channels, optionally restricted by time window / weekday.
type Rule struct {
	ID           int
	Source       string
	Keyword      string
	StartHour    *int // nil = no limit
	EndHour      *int
	WeekdaysOnly bool
	TargetGroup  string   // default group when no Schedule entry matches
	Schedule     []ScheduleEntry
	Channels            []string
	Priority            int  // lower value = higher priority
	RequireAck          bool // whether recipients are expected to acknowledge
	EscalationChannels  []string // channels to try if primary voice call is not answered
	EscalationDelaySec  int      // seconds to wait before escalating (0 = immediate)
	RepeatIntervalSec   int      // 0 = no repeat; seconds between repeat notifications
	RepeatMax           int      // max repeat attempts (0 = unlimited until ACK)
	EscalationPolicyID  *int     // optional; policy to run if no ACK after step.DelaySec
}

// EscalationStep is one stage in an escalation policy.
// DelaySec is absolute from the moment the event was dispatched.
// Channels is empty → inherit from the triggering rule's Channels.
type EscalationStep struct {
	Step      int
	GroupName string
	DelaySec  int
	Channels  []string
}

// EscalationPolicy holds an ordered list of escalation steps.
type EscalationPolicy struct {
	ID    int
	Name  string
	Steps []EscalationStep
}

// EscalationPolicyLoader resolves a policy by ID (used by Processor).
type EscalationPolicyLoader interface {
	GetPolicy(ctx context.Context, id int) (*EscalationPolicy, error)
}

// RuleLoader loads the current rule set from a data source (e.g. a database).
type RuleLoader interface {
	LoadRules(ctx context.Context) ([]Rule, error)
}

// RuleEngine holds the rule set in memory and matches incoming events
// against it.
type RuleEngine struct {
	mu         sync.RWMutex
	rules      []Rule
	loader     RuleLoader
	lastReload time.Time
}

func NewRuleEngine(loader RuleLoader) *RuleEngine {
	return &RuleEngine{loader: loader}
}

// Reload reloads the rule set. On error the previously known rules are kept
// unchanged so the engine keeps working if the data source is unavailable.
func (e *RuleEngine) Reload(ctx context.Context) error {
	rules, err := e.loader.LoadRules(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = rules
	e.lastReload = time.Now()
	e.mu.Unlock()
	return nil
}

// StartBackgroundReload reloads the rule set periodically until ctx is done.
func (e *RuleEngine) StartBackgroundReload(ctx context.Context, interval time.Duration, logger *slog.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := e.Reload(ctx); err != nil {
					logger.Warn("rule_reload_failed",
						"error", err.Error(),
						"cache_age_s", time.Since(e.lastReload).Seconds(),
					)
				}
			}
		}
	}()
}

// FindMatchingRule matches an event against the loaded rule set. If multiple
// rules match, the one with the lowest Priority value wins.
//
// Known gap (carried over unchanged from the original design): a time
// window that wraps past midnight (e.g. StartHour=17, EndHour=6) never
// matches under this [start,end) check, since hour>=17 AND hour<6 can't
// both be true. Such windows currently need two separate rules (17-24, 0-6).
func (e *RuleEngine) FindMatchingRule(event Event) (*Rule, error) {
	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	now := event.ReceivedAt
	hour := now.Hour()
	isWeekday := now.Weekday() >= time.Monday && now.Weekday() <= time.Friday

	var matches []Rule
	for _, rule := range rules {
		if rule.Source != event.Source || rule.Keyword != event.Keyword {
			continue
		}
		if rule.WeekdaysOnly && !isWeekday {
			continue
		}
		if rule.StartHour != nil && hour < *rule.StartHour {
			continue
		}
		if rule.EndHour != nil && hour >= *rule.EndHour {
			continue
		}
		matches = append(matches, rule)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no rule for source=%s keyword=%s", event.Source, event.Keyword)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Priority < matches[j].Priority
	})
	return &matches[0], nil
}
