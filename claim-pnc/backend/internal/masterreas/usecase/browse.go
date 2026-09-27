// Package usecase mengorkestrasi perkara master reas.
//
// Ia yang mengetahui urutan langkah; bentuk datanya ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
//
// # Kenapa berkasnya bernama browse, bukan manage
//
// Modul lain menamainya `manage.go` karena memang mengelola — menambah, menyimpan,
// memutuskan. Modul ini hanya MEMBACA; lihat banner paket masterreas untuk alasannya. Nama
// berkas yang menjanjikan pengelolaan akan membuat pembaca berikutnya mencari jalur simpan
// yang memang tidak ada.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterreas"
)

// Service adalah pintu masuk seluruh perkara master reas.
type Service struct {
	repoSelector masterreas.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterreas.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// TANPA Clock dan TANPA Logger. `POOLDATA.T_REINSURER` tidak punya satu pun kolom waktu,
// dan modul ini tidak mengubah apa pun — tidak ada peristiwa yang perlu dicatat maupun
// distempel. Menerima keduanya "untuk jaga-jaga" berarti menerima bahan yang tidak pernah
// dipakai, dan itu menyesatkan pembaca berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterreas/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan member reasuransi satu portal yang cocok dengan penyaring.
//
// Cakupannya SELURUH baris tabel entitas itu; alasannya beserta keterbatasan buktinya ada
// pada doc comment masterreas.Filter.
func (l *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) ([]masterreas.Member, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx, masterreas.Filter{Keyword: keyword}.Clean())
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum kueri dijalankan.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterreas/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}
