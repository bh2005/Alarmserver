package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bh2005/alarmserver/internal/db"
)

// knownChannels defines the order channels appear in the UI and their
// human-readable labels. New channels must be added here to be visible.
var knownChannels = []channelMeta{
	{ID: "voice", Label: "Voice (Asterisk)", Fields: voiceFields},
	{ID: "voice-smseagle", Label: "Voice (SMSEagle)", Fields: smseagleVoiceFields},
	{ID: "sms", Label: "SMS (SMSEagle)", Fields: smseagleFields},
	{ID: "email", Label: "E-Mail (SMTP)", Fields: emailFields},
	{ID: "teams", Label: "Microsoft Teams", Fields: teamsFields},
	{ID: "ldap", Label: "LDAP / Active Directory", Fields: ldapFields},
	{ID: "imap", Label: "IMAP-Polling (Eingangskanal)", Fields: imapFields},
}

type channelMeta struct {
	ID     string
	Label  string
	Fields []fieldMeta
}

type fieldMeta struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Sensitive bool   `json:"sensitive"`
	Default   string `json:"default,omitempty"`
}

var voiceFields = []fieldMeta{
	{Key: "host", Label: "AMI Host"},
	{Key: "port", Label: "AMI Port", Default: "5038"},
	{Key: "ami_user", Label: "AMI User"},
	{Key: "ami_password", Label: "AMI Password", Sensitive: true},
	{Key: "channel_template", Label: "Channel Template", Default: "PJSIP/{number}"},
	{Key: "announcement", Label: "Announcement Sound", Default: "alarm-announcement"},
	{Key: "sounds_dir", Label: "Sounds Directory", Default: "/var/lib/asterisk/sounds/custom"},
	{Key: "tts_enabled", Label: "TTS aktivieren (true/false)", Default: "false"},
	{Key: "tts_voice", Label: "TTS Voice", Default: "de"},
	{Key: "tts_speed", Label: "TTS Speed (0 = Standard)", Default: "0"},
}

var smseagleVoiceFields = []fieldMeta{
	{Key: "base_url", Label: "SMSEagle Base URL"},
	{Key: "login", Label: "Login"},
	{Key: "password", Label: "Password", Sensitive: true},
	{Key: "api_key", Label: "API Key (alternativ zu Login/Passwort)", Sensitive: true},
}

var smseagleFields = []fieldMeta{
	{Key: "base_url", Label: "SMSEagle Base URL"},
	{Key: "login", Label: "Login"},
	{Key: "password", Label: "Password", Sensitive: true},
	{Key: "api_key", Label: "API Key (alternativ zu Login/Passwort)", Sensitive: true},
}

var ldapFields = []fieldMeta{
	{Key: "host", Label: "LDAP-Server (Hostname/IP)"},
	{Key: "port", Label: "Port", Default: "389"},
	{Key: "tls", Label: "LDAPS (TLS)", Default: "false"},
	{Key: "bind_dn", Label: "Bind-DN (Dienst-Account)"},
	{Key: "bind_password", Label: "Bind-Passwort", Sensitive: true},
	{Key: "user_base", Label: "User-Suchbasis (Base DN)"},
	{Key: "user_filter", Label: "Suchfilter", Default: "(&(objectClass=user)(sAMAccountName={username}))"},
	{Key: "admin_group", Label: "Admin-Gruppe (DN — Mitglieder erhalten Rolle admin)"},
	{Key: "default_role", Label: "Standard-Rolle für alle anderen LDAP-User", Default: "viewer"},
}

var emailFields = []fieldMeta{
	{Key: "host", Label: "SMTP Host"},
	{Key: "port", Label: "SMTP Port", Default: "587"},
	{Key: "username", Label: "Username"},
	{Key: "password", Label: "Password", Sensitive: true},
	{Key: "from", Label: "Absender (From)"},
}

var teamsFields = []fieldMeta{
	{Key: "webhook_url", Label: "Incoming Webhook URL", Sensitive: true},
}

