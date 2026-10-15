// Package notification memenuhi seam inboxmanager.PaymentNotifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// Langkah terakhir `Activity/SaveApprovalAkseptasiPaymentLeader_Act` — pemanggilan
// `UploadingFileUntukSendByEmail` setelah SIMPAN pada Payment Klaim Akseptasi.
//
// # Yang terbaca utuh dari export, dan yang TIDAK
//
//	Terbaca   subjek     "Pengecekan akseptasi CLAIM <no klaim> | <tertanggung> | No Aksep <no>" (:2806)
//	Terbaca   penerima   `UserTeknisEmail` — POOLDATA.MST_USER_TEKNIK.EMAIL milik PIC
//	Terbaca   CC         KOSONG: parameternya `Local.bcemail`, yang tidak pernah diisi
//	HILANG    badan      rule HTML `EmailAkseptasiPaymentLeader` tidak ada di export
//
// Badan surel karena itu DISUSUN, bukan disalin — preseden yang sama dengan Salvage dan
// Master Rekening — dan selisihnya dinyatakan di inboxmanager.PlannedDifferences. Isinya
// dibatasi pada keterangan yang memang ada di SIMPAN.
//
// # Akun pengirim
//
// Pega mengirim lewat akun "ASM"/"GCNM_ASI" milik `UploadingFileUntukSendByEmail`. Di aplikasi
// ini seluruh pengganti activity itu memakai akun bawaan POOLDATA.M_EMAIL_SERVER_PNC
// (Work Owner 2026-10-10).
package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/platform/emailserver"
)

// Config adalah parameter sambungan SMTP.
type Config struct {
	// Account membaca akun server surel dari POOLDATA.M_EMAIL_SERVER_PNC setiap kali surel
	// dikirim. Bila terisi, Host, Port, User, Password, dan From diambil dari akun itu.
	// Nil: isian di bawah yang dipakai (mode tanpa Oracle).
	Account func(ctx context.Context) (emailserver.Account, error)

	Host string
	Port int

	// User dan Password boleh kosong — relay internal sering menerima pengirim dari
	// jaringan tepercaya tanpa autentikasi.
	User     string
	Password string

	From string

	// Timeout membatasi lama menunggu server SMTP. Nol berarti nilai baku.
	Timeout time.Duration

	// SkipTLSVerify mematikan pemeriksaan sertifikat server saat STARTTLS —
	// SMTP_TLS_LEWATI_VERIFIKASI.
	SkipTLSVerify bool
}

// Complete menyatakan konfigurasi ini cukup untuk mengirim surel. Penerimanya tidak ikut
// diperiksa: ia alamat PIC yang dibaca per SIMPAN, bukan konfigurasi.
func (c Config) Complete() bool {
	return c.Account != nil ||
		strings.TrimSpace(c.Host) != "" && c.Port > 0 && strings.TrimSpace(c.From) != ""
}

const defaultTimeout = 20 * time.Second

// Sender mengirim surel SIMPAN lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(c Config) *Sender { return &Sender{cfg: c} }

// ErrNoRecipient berarti PIC tidak punya alamat surel di MST_USER_TEKNIK.
var ErrNoRecipient = errors.New("inboxmanager/notification: PIC tidak punya alamat surel")

func (s *Sender) withAccount(ctx context.Context) (*Sender, error) {
	if s.cfg.Account == nil {
		return s, nil
	}
	account, err := s.cfg.Account(ctx)
	if err != nil {
		return nil, fmt.Errorf("inboxmanager/notification: akun surel: %w", err)
	}
	c := s.cfg
	c.Host, c.Port = account.Host, account.Port
	c.User, c.Password, c.From = account.Address, account.Password, account.Address
	return &Sender{cfg: c}, nil
}

