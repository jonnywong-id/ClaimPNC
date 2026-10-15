package usecase

import (
	"context"
	"math/big"

	"claim-pnc/internal/registrasi"
)

// PremiumAging adalah isian "Aging Amount" layar Input Estimasi — `.PaymentData.AgingAmount`,
// jawaban layanan premi (`getPremiumPaidOn_before`) yang juga dipakai aturan akseptasi.
type PremiumAging struct {
	// Amount adalah AgingAmount apa adanya; nil bila layanan tidak mengisinya.
	Amount *big.Rat
	// Available false berarti layanan premi tidak dapat dihubungi.
	Available bool
}

// PremiumAgingOf membaca AgingAmount polis klaim dari layanan premi.
//
// Layanan yang tidak dapat dihubungi BUKAN galat layar: isian ini hanya informasi, dan
// menggagalkan seluruh layar karenanya akan menghentikan pekerjaan estimasi. Ia dilaporkan
// sebagai Available=false supaya layar dapat menyebutnya, bukan menampilkan kosong.
func (l *Service) PremiumAgingOf(ctx context.Context, claimID string) (PremiumAging, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return PremiumAging{}, err
	}
	caseID, err := l.acceptance.PolicyCaseID(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return PremiumAging{}, err
	}
	statement, err := l.premium.Statement(ctx, portalOf("", claim.Portal), registrasi.PremiumQuery{
		PolicyNumber: claim.Policy.Number, ProdKe: claim.Policy.ProdKe,
		RequestedAt: claim.Policy.CoverageEnd, CaseID: registrasi.PremiumCaseID(caseID),
	})
	if err != nil {
		return PremiumAging{}, nil
	}
	return PremiumAging{Amount: statement.AgingAmount, Available: true}, nil
}
