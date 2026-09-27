// Package usecase mengorkestrasi modul Inbox Investigator.
//
// Ia yang mengetahui urutan langkah; bentuk datanya ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
//
// # Kenapa berkasnya bernama browse, bukan manage
//
// Modul yang mengelola menamainya `manage.go` — menambah, menyimpan, memutuskan. Modul ini
// hanya MEMBACA: mengambil pekerjaan dari antrean dan mencatat hasil investigasi terjadi di
// layar kerja yang belum dibangun, bukan di sini. Nama berkas yang menjanjikan pengelolaan
// akan membuat pembaca berikutnya mencari jalur simpan yang memang tidak ada.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxinvestigator"
)

// Service adalah pintu masuk seluruh perkara Inbox Investigator.
type Service struct {
	repoSelector inboxinvestigator.RepoSelector
}

// Options adalah bahan pembentuk Service.
//
// TANPA Clock, dan ketiadaannya adalah hasil koreksi. Modul ini sempat memilikinya karena
// kolom "Lama Masuk Inbox" dibaca sebagai durasi terhadap sekarang; penelusuran sel per sel
// membuktikan kolom itu menampilkan tanggal survei apa adanya. Tidak ada satu pun nilai di
// modul ini yang bergantung pada jam dinding.
//
// TANPA Logger pula: modul ini tidak mengubah apa pun, sehingga tidak ada peristiwa yang
// perlu dicatat. Menerima bahan yang tidak pernah dipakai menyesatkan pembaca berikutnya.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector inboxinvestigator.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang: rakitan
// yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxinvestigator/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan isi inbox investigator satu portal.
//
// Portal dipilih LEBIH DULU, sebelum satu baris pun dibaca. Portal yang tidak dapat dilayani
// harus ditolak sebagai penolakan portal — bukan sebagai kegagalan membaca antrean, yang
// akan membuat pengguna menduga antreannya kosong.
func (s *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) (inboxinvestigator.Page, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxinvestigator.Page{}, err
	}

	page, err := repo.List(ctx, inboxinvestigator.Filter{Keyword: keyword}.Clean())
	if err != nil {
		return inboxinvestigator.Page{}, fmt.Errorf("membaca inbox investigator: %w", err)
	}
	return page, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum kueri dijalankan.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("inboxinvestigator/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}
