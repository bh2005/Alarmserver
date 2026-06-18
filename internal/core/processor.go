package core

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// MaintenanceChecker reports whether an event source+keyword is under maintenance
// at the given time. Returning (true, nil) suppresses dispatch.
type MaintenanceChecker interface {
	IsUnderMaintenance(ctx context.Context, source, keyword string, at time.Time) (bool, error)
}

// TemplateLoader returns a Go text template string for the given channel name.
// An empty string means "use the raw event text".
// Placeholders: {text}, {source}, {keyword}, {time}, {group}
type TemplateLoader interface {
	GetTemplate(ctx context.Context, channel string) (string, error)
}

// AckStatusChecker reports whether an event_log row has at least one
// acknowledgement. Used by the repeat-notification goroutine to stop early.
type AckStatusChecker interface {
	IsAcknowledged(ctx context.Context, eventLogID int) (bool, error)
}

// Processor consumes Events, matches them against the RuleEngine, dispatches
// to the Notifiers for the matched Rule's channels and writes the outcome
// to the audit log.
type Processor struct {
	Rules              *RuleEngine
	Audit              AuditWriter
	Recipients         RecipientLoader
	Notifiers          map[string]Notifier    // keyed by Notifier.Channel()
	Maintenance        MaintenanceChecker     // optional — nil = no maintenance check
	Templates          TemplateLoader         // optional — nil = use raw event text
	AckChecker         AckStatusChecker       // optional — required for repeat notifications
	EscalationPolicies EscalationPolicyLoader // optional — nil = no policy escalation
	Logger             *slog.Logger
}

// Run processes incoming events until the channel is closed or ctx is done.
func (p *Processor) Run(ctx context.Context, events <-chan Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			p.process(ctx, event)
		}
	}
}

func (p *Processor) process(ctx context.Context, event Event) {
	p.Logger.Info("event_received",
		"source", event.Source,
		"keyword", event.Keyword,
		"received_at", event.ReceivedAt,
	)

	// Maintenance window check: suppress dispatch and write a MAINTENANCE audit entry.
	if p.Maintenance != nil {
		inMaint, err := p.Maintenance.IsUnderMaintenance(ctx, event.Source, event.Keyword, event.ReceivedAt)
		if err != nil {
			p.Logger.Warn("maintenance_check_failed", "error", err.Error())
		} else if inMaint {
			p.Logger.Info("event_suppressed_maintenance", "source", event.Source, "keyword", event.Keyword)
			if _, auditErr := p.Audit.WriteAuditLog(ctx, event, nil, StatusMaintenance, "suppressed by active maintenance window"); auditErr != nil {
				p.Logger.Error("audit_log_failed", "error", auditErr.Error())
			}
			return
		}
	}

	rule, err := p.Rules.FindMatchingRule(event)
	if err != nil {
		p.Logger.Warn("no_rule_matched", "source", event.Source, "keyword", event.Keyword, "error", err.Error())
		if _, auditErr := p.Audit.WriteAuditLog(ctx, event, nil, StatusNoMatch, err.Error()); auditErr != nil {
			p.Logger.Error("audit_log_failed", "error", auditErr.Error())
		}
		return
	}

	p.Logger.Info("rule_matched",
		"rule_id", rule.ID,
		"target_group", rule.TargetGroup,
		"channels", rule.Channels,
		"require_ack", rule.RequireAck,
	)

	p.dispatch(ctx, event, rule)

	eventLogID, auditErr := p.Audit.WriteAuditLog(ctx, event, rule, StatusDispatched, "")
	if auditErr != nil {
		p.Logger.Error("audit_log_failed", "error", auditErr.Error())
		return
	}
	if rule.RequireAck {
		p.Logger.Info("acknowledgement_expected", "event_log_id", eventLogID, "target_group", rule.TargetGroup)
	}

	// Start background repeat notifications if configured and we can check ACK status.
	if rule.RequireAck && rule.RepeatIntervalSec > 0 && p.AckChecker != nil {
		go p.repeatNotify(ctx, eventLogID, event, *rule)
	}

	// Start escalation policy goroutine if the rule references one.
	if rule.EscalationPolicyID != nil && p.EscalationPolicies != nil && p.AckChecker != nil {
		policy, err := p.EscalationPolicies.GetPolicy(ctx, *rule.EscalationPolicyID)
		if err != nil {
			p.Logger.Warn("escalation_policy_load_failed", "policy_id", *rule.EscalationPolicyID, "error", err.Error())
		} else if len(policy.Steps) > 0 {
			go p.runEscalationPolicy(ctx, eventLogID, event, *rule, policy)
		}
	}
}

