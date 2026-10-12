// Package notification memenuhi seam inboxrclpucl.ReminderNotifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// `Activity/ReminderPUCLPA-Act.xml` — langkah `Call ASMSendsEmailAttachments` yang dipicu
// tombol "Reminder PUCL" pada antrean Kelengkapan Dokumen.
//
// # Yang terbaca utuh dari export, dan yang TIDAK
//
//	Terbaca   penerima     `.ClaimData.EmailLOD` — `T_CLAIM_PNC.EMAIL_LOD` (`:1350`)
//	Terbaca   subjek       lihat inboxrclpucl.ReminderSubject (`:1415`)
//	Terbaca   akun surel   `"Admin-PNC"` (`:3629`)
//	HILANG    badan surel  rule HTML `SendReminderPUCLKeTertanggungPA` (`:3521`)
//
// Rule HTML-nya tidak ada di export sama sekali — folder `HTML/` memuat 42 berkas dan tidak
// satu pun bernama mirip itu. Ia bagian dari `R-16`. Badan surel di bawah karena itu
// DISUSUN, bukan disalin, dan selisihnya dinyatakan di inboxrclpucl.PlannedDifferences.
// Preseden yang sama sudah ditempuh modul Salvage untuk alasan yang persis sama.
//
// # Kredensial SMTP di export TIDAK dibawa
//
// Langkah pengirim di activity itu memasang host, nama pengguna, dan KATA SANDI langsung di
// dalam rule. Itu `R-17`, dan `D-40` masih terbuka. Tidak satu pun nilainya direproduksi di
// berkas yang di-commit (`D-69`); konfigurasinya dibaca dari `POOLDATA.M_EMAIL_SERVER_PNC`
// lewat `platform/emailserver` dengan akun `Admin-PNC` — jalur yang sudah dipakai modul
// lain.
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

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/platform/emailserver"
)

