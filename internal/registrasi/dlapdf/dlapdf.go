// Package dlapdf membentuk dokumen PDF Definite Loss Advice.
//
// Kerangkanya sama untuk keempat template Pega — `DLA_CoinsHTML`, `DLAFACOUT_HTML`,
// `DLABPPDAN_HTML`, dan `DLATREATY_HTML`: kop, judul "DEFINITE LOSS ADVICE", nomor, daftar
// label–nilai, "Jakarta, tanggal", tanda tangan, nama komite penyetuju, lalu alamat
// perusahaan di kaki halaman. Yang berbeda hanya label penerima dan baris bagiannya.
package dlapdf

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/facesheetpdf"
)

// Renderer memenuhi registrasi.DLARenderer. Logo adalah PNG kop per entitas (ASM, ASI).
type Renderer struct {
	Logo map[string][]byte
}

var _ registrasi.DLARenderer = Renderer{}

const (
	font        = "Helvetica"
	marginLeft  = 22.0
	labelWidth  = 52.0
	valueWidth  = 115.0
	lineHeight  = 5.5
	labelSize   = 10.0
	valueSize   = 9.0
	pageMarginB = 30.0
)

// footer adalah alamat kaki halaman per entitas — blok footer template DLA.
var footer = map[string][2]string{
	"ASM": {"PT Asuransi Sinar Mas",
		"Plaza Simas, Jl.KH.Fachrudin no.18 Jakarta Pusat 10250-Indonesia\n24 Hour Customer Care:(021)-235-67-888; Telp:(021)-390-2141 (Hunting); Faks:(021)-390-2159/60,\nwww.sinarmas.co.id"},
	"ASI": {"PT ASURANSI SIMAS INSURTECH",
		"Gedung Menara Tekno Lantai 5, Jl. K. H. Fachrudin No. 19, Jakarta, 10250 - Indonesia\nTelp : (021) 5050 7777  Faks : (021) 4060 0009  Email : Info@simasinsurtech.com"},
}

// Render membentuk PDF satu DLA.
func (r Renderer) Render(d registrasi.DLADocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 15, marginLeft)
	pdf.SetAutoPageBreak(true, pageMarginB)
	pdf.SetTitle("DLA "+d.Number, true)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	entity := d.Entity
	if _, ok := footer[entity]; !ok {
		entity = "ASM"
	}
	pdf.SetFooterFunc(func() {
		f := footer[entity]
		pdf.SetY(-26)
		pdf.SetFont(font, "B", 8)
		pdf.CellFormat(0, 4, tr(f[0]), "", 1, "L", false, 0, "")
		pdf.SetFont(font, "", 8)
		pdf.MultiCell(0, 3.8, tr(f[1]), "", "L", false)
	})
	pdf.AddPage()

	if logo := normalizePNG(r.Logo[entity]); len(logo) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("logo", opt, bytes.NewReader(logo))
		pdf.ImageOptions("logo", marginLeft, 12, 55, 0, false, opt, 0, "")
		pdf.SetY(38)
	} else {
		pdf.SetY(30)
	}

	pdf.SetFont(font, "BU", 12)
	pdf.CellFormat(0, 7, "DEFINITE LOSS ADVICE", "", 1, "C", false, 0, "")
	pdf.SetFont(font, "", 9)
	pdf.CellFormat(0, 6, tr("NO : "+d.Number), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	row := func(label, value string) {
		y := pdf.GetY()
		pdf.SetFont(font, "B", labelSize)
		pdf.SetXY(marginLeft, y)
		pdf.MultiCell(labelWidth, lineHeight, tr(label), "", "L", false)
		labelEnd := pdf.GetY()
		pdf.SetXY(marginLeft+labelWidth, y)
		pdf.SetFont(font, "", valueSize)
		pdf.MultiCell(valueWidth, lineHeight, tr(": "+value), "", "L", false)
		if pdf.GetY() < labelEnd {
			pdf.SetY(labelEnd)
		}
		pdf.Ln(1.2)
	}

	recipientLabel := "Reinsurer"
	if d.Type == registrasi.DLATypeCoins {
		recipientLabel = "Coinsurer"
	}
	cur := d.Currency
	row(recipientLabel, d.Recipient)
	row("Line Of Business", strings.ToUpper(d.BusinessName))
	row("Policy No.", d.PolicyNumber)
	row("Claim No.", d.ClaimNumber)
	row("Name of Insured", d.Insured)
	row("Total Sum Insured", money(d.SumInsured.Currency, d.SumInsured.Value))
	row("Location of Loss", d.LossLocation)
	row("Policy Condition", d.PolicyCondition)
	row("Period", moment(d.PeriodStart)+" To "+moment(d.PeriodEnd))
	row("Date of Loss", moment(d.DateOfLoss))
	row("Nature of Loss", d.NatureOfLoss)

	total := totalLabel(d.PaymentType)
	own := " (ASM)"
	if d.Entity == "ASI" {
		own = " (ASI)"
	}
	switch d.Type {
	case registrasi.DLATypeCoins:
		row(total, money(cur, d.Gross))
		row("Your Share", fmt.Sprintf("COAS = %s x %s %% = %s",
			money(cur, d.Gross), decimal(d.Percent), amount(cur, d.Value)))
	case registrasi.DLATypeFacOut:
		row(total, money(cur, d.Gross))
		if total != "" {
			row(total+own, amount(cur, d.ClaimAmount))
		}
		row("Your Share Under "+d.ShareUnder, fmt.Sprintf("%s / %s * %s",
			decimal(d.ShareSpread), decimal(d.Percent), amount(cur, d.ClaimAmount)))
	case registrasi.DLATypeBPPDAN, registrasi.DLATypeEQPool:
		row(total, money(cur, d.Gross))
		if total != "" {
			row(total+own, amount(cur, d.ClaimAmount))
		}
		row("Your Share Under", fmt.Sprintf("%s = %s %% * %s",
			d.ShareUnder, decimal(d.ShareSpread), amount(cur, d.ClaimAmount)))
	default: // TREATY, FAC-OBLIG, FACOBLIGINDT
		row("Total", treatyTotal(d.PaymentType)+" : "+money(cur, d.Gross))
		if total != "" {
			row(total+own, amount(cur, d.ClaimAmount))
		}
		row("Your Share", treatyShare(d))
	}
	due := "Amount due to us"
	if d.PaymentType == registrasi.PaymentSalvage {
		due = "Amount due to you"
	}
	row(due, amount(cur, d.DueTotal))
	row("Remarks", d.Remarks)

	pdf.Ln(6)
	pdf.SetFont(font, "", 9)
	right := 150.0
	pdf.SetX(right)
	pdf.CellFormat(40, 5, tr("Jakarta, "+date(d.Date)), "", 1, "C", false, 0, "")
	if signature := normalizePNG(d.Signature); len(signature) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("ttd", opt, bytes.NewReader(signature))
		pdf.ImageOptions("ttd", right+10, pdf.GetY()+1, 18, 0, false, opt, 0, "")
		pdf.SetY(pdf.GetY() + 25)
	} else {
		pdf.Ln(20)
	}
	pdf.SetFont(font, "B", 9)
	pdf.SetX(right - 5)
	pdf.CellFormat(50, 5, tr(d.SignerName), "", 1, "C", false, 0, "")

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("dlapdf: %w", err)
	}
	return out.Bytes(), nil
}

