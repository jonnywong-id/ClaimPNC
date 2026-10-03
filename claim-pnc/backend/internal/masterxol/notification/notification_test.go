package notification

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

	"claim-pnc/internal/masterxol"
)

func sampleSubmission() masterxol.CommitteeSubmission {
	return masterxol.CommitteeSubmission{MasterID: "10001", Year: "2025", ExchangeRate: 15000,
		Remark: "<b>tolong</b>", SubmittedBy: "PIC1"}
}

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
		command := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch command {
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

func (s *smtpServer) config() Config {
	host, port, _ := net.SplitHostPort(s.listener.Addr().String())
	number, _ := strconv.Atoi(port)
	return Config{Host: host, Port: number, From: "pnc@contoh.invalid",
		To: []string{" komite@contoh.invalid ", " "}, Timeout: 5 * time.Second}
}

func TestConfigComplete(t *testing.T) {
	full := Config{Host: "h", Port: 25, From: "a@b", To: []string{"c@d"}}
	require.True(t, full.Complete())
	for name, mutate := range map[string]func(*Config){
		"host": func(c *Config) { c.Host = " " },
		"port": func(c *Config) { c.Port = 0 },
		"from": func(c *Config) { c.From = "" },
		"to":   func(c *Config) { c.To = []string{" "} },
	} {
		c := full
		mutate(&c)
		require.False(t, c.Complete(), name)
	}
}

func TestNotifyRefusesIncompleteConfig(t *testing.T) {
	err := NewSender(Config{}).NotifyCommitteeSubmission(context.Background(), sampleSubmission())
	require.ErrorContains(t, err, "SMTP belum dikonfigurasi")
}

func TestNotifySendsMessage(t *testing.T) {
	server := startSMTP(t, nil)
	err := NewSender(server.config()).NotifyCommitteeSubmission(context.Background(), sampleSubmission())
	require.NoError(t, err)
	transcript := server.transcript()
	require.Contains(t, transcript, "MAIL FROM:<pnc@contoh.invalid>")
	require.Contains(t, transcript, "RCPT TO:<komite@contoh.invalid>")
	require.Contains(t, transcript, "Subject: Notif Pemberitahuan Pengajuan Master XOL Tahun - 2025 dengan ID - 10001")
	require.Contains(t, transcript, "&lt;b&gt;tolong&lt;/b&gt;")
}

func TestNotifyServerRejections(t *testing.T) {
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
			err := NewSender(server.config()).NotifyCommitteeSubmission(context.Background(), sampleSubmission())
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestNotifyRefusesCredentialsWithoutTLS(t *testing.T) {
	server := startSMTP(t, nil)
	config := server.config()
	config.User, config.Password = "user", "rahasia"
	config.Timeout = 0 // jatuh ke batas waktu bawaan
	err := NewSender(config).NotifyCommitteeSubmission(context.Background(), sampleSubmission())
	require.ErrorContains(t, err, "tidak mendukung STARTTLS")
	require.NotContains(t, server.transcript(), "rahasia")
}

func TestNotifyDialFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	host, port, _ := net.SplitHostPort(address)
	number, _ := strconv.Atoi(port)

	err = NewSender(Config{Host: host, Port: number, From: "a@b", To: []string{"c@d"}, Timeout: time.Second}).
		NotifyCommitteeSubmission(context.Background(), sampleSubmission())
	require.ErrorContains(t, err, "menghubungi server surel")
}

func TestComposeEmailStripsHeaderInjection(t *testing.T) {
	message := string(composeEmail("a@b\r\nBcc: x@y", []string{"c@d"}, masterxol.CommitteeSubmission{Year: "2025\nX", MasterID: "1"}))
	require.Contains(t, message, "From: a@b  Bcc: x@y\r\n")
	require.NotContains(t, message, "\nBcc:")
	require.Contains(t, message, "Content-Type: text/html; charset=UTF-8")
}

func TestHTMLBodyEscapesValues(t *testing.T) {
	body := HTMLBody(sampleSubmission())
	require.Contains(t, body, "<strong>15000</strong>")
	require.Contains(t, body, "&lt;b&gt;tolong&lt;/b&gt;")
}

func TestFakeRecordsSubmissions(t *testing.T) {
	fake := NewFake()
	require.Equal(t, 0, fake.Count())
	require.NoError(t, fake.NotifyCommitteeSubmission(context.Background(), sampleSubmission()))
	require.Equal(t, 1, fake.Count())
	sent := fake.Sent()
	require.Equal(t, []masterxol.CommitteeSubmission{sampleSubmission()}, sent)
	sent[0].MasterID = "diubah"
	require.Equal(t, "10001", fake.Sent()[0].MasterID)
}
