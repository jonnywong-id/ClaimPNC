package notification_test

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/notification"
)

// relayPalsu adalah peladen SMTP minimal di loopback. Ia menjawab setiap perintah menurut
// tabel jawaban, dan merekam perintah serta badan surat yang diterimanya.
type relayPalsu struct {
	listener net.Listener
	greeting string
	ehlo     []string
	replies  map[string]string

	mu       sync.Mutex
	commands []string
	data     string
	done     chan struct{}
}

func startRelay(t *testing.T, configure func(*relayPalsu)) *relayPalsu {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	relay := &relayPalsu{
		listener: listener,
		greeting: "220 relay.uji ESMTP",
		ehlo:     []string{"relay.uji"},
		replies: map[string]string{
			"MAIL": "250 OK", "RCPT": "250 OK", "DATA": "354 lanjut",
			"QUIT": "221 sampai jumpa", "STARTTLS": "220 siap",
			"BODY": "250 diterima", "RSET": "250 OK", "NOOP": "250 OK",
		},
		done: make(chan struct{}),
	}
	if configure != nil {
		configure(relay)
	}

	go relay.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-relay.done:
		case <-time.After(5 * time.Second):
		}
	})
	return relay
}

func (r *relayPalsu) port() int { return r.listener.Addr().(*net.TCPAddr).Port }

func (r *relayPalsu) serve() {
	defer close(r.done)

	conn, err := r.listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)
	write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	write(r.greeting)
	if !strings.HasPrefix(r.greeting, "220") {
		return
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimSpace(line)
		verb := strings.ToUpper(strings.SplitN(command, " ", 2)[0])

		r.mu.Lock()
		r.commands = append(r.commands, command)
		r.mu.Unlock()

		switch verb {
		case "EHLO", "HELO":
			for i, ext := range r.ehlo {
				sep := "-"
				if i == len(r.ehlo)-1 {
					sep = " "
				}
				write("250" + sep + ext)
			}
		case "DATA":
			reply := r.replies["DATA"]
			write(reply)
			if !strings.HasPrefix(reply, "354") {
				continue
			}
			var body strings.Builder
			for {
				part, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if part == ".\r\n" {
					break
				}
				body.WriteString(part)
			}
			r.mu.Lock()
			r.data = body.String()
			r.mu.Unlock()
			write(r.replies["BODY"])
		case "STARTTLS":
			write(r.replies["STARTTLS"])
			// Klien tidak dapat berjabat tangan TLS tanpa konfigurasi; tunggu ia menutup.
		case "QUIT":
			write(r.replies["QUIT"])
			return
		default:
			reply, ok := r.replies[verb]
			if !ok {
				reply = "250 OK"
			}
			write(reply)
		}
	}
}

func (r *relayPalsu) received() ([]string, string) {
	<-r.done
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.commands...), r.data
}

func senderFor(relay *relayPalsu) *notification.Sender {
	return notification.NewSender(notification.Config{
		Host: "127.0.0.1", Port: relay.port(), From: "auto@contoh.example",
		Timeout: 3 * time.Second,
	})
}

func sampleLetter() inboxpladlapredla.Letter {
	return inboxpladlapredla.Letter{
		To:       []string{"reas-a@contoh.example", "reas-b@contoh.example"},
		Subject:  "PLA Supporting Document Claim Ané -- PNC-1",
		HTMLBody: "<p>badan</p>",
		Attachments: []inboxpladlapredla.Attachment{
			{Name: "dokumen.pdf", Content: []byte("isi pdf")},
			{Name: "berkas.zzzunknown", Content: []byte("x")},
			{Name: "folder/gambar.png", MIMEType: "image/png", Content: []byte("png")},
		},
	}
}

