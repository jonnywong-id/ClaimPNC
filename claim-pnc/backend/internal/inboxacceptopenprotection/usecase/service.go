// Package usecase mengorkestrasi modul Inbox Accept Open Protection.
//
// Isinya dua hal yang tidak boleh hidup di handler maupun di penyimpanan:
//
//  1. pemilihan ANTREAN yang boleh dibuka pemanggil — PREMI atau NON PREMI;
//  2. pemilihan PENYIMPANAN menurut portal yang sedang dibuka.
//
// Keduanya adalah batas data. Menaruhnya di handler berarti setiap rute baru harus
// mengingat untuk melakukannya, dan yang lupa tidak menghasilkan galat apa pun — hanya
// proteksi milik antrean lain yang ikut tampil, atau proteksi badan hukum lain yang ikut
// diakseptasi.
package usecase

import (
	"context"
	"fmt"
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// Service melayani antrean akseptasi proteksi.
type Service struct {
	protections inboxacceptopenprotection.RepoSelector
	groups      inboxacceptopenprotection.GroupReader
	now         func() time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// Protections wajib diisi.
	Protections inboxacceptopenprotection.RepoSelector

	// Groups wajib diisi. Ia membaca access group pemanggil dari
	// `POOLDATA.M_LOGIN_GROUP_PNC` pada koneksi UTAMA.
	//
	// Sengaja TIDAK boleh kosong. Membiarkannya opsional berarti sebuah perakitan yang lupa
	// memasangnya akan berjalan tanpa pemeriksaan kewenangan sama sekali — dan lupanya tidak
	// menghasilkan galat apa pun, hanya layar akseptasi yang terbuka bagi siapa saja.
	Groups inboxacceptopenprotection.GroupReader

	// Now dapat diisi uji supaya waktu akseptasi dapat diperiksa secara deterministik.
	Now func() time.Time
}

// NewService membentuk service.
func NewService(o Options) (*Service, error) {
	if o.Protections == nil {
		return nil, fmt.Errorf("inboxacceptopenprotection/usecase: pemilih repo proteksi wajib diisi")
	}
	if o.Groups == nil {
		return nil, fmt.Errorf("inboxacceptopenprotection/usecase: pembaca access group wajib diisi")
	}

	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Service{protections: o.Protections, groups: o.Groups, now: now}, nil
}

// authorize membaca access group pemanggil lalu memastikan ia berhak atas layar ini.
//
// Dipanggil pada SETIAP jalur — daftar, rincian, dan keputusan. Memeriksanya hanya pada
// daftar akan menutup pintu depan sementara membiarkan tautan langsung ke rincian dan jalur
// keputusan tetap terbuka.
//
// Galat pembacaan TIDAK diperlakukan sebagai "tidak berwenang": keduanya menuntut jawaban
// berbeda — yang satu 403, yang lain 500 — dan menyamakannya akan menyembunyikan gangguan
// basis data di balik pesan yang menyalahkan pengguna.
func (s *Service) authorize(ctx context.Context, login string) ([]string, error) {
	groups, err := s.groups.GroupsOf(ctx, login)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxacceptopenprotection/usecase: membaca access group pemanggil: %w", err)
	}
	if !inboxacceptopenprotection.CanOpenScreen(groups) {
		return nil, inboxacceptopenprotection.ErrForbidden
	}
	return groups, nil
}

// ListQuery adalah permintaan daftar dari layar.
type ListQuery struct {
	// PortalAlias adalah entitas yang sedang dibuka, dari header `X-Portal` yang sudah
	// diperiksa middleware.
	PortalAlias string

	// Queue adalah antrean yang diminta layar.
	//
	// Di sistem lama ia TIDAK diminta klien melainkan ditentukan access group pemanggil:
	// grid PREMI hanya tampil bagi `GCNMFW:PncCollection`, grid NON PREMI bagi peran lain.
	//
	// Layar tetap mengirimkannya, tetapi ia DIPERIKSA di sini terhadap access group
	// pemanggil — bukan dipercaya apa adanya. Sejak `M_LOGIN_GROUP_PNC` diisi nama access
	// group Pega (Work Owner, 2026-09-25), antrean yang bukan haknya dijawab 403.
	Queue inboxacceptopenprotection.Queue

	// Login adalah login yang DIKETIK pengguna, bukan NIK — itulah yang dicocokkan ke
	// `M_LOGIN_GROUP_PNC.LOGIN_ID`.
	Login string

	Search string
	Limit  int
	Offset int
}

// List membaca satu halaman antrean akseptasi.
func (s *Service) List(ctx context.Context, q ListQuery) (inboxacceptopenprotection.Page, error) {
	groups, err := s.authorize(ctx, q.Login)
	if err != nil {
		return inboxacceptopenprotection.Page{}, err
	}
	if !inboxacceptopenprotection.CanOpenQueue(groups, q.Queue.Normalize()) {
		return inboxacceptopenprotection.Page{}, inboxacceptopenprotection.ErrForbidden
	}

	repo, err := s.protections(q.PortalAlias)
	if err != nil {
		return inboxacceptopenprotection.Page{}, err
	}

	page, err := repo.List(ctx, inboxacceptopenprotection.Filter{
		Queue:  q.Queue,
		Search: q.Search,
		Limit:  q.Limit,
		Offset: q.Offset,
	})
	if err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/usecase: membaca antrean akseptasi: %w", err)
	}
	return page, nil
}

