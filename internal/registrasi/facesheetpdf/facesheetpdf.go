// Package facesheetpdf membentuk dokumen PDF Claim Face Sheet.
//
// Susunannya mengikuti rule HTML `ClaimFaceSheetHTML` dan satu contoh PDF produksi:
// judul bergaris bawah, POLICY DETAILS, CLAIM DETAILS, RESERVES, SPREADING CLAIM,
// REINS FAC OUT MEMBER SHARE, lalu CO MEMBER SHARE. Pega mengubah HTML itu menjadi PDF
// (generatePDF/PD4ML, tidak ada di export); di sini PDF disusun langsung.
//
// # Pustaka
//
// `github.com/go-pdf/fpdf` — mesin di bawah maroto, salah satu dari dua pustaka yang
// disebut `D-11` / Steering §1. Dipakai langsung karena ia membawa font dasar PDF (Times,
// Helvetica) tanpa berkas font: contoh PDF memakai Times, dan tidak ada berkas font yang
// perlu ikut di-deploy ke VM.
package facesheetpdf

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

// Renderer memenuhi registrasi.FaceSheetRenderer.
type Renderer struct{}

var _ registrasi.FaceSheetRenderer = Renderer{}

const (
	fontFamily = "Times"
	fontSize   = 10.0
	lineHeight = 6.0

	marginLeft  = 22.0
	marginTop   = 18.0
	labelWidth  = 42.0
	colonWidth  = 5.0
	valueWidth  = 120.0
	rightColumn = 120.0 // posisi X kolom Tanggal / PIC Admin / PIC Teknis
)

// Render membentuk PDF satu Claim Face Sheet.
func (Renderer) Render(f registrasi.FaceSheet) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, marginTop, marginLeft)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetTitle("Claim Face Sheet "+f.ClaimNumber, true)
	pdf.AddPage()
	w := writer{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	w.title("CLAIM FACE SHEET - " + strings.ToUpper(f.Title))
	w.sectionWithSide("POLICY DETAILS", [][2]string{
		{"Tanggal", date(f.Date)},
		{"PIC Admin", strings.ToUpper(f.AdminName)},
	})

	w.row("INSURED", f.Insured)
	w.row("POLICY NO", f.PolicyNumber)
	w.row("RISK LOCATION", f.RiskLocation)
	w.row("PERIODE OF", date(f.PeriodStart)+" - "+date(f.PeriodEnd))
	w.row("CLASS", f.Class)
	w.row("COVERAGE", f.Coverage)
	if f.Travel {
		w.row("PARTICIPANT(S)", f.Object)
	} else {
		w.row("OBJECT", f.Object)
	}
	w.row("TOTAL SUM INSURED", amount(f.SumInsured.Currency, f.SumInsured.Value))
	w.row("BROKER/AGENT", f.Broker)

	w.gap()
	w.sectionWithSide("CLAIM DETAILS", [][2]string{{"PIC Teknis", f.TechnicalPIC}})

	w.row("CLAIM NO", f.ClaimNumber)
	w.row("NOTIFICATION DATE", date(f.NotificationDate))
	w.row("DATE OF LOSS", date(f.DateOfLoss))
	w.row("REGISTRATION DATE", date(f.RegistrationDate))
	if !f.Travel {
		w.row("NATURE OF LOSS", f.NatureOfLoss)
		w.row("PREMIUM PAID ON", "")
	}
	w.boldRow("PAYMENTS", f.Payments)
	reserve := make([]string, 0, len(f.Reserve))
	for _, r := range f.Reserve {
		reserve = append(reserve, amount(r.Currency, r.Value))
	}
	w.row("RESERVES", strings.Join(reserve, "\n"))

	w.gap()
	w.heading("SPREADING CLAIM")
	for _, s := range f.Spreading {
		w.shareRow([]float64{28, 24, 10, 40}, s.Name, FormatPercent(s.Share)+" %", s.Currency, FormatMoney(s.Value))
	}

	w.gap()
	w.heading("REINS FAC OUT MEMBER SHARE")
	for _, s := range f.Reinsurer {
		w.shareRow([]float64{88, 24, 10, 40}, s.Name, FormatPercent(s.Share)+" %", s.Currency, FormatMoney(s.Value))
	}

	w.gap()
	w.heading("CO MEMBER SHARE")
	for _, s := range f.CoMember {
		w.shareRow([]float64{80, 18, 10, 40}, s.Name, FormatPercent(s.Share)+"%", s.Currency, FormatMoney(s.Value))
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("facesheetpdf: %w", err)
	}
	return out.Bytes(), nil
}

type writer struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func (w writer) font(style string, size float64) { w.pdf.SetFont(fontFamily, style, size) }

