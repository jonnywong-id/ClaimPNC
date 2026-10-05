package registrasi

import (
	"strings"
	"time"
)

// # Settlement Line — `AdjustmentList` di Pega
//
// Satu baris nilai penyelesaian pada sebuah jaminan: `.ObjectList(n).ObjectCoverageList(n)
// .AdjustmentList(n)`, disalin ke POOLDATA.T_CLAIM_ADJUSTMENT. Istilah lama "Adjustment"
// menyesatkan (lihat CONTEXT.md); di sini namanya Settlement Line, di layar tetap
// "Adjustment" seperti Pega.
//
// # Dari mana aturannya
//
// Isian barisnya adalah `Section/InputAdjustment_sect.xml`; grid yang memuatnya
// (`InputEstimasi`) tidak ada di export. Hitungan dan pemeriksaannya dari activity:
//
//   - `SetNilaiResikoSendiri`   — LOC, resiko sendiri, nilai final, gross, bagian Sinar Mas
//   - `ValidationTypePaymentAdj` — isian wajib per tipe pembayaran
//   - `SetValueAdjusterFee`      — gross fee adjuster = Professional Fee + Survey Expenses + VAT%
//   - `ValidTotalEstimation`     — jumlah adjustment tidak melebihi estimasi
//
// Rumusnya dicocokkan dengan 47.499 baris T_CLAIM_ADJUSTMENT: ASM_SHARE_VALUE = GROSSVALUE ×
// ASM_SHARE pada 99,9% baris tipe 1/2/5/6, dan GROSSVALUE = TOTAL_CLAIM − LOC − NILAI_SALVAGE_A
// − INDIVIDUAL_RISK_VALUE pada mayoritasnya.

// Kode Tipe Pembayaran (PAYMENTTYPE). Artinya dari `ExportDataKomitesKlaimNONMBU-SQL` dan
// `SetKomiteTerimaTolakBasedLimit-SQL`.
const (
	PaymentFinal       = "1"
	PaymentInterim     = "2"
	PaymentSalvage     = "3"
	PaymentAdjusterFee = "4"
	PaymentAdjustment  = "5"
	PaymentReject      = "6"
)

// PaymentTypeName memberi nama Tipe Pembayaran seperti layar komite Pega.
func PaymentTypeName(code string) string {
	switch code {
	case PaymentFinal:
		return "Final"
	case PaymentInterim:
		return "Interim"
	case PaymentSalvage:
		return "Salvage"
	case PaymentAdjusterFee:
		return "Adjuster Fee"
	case PaymentAdjustment:
		return "Adjustment"
	case PaymentReject:
		return "Tolak Klaim"
	}
	return code
}

// Tipe Resiko Sendiri (INDIVIDUAL_RISK_TYPE), dari `SetNilaiResikoSendiri`: 1 persen dari
// nilai klaim, 2 persen dari TSI, 3 "Lainnya" — nilainya DIISI PETUGAS dan persennya nol
// (section: Persen hanya-baca bila `.IndividualRiskType = 3`). Data Pega: 1.132 baris tipe 3
// bernilai resiko terisi dengan persen nol.
const (
	RiskOfClaim = "1"
	RiskOfTSI   = "2"
	RiskOther   = "3"
)

