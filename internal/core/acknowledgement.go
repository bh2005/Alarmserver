package core

import (
	"context"
	"fmt"
	"time"
)

// Response values for an Acknowledgement.
const (
	ResponseComing    = "COMING"
	ResponseNotComing = "NOT_COMING"
	ResponseLater     = "LATER"
)

var validResponses = map[string]bool{
	ResponseComing:    true,
	ResponseNotComing: true,
	ResponseLater:     true,
}

// ValidResponse reports whether s is a recognized acknowledgement response.
func ValidResponse(s string) bool { return validResponses[s] }

// Acknowledgement records one recipient's reply ("coming"/"not coming"/
// "later") to a dispatched event. Only meaningful for rules that have
// RequireAck set — but acknowledgements can be recorded regardless, since
// the inbound channel (SMS reply, app, DTMF) decides what to forward.
type Acknowledgement struct {
	EventLogID     int       `json:"event_log_id"`
	RecipientID    int       `json:"recipient_id"`
	Response       string    `json:"response"`
	Channel        string    `json:"channel"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

// AcknowledgementWriter persists an Acknowledgement.
type AcknowledgementWriter interface {
	RecordAcknowledgement(ctx context.Context, ack Acknowledgement) error
}

// AcknowledgementReader summarizes acknowledgements recorded for one event.
type AcknowledgementReader interface {
	LoadAcknowledgements(ctx context.Context, eventLogID int) ([]Acknowledgement, error)
}

// NewAcknowledgement validates and builds an Acknowledgement.
func NewAcknowledgement(eventLogID, recipientID int, response, channel string) (Acknowledgement, error) {
	if !ValidResponse(response) {
		return Acknowledgement{}, fmt.Errorf("invalid acknowledgement response %q", response)
	}
	return Acknowledgement{
		EventLogID:  eventLogID,
		RecipientID: recipientID,
		Response:    response,
		Channel:     channel,
	}, nil
}
