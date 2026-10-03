// Package lodpdf membentuk PDF Letter of Discharge (LOD) — tombol "Print LOD".
//
// Template Pega-nya tidak ada di export. Susunan dan kalimat setiap jenis disalin dari
// contoh PDF Work Owner di `Sample Form/` (lihat templates.go), termasuk salah ketik yang
// ada pada contohnya ("Penyataan", "ketentutan", "Materai", …): hasil yang benar selama
// migrasi adalah hasil yang sama dengan Pega (P-5).
//
// Susunan umumnya:
//
//	halaman 1  kop tertanggung, "SURAT PERNYATAAN (Letter of Discharge)", nilai + terbilang,
//	           tabel co member × share (jenis "dengan Co Member"), pernyataan, rekening
//	           tujuan, tempat–tanggal, meterai
//	lalu       Indeks Kepuasan Pelanggan
//
// Isian Nama/Jabatan/Alamat dan rekening tujuan dibiarkan kosong seperti contoh: diisi
// tertanggung pada kertasnya. Pengecualiannya Alamat pada LOD Ex Gratia (alamat polis).
package lodpdf

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Renderer memenuhi registrasi.LODRenderer.
type Renderer struct{}

var _ registrasi.LODRenderer = Renderer{}

const (
	font   = "Helvetica"
	margin = 20.0
	body   = 9.0
	line   = 4.3
)

// Render membentuk PDF satu LOD sesuai jenisnya (d.Type).
func (Renderer) Render(d registrasi.LODDocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, 18, margin)
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetTitle("Letter of Discharge", true)
	w := newWriter(pdf, d)

	if strings.TrimSpace(d.Type) == registrasi.LODMarineHull {
		marineHull(w)
	} else {
		s, known := letters[strings.TrimSpace(d.Type)]
		if !known {
			return nil, fmt.Errorf("lodpdf: jenis LOD %q tidak punya template", d.Type)
		}
		for page := &s; page != nil; page = page.next {
			w.letter(*page)
		}
		w.survey(s.survey)
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("lodpdf: %w", err)
	}
	return out.Bytes(), nil
}

// writer membungkus fpdf dengan isian dokumen dan markah teks templates.go.
type writer struct {
	pdf   *fpdf.Fpdf
	tr    func(string) string
	d     registrasi.LODDocument
	vals  map[string]string
	width float64 // lebar halaman
	text  float64 // lebar area tulis
}

func newWriter(pdf *fpdf.Fpdf, d registrasi.LODDocument) *writer {
	width, _ := pdf.GetPageSize()
	cur := strings.TrimSpace(d.Currency)
	return &writer{
		pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor(""), d: d, width: width, text: width - 2*margin,
		vals: map[string]string{
			"cur":       cur,
			"amt":       Amount(d.Amount),
			"words":     registrasi.Terbilang(d.Amount, currencyWord(cur)),
			"wordsbare": registrasi.Terbilang(d.Amount, ""),
			"pol":       strings.TrimSpace(d.PolicyNumber),
			"dol":       Date(d.DateOfLoss),
			"loc":       strings.TrimSpace(d.LossLocation),
			"addr":      strings.TrimSpace(d.InsuredAddress),
			"insured":   strings.TrimSpace(d.InsuredName),
			"long":      LongDate(d.PrintedAt),
			"short":     ShortDate(d.PrintedAt),
		},
	}
}

// segment adalah sepotong teks bergaya seragam.
type segment struct {
	text  string
	style string // gaya fpdf: "", "B", "I", "BI", "BU"
}

