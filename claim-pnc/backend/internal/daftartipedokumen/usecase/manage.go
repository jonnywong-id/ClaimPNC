// Package usecase mengorkestrasi modul Daftar Tipe Dokumen.
//
// Tugasnya tiga, dan tidak lebih: memilih Repo milik portal yang sedang aktif, menyusun
// jejak simpan (siapa dan kapan), lalu memanggil repo. Ia tidak tahu apa pun tentang HTTP
// maupun SQL.
//
// # Kenapa lapisan ini nyaris kosong
//
// Karena memang tidak ada aturan isian yang perlu diorkestrasi. Work Owner menetapkan
// 2026-09-21 layar ini meniru Pega apa adanya, tanpa validasi — dan layar Pega memang
// tidak memeriksa apa pun sebelum memanggil procedure penyimpannya.
//
// Lapisannya tetap ada, dan itu disengaja: ia tempat pemilihan portal dijalankan, ia
// tempat jam dibaca lewat satu seam, dan ia tempat aturan pertama akan tinggal bila kelak
// diputuskan. Menghapusnya berarti handler HTTP memanggil repo langsung, dan aturan
// berikutnya akan mendarat di lapisan transport.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/platform/clock"
)

// Service adalah pintu masuk seluruh perkara daftar tipe dokumen.
type Service struct {
	repoSelector daftartipedokumen.RepoSelector
	clock        clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector daftartipedokumen.RepoSelector

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
		return nil, errors.New("daftartipedokumen/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("daftartipedokumen/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock}, nil
}

// List mengembalikan seluruh tipe dokumen milik satu portal.
//
// Tidak dipaginasi di server. Report Definition lama pun memuat seluruhnya sekaligus
// dengan batas `pyMaxRecords=500`, dan isi master ini hanya berupa daftar kategori
// dokumen — puluhan baris, bukan puluhan ribu. Paginasi 50 baris yang terlihat pengguna
// (`pyPageSize=50` pada layar lama) dikerjakan di layar atas baris yang sudah di tangan.
func (s *Service) List(ctx context.Context, portalAlias string) ([]daftartipedokumen.DocumentType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu tipe dokumen.
//
// Ia menggantikan `CNMSetListDocumentType_act`, yang menyalin baris yang dipilih ke
// halaman `TempDcol` untuk diisikan ke form.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftartipedokumen.DocumentType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumen.DocumentType{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan tipe dokumen baru dan mengembalikan baris tersimpannya.
//
// # Sentinel "UnknownID" tidak dibawa
//
// Sistem lama membedakan tambah dari ubah dengan mengirim teks `"UnknownID"` sebagai ID
// (`Activity/CNMInsertListDocumentType_act-Act.xml` langkah 1 —
// `@If(TempDcol.ID!="",TempDcol.ID,"UnknownID")`), lalu procedure memeriksanya untuk
// memilih INSERT atau UPDATE. Di sini keduanya adalah dua operasi yang berbeda sejak dari
// rutenya, sehingga tidak ada nilai ajaib yang perlu dikenali siapa pun — dan tidak ada
// kemungkinan sebuah tipe dokumen benar-benar bernama "UnknownID" menabrak mekanismenya.
//
// Perlu dicatat: procedure lama TIDAK HANYA membandingkan sentinel itu, ia juga
// MENGGANTINYA di dalam dokumen JSON yang disimpan
// (`replace(DataPega,'UnknownID',id_lst_doc_type)`, `PEGA_LST_DOC_TYPE.prc:23`) — sehingga
// teks "UnknownID" yang kebetulan diketik petugas di isian mana pun akan ikut tertimpa
// ID baris itu. Cacat itu ikut hilang bersama JSON-nya.
//
// ID tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang bersangkutan.
// Nomor urutnya karena itu berdiri sendiri per entitas — dua portal dapat menerbitkan ID
// yang sama untuk tipe dokumen yang berbeda, persis seperti sistem lama, karena setiap
// entitas punya basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	input daftartipedokumen.Input,
	by string,
) (daftartipedokumen.DocumentType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumen.DocumentType{}, err
	}
	return repo.InsertNew(ctx, input.Clean(), s.editor(by))
}

// Update mengganti isi tipe dokumen yang sudah ada.
//
// Hanya nama dan status proses yang dapat berubah. ID bersifat tetap seumur hidup baris
// itu — procedure lama pun hanya memakainya sebagai penyaring `WHERE`, tidak pernah
// sebagai kolom yang di-`SET` (`Database/PEGA_LST_DOC_TYPE.prc:34`).
func (s *Service) Update(
	ctx context.Context,
	portalAlias, id string,
	input daftartipedokumen.Input,
	by string,
) (daftartipedokumen.DocumentType, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumen.DocumentType{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari "baris ada
	// tetapi isinya sama persis". UPDATE yang mengenai nol baris tidak membedakan
	// keduanya, dan menjawab "tidak ditemukan" atas penyimpanan yang sebenarnya berhasil
	// akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Get(ctx, id); err != nil {
		return daftartipedokumen.DocumentType{}, err
	}

	clean := input.Clean()
	updated := daftartipedokumen.DocumentType{
		ID:            strings.TrimSpace(id),
		Type:          clean.Type,
		ProcessStatus: clean.ProcessStatus,
	}
	if err := repo.Update(ctx, updated, s.editor(by)); err != nil {
		return daftartipedokumen.DocumentType{}, err
	}
	return updated, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("daftartipedokumen/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
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
func (s *Service) editor(by string) daftartipedokumen.Editor {
	return daftartipedokumen.Editor{
		Identity: strings.TrimSpace(by),
		At:       s.clock.Now(),
	}
}
