// Package usecase merangkai seam modul Daftar Tipe Dokumen Bisnis menjadi satuan kerja
// yang dipanggil lapisan transport.
//
// Tugasnya tiga, dan hanya tiga:
//
//   - memilih Repo milik portal yang sedang aktif (`ADR-0030`)
//   - menyusun jejak simpan dari identitas pemanggil dan seam Clock (`F-5`)
//   - menegakkan satu-satunya aturan isian yang ditiru dari layar lama
//
// Lapisan ini tipis, dan itu disengaja. Ia tetap ada karena ketiga tugas di atas tidak
// pantas dimiliki handler HTTP — yang pertama akan membuat setiap handler mengulang
// pemilihan portal, dan yang kedua akan menaruh jam sistem di lapisan yang tidak dapat
// diuji secara deterministik.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/platform/clock"
)

// Service melayani seluruh operasi modul.
type Service struct {
	repoSelector          daftartipedokumenbisnis.RepoSelector
	businessSelector      daftartipedokumenbisnis.BusinessRepoSelector
	documentTypeSelector  daftartipedokumenbisnis.DocumentTypeRepoSelector
	detailTypeDocSelector daftartipedokumenbisnis.DetailTypeDocRepoSelector
	objectDocSelector     daftartipedokumenbisnis.ObjectDocRepoSelector
	clock                 clock.Clock

	// bulkExcluded adalah himpunan kode bisnis yang dikecualikan tombol "Pilih semua".
	//
	// Himpunan, bukan senarai: ia dibaca sekali per baris pada setiap pemuatan daftar
	// bisnis, dan penelusuran senarai akan menjadikannya perkalian dua daftar.
	bulkExcluded map[string]struct{}
}

// Options adalah ketergantungan Service.
type Options struct {
	RepoSelector          daftartipedokumenbisnis.RepoSelector
	BusinessSelector      daftartipedokumenbisnis.BusinessRepoSelector
	DocumentTypeSelector  daftartipedokumenbisnis.DocumentTypeRepoSelector
	DetailTypeDocSelector daftartipedokumenbisnis.DetailTypeDocRepoSelector
	ObjectDocSelector     daftartipedokumenbisnis.ObjectDocRepoSelector
	Clock                 clock.Clock

	// BulkSelectExcludedBusinesses boleh kosong: tanpa isi, "Pilih semua" memilih
	// seluruh bisnis. Ia TIDAK diwajibkan supaya modul tetap dapat dirakit dalam
	// pengujian tanpa menyeret konfigurasi.
	BulkSelectExcludedBusinesses []string
}

// NewService membentuk Service dan menolak ketergantungan yang belum diisi.
//
// Menolak di sini, bukan membiarkannya nil, supaya kesalahan perakitan terlihat saat
// aplikasi start — bukan sebagai panic pada permintaan pertama pengguna.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: RepoSelector wajib diisi")
	}
	if o.BusinessSelector == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: BusinessSelector wajib diisi")
	}
	if o.DocumentTypeSelector == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: DocumentTypeSelector wajib diisi")
	}
	if o.DetailTypeDocSelector == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: DetailTypeDocSelector wajib diisi")
	}
	if o.ObjectDocSelector == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: ObjectDocSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("daftartipedokumenbisnis/usecase: Clock wajib diisi")
	}
	excluded := make(map[string]struct{}, len(o.BulkSelectExcludedBusinesses))
	for _, code := range o.BulkSelectExcludedBusinesses {
		if clean := strings.TrimSpace(code); clean != "" {
			excluded[clean] = struct{}{}
		}
	}

	return &Service{
		repoSelector:          o.RepoSelector,
		businessSelector:      o.BusinessSelector,
		documentTypeSelector:  o.DocumentTypeSelector,
		detailTypeDocSelector: o.DetailTypeDocSelector,
		objectDocSelector:     o.ObjectDocSelector,
		clock:                 o.Clock,
		bulkExcluded:          excluded,
	}, nil
}

// ListBusinesses mengembalikan lini bisnis yang sudah punya aturan dokumen.
func (s *Service) ListBusinesses(ctx context.Context, portalAlias string) ([]daftartipedokumenbisnis.Business, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListBusinesses(ctx)
}

// ListByBusiness mengembalikan aturan dokumen milik satu lini bisnis.
func (s *Service) ListByBusiness(ctx context.Context, portalAlias, businessID string) ([]daftartipedokumenbisnis.DocumentRule, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListByBusiness(ctx, strings.TrimSpace(businessID))
}

// Get mengembalikan satu aturan lengkap dengan jaminannya.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (daftartipedokumenbisnis.DocumentRule, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}
	return repo.Get(ctx, strings.TrimSpace(id))
}

// Create menyisipkan perkalian bisnis kali baris aturan.
//
// Membersihkan lebih dulu, lalu memvalidasi — urutannya mengikat. Membalikkannya akan
// meloloskan pilihan bisnis yang isinya hanya spasi, dan menyimpan baris ber-BUSINESSID
// kosong yang tidak muncul di grid mana pun.
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	input daftartipedokumenbisnis.BatchInput,
	by string,
) ([]daftartipedokumenbisnis.DocumentRule, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	clean := input.Clean()
	if err := clean.Validate(); err != nil {
		return nil, err
	}
	return repo.InsertBatch(ctx, clean, s.editor(by))
}

