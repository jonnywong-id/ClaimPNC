// Package usecase mengorkestrasi modul Inbox Progress Claim.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar region, kolomnya, isi dropdown, dan kontrol yang mati
//	List      mengambil isi satu region
//
// Tidak ada operasi yang menulis. Region yang menulis di layar lama — Approval Progress
// Klaim dan tombol Input Progress Claim — berada di luar lingkup atas keputusan Work Owner
// 2026-09-21, karena tabel yang akan ditulisnya masih dimiliki Pega (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxprogressclaim"
)

// Service melayani modul Inbox Progress Claim.
type Service struct {
	repoSelector inboxprogressclaim.RepoSelector
	clock        inboxprogressclaim.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxprogressclaim.RepoSelector
	Clock        inboxprogressclaim.Clock

	// Logger dipakai memperingatkan rekap yang sangat besar. Ia boleh nil; bila nil,
	// peringatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Progress Claim.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxprogressclaim/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxprogressclaim/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Views adalah keempat region beserta kolomnya.
	Views []inboxprogressclaim.View

	// DefaultView adalah region yang dimuat pertama kali.
	DefaultView string

	// BusinessLines adalah isi dropdown lini bisnis pada rekap per PIC.
	BusinessLines []inboxprogressclaim.BusinessLine

	// DeadControls adalah kontrol yang digambar tetapi tidak menyaring apa pun.
	//
	// Ia dikirim ke layar, bukan disembunyikan: pengguna yang menekan penyaring lalu
	// melihat hasil yang tidak berubah akan melaporkannya sebagai kerusakan, berulang
	// kali, sampai ada yang menjelaskan bahwa memang begitu di sistem lama.
	DeadControls []inboxprogressclaim.DeadControl

	// PageSize adalah ukuran halaman bawaan pada region yang dipaginasi.
	PageSize int
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar region dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Views:         inboxprogressclaim.Views(),
		DefaultView:   inboxprogressclaim.DefaultView,
		BusinessLines: inboxprogressclaim.BusinessLines(),
		DeadControls:  inboxprogressclaim.DeadControls,
		PageSize:      inboxprogressclaim.DefaultPageSize,
	}
}

// Listed adalah isi satu region beserta permintaan yang benar-benar dipakai.
//
// # Kenapa satu bentuk membawa dua jenis baris
//
// Karena keempat region dilayani satu endpoint, dan bentuk barisnya ditentukan
// `Query.View.Kind`. Memecahnya menjadi dua endpoint akan memaksa layar memilih endpoint
// sebelum ia tahu region mana yang diminta — padahal yang menentukannya adalah metadata
// dari server.
//
// Hanya SATU dari kedua isian di bawah yang terisi pada satu jawaban; region Evaluasi tidak
// mengisi keduanya.
type Listed struct {
	// Query adalah permintaan setelah divalidasi dan disesuaikan dengan kemampuan region.
	//
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim:
	// region yang tidak mendukung pencarian mengembalikan kata kunci kosong, sehingga
	// kotak yang terlanjur terisi dapat dibersihkan.
	Query inboxprogressclaim.Query

	// Claims terisi bila region ini berbentuk KindClaim.
	Claims inboxprogressclaim.ClaimPage

	// PICRows terisi bila region ini berbentuk KindPIC.
	PICRows []inboxprogressclaim.PICSummary
}

// LargeRecapWarning adalah jumlah baris rekap per PIC yang, begitu terlampaui, dicatat
// sebagai peringatan di log.
//
// Rekap ini TIDAK dipaginasi, mengikuti sistem lama yang juga tidak memaginasinya. Itu aman
// selama barisnya satu per petugas — dan ambang ini yang membuat asumsi itu terlihat bila
// suatu saat tidak lagi benar.
const LargeRecapWarning = 500

// List mengambil isi satu region.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxprogressclaim.Caller,
	input inboxprogressclaim.QueryInput,
	page inboxprogressclaim.Pagination,
) (Listed, error) {
	query, err := inboxprogressclaim.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	// Region Evaluasi dijawab TANPA menyentuh basis data. Ia memang tidak punya kueri di
	// sistem lama, dan memanggil repo untuknya berarti mengarang kueri yang tidak pernah
	// ada.
	if query.View.Kind == inboxprogressclaim.KindEmpty {
		return Listed{Query: query}, nil
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	now := s.clock.Now()

	switch query.View.Kind {
	case inboxprogressclaim.KindClaim:
		claims, err := repo.ListClaims(ctx, query.ClaimQuery(now), page)
		if err != nil {
			return Listed{}, fmt.Errorf("mengambil isi bagian %s: %w", query.View.Code, err)
		}
		return Listed{Query: query, Claims: claims}, nil

	case inboxprogressclaim.KindPIC:
		rows, err := repo.ListPICSummary(ctx, query.PICQuery(now))
		if err != nil {
			return Listed{}, fmt.Errorf("mengambil isi bagian %s: %w", query.View.Code, err)
		}

		if s.logger != nil && len(rows) > LargeRecapWarning {
			s.logger.Warn(
				"rekap Progress Klaim per PIC menarik sangat banyak baris",
				slog.String("bagian", query.View.Code),
				slog.Int("jumlah_baris", len(rows)),
				slog.Int("ambang", LargeRecapWarning),
				slog.String("sebab",
					"rekap ini tidak dipaginasi, mengikuti sistem lama yang juga tidak memaginasinya"),
			)
		}

		return Listed{Query: query, PICRows: rows}, nil

	default:
		// Tidak dapat terjadi selama setiap View punya Kind yang dikenal. Ia tetap
		// dijawab galat, bukan senarai kosong: senarai kosong akan terbaca sebagai
		// "memang tidak ada datanya".
		return Listed{}, fmt.Errorf(
			"inboxprogressclaim/usecase: bentuk baris %q belum dilayani", query.View.Kind)
	}
}
