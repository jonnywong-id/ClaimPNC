// Package memory memenuhi seam reportkpi.Repo tanpa basis data.
//
// Ia dipakai di development dan di seluruh uji, sehingga uji modul ini berjalan tanpa
// basis data dan tanpa jaringan.
//
// # Kenapa ia MENIRU aturan SQL, bukan menyederhanakannya
//
// Karena kalau tidak, ia berbohong. Tiga aturan ditiru dengan sengaja, dan ketiganya
// adalah tempat pengisi memori paling mudah menyimpang tanpa ketahuan:
//
//   - `ALL` menghasilkan DUA baris per adjuster, satu per tipe — bukan satu baris
//     gabungan. Itu yang membuat `UNION ALL` di sistem lama terasa sama di sini.
//   - Rentang tanggal SETENGAH TERBUKA: `>= dari` dan `< sampai + 1 hari`, persis
//     bentuk yang dipakai kuerinya.
//   - Nilai yang tidak terbaca sebagai angka DIABAIKAN dari rata-rata, tetapi barisnya
//     tetap ikut. Itu perilaku `AVG` terhadap NULL.
package memory

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Row adalah satu baris POOLDATA.DETAIL_KPI_ADJUSTER apa adanya.
//
// Kesembilan nilainya TEKS, sama dengan tipe kolomnya di Oracle
// (`INSERT_KPIADJUSTER.prc` mendeklarasikan seluruhnya `in varchar2`). Menyimpannya
// sebagai float di sini akan menyembunyikan justru keadaan yang paling perlu diuji: nilai
// yang tidak terbaca sebagai angka.
type Row struct {
	Adjuster   string
	CaseID     string
	ReportType reportkpi.ReportType
	ScoredOn   string // `YYYY-MM-DD`

	// Scores dikunci dengan kode komponen reportkpi. Komponen yang tidak ada di peta
	// berarti kolomnya NULL.
	Scores map[string]string
}

// Store adalah penyimpanan Report KPI di memori.
//
// Ia memegang DUA kumpulan baris karena kedua tab membaca sumber yang berbeda: tab
// Adjuster membaca `DETAIL_KPI_ADJUSTER`, tab Admin menghitung dari tabel klaim.
type Store struct {
	mu        sync.RWMutex
	rows      []Row
	adminRows []AdminRow
	picTeknik PICTeknikData
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store { return &Store{} }

// NewStoreWith membentuk penyimpanan berisi baris tab Adjuster.
func NewStoreWith(rows []Row) *Store {
	salinan := make([]Row, len(rows))
	copy(salinan, rows)
	return &Store{rows: salinan}
}

// WithAdminRows menambahkan baris tab KPI Admin dan mengembalikan penyimpanan yang sama.
//
// Ia memisahkan pengisian kedua tab supaya uji yang hanya menguji satu tab tidak perlu
// menyediakan contoh bagi tab yang lain.
func (s *Store) WithAdminRows(rows []AdminRow) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.adminRows = make([]AdminRow, len(rows))
	copy(s.adminRows, rows)
	return s
}

// Store memenuhi seam yang dideklarasikan domain.
var _ reportkpi.Repo = (*Store)(nil)

// Summary mengambil grid Summary: satu baris per pasangan adjuster × tipe.
func (s *Store) Summary(
	_ context.Context,
	q reportkpi.Query,
) ([]reportkpi.AdjusterSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Kuncinya PASANGAN adjuster dan tipe, bukan adjuster saja. Itulah `GROUP BY
	// k.ADJUSTER, k.TIPE` — dan itu pula yang membuat tipe ALL menghasilkan dua baris.
	type key struct {
		adjuster string
		tipe     reportkpi.ReportType
	}

	sums := map[key]map[string]float64{}
	counts := map[key]map[string]int{}
	order := []key{}

	for _, row := range s.rows {
		if !matches(row, q) {
			continue
		}

		k := key{adjuster: row.Adjuster, tipe: row.ReportType}
		if _, seen := sums[k]; !seen {
			sums[k] = map[string]float64{}
			counts[k] = map[string]int{}
			order = append(order, k)
		}

		for _, code := range reportkpi.ComponentCodes() {
			value, ok := numeric(row.Scores[code])
			if !ok {
				// Nilai yang tidak terbaca diabaikan dari rata-rata, persis seperti
				// `AVG` mengabaikan NULL. Barisnya tetap ikut untuk komponen lain.
				continue
			}
			sums[k][code] += value
			counts[k][code]++
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].adjuster != order[j].adjuster {
			return order[i].adjuster < order[j].adjuster
		}
		return order[i].tipe < order[j].tipe
	})

	result := make([]reportkpi.AdjusterSummary, 0, len(order))
	for _, k := range order {
		scores := map[string]reportkpi.Score{}
		for _, code := range reportkpi.ComponentCodes() {
			if counts[k][code] == 0 {
				scores[code] = reportkpi.EmptyScore()
				continue
			}
			average := sums[k][code] / float64(counts[k][code])
			scores[code] = reportkpi.NewScore(roundTo2(average))
		}

		result = append(result, reportkpi.AdjusterSummary{
			Adjuster:   k.adjuster,
			ReportType: k.tipe,
			Scores:     scores,
		})
	}
	return result, nil
}

