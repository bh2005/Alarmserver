package core

import "context"

// Status values for the event_log.status column.
const (
	StatusDispatched   = "DISPATCHED"
	StatusError        = "ERROR"
	StatusNoMatch      = "NO_MATCH"
	StatusMaintenance  = "MAINTENANCE"
)

// AuditWriter persists the outcome of one processed event for later review
// and returns the event_log row's ID, e.g. so it can be referenced by a
// later Acknowledgement.
type AuditWriter interface {
	WriteAuditLog(ctx context.Context, event Event, rule *Rule, status string, errorDetail string) (int, error)
}
