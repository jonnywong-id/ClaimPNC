package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxadmin"
)

// ExportLOD mengambil SELURUH baris tab Branch Claim untuk tombol "Export LOD".
//
// Penyaring lini bisnis dan kata kunci yang sedang dipakai layar ikut berlaku, sehingga
// berkasnya berisi apa yang sedang dilihat petugas — tetapi SELURUH halamannya, bukan
// hanya halaman yang tampil.
func (s *Service) ExportLOD(
	ctx context.Context,
	portalAlias string,
	caller inboxadmin.Caller,
	input inboxadmin.QueryInput,
) ([]inboxadmin.WorkItem, error) {
	input.Tab = inboxadmin.TabBranchClaim
	query, err := inboxadmin.NewQuery(input, caller)
	if err != nil {
		return nil, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	if query.Scope, _, err = s.resolveScope(ctx, repo, query.Caller, input.Region); err != nil {
		return nil, err
	}
	rows, err := repo.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil baris Export LOD: %w", err)
	}
	return rows, nil
}

// AutoClaimResults mengambil hasil batch Auto Claim hari ini milik pemanggil.
func (s *Service) AutoClaimResults(
	ctx context.Context, portalAlias string, caller inboxadmin.Caller,
) ([]inboxadmin.AutoClaimResult, error) {
	if caller.Login == "" {
		return nil, inboxadmin.ErrCallerUnknown
	}
	repo, err := s.autoClaimRepo(portalAlias)
	if err != nil {
		return nil, err
	}
	from, to := inboxadmin.ProcessingDay(s.clock.Now())
	rows, err := repo.AutoClaimResults(ctx, caller.Login, from, to)
	if err != nil {
		return nil, fmt.Errorf("mengambil hasil Auto Claim: %w", err)
	}
	return rows, nil
}

// AutoClaimFailures mengambil seluruh baris batch Auto Claim yang gagal.
func (s *Service) AutoClaimFailures(ctx context.Context, portalAlias string) ([]inboxadmin.AutoClaimFailure, error) {
	repo, err := s.autoClaimRepo(portalAlias)
	if err != nil {
		return nil, err
	}
	rows, err := repo.AutoClaimFailures(ctx)
	if err != nil {
		return nil, fmt.Errorf("mengambil klaim gagal Auto Claim: %w", err)
	}
	return rows, nil
}

// autoClaimRepo memilih repo portal lalu memastikan ia mampu membaca data Auto Claim.
func (s *Service) autoClaimRepo(portalAlias string) (inboxadmin.AutoClaimRepo, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	reader, ok := repo.(inboxadmin.AutoClaimRepo)
	if !ok {
		return nil, inboxadmin.ErrExportUnavailable
	}
	return reader, nil
}
