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
	exports []inboxinvestigator.ExportRow
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Tasks []inboxinvestigator.Task

	// Exports adalah isi berkas Export Data Investigation.
	//
	// Ia DAFTAR TERSENDIRI, bukan turunan Tasks, karena di adapter SQL pun keduanya datang
	// dari sumber yang berbeda: inbox membaca antrean workbasket, ekspor membaca klaim
	// menurut rentang `INVESTIGATOR_TF_DATE`. Menurunkannya dari Tasks akan membuat uji
	// memori meloloskan pemahaman yang keliru tentang hubungan keduanya.
	Exports []inboxinvestigator.ExportRow
}

// NewRepo membentuk repo berisi pekerjaan yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{}
	r.tasks = append(r.tasks, o.Tasks...)
	r.exports = append(r.exports, o.Exports...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{Tasks: SampleTasks(), Exports: SampleExportRows()})
}

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
// `ORDER BY PYID DESC, PXCREATEDATETIME DESC, PZINSKEY DESC` ditiru persis. Bila keduanya
// berbeda, uji urutan yang lulus di sini tidak membuktikan apa pun tentang perilaku nyata —
// dan perbedaannya baru terlihat setelah layar dipakai terhadap Oracle.
//
// Arah MENURUN karena Report Definition menyatakannya begitu: `pySortType = DESC` pada
// `.pyID` maupun `.pxCreateDateTime`. Pekerjaan terbaru di atas.
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
			return left.CaseNumber > right.CaseNumber
		}
		// Tanggal Pendaftaran yang kosong diperlakukan paling akhir, sama seperti NULL pada
		// `ORDER BY ... DESC` di PostgreSQL. Oracle menaruh NULL di awal pada urutan
		// menurun; perbedaan itu tidak dijembatani di sini karena kolomnya adalah stempel
		// waktu pembuatan kasus, yang tidak pernah kosong pada baris yang sah.
		leftAt, rightAt := left.RegisteredAt, right.RegisteredAt
		if leftAt != nil && rightAt != nil && !leftAt.Equal(*rightAt) {
			return leftAt.After(*rightAt)
		}
		if (leftAt == nil) != (rightAt == nil) {
			return rightAt == nil
		}
		return left.Reference > right.Reference
	})

	truncated := len(matched) > inboxinvestigator.MaxRows
	if truncated {
		matched = matched[:inboxinvestigator.MaxRows]
	}
	return inboxinvestigator.Page{Tasks: matched, Truncated: truncated}, nil
}

// Export mengembalikan baris ekspor yang cocok dengan penyaing rentang dan pilihan
// investigasi.
//
// # Penyaringnya ditiru, bukan diabaikan
//
// Repo memori yang mengembalikan seluruh isinya akan membuat uji penyaring ekspor selalu
// lulus tanpa membuktikan apa pun — dan penyaring itulah satu-satunya yang menentukan isi
// berkas. Ketiganya ditiru: batas bawah inklusif, batas atas inklusif pada HARI-nya, dan
// pencocokan tepat pada `IsInvestigated`.
//
// Perhatikan batas atas: kueri SQL membandingkan `< (Sampai + 1 hari)` karena kolomnya
// bertipe tanggal-berwaktu, sementara di sini perbandingannya dilakukan pada hari yang sama
// lewat Before pada hari berikutnya. Keduanya menjawab pertanyaan yang sama — "apakah ia
// terjadi pada hari Sampai atau sebelumnya" — dan itulah yang diuji, bukan cara menuliskannya.
func (r *Repo) Export(
	_ context.Context,
	filter inboxinvestigator.ExportFilter,
) ([]inboxinvestigator.ExportRow, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return nil, r.failure
	}

	beforeDay := filter.To.AddDate(0, 0, 1)

	matched := make([]inboxinvestigator.ExportRow, 0, len(r.exports))
	for _, one := range r.exports {
		if one.Investigated != filter.Investigated {
			continue
		}
		if one.InvestigatedAt == nil {
			continue
		}
		if one.InvestigatedAt.Before(filter.From) || !one.InvestigatedAt.Before(beforeDay) {
			continue
		}
		matched = append(matched, one)
	}

	sort.SliceStable(matched, func(a, b int) bool {
		return matched[a].InvestigatedAt.After(*matched[b].InvestigatedAt)
	})

	if len(matched) > inboxinvestigator.MaxExportRows {
		matched = matched[:inboxinvestigator.MaxExportRows]
	}
	return matched, nil
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
