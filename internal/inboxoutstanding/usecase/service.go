// Package usecase mengorkestrasi modul My Inbox.
//
// Isinya satu hal, dan justru karena satu hal itulah ia layak dipisahkan dari transport:
// MENGIKAT DAFTAR KE IDENTITAS PEMANGGIL sebelum klaim dibaca.
//
// Menaruhnya di handler akan membuat setiap rute baru harus mengingat untuk melakukannya,
// dan yang lupa tidak menghasilkan galat apa pun — hanya pekerjaan operator lain yang ikut
// tampil di layar bernama "My Inbox". Kegagalan seperti itu tidak terlihat saat dibaca
// maupun saat diuji secara sepintas.
package usecase

import (
	"context"
	"fmt"
	"time"

	"claim-pnc/internal/inboxoutstanding"
)

// Service membaca pekerjaan milik pemanggil pada portal yang sedang dibukanya.
type Service struct {
	claims inboxoutstanding.RepoSelector
}

// NewService membentuk service.
//
// Klaim datang lewat SELECTOR, bukan repo tunggal: tabelnya ada di basis data setiap
// entitas (`ADR-0030`).
//
// # Kenapa tidak ada lagi repo lini bisnis
//
// Service ini sempat membaca `M_LOGIN_PNC.LINE_BUSINESS` untuk menurunkan batas data per
// lini. Batas itu berasal dari `Activity/InboxOutstanding_Act-Act.xml` — activity layar
// LAIN, yang tidak pernah dipanggil harness maupun section rujukan modul ini.
//
// `Report Definition/InboxRegister_RD-RD.xml` memperlakukan panel sebagai PARAMETER
// (`Param.panel`), bukan sebagai pita tetap per pengguna. Yang mengikat daftar ke
// penggunanya adalah `Param.assign` — operator pemegang tugas.
func NewService(claims inboxoutstanding.RepoSelector) (*Service, error) {
	if claims == nil {
		return nil, fmt.Errorf("inboxoutstanding/usecase: pemilih repo klaim wajib diisi")
	}
	return &Service{claims: claims}, nil
}

// Query adalah permintaan dari layar.
//
// Ia sengaja TIDAK memuat pemilik pekerjaan: siapa "saya" bukan sesuatu yang boleh diminta
// klien. Yang datang dari klien hanyalah pencarian, penyaring opsional, dan paginasi;
// pemiliknya diturunkan di sini dari LoginID.
type Query struct {
	// LoginID adalah identitas pemanggil, diambil dari sesi — bukan dari badan permintaan.
	//
	// Dari sinilah penyaring `PXASSIGNEDOPERATORID` diisi. Membiarkannya datang dari klien
	// berarti siapa pun dapat membaca pekerjaan orang lain dengan mengubah satu parameter.
	LoginID string

	// PortalAlias adalah entitas yang sedang dibuka, diambil dari header `X-Portal` yang
	// sudah diperiksa middleware. Ia menentukan BASIS DATA mana yang dibaca.
	//
	// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai
	// cadangan (`R-20`).
	PortalAlias string

	Search     string
	GroupPanel string
	RCVID      string
	Stage      string
	BranchCode string

	// DocumentStatus diisi saat pengguna mengeklik irisan donut ringkasan.
	DocumentStatus inboxoutstanding.DocumentStatus

	Limit  int
	Offset int
}

// Result adalah satu halaman hasil.
type Result struct {
	Page inboxoutstanding.Page

	// AssignedTo adalah pemilik pekerjaan yang dipakai menyaring.
	//
	// Ia ikut dikembalikan supaya LAYAR DAPAT MENYATAKANNYA: pengguna yang melihat daftar
	// kosong perlu tahu bahwa yang ditampilkan memang pekerjaan miliknya, bukan hasil
	// penyaring yang salah.
	AssignedTo string
}

