// Package plapdf membentuk dokumen PDF Preliminary Loss Advice.
//
// Susunannya mengikuti HTML `PLAHTML` (kelas Work-PNC) dan contoh PDF produksi dari Work
// Owner: kop, judul bergaris bawah, nomor PLA, daftar label–nilai, "Jakarta, tanggal",
// tanda tangan, nama penanda tangan, lalu alamat perusahaan di kaki halaman.
package plapdf

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

// Renderer memenuhi registrasi.PLARenderer. Logo adalah PNG kop per entitas (ASM, ASI);
// tanpa berkas logo kop dicetak tanpa gambar.
type Renderer struct {
	Logo map[string][]byte
}

var _ registrasi.PLARenderer = Renderer{}

const (
	font        = "Helvetica"
	marginLeft  = 22.0
	labelWidth  = 52.0
	valueWidth  = 115.0
	lineHeight  = 6.0
	labelSize   = 11.0
	valueSize   = 9.0
	pageMarginB = 30.0
)

// footer adalah alamat kaki halaman per entitas — `PLAHTML` blok footer.
var footer = map[string][2]string{
	"ASM": {"PT Asuransi Sinar Mas",
		"Plaza Simas, Jl.KH.Fachrudin no.18 Jakarta Pusat 10250-Indonesia\n24 Hour Customer Care:(021)-235-67-888; Telp:(021)-390-2141 (Hunting); Faks:(021)-390-2159/60,\nwww.sinarmas.co.id"},
	"ASI": {"PT ASURANSI SIMAS INSURTECH",
		"Gedung Menara Tekno Lantai 5, Jl. K. H. Fachrudin No. 19, Jakarta, 10250 - Indonesia\nTelp : (021) 5050 7777  Faks : (021) 4060 0009  Email : Info@simasinsurtech.com"},
}

// Render membentuk PDF satu PLA.
func (r Renderer) Render(d registrasi.PLADocument) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 15, marginLeft)
	pdf.SetAutoPageBreak(true, pageMarginB)
	pdf.SetTitle("PLA "+d.Number, true)
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
	pdf.CellFormat(0, 7, "PRELIMINARY LOSS ADVICE", "", 1, "C", false, 0, "")
	pdf.SetFont(font, "", 8)
	pdf.CellFormat(0, 6, tr(d.Number), "", 1, "C", false, 0, "")
	pdf.Ln(3)

	row := func(label, value string, bold bool) {
		y := pdf.GetY()
		pdf.SetFont(font, "B", labelSize)
		pdf.SetXY(marginLeft, y)
		pdf.MultiCell(labelWidth, lineHeight, tr(label), "", "L", false)
		labelEnd := pdf.GetY()
		pdf.SetXY(marginLeft+labelWidth, y)
		if bold {
			pdf.SetFont(font, "B", labelSize)
		} else {
			pdf.SetFont(font, "", valueSize)
		}
		pdf.MultiCell(valueWidth, lineHeight, tr(": "+value), "", "L", false)
		if pdf.GetY() < labelEnd {
			pdf.SetY(labelEnd)
		}
		pdf.Ln(1.5)
	}

	// PLA reasuransi (PLAHTML_FACOUT, PLAHTML_BPPDAN): penerimanya "Reinsurer".
	facOut := d.Type == registrasi.PLATypeFacOut
	bppdan := d.Type == registrasi.PLATypeBPPDAN || d.Type == registrasi.PLATypeEQPool
	recipientLabel := "Coinsurer"
	if facOut || bppdan {
		recipientLabel = "Reinsurer"
	}
	row(recipientLabel, d.Recipient, false)
	row("Line Of Business", strings.ToUpper(d.BusinessName), true)
	row("Policy No.", d.PolicyNumber, false)
	row("Claim No.", d.ClaimNumber, false)
	row("Name of Insured", d.Insured, false)
	if strings.TrimSpace(d.Interest) != "" {
		row("Interest", d.Interest, false)
	}
	row("Total Sum Insured", amount(d.SumInsured.Currency, d.SumInsured.Value), false)
	row("Policy Period", moment(d.PeriodStart)+" To "+moment(d.PeriodEnd), false)
	row("Policy Condition", d.PolicyCondition, false)
	row("Date of Loss", moment(d.DateOfLoss), false)
	row("Nature of Loss", d.NatureOfLoss, false)
	row("Location of Loss", strings.ToUpper(d.LossLocation), false)

	var reserve, share []string
	for _, a := range d.Amount {
		if a.Reserve > 0 {
			reserve = append(reserve, amount(a.Currency, a.Reserve))
		}
		if a.Result > 0 && bppdan {
			// PLAHTML_BPPDAN: SharePLA % X <mata uang> EstimationValue = <mata uang> ResultPLA
			share = append(share, fmt.Sprintf("%s %% X %s = %s",
				facesheetpdf.FormatMoney(a.FacShare), amount(a.Currency, a.Base), amount(a.Currency, a.Result)))
			continue
		}
		if a.Result > 0 && facOut {
			// PLAHTML_FACOUT: SharePLA / PercentPLA X <mata uang> EstimationValue = <mata uang> ResultPLA
			share = append(share, fmt.Sprintf("%s / %s X %s = %s",
				facesheetpdf.FormatMoney(a.FacShare), facesheetpdf.FormatMoney(a.FacBase),
				amount(a.Currency, a.Base), amount(a.Currency, a.Result)))
			continue
		}
		if a.Result > 0 {
			share = append(share, fmt.Sprintf("%s X %s%% = %s",
				amount(a.Currency, a.Base), facesheetpdf.FormatPercent(a.Share), amount(a.Currency, a.Result)))
		}
	}
	row("Est. Claim Amount", strings.Join(reserve, "\n"), false)
	shareLabel := "Your Share"
	if facOut || bppdan {
		shareLabel = strings.TrimSpace("Your Share " + d.ShareLabel)
	} else {
		for i, s := range share {
			share[i] = "COAS = " + s
		}
	}
	row(shareLabel, strings.Join(share, "\n"), false)
	row("Remarks", d.Note, false)

	pdf.Ln(6)
	pdf.SetFont(font, "", 9)
	right := 150.0
	place := d.Place
	if place == "" {
		place = "Jakarta"
	}
	pdf.SetX(right)
	pdf.CellFormat(40, 5, tr(place+", "+date(d.Date)), "", 1, "C", false, 0, "")
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
		return nil, fmt.Errorf("plapdf: %w", err)
	}
	return out.Bytes(), nil
}

func amount(currency string, v registrasi.Money) string {
	return strings.TrimSpace(currency + " " + facesheetpdf.FormatMoney(v))
}

// moment mencetak `05/06/25 12:00` dalam WIB, seperti contoh PDF.
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

// normalizePNG menyimpan ulang gambar sebagai PNG tanpa interlace. Tanda tangan di
// POOLDATA.MTTD tersimpan interlaced, dan fpdf tidak dapat membacanya. Gambar yang tidak
// terbaca dilewati — dokumen tetap terbentuk tanpa gambar itu.
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