// Queues menyebut antrean yang boleh dibuka pemanggil.
//
// Layar memakainya untuk menggambar tab. Di Pega tidak ada pemilihan sama sekali — grid yang
// bukan haknya tidak dirender — sehingga menggambar tab yang pasti ditolak adalah perilaku
// yang tidak ada padanannya, dan hanya membuat pengguna mencobanya.
func (s *Service) Queues(ctx context.Context, login string) ([]inboxacceptopenprotection.Queue, error) {
	groups, err := s.authorize(ctx, login)
	if err != nil {
		return nil, err
	}
	return inboxacceptopenprotection.QueuesFor(groups), nil
}

// Get membaca satu proteksi untuk ditampilkan di form akseptasi.
//
// Galat ErrNotFound diteruskan APA ADANYA supaya transport dapat menjawab 404.
//
// Kewenangan diperiksa atas LAYAR, bukan atas antrean baris yang dibuka. Sebuah proteksi
// dapat berpindah antrean setelah tautannya disimpan — tipe proteksinya disunting pemohon —
// dan menolak pembacaan karenanya akan menampilkan 403 pada baris yang tadinya sah dibuka.
// Yang dijaga ketat adalah KEPUTUSAN, bukan pembacaan.
func (s *Service) Get(
	ctx context.Context,
	portalAlias, number, login string,
) (inboxacceptopenprotection.Protection, error) {
	if _, err := s.authorize(ctx, login); err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	repo, err := s.protections(portalAlias)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}
	return repo.Get(ctx, number)
}

// DecideCommand adalah permintaan mengakseptasi.
type DecideCommand struct {
	PortalAlias string
	Number      string

	// Decision adalah setuju atau tolak. Nilai lain ditolak; tidak ada nilai bawaan.
	Decision inboxacceptopenprotection.Decision

	// By adalah identitas pemanggil, diambil dari sesi — BUKAN dari badan permintaan.
	//
	// Ia disimpan sebagai `RESOLVED_BY`, dan itulah satu-satunya jejak siapa yang
	// menyetujui pembukaan proteksi. `D-59` menetapkan tidak ada pemisahan tugas formal,
	// sehingga jejak audit menjadi satu-satunya kontrol pengimbang yang tersisa.
	By string
}

// Decide menuliskan keputusan akseptasi.
//
// # Urutan pemeriksaannya mengikat
//
//  1. keputusan dikenali
//  2. pemanggil berwenang atas layar INI dan atas antrean baris yang diputuskan
//  3. proteksi ada
//  4. proteksi lengkap — tertaut klaim dan berpolis
//  5. proteksi belum diputuskan orang lain
//
// Butir 5 diperiksa DI SINI dan DIULANG di penyimpanan. Itu bukan pengulangan yang sia-sia:
// pemeriksaan di sini menghasilkan pesan yang menjelaskan siapa yang sudah memutuskan;
// pemeriksaan di sana yang benar-benar menahan petugas kedua pada antrean bersama.
//
// # Butir 2 lebih ketat daripada pada Get, dan itu disengaja
//
// Pembacaan cukup menuntut kewenangan atas LAYAR; keputusan menuntut kewenangan atas
// ANTREAN baris yang bersangkutan. Petugas non-premi yang membuka proteksi PREMI lewat
// tautan boleh melihatnya, tetapi tidak boleh memutuskannya — di Pega, gridnya bahkan tidak
// pernah dirender baginya.
//
// Pemeriksaannya memakai tipe proteksi yang BARU DIBACA dari penyimpanan, bukan yang dikirim
// klien. Nilai dari klien dapat dipilih sendiri oleh pengirimnya.
func (s *Service) Decide(ctx context.Context, cmd DecideCommand) (inboxacceptopenprotection.Protection, error) {
	if !cmd.Decision.Valid() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrUnknownDecision
	}
	if cmd.By == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Menyimpannya dengan pelaku kosong akan
		// menghasilkan akseptasi yang tidak dapat ditelusuri — dan akseptasi adalah
		// persetujuan atas uang.
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/usecase: identitas pemanggil kosong")
	}

	groups, err := s.authorize(ctx, cmd.By)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	repo, err := s.protections(cmd.PortalAlias)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	existing, err := repo.Get(ctx, cmd.Number)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	// Antrean baris ini diturunkan dari TIPE yang tersimpan, bukan dari apa pun yang dikirim
	// klien. Petugas non-premi tidak boleh memutuskan proteksi PREMI, dan sebaliknya.
	if !inboxacceptopenprotection.CanOpenQueue(groups, inboxacceptopenprotection.QueueOf(existing.Type)) {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrForbidden
	}

	if existing.ClaimNumber == "" || existing.PolicyNumber == "" {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrIncomplete
	}
	if !existing.Pending() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrAlreadyDecided
	}

	saved, err := repo.Decide(ctx, cmd.Number, cmd.Decision, cmd.By, s.now())
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}
	return saved, nil
}