// Update mengubah satu baris aturan.
//
// Membaca barisnya lebih dulu supaya "barisnya tidak ada" dapat dibedakan dari "barisnya
// ada tetapi tidak ada yang berubah". Tanpa itu keduanya menghasilkan nol baris tersentuh,
// dan layar akan melaporkan baris hilang kepada petugas yang sekadar menekan Simpan dua
// kali.
func (s *Service) Update(
	ctx context.Context,
	portalAlias, id string,
	input daftartipedokumenbisnis.Input,
	by string,
) (daftartipedokumenbisnis.DocumentRule, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}

	key := strings.TrimSpace(id)
	if _, err := repo.Get(ctx, key); err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}
	return repo.Update(ctx, key, input.Clean(), s.editor(by))
}

// AddCoverage menambahkan satu jaminan pada sebuah aturan.
//
// # Jaminan kosong DILEWATI, bukan ditolak
//
// Meniru `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:3547`, yang precondition
// langkahnya berbunyi `.ContentNote==""` lalu KELUAR dari iterasi — baris jaminan kosong
// sekadar tidak disimpan, tanpa pesan galat apa pun.
//
// Versi pertama modul ini menolaknya sebagai galat. Itu penyimpangan yang saya buat
// sendiri, dan dicabut atas keputusan Work Owner 2026-09-23. Barisnya tetap tidak tersimpan
// — yang berubah hanya bahwa layar tidak lagi menampilkan galat untuk hal yang di Pega
// berlalu diam-diam.
func (s *Service) AddCoverage(
	ctx context.Context,
	portalAlias, id, coverageID string,
) (daftartipedokumenbisnis.DocumentRule, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}

	key := strings.TrimSpace(id)
	if strings.TrimSpace(coverageID) == "" {
		// Barisnya tetap dibaca kembali supaya jawabannya berbentuk sama dengan
		// penambahan yang berhasil — layar tidak perlu membedakan keduanya, persis
		// seperti Pega yang juga tidak membedakannya.
		return repo.Get(ctx, key)
	}
	return repo.AddCoverage(ctx, key, strings.TrimSpace(coverageID))
}

// ListBusinessChoices mengembalikan SELURUH lini bisnis sebagai daftar pilihan, beserta
// penanda mana yang dikecualikan dari pemilihan massal.
//
// Penandanya dipasang DI SINI, bukan di repo, karena ia bukan isi tabel: daftarnya datang
// dari konfigurasi. Repo tetap bersih membaca POOLDATA.BUSINESS apa adanya.
func (s *Service) ListBusinessChoices(ctx context.Context, portalAlias string) ([]daftartipedokumenbisnis.Business, error) {
	repo, err := s.businessSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range list {
		if _, excluded := s.bulkExcluded[list[i].ID]; excluded {
			list[i].ExcludedFromBulkSelect = true
		}
	}
	return list, nil
}

// MayBulkSelect menjawab apakah pemanggil boleh memakai tombol "Pilih semua".
//
// Di Pega tombolnya hanya TERLIHAT oleh operator ber-`pyPosition='NONMBU'`
// (`Section/InputListDetailTypeDocumentBusiness_sect-Section.xml:3060`, `pyVisible=OTHER`).
// Ia penyembunyian tampilan, bukan penegakan kewenangan — dan ditiru sebagai penyembunyian
// pula, bukan dinaikkan menjadi pemeriksaan izin.
//
// Alasannya bukan kemalasan: menjadikannya kewenangan berarti memperkenalkan model izin
// berbasis properti operator, yang justru `D-59` gantikan dengan izin per menu. Tombol ini
// tidak membuka satu pun kemampuan baru — bisnis yang sama tetap dapat dipilih satu per
// satu oleh siapa pun yang membuka layarnya.
func (s *Service) MayBulkSelect(position string) bool {
	return strings.EqualFold(strings.TrimSpace(position), bulkSelectPosition)
}

// bulkSelectPosition adalah nilai pyPosition yang melihat tombol "Pilih semua".
//
// Dibaca dari `Section/InputListDetailTypeDocumentBusiness_sect-Section.xml:3060`:
// `OperatorID.pyPosition='NONMBU'`. Ia penanda ORGANISASI, bukan nilai bisnis seperti
// ambang atau kode lini — sehingga `D-15` tidak menuntutnya pindah ke master data.
const bulkSelectPosition = "NONMBU"

// ListDocumentTypes mengembalikan tahap dokumen sebagai daftar pilihan.
func (s *Service) ListDocumentTypes(ctx context.Context, portalAlias string) ([]daftartipedokumenbisnis.Reference, error) {
	repo, err := s.documentTypeSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// ListDetailTypeDocs mengembalikan rincian dokumen sebagai daftar pilihan.
func (s *Service) ListDetailTypeDocs(ctx context.Context, portalAlias string) ([]daftartipedokumenbisnis.Reference, error) {
	repo, err := s.detailTypeDocSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// ListObjectDocs mengembalikan objek dokumen sebagai daftar pilihan.
func (s *Service) ListObjectDocs(ctx context.Context, portalAlias string) ([]daftartipedokumenbisnis.Reference, error) {
	repo, err := s.objectDocSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// EnsurePortalReady memastikan portal dapat dilayani, dipakai saat perakitan dan mode
// periksa.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("daftartipedokumenbisnis/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// editor menyusun jejak simpan.
//
// Identitas kosong DITERIMA. Ia hanya mengisi kolom jejak, bukan menentukan apa yang
// tersimpan — menolak penyimpanan karenanya akan membuat petugas kehilangan isian yang
// sudah diketik demi kolom yang tidak dilihat siapa pun di layar.
func (s *Service) editor(by string) daftartipedokumenbisnis.Editor {
	return daftartipedokumenbisnis.Editor{
		Identity: strings.TrimSpace(by),
		At:       s.clock.Now(),
	}
}
