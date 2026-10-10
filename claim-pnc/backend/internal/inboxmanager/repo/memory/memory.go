// Package memory memenuhi seam inboxmanager.Repo tanpa basis data.
//
// Ia dipakai `PENYIMPANAN=memori` dan oleh uji. Isinya BUKAN data nyata dan tidak pernah
// menjadi acuan perilaku — yang menjadi acuan adalah berkas .sql di repo/sqlstore.
//
// # Satu pelajaran yang dibawa dari modul lain
//
// Data contoh di sini TIDAK menyimpan nilai yang seharusnya diturunkan. Fixture yang
// menyimpan hasil jadi akan menyimpang dari SQL tanpa satu pun uji gagal, karena isinya
// dikarang dari kode yang sedang diuji — persis kekeliruan yang tercatat pada modul Inbox
// Manager Admin (`keputusan-implementasi.md` §64.4).
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxmanager"
)

// Store adalah penyimpanan di memori untuk satu portal.
type Store struct {
	mu sync.RWMutex

	// queues memetakan kode tab ke barisnya.
	queues map[string][]inboxmanager.QueueRow

	// panels memetakan kode tab dashboard ke barisnya per panel.
	panels map[string]map[string][]inboxmanager.DashboardRow

	// lineBusiness memetakan login petugas ke lini bisnisnya.
	lineBusiness map[string]string

	// defaultLine dipakai bagi petugas yang tidak punya entri.
	//
	// # Kenapa ia ada, dan kenapa hanya pada data contoh
	//
	// Tanpa nilai bawaan, mode memori mengembalikan lini bisnis kosong untuk setiap login.
	// Di modul ini kosong berarti "seluruh lini" sehingga layarnya tetap terisi — tetapi
	// cabang penyaring lini bisnis tidak akan pernah terlewati satu kali pun saat
	// pengembangan, dan cacat di sana baru muncul di Oracle.
	//
	// NewStore dan NewStrictStore sengaja TIDAK mengisinya, supaya uji tetap ketat.
	defaultLine string

	// unavailable memetakan kode tab ke alasan sumbernya tidak dapat dibaca.
	//
	// Ia ada supaya keadaan "view rusak" — yang nyata di basis data sejak 2026-09-28 —
	// dapat dikembangkan dan diuji tanpa merusak apa pun.
	unavailable map[string]string
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{
		queues:       map[string][]inboxmanager.QueueRow{},
		panels:       map[string]map[string][]inboxmanager.DashboardRow{},
		lineBusiness: map[string]string{},
		unavailable:  map[string]string{},
	}
}

// NewStrictStore membentuk penyimpanan kosong tanpa satu pun nilai bawaan.
//
// Ia sama dengan NewStore hari ini, dan dipisah supaya uji yang menuntut ketiadaan nilai
// bawaan tidak ikut berubah bila NewStore kelak diberi kemudahan.
func NewStrictStore() *Store {
	return NewStore()
}

// SetLineBusiness menetapkan lini bisnis seorang petugas.
func (s *Store) SetLineBusiness(login, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lineBusiness[normalizeLogin(login)] = strings.TrimSpace(line)
}

// SetDefaultLineBusiness menetapkan lini bisnis bagi petugas tanpa entri.
func (s *Store) SetDefaultLineBusiness(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultLine = strings.TrimSpace(line)
}

// SetQueue menetapkan isi sebuah antrean.
func (s *Store) SetQueue(tabCode string, rows []inboxmanager.QueueRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queues[tabCode] = rows
}

// SetPanel menetapkan isi sebuah panel dashboard.
func (s *Store) SetPanel(tabCode, panelKey string, rows []inboxmanager.DashboardRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.panels[tabCode] == nil {
		s.panels[tabCode] = map[string][]inboxmanager.DashboardRow{}
	}
	s.panels[tabCode][panelKey] = rows
}

// SetUnavailable menandai sebuah antrean sebagai tidak dapat dibaca.
func (s *Store) SetUnavailable(tabCode, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unavailable[tabCode] = reason
}

// Counters menghitung pencacah dari isi penyimpanan.
//
// Pencacah antrean DITURUNKAN dari jumlah barisnya, tidak disimpan terpisah. Menyimpannya
// terpisah akan membuat pencacah dan daftar dapat menyimpang — dan menyimpang tanpa satu pun
// uji gagal, karena keduanya diisi tangan oleh data contoh yang sama.
func (s *Store) Counters(
	_ context.Context,
	_ inboxmanager.Caller,
) ([]inboxmanager.Counter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := []inboxmanager.Counter{}
	for _, tab := range inboxmanager.Tabs() {
		if tab.Kind != inboxmanager.KindQueue && tab.Code != inboxmanager.TabOutstanding {
			continue
		}

		counter := inboxmanager.Counter{TabCode: tab.Code, Label: tab.Name}
		if tab.Kind == inboxmanager.KindQueue {
			counter.Parent = inboxmanager.TabApprovalMaster
		}

		if reason, blocked := s.unavailable[tab.Code]; blocked {
			counter.Unavailable = reason
		} else if tab.Code == inboxmanager.TabOutstanding {
			counter.Count = len(s.panels[tab.Code]["pic"])
		} else {
			counter.Count = len(s.queues[tab.Code])
		}

		result = append(result, counter)
	}

	return inboxmanager.SortCounters(result), nil
}

