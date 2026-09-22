// Package usecase mengorkestrasi modul Inbox Admin.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan isi dropdown penyaring
//	List      mengambil isi satu tab
//
// Tidak ada operasi yang menulis. Layar ini membaca antrean kerja; yang mengubahnya adalah
// layar registrasi, estimasi, dan akseptasi — masing-masing modulnya sendiri.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxadmin"
)

// Service melayani modul Inbox Admin.
type Service struct {
	repoSelector inboxadmin.RepoSelector
	clock        inboxadmin.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxadmin.RepoSelector
	Clock        inboxadmin.Clock

	// Logger dipakai memperingatkan hasil yang sangat besar. Ia boleh nil; bila nil,
	// peringatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Admin.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxadmin/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxadmin/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah kedelapan tab beserta kolomnya.
	Tabs []inboxadmin.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// BusinessLines adalah isi dropdown penyaring lini bisnis.
	BusinessLines []inboxadmin.BusinessLine

	// DisabledTabs adalah tab sistem lama yang sengaja tidak dibangun.
	//
	// Ia dikirim ke layar, bukan disembunyikan, supaya pengguna yang mencari tab
	// "Not Answered" memperoleh jawaban alih-alih menduga modulnya belum selesai.
	DisabledTabs []inboxadmin.DisabledTab
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:          inboxadmin.Tabs(),
		DefaultTab:    inboxadmin.DefaultTab,
		BusinessLines: inboxadmin.BusinessLines(),
		DisabledTabs:  inboxadmin.DisabledTabs,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxadmin.Page

	// Query adalah permintaan setelah divalidasi dan disesuaikan dengan kemampuan tab.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim:
	// tab yang tidak mendukung pencarian mengembalikan kata kunci kosong, dan kotak cari
	// yang terlanjur terisi karena itu dapat dibersihkan.
	Query inboxadmin.Query
}

// List mengambil isi satu tab.
//
// # Kenapa seluruh baris ditarik lebih dulu, lalu dipotong di sini
//
// Keputusan Work Owner 2026-09-20: paginasi layar ini direplikasi apa adanya. Penjelasan
// lengkapnya ada di inboxadmin.Slice, termasuk konsekuensi yang diterima secara sadar.
//
// Yang ditambahkan di sini hanyalah PERINGATAN — bukan pemotongan. Memotong hasil akan
// menyalahi keputusan itu; memperingatkan tidak mengubah apa pun dan membuat akibatnya
// terlihat operator sebelum terlihat sebagai aplikasi yang kehabisan memori.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxadmin.Caller,
	input inboxadmin.QueryInput,
	page inboxadmin.Pagination,
) (Listed, error) {
	query, err := inboxadmin.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	rows, err := repo.List(ctx, query)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	if s.logger != nil && len(rows) > inboxadmin.LargeResultWarning {
		s.logger.Warn(
			"satu permintaan Inbox Admin menarik sangat banyak baris",
			slog.String("tab", query.Tab.Code),
			slog.String("nama_tab", query.Tab.Name),
			slog.Int("jumlah_baris", len(rows)),
			slog.Int("ambang", inboxadmin.LargeResultWarning),
			slog.String("sebab",
				"paginasi direplikasi apa adanya dari Pega — seluruh baris ditarik lalu dipotong di aplikasi"),
			slog.String("penyaring_cabang",
				"belum aktif — menunggu API pengganti DB Link HRD (R-03)"),
		)
	}

	now := s.clock.Now()
	for i := range rows {
		rows[i] = rows[i].WithAging(now)
	}

	return Listed{Page: inboxadmin.Slice(rows, page), Query: query}, nil
}
