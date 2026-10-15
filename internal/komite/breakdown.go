package komite

import (
	"strings"
	"time"

	"claim-pnc/internal/platform/money"
)

// Bagian bawah `Section/ShowTransferDetail`: tabel Claim Adjustment / Salvage / Adjuster
// Fee, History of Previous Adjustment Committees, dan Daftar Komite.
//
// ============================================================================
// TABEL NILAI — tiga bentuk menurut `.Type` (PAYMENTTYPE)
// ============================================================================
//
//	Type 1, 2, 5, bukan Travel   "Claim Adjustment", kolom nilai "ADJUSTMENT", sembilan baris
//	Type 1, 2, 5, Travel         "Claim Adjustment", kolom nilai "CLAIM ACCEPTED", lima baris
//	Type 3                       "Salvage"
//	Type 4                       "Adjuster Fee"
//
// Properti Pega dipetakan ke kolom `T_CLAIM_ADJUSTMENT` persis seperti modul registrasi
// menulisnya (`registrasi/settlement.go`):
//
//	ProposeAdjustmentValue  TOTAL_CLAIM              Total Claim
//	EstimationValue         Σ T_CLAIM_ESTIMASI       (jenis 1 klaim, jenis 2 fee adjuster)
//	LOC / LOCValue          LOC, TOTAL_CLAIM × LOC   Less Lack Of Document
//	SalvageValueA           NILAI_SALVAGE_A          Less Salvage (pertama)
//	IndividualRisk*         INDIVIDUAL_RISK_*        Less Deductible
//	ProposeAdjustmentFinal  (dihitung)               Nett Of Deductible
//	SalvageValue            (sisa, lihat di bawah)   Less Salvage (kedua)
//	InterimPayment          Σ GROSSVALUE Interim     Interim Payment
//	GrossValue              GROSSVALUE               Nett Of Deductible and Salvage
//	ShareASM / Adjustment   ASM_SHARE, ASM_SHARE_VALUE  Total Pembayaran ASM
//
// # Yang tidak punya kolom
//
// SalvageValue ("Salvage B") dan InterimPayment tidak disimpan; yang tersimpan hanya gross
// hasilnya: `Gross = Final − SalvageB − Interim` untuk Type 1 dan 2. Interim dihitung ulang
// seperti registrasi (`InterimPaid`: gross baris Interim yang sudah diakseptasi pada coverage
// yang sama, hanya lini Non-MBU), lalu Salvage B adalah sisanya. Bila sisanya tidak positif,
// sel dibiarkan kosong — persis tampilan Pega untuk Salvage B yang tidak diisi.
//
// Komponen fee adjuster (Professional Fee, Survey Expenses, VAT) tidak punya kolom sama sekali
// dan dibiarkan kosong; totalnya tetap dari GROSSVALUE.

// BreakdownRow adalah satu baris tabel nilai.
//
// Estimate dan Value bertipe money; HasEstimate/HasValue salah berarti sel dikosongkan, bukan
// nol. EstimateLabel mengisi kolom ESTIMATION dengan teks (tipe resiko sendiri, "Lainnya").
type BreakdownRow struct {
	Description string

	EstimateCurrency string
	Estimate         money.Money
	HasEstimate      bool
	EstimateLabel    string

	Percent  string
	Currency string
	Value    money.Money
	HasValue bool

	// Total menandai baris penutup yang ditebalkan di Pega.
	Total bool
}

// Breakdown adalah tabel nilai satu baris adjustment.
type Breakdown struct {
	Title       string
	ValueHeader string
	Rows        []BreakdownRow

	// Known salah untuk Type yang tabelnya tidak ada di sini (6 Tolak Klaim, 7).
	Known bool
}

// Label tipe resiko sendiri — sama dengan pilihan layar InputSurveyor.
func riskTypeLabel(code string) string {
	switch strings.TrimSpace(code) {
	case "1":
		return "% dari Nilai Klaim"
	case "2":
		return "% dari TSI"
	case "3":
		return "Lainnya"
	}
	return strings.TrimSpace(code)
}

// nonMBU mengikuti `LineOfBusiness.IsNonMBU` registrasi: Group Panel 003, 004, 006, 009.
func nonMBU(groupPanel string) bool {
	switch strings.TrimSpace(groupPanel) {
	case "003", "004", "006", "009":
		return true
	}
	return false
}

