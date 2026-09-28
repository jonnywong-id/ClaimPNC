// Package usecase mengorkestrasi modul Inbox Manager.
//
// Empat operasi:
//
//	Metadata  daftar tab yang BOLEH dilihat pemanggil, kolomnya, dan selisih terencana
//	Counters  kesepuluh pencacah di kepala layar
//	List      isi satu tab — dashboard atau antrean
//	Decide    menuliskan keputusan atas sejumlah baris antrean
//
// Yang terakhir membedakan modul ini dari seluruh modul inbox lain: ia MENULIS. Lihat
// decide.go.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"claim-pnc/internal/inboxmanager"
)

// Service melayani modul Inbox Manager.
type Service struct {
	repoSelector inboxmanager.RepoSelector
	lineSelector inboxmanager.LineBusinessRepoSelector
	clock        inboxmanager.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxmanager.RepoSelector

	// LineBusinessSelector membaca lini bisnis pemanggil, yang menyaring ketiga dashboard.
	//
	// Ia WAJIB diisi. Di Pega penyaring itu dibaca dari `OperatorID.pyPosition`, dan
	// memakai jabatan kepegawaian HCQ sebagai gantinya adalah cacat yang sudah pernah
	// membuat modul lain kosong bagi setiap pengguna selama berhari-hari tanpa satu pun
	// galat (koreksi 2026-09-27).
	LineBusinessSelector inboxmanager.LineBusinessRepoSelector

	// Clock dipakai menentukan periode bawaan tab Produktivitas Klaim.
	//
	// Ia WAJIB diisi. Tab itu tidak dapat berjalan tanpa periode — kedelapan pencacahnya
	// DIBANGUN dari perbandingan dua periode — sehingga periodenya harus punya nilai bawaan,
	// dan nilai bawaan itu tidak boleh diambil dari jam sistem secara langsung agar dapat
	// diuji.
	Clock inboxmanager.Clock

	// Logger mencatat pembukaan layar, keputusan yang ditulis, dan hasil yang sangat besar.
	// Ia boleh nil; bila nil, catatannya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Manager.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxmanager/usecase: RepoSelector wajib diisi")
	}
	if o.LineBusinessSelector == nil {
		return nil, errors.New("inboxmanager/usecase: LineBusinessSelector wajib diisi")
	}
	if o.Clock == nil {
		return nil, errors.New("inboxmanager/usecase: Clock wajib diisi")
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
// seluruh portal berarti menyaring dashboard atas dasar kewenangan entitas lain (`ADR-0030`,
// `R-20`).
func (s *Service) resolveCaller(
	ctx context.Context,
	portalAlias string,
	caller inboxmanager.Caller,
) (inboxmanager.Caller, error) {
	clean := caller.Clean()

	lines, err := s.lineSelector(portalAlias)
	if err != nil {
		return inboxmanager.Caller{}, err
	}

	line, err := lines.LineBusinessFor(ctx, clean.Login)
	if err != nil {
		return inboxmanager.Caller{}, fmt.Errorf(
			"membaca lini bisnis pemanggil %s: %w", clean.Login, err)
	}

	return clean.WithLineBusiness(line), nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi tab.
type Metadata struct {
	// Tabs adalah tab yang BOLEH dilihat pemanggil.
	Tabs []inboxmanager.Tab

	// DefaultTab adalah tab pertama yang boleh dilihat pemanggil.
	DefaultTab string

	// LineBusiness adalah lini bisnis pemanggil yang benar-benar terbaca dari basis data.
	//
	// Ia dikirim ke layar supaya penyelia dapat melihat DASAR penyaringan angkanya. Tanpa
	// itu, dua penyelia yang membandingkan layar masing-masing akan melihat angka berbeda
	// tanpa satu pun petunjuk kenapa.
	LineBusiness string

	// PlannedDifferences adalah selisih terhadap sistem lama yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyusun keterangan layar bagi seorang pemanggil.
func (s *Service) Metadata(
	ctx context.Context,
	portalAlias string,
	caller inboxmanager.Caller,
) (Metadata, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return Metadata{}, err
	}
	if resolved.Login == "" {
		return Metadata{}, inboxmanager.ErrCallerUnknown
	}

	return Metadata{
		Tabs:               inboxmanager.VisibleTabs(resolved),
		DefaultTab:         inboxmanager.DefaultTabFor(resolved),
		LineBusiness:       resolved.LineBusiness,
		PlannedDifferences: inboxmanager.PlannedDifferences,
	}, nil
}

// Counters menyusun kesepuluh pencacah di kepala layar.
//
// # Kenapa ia endpoint TERSENDIRI
//
// Karena isinya tidak berubah saat tab berpindah. Menempelkannya ke setiap pemuatan tab
// berarti sepuluh kueri pencacah dijalankan ulang pada setiap klik tab — sepuluh perjalanan
// basis data untuk angka yang sama.
func (s *Service) Counters(
	ctx context.Context,
	portalAlias string,
	caller inboxmanager.Caller,
) ([]inboxmanager.Counter, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return nil, err
	}
	if resolved.Login == "" {
		return nil, inboxmanager.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	counters, err := repo.Counters(ctx, resolved)
	if err != nil {
		return nil, err
	}

	return append(counters, s.overviewCounter(counters)), nil
}

// overviewCounter menyusun pencacah tab "Approval Master" dari jumlah anak-anaknya.
//
// # Ia SELISIH terhadap Pega, dan selisih yang menguntungkan
//
// `Activity/CountDashbroardManager` tidak punya kueri pencacah untuk tab ini; yang ditulisnya
// angka tetap `"1"` — sebuah penanda, bukan hitungan. Menyalinnya berarti menggambar angka
// yang tidak berarti apa-apa di sebelah judul "Approval Master".
//
// Yang dipakai di sini adalah jumlah seluruh anaknya, yakni jumlah pekerjaan yang benar-benar
// menunggu. Pencacah yang sumbernya tidak dapat dibaca TIDAK ikut dijumlahkan, dan tab ini
// menyatakannya lewat Unavailable-nya sendiri supaya angka yang kurang tidak terbaca sebagai
// angka yang lengkap.
func (s *Service) overviewCounter(children []inboxmanager.Counter) inboxmanager.Counter {
	tab, _ := inboxmanager.FindTab(inboxmanager.TabApprovalMaster)

	counter := inboxmanager.Counter{TabCode: tab.Code, Label: tab.Name}
	missing := 0

	for _, child := range children {
		if child.Parent != inboxmanager.TabApprovalMaster {
			continue
		}
		if child.Unavailable != "" {
			missing++
			continue
		}
		counter.Count += child.Count
	}

	if missing > 0 {
		counter.Unavailable = fmt.Sprintf(
			"Angka ini BELUM lengkap: %d antrean sumbernya sedang tidak dapat dibaca "+
				"basis data. Buka antreannya untuk melihat sebabnya.", missing)
	}

	return counter
}

// View adalah isi satu tab.
//
// Salah satu dari Dashboard atau Queue terisi, tidak pernah keduanya — yang menentukan adalah
// Tab.Kind. Tab "Approval Master" tidak mengisi keduanya: isinya adalah pencacah, yang sudah
// dilayani endpoint tersendiri.
type View struct {
	// Tab adalah tab yang dibuka.
	Tab inboxmanager.Tab

	// Dashboard terisi pada tab dashboard.
	Dashboard inboxmanager.DashboardView

	// Queue terisi pada tab antrean.
	Queue inboxmanager.QueuePage

	// Period adalah periode yang BENAR-BENAR dipakai, termasuk bila ia nilai bawaan.
	//
	// Ia dikirim ke layar supaya penyelia melihat rentang yang sedang ia baca. Periode
	// bawaan yang tidak ditampilkan membuat angka pada layar tidak dapat dipertanggungkan:
	// dua orang akan membandingkan angka dari rentang yang berbeda tanpa menyadarinya.
	Period inboxmanager.Period
}

// List mengambil isi satu tab.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	input inboxmanager.QueryInput,
	caller inboxmanager.Caller,
) (View, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return View{}, err
	}

	query, err := inboxmanager.NewQuery(input, resolved)
	if err != nil {
		return View{}, err
	}

	query = s.withDefaultPeriod(query)

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return View{}, err
	}

	s.logOpen(query, portalAlias)

	view := View{Tab: query.Tab, Period: query.Period}

	switch query.Tab.Kind {
	case inboxmanager.KindDashboard:
		dashboard, err := repo.Dashboard(ctx, query)
		if err != nil {
			return View{}, err
		}
		view.Dashboard = dashboard

	case inboxmanager.KindQueue:
		rows, err := repo.Queue(ctx, query)
		if err != nil {
			return View{}, err
		}
		s.warnLargeResult(query, len(rows))
		view.Queue = inboxmanager.SliceQueue(rows, query.Page)
	}

	return view, nil
}

