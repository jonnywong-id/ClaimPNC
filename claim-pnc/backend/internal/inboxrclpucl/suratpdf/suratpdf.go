// Package suratpdf membentuk PDF surat RCL/PUCL — tombol "Download Dokumen".
//
// # Sumbernya templat Pega, bukan rancangan kami
//
// Susunan, urutan baris, dan seluruh teks tetapnya disalin dari `HTML/SuratPUCL-HTML.xml`
// (`pySourceStream`), yang dikirim Work Owner 2026-10-02. Sebelum itu templatnya tidak ada di
// export sama sekali, dan modul ini menolak menerbitkan surat apa pun.
//
// Kalimat tetapnya dibawa APA ADANYA, termasuk tanda baca dan spasinya — `D-13` menetapkan
// teks layar mengikuti Pega, dan surat ini keluar ke cabang.
//
// # Tiga baris yang tergambar KOSONG, dan kenapa
//
//	Nomor Kontrak         `PUCLStatus.NIK` — tidak punya kolom di tabel mana pun
//	Unit Bisnis / Seksi   `PUCLStatus.BusinessUnitSeksi` — idem
//	Nama penanda tangan   `SIGNATURE_NAME` — bukan kolom `POOLDATA.M_SIGNATURE1`
//
// Ketiganya hidup di clipboard Pega, bukan di basis data. Barisnya tetap DIGAMBAR beserta
// labelnya, bukan dihilangkan: baris yang hilang membuat surat terbaca seolah memang tidak
// punya isian itu, sedangkan baris kosong menunjukkan ada yang belum terbawa.
//
// Logo `webweb/LogoASM.PNG` juga tidak ada di export, sehingga kop gambarnya belum tergambar.
package suratpdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"claim-pnc/internal/inboxrclpucl"
)

// Renderer memenuhi inboxrclpucl.LetterRenderer.
type Renderer struct{}

var _ inboxrclpucl.LetterRenderer = Renderer{}

// Ukuran halaman mengikuti CSS templat: `width: 595px` adalah lebar A4 dalam titik, dan
// `margin-left:55px; margin-right:45px` menjadi tepi kiri dan kanannya.
const (
	fontSurat = "Times"
	body      = 10.0
	line      = 4.6
	kiri      = 19.0
	kanan     = 16.0
	atas      = 14.0
)

// Render membentuk PDF satu surat RCL/PUCL.
func (Renderer) Render(d inboxrclpucl.LetterDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(kiri, atas, kanan)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetTitle("Dokumen RCL/PUCL", true)
	pdf.AddPage()

	w := &writer{pdf: pdf}

	// Kop: dua baris kosong seperti `<br><br>` pada templat, lalu tanggal dan nomor surat.
	w.gap(line * 2)
	w.text(body, false, "Jakarta, "+d.LetterDate)
	w.text(body, false, d.LetterNumber)
	w.gap(line)

	w.text(body, false, "Kepada Yth.")
	w.text(body, true, d.Recipient)
	if alamat := strings.TrimSpace(d.RecipientAddress); alamat != "" {
		w.wrapped(body, true, alamat)
	}
	w.gap(line)

	// Tabel pertama — seluruh barisnya tebal di templat, termasuk labelnya.
	for _, baris := range [][2]string{
		{"UP", d.SumInsured},
		{"Nomor Kontrak", d.ContractNumber},
		{"Polis", d.PolicyNumber},
		{"Unit Bisnis / Seksi", d.BusinessUnit},
		{"Nama Peserta", d.InsuredName},
		{"Jumlah Tagihan", strings.TrimSpace(d.BillCurrency + " " + d.BillAmount)},
		{"Tanggal Kejadian", d.LossDate},
	} {
		w.row(baris[0], baris[1])
	}
	w.gap(line)

	w.text(body, false, "Dengan Hormat,")
	w.gap(line)
	w.subject(d.Subject)
	w.gap(line)

	// Ketiga keterangan: pembuka biasa, isi TEBAL, penutup biasa — seperti templatnya.
	w.paragraph(false, d.OpeningNote)
	w.paragraph(true, d.BodyNote)
	w.paragraph(false, d.ClosingNote)

	w.gap(line)
	w.text(body, false, "Hormat Kami,")

	w.signatures(d)
	w.footer()

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("suratpdf: menulis PDF: %w", err)
	}
	return buf.Bytes(), nil
}

type writer struct{ pdf *fpdf.Fpdf }

func (w *writer) set(size float64, tebal bool) {
	gaya := ""
	if tebal {
		gaya = "B"
	}
	w.pdf.SetFont(fontSurat, gaya, size)
}