// Detail mengambil satu halaman grid Detail.
func (s *Store) Detail(
	_ context.Context,
	q reportkpi.Query,
	page reportkpi.Pagination,
) (reportkpi.DetailPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := []Row{}
	for _, row := range s.rows {
		if matches(row, q) {
			matched = append(matched, row)
		}
	}

	// Urutannya WAJIB sama dengan `ORDER BY k.ADJUSTER, k.TANGGAL DESC, k.CASEID` —
	// paginasi di atas urutan yang berbeda menghasilkan halaman yang berbeda pula.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Adjuster != matched[j].Adjuster {
			return matched[i].Adjuster < matched[j].Adjuster
		}
		if matched[i].ScoredOn != matched[j].ScoredOn {
			return matched[i].ScoredOn > matched[j].ScoredOn
		}
		return matched[i].CaseID < matched[j].CaseID
	})

	clean := page.Normalize()
	total := len(matched)

	from := clean.Offset()
	if from > total {
		from = total
	}
	to := from + clean.Size
	if to > total {
		to = total
	}

	rows := make([]reportkpi.AdjusterDetail, 0, to-from)
	for _, row := range matched[from:to] {
		scores := map[string]reportkpi.Score{}
		for _, code := range reportkpi.ComponentCodes() {
			if value, ok := numeric(row.Scores[code]); ok {
				scores[code] = reportkpi.NewScore(value)
				continue
			}
			scores[code] = reportkpi.EmptyScore()
		}

		rows = append(rows, reportkpi.AdjusterDetail{
			Adjuster:   row.Adjuster,
			CaseID:     row.CaseID,
			ReportType: row.ReportType,
			ScoredOn:   row.ScoredOn,
			Scores:     scores,
		})
	}

	return reportkpi.DetailPage{Rows: rows, Total: total}, nil
}

// Adjusters mengambil nama adjuster yang punya baris pada penyaring yang berlaku.
func (s *Store) Adjusters(_ context.Context, q reportkpi.Query) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := map[string]bool{}
	for _, row := range s.rows {
		// Penyaring adjuster sengaja DIABAIKAN di sini — lihat usecase.Adjusters.
		probe := q
		probe.Adjuster = ""
		if !matches(row, probe) {
			continue
		}
		if name := strings.TrimSpace(row.Adjuster); name != "" {
			seen[name] = true
		}
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

// matches menyatakan satu baris lolos seluruh penyaring permintaan.
func matches(row Row, q reportkpi.Query) bool {
	if q.Adjuster != "" && row.Adjuster != q.Adjuster {
		return false
	}

	allowed := false
	for _, tipe := range q.ReportType.SplitsIntoTypes() {
		if row.ReportType == tipe {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}

	return withinRange(row.ScoredOn, q.Range)
}

// withinRange meniru penyaring tanggal SETENGAH TERBUKA pada kuerinya.
//
// Baris tanpa tanggal TIDAK PERNAH lolos, dan itu sama dengan perilaku SQL: pembandingan
// terhadap NULL menghasilkan UNKNOWN, bukan TRUE.
func withinRange(scoredOn string, rng reportkpi.DateRange) bool {
	moment, err := time.Parse("2006-01-02", strings.TrimSpace(scoredOn))
	if err != nil {
		return false
	}
	from, errFrom := time.Parse("2006-01-02", rng.From)
	to, errTo := time.Parse("2006-01-02", rng.To)
	if errFrom != nil || errTo != nil {
		return false
	}
	return !moment.Before(from) && moment.Before(to.AddDate(0, 0, 1))
}

// numeric membaca satu nilai komponen.
//
// Teks yang tidak terbaca sebagai angka dikembalikan sebagai "tidak ada", bukan sebagai
// nol — keduanya berbeda artinya bagi pembaca laporan kinerja.
func numeric(raw string) (float64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// roundTo2 membulatkan ke dua desimal, meniru `ROUND(..., 2)` di kueri.
func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}