// SettlementLine adalah satu baris AdjustmentList.
type SettlementLine struct {
	PaymentType string       // PAYMENTTYPE
	Currency    string       // CURRENCY
	Rate        ExchangeRate // CURRENCYVALUE — "Nilai Dalam IDR", kurs tanggal kejadian (`D-48`)

	Propose   Money   // TOTAL_CLAIM — ProposeAdjustmentValue, "Total Klaim"
	Submitted Money   // PROPOSE_VALUE — ProposeValue, "Nilai Pengajuan Tertanggung"
	LOC       Percent // LOC — "Lack Of Document (%)"
	SalvageA  Money   // NILAI_SALVAGE_A — SalvageValueA

	// SalvageB (SalvageValue, "Nilai Salvage B") dan Interim (InterimPayment, "Nilai
	// Interim") mengurangi gross tipe Final dan Interim. Keduanya tidak punya kolom di
	// T_CLAIM_ADJUSTMENT; yang tersimpan hanya gross hasilnya.
	SalvageB Money
	Interim  Money

	RiskType    string  // INDIVIDUAL_RISK_TYPE
	RiskPercent Percent // INDIVIDUAL_RISK_PERCENT
	RiskValue   Money   // INDIVIDUAL_RISK_VALUE

	Gross    Money   // GROSSVALUE — "Nilai Nett Pembayaran"
	ShareASM Percent // ASM_SHARE
	Value    Money   // ASM_SHARE_VALUE — AdjustmentValue, "Nilai Yang Dibayarkan ASM"

	// Accepted adalah NILAIAKSEPTASI: AdjustmentValue bila TYPEOFCOINS '1' atau 'F',
	// selain itu GrossValue (`PEGA_CONVERT_JSONKLAIM_PNC.prc` baris 1142–1152).
	Accepted Money

	// Estimation adalah "Nilai Estimasi" (EstimationValue) pembanding baris ini, dalam mata
	// uang estimasinya. Ia tampilan saja, tidak disimpan.
	Estimation Money

	ExGratia   bool   // EXGRATIA
	Chronology string // CIRCUMCAUSEOFLOSS — kronologi teknis
	Notes      string // NOTES

	// AcceptanceStatus (STATUSAKSEPTASI) dan AcceptedNo (NOAKSEPTASI) kosong sampai baris
	// ditransfer ke komite dan diakseptasi. Baris baru selalu kosong.
	AcceptanceStatus string
	AcceptedNo       string

	// AcceptanceLODStatus adalah STATUSAKSEPTASILOD — `.AcceptationStatusLOD`: kosong sebelum
	// persetujuan LOD, "1" disetujui (Nomor Akseptasi terbit), "0" tidak disetujui. Hanya
	// DIBACA: ia menentukan tombol Print LOD, Persetujuan / Akseptasi, dan Print DLA
	// (`ShowAdjustment_sect`), dan tidak pernah ditulis `adjustment_perbarui`.
	AcceptanceLODStatus string

	// CommitteeCaseID (CASEIDKOMITE) terisi begitu baris ditransfer ke komite; baris yang
	// sudah ditransfer tidak dapat ditransfer ulang (`IsKomiteTransfer := 1`).
	// CommitteeTransferredAt (ANALYST_TFKOMITEDATE, `TanggalComitee`) dan CommitteeDecidedAt
	// (ACCEPTANCE_DATECOMITEE, `AcceptedDateKomite`) — PEGA_CONVERT_JSONKLAIM_PNC.prc
	// baris 1095–1099.
	CommitteeCaseID        string
	CommitteeTransferredAt time.Time
	CommitteeDecidedAt     time.Time

	// CashierTransferredAt (TRANSFER_CASHIER_DATE) dan CashierCaseID (IDCHASIER, `.CaseIDCashier`)
	// terisi setelah Transfer Kasir berhasil. `.TransferCashierStatus` Pega tidak punya kolom;
	// di sini "sudah ditransfer" dibaca dari keduanya.
	CashierTransferredAt time.Time
	CashierCaseID        string

	// Acceptance adalah isian form Persetujuan / Akseptasi (acceptance.go). Dibaca dan
	// ditulis terpisah dari kolom adjustment lain — tujuh kolomnya baru ada setelah migrasi
	// 0013 dijalankan DBA, dan klaim tanpa akseptasi tidak boleh ikut gagal dimuat.
	Acceptance Acceptance

	CreatedAt time.Time
}

// Transferred menyatakan baris sudah ditransfer ke komite.
func (s SettlementLine) Transferred() bool { return strings.TrimSpace(s.CommitteeCaseID) != "" }