// parse memecah teks bermarkah menjadi potongan bergaya, lalu mengisi {placeholder}.
// Markah: *tebal*, ~miring~, ^tebal miring^, =tebal bergaris bawah=. Placeholder diisi
// SESUDAH dipecah, sehingga isian data (alamat, nama) tidak pernah dibaca sebagai markah.
func (w *writer) parse(tpl string) []segment {
	var out []segment
	var cur strings.Builder
	var bold, italic, boldItalic, underline bool
	style := func() string {
		s := ""
		if bold || boldItalic || underline {
			s += "B"
		}
		if italic || boldItalic {
			s += "I"
		}
		if underline {
			s += "U"
		}
		return s
	}
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, segment{text: w.fill(cur.String()), style: style()})
			cur.Reset()
		}
	}
	for _, r := range tpl {
		switch r {
		case '*':
			flush()
			bold = !bold
		case '~':
			flush()
			italic = !italic
		case '^':
			flush()
			boldItalic = !boldItalic
		case '=':
			flush()
			underline = !underline
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

func (w *writer) fill(s string) string {
	for k, v := range w.vals {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// para menulis satu paragraf: rata kiri-kanan bila tanpa markah, mengalir bila bermarkah.
func (w *writer) para(tpl string) {
	segs := w.parse(tpl)
	if len(segs) == 1 && segs[0].style == "" {
		w.pdf.SetFont(font, "", body)
		w.pdf.MultiCell(w.text, line, w.tr(segs[0].text), "", "J", false)
		return
	}
	w.flow(segs)
	w.pdf.Ln(line)
}

func (w *writer) flow(segs []segment) {
	for _, s := range segs {
		w.pdf.SetFont(font, s.style, body)
		w.pdf.Write(line, w.tr(s.text))
	}
}

// center menulis satu baris bermarkah di tengah.
func (w *writer) center(tpl string) {
	segs := w.parse(tpl)
	total := 0.0
	for _, s := range segs {
		w.pdf.SetFont(font, s.style, body)
		total += w.pdf.GetStringWidth(w.tr(s.text))
	}
	w.pdf.SetX((w.width - total) / 2)
	w.flow(segs)
	w.pdf.Ln(line)
}

func (w *writer) gap(h float64) { w.pdf.Ln(h) }

// letterhead menulis kop tertanggung dan judul surat.
func (w *writer) letterhead(s letter) {
	size := 10.0
	if s.kopSmall {
		size = 7.5
	}
	w.pdf.SetFont(font, "B", size)
	w.pdf.CellFormat(0, 5, w.tr("<INSURED LETTERHEAD>"), "", 1, "C", false, 0, "")
	second := ""
	if s.kopBold2 {
		second = "B"
	}
	w.pdf.SetFont(font, second, size)
	w.pdf.CellFormat(0, 5, w.tr("(KOP SURAT TERTANGGUNG)"), "", 1, "C", false, 0, "")
	w.gap(6)
	title := "B"
	if s.underline {
		title = "BU"
	}
	w.pdf.SetFont(font, title, 10)
	w.pdf.CellFormat(0, 5, "SURAT PERNYATAAN", "", 1, "C", false, 0, "")
	w.pdf.SetFont(font, "B", 9)
	w.pdf.CellFormat(0, 5, "(Letter of Discharge)", "", 1, "C", false, 0, "")
	w.gap(7)
}

// fields menulis baris isian "Label : nilai". values boleh lebih pendek dari labels.
func (w *writer) fields(style fieldStyle, labels []string, values ...string) {
	for i, l := range labels {
		v := ""
		if i < len(values) {
			v = values[i]
		}
		w.pdf.SetFont(font, "", body)
		switch style {
		case fieldsInline:
			w.pdf.MultiCell(w.text, 5.5, w.tr(strings.TrimSpace(l+" : "+v)), "", "L", false)
		case fieldsFar:
			w.pdf.CellFormat(w.text-15, 5.5, w.tr(l), "", 0, "L", false, 0, "")
			w.pdf.CellFormat(15, 5.5, ":", "", 1, "L", false, 0, "")
		default:
			labelW := 35.0
			if style == fieldsTight {
				labelW = 15
			}
			w.pdf.CellFormat(labelW, 5.5, w.tr(l), "", 0, "L", false, 0, "")
			w.pdf.CellFormat(4, 5.5, ":", "", 0, "L", false, 0, "")
			w.pdf.MultiCell(w.text-labelW-4, 5.5, w.tr(v), "", "L", false)
		}
	}
}

// members menulis tabel co member: nama, share, dan bagiannya.
func (w *writer) members() {
	nameW, pctW := w.text*0.64, w.text*0.09
	valW := w.text - nameW - pctW
	w.pdf.SetDrawColor(150, 150, 150)
	w.pdf.SetFont(font, "", body)
	for _, m := range w.d.Members {
		w.pdf.CellFormat(nameW, 5.2, w.tr(m.Name), "1", 0, "L", false, 0, "")
		w.pdf.CellFormat(pctW, 5.2, Percent(m.Percent)+" %", "1", 0, "L", false, 0, "")
		w.pdf.CellFormat(valW, 5.2, Decimal(m.Value, 3), "1", 1, "L", false, 0, "")
	}
	w.pdf.SetDrawColor(0, 0, 0)
}

// recipient menulis blok rekening tujuan pembayaran.
func (w *writer) recipient(style recipientStyle) {
	switch style {
	case recipientBranch:
		w.fields(fieldsTable, []string{"Nama", "No Rekening"})
		w.pdf.SetFont(font, "", body)
		w.pdf.CellFormat(35, 5.5, "Bank", "", 0, "L", false, 0, "")
		w.pdf.CellFormat(20, 5.5, ":", "", 0, "L", false, 0, "")
		w.pdf.CellFormat(w.text-60, 5.5, "cabang", "", 0, "L", false, 0, "")
		w.pdf.CellFormat(5, 5.5, ":", "", 1, "L", false, 0, "")
	case recipientDotted:
		for _, l := range []string{"Nama", "No Rekening", "Bank"} {
			w.para("*" + l + " : ............................*")
			w.gap(3)
		}
	default:
		labels := []string{"Nama", "Email", "Bank", "No Rekening"}
		if style == recipientTight {
			w.fields(fieldsTight, labels)
			return
		}
		w.fields(fieldsTable, labels)
	}
}

// letter menulis satu halaman surat LOD berbahasa Indonesia.
func (w *writer) letter(s letter) {
	w.pdf.AddPage()
	w.letterhead(s)
	w.para("Kepada : PT. Asuransi Sinar Mas")
	w.gap(3)
	w.para(s.signer)
	w.gap(2)
	var values []string
	if s.address {
		values = []string{"", "", w.vals["addr"]}
	}
	w.fields(s.fields, []string{"Nama", "Jabatan", "Alamat"}, values...)
	w.gap(4)

	w.para(s.amount)
	if s.table {
		w.gap(1)
		w.members()
		w.gap(2)
	} else {
		w.gap(4)
	}
	for _, p := range s.body {
		w.para(p)
		w.gap(4)
	}
	w.para(s.intro)
	w.gap(1)
	w.recipient(s.recipient)
	w.gap(4)
	w.para(s.closing)
	w.gap(5)
	w.center(s.place)
	w.gap(6)
	for _, l := range s.sign {
		w.center(l)
	}
	w.gap(9)
	for _, l := range s.roles {
		w.center(l)
	}
	if s.footnote != "" {
		if s.footnotePage2 {
			w.pdf.AddPage()
		} else {
			w.gap(4)
		}
		w.para(s.footnote)
	}
}

// survey menggambar halaman Indeks Kepuasan Pelanggan: lima wajah dari sangat puas sampai
// sangat tidak puas, masing-masing dengan kotak centang di bawahnya.
func (w *writer) survey(s surveyStyle) {
	pdf := w.pdf
	pdf.AddPage()
	start := pdf.GetY()
	pdf.Ln(8)
	pdf.SetFont(font, "B", 12)
	pdf.CellFormat(0, 7, "Indeks Kepuasan Pelanggan", "", 1, "C", false, 0, "")
	pdf.Ln(3)
	pdf.SetFont(font, "", body)
	pdf.CellFormat(0, 5, w.tr("Mohon diberi tanda centang (V) sesuai tingkat kepuasan Anda terhadap pelayanan kami"), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	top := pdf.GetY()
	if s.inner {
		pdf.Rect(margin, top, w.text, 34, "D")
	}
	cell := 17.0
	x0 := margin + 3
	if s.centered {
		x0 = (w.width - cell*5) / 2
	}
	pdf.SetLineWidth(0.6)
	pdf.Rect(x0-0.5, top+2.5, cell*5+1, 27, "D")
	for i := 0; i < 5; i++ {
		x := x0 + float64(i)*cell
		pdf.Rect(x, top+3, cell, 16, "D")
		pdf.Rect(x, top+19, cell, 10, "D")
		face(pdf, x+cell/2, top+11, 6.2, i)
	}
	pdf.SetLineWidth(0.2)
	pdf.SetY(top + 40)

	pdf.SetFont(font, "", body)
	pdf.CellFormat(0, 5, "Saran dan Komentar:", "", 1, "C", false, 0, "")
	for i := 0; i < 3; i++ {
		pdf.CellFormat(0, 5, strings.Repeat("_", 75), "", 1, "C", false, 0, "")
	}
	pdf.Ln(6)
	pdf.CellFormat(w.text/2, 5, "Nama: _________________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(w.text/2, 5, "Email: ____________________________", "", 1, "C", false, 0, "")
	pdf.Ln(6)
	pdf.CellFormat(0, 5, "Terimakasih Atas Masukan Anda", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, w.tr(s.closing), "", 1, "C", false, 0, "")
	if s.outer {
		pdf.Rect(margin-3, start, w.text+6, pdf.GetY()-start+4, "D")
	}
}

// face menggambar satu wajah: mood 0 sangat puas … 4 sangat tidak puas.
func face(pdf *fpdf.Fpdf, cx, cy, r float64, mood int) {
	pdf.SetLineWidth(0.5)
	pdf.Circle(cx, cy, r, "D")
	pdf.Circle(cx-r*0.35, cy-r*0.3, r*0.08, "F")
	pdf.Circle(cx+r*0.35, cy-r*0.3, r*0.08, "F")
	mouthY := cy + r*0.35
	switch mood {
	case 0, 1: // senyum
		depth := r * 0.25
		if mood == 0 {
			depth = r * 0.4
		}
		// Sumbu y fpdf mengarah ke bawah: sudut 180–360 adalah busur bawah (senyum).
		pdf.Arc(cx, mouthY-depth, r*0.45, depth, 0, 180, 360, "D")
	case 2: // datar
		pdf.Line(cx-r*0.4, mouthY, cx+r*0.4, mouthY)
	default: // cemberut
		depth := r * 0.2
		if mood == 4 {
			depth = r * 0.35
		}
		pdf.Arc(cx, mouthY+depth, r*0.45, depth, 0, 0, 180, "D")
	}
	pdf.SetLineWidth(0.6)
}

// currencyWord adalah kata mata uang pada terbilang. Contoh hanya IDR ("rupiah"); mata uang
// lain ditulis kodenya.
func currencyWord(code string) string {
	if strings.EqualFold(strings.TrimSpace(code), "IDR") {
		return "rupiah"
	}
	return strings.TrimSpace(code)
}

// Amount menulis sen dengan pemisah ribuan titik dan desimal koma, nol di belakang dibuang:
// 7.718.764.830 sen → "77.187.648,3".
func Amount(v registrasi.Money) string {
	return Decimal(new(big.Rat).SetFrac(big.NewInt(int64(v)), big.NewInt(100)), 2)
}

// Decimal membulatkan setengah ke atas pada `places` desimal, lalu membuang nol di belakang:
// 44.382.897,7725 → "44.382.897,773"; 2.508.598,56975 → "2.508.598,57".
func Decimal(v *big.Rat, places int) string {
	if v == nil {
		return ""
	}
	s := v.FloatString(places) // FloatString membulatkan setengah menjauhi nol
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

// Percent menulis persen polis seperti contoh: 57,5% → "57.5" (titik, tanpa nol di belakang).
func Percent(p registrasi.Percent) string {
	s := new(big.Rat).SetFrac(big.NewInt(int64(p)), big.NewInt(10_000)).FloatString(4)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Date menulis tanggal kejadian dd/MM/yyyy (WIB).
func Date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format("02/01/2006")
}

// ShortDate menulis tanggal cetak dd/MM/yy (WIB) — "30/09/26" pada contoh reinstatement dan
// indemnity terlebih dahulu.
func ShortDate(t time.Time) string {
	return t.In(clock.ZoneWIB).Format("02/01/06")
}

var months = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus",
	"September", "Oktober", "November", "Desember"}

// LongDate menulis "16 September 2026" (WIB).
func LongDate(t time.Time) string {
	w := t.In(clock.ZoneWIB)
	return fmt.Sprintf("%d %s %d", w.Day(), months[w.Month()-1], w.Year())
}
