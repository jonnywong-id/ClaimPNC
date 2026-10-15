// Package usecase mengorkestrasi modul Master Dokumen Travel.
//
// Tugasnya dua, dan tidak lebih: memilih Repo milik portal yang sedang aktif, lalu
// memanggilnya. Ia tidak tahu apa pun tentang HTTP maupun SQL.
//
// # Kenapa lapisan ini nyaris kosong
//
// Karena memang tidak ada aturan yang perlu diorkestrasi. Work Owner menetapkan layar
// ini meniru Pega apa adanya, tanpa validasi — dan layar Pega memang tidak memeriksa
// apa pun sebelum memanggil procedure penyimpannya.
//
// Lapisannya tetap ada, dan itu disengaja: ia tempat pemilihan portal dijalankan, dan
// ia tempat aturan pertama akan tinggal bila kelak diputuskan. Menghapusnya berarti
// handler HTTP memanggil repo langsung, dan aturan berikutnya akan mendarat di lapisan
// transport.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterdokumentravel"
)

// Service adalah pintu masuk seluruh perkara master dokumen travel.
type Service struct {
	repoSelector masterdokumentravel.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterdokumentravel.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterdokumentravel/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan seluruh dokumen travel milik satu portal.
//
// Tidak dipaginasi. Report Definition lama pun memuat seluruhnya sekaligus dengan batas
// `pyMaxRecords=500`, dan isi master ini hanya berupa daftar jenis dokumen — puluhan
// baris, bukan puluhan ribu. Penyaringan dan pengurutan cukup dikerjakan di layar atas
// baris yang sudah di tangan.
func (s *Service) List(ctx context.Context, portalAlias string) ([]masterdokumentravel.TravelDocument, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu dokumen travel.
//
// Ia menggantikan `SetMstDocTravelValue_act(docid)`, yang menjalankan Report Definition
// `BrowseMstDocTravel_RD` lalu menyalin baris yang cocok ke halaman `TempMstDocTravel`
// untuk diisikan ke form.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (masterdokumentravel.TravelDocument, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan dokumen baru dan mengembalikan baris tersimpannya.
//
// # Sentinel "UnknownID" tidak dibawa
//
// Sistem lama membedakan tambah dari ubah dengan mengirim teks `"UnknownID"` sebagai
// DOCID (`Activity/CNMInsertMstDocTravel_act-Act.xml` langkah 1), lalu procedure
// memeriksanya untuk memilih INSERT atau UPDATE. Di sini keduanya adalah dua operasi
// yang berbeda sejak dari rutenya, sehingga tidak ada nilai ajaib yang perlu dikenali
// siapa pun — dan tidak ada kemungkinan sebuah dokumen benar-benar bernama "UnknownID"
// menabrak mekanismenya.
//
// ID tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang bersangkutan.
// Nomor urutnya karena itu berdiri sendiri per entitas — dua portal dapat menerbitkan
// DOCID yang sama untuk dokumen yang berbeda, persis seperti sistem lama, karena setiap
// entitas punya basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(ctx context.Context, portalAlias string, input masterdokumentravel.Input) (masterdokumentravel.TravelDocument, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}
	return repo.InsertNew(ctx, input.Clean())
}

// Update mengganti judul dokumen yang sudah ada.
//
// Hanya judul yang dapat berubah. DOCID bersifat tetap seumur hidup baris itu —
// procedure lama pun hanya memakainya sebagai penyaring `WHERE`, tidak pernah sebagai
// kolom yang di-`SET` (`Database/DOCTRAVEL_CVG.prc:37`).
func (s *Service) Update(ctx context.Context, portalAlias, id string, input masterdokumentravel.Input) (masterdokumentravel.TravelDocument, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari "baris
	// ada tetapi judulnya sama persis". UPDATE yang mengenai nol baris tidak membedakan
	// keduanya, dan menjawab "tidak ditemukan" atas penyimpanan yang sebenarnya berhasil
	// akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Get(ctx, id); err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}

	updated := masterdokumentravel.TravelDocument{ID: id, Name: input.Clean().Name}
	if err := repo.Update(ctx, updated); err != nil {
		return masterdokumentravel.TravelDocument{}, err
	}
	return updated, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterdokumentravel/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
