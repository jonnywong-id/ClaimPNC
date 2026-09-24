// Package usecase mengorkestrasi perkara Detail Penyebab Kerugian.
//
// Ia yang mengetahui urutan langkah; aturan isian ada di paket domain, dan cara membacanya
// dari basis data ada di repo. Lapisan ini tidak tahu apa pun tentang HTTP.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/detailpenyebab"
)

// Service adalah pintu masuk seluruh perkara Detail Penyebab Kerugian.
type Service struct {
	repoSelector detailpenyebab.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector detailpenyebab.RepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
//
// TANPA Clock. `POOLDATA.D_CAUSE_OF_LOSS` tidak punya satu pun kolom waktu — ia hanya
// punya D_COL_ID dan JSONDATA — sehingga tidak ada yang perlu distempel. Menerimanya
// "untuk jaga-jaga" berarti menerima bahan yang tidak pernah dipakai, dan itu menyesatkan
// pembaca berikutnya.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("detailpenyebab/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// Actor adalah pengguna yang sedang melakukan sesuatu.
//
// Ia dipakai HANYA untuk mengisi log. Tabelnya tidak punya kolom pencatat pelaku sama
// sekali, sehingga log adalah satu-satunya tempat "siapa yang mengubah baris ini" terekam.
//
// Itu BUKAN pengganti jejak audit `S-5`, dan tidak diklaim demikian. `D-59` menjadikan
// jejak audit satu-satunya kontrol pengimbang karena tidak ada pemisahan tugas — dan modul
// ini belum dapat menyandarkan apa pun padanya sampai kolomnya ada (`D-63`).
type Actor struct {
	Login string
}

// List mengembalikan baris satu portal yang cocok dengan penyaring.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	filter detailpenyebab.Filter,
) ([]detailpenyebab.CauseOfLossDetail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	return repo.List(ctx, detailpenyebab.Filter{
		Keyword:    strings.TrimSpace(filter.Keyword),
		MasterID:   strings.TrimSpace(filter.MasterID),
		BusinessID: strings.TrimSpace(filter.BusinessID),
	})
}

// Get mengembalikan satu baris untuk dimuat ke form penyuntingan.
//
// Padanan `Activity/CNMSetDetailCauseOfLoss_act`, yang dijalankan tombol Ubah pada setiap
// baris grid.
func (s *Service) Get(
	ctx context.Context,
	portalAlias, id string,
) (detailpenyebab.CauseOfLossDetail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}
	return repo.Get(ctx, key)
}

// Create menyisipkan satu Detail Penyebab Kerugian baru.
//
// # Urutannya mengikuti sistem lama
//
//	CNMInsertDetailCauseOfLoss_act langkah 1  D_COL_ID kosong -> "UnknownID"
//	                               langkah 2  seluruh page diserialkan menjadi JSON
//	                               langkah 3  PEGA_D_CAUSE_OF_LOSS menerbitkan ID lalu menyisipkan
//
// Satu perbedaan urutan yang disengaja: penanda `"UnknownID"` **tidak dibawa**. Di sistem
// lama ia penanda "belum punya ID" yang diselipkan ke dalam dokumen lalu ditimpa procedure
// dengan `replace()`; di sini ID diterbitkan lebih dulu, sehingga penanda itu tidak
// diperlukan. Alasan lengkapnya — termasuk cacat yang ikut tertutup — ada pada komentar
// kueri `detail_insert`.
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	input detailpenyebab.Input,
	by Actor,
	logger *slog.Logger,
) (detailpenyebab.CauseOfLossDetail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	saved, err := repo.Insert(ctx, clean)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	noteSaved(logger, "detail penyebab kerugian ditambahkan", portalAlias, saved, by)
	return saved, nil
}