// Surat terkirim ke setiap penerima, dengan lampiran berjenis yang dapat ditebak.
func TestSendAdviceDeliversTheLetterToEveryRecipient(t *testing.T) {
	relay := startRelay(t, nil)

	err := senderFor(relay).SendAdvice(context.Background(), sampleLetter())
	require.NoError(t, err)

	commands, data := relay.received()
	require.Contains(t, commands, "MAIL FROM:<auto@contoh.example>")
	require.Contains(t, commands, "RCPT TO:<reas-a@contoh.example>")
	require.Contains(t, commands, "RCPT TO:<reas-b@contoh.example>")

	require.Contains(t, data, "From: auto@contoh.example")
	require.Contains(t, data, "To: reas-a@contoh.example, reas-b@contoh.example")
	require.Contains(t, data, "Subject: =?UTF-8?q?")
	require.Contains(t, data, "Content-Type: application/pdf")
	require.Contains(t, data, "Content-Type: application/octet-stream")
	require.Contains(t, data, "Content-Type: image/png")
	require.Contains(t, data, `filename="gambar.png"`)
	require.Contains(t, data, base64.StdEncoding.EncodeToString([]byte("<p>badan</p>")))
}

// Base64 yang panjang dipotong menjadi baris 76 karakter.
func TestLongAttachmentsAreWrappedAtSeventySixCharacters(t *testing.T) {
	relay := startRelay(t, nil)

	letter := inboxpladlapredla.Letter{
		To: []string{"a@contoh.example"}, Subject: "ascii saja",
		Attachments: []inboxpladlapredla.Attachment{
			{Name: "besar.bin", Content: []byte(strings.Repeat("a", 300))},
		},
	}
	require.NoError(t, senderFor(relay).SendAdvice(context.Background(), letter))

	_, data := relay.received()
	require.Contains(t, data, "Subject: ascii saja\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 300)))
	require.Contains(t, data, encoded[:76]+"\r\n"+encoded[76:152])
}

// Pemutus baris pada data tidak dapat menyisipkan header baru.
func TestHeaderInjectionIsNeutralised(t *testing.T) {
	relay := startRelay(t, nil)

	letter := inboxpladlapredla.Letter{
		To:      []string{"a@contoh.example"},
		Subject: "Klaim\r\nBcc: penyusup@contoh.example",
	}
	require.NoError(t, senderFor(relay).SendAdvice(context.Background(), letter))

	_, data := relay.received()
	require.Contains(t, data, "Subject: Klaim  Bcc: penyusup@contoh.example\r\n")
	require.NotContains(t, data, "\r\nBcc:")
}

// Konfigurasi yang belum lengkap dan surat tanpa penerima ditolak sebelum tersambung.
func TestSendAdviceRejectsBeforeConnecting(t *testing.T) {
	err := notification.NewSender(notification.Config{}).
		SendAdvice(context.Background(), sampleLetter())
	require.ErrorIs(t, err, inboxpladlapredla.ErrNotifierUnavailable)
	require.Contains(t, err.Error(), "host SMTP, port SMTP, alamat pengirim belum diisi")

	err = notification.NewSender(notification.Config{
		Host: "127.0.0.1", Port: 25, From: "a@contoh.example",
	}).SendAdvice(context.Background(), inboxpladlapredla.Letter{})
	require.EqualError(t, err, "inboxpladlapredla/notification: tidak ada penerima")
}

// Peladen yang tidak dapat dihubungi menghasilkan galat yang menyebut alamatnya.
func TestAnUnreachableRelayIsReported(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	err = notification.NewSender(notification.Config{
		Host: "127.0.0.1", Port: port, From: "a@contoh.example",
	}).SendAdvice(context.Background(), sampleLetter())
	require.ErrorContains(t, err, "menghubungi 127.0.0.1:"+strconv.Itoa(port))
}

// Setiap penolakan relay menghentikan pengiriman dengan galat yang menyebut tahapnya.
func TestRelayRejectionsStopTheSending(t *testing.T) {
	cases := []struct {
		name      string
		configure func(*relayPalsu)
		message   string
	}{
		{"salam ditolak", func(r *relayPalsu) { r.greeting = "554 tidak melayani" },
			"memulai SMTP"},
		{"pengirim ditolak", func(r *relayPalsu) { r.replies["MAIL"] = "550 pengirim" },
			"pengirim ditolak"},
		{"penerima ditolak", func(r *relayPalsu) { r.replies["RCPT"] = "550 penerima" },
			"penerima ditolak"},
		{"badan ditolak", func(r *relayPalsu) { r.replies["DATA"] = "554 tidak" },
			"membuka badan"},
		{"penutupan ditolak", func(r *relayPalsu) { r.replies["BODY"] = "554 spam" },
			"menutup badan"},
		{"STARTTLS gagal", func(r *relayPalsu) { r.ehlo = []string{"relay.uji", "STARTTLS"} },
			"menegakkan TLS"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			relay := startRelay(t, c.configure)
			err := senderFor(relay).SendAdvice(context.Background(), sampleLetter())
			require.ErrorContains(t, err, c.message)
		})
	}
}

