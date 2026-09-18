// Package notifikasi memenuhi seam masterrekening.Notifier dengan surel SMTP.
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
package notifikasi

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

// Konfigurasi adalah parameter sambungan SMTP.
type Konfigurasi struct {
	Host string
	Port int

	// Pengguna dan KataSandi boleh kosong. Banyak relay SMTP internal menerima
	// pengirim dari jaringan tepercaya tanpa autentikasi; memaksakan kredensial akan
	// menolak konfigurasi yang sah.
	Pengguna  string
	KataSandi string

	// Dari adalah alamat pengirim.
	Dari string

	// Kepada adalah mailbox Tim IT yang menerima peringatan kegagalan integrasi.
	//
	// Ia konfigurasi, bukan diturunkan dari data: peringatan ini ditujukan ke pihak
	// yang dapat MEMPERBAIKI kegagalan integrasi, dan itu bukan orang yang kebetulan
	// menekan tombol approve (keputusan Work Owner 2026-09-17).
	Kepada []string

	// Tenggang membatasi lama menunggu server SMTP. Nol berarti nilai baku.
	Tenggang time.Duration
}

// Lengkap menyatakan konfigurasi ini cukup untuk mengirim surel.
func (k Konfigurasi) Lengkap() bool {
	return strings.TrimSpace(k.Host) != "" &&
		k.Port > 0 &&
		strings.TrimSpace(k.Dari) != "" &&
		len(k.tujuan()) > 0
}

// tujuan mengembalikan alamat penerima yang benar-benar terisi.
func (k Konfigurasi) tujuan() []string {
	hasil := make([]string, 0, len(k.Kepada))
	for _, alamat := range k.Kepada {
		if potong := strings.TrimSpace(alamat); potong != "" {
			hasil = append(hasil, potong)
		}
	}
	return hasil
}

const tenggangBaku = 20 * time.Second

// Pengirim mengirim peringatan lewat SMTP.
type Pengirim struct {
	konf Konfigurasi
}

// PengirimBaru membentuk pengirim SMTP.
func PengirimBaru(k Konfigurasi) *Pengirim { return &Pengirim{konf: k} }

// PeringatkanKegagalanKasir mengirim satu surel peringatan ke mailbox Tim IT.
func (p *Pengirim) PeringatkanKegagalanKasir(ctx context.Context, per masterrekening.Peringatan) error {
	if !p.konf.Lengkap() {
		return errors.New("masterrekening/notifikasi: SMTP belum dikonfigurasi")
	}

	tujuan := p.konf.tujuan()
	pesan := susunSurel(p.konf.Dari, tujuan, per)
	return p.kirim(ctx, tujuan, pesan)
}

func (p *Pengirim) kirim(ctx context.Context, tujuan []string, pesan []byte) error {
	tenggang := p.konf.Tenggang
	if tenggang <= 0 {
		tenggang = tenggangBaku
	}

	alamat := net.JoinHostPort(p.konf.Host, fmt.Sprint(p.konf.Port))
	pemutar := &net.Dialer{Timeout: tenggang}

	sambungan, err := pemutar.DialContext(ctx, "tcp", alamat)
	if err != nil {
		return fmt.Errorf("masterrekening/notifikasi: menghubungi server surel: %w", err)
	}
	// Tenggang dipasang pada sambungan, bukan hanya pada pemutarnya: server SMTP yang
	// menerima koneksi lalu diam adalah kegagalan yang paling sering menggantung
	// proses.
	_ = sambungan.SetDeadline(time.Now().Add(tenggang))

	klien, err := smtp.NewClient(sambungan, p.konf.Host)
	if err != nil {
		_ = sambungan.Close()
		return fmt.Errorf("masterrekening/notifikasi: memulai percakapan SMTP: %w", err)
	}
	defer func() { _ = klien.Close() }()

	// STARTTLS dipakai bila server menawarkannya. Ia tidak dipaksakan karena relay
	// internal sering belum memasang sertifikat; yang dipaksakan justru sebaliknya —
	// kredensial hanya dikirim setelah sambungan terenkripsi.
	adaTLS := false
	if ok, _ := klien.Extension("STARTTLS"); ok {
		if err := klien.StartTLS(nil); err != nil {
			return fmt.Errorf("masterrekening/notifikasi: menegakkan TLS: %w", err)
		}
		adaTLS = true
	}

	if p.konf.Pengguna != "" {
		if !adaTLS {
			return errors.New(
				"masterrekening/notifikasi: server surel tidak mendukung STARTTLS; " +
					"kredensial SMTP tidak dikirim melalui sambungan terbuka")
		}
		auth := smtp.PlainAuth("", p.konf.Pengguna, p.konf.KataSandi, p.konf.Host)
		if err := klien.Auth(auth); err != nil {
			// Galatnya tidak memuat kredensial, dan tidak boleh memuatnya: galat ini
			// berakhir di log.
			return fmt.Errorf("masterrekening/notifikasi: autentikasi SMTP ditolak: %w", err)
		}
	}

	if err := klien.Mail(p.konf.Dari); err != nil {
		return fmt.Errorf("masterrekening/notifikasi: server menolak alamat pengirim: %w", err)
	}
	for _, alamat := range tujuan {
		if err := klien.Rcpt(alamat); err != nil {
			return fmt.Errorf("masterrekening/notifikasi: server menolak alamat tujuan: %w", err)
		}
	}

	tulis, err := klien.Data()
	if err != nil {
		return fmt.Errorf("masterrekening/notifikasi: membuka badan surel: %w", err)
	}
	if _, err := tulis.Write(pesan); err != nil {
		_ = tulis.Close()
		return fmt.Errorf("masterrekening/notifikasi: menulis badan surel: %w", err)
	}
	if err := tulis.Close(); err != nil {
		return fmt.Errorf("masterrekening/notifikasi: menutup badan surel: %w", err)
	}
	return klien.Quit()
}

