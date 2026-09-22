// Package memory adalah pengisi seam inboxlaporanklaim.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// # Yang ditiru bukan hanya bentuk datanya
//
// Urutan baris, cara memotong halaman, cara menurunkan Position, dan cara menerbitkan
// nomor ditiru pula. Kalau tidak, uji yang lulus di sini tidak membuktikan apa pun
// tentang adapter SQL — dan seam yang kedua adapternya berperilaku berbeda adalah seam
// yang menyembunyikan cacat, bukan yang menemukannya.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Repo menyimpan berkas laporan satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang ADR-0030 dan R-20 cegah.
type Repo struct {
	// mutex melindungi seluruh isi. Permintaan HTTP dilayani beberapa goroutine
	// sekaligus, dan penerbitan nomor harus berjalan satu per satu — persis seperti
	// pengambilan sequence pada adapter SQL.
	mutex sync.Mutex

	rows    []inboxlaporanklaim.ClaimReport
	region  []inboxlaporanklaim.Region
	branch  map[string]string // kode cabang -> kode kanwil
	message map[string][]messageRow

	sequence int64
	clock    inboxlaporanklaim.Clock
	failure  error
}

// messageRow adalah satu baris percakapan POOLDATA.M_KOMUNIKASI_PNC.
//
// Hanya tiga kolomnya yang dipakai layar ini: status, pengirim, dan isi pesan terakhir.
// Sisanya milik layar percakapan, bukan milik inbox.
type messageRow struct {
	Status string // KOMUNIKASISTATUS: "0" belum dijawab, "1" dijawab
	Sender string
	Text   string
	SentAt time.Time
}

// Options adalah bahan pembentuk Repo.
type Options struct {
	Rows   []inboxlaporanklaim.ClaimReport
	Region []inboxlaporanklaim.Region

	// Branch memetakan kode cabang ke kode kanwilnya — pengganti POOLDATA.BRANCH.
	Branch map[string]string

	// Message memetakan nomor register laporan ke percakapannya.
	Message map[string][]messageRow

	// Clock wajib: nomor berkas baru memuat tahun terbitnya.
	Clock inboxlaporanklaim.Clock
}

// NewRepo membentuk repo berisi bahan yang diberikan.
func NewRepo(o Options) *Repo {
	rows := make([]inboxlaporanklaim.ClaimReport, len(o.Rows))
	copy(rows, o.Rows)

	region := make([]inboxlaporanklaim.Region, len(o.Region))
	copy(region, o.Region)

	branch := map[string]string{}
	for code, area := range o.Branch {
		branch[code] = area
	}

	message := map[string][]messageRow{}
	for id, list := range o.Message {
		copied := make([]messageRow, len(list))
		copy(copied, list)
		message[id] = copied
	}

	return &Repo{
		rows:    rows,
		region:  region,
		branch:  branch,
		message: message,
		clock:   o.Clock,
	}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List menyaring, mengurutkan, lalu memotong satu halaman.
func (r *Repo) List(
	_ context.Context,
	filter inboxlaporanklaim.Filter,
	page inboxlaporanklaim.Pagination,
) (inboxlaporanklaim.Page, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return inboxlaporanklaim.Page{}, r.failure
	}

	matched := r.match(filter)
	clean := page.Clean()

	// Pengurutan meniru ORDER BY DateForAging_1 DESC yang dipakai KESEMBILAN kueri lama.
	// Nomor register dipakai sebagai pemutus seri supaya urutannya tetap sama pada dua
	// pemanggilan yang sama — tanpa itu, halaman kedua dapat memuat baris yang sudah
	// tampil di halaman pertama.
	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].AgingAt.Equal(matched[j].AgingAt) {
			return matched[i].AgingAt.After(matched[j].AgingAt)
		}
		return matched[i].ID > matched[j].ID
	})

	total := len(matched)
	from := clean.Offset()
	if from > total {
		from = total
	}
	to := from + clean.Size
	if to > total {
		to = total
	}

	window := make([]inboxlaporanklaim.ClaimReport, to-from)
	copy(window, matched[from:to])

	return inboxlaporanklaim.Page{
		Report:     window,
		Total:      total,
		Pagination: clean,
	}, nil
}

