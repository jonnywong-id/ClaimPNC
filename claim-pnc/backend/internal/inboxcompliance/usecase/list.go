// Package usecase mengorkestrasi modul Inbox Compliance.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab beserta kolomnya
//	List      mengambil satu halaman isi sebuah tab
//
// Tidak ada operasi yang menulis. Layar ini membaca antrean kerja; pemeriksaan kepatuhan,
// penerusan ke Post Audit, dan penutupan klaim terjadi di layar lain.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxcompliance"
)

// Service melayani modul Inbox Compliance.
type Service struct {
	repoSelector inboxcompliance.RepoSelector
	clock        inboxcompliance.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxcompliance.RepoSelector
	Clock        inboxcompliance.Clock

	// Logger boleh nil; bila nil, tidak ada yang dicatat dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Compliance.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxcompliance/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxcompliance/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah kedua tab beserta kolomnya, termasuk yang belum dapat dilayani.
	Tabs []inboxcompliance.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:       inboxcompliance.Tabs(),
		DefaultTab: inboxcompliance.DefaultTab,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxcompliance.Page

	// Query adalah permintaan setelah divalidasi. Layar menggambar keadaan tabnya dari
	// sini, bukan dari isian yang ia kirim — sehingga tab yang kosong pada permintaan
	// tetap tergambar sebagai tab bawaan yang benar-benar dilayani.
	Query inboxcompliance.Query
}

// List mengambil satu halaman isi sebuah tab.
//
// # Kenapa paginasinya diserahkan ke repo
//
// Karena layar ini memotong halamannya di BASIS DATA, bukan di aplikasi — berbeda dari
// modul Inbox Admin. Alasan lengkapnya ada di dokumentasi paket inboxcompliance, bagian
// "Kenapa paginasinya BERBEDA dari modul Inbox Admin".
//
// Akibatnya di sini: tidak ada peringatan hasil-sangat-besar seperti pada Inbox Admin, dan
// memang tidak diperlukan — jumlah baris yang sampai ke memori aplikasi dibatasi
// Pagination.Size, yang tidak pernah melebihi MaxPageSize.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	input inboxcompliance.QueryInput,
	page inboxcompliance.Pagination,
) (Listed, error) {
	query, err := inboxcompliance.NewQuery(input)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page.Normalize())
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	// Kolom Aging (tab Compliance) dan OutStanding (tab Post Audit) dihitung di sini,
	// bukan di kueri, supaya SELURUH baris satu halaman memakai satu jam yang sama.
	// Menghitungnya per baris di basis data membuat baris pertama dan baris terakhir
	// diukur terhadap saat yang berbeda — selisihnya tidak terlihat, tetapi membuat hasil
	// uji tidak dapat diulang.
	now := s.clock.Now()
	for i := range result.Items {
		result.Items[i] = result.Items[i].WithElapsed(now)
	}

	return Listed{Page: result, Query: query}, nil
}
