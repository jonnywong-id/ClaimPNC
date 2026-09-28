// Package usecase mengorkestrasi modul Report KPI PNC.
//
// Empat operasi:
//
//	Metadata   menyerahkan daftar tab, grid, komponen, tipe report, dan selisih terencana
//	Summary    mengambil grid Summary KPI Adjuster
//	Detail     mengambil satu halaman grid Detail KPI Adjuster
//	Adjusters  mengambil isi dropdown Adjuster
//
// Ekspor TIDAK menjadi operasi kelima: ia memanggil Summary sekali atau Detail berulang
// kali, halaman demi halaman, lalu menuliskan hasilnya langsung ke jawaban. Menaruhnya di
// sini akan memaksa seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// Tidak ada operasi yang menulis, dan itu bukan pekerjaan yang tertinggal — lihat doc
// paket `reportkpi`.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/reportkpi"
)

// Service melayani modul Report KPI PNC.
type Service struct {
	repoSelector reportkpi.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector reportkpi.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Report KPI PNC.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("reportkpi/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi laporan.
type Metadata struct {
	// Tabs adalah ketiga tab layar, termasuk kedua yang belum dibangun beserta sebabnya.
	Tabs []reportkpi.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// Components adalah kesembilan komponen KPI, dalam urutan kolom layar lama.
	//
	// Layar merakit kolom grid dari Grid.Columns lebih dulu, lalu menambahkan kesembilan
	// ini di belakangnya. Keduanya dikirim terpisah supaya satu daftar komponen melayani
	// kedua grid — lihat catatan pada Grid.Columns.
	Components []reportkpi.Component

	// ReportTypes adalah ketiga pilihan dropdown "Pilih Tipe Report".
	ReportTypes []reportkpi.ReportTypeOption

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan, tab Adjuster.
	PlannedDifferences []string

	// AdminGroups adalah isi dropdown "Pilih Data KPI" pada tab KPI Admin.
	AdminGroups []reportkpi.AdminGroupOption

	// AdminPlannedDifferences adalah selisih terencana tab KPI Admin.
	//
	// Ia TERPISAH dari yang di atas: keduanya menyangkut layar yang berbeda, dan
	// menggabungkannya akan membuat pengguna satu tab membaca butir yang tidak berlaku
	// baginya — lalu berhenti membaca seluruhnya.
	AdminPlannedDifferences []string

	// CoordinatorInQuery adalah nama koordinator sebagaimana ditulis di TEKS KUERI lama,
	// yang BERBEDA dari yang ditampilkan. Dikirim supaya layar dapat menjelaskan
	// selisihnya kepada penguji yang membandingkannya dengan rule Pega.
	CoordinatorInQuery string

	// BusinessLines adalah isi dropdown lini bisnis pada tab KPI PIC Teknik.
	BusinessLines []reportkpi.BusinessLineOption

	// PICComponents adalah keempat komponen penilaian PIC Teknik, dalam urutan barisnya.
	PICComponents []reportkpi.PICComponent

	// PICTeknikPlannedDifferences adalah selisih terencana tab KPI PIC Teknik.
	PICTeknikPlannedDifferences []string

	// SLAExcludedPICs adalah petugas yang dikecualikan dari penilaian SLA di sistem lama.
	//
	// Dikirim ke layar dan DITAMPILKAN. Pengecualian yang tidak terlihat adalah
	// pengecualian yang tidak dapat dipertanyakan — dan ini pengecualian berbasis nama
	// orang di dalam kode (`D-15`).
	SLAExcludedPICs []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: susunan tab,
// grid, dan komponen sama di seluruh entitas karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:                    reportkpi.Tabs(),
		DefaultTab:              reportkpi.DefaultTabKPI,
		Components:              reportkpi.Components(),
		ReportTypes:             reportkpi.ReportTypes(),
		PlannedDifferences:      reportkpi.PlannedDifferences,
		AdminGroups:             reportkpi.AdminGroups(),
		AdminPlannedDifferences: reportkpi.AdminPlannedDifferences,
		CoordinatorInQuery:      reportkpi.CoordinatorInQuery,

		BusinessLines:               reportkpi.BusinessLines(),
		PICComponents:               reportkpi.PICComponents(),
		PICTeknikPlannedDifferences: reportkpi.PICTeknikPlannedDifferences,
		SLAExcludedPICs:             reportkpi.SLAExcludedPICs(),
	}
}

// Summarized adalah isi grid Summary beserta permintaan yang benar-benar dipakai.
type Summarized struct {
	Rows []reportkpi.AdjusterSummary

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar keterangan penyaring dari sini, bukan dari isian yang ia kirim:
	// tipe report yang dikirim huruf kecil dibakukan menjadi huruf besar, dan layar harus
	// menyatakan penyaring yang sebenarnya dipakai.
	Query reportkpi.Query
}

// Summary mengambil grid Summary KPI Adjuster.
func (s *Service) Summary(
	ctx context.Context,
	portalAlias string,
	caller reportkpi.Caller,
	input reportkpi.QueryInput,
) (Summarized, error) {
	query, err := reportkpi.NewQuery(input, caller)
	if err != nil {
		return Summarized{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Summarized{}, err
	}

	rows, err := repo.Summary(ctx, query)
	if err != nil {
		return Summarized{}, fmt.Errorf("mengambil ringkasan KPI adjuster: %w", err)
	}

	s.record(ctx, "ringkasan KPI adjuster dibuka", portalAlias, query, len(rows))
	return Summarized{Rows: rows, Query: query}, nil
}

// Detailed adalah satu halaman grid Detail beserta permintaan yang dipakai.
type Detailed struct {
	Page  reportkpi.DetailPage
	Query reportkpi.Query
}

// Detail mengambil satu halaman grid Detail KPI Adjuster.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller reportkpi.Caller,
	input reportkpi.QueryInput,
	page reportkpi.Pagination,
) (Detailed, error) {
	query, err := reportkpi.NewQuery(input, caller)
	if err != nil {
		return Detailed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Detailed{}, err
	}

	result, err := repo.Detail(ctx, query, page)
	if err != nil {
		return Detailed{}, fmt.Errorf("mengambil rincian KPI adjuster: %w", err)
	}

	s.record(ctx, "rincian KPI adjuster dibuka", portalAlias, query, len(result.Rows))
	return Detailed{Page: result, Query: query}, nil
}

// Adjusters mengambil isi dropdown Adjuster.
//
// # Kenapa ia menerima permintaan yang LENGKAP, bukan hanya alias portal
//
// Karena daftarnya diambil dari tabel penilaian itu sendiri, sehingga ia ikut menyempit
// mengikuti tipe report dan periode yang sedang dipilih. Itu yang menjaga janji pada
// PlannedDifferences: setiap pilihan yang muncul pasti menghasilkan baris.
//
// Isian `Adjuster` pada permintaan diabaikan di sini — menyaring daftar pilihan dengan
// pilihan yang sedang aktif akan menyisakan satu pilihan saja, dan pengguna tidak dapat
// berpindah adjuster lagi.
func (s *Service) Adjusters(
	ctx context.Context,
	portalAlias string,
	caller reportkpi.Caller,
	input reportkpi.QueryInput,
) ([]string, error) {
	input.Adjuster = ""

	query, err := reportkpi.NewQuery(input, caller)
	if err != nil {
		return nil, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	names, err := repo.Adjusters(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar adjuster: %w", err)
	}
	return names, nil
}

// record menuliskan jejak satu pembukaan laporan.
//
// # Kenapa SETIAP pembukaan dicatat, bukan hanya yang mencurigakan
//
// Karena barisnya adalah PENILAIAN KINERJA orang yang dapat dinamai, dan laporannya tidak
// disaring per pengguna — siapa pun yang dapat masuk melihat seluruh adjuster pada
// portalnya. Selama pemeriksaan peran belum ada (`TKT-F3-004`), jejak inilah satu-satunya
// hal yang menyatakan siapa yang membukanya, dan `D-59` menjadikan jejak audit
// satu-satunya kontrol pengimbang justru untuk keadaan seperti ini.
//
// Penyaringnya ikut dicatat — termasuk periode dan adjuster yang dipilih — karena
// "membuka laporan" dan "membuka penilaian satu orang tertentu pada satu periode" adalah
// dua hal yang berbeda bagi siapa pun yang kelak menelusurinya.
func (s *Service) record(
	ctx context.Context,
	message string,
	portalAlias string,
	query reportkpi.Query,
	rows int,
) {
	if s.logger == nil {
		return
	}

	adjuster := query.Adjuster
	if adjuster == "" {
		adjuster = "(seluruhnya)"
	}

	s.logger.InfoContext(ctx, message,
		slog.String("modul", "report-kpi"),
		slog.String("tab", reportkpi.TabAdjuster),
		slog.String("tipe_report", string(query.ReportType)),
		slog.String("adjuster", adjuster),
		slog.String("periode_dari", query.Range.From),
		slog.String("periode_sampai", query.Range.To),
		slog.String("pemanggil", query.Caller.Login),
		slog.String("portal", portalAlias),
		slog.Int("baris", rows),
	)
}