// Dashboard mengembalikan panel sebuah tab dashboard.
func (s *Store) Dashboard(
	_ context.Context,
	q inboxmanager.Query,
) (inboxmanager.DashboardView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	panels := make([]inboxmanager.Panel, len(q.Tab.Panels))
	copy(panels, q.Tab.Panels)

	for i := range panels {
		panels[i].Rows = s.panels[q.Tab.Code][panels[i].Key]
		if panels[i].Rows == nil {
			panels[i].Rows = []inboxmanager.DashboardRow{}
		}

		// Kolom tahun grid silang disimpulkan dari kunci sel yang BUKAN kolom tetap.
		// Dengan begitu penyimpanan memori menggambar bentuk yang sama dengan SQL tanpa
		// perlu menyimpan daftar tahun tersendiri.
		panels[i].Columns = append(
			append([]inboxmanager.Column{}, panels[i].Columns...),
			extraColumns(panels[i])...,
		)
	}

	view := inboxmanager.DashboardView{Panels: panels}
	if q.Tab.Code == inboxmanager.TabOutstanding {
		view.Filters = sampleFilters()
	}
	return view, nil
}

// extraColumns menyebut kolom yang hanya diketahui dari isinya, terurut.
func extraColumns(panel inboxmanager.Panel) []inboxmanager.Column {
	declared := map[string]bool{}
	for _, column := range panel.Columns {
		declared[column.Key] = true
	}

	found := map[string]bool{}
	for _, row := range panel.Rows {
		for key := range row.Cells {
			if !declared[key] {
				found[key] = true
			}
		}
	}

	keys := make([]string, 0, len(found))
	for key := range found {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]inboxmanager.Column, 0, len(keys))
	for _, key := range keys {
		result = append(result, inboxmanager.Column{Key: key, Title: key})
	}
	return result
}

// sampleFilters meniru kedua penyaring dashboard Outstanding.
func sampleFilters() []inboxmanager.FilterView {
	return []inboxmanager.FilterView{
		{
			Key:   inboxmanager.FilterReinsurer,
			Label: "Reinsurer",
			Options: []inboxmanager.FilterOption{
				{Label: "All"},
				{Value: "LEADER", Label: "Leader"},
				{Value: "MEMBER", Label: "Member"},
				{Value: "FAC-IN", Label: "Fac In"},
			},
		},
		{
			Key:   inboxmanager.FilterCategoryOS,
			Label: "Kategori OS",
			Options: []inboxmanager.FilterOption{
				{Label: "All"},
				{Value: "REGISTRASI", Label: "REGISTRASI"},
				{Value: "SURVEY", Label: "SURVEY"},
				{Value: "ACCEPTATION", Label: "ACCEPTATION"},
				{Value: "CLAIM COMMITTEE", Label: "CLAIM COMMITTEE"},
			},
		},
	}
}

// Queue mengembalikan seluruh baris sebuah antrean.
func (s *Store) Queue(
	_ context.Context,
	q inboxmanager.Query,
) ([]inboxmanager.QueueRow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if reason, blocked := s.unavailable[q.Tab.Code]; blocked && reason != "" {
		return nil, inboxmanager.ErrSourceUnavailable
	}

	rows := s.queues[q.Tab.Code]
	result := make([]inboxmanager.QueueRow, len(rows))
	copy(result, rows)

	sort.SliceStable(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

// Decide membuang baris yang diputuskan dari antreannya.
//
// # Kenapa barisnya DIBUANG, bukan ditandai
//
// Karena kesembilan kueri antrean hanya memuat baris berstatus menunggu, sehingga baris yang
// sudah diputuskan memang tidak muncul lagi. Menandainya di sini akan membuat penyimpanan
// memori berperilaku berbeda dari SQL pada hal yang paling sering diuji.
//
// Kunci yang TIDAK ditemukan tidak dihitung sebagai berubah — persis seperti penyaring status
// menunggu pada pernyataan SQL-nya, dan itulah yang membuat DecisionResult.Stale dapat diuji
// tanpa basis data.
func (s *Store) Decide(_ context.Context, d inboxmanager.Decision) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wanted := map[string]struct{}{}
	for _, key := range d.Keys {
		wanted[key] = struct{}{}
	}

	rows := s.queues[d.Tab.Code]
	kept := make([]inboxmanager.QueueRow, 0, len(rows))
	changed := 0

	for _, row := range rows {
		if _, hit := wanted[row.Key]; hit {
			changed++
			continue
		}
		kept = append(kept, row)
	}

	s.queues[d.Tab.Code] = kept
	return changed, nil
}

// LineBusinessFor membaca lini bisnis seorang petugas.
func (s *Store) LineBusinessFor(_ context.Context, loginID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if line, found := s.lineBusiness[normalizeLogin(loginID)]; found {
		return line, nil
	}
	return s.defaultLine, nil
}

// normalizeLogin menyamakan bentuk login supaya pencarian tidak bergantung huruf besar-kecil.
//
// Bentuknya sama dengan yang dipakai kueri `line_business_for`: `UPPER(TRIM(...))`.
func normalizeLogin(login string) string {
	return strings.ToUpper(strings.TrimSpace(login))
}
