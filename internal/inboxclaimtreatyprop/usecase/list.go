// Package usecase mengorkestrasi modul Inbox Claim Treaty Prop.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu tab
//
// Tidak ada operasi yang menulis. Layar ini membaca antrean; pembuatan klaim treaty masih
// dimiliki Pega selama masa paralel (`P-1`, keputusan Work Owner 2026-09-21).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// Service melayani modul Inbox Claim Treaty Prop.
type Service struct {
	repoSelector inboxclaimtreatyprop.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxclaimtreatyprop.RepoSelector

	// Logger boleh nil; bila nil, peringatannya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Claim Treaty Prop.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxclaimtreatyprop/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah ketiga tab beserta kolomnya, termasuk yang terhalang.
	Tabs []inboxclaimtreatyprop.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	//
	// Ia dikirim ke layar, bukan disimpan sebagai komentar, supaya pengguna yang
	// membandingkan kedua layar berdampingan memperoleh jawaban alih-alih melaporkannya
	// sebagai kerusakan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:               inboxclaimtreatyprop.Tabs(),
		DefaultTab:         inboxclaimtreatyprop.DefaultTab,
		PlannedDifferences: inboxclaimtreatyprop.PlannedDifferences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxclaimtreatyprop.Page

	// Query adalah permintaan setelah divalidasi dan disesuaikan dengan kemampuan tab.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim:
	// tab yang tidak mendukung "See All Claim" mengembalikan false, dan centang yang
	// terlanjur menyala karena itu dapat dimatikan.
	Query inboxclaimtreatyprop.Query
}

// List mengambil isi satu tab.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxclaimtreatyprop.Caller,
	input inboxclaimtreatyprop.QueryInput,
	page inboxclaimtreatyprop.Pagination,
) (Listed, error) {
	query, err := inboxclaimtreatyprop.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	// Membuka antrean bersama dicatat, bukan dilarang.
	//
	// "See All Claim" melepas penyaring kepemilikan, sehingga petugas melihat nama
	// tertanggung pada pekerjaan yang bukan miliknya. Itu memang perilaku sistem lama dan
	// tetap dibawa — tetapi sampai pemeriksaan peran ada (`TKT-F3-005`), jejaknya di log
	// adalah satu-satunya hal yang menyatakan siapa yang memakainya.
	if s.logger != nil && query.SeeAll {
		s.logger.Info(
			"antrean treaty dibuka tanpa penyaring kepemilikan",
			slog.String("modul", "inbox-claim-treaty-prop"),
			slog.String("tab", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
		)
	}

	return Listed{Page: result, Query: query}, nil
}
