package registrasi

import (
	"math/big"
	"strings"
	"time"
)

// Letter of Discharge (LOD) — tombol "Print LOD" pada grid Adjustment & Akseptasi.
//
// # Sumbernya
//
//   - Tombol: `Section/ShowAdjustment_sect.xml` — tampil bila `.AcceptanceStatus == 1`, mati bila
//     `.AcceptationStatusLOD != ''`, Bonding/BondingKBG, Group Panel 002/005, atau Type 3/4.
//   - Flow action `PrintLODUP`: section `PrintLODdanEmail`, pra-proses `SetDataEmailTertanggung`.
//     KEDUANYA tidak ada di export; mengirim LOD ke email tertanggung karena itu belum dibangun.
//   - Jenis LOD: daftar `SetTypePDFAdjustment` (16 jenis, `.PDFType` = `.IDAdjustClaim`).
//     `T_CLAIM_ADJUSTMENT.PDFTYPE` kosong pada seluruh 47.504 baris — Pega menyimpannya di
//     clipboard — sehingga jenisnya dipilih pada dialog Print LOD, lalu disimpan ke PDFTYPE saat
//     dicetak agar form akseptasi dapat menampilkannya (Pega menampilkannya dari clipboard).
//   - Dokumen: template Pega-nya tidak ada di export; susunannya mengikuti 15 contoh PDF
//     dari Work Owner di `Sample Form/` (2026-09-29 dan 2026-09-30), satu per jenis.
//     Jenis 16 "Pembayaran Final" satu-satunya yang tidak punya contoh dan tetap ditolak.

// LODType adalah satu jenis LOD — `.IDAdjustClaim` dan `.PDFType` pada `tempPDFType`.
type LODType struct {
	ID   string
	Name string
}

// Jenis LOD yang perilakunya dibedakan di luar isi suratnya.
const (
	LODPropertyFinalCoMember = "3"
	LODMarineHull            = "11"
	LODFinalPayment          = "16" // tidak ada contoh PDF — template belum dibangun
)

// lodTypes disalin dari `SetTypePDFAdjustment` langkah 20, 24, dan 25 (urutan Pega: 1–11,
// 13, 14, lalu 16, lalu 12, 15). Langkah 21 (IsFire) dan 22 (IsMarineCargo) hanya bagian dari
// daftar langkah 20; langkah 23 (IsMarineHull) menunjuk When yang tidak ada di export.
var lodTypes = []LODType{
	{"1", "Property - Interim dengan Co Member"},
	{"2", "Property - Interim tanpa Co Member"},
	{"3", "Property - Final dengan Co Member"},
	{"4", "Property - Final tanpa Co Member"},
	{"5", "Property - Final Reinstatement tanpa Co Member"},
	{"6", "Properti - Pembayaran Indemnity dan Setelah Tertanggung Reinstatement"},
	{"7", "Marine Cargo - Interim dengan Co Member"},
	{"8", "Marine Cargo - Interim tanpa Co Member"},
	{"9", "Marine Cargo - Final dengan Co Member"},
	{"10", "Marine Cargo - Final tanpa Co Member"},
	{"11", "Marine Hull - LOD"},
	{"13", "Property - Indemnity Terlebih Dahulu Dengan Member"},
	{"14", "Property - Indemnity Terlebih Dahulu Tanpa Member"},
	{"16", "Pembayaran Final"},
	{"12", "Pembayaran Final Tanpa Co Member Ex Gratia"},
	{"15", "Pembayaran Final dengan Co Member Ex Gratia"},
}

// LODTypeName adalah nama jenis LOD; kosong bila jenisnya tidak dikenal.
func LODTypeName(id string) string {
	for _, t := range lodTypes {
		if t.ID == strings.TrimSpace(id) {
			return t.Name
		}
	}
	return ""
}

// LODTypesFor adalah pilihan jenis LOD sebuah polis. Seluruh langkah `SetTypePDFAdjustment`
// bersyarat `IsNotTravelPA` (Group Panel bukan 002 dan bukan 005); PA dan Travel tidak punya
// pilihan — dan tombol Print LOD-nya memang mati.
func LODTypesFor(p Policy) []LODType {
	if p.Line == LinePersonalAccident || p.Line == LineTravel {
		return nil
	}
	return append([]LODType(nil), lodTypes...)
}

