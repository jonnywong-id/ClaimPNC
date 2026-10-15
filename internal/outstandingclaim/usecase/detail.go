// Package usecase mengorkestrasi modul Outstanding Claim.
//
// Dua operasi:
//
//	Layout  menyerahkan susunan layar — kelompok isian, grid, dan yang terhalang
//	Detail  mengambil isi satu klaim
//
// Tidak ada operasi yang menulis. Flow Action `OutstandingClaim` di Pega menyimpan kembali
// objek kerjanya, tetapi selama masa paralel tabel itu dimiliki Pega (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/outstandingclaim"
)

// Service melayani modul Outstanding Claim.
type Service struct {
	repoSelector outstandingclaim.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector outstandingclaim.RepoSelector

	// Logger boleh nil; bila nil, jejak pembukaan tidak ditulis dan tidak ada yang gagal
	// karenanya.
	//
	// Bahwa ia BOLEH nil bukan berarti ia opsional di produksi: lihat Detail.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Outstanding Claim.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("outstandingclaim/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Layout adalah susunan layar yang tidak bergantung isi klaim.
type Layout struct {
	// Groups adalah kesepuluh kelompok beserta isian dan gridnya, berurutan.
	Groups []outstandingclaim.Group

	// Grids adalah susunan setiap grid, termasuk yang terhalang.
	Grids []outstandingclaim.Grid
}

// Layout menyerahkan susunan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: susunan layar sama
// di seluruh entitas, karena ia bentuk layar — bukan data entitas.
func (s *Service) Layout() Layout {
	return Layout{
		Groups: outstandingclaim.Groups(),
		Grids:  outstandingclaim.GridList(),
	}
}

// Detail mengambil isi satu klaim.
//
// # Kenapa setiap pembukaan dicatat, termasuk yang wajar
//
// Karena tidak ada satu pun penyaring yang membatasi klaim mana yang boleh dibuka seseorang —
// di Pega pun tidak (lihat outstandingclaim.NewQuery). Nomor klaimnya berurutan, sehingga
// siapa pun yang sudah masuk dapat menelusuri seluruh klaim treaty di portalnya satu per
// satu.
//
// Pencatatan BUKAN kendali dan tidak diklaim sebagai kendali. Kendalinya adalah pemeriksaan
// kewenangan menu, `TKT-F3-005`, yang belum ada. Sampai itu ada, jejak inilah satu-satunya
// hal yang membuat penelusuran seperti itu dapat dilihat setelah terjadi.
//
// Dicatat pada tingkat Info, bukan Warn: membuka rincian klaim adalah pekerjaan sehari-hari,
// dan menandainya sebagai peringatan akan membuat peringatan berhenti dibaca.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller outstandingclaim.Caller,
	claimID string,
) (outstandingclaim.Detail, error) {
	query, err := outstandingclaim.NewQuery(claimID, caller)
	if err != nil {
		return outstandingclaim.Detail{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return outstandingclaim.Detail{}, err
	}

	detail, err := repo.Find(ctx, query)
	if err != nil {
		// ErrNotFound diteruskan APA ADANYA, tanpa dibungkus konteks tambahan. Lapisan
		// transport memetakannya ke 404, dan pembungkusan membuat `errors.Is` di sana
		// bergantung pada bentuk kalimat pembungkusnya.
		if errors.Is(err, outstandingclaim.ErrNotFound) {
			return outstandingclaim.Detail{}, err
		}
		return outstandingclaim.Detail{},
			fmt.Errorf("mengambil rincian klaim %s: %w", query.ClaimID, err)
	}

	if s.logger != nil {
		s.logger.Info("rincian klaim treaty dibuka",
			slog.String("modul", "outstanding-claim"),
			slog.String("no_klaim", query.ClaimID),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
		)
	}

	return detail, nil
}
