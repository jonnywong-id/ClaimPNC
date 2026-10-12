package registrasi

import (
	"math/big"
	"strings"
	"time"
)

// # Draft Persetujuan — tombol "PRINT" di samping Nomor Akseptasi
//
// Sumbernya:
//
//   - Tombol: `Section/InputAdjustment_sect.xml` — tampil bila `.AcceptedNo != ''`, menjalankan
//     activity `PrintPDFAcceptanceNote` dengan objek, jaminan, adjustment, penerima, komite
//     penyetuju, dan remark akseptasi baris itu.
//   - Isi: template `HTML/AcceptanceNotePDF-html.xml`, dibandingkan dengan contoh
//     `Sample Form/DraftPersetujuan_NO.<nomor>.pdf` (nilainya data nasabah — tidak disalin, `D-69`).
//
// Yang dibawa hanya pembentukan PDF-nya (keputusan 2026-09-30, sama dengan Print DLA: diunduh
// langsung). Langkah lain activity itu TIDAK dijalankan di sini:
//
//   - `InsertJsonClaimNonMBU_act` — prosedur konversi JSON (`D-02`);
//   - `HitServiceOSAkseptasiClaimNonMBU` — layanan Outstanding Acceptance, dilewati sejak
//     akseptasi tahap 1;
//   - `GenerateDLAListAdjustment` — DLA diterbitkan saat daftar Print DLA dibuka;
//   - lampiran `AttachAsPDFC` kategori AcceptanceNote dan email Draft Akseptasi (Travel
//     TRAVELOKASVC) serta `SendEmailAccCollection` (salvage).
//
// Travel dan Personal Accident memakai template yang SAMA (`AcceptanceNotePDF`, tanpa
// percabangan pemilihan template di activity); bedanya ada di blok `pega:when` di dalamnya.
// Personal Accident (`GroupPanel='002'`) dibangun — label "Accident", dua tanda tangan
// `SetSignaturePA` dari POOLDATA.M_SIGNATURE1, tanpa alamat kaki. Travel belum: tabel "Jenis
// Jaminan"-nya diisi `ObjectCoverageDetailList` jaminan (nama, limit, ID rincian), dan daftar
// itu belum dibaca maupun disimpan di mana pun di sistem ini.

// AcceptanceNoteFileName adalah nama berkas Pega: `"DraftPersetujuan_"+"NO."+acceptedNo+".pdf"`.
func AcceptanceNoteFileName(acceptedNo string) string {
	return "DraftPersetujuan_NO." + strings.TrimSpace(acceptedNo) + ".pdf"
}

// Kode pelanggaran Draft Persetujuan. Pesan tambahan berbahasa Inggris (`D-80`).
const (
	ViolationAcceptanceNoteNotAllowed ViolationCode = "draft_akseptasi_tidak_berlaku"
	ViolationAcceptanceNoteLayout     ViolationCode = "draft_akseptasi_tata_letak"
)

// CanPrintAcceptanceNote memeriksa tombol PRINT seperti InputAdjustment_sect: Nomor Akseptasi
// terisi. Tata letak Travel belum tersedia (lihat catatan kepala berkas).
func CanPrintAcceptanceNote(line SettlementLine, p Policy) error {
	if strings.TrimSpace(line.AcceptedNo) == "" {
		return &ValidationError{Violation: []Violation{{Code: ViolationAcceptanceNoteNotAllowed, Field: "akseptasi",
			Message: "The adjustment has no Nomor Akseptasi yet."}}}
	}
	if p.Line == LineTravel || p.BusinessType == "Travel" {
		return &ValidationError{Violation: []Violation{{Code: ViolationAcceptanceNoteLayout, Field: "akseptasi",
			Message: "The Draft Persetujuan layout for Travel is not available yet: its Jenis Jaminan table needs the coverage detail list, which this system does not read yet."}}}
	}
	return nil
}