// Config adalah parameter sambungan SMTP.
type Config struct {
	// Account membaca akun server surel dari `POOLDATA.M_EMAIL_SERVER_PNC` setiap kali
	// surel dikirim — pengganti Email Account Pega. Bila terisi, Host, Port, User,
	// Password, dan From diambil dari akun itu.
	Account func(ctx context.Context) (emailserver.Account, error)

	Host string
	Port int

	// User dan Password boleh kosong. Banyak relay SMTP internal menerima pengirim dari
	// jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan menolak konfigurasi
	// yang sah.
	User     string
	Password string

	// From adalah alamat pengirim.
	From string

	// Timeout membatasi lama menunggu server SMTP. Nol berarti nilai baku.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi ini cukup untuk mengirim surel.
//
// Penerima TIDAK ikut diperiksa di sini, dan itu pembeda terhadap modul lain: penerimanya
// bukan konfigurasi melainkan alamat tertanggung yang dibaca per klaim.
func (k Config) Complete() bool {
	return k.Account != nil ||
		strings.TrimSpace(k.Host) != "" && k.Port > 0 && strings.TrimSpace(k.From) != ""
}

const defaultTimeout = 20 * time.Second

// Sender mengirim Reminder PUCL lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(k Config) *Sender { return &Sender{cfg: k} }

// withAccount mengembalikan pengirim berkonfigurasi akun terbaru, atau pengirim itu sendiri
// bila Account tidak dipasang.
func (p *Sender) withAccount(ctx context.Context) (*Sender, error) {
	if p.cfg.Account == nil {
		return p, nil
	}
	account, err := p.cfg.Account(ctx)
	if err != nil {
		return nil, fmt.Errorf("inboxrclpucl/notification: akun surel: %w", err)
	}
	c := p.cfg
	c.Host, c.Port = account.Host, account.Port
	c.User, c.Password, c.From = account.Address, account.Password, account.Address
	return &Sender{cfg: c}, nil
}

// SendReminder mengirim satu pengingat ke tertanggung sebuah klaim.
func (p *Sender) SendReminder(
	ctx context.Context,
	recipient inboxrclpucl.ReminderRecipient,
) error {
	to := sanitizeHeader(recipient.Email)
	if to == "" {
		// Pemanggil SEHARUSNYA sudah menyaringnya. Diperiksa lagi di sini karena mengirim
		// ke alamat kosong bukan galat yang terbaca di log mana pun — ia hanya menjadi
		// percakapan SMTP yang ditolak tanpa sebab yang jelas.
		return errors.New("inboxrclpucl/notification: alamat tertanggung kosong")
	}

	p, err := p.withAccount(ctx)
	if err != nil {
		return err
	}
	if !p.cfg.Complete() {
		return errors.New("inboxrclpucl/notification: SMTP belum dikonfigurasi")
	}

	return p.send(ctx, []string{to}, composeEmail(p.cfg.From, to, recipient))
}

func (p *Sender) send(ctx context.Context, to []string, message []byte) error {
	timeout := p.cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	address := net.JoinHostPort(p.cfg.Host, fmt.Sprint(p.cfg.Port))
	pemutar := &net.Dialer{Timeout: timeout}

	sambungan, err := pemutar.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("inboxrclpucl/notification: menghubungi server surel: %w: %w", inboxrclpucl.ErrReminderUnreachable, err)
	}
	// Tenggang dipasang pada sambungan, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung proses.
	_ = sambungan.SetDeadline(time.Now().Add(timeout))

	klien, err := smtp.NewClient(sambungan, p.cfg.Host)
	if err != nil {
		_ = sambungan.Close()
		return fmt.Errorf("inboxrclpucl/notification: memulai percakapan SMTP: %w: %w", inboxrclpucl.ErrReminderUnreachable, err)
	}
	defer func() { _ = klien.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay internal
	// sering belum memasang sertifikat; yang dipaksakan justru sebaliknya — kredensial
	// hanya dikirim setelah sambungan terenkripsi.
	adaTLS := false
	if ok, _ := klien.Extension("STARTTLS"); ok {
		if err := klien.StartTLS(&tls.Config{ServerName: p.cfg.Host}); err != nil {
			return fmt.Errorf("inboxrclpucl/notification: menegakkan TLS: %w: %w", inboxrclpucl.ErrReminderRejected, err)
		}
		adaTLS = true
	}

	if p.cfg.User != "" {
		if !adaTLS {
			return errors.New(
				"inboxrclpucl/notification: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		auth := smtp.PlainAuth("", p.cfg.User, p.cfg.Password, p.cfg.Host)
		if err := klien.Auth(auth); err != nil {
			return fmt.Errorf("inboxrclpucl/notification: autentikasi SMTP ditolak: %w: %w", inboxrclpucl.ErrReminderRejected, err)
		}
	}

	if err := klien.Mail(p.cfg.From); err != nil {
		return fmt.Errorf("inboxrclpucl/notification: server menolak alamat pengirim: %w: %w", inboxrclpucl.ErrReminderRejected, err)
	}
	for _, alamat := range to {
		if err := klien.Rcpt(alamat); err != nil {
			return fmt.Errorf("inboxrclpucl/notification: server menolak alamat tujuan: %w: %w", inboxrclpucl.ErrReminderRejected, err)
		}
	}

	write, err := klien.Data()
	if err != nil {
		return fmt.Errorf("inboxrclpucl/notification: membuka badan surel: %w: %w", inboxrclpucl.ErrReminderRejected, err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("inboxrclpucl/notification: menulis badan surel: %w: %w", inboxrclpucl.ErrReminderRejected, err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("inboxrclpucl/notification: menutup badan surel: %w: %w", inboxrclpucl.ErrReminderRejected, err)
	}
	return klien.Quit()
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
func composeEmail(from, to string, recipient inboxrclpucl.ReminderRecipient) []byte {
	var b strings.Builder

	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(Subject(recipient)) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(recipient))

	return []byte(b.String())
}

// Subject adalah baris subjek, mengikuti Pega apa adanya — termasuk salah ketiknya.
// Lihat inboxrclpucl.ReminderSubject.
func Subject(recipient inboxrclpucl.ReminderRecipient) string {
	return inboxrclpucl.ReminderSubject(recipient.InsuredName)
}

// HTMLBody menyusun badan surel.
//
// # Isinya DISUSUN, bukan disalin
//
// Rule HTML aslinya (`SendReminderPUCLKeTertanggungPA`) tidak ada di export (`R-16`).
// Isinya karena itu dibatasi pada keterangan yang memang dimiliki antrean ini — nomor klaim
// dan nama tertanggung. Menambahkan keterangan lain berarti mengarang isi surat yang dibaca
// pihak LUAR.
//
// Seluruh nilai yang berasal dari data di-escape: nama tertanggung adalah teks yang
// dimasukkan orang lain, dan menempelkannya mentah ke dalam HTML yang dikirim keluar adalah
// celah penyusupan isi.
func HTMLBody(recipient inboxrclpucl.ReminderRecipient) string {
	var b strings.Builder

	b.WriteString(`<!doctype html><html lang="id">`)
	b.WriteString(`<body style="font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#1f2937">`)
	b.WriteString(`<p>Yth. ` + html.EscapeString(recipient.InsuredName) + `,</p>`)
	b.WriteString(`<p>Bersama ini kami mengingatkan bahwa dokumen untuk klaim berikut masih kami tunggu kelengkapannya.</p>`)
	b.WriteString(`<p><strong>Nomor Klaim:</strong> ` + html.EscapeString(recipient.ClaimNumber) + `</p>`)
	b.WriteString(`<p>Mohon kelengkapan dokumen dapat segera disampaikan agar proses klaim dapat kami lanjutkan.</p>`)
	b.WriteString(`<p>Terima kasih atas perhatian Anda.</p>`)
	b.WriteString(`<p>Hormat kami,<br>Divisi Klaim<br>PT Asuransi Sinar Mas</p>`)
	b.WriteString(`<p style="color:#6b7280;font-size:12px">Surel ini dikirim otomatis oleh sistem. Mohon tidak membalas surel ini.</p>`)
	b.WriteString(`</body></html>`)

	return b.String()
}

// sanitizeHeader memotong nilai pada baris baru pertama.
//
// Menempelkan nilai mentah ke dalam header surel memungkinkan header tambahan disusupkan —
// termasuk penerima tambahan. Yang sah selalu berada sebelum baris baru pertama.
func sanitizeHeader(value string) string {
	if i := strings.IndexAny(value, "\r\n"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

var _ inboxrclpucl.ReminderNotifier = (*Sender)(nil)