// panelOf memilih Group Panel dari kepala case, atau dari klaim bila kepala kosong.
func (d TransferDetail) panelOf() string {
	if p := strings.TrimSpace(d.GroupPanel); p != "" {
		return p
	}
	return strings.TrimSpace(d.Claim.GroupPanel)
}

// BreakdownFor menyusun tabel nilai satu baris adjustment.
func (d TransferDetail) BreakdownFor(l AdjustmentLine) Breakdown {
	cur := l.CurrencyCode
	travel := d.panelOf() == "005"
	row := func(desc string, v money.Money) BreakdownRow {
		return BreakdownRow{Description: desc, Currency: cur, Value: v, HasValue: true}
	}
	total := BreakdownRow{
		Description: "Total Pembayaran ASM", Percent: l.ASMSharePercent,
		Currency: cur, Value: l.ASMShareValue, HasValue: true, Total: true,
	}

	switch kodeRingkas(l.PaymentType) {
	case "1", "2", "5":
		locValue, _ := ShareOf(l.TotalClaim, l.LOCPercent)
		final := l.TotalClaim - locValue - l.SalvageValue - l.IndividualRisk

		first := row("Total Claim", l.TotalClaim)
		first.EstimateCurrency = cur
		first.Estimate, first.HasEstimate = l.Estimation, l.HasEstimation
		loc := row("Less Lack Of Document", locValue)
		loc.Percent = l.LOCPercent
		risk := row("Less Deductible", l.IndividualRisk)
		risk.EstimateLabel = riskTypeLabel(l.RiskType)
		risk.Percent = l.RiskPercent

		if travel {
			return Breakdown{
				Title: "Claim Adjustment", ValueHeader: "CLAIM ACCEPTED", Known: true,
				Rows: []BreakdownRow{first, loc, risk, row("Nett Of Deductible", final), total},
			}
		}

		salvageA := row("Less Salvage", l.SalvageValue)
		interim := BreakdownRow{Description: "Interim Payment", Currency: cur}
		salvageB := BreakdownRow{Description: "Less Salvage", Currency: cur}
		if t := kodeRingkas(l.PaymentType); t == "1" || t == "2" {
			var paid money.Money
			if nonMBU(d.panelOf()) {
				paid = l.InterimPaid
			}
			interim.Value, interim.HasValue = paid, true
			if rest := final - l.GrossValue - paid; rest > 0 {
				salvageB.Value, salvageB.HasValue = rest, true
			}
		}
		return Breakdown{
			Title: "Claim Adjustment", ValueHeader: "ADJUSTMENT", Known: true,
			Rows: []BreakdownRow{
				first, loc, salvageA, risk, row("Nett Of Deductible", final),
				salvageB, interim, row("Nett Of Deductible and Salvage", l.GrossValue), total,
			},
		}

	case "3":
		total.Description = "Total Penerimaan Salvage ASM"
		return Breakdown{
			Title: "Salvage", ValueHeader: "NILAI SALVAGE", Known: true,
			Rows: []BreakdownRow{row("Nilai Salvage", l.GrossValue), total},
		}

	case "4":
		empty := func(desc string) BreakdownRow { return BreakdownRow{Description: desc, Currency: cur} }
		return Breakdown{
			Title: "Adjuster Fee", ValueHeader: "NILAI", Known: true,
			Rows: []BreakdownRow{
				empty("Professional Fee"), empty("Survey Expenses"), empty("VAT"),
				row("Total Adjuster Fee", l.GrossValue), total,
			},
		}
	}
	return Breakdown{Title: "Claim Adjustment", Known: false}
}

// CommitteeEntry adalah satu baris anggota komite — "Daftar Komite" untuk case ini, atau
// "History of Previous Adjustment Committees" untuk case komite sebelumnya pada klaim yang
// sama. Keduanya satu baris `T_CLAIM_KOMITE_LIST`.
type CommitteeEntry struct {
	CaseID     string
	MemberName string
	Tier       int
	Status     string // STATUSAPPROVE mentah: 1 setuju, 2 tidak setuju, lainnya menunggu
	Note       string
	DecidedAt  time.Time
}

// StatusLabel adalah teks kolom Status seperti Pega menampilkan `.KomiteAproval`.
func (e CommitteeEntry) StatusLabel() string {
	switch strings.TrimSpace(e.Status) {
	case "1":
		return "Setuju"
	case "2":
		return "Tidak Setuju"
	}
	return "Menunggu"
}
