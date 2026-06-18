package asterisk

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/bh2005/alarmserver/internal/core"
)

// fakeAMIServer accepts one connection, sends a banner, accepts Login and
// then responds Success to every Originate it receives.
func fakeAMIServer(t *testing.T, originateOutcome func(channel string) string) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("Asterisk Call Manager/9.0.0\r\n"))

		reader := bufio.NewReader(conn)
		for {
			block, err := readBlockRaw(reader)
			if err != nil {
				return
			}
			if strings.Contains(block, "Action: Login") {
				conn.Write([]byte("Response: Success\r\nMessage: Authentication accepted\r\n\r\n"))
				continue
			}
			if strings.Contains(block, "Action: Originate") {
				outcome := "Success"
				if originateOutcome != nil {
					outcome = originateOutcome(block)
				}
				conn.Write([]byte("Response: " + outcome + "\r\nMessage: done\r\n\r\n"))
			}
		}
	}()
	return ln
}

// readBlockRaw reads lines until a blank line and returns them joined.
func readBlockRaw(r *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}
		sb.WriteString(line)
		if strings.TrimRight(line, "\r\n") == "" {
			return sb.String(), nil
		}
	}
}

func clientFor(t *testing.T, ln net.Listener) *Client {
	addr := ln.Addr().(*net.TCPAddr)
	c := New(Config{
		Host:         "127.0.0.1",
		Port:         addr.Port,
		Username:     "user",
		Password:     "pass",
		Announcement: "alarm-announcement",
		DialTimeout:  2 * time.Second,
	})
	return c
}

func TestNotify_AllSucceed(t *testing.T) {
	ln := fakeAMIServer(t, nil)
	defer ln.Close()
	c := clientFor(t, ln)

	err := c.Notify(context.Background(), []core.Recipient{
		{Name: "Alice", Voice: "1001"},
		{Name: "Bob", Voice: "1002"},
	}, "test message")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNotify_SkipsRecipientsWithoutVoiceNumber(t *testing.T) {
	ln := fakeAMIServer(t, nil)
	defer ln.Close()
	c := clientFor(t, ln)

	err := c.Notify(context.Background(), []core.Recipient{
		{Name: "NoPhone"},
	}, "test message")
	if err != nil {
		t.Fatalf("expected no error when no recipient has a voice number, got %v", err)
	}
}

func TestNotify_ReportsFailedOriginate(t *testing.T) {
	ln := fakeAMIServer(t, func(block string) string {
		if strings.Contains(block, "1002") {
			return "Error"
		}
		return "Success"
	})
	defer ln.Close()
	c := clientFor(t, ln)

	err := c.Notify(context.Background(), []core.Recipient{
		{Name: "Alice", Voice: "1001"},
		{Name: "Bob", Voice: "1002"},
	}, "test message")
	if err == nil {
		t.Fatal("expected error for failed originate")
	}
	if !strings.Contains(err.Error(), "1002") {
		t.Fatalf("expected error to mention failed number 1002, got: %v", err)
	}
}

func TestNotify_LoginRejected(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.Write([]byte("Asterisk Call Manager/9.0.0\r\n"))
		reader := bufio.NewReader(conn)
		readBlockRaw(reader)
		conn.Write([]byte("Response: Error\r\nMessage: Authentication failed\r\n\r\n"))
	}()

	c := clientFor(t, ln)
	err = c.Notify(context.Background(), []core.Recipient{{Voice: "1001"}}, "test")
	if err == nil {
		t.Fatal("expected login error")
	}
}
