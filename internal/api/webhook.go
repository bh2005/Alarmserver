package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

// Server bundles the HTTP handlers and the channel events are pushed to
// for the Processor to consume.
type Server struct {
	Events chan core.Event
}

func NewServer(events chan core.Event) *Server {
	return &Server{Events: events}
}

// HandleWebhook accepts a generic JSON webhook for the "WEBHOOK" source,
// normalizes it into an Event and queues it for processing.
func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	var payload struct {
		Keyword string `json:"keyword"`
		Text    string `json:"text"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if payload.Keyword == "" {
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return
	}

	event := core.Event{
		Source:     "WEBHOOK",
		Keyword:    payload.Keyword,
		Text:       payload.Text,
		ReceivedAt: time.Now(),
		RawPayload: json.RawMessage(body),
	}

	select {
	case s.Events <- event:
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "event queue full", http.StatusServiceUnavailable)
	}
}
