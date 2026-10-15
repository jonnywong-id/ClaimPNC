package notification

import (
	"bufio"
	"context"
	"errors"
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
	return Config{
		Host: "127.0.0.1", Port: server.port(), From: "claimpnc@contoh.co.id",
		To: []string{" it1@contoh.co.id ", "", "it2@contoh.co.id"}, Timeout: 5 * time.Second,
	}
}

func TestTheAlertIsSentToEveryRecipient(t *testing.T) {
	server := startFakeSMTP(t, "")
	err := NewSender(configFor(server)).WarnCashierFailure(context.Background(), sampleAlert())
	require.NoError(t, err)

	commands, data := server.snapshot()
	require.Contains(t, commands, "MAIL FROM:<claimpnc@contoh.co.id>")
	require.Contains(t, commands, "RCPT TO:<it1@contoh.co.id>")
	require.Contains(t, commands, "RCPT TO:<it2@contoh.co.id>")
	require.Contains(t, data, "Subject: Pendaftaran rekening ke Kasir GAGAL")
	require.Contains(t, data, "1234567890")
}

func TestCredentialsAreNotSentWithoutTLS(t *testing.T) {
	server := startFakeSMTP(t, "")
	config := configFor(server)
	config.User, config.Password = "u", "rahasia"

	err := NewSender(config).WarnCashierFailure(context.Background(), sampleAlert())
	require.ErrorContains(t, err, "tidak mendukung STARTTLS")
	commands, _ := server.snapshot()
	for _, command := range commands {
		require.NotContains(t, command, "AUTH")
	}
}

func TestServerRejectionsAreReported(t *testing.T) {
	cases := map[string]string{
		"MAIL": "menolak alamat pengirim",
		"RCPT": "menolak alamat tujuan",
		"DATA": "membuka badan surel",
	}
	for verb, message := range cases {
		t.Run(verb, func(t *testing.T) {
			server := startFakeSMTP(t, verb)
			err := NewSender(configFor(server)).WarnCashierFailure(context.Background(), sampleAlert())
			require.ErrorContains(t, err, message)
		})
	}
}

func TestAnUnreachableServerIsReported(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	err = NewSender(Config{
		Host: "127.0.0.1", Port: port, From: "a@contoh.co.id", To: []string{"b@contoh.co.id"},
	}).WarnCashierFailure(context.Background(), sampleAlert())
	require.ErrorContains(t, err, "menghubungi server surel")
}

func TestAServerThatIsNotSMTPIsReported(t *testing.T) {
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

	err = NewSender(Config{
		Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port,
		From: "a@contoh.co.id", To: []string{"b@contoh.co.id"}, Timeout: 5 * time.Second,
	}).WarnCashierFailure(context.Background(), sampleAlert())
	require.ErrorContains(t, err, "memulai percakapan SMTP")
}

func TestFakeRecordsEveryAlert(t *testing.T) {
	fake := &Fake{}
	require.NoError(t, fake.WarnCashierFailure(context.Background(), sampleAlert()))
	fake.Error = errors.New("mati")
	require.EqualError(t, fake.WarnCashierFailure(context.Background(), sampleAlert()), "mati")
	require.Equal(t, 2, fake.Count())
}
