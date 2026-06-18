package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/bh2005/alarmserver/internal/core"
)

// InboundDTMFServer handles POST /api/inbound/dtmf.
// Called by the Asterisk [alarm-ack] dialplan context via curl when the
// called party presses digit "1" during a voice alarm call.
//
// Expected form fields: event_id, recipient_id
type InboundDTMFServer struct {
	Ack    core.AcknowledgementWriter
	Logger *slog.Logger
}

func NewInboundDTMFServer(ack core.AcknowledgementWriter, logger *slog.Logger) *InboundDTMFServer {
	return &InboundDTMFServer{Ack: ack, Logger: logger}
}

func (s *InboundDTMFServer) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	eventID, err1 := strconv.Atoi(r.FormValue("event_id"))
	recipientID, err2 := strconv.Atoi(r.FormValue("recipient_id"))
	if err1 != nil || err2 != nil || eventID == 0 || recipientID == 0 {
		http.Error(w, "event_id and recipient_id are required", http.StatusBadRequest)
		return
	}

	ack, err := core.NewAcknowledgement(eventID, recipientID, "ack", "voice")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Ack.RecordAcknowledgement(r.Context(), ack); err != nil {
		s.Logger.Error("inbound_dtmf_ack_failed", "err", err)
		http.Error(w, "could not record acknowledgement", http.StatusInternalServerError)
		return
	}
	s.Logger.Info("inbound_dtmf_ack", "event_id", eventID, "recipient_id", recipientID)
	w.WriteHeader(http.StatusNoContent)
}
