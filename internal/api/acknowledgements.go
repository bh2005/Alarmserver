package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bh2005/alarmserver/internal/core"
)

// AcknowledgementServer exposes recording and reading of recipient
// acknowledgements ("coming"/"not coming"/"later") for dispatched events.
// Channel-specific inbound integrations (SMS reply parser, DTMF capture,
// app) call HandleRecord; nothing here is channel-specific.
type AcknowledgementServer struct {
	Writer core.AcknowledgementWriter
	Reader core.AcknowledgementReader
}

func NewAcknowledgementServer(writer core.AcknowledgementWriter, reader core.AcknowledgementReader) *AcknowledgementServer {
	return &AcknowledgementServer{Writer: writer, Reader: reader}
}

// HandleRecord accepts {"event_id": N, "recipient_id": N, "response": "...",
// "channel": "..."} and stores it. A repeated reply from the same
// recipient for the same event replaces the previous one.
func (s *AcknowledgementServer) HandleRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		EventID     int    `json:"event_id"`
		RecipientID int    `json:"recipient_id"`
		Response    string `json:"response"`
		Channel     string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ack, err := core.NewAcknowledgement(payload.EventID, payload.RecipientID, payload.Response, payload.Channel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Writer.RecordAcknowledgement(r.Context(), ack); err != nil {
		http.Error(w, "could not record acknowledgement", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// HandleList returns all acknowledgements recorded for ?event_id=N.
func (s *AcknowledgementServer) HandleList(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.URL.Query().Get("event_id"))
	if err != nil {
		http.Error(w, "event_id query parameter is required", http.StatusBadRequest)
		return
	}
	acks, err := s.Reader.LoadAcknowledgements(r.Context(), eventID)
	if err != nil {
		http.Error(w, "could not load acknowledgements", http.StatusInternalServerError)
		return
	}
	if acks == nil {
		acks = []core.Acknowledgement{}
	}
	writeJSON(w, map[string][]core.Acknowledgement{"acknowledgements": acks})
}
