// Package memory adalah pengisi seam inboxinvestigator.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang membuat
// seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan tanpa
// Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga URUTAN barisnya, kolom mana saja yang
// ikut dicari, dan CARA PEMOTONGANNYA — kalau tidak, uji yang lulus di sini tidak
// membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxinvestigator"
)

// Repo menyimpan antrean pekerjaan Investigator satu portal.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah antrean.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah `ADR-0030`
// dan `R-20`.
type Repo struct {
	// mutex melindungi seluruh isi.
	//
	// Modul ini tidak menulis, sehingga mutex di sini hanya menjaga SetError dan pembacaan
	// bersamaan — bukan padanan kunci tabel seperti pada modul yang menyisipkan baris.
	mutex   sync.Mutex
	tasks   []inboxinvestigator.Task
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Tasks []inboxinvestigator.Task
}

// NewRepo membentuk repo berisi pekerjaan yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{}
	r.tasks = append(r.tasks, o.Tasks...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo { return NewRepo(Options{Tasks: SampleTasks()}) }

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan pekerjaan yang cocok dengan penyaring.
//
// # Urutannya SAMA dengan adapter SQL, dan itu bukan kerapian
//
// `ORDER BY PYID, PXCREATEDATETIME, PZINSKEY` ditiru persis. Bila keduanya berbeda, uji
// urutan yang lulus di sini tidak membuktikan apa pun tentang perilaku nyata — dan
// perbedaannya baru terlihat setelah layar dipakai terhadap Oracle.
//
// # Pemotongannya juga SAMA
//
// Dipotong pada MaxRows, dan Truncated menyatakan masih ada sisa. Repo memori yang tidak
// pernah memotong akan membuat jalur "terpotong" tidak pernah tercoba sampai produksi.
//
// # Yang TIDAK ditiru: penyaring workbasket dan status
//
// Keduanya ada di dalam kueri SQL, bukan di dalam penyaring. Repo ini menganggap seluruh
// isinya SUDAH merupakan antrean investigator yang belum selesai — persis seperti kueri SQL
// yang tidak pernah mengembalikan baris di luar itu. Lihat SampleTasks.
func (r *Repo) List(
	_ context.Context,
	filter inboxinvestigator.Filter,
) (inboxinvestigator.Page, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return inboxinvestigator.Page{}, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	matched := make([]inboxinvestigator.Task, 0, len(r.tasks))
	for _, one := range r.tasks {
		if keyword != "" && !matches(one, keyword) {
			continue
		}
		matched = append(matched, one)
	}

	sort.SliceStable(matched, func(a, b int) bool {
		left, right := matched[a], matched[b]
		if left.CaseNumber != right.CaseNumber {
			return left.CaseNumber < right.CaseNumber
		}
		// Tanggal Pendaftaran yang kosong diperlakukan paling awal, sama seperti NULL pada
		// `ORDER BY ... ASC` di PostgreSQL. Oracle menaruh NULL di akhir secara bawaan;
		// perbedaan itu tidak dijembatani di sini karena kolomnya adalah stempel waktu
		// pembuatan kasus, yang tidak pernah kosong pada baris yang sah.
		leftAt, rightAt := left.RegisteredAt, right.RegisteredAt
		if leftAt != nil && rightAt != nil && !leftAt.Equal(*rightAt) {
			return leftAt.Before(*rightAt)
		}
		if (leftAt == nil) != (rightAt == nil) {
			return leftAt == nil
		}
		return left.Reference < right.Reference
	})

	truncated := len(matched) > inboxinvestigator.MaxRows
	if truncated {
		matched = matched[:inboxinvestigator.MaxRows]
	}
	return inboxinvestigator.Page{Tasks: matched, Truncated: truncated}, nil
}

// matches meniru klausa LIKE pada investigator_inbox_search.
//
// Perbandingannya TANPA memandang huruf besar-kecil, sama seperti `UPPER(...) LIKE ...` di
// sana. Kata kuncinya sudah di-uppercase oleh pemanggil.
//
// Ketujuh kolomnya sama persis dengan yang dicari kueri SQL. Tanggal Pendaftaran dan Tanggal
// Survey tidak ikut di kedua tempat, dengan alasan yang sama.
func matches(one inboxinvestigator.Task, keyword string) bool {
	for _, field := range []string{
		one.CaseNumber,
		one.PolicyNumber,
		one.InsuredName,
		one.ParticipantName,
		one.BusinessName,
		one.BranchName,
		one.AdminName,
	} {
		if strings.Contains(strings.ToUpper(strings.TrimSpace(field)), keyword) {
			return true
		}
	}
	return false
}

var _ inboxinvestigator.Repo = (*Repo)(nil)
