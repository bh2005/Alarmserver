package core

import (
	"encoding/json"
	"time"
)

// Event is the common type all input channels normalize into before
// handing it to the core (webhook, e-mail poller, serial reader, ...).
type Event struct {
	Source     string          // e.g. "WEBHOOK", "EMAIL", "SERIAL", "MANUAL"
	Keyword    string          // matched against Rule.Keyword
	Text       string          // free text for display/audit
	ReceivedAt time.Time
	RawPayload json.RawMessage // original payload for the audit log
}
