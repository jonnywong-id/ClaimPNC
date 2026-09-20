// Package notification memenuhi seam masterxol.Notifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// `Activity/SendDataMasterXOLToKomites-Act.xml`, yang dipanggil setiap kali induk XOL
// disimpan. Alurnya di sana:
//
//  1. Menyusun subjek "Notif Pemberitahuan Pengajuan Master XOL Tahun - {tahun} dengan
//     ID - {idmaster}".
//  2. Mencari alamat komite lewat `GetDataMasterXOLForKomiteApprove`, yang membaca
//     `(select email from POOLDATA.MST_USER_TEKNIK where operator_id = PIC)`.
//  3. **Menimpa** hasil pencarian itu dengan satu alamat perorangan.
//  4. **Menimpanya sekali lagi** dengan alamat perorangan yang berbeda ketika
//     `pxRequestor.pxReqServer` sama dengan nama host dev.
//  5. Mengirim lewat `UploadingFileUntukSendByEmail`.
//
// # Yang sengaja TIDAK ditiru, dan kenapa
//
// **Langkah 3 dan 4 tidak dibawa.** Keduanya adalah pola penimpaan penerima yang `D-15`
// larang dibawa ke sistem baru — bentuknya persis sama dengan blok `// TESTING` di
// `InputRegister_act` yang menimpa surel Underwriting dengan alamat penguji lalu tetap
// berada di jalur produksi. `D-67` menegaskan tidak ada akun pribadi yang dibawa.
//
// **Percabangan berdasarkan nama host tidak dibawa.** `docs/Steering/12-CROSSCUTTING.md`
// §3.4 melarangnya tanpa perkecualian: perilaku yang bergantung pada nama server tidak
// dapat diuji, tidak terlihat saat membaca kode, dan berubah diam-diam ketika servernya
// dipindahkan. Perbedaan antarlingkungan dinyatakan sebagai konfigurasi.
//
// Penerimanya karena itu datang dari konfigurasi. Itu tempat sementara: `D-67` menetapkan
// penerima notifikasi berasal dari master **Penerima Notifikasi** (`F-4`), dan master itu
// belum dibangun. Dicatat terbuka di docs/keputusan-implementasi.md.
//
// **Badan suratnya disusun di sini, bukan disalin.** Rule korespondensi yang dipakai
// `UploadingFileUntukSendByEmail` tidak ada di dalam export — tidak ada satu pun direktori
// `Correspondence/` (`R-16`). Isinya tidak dapat direproduksi.
package notification

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/masterxol"
)

