// Package usecase mengorkestrasi modul Inbox Claim Treaty Non Prop.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu tab
//
// Ekspor TIDAK menjadi operasi ketiga: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// Tidak ada operasi yang menulis. Layar ini membaca antrean; pembuatan klaim treaty
// non-proporsional masih dimiliki Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// Service melayani modul Inbox Claim Treaty Non Prop.
type Service struct {
	repoSelector inboxclaimtreatynonprop.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxclaimtreatynonprop.RepoSelector

	// Logger boleh nil; bila nil, peringatannya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Claim Treaty Non Prop.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New(
			"inboxclaimtreatynonprop/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah ketiga tab beserta kolomnya, termasuk yang terhalang.
	Tabs []inboxclaimtreatynonprop.Tab

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
		Tabs:               inboxclaimtreatynonprop.Tabs(),
		DefaultTab:         inboxclaimtreatynonprop.DefaultTab,
		PlannedDifferences: inboxclaimtreatynonprop.PlannedDifferences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxclaimtreatynonprop.Page

	// Query adalah permintaan setelah divalidasi dan disesuaikan dengan kemampuan tab.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim:
	// tab yang tidak mendukung sebuah checkbox mengembalikan false, dan centang yang
	// terlanjur menyala karena itu dapat dimatikan.
	Query inboxclaimtreatynonprop.Query
}

// List mengambil isi satu tab.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxclaimtreatynonprop.Caller,
	input inboxclaimtreatynonprop.QueryInput,
	page inboxclaimtreatynonprop.Pagination,
) (Listed, error) {
	query, err := inboxclaimtreatynonprop.NewQuery(input, caller)
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
	// tertanggung dan nama Ceding Co pada pekerjaan yang bukan miliknya. Itu memang
	// perilaku sistem lama dan tetap dibawa — tetapi sampai pemeriksaan peran ada
	// (`TKT-F3-005`), jejaknya di log adalah satu-satunya hal yang menyatakan siapa yang
	// memakainya.
	if s.logger != nil && query.SeeAll {
		s.logger.Info(
			"antrean treaty non-prop dibuka tanpa penyaring kepemilikan",
			slog.String("modul", "inbox-claim-treaty-non-prop"),
			slog.String("tab", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
		)
	}

	return Listed{Page: result, Query: query}, nil
}
