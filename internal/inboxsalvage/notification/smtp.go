// Package notification memenuhi seam inboxsalvage.Notifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// Langkah ke-23 `Activity/SetStsSalvagePNC_act-Act.xml` — `Send Ke Email Hasil Datanya` —
// yang memanggil `UploadingFileUntukSendByEmail` dengan lima parameter: penerima, CC, BCC,
// subjek, dan badan surel.
//
// # Yang terbaca utuh dari export, dan yang TIDAK
//
//	Terbaca   subjek          `"Pengajuan Salvage an " + pyWorkPage.Policy.QQName` (`:10212`)
//	Terbaca   penerima        dua alamat tetap di `:1757-1760`
//	Terbaca   CC dan BCC      keduanya KOSONG (`:10200`, `:10215`)
//	HILANG    badan surel     rule HTML `SendEmailRejectedApprovetochecker` (`:10195`)
//
// Rule HTML-nya tidak ada di export sama sekali — ia bagian dari `R-16`. Badan surel di
// bawah karena itu DISUSUN, bukan disalin, dan selisihnya dinyatakan di
// inboxsalvage.PlannedDifferences. Isinya dibatasi pada keterangan yang memang ada di
// pengajuan: nomor klaim, jenis salvage, nilai minimum, nilai penawaran, lokasi, dan
// jumlah barang.
//
// # Penerima yang di-hardcode TIDAK dibawa
//
// `Activity/SetStsSalvagePNC_act-Act.xml:1757-1760` memasang dua alamat langsung di dalam
// activity, dan yang kedua adalah akun surel PRIBADI di jalur produksi. Pola itu persis
// yang `D-15` larang dibawa, dan `D-67` menegaskan tidak ada akun pribadi yang ikut ke
// sistem baru. Nilainya tidak direproduksi di berkas mana pun yang di-commit (`D-69`).
//
// Penggantinya `SMTP_PENERIMA_SALVAGE` — mailbox fungsional, bukan orang. Tempat yang benar
// baginya adalah master Penerima Notifikasi (`F-4`); sampai master itu ada, variabel
// lingkungan adalah tempat terdekat yang memenuhi `D-15`.
//
// # Isian "Email" pada form MENAMBAH, tidak menggantikan
//
// Lihat inboxsalvage.SubmissionNotice.ExtraRecipients.
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

	"claim-pnc/internal/inboxsalvage"
)

// Config adalah parameter sambungan SMTP.
type Config struct {
	Host string
	Port int

	// User dan Password boleh kosong. Banyak relay SMTP internal menerima pengirim dari
	// jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan menolak konfigurasi
	// yang sah.
	User     string
	Password string

	// From adalah alamat pengirim.
	From string

	// To adalah mailbox tetap penerima pemberitahuan pengajuan salvage.
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

// to mengembalikan alamat penerima tetap yang benar-benar terisi.
func (k Config) to() []string {
	return cleanAddresses(k.To)
}

// cleanAddresses membuang alamat kosong dan yang berulang.
//
// Pengulangan nyata di modul ini: isian "Email" pada form sering diisi alamat yang sudah ada
// di daftar tetap. Mengirimnya dua kali membuat penerima menerima surel ganda untuk satu
// pengajuan, dan itu terbaca seperti sistem yang mengirim berulang.
func cleanAddresses(list []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(list))
	for _, address := range list {
		trimmed := strings.TrimSpace(address)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, trimmed)
	}
	return result
}

const defaultTimeout = 20 * time.Second

// Sender mengirim pemberitahuan lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(k Config) *Sender { return &Sender{cfg: k} }

// NotifySalvageSubmitted mengirim satu surel pemberitahuan pengajuan salvage.
func (p *Sender) NotifySalvageSubmitted(
	ctx context.Context,
	notice inboxsalvage.SubmissionNotice,
) error {
	if !p.cfg.Complete() {
		return errors.New("inboxsalvage/notification: SMTP belum dikonfigurasi")
	}

	to := cleanAddresses(append(p.cfg.to(), notice.ExtraRecipients...))
	return p.send(ctx, to, composeEmail(p.cfg.From, to, notice))
}

