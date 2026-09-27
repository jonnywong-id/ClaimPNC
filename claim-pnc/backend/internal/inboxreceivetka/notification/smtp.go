// Package notification memenuhi seam inboxreceivetka.Notifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// Langkah ke-7 dan ke-8 `Activity/SubmitTanggalLengkapTKA-Act.xml`: `Property-Set-HTML`
// menyusun badan surel dari rule `NotificationKelengkapanTKA`, lalu
// `Call SendEmailNotification` mengirimkannya.
//
// Berbeda dari kebanyakan gap export, **kedua rule itu ADA dan terbaca utuh**:
//
//	HTML/NotificationKelengkapanTKA-HTML.xml   badan surel, enam baris tabel
//	Activity/SendEmailNotification_act.xml     pengirimnya
//
// Yang kedua sempat dikira hilang. Berkas `Activity/SendEmailNotification-Act.xml` — dengan
// akhiran `-Act` seperti berkas lain — ternyata berisi rule yang BERBEDA (`CompressImage_Act`,
// ruleset GISFW), yaitu cacat export yang `D-39` catat. Rule yang benar ada pada berkas
// berakhiran `_act`, dan isinya **activity bawaan Pega** (`Pega-IntegrationEngine 08-03-01`,
// kelas `@baseclass`) dengan satu langkah Java pengirim SMTP. Tidak ada logika bisnis di
// dalamnya yang perlu direkayasa-balik.
//
// # Tiga hal dari pemanggilan lamanya yang TIDAK dibawa
//
//  1. **Penerima yang ditentukan siapa yang login.** Activity lama bercabang pada tiga
//     Operator ID yang tertanam di dalam rule, masing-masing memilih daftar penerima
//     berbeda — salah satunya akun Gmail pribadi di jalur produksi. Pola "nama orang menjadi
//     syarat" sudah dicabut sekali pada `D-52`, dan dicabut lagi di sini (`D-15`, `D-67`).
//     Penerima datang dari `SMTP_PENERIMA_TKA`.
//
//  2. **Kata sandi dan alamat server yang tertanam di dalam rule**, bersama `UseSSL=false`
//     (`R-17`). Nilainya tidak direproduksi di berkas mana pun yang di-commit (`D-69`).
//
//  3. **Pengiriman di dalam transaksi basis data.** Lihat catatan pada usecase.Complete.
//
// # Apa yang DIBAWA apa adanya
//
// Subjek, alamat pengirim, keenam baris badan surel beserta labelnya, dan urutannya —
// seluruhnya dari rule aslinya, supaya penerima yang sudah terbiasa tidak menghadapi surel
// yang terasa datang dari sistem lain (`D-13`).
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

	"claim-pnc/internal/inboxreceivetka"
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
	//
	// Di sistem lama nilainya tertanam di dalam rule sebagai alamat otomatis milik
	// perusahaan. Ia dipindahkan ke konfigurasi, bukan karena rahasia, melainkan karena
	// alamat pengirim berbeda antarlingkungan — dan surel pengujian yang mengaku berasal
	// dari alamat produksi adalah cara tercepat membuat penerima berhenti mempercayainya.
	From string

	// To adalah mailbox penerima pemberitahuan kelengkapan dokumen TKA.
	//
	// Ia konfigurasi, bukan diturunkan dari data dan bukan dari identitas penekan tombol.
	// Lihat banner paket.
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

// Sender mengirim pemberitahuan lewat SMTP.
type Sender struct {
	cfg Config
}

// NewSender membentuk pengirim SMTP.
func NewSender(k Config) *Sender { return &Sender{cfg: k} }