// Penerima yang ditolak menghentikan SELURUH pengiriman — tidak ada DATA yang dikirim.
func TestARejectedRecipientSendsNoBody(t *testing.T) {
	relay := startRelay(t, func(r *relayPalsu) { r.replies["RCPT"] = "550 penerima" })

	err := senderFor(relay).SendAdvice(context.Background(), sampleLetter())
	require.Error(t, err)

	commands, data := relay.received()
	for _, command := range commands {
		require.NotEqual(t, "DATA", command)
	}
	require.Empty(t, data)
}

// Kredensial tidak pernah dikirim lewat sambungan tanpa STARTTLS.
func TestCredentialsAreNeverSentWithoutStartTLS(t *testing.T) {
	relay := startRelay(t, nil)

	sender := notification.NewSender(notification.Config{
		Host: "127.0.0.1", Port: relay.port(), From: "auto@contoh.example",
		User: "pengguna", Password: "rahasia",
	})
	err := sender.SendAdvice(context.Background(), sampleLetter())
	require.ErrorContains(t, err, "kredensial SMTP tidak dikirim lewat sambungan terbuka")
	require.NotContains(t, err.Error(), "rahasia")

	commands, _ := relay.received()
	for _, command := range commands {
		require.NotContains(t, command, "AUTH")
	}
}

// Tanpa batas waktu tertulis, pengiriman tetap berjalan dengan batas bawaan.
func TestTheDefaultTimeoutStillDelivers(t *testing.T) {
	relay := startRelay(t, nil)

	sender := notification.NewSender(notification.Config{
		Host: "127.0.0.1", Port: relay.port(), From: "auto@contoh.example",
	})
	require.NoError(t, sender.SendAdvice(context.Background(), sampleLetter()))
}

// Badan surat berbahasa Indonesia bagi reasuradur Indonesia, dan meng-escape isian manusia.
func TestHTMLBodyFollowsTheReinsurerCountry(t *testing.T) {
	pla, _ := inboxpladlapredla.FindTab("pla")
	dla, _ := inboxpladlapredla.FindTab("dla")
	claim := inboxpladlapredla.ClaimSummary{
		ClaimNo: "PNC-1", PolicyNo: "POL-1", Insured: "PT <Satu> & Co", LossDate: "2026-01-02",
	}

	indonesian := notification.HTMLBody(pla, inboxpladlapredla.SendableAdvice{
		AdviceNo: "PLA/1", Reinsurer: "Reas A", Country: "INDONESIA"}, claim, 0)
	require.Contains(t, indonesian, "<p>Kepada Yth. Reas A,</p>")
	require.Contains(t, indonesian, "dokumen pendukung PLA")
	require.Contains(t, indonesian, "<b>No PLA</b></td><td>PLA/1</td>")
	require.Contains(t, indonesian, "PT &lt;Satu&gt; &amp; Co")
	require.Contains(t, indonesian, "Dokumen terlampir: 0 berkas.")

	english := notification.HTMLBody(dla, inboxpladlapredla.SendableAdvice{
		AdviceNo: "DLA/1", Reinsurer: "Reas <SG>", Country: "SINGAPORE"}, claim, 2)
	require.Contains(t, english, "<p>Dear Reas &lt;SG&gt;,</p>")
	require.Contains(t, english, "the DLA supporting documents")
	require.Contains(t, english, "<b>No DLA</b></td><td>DLA/1</td>")
	require.Contains(t, english, "Attached documents: 2 file(s).")
}