// AdjusterFee adalah komponen fee adjuster (`SetValueAdjusterFee`). Tidak satu pun punya
// kolom di T_CLAIM_ADJUSTMENT; yang tersimpan hanya hasilnya di GROSSVALUE.
//
// VAT adalah PERSEN, bukan nominal: VATType "1" menghitungnya dari Professional Fee,
// VATType "2" dari Professional Fee + Survey Expenses.
type AdjusterFee struct {
	ProfessionalFee Money
	SurveyExpenses  Money
	VAT             Percent
	VATType         string
}

// Kode VATType pada `SetValueAdjusterFee`.
const (
	VATOfProfessionalFee = "1"
	VATOfSubtotal        = "2"
)

// Gross adalah GrossValue fee adjuster: VATValue + Professional Fee + Survey Expenses.
func (f AdjusterFee) Gross() Money {
	subtotal := f.ProfessionalFee + f.SurveyExpenses
	var vat Money
	switch f.VATType {
	case VATOfProfessionalFee:
		vat = f.ProfessionalFee.Share(f.VAT)
	case VATOfSubtotal:
		vat = subtotal.Share(f.VAT)
	}
	return subtotal + vat
}

// SettlementInput adalah isian satu baris dari layar InputAdjustment.
type SettlementInput struct {
	PaymentType string
	Currency    string
	Propose     Money // Total Klaim
	Submitted   Money // Nilai Pengajuan Tertanggung
	LOC         Percent
	SalvageA    Money
	SalvageB    Money
	RiskType    string
	RiskPercent Percent
	RiskValue   Money // diisi petugas hanya untuk tipe resiko 3 (Lainnya)
	Fee         AdjusterFee
	Chronology  string
	Notes       string
}

// SettlementContext adalah bahan dari klaim yang dipakai menghitung dan memeriksa baris.
type SettlementContext struct {
	Claim    Claim
	Coverage Coverage

	// Rate adalah kurs mata uang baris pada tanggal kejadian; TSIRate kurs mata uang polis
	// (TSI dicatat dalam mata uang polis).
	Rate    ExchangeRate
	TSIRate ExchangeRate

	Now time.Time

	// Analyst: pemanggil anggota grup Analyst (When `IsAnalisator`). Pada PA, "Nilai Pengajuan"
	// nonaktif bagi Analyst (`InputAdjustment_sect`: `.AcceptanceStatus != '' || IsAnalisator`),
	// sehingga wajibnya tidak diperiksa — isian nonaktif tidak divalidasi Pega.
	Analyst bool
}

// Pesan aturan. Yang berasal dari Pega ditulis apa adanya; yang tambahan berbahasa
// Inggris (`D-80`).
const (
	msgPaymentTypeEmpty     = "Silahkan Pilih Tipe Pembayaran"
	msgProposeEmpty         = "Nilai Proposed Adjustment Harus Diisi"
	msgProposeOverEstimate  = "Nilai propose adjustment tidak boleh lebih besar dari nilai estimasi"
	msgRiskTypeEmpty        = "Tipe Resiko Harus Diisi"
	msgRiskPercentEmpty     = "Percent Resiko Harus Diisi"
	msgFeeEmpty             = "Nilai Professional Fee, Survey Expenses, dan VAT Harus Diisi"
	msgValueOverEstimate    = "Nilai adjustment tidak boleh lebih besar dari nilai estimasi"
	msgValueOverTSI         = "Nilai adjustment tidak boleh lebih besar dari nilai TSI"
	msgRiskOverValue        = "Nilai Resiko sendiri harus lebih kecil dari nilai adjustment"
	msgAdjustmentNegative   = "Nilai adjust Klaim tidak boleh Kecil dari 0"
	msgTotalOverEstimate    = "Total Nilai Adjustment Melebihi Estimasi"
	msgSubmittedEmpty       = "Nilai Pengajuan Tertanggung is required."
	msgSubmittedEmptyPA     = "Nilai Pengajuan is required."
	msgProposeOverSubmitted = "Nilai Total Klaim melebihi Nilai Pengajuan Klaim"
	msgSalvageNotHere       = "Salvage settlement lines are added from the Salvage module."
	msgRejectNeedsReason    = "Tolak Klaim needs Alasan Tolak Klaim 1 and 2, whose list (GetDataPenolakanKlaimMas) is not in the export yet."
	msgPaymentTypeUnknown   = "Unknown payment type."
	msgSettlementNeedsSheet = "Download the Claim Face Sheet before adding an adjustment."
)

