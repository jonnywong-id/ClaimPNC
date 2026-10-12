package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Summarized adalah isi panel ringkasan beserta cabang yang dipakai.
type Summarized struct {
	Summary inboxosclaimpercabang.Summary
	Query   inboxosclaimpercabang.Query
}

// Summary mengambil isi panel ringkasan untuk cabang pemanggil.
//
// Pemeriksaannya SAMA PERSIS dengan List — identitas, lalu cabang. Panel memuat nilai uang
// satu cabang, jadi pemeriksaannya tidak boleh lebih longgar daripada daftarnya.
func (s *Service) Summary(
	ctx context.Context,
	portalAlias string,
	caller inboxosclaimpercabang.Caller,
) (Summarized, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Summarized{}, inboxosclaimpercabang.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Summarized{}, err
	}

	branch, resolved, err := repo.BranchOf(ctx, clean.DetailBranchCode)
	if err != nil {
		return Summarized{}, fmt.Errorf("menentukan cabang pemanggil: %w", err)
	}
	if !resolved {
		return Summarized{}, inboxosclaimpercabang.ErrBranchUnknown
	}

	query := inboxosclaimpercabang.Query{Branch: branch}

	rows, err := repo.SummaryRows(ctx, query)
	if err != nil {
		return Summarized{}, fmt.Errorf(
			"membaca ringkasan cabang %q: %w", branch.Code, err)
	}

	summary := inboxosclaimpercabang.Summarize(rows, s.clock.Now())

	// Porsi treaty OR dibaca TERPISAH, dan kegagalannya tidak menghentikan panel.
	//
	// Ia satu-satunya bagian yang menyentuh DB Link `@asmd`. Membiarkannya menggagalkan
	// seluruh panel berarti gangguan jaringan ke sistem lain mematikan layar yang seluruh
	// angka lainnya sudah terbaca.
	total, available, err := repo.TreatyOR(ctx, query)
	switch {
	case err != nil:
		if s.logger != nil {
			s.logger.Warn(
				"total treaty OR tidak dapat dibaca; kartu angkanya dikosongkan",
				slog.String("modul", "inbox-os-claim-per-cabang"),
				slog.String("cabang", branch.Code),
				slog.String("portal", portalAlias),
				slog.String("galat", err.Error()),
			)
		}
		summary.ReserveORAvailable = false
	default:
		summary.ReserveOR = total
		summary.ReserveORAvailable = available
	}

	return Summarized{Summary: summary, Query: query}, nil
}
