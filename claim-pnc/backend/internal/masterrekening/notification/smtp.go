// Package notification memenuhi seam masterrekening.Notifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// Activity SendEmailAlertRekening pada sistem lama. Alurnya:
//
//  1. Membaca status transfer terakhir dari POOLDATA.CLAIM_SERVICE_LOG
//     (QueryCekStatusTransferKasir), mengambil ResponseCode dan ResponseMessage
//     dari kolom JSONOUT.
//  2. Bila ResponseCode == "9", mencari alamat surel dengan
//     `select email from pooldata.mst_user_teknik where operator_id = {TempIns.pyID}`,
//     yang diisi OperatorID.pxInsName — operator yang sedang masuk.
//  3. Menyusun badan surel dari rule HTML EmailAlertRekeningToPIC.
//  4. Mengirim lewat ASMSendsEmailAttachments.
//
// # Dua hal yang sengaja berbeda, dan alasannya
//
// **Langkah 1 tidak ditiru.** Rule lama menulis hasil panggilan Kasir ke tabel log,
// lalu MEMBACANYA KEMBALI untuk mengetahui kode responsnya — dengan `rownum=1 order by
// insertdate desc`, yang pada Oracle memotong baris SEBELUM diurutkan dan karena itu
// tidak menjamin baris terbaru. Di sini kode respons sudah ada di tangan sebagai nilai
// balik panggilan Kasir, sehingga tidak ada yang perlu dibaca ulang dan tidak ada
// kemungkinan membaca baris yang salah.
//
// **Badan suratnya disusun di sini, bukan disalin.** Rule HTML
// `EmailAlertRekeningToPIC` TIDAK ADA di dalam export — hanya pemanggilannya yang
// terlihat. Isinya karena itu tidak dapat direproduksi dan disusun ulang dari informasi
// yang tersedia pada peringatan. Dicatat sebagai gap export di
// docs/keputusan-implementasi.md.
package notification

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strings"
	"time"

	"claim-pnc/internal/masterrekening"
)

// Config adalah parameter sambungan SMTP.
type Config struct {
	Host string
	Port int

	// User dan KataSandi boleh kosong. Banyak relay SMTP internal menerima
	// pengirim dari jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan
	// menolak konfigurasi yang sah.
	User     string
	Password string

	// From adalah alamat pengirim.
	From string

	// To adalah mailbox Tim IT yang menerima peringatan kegagalan integrasi.
	//
	// Ia konfigurasi, bukan diturunkan dari data: peringatan ini ditujukan ke pihak
	// yang dapat MEMPERBAIKI kegagalan integrasi, dan itu bukan orang yang kebetulan
	// menekan tombol approve (keputusan Work Owner 2026-09-17).
	To []string

	// Timeout membatasi lama menunggu server SMTP. Nol berarti nilai baku.
	Timeout time.Duration
}

// Complete menyatakan konfigurasi ini cukup untuk mengirim surel.
func (k Config) Complete() bool {
	return strings.TrimSpace(k.Host) != "" &&
		k.Port > 0 &&
		strings.TrimSpace(k.From) != "" &&
		len(k.to()) > 0
}

