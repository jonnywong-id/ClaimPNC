// Package usecase mengorkestrasi pengelolaan Master Dominan Factor: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterdominanfactor
// dan tidak tahu apa pun soal HTTP maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian: `Database/PEGA_M_DOMINAN_FACTOR.prc` hanya
// mengenal INSERT dan UPDATE — cabang ketiganya tidak ada — dan layar Pega pun tidak
// punya tombolnya. Menghapus satu baris akan membuat setiap klaim yang menyimpan ID itu
// di `T_CLAIM_DOMINANFACTOR` kehilangan artinya, sehingga laporan Outstanding per Cabang
// menampilkan faktor yang hilang tanpa penjelasan (`ADR-0012`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterdominanfactor"
)

// Service mengelola master faktor dominan di atas seam penyimpanan PER PORTAL.
type Service struct {
	repoSelector masterdominanfactor.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterdominanfactor.RepoSelector
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterdominanfactor/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterdominanfactor/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// List mengembalikan seluruh faktor dominan.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan alasan yang dapat diperiksa:
// layar Pega pun memuat seluruhnya sekaligus — `RDB List/GetDataDominanFactor-SQL.xml`
// adalah `SELECT ID, NAME FROM pooldata.M_DOMINAN_FACTOR` **tanpa satu pun klausa
// WHERE, ORDER BY, maupun pembatas baris**. Daftar acuan seperti ini berisi puluhan
// baris dan bertambah beberapa baris per tahun; menambahkan paginasi server akan
// menambah kerumitan yang tidak menyelesaikan satu pun masalah nyata.
//
// Layar yang datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
func (s *Service) List(ctx context.Context, portalAlias string) ([]masterdominanfactor.DominantFactor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterdominanfactor/usecase: membaca daftar faktor dominan: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu faktor dominan.
//
// Ia menggantikan `Activity/SetDominanFactor-Act.xml`, yang mengisi halaman `TempFactor`
// dari baris yang dipilih di grid sebelum form Ubah dibuka.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (masterdominanfactor.DominantFactor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}

	factor, err := repo.Get(ctx, trim(id))
	if err != nil {
		if errors.Is(err, masterdominanfactor.ErrNotFound) {
			return masterdominanfactor.DominantFactor{}, err
		}
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/usecase: membaca faktor %q: %w", id, err)
	}
	return factor, nil
}

// Create menyisipkan faktor baru dan mengembalikannya lengkap dengan ID yang dibuat
// penyimpanan.
//
// ID TIDAK diterima dari pemanggil. Sistem lama pun demikian: layar mengirim
// `TempFactor.Status = "Insert"` dan procedure yang menentukan nomornya
// (`Database/PEGA_M_DOMINAN_FACTOR.prc:11`). Menerima ID dari luar akan membuat dua
// faktor berbeda dapat memperebutkan nomor yang sama.
func (s *Service) Create(ctx context.Context, portalAlias, name string) (masterdominanfactor.DominantFactor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}

	if err := masterdominanfactor.NewValidationError(masterdominanfactor.CheckName(name)); err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}

	// Tidak ada pemeriksaan keunikan nama di sini, dan itu disengaja: nama ganda
	// DITERIMA di modul ini (keputusan Work Owner 2026-09-20). Perbandingannya dengan
	// Master Status Klaim, yang justru menolaknya, dijelaskan di masterdominanfactor.CheckName.
	factor, err := repo.Insert(ctx, trim(name))
	if err != nil {
		// ErrIDTaken datang dari penegakan di basis data saat dua penyimpanan tiba
		// bersamaan dan `max(ID)+1` mengeluarkan nomor yang sama. Ia diteruskan apa
		// adanya supaya pengguna melihat sebab yang benar, bukan 500.
		if errors.Is(err, masterdominanfactor.ErrIDTaken) {
			return masterdominanfactor.DominantFactor{}, err
		}
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/usecase: menambah faktor dominan: %w", err)
	}
	return factor, nil
}

// Update mengganti nama faktor yang sudah ada.
//
// Hanya nama yang dapat berubah. ID bersifat tetap seumur hidup baris itu — layar Pega
// menandainya read-only, dan mengubahnya akan memutus setiap baris
// `T_CLAIM_DOMINANFACTOR` yang menyimpan ID tersebut.
func (s *Service) Update(ctx context.Context, portalAlias, id, name string) (masterdominanfactor.DominantFactor, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}
	id = trim(id)

	if err := masterdominanfactor.NewValidationError(masterdominanfactor.CheckName(name)); err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}

	factor, err := repo.Update(ctx, id, trim(name))
	if err != nil {
		if errors.Is(err, masterdominanfactor.ErrNotFound) {
			return masterdominanfactor.DominantFactor{}, err
		}
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/usecase: mengubah faktor %q: %w", id, err)
	}
	return factor, nil
}

// trim membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