// AcceptanceNotePaymentLabel adalah teks miring di kanan atas — `TempAcceptedNo.PaymentData.pyNote`.
func AcceptanceNotePaymentLabel(paymentType string) string {
	switch strings.TrimSpace(paymentType) {
	case PaymentFinal:
		return "Final"
	case PaymentInterim:
		return "Interim"
	case PaymentSalvage:
		return "Salvage"
	case PaymentAdjusterFee:
		return "Adjuster Fee"
	case paymentCollectionFee:
		return "Collection Fee"
	case PaymentAdjustment:
		return "Adjustment"
	}
	return ""
}

// StatusBusinessLabel adalah kolom kedua baris Premium Paid On — `Quotation.StatusBusiness`.
func StatusBusinessLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "1":
		return "New Business"
	case "2":
		return "Endorsement"
	case "3":
		return "Renewal"
	}
	return ""
}

// AcceptanceNoteInstallment adalah satu baris Premium Paid On — cicilan
// `PaymentData.Payment.ListInstallment` yang PaymentAmount-nya > 0.
type AcceptanceNoteInstallment struct {
	Number string
	PaidAt time.Time // nol bila tanggalnya tidak terbaca; PaidText dipakai apa adanya
	Paid   string
	Amount *big.Rat
}

// AcceptanceNoteSpread adalah satu baris spreading: nama treaty, share, dan share × nilai.
type AcceptanceNoteSpread struct {
	Name   string
	Share  Percent
	Amount *big.Rat
}

// AcceptanceNoteQS adalah rincian QS (OR) / QS (R/I) — `searchQSReins2_SQL`.
type AcceptanceNoteQS struct {
	Name   string
	Amount *big.Rat
}

// AcceptanceNoteReceiver adalah blok Payable To / Receive From — `TempReceiver`.
type AcceptanceNoteReceiver struct {
	Name, Bank, Branch, AccountNo string
}

// AcceptanceNote adalah isi satu Draft Persetujuan: tata letak umum, atau Personal Accident
// bila PersonalAccident bernilai benar.
type AcceptanceNote struct {
	Entity string // ASM atau ASI
	// PersonalAccident memilih blok `GroupPanel='002'` template: judul dan label "Accident",
	// dua tanda tangan PASigners, tanpa alamat perusahaan di kaki halaman.
	PersonalAccident bool
	AcceptedNo       string
	PaymentLabel     string

	PolicyNumber    string
	ClaimNumber     string
	InsuredName     string
	PolicyCurrency  string
	SumInsured      Money // SumTSI jaminan
	LossLocation    string
	PolicyCondition string // CoverageNote jaminan
	ObjectName      string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	DateOfLoss      time.Time
	NatureOfLoss    string

	StatusBusiness string
	Installments   []AcceptanceNoteInstallment

	// Amounts: Currency adalah kode mata uang adjustment. Gross dan Own menjadi baris
	// "Claim Accepted" / "(ASM)" untuk tipe selain 3, 4, 7; baris Salvage atau Adjuster Fee
	// untuk tipe 3 atau 4/7 (Pega mengubah tipe 7 menjadi 4 sebelum template membacanya,
	// sehingga blok Collection Fee tidak pernah tampil).
	Currency    string
	PaymentType string
	Gross       Money
	Own         Money

	Spread []AcceptanceNoteSpread
	QS     []AcceptanceNoteQS

	Receiver AcceptanceNoteReceiver
	Remark   string

	SignedAt   time.Time // AcceptedDate — "Jakarta, dd MMMM yyyy"
	SignerName string    // KomiteAccepted
	SignerID   string    // Tempinput.BranchID — ukuran gambar BAMBANGSG berbeda
	Signature  []byte

	// PASigners adalah dua tanda tangan blok PA (`SetSignaturePA`), kiri lalu kanan.
	PASigners []AcceptanceNoteSigner
}

// AcceptanceNoteSigner adalah satu tanda tangan blok PA: gambar, nama, dan jabatannya.
type AcceptanceNoteSigner struct {
	Name      string
	Title     string
	Signature []byte
}