func (w *writer) gap(h float64) { w.pdf.Ln(h) }

func (w *writer) text(size float64, tebal bool, s string) {
	w.set(size, tebal)
	w.pdf.MultiCell(0, line, teksAman(s), "", "L", false)
}

// wrapped menggambar teks yang boleh memuat pergantian baris sendiri — alamat, misalnya.
func (w *writer) wrapped(size float64, tebal bool, s string) {
	for _, baris := range strings.Split(teksAman(s), "\n") {
		w.text(size, tebal, baris)
	}
}

// row menggambar satu baris "Label : Nilai" dengan lebar label tetap.
//
// Lebarnya TETAP, bukan mengikuti isi: di templat ia tabel, dan tanda titik duanya berderet
// lurus. Label yang melebar mengikuti isi akan membuat deretnya patah.
func (w *writer) row(label, nilai string) {
	const lebarLabel = 38.0

	w.set(body, true)
	w.pdf.CellFormat(lebarLabel, line, teksAman(label), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(5, line, ":", "", 0, "L", false, 0, "")

	// Nilainya MultiCell supaya isian panjang — nama objek, misalnya — turun ke baris
	// berikutnya alih-alih tertimpa tepi kanan.
	sisa, _ := w.pdf.GetPageSize()
	w.pdf.MultiCell(sisa-kiri-kanan-lebarLabel-5, line, teksAman(nilai), "", "L", false)
}

// subject menggambar baris "Perihal", yang di templat BERGARIS BAWAH dan tebal.
func (w *writer) subject(s string) {
	w.pdf.SetFont(fontSurat, "B", body)
	w.pdf.CellFormat(38, line, "Perihal", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(5, line, ":", "", 0, "L", false, 0, "")

	w.pdf.SetFont(fontSurat, "BU", body)
	w.pdf.MultiCell(0, line, teksAman(s), "", "L", false)
}

func (w *writer) paragraph(tebal bool, s string) {
	if strings.TrimSpace(s) == "" {
		return
	}
	w.text(body, tebal, s)
	w.gap(line)
}

// signatures menggambar blok tanda tangan — dua kolom, seperti templatnya.
//
// Jabatannya teks TETAP di templat; namanya datang dari data dan untuk sekarang kosong.
func (w *writer) signatures(d inboxrclpucl.LetterDocument) {
	const lebar = 58.0

	// Ruang untuk tanda tangan basah atau cap. Di templat tingginya 90px.
	w.gap(line * 5)

	w.set(body, true)
	w.pdf.CellFormat(lebar, line, teksAman(d.SignerLeftName), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(14, line, "", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(lebar, line, teksAman(d.SignerRightName), "", 1, "L", false, 0, "")

	w.set(body, false)
	w.pdf.CellFormat(lebar, line, "Accident & Health Ins. Claim Dept.Head", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(14, line, "", "", 0, "L", false, 0, "")
	w.pdf.CellFormat(lebar, line, "Claim Section Head", "", 1, "L", false, 0, "")
}

// footer menggambar kaki surat — seluruhnya teks tetap pada templat.
func (w *writer) footer() {
	w.gap(line * 2)
	w.text(body, true, "PT Asuransi Sinar Mas")
	w.text(body, false, "Plaza Simas, Jl.KH. Fachrudin no. 18 Jakarta Pusat 10250 - Indonesia")
	w.text(body, false,
		"24 Hour Customer Care :(021)235 67 888; Telp: (021)390 2141 (Hunting); "+
			"Faks: (021) 390 2159 / 60, www.sinarmas.co.id")
}

// teksAman menyiapkan teks untuk fpdf, yang bekerja pada pengkodean satu bita.
//
// Isian surat datang dari ketikan petugas dan dapat memuat karakter di luar Latin-1 — tanda
// kutip miring yang disalin dari Word adalah contoh yang paling sering. Tanpa penyiapan ini
// karakter itu tergambar sebagai sampah, dan yang membacanya adalah cabang.
func teksAman(s string) string {
	bersih := strings.NewReplacer(
		"‘", "'", "’", "'",
		"“", `"`, "”", `"`,
		"–", "-", "—", "-",
		"…", "...",
		" ", " ",
		"\r\n", "\n", "\r", "\n",
	).Replace(s)

	var b strings.Builder
	for _, r := range bersih {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 32:
			// Kendali lain dibuang — ia tidak tergambar, dan sebagiannya merusak tata letak.
		case r <= 255:
			b.WriteRune(r)
		default:
			b.WriteRune('?')
		}
	}
	return b.String()
}
