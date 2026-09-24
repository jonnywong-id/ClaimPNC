// Package usecase mengorkestrasi modul Inbox Outstanding.
//
// Isinya satu hal, dan justru karena satu hal itulah ia layak dipisahkan dari transport:
// MENURUNKAN BATAS DATA dari identitas pemanggil sebelum membaca klaim.
//
// Menaruhnya di handler akan membuat setiap rute baru harus mengingat untuk melakukannya,
// dan yang lupa tidak menghasilkan galat apa pun — hanya klaim lini lain yang ikut tampil.
// Kegagalan seperti itu tidak terlihat saat dibaca maupun saat diuji secara sepintas.
package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxoutstanding"
)

// Service membaca daftar klaim yang masih berjalan, terbatas pada lini yang boleh dilihat
// pemanggil dan pada portal yang sedang dibukanya.
type Service struct {
	claims inboxoutstanding.RepoSelector
	lines  inboxoutstanding.LineBusinessRepo
}

// NewService membentuk service. Keduanya wajib diisi.
//
// Klaim datang lewat SELECTOR, bukan repo tunggal: tabelnya ada di basis data setiap
// entitas (`ADR-0030`). Lini bisnis datang lewat repo tunggal, dan itu disengaja —
// `M_LOGIN_PNC` adalah master pengguna, dan satu login berlaku di keempat entitas
// (`D-78`).
func NewService(claims inboxoutstanding.RepoSelector, lines inboxoutstanding.LineBusinessRepo) (*Service, error) {
	if claims == nil {
		return nil, fmt.Errorf("inboxoutstanding/usecase: pemilih repo klaim wajib diisi")
	}
	if lines == nil {
		return nil, fmt.Errorf("inboxoutstanding/usecase: repo lini bisnis wajib diisi")
	}
	return &Service{claims: claims, lines: lines}, nil
}

// Query adalah permintaan dari layar.
//
// Ia sengaja TIDAK memuat LineScope: batas data bukan sesuatu yang boleh diminta klien.
// Yang datang dari klien hanyalah pencarian, tahap, cabang, dan paginasi; batasnya
// diturunkan di sini dari LoginID.
type Query struct {
	// LoginID adalah identitas pemanggil, diambil dari sesi — bukan dari badan permintaan.
	LoginID string

	// PortalAlias adalah entitas yang sedang dibuka, diambil dari header `X-Portal` yang
	// sudah diperiksa middleware. Ia menentukan BASIS DATA mana yang dibaca.
	//
	// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai
	// cadangan (`R-20`).
	PortalAlias string

	Search     string
	Stage      string
	BranchCode string
	Limit      int
	Offset     int
}

// Result adalah satu halaman hasil beserta batas data yang berlaku saat membacanya.
type Result struct {
	Page inboxoutstanding.Page

	// Scope ikut dikembalikan supaya LAYAR DAPAT MENYATAKANNYA kepada pengguna.
	//
	// Ini bukan kebocoran lapisan melainkan kebutuhan nyata: ketika seorang pengguna tidak
	// punya lini bisnis, ia melihat klaim SELURUH lini — dan tanpa keterangan, ia tidak
	// punya cara mengetahui bahwa yang dilihatnya lebih luas dari yang seharusnya.
	Scope inboxoutstanding.LineScope

	// LineLookupError terisi bila lini bisnis GAGAL DIBACA, bukan sekadar tidak ada.
	//
	// Keduanya menghasilkan Scope yang sama — tanpa batas — sehingga tanpa field ini
	// kegagalan basis data tidak dapat dibedakan dari pengguna yang datanya memang belum
	// diisi. Yang pertama perlu diketahui operator; yang kedua tidak.
	//
	// Ia sengaja BUKAN galat yang dikembalikan: mengembalikannya akan mematikan layar.
	// Pemanggil mencatatnya, lalu melanjutkan. Lihat scopeFor.
	LineLookupError error
}

// List membaca satu halaman klaim yang masih berjalan.
//
// Urutannya mengikat: batas data ditentukan LEBIH DULU, baru klaim dibaca. Membalik
// urutannya berarti membaca dulu lalu menyaring di memori — dan data yang tidak boleh
// dilihat sempat berada di proses aplikasi serta ikut terhitung pada paginasi, sehingga
// bocor lewat jumlah baris (`11-SECURITY.md` §3.2).
func (s *Service) List(ctx context.Context, q Query) (Result, error) {
	scope, lookupErr, err := s.scopeFor(ctx, q.LoginID)
	if err != nil {
		return Result{}, err
	}

	// Penyimpanan dipilih menurut portal SEBELUM klaim dibaca. Galat di sini tidak
	// dialihkan ke koneksi mana pun sebagai cadangan — ia dikembalikan apa adanya.
	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return Result{}, err
	}

	page, err := claims.List(ctx, inboxoutstanding.Filter{
		Search:     q.Search,
		Stage:      q.Stage,
		BranchCode: q.BranchCode,
		Scope:      scope,
		Limit:      q.Limit,
		Offset:     q.Offset,
	})
	if err != nil {
		return Result{}, fmt.Errorf("inboxoutstanding/usecase: membaca daftar klaim: %w", err)
	}

	return Result{Page: page, Scope: scope, LineLookupError: lookupErr}, nil
}

// scopeFor menurunkan batas data dari identitas pemanggil.
//
// Mengembalikan tiga nilai: batas yang berlaku, galat pembacaan yang DITELAN dengan
// sengaja, dan galat yang benar-benar menghentikan permintaan.
//
// # Kenapa galat pembacaan tidak menghentikan permintaan
//
// Kolom `LINEBUSINESS` belum ada sampai migrasi 0004 dijalankan DBA, sehingga pembacaannya
// akan gagal di setiap lingkungan hari ini. Menghentikan permintaan berarti layar ini mati
// total sampai perubahan skema selesai — padahal Work Owner menetapkan pengguna tanpa lini
// tetap melihat data, sama seperti di Pega.
//
// Yang dilakukan: galat diperlakukan sama dengan "lini tidak diketahui", yaitu jatuh ke
// Unrestricted. Hasil akhirnya persis sama dengan perilaku Pega, dan layar tetap berguna.
//
// KONSEKUENSI YANG DITERIMA SADAR: galat basis data yang sesungguhnya — koneksi putus,
// tabel terkunci — menghasilkan batas data yang sama dengan pengguna tanpa lini, yaitu
// melihat semuanya. Yang membedakannya hanyalah galat yang dikembalikan di sini, dan
// pemanggillah yang wajib mencatatnya. Bila pemanggil mengabaikannya, kegagalan itu
// menjadi tidak terlihat — karena itu handler WAJIB mencatat Result.LineLookupError.
func (s *Service) scopeFor(ctx context.Context, loginID string) (scope inboxoutstanding.LineScope, lookupErr error, fatal error) {
	if loginID == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, ia cacat pemrograman —
		// dan menjawabnya dengan "lihat semuanya" akan menyembunyikan cacat itu.
		return inboxoutstanding.LineScope{}, nil,
			fmt.Errorf("inboxoutstanding/usecase: identitas pemanggil kosong")
	}

	line, err := s.lines.LineBusinessFor(ctx, loginID)
	if err != nil {
		return inboxoutstanding.ScopeFor(""),
			fmt.Errorf("inboxoutstanding/usecase: membaca lini bisnis %q: %w", loginID, err),
			nil
	}
	return inboxoutstanding.ScopeFor(line), nil, nil
}
