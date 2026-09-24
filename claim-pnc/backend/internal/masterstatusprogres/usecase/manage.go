// Package usecase mengorkestrasi modul Master Status Progres.
//
// Tugasnya tiga, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, lalu memanggil Repo. Ia tidak tahu apa pun tentang
// HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterstatusprogres"
)

// Service adalah pintu masuk seluruh perkara master status progres.
type Service struct {
	repoSelector masterstatusprogres.RepoSelector
}

// Options adalah bahan pembentuk Layanan.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterstatusprogres.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterstatusprogres/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan seluruh status progres milik satu portal.
func (l *Service) List(ctx context.Context, portalAlias string) ([]masterstatusprogres.ProgressStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu status progres milik satu portal.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (masterstatusprogres.ProgressStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan satu status progres baru dan mengembalikan baris tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diturunkan dari isi tabel portal yang
// bersangkutan (masterstatusprogres.FormatID). Nomor urut karena itu berdiri sendiri per
// entitas — dua portal dapat memiliki ID yang sama untuk status yang berbeda, persis
// seperti sistem lama, karena setiap entitas punya basis datanya sendiri (ADR-0030).
func (l *Service) Create(ctx context.Context, portalAlias string, input masterstatusprogres.Input) (masterstatusprogres.ProgressStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}
	return repo.InsertNew(ctx, clean)
}

// Update menyimpan perubahan Name dan PositionCode pada baris yang sudah ada.
//
// ID tidak pernah ikut berubah. Di layar lama pun ia begitu: `UpdateStatusProgress1_act`
// memuat baris lalu menandai modalnya "Update", dan `UpdateStatusProgress1_sql`
// memakai ID_PROGRESS hanya sebagai penyaring `WHERE`, tidak pernah sebagai kolom yang
// di-`SET`. Membiarkannya berubah akan memutus baris `GCNM_PROGRESS_CLAIM` dan
// `GCNM_MST_PROGRESS` yang sudah merujuk ID lamanya.
func (l *Service) Update(ctx context.Context, portalAlias, id string, input masterstatusprogres.Input) (masterstatusprogres.ProgressStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari
	// "baris ada tetapi nilainya sama persis". UPDATE yang mengenai nol baris tidak
	// membedakan keduanya, dan menjawab "tidak ditemukan" untuk penyimpanan yang
	// sebenarnya berhasil akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Get(ctx, id); err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}

	updated := masterstatusprogres.ProgressStatus{
		ID:           id,
		Name:         clean.Name,
		PositionCode: clean.PositionCode,
	}
	if err := repo.Update(ctx, updated); err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}
	return updated, nil
}

// Position mengembalikan daftar posisi klaim untuk dropdown di layar.
//
// Ia dilayani dari sini, bukan disalin ke frontend, supaya nilainya hidup di SATU
// tempat. Frontend yang memuat daftarnya sendiri akan menjadi tempat kedua yang
// harus diingat saat daftarnya kelak pindah menjadi master data `F-4`.
func (l *Service) Position() []masterstatusprogres.Position {
	return masterstatusprogres.ListPositions()
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterstatusprogres/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
