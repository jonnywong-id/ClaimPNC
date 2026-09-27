package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini mengorkestrasi tab **KPI Admin**.
//
// Dua operasi: kartu skor dan grid rinciannya. Keduanya terpisah karena di layar pun
// terpisah — kartu skor selalu satu "baris" dan tidak dipaginasi, grid rinciannya
// dipaginasi.

// AdminScored adalah kartu skor beserta permintaan yang benar-benar dipakai.
type AdminScored struct {
	Card  reportkpi.AdminScorecard
	Query reportkpi.AdminQuery
}

// AdminScorecard mengambil kartu skor satu kelompok.
//
// Penyusunan kartunya dikerjakan DOMAIN (`reportkpi.BuildScorecard`), bukan di sini dan
// bukan di penyimpanan: urutan metrik, labelnya, dan kesimpulan tercapai atau tidak adalah
// fakta tentang layar lama, dan tempatnya di lapisan yang dapat diuji tanpa basis data.
func (s *Service) AdminScorecard(
	ctx context.Context,
	portalAlias string,
	caller reportkpi.Caller,
	input reportkpi.AdminQueryInput,
) (AdminScored, error) {
	query, err := reportkpi.NewAdminQuery(input, caller)
	if err != nil {
		return AdminScored{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return AdminScored{}, err
	}

	totals, err := repo.AdminTotals(ctx, query)
	if err != nil {
		return AdminScored{}, fmt.Errorf("mengambil kartu skor KPI admin: %w", err)
	}

	s.recordAdmin(ctx, "kartu skor KPI admin dibuka", portalAlias, query, 1)

	return AdminScored{
		Card:  reportkpi.BuildScorecard(query.Group, query.Range, totals),
		Query: query,
	}, nil
}

// AdminDetailed adalah satu halaman grid rincian beserta permintaan yang dipakai.
type AdminDetailed struct {
	Page  reportkpi.AdminDetailPage
	Query reportkpi.AdminQuery
}

// AdminDetail mengambil satu halaman grid rincian tab KPI Admin.
func (s *Service) AdminDetail(
	ctx context.Context,
	portalAlias string,
	caller reportkpi.Caller,
	input reportkpi.AdminQueryInput,
	page reportkpi.Pagination,
) (AdminDetailed, error) {
	query, err := reportkpi.NewAdminQuery(input, caller)
	if err != nil {
		return AdminDetailed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return AdminDetailed{}, err
	}

	result, err := repo.AdminDetail(ctx, query, page)
	if err != nil {
		return AdminDetailed{}, fmt.Errorf("mengambil rincian KPI admin: %w", err)
	}

	s.recordAdmin(ctx, "rincian KPI admin dibuka", portalAlias, query, len(result.Rows))
	return AdminDetailed{Page: result, Query: query}, nil
}

// recordAdmin menuliskan jejak satu pembukaan tab KPI Admin.
//
// Alasannya sama dengan tab Adjuster, dan di sini sedikit lebih tajam: yang dinilai adalah
// SATU orang yang dinamai di kartu skornya — koordinator tim admin. Laporan yang menilai
// satu orang dan dapat dibuka siapa pun menuntut jejak siapa yang membukanya (`D-59`).
func (s *Service) recordAdmin(
	ctx context.Context,
	message string,
	portalAlias string,
	query reportkpi.AdminQuery,
	rows int,
) {
	if s.logger == nil {
		return
	}

	s.logger.InfoContext(ctx, message,
		slog.String("modul", "report-kpi"),
		slog.String("tab", reportkpi.TabAdmin),
		slog.String("kelompok", string(query.Group)),
		slog.String("koordinator", reportkpi.AdminIdentityFor(query.Group).Coordinator),
		slog.String("periode_dari", query.Range.From),
		slog.String("periode_sampai", query.Range.To),
		slog.String("pemanggil", query.Caller.Login),
		slog.String("portal", portalAlias),
		slog.Int("baris", rows),
	)
}
