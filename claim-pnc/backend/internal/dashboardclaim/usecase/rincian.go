package usecase

import (
	"context"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// ClaimDetailQuery adalah permintaan rincian satu klaim.
type ClaimDetailQuery struct {
	PortalAlias string

	// ClaimID adalah PZINSKEY baris grid, bukan nomor klaimnya.
	//
	// Kunci teknis yang dipakai, karena itu yang dipakai Pega: `GetJsonKlaimPNC` menyaring
	// `idpega`. Memakai nomor klaim akan menuntut pencarian tambahan, dan pada data warisan
	// yang nomornya kembar ia dapat membuka klaim yang salah.
	ClaimID string
}

// ClaimDetail membaca rincian satu klaim untuk popup yang terbuka dari nomor klaim.
//
// Menggantikan `setDataViewKlaim_Act` + harness `ViewTempDetailClaim` (`pyTarget: popup`).
func (s *Service) ClaimDetail(
	ctx context.Context,
	q ClaimDetailQuery,
) (dashboardclaim.ClaimDetail, error) {
	if strings.TrimSpace(q.ClaimID) == "" {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}

	repo, err := s.claims(q.PortalAlias)
	if err != nil {
		return dashboardclaim.ClaimDetail{}, err
	}

	return repo.FindClaimDetail(ctx, q.ClaimID)
}
