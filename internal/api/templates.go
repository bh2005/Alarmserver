package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

type TemplateStoreIface interface {
	List(ctx context.Context) ([]db.MessageTemplate, error)
	Upsert(ctx context.Context, channel, template string) error
	Delete(ctx context.Context, channel string) error
}

type TemplatesServer struct {
	Store TemplateStoreIface
}

func NewTemplatesServer(store TemplateStoreIface) *TemplatesServer {
	return &TemplatesServer{Store: store}
}

func (s *TemplatesServer) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.Store.List(r.Context())
	if err != nil {
		http.Error(w, "could not list templates", http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []db.MessageTemplate{}
	}
	writeJSON(w, map[string]any{"templates": list})
}

func (s *TemplatesServer) HandleItem(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimPrefix(r.URL.Path, "/api/templates/")
	channel = strings.TrimSuffix(channel, "/")
	if channel == "" {
		http.Error(w, "channel name required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var payload struct {
			Template string `json:"template"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Template == "" {
			http.Error(w, "template field required", http.StatusBadRequest)
			return
		}
		if err := s.Store.Upsert(r.Context(), channel, payload.Template); err != nil {
			http.Error(w, "could not save template", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := s.Store.Delete(r.Context(), channel); err != nil {
			http.Error(w, "could not delete template", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