// Kode pelanggaran baris adjustment.
const (
	ViolationSettlementPaymentType ViolationCode = "adjustment_tipe_pembayaran"
	ViolationSettlementPropose     ViolationCode = "adjustment_nilai_propose"
	ViolationSettlementSubmitted   ViolationCode = "adjustment_nilai_pengajuan"
	ViolationSettlementRisk        ViolationCode = "adjustment_resiko_sendiri"
	ViolationSettlementFee         ViolationCode = "adjustment_fee_adjuster"
	ViolationSettlementEstimate    ViolationCode = "adjustment_melebihi_estimasi"
	ViolationSettlementTSI         ViolationCode = "adjustment_melebihi_tsi"
	ViolationSettlementFaceSheet   ViolationCode = "adjustment_belum_cfs"
)

// ShareASMOf adalah bagian Sinar Mas pada baris adjustment: SHAREASM polis, atau 100%
// bila tidak diketahui. Pada data Pega ASM_SHARE sama dengan SHAREASM klaimnya pada 94%
// baris, untuk ketiga peran (LEADER, MEMBER, FAC IN).
func ShareASMOf(p Policy) Percent {
	if p.Coinsurance.HasShare {
		return p.Coinsurance.ShareASM
	}
	return PercentFull
}

// CoverageEstimate adalah jumlah estimasi satu jenis pada jaminan: nilai asli (dalam mata
// uang estimasinya) dan nilai rupiahnya.
func CoverageEstimate(c Coverage, kind string) (value, idr Money) {
	for _, it := range c.Item {
		for _, e := range it.Estimation {
			if e.Type == kind {
				value += e.Value
				idr += e.Converted
			}
		}
	}
	return value, idr
}

// InterimPaid adalah "Nilai Interim": jumlah gross baris Interim yang SUDAH diakseptasi
// pada jaminan ini. Rule yang mengisi InterimPayment tidak ada di export; rumus ini
// dicocokkan ke data Pega — pada baris Final/Interim yang gross-nya tidak sama dengan
// nilai final, 75 cocok dengan pengurangan interim terakseptasi, 1 dengan seluruh interim.
func InterimPaid(c Coverage) Money {
	var total Money
	for _, s := range c.Settlement {
		if s.PaymentType == PaymentInterim && s.AcceptanceStatus == "1" {
			total += s.Gross
		}
	}
	return total
}

func proposeBased(pt string) bool {
	return pt == PaymentFinal || pt == PaymentInterim || pt == PaymentAdjustment
}

// CurrencyIDR adalah kode mata uang rupiah (POOLDATA.CURRENCY) — `NewEstimationPA` mengisi
// `.Currency := "10026"` dan `.KursValue := 1`.
const CurrencyIDR = "10026"

// NewEstimationPA adalah aktivitas Pega `NewEstimationPA`: satu item baru dengan satu estimasi
// klaim bernilai TSI jaminan, bermata uang IDR, kurs 1, bertanggal saat ini. Ia dipanggil
// `ValidationAdjustment` step 15 hanya bila lini PA dan jaminan belum punya baris adjustment —
// yaitu pada Tambah pertama. Estimasi inilah yang dicetak Claim Face Sheet PA, karena klaim PA
// tidak melewati tahap Input Estimasi.
//
// Mengembalikan false bila tidak ada yang ditambahkan: bukan PA, jaminan sudah punya adjustment,
// atau masih ada estimasi yang belum dibuatkan CFS (Tambah berulang tanpa simpan tidak menumpuk
// estimasi — di Pega setiap Tambah langsung membuat baris, sehingga hal itu tidak terjadi).
func NewEstimationPA(c Claim, cov *Coverage, now time.Time) bool {
	if c.Policy.Line != LinePersonalAccident || len(cov.Settlement) > 0 || HasUnprintedEstimate(*cov) {
		return false
	}
	cov.Item = append(cov.Item, ObjectItem{Estimation: []Estimation{{
		Type: EstimateClaim, Currency: CurrencyIDR, Date: now, Value: cov.TSI,
		Rate: ExchangeRateOne, Converted: cov.TSI,
	}}})
	return true
}

