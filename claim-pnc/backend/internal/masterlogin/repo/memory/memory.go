// Package memory adalah pengisi seam masterlogin.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga URUTAN barisnya, kolom mana saja yang
// ikut dicari, cakupan pemeriksaan login ganda, dan perbedaan huruf besar-kecil antara
// pengambilan dan pemeriksaan keunikan — kalau tidak, uji yang lulus di sini tidak
// membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterlogin"
)

// Repo menyimpan master login surveyor satu portal.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah ADR-0030
// dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi.
	//
	// Ia padanan `LOCK TABLE ... IN EXCLUSIVE MODE` pada adapter SQL: dua penambahan
	// bersamaan dengan Nama yang sama menghasilkan LOGIN yang sama, dan keduanya akan lolos
	// pemeriksaan keunikan sebelum salah satunya menyisipkan. Mutex di sini menutupnya
	// dengan cara yang sama — satu penulis pada satu saat.
	mutex   sync.Mutex
	rows    []masterlogin.SurveyorLogin
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows []masterlogin.SurveyorLogin
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
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk KOLOM MANA SAJA yang
// ikut dicari — Nama, Login, dan Email saja. Telp dan Alamat sengaja tidak ikut di kedua
// adapter; menambahkannya di salah satu akan membuat uji yang lulus di sini tidak berlaku
// di sana.
func (r *Repo) List(
	_ context.Context,
	filter masterlogin.Filter,
) ([]masterlogin.SurveyorLogin, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []masterlogin.SurveyorLogin
	for _, one := range r.rows {
		if keyword != "" && !matches(one, keyword) {
			continue
		}
		result = append(result, one)
	}

	sortByName(result)
	return result, nil
}

// matches meniru ketiga klausa LIKE pada login_list_search.
func matches(one masterlogin.SurveyorLogin, keyword string) bool {
	return strings.Contains(strings.ToUpper(one.Name), keyword) ||
		strings.Contains(strings.ToUpper(one.Login), keyword) ||
		strings.Contains(strings.ToUpper(one.Email), keyword)
}

// sortByName mengurutkan seperti `ORDER BY NAMA, LOGIN`.
//
// LOGIN menjadi pemutus supaya urutannya tetap ketika ada dua nama yang sama — keadaan yang
// benar-benar mungkin, karena tabelnya tidak punya constraint unik apa pun pada NAMA.
func sortByName(list []masterlogin.SurveyorLogin) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Name != list[j].Name {
			return list[i].Name < list[j].Name
		}
		return list[i].Login < list[j].Login
	})
}

// Get mengembalikan satu baris berdasarkan LOGIN-nya.
//
// Pembandingnya PEKA huruf besar-kecil, meniru login_get yang tidak memakai UPPER — dan itu
// sengaja berbeda dari pemeriksaan keunikan pada Insert. Lihat banner berkas .sql.
func (r *Repo) Get(
	_ context.Context,
	login string,
) (masterlogin.SurveyorLogin, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterlogin.SurveyorLogin{}, r.failure
	}

	index := r.indexOf(login)
	if index < 0 {
		return masterlogin.SurveyorLogin{}, masterlogin.ErrNotFound
	}
	return r.rows[index], nil
}

// FindLeaderOf mengembalikan isi LOGINLEADER milik satu login.
//
// Baris yang tidak ada menghasilkan ErrNotFound, bukan teks kosong — sama seperti adapter
// SQL. Keduanya berbeda arti, dan hanya pemanggil yang tahu mana yang boleh dianggap wajar.
func (r *Repo) FindLeaderOf(_ context.Context, login string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	index := r.indexOf(login)
	if index < 0 {
		return "", masterlogin.ErrNotFound
	}
	return strings.TrimSpace(r.rows[index].LeaderLogin), nil
}

// Insert memeriksa keunikan LOGIN lalu menyisipkan.
//
// Keduanya di bawah satu mutex, meniru transaksi bertahan kunci tabel pada adapter SQL.
//
// Pemeriksaannya TIDAK peka huruf besar-kecil, meniru `UPPER(TRIM(LOGIN))` pada
// login_find_by_key — dan itu memang berbeda dari Get. Lihat banner berkas .sql.
func (r *Repo) Insert(
	_ context.Context,
	one masterlogin.SurveyorLogin,
) (masterlogin.SurveyorLogin, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterlogin.SurveyorLogin{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(one.Login))
	for _, existing := range r.rows {
		if strings.ToUpper(strings.TrimSpace(existing.Login)) == wanted {
			return masterlogin.SurveyorLogin{}, masterlogin.ErrLoginTaken
		}
	}

	r.rows = append(r.rows, one)
	return one, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// KETUJUH kolom ditulis, sama persis dengan `login_update`. Kuncinya dicari dengan
// perbandingan yang peka huruf besar-kecil, meniru penyaring `TRIM(LOGIN) = :8` pada kueri
// itu.
func (r *Repo) Update(_ context.Context, one masterlogin.SurveyorLogin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	index := r.indexOf(one.Login)
	if index < 0 {
		return masterlogin.ErrNotFound
	}

	r.rows[index] = one
	return nil
}

// indexOf mencari posisi baris menurut LOGIN-nya; -1 bila tidak ada.
//
// Pembandingnya dipangkas di kedua sisi, meniru `TRIM(...)` pada kueri SQL. Kunci kosong
// TIDAK pernah cocok: pada adapter SQL, penyaring berkunci kosong akan mengenai setiap
// baris berlogin kosong sekaligus — keadaan yang dilaporkan `claimpnc -periksa` lewat
// login_count_empty_key, dan yang tidak boleh dijadikan jalan membuka baris mana pun.
func (r *Repo) indexOf(login string) int {
	wanted := strings.TrimSpace(login)
	if wanted == "" {
		return -1
	}
	for i, one := range r.rows {
		if strings.TrimSpace(one.Login) == wanted {
			return i
		}
	}
	return -1
}

var _ masterlogin.Repo = (*Repo)(nil)
