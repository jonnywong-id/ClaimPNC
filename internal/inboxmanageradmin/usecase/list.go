// Package usecase mengorkestrasi modul Inbox Manager Admin.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab YANG BOLEH DILIHAT pemanggil, kolomnya, dan selisih
//	          terencana
//	List      mengambil isi satu tab
//
// Tidak ada operasi yang menulis. Layar ini membaca antrean; yang mengubahnya adalah layar
// registrasi dan estimasi — masing-masing modulnya sendiri. Lihat catatan tentang
// `SetAssignmentInboxReg_act` di kepala paket inboxmanageradmin.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxmanageradmin"
)

// Service melayani modul Inbox Manager Admin.
type Service struct {
	repoSelector inboxmanageradmin.RepoSelector
	lineSelector inboxmanageradmin.LineBusinessRepoSelector
	clock        inboxmanageradmin.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxmanageradmin.RepoSelector

	// LineBusinessSelector membaca lini bisnis pemanggil, yang menentukan tab mana yang
	// boleh ia buka.
	//
	// Ia WAJIB diisi. Membiarkannya kosong dan memperlakukan ketiadaannya sebagai "tanpa
	// lini bisnis" akan menghasilkan layar yang tidak pernah menampilkan satu tab pun
	// kepada siapa pun — persis cacat yang diperbaiki pada 2026-09-27, dan cacat itu
	// berjalan berhari-hari tanpa satu pun galat.
	LineBusinessSelector inboxmanageradmin.LineBusinessRepoSelector

	Clock inboxmanageradmin.Clock

	// Logger dipakai memperingatkan hasil yang sangat besar dan mencatat pembukaan layar.
	// Ia boleh nil; bila nil, catatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Manager Admin.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxmanageradmin/usecase: RepoSelector wajib diisi")
	}
	if o.LineBusinessSelector == nil {
		return nil, errors.New("inboxmanageradmin/usecase: LineBusinessSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxmanageradmin/usecase: Clock wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		lineSelector: o.LineBusinessSelector,
		clock:        o.Clock,
		logger:       o.Logger,
	}, nil
}

// resolveCaller melengkapi identitas pemanggil dengan lini bisnisnya.
//
// # Kenapa ia dibaca di sini, bukan dibawa dari sesi
//
// Karena `M_LOGIN_PNC` adalah tabel PER ENTITAS: petugas yang sama dapat punya lini bisnis
// berbeda di badan hukum yang berbeda. Membacanya sekali saat login lalu memakainya di
// seluruh portal berarti membuka tab atas dasar kewenangan entitas lain (`ADR-0030`,
// `R-20`).
//
// Sampai 2026-09-27 nilainya justru datang dari sesi — dari jabatan kepegawaian HCQ — dan
// itulah cacat yang diperbaiki.
func (s *Service) resolveCaller(
	ctx context.Context,
	portalAlias string,
	caller inboxmanageradmin.Caller,
) (inboxmanageradmin.Caller, error) {
	clean := caller.Clean()

	lines, err := s.lineSelector(portalAlias)
	if err != nil {
		return inboxmanageradmin.Caller{}, err
	}

	line, err := lines.LineBusinessFor(ctx, clean.Login)
	if err != nil {
		return inboxmanageradmin.Caller{}, fmt.Errorf(
			"membaca lini bisnis pemanggil %s: %w", clean.Login, err)
	}

	return clean.WithLineBusiness(line), nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah tab yang BOLEH dilihat pemanggil — bukan ketiganya.
	//
	// Penyaringannya terjadi di sini, bukan di layar, karena penyembunyian di layar
	// hanyalah kenyamanan tampilan (`11-SECURITY.md` §3.1). Server yang memutuskan, dan
	// NewQuery menolak permintaan atas tab yang tidak ada di daftar ini.
	Tabs []inboxmanageradmin.Tab

	// DefaultTab adalah tab pertama yang boleh dilihat pemanggil.
	//
	// Kosong bila tidak ada satu pun — keadaan yang mungkin terjadi, dan yang layar
	// jelaskan alih-alih membuka tab yang pasti ditolak.
	DefaultTab string

	// AllTabs adalah KETIGA tab tanpa memandang kewenangan.
	//
	// Ia dikirim supaya layar dapat menyebutkan apa saja yang ada ketika pengguna tidak
	// berhak atas satu pun — "Anda tidak melihat tab apa pun" jauh kurang berguna
	// daripada menyebut ketiganya beserta jabatan yang membukanya.
	AllTabs []inboxmanageradmin.Tab

	// ExpectedLineBusinesses adalah lini bisnis yang membuka tab, untuk ditampilkan pada
	// keadaan di atas.
	ExpectedLineBusinesses []string

	// CallerLineBusiness adalah lini bisnis pemanggil apa adanya, dibaca dari
	// `M_LOGIN_PNC.LINE_BUSINESS`.
	//
	// Ia dipantulkan kembali ke layar dengan sengaja: pengguna yang tidak melihat satu tab
	// pun perlu tahu nilai APA yang terbaca sistem, karena itulah satu-satunya petunjuk
	// yang dapat ia sampaikan saat melapor — dan yang membedakan "kolomnya belum diisi"
	// dari "diisi dengan nilai yang tidak dikenal".
	CallerLineBusiness string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar bagi seorang pemanggil.
//
// Daftar tab dan kolomnya sama di seluruh entitas — ia bentuk layar, bukan data entitas.
// Yang bergantung pemanggil hanyalah tab mana yang boleh ia lihat, dan sejak 2026-09-27
// itu menuntut satu pembacaan basis data: lini bisnis petugas ada di `M_LOGIN_PNC` milik
// portal yang sedang aktif, bukan di sesinya.
//
// Ia karena itu tidak lagi bebas galat, dan tidak lagi bebas portal. Itu harga dari membaca
// kewenangan di tempat yang benar.
func (s *Service) Metadata(
	ctx context.Context,
	portalAlias string,
	caller inboxmanageradmin.Caller,
) (Metadata, error) {
	clean, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return Metadata{}, err
	}

	return Metadata{
		Tabs:                   inboxmanageradmin.VisibleTabs(clean),
		DefaultTab:             inboxmanageradmin.DefaultTabFor(clean),
		AllTabs:                inboxmanageradmin.Tabs(),
		ExpectedLineBusinesses: inboxmanageradmin.ExpectedLineBusinesses(),
		CallerLineBusiness:     clean.LineBusiness,
		PlannedDifferences:     inboxmanageradmin.PlannedDifferences,
	}, nil
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxmanageradmin.Page

	// Query adalah permintaan setelah divalidasi dan diperiksa kewenangannya.
	//
	// Layar menggambar keadaan tabnya dari sini, bukan dari isian yang ia kirim: permintaan
	// tanpa kode tab dijawab dengan tab pertama yang boleh dilihat pemanggil, dan layar
	// perlu tahu yang mana.
	Query inboxmanageradmin.Query
}

