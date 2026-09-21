// Package memory adalah pengisi seam penyimpanan Pelaporan Klaim yang hidup di dalam
// memori.
//
// Ia ada supaya seluruh alur — mencatat, mengubah, mentransfer, menautkan klaim — dapat
// diuji tanpa basis data dan tanpa jaringan sama sekali. Adapter kedua inilah yang
// membuat seam penyimpanan menjadi seam NYATA, bukan seam hipotetis
// (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: laporan kerugian yang hilang saat proses
// dijalankan ulang adalah kewajiban ke tertanggung yang hilang bersamanya.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/pelaporanklaim"
)

// Repo menyimpan laporan klaim di memori, dikunci nomor laporan.
type Repo struct {
	mu sync.RWMutex

	items map[string]pelaporanklaim.ClaimReport

	// order menyimpan nomor laporan sesuai urutan penyisipan.
	//
	// Map Go tidak punya urutan, dan mengurutkan hanya dengan waktu pencatatan tidak
	// cukup: jam tetap pada pengujian memberi waktu yang SAMA PERSIS untuk beberapa
	// laporan, sehingga dua baris dapat bertukar tempat di antara dua permintaan dan
	// membuat paginasi melewatkan baris.
	order []string

	// nextSequence adalah nomor urut yang akan dipakai laporan berikutnya.
	nextSequence int
}

// NewRepo membentuk penyimpanan berisi laporan awal yang diberikan.
//
// Laporan awal dipakai apa adanya beserta nomornya; pencacah nomor digeser melewati
// nomor tertinggi yang sudah ada supaya laporan berikutnya tidak bentrok dengannya.
func NewRepo(initial ...pelaporanklaim.ClaimReport) *Repo {
	r := &Repo{
		items:        map[string]pelaporanklaim.ClaimReport{},
		nextSequence: 1,
	}
	for _, report := range initial {
		report = report.Clean()
		if report.Number == "" {
			continue
		}
		if _, exists := r.items[report.Number]; exists {
			continue
		}
		r.items[report.Number] = report
		r.order = append(r.order, report.Number)
		if n := sequenceFromNumber(report.Number); n >= r.nextSequence {
			r.nextSequence = n + 1
		}
	}
	return r
}

// List membaca satu halaman laporan yang cocok dengan filter.
func (r *Repo) List(_ context.Context, f pelaporanklaim.Filter) (pelaporanklaim.Page, error) {
	f = f.Normalize()

	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := r.matchLocked(f, true)
	total := len(matched)

	return pelaporanklaim.Page{
		Reports: slice(matched, f),
		Total:   total,
	}, nil
}

// Summary menghitung jumlah laporan per tahap.
//
// Penyaring Stage SENGAJA diabaikan di sini: angkanya dipakai seluruh tab sekaligus, dan
// menghormatinya akan membuat setiap tab melaporkan jumlah dirinya sendiri sebagai satu-
// satunya yang berisi.
func (r *Repo) Summary(_ context.Context, f pelaporanklaim.Filter) (pelaporanklaim.StageSummary, error) {
	f = f.Normalize()

	r.mu.RLock()
	defer r.mu.RUnlock()

	summary := pelaporanklaim.StageSummary{
		pelaporanklaim.StageNotTransferred: 0,
		pelaporanklaim.StageNotRegistered:  0,
		pelaporanklaim.StageRegistered:     0,
		pelaporanklaim.StageAccepted:       0,
		pelaporanklaim.StageRejected:       0,
	}
	for _, report := range r.matchLocked(f, false) {
		summary[report.Stage()]++
	}
	return summary, nil
}

// Get mengembalikan satu laporan.
func (r *Repo) Get(_ context.Context, number string) (pelaporanklaim.ClaimReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report, exists := r.items[strings.TrimSpace(number)]
	if !exists {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrNotFound
	}
	return report, nil
}

