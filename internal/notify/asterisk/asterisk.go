// Package asterisk implements core.Notifier for the "voice" channel via
// the Asterisk Manager Interface (AMI). It originates one outbound call
// per recipient and plays an announcement on answer.
package asterisk

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

// amiReasonText maps an Asterisk OriginateResponse Reason code to a readable string.
// Reason 4 = ANSWER (call was answered); all others mean not answered.
func amiReasonText(reason string) string {
	switch reason {
	case "4":
		return "ANSWER"
	case "0":
		return "HANGUP"
	case "1":
		return "BUSY"
	case "3":
		return "NOANSWER"
	case "5":
		return "CONGESTION"
	default:
		return "REASON_" + reason
	}
}

// cleanupDelay must outlast the AMI Originate Timeout below: a successful
// "Response: Success" only means the call setup was accepted, not that the
// person answered and Playback ran. Deleting the sound file right after
// Notify returns would race actual playback, so cleanup is deferred well
// past the 30s Originate timeout instead of running synchronously.
const cleanupDelay = 90 * time.Second

type Config struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Announcement string // static sound file passed to Playback, e.g. "alarm-announcement"
	// Synthesizer, if set, renders the notification text to speech and
	// plays that instead of the static Announcement. Falls back to
	// Announcement if synthesis fails for a given call.
	Synthesizer core.Synthesizer
	// ChannelTemplate builds the Originate "Channel:" value for a
	// recipient's Voice number. "{number}" is replaced with that number.
	// Defaults to "PJSIP/{number}": chan_sip is deprecated/removed in
	// modern Asterisk (20+, as shipped on Debian 13), PJSIP is the
	// supported replacement. Override per deployment to route through a
	// specific trunk, e.g. "PJSIP/{number}@out-trunk".
	ChannelTemplate string
	DialTimeout     time.Duration
	Logger          *slog.Logger
	// DTMFAck, when true, uses a dialplan context instead of Application:Playback
	// so Asterisk can collect a DTMF digit and POST an ACK back to AckURL.
	// Requires the [alarm-ack] context to be configured in Asterisk (see README).
	DTMFAck    bool
	DTMFContext string // defaults to "alarm-ack"
	AckURL      string // e.g. "http://localhost:8080/api/inbound/dtmf"
}

type Client struct {
	cfg Config
	// dial is overridable in tests to avoid a real TCP connection.
	dial func(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error)
}

func New(cfg Config) *Client {
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ChannelTemplate == "" {
		cfg.ChannelTemplate = "PJSIP/{number}"
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Client{cfg: cfg, dial: dialTCP}
}

func dialTCP(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, "tcp", addr)
}

func (c *Client) Channel() string { return "voice" }