// List mengambil isi satu tab.
//
// # Kenapa seluruh baris ditarik lebih dulu, lalu dipotong di sini
//
// Keputusan Work Owner 2026-09-26: paginasi layar ini direplikasi apa adanya. Penjelasan
// lengkapnya ada di inboxmanageradmin.Slice, termasuk konsekuensi yang diterima secara
// sadar.
//
// Yang ditambahkan di sini hanyalah PERINGATAN — bukan pemotongan. Memotong hasil akan
// menyalahi keputusan itu; memperingatkan tidak mengubah apa pun dan membuat akibatnya
// terlihat operator sebelum terlihat sebagai aplikasi yang kehabisan memori.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxmanageradmin.Caller,
	input inboxmanageradmin.QueryInput,
	page inboxmanageradmin.Pagination,
) (Listed, error) {
	// Lini bisnis dibaca LEBIH DULU, sebelum kewenangan tab diperiksa. Urutan itu bukan
	// kenyamanan: NewQuery menolak tab yang tidak boleh dibuka pemanggil, dan tanpa lini
	// bisnis yang terisi ia akan menolak SELURUH tab bagi setiap orang.
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return Listed{}, err
	}

	query, err := inboxmanageradmin.NewQuery(input, resolved)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	rows, err := repo.List(ctx, query)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Name, err)
	}

	// Pembukaan layar DICATAT, dan itu bukan kelengkapan operasional belaka.
	//
	// `D-59` menetapkan satuan izin adalah menu tanpa pemisahan tugas, sehingga jejak
	// audit menjadi satu-satunya kontrol pengimbang yang tersisa (`11-SECURITY.md` §4.4).
	// Di layar ini catatan itu lebih penting daripada biasa: barisnya memuat nama
	// tertanggung dan nomor polis MILIK PETUGAS LAIN, dan pemeriksaan peran yang
	// seharusnya menggerbangnya belum ada (`TKT-F3-004`).
	if s.logger != nil {
		s.logger.Info(
			"antrean Inbox Manager Admin dibuka",
			slog.String("modul", "inbox-manager-admin"),
			slog.String("portal", portalAlias),
			slog.String("pengguna", query.Caller.Login),
			slog.String("lini_bisnis", query.Caller.LineBusiness),
			slog.String("tab", query.Tab.Name),
			slog.String("unit_organisasi", query.Tab.OrgUnit),
			slog.Int("jumlah_baris", len(rows)),
		)
	}

	if s.logger != nil && len(rows) > inboxmanageradmin.LargeResultWarning {
		s.logger.Warn(
			"satu permintaan Inbox Manager Admin menarik sangat banyak baris",
			slog.String("tab", query.Tab.Name),
			slog.String("unit_organisasi", query.Tab.OrgUnit),
			slog.Int("jumlah_baris", len(rows)),
			slog.Int("ambang", inboxmanageradmin.LargeResultWarning),
			slog.String("sebab",
				"paginasi direplikasi apa adanya dari Pega — seluruh baris ditarik lalu dipotong di aplikasi"),
		)
	}

	now := s.clock.Now()
	for i := range rows {
		rows[i] = rows[i].WithElapsed(now)
	}

	return Listed{Page: inboxmanageradmin.Slice(rows, page), Query: query}, nil
}
