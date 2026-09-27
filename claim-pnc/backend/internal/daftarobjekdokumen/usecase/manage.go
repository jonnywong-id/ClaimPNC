// Package usecase mengorkestrasi modul Daftar Objek Dokumen.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, menyelesaikan nama bisnis menjadi ID, lalu memanggil
// Repo. Ia tidak tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/daftarobjekdokumen"
)

// Service adalah pintu masuk seluruh perkara objek dokumen.
type Service struct {
	repoSelector     daftarobjekdokumen.RepoSelector
	businessSelector daftarobjekdokumen.BusinessRepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector daftarobjekdokumen.RepoSelector

	// BusinessSelector memilih master bisnis milik satu portal entitas. Wajib.
	//
	// Tanpa ini, nama bisnis yang dipilih pengguna tidak dapat diselesaikan menjadi ID —
	// dan seluruh pemetaan akan tersimpan tanpa rujukan ke master, tanpa satu pun tanda.
	BusinessSelector daftarobjekdokumen.BusinessRepoSelector
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("daftarobjekdokumen/usecase: RepoSelector wajib diisi")
	}
	if o.BusinessSelector == nil {
		return nil, errors.New("daftarobjekdokumen/usecase: BusinessSelector wajib diisi")
	}
	return &Service{
		repoSelector:     o.RepoSelector,
		businessSelector: o.BusinessSelector,
	}, nil
}

// List mengembalikan seluruh objek dokumen milik satu portal.
func (s *Service) List(ctx context.Context, portalAlias string) ([]daftarobjekdokumen.DocumentObject, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu objek dokumen LENGKAP dengan pemetaan bisnisnya.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftarobjekdokumen.DocumentObject, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return repo.Get(ctx, id)
}

