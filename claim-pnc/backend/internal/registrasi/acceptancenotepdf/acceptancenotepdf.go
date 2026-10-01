// Package acceptancenotepdf membentuk PDF Draft Persetujuan ("ACCEPTED CLAIM INSURANCE") —
// template Pega `AcceptanceNotePDF`, tata letak umum (bukan Travel, bukan Personal Accident).
//
// Susunannya mengikuti template dan contoh `Sample Form/DraftPersetujuan_NO.<nomor>.pdf`:
// judul dan Accepted No di tengah, tipe pembayaran miring di kanan, daftar label–nilai, baris
// spreading dan rincian QS, penerima, remark, lalu "Jakarta, <tanggal akseptasi>", tanda tangan,
// nama komite penyetuju, dan catatan beserta alamat perusahaan di kaki halaman.
//
// Kop logo belum dipasang — sama dengan PDF DLA (keputusan "PDF tanpa logo dulu").
package acceptancenotepdf

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // tanda tangan POOLDATA.MTTD dapat tersimpan sebagai GIF
	_ "image/jpeg" // … atau JPEG; keduanya diubah menjadi PNG sebelum dipasang
	"image/png"
	"math/big"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Renderer memenuhi registrasi.AcceptanceNoteRenderer.
type Renderer struct{}

var _ registrasi.AcceptanceNoteRenderer = Renderer{}

const (
	font       = "Times"
	marginLeft = 18.0
	labelWidth = 36.0
	colonWidth = 5.0
	lineHeight = 5.2
	textSize   = 10.0
)

// company adalah nama dan alamat kaki halaman per entitas — blok footer template.
var company = map[string][2]string{
	"ASM": {"PT. Asuransi Sinar Mas",
		"Plaza Simas, Jl.KH.Fachrudin no.18 Jakarta Pusat 10250-Indonesia\n24 Hour Customer Care:(021)-235-67-888; Telp:(021)-390-2141 (Hunting); Faks:(021)-390-2159/60, www.sinarmas.co.id"},
	"ASI": {"PT. Asuransi Simas Insurtech",
		"Gedung Menara Tekno Lantai 5, Jl. K. H. Fachrudin No. 19, Jakarta, 10250 - Indonesia\nTelp : (021) 5050 7777  Faks : (021) 4060 0009  Email : Info@simasinsurtech.com"},
}

