package notification_test

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka/notification"
)

// smtpServer adalah server SMTP tiruan di loopback. Balasannya dapat diatur per perintah,
// dan seluruh percakapan direkam supaya uji dapat memeriksa apa yang dikirim.
type smtpServer struct {
	listener net.Listener
	replies  map[string]string
	mutex    sync.Mutex
	lines    []string
	done     chan struct{}
}

func startSMTP(t *testing.T, replies map[string]string) *smtpServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &smtpServer{listener: listener, replies: replies, done: make(chan struct{})}
	t.Cleanup(func() { _ = listener.Close(); <-server.done })
	go server.serve()
	return server
}

func (s *smtpServer) reply(command, fallback string) string {
	if answer, ok := s.replies[command]; ok {
		return answer
	}
	return fallback
}

func (s *smtpServer) serve() {
	defer close(s.done)
	connection, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()
	reader := bufio.NewReader(connection)
	write := func(text string) { _, _ = connection.Write([]byte(text + "\r\n")) }

	write(s.reply("GREETING", "220 tiruan siap"))
	inData := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		s.mutex.Lock()
		s.lines = append(s.lines, line)
		s.mutex.Unlock()
		if inData {
			if line == "." {
				inData = false
				write(s.reply("END", "250 diterima"))
			}
			continue
		}
		switch strings.ToUpper(strings.SplitN(line, " ", 2)[0]) {
		case "EHLO", "HELO":
			write(s.reply("EHLO", "250 tiruan"))
		case "MAIL":
			write(s.reply("MAIL", "250 ok"))
		case "RCPT":
			write(s.reply("RCPT", "250 ok"))
		case "DATA":
			answer := s.reply("DATA", "354 lanjut")
			write(answer)
			inData = strings.HasPrefix(answer, "354")
		case "STARTTLS":
			write("220 mulai")
			return
		case "QUIT":
			write("221 sampai jumpa")
			return
		default:
			write("250 ok")
		}
	}
}

func (s *smtpServer) transcript() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return strings.Join(s.lines, "\n")
}

func (s *smtpServer) config() notification.Config {
	host, port, _ := net.SplitHostPort(s.listener.Addr().String())
	number, _ := strconv.Atoi(port)
	return notification.Config{Host: host, Port: number, From: "pnc@contoh.invalid",
		To: []string{" admin@contoh.invalid ", "  "}, Timeout: 5 * time.Second}
}

func TestSenderDeliversThroughSMTP(t *testing.T) {
	server := startSMTP(t, nil)
	err := notification.NewSender(server.config()).NotifyDocumentCompleted(context.Background(), sampleNotice())
	require.NoError(t, err)
	transcript := server.transcript()
	require.Contains(t, transcript, "MAIL FROM:<pnc@contoh.invalid>")
	require.Contains(t, transcript, "RCPT TO:<admin@contoh.invalid>")
	require.Contains(t, transcript, "Subject: "+notification.Subject(sampleNotice()))
	require.Contains(t, transcript, "PNC-200118")
}

func TestSenderReportsServerRejections(t *testing.T) {
	cases := map[string]struct {
		replies map[string]string
		message string
	}{
		"greeting": {map[string]string{"GREETING": "554 tidak"}, "memulai percakapan SMTP"},
		"mail":     {map[string]string{"MAIL": "550 tidak"}, "alamat pengirim"},
		"rcpt":     {map[string]string{"RCPT": "550 tidak"}, "alamat tujuan"},
		"data":     {map[string]string{"DATA": "554 tidak"}, "membuka badan surel"},
		"end":      {map[string]string{"END": "554 tidak"}, "menutup badan surel"},
		"starttls": {map[string]string{"EHLO": "250-tiruan\r\n250 STARTTLS"}, "menegakkan TLS"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := startSMTP(t, tc.replies)
			err := notification.NewSender(server.config()).NotifyDocumentCompleted(context.Background(), sampleNotice())
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestSenderRefusesCredentialsWithoutTLS(t *testing.T) {
	server := startSMTP(t, nil)
	config := server.config()
	config.User, config.Password = "user", "rahasia"
	config.Timeout = 0 // jatuh ke batas waktu bawaan
	err := notification.NewSender(config).NotifyDocumentCompleted(context.Background(), sampleNotice())
	require.ErrorContains(t, err, "tidak mendukung STARTTLS")
	require.NotContains(t, server.transcript(), "rahasia")
}

func TestSenderDialFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, listener.Close())
	number, _ := strconv.Atoi(port)

	err = notification.NewSender(notification.Config{Host: host, Port: number, From: "a@b",
		To: []string{"c@d"}, Timeout: time.Second}).NotifyDocumentCompleted(context.Background(), sampleNotice())
	require.ErrorContains(t, err, "menghubungi server surel")
}

func TestSenderStripsHeaderInjection(t *testing.T) {
	server := startSMTP(t, nil)
	config := server.config()
	config.From = "pnc@contoh.invalid"
	notice := sampleNotice()
	notice.ClaimNumber = "PNC-1\r\nBcc: x@y"
	require.NoError(t, notification.NewSender(config).NotifyDocumentCompleted(context.Background(), notice))
	// Baris baru di subjek diganti spasi sehingga tidak menjadi header sendiri.
	require.Contains(t, server.transcript(), "PNC-1  Bcc: x@y")
}
