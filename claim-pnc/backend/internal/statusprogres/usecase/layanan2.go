package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/statusprogres"
)

// Layanan2 adalah pintu masuk seluruh perkara master status progres tingkat 2.
//
// Ia terpisah dari Layanan tingkat 1 meski keduanya berada di paket yang sama, karena
// keduanya memilih penyimpanan yang berbeda (PemilihRepo versus PemilihRepo2). Yang
// disatukan adalah paket domainnya, bukan layanannya — satu layanan yang melayani dua
// tabel akan menerima dua pemilih repo dan bercabang di setiap method.
type Layanan2 struct {
	pemilihRepo statusprogres.PemilihRepo2

	// pemilihInduk dipakai HANYA untuk menyusun dropdown "Status Progres 1" di layar.
	//
	// Pemeriksaan keberadaan induk saat menyimpan TIDAK memakai ini: ia dikerjakan di
	// dalam Repo2.SisipBaru, di dalam transaksi yang sama dengan penyisipannya. Memeriksa
	// di sini lebih dulu hanya akan memindahkan jaraknya — induk tetap dapat hilang di
	// antara pemeriksaan dan penyisipan.
	pemilihInduk statusprogres.PemilihRepo
}

// Opsi2 adalah bahan pembentuk Layanan2.
type Opsi2 struct {
	// PemilihRepo memilih penyimpanan tingkat 2 milik satu portal entitas. Wajib.
	PemilihRepo statusprogres.PemilihRepo2

	// PemilihInduk memilih penyimpanan tingkat 1 milik portal yang sama. Wajib.
	PemilihInduk statusprogres.PemilihRepo
}

// Layanan2Baru membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func Layanan2Baru(o Opsi2) (*Layanan2, error) {
	if o.PemilihRepo == nil {
		return nil, errors.New("statusprogres/usecase: PemilihRepo tingkat 2 wajib diisi")
	}
	if o.PemilihInduk == nil {
		return nil, errors.New("statusprogres/usecase: PemilihInduk wajib diisi")
	}
	return &Layanan2{pemilihRepo: o.PemilihRepo, pemilihInduk: o.PemilihInduk}, nil
}

// Daftar mengembalikan seluruh status progres tingkat 2 milik satu portal.
func (l *Layanan2) Daftar(ctx context.Context, portalAlias string) ([]statusprogres.StatusProgres2, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.Daftar(ctx)
}

// Ambil mengembalikan satu status progres tingkat 2 milik satu portal.
func (l *Layanan2) Ambil(ctx context.Context, portalAlias, id string) (statusprogres.StatusProgres2, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return statusprogres.StatusProgres2{}, err
	}
	return repo.Ambil(ctx, id)
}

// DaftarInduk mengembalikan pilihan dropdown "Status Progres 1".
//
// Asalnya `RDB List/BrowseMstProgress1-SQL.xml`, yang membaca kolom yang sama persis
// dengan daftar tingkat 1 — karena itu ia dilayani repo tingkat 1 yang sudah ada, bukan
// kueri baru. Satu tabel, satu cara membacanya.
func (l *Layanan2) DaftarInduk(ctx context.Context, portalAlias string) ([]statusprogres.StatusProgres, error) {
	repo, err := l.pemilihInduk(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.Daftar(ctx)
}

// Tambah menyisipkan satu status progres tingkat 2 baru dan mengembalikan baris
// tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diturunkan dari isi tabel portal yang
// bersangkutan (statusprogres.FormatNomor2). Begitu pula NamaInduk — ia disalin dari
// baris induk yang dibaca repo, bukan dikirim layar. Mengirimkannya dari layar berarti
// mempercayai peramban untuk menyebut nama induk dengan benar, padahal nilainya sudah
// ada di basis data.
func (l *Layanan2) Tambah(ctx context.Context, portalAlias string, isian statusprogres.Isian2) (statusprogres.StatusProgres2, error) {
	repo, err := l.pemilihRepo(portalAlias)
	if err != nil {
		return statusprogres.StatusProgres2{}, err
	}

	bersih := isian.Bersihkan()
	if err := bersih.Periksa(); err != nil {
		return statusprogres.StatusProgres2{}, err
	}
	return repo.SisipBaru(ctx, bersih)
}

// PastikanPortalSiap memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Layanan2) PastikanPortalSiap(portalAlias string) error {
	if _, err := l.pemilihRepo(portalAlias); err != nil {
		return fmt.Errorf("statusprogres/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// Catatan: TIDAK ADA method Ubah maupun Hapus di sini, dan itu bukan pekerjaan yang
// belum selesai.
//
// Sistem lama tidak memiliki satu pun pernyataan yang mengubah atau menghapus isi
// POOLDATA.GCNM_MST_PROGRESS setelah barisnya tersimpan; yang tampak seperti
// penyuntingan ternyata menulis ke tabel lain dengan parameter yang tidak pernah diisi.
// Alasan lengkapnya beserta buktinya ada pada doc comment statusprogres.Repo2.