// NotifyDocumentCompleted mengirim satu surel pemberitahuan kelengkapan dokumen.
func (p *Sender) NotifyDocumentCompleted(
	ctx context.Context,
	notice inboxreceivetka.Notice,
) error {
	if !p.cfg.Complete() {
		return errors.New("inboxreceivetka/notification: SMTP belum dikonfigurasi")
	}

	to := p.cfg.to()
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
		return fmt.Errorf("inboxreceivetka/notification: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungan, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung proses.
	_ = connection.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(connection, p.cfg.Host)
	if err != nil {
		_ = connection.Close()
		return fmt.Errorf("inboxreceivetka/notification: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay internal
	// sering belum memasang sertifikat; yang dipaksakan justru sebaliknya — kredensial hanya
	// dikirim setelah sambungan terenkripsi.
	//
	// Rule lama menetapkan `UseSSL=false` dan mengirimkan kata sandi apa adanya (`R-17`).
	// Itu tidak ditiru: menyalin kelemahan keamanan bukan kesetaraan perilaku.
	secured := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(nil); err != nil {
			return fmt.Errorf("inboxreceivetka/notification: menegakkan TLS: %w", err)
		}
		secured = true
	}

	if p.cfg.User != "" {
		if !secured {
			return errors.New(
				"inboxreceivetka/notification: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		auth := smtp.PlainAuth("", p.cfg.User, p.cfg.Password, p.cfg.Host)
		if err := client.Auth(auth); err != nil {
			// Galatnya tidak memuat kredensial, dan tidak boleh memuatnya: galat ini
			// berakhir di log.
			return fmt.Errorf(
				"inboxreceivetka/notification: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := client.Mail(p.cfg.From); err != nil {
		return fmt.Errorf(
			"inboxreceivetka/notification: server menolak alamat pengirim: %w", err)
	}
	for _, address := range to {
		if err := client.Rcpt(address); err != nil {
			return fmt.Errorf(
				"inboxreceivetka/notification: server menolak alamat tujuan: %w", err)
		}
	}

	write, err := client.Data()
	if err != nil {
		return fmt.Errorf("inboxreceivetka/notification: membuka badan surel: %w", err)
	}
	if _, err := write.Write(message); err != nil {
		_ = write.Close()
		return fmt.Errorf("inboxreceivetka/notification: menulis badan surel: %w", err)
	}
	if err := write.Close(); err != nil {
		return fmt.Errorf("inboxreceivetka/notification: menutup badan surel: %w", err)
	}
	return client.Quit()
}

// composeEmail membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
func composeEmail(from string, to []string, notice inboxreceivetka.Notice) []byte {
	var b strings.Builder

	clean := make([]string, 0, len(to))
	for _, address := range to {
		clean = append(clean, sanitizeHeader(address))
	}

	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + strings.Join(clean, ", ") + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(Subject(notice)) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(HTMLBody(notice))

	return []byte(b.String())
}

// Subject adalah baris subjek surel.
//
// Bentuknya dari `Activity/SubmitTanggalLengkapTKA-Act.xml`, yang merangkainya sebagai
// `"Notifikasi Kelengkapan Dokumen Klaim TKA " + "/ " + TempHTML.ClaimNo`. Spasi ganda di
// tengah rangkaian itu dirapikan; selebihnya sama persis.
func Subject(notice inboxreceivetka.Notice) string {
	return "Notifikasi Kelengkapan Dokumen Klaim TKA / " + strings.TrimSpace(notice.ClaimNumber)
}

// HTMLBody menyusun badan surel.
//
// # Keenam barisnya dari rule aslinya, pada urutan aslinya
//
// `HTML/NotificationKelengkapanTKA-HTML.xml` memuat satu tabel berisi enam baris. Label yang
// dibaca penerima dipakai apa adanya; yang TIDAK dipakai adalah nama properti penampungnya,
// yang di sana menyesatkan:
//
//	Label surel                  Properti Pega          Diisi dari
//	---------------------------- ---------------------- ------------------
//	No Klaim                     TempHTML.ClaimNo       Param.ClaimNo
//	No Polis                     TempHTML.PolicyNo      Param.NoPolis
//	Nama Tertanggung             TempHTML.District      Param.QQName
//	Nama Peserta                 TempHTML.DistrictID    Param.Insured
//	Date Of Loss                 TempHTML.Country       Param.DOL
//	Tanggal Kelengkapan Dokumen  TempHTML.CountryID     Param.Tanggal
//
// Kolom ketiga tabel itulah yang menyelesaikan pertentangan label antara Section dan Report
// Definition: `Param.QQName` berpasangan dengan label **Nama Tertanggung**, bukan Nama
// Peserta. Lihat inboxreceivetka.Task.
//
// Memakai `District` untuk menampung nama orang dan `Country` untuk menampung tanggal adalah
// bentuk yang sama dengan alias menyesatkan pada `03-CURRENT-ARCHITECTURE.md` §4.2, dan
// tidak dibawa.
//
// # Seluruh nilai di-escape
//
// Nama tertanggung dan nama peserta adalah teks yang dimasukkan orang lain; menempelkannya
// mentah ke dalam HTML berarti isi basis data dapat menyuntikkan markup ke dalam kotak masuk
// penerimanya.
func HTMLBody(notice inboxreceivetka.Notice) string {
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

	var b strings.Builder
	b.WriteString(`<html><body>`)
	b.WriteString(`<p>Dear,</p>`)
	b.WriteString(
		`<p>Berikut detail klaim TKA yang sudah diinput tanggal terima dokumen asli :</p>`)
	b.WriteString(`<table>`)
	b.WriteString(row("No Klaim", notice.ClaimNumber))
	b.WriteString(row("No Polis", notice.PolicyNumber))
	b.WriteString(row("Nama Tertanggung", notice.InsuredName))
	b.WriteString(row("Nama Peserta", notice.ParticipantName))
	b.WriteString(row("Date Of Loss", formatDate(notice.DateOfLoss)))
	b.WriteString(row("Tanggal Kelengkapan Dokumen", formatDate(&notice.CompletedAt)))
	b.WriteString(`</table>`)
	b.WriteString(`<p>Terima Kasih.</p>`)
	b.WriteString(`</body></html>`)
	return b.String()
}

// formatDate menuliskan tanggal dalam bentuk dd/mm/yyyy.
//
// Bentuk itu dari activity aslinya, yang menyusunnya dengan
// `@substring(Param.DOL,6,8)+"/"+@substring(Param.DOL,4,6)+"/"+@substring(Param.DOL,0,4)` —
// memotong teks `yyyymmdd` menjadi `dd/mm/yyyy`. Hasilnya ditiru; caranya tidak.
//
// Surel ditujukan kepada pembaca di Indonesia, sehingga tanggalnya ditulis dalam bentuk yang
// mereka baca sehari-hari — bukan ISO 8601 seperti pada kontrak API, yang pembacanya mesin.
func formatDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format("02/01/2006")
}

// sanitizeHeader membuang karakter yang dapat menyuntikkan baris kepala surel baru.
//
// Tanpa ini, satu nilai yang memuat CR atau LF dapat menambahkan penerima tersembunyi atau
// mengganti subjeknya — dan pada modul ini nilai itu dapat berasal dari nomor klaim, yaitu
// isi basis data.
func sanitizeHeader(value string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return strings.TrimSpace(replacer.Replace(value))
}

var _ inboxreceivetka.Notifier = (*Sender)(nil)
