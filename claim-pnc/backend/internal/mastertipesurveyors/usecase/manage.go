// Package usecase mengorkestrasi pengelolaan Master Tipe Surveyors: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Tugasnya tiga, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, lalu memanggil Repo. Ia tidak tahu apa pun tentang HTTP
// maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian:
//
//   - Layar Pega tidak punya tombol hapus — `Section/GridSurveyors-Section.xml` hanya
//     memuat **Tambah** dan **Refresh**.
//   - `Database/PEGA_M_SURVEYORS.prc` hanya mengenal INSERT dan UPDATE; tidak ada satu
//     pun pernyataan DELETE terhadap M_SURVEYORS di seluruh export — sudah diperiksa.
//   - Menghapus satu tipe akan membuat setiap baris D_SURVEYORS yang menyimpan kodenya
//     kehilangan golongannya, dan ketiga kueri `BrowseSurveyorType*` yang mematok kode
//     `1002`/`1003`/`1004` berhenti mengembalikan apa pun.
//
// Persis alasan `ADR-0012` menetapkan master tidak dihapus permanen.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/mastertipesurveyors"
)

// Service mengelola master tipe surveyor di atas seam penyimpanan per portal.
type Service struct {
	repoSelector mastertipesurveyors.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastertipesurveyors.RepoSelector
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastertipesurveyors/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan seluruh tipe surveyor milik satu portal, terurut menurut kode.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan angka: isinya EMPAT baris pada
// portal ASM per 2026-09-19, dan tipe surveyor adalah golongan yang nyaris tidak pernah
// bertambah. Report Definition lama pun memuat seluruhnya sekaligus dengan batas
// `pyMaxRecords=500`. Menambahkan paginasi server di sini akan menambah kerumitan yang
// tidak menyelesaikan satu pun masalah nyata.
func (s *Service) List(ctx context.Context, portalAlias string) ([]mastertipesurveyors.SurveyorType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("mastertipesurveyors/usecase: membaca daftar tipe surveyor: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu tipe surveyor.
//
// Ia menggantikan `SetSurveryorsValue_act(msurveyid)`, yang menjalankan Report Definition
// `SelectVMSurveyors_RD` lalu menyalin hasilnya ke halaman `TempSurveyors`.
func (s *Service) Get(ctx context.Context, portalAlias, code string) (mastertipesurveyors.SurveyorType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	surveyorType, err := repo.Get(ctx, trim(code))
	if err != nil {
		if errors.Is(err, mastertipesurveyors.ErrNotFound) {
			return mastertipesurveyors.SurveyorType{}, err
		}
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/usecase: membaca tipe surveyor %q: %w", code, err)
	}
	return surveyorType, nil
}

// Create menyisipkan tipe baru dan mengembalikannya lengkap dengan kode yang dibuat
// penyimpanan.
//
// Kode TIDAK diterima dari pemanggil. Sistem lama pun demikian: layar mengirim sentinel
// `"UnknownID"` dan procedure yang menentukan kodenya
// (`Activity/CNMInsertSurveyors_act-Act.xml` → `PEGA_M_SURVEYORS`). Menerima kode dari
// luar akan membuat dua tipe berbeda dapat memperebutkan nomor yang sama — dan karena
// kodenya dipatok langsung di tiga kueri Pega, bentrokannya akan terbawa sampai ke
// pemilihan surveyor.
func (s *Service) Create(ctx context.Context, portalAlias, description string) (mastertipesurveyors.SurveyorType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	if err := mastertipesurveyors.NewValidationError(mastertipesurveyors.CheckDescription(description)); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}
	if err := ensureDescriptionFree(ctx, repo, description, ""); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	saved, err := repo.Insert(ctx, trim(description))
	if err != nil {
		// Ketiga galat di bawah datang dari penegakan di basis data, yang menang atas
		// pemeriksaan di atas bila dua permintaan tiba bersamaan. Ia diteruskan apa adanya
		// supaya pengguna melihat sebab yang benar.
		if errors.Is(err, mastertipesurveyors.ErrDescriptionTaken) ||
			errors.Is(err, mastertipesurveyors.ErrCodeTaken) ||
			errors.Is(err, mastertipesurveyors.ErrNoSite) {
			return mastertipesurveyors.SurveyorType{}, err
		}
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/usecase: menambah tipe surveyor: %w", err)
	}
	return saved, nil
}

// Update mengganti deskripsi tipe yang sudah ada.
//
// Hanya deskripsi yang dapat berubah. Kode bersifat tetap seumur hidup baris itu — layar
// Pega menandainya `pyEditOptions=Read-only`, dan mengubahnya akan memutus setiap baris
// D_SURVEYORS yang menyimpan kode tersebut beserta ketiga kueri `BrowseSurveyorType*`
// yang mematoknya langsung.
func (s *Service) Update(ctx context.Context, portalAlias, code, description string) (mastertipesurveyors.SurveyorType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}
	code = trim(code)

	if err := mastertipesurveyors.NewValidationError(mastertipesurveyors.CheckDescription(description)); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}
	// Kode diperiksa lebih dulu supaya mengubah tipe yang tidak ada dijawab "tidak
	// ditemukan", bukan "nama sudah dipakai" yang menyesatkan.
	if _, err := repo.Get(ctx, code); err != nil {
		if errors.Is(err, mastertipesurveyors.ErrNotFound) {
			return mastertipesurveyors.SurveyorType{}, err
		}
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/usecase: membaca tipe surveyor %q: %w", code, err)
	}
	if err := ensureDescriptionFree(ctx, repo, description, code); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	saved, err := repo.Update(ctx, code, trim(description))
	if err != nil {
		if errors.Is(err, mastertipesurveyors.ErrNotFound) || errors.Is(err, mastertipesurveyors.ErrDescriptionTaken) {
			return mastertipesurveyors.SurveyorType{}, err
		}
		return mastertipesurveyors.SurveyorType{}, fmt.Errorf("mastertipesurveyors/usecase: mengubah tipe surveyor %q: %w", code, err)
	}
	return saved, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastertipesurveyors/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// ensureDescriptionFree menolak deskripsi yang sudah dipakai tipe lain.
//
// exceptCode dikosongkan saat menambah, dan diisi kode yang sedang diubah saat mengubah —
// tanpa itu, menyimpan ulang tipe tanpa mengubah namanya akan ditolak karena bentrok
// dengan dirinya sendiri.
//
// Pemeriksaan ini adalah KENYAMANAN, bukan jaminan: dua permintaan yang tiba bersamaan
// dapat sama-sama lolos di sini. Jaminannya ada di indeks unik basis data
// (`UX_M_SURVEYORS_DESC` pada migrasi 0003), dan galatnya diterjemahkan kembali menjadi
// ErrDescriptionTaken oleh repo. Yang di sini hanya membuat pesannya tiba lebih cepat dan
// lebih jelas.
//
// Selama migrasi 0003 belum dijalankan DBA, pemeriksaan inilah SATU-SATUNYA yang menolak
// nama ganda — dan itu keadaan yang dicatat terbuka, bukan yang disembunyikan.
func ensureDescriptionFree(ctx context.Context, repo mastertipesurveyors.Repo, description, exceptCode string) error {
	list, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("mastertipesurveyors/usecase: memeriksa keunikan tipe surveyor: %w", err)
	}

	wanted := mastertipesurveyors.DescriptionKey(description)
	for _, t := range list {
		if t.Code == exceptCode {
			continue
		}
		if mastertipesurveyors.DescriptionKey(t.Description) == wanted {
			return mastertipesurveyors.ErrDescriptionTaken
		}
	}
	return nil
}

// trim membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