// Save menyimpan perubahan atas baris yang sudah ada.
//
// # Baris dibaca lebih dulu, dan itu bukan formalitas
//
// Dua hal bergantung padanya:
//
//  1. **ErrNotFound yang benar.** Baris yang sudah hilang harus ditolak sebagai tidak
//     ditemukan, bukan dilaporkan berhasil karena `UPDATE` menyentuh nol baris.
//  2. **LegacyID yang tidak hilang.** Lihat di bawah.
//
// # LegacyID diambil dari baris TERSIMPAN bila layar tidak mengirimnya
//
// `OLD_D_COL_ID` tidak digambar di form mana pun, tetapi DIMUAT dan DISIMPAN ulang oleh
// sistem lama (`Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1297-1299`). Klien yang tidak
// mengirimnya karena itu tidak boleh menghapusnya — tautan ke sistem sebelum Pega akan
// lenyap pada setiap penyuntingan, tanpa satu pun tanda.
//
// Klien yang MENGIRIMNYA tetap dihormati: nilainya dipakai apa adanya. Dengan begitu
// koreksi atas ID warisan tetap mungkin, dan yang dicegah hanyalah penghapusan yang tidak
// disengaja.
func (s *Service) Save(
	ctx context.Context,
	portalAlias, id string,
	input detailpenyebab.Input,
	by Actor,
	logger *slog.Logger,
) (detailpenyebab.CauseOfLossDetail, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	key := strings.TrimSpace(id)
	if key == "" {
		return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	stored, err := repo.Get(ctx, key)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}
	if clean.LegacyID == "" {
		clean.LegacyID = stored.LegacyID
	}

	saved, err := repo.Update(ctx, key, clean)
	if err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	noteSaved(logger, "detail penyebab kerugian diubah", portalAlias, saved, by)
	return saved, nil
}

// SearchMaster mencari Master Penyebab Kerugian untuk isian "ID Master Kerugian".
//
// Kata kunci yang terlalu pendek menghasilkan daftar KOSONG, bukan galat: isian
// autocomplete memanggilnya pada setiap ketukan, dan menjawab galat pada huruf pertama
// akan menampilkan pesan merah di bawah isian yang sedang diketik.
func (s *Service) SearchMaster(
	ctx context.Context,
	portalAlias, keyword string,
) ([]detailpenyebab.MasterOption, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := strings.TrimSpace(keyword)
	if len([]rune(clean)) < detailpenyebab.MinLookupKeyword {
		return nil, nil
	}
	return repo.SearchMaster(ctx, clean)
}

// SearchBusiness mencari lini bisnis untuk grid "Bisnis".
//
// Perlakuan kata kunci pendeknya sama dengan SearchMaster, dan alasannya sama.
func (s *Service) SearchBusiness(
	ctx context.Context,
	portalAlias, keyword string,
) ([]detailpenyebab.Business, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := strings.TrimSpace(keyword)
	if len([]rune(clean)) < detailpenyebab.MinLookupKeyword {
		return nil, nil
	}
	return repo.SearchBusiness(ctx, clean)
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("detailpenyebab/usecase: portal %q tidak dapat dilayani: %w",
			portalAlias, err)
	}
	return nil
}

// noteSaved mencatat penyimpanan.
//
// Isi baris TIDAK ikut dicatat selain ID dan deskripsinya. Deskripsi Kerugian adalah teks
// acuan, bukan data nasabah, sehingga ia aman dicatat — sedangkan mencatat seluruh baris
// akan menaruh isi master ke dalam log yang retensinya lebih longgar daripada basis data.
func noteSaved(
	logger *slog.Logger,
	message, portalAlias string,
	one detailpenyebab.CauseOfLossDetail,
	by Actor,
) {
	if logger == nil {
		return
	}
	logger.Info(message,
		slog.String("portal", portalAlias),
		slog.String("id", one.ID),
		slog.String("id_master", one.MasterID),
		slog.String("deskripsi", one.Description),
		slog.String("status_aktif", one.Active),
		slog.Int("jumlah_bisnis", len(one.Business)),
		slog.String("oleh", by.Login))
}
