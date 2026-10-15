// Package usecase mengorkestrasi modul Inbox Service Center.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan keterbatasan yang berlaku
//	List      mengambil isi satu tab
//
// Tidak ada operasi yang menulis, dan ketiadaannya disengaja — alasannya ada di komentar
// seam inboxservicecenter.Repo: seluruh jalur tulis layar ini bermuara pada stored procedure
// yang sumbernya tidak ada di export (`R-01`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxservicecenter"
)

// Service melayani modul Inbox Service Center.
type Service struct {
	repoSelector inboxservicecenter.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxservicecenter.RepoSelector

	// Logger dipakai memperingatkan hasil pencarian yang sangat besar. Ia boleh nil; bila
	// nil, peringatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Service Center.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxservicecenter/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah keempat tab beserta kolomnya.
	Tabs []inboxservicecenter.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// Limitations menyatakan hal yang belum berjalan penuh beserta alasannya.
	Limitations []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar — bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:        inboxservicecenter.Tabs(),
		DefaultTab:  inboxservicecenter.DefaultTab,
		Limitations: inboxservicecenter.Limitations(),
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxservicecenter.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim —
	// sehingga kotak cari selalu memperlihatkan kata kunci yang BENAR-BENAR dipakai.
	Query inboxservicecenter.Query
}

// LargeSearchWarning adalah jumlah baris yang, begitu terlampaui pada hasil pencarian,
// dicatat sebagai peringatan di log.
//
// Ia TIDAK memotong hasil dan tidak mengubah perilaku apa pun — memotongnya akan menyalahi
// `P-5`, karena sistem lama memang mengembalikan seluruh baris yang cocok saat mencari. Yang
// dilakukannya hanya membuat akibat perilaku itu terlihat operator sebelum ia terlihat
// sebagai layar yang menggantung.
const LargeSearchWarning = 2000

// List mengambil isi satu tab.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxservicecenter.Caller,
	input inboxservicecenter.QueryInput,
	page inboxservicecenter.Pagination,
) (Listed, error) {
	query, err := inboxservicecenter.NewQuery(input, caller)
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

	if s.logger != nil && !query.Paginated() && len(result.Items) > LargeSearchWarning {
		s.logger.Warn(
			"satu pencarian Inbox Service Center mengembalikan sangat banyak baris",
			slog.String("tab", query.Tab.Code),
			slog.String("nama_tab", query.Tab.Name),
			slog.Int("jumlah_baris", len(result.Items)),
			slog.Int("ambang", LargeSearchWarning),
			slog.String("sebab",
				"paginasi dimatikan saat mencari — ditiru apa adanya dari Pega (P-5)"),
		)
	}

	return Listed{Page: result, Query: query}, nil
}
