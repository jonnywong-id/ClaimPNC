// Package notification memenuhi seam inboxpladlapredla.Notifier dengan surel SMTP.
//
// # Apa yang digantikan
//
// `Activity/ASMSendsEmailAttachments`, dipanggil `UpdateDetailPLA2` sebagai langkah
// terakhir yang menyentuh dunia luar. Surat itu membawa dokumen pendukung PLA/DLA ke
// reasuradur, dan ia **tidak dapat ditarik kembali**.
//
// # Yang sengaja TIDAK ditiru
//
// **Badan suratnya disusun di sini, bukan disalin.** `UpdateDetailPLA2:426250` memilih
// antara rule HTML `SendPLADLA` dan `SendPLADLAEnglish`, dan isi kedua rule itu adalah
// markup Pega yang merujuk properti klipboard — bukan HTML yang dapat dipindahkan apa
// adanya. Yang dibawa adalah **isi informasinya**: kepada siapa, klaim mana, dan dokumen
// apa yang dilampirkan.
//
// **Pemilihan bahasanya ditiru.** Pega memakai
// `@if(@contains(param.country,"INDONESIA"), …)` — pencocokan teks pada nama negara
// reasuradur. Aturan itu dibawa apa adanya di `SendableAdvice.InIndonesian`.
//
// **Penerimanya datang dari baris dokumennya**, bukan dari konfigurasi. Itu berbeda dari
// modul notifikasi lain di sistem ini, dan sengaja: surat ini ditujukan kepada reasuradur
// SATU dokumen, bukan kepada daftar penerima tetap. `D-67` menyangkut penerima
// notifikasi internal, bukan lawan bicara di luar perusahaan.
package notification

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"mime"
	"net"
	"net/smtp"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/platform/emailserver"
)

// Config adalah parameter sambungan SMTP.
type Config struct {
	// Account membaca akun server surel dari POOLDATA.M_EMAIL_SERVER_PNC (EMAIL_ACCOUNT)
	// setiap kali surel dikirim — pengganti Email Account Pega (Work Owner 2026-10-10).
	// Bila terisi, Host, Port, User, Password, dan From diambil dari akun itu; User dan From
	// sama-sama EMAIL_ADDRESS. Nil: isian di atas yang dipakai (mode tanpa Oracle).
	Account func(ctx context.Context) (emailserver.Account, error)

	Host string
	Port int

	// User dan Password boleh kosong. Banyak relay SMTP internal menerima pengirim dari
	// jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan menolak
	// konfigurasi yang sah.
	User     string
	Password string

	// From adalah alamat pengirim.
	From string

	// Timeout membatasi lama satu pengiriman. Kosong berarti 30 detik.
	//
	// Batas waktu WAJIB ada (`10-API-STRATEGY.md` §8.2): tanpa itu, satu relay yang
	// menggantung menahan permintaan pengguna tanpa batas — dan pengguna akan menekan
	// tombolnya lagi.
	Timeout time.Duration
}

// Complete menyatakan konfigurasinya cukup untuk mengirim.
//
// Ia diperiksa SEBELUM apa pun dikerjakan, dan itu yang membuat alasan penolakannya dapat
// menyebut apa yang kurang — bukan gagal di tengah dengan galat jaringan yang tidak
// menjelaskan apa-apa.
func (c Config) Complete() bool {
	return c.Account != nil || strings.TrimSpace(c.Host) != "" && c.Port > 0 &&
		strings.TrimSpace(c.From) != ""
}

