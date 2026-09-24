// Package usecase mengorkestrasi modul Inbox RCL/PUCL.
//
// Tiga operasi:
//
//	Metadata     menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List         mengambil isi satu tab
//	DailyReport  mengambil satu halaman laporan harian pada rentang tanggal
//
// Ekspor grid TIDAK menjadi operasi keempat: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go,
// dan DailyReport dipakai dengan pola yang sama.
//
// Tidak ada operasi yang menulis. Layar lama punya dua tindakan yang menulis — mencetak
// surat PUCL/RCL dan mengirim Reminder PUCL — dan keduanya menyentuh tabel yang masih
// dimiliki Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxrclpucl"
)

// Service melayani modul Inbox RCL/PUCL.
type Service struct {
	repoSelector inboxrclpucl.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxrclpucl.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox RCL/PUCL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxrclpucl/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah ketiga tab beserta kolomnya.
	Tabs []inboxrclpucl.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// ReportColumns adalah kolom berkas laporan harian.
	//
	// Ia dikirim ke layar meski layar tidak menggambarnya sebagai tabel: yang membacanya
	// adalah keterangan di dekat tombol unduh, supaya pengguna tahu isi berkasnya berbeda
	// dari tabel yang sedang dilihatnya SEBELUM mengunduh — bukan setelah membukanya.
	ReportColumns []inboxrclpucl.Column

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	columns := make([]inboxrclpucl.Column, len(inboxrclpucl.DailyReportColumns))
	copy(columns, inboxrclpucl.DailyReportColumns)

	return Metadata{
		Tabs:               inboxrclpucl.Tabs(),
		DefaultTab:         inboxrclpucl.DefaultTab,
		ReportColumns:      columns,
		PlannedDifferences: inboxrclpucl.PlannedDifferences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxrclpucl.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim: tab
	// yang diminta kosong menjadi tab bawaan, dan layar harus tahu tab mana yang
	// sebenarnya dijawab.
	Query inboxrclpucl.Query
}

// List mengambil isi satu tab.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	input inboxrclpucl.QueryInput,
	page inboxrclpucl.Pagination,
) (Listed, error) {
	query, err := inboxrclpucl.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan.
	//
	// Antrean layar ini BERSAMA: penyaringnya akun `RCLPUCL`, bukan pengguna yang login,
	// sehingga setiap petugas melihat daftar yang sama dan seluruh barisnya memuat nomor
	// polis serta nama tertanggung milik pekerjaan bersama.
	//
	// Selama pemeriksaan peran belum ada (`TKT-F3-004`), jejak inilah satu-satunya hal yang
	// menyatakan siapa yang membukanya — dan `D-59` menjadikan jejak audit satu-satunya
	// kontrol pengimbang justru untuk keadaan seperti ini.
	if s.logger != nil {
		s.logger.Info(
			"antrean RCL/PUCL dibuka",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("tab", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Reported adalah satu halaman laporan harian beserta permintaan yang dipakai.
type Reported struct {
	Rows []inboxrclpucl.DailyReportRow

	// Total adalah jumlah SELURUH baris laporan yang cocok, bukan yang ada di halaman ini.
	Total int

	// Pagination adalah paginasi yang benar-benar dipakai setelah dibetulkan.
	Pagination inboxrclpucl.Pagination

	// Request adalah permintaan setelah divalidasi.
	Request inboxrclpucl.ReportRequest
}

// DailyReport mengambil satu halaman laporan harian.
//
// # Kenapa ia operasi tersendiri, bukan penyaring pada List
//
// Karena isinya memang bukan isi tabel. Kuerinya berbeda, kolomnya berbeda, dan
// penyaringnya jauh lebih longgar — ia bahkan memuat klaim yang sudah selesai dan klaim
// Personal Accident yang tidak pernah masuk antrean RCL/PUCL. Lihat
// inboxrclpucl.DailyReportRow.
//
// Menjadikannya penyaring pada List akan membuat layar mengira ia dapat menampilkan
// hasilnya di tabel yang sama, padahal kolomnya pun tidak sama.
func (s *Service) DailyReport(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	input inboxrclpucl.ReportInput,
	page inboxrclpucl.Pagination,
) (Reported, error) {
	request, err := inboxrclpucl.NewReportRequest(input, caller)
	if err != nil {
		return Reported{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Reported{}, err
	}

	clean := page.Normalize()
	rows, total, err := repo.DailyReport(ctx, request.Range, clean)
	if err != nil {
		return Reported{}, fmt.Errorf(
			"mengambil laporan harian %s..%s: %w",
			request.Range.From, request.Range.To, err)
	}

	// Laporan dicatat TERPISAH dari pembukaan tab, dan rentang tanggalnya ikut.
	//
	// Alasannya bukan kelengkapan: berkas laporan dapat diunduh dan dibawa keluar, dan ia
	// memuat nomor polis serta nama tertanggung dalam jumlah yang ditentukan penggunanya
	// sendiri lewat rentang tanggal. Rentang yang lebar adalah hal yang harus dapat
	// ditelusuri setelahnya.
	if s.logger != nil {
		s.logger.Info(
			"laporan harian RCL/PUCL diminta",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("pemanggil", request.Caller.Login),
			slog.String("portal", portalAlias),
			slog.String("dari", request.Range.From),
			slog.String("sampai", request.Range.To),
			slog.Int("baris", len(rows)),
			slog.Int("total", total),
		)
	}

	return Reported{
		Rows:       rows,
		Total:      total,
		Pagination: clean,
		Request:    request,
	}, nil
}