// susunSurel membentuk pesan RFC 5322 lengkap dengan badan HTML.
//
// Ia dipisahkan dari pengiriman supaya isinya dapat diuji tanpa server SMTP.
func susunSurel(dari string, tujuan []string, p masterrekening.Peringatan) []byte {
	var b strings.Builder

	bersih := make([]string, 0, len(tujuan))
	for _, alamat := range tujuan {
		bersih = append(bersih, bersihkanHeader(alamat))
	}

	b.WriteString("From: " + bersihkanHeader(dari) + "\r\n")
	b.WriteString("To: " + strings.Join(bersih, ", ") + "\r\n")
	b.WriteString("Subject: " + bersihkanHeader(Subjek(p)) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(BadanHTML(p))

	return []byte(b.String())
}

// Subjek adalah baris subjek surel peringatan.
func Subjek(p masterrekening.Peringatan) string {
	return "Pendaftaran rekening ke Kasir GAGAL — " + p.Rekening.NomorRekening
}

// BadanHTML menyusun badan surel.
//
// Seluruh nilai yang berasal dari data di-escape. Nama pemilik rekening dan pesan dari
// Kasir adalah teks yang dimasukkan orang lain; menempelkannya mentah ke dalam HTML
// berarti isi basis data dapat menyuntikkan markup ke dalam kotak masuk penerimanya.
func BadanHTML(p masterrekening.Peringatan) string {
	r := p.Rekening

	baris := func(label, nilai string) string {
		if strings.TrimSpace(nilai) == "" {
			nilai = "—"
		}
		return "<tr><td style=\"padding:4px 12px 4px 0;color:#555\">" + html.EscapeString(label) +
			"</td><td style=\"padding:4px 0\"><b>" + html.EscapeString(nilai) + "</b></td></tr>"
	}

	diputuskanOleh := strings.TrimSpace(p.Diputuskan.Nama)
	if surel := strings.TrimSpace(p.Diputuskan.Email); surel != "" {
		if diputuskanOleh == "" {
			diputuskanOleh = surel
		} else {
			diputuskanOleh += " (" + surel + ")"
		}
	}

	var b strings.Builder
	b.WriteString("<html><body style=\"font-family:Arial,Helvetica,sans-serif;font-size:14px;color:#222\">")
	b.WriteString("<p>Yth. Tim IT,</p>")
	b.WriteString("<p>Rekening berikut telah <b>disetujui komite</b>, tetapi <b>gagal didaftarkan " +
		"ke sistem Kasir</b>. Selama pendaftaran belum berhasil, rekening ini <b>tidak dapat " +
		"dipakai membayar klaim</b>.</p>")
	b.WriteString("<table style=\"border-collapse:collapse\">")
	b.WriteString(baris("Nomor rekening", r.NomorRekening))
	b.WriteString(baris("Nama pemilik", r.NamaPemilik))
	b.WriteString(baris("Bank", r.NamaBank))
	b.WriteString(baris("Cabang", r.CabangBank))
	b.WriteString(baris("Tipe rekening", r.TipeRekening))
	b.WriteString(baris("Kode respons Kasir", p.Kode))
	b.WriteString(baris("Pesan dari Kasir", p.Pesan))
	b.WriteString(baris("Diputuskan oleh", diputuskanOleh))
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

// bersihkanHeader memotong nilai header pada CR atau LF yang pertama.
//
// Tanpa ini, satu baris baru di dalam nama atau alamat cukup untuk menyisipkan header
// tambahan — termasuk penerima tambahan — ke dalam surel yang dikirim aplikasi.
//
// Ia MEMOTONG, bukan mengganti baris baru dengan spasi. Mengganti dengan spasi memang
// sudah menghalangi terbentuknya header baru, tetapi teks yang disusupkan tetap ikut
// terkirim di dalam header — dan sesuatu yang memuat baris baru di dalam alamat surel
// sudah cacat sejak awal. Yang sah selalu berada sebelum baris baru pertama.
func bersihkanHeader(nilai string) string {
	if i := strings.IndexAny(nilai, "\r\n"); i >= 0 {
		nilai = nilai[:i]
	}
	return strings.TrimSpace(nilai)
}

var _ masterrekening.Notifier = (*Pengirim)(nil)