// LODTemplateReady menyatakan jenis LOD sudah punya template: seluruh jenis yang dikenal
// kecuali 16, yang contoh PDF-nya tidak ada.
func LODTemplateReady(id string) bool {
	id = strings.TrimSpace(id)
	if id == LODFinalPayment {
		return false
	}
	for _, t := range lodTypes {
		if t.ID == id {
			return true
		}
	}
	return false
}

// LODPrintsInsuredAddress menyatakan jenis LOD mengisi Alamat penanda tangan dengan alamat
// tertanggung — hanya kedua jenis Ex Gratia (12 dan 15), seperti contohnya.
func LODPrintsInsuredAddress(id string) bool {
	id = strings.TrimSpace(id)
	return id == "12" || id == "15"
}

// LODEmailDraft menyusun isian Email LOD seperti `SetDataEmailTertanggung` langkah 1:
// `ClaimData.Email + "," + ClaimData.UserTeknisEmail`. Bagian yang kosong dilewati agar
// isian tidak diawali atau diakhiri koma kosong.
func LODEmailDraft(insured, pic string) string {
	var parts []string
	for _, v := range []string{insured, pic} {
		if v = strings.TrimSpace(v); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ",")
}

// Kode dan pesan pelanggaran Print LOD. Pesan tambahan berbahasa Inggris (`D-80`).
const (
	ViolationLODNotAccepted     ViolationCode = "lod_belum_akseptasi_komite"
	ViolationLODNotAllowed      ViolationCode = "lod_tidak_berlaku"
	ViolationLODTypeUnknown     ViolationCode = "lod_tipe_pdf"
	ViolationLODTemplateMissing ViolationCode = "lod_template_belum_ada"
)

func lodViolation(code ViolationCode, message string) error {
	return &ValidationError{Violation: []Violation{{Code: code, Field: "lod", Message: message}}}
}

// ErrLODTypeUnknown dan ErrLODTemplateMissing dipakai usecase saat memeriksa pilihan Tipe PDF.
func ErrLODTypeUnknown() error {
	return lodViolation(ViolationLODTypeUnknown, "Choose a Tipe PDF for this policy.")
}

func ErrLODTemplateMissing() error {
	return lodViolation(ViolationLODTemplateMissing,
		"The template for this Tipe PDF is not available yet: no sample PDF has been provided for Pembayaran Final.")
}

// CanPrintLOD memeriksa tombol Print LOD persis seperti `ShowAdjustment_sect`.
func CanPrintLOD(line SettlementLine, p Policy) error {
	if strings.TrimSpace(line.AcceptanceStatus) != DecisionApprove {
		return lodViolation(ViolationLODNotAccepted, "The adjustment has not been approved by the committee.")
	}
	bonding := p.BusinessType == "Bonding" || p.BusinessType == "BondingKBG"
	if strings.TrimSpace(line.AcceptanceLODStatus) != "" || bonding ||
		p.Line == LinePersonalAccident || p.Line == LineTravel ||
		line.PaymentType == PaymentSalvage || line.PaymentType == PaymentAdjusterFee {
		return lodViolation(ViolationLODNotAllowed, "Print LOD is not available for this adjustment.")
	}
	return nil
}

// LODMember adalah satu baris tabel co member: nama, share, dan bagiannya.
type LODMember struct {
	Name    string
	Percent Percent
	Value   *big.Rat // rupiah, TANPA pembulatan — dibulatkan saat ditampilkan (`I-12`)
}

// LODFacts adalah isian klaim dan polis yang dipakai surat LOD.
type LODFacts struct {
	Currency     string // kode mata uang, mis. "IDR"
	PolicyNumber string
	DateOfLoss   time.Time

	// LossLocation adalah lokasi kejadian — T_CLAIM_PNC.LOCATION; dipakai jenis interim,
	// indemnity, dan reinstatement ("… pada tanggal … di <lokasi>").
	LossLocation string

	// InsuredName adalah QQName polis (cadangan TheInsured); dipakai LOD Marine Hull.
	InsuredName string

	// InsuredAddress adalah `.Policy.DeliveryAddressList(1).ASMAddress`; dipakai isian Alamat
	// LOD Ex Gratia. Terverifikasi 2026-09-30: alamat pada contoh sama dengan jalur ini (dan
	// CIFData.AddressList(1).ASMAddress), bukan lokasi kejadian.
	InsuredAddress string
}

