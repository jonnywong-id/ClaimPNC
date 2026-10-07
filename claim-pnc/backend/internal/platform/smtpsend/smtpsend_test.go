package smtpsend

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeSMTP adalah peladen SMTP minimal di localhost untuk menguji pengiriman tanpa
// jaringan luar. Ia tidak menawarkan STARTTLS, dan dapat diatur untuk menolak satu
// perintah tertentu.
type fakeSMTP struct {
	listener net.Listener
	rejectOn string

	mu       sync.Mutex
	commands []string
	data     string
}

func startFakeSMTP(t *testing.T, rejectOn string) *fakeSMTP {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &fakeSMTP{listener: listener, rejectOn: rejectOn}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		server.serve(conn)
	}()
	return server
}

func (s *fakeSMTP) serve(conn net.Conn) {
	reader := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
	write("220 uji ESMTP")

	inData := false
	var body strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.data = body.String()
				s.mu.Unlock()
				write("250 diterima")
				continue
			}
			body.WriteString(line + "\n")
			continue
		}

		verb := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		s.mu.Lock()
		s.commands = append(s.commands, line)
		s.mu.Unlock()

		if verb == s.rejectOn {
			write("550 ditolak")
			continue
		}
		switch verb {
		case "EHLO", "HELO":
			write("250 uji")
		case "MAIL", "RCPT":
			write("250 ok")
		case "DATA":
			inData = true
			write("354 lanjut")
		case "QUIT":
			write("221 sampai jumpa")
			return
		default:
			write("502 tidak dikenal")
		}
	}
}

func (s *fakeSMTP) port() int { return s.listener.Addr().(*net.TCPAddr).Port }

func (s *fakeSMTP) snapshot() ([]string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...), s.data
}

func configFor(server *fakeSMTP) Config {
	return Config{Host: "127.0.0.1", Port: server.port(), From: "claimpnc@contoh.co.id",
		To: []string{" a@contoh.co.id ", "", "b@contoh.co.id"}, Timeout: 5 * time.Second}
}

func TestConfigCompleteAndRecipients(t *testing.T) {
	require.False(t, Config{}.Complete())
	c := Config{Host: "h", Port: 25, From: "f", To: []string{" x ", ""}}
	require.True(t, c.Complete())
	require.Equal(t, []string{"x"}, c.Recipients())
}

func TestSendDeliversToEveryRecipient(t *testing.T) {
	server := startFakeSMTP(t, "")
	c := configFor(server)
	require.NoError(t, c.Send(context.Background(), c.Recipients(), []byte("Subject: uji\r\n\r\nisi\r\n"), "m/notification"))
	commands, data := server.snapshot()
	require.Contains(t, commands, "RCPT TO:<a@contoh.co.id>")
	require.Contains(t, commands, "RCPT TO:<b@contoh.co.id>")
	require.Contains(t, data, "isi")
}

func TestSendReportsFailures(t *testing.T) {
	for _, verb := range []string{"MAIL", "RCPT", "DATA"} {
		server := startFakeSMTP(t, verb)
		c := configFor(server)
		err := c.Send(context.Background(), c.Recipients(), []byte("x"), "m/notification")
		require.ErrorContains(t, err, "m/notification: ")
	}

	server := startFakeSMTP(t, "")
	c := configFor(server)
	c.User, c.Password = "u", "p"
	require.ErrorContains(t, c.Send(context.Background(), c.Recipients(), []byte("x"), "m/notification"),
		"tidak mendukung STARTTLS")

	closed, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := closed.Addr().(*net.TCPAddr).Port
	_ = closed.Close()
	unreachable := Config{Host: "127.0.0.1", Port: port, From: "f", To: []string{"x"}}
	require.ErrorContains(t, unreachable.Send(context.Background(), []string{"x"}, nil, "m/notification"),
		"menghubungi server surel")
}

func TestSendRejectsNonSMTPServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_, _ = conn.Write([]byte("bukan smtp\r\n"))
			_ = conn.Close()
		}
	}()
	c := Config{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, From: "f", To: []string{"x"}, Timeout: time.Second}
	require.ErrorContains(t, c.Send(context.Background(), []string{"x"}, nil, "m/notification"), "memulai percakapan SMTP")
}