// ListBusiness mengembalikan seluruh bisnis untuk saran isian di layar.
//
// Modul ini TIDAK mendaftarkan rutenya sendiri — `/master/bisnis` sudah dimiliki modul
// Master COL Simas Online, dan dua modul yang mendaftarkan jalur yang sama akan membuat
// chi panik saat start. Method ini tetap ada karena layanan ini yang tahu cara membaca
// master bisnis milik portal aktif, dan uji modul memakainya.
func (s *Service) ListBusiness(ctx context.Context, portalAlias string) ([]daftarobjekdokumen.Business, error) {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Create menyimpan satu objek dokumen baru dan mengembalikan baris tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diterbitkan penyimpanan portal yang bersangkutan.
// Deretnya karena itu berdiri sendiri per entitas — dua portal dapat memiliki ID yang sama
// untuk objek dokumen yang berbeda, persis seperti sistem lama, karena setiap entitas
// punya basis datanya sendiri (`ADR-0030`).
func (s *Service) Create(ctx context.Context, portalAlias string, input daftarobjekdokumen.Input) (daftarobjekdokumen.DocumentObject, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	clean, err := s.checkInput(input)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return repo.Insert(ctx, s.toSaveData(ctx, portalAlias, clean))
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// ID tidak pernah ikut berubah. Di layar lama pun isiannya `pyReadOnly=true`, dan
// membiarkannya berubah akan memutus setiap baris LST_TYPE_DOC_BUSINESS yang sudah merujuk
// ID lamanya lewat kolom OBJECT_DOC_ID.
func (s *Service) Update(ctx context.Context, portalAlias, id string, input daftarobjekdokumen.Input) (daftarobjekdokumen.DocumentObject, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	// Barisnya dimuat lebih dulu supaya "baris tidak ada" dapat dibedakan dari "baris ada
	// tetapi nilainya sama persis". UPDATE yang mengenai nol baris tidak membedakan
	// keduanya, dan menjawab "tidak ditemukan" untuk penyimpanan yang sebenarnya berhasil
	// akan membuat pengguna menyimpan berulang kali.
	if _, err := repo.Get(ctx, id); err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	clean, err := s.checkInput(input)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return repo.Update(ctx, id, s.toSaveData(ctx, portalAlias, clean))
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("daftarobjekdokumen/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// checkInput merapikan isian lalu menjalankan seluruh aturan murni.
//
// SELURUH pelanggaran dikumpulkan sekaligus, bukan yang pertama saja, supaya pengguna yang
// salah pada dua hal melihat keduanya dalam satu kali simpan.
//
// Keberadaan bisnis TIDAK diperiksa di sini, dan itu disengaja: sel Bisnis di Pega memakai
// kontrol autocomplete yang menerima ketikan bebas, sehingga nama di luar master memang
// boleh disimpan. Yang dikerjakan atas nama bisnis bukan penolakan melainkan
// **penyelesaian ke ID** — lihat resolveBusinesses.
func (s *Service) checkInput(input daftarobjekdokumen.Input) (daftarobjekdokumen.Input, error) {
	clean := input.Clean()

	var validationError *daftarobjekdokumen.ValidationError
	if err := clean.Check(); err != nil {
		if !errors.As(err, &validationError) {
			return daftarobjekdokumen.Input{}, err
		}
		return daftarobjekdokumen.Input{}, validationError
	}
	return clean, nil
}

// toSaveData menyelesaikan nama bisnis menjadi pasangan nama dan ID, lalu menyusun isi yang
// benar-benar disimpan.
func (s *Service) toSaveData(
	ctx context.Context,
	portalAlias string,
	clean daftarobjekdokumen.Input,
) daftarobjekdokumen.SaveData {
	return daftarobjekdokumen.SaveData{
		Description: clean.Description,
		Businesses:  s.resolveBusinesses(ctx, portalAlias, clean.BusinessNames),
	}
}

// resolveBusinesses mengubah nama bisnis menjadi pasangan nama dan ID.
//
// Nama yang cocok dengan master mendapat ID-nya; yang tidak cocok TETAP DIKEMBALIKAN dengan
// ID kosong — itu perilaku layar lama yang dipertahankan.
//
// Nama yang dikembalikan adalah nama dari MASTER bila cocok, bukan yang diketik pengguna:
// dengan begitu "fire / property" yang diketik huruf kecil tersimpan dalam ejaan resmi
// masternya, dan daftar di layar tidak menampilkan satu bisnis dalam dua ejaan.
//
// Master dibaca SEKALI lalu dicocokkan di memori, bukan satu kueri per baris: satu kueri
// per baris grid adalah N+1 yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 butir 4.
//
// Kegagalan membaca master TIDAK menggagalkan penyimpanan. Karena nama bebas memang
// diterima, master hanya dipakai untuk MELENGKAPI ID — dan melengkapi yang gagal lebih baik
// daripada menolak penyimpanan yang sebenarnya sah. Yang hilang hanya ID-nya, dan itu
// keadaan yang memang sudah harus ditangani setiap pembaca.
func (s *Service) resolveBusinesses(
	ctx context.Context,
	portalAlias string,
	names []string,
) []daftarobjekdokumen.Business {
	result := make([]daftarobjekdokumen.Business, 0, len(names))
	if len(names) == 0 {
		return result
	}

	master := s.businessMaster(ctx, portalAlias)
	for _, name := range names {
		if matched, found := master[daftarobjekdokumen.NormalizeBusinessName(name)]; found {
			result = append(result, matched)
			continue
		}
		result = append(result, daftarobjekdokumen.Business{Name: name})
	}
	return result
}

// businessMaster membaca master bisnis menjadi peta bernama, atau peta kosong bila tidak
// dapat dibaca.
func (s *Service) businessMaster(ctx context.Context, portalAlias string) map[string]daftarobjekdokumen.Business {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return map[string]daftarobjekdokumen.Business{}
	}
	list, err := repo.List(ctx)
	if err != nil {
		return map[string]daftarobjekdokumen.Business{}
	}

	result := make(map[string]daftarobjekdokumen.Business, len(list))
	for _, b := range list {
		result[daftarobjekdokumen.NormalizeBusinessName(b.Name)] = b
	}
	return result
}
