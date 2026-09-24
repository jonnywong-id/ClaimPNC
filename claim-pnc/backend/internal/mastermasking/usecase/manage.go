// Package usecase mengorkestrasi pengelolaan Master Masking: melihat daftar, membuka satu
// baris, menambah, mengubah, dan mengaktifkan atau menonaktifkannya.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, menegakkan aturan yang menuntut membaca penyimpanan, lalu
// memanggil Repo. Ia tidak tahu apa pun tentang HTTP maupun SQL.
//
// # Kenapa tidak ada Hapus
//
// Karena sistem lama pun tidak punya, meski tombolnya bernama DELETE.
// `RDB List/DeleteMstProteksi_SQL-SQL.xml` hanya menjalankan
// `UPDATE … SET STS_AKTF = …`, dan `Activity/DeleteMasking-Act.xml` mengisinya dengan
// `"TIDAK AKTIF"`. Tidak ada satu pun pernyataan DELETE terhadap tabel ini di seluruh
// export — sudah diperiksa. `D-66` menetapkan hal yang sama untuk sistem baru.
//
// Membuang barisnya pun akan merusak hal yang nyata: sepuluh dari 25 baris portal ASM
// berstatus tidak aktif, dan masing-masing adalah catatan bahwa seseorang PERNAH diberi
// kewenangan melihat data pribadi. Menghapusnya menghapus jejak itu.
//
// # Kepemilikan transaksi
//
// Procedure lama `POOLDATA.Update_Log_Proteksi` melakukan COMMIT sendiri di tiga cabangnya
// dan menaruh ROLLBACK setelahnya. Modul ini TIDAK memanggilnya sama sekali (`D-02`);
// pernyataan SQL-nya ditulis langsung dan transaksinya dimiliki Go (`D-68`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/mastermasking"
)

// DefaultBranchLimit membatasi jumlah pilihan cabang yang dikembalikan sekali minta.
//
// POOLDATA.BRANCH memuat 803 baris pada portal ASM — dibaca langsung pada 2026-09-20.
// Mengirim seluruhnya ke peramban setiap kali form dibuka membebani jaringan untuk daftar
// yang hampir seluruhnya tidak akan dilihat. Lima puluh cukup untuk mengisi satu layar
// saran, dan pengguna mempersempitnya dengan mengetik — persis cara kerja autocomplete di
// layar Pega.
const DefaultBranchLimit = 50

// Service mengelola master masking di atas seam penyimpanan per portal.
type Service struct {
	repoSelector mastermasking.RepoSelector
	now          func() time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector mastermasking.RepoSelector

	// Now memasok waktu penyimpanan. Boleh kosong; bawaannya jam sistem.
	//
	// Ia seam Clock (`F-5`), bukan kemudahan pengujian belaka: `docs/Steering/07` §3.2
	// melarang penambahan tujuh jam manual yang tersebar di sistem lama, dan satu-satunya
	// cara menegakkannya adalah membuat waktu datang dari satu tempat.
	Now func() time.Time
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("mastermasking/usecase: RepoSelector wajib diisi")
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Service{repoSelector: o.RepoSelector, now: now}, nil
}

// List mengembalikan baris yang cocok dengan penyaring, terbaru lebih dulu.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan angka: isinya 25 baris pada
// portal ASM per 2026-09-20, dan master ini bertambah hanya ketika seorang petugas diberi
// kewenangan baru. Layar lama pun memuat seluruhnya sekaligus. Menambahkan paginasi server
// di sini akan menambah kerumitan yang tidak menyelesaikan satu pun masalah nyata — layar
// yang datanya besar, seperti inbox dan laporan, tidak boleh mengikuti pola ini.
func (s *Service) List(ctx context.Context, portalAlias string, filter mastermasking.Filter) ([]mastermasking.Masking, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	filter = filter.Clean()
	if !filter.By.Known() {
		// Tipe pencarian yang tidak dikenal DITOLAK, tidak diam-diam diperlakukan sebagai
		// "tanpa penyaring". Menganggapnya tanpa penyaring akan menampilkan SELURUH baris
		// kepada pengguna yang mengira ia sedang mencari sesuatu yang sempit — dan isinya
		// adalah daftar siapa yang boleh melihat data pribadi.
		return nil, fmt.Errorf("mastermasking/usecase: tipe pencarian %q tidak dikenal", filter.By)
	}
	// Kata kunci tanpa tipe tidak menyaring apa pun, dan itu menyesatkan. Bila pengguna
	// mengetik tanpa memilih kolom, pencarian diarahkan ke login — kolom yang paling sering
	// dicari orang pada layar ini, dan satu-satunya yang ada di tabel itu sendiri.
	if filter.By == mastermasking.SearchByAll && filter.Keyword != "" {
		filter.By = mastermasking.SearchByLogin
	}
	// Status yang belum dipilih ditolak, bukan diabaikan. Layar lama berperilaku sama:
	// `Activity/SearchDataMasking-Act.xml` menyiapkan pesan "Pilih Status Aktif" untuk
	// keadaan ini. Mengabaikannya akan menampilkan seluruh baris kepada pengguna yang
	// mengira ia sedang menyaring menurut status.
	if filter.By == mastermasking.SearchByStatus {
		if _, valid := filter.StatusKeyword(); !valid {
			return nil, mastermasking.ErrStatusNotChosen
		}
	}

	list, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mastermasking/usecase: membaca daftar masking: %w", err)
	}
	return list, nil
}