// Notify logs into AMI once and originates one call per recipient with a
// non-empty Voice number, playing message as speech if a Synthesizer is
// configured (falling back to the static Announcement otherwise or on
// synthesis failure). It returns an error summarizing any failed originate
// attempts, but always tries every recipient.
func (c *Client) Notify(ctx context.Context, recipients []core.Recipient, message string) error {
	soundName := c.cfg.Announcement
	if c.cfg.Synthesizer != nil {
		name, cleanup, err := c.cfg.Synthesizer.Synthesize(ctx, message)
		if err != nil {
			c.cfg.Logger.Warn("tts_synthesis_failed", "error", err.Error(), "fallback", c.cfg.Announcement)
		} else {
			soundName = name
			time.AfterFunc(cleanupDelay, cleanup)
		}
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	conn, err := c.dial(ctx, addr, c.cfg.DialTimeout)
	if err != nil {
		return fmt.Errorf("asterisk: dial %s: %w", addr, err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	// Banner line, e.g. "Asterisk Call Manager/X.Y.Z".
	if _, err := reader.ReadString('\n'); err != nil {
		return fmt.Errorf("asterisk: read banner: %w", err)
	}

	if err := c.login(conn, reader); err != nil {
		return fmt.Errorf("asterisk: login: %w", err)
	}

	var failed []string
	attempted := 0
	for _, r := range recipients {
		if r.Voice == "" {
			continue
		}
		attempted++
		if err := c.originate(conn, reader, r.Voice, soundName); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", r.Voice, err))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("asterisk: %d/%d calls failed: %s", len(failed), attempted, strings.Join(failed, "; "))
	}
	return nil
}

func (c *Client) login(conn net.Conn, reader *bufio.Reader) error {
	action := fmt.Sprintf("Action: Login\r\nUsername: %s\r\nSecret: %s\r\n\r\n", c.cfg.Username, c.cfg.Password)
	if _, err := conn.Write([]byte(action)); err != nil {
		return err
	}
	resp, err := readAMIBlock(reader)
	if err != nil {
		return err
	}
	if resp["Response"] != "Success" {
		return fmt.Errorf("login rejected: %s", resp["Message"])
	}
	return nil
}

func (c *Client) originate(conn net.Conn, reader *bufio.Reader, number, soundName string) error {
	return c.originateFull(conn, reader, number, soundName, 0, 0)
}

func (c *Client) originateFull(conn net.Conn, reader *bufio.Reader, number, soundName string, eventID, recipientID int) error {
	channel := strings.ReplaceAll(c.cfg.ChannelTemplate, "{number}", number)

	var action string
	if c.cfg.DTMFAck && c.cfg.AckURL != "" {
		ctx := c.cfg.DTMFContext
		if ctx == "" {
			ctx = "alarm-ack"
		}
		action = fmt.Sprintf(
			"Action: Originate\r\n"+
				"Channel: %s\r\n"+
				"Context: %s\r\n"+
				"Exten: s\r\n"+
				"Priority: 1\r\n"+
				"Timeout: 30000\r\n"+
				"Variable: ALARM_SOUND=%s\r\n"+
				"Variable: ALARM_EVENT_ID=%d\r\n"+
				"Variable: ALARM_RECIPIENT_ID=%d\r\n"+
				"Variable: ALARM_ACK_URL=%s\r\n\r\n",
			channel, ctx, soundName, eventID, recipientID, c.cfg.AckURL,
		)
	} else {
		action = fmt.Sprintf(
			"Action: Originate\r\n"+
				"Channel: %s\r\n"+
				"Application: Playback\r\n"+
				"Data: %s\r\n"+
				"Timeout: 30000\r\n\r\n",
			channel, soundName,
		)
	}

	if _, err := conn.Write([]byte(action)); err != nil {
		return err
	}
	resp, err := readAMIBlock(reader)
	if err != nil {
		return err
	}
	if resp["Response"] != "Success" {
		return fmt.Errorf("originate rejected: %s", resp["Message"])
	}
	return nil
}

// readAMIBlock reads one AMI key/value block terminated by a blank line.
func readAMIBlock(r *bufio.Reader) (map[string]string, error) {
	fields := map[string]string{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if key, val, ok := strings.Cut(line, ": "); ok {
			fields[key] = val
		}
	}
	return fields, nil
}

// NotifyTracked implements core.VoiceNotifier: it originates one async call
// per recipient and sends a core.CallResult for each when the call completes
// (answered, busy, no-answer) or the 45-second deadline expires. The caller
// reads from the returned channel until it is closed to know which recipients
// did not answer and need escalation.
func (c *Client) NotifyTracked(ctx context.Context, recipients []core.Recipient, message string) (<-chan core.CallResult, error) {
	soundName := c.cfg.Announcement
	if c.cfg.Synthesizer != nil {
		name, cleanup, err := c.cfg.Synthesizer.Synthesize(ctx, message)
		if err != nil {
			c.cfg.Logger.Warn("tts_synthesis_failed", "error", err.Error(), "fallback", c.cfg.Announcement)
		} else {
			soundName = name
			time.AfterFunc(cleanupDelay, cleanup)
		}
	}

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	conn, err := c.dial(ctx, addr, c.cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("asterisk: dial %s: %w", addr, err)
	}

	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		conn.Close()
		return nil, fmt.Errorf("asterisk: read banner: %w", err)
	}
	if err := c.login(conn, reader); err != nil {
		conn.Close()
		return nil, fmt.Errorf("asterisk: login: %w", err)
	}

	// pending maps ActionID → recipientID for every call we successfully queued.
	pending := make(map[string]int)
	for _, r := range recipients {
		if r.Voice == "" {
			continue
		}
		actionID := fmt.Sprintf("alarm-%d-%d", r.ID, time.Now().UnixNano())
		if err := c.originateAsync(conn, r.Voice, soundName, actionID); err != nil {
			c.cfg.Logger.Warn("originate_async_failed", "recipient_id", r.ID, "error", err.Error())
			continue
		}
		pending[actionID] = r.ID
	}

	results := make(chan core.CallResult, max(len(pending), 1))
	if len(pending) == 0 {
		close(results)
		conn.Close()
		return results, nil
	}

	// Collect OriginateResponse events. The Originate timeout is 30 s (30000 ms
	// in the action), so 45 s is enough to catch every outcome.
	go func() {
		defer conn.Close()
		defer close(results)
		conn.SetDeadline(time.Now().Add(45 * time.Second)) //nolint:errcheck
		for len(pending) > 0 {
			block, err := readAMIBlock(reader)
			if err != nil {
				for _, rid := range pending {
					results <- core.CallResult{RecipientID: rid, Answered: false, Reason: "TIMEOUT"}
				}
				return
			}
			if block["Event"] != "OriginateResponse" {
				continue // skip ACK blocks, FullyBooted, etc.
			}
			rid, ok := pending[block["ActionID"]]
			if !ok {
				continue
			}
			delete(pending, block["ActionID"])
			answered := block["Reason"] == "4"
			results <- core.CallResult{RecipientID: rid, Answered: answered, Reason: amiReasonText(block["Reason"])}
		}
	}()

	return results, nil
}

func (c *Client) originateAsync(conn net.Conn, number, soundName, actionID string) error {
	channel := strings.ReplaceAll(c.cfg.ChannelTemplate, "{number}", number)
	action := fmt.Sprintf(
		"Action: Originate\r\n"+
			"Channel: %s\r\n"+
			"Application: Playback\r\n"+
			"Data: %s\r\n"+
			"Timeout: 30000\r\n"+
			"Async: yes\r\n"+
			"ActionID: %s\r\n\r\n",
		channel, soundName, actionID,
	)
	_, err := conn.Write([]byte(action))
	return err
}