// List membaca satu halaman pekerjaan milik pemanggil.
//
// Urutannya mengikat: identitas dipastikan LEBIH DULU, baru klaim dibaca. Membalik
// urutannya berarti membaca dulu lalu menyaring di memori — dan pekerjaan yang bukan milik
// pemanggil sempat berada di proses aplikasi serta ikut terhitung pada paginasi, sehingga
// bocor lewat jumlah baris (`11-SECURITY.md` §3.2).
func (s *Service) List(ctx context.Context, q Query) (Result, error) {
	if q.LoginID == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, ia cacat pemrograman —
		// dan menjawabnya dengan "tampilkan semuanya" akan menyembunyikan cacat itu.
		return Result{}, fmt.Errorf("inboxoutstanding/usecase: identitas pemanggil kosong")
	}

	// Penyimpanan dipilih menurut portal SEBELUM klaim dibaca. Galat di sini tidak
	// dialihkan ke koneksi mana pun sebagai cadangan — ia dikembalikan apa adanya.
	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return Result{}, err
	}

	// Identitas LAMA dibaca sebelum klaim, dan dari portal yang sama.
	//
	// Login baru memakai email lewat HCC, sedangkan klaim warisan tertugas ke nama
	// operator Pega. Tanpa langkah ini, 20 dari 29 operator pada data ASM membuka layar
	// kosong yang tampak rapi — salah satunya menyembunyikan 79 klaim berjalan.
	//
	// Kegagalan membacanya DIKEMBALIKAN, tidak ditelan: melanjutkan dengan identitas
	// tunggal akan menampilkan daftar yang tampak sah tetapi kurang.
	legacy, err := claims.LegacyOperatorFor(ctx, q.LoginID)
	if err != nil {
		return Result{}, fmt.Errorf("inboxoutstanding/usecase: membaca identitas lama pemanggil: %w", err)
	}

	page, err := claims.List(ctx, q.filter(legacy))
	if err != nil {
		return Result{}, fmt.Errorf("inboxoutstanding/usecase: membaca daftar klaim: %w", err)
	}

	return Result{Page: page, AssignedTo: q.LoginID}, nil
}

// filter menyusun penyaring penyimpanan dari permintaan layar.
//
// Ia satu tempat supaya daftar dan ringkasan TIDAK dapat menyimpang: keduanya wajib
// menyaring populasi yang sama, kalau tidak angka pada donut berbeda dari isi grid dan
// tidak ada yang menandainya.
func (q Query) filter(legacy string) inboxoutstanding.Filter {
	return inboxoutstanding.Filter{
		AssignedTo:       q.LoginID,
		AssignedToLegacy: legacy,
		Search:           q.Search,
		GroupPanel:       q.GroupPanel,
		RCVID:            q.RCVID,
		Stage:            q.Stage,
		BranchCode:       q.BranchCode,
		DocumentStatus:   q.DocumentStatus,
		Limit:            q.Limit,
		Offset:           q.Offset,
	}
}

// SummaryResult adalah ringkasan inbox beserta pemiliknya.
type SummaryResult struct {
	Summary inboxoutstanding.Summary

	// AssignedTo ikut dikembalikan dengan alasan yang sama seperti pada Result: donut
	// kosong punya dua sebab yang tampak sama, dan menyebut pemiliknya membedakan keduanya.
	AssignedTo string
}