// Summarize menghitung kedelapan pencacah di atas daftar.
func (r *Repo) Summarize(
	_ context.Context,
	filter inboxlaporanklaim.Filter,
) (inboxlaporanklaim.Summary, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return inboxlaporanklaim.Summary{}, r.failure
	}

	var summary inboxlaporanklaim.Summary

	// Pencacah lama mengabaikan KATEGORI dan menghitung seluruh kategori sekaligus;
	// penyaring lainnya tetap berlaku. Kategori karena itu dikosongkan di sini.
	scope := filter
	scope.Category = inboxlaporanklaim.CategoryAll

	for _, row := range r.rows {
		if !r.matchScope(row, scope) {
			continue
		}
		// Berkas yang sudah selesai atau ditolak DIKECUALIKAN pencacah lama lewat
		// PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected').
		if r.resolved(row) {
			continue
		}

		summary.Total++
		switch row.Position {
		case inboxlaporanklaim.PositionNotTransferred:
			summary.NotTransferred++
		case inboxlaporanklaim.PositionNotRegistered:
			summary.Unregistered++
		case inboxlaporanklaim.PositionOutstanding:
			summary.Outstanding++
		}
		if r.accepted(row) {
			summary.Accepted++
		}
	}

	// Ketiga pencacah komunikasi memakai pasangan (status, pengirim) milik PENCACAH lama,
	// yang untuk tab pertama berbeda dari kueri gridnya. Lihat inboxlaporanklaim.MessageFilter.
	summary.MessageUnanswered = r.countMessage(scope, "0", false)
	summary.MessageWaiting = r.countMessage(scope, "0", true)
	summary.MessageReplied = r.countMessage(scope, "1", true)

	return summary, nil
}

// ListRegions mengembalikan isi dropdown "Pilih Kanwil".
func (r *Repo) ListRegions(_ context.Context) ([]inboxlaporanklaim.Region, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]inboxlaporanklaim.Region, len(r.region))
	copy(result, r.region)
	return result, nil
}

// Get mengembalikan satu berkas laporan.
func (r *Repo) Get(_ context.Context, id string) (inboxlaporanklaim.ClaimReport, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return inboxlaporanklaim.ClaimReport{}, r.failure
	}

	clean := strings.TrimSpace(id)
	for _, row := range r.rows {
		if row.ID == clean {
			return row, nil
		}
	}
	return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrNotFound
}

// Insert menerbitkan nomor lalu menyimpan berkas baru.
func (r *Repo) Insert(
	_ context.Context,
	report inboxlaporanklaim.ClaimReport,
) (inboxlaporanklaim.ClaimReport, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return inboxlaporanklaim.ClaimReport{}, r.failure
	}

	r.sequence++
	saved := report
	saved.Origin = inboxlaporanklaim.OriginNew
	saved.ID = inboxlaporanklaim.FormatReportNumber(r.clock.Now().Year(), r.sequence)
	saved.Position = inboxlaporanklaim.DerivePosition(saved.Registered(), saved.Transferred)
	saved.BranchName = r.branchName(saved.BranchCode)

	r.rows = append(r.rows, saved)
	return saved, nil
}

// match menyaring seluruh baris menurut kategori dan penyaring lainnya.
func (r *Repo) match(filter inboxlaporanklaim.Filter) []inboxlaporanklaim.ClaimReport {
	var result []inboxlaporanklaim.ClaimReport

	for _, row := range r.rows {
		if !r.matchScope(row, filter) {
			continue
		}
		if !r.matchCategory(row, filter) {
			continue
		}

		enriched := row
		if message, filtered := inboxlaporanklaim.MessageFilterOf(filter.Category); filtered {
			enriched.LastMessage = r.lastMessage(row.ID, message, filter.Operator)
		} else {
			enriched.LastMessage = ""
		}
		result = append(result, enriched)
	}
	return result
}

// matchScope memberlakukan penyaring yang berlaku di SELURUH tab.
func (r *Repo) matchScope(row inboxlaporanklaim.ClaimReport, filter inboxlaporanklaim.Filter) bool {
	if filter.BranchCode != "" && row.BranchCode != filter.BranchCode {
		return false
	}
	if filter.RegionCode != "" && r.branch[row.BranchCode] != filter.RegionCode {
		return false
	}
	// Pencarian PERSIS, bukan sebagian — lihat catatan pada Filter.Keyword.
	if filter.Keyword != "" && !strings.EqualFold(row.ID, filter.Keyword) {
		return false
	}
	return matchBusinessLine(row, filter.BusinessLine)
}