// NotifyPaymentChecked mengirim satu surel ke PIC.
func (s *Sender) NotifyPaymentChecked(ctx context.Context, notice inboxmanager.PaymentNotice) error {
	to := sanitizeHeader(notice.To)
	if to == "" {
		return ErrNoRecipient
	}
	s, err := s.withAccount(ctx)
	if err != nil {
		return err
	}
	if !s.cfg.Complete() || strings.TrimSpace(s.cfg.From) == "" {
		return errors.New("inboxmanager/notification: SMTP belum dikonfigurasi")
	}
	return s.send(ctx, to, composeEmail(s.cfg.From, to, notice))
}

func (s *Sender) send(ctx context.Context, to string, message []byte) error {
	timeout := s.cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port)))
	if err != nil {
		return fmt.Errorf("inboxmanager/notification: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungan: server yang menerima koneksi lalu diam adalah
	// kegagalan yang paling sering menggantung proses.
	_ = conn.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("inboxmanager/notification: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS dipakai bila ditawarkan; kredensial hanya dikirim setelah terenkripsi.
	secured := false
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, InsecureSkipVerify: s.cfg.SkipTLSVerify} //nolint:gosec // sakelar SMTP_TLS_LEWATI_VERIFIKASI, mati secara baku
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("inboxmanager/notification: menegakkan TLS: %w", err)
		}
		secured = true
	}

	if s.cfg.User != "" {
		if !secured {
			return errors.New("inboxmanager/notification: server surel tidak mendukung " +
				"STARTTLS; kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		if err := client.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("inboxmanager/notification: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("inboxmanager/notification: server menolak alamat pengirim: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("inboxmanager/notification: server menolak alamat tujuan: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("inboxmanager/notification: membuka badan surel: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("inboxmanager/notification: menulis badan surel: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("inboxmanager/notification: menutup badan surel: %w", err)
	}
	return client.Quit()
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML. Dipisah dari pengiriman
// supaya isinya dapat diuji tanpa server SMTP.
func composeEmail(from, to string, n inboxmanager.PaymentNotice) []byte {
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(n.Subject()) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(n))
	return []byte(b.String())
}

// HTMLBody menyusun badan surel. Seluruh nilai dari data di-escape — catatan atasan adalah
// teks ketikan orang, dan menempelkannya mentah berarti ia dapat menyuntikkan markup.
func HTMLBody(n inboxmanager.PaymentNotice) string {
	row := func(label, value string) string {
		if strings.TrimSpace(value) == "" {
			value = "—"
		}
		return "<tr><td style=\"padding:4px 12px 4px 0;color:#555\">" + html.EscapeString(label) +
			"</td><td style=\"padding:4px 0\"><b>" + html.EscapeString(value) + "</b></td></tr>"
	}

	var b strings.Builder
	b.WriteString("<html><body style=\"font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#222\">")
	b.WriteString("<p>Pengecekan akseptasi pembayaran klaim berikut telah disimpan oleh atasan.</p>")
	b.WriteString("<table style=\"border-collapse:collapse\">")
	b.WriteString(row("No Klaim", n.ClaimNo))
	b.WriteString(row("Nama Tertanggung", n.InsuredName))
	b.WriteString(row("No Akseptasi", n.AcceptedNo))
	b.WriteString(row("Status Approval", inboxmanager.PaymentStatusLabel(n.Status)))
	b.WriteString(row("Remark Approval", n.Remark))
	b.WriteString(row("Disimpan oleh", n.DecidedBy))
	b.WriteString("</table>")
	b.WriteString("<p style=\"color:#888;font-size:12px\">Surel ini dikirim otomatis oleh " +
		"aplikasi Claim PNC. Mohon tidak membalas surel ini.<br>" +
		"Susunan surel ini <b>sementara</b>: rule HTML aslinya (EmailAkseptasiPaymentLeader) " +
		"tidak ikut ter-export, sehingga isinya disusun ulang dan menunggu wording resmi.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

// sanitizeHeader memotong nilai header pada CR atau LF pertama, supaya tidak ada header
// tambahan — termasuk penerima tambahan — yang dapat disusupkan lewat data.
func sanitizeHeader(value string) string {
	if i := strings.IndexAny(value, "\r\n"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

var _ inboxmanager.PaymentNotifier = (*Sender)(nil)
