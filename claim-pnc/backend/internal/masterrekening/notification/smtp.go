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
	"html"
	"strings"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/platform/smtpsend"
)

// Config adalah parameter sambungan SMTP.
type Config = smtpsend.Config

const defaultTimeout = smtpsend.DefaultTimeout

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

	to := p.cfg.Recipients()
	message := composeEmail(p.cfg.From, to, per)
	return p.send(ctx, to, message)
}

func (p *Sender) send(ctx context.Context, to []string, message []byte) error {
	return p.cfg.Send(ctx, to, message, "masterrekening/notification")
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
