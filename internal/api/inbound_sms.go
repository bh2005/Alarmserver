package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bh2005/alarmserver/internal/core"
	"github.com/bh2005/alarmserver/internal/db"
)

// InboundSMSLookup is satisfied by db.RecipientStore.
type InboundSMSLookup interface {
	FindByMobile(ctx context.Context, mobile string) (*db.RecipientRecord, error)
	LatestUnackedEventForRecipient(ctx context.Context, recipientID int) (int, error)
}

// InboundSMSServer handles POST /api/inbound/smseagle.
// SMSEagle is configured to POST each received SMS to this URL.
// Supported formats:
//   - application/x-www-form-urlencoded: from=+49…&text=ACK
//   - application/json: {"from":"+49…","text":"ACK"}
//
// ACK syntax (case-insensitive): "ACK", "ACK 123" (with event ID),
// "JA", "OK" — all treated as acknowledgement with response "ack".
type InboundSMSServer struct {
	Lookup  InboundSMSLookup
	Ack     core.AcknowledgementWriter
	Logger  *slog.Logger
}

func NewInboundSMSServer(lookup InboundSMSLookup, ack core.AcknowledgementWriter, logger *slog.Logger) *InboundSMSServer {
	return &InboundSMSServer{Lookup: lookup, Ack: ack, Logger: logger}
}

func (s *InboundSMSServer) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	from, text, err := s.parseRequest(r)
	if err != nil || from == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	text = strings.TrimSpace(text)
	if !isACKText(text) {
		// Not an ACK reply — silently ignore (could be anything).
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Optional explicit event ID: "ACK 42"
	eventID := 0
	parts := strings.Fields(text)
	if len(parts) >= 2 {
		eventID, _ = strconv.Atoi(parts[1])
	}

	ctx := r.Context()
	recipient, err := s.Lookup.FindByMobile(ctx, from)
	if err != nil || recipient == nil {
		s.Logger.Warn("inbound_sms_unknown_sender", "from", from)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if eventID == 0 {
		eventID, err = s.Lookup.LatestUnackedEventForRecipient(ctx, recipient.ID)
		if err != nil || eventID == 0 {
			s.Logger.Info("inbound_sms_no_open_event", "recipient_id", recipient.ID)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	ack, err := core.NewAcknowledgement(eventID, recipient.ID, "ack", "sms")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Ack.RecordAcknowledgement(ctx, ack); err != nil {
		s.Logger.Error("inbound_sms_ack_failed", "err", err)
		http.Error(w, "could not record acknowledgement", http.StatusInternalServerError)
		return
	}
	s.Logger.Info("inbound_sms_ack", "recipient", recipient.FirstName+" "+recipient.LastName, "event_id", eventID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *InboundSMSServer) parseRequest(r *http.Request) (from, text string, err error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var body struct {
			From string `json:"from"`
			Text string `json:"text"`
		}
		if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
			return
		}
		return body.From, body.Text, nil
	}
	// Fallback: form-encoded (SMSEagle default)
	if err = r.ParseForm(); err != nil {
		return
	}
	return r.FormValue("from"), r.FormValue("text"), nil
}

func isACKText(s string) bool {
	upper := strings.ToUpper(strings.Fields(s)[0])
	switch upper {
	case "ACK", "JA", "OK", "QUITTIERT", "QUI":
		return true
	}
	return false
}