// to mengembalikan alamat penerima yang benar-benar terisi.
func (k Config) to() []string {
	result := make([]string, 0, len(k.To))
	for _, address := range k.To {
		if trimmed := strings.TrimSpace(address); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

const defaultTimeout = 20 * time.Second

// Sender mengirim peringatan lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(k Config) *Sender { return &Sender{cfg: k} }

// WarnCashierFailure mengirim satu surel peringatan ke mailbox Tim IT.
func (p *Sender) WarnCashierFailure(ctx context.Context, per masterrekening.Alert) error {
	if !p.cfg.Complete() {
		return errors.New("masterrekening/notification: SMTP belum dikonfigurasi")
	}

	to := p.cfg.to()
	message := composeEmail(p.cfg.From, to, per)
	return p.send(ctx, to, message)
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
		return fmt.Errorf("masterrekening/notification: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungan, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung
	// proses.
	_ = sambungan.SetDeadline(time.Now().Add(timeout))

	klien, err := smtp.NewClient(sambungan, p.cfg.Host)
	if err != nil {
		_ = sambungan.Close()
		return fmt.Errorf("masterrekening/notification: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = klien.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay
	// internal sering belum memasang sertifikat; yang dipaksakan justru sebaliknya —
	// kredensial hanya dikirim setelah sambungan terenkripsi.
	adaTLS := false
	if ok, _ := klien.Extension("STARTTLS"); ok {
		if err := klien.StartTLS(nil); err != nil {
			return fmt.Errorf("masterrekening/notification: menegakkan TLS: %w", err)
		}
		adaTLS = true
	}

	if p.cfg.User != "" {
		if !adaTLS {
			return errors.New(
				"masterrekening/notification: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		auth := smtp.PlainAuth("", p.cfg.User, p.cfg.Password, p.cfg.Host)
		if err := klien.Auth(auth); err != nil {
			// Galatnya tidak memuat kredensial, dan tidak boleh memuatnya: galat ini
			// berakhir di log.
			return fmt.Errorf("masterrekening/notification: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := klien.Mail(p.cfg.From); err != nil {
		return fmt.Errorf("masterrekening/notification: server menolak alamat pengirim: %w", err)
	}
	for _, address := range to {
		if err := klien.Rcpt(address); err != nil {
			return fmt.Errorf("masterrekening/notification: server menolak alamat tujuan: %w", err)
		}
	}

	write, err := klien.Data()
	if err != nil {
		return fmt.Errorf("masterrekening/notification: membuka badan surel: %w", err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("masterrekening/notification: menulis badan surel: %w", err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("masterrekening/notification: menutup badan surel: %w", err)
	}
	return klien.Quit()
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
func composeEmail(from string, to []string, p masterrekening.Alert) []byte {
	var b strings.Builder

	bersih := make([]string, 0, len(to))
	for _, address := range to {
		bersih = append(bersih, sanitizeHeader(address))
	}

	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + strings.Join(bersih, ", ") + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(Subject(p)) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(p))

	return []byte(b.String())
}

// Subject adalah baris subjek surel peringatan.
func Subject(p masterrekening.Alert) string {
	return "Pendaftaran rekening ke Kasir GAGAL — " + p.Account.Number
}

// HTMLBody menyusun badan surel.
//
// Seluruh nilai yang berasal dari data di-escape. Nama pemilik rekening dan pesan dari
// Kasir adalah teks yang dimasukkan orang lain; menempelkannya mentah ke dalam HTML
// berarti isi basis data dapat menyuntikkan markup ke dalam kotak masuk penerimanya.
func HTMLBody(p masterrekening.Alert) string {
	r := p.Account

	rows := func(label, value string) string {
		if strings.TrimSpace(value) == "" {
			value = "—"
		}
		return "<tr><td style=\"padding:4px 12px 4px 0;color:#555\">" + html.EscapeString(label) +
			"</td><td style=\"padding:4px 0\"><b>" + html.EscapeString(value) + "</b></td></tr>"
	}

	decidedBy := strings.TrimSpace(p.DecidedBy.Name)
	if surel := strings.TrimSpace(p.DecidedBy.Email); surel != "" {
		if decidedBy == "" {
			decidedBy = surel
		} else {
			decidedBy += " (" + surel + ")"
		}
	}

	var b strings.Builder
	b.WriteString("<html><body style=\"font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#222\">")
	b.WriteString("<p>Yth. Tim IT,</p>")
	b.WriteString("<p>Rekening berikut telah <b>disetujui komite</b>, tetapi <b>gagal didaftarkan " +
		"ke sistem Kasir</b>. Selama pendaftaran belum berhasil, rekening ini <b>tidak dapat " +
		"dipakai membayar klaim</b>.</p>")
	b.WriteString("<table style=\"border-collapse:collapse\">")
	b.WriteString(rows("Nomor rekening", r.Number))
	b.WriteString(rows("Nama pemilik", r.OwnerName))
	b.WriteString(rows("Bank", r.BankName))
	b.WriteString(rows("Cabang", r.BankBranch))
	b.WriteString(rows("Tipe rekening", r.AccountType))
	b.WriteString(rows("Kode respons Kasir", p.Code))
	b.WriteString(rows("Pesan dari Kasir", p.Message))
	b.WriteString(rows("Diputuskan oleh", decidedBy))
	b.WriteString("</table>")
	b.WriteString("<p>Mohon ditindaklanjuti. Setelah masalahnya selesai, rekening dapat " +
		"didaftarkan ulang dari layar Master Rekening.</p>")
	b.WriteString("<p>Terima kasih.</p>")
	b.WriteString("<p style=\"color:#888;font-size:12px\">Surel ini dikirim otomatis oleh " +
		"aplikasi Claim PNC. Mohon tidak membalas surel ini.<br>" +
		"Susunan surel ini <b>sementara</b>: rule HTML aslinya (EmailAlertRekeningToPIC) " +
		"tidak ikut ter-export, sehingga isinya disusun ulang dan menunggu wording resmi.</p>")
	b.WriteString("</body></html>")

	return b.String()
}

// sanitizeHeader memotong nilai header pada CR atau LF yang pertama.
//
// Tanpa ini, satu baris baru di dalam nama atau alamat cukup untuk menyisipkan header
// tambahan — termasuk penerima tambahan — ke dalam surel yang dikirim aplikasi.
//
// Ia MEMOTONG, bukan mengganti baris baru dengan spasi. Mengganti dengan spasi memang
// sudah menghalangi terbentuknya header baru, tetapi teks yang disusupkan tetap ikut
// terkirim di dalam header — dan sesuatu yang memuat baris baru di dalam alamat surel
// sudah cacat sejak awal. Yang sah selalu berada sebelum baris baru pertama.
func sanitizeHeader(value string) string {
	if i := strings.IndexAny(value, "\r\n"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

var _ masterrekening.Notifier = (*Sender)(nil)
