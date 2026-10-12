// Package usecase mengorkestrasi modul Inbox Investigator.
//
// Ia yang mengetahui urutan langkah; bentuk datanya ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
//
// # Kenapa berkasnya bernama browse, bukan manage
//
// Modul yang mengelola menamainya `manage.go` — menambah, menyimpan, memutuskan. BERKAS INI
// hanya MEMBACA: ia melayani daftar antrean dan ekspor. Menyimpan hasil investigasi ada di
// investigate.go, bersebelahan. Nama berkas yang menjanjikan pengelolaan akan membuat
// pembaca mencari jalur simpan di tempat yang salah.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// Service adalah pintu masuk seluruh perkara Inbox Investigator.
type Service struct {
	repoSelector          inboxinvestigator.RepoSelector
	investigationSelector inboxinvestigator.InvestigationRepoSelector
	clock                 func() time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan antrean milik satu portal entitas. Wajib.
	RepoSelector inboxinvestigator.RepoSelector

	// InvestigationSelector memilih penyimpanan hasil investigasi milik portal yang sama.
	// Wajib.
	//
	// Ia TERPISAH dari RepoSelector, mengikuti pemisahan seam-nya di domain: yang pertama
	// hanya membaca, yang ini menulis.
	InvestigationSelector inboxinvestigator.InvestigationRepoSelector

	// Clock dapat diganti untuk pengujian. Kosong berarti jam sistem dalam UTC.
	//
	// # Modul ini sempat dinyatakan tidak memerlukan jam, dan itu benar saat itu
	//
	// Selama ia hanya membaca antrean, tidak ada satu pun nilainya yang lahir dari waktu
	// kini — kolom "Lama Masuk Inbox" menampilkan tanggal survei apa adanya, bukan durasi.
	//
	// Formulir investigasi mengubahnya: TIGA nilai lahir dari waktu kini — Tanggal
	// Investigasi saat formulir dibuka, lalu `InvestTfDate` dan `AnalystTransferDate` saat
	// disimpan. Jam yang dapat diganti membuat ketiganya dapat diuji, dan memastikan satu
	// permintaan tidak menghasilkan tiga waktu yang berbeda sepersekian detik.
	Clock func() time.Time
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang: rakitan
// yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxinvestigator/usecase: RepoSelector wajib diisi")
	}
	if o.InvestigationSelector == nil {
		return nil, errors.New(
			"inboxinvestigator/usecase: InvestigationSelector wajib diisi")
	}
	return &Service{
		repoSelector:          o.RepoSelector,
		investigationSelector: o.InvestigationSelector,
		clock:                 o.Clock,
	}, nil
}

// List mengembalikan isi inbox investigator satu portal.
//
// Portal dipilih LEBIH DULU, sebelum satu baris pun dibaca. Portal yang tidak dapat dilayani
// harus ditolak sebagai penolakan portal — bukan sebagai kegagalan membaca antrean, yang
// akan membuat pengguna menduga antreannya kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) (inboxinvestigator.Page, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxinvestigator.Page{}, err
	}

	page, err := repo.List(ctx, inboxinvestigator.Filter{Keyword: keyword}.Clean())
	if err != nil {
		return inboxinvestigator.Page{}, fmt.Errorf("membaca inbox investigator: %w", err)
	}
	return page, nil
}

// Export mengembalikan isi berkas Export Data Investigation satu portal.
//
// # Ia TIDAK memanggil List, dan itu disengaja
//
// Berkas ini bukan "daftar yang sedang tampil, dalam bentuk CSV". Sumber barisnya berbeda —
// klaim menurut rentang `INVESTIGATOR_TF_DATE`, bukan antrean workbasket — sehingga
// menurunkannya dari daftar akan menghasilkan berkas yang isinya BERBEDA dari berkas Pega
// tanpa satu pun tanda. Rinciannya di kepala kueri `investigator_export`.
//
// Penyaringnya diperiksa di domain, bukan di sini; lihat ExportFilter.Validate. Galatnya
// diteruskan APA ADANYA, tanpa dibungkus, supaya lapisan transport dapat mengenalinya
// dengan errors.Is dan menjawabnya 400 — bukan 500.
func (s *Service) Export(
	ctx context.Context,
	portalAlias string,
	filter inboxinvestigator.ExportFilter,
) ([]inboxinvestigator.ExportRow, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	if err := filter.Validate(); err != nil {
		return nil, err
	}

	rows, err := repo.Export(ctx, filter)
	if err != nil {
		if errors.Is(err, inboxinvestigator.ErrExportFilterInvalid) {
			return nil, err
		}
		return nil, fmt.Errorf("menyusun ekspor data investigasi: %w", err)
	}
	return rows, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum kueri dijalankan.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("inboxinvestigator/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}
