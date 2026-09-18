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

	"claim-pnc/internal/statusprogres"
)

// Layanan adalah pintu masuk seluruh perkara master status progres.
type Layanan struct {
	pemilihRepo statusprogres.PemilihRepo
}

// Opsi adalah bahan pembentuk Layanan.
type Opsi struct {
	// PemilihRepo memilih penyimpanan milik satu portal entitas. Wajib.
	PemilihRepo statusprogres.PemilihRepo
}

// LayananBaru membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func LayananBaru(o Opsi) (*Layanan, error) {
	if o.PemilihRepo == nil {
		return nil, errors.New("statusprogres/usecase: PemilihRepo wajib diisi")
	}
	return &Layanan{pemilihRepo: o.PemilihRepo}, nil
}

// Daftar mengembalikan seluruh status progres milik satu portal.
func (l *Layanan) Daftar(ctx context.Context, portalAlias string) ([]statusprogres.StatusProgres, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.Daftar(ctx)
}

// Ambil mengembalikan satu status progres milik satu portal.
func (l *Layanan) Ambil(ctx context.Context, portalAlias, id string) (statusprogres.StatusProgres, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return statusprogres.StatusProgres{}, err
	}
	return repo.Ambil(ctx, id)
}

// Tambah menyisipkan satu status progres baru dan mengembalikan baris tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diturunkan dari isi tabel portal yang
// bersangkutan (statusprogres.FormatNomor). Nomor urut karena itu berdiri sendiri per
// entitas — dua portal dapat memiliki ID yang sama untuk status yang berbeda, persis
// seperti sistem lama, karena setiap entitas punya basis datanya sendiri (ADR-0030).
func (l *Layanan) Tambah(ctx context.Context, portalAlias string, isian statusprogres.Isian) (statusprogres.StatusProgres, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return statusprogres.StatusProgres{}, err
	}

	bersih := isian.Bersihkan()
	if err := bersih.Periksa(); err != nil {
		return statusprogres.StatusProgres{}, err
	}
	return repo.SisipBaru(ctx, bersih)
}

// Ubah menyimpan perubahan Nama dan KodePosisi pada baris yang sudah ada.
//
// ID tidak pernah ikut berubah. Di layar lama pun ia begitu: `UpdateStatusProgress1_act`
// memuat baris lalu menandai modalnya "Update", dan `UpdateStatusProgress1_sql`
// memakai ID_PROGRESS hanya sebagai penyaring `WHERE`, tidak pernah sebagai kolom yang
// di-`SET`. Membiarkannya berubah akan memutus baris `GCNM_PROGRESS_CLAIM` dan
// `GCNM_MST_PROGRESS` yang sudah merujuk ID lamanya.
func (l *Layanan) Ubah(ctx context.Context, portalAlias, id string, isian statusprogres.Isian) (statusprogres.StatusProgres, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return statusprogres.StatusProgres{}, err
	}

	bersih := isian.Bersihkan()
	if err := bersih.Periksa(); err != nil {
		return statusprogres.StatusProgres{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari
	// "baris ada tetapi nilainya sama persis". UPDATE yang mengenai nol baris tidak
	// membedakan keduanya, dan menjawab "tidak ditemukan" untuk penyimpanan yang
	// sebenarnya berhasil akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Ambil(ctx, id); err != nil {
		return statusprogres.StatusProgres{}, err
	}

	diperbarui := statusprogres.StatusProgres{
		ID:         id,
		Nama:       bersih.Nama,
		KodePosisi: bersih.KodePosisi,
	}
	if err := repo.Perbarui(ctx, diperbarui); err != nil {
		return statusprogres.StatusProgres{}, err
	}
	return diperbarui, nil
}

// Posisi mengembalikan daftar posisi klaim untuk dropdown di layar.
//
// Ia dilayani dari sini, bukan disalin ke frontend, supaya keempat nilainya hidup di
// SATU tempat. Frontend yang memuat daftarnya sendiri akan menjadi tempat kedua yang
// harus diingat saat daftarnya kelak pindah menjadi master data `F-4`.
func (l *Layanan) Posisi() []statusprogres.Posisi {
	return statusprogres.DaftarPosisi()
}

// PastikanPortalSiap memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Layanan) PastikanPortalSiap(portalAlias string) error {
	if _, err := l.pemilihRepo(portalAlias); err != nil {
		return fmt.Errorf("statusprogres/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
