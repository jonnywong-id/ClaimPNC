// Package usecase mengorkestrasi perkara Master Pasal Kerugian.
//
// Ia yang mengetahui urutan langkah; aturan isian ada di paket domain, dan cara
// membacanya dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"
	"strings"

	"claim-pnc/internal/masterpasal"
)

// Service adalah pintu masuk seluruh perkara Master Pasal Kerugian.
type Service struct {
	repoSelector masterpasal.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpasal.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpasal/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan seluruh pasal kerugian satu portal.
//
// Padanan `PNCGetListPasalDataCOL_Act` pada jalur `Param.idstatusp == "0"`, yang
// menjalankan `GetDataCOLByPasalBisnis_Sql` tanpa penyaring apa pun.
func (l *Service) List(ctx context.Context, portalAlias string) ([]masterpasal.Clause, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return store.List(ctx)
}

// Get mengembalikan satu pasal beserta daftar lini bisnisnya, untuk dimuat ke form.
//
// Padanan jalur `Param.idstatusp` berisi IDDATA pada activity yang sama.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (masterpasal.Clause, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpasal.Clause{}, err
	}
	return store.Get(ctx, strings.TrimSpace(id))
}

// Create menyimpan satu pasal kerugian baru.
//
// Urutannya mengikuti `Activity/CNMInsertPasalDataMaster-Act.xml` pada jalur
// `Param.DeleteFlag != "1"`:
//
//	langkah 1  turunkan sebutan kategori dari kodenya  -> di bawah
//	langkah 3  tolak bila No Pasal kosong              -> Input.Check
//	langkah 4  susun dokumen JSON-nya                  -> repo
//	langkah 5  sisipkan                                -> Repo.Insert
//
// Satu perbedaan urutan yang disengaja: penurunan nomor IDDATA berada DI DALAM
// Repo.Insert, bukan sebagai langkah terpisah di sini. Alasannya ada pada doc comment
// masterpasal.Repo.Insert.
func (l *Service) Create(ctx context.Context, portalAlias string, input masterpasal.Input) (masterpasal.Clause, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpasal.Clause{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpasal.Clause{}, err
	}
	return store.Insert(ctx, clean)
}

// Update menyimpan perubahan pada pasal kerugian yang sudah ada.
//
// Di Pega, menambah dan mengubah adalah SATU jalur: `Pega_D_Pasal_Master` memilih sisip
// atau perbarui berdasarkan ada-tidaknya IDDATA yang dikirim
// (`Database/PEGA_D_PASAL_MASTER.prc:6-8`). Di sini keduanya dipisah menjadi dua method,
// dan itu bukan perubahan perilaku melainkan perubahan letak percabangan: yang memilih
// adalah rute HTTP — `POST` tanpa id, `PUT` dengan id — sehingga "menambah" tidak dapat
// berubah menjadi "mengubah" hanya karena satu nilai tersembunyi ikut terkirim.
func (l *Service) Update(
	ctx context.Context,
	portalAlias, id string,
	input masterpasal.Input,
) (masterpasal.Clause, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpasal.Clause{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpasal.Clause{}, err
	}
	return store.Update(ctx, strings.TrimSpace(id), clean)
}

// Delete membuang satu pasal kerugian secara FISIK.
//
// Padanan `CNMInsertPasalDataMaster(DeleteFlag="1")`. Ia menyupersede `D-66` untuk tabel
// ini atas keputusan Work Owner 2026-09-19; peringatan lengkapnya ada pada doc comment
// masterpasal.Repo.
//
// Tidak ada pemeriksaan isian sama sekali di sini, dan memang tidak ada yang dapat
// diperiksa: yang dibutuhkan hanyalah kunci barisnya.
func (l *Service) Delete(ctx context.Context, portalAlias, id string) error {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return err
	}
	return store.Delete(ctx, strings.TrimSpace(id))
}

// SearchBusiness mencari lini bisnis untuk daftar pilihan di form.
//
// Kata kunci yang lebih pendek dari masterpasal.MinLookupKeyword dijawab daftar KOSONG,
// bukan galat. Pengguna yang baru mengetik satu huruf belum melakukan kesalahan apa pun —
// ia baru mulai mengetik — dan menjawabnya dengan galat akan menyalakan pesan merah pada
// setiap pencarian yang normal.
//
// Pemeriksaannya ada DI SINI, bukan hanya di peramban: frontend memang menahan permintaan
// yang terlalu pendek, tetapi API dapat ditembak langsung, dan `like '%%'` menarik seluruh
// tabel.
func (l *Service) SearchBusiness(ctx context.Context, portalAlias, keyword string) ([]masterpasal.Business, error) {
	store, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := strings.TrimSpace(keyword)
	if len(clean) < masterpasal.MinLookupKeyword {
		return nil, nil
	}
	return store.SearchBusiness(ctx, clean)
}