// totalLabel adalah label baris jumlah menurut tipe pembayaran (`tempObjeCvg.CityID`).
// Tipe 7 tidak punya cabang di template, sehingga labelnya kosong.
func totalLabel(paymentType string) string {
	switch paymentType {
	case registrasi.PaymentSalvage:
		return "Total Salvage"
	case registrasi.PaymentAdjusterFee:
		return "Total Adjuster Fee"
	case registrasi.PaymentReject:
		return "Total Surveyor Fee"
	case registrasi.PaymentFinal, registrasi.PaymentInterim, registrasi.PaymentAdjustment:
		return "Total Claim Amount"
	}
	return ""
}

// treatyTotal adalah CoverageID baris "Total :" template treaty (`TempAdjustment`).
func treatyTotal(paymentType string) string {
	switch paymentType {
	case registrasi.PaymentSalvage:
		return "Salvage"
	case registrasi.PaymentAdjusterFee:
		return "Adjuster Fee"
	}
	return "Claim Amount"
}

// treatyShare adalah baris "Your Share" `DLATREATY_HTML`.
func treatyShare(d registrasi.DLADocument) string {
	cur := d.Currency
	qsri := registrasi.DecimalOf(d.QSRI)
	if qsri.Sign() == 0 {
		parts := []string{}
		if d.ShareUnder != "FACOBLIG" && d.ShareUnder != "FACOBLIGINDT" {
			parts = append(parts, decimal(d.ShareSpread)+"% X")
		}
		parts = append(parts, decimal(d.Percent)+"% X", amount(cur, d.ClaimAmount))
		return d.ShareUnder + " = " + strings.Join(parts, " ")
	}
	return fmt.Sprintf("%s = %s%% X %s%% X %s%% X %s",
		d.ShareUnder, decimal(d.ShareSpread), decimal(d.Percent), decimal(d.QSRI), amount(cur, d.ClaimAmount))
}

func money(currency string, v registrasi.Money) string {
	return strings.TrimSpace(currency + " " + facesheetpdf.FormatMoney(v))
}

// amount mencetak nilai teks desimal (kolom VARCHAR2) dengan pemisah ribuan.
func amount(currency, value string) string {
	return strings.TrimSpace(currency + " " + decimal(value))
}

// decimal memberi pemisah ribuan pada teks desimal tanpa mengubah jumlah desimalnya.
func decimal(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0"
	}
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	whole, frac, hasFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if hasFrac {
		return sign + b.String() + "." + frac
	}
	return sign + b.String()
}

func moment(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format("02/01/06 15:04")
}

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format("02/01/2006")
}

// normalizePNG menyimpan ulang gambar sebagai PNG tanpa interlace (lihat plapdf).
func normalizePNG(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil
	}
	return out.Bytes()
}