// Missing menyebut isian konfigurasi yang belum diisi.
func (c Config) Missing() []string {
	kurang := []string{}
	if c.Account != nil {
		return kurang
	}
	if strings.TrimSpace(c.Host) == "" {
		kurang = append(kurang, "host SMTP")
	}
	if c.Port <= 0 {
		kurang = append(kurang, "port SMTP")
	}
	if strings.TrimSpace(c.From) == "" {
		kurang = append(kurang, "alamat pengirim")
	}
	return kurang
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

// Sender mengirim surat PLA/DLA lewat SMTP.
type Sender struct{ cfg Config }

// NewSender membentuk pengirim.
func NewSender(c Config) *Sender { return &Sender{cfg: c} }

// withAccount mengembalikan pengirim berkonfigurasi akun M_EMAIL_SERVER_PNC terbaru, atau
// pengirim itu sendiri bila Account tidak dipasang.
func (s *Sender) withAccount(ctx context.Context) (*Sender, error) {
	if s.cfg.Account == nil {
		return s, nil
	}
	account, err := s.cfg.Account(ctx)
	if err != nil {
		return nil, fmt.Errorf("inboxpladlapredla/notification: akun surel: %w", err)
	}
	c := s.cfg
	c.Host, c.Port = account.Host, account.Port
	c.User, c.Password, c.From = account.Address, account.Password, account.Address
	return &Sender{cfg: c}, nil
}

// SendAdvice mengirim satu surat beserta lampirannya.
//
// Galat berarti surat TIDAK terkirim — kontrak seam-nya, dan seluruh urutan Send
// bergantung padanya. Lihat catatan urutan di inboxpladlapredla/send.go.
func (s *Sender) SendAdvice(
	ctx context.Context,
	letter inboxpladlapredla.Letter,
) error {
	s, err := s.withAccount(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", inboxpladlapredla.ErrNotifierUnavailable, err)
	}
	if !s.cfg.Complete() {
		return fmt.Errorf("%w: %s belum diisi",
			inboxpladlapredla.ErrNotifierUnavailable,
			strings.Join(s.cfg.Missing(), ", "))
	}
	if len(letter.To) == 0 {
		return errors.New("inboxpladlapredla/notification: tidak ada penerima")
	}

	pesan, err := compose(s.cfg.From, letter)
	if err != nil {
		return err
	}

	return s.send(ctx, letter.To, pesan)
}

// send membuka sambungan dan menyerahkan suratnya.
func (s *Sender) send(ctx context.Context, to []string, pesan []byte) error {
	alamat := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))

	dialer := &net.Dialer{Timeout: s.cfg.timeout()}
	conn, err := dialer.DialContext(ctx, "tcp", alamat)
	if err != nil {
		return fmt.Errorf(
			"inboxpladlapredla/notification: menghubungi %s: %w", alamat, err)
	}
	defer func() { _ = conn.Close() }()

	// Batas waktu dipasang pada SAMBUNGANNYA, bukan hanya pada pembukaannya.
	//
	// Relay yang menerima sambungan lalu berhenti menjawab adalah kegagalan yang paling
	// sering terjadi, dan batas waktu pada dial saja tidak menangkapnya.
	_ = conn.SetDeadline(time.Now().Add(s.cfg.timeout()))

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("inboxpladlapredla/notification: memulai SMTP: %w", err)
	}
	defer func() { _ = client.Close() }()

	// STARTTLS dinegakkan bila server menawarkannya.
	//
	// Ia LEBIH penting di sini daripada di modul notifikasi lain: surat ini membawa nama
	// tertanggung, nomor polis, dan **berkas dokumen klaim** ke luar perusahaan. Sambungan
	// terbuka berarti seluruhnya melintas jaringan tanpa enkripsi.
	//
	// Ia tidak DIPAKSAKAN, mengikuti pola modul lain: relay internal sering belum memasang
	// sertifikat, dan memaksanya akan menolak konfigurasi yang sah. Yang dipaksakan justru
	// sebaliknya — kredensial tidak pernah dikirim lewat sambungan terbuka.
	//
	// Catatan yang perlu disadari saat menyalakan pengiriman: export Pega memuat
	// `UseSSL=false` pada SELURUH 31 lokasi SMTP-nya, sementara 16 di antaranya memakai
	// port 587 (`R-17`). Bila relay yang dipakai memang tidak menawarkan STARTTLS, surat
	// ini akan terkirim tanpa enkripsi — dan itu keputusan yang layak diambil sadar,
	// bukan diwarisi.
	adaTLS := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("inboxpladlapredla/notification: menegakkan TLS: %w", err)
		}
		adaTLS = true
	}

	if strings.TrimSpace(s.cfg.User) != "" {
		if !adaTLS {
			return errors.New(
				"inboxpladlapredla/notification: server surel tidak mendukung " +
					"STARTTLS; kredensial SMTP tidak dikirim lewat sambungan terbuka")
		}
		auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			// Galatnya TIDAK memuat kredensial, dan tidak boleh memuatnya: ia berakhir
			// di log.
			return fmt.Errorf("inboxpladlapredla/notification: autentikasi: %w", err)
		}
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("inboxpladlapredla/notification: pengirim ditolak: %w", err)
	}
	for _, penerima := range to {
		if err := client.Rcpt(penerima); err != nil {
			// Penerima yang ditolak menghentikan SELURUH pengiriman.
			//
			// Melanjutkan ke penerima berikutnya akan menghasilkan surat yang sampai
			// sebagian — lalu dokumennya ditandai terkirim, padahal salah satu
			// reasuradur tidak pernah menerimanya.
			return fmt.Errorf(
				"inboxpladlapredla/notification: penerima ditolak: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("inboxpladlapredla/notification: membuka badan: %w", err)
	}
	if _, err := w.Write(pesan); err != nil {
		return fmt.Errorf("inboxpladlapredla/notification: menulis badan: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("inboxpladlapredla/notification: menutup badan: %w", err)
	}

	return client.Quit()
}

// compose menyusun surat MIME beserta lampirannya.
func compose(from string, letter inboxpladlapredla.Letter) ([]byte, error) {
	batas, err := boundary()
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(strings.Join(letter.To, ", ")) + "\r\n")
	b.WriteString("Subject: " + encodeSubject(letter.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString(
		"Content-Type: multipart/mixed; boundary=\"" + batas + "\"\r\n\r\n")

	b.WriteString("--" + batas + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(wrap(base64.StdEncoding.EncodeToString([]byte(letter.HTMLBody))))
	b.WriteString("\r\n")

	for _, lampiran := range letter.Attachments {
		jenis := strings.TrimSpace(lampiran.MIMEType)
		if jenis == "" {
			jenis = mime.TypeByExtension(
				strings.ToLower(filepath.Ext(lampiran.Name)))
		}
		if jenis == "" {
			jenis = "application/octet-stream"
		}

		b.WriteString("--" + batas + "\r\n")
		b.WriteString("Content-Type: " + sanitizeHeader(jenis) + "\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("Content-Disposition: attachment; filename=\"" +
			sanitizeHeader(filepath.Base(lampiran.Name)) + "\"\r\n\r\n")
		b.WriteString(wrap(base64.StdEncoding.EncodeToString(lampiran.Content)))
		b.WriteString("\r\n")
	}

	b.WriteString("--" + batas + "--\r\n")
	return []byte(b.String()), nil
}

// boundary membuat pembatas MIME yang tidak mungkin muncul di dalam isinya.
//
// Ia diambil dari sumber acak kriptografis, bukan dari cap waktu. Pembatas yang dapat
// ditebak dapat muncul di dalam lampiran, dan surat yang pembatasnya tertabrak isi akan
// terbaca rusak oleh penerimanya.
func boundary() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf(
			"inboxpladlapredla/notification: membuat pembatas MIME: %w", err)
	}
	return "pladla-" + base64.RawURLEncoding.EncodeToString(buf), nil
}

// wrap memotong teks base64 menjadi baris 76 karakter sesuai MIME.
func wrap(teks string) string {
	const lebar = 76

	var b strings.Builder
	for len(teks) > lebar {
		b.WriteString(teks[:lebar])
		b.WriteString("\r\n")
		teks = teks[lebar:]
	}
	b.WriteString(teks)
	return b.String()
}

// encodeSubject menuliskan subjek yang aman bagi header surat.
//
// Subjeknya memuat nama tertanggung, dan nama dapat mengandung huruf di luar ASCII.
// Header yang memuat bita non-ASCII apa adanya akan dipotong atau diacak sebagian relay.
func encodeSubject(subjek string) string {
	bersih := sanitizeHeader(subjek)
	for _, r := range bersih {
		if r > 127 {
			return mime.QEncoding.Encode("UTF-8", bersih)
		}
	}
	return bersih
}

// sanitizeHeader membuang pemutus baris dari nilai header.
//
// Tanpa ini, satu baris baru yang ikut dari data — nama tertanggung, nama berkas —
// menyisipkan header baru ke dalam surat. Itu penyusupan header surat, dan di sini
// datanya memang berasal dari isian manusia.
func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

// HTMLBody menyusun badan surat.
//
// Isinya menjawab tiga hal yang harus diketahui penerimanya: surat apa ini, klaim mana,
// dan berapa dokumen yang dilampirkan. Bahasanya mengikuti negara reasuradur.
//
// SELURUH nilai yang berasal dari data di-escape. Nama tertanggung dan nama reasuradur
// adalah isian manusia, dan keduanya masuk ke dalam HTML.
func HTMLBody(
	tab inboxpladlapredla.Tab,
	advice inboxpladlapredla.SendableAdvice,
	claim inboxpladlapredla.ClaimSummary,
	attachments int,
) string {
	jenis := "PLA"
	if tab.Kind == inboxpladlapredla.KindDLA {
		jenis = "DLA"
	}

	e := html.EscapeString
	var b strings.Builder

	if advice.InIndonesian() {
		b.WriteString("<p>Kepada Yth. " + e(advice.Reinsurer) + ",</p>")
		b.WriteString("<p>Bersama ini kami sampaikan dokumen pendukung " + jenis +
			" untuk klaim berikut.</p>")
	} else {
		b.WriteString("<p>Dear " + e(advice.Reinsurer) + ",</p>")
		b.WriteString("<p>Please find attached the " + jenis +
			" supporting documents for the following claim.</p>")
	}

	baris := [][2]string{
		{"No " + jenis, advice.AdviceNo},
		{"No Klaim", claim.ClaimNo},
		{"No Polis", claim.PolicyNo},
		{"Tertanggung", claim.Insured},
		{"Tanggal Kejadian", claim.LossDate},
	}

	b.WriteString("<table cellpadding=\"4\">")
	for _, satu := range baris {
		b.WriteString("<tr><td><b>" + e(satu[0]) + "</b></td><td>" +
			e(satu[1]) + "</td></tr>")
	}
	b.WriteString("</table>")

	// Jumlah lampiran DISEBUTKAN, termasuk ketika nol.
	//
	// Penerima yang membaca "0 dokumen" tahu suratnya memang tanpa lampiran; penerima
	// yang tidak diberi tahu apa pun akan mengira lampirannya hilang di perjalanan.
	if advice.InIndonesian() {
		b.WriteString("<p>Dokumen terlampir: " + strconv.Itoa(attachments) +
			" berkas.</p>")
		b.WriteString("<p>Surat ini dikirim otomatis oleh sistem Claim PNC.</p>")
	} else {
		b.WriteString("<p>Attached documents: " + strconv.Itoa(attachments) +
			" file(s).</p>")
		b.WriteString("<p>This message was sent automatically by Claim PNC.</p>")
	}

	return b.String()
}
