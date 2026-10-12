// Package usecase mengorkestrasi modul View History Claim.
//
// Dua operasi, dan keduanya menempuh gerbang proteksi data lebih dulu:
//
//	Open    membuka layar — memakai satu jatah pencarian, lalu menyerahkan daftar tipe
//	Search  menjalankan pencarian — memeriksa gerbang tanpa memakai jatah, lalu mencatat
//
// Pembagian itu bukan pilihan rancangan melainkan bentuk sistem lama: gerbangnya berjalan
// pada langkah yang prakondisinya `TempSearch.SearchType==""`, yaitu hanya sebelum tipe
// pencarian dipilih. Lihat riwayatklaim.Usage.ConsumesQuota.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/riwayatklaim"
)

// Service melayani modul View History Claim.
type Service struct {
	repoSelector       riwayatklaim.RepoSelector
	protectionSelector riwayatklaim.ProtectionRepoSelector
	clock              riwayatklaim.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector       riwayatklaim.RepoSelector
	ProtectionSelector riwayatklaim.ProtectionRepoSelector
	Clock              riwayatklaim.Clock
}

// NewService membentuk layanan modul View History Claim.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("riwayatklaim/usecase: RepoSelector wajib diisi")
	}
	if o.ProtectionSelector == nil {
		return nil, errors.New("riwayatklaim/usecase: ProtectionSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("riwayatklaim/usecase: Clock wajib diisi")
	}
	return &Service{
		repoSelector:       o.RepoSelector,
		protectionSelector: o.ProtectionSelector,
		clock:              o.Clock,
	}, nil
}

// Opened adalah keadaan layar tepat setelah dibuka.
type Opened struct {
	// Access adalah sisa jatah pencarian setelah pembukaan ini.
	Access riwayatklaim.Access

	// SearchTypes adalah isi dropdown "Tipe Pencarian", termasuk yang belum tersedia.
	SearchTypes []riwayatklaim.SearchType
}

// Open menjalankan gerbang proteksi data dan menyerahkan isi dropdown.
//
// # Kenapa membuka layar memakai jatah
//
// Karena begitulah sistem lama berperilaku, dan Work Owner memutuskan 2026-09-20 gerbang
// ini dibangun penuh. Langkah "Cek Log search" pada activity lama berjalan dengan
// prakondisi `TempSearch.SearchType==""` — yaitu saat layar baru dibuka dan tipe
// pencarian belum dipilih — lalu mengurangi kolom LOGSEARCH sebanyak satu.
//
// Akibatnya nyata dan perlu diketahui penguji: membuka layar lima kali menghabiskan lima
// jatah, tanpa satu pun pencarian dijalankan. Layar karena itu wajib memanggil operasi
// ini SEKALI per kunjungan, bukan pada setiap penggambaran ulang.
func (s *Service) Open(
	ctx context.Context,
	portalAlias string,
	caller riwayatklaim.Caller,
) (Opened, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Opened{}, riwayatklaim.ErrCallerUnknown
	}

	protectionRepo, err := s.protectionSelector(portalAlias)
	if err != nil {
		return Opened{}, err
	}

	protection, registered, err := protectionRepo.Find(ctx, clean.Login, riwayatklaim.ModuleKey)
	if err != nil {
		return Opened{}, fmt.Errorf("membaca proteksi data: %w", err)
	}

	used, err := protectionRepo.CountUsage(ctx, clean.Login, riwayatklaim.ModuleKey)
	if err != nil {
		return Opened{}, fmt.Errorf("menghitung pemakaian jatah: %w", err)
	}

	access, err := riwayatklaim.Grant(protection, registered, used)
	if err != nil {
		return Opened{}, err
	}

	// Pemakaian dicatat SETELAH gerbang meluluskan, bukan sebelum: mencatat lebih dulu
	// berarti percobaan yang ditolak pun ikut mengurangi jatah.
	record := riwayatklaim.Usage{
		Login:         clean.Login,
		Module:        riwayatklaim.ModuleKey,
		ConsumesQuota: true,
		At:            s.clock.Now().UTC(),
	}
	if err := protectionRepo.RecordUsage(ctx, record); err != nil {
		return Opened{}, fmt.Errorf("mencatat pemakaian jatah: %w", err)
	}

	return Opened{Access: access, SearchTypes: riwayatklaim.SearchTypes()}, nil
}