// PHKCoverages adalah kode jaminan When `IsPHK` (`.CoverageOldID` 10010, 10023, 10018).
var PHKCoverages = map[string]bool{"10010": true, "10023": true, "10018": true}

// AnalystTransferLine adalah When `isAnalistorTransfer`: lini PA dan `.IsAnalisatorTransfer = 1`.
// Penanda baris itu diisi `ValidationAdjustment` step 30 bila jaminannya PHK (atau klaim TKI — penanda
// TKI belum dibaca aplikasi ini), dan `setTicketToAnalyst` step 5/8 pada jaminan yang ditransfer ke
// Analyst (ISANALISTRANSFER). Hanya pada baris seperti ini "Total Klaim" (`.ProposeAdjustmentValue`)
// tampil dan wajib (`InputAdjustment_sect`, kontainer `isAnalistorTransfer || isTKIPHK`).
func AnalystTransferLine(c Claim, cov Coverage) bool {
	return c.Policy.Line == LinePersonalAccident && (PHKCoverages[strings.TrimSpace(cov.ID)] || cov.AnalystTransferred)
}

// ComputeSettlementLine menghitung nilai-nilai sebuah baris tanpa memeriksanya — dipakai
// layar untuk menampilkan hasil hitungan setiap kali isian berubah, seperti Pega yang
// menjalankan `SetNilaiResikoSendiri` pada perubahan field.
//
// # Hitungan (`SetNilaiResikoSendiri`, InpatientDay = 1 di luar PA)
//
//	LOCValue   = Total Klaim × LOC%
//	RiskValue  = (Total Klaim − LOCValue − SalvageA) × Risk%   tipe resiko 1
//	           = TSI × Risk%                                   tipe resiko 2
//	           = isian petugas                                 tipe resiko 3 (Lainnya)
//	Final      = Total Klaim − LOCValue − SalvageA − RiskValue
//	Gross      = Final − SalvageB − Interim                    tipe 1, 2 (Interim hanya Non-MBU)
//	           = Final                                         tipe 5
//	           = Professional Fee + Survey Expenses + VAT      tipe 4 (`SetValueAdjusterFee`)
//	Value      = Gross × ShareASM
func ComputeSettlementLine(in SettlementInput, sc SettlementContext) SettlementLine {
	pt := strings.TrimSpace(in.PaymentType)
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = sc.Claim.Policy.Currency
	}
	line := SettlementLine{
		PaymentType: pt,
		Currency:    currency,
		Rate:        sc.Rate,
		ShareASM:    ShareASMOf(sc.Claim.Policy),
		ExGratia:    sc.Claim.ExGratia,
		Chronology:  strings.TrimSpace(in.Chronology),
		Notes:       strings.TrimSpace(in.Notes),
		CreatedAt:   sc.Now,
	}
	claimEstimate, _ := CoverageEstimate(sc.Coverage, EstimateClaim)
	adjusterEstimate, _ := CoverageEstimate(sc.Coverage, EstimateAdjuster)

	switch {
	case proposeBased(pt):
		line.Propose = in.Propose
		line.Submitted = in.Submitted
		line.LOC = in.LOC
		line.SalvageA = in.SalvageA
		line.SalvageB = in.SalvageB
		line.RiskType = strings.TrimSpace(in.RiskType)
		line.Estimation = claimEstimate
		if sc.Claim.Policy.Line.IsNonMBU() {
			line.Interim = InterimPaid(sc.Coverage)
		}

		locValue := line.Propose.Share(line.LOC)
		switch line.RiskType {
		case RiskOfClaim:
			line.RiskPercent = in.RiskPercent
			line.RiskValue = (line.Propose - locValue - line.SalvageA).Share(line.RiskPercent)
		case RiskOfTSI:
			line.RiskPercent = in.RiskPercent
			line.RiskValue = sc.Coverage.TSI.Share(line.RiskPercent)
		case RiskOther:
			line.RiskValue = in.RiskValue
		}
		final := line.Propose - locValue - line.SalvageA - line.RiskValue
		line.Gross = final
		if pt == PaymentFinal || pt == PaymentInterim {
			line.Gross = final - line.SalvageB - line.Interim
		}
		// PA: Nilai Propose Adjustment hanya diisi pada baris `isAnalistorTransfer`; selama ia kosong
		// atau di bawah resiko sendiri, Total Klaim dan Nilai Adjustment nol (`SetNilaiResikoSendiri`
		// step 28).
		if sc.Claim.Policy.Line == LinePersonalAccident && (line.Propose <= 0 || line.Propose < line.RiskValue) {
			line.Gross = 0
		}
		// Travel: klaim di bawah resiko sendiri bernilai nol (`SetNilaiResikoSendiri`
		// langkah 4789). Non-MBU menolaknya saat diperiksa.
		if line.RiskValue > line.Propose && sc.Claim.Policy.Line == LineTravel {
			line.Gross = 0
		}

	case pt == PaymentAdjusterFee:
		line.Gross = in.Fee.Gross()
		line.Estimation = adjusterEstimate
	}

	line.Value = line.Gross.Share(line.ShareASM)
	line.Accepted = line.Gross
	if c := sc.Claim.Policy.TypeOfCoins; c == "1" || c == "F" {
		line.Accepted = line.Value
	}
	return line
}