// Render membentuk PDF satu Draft Persetujuan.
func (Renderer) Render(n registrasi.AcceptanceNote) ([]byte, error) {
	entity := n.Entity
	if _, ok := company[entity]; !ok {
		entity = "ASM"
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, 15, marginLeft)
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetTitle("Draft Persetujuan "+n.AcceptedNo, true)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetY(28)

	pdf.SetFont(font, "B", 14)
	pdf.CellFormat(0, 7, "ACCEPTED CLAIM INSURANCE", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, tr("Accepted No : "+n.AcceptedNo), "", 1, "C", false, 0, "")
	pdf.SetFont(font, "I", 9)
	pdf.CellFormat(0, 5, tr(n.PaymentLabel), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	valueX := marginLeft + labelWidth + colonWidth
	valueWidth := 210 - marginLeft - valueX
	row := func(label, value string) {
		y := pdf.GetY()
		pdf.SetFont(font, "", textSize)
		pdf.SetXY(valueX, y)
		pdf.MultiCell(valueWidth, lineHeight, tr(value), "", "L", false)
		end := pdf.GetY()
		// Label dan titik dua berada di tengah tinggi nilai, seperti sel tabel HTML.
		middle := y + (end-y-lineHeight)/2
		pdf.SetXY(marginLeft, middle)
		pdf.CellFormat(labelWidth, lineHeight, tr(label), "", 0, "L", false, 0, "")
		pdf.CellFormat(colonWidth, lineHeight, ":", "", 0, "L", false, 0, "")
		pdf.SetY(end + 0.8)
	}

	row("Policy Number", n.PolicyNumber)
	row("Claim Number", n.ClaimNumber)
	row("Name of Insured", n.InsuredName)
	row("Sum Insured", join(n.PolicyCurrency, number(money(n.SumInsured))))
	row("Location of Loss", n.LossLocation)
	row("Policy Condition", n.PolicyCondition)
	row("Object", n.ObjectName)
	row("Policy Period", date(n.PeriodStart)+" s/d "+date(n.PeriodEnd))
	row("Date of Loss", date(n.DateOfLoss))
	row("Nature Of Loss", n.NatureOfLoss)

	// Premium Paid On: satu baris per cicilan; label dan titik dua hanya pada baris pertama.
	pdf.Ln(1)
	pdf.SetFont(font, "", textSize)
	pdf.SetX(marginLeft)
	pdf.CellFormat(labelWidth, lineHeight, "Premium Paid On", "", 0, "L", false, 0, "")
	pdf.CellFormat(colonWidth, lineHeight, ":", "", 0, "L", false, 0, "")
	status := registrasi.StatusBusinessLabel(n.StatusBusiness)
	if len(n.Installments) == 0 {
		pdf.Ln(lineHeight + 0.8)
	}
	for i, inst := range n.Installments {
		if i > 0 {
			pdf.SetX(valueX)
		}
		pdf.CellFormat(6, lineHeight, tr(inst.Number), "", 0, "L", false, 0, "")
		pdf.CellFormat(18, lineHeight, tr(status), "", 0, "L", false, 0, "")
		paid := inst.Paid
		if !inst.PaidAt.IsZero() {
			paid = inst.PaidAt.In(clock.ZoneWIB).Format("02/01/06 15:04")
		}
		pdf.CellFormat(28, lineHeight, tr(paid), "", 0, "L", false, 0, "")
		pdf.CellFormat(10, lineHeight, tr(n.PolicyCurrency), "", 0, "L", false, 0, "")
		pdf.CellFormat(40, lineHeight, number(inst.Amount), "", 1, "L", false, 0, "")
		pdf.Ln(0.8)
	}
	pdf.Ln(1)

	block := n.AcceptanceNoteBlock()
	own := "(ASM)"
	if entity == "ASI" {
		own = "(ASI)"
	}
	switch block {
	case "Salvage":
		row("Salvage", join(n.Currency, number(money(n.Gross))))
		row("Salvage "+own, join(n.Currency, number(money(n.Own))))
	case "Adjuster Fee":
		row("Adjuster Fee", join(n.Currency, number(money(n.Gross))))
		row("Adjuster Fee"+own, join(n.Currency, number(money(n.Own))))
	default:
		row("Claim Accepted", join(n.Currency, number(money(n.Gross))))
		row("Claim Accepted"+own, join(n.Currency, number(money(n.Own))))
	}

	// Spreading: nama treaty (huruf besar), share, dan bagiannya — di kolom nilai.
	pdf.SetFont(font, "", textSize)
	for _, s := range n.Spread {
		pdf.SetX(valueX)
		pdf.CellFormat(28, lineHeight, tr(strings.ToUpper(s.Name)), "", 0, "L", false, 0, "")
		pdf.CellFormat(24, lineHeight, number(big.NewRat(int64(s.Share), 10_000))+" %", "", 0, "R", false, 0, "")
		pdf.CellFormat(6, lineHeight, "", "", 0, "L", false, 0, "")
		pdf.CellFormat(60, lineHeight, tr(join(n.Currency, number(s.Amount))), "", 1, "L", false, 0, "")
	}
	// Rincian QS — di margin kiri, seperti tabel kedua template.
	for _, q := range n.QS {
		pdf.SetX(marginLeft - 2)
		pdf.CellFormat(labelWidth+2, lineHeight, tr(q.Name), "", 0, "L", false, 0, "")
		pdf.CellFormat(14, lineHeight, tr(n.Currency), "", 0, "L", false, 0, "")
		pdf.CellFormat(40, lineHeight, number(q.Amount), "", 1, "L", false, 0, "")
	}
	pdf.Ln(1.5)

	receiverLabel := "Payable To"
	if strings.TrimSpace(n.PaymentType) == registrasi.PaymentSalvage {
		receiverLabel = "Receive From"
	}
	receiver := []string{n.Receiver.Name, n.Receiver.Bank, "Cabang : " + n.Receiver.Branch, "No Rek : " + n.Receiver.AccountNo}
	row(receiverLabel, strings.Join(receiver, "\n"))
	row("Remark", n.Remark)

	// Tanda tangan di kanan: tanggal akseptasi, gambar, nama komite penyetuju.
	pdf.Ln(8)
	right := 138.0
	pdf.SetFont(font, "", textSize)
	pdf.SetX(right)
	pdf.CellFormat(60, lineHeight, tr("Jakarta, "+longDate(n.SignedAt)), "", 1, "C", false, 0, "")
	width, height := 32.0, 21.0 // 120×80 px
	if n.SignerID == "BAMBANGSG" {
		width, height = 26.5, 40.0 // 100×150 px
	}
	if signature := normalizeImage(n.Signature); len(signature) > 0 {
		opt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
		pdf.RegisterImageOptionsReader("ttd", opt, bytes.NewReader(signature))
		pdf.ImageOptions("ttd", right+30-width/2, pdf.GetY()+1, width, height, false, opt, 0, "")
	}
	pdf.SetY(pdf.GetY() + height + 2)
	pdf.SetFont(font, "B", textSize)
	pdf.SetX(right)
	pdf.CellFormat(60, lineHeight, tr(n.SignerName), "", 1, "C", false, 0, "")

	// Catatan dan alamat perusahaan.
	c := company[entity]
	pdf.Ln(12)
	pdf.SetFont(font, "B", 9)
	pdf.MultiCell(0, 4.2, tr("Catatan :\nSurat ini sah dikeluarkan oleh "+c[0]+" dan tercatat pada sistem "+c[0]), "", "L", false)
	pdf.Ln(4)
	pdf.CellFormat(0, 4.2, tr(c[0]), "", 1, "L", false, 0, "")
	pdf.SetFont(font, "", 9)
	pdf.MultiCell(0, 4.2, tr(c[1]), "", "L", false)

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("acceptancenotepdf: %w", err)
	}
	return out.Bytes(), nil
}

func join(currency, value string) string { return strings.TrimSpace(currency + " " + value) }

func money(v registrasi.Money) *big.Rat { return big.NewRat(int64(v), 100) }

// number meniru @NumberAddSeparator pada contoh: pemisah ribuan titik, desimal koma, paling
// banyak tiga angka di belakang koma, dan nol di belakang dibuang ("201.713,8", "2.000.000").
func number(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := r.FloatString(3)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	whole, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if frac != "" {
		return sign + b.String() + "," + frac
	}
	return sign + b.String()
}

func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format("02/01/2006")
}

var months = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus",
	"September", "Oktober", "November", "Desember"}

// longDate adalah "dd MMMM yyyy" berlokal in_ID — nama bulan bahasa Indonesia.
func longDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	w := t.In(clock.ZoneWIB)
	return fmt.Sprintf("%02d %s %d", w.Day(), months[w.Month()-1], w.Year())
}

// normalizeImage menyimpan ulang gambar tanda tangan sebagai PNG tanpa interlace.
func normalizeImage(b []byte) []byte {
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
