// Package usecase mengorkestrasi modul Master COL Simas Online.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, memeriksa hal-hal yang menuntut penyimpanan, lalu
// memanggil Repo. Ia tidak tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/mastercolsimasonline"
)

// Service adalah pintu masuk seluruh perkara master COL Simas Online.
type Service struct {
	repoSelector     mastercolsimasonline.RepoSelector
	businessSelector mastercolsimasonline.BusinessRepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastercolsimasonline.RepoSelector

	// BusinessSelector memilih master bisnis milik satu portal entitas. Wajib.
	//
	// Tanpa ini, keberadaan bisnis yang dipilih pengguna tidak dapat diperiksa — dan
	// pemetaan ke bisnis yang tidak ada akan tersimpan tanpa satu pun tanda.
	BusinessSelector mastercolsimasonline.BusinessRepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastercolsimasonline/usecase: RepoSelector wajib diisi")
	}
	if o.BusinessSelector == nil {
		return nil, errors.New("mastercolsimasonline/usecase: BusinessSelector wajib diisi")
	}
	return &Service{
		repoSelector:     o.RepoSelector,
		businessSelector: o.BusinessSelector,
	}, nil
}

// List mengembalikan seluruh penyebab kerugian milik satu portal.
func (s *Service) List(ctx context.Context, portalAlias string) ([]mastercolsimasonline.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu penyebab kerugian LENGKAP dengan pemetaan bisnisnya.
func (s *Service) Get(ctx context.Context, portalAlias, code string) (mastercolsimasonline.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	return repo.Get(ctx, code)
}

// ListBusiness mengembalikan seluruh bisnis untuk pilihan di layar.
func (s *Service) ListBusiness(ctx context.Context, portalAlias string) ([]mastercolsimasonline.Business, error) {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Create menyimpan satu penyebab kerugian baru dan mengembalikan baris tersimpannya.
//
// Kode tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang
// bersangkutan. Deretnya karena itu berdiri sendiri per entitas — dua portal dapat
// memiliki kode yang sama untuk penyebab kerugian yang berbeda, persis seperti sistem
// lama, karena setiap entitas punya basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(ctx context.Context, portalAlias string, input mastercolsimasonline.Input) (mastercolsimasonline.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}

	clean, err := s.checkInput(ctx, portalAlias, input)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	return repo.Insert(ctx, s.toSaveData(ctx, portalAlias, clean))
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Kode tidak pernah ikut berubah. Di layar lama pun ia begitu: isian M_COL_ID ditandai
// `pyReadOnly=true`, dan `PEGA_M_CAUSE_OF_LOSS.prc:34` memakai M_COL_ID hanya sebagai
// penyaring `WHERE`, tidak pernah sebagai kolom yang di-`SET`. Membiarkannya berubah
// akan memutus setiap baris `D_CAUSE_OF_LOSS` yang sudah merujuk kode lamanya.
func (s *Service) Update(ctx context.Context, portalAlias, code string, input mastercolsimasonline.Input) (mastercolsimasonline.CauseOfLoss, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari "baris ada
	// tetapi nilainya sama persis". UPDATE yang mengenai nol baris tidak membedakan
	// keduanya, dan menjawab "tidak ditemukan" untuk penyimpanan yang sebenarnya berhasil
	// akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Get(ctx, code); err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}

	clean, err := s.checkInput(ctx, portalAlias, input)
	if err != nil {
		return mastercolsimasonline.CauseOfLoss{}, err
	}
	return repo.Update(ctx, code, s.toSaveData(ctx, portalAlias, clean))
}

// toSaveData menyelesaikan nama bisnis menjadi pasangan nama dan ID, lalu menyusun isi
// yang benar-benar disimpan.
func (s *Service) toSaveData(
	ctx context.Context,
	portalAlias string,
	clean mastercolsimasonline.Input,
) mastercolsimasonline.SaveData {
	return mastercolsimasonline.SaveData{
		Description: clean.Description,
		Businesses:  s.resolveBusinesses(ctx, portalAlias, clean.BusinessNames),
	}
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastercolsimasonline/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// checkInput merapikan isian lalu menjalankan seluruh aturan murni.
//
// SELURUH pelanggaran dikumpulkan sekaligus, bukan yang pertama saja, supaya pengguna
// yang salah pada dua hal melihat keduanya dalam satu kali simpan.
//
// # Apa yang TIDAK diperiksa di sini, dan kenapa
//
// Keberadaan bisnis. Isian Bisnis di Pega ber-`pyAllowFreeFormInput=true`, sehingga nama
// yang tidak ada di master memang boleh diketik dan disimpan (Work Owner 2026-09-21).
// Yang dikerjakan atas nama bisnis bukan penolakan melainkan **penyelesaian ke ID** —
// lihat resolveBusinesses.
//
// Induk juga tidak lagi diperiksa: isian "ID Master Kerugian" (`MST_COL_ID`) dicabut
// seluruhnya pada 2026-09-23 atas keputusan Work Owner, sehingga tidak ada rujukan yang
// perlu dibuktikan keberadaannya.
func (s *Service) checkInput(
	ctx context.Context,
	portalAlias string,
	input mastercolsimasonline.Input,
) (mastercolsimasonline.Input, error) {
	clean := input.Clean()

	var violation []mastercolsimasonline.Violation

	var validationError *mastercolsimasonline.ValidationError
	if err := clean.Check(); err != nil {
		if !errors.As(err, &validationError) {
			return mastercolsimasonline.Input{}, err
		}
		violation = append(violation, validationError.Violation...)
	}

	if len(violation) > 0 {
		return mastercolsimasonline.Input{}, &mastercolsimasonline.ValidationError{Violation: violation}
	}
	return clean, nil
}

// resolveBusinesses mengubah nama bisnis menjadi pasangan nama dan ID.
//
// Nama yang cocok dengan master mendapat ID-nya; yang tidak cocok TETAP DIKEMBALIKAN
// dengan ID kosong — itu perilaku Pega yang dipertahankan (`pyAllowFreeFormInput=true`).
//
// Nama yang dikembalikan adalah nama dari MASTER bila cocok, bukan yang diketik pengguna:
// dengan begitu "fire / property" yang diketik huruf kecil tersimpan dalam ejaan resmi
// masternya, dan daftar di layar tidak menampilkan satu bisnis dalam dua ejaan.
//
// Master dibaca SEKALI lalu dicocokkan di memori, bukan satu kueri per baris: satu kueri
// per baris grid adalah N+1 yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2
// butir 4.
//
// Kegagalan membaca master TIDAK menggagalkan penyimpanan. Karena nama bebas memang
// diterima, master hanya dipakai untuk MELENGKAPI ID — dan melengkapi yang gagal lebih
// baik daripada menolak penyimpanan yang sebenarnya sah. Yang hilang hanya ID-nya, dan
// itu keadaan yang memang sudah harus ditangani setiap pembaca.
func (s *Service) resolveBusinesses(
	ctx context.Context,
	portalAlias string,
	names []string,
) []mastercolsimasonline.Business {
	result := make([]mastercolsimasonline.Business, 0, len(names))
	if len(names) == 0 {
		return result
	}

	master := s.businessMaster(ctx, portalAlias)
	for _, name := range names {
		if matched, found := master[mastercolsimasonline.NormalizeBusinessName(name)]; found {
			result = append(result, matched)
			continue
		}
		result = append(result, mastercolsimasonline.Business{Name: name})
	}
	return result
}

// businessMaster membaca master bisnis menjadi peta bernama, atau peta kosong bila tidak
// dapat dibaca.
func (s *Service) businessMaster(ctx context.Context, portalAlias string) map[string]mastercolsimasonline.Business {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return map[string]mastercolsimasonline.Business{}
	}
	list, err := repo.List(ctx)
	if err != nil {
		return map[string]mastercolsimasonline.Business{}
	}

	result := make(map[string]mastercolsimasonline.Business, len(list))
	for _, b := range list {
		result[mastercolsimasonline.NormalizeBusinessName(b.Name)] = b
	}
	return result
}