// Found adalah hasil satu pencarian.
type Found struct {
	// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
	Page riwayatklaim.Page

	// Criteria adalah kriteria yang BENAR-BENAR dipakai setelah divalidasi.
	Criteria riwayatklaim.Criteria

	// Access adalah sisa jatah pencarian — tidak berkurang oleh pencarian ini.
	Access riwayatklaim.Access
}

// Search menjalankan pencarian riwayat klaim.
//
// Gerbang diperiksa ulang di sini meski jatahnya tidak dipakai. Tanpa itu, pencarian
// dapat dijalankan langsung ke endpoint-nya tanpa pernah melewati pembukaan layar — dan
// gerbang yang hanya dijaga antarmuka bukan gerbang (`D-59`: yang menjadi kendali adalah
// pemeriksaan di server pada setiap endpoint).
func (s *Service) Search(
	ctx context.Context,
	portalAlias string,
	caller riwayatklaim.Caller,
	input riwayatklaim.CriteriaInput,
	page riwayatklaim.Pagination,
) (Found, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Found{}, riwayatklaim.ErrCallerUnknown
	}

	// Kriteria divalidasi LEBIH DULU, sebelum gerbang disentuh: isian yang cacat bukan
	// pemakaian jatah, dan menolaknya setelah gerbang berarti pengguna yang salah ketik
	// ikut terhitung memakai layar.
	criteria, err := riwayatklaim.NewCriteria(input)
	if err != nil {
		return Found{}, err
	}

	protectionRepo, err := s.protectionSelector(portalAlias)
	if err != nil {
		return Found{}, err
	}

	protection, registered, err := protectionRepo.Find(ctx, clean.Login, riwayatklaim.ModuleKey)
	if err != nil {
		return Found{}, fmt.Errorf("membaca proteksi data: %w", err)
	}

	used, err := protectionRepo.CountUsage(ctx, clean.Login, riwayatklaim.ModuleKey)
	if err != nil {
		return Found{}, fmt.Errorf("menghitung pemakaian jatah: %w", err)
	}

	access, err := riwayatklaim.Check(protection, registered, used)
	if err != nil {
		return Found{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Found{}, err
	}

	result, err := repo.Search(ctx, criteria, page.Normalize())
	if err != nil {
		return Found{}, fmt.Errorf("mencari riwayat klaim: %w", err)
	}

	// Pencarian dicatat meski tidak mengurangi jatah. Kegagalan mencatat TIDAK
	// membatalkan hasil yang sudah didapat — tetapi juga tidak ditelan diam-diam: ia
	// dikembalikan sebagai galat supaya lapisan transport mencatatnya di log.
	record := riwayatklaim.Usage{
		Login:          clean.Login,
		Module:         riwayatklaim.ModuleKey,
		SearchTypeCode: criteria.Type.Code,
		SearchValue:    searchValueOf(criteria),
		ConsumesQuota:  false,
		At:             s.clock.Now().UTC(),
	}
	if err := protectionRepo.RecordUsage(ctx, record); err != nil {
		return Found{}, fmt.Errorf("mencatat pencarian: %w", err)
	}

	return Found{Page: result, Criteria: criteria, Access: access}, nil
}

// searchValueOf menyusun nilai yang dicatat di jejak audit.
//
// Yang dicatat adalah nilai yang BENAR-BENAR dipakai kueri, bukan isian yang tampak di
// layar. Keduanya berbeda pada tipe Tanggal Lahir — lihat riwayatklaim.Criteria.QueryValue
// — dan mencatat isian yang tampak akan membuat jejaknya menyatakan pencarian yang tidak
// pernah terjadi.
func searchValueOf(criteria riwayatklaim.Criteria) string {
	text, date, isDate := criteria.QueryValue()
	if !isDate {
		return text
	}
	if date == nil {
		return ""
	}
	return date.Format("2006-01-02")
}