// Get mengembalikan satu baris masking.
func (s *Service) Get(ctx context.Context, portalAlias, id string) (mastermasking.Masking, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastermasking.Masking{}, err
	}

	found, err := repo.Get(ctx, trim(id))
	if err != nil {
		if errors.Is(err, mastermasking.ErrNotFound) {
			return mastermasking.Masking{}, err
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/usecase: membaca masking %q: %w", id, err)
	}
	return found, nil
}

// Create menyisipkan baris baru dan mengembalikannya lengkap dengan ID yang dibuat
// penyimpanan.
//
// ID TIDAK diterima dari pemanggil. Sistem lama pun demikian: procedure yang
// menentukannya, dengan `MAX(TO_NUMBER(ID_MST))+1`
// (`Database/UPDATE_LOG_PROTEKSI.prc:33-34`).
//
// `by` adalah pelaku dari SESI, bukan dari badan permintaan. Karena `D-59` menjadikan
// jejak audit satu-satunya kontrol pengimbang, membiarkan klien menyebut pelakunya sendiri
// akan membuat jejak itu tidak membuktikan apa pun.
func (s *Service) Create(ctx context.Context, portalAlias string, m mastermasking.Masking, by string) (mastermasking.Masking, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastermasking.Masking{}, err
	}

	m = m.Clean()
	if err := mastermasking.NewValidationError(mastermasking.CheckMasking(m)); err != nil {
		return mastermasking.Masking{}, err
	}
	if err := ensureBranchExists(ctx, repo, m.BranchID); err != nil {
		return mastermasking.Masking{}, err
	}
	if err := ensurePairFree(ctx, repo, m.BranchID, m.Login, ""); err != nil {
		return mastermasking.Masking{}, err
	}

	m.InputBy = trim(by)
	m.InputAt = s.now()

	saved, err := repo.Insert(ctx, m)
	if err != nil {
		// Ketiga galat di bawah datang dari penegakan di penyimpanan, yang menang atas
		// pemeriksaan di atas bila dua permintaan tiba bersamaan. Ia diteruskan apa adanya
		// supaya pengguna melihat sebab yang benar.
		if errors.Is(err, mastermasking.ErrPairTaken) ||
			errors.Is(err, mastermasking.ErrIDTaken) ||
			errors.Is(err, mastermasking.ErrNoSequence) {
			return mastermasking.Masking{}, err
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/usecase: menambah masking: %w", err)
	}
	return saved, nil
}

// Update mengubah baris yang sudah ada.
//
// Cabang dan login IKUT dapat diubah, karena layar lama pun mengirimkan keduanya pada
// setiap penyimpanan. Konsekuensinya keunikan pasangan harus diperiksa ulang di sini —
// memindahkan seseorang ke cabang yang sudah punya barisnya sendiri harus ditolak, bukan
// menghasilkan dua kewenangan untuk orang yang sama di satu cabang.
func (s *Service) Update(ctx context.Context, portalAlias, id string, m mastermasking.Masking, by string) (mastermasking.Masking, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastermasking.Masking{}, err
	}

	id = trim(id)
	m = m.Clean()
	m.ID = id

	if err := mastermasking.NewValidationError(mastermasking.CheckMasking(m)); err != nil {
		return mastermasking.Masking{}, err
	}
	// Keberadaan baris diperiksa lebih dulu supaya mengubah baris yang tidak ada dijawab
	// "tidak ditemukan", bukan "sudah dipakai" yang menyesatkan.
	current, err := repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, mastermasking.ErrNotFound) {
			return mastermasking.Masking{}, err
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/usecase: membaca masking %q: %w", id, err)
	}
	if err := ensureBranchExists(ctx, repo, m.BranchID); err != nil {
		return mastermasking.Masking{}, err
	}
	if err := ensurePairFree(ctx, repo, m.BranchID, m.Login, id); err != nil {
		return mastermasking.Masking{}, err
	}

	// Status aktif DIAMBIL dari form, meniru layar lama.
	//
	// `Section/MasterProteksi_Sec-Section.xml` memuat isian berlabel "STATUS" yang terikat
	// `InputData.BranchID` (`:22449`) — dan `Activity/InsermaskingDataKlaimPnc_-Act.xml`
	// memetakan properti itu ke parameter `T_STSAKTF`. Jadi menyimpan form memang mengubah
	// status di sistem lama.
	//
	// Pagarnya ada di layar, bukan di sini, dan itu pun ditiru: tombol Ubah hanya muncul
	// pada baris AKTIF (`ActionMaskingData_Sec` bersyarat `.STS_AKTF=='AKTIF'`), sehingga
	// form tidak pernah terbuka untuk baris yang sudah nonaktif.
	//
	// `current` tetap dibaca — bukan untuk status, melainkan untuk membuktikan barisnya ada
	// sebelum keunikan pasangan diperiksa.
	_ = current
	m.InputBy = trim(by)
	m.InputAt = s.now()

	saved, err := repo.Update(ctx, m)
	if err != nil {
		if errors.Is(err, mastermasking.ErrNotFound) || errors.Is(err, mastermasking.ErrPairTaken) {
			return mastermasking.Masking{}, err
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/usecase: mengubah masking %q: %w", id, err)
	}
	return saved, nil
}

