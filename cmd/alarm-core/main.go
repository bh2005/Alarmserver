package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bh2005/alarmserver/internal/api"
	"github.com/bh2005/alarmserver/internal/auth"
	ldapauth "github.com/bh2005/alarmserver/internal/auth/ldap"
	"github.com/bh2005/alarmserver/internal/core"
	"github.com/bh2005/alarmserver/internal/db"
	"github.com/bh2005/alarmserver/internal/notify/asterisk"
	"github.com/bh2005/alarmserver/internal/notify/email"
	"github.com/bh2005/alarmserver/internal/notify/smseagle"
	"github.com/bh2005/alarmserver/internal/notify/teams"
	"github.com/bh2005/alarmserver/internal/notify/webhook"
	"github.com/bh2005/alarmserver/internal/tts/espeak"
	imapin "github.com/bh2005/alarmserver/internal/inbound/imap"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dsn := getenv("ALARM_DB_DSN", "postgres://alarmcore:alarmcore@localhost:5432/alarmcore")
	listenAddr := getenv("ALARM_LISTEN_ADDR", ":8080")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		logger.Error("db_connect_failed", "error", err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	ruleStore := db.NewRuleStore(pool)
	auditStore := db.NewAuditStore(pool)
	ackStore := db.NewAcknowledgementStore(pool)
	groupStore := db.NewGroupStore(pool)
	userStore := db.NewUserStore(pool)
	channelConfigStore := db.NewChannelConfigStore(pool)
	apiKeyStore := db.NewAPIKeyStore(pool)
	maintenanceStore := db.NewMaintenanceStore(pool)
	heartbeatStore := db.NewHeartbeatStore(pool)
	templateStore := db.NewTemplateStore(pool)
	statsStore := db.NewStatsStore(pool)
	escalationStore := db.NewEscalationStore(pool)
	webhookTargetStore := db.NewWebhookTargetStore(pool)

	if err := bootstrapAdmin(ctx, userStore, logger); err != nil {
		logger.Error("bootstrap_admin_failed", "error", err.Error())
		os.Exit(1)
	}

	sessionSecret := os.Getenv("ALARM_SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = randomHex(32)
		logger.Warn("session_secret_not_set",
			"msg", "ALARM_SESSION_SECRET is not set — using a random secret for this process only. All UI sessions will be invalidated on restart. Set it for persistent sessions.")
	}
	authService := auth.NewService(userStore, sessionSecret, 12*time.Hour)
	if ldapCfg := loadLDAPConfig(ctx, channelConfigStore, logger); ldapCfg != nil {
		authService.LDAPAuth = ldapCfg
	}

	ruleEngine := core.NewRuleEngine(ruleStore)
	if err := ruleEngine.Reload(ctx); err != nil {
		logger.Error("initial_rule_load_failed", "error", err.Error())
		os.Exit(1)
	}
	ruleEngine.StartBackgroundReload(ctx, 60*time.Second, logger)

	recipientStore := db.NewRecipientStore(pool)
	notifiers := buildNotifiers(logger)

	// Register active webhook targets as notifiers.
	if wts, err := webhookTargetStore.ListActive(ctx); err == nil {
		for _, wt := range wts {
			n := webhook.New(webhook.Config{
				Name:         wt.Name,
				URL:          wt.URL,
				Method:       wt.Method,
				Headers:      wt.Headers,
				BodyTemplate: wt.BodyTemplate,
				TimeoutSec:   wt.TimeoutSec,
			})
			notifiers[n.Channel()] = n
			logger.Info("notifier_registered", "channel", n.Channel(), "url", wt.URL)
		}
	}

	events := make(chan core.Event, 100)
	processor := &core.Processor{
		Rules:              ruleEngine,
		Audit:              auditStore,
		Recipients:         recipientStore,
		Notifiers:          notifiers,
		Maintenance:        maintenanceStore,
		Templates:          templateStore,
		AckChecker:         ackStore,
		EscalationPolicies: escalationStore,
		Logger:             logger,
	}
	go processor.Run(ctx, events)

	// Heartbeat expiry checker: fires every 60s and injects an alarm event for
	// each overdue heartbeat that hasn't been alerted in the last hour.
	go runHeartbeatChecker(ctx, heartbeatStore, events, logger)

	// IMAP inbound poller — started if channel "imap" is configured and enabled.
	if imapCfg, err := channelConfigStore.Load(ctx, "imap"); err == nil && imapCfg != nil && imapCfg.Enabled {
		if cfg, err := imapin.ConfigFromMap(imapCfg.Config); err == nil {
			poller := imapin.New(cfg, events, logger)
			go poller.Run(ctx)
			logger.Info("imap_poller_started", "host", cfg.Host, "folder", cfg.Folder)
		} else {
			logger.Warn("imap_config_invalid", "err", err)
		}
	}

	server := api.NewServer(events)
	announcements := api.NewAnnouncementServer(getenv("ASTERISK_SOUNDS_DIR", "/var/lib/asterisk/sounds/custom"))
	acknowledgements := api.NewAcknowledgementServer(ackStore, ackStore)
	sessionServer := api.NewSessionServer(authService)
	groupsServer := api.NewGroupsServer(groupStore)
	recipientsServer := api.NewRecipientsServer(recipientStore)
	rulesServer := api.NewRulesServer(ruleStore)
	eventLogServer := api.NewEventLogServer(auditStore)
	usersServer := api.NewUsersServer(userStore)
	configServer := api.NewConfigServer(channelConfigStore)
	apiKeysServer := api.NewAPIKeysServer(apiKeyStore)
	selfServiceStore := db.NewSelfServiceStore(pool)
	meServer := api.NewMeServer(userStore, selfServiceStore)
	maintenanceServer := api.NewMaintenanceServer(maintenanceStore)
	heartbeatsServer := api.NewHeartbeatsServer(heartbeatStore)
	templatesServer := api.NewTemplatesServer(templateStore)
	statsServer := api.NewStatsServer(statsStore)
	dashboardServer := api.NewDashboardServer(auditStore, maintenanceStore, heartbeatStore)
	escalationServer     := api.NewEscalationServer(escalationStore)
	webhookTargetsServer := api.NewWebhookTargetsServer(webhookTargetStore)
	inboundSMSServer    := api.NewInboundSMSServer(recipientStore, ackStore, logger)
	inboundDTMFServer   := api.NewInboundDTMFServer(ackStore, logger)

	apiKey := os.Getenv("ALARM_API_KEY")
	if apiKey == "" {
		logger.Warn("api_key_not_set",
			"msg", "ALARM_API_KEY is not set — /api/* endpoints are unauthenticated. Do not expose this host to the internet.")
	}
	apiAuth := api.CombinedAuthenticator(apiKey, apiKeyStore)
	protect := func(h http.HandlerFunc) http.HandlerFunc {
		return api.RequireAPIKey(apiAuth, h)
	}
	protectShared := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if sessionServer.HasValidSession(r) {
				h(w, r)
				return
			}
			protect(h)(w, r)
		}
	}

	mux := http.NewServeMux()

	// Machine-to-machine only: webhook ingestion.
	mux.HandleFunc("/api/events", protect(server.HandleWebhook))

	// Heartbeat ping — machine-to-machine, requires API key.
	mux.HandleFunc("/api/heartbeat/", protect(heartbeatsServer.HandlePing))

	// Reachable from both the UI and machine clients.
	mux.HandleFunc("/api/announcements", protectShared(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			announcements.HandleUpload(w, r)
			return
		}
		announcements.HandleList(w, r)
	}))
	mux.HandleFunc("/api/acknowledgements", protectShared(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			acknowledgements.HandleRecord(w, r)
			return
		}
		acknowledgements.HandleList(w, r)
	}))

	// UI login and self-service.
	mux.HandleFunc("/auth/login", sessionServer.HandleLogin)
	mux.HandleFunc("/auth/me", sessionServer.RequireSession(sessionServer.HandleMe))
	mux.HandleFunc("/api/events/log", sessionServer.RequireSession(eventLogServer.HandleList))
	mux.HandleFunc("/api/dashboard", sessionServer.RequireSession(dashboardServer.HandleDashboard))
	mux.HandleFunc("/api/stats", sessionServer.RequireAdmin(statsServer.HandleStats))

	// Admin-only CRUD.
	mux.HandleFunc("/api/groups", sessionServer.RequireAdmin(groupsServer.HandleCollection))
	mux.HandleFunc("/api/groups/", sessionServer.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/members"):
			groupsServer.HandleMembers(w, r)
		case strings.Contains(r.URL.Path, "/members/"):
			groupsServer.HandleMemberItem(w, r)
		default:
			groupsServer.HandleItem(w, r)
		}
	}))
	mux.HandleFunc("/api/recipients", sessionServer.RequireAdmin(recipientsServer.HandleCollection))
	mux.HandleFunc("/api/recipients/", sessionServer.RequireAdmin(recipientsServer.HandleItem))
	mux.HandleFunc("/api/rules", sessionServer.RequireAdmin(rulesServer.HandleCollection))
	mux.HandleFunc("/api/rules/", sessionServer.RequireAdmin(rulesServer.HandleItem))
	mux.HandleFunc("/api/users", sessionServer.RequireAdmin(usersServer.HandleList))
	mux.HandleFunc("/api/users/create", sessionServer.RequireAdmin(usersServer.HandleCreate))
	mux.HandleFunc("/api/users/", sessionServer.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/active"):
			usersServer.HandleSetActive(w, r)
		case strings.HasSuffix(r.URL.Path, "/recipient"):
			usersServer.HandleSetRecipient(w, r)
		default:
			usersServer.HandleDelete(w, r)
		}
	}))

	mux.HandleFunc("/api/me/groups", sessionServer.RequireSession(meServer.HandleGroups))
	mux.HandleFunc("/api/me/groups/", sessionServer.RequireSession(meServer.HandleGroupActive))

	mux.HandleFunc("/api/config/channels", sessionServer.RequireAdmin(configServer.HandleChannels))
	mux.HandleFunc("/api/config/channels/", sessionServer.RequireAdmin(configServer.HandleChannel))
	mux.HandleFunc("/api/apikeys", sessionServer.RequireAdmin(apiKeysServer.HandleCollection))
	mux.HandleFunc("/api/apikeys/", sessionServer.RequireAdmin(apiKeysServer.HandleItem))

	mux.HandleFunc("/api/maintenance", sessionServer.RequireAdmin(maintenanceServer.HandleCollection))
	mux.HandleFunc("/api/maintenance/", sessionServer.RequireAdmin(maintenanceServer.HandleItem))

	mux.HandleFunc("/api/heartbeats", sessionServer.RequireAdmin(heartbeatsServer.HandleCollection))
	mux.HandleFunc("/api/heartbeats/", sessionServer.RequireAdmin(heartbeatsServer.HandleItem))

	mux.HandleFunc("/api/templates", sessionServer.RequireAdmin(templatesServer.HandleCollection))
	mux.HandleFunc("/api/templates/", sessionServer.RequireAdmin(templatesServer.HandleItem))

	mux.HandleFunc("/api/escalation-policies", sessionServer.RequireAdmin(escalationServer.HandleCollection))
	mux.HandleFunc("/api/escalation-policies/", sessionServer.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/steps"):
			escalationServer.HandleSteps(w, r)
		case strings.Contains(path, "/steps/"):
			escalationServer.HandleStepItem(w, r)
		default:
			escalationServer.HandleItem(w, r)
		}
	}))

	mux.HandleFunc("/api/webhook-targets", sessionServer.RequireAdmin(webhookTargetsServer.HandleCollection))
	mux.HandleFunc("/api/webhook-targets/", sessionServer.RequireAdmin(webhookTargetsServer.HandleItem))

	// Inbound channels — unauthenticated (called by SMSEagle / Asterisk dialplan).
	mux.HandleFunc("/api/inbound/smseagle", inboundSMSServer.Handle)
	mux.HandleFunc("/api/inbound/dtmf", inboundDTMFServer.Handle)

	mux.HandleFunc("/api/openapi.yaml", api.HandleOpenAPISpec)
	mux.HandleFunc("/api/docs", api.HandleSwaggerUI)

	mux.HandleFunc("/health", api.HealthHandler(map[string]api.HealthChecker{
		"db": func(ctx context.Context) string { return pool.Ping(ctx) },
		"rules": func(ctx context.Context) string {
			rules, err := ruleStore.List(ctx)
			if err != nil {
				return "error: " + err.Error()
			}
			active := 0
			for _, r := range rules {
				if r.Active {
					active++
				}
			}
			return fmt.Sprintf("ok: %d active", active)
		},
		"event_queue": func(_ context.Context) string {
			used := len(events)
			cap := cap(events)
			pct := used * 100 / cap
			if pct >= 90 {
				return fmt.Sprintf("warn: %d/%d (%.0f%%)", used, cap, float64(pct))
			}
			return fmt.Sprintf("ok: %d/%d", used, cap)
		},
		"notifiers": func(_ context.Context) string {
			if len(notifiers) == 0 {
				return "warn: no notifiers configured"
			}
			channels := make([]string, 0, len(notifiers))
			for ch := range notifiers {
				channels = append(channels, ch)
			}
			return fmt.Sprintf("ok: %s", strings.Join(channels, ","))
		},
	}))

	httpServer := &http.Server{Addr: listenAddr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("alarm_core_started", "listen_addr", listenAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http_server_failed", "error", err.Error())
		os.Exit(1)
	}
}

