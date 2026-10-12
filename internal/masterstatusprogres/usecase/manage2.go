package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterstatusprogres"
)

// Service2 adalah pintu masuk seluruh perkara master status progres tingkat 2.
//
// Ia terpisah dari Layanan tingkat 1 meski keduanya berada di paket yang sama, karena
// keduanya memilih penyimpanan yang berbeda (RepoSelector versus RepoSelector2). Yang
// disatukan adalah paket domainnya, bukan layanannya — satu layanan yang melayani dua
// tabel akan menerima dua pemilih repo dan bercabang di setiap method.
type Service2 struct {
	repoSelector masterstatusprogres.RepoSelector2

	// pemilihInduk dipakai HANYA untuk menyusun dropdown "Status Progres 1" di layar.
	//
	// Pemeriksaan keberadaan induk saat menyimpan TIDAK memakai ini: ia dikerjakan di
	// dalam Repo2.InsertNew, di dalam transaksi yang sama dengan penyisipannya. Memeriksa
	// di sini lebih dulu hanya akan memindahkan jaraknya — induk tetap dapat hilang di
	// antara pemeriksaan dan penyisipan.
	parentSelector masterstatusprogres.RepoSelector
}

// Options2 adalah bahan pembentuk Layanan2.
type Options2 struct {
	// RepoSelector memilih penyimpanan tingkat 2 milik satu portal entitas. Wajib.
	RepoSelector masterstatusprogres.RepoSelector2

	// ParentSelector memilih penyimpanan tingkat 1 milik portal yang sama. Wajib.
	ParentSelector masterstatusprogres.RepoSelector
}

// NewService2 membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService2(o Options2) (*Service2, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterstatusprogres/usecase: RepoSelector tingkat 2 wajib diisi")
	}
	if o.ParentSelector == nil {
		return nil, errors.New("masterstatusprogres/usecase: ParentSelector wajib diisi")
	}
	return &Service2{repoSelector: o.RepoSelector, parentSelector: o.ParentSelector}, nil
}

// List mengembalikan seluruh status progres tingkat 2 milik satu portal.
func (l *Service2) List(ctx context.Context, portalAlias string) ([]masterstatusprogres.ProgressStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu status progres tingkat 2 milik satu portal.
func (l *Service2) Get(ctx context.Context, portalAlias, id string) (masterstatusprogres.ProgressStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}
	return repo.Get(ctx, id)
}

// ListParents mengembalikan pilihan dropdown "Status Progres 1".
//
// Asalnya `RDB List/BrowseMstProgress1-SQL.xml`, yang membaca kolom yang sama persis
// dengan daftar tingkat 1 — karena itu ia dilayani repo tingkat 1 yang sudah ada, bukan
// kueri baru. Satu tabel, satu cara membacanya.
func (l *Service2) ListParents(ctx context.Context, portalAlias string) ([]masterstatusprogres.ProgressStatus, error) {
	repo, err := l.parentSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Create menyisipkan satu status progres tingkat 2 baru dan mengembalikan baris
// tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diturunkan dari isi tabel portal yang
// bersangkutan (masterstatusprogres.FormatID2). Begitu pula ParentName — ia disalin dari
// baris induk yang dibaca repo, bukan dikirim layar. Mengirimkannya dari layar berarti
// mempercayai peramban untuk menyebut nama induk dengan benar, padahal nilainya sudah
// ada di basis data.
func (l *Service2) Create(ctx context.Context, portalAlias string, input masterstatusprogres.Input2) (masterstatusprogres.ProgressStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}
	return repo.InsertNew(ctx, clean)
}

// Update menyimpan perubahan nama dan induk, lalu mengembalikan baris tersimpannya.
//
// ID tidak pernah berubah: ia penyaring, bukan isian. Nama induk juga tidak diterima dari
// layar — ia disalin repo dari baris induk yang dibacanya sendiri.
//
// Berbeda dari tingkat 1, barisnya TIDAK dimuat lebih dulu dengan Get. Repo tingkat 2
// sudah membuka transaksi untuk membaca induknya, dan pemeriksaan "baris ada atau tidak"
// dikerjakan di dalam transaksi itu lewat jumlah baris terpengaruh. Memuatnya lebih dulu
// di sini hanya menambah satu perjalanan ke basis data yang jawabannya dapat basi sebelum
// transaksi dimulai.
func (l *Service2) Update(ctx context.Context, portalAlias, id string, input masterstatusprogres.Input2) (masterstatusprogres.ProgressStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterstatusprogres.ProgressStatus2{}, err
	}
	return repo.Update(ctx, id, clean)
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service2) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterstatusprogres/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// Catatan: TIDAK ADA method Ubah maupun Hapus di sini, dan itu bukan pekerjaan yang
// belum selesai.
//
// Sistem lama tidak memiliki satu pun pernyataan yang mengubah atau menghapus isi
// POOLDATA.GCNM_MST_PROGRESS setelah barisnya tersimpan; yang tampak seperti
// penyuntingan ternyata menulis ke tabel lain dengan parameter yang tidak pernah diisi.
// Alasan lengkapnya beserta buktinya ada pada doc comment masterstatusprogres.Repo2.