// SetActive mengaktifkan atau menonaktifkan satu baris.
//
// Inilah pengganti tombol DELETE layar lama. Menonaktifkan MENCABUT kewenangan melihat
// data pribadi: `RDB List/GetTipeProteksi-SQL.xml` dan
// `Activity/CekmaskingDataPerLoginUserKlaim-Act.xml` keduanya menyaring `STS_AKTF='AKTIF'`,
// sehingga baris nonaktif tidak lagi berlaku di mana pun.
func (s *Service) SetActive(ctx context.Context, portalAlias, id string, active bool, by string) (mastermasking.Masking, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return mastermasking.Masking{}, err
	}

	saved, err := repo.SetActive(ctx, trim(id), active, trim(by), s.now())
	if err != nil {
		if errors.Is(err, mastermasking.ErrNotFound) {
			return mastermasking.Masking{}, err
		}
		return mastermasking.Masking{}, fmt.Errorf("mastermasking/usecase: mengubah status masking %q: %w", id, err)
	}
	return saved, nil
}

// Branches mengembalikan pilihan cabang untuk isian CABANG.
//
// Ia membaca POOLDATA.BRANCH yang dimiliki sistem lain — hanya DIBACA, tidak pernah
// ditulis. Ia ada di modul ini, bukan di modul bersama, karena modul inilah satu-satunya
// yang memperlakukan kode cabang sebagai kunci asing sungguhan. Modul Master Surveyors
// menyimpan nama cabang sebagai teks biasa, sehingga kebutuhannya berbeda dan
// menyatukannya sekarang akan memaksa dua hal yang tidak sama menjadi satu.
func (s *Service) Branches(ctx context.Context, portalAlias, keyword string) ([]mastermasking.Branch, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.ListBranches(ctx, trim(keyword), DefaultBranchLimit)
	if err != nil {
		return nil, fmt.Errorf("mastermasking/usecase: membaca daftar cabang: %w", err)
	}
	return list, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (s *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := s.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("mastermasking/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// ensurePairFree menolak pasangan cabang+login yang sudah dipakai baris lain.
//
// exceptID dikosongkan saat menambah, dan diisi ID yang sedang diubah saat mengubah —
// tanpa itu, menyimpan ulang baris tanpa mengubah cabang maupun loginnya akan ditolak
// karena bentrok dengan dirinya sendiri.
//
// Pemeriksaan ini adalah KENYAMANAN, bukan jaminan: dua permintaan yang tiba bersamaan
// dapat sama-sama lolos di sini. Dan berbeda dari master lain yang sudah dibangun, di sini
// TIDAK ADA indeks unik yang menjadi jaring pengaman — ALL_INDEXES pada 2026-09-20 hanya
// memuat satu indeks NONUNIQUE. Penegakan keunikan di sistem lama pun sepenuhnya bersandar
// pada procedure, bukan pada basis data.
//
// Artinya pemeriksaan di sini adalah SATU-SATUNYA yang menolak pasangan ganda, dan itu
// keadaan yang dicatat terbuka — bukan yang disembunyikan. Indeks uniknya diusulkan
// bersama permintaan migrasi ke DBA.
func ensurePairFree(ctx context.Context, repo mastermasking.Repo, branchID, login, exceptID string) error {
	found, err := repo.FindByPair(ctx, branchID, login)
	if err != nil {
		if errors.Is(err, mastermasking.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("mastermasking/usecase: memeriksa keunikan cabang+pengguna: %w", err)
	}
	if exceptID != "" && found.ID == exceptID {
		return nil
	}
	return mastermasking.ErrPairTaken
}

// ensureBranchExists menolak kode cabang yang tidak ada di POOLDATA.BRANCH.
//
// Seluruh 25 baris portal ASM menunjuk cabang yang benar-benar ada, dan tidak ada foreign
// key yang menjaganya. Memeriksanya di sini mencegah baris menggantung yang namanya akan
// tampil kosong di grid — dan lebih penting lagi, mencegah kewenangan diberikan pada
// cabang yang tidak pernah ada, yang berarti ia tidak pernah berlaku sekaligus tidak
// pernah terlihat salah.
func ensureBranchExists(ctx context.Context, repo mastermasking.Repo, branchID string) error {
	exists, err := repo.BranchExists(ctx, branchID)
	if err != nil {
		return fmt.Errorf("mastermasking/usecase: memeriksa cabang %q: %w", branchID, err)
	}
	if !exists {
		return mastermasking.ErrBranchUnknown
	}
	return nil
}

// trim membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
