// Package usecase mengorkestrasi modul Daftar Detail Tipe Dokumen.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif, memeriksa
// isian yang dapat diperiksa tanpa menyentuh penyimpanan, menyusun jejak simpan (siapa
// dan kapan), lalu memanggil repo. Ia tidak tahu apa pun tentang HTTP maupun SQL.
//
// # Kenapa lapisan ini tipis
//
// Karena layar lama nyaris tidak memeriksa apa pun sebelum menyimpan — lihat
// `Input.Check` di lapisan domain untuk buktinya. Yang tersisa hanyalah penjaga panjang
// isian, dan itu bentuk kolom, bukan aturan bisnis.
//
// Lapisannya tetap ada, dan itu disengaja: ia tempat pemilihan portal dijalankan, ia
// tempat jam dibaca lewat satu seam, ia tempat dua seam yang berbeda dipilih bersamaan
// untuk satu permintaan, dan ia tempat aturan pertama akan tinggal bila kelak diputuskan.
// Menghapusnya berarti handler HTTP memanggil repo langsung, dan aturan berikutnya akan
// mendarat di lapisan transport.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/platform/clock"
)

// Service adalah pintu masuk seluruh perkara detail tipe dokumen.
type Service struct {
	repoSelector      daftardetailtipedokumen.RepoSelector
	referenceSelector daftardetailtipedokumen.ReferenceRepoSelector
	clock             clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan detail milik satu portal entitas. Wajib.
	RepoSelector daftardetailtipedokumen.RepoSelector

	// ReferenceSelector memilih pembaca keempat master rujukan. Wajib.
	ReferenceSelector daftardetailtipedokumen.ReferenceRepoSelector

	// Clock memasok waktu yang mengisi TGL_EDIT. Wajib.
	//
	// Ia seam, bukan `time.Now()` yang dipanggil di tempat penyimpanan, karena `F-5`
	// menetapkan waktu dibaca dan dikonversi hanya di satu tempat — dan karena tanpa itu
	// penyimpanan tidak dapat diuji secara deterministik.
	Clock clock.Clock
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("daftardetailtipedokumen/usecase: RepoSelector wajib diisi")
	}
	if o.ReferenceSelector == nil {
		return nil, errors.New("daftardetailtipedokumen/usecase: ReferenceSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("daftardetailtipedokumen/usecase: Clock wajib diisi")
	}
	return &Service{
		repoSelector:      o.RepoSelector,
		referenceSelector: o.ReferenceSelector,
		clock:             o.Clock,
	}, nil
}

// List mengembalikan seluruh rincian milik satu portal, tanpa daftar bisnisnya.
//
// Tidak dipaginasi di server. Report Definition lama pun memuat seluruhnya sekaligus
// dengan batas `pyMaxRecords=500` (`BrowseVLstDetTypeDoc_RD-RD.xml`), dan isi master ini
// berupa daftar rincian dokumen — ratusan baris, bukan puluhan ribu. Penyaringan dan
// pengurutan cukup dikerjakan di layar atas baris yang sudah di tangan.
//
// Batas 500 baris milik Pega sengaja TIDAK ditiru. Ia bukan aturan bisnis melainkan
// pemotongan senyap — laporan lama terpotong tanpa memberi tahu siapa pun
// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), dan menirunya berarti menyembunyikan baris
// yang benar-benar ada dari petugas yang sedang menyuntingnya.
func (s *Service) List(ctx context.Context, portalAlias string) ([]daftardetailtipedokumen.DetailType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu rincian LENGKAP dengan daftar bisnisnya.
//
// Ia menggantikan `Activity/CNMSetDetailTypeDocument_act-Act.xml`, yang mengisi form dari
// baris terpilih lalu memanggil `RDB List/GetLbuDetType-SQL.xml` untuk mengisi grid
// bisnisnya. Dua langkah di sana, satu permintaan di sini — bentuk yang sama dengan modul
// Daftar Detail Dokumen Travel.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftardetailtipedokumen.DetailType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan rincian baru dan mengembalikan baris tersimpannya.
//
// # Sentinel "UnknownID" tidak dibawa
//
// Sistem lama membedakan tambah dari ubah dengan mengirim teks `"UnknownID"` sebagai ID
// (`Activity/CNMInsertDetailTypeDocument_act-Act.xml` langkah 1 —
// `@if(TempDTDoc.ID!="",TempDTDoc.ID,"UnknownID")`), lalu procedure memeriksanya untuk
// memilih INSERT atau UPDATE. Di sini keduanya adalah dua operasi yang berbeda sejak dari
// rutenya, sehingga tidak ada nilai ajaib yang perlu dikenali siapa pun.
//
// Perlu dicatat: procedure lama TIDAK HANYA membandingkan sentinel itu, ia juga
// MENGGANTINYA di dalam dokumen JSON yang disimpan
// (`replace(DataPega,'UnknownID',id_detype_ins)`, `PEGA_LST_DET_TYPE_DOC.prc:22`) —
// sehingga teks "UnknownID" yang kebetulan diketik petugas di isian mana pun akan ikut
// tertimpa ID baris itu. Cacat itu ikut hilang bersama JSON-nya.
//
// ID tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang bersangkutan.
// Nomor urutnya karena itu berdiri sendiri per entitas — dua portal dapat menerbitkan ID
// yang sama untuk rincian yang berbeda, persis seperti sistem lama, karena setiap entitas
// punya basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	input daftardetailtipedokumen.Input,
	by string,
) (daftardetailtipedokumen.DetailType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}
	return repo.InsertNew(ctx, clean, s.editor(by))
}