// Summary menghitung isi inbox pemanggil per status kelengkapan dokumen.
//
// Ia menempuh JALUR YANG SAMA dengan List — identitas lama dibaca lebih dulu, penyaring
// disusun oleh Query.filter — supaya angka donut dan isi grid tidak mungkin berasal dari
// populasi yang berbeda.
func (s *Service) Summary(ctx context.Context, q Query) (SummaryResult, error) {
	if q.LoginID == "" {
		return SummaryResult{}, fmt.Errorf("inboxoutstanding/usecase: identitas pemanggil kosong")
	}

	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return SummaryResult{}, err
	}

	legacy, err := claims.LegacyOperatorFor(ctx, q.LoginID)
	if err != nil {
		return SummaryResult{}, fmt.Errorf("inboxoutstanding/usecase: membaca identitas lama pemanggil: %w", err)
	}

	summary, err := claims.SummarizeDocumentStatus(ctx, q.filter(legacy))
	if err != nil {
		return SummaryResult{}, fmt.Errorf("inboxoutstanding/usecase: meringkas status dokumen: %w", err)
	}

	return SummaryResult{Summary: summary, AssignedTo: q.LoginID}, nil
}

// ExportQuery adalah permintaan unduhan CSV.
//
// Ia TIDAK memuat pemilik pekerjaan, dan itu bukan kelalaian. LoginID tetap ada, tetapi
// dipakai untuk hal yang berbeda: mencari LINI BISNIS pemanggil, bukan menyaring baris
// menurut siapa yang memegangnya.
type ExportQuery struct {
	// LoginID dipakai HANYA untuk membaca lini bisnis pemanggil.
	LoginID string

	// PortalAlias menentukan basis data yang dibaca — sama seperti pada Query.
	PortalAlias string

	// From dan To menyaring tanggal pendaftaran; nil berarti tidak menyaring.
	//
	// To bersifat EKSKLUSIF: pemanggil mengirim awal hari BERIKUTNYA.
	From *time.Time
	To   *time.Time

	Limit  int
	Offset int
}

// ExportResult adalah sekumpulan baris unduhan beserta cakupannya.
type ExportResult struct {
	Page inboxoutstanding.Page

	// LineBusiness adalah cakupan yang benar-benar dipakai.
	//
	// Ia ikut dikembalikan supaya dapat dicatat dan, kelak, disebut pada berkasnya. Berkas
	// 354 baris dan berkas 862 baris sama-sama tampak wajar; yang membedakan keduanya hanya
	// cakupan ini, dan tanpa menyebutnya tidak ada cara tahu mana yang sedang dipegang.
	LineBusiness inboxoutstanding.LineBusiness
}

// Export membaca sekumpulan baris untuk unduhan CSV.
//
// # Kenapa ia tidak memanggil List
//
// Export SEMPAT memanggil List dengan paginasi diputar. Akibatnya ia ikut terkena penyaring
// `PXASSIGNEDOPERATORID`, sehingga petugas yang inbox-nya kosong mengunduh berkas kosong —
// padahal `RDB List/ExportDataDetailKlaim-SQL.xml` **tidak menyaring operator sama sekali**
// dan berkasnya di Pega tetap berisi.
//
// Yang membatasi export adalah cakupan lini bisnis pemanggil, dan itu dibaca di sini.
func (s *Service) Export(ctx context.Context, q ExportQuery) (ExportResult, error) {
	if q.LoginID == "" {
		return ExportResult{}, fmt.Errorf("inboxoutstanding/usecase: identitas pemanggil kosong")
	}

	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return ExportResult{}, err
	}

	// Lini bisnis dibaca dari portal yang SAMA dengan klaimnya. Membacanya dari portal lain
	// akan memberi cakupan satu entitas pada data entitas lain (`R-20`).
	line, err := claims.LineBusinessFor(ctx, q.LoginID)
	if err != nil {
		return ExportResult{}, fmt.Errorf("inboxoutstanding/usecase: membaca lini bisnis pemanggil: %w", err)
	}

	page, err := claims.Export(ctx, inboxoutstanding.ExportFilter{
		LineBusiness: line,
		From:         q.From,
		To:           q.To,
		Limit:        q.Limit,
		Offset:       q.Offset,
	})
	if err != nil {
		return ExportResult{}, fmt.Errorf("inboxoutstanding/usecase: membaca baris export: %w", err)
	}

	return ExportResult{Page: page, LineBusiness: line}, nil
}
