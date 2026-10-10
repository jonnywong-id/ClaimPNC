// Package memory adalah pengisi seam masterreas.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang membuat
// seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan tanpa Oracle
// saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga URUTAN barisnya dan kolom mana saja
// yang ikut dicari — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterreas"
)

// Repo menyimpan member reasuransi satu portal.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah `ADR-0030`
// dan `R-20`.
type Repo struct {
	// mutex melindungi seluruh isi.
	//
	// Modul ini tidak menulis, sehingga mutex di sini hanya menjaga SetError dan pembacaan
	// bersamaan — bukan padanan kunci tabel seperti pada modul yang menyisipkan baris.
	mutex   sync.Mutex
	rows    []masterreas.Member
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows []masterreas.Member
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{}
	r.rows = append(r.rows, o.Rows...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo { return NewRepo(Options{Rows: SampleList()}) }

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan baris yang cocok dengan penyaring.
//
// # Urutannya SAMA dengan adapter SQL, dan itu bukan kerapian
//
// `ORDER BY REINSURERNAME, TYPE, REINSURERID` ditiru persis. Bila keduanya berbeda, uji
// paginasi dan uji urutan yang lulus di sini tidak membuktikan apa pun tentang perilaku
// nyata — dan perbedaannya baru terlihat setelah layar dipakai terhadap Oracle.
//
// # Kolom yang dicari juga SAMA
//
// Empat: kode reas, nama reas, login, dan email. Country dan Type sengaja tidak ikut; lihat
// masterreas.Filter.
func (r *Repo) List(
	_ context.Context,
	filter masterreas.Filter,
) ([]masterreas.Member, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []masterreas.Member
	for _, one := range r.rows {
		if keyword != "" && !matches(one, keyword) {
			continue
		}
		result = append(result, one)
	}

	sort.SliceStable(result, func(a, b int) bool {
		left, right := result[a], result[b]
		if left.ReinsurerName != right.ReinsurerName {
			return left.ReinsurerName < right.ReinsurerName
		}
		if left.Type != right.Type {
			return left.Type < right.Type
		}
		return left.ReinsurerID < right.ReinsurerID
	})
	return result, nil
}

// Update mengubah surel baris yang kuncinya cocok.
//
// Meniru adapter SQL pada dua hal yang menentukan, dan keduanya mudah terlewat:
//
//   - Pembandingnya memangkas spasi di KEDUA sisi, sama seperti `COALESCE(TRIM(...), ”)`.
//   - Baris yang tidak ada menghasilkan ErrNotFound, bukan diam-diam tidak melakukan apa pun.
//
// Seluruh baris yang cocok diubah, bukan hanya yang pertama — kunci alaminya tidak dijamin
// unik (`R-08`), dan adapter SQL pun menyentuh seluruhnya.
func (r *Repo) Update(_ context.Context, key masterreas.Key, email string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return r.failure
	}

	clean := key.Clean()
	changed := false
	for i, one := range r.rows {
		if strings.TrimSpace(one.ReinsurerID) != clean.ReinsurerID ||
			strings.TrimSpace(one.ReinsurerName) != clean.ReinsurerName ||
			strings.TrimSpace(one.Type) != clean.Type {
			continue
		}
		r.rows[i].Email = strings.TrimSpace(email)
		changed = true
	}

	if !changed {
		return masterreas.ErrNotFound
	}
	return nil
}

// matches meniru klausa LIKE pada reas_list_search.
//
// Perbandingannya TANPA memandang huruf besar-kecil, sama seperti `UPPER(...) LIKE ...` di
// sana. Kata kuncinya sudah di-uppercase oleh pemanggil.
func matches(one masterreas.Member, keyword string) bool {
	for _, field := range []string{
		one.ReinsurerID,
		one.ReinsurerName,
		one.Login,
		one.Email,
	} {
		if strings.Contains(strings.ToUpper(strings.TrimSpace(field)), keyword) {
			return true
		}
	}
	return false
}

var _ masterreas.Repo = (*Repo)(nil)