// withDefaultPeriod mengisi periode tab Produktivitas Klaim bila pemanggil tidak memilih.
//
// # Kenapa hanya tab itu
//
// Karena hanya tab itu yang TIDAK DAPAT berjalan tanpa periode: kedelapan pencacahnya
// dibangun dari perbandingan periode berjalan dengan periode yang sama tahun lalu, sehingga
// predikat kosong di sana menghasilkan SQL yang tidak sah. Di Pega pun layarnya selalu punya
// bulan terpilih.
//
// Dashboard Klaim sebaliknya memang boleh tanpa periode — pola `{ASIS}`-nya dapat berisi teks
// kosong, dan artinya seluruh periode. Nilai bawaan tidak dipaksakan di sana.
func (s *Service) withDefaultPeriod(q inboxmanager.Query) inboxmanager.Query {
	if q.Tab.Code != inboxmanager.TabProduktivitas || !q.Period.Empty() {
		return q
	}

	now := s.clock.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	until := from.AddDate(0, 1, 0)

	q.Period = inboxmanager.Period{
		From:       from,
		Until:      until,
		PriorFrom:  from.AddDate(-1, 0, 0),
		PriorUntil: until.AddDate(-1, 0, 0),
	}
	return q
}

// logOpen mencatat setiap pembukaan tab.
//
// # Kenapa SETIAP pembukaan, bukan hanya yang mencurigakan
//
// Karena layar ini pandangan penyelia atas pekerjaan orang lain, dan pemeriksaan peran belum
// ada (`TKT-F3-004`). `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru
// untuk keadaan seperti ini. Jejak yang hanya mencatat yang mencurigakan menuntut lebih dulu
// mengetahui apa yang mencurigakan.
func (s *Service) logOpen(q inboxmanager.Query, portalAlias string) {
	if s.logger == nil {
		return
	}
	s.logger.InfoContext(context.Background(), "inbox manager dibuka",
		slog.String("modul", "inbox-manager"),
		slog.String("portal", portalAlias),
		slog.String("login", q.Caller.Login),
		slog.String("lini_bisnis", q.Caller.LineBusiness),
		slog.String("tab", q.Tab.Code),
		slog.String("tab_nama", q.Tab.Name),
	)
}

// warnLargeResult memperingatkan hasil yang sangat besar.
//
// Ia TIDAK memotong apa pun — memotongnya akan menyalahi keputusan memaginasi di aplikasi.
// Yang dilakukannya hanya membuat akibat keputusan itu terlihat operator sebelum ia terlihat
// sebagai aplikasi yang kehabisan memori.
func (s *Service) warnLargeResult(q inboxmanager.Query, rows int) {
	if s.logger == nil || rows <= inboxmanager.LargeResultWarning {
		return
	}
	s.logger.WarnContext(context.Background(), "antrean inbox manager sangat besar",
		slog.String("modul", "inbox-manager"),
		slog.String("tab", q.Tab.Code),
		slog.Int("baris", rows),
		slog.Int("ambang", inboxmanager.LargeResultWarning),
	)
}
