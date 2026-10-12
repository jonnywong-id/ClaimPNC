// Package usecase mengorkestrasi modul Acceptation Claim.
//
// Tiga operasi:
//
//	Metadata  menyerahkan bentuk layar — kelompok, isian, grid, dan selisih terencana
//	Find      mengambil satu rincian akseptasi
//	Submit    menerima perubahan akseptasi
//
// Metadata TIDAK menyentuh basis data sama sekali dan tidak bergantung portal: bentuk layar
// sama di seluruh entitas, karena ia hasil pembacaan export, bukan data entitas.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inputacceptation"
)

// Service melayani modul Acceptation Claim.
type Service struct {
	repoSelector inputacceptation.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inputacceptation.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	//
	// Pada modul INI nil punya harga yang perlu disadari: layar ini tidak menyaring menurut
	// pemanggil, sehingga catatan pembukaan adalah satu-satunya hal yang menyatakan siapa
	// membuka akseptasi klaim siapa.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Acceptation Claim.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inputacceptation/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah bentuk layar yang tidak bergantung isi klaim.
type Metadata struct {
	// Groups adalah kelompok isian beserta grid yang digambar sesudahnya, berurutan.
	Groups []inputacceptation.Group

	// Grids adalah seluruh grid beserta kolomnya.
	Grids []inputacceptation.Grid

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan bentuk layar.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Groups:             inputacceptation.Groups(),
		Grids:              inputacceptation.GridList(),
		PlannedDifferences: inputacceptation.PlannedDifferences,
	}
}

// Find mengambil satu rincian akseptasi.
func (s *Service) Find(
	ctx context.Context,
	portalAlias string,
	caller inputacceptation.Caller,
	claimID string,
) (inputacceptation.Detail, error) {
	q, err := inputacceptation.NewQuery(claimID, caller)
	if err != nil {
		return inputacceptation.Detail{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inputacceptation.Detail{}, err
	}

	detail, err := repo.Find(ctx, q)
	if err != nil {
		if errors.Is(err, inputacceptation.ErrNotFound) {
			return inputacceptation.Detail{}, err
		}
		return inputacceptation.Detail{}, fmt.Errorf("membuka akseptasi %s: %w", q.ClaimID, err)
	}

	// SETIAP pembukaan dicatat beserta pelakunya.
	//
	// Layar ini tidak menyaring menurut pemanggil — di Pega pun tidak — dan nomor klaimnya
	// berurutan, sehingga siapa pun yang sudah masuk dapat membuka akseptasi klaim mana pun
	// di portalnya hanya dengan menaikkan angkanya. Pencatatan BUKAN kendali dan tidak
	// diklaim sebagai kendali; ia yang membuat penyalahgunaannya dapat ditelusuri setelah
	// terjadi, sampai pemeriksaan kewenangan menu ada (`TKT-F3-005`).
	if s.logger != nil {
		s.logger.InfoContext(ctx, "akseptasi klaim treaty non-prop dibuka",
			slog.String("modul", "input-acceptation"),
			slog.String("no_klaim", q.ClaimID),
			slog.String("pemanggil", q.Caller.Login),
			slog.String("portal", portalAlias),
		)
	}

	return detail, nil
}

// Submit menerima perubahan akseptasi.
//
// # Urutannya disengaja: BACA dulu, baru validasi muatan
//
// Kunci teknis objek kerja (`Reference`) diambil dari penyimpanan, BUKAN dari klien. Kunci
// teknis yang datang dari klien adalah kunci yang dapat ditukar klien — dan pada layar yang
// menetapkan Nomor Akseptasi, menukarnya berarti menulisi klaim yang lain.
//
// Pembacaan itu sekaligus memastikan klaimnya memang ada dan memang klaim treaty
// non-proporsional sebelum satu pun nilai diperiksa.
func (s *Service) Submit(
	ctx context.Context,
	portalAlias string,
	caller inputacceptation.Caller,
	claimID string,
	values map[string]string,
	gridRows map[string][]inputacceptation.GridRow,
) error {
	q, err := inputacceptation.NewQuery(claimID, caller)
	if err != nil {
		return err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	existing, err := repo.Find(ctx, q)
	if err != nil {
		if errors.Is(err, inputacceptation.ErrNotFound) {
			return err
		}
		return fmt.Errorf("membaca akseptasi %s sebelum menyimpan: %w", q.ClaimID, err)
	}

	cmd, err := inputacceptation.NewSaveCommand(q, existing.Reference, values, gridRows)
	if err != nil {
		return err
	}

	if err := repo.Save(ctx, cmd); err != nil {
		if errors.Is(err, inputacceptation.ErrWriteNotOwned) {
			return err
		}
		return fmt.Errorf("menyimpan akseptasi %s: %w", q.ClaimID, err)
	}

	if s.logger != nil {
		s.logger.InfoContext(ctx, "akseptasi klaim treaty non-prop disimpan",
			slog.String("modul", "input-acceptation"),
			slog.String("no_klaim", q.ClaimID),
			slog.String("pemanggil", q.Caller.Login),
			slog.String("portal", portalAlias),
		)
	}
	return nil
}
