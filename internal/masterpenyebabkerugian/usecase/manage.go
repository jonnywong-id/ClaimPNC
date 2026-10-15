// Package usecase mengorkestrasi pengelolaan Master Penyebab Kerugian: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterpenyebabkerugian
// dan tidak tahu apa pun soal HTTP maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian: `Database/PEGA_M_CAUSE_OF_LOSS.prc` hanya mengenal
// INSERT dan UPDATE — cabang ketiganya tidak ada — dan layar Pega pun tidak punya
// tombolnya. Menghapus satu golongan akan membuat setiap baris `D_CAUSE_OF_LOSS` yang
// menyimpan `M_COL_ID` itu kehilangan induknya (`ADR-0012`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpenyebabkerugian"
)

// Service mengelola master penyebab kerugian di atas seam penyimpanan PER PORTAL.
type Service struct {
	repoSelector masterpenyebabkerugian.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpenyebabkerugian.RepoSelector
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya terjadi
// saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpenyebabkerugian/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterpenyebabkerugian/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// List mengembalikan seluruh golongan penyebab kerugian.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan alasan yang dapat diperiksa:
// layar Pega pun memuat seluruhnya sekaligus — `Report Definition/BrowseVMCauseOfLoss_RD`
// tidak memasang satu pun penyaring maupun pembatas baris. Daftar acuan seperti ini berisi
// puluhan baris dan bertambah beberapa baris per tahun; menambahkan paginasi server akan
// menambah kerumitan yang tidak menyelesaikan satu pun masalah nyata.
//
// Layar yang datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
func (s *Service) List(ctx context.Context, portalAlias string) ([]masterpenyebabkerugian.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterpenyebabkerugian/usecase: membaca daftar penyebab kerugian: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu golongan penyebab kerugian.
//
// Ia menggantikan pengisian halaman `TempCauseOfLoss` dari baris yang dipilih di grid
// sebelum form Ubah dibuka (`Activity/CNMInsertCauseOfLoss_act-Act.xml` langkah pertama).
func (s *Service) Get(ctx context.Context, portalAlias, id string) (masterpenyebabkerugian.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}

	cause, err := repo.Get(ctx, trim(id))
	if err != nil {
		if errors.Is(err, masterpenyebabkerugian.ErrNotFound) {
			return masterpenyebabkerugian.CauseOfLoss{}, err
		}
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/usecase: membaca penyebab kerugian %q: %w", id, err)
	}
	return cause, nil
}

// Create menyisipkan golongan baru dan mengembalikannya lengkap dengan ID yang dibuat
// penyimpanan.
//
// ID TIDAK diterima dari pemanggil. Sistem lama pun demikian: layar mengirim
// `TempCauseOfLoss.M_COL_ID = "UnknownID"` dan procedure yang menentukan nomornya
// (`Database/PEGA_M_CAUSE_OF_LOSS.prc:20`). Menerima ID dari luar akan membuat dua
// golongan berbeda dapat memperebutkan nomor yang sama.
func (s *Service) Create(ctx context.Context, portalAlias, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}

	if err := masterpenyebabkerugian.NewValidationError(masterpenyebabkerugian.CheckDescription(description)); err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}

	// Tidak ada pemeriksaan keunikan deskripsi di sini, dan itu disengaja: deskripsi
	// ganda DITERIMA di modul ini (keputusan Work Owner 2026-09-20). Perbandingannya
	// dengan Master Status Klaim, yang justru menolaknya, dijelaskan di
	// masterpenyebabkerugian.CheckDescription.
	cause, err := repo.Insert(ctx, trim(description))
	if err != nil {
		// Ketiga galat ini diteruskan apa adanya supaya pengguna melihat sebab yang benar,
		// bukan 500 tanpa sebab yang terbaca.
		switch {
		case errors.Is(err, masterpenyebabkerugian.ErrIDTaken),
			errors.Is(err, masterpenyebabkerugian.ErrNoSite):
			return masterpenyebabkerugian.CauseOfLoss{}, err
		}
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/usecase: menambah penyebab kerugian: %w", err)
	}
	return cause, nil
}

// Update mengganti deskripsi golongan yang sudah ada.
//
// Hanya deskripsi yang dapat berubah. ID bersifat tetap seumur hidup baris itu — layar
// Pega menandainya read-only, dan mengubahnya akan memutus setiap baris `D_CAUSE_OF_LOSS`
// yang bernaung di bawahnya. ID lama pun tidak disentuh: ia jejak sejarah, bukan isian.
func (s *Service) Update(ctx context.Context, portalAlias, id, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}
	id = trim(id)

	if err := masterpenyebabkerugian.NewValidationError(masterpenyebabkerugian.CheckDescription(description)); err != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, err
	}

	cause, err := repo.Update(ctx, id, trim(description))
	if err != nil {
		if errors.Is(err, masterpenyebabkerugian.ErrNotFound) {
			return masterpenyebabkerugian.CauseOfLoss{}, err
		}
		return masterpenyebabkerugian.CauseOfLoss{}, fmt.Errorf("masterpenyebabkerugian/usecase: mengubah penyebab kerugian %q: %w", id, err)
	}
	return cause, nil
}

// trim membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
