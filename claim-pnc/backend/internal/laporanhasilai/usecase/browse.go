// Package usecase mengorkestrasi perkara Laporan Hasil AI.
//
// Ia yang mengetahui urutan langkah; aturan penyaring ada di paket domain, dan cara
// membacanya dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"

	"claim-pnc/internal/laporanhasilai"
)

// Service adalah pintu masuk seluruh perkara Laporan Hasil AI.
//
// Modul ini BACA-SAJA. Layar lamanya hanya punya dua tombol — "Cari Data" dan "Export To
// Excel" — dan keduanya memanggil activity yang sama, `SearchDataLaporanAI(flagss=2)`.
// Tidak ada Tambah, Simpan, Ubah, maupun Hapus.
type Service struct {
	repoSelector laporanhasilai.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector laporanhasilai.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("laporanhasilai/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Search mengisi grid rincian.
//
// # Urutannya mengikuti `Activity/SearchDataLaporanAI-Act.xml`
//
//	langkah 1   susun klausa tanggal dari kedua isian   -> Filter.Clean + Validate
//	langkah 4   baca barisnya lewat RDB-List            -> Repo.List
//
// # Langkah pencacah activity lama TIDAK punya padanan di sini
//
// Langkah 6-9 pada activity itu mencacah terima/tolak lalu mengisi `TempTotal` — grid
// ringkasan. Pega yang berjalan tidak memiliki grid itu (Work Owner, 2026-10-03), sehingga
// pencacahnya pun tidak dibangun. Lihat doc paket `laporanhasilai`.
//
// # Validasi dijalankan SEKALI, di sini
//
// Bukan di handler dan bukan di repo. Di handler, ia akan terlewat oleh pemanggil lain —
// ekspor CSV, misalnya, yang menempuh jalur berbeda. Di repo, ia akan ditulis dua kali:
// sekali di sqlstore dan sekali di memory, dan keduanya dapat berbeda tanpa ada yang
// memberi tahu.
func (s *Service) Search(
	ctx context.Context,
	portalAlias string,
	filter laporanhasilai.Filter,
	page laporanhasilai.Pagination,
) (laporanhasilai.Page, error) {
	store, clean, err := s.prepare(portalAlias, filter)
	if err != nil {
		return laporanhasilai.Page{}, err
	}
	return store.List(ctx, clean, page.Normalize())
}

// ListForExport membaca satu potong baris untuk ditulis ke berkas CSV.
//
// Ia memakai penyaring dan pembacaan yang SAMA dengan Search — hanya jendelanya yang
// digeser pemanggil. Ekspor yang membaca dengan cara berbeda akan menghasilkan berkas yang
// isinya tidak dapat dicocokkan dengan apa pun di layar.
//
// Ringkasannya TIDAK ikut: `CSVPropHeaders` pada activity lama menyebut sepuluh kolom
// grid rincian, dan tidak satu pun berasal dari grid ringkasan.
func (s *Service) ListForExport(
	ctx context.Context,
	portalAlias string,
	filter laporanhasilai.Filter,
	page laporanhasilai.Pagination,
) (laporanhasilai.Page, error) {
	store, clean, err := s.prepare(portalAlias, filter)
	if err != nil {
		return laporanhasilai.Page{}, err
	}
	return store.List(ctx, clean, page.Normalize())
}

// prepare memilih penyimpanan portal lalu membersihkan dan memeriksa penyaringnya.
//
// Urutannya penting: portal dipilih LEBIH DULU. Permintaan tanpa portal yang sah harus
// ditolak sebagai soal portal, bukan sebagai soal isian tanggal — menukarnya akan membuat
// pengguna sibuk membetulkan tanggal yang sebenarnya sudah benar.
func (s *Service) prepare(
	portalAlias string,
	filter laporanhasilai.Filter,
) (laporanhasilai.Repo, laporanhasilai.Filter, error) {
	store, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, laporanhasilai.Filter{}, err
	}

	clean := filter.Clean()
	if err := clean.Validate(); err != nil {
		return nil, laporanhasilai.Filter{}, err
	}
	return store, clean, nil
}

// EnsurePortalReady memastikan penyimpanan portal dapat dipilih, tanpa membaca apa pun.
//
// Dipakai pemeriksaan kesiapan; ia memisahkan "portal ini belum siap" dari "pembacaannya
// gagal", dua hal yang di mata pengguna terlihat sama tetapi ditangani berbeda.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	_, err := s.repoSelector(portalAlias)
	return err
}