// LODDocument adalah isi satu LOD.
type LODDocument struct {
	Type    string // `.PDFType` — menentukan template
	Amount  Money  // GROSSVALUE baris — "Nett Of Deductible and Salvage", 100%
	Members []LODMember
	LODFacts
	PrintedAt time.Time
}

// LODInsurer adalah baris tabel co member bila CoinsList polis kosong. Terverifikasi dari
// contoh jenis "dengan Co Member" atas polis tanpa CoinsList (2026-09-30): tabelnya berisi
// satu baris "PT. Asuransi Sinar Mas 100 %" senilai seluruh LOD.
const LODInsurer = "PT. Asuransi Sinar Mas"

// BuildLOD menyusun isi LOD: nilai = GROSSVALUE baris (pada contoh 77.187.648,3 sama dengan
// jumlah seluruh bagian anggota), bagian tiap anggota = nilai × share, urut CoinsList polis.
// Anggota bertanda hapus dilewati; tanpa anggota sama sekali tabelnya satu baris LODInsurer
// 100%. Jenis "tanpa Co Member" tidak mencetak tabel.
func BuildLOD(line SettlementLine, typeID string, members []PLACoinsMember, facts LODFacts, now time.Time) LODDocument {
	doc := LODDocument{Type: strings.TrimSpace(typeID), Amount: line.Gross, LODFacts: facts, PrintedAt: now}
	gross := new(big.Rat).SetFrac(big.NewInt(int64(line.Gross)), big.NewInt(100))
	for _, m := range members {
		if m.Deleted || strings.TrimSpace(m.Name) == "" {
			continue
		}
		v := new(big.Rat).Mul(gross, new(big.Rat).SetFrac(big.NewInt(int64(m.Share)), big.NewInt(int64(PercentFull))))
		doc.Members = append(doc.Members, LODMember{Name: strings.TrimSpace(m.Name), Percent: m.Share, Value: v})
	}
	if len(doc.Members) == 0 {
		doc.Members = []LODMember{{Name: LODInsurer, Percent: PercentFull, Value: gross}}
	}
	return doc
}

// LODRenderer mengubah LODDocument menjadi PDF.
type LODRenderer interface {
	Render(d LODDocument) ([]byte, error)
}

// Terbilang menulis nilai uang dalam kata-kata bahasa Indonesia, seperti contoh LOD:
// 77.187.648,30 → "tujuh puluh tujuh juta seratus delapan puluh tujuh ribu enam ratus empat
// puluh delapan koma tiga puluh rupiah". Sen dibaca sebagai bilangan dua angka; tanpa sen,
// bagian "koma" tidak ditulis. currencyWord adalah kata mata uang di akhir ("rupiah").
func Terbilang(v Money, currencyWord string) string {
	n := int64(v)
	if n < 0 {
		n = -n
	}
	whole, cents := n/100, n%100
	words := spell(whole)
	if whole == 0 {
		words = "nol"
	}
	if cents > 0 {
		words += " koma " + spell(cents)
	}
	if w := strings.TrimSpace(currencyWord); w != "" {
		words += " " + w
	}
	return words
}

var digitWords = []string{"", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh", "delapan", "sembilan",
	"sepuluh", "sebelas"}

func spell(n int64) string {
	switch {
	case n < 12:
		return digitWords[n]
	case n < 20:
		return digitWords[n-10] + " belas"
	case n < 100:
		return join(digitWords[n/10]+" puluh", spell(n%10))
	case n < 200:
		return join("seratus", spell(n-100))
	case n < 1000:
		return join(digitWords[n/100]+" ratus", spell(n%100))
	case n < 2000:
		return join("seribu", spell(n-1000))
	case n < 1_000_000:
		return join(spell(n/1000)+" ribu", spell(n%1000))
	case n < 1_000_000_000:
		return join(spell(n/1_000_000)+" juta", spell(n%1_000_000))
	case n < 1_000_000_000_000:
		return join(spell(n/1_000_000_000)+" miliar", spell(n%1_000_000_000))
	default:
		return join(spell(n/1_000_000_000_000)+" triliun", spell(n%1_000_000_000_000))
	}
}

func join(a, b string) string {
	if b == "" {
		return a
	}
	return a + " " + b
}