func (p *Sender) send(ctx context.Context, to []string, message []byte) error {
	timeout := p.cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	address := net.JoinHostPort(p.cfg.Host, fmt.Sprint(p.cfg.Port))
	dialer := &net.Dialer{Timeout: timeout}

	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("inboxsalvage/notification: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungan, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung proses.
	_ = connection.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(connection, p.cfg.Host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("inboxsalvage/notification: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay internal
	// sering belum memasang sertifikat; yang dipaksakan justru sebaliknya — kredensial
	// hanya dikirim setelah sambungan terenkripsi.
	//
	// Rule lama menetapkan `UseSSL=false` pada SELURUH kemunculannya dan mengirimkan kata
	// sandi apa adanya (`R-17`). Itu tidak ditiru: menyalin kelemahan keamanan bukan
	// kesetaraan perilaku.
	secured := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(nil); err != nil {
			return fmt.Errorf("inboxsalvage/notification: menegakkan TLS: %w", err)
		}
		secured = true
	}

	if p.cfg.User != "" {
		if !secured {
			return errors.New(
				"inboxsalvage/notification: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		auth := smtp.PlainAuth("", p.cfg.User, p.cfg.Password, p.cfg.Host)
		if err := client.Auth(auth); err != nil {
			// Galatnya tidak memuat kredensial, dan tidak boleh memuatnya: galat ini
			// berakhir di log.
			return fmt.Errorf(
				"inboxsalvage/notification: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := client.Mail(p.cfg.From); err != nil {
		return fmt.Errorf(
			"inboxsalvage/notification: server menolak alamat pengirim: %w", err)
	}
	for _, address := range to {
		if err := client.Rcpt(address); err != nil {
			return fmt.Errorf(
				"inboxsalvage/notification: server menolak alamat tujuan: %w", err)
		}
	}

	write, err := client.Data()
	if err != nil {
		return fmt.Errorf("inboxsalvage/notification: membuka badan surel: %w", err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("inboxsalvage/notification: menulis badan surel: %w", err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("inboxsalvage/notification: menutup badan surel: %w", err)
	}
	return client.Quit()
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
//
// CC dan BCC TIDAK ditulis, mengikuti pemanggilan lamanya yang mengosongkan keduanya
// (`:10200`, `:10215`).
func composeEmail(from string, to []string, notice inboxsalvage.SubmissionNotice) []byte {
	var b strings.Builder

	clean := make([]string, 0, len(to))
	for _, address := range to {
		clean = append(clean, sanitizeHeader(address))
	}

	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + strings.Join(clean, ", ") + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(notice.Subject()) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(notice))

	return []byte(b.String())
}

// HTMLBody menyusun badan surel.
//
// # Ia DISUSUN, bukan disalin — dan itu selisih yang dinyatakan
//
// Rule HTML pemasoknya di sistem lama tidak ada di export (lihat banner paket). Bentuk di
// bawah mengikuti pola surel salvage lain yang MEMANG terbaca —
// `HTML/NotifikasiSalvageRequest-HTML.xml` — yaitu sapaan, satu kalimat pembuka, lalu
// tabel berlabel. Yang berbeda hanyalah barisnya, karena peristiwanya berbeda.
//
// # Seluruh nilai di-escape
//
// Lokasi, remark, dan nama tertanggung adalah teks yang diketik orang lain; menempelkannya
// mentah ke dalam HTML berarti isi basis data dapat menyuntikkan markup ke dalam kotak
// masuk penerimanya.
func HTMLBody(notice inboxsalvage.SubmissionNotice) string {
	row := func(label, value string) string {
		shown := strings.TrimSpace(value)
		if shown == "" {
			// Tanda hubung, bukan sel kosong: sel kosong terbaca seperti surel yang rusak,
			// sedangkan tanda hubung menyatakan datanya memang tidak ada.
			shown = "-"
		}
		return "<tr><td>" + html.EscapeString(label) + "</td><td>&nbsp;:&nbsp;</td><td>" +
			html.EscapeString(shown) + "</td></tr>"
	}

	// Nilai uang ditulis APA ADANYA, tanpa dibulatkan dan tanpa dipisah ribuan.
	//
	// `I-12` menetapkan nilai uang disimpan presisi penuh dan dibulatkan hanya saat
	// ditampilkan — dan surel bukan tempat memutuskan pembulatan itu. Mata uangnya
	// disebutkan di sebelahnya supaya angkanya tidak terbaca sebagai rupiah begitu saja.
	amount := func(value string) string {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return ""
		}
		if currency := strings.TrimSpace(notice.Currency); currency != "" {
			return currency + " " + trimmed
		}
		return trimmed
	}

	var b strings.Builder
	b.WriteString(`<html><body style="font-family: Arial, Helvetica, sans-serif;">`)
	b.WriteString(`<p>Dear,</p>`)
	b.WriteString(
		`<p>Dengan ini kami memberitahukan terdapat pengajuan salvage, ` +
			`dengan detail sebagai berikut :</p>`)
	b.WriteString(`<table>`)
	b.WriteString(row("No Klaim", notice.ClaimNo))
	b.WriteString(row("ID Salvage", notice.SalvageID))
	b.WriteString(row("Nama Tertanggung", notice.InsuredName))
	b.WriteString(row("Jenis Salvage", notice.SalvageType))
	b.WriteString(row("Lokasi Salvage", notice.Location))
	b.WriteString(row("Minimum Salvage", amount(notice.MinimumValue)))
	b.WriteString(row("Nilai Penawaran", amount(notice.OfferValue)))
	b.WriteString(row("Jumlah Item", itemCount(notice.ItemCount)))
	b.WriteString(row("Remark", notice.Remark))
	b.WriteString(row("Diajukan Oleh", notice.Submitter))
	b.WriteString(`</table>`)
	b.WriteString(`<p>Terima Kasih.</p>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

// itemCount menuliskan jumlah barang, dan membedakan NOL dari tidak diketahui.
//
// Pengajuan tanpa barang memang ada — grid Detail Item Salvage boleh kosong — sehingga
// "0" adalah jawaban yang benar, bukan data yang hilang. Membiarkannya jatuh ke tanda
// hubung akan menyamakan keduanya.
func itemCount(count int) string {
	return fmt.Sprintf("%d item", count)
}

// sanitizeHeader membuang karakter yang dapat menyuntikkan baris kepala surel baru.
//
// Tanpa ini, satu nilai yang memuat CR atau LF dapat menambahkan penerima tersembunyi atau
// mengganti subjeknya — dan pada modul ini nilai itu dapat berasal dari nama tertanggung
// dan dari isian "Email" yang diketik petugas.
func sanitizeHeader(value string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return strings.TrimSpace(replacer.Replace(value))
}

var _ inboxsalvage.Notifier = (*Sender)(nil)