// matchCategory memberlakukan aturan tab.
func (r *Repo) matchCategory(row inboxlaporanklaim.ClaimReport, filter inboxlaporanklaim.Filter) bool {
	switch filter.Category {
	case inboxlaporanklaim.CategoryAll:
		return !r.resolved(row)

	case inboxlaporanklaim.CategoryOutstanding:
		return !r.resolved(row) && row.Position == inboxlaporanklaim.PositionOutstanding

	case inboxlaporanklaim.CategoryUnregistered:
		return !r.resolved(row) && row.Position == inboxlaporanklaim.PositionNotRegistered

	case inboxlaporanklaim.CategoryNotTransferred:
		return !r.resolved(row) && row.Position == inboxlaporanklaim.PositionNotTransferred

	case inboxlaporanklaim.CategoryAccepted:
		// Kueri lama TIDAK menyaring PYSTATUSWORK di tab ini — satu-satunya tab yang
		// tidak. Perilakunya dipertahankan apa adanya.
		return row.Position == inboxlaporanklaim.PositionOutstanding && r.accepted(row)

	case inboxlaporanklaim.CategoryRejected:
		return r.rejected(row)

	case inboxlaporanklaim.CategoryMessageUnanswered,
		inboxlaporanklaim.CategoryMessageWaiting,
		inboxlaporanklaim.CategoryMessageReplied:
		// Ketiga tab komunikasi menyaring berkas milik pemanggil sendiri, lalu
		// keberadaan percakapan yang cocok.
		if filter.Operator == "" || row.CreatedBy != filter.Operator {
			return false
		}
		message, _ := inboxlaporanklaim.MessageFilterOf(filter.Category)
		return r.lastMessage(row.ID, message, filter.Operator) != ""

	default:
		return false
	}
}

// lastMessage mengembalikan pesan terakhir yang cocok, atau kosong bila tidak ada.
func (r *Repo) lastMessage(
	id string,
	filter inboxlaporanklaim.MessageFilter,
	operator string,
) string {
	var newest *messageRow
	for i := range r.message[id] {
		row := &r.message[id][i]
		if row.Status != filter.Status {
			continue
		}
		if (row.Sender == operator) != filter.FromSelf {
			continue
		}
		if newest == nil || row.SentAt.After(newest.SentAt) {
			newest = row
		}
	}
	if newest == nil {
		return ""
	}
	return newest.Text
}

// countMessage mencacah berkas yang punya percakapan cocok, seperti pencacah lama.
func (r *Repo) countMessage(filter inboxlaporanklaim.Filter, status string, fromSelf bool) int {
	if filter.Operator == "" {
		return 0
	}

	count := 0
	for _, row := range r.rows {
		if !r.matchScope(row, filter) || r.resolved(row) {
			continue
		}
		if row.CreatedBy != filter.Operator {
			continue
		}
		hit := r.lastMessage(row.ID, inboxlaporanklaim.MessageFilter{
			Status:   status,
			FromSelf: fromSelf,
		}, filter.Operator)
		if hit != "" {
			count++
		}
	}
	return count
}

func (r *Repo) branchName(code string) string {
	for _, region := range r.region {
		if region.Code == r.branch[code] {
			return region.Name
		}
	}
	return ""
}

// resolved dan rejected menirukan penyaring PYSTATUSWORK pada kueri lama.
//
// Penyimpanan memori tidak menyimpan kolom itu sebagai teks; yang disimpan adalah
// akibatnya. Berkas contoh yang ditolak ditandai lewat rejectedID.
func (r *Repo) resolved(row inboxlaporanklaim.ClaimReport) bool {
	return r.rejected(row) || completedID[row.ID]
}

func (r *Repo) rejected(row inboxlaporanklaim.ClaimReport) bool { return rejectedID[row.ID] }

func (r *Repo) accepted(row inboxlaporanklaim.ClaimReport) bool { return acceptedID[row.ID] }

// matchBusinessLine memberlakukan dropdown "Bisnis".
//
// Penyimpanan memori tidak memuat POOLDATA.BUSINESS maupun BUSINESSGROUP, sehingga kode
// Group Panel dan kelompok bisnis dititipkan pada berkas contoh (lihat sample.go).
func matchBusinessLine(row inboxlaporanklaim.ClaimReport, line inboxlaporanklaim.BusinessLine) bool {
	if line == inboxlaporanklaim.BusinessLineAll {
		return true
	}

	panel, groupIn, groupNotIn := line.Criteria()
	meta := businessOf(row.ID)

	if len(panel) > 0 && !contains(panel, meta.groupPanel) {
		return false
	}
	if len(groupIn) > 0 && !contains(groupIn, meta.businessGroup) {
		return false
	}
	if len(groupNotIn) > 0 && contains(groupNotIn, meta.businessGroup) {
		return false
	}
	return true
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

var _ inboxlaporanklaim.Repo = (*Repo)(nil)