var imapFields = []fieldMeta{
	{Key: "host", Label: "IMAP-Server (Hostname/IP)"},
	{Key: "port", Label: "Port", Default: "993"},
	{Key: "tls", Label: "IMAPS (TLS)", Default: "true"},
	{Key: "username", Label: "Benutzername / E-Mail"},
	{Key: "password", Label: "Passwort", Sensitive: true},
	{Key: "folder", Label: "Ordner", Default: "INBOX"},
	{Key: "poll_sec", Label: "Abfrageintervall (Sekunden)", Default: "60"},
	{Key: "source", Label: "Event-Source (wird in Regeln abgeglichen)", Default: "imap"},
}

const sensitiveMarker = "***"

// ChannelConfigResponse is what the API returns per channel.
type ChannelConfigResponse struct {
	Channel string            `json:"channel"`
	Label   string            `json:"label"`
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config"`
	Fields  []fieldMeta       `json:"fields"`
}

// ChannelConfigStore is implemented by db.ChannelConfigStore.
type ChannelConfigStore interface {
	LoadAll(ctx context.Context) ([]db.ChannelConfig, error)
	Load(ctx context.Context, channel string) (*db.ChannelConfig, error)
	Save(ctx context.Context, channel string, enabled bool, cfg map[string]string) error
}

// ConfigServer serves channel configuration for the UI settings page.
type ConfigServer struct {
	Store ChannelConfigStore
}

func NewConfigServer(store ChannelConfigStore) *ConfigServer {
	return &ConfigServer{Store: store}
}

// HandleChannels serves GET /api/config/channels — all channels with
// sensitive fields masked.
func (s *ConfigServer) HandleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	saved, err := s.Store.LoadAll(r.Context())
	if err != nil {
		http.Error(w, "could not load channel configs", http.StatusInternalServerError)
		return
	}

	byChannel := make(map[string]db.ChannelConfig, len(saved))
	for _, cc := range saved {
		byChannel[cc.Channel] = cc
	}

	var out []ChannelConfigResponse
	for _, meta := range knownChannels {
		resp := ChannelConfigResponse{
			Channel: meta.ID,
			Label:   meta.Label,
			Fields:  meta.Fields,
			Config:  make(map[string]string),
		}
		if cc, ok := byChannel[meta.ID]; ok {
			resp.Enabled = cc.Enabled
			for _, f := range meta.Fields {
				if f.Sensitive {
					if cc.Config[f.Key] != "" {
						resp.Config[f.Key] = sensitiveMarker
					}
				} else {
					resp.Config[f.Key] = cc.Config[f.Key]
				}
			}
		} else {
			for _, f := range meta.Fields {
				if !f.Sensitive {
					resp.Config[f.Key] = ""
				}
			}
		}
		out = append(out, resp)
	}

	writeJSON(w, map[string][]ChannelConfigResponse{"channels": out})
}

// HandleChannel serves PUT /api/config/channels/{channel}.
// Sensitive fields sent as "***" are kept unchanged in the DB.
func (s *ConfigServer) HandleChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channel := strings.TrimPrefix(r.URL.Path, "/api/config/channels/")
	if channel == "" {
		http.Error(w, "channel required", http.StatusBadRequest)
		return
	}

	var meta *channelMeta
	for i := range knownChannels {
		if knownChannels[i].ID == channel {
			meta = &knownChannels[i]
			break
		}
	}
	if meta == nil {
		http.Error(w, "unknown channel", http.StatusBadRequest)
		return
	}

	var body struct {
		Enabled bool              `json:"enabled"`
		Config  map[string]string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Load existing config to preserve sensitive fields when marker is sent.
	existing, _ := s.Store.Load(r.Context(), channel)
	existingCfg := map[string]string{}
	if existing != nil {
		existingCfg = existing.Config
	}

	merged := make(map[string]string, len(body.Config))
	for _, f := range meta.Fields {
		v := body.Config[f.Key]
		if f.Sensitive && v == sensitiveMarker {
			merged[f.Key] = existingCfg[f.Key] // keep existing
		} else {
			merged[f.Key] = v
		}
	}

	if err := s.Store.Save(r.Context(), channel, body.Enabled, merged); err != nil {
		http.Error(w, "could not save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
