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
	"log/slog"

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

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Ia dipakai HANYA untuk mengisi log. `POOLDATA.T_REINSURER` tidak punya satu pun kolom
// pencatat pelaku maupun stempel waktu, sehingga log adalah satu-satunya tempat "siapa yang
// mengubah surel ini" terekam.
//
// Itu BUKAN pengganti jejak audit `S-5`, dan tidak diklaim demikian. Pada tabel yang
// menentukan ke mana pemberitahuan klaim dikirim, ketiadaan jejak audit adalah keterbatasan
// yang dicatat — bukan yang ditutupi.
type Actor struct {
	Login string
}

// Save mengubah surel satu member reas.
//
// # Hanya SURAT ELEKTRONIK yang berubah
//
// Alasannya ada pada doc comment masterreas.Input dan masterreas.Repo.Update: `UPDATEREAS`
// pada baris yang sudah ada hanya menyentuh kolom `EMAIl`. Ketiga kolom kunci beserta
// `LOGIN` dan `COUNTRY` tidak dapat dikirim klien sama sekali.
//
// # Baris yang tidak ada DITOLAK, bukan disisipkan
//
// Berbeda dari `UPDATEREAS`, yang menyisipkan baris baru bila kuncinya tidak ditemukan.
// Perbedaannya disengaja: penyisipan itu milik alur PLA/DLA, yang memang sedang menerbitkan
// dokumen untuk reasuransi yang belum terdaftar. Petugas yang menekan Simpan di layar master
// sedang mengubah baris yang dilihatnya — bila baris itu sudah tidak ada, yang benar adalah
// mengatakannya, bukan diam-diam membuat baris baru.
func (l *Service) Save(
	ctx context.Context,
	portalAlias string,
	key masterreas.Key,
	input masterreas.Input,
	by Actor,
	logger *slog.Logger,
) error {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	clean := key.Clean()
	if clean.IsEmpty() {
		return masterreas.ErrNotFound
	}

	value := input.Clean()
	if err := value.Check(); err != nil {
		return err
	}

	if err := repo.Update(ctx, clean, value.Email); err != nil {
		return err
	}

	noteSaved(logger, portalAlias, clean, by)
	return nil
}

// noteSaved mencatat perubahan surel.
//
// Alamat surelnya TIDAK ikut dicatat (`D-69`): yang direkam adalah peristiwanya beserta
// baris mana yang disentuh, bukan isinya.
func noteSaved(
	logger *slog.Logger,
	portalAlias string,
	key masterreas.Key,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info("surel member reas diubah",
		slog.String("portal", portalAlias),
		slog.String("kode_reas", key.ReinsurerID),
		slog.String("nama_reas", key.ReinsurerName),
		slog.String("tipe", key.Type),
		slog.String("oleh", by.Login))
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