// dispatch sends the event's message to every channel configured on the
// matched rule. Failures are logged but never block other channels or the
// audit write — a broken SMS gateway must not prevent the voice call.
func (p *Processor) dispatch(ctx context.Context, event Event, rule *Rule) {
	group := resolveTargetGroup(rule, event.ReceivedAt)
	recipients, err := p.Recipients.LoadGroupMembers(ctx, group)
	if err != nil {
		p.Logger.Error("load_recipients_failed", "target_group", group, "error", err.Error())
		return
	}

	for _, channel := range rule.Channels {
		notifier, ok := p.Notifiers[channel]
		if !ok {
			p.Logger.Warn("no_notifier_for_channel", "channel", channel)
			continue
		}

		msg := p.resolveMessage(ctx, channel, event, group)

		if vn, isVoice := notifier.(VoiceNotifier); isVoice && len(rule.EscalationChannels) > 0 {
			results, err := vn.NotifyTracked(ctx, recipients, msg)
			if err != nil {
				p.Logger.Error("notify_tracked_failed", "channel", channel, "error", err.Error())
				continue
			}
			go p.escalate(ctx, results, recipients, rule, event)
		} else {
			if err := notifier.Notify(ctx, recipients, msg); err != nil {
				p.Logger.Error("notify_failed", "channel", channel, "error", err.Error())
			}
		}
	}
}

// resolveMessage applies a channel template if one is configured, otherwise
// returns the raw event text.
func (p *Processor) resolveMessage(ctx context.Context, channel string, event Event, group string) string {
	if p.Templates == nil {
		return event.Text
	}
	tmpl, err := p.Templates.GetTemplate(ctx, channel)
	if err != nil || tmpl == "" {
		return event.Text
	}
	return strings.NewReplacer(
		"{text}", event.Text,
		"{source}", event.Source,
		"{keyword}", event.Keyword,
		"{time}", event.ReceivedAt.Format("02.01.2006 15:04:05"),
		"{group}", group,
	).Replace(tmpl)
}

// repeatNotify re-dispatches an event at RepeatIntervalSec intervals until
// it is acknowledged or RepeatMax attempts are exhausted.
func (p *Processor) repeatNotify(ctx context.Context, eventLogID int, event Event, rule Rule) {
	limit := rule.RepeatMax
	if limit <= 0 {
		limit = 10 // hard cap to prevent runaway goroutines
	}
	for i := 0; i < limit; i++ {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(rule.RepeatIntervalSec) * time.Second):
		}
		acked, err := p.AckChecker.IsAcknowledged(ctx, eventLogID)
		if err != nil {
			p.Logger.Warn("ack_check_failed", "event_log_id", eventLogID, "error", err.Error())
		}
		if acked {
			p.Logger.Info("repeat_stopped_acked", "event_log_id", eventLogID, "attempt", i+1)
			return
		}
		p.Logger.Info("repeat_notify", "event_log_id", eventLogID, "attempt", i+1, "rule_id", rule.ID)
		p.dispatch(ctx, event, &rule)
	}
}