// NewSettlementLine menghitung lalu memeriksa satu baris adjustment baru.
func NewSettlementLine(in SettlementInput, sc SettlementContext) (SettlementLine, error) {
	var v collector
	pt := strings.TrimSpace(in.PaymentType)
	switch pt {
	case "":
		v.add(ViolationSettlementPaymentType, "tipe_pembayaran", msgPaymentTypeEmpty)
		return SettlementLine{}, v.err()
	case PaymentSalvage:
		v.add(ViolationSettlementPaymentType, "tipe_pembayaran", msgSalvageNotHere)
		return SettlementLine{}, v.err()
	case PaymentReject:
		v.add(ViolationSettlementPaymentType, "tipe_pembayaran", msgRejectNeedsReason)
		return SettlementLine{}, v.err()
	case PaymentFinal, PaymentInterim, PaymentAdjusterFee, PaymentAdjustment:
	default:
		v.add(ViolationSettlementPaymentType, "tipe_pembayaran", msgPaymentTypeUnknown)
		return SettlementLine{}, v.err()
	}

	// Tahap InputSurveyor baru dapat dicapai sesudah CFS, tetapi aturannya dijaga di sini
	// juga: adjustment dibentuk dari estimasi yang sudah dikunci.
	if !sc.Claim.HasFaceSheet() {
		v.add(ViolationSettlementFaceSheet, "adjustment", msgSettlementNeedsSheet)
		return SettlementLine{}, v.err()
	}

	line := ComputeSettlementLine(in, sc)
	_, claimEstimateIDR := CoverageEstimate(sc.Coverage, EstimateClaim)
	_, adjusterEstimateIDR := CoverageEstimate(sc.Coverage, EstimateAdjuster)
	valueIDR := line.Value.Convert(line.Rate)

	if proposeBased(pt) {
		pa := sc.Claim.Policy.Line == LinePersonalAccident
		// PA: "Nilai Propose Adjustment" hanya ada (dan wajib) pada baris isAnalistorTransfer;
		// lini lain mewajibkan "Total Klaim".
		if in.Propose == 0 && (!pa || AnalystTransferLine(sc.Claim, sc.Coverage)) {
			v.add(ViolationSettlementPropose, "nilai_propose", msgProposeEmpty)
		}
		// Section: "Nilai Pengajuan Tertanggung" (PA: "Nilai Pengajuan", .ProposeValue) ber-pyRequired
		// untuk tipe selain 3/4/7.
		if in.Submitted == 0 && !(pa && sc.Analyst) {
			msg := msgSubmittedEmpty
			if pa {
				msg = msgSubmittedEmptyPA
			}
			v.add(ViolationSettlementSubmitted, "nilai_pengajuan", msg)
		}
		// SetNilaiResikoSendiri step 39: ProposeAdjustmentValue > ProposeValue.
		if pa && in.Propose > in.Submitted && in.Submitted > 0 {
			v.add(ViolationSettlementPropose, "nilai_propose", msgProposeOverSubmitted)
		}
		switch line.RiskType {
		case RiskOfClaim, RiskOfTSI:
			if in.RiskPercent <= 0 {
				v.add(ViolationSettlementRisk, "persen_resiko", msgRiskPercentEmpty)
			}
		case RiskOther:
		default:
			v.add(ViolationSettlementRisk, "tipe_resiko", msgRiskTypeEmpty)
		}
		// Non-MBU: resiko sendiri tidak boleh melampaui Total Klaim. PA tanpa Nilai Propose
		// Adjustment tidak diperiksa (nilainya nol, step 28).
		if line.RiskValue > line.Propose && sc.Claim.Policy.Line != LineTravel && !(pa && in.Propose == 0) {
			v.add(ViolationSettlementRisk, "nilai_resiko", msgRiskOverValue)
		}
		if line.Propose.Convert(line.Rate) > claimEstimateIDR {
			v.add(ViolationSettlementEstimate, "nilai_propose", msgProposeOverEstimate)
		}
		switch {
		case pt == PaymentAdjustment && claimEstimateIDR+valueIDR < 0:
			v.add(ViolationSettlementEstimate, "nilai_propose", msgAdjustmentNegative)
		case valueIDR > claimEstimateIDR:
			v.add(ViolationSettlementEstimate, "nilai_propose", msgValueOverEstimate)
		}
		if valueIDR > sc.Coverage.TSI.Convert(sc.TSIRate) {
			v.add(ViolationSettlementTSI, "nilai_propose", msgValueOverTSI)
		}

		// ValidTotalEstimation: seluruh adjustment klaim pada jaminan ini, termasuk yang
		// baru, tidak boleh melampaui estimasi klaimnya.
		total := valueIDR
		for _, s := range sc.Coverage.Settlement {
			if proposeBased(s.PaymentType) {
				total += s.Value.Convert(s.Rate)
			}
		}
		if total > claimEstimateIDR {
			v.add(ViolationSettlementEstimate, "nilai_propose", msgTotalOverEstimate)
		}
	}

	if pt == PaymentAdjusterFee {
		// ValidationTypePaymentAdj: AdjusterFeeValue nol berarti komponennya belum diisi.
		if line.Value == 0 {
			v.add(ViolationSettlementFee, "fee", msgFeeEmpty)
		}
		// SetValueAdjusterFee: seluruh fee adjuster jaminan ini, termasuk yang baru, tidak
		// melampaui estimasi adjuster. Polis leader (TYPEOFCOINS "2") mengalikan jumlahnya
		// sekali lagi dengan share ASM — dibawa apa adanya (`P-5`).
		total := valueIDR
		for _, s := range sc.Coverage.Settlement {
			if s.PaymentType == PaymentAdjusterFee {
				total += s.Value.Convert(s.Rate)
			}
		}
		if sc.Claim.Policy.TypeOfCoins == "2" {
			total = total.Share(line.ShareASM)
		}
		if total > adjusterEstimateIDR {
			v.add(ViolationSettlementEstimate, "fee", msgValueOverEstimate)
		}
	}

	if err := v.err(); err != nil {
		return SettlementLine{}, err
	}
	return line, nil
}

func (p *collector) err() error {
	if len(p.violations) == 0 {
		return nil
	}
	return &ValidationError{Violation: p.violations}
}
