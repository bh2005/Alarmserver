package core

import (
	"context"
	"testing"
	"time"
)

type fakeRuleLoader struct {
	rules []Rule
	err   error
}

func (f *fakeRuleLoader) LoadRules(ctx context.Context) ([]Rule, error) {
	return f.rules, f.err
}

func intp(i int) *int { return &i }

func TestFindMatchingRule_NoRules(t *testing.T) {
	e := NewRuleEngine(&fakeRuleLoader{})
	_, err := e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: time.Now()})
	if err == nil {
		t.Fatal("expected error when no rules are loaded")
	}
}

func TestFindMatchingRule_TimeWindowAndWeekday(t *testing.T) {
	rules := []Rule{
		{ID: 1, Source: "WEBHOOK", Keyword: "F2", StartHour: intp(6), EndHour: intp(17), WeekdaysOnly: true, TargetGroup: "Day", Priority: 5},
		{ID: 2, Source: "WEBHOOK", Keyword: "F2", WeekdaysOnly: false, TargetGroup: "Night", Priority: 10},
	}
	e := NewRuleEngine(&fakeRuleLoader{rules: rules})
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}

	// Monday 10:00 -> matches both, but rule 1 has the higher priority (lower value).
	monday10 := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC) // a Monday
	rule, err := e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: monday10})
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if rule.ID != 1 {
		t.Fatalf("expected rule 1 (day) to win during business hours, got rule %d", rule.ID)
	}

	// Monday 20:00 -> outside the day window, only the night/always rule matches.
	monday20 := time.Date(2026, 6, 15, 20, 0, 0, 0, time.UTC)
	rule, err = e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: monday20})
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if rule.ID != 2 {
		t.Fatalf("expected rule 2 (night/always) outside business hours, got rule %d", rule.ID)
	}

	// Saturday 10:00 -> weekdays-only rule excluded, only rule 2 matches.
	saturday10 := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	rule, err = e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: saturday10})
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if rule.ID != 2 {
		t.Fatalf("expected rule 2 on weekend, got rule %d", rule.ID)
	}
}

func TestFindMatchingRule_PriorityOrder(t *testing.T) {
	rules := []Rule{
		{ID: 1, Source: "WEBHOOK", Keyword: "F2", TargetGroup: "Low", Priority: 20},
		{ID: 2, Source: "WEBHOOK", Keyword: "F2", TargetGroup: "High", Priority: 5},
	}
	e := NewRuleEngine(&fakeRuleLoader{rules: rules})
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}

	rule, err := e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: time.Now()})
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if rule.ID != 2 {
		t.Fatalf("expected rule with lowest priority value (2) to win, got rule %d", rule.ID)
	}
}

func TestFindMatchingRule_SourceAndKeywordMustMatch(t *testing.T) {
	rules := []Rule{
		{ID: 1, Source: "WEBHOOK", Keyword: "F2", TargetGroup: "A", Priority: 10},
	}
	e := NewRuleEngine(&fakeRuleLoader{rules: rules})
	e.Reload(context.Background())

	if _, err := e.FindMatchingRule(Event{Source: "EMAIL", Keyword: "F2", ReceivedAt: time.Now()}); err == nil {
		t.Fatal("expected no match for different source")
	}
	if _, err := e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "H1", ReceivedAt: time.Now()}); err == nil {
		t.Fatal("expected no match for different keyword")
	}
}

func TestReload_KeepsOldRulesOnError(t *testing.T) {
	loader := &fakeRuleLoader{rules: []Rule{{ID: 1, Source: "WEBHOOK", Keyword: "F2", TargetGroup: "A", Priority: 10}}}
	e := NewRuleEngine(loader)
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("initial reload: %v", err)
	}

	loader.err = context.DeadlineExceeded
	if err := e.Reload(context.Background()); err == nil {
		t.Fatal("expected reload to surface the loader error")
	}

	// Old rules must still be matchable.
	if _, err := e.FindMatchingRule(Event{Source: "WEBHOOK", Keyword: "F2", ReceivedAt: time.Now()}); err != nil {
		t.Fatalf("expected old rules to remain after failed reload, got error: %v", err)
	}
}