// Insert menyimpan laporan baru dan mengembalikannya lengkap dengan nomor yang dibuat di
// sini.
func (r *Repo) Insert(_ context.Context, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	report = report.Clean()
	report.Number = fmt.Sprintf("%s%04d", numberPrefix, r.nextSequence)
	if _, exists := r.items[report.Number]; exists {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrNumberTaken
	}
	r.nextSequence++

	r.items[report.Number] = report
	r.order = append(r.order, report.Number)
	return report, nil
}

// Update mengubah laporan yang sudah ada.
func (r *Repo) Update(_ context.Context, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	report = report.Clean()
	previous, exists := r.items[report.Number]
	if !exists {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrNotFound
	}

	// Nomor, pencatat, dan waktu pencatatan tidak pernah ikut berubah — apa pun yang
	// dikirim pemanggil. Penegakannya ada di sini DAN di usecase; yang di sini menjaga
	// adapter tetap benar bila kelak dipanggil dari tempat lain.
	report.Number = previous.Number
	report.CreatedBy = previous.CreatedBy
	report.CreatedAt = previous.CreatedAt

	r.items[report.Number] = report
	return report, nil
}

// matchLocked mengembalikan laporan yang lolos filter, terurut.
//
// honorStage dimatikan saat menghitung ringkasan — lihat Summary.
func (r *Repo) matchLocked(f pelaporanklaim.Filter, honorStage bool) []pelaporanklaim.ClaimReport {
	matched := make([]pelaporanklaim.ClaimReport, 0, len(r.items))
	for _, number := range r.order {
		report, exists := r.items[number]
		if !exists {
			continue
		}
		if honorStage && f.Stage != "" && report.Stage() != f.Stage {
			continue
		}
		if f.BranchCode != "" && !equalFold(report.BranchCode, f.BranchCode) {
			continue
		}
		if !matchesSearch(report, f.Search) {
			continue
		}
		matched = append(matched, report)
	}

	// Urutan tetap: laporan terbaru lebih dulu, nomor sebagai pemutus seri. Ini sama
	// dengan `ORDER BY DateForAging_1 DESC` pada kueri inbox lama
	// (`RDB List/ViewTableBrowseRCVInProcess-SQL.xml`), ditambah pemutus seri yang di
	// sana tidak ada.
	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		}
		return matched[i].Number > matched[j].Number
	})
	return matched
}

// matchesSearch mencocokkan kata kunci ke lima field yang dipakai orang mencari.
//
// Daftarnya SAMA PERSIS dengan yang dipakai adapter SQL. Bila keduanya berbeda, pengujian
// akan lulus terhadap memori lalu gagal diam-diam terhadap Oracle.
func matchesSearch(report pelaporanklaim.ClaimReport, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return true
	}
	fields := []string{
		report.Number,
		report.ClaimNumber,
		report.PolicyNumber,
		report.InsuredName,
		report.ReporterName,
	}
	for _, value := range fields {
		if strings.Contains(strings.ToLower(value), search) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func slice(rows []pelaporanklaim.ClaimReport, f pelaporanklaim.Filter) []pelaporanklaim.ClaimReport {
	start := f.Offset
	if start >= len(rows) {
		return []pelaporanklaim.ClaimReport{}
	}
	rest := rows[start:]
	if f.Limit > 0 && f.Limit < len(rest) {
		rest = rest[:f.Limit]
	}
	return append([]pelaporanklaim.ClaimReport(nil), rest...)
}

// Memastikan adapter ini benar-benar memenuhi seam. Tanpa pernyataan ini, ketidakcocokan
// baru terlihat di berkas perakitan — jauh dari tempat sebabnya.
var _ pelaporanklaim.Repo = (*Repo)(nil)

// numberPrefix dipakai adapter memori supaya nomornya terbaca berbeda dari nomor yang
// terbit di Oracle — laporan yang dibuat saat mencoba aplikasi tidak boleh tertukar
// dengan laporan sungguhan bila keduanya kebetulan bertemu di satu layar.
const numberPrefix = "LPK.00."

// sequenceFromNumber membaca nomor urut dari sebuah nomor laporan, atau 0 bila bentuknya
// tidak dikenali.
func sequenceFromNumber(number string) int {
	parts := strings.Split(number, ".")
	last := parts[len(parts)-1]

	n := 0
	for _, c := range last {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