// AcceptanceNotePATitles adalah jabatan di bawah kedua tanda tangan blok PA — teks template.
var AcceptanceNotePATitles = [2]string{"Accident & Health Ins. Claim Dept.Head", "Claim Section Head"}

// AcceptanceNotePASignatureIDs adalah `SetSignaturePA`: SIGNATURE_ID POOLDATA.M_SIGNATURE1
// kedua tanda tangan PA per entitas; ID kosong berarti tanda tangan itu tidak dibaca.
//
// ASI (SIMASNET) di Pega memakai 60792 untuk tanda tangan pertama dan tanda tangan PENGGUNA
// YANG LOGIN untuk yang kedua (`BrowseSignatureNMBU` per login). Yang kedua belum dibawa —
// kolomnya tercetak tanpa gambar dan nama.
//
// PENGECUALIAN `D-15` yang disadari, sama dengan tanda tangan PLA, DLA, dan Draft
// Persetujuan umum: ID penanda tangan tertanam di rule Pega, belum ada master per entitas.
func AcceptanceNotePASignatureIDs(entity string) [2]string {
	if strings.TrimSpace(entity) == "ASI" {
		return [2]string{"60792", ""}
	}
	return [2]string{"00924", "60018"}
}

// AcceptanceNoteBlock menyatakan blok nilai yang dicetak menurut tipe pembayaran.
func (n AcceptanceNote) AcceptanceNoteBlock() string {
	switch strings.TrimSpace(n.PaymentType) {
	case PaymentSalvage:
		return "Salvage"
	case PaymentAdjusterFee, paymentCollectionFee:
		return "Adjuster Fee"
	}
	return "Claim Accepted"
}

// AcceptanceNoteSpreading menghitung baris spreading (langkah 72): share × nilai / 100, dan
// untuk treaty QS (10003, 10019) rincian per REINSTYPENAME: share × PCT × nilai / 100.
// Nilai adalah AdjustmentValue, atau AdjusterFeeValue untuk tipe 4 dan 7 — keduanya disimpan
// sebagai ASM_SHARE_VALUE di model ini. Tanpa pembulatan (`I-12`).
func AcceptanceNoteSpreading(spreading []Spreading, value Money, qs map[string][]TreatyQSPart) ([]AcceptanceNoteSpread, []AcceptanceNoteQS) {
	base := div(moneyRat(value), big.NewRat(100, 1))
	var spread []AcceptanceNoteSpread
	var parts []AcceptanceNoteQS
	for _, s := range spreading {
		if s.Removed {
			continue
		}
		share := percentRat(s.Share)
		spread = append(spread, AcceptanceNoteSpread{Name: s.Name, Share: s.Share, Amount: mul(share, base)})
		for _, q := range qs[s.TreatyKind] {
			parts = append(parts, AcceptanceNoteQS{Name: q.Name, Amount: mul(mul(share, decimalOf(q.Pct)), base)})
		}
	}
	return spread, parts
}

// AcceptanceNoteSignerID memilih tanda tangan `SetSignatureNonMBU` kategori "IsAkseptasi":
// sama dengan kategori DLA kecuali DHARMANTO, yang hanya berlaku untuk PLA dan DLA.
//
// PENGECUALIAN `D-15` yang disadari, sama dengan tanda tangan PLA dan DLA: aturannya tertanam
// nama orang di rule Pega, dan belum ada master penanda tangan per komite.
func AcceptanceNoteSignerID(committee string) string {
	name := strings.ToUpper(committee)
	id := ""
	if strings.Contains(name, "BAMBANG") || strings.Contains(name, "LINDA") {
		id = "BAMBANGSG"
	}
	if strings.Contains(name, "ELLEN") {
		id = "ELLENSP"
	}
	if strings.Contains(name, "LINDA") {
		id = "LINDANOVA"
	}
	if strings.Contains(name, "INDRA") {
		id = "INDRAGN"
	}
	return id
}

// AcceptanceNoteRenderer mengubah AcceptanceNote menjadi PDF.
type AcceptanceNoteRenderer interface {
	Render(n AcceptanceNote) ([]byte, error)
}
