package registrasi

import (
	"context"
	"math/big"
	"strings"
	"time"
)

// # Pemeriksaan premi saat akseptasi
//
// `SetAdjustmentAcceptation` langkah 36–49: status premi dibaca `GetStatusPremi` (Connect
// REST `getPremiumPaidOn_before`), lalu akseptasi ditolak "Premi belum lunas, tidak bisa
// akseptasi adjustment." kecuali salah satu pengecualian berlaku. Pengecualian yang
// DI-HARDCODE di Pega tidak dibawa (`D-15`): satu nomor klaim (langkah 36), satu nomor polis
// (langkah 47), dan operator quotation TRAVELOKASVC (langkah 48).

// Status premi — `Policy.PremiumPaymentStatus`.
const (
	PremiumPaid   = "LUNAS"
	PremiumUnpaid = "BELUM LUNAS"
)

// PremiumQuery adalah parameter `getPremiumPaidOn_before`.
type PremiumQuery struct {
	PolicyNumber string    // noPolis — Policy.PolicyNo
	ProdKe       string    // prodKe — ClaimData.ProdKe
	RequestedAt  time.Time // tglRequest — Policy.EndDateTime
	CaseID       string    // caseId — Policy.CaseID bila memuat "ASM", selain itu "-"
}

// PremiumInstallment adalah satu baris `PaymentData.Payment.ListInstallment`.
type PremiumInstallment struct {
	DueDate       time.Time
	PaymentDate   string // teks apa adanya; yang dinilai Pega hanya kosong atau tidak
	PaymentAmount string
}

// PremiumStatement adalah jawaban layanan premi yang dipakai aturan akseptasi.
type PremiumStatement struct {
	AgingAmount  *big.Rat // nil bila kosong
	Installments []PremiumInstallment
}

// PremiumStatusOf menurunkan `Policy.PremiumPaymentStatus` seperti `GetStatusPremi` langkah
// 7–9. Hanya cicilan PERTAMA yang dinilai, karena kedua cabang langkah 7 melompat keluar dari
// perulangan. Kosong berarti status tidak ditetapkan — aturan akseptasi meloloskannya.
func PremiumStatusOf(s PremiumStatement, now time.Time) string {
	one := big.NewRat(1, 1)
	status := ""
	loop := s.AgingAmount == nil || s.AgingAmount.Cmp(one) >= 0
	if loop && len(s.Installments) > 0 {
		first := s.Installments[0]
		dueLater := first.DueDate.After(now) // @CompareDates(.DueDate, now)
		unpaid := strings.TrimSpace(first.PaymentDate) == "" || strings.TrimSpace(first.PaymentAmount) == ""
		if !dueLater && unpaid {
			status = PremiumUnpaid // langkah 7.1, lalu ke langkah 8
		} else {
			return PremiumPaid // langkah 7.2, lompat ke akhir
		}
	}
	if s.AgingAmount != nil {
		switch {
		case s.AgingAmount.Cmp(one) <= 0:
			status = PremiumPaid // langkah 8
		default:
			status = PremiumUnpaid // langkah 9
		}
	}
	return status
}

// PremiumExemption adalah bahan pengecualian blok premi yang dibaca dari luar baris.
type PremiumExemption struct {
	// KBRU: sumber bisnis grup KBRU pada entitas ASM (langkah 38 — email pemberitahuannya
	// tidak pernah benar-benar terkirim di Pega, dan tidak dibangun).
	KBRU bool
	// TravelClientName adalah CLIENTNAME agen leader polis (`BrowseClientNameTravel_SQL`),
	// hanya untuk Travel.
	TravelClientName string
	// OpenProtection: ada Open Protection tipe premi (TypePro 2) yang disetujui (langkah 41–46).
	OpenProtection bool
}

// KBRUSourcesOfBusiness adalah `When/SimasGroupKBRU`: Quotation.SourceOfBusiness grup KBRU.
var KBRUSourcesOfBusiness = map[string]bool{
	"10011135": true, "10011766": true, "10011767": true, "10011770": true,
	"10014533": true, "10027808": true, "10057975": true,
}

// PremiumBlocks menilai langkah 37–49: true bila akseptasi harus ditolak karena premi.
func PremiumBlocks(line SettlementLine, status string, s PremiumStatement, e PremiumExemption) bool {
	// Langkah 37: lunas atau kosong, atau Tipe Pembayaran 3/4/7, lolos.
	if status == PremiumPaid || status == "" {
		return false
	}
	switch strings.TrimSpace(line.PaymentType) {
	case PaymentSalvage, PaymentAdjusterFee, paymentCollectionFee:
		return false
	}
	if e.KBRU { // langkah 38
		return false
	}
	// Langkah 39–40: Travel dengan agen "AGENCY INDIVIDU"/"DIRECT" dan cicilan terakhir yang
	// terisi sudah menutup nilai adjustment.
	if c := strings.TrimSpace(e.TravelClientName); c == "AGENCY INDIVIDU" || c == "DIRECT" {
		var last *big.Rat
		for _, i := range s.Installments {
			if v, ok := new(big.Rat).SetString(strings.TrimSpace(i.PaymentAmount)); ok {
				last = v
			}
		}
		if last != nil && last.Cmp(new(big.Rat).SetFrac64(int64(line.Value), 100)) >= 0 {
			return false
		}
	}
	return !e.OpenProtection // langkah 48–49
}

// PremiumService adalah seam ke layanan premi (`getPremiumPaidOn_before`). App adalah entitas
// baris POOLDATA.GCNM_CONNECT_REST (APP), bukan nama server.
type PremiumService interface {
	Statement(ctx context.Context, app string, q PremiumQuery) (PremiumStatement, error)
}

// PremiumCaseID adalah nilai caseId permintaan — `GetStatusPremi` langkah 3.
func PremiumCaseID(policyCaseID string) string {
	if strings.Contains(policyCaseID, "ASM") {
		return policyCaseID
	}
	return "-"
}