func (w writer) title(text string) {
	w.font("BU", 14)
	w.pdf.CellFormat(0, 10, w.tr(text), "", 1, "C", false, 0, "")
	w.pdf.Ln(6)
}

func (w writer) gap() { w.pdf.Ln(4) }

func (w writer) heading(text string) {
	w.font("B", fontSize)
	w.pdf.CellFormat(0, lineHeight+1, w.tr(text), "", 1, "L", false, 0, "")
}

// sectionWithSide mencetak judul bagian di kiri dan pasangan label–nilai di kanan,
// seperti tabel dua kolom pada template.
func (w writer) sectionWithSide(title string, side [][2]string) {
	top := w.pdf.GetY()
	w.font("B", fontSize)
	w.pdf.SetXY(marginLeft, top)
	w.pdf.CellFormat(rightColumn-marginLeft-10, lineHeight*float64(max(len(side), 1)), w.tr(title), "", 0, "C", false, 0, "")
	w.font("", fontSize)
	y := top
	for _, pair := range side {
		w.pdf.SetXY(rightColumn, y)
		w.pdf.CellFormat(20, lineHeight, w.tr(pair[0]), "", 0, "L", false, 0, "")
		w.pdf.MultiCell(0, lineHeight, w.tr(": "+pair[1]), "", "L", false)
		y = w.pdf.GetY()
	}
	w.pdf.SetXY(marginLeft, max(y, top+lineHeight*float64(len(side))))
	w.pdf.Ln(4)
}

func (w writer) row(label, value string)     { w.labelRow(label, value, "") }
func (w writer) boldRow(label, value string) { w.labelRow(label, value, "B") }

// labelRow mencetak `LABEL : nilai`, dengan nilai yang dibungkus bila panjang.
func (w writer) labelRow(label, value, style string) {
	w.font("", fontSize)
	w.pdf.SetX(marginLeft)
	w.pdf.CellFormat(labelWidth, lineHeight, w.tr(label), "", 0, "L", false, 0, "")
	w.pdf.CellFormat(colonWidth, lineHeight, ":", "", 0, "L", false, 0, "")
	w.font(style, fontSize)
	w.pdf.MultiCell(valueWidth, lineHeight, w.tr(value), "", "L", false)
	w.font("", fontSize)
}

// shareRow mencetak satu baris pembagian; kolom pertama dibungkus bila panjang.
func (w writer) shareRow(width []float64, cells ...string) {
	w.font("", fontSize)
	lines := w.pdf.SplitText(w.tr(cells[0]), width[0])
	height := lineHeight * float64(max(len(lines), 1))
	_, pageHeight := w.pdf.GetPageSize()
	_, _, _, bottom := w.pdf.GetMargins()
	if w.pdf.GetY()+height > pageHeight-bottom {
		w.pdf.AddPage()
	}
	x, y := marginLeft, w.pdf.GetY()
	w.pdf.SetXY(x, y)
	w.pdf.MultiCell(width[0], lineHeight, w.tr(cells[0]), "", "L", false)
	x += width[0]
	for i := 1; i < len(cells); i++ {
		w.pdf.SetXY(x, y)
		w.pdf.CellFormat(width[i], lineHeight, w.tr(cells[i]), "", 0, "L", false, 0, "")
		x += width[i]
	}
	w.pdf.SetXY(marginLeft, y+height)
}

// date mencetak tanggal kalender WIB sebagai dd/mm/yyyy; kosong bila tidak ada.
func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return clock.DateWIB(t).Format("02/01/2006")
}

func amount(currency string, value registrasi.Money) string {
	return strings.TrimSpace(currency + " " + FormatMoney(value))
}

// FormatMoney mencetak rupiah bulat dengan pemisah ribuan titik — `IDR 610.175.716.556`.
// Nilai disimpan dalam sen; pembulatan ke rupiah hanya terjadi di sini (`ADR-0016`).
func FormatMoney(v registrasi.Money) string {
	sen := int64(v)
	negative := sen < 0
	if negative {
		sen = -sen
	}
	rupiah := (sen + 50) / 100
	digits := fmt.Sprintf("%d", rupiah)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	if negative {
		return "-" + b.String()
	}
	return b.String()
}

// FormatPercent mencetak persentase dengan paling banyak tiga desimal dan koma desimal —
// `7,018`, `57`, `4,75`.
func FormatPercent(p registrasi.Percent) string {
	r := new(big.Rat).SetFrac64(int64(p), int64(registrasi.PercentFull)/100)
	text := strings.TrimRight(strings.TrimRight(r.FloatString(3), "0"), ".")
	return strings.Replace(text, ".", ",", 1)
}