func runHeartbeatChecker(ctx context.Context, store *db.HeartbeatStore, events chan<- core.Event, logger *slog.Logger) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := store.Expired(ctx)
			if err != nil {
				logger.Error("heartbeat_checker_failed", "error", err.Error())
				continue
			}
			for _, h := range expired {
				logger.Warn("heartbeat_expired", "name", h.Name, "last_seen", h.LastSeen)
				events <- core.Event{
					Source:     h.Source,
					Keyword:    h.Keyword,
					Text:       "Heartbeat-Ausfall: " + h.Name + " — kein Signal seit " + formatLastSeen(h.LastSeen),
					ReceivedAt: time.Now(),
				}
				if err := store.MarkAlerted(ctx, h.ID); err != nil {
					logger.Error("heartbeat_mark_alerted_failed", "id", h.ID, "error", err.Error())
				}
			}
		}
	}
}

func formatLastSeen(t *time.Time) string {
	if t == nil {
		return "unbekannt"
	}
	return t.Format("02.01.2006 15:04:05")
}

func bootstrapAdmin(ctx context.Context, users *db.UserStore, logger *slog.Logger) error {
	count, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	username := getenv("ALARM_ADMIN_USERNAME", "admin")
	password := getenv("ALARM_ADMIN_PASSWORD", "admin123")
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := users.Create(ctx, username, hash, auth.RoleAdmin); err != nil {
		return err
	}
	logger.Warn("admin_user_bootstrapped",
		"username", username,
		"msg", "No users existed yet — created the initial admin account. Change its password via the UI.")
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func buildNotifiers(logger *slog.Logger) map[string]core.Notifier {
	notifiers := make(map[string]core.Notifier)

	if host := os.Getenv("ASTERISK_HOST"); host != "" {
		var synth core.Synthesizer
		if os.Getenv("TTS_ENABLED") == "true" {
			synth = espeak.New(espeak.Config{
				SoundsDir: getenv("ASTERISK_SOUNDS_DIR", "/var/lib/asterisk/sounds/custom"),
				Voice:     getenv("TTS_VOICE", "de"),
				Speed:     getenvInt("TTS_SPEED", 0),
			})
			logger.Info("tts_enabled", "voice", getenv("TTS_VOICE", "de"))
		}
		client := asterisk.New(asterisk.Config{
			Host:            host,
			Port:            getenvInt("ASTERISK_PORT", 5038),
			Username:        os.Getenv("ASTERISK_AMI_USER"),
			Password:        os.Getenv("ASTERISK_AMI_PASSWORD"),
			Announcement:    getenv("ASTERISK_ANNOUNCEMENT", "alarm-announcement"),
			ChannelTemplate: os.Getenv("ASTERISK_CHANNEL_TEMPLATE"),
			Synthesizer:     synth,
			Logger:          logger,
		})
		notifiers[client.Channel()] = client
		logger.Info("notifier_registered", "channel", client.Channel())
	}

	if baseURL := os.Getenv("SMSEAGLE_BASE_URL"); baseURL != "" {
		cfg := smseagle.Config{
			BaseURL:  baseURL,
			Login:    os.Getenv("SMSEAGLE_LOGIN"),
			Password: os.Getenv("SMSEAGLE_PASSWORD"),
			APIKey:   os.Getenv("SMSEAGLE_API_KEY"),
		}
		client := smseagle.New(cfg)
		notifiers[client.Channel()] = client
		logger.Info("notifier_registered", "channel", client.Channel())

		if os.Getenv("SMSEAGLE_VOICE_ENABLED") == "true" {
			voiceClient := smseagle.NewVoice(cfg)
			notifiers[voiceClient.Channel()] = voiceClient
			logger.Info("notifier_registered", "channel", voiceClient.Channel())
		}
	}

	if host := os.Getenv("SMTP_HOST"); host != "" {
		client := email.New(email.Config{
			Host:     host,
			Port:     getenvInt("SMTP_PORT", 587),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		})
		notifiers[client.Channel()] = client
		logger.Info("notifier_registered", "channel", client.Channel())
	}

	if webhookURL := os.Getenv("TEAMS_WEBHOOK_URL"); webhookURL != "" {
		client := teams.New(teams.Config{WebhookURL: webhookURL})
		notifiers[client.Channel()] = client
		logger.Info("notifier_registered", "channel", client.Channel())
	}

	return notifiers
}

func loadLDAPConfig(ctx context.Context, store *db.ChannelConfigStore, logger *slog.Logger) *ldapauth.Authenticator {
	cc, err := store.Load(ctx, "ldap")
	if err != nil || cc == nil || !cc.Enabled {
		return nil
	}
	cfg := cc.Config
	host := cfg["host"]
	if host == "" || cfg["bind_dn"] == "" || cfg["user_base"] == "" || cfg["user_filter"] == "" {
		logger.Warn("ldap_config_incomplete", "msg", "LDAP is enabled but missing required fields (host, bind_dn, user_base, user_filter) — skipping")
		return nil
	}
	ldapCfg := ldapauth.Config{
		Host:         host,
		TLS:          cfg["tls"] == "true",
		BindDN:       cfg["bind_dn"],
		BindPassword: cfg["bind_password"],
		UserBase:     cfg["user_base"],
		UserFilter:   cfg["user_filter"],
		AdminGroup:   cfg["admin_group"],
		DefaultRole:  cfg["default_role"],
	}
	if p, err := strconv.Atoi(cfg["port"]); err == nil && p > 0 {
		ldapCfg.Port = p
	}
	logger.Info("ldap_auth_enabled", "host", host, "tls", ldapCfg.TLS, "default_role", ldapCfg.DefaultRole)
	return ldapauth.New(ldapCfg)
}
