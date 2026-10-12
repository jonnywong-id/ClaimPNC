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
	"strings"

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
//
// Nama bisnis DILENGKAPI di sini, bukan di penyimpanan. Yang tersimpan bersama pemetaan
// hanyalah ID — dokumen JSON-nya memang tidak punya tempat untuk nama — sehingga nama harus
// dicari ke master setiap kali dibaca.
//
// Itu juga yang membuat nama selalu mutakhir: bisnis yang berganti nama di master langsung
// terbaca benar di sini, tanpa ada yang perlu memperbarui baris pemetaannya.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftarobjekdokumen.DocumentObject, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	row, err := repo.Get(ctx, id)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	s.fillBusinessNames(ctx, portalAlias, row.Businesses)
	return row, nil
}

// fillBusinessNames melengkapi nama pada setiap pemetaan, di tempat.
//
// Pemetaan yang ID-nya TIDAK ketemu di master dibiarkan bernama kosong — bukan dibuang.
// Membuangnya berarti pengguna membuka form, tidak melihat barisnya, lalu menyimpan — dan
// pemetaan yang sebenarnya ada ikut terhapus tanpa ia pernah tahu.
//
// Kegagalan membaca master juga tidak menggagalkan apa pun: yang hilang hanya namanya.
func (s *Service) fillBusinessNames(ctx context.Context, portalAlias string, list []daftarobjekdokumen.Business) {
	if len(list) == 0 {
		return
	}

	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return
	}
	master, err := repo.List(ctx)
	if err != nil {
		return
	}

	byID := make(map[string]string, len(master))
	for _, b := range master {
		byID[strings.ToUpper(strings.TrimSpace(b.ID))] = b.Name
	}

	for i := range list {
		if name, found := byID[strings.ToUpper(strings.TrimSpace(list[i].ID))]; found {
			list[i].Name = name
		}
	}
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

	data, err := s.toSaveData(ctx, portalAlias, clean)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	saved, err := repo.Insert(ctx, data)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	s.fillBusinessNames(ctx, portalAlias, saved.Businesses)
	return saved, nil
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

	data, err := s.toSaveData(ctx, portalAlias, clean)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	saved, err := repo.Update(ctx, id, data)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	s.fillBusinessNames(ctx, portalAlias, saved.Businesses)
	return saved, nil
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

// toSaveData menyelesaikan nama bisnis menjadi ID, lalu menyusun isi yang benar-benar
// disimpan.
func (s *Service) toSaveData(
	ctx context.Context,
	portalAlias string,
	clean daftarobjekdokumen.Input,
) (daftarobjekdokumen.SaveData, error) {
	businesses, err := s.resolveBusinesses(ctx, portalAlias, clean.BusinessNames)
	if err != nil {
		return daftarobjekdokumen.SaveData{}, err
	}
	return daftarobjekdokumen.SaveData{
		Description: clean.Description,
		Businesses:  businesses,
	}, nil
}

// resolveBusinesses mengubah nama bisnis menjadi ID.
//
// # Nama yang tidak dikenali master DITOLAK
//
// Bukan dibuang diam-diam, dan bukan disimpan apa adanya. Keduanya tidak mungkin: yang
// tersimpan di dokumen JSON hanyalah ID, sehingga nama tanpa ID tidak punya tempat.
//
// Membuangnya diam-diam adalah perilaku terburuk dari ketiganya — pengguna menekan Simpan,
// melihat "berhasil", lalu menemukan barisnya hilang saat form dibuka lagi. Galat validasi
// yang menyebut nama mana yang tidak dikenali dapat langsung ditindaklanjuti.
//
// Ini BERBEDA dari modul Master COL Simas Online, yang menerima nama bebas. Perbedaannya
// dipaksa penyimpanan, bukan dipilih: tabel pemetaan di sana punya kolom NOTE.
//
// Master dibaca SEKALI lalu dicocokkan di memori, bukan satu kueri per baris: satu kueri
// per baris grid adalah N+1 yang dilarang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 butir 4.
//
// # Kegagalan membaca master menggagalkan penyimpanan
//
// Juga berbeda dari modul itu, dan dengan alasan yang sama. Di sana master hanya MELENGKAPI
// ID, sehingga gagal membacanya cukup kehilangan ID. Di sini master MENENTUKAN apa yang
// disimpan: tanpa dapat membacanya, satu-satunya pilihan adalah menyimpan pemetaan kosong —
// yang berarti menghapus seluruh pemetaan yang sudah ada tanpa ada yang memintanya.
func (s *Service) resolveBusinesses(
	ctx context.Context,
	portalAlias string,
	names []string,
) ([]daftarobjekdokumen.Business, error) {
	result := make([]daftarobjekdokumen.Business, 0, len(names))
	if len(names) == 0 {
		return result, nil
	}

	master, err := s.businessMaster(ctx, portalAlias)
	if err != nil {
		return nil, err
	}

	var unknown []string
	for _, name := range names {
		matched, found := master[daftarobjekdokumen.NormalizeBusinessName(name)]
		if !found {
			unknown = append(unknown, name)
			continue
		}
		result = append(result, matched)
	}

	if len(unknown) > 0 {
		return nil, &daftarobjekdokumen.ValidationError{
			Violation: []daftarobjekdokumen.Violation{{
				Field: daftarobjekdokumen.FieldBusiness,
				Message: "Bisnis berikut tidak ada di master dan tidak dapat disimpan: " +
					strings.Join(unknown, ", ") + ". Pilih dari daftar yang tersedia.",
			}},
		}
	}
	return result, nil
}

// businessMaster membaca master bisnis menjadi peta bernama.
//
// Galatnya diteruskan, tidak ditelan — lihat alasannya pada resolveBusinesses.
func (s *Service) businessMaster(
	ctx context.Context,
	portalAlias string,
) (map[string]daftarobjekdokumen.Business, error) {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	list, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]daftarobjekdokumen.Business, len(list))
	for _, b := range list {
		result[daftarobjekdokumen.NormalizeBusinessName(b.Name)] = b
	}
	return result, nil
}
