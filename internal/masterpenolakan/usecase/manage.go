// Package usecase mengorkestrasi modul Master Penolakan Klaim.
//
// Tugasnya empat, dan tidak lebih: memilih Repo milik portal yang sedang aktif,
// menjalankan pemeriksaan isian, melengkapi jejak siapa dan kapan, lalu memanggil Repo.
// Ia tidak tahu apa pun tentang HTTP maupun SQL.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/platform/clock"
)

// Service adalah pintu masuk seluruh perkara Master Penolakan Klaim — tingkat 1 dan 2.
//
// Master Penolakan Komite dilayani ServiceKomite di berkas sebelah: keduanya memilih
// penyimpanan yang berbeda, dan satu layanan yang melayani dua tabel akan menerima dua
// pemilih repo lalu bercabang di setiap method.
type Service struct {
	repoSelector masterpenolakan.RepoSelector

	// clock mengisi kolom TANGGALKIRIM.
	//
	// Ia seam, bukan time.Now() langsung, supaya jejak waktunya dapat diuji deterministik
	// dan supaya tidak ada satu pun penambahan 7 jam manual yang menyelinap masuk
	// (`08-TECHNICAL-STRATEGY.md` §4.4).
	clock clock.Clock
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpenolakan.RepoSelector

	// Clock menjadi sumber TANGGALKIRIM. Wajib.
	Clock clock.Clock
}

// NewService membentuk layanan dan menolak bahan yang tidak lengkap.
//
// Penolakannya terjadi saat perakitan di cmd, bukan saat permintaan pertama datang:
// rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpenolakan/usecase: RepoSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("masterpenolakan/usecase: Clock wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, clock: o.Clock}, nil
}

// List mengembalikan seluruh Status Penolakan 2 milik satu portal.
//
// # Kenapa TANPA penyaring status
//
// Kueri lama memuat `{ASIS:MasterCheckerPenolakan.RemakApprove}`
// (`RDB List/BrowseStatusPenolakanKlaim2-SQL.xml`), dan activity yang memanggilnya
// menyetel potongan itu menjadi `"WHERE STATUS='0'"`. Tetapi langkah penyetelan itu
// BERPRASYARAT `Param.master=="1"` (`Activity/BrowseStatusPenolakanKlaim_2-Act.xml`
// langkah 2), dan layar Master Penolakan Klaim TIDAK mengirim parameter itu — yang
// mengirimnya adalah layar checker pada Inbox Manager.
//
// Jadi di layar ini potongan penyaring tetap kosong dan grid menampilkan SELURUH baris.
// Menyalin penyaringnya ke sini akan menyembunyikan baris yang sudah diputuskan, dan itu
// perubahan perilaku — bukan replikasi.
//
// Potongan `{ASIS:...}` itu sendiri tidak dibawa dalam bentuk apa pun: merangkai teks SQL
// dari nilai klipboard adalah persis celah yang `08-TECHNICAL-STRATEGY.md` §4.3 tutup,
// dan di sini ia juga membawa kebocoran antarlayar — nilai yang tertinggal dari layar
// checker dalam sesi yang sama akan diam-diam menyaring layar ini.
func (l *Service) List(ctx context.Context, portalAlias string) ([]masterpenolakan.RejectionStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// ListParent mengembalikan pilihan "Status Penolakan 1" untuk daftar di layar.
//
// Ia dibaca dari tabel milik entitas yang bersangkutan, bukan daftar tetap milik
// aplikasi — dua entitas punya alasan penolakan yang berbeda, dan menyajikan daftar satu
// entitas kepada entitas lain adalah kebocoran yang justru dicegah R-20.
func (l *Service) ListParent(ctx context.Context, portalAlias string) ([]masterpenolakan.RejectionStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.ListParent(ctx)
}

// Get mengembalikan satu Status Penolakan 2 milik satu portal.
func (l *Service) Get(ctx context.Context, portalAlias, id string) (masterpenolakan.RejectionStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyimpan satu Status Penolakan 2 baru dan mengembalikan baris tersimpannya.
//
// `by` adalah login pemanggil, yang mengisi kolom USER_INPUT. Ia datang dari sesi, tidak
// pernah dari badan permintaan: kolom itu adalah jejak pertanggungjawaban, dan menerimanya
// dari peramban berarti siapa pun dapat mengaku sebagai orang lain.
//
// Baik ID baris, ID induk yang baru, maupun statusnya tidak diterima dari pemanggil —
// ketiganya diterbitkan penyimpanan.
func (l *Service) Create(ctx context.Context, portalAlias, by string, input masterpenolakan.Input) (masterpenolakan.RejectionStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	submission, err := l.prepare(by, input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}
	return repo.InsertNew(ctx, submission)
}

// Update menyimpan perubahan pada Status Penolakan 2 yang sudah ada.
//
// ID tidak pernah ikut berubah: ia diambil dari jalur URL dan dipakai hanya sebagai
// penyaring, persis seperti `Database/MASTERPENOLAKANKLAIM2.prc:15`. Baris lain yang
// sudah merujuk ID itu karena itu tidak terputus.
//
// Pengubahan MENGEMBALIKAN baris ke antrean persetujuan — alasannya beserta buktinya ada
// pada doc comment masterpenolakan.Repo.Update.
func (l *Service) Update(ctx context.Context, portalAlias, by, id string, input masterpenolakan.Input) (masterpenolakan.RejectionStatus2, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	submission, err := l.prepare(by, input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}
	return repo.Update(ctx, id, submission)
}

// prepare membersihkan isian, memeriksanya, lalu melengkapinya dengan siapa dan kapan.
//
// Satu fungsi untuk Create dan Update supaya keduanya tidak pernah berbeda urutan: yang
// dibersihkan lebih dulu, baru diperiksa, baru diberi jejak. Membalik dua yang pertama
// membuat isian berisi spasi saja lolos pemeriksaan wajib-isi.
func (l *Service) prepare(by string, input masterpenolakan.Input) (masterpenolakan.Submission, error) {
	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpenolakan.Submission{}, err
	}
	return masterpenolakan.Submission{
		Input: clean,
		By:    by,
		At:    l.clock.Now(),
	}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
//
// Dipakai transport untuk menolak lebih awal, sebelum badan permintaan dibaca.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterpenolakan/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// Catatan: TIDAK ADA method Delete di sini, dan itu bukan pekerjaan yang belum selesai.
//
// Seluruh export tidak memuat satu pun DELETE terhadap kedua tabel ini, layar lama pun
// tidak punya tombolnya. Alasan lengkapnya ada pada doc comment masterpenolakan.Repo.