// resolveTargetGroup picks the group for the event time. If the rule has no
// Schedule entries the static TargetGroup is returned. From > To wraps midnight.
func resolveTargetGroup(rule *Rule, at time.Time) string {
	if len(rule.Schedule) == 0 {
		return rule.TargetGroup
	}
	day := (int(at.Weekday()) + 6) % 7 // Mon=0 … Sun=6
	now := at.Hour()*60 + at.Minute()
	for _, s := range rule.Schedule {
		if !containsDay(s.Days, day) {
			continue
		}
		from := parseHHMM(s.From)
		to := parseHHMM(s.To)
		var inRange bool
		if from <= to {
			inRange = now >= from && now < to
		} else {
			inRange = now >= from || now < to // wraps midnight
		}
		if inRange {
			return s.TargetGroup
		}
	}
	return rule.TargetGroup
}

func containsDay(days []int, day int) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

func parseHHMM(s string) int {
	if len(s) < 5 {
		return 0
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	return h*60 + m
}

// runEscalationPolicy runs after an event is dispatched. It waits until each
// step's absolute delay (from event dispatch) and, if the event is still
// unacknowledged, notifies the step's group. Stops early on ACK or ctx done.
func (p *Processor) runEscalationPolicy(ctx context.Context, eventLogID int, event Event, rule Rule, policy *EscalationPolicy) {
	start := time.Now()
	for _, step := range policy.Steps {
		wait := time.Duration(step.DelaySec)*time.Second - time.Since(start)
		if wait > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}

		acked, err := p.AckChecker.IsAcknowledged(ctx, eventLogID)
		if err != nil {
			p.Logger.Warn("escalation_ack_check_failed", "event_log_id", eventLogID, "error", err.Error())
		}
		if acked {
			p.Logger.Info("escalation_stopped_acked", "event_log_id", eventLogID, "step", step.Step)
			return
		}

		channels := step.Channels
		if len(channels) == 0 {
			channels = rule.Channels
		}
		recipients, err := p.Recipients.LoadGroupMembers(ctx, step.GroupName)
		if err != nil {
			p.Logger.Error("escalation_load_recipients_failed", "group", step.GroupName, "error", err.Error())
			continue
		}
		p.Logger.Info("escalation_step_dispatched", "event_log_id", eventLogID, "step", step.Step, "group", step.GroupName)
		for _, ch := range channels {
			notifier, ok := p.Notifiers[ch]
			if !ok {
				p.Logger.Warn("escalation_no_notifier", "channel", ch)
				continue
			}
			msg := p.resolveMessage(ctx, ch, event, step.GroupName)
			if err := notifier.Notify(ctx, recipients, msg); err != nil {
				p.Logger.Error("escalation_notify_failed", "channel", ch, "error", err.Error())
			}
		}
	}
}

// escalate collects call results from a VoiceNotifier, then dispatches the
// escalation channels to every recipient who did not answer.
func (p *Processor) escalate(ctx context.Context, results <-chan CallResult, recipients []Recipient, rule *Rule, event Event) {
	byID := make(map[int]Recipient, len(recipients))
	for _, r := range recipients {
		byID[r.ID] = r
	}

	for result := range results {
		if result.Answered {
			p.Logger.Info("call_answered", "recipient_id", result.RecipientID)
			delete(byID, result.RecipientID)
		} else {
			p.Logger.Info("call_not_answered", "recipient_id", result.RecipientID, "reason", result.Reason)
		}
	}

	if len(byID) == 0 {
		p.Logger.Info("escalation_not_needed", "rule_id", rule.ID)
		return
	}

	if rule.EscalationDelaySec > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(rule.EscalationDelaySec) * time.Second):
		}
	}

	notAnswered := make([]Recipient, 0, len(byID))
	for _, r := range byID {
		notAnswered = append(notAnswered, r)
	}
	p.Logger.Info("escalating", "rule_id", rule.ID, "count", len(notAnswered), "channels", rule.EscalationChannels)

	for _, ch := range rule.EscalationChannels {
		notifier, ok := p.Notifiers[ch]
		if !ok {
			p.Logger.Warn("no_notifier_for_escalation_channel", "channel", ch)
			continue
		}
		if err := notifier.Notify(ctx, notAnswered, event.Text); err != nil {
			p.Logger.Error("escalation_notify_failed", "channel", ch, "error", err.Error())
		}
	}
}
