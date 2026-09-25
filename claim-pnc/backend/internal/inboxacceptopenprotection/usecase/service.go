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
	now         func() time.Time
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// Protections wajib diisi.
	Protections inboxacceptopenprotection.RepoSelector

	// Now dapat diisi uji supaya waktu akseptasi dapat diperiksa secara deterministik.
	Now func() time.Time
}

// NewService membentuk service.
func NewService(o Options) (*Service, error) {
	if o.Protections == nil {
		return nil, fmt.Errorf("inboxacceptopenprotection/usecase: pemilih repo proteksi wajib diisi")
	}

	now := o.Now
	if now == nil {
		now = time.Now
	}
	return &Service{protections: o.Protections, now: now}, nil
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
	// Di sini ia datang dari layar, dan itu KELEMAHAN YANG DISADARI: sampai tabel peran
	// (`TKT-F3-004`) dapat diisi, tidak ada yang mencegah pemanggil membuka antrean yang
	// bukan haknya. Begitu peran tersedia, pemeriksaannya masuk di sini — bukan di handler,
	// bukan di layar. Dicatat di `docs/keputusan-implementasi.md`.
	Queue inboxacceptopenprotection.Queue

	Search string
	Limit  int
	Offset int
}

// List membaca satu halaman antrean akseptasi.
func (s *Service) List(ctx context.Context, q ListQuery) (inboxacceptopenprotection.Page, error) {
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

// Get membaca satu proteksi untuk ditampilkan di form akseptasi.
//
// Galat ErrNotFound diteruskan APA ADANYA supaya transport dapat menjawab 404.
func (s *Service) Get(ctx context.Context, portalAlias, number string) (inboxacceptopenprotection.Protection, error) {
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
	// Ia disimpan sebagai `DIAKSEP_OLEH`, dan itulah satu-satunya jejak siapa yang
	// menyetujui pembukaan proteksi. `D-59` menetapkan tidak ada pemisahan tugas formal,
	// sehingga jejak audit menjadi satu-satunya kontrol pengimbang yang tersisa.
	By string
}

// Decide menuliskan keputusan akseptasi.
//
// # Urutan pemeriksaannya mengikat
//
//  1. keputusan dikenali
//  2. proteksi ada
//  3. proteksi lengkap — tertaut klaim dan berpolis
//  4. proteksi belum diputuskan orang lain
//
// Butir 4 diperiksa DI SINI dan DIULANG di penyimpanan. Itu bukan pengulangan yang sia-sia:
// pemeriksaan di sini menghasilkan pesan yang menjelaskan siapa yang sudah memutuskan;
// pemeriksaan di sana yang benar-benar menahan petugas kedua pada antrean bersama.
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

	repo, err := s.protections(cmd.PortalAlias)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	existing, err := repo.Get(ctx, cmd.Number)
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
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