// Config adalah parameter sambungan SMTP beserta penerimanya.
type Config struct {
	Host string
	Port int

	// User dan Password boleh kosong. Banyak relay SMTP internal menerima pengirim dari
	// jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan menolak
	// konfigurasi yang sah.
	User     string
	Password string

	// From adalah alamat pengirim.
	From string

	// To adalah mailbox komite yang menerima pemberitahuan pengajuan.
	//
	// Ia KONFIGURASI, bukan nilai di dalam kode — itulah pokok perbedaannya dari sistem
	// lama, yang menuliskan dua alamat perorangan langsung di dalam activity-nya.
	To []string

	// Timeout membatasi lama menunggu server SMTP. Nol berarti nilai baku.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi ini cukup untuk mengirim surel.
//
// Penerima ikut disyaratkan: pengirim surel tanpa tujuan bukan setengah lengkap, ia tidak
// lengkap — dan menyatakannya lengkap akan menyembunyikan konfigurasi yang belum selesai
// di balik pengiriman yang tidak pernah sampai ke siapa pun.
func (c Config) Complete() bool {
	return strings.TrimSpace(c.Host) != "" &&
		c.Port > 0 &&
		strings.TrimSpace(c.From) != "" &&
		len(c.to()) > 0
}

// to mengembalikan alamat penerima yang benar-benar terisi.
func (c Config) to() []string {
	result := make([]string, 0, len(c.To))
	for _, address := range c.To {
		if trimmed := strings.TrimSpace(address); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

const defaultTimeout = 20 * time.Second

// Sender mengirim pemberitahuan lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(c Config) *Sender { return &Sender{cfg: c} }

// NotifyCommitteeSubmission mengirim satu surel pemberitahuan ke mailbox komite.
func (s *Sender) NotifyCommitteeSubmission(ctx context.Context, submission masterxol.CommitteeSubmission) error {
	if !s.cfg.Complete() {
		return errors.New("masterxol/notification: SMTP belum dikonfigurasi")
	}

	to := s.cfg.to()
	return s.send(ctx, to, composeEmail(s.cfg.From, to, submission))
}

func (s *Sender) send(ctx context.Context, to []string, message []byte) error {
	timeout := s.cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	address := net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port))
	dialer := &net.Dialer{Timeout: timeout}

	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("masterxol/notification: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungannya, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung proses.
	_ = connection.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(connection, s.cfg.Host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("masterxol/notification: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay
	// internal sering belum memasang sertifikat; yang dipaksakan justru sebaliknya —
	// kredensial hanya dikirim setelah sambungan terenkripsi.
	//
	// Ini menutup satu temuan nyata: export memuat `UseSSL=false` pada SELURUH
	// kemunculannya, sementara 16 dari 31 lokasi memakai port 587 (`R-17`).
	secured := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(nil); err != nil {
			return fmt.Errorf("masterxol/notification: menegakkan TLS: %w", err)
		}
		secured = true
	}

	if s.cfg.User != "" {
		if !secured {
			return errors.New(
				"masterxol/notification: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		if err := client.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
			// Galatnya tidak memuat kredensial, dan tidak boleh memuatnya: galat ini
			// berakhir di log.
			return fmt.Errorf("masterxol/notification: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("masterxol/notification: server menolak alamat pengirim: %w", err)
	}
	for _, address := range to {
		if err := client.Rcpt(address); err != nil {
			return fmt.Errorf("masterxol/notification: server menolak alamat tujuan: %w", err)
		}
	}

	write, err := client.Data()
	if err != nil {
		return fmt.Errorf("masterxol/notification: membuka badan surel: %w", err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("masterxol/notification: menulis badan surel: %w", err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("masterxol/notification: menutup badan surel: %w", err)
	}
	return client.Quit()
}

// Subject menyusun baris subjek.
//
// Kalimatnya DISALIN dari `Activity/SendDataMasterXOLToKomites-Act.xml`, yang menyusun
// `"Notif Pemberitahuan Pengajuan Master XOL Tahun - " + Param.Tahun + " dengan ID - " +
// Param.idmaster`. `D-13` menetapkan teks yang dilihat pengguna mengikuti sistem lama apa
// adanya, dan subjek surel adalah teks yang dilihat pengguna.
func Subject(submission masterxol.CommitteeSubmission) string {
	return "Notif Pemberitahuan Pengajuan Master XOL Tahun - " + submission.Year +
		" dengan ID - " + submission.MasterID
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
func composeEmail(from string, to []string, submission masterxol.CommitteeSubmission) []byte {
	var b strings.Builder

	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(strings.Join(to, ", ")) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(Subject(submission)) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(submission))
	return []byte(b.String())
}

// HTMLBody menyusun badan surat.
//
// Seluruh nilai di-escape: isinya memuat catatan yang diketik pengguna, dan catatan itu
// dapat memuat tanda kurung siku.
func HTMLBody(submission masterxol.CommitteeSubmission) string {
	rate := strconv.FormatInt(int64(submission.ExchangeRate), 10)

	var b strings.Builder
	b.WriteString(`<html><body style="font-family:sans-serif;font-size:14px;color:#0f172a">`)
	b.WriteString(`<p>Sebuah Master XOL diajukan untuk mendapat persetujuan komite.</p>`)
	b.WriteString(`<table cellpadding="6" style="border-collapse:collapse">`)
	row := func(label, value string) {
		b.WriteString(`<tr><td style="color:#475569">` + html.EscapeString(label) + `</td>`)
		b.WriteString(`<td><strong>` + html.EscapeString(value) + `</strong></td></tr>`)
	}
	row("ID Master XOL", submission.MasterID)
	row("Tahun", submission.Year)
	row("Kurs IDR", rate)
	row("Diajukan oleh", submission.SubmittedBy)
	row("Remark PIC", submission.Remark)
	b.WriteString(`</table>`)
	b.WriteString(`<p style="color:#64748b;font-size:12px">` +
		`Pesan ini dikirim otomatis oleh aplikasi Claim PNC. Mohon tidak membalas surel ini.</p>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

// sanitizeHeader membuang pemisah baris dari nilai header.
//
// Tanpa ini, satu baris baru di dalam catatan pengguna cukup untuk menyisipkan header
// surel tambahan — termasuk penerima tambahan yang tidak dikehendaki siapa pun.
func sanitizeHeader(value string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return strings.TrimSpace(replacer.Replace(value))
}

var _ masterxol.Notifier = (*Sender)(nil)
