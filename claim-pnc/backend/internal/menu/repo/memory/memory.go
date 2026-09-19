// Package memory adalah pengisi seam menu.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Berbeda dari adapter memori modul lain, isi contohnya BUKAN susunan sendiri: ketiga
// daftarnya disalin apa adanya dari `Database/m_menu_aplikasi_pnc.csv`,
// `m_login_group_pnc.csv`, dan `m_otorisasi_pnc.csv` yang diterima bersama DDL-nya.
// Menu yang terlihat saat pengembangan karena itu sama persis dengan menu produksi.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/menu"
)

// Repo menyimpan peta menu dan kewenangannya di memori.
type Repo struct {
	// mutex melindungi ketiga simpanan. Permintaan HTTP dilayani beberapa goroutine
	// sekaligus, dan SetError dipanggil dari uji sementara pembacaan sedang berjalan.
	mutex   sync.Mutex
	items   []menu.Item
	groups  map[string][]string
	grants  map[string][]int
	failure error
}

// NewRepo membentuk repo berisi peta menu, keanggotaan group, dan izin yang diberikan.
//
// Kunci kedua peta diseragamkan menjadi huruf besar, sama seperti yang dilakukan
// adapter SQL lewat UPPER(TRIM(...)). Tanpa itu, adapter memori akan menerima hal yang
// ditolak adapter SQL — dan uji yang lulus di sini tidak membuktikan apa pun tentang
// yang berjalan di produksi.
func NewRepo(items []menu.Item, groups map[string][]string, grants map[string][]int) *Repo {
	r := &Repo{
		items:  append([]menu.Item(nil), items...),
		groups: map[string][]string{},
		grants: map[string][]int{},
	}
	for login, list := range groups {
		r.groups[normalize(login)] = append([]string(nil), list...)
	}
	for subject, list := range grants {
		r.grants[normalize(subject)] = append([]int(nil), list...)
	}
	return r
}

// NewSampleRepo membentuk repo berisi seluruh isi contoh yang diterima bersama DDL,
// apa adanya.
func NewSampleRepo() *Repo { return NewRepo(SampleItems(), SampleGroups(), SampleGrants()) }

// NewDevRepo membentuk repo untuk menjalankan aplikasi tanpa Oracle.
//
// Isinya sama dengan NewSampleRepo, DITAMBAH keanggotaan group untuk login contoh milik
// provider identitas tiruan. Tanpa tambahan itu, masuk sebagai `adminpnc` saat
// pengembangan menghasilkan menu kosong — bukan karena ada yang rusak, melainkan karena
// login itu memang tidak ada di `m_login_group_pnc.csv`, yang saat diterima hanya memuat
// satu baris.
//
// Tambahan ini HANYA hidup di adapter memori. Jalur Oracle membaca tabel yang
// sebenarnya, sehingga tidak ada satu pun izin karangan yang sampai ke produksi.
func NewDevRepo() *Repo {
	groups := map[string][]string{}
	for login, list := range SampleGroups() {
		groups[login] = list
	}
	// Login tiruan diikutkan ke group IT, yang di data contoh memegang MASTER, INBOX,
	// dan VIEW. REPORT sengaja TIDAK diberikan: ia hanya dimiliki login JONNY, dan
	// perbedaan itulah yang membuat penggabungan izin group dan izin login terlihat
	// saat aplikasi dicoba.
	for _, login := range []string{"adminpnc", "pictekniks", "brokercontoh", "profilbolong"} {
		groups[normalize(login)] = []string{"IT"}
	}
	return NewRepo(SampleItems(), groups, SampleGrants())
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh butir menu, terurut MENU_SEQUENCE seperti kueri lama.
func (r *Repo) List(_ context.Context, appName string) ([]menu.Item, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	// Hanya satu aplikasi yang dilayani adapter ini. Nama lain ditolak dengan galat yang
	// sama seperti di produksi, sehingga perilaku penolakannya ikut teruji saat
	// pengembangan — bukan hanya nanti.
	if normalize(appName) != normalize(menu.AppName) {
		return nil, menu.ErrAppNotFound
	}

	result := make([]menu.Item, len(r.items))
	copy(result, r.items)
	sort.SliceStable(result, func(a, b int) bool { return result[a].Sequence < result[b].Sequence })
	return result, nil
}

// GroupsOf mengembalikan GROUP_ID yang diikuti sebuah login.
func (r *Repo) GroupsOf(_ context.Context, loginID string) ([]string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	return append([]string(nil), r.groups[normalize(loginID)]...), nil
}

// AuthorizedIDs mengembalikan MENU_ID yang diizinkan untuk subjek-subjek yang disebut.
func (r *Repo) AuthorizedIDs(_ context.Context, appName string, subjects []string) ([]int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	if normalize(appName) != normalize(menu.AppName) {
		return nil, menu.ErrAppNotFound
	}

	seen := map[int]bool{}
	var result []int
	for _, subject := range subjects {
		for _, id := range r.grants[normalize(subject)] {
			if seen[id] {
				continue
			}
			seen[id] = true
			result = append(result, id)
		}
	}
	sort.Ints(result)
	return result, nil
}

func normalize(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func parent(id int) *int { return &id }

var _ menu.Repo = (*Repo)(nil)