// Update mengganti isi satu rincian beserta seluruh daftar bisnisnya.
//
// ID bersifat tetap seumur hidup baris itu — procedure lama pun hanya memakainya sebagai
// penyaring `WHERE`, tidak pernah sebagai kolom yang di-`SET`
// (`Database/PEGA_LST_DET_TYPE_DOC.prc:34`).
func (s *Service) Update(
	ctx context.Context,
	portalAlias, id string,
	input daftardetailtipedokumen.Input,
	by string,
) (daftardetailtipedokumen.DetailType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}
	return repo.Update(ctx, id, clean, s.editor(by))
}

// References mengembalikan keempat daftar pilihan yang dibutuhkan form.
//
// Keempatnya dikirim dalam SATU jawaban meski berasal dari empat pemanggilan seam, karena
// form selalu membutuhkan keempatnya bersamaan: tidak ada satu pun isian rujukan yang
// dapat ditampilkan tanpa daftarnya, dan memisahkannya menjadi empat permintaan hanya
// menambah tiga keadaan setengah-siap yang harus dijaga layar.
//
// # Kegagalan salah satunya TIDAK menggagalkan seluruhnya
//
// Keempat kode boleh diketik sendiri — layar lama pun memakai autocomplete yang menerima
// ketikan di luar daftar — sehingga daftar yang gagal dimuat hanya menghilangkan
// kenyamanan memilih, bukan kemampuan menyimpan. Menggagalkan seluruh permintaan karena
// satu master bermasalah akan menutup form yang sebenarnya masih dapat dipakai.
//
// Kegagalannya tetap DILAPORKAN, bukan disembunyikan: senarai yang bersangkutan kosong
// dan namanya masuk ke Unavailable, sehingga layar dapat mengatakan apa yang hilang
// alih-alih menampilkan daftar kosong yang terbaca sebagai "masternya memang kosong".
func (s *Service) References(ctx context.Context, portalAlias string) (References, error) {
	repo, err := s.referenceSelector(portalAlias)
	if err != nil {
		return References{}, err
	}

	result := References{}

	if list, err := repo.ListDocumentTypes(ctx); err != nil {
		result.Unavailable = append(result.Unavailable, ReferenceDocumentType)
	} else {
		result.DocumentTypes = list
	}

	if list, err := repo.ListCausesOfLoss(ctx); err != nil {
		result.Unavailable = append(result.Unavailable, ReferenceCauseOfLoss)
	} else {
		result.CausesOfLoss = list
	}

	if list, err := repo.ListObjectDocuments(ctx); err != nil {
		result.Unavailable = append(result.Unavailable, ReferenceObjectDocument)
	} else {
		result.ObjectDocuments = list
	}

	if list, err := repo.ListBusinesses(ctx); err != nil {
		result.Unavailable = append(result.Unavailable, ReferenceBusiness)
	} else {
		result.Businesses = list
	}

	return result, nil
}

// Nama master rujukan pada daftar Unavailable.
//
// Teks, bukan nomor, supaya layar dapat menyebutkan mana yang hilang tanpa memelihara
// tabel penerjemah — dan supaya penambahan master berikutnya tidak menggeser arti nomor
// yang sudah dipakai klien lama.
const (
	ReferenceDocumentType   = "tipe_dokumen"
	ReferenceCauseOfLoss    = "penyebab_kerugian"
	ReferenceObjectDocument = "objek_dokumen"
	ReferenceBusiness       = "bisnis"
)

// References adalah keempat daftar pilihan beserta catatan mana yang gagal dimuat.
type References struct {
	DocumentTypes   []daftardetailtipedokumen.DocumentTypeOption
	CausesOfLoss    []daftardetailtipedokumen.CauseOfLossOption
	ObjectDocuments []daftardetailtipedokumen.ObjectDocumentOption
	Businesses      []daftardetailtipedokumen.Business

	// Unavailable menyebut master mana yang gagal dibaca. Kosong berarti keempatnya
	// terbaca.
	Unavailable []string
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("daftardetailtipedokumen/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// editor menyusun jejak simpan dari identitas pemanggil dan jam sistem.
//
// Identitas yang kosong TIDAK ditolak, dan itu keputusan sadar: ia hanya mengisi kolom
// jejak, bukan menentukan apa yang tersimpan. Menolak penyimpanan karena identitasnya
// tidak terbaca akan membuat petugas kehilangan isian yang sudah diketik demi kolom yang
// tidak dilihat siapa pun di layar — sementara barisnya sendiri tetap dapat ditelusuri
// lewat ID-nya.
func (s *Service) editor(by string) daftardetailtipedokumen.Editor {
	return daftardetailtipedokumen.Editor{
		Identity: strings.TrimSpace(by),
		At:       s.clock.Now(),
	}
}
