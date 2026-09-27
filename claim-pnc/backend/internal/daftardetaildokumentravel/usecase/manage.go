// Package usecase mengorkestrasi modul Daftar Detail Dokumen Travel.
//
// Tugasnya dua, dan tidak lebih: memilih Repo milik portal yang sedang aktif, lalu
// memanggilnya. Ia tidak tahu apa pun tentang HTTP maupun SQL.
//
// # Kenapa lapisan ini tipis
//
// Karena memang tidak ada aturan yang perlu diorkestrasi. Work Owner menetapkan layar
// ini meniru Pega apa adanya, tanpa validasi — dan layar Pega memang tidak memeriksa apa
// pun sebelum menyimpan.
//
// Lapisannya tetap ada, dan itu disengaja: ia tempat pemilihan portal dijalankan, ia
// tempat tiga seam yang berbeda dipilih bersamaan untuk satu permintaan, dan ia tempat
// aturan pertama akan tinggal bila kelak diputuskan. Menghapusnya berarti handler HTTP
// memanggil tiga repo langsung, dan aturan berikutnya akan mendarat di lapisan transport.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/daftardetaildokumentravel"
)

// Service adalah pintu masuk seluruh perkara detail dokumen travel.
type Service struct {
	repoSelector     daftardetaildokumentravel.RepoSelector
	documentSelector daftardetaildokumentravel.DocumentRepoSelector
	planSelector     daftardetaildokumentravel.PlanRepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan detail milik satu portal entitas. Wajib.
	RepoSelector daftardetaildokumentravel.RepoSelector

	// DocumentSelector memilih pembaca master dokumen travel (M_DOCTRAVEL). Wajib.
	DocumentSelector daftardetaildokumentravel.DocumentRepoSelector

	// PlanSelector memilih pembaca master plan dan jaminan Travel. Wajib.
	PlanSelector daftardetaildokumentravel.PlanRepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("daftardetaildokumentravel/usecase: RepoSelector wajib diisi")
	}
	if o.DocumentSelector == nil {
		return nil, errors.New("daftardetaildokumentravel/usecase: DocumentSelector wajib diisi")
	}
	if o.PlanSelector == nil {
		return nil, errors.New("daftardetaildokumentravel/usecase: PlanSelector wajib diisi")
	}
	return &Service{
		repoSelector:     o.RepoSelector,
		documentSelector: o.DocumentSelector,
		planSelector:     o.PlanSelector,
	}, nil
}

// List mengembalikan seluruh aturan dokumen milik satu portal, tanpa coverage-nya.
//
// Tidak dipaginasi. Report Definition lama pun memuat seluruhnya sekaligus dengan batas
// `pyMaxRecords=500` (`BrowseLstDocTravel_RD-RD.xml`), dan isi master ini berupa daftar
// aturan dokumen — puluhan baris, bukan puluhan ribu. Penyaringan dan pengurutan cukup
// dikerjakan di layar atas baris yang sudah di tangan.
//
// Batas 500 baris milik Pega sengaja TIDAK ditiru. Ia bukan aturan bisnis melainkan
// pemotongan senyap — laporan lama terpotong tanpa memberi tahu siapa pun
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan menirunya berarti menyembunyikan baris
// yang benar-benar ada dari petugas yang sedang menyuntingnya.
func (s *Service) List(ctx context.Context, portalAlias string) ([]daftardetaildokumentravel.Detail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu aturan LENGKAP dengan daftar coverage-nya.
//
// Ia menggantikan `CNMSetDetailTravelDocument_act`, activity yang mengisi form dari
// baris yang dipilih. Activity itu sendiri TIDAK ADA di export (`R-16`), sehingga yang
// ditiru adalah apa yang dibutuhkan formnya — terbaca dari isian di
// `Section/BrowseDocumentTravel-Section.xml` — bukan urutan langkahnya.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftardetaildokumentravel.Detail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan aturan baru dan mengembalikan baris tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang bersangkutan.
// Nomor urutnya karena itu berdiri sendiri per entitas — dua portal dapat menerbitkan ID
// yang sama untuk aturan yang berbeda, dan itu memang benar karena setiap entitas punya
// basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return repo.InsertNew(ctx, input.Clean())
}

// Update mengganti isi satu aturan beserta seluruh daftar coverage-nya.
func (s *Service) Update(
	ctx context.Context,
	portalAlias, id string,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetaildokumentravel.Detail{}, err
	}
	return repo.Update(ctx, id, input.Clean())
}

// Documents mengembalikan pilihan ID Dokumen dari master induk.
//
// Menggantikan autocomplete `BrowseMstDocTravel_RD` pada isian ID Dokumen, yang
// menampilkan DOCID beserta NAMADOKUMEN-nya.
func (s *Service) Documents(ctx context.Context, portalAlias string) ([]daftardetaildokumentravel.Document, error) {
	repo, err := s.documentSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Plans mengembalikan pilihan Nama Plan.
//
// Menggantikan autocomplete `BrowsePlanTravelMaster_RD`.
func (s *Service) Plans(ctx context.Context, portalAlias string) ([]daftardetaildokumentravel.Plan, error) {
	repo, err := s.planSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListPlans(ctx)
}

// Coverages mengembalikan pilihan Nama Jaminan beserta plan pemiliknya.
//
// Menggantikan autocomplete `SearchCoverageTravel_RD`, yang di Pega disaring server
// lewat parameter `plan`. Di sini penyaringannya dikerjakan layar — alasannya ada di
// komentar PlanRepo.ListCoverages.
func (s *Service) Coverages(ctx context.Context, portalAlias string) ([]daftardetaildokumentravel.CoverageOption, error) {
	repo, err := s.planSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListCoverages(ctx)
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("daftardetaildokumentravel/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
