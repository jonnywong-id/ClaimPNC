package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Detailed adalah isi popup beserta cabang yang benar-benar dipakai.
type Detailed struct {
	// Detail adalah seluruh isi popup untuk satu klaim.
	Detail inboxosclaimpercabang.Detail

	// Query adalah cabang yang benar-benar dipakai, lengkap dengan namanya.
	Query inboxosclaimpercabang.Query

	// PlannedDifferences adalah selisih POPUP terhadap Pega yang sudah diputuskan.
	//
	// Ia daftar yang berbeda dari milik daftar: catatan tentang "Total Sum Insured" tidak
	// berlaku di grid, dan catatan tentang paginasi tidak berlaku di popup (`D-54`).
	PlannedDifferences []string
}

// Detail mengambil isi popup untuk satu klaim.
//
// Urutan pemeriksaannya SAMA PERSIS dengan List — identitas, lalu cabang, baru isinya — dan
// itu bukan pengulangan yang dapat dihemat. Popup memuat nama tertanggung, kronologi, dan
// nilai uang; pemeriksaannya tidak boleh lebih longgar daripada layar yang membukanya.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxosclaimpercabang.Caller,
	claimNumber string,
) (Detailed, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Detailed{}, inboxosclaimpercabang.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Detailed{}, err
	}

	branch, resolved, err := repo.BranchOf(ctx, clean.DetailBranchCode)
	if err != nil {
		return Detailed{}, fmt.Errorf("menentukan cabang pemanggil: %w", err)
	}
	if !resolved {
		return Detailed{}, inboxosclaimpercabang.ErrBranchUnknown
	}

	query := inboxosclaimpercabang.Query{Branch: branch}

	detail, found, err := repo.FindDetail(ctx, query, claimNumber)
	if err != nil {
		return Detailed{}, fmt.Errorf(
			"mengambil detail klaim pada cabang %q: %w", branch.Code, err)
	}
	if !found {
		// Dicatat, karena ia punya dua sebab yang berbeda jauh: nomor yang memang tidak ada,
		// dan nomor milik cabang lain. Yang kedua adalah upaya membaca data badan hukum lain,
		// entah disengaja atau akibat daftar yang sudah usang — dan log inilah satu-satunya
		// tempat keduanya masih dapat dibedakan (`R-20`).
		if s.logger != nil {
			s.logger.Info(
				"detail klaim diminta untuk nomor yang tidak ada di cabang pemanggil",
				slog.String("modul", "inbox-os-claim-per-cabang"),
				slog.String("pemanggil", clean.Login),
				slog.String("cabang", branch.Code),
				slog.String("nomor_klaim", claimNumber),
				slog.String("portal", portalAlias),
			)
		}
		return Detailed{}, inboxosclaimpercabang.ErrClaimNotFound
	}

	// Aging diisi DI SINI, bukan di dalam kueri — alasannya sama dengan pada List, dan ada di
	// inboxosclaimpercabang.AgingDaysSince.
	detail.AgingDays = inboxosclaimpercabang.AgingDaysSince(detail.RegisterDate, s.clock.Now())

	return Detailed{
		Detail:             detail,
		Query:              query,
		PlannedDifferences: inboxosclaimpercabang.DetailPlannedDifferences,
	}, nil
}
