// Package memory adalah pengisi seam inboxreceivetka.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang membuat
// seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan tanpa
// Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, kolom mana saja yang
// ikut dicari, cara pemotongannya, dan SELURUH galat yang dapat dihasilkan jalur tulis —
// kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxreceivetka"
)

// Repo menyimpan daftar pekerjaan Inbox Receive TKA satu portal.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah daftar.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah `ADR-0030`
// dan `R-20`.
type Repo struct {
	// mutex melindungi seluruh isi.
	//
	// Modul ini MENULIS, sehingga mutex di sini benar-benar berperan sebagai padanan
	// transaksi pada adapter SQL: ia yang membuat dua Complete bersamaan atas nomor klaim
	// yang sama tidak dapat sama-sama lolos, persis seperti `SELECT ... FOR UPDATE` di sana.
	mutex   sync.Mutex
	tasks   []inboxreceivetka.Task
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Tasks []inboxreceivetka.Task
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
// `ORDER BY w.REGISTERDATE_1, w.PYID` ditiru persis: tanggal registrasi menaik, sehingga
// yang paling lama menunggu tampil lebih dulu, dengan nomor kasus sebagai pemutus.
//
// Tanggal yang KOSONG jatuh di AKHIR — sejajar dengan NULL pada `ORDER BY ... ASC`, yang
// di Oracle maupun PostgreSQL ditaruh terakhir. Bila keduanya berbeda, uji urutan yang
// lulus di sini tidak membuktikan apa pun tentang perilaku nyata.
//
// # Pemotongannya SAMA
//
// Dipotong pada MaxRows, dan Truncated menyatakan masih ada sisa. Repo memori yang tidak
// pernah memotong akan membuat jalur "terpotong" tidak pernah tercoba sampai produksi.
//
// # Yang TIDAK ditiru: penyaring penanda TKA dan status pekerjaan
//
// Keduanya ada di dalam kueri SQL, bukan di dalam penyaring. Repo ini menganggap seluruh
// isinya SUDAH merupakan klaim TKA yang belum selesai — persis seperti kueri SQL yang tidak
// pernah mengembalikan baris di luar itu. Yang ditiru justru penyaring KETIGA: baris yang
// sudah diisi tanggalnya dibuang dari daftar oleh Complete, bukan ditandai.
func (r *Repo) List(
	_ context.Context,
	filter inboxreceivetka.Filter,
) (inboxreceivetka.Page, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return inboxreceivetka.Page{}, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	matched := make([]inboxreceivetka.Task, 0, len(r.tasks))
	for _, one := range r.tasks {
		if keyword != "" && !matches(one, keyword) {
			continue
		}
		matched = append(matched, one)
	}

	sort.SliceStable(matched, func(a, b int) bool {
		left, right := matched[a], matched[b]
		leftOn, rightOn := left.RegisteredOn, right.RegisteredOn
		if (leftOn == nil) != (rightOn == nil) {
			// Yang kosong jatuh di akhir, sama seperti NULL pada ORDER BY ... ASC.
			return rightOn == nil
		}
		if leftOn != nil && !leftOn.Equal(*rightOn) {
			return leftOn.Before(*rightOn)
		}
		return left.ClaimNumber < right.ClaimNumber
	})

	truncated := len(matched) > inboxreceivetka.MaxRows
	if truncated {
		matched = matched[:inboxreceivetka.MaxRows]
	}
	return inboxreceivetka.Page{Tasks: matched, Truncated: truncated}, nil
}

// Complete mengisi tanggal kelengkapan dokumen dan mengembalikan baris yang terisi.
//
// # Ketiga galat adapter SQL ditiru, dan itu yang membuat pengujian bermakna
//
//	tidak ada / sudah terisi   -> ErrTaskNotFound
//	nomor kasus kembar         -> ErrClaimAmbiguous
//	klaim tidak ada di bisnis  -> ErrClaimMissing
//
// Yang terakhir ditiru lewat Task.ClaimKey yang kosong. Pada adapter SQL, ClaimKey berasal
// dari gabungan LEFT ke `T_CLAIM_PNC`; kosong berarti pekerjaannya ada tetapi klaimnya
// tidak, dan di sana penulisannya ditolak SEBELUM menyentuh apa pun. Repo ini menolaknya
// dengan syarat yang sama, sehingga layar dapat diuji menghadapi baris yatim tanpa basis
// data.
//
// # Barisnya DIBUANG, bukan ditandai
//
// Adapter SQL mengisi `T_CLAIM_PNC.TGLDOKLENGKAP`, dan penyaring `IS NULL` pada kueri
// daftarnya yang membuat barisnya hilang. Akibatnya sama dengan membuang, dan membuang di
// sini membuat kedua adapter berperilaku identik dari sudut pandang pemanggil.
func (r *Repo) Complete(
	_ context.Context,
	one inboxreceivetka.Completion,
) (inboxreceivetka.Task, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return inboxreceivetka.Task{}, r.failure
	}

	cleaned := one.Clean()
	key := strings.ToUpper(strings.TrimSpace(cleaned.ClaimNumber))

	var found []int
	for index, task := range r.tasks {
		if strings.ToUpper(strings.TrimSpace(task.ClaimNumber)) == key {
			found = append(found, index)
		}
	}

	switch len(found) {
	case 0:
		return inboxreceivetka.Task{}, inboxreceivetka.ErrTaskNotFound
	case 1:
	default:
		return inboxreceivetka.Task{}, inboxreceivetka.ErrClaimAmbiguous
	}

	at := found[0]
	task := r.tasks[at]
	if strings.TrimSpace(task.ClaimKey) == "" {
		return inboxreceivetka.Task{}, inboxreceivetka.ErrClaimMissing
	}

	r.tasks = append(r.tasks[:at], r.tasks[at+1:]...)
	return task, nil
}

// matches meniru klausa LIKE pada tka_inbox_search.
//
// Perbandingannya TANPA memandang huruf besar-kecil, sama seperti `UPPER(...) LIKE ...` di
// sana. Kata kuncinya sudah di-uppercase oleh pemanggil.
//
// Keempat kolomnya sama persis dengan yang dicari kueri SQL. Date Of Loss dan Aging tidak
// ikut di kedua tempat, dengan alasan yang sama.
func matches(one inboxreceivetka.Task, keyword string) bool {
	for _, field := range []string{
		one.ClaimNumber,
		one.PolicyNumber,
		one.InsuredName,
		one.ParticipantName,
	} {
		if strings.Contains(strings.ToUpper(strings.TrimSpace(field)), keyword) {
			return true
		}
	}
	return false
}

var _ inboxreceivetka.Repo = (*Repo)(nil)
