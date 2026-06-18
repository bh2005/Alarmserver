package core

import "context"

// Recipient is someone who can be notified over one or more channels.
type Recipient struct {
	ID     int
	Name   string
	Voice  string // phone number for voice/SIP channels
	Mobile string // phone number for SMS channels
	Email  string // e-mail address for the email channel
}

// Notifier sends a message to a set of recipients over one specific
// channel (e.g. voice call, SMS, push, e-mail). Concrete implementations
// are pluggable — the core never depends on a specific channel provider.
type Notifier interface {
	Channel() string // channel identifier, e.g. "voice", "sms"
	Notify(ctx context.Context, recipients []Recipient, message string) error
}

// CallResult is the outcome of a single voice call attempt.
type CallResult struct {
	RecipientID int
	Answered    bool
	Reason      string // e.g. "ANSWER", "NOANSWER", "BUSY", "TIMEOUT"
}

// VoiceNotifier extends Notifier for channels that can report per-call
// outcomes. Implemented by the Asterisk notifier to drive escalation:
// the Processor reads results from the returned channel and dispatches
// escalation channels only to recipients who did not answer.
type VoiceNotifier interface {
	Notifier
	// NotifyTracked originates one call per recipient and sends a CallResult
	// for each. The channel is closed when all results are in or ctx expires.
	NotifyTracked(ctx context.Context, recipients []Recipient, message string) (<-chan CallResult, error)
}

// RecipientLoader resolves the members of a target group at dispatch time.
type RecipientLoader interface {
	LoadGroupMembers(ctx context.Context, groupName string) ([]Recipient, error)
}
