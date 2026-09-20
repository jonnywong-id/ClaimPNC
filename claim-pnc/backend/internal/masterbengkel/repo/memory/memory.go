// Package memory adalah pengisi seam masterbengkel.Store yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara penyaring
// bekerja, kolom mana yang TIDAK ikut berubah saat disimpan, dan bentuk kunci yang
// diterbitkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterbengkel"
)

// sequenceWidth disalin dari adapter SQL, dan harus sama dengannya.
//
// Ia TIDAK diimpor dari sana: paket memory tidak boleh bergantung pada paket sqlstore,
// yang menyeret driver basis data ke dalam uji yang justru dibuat agar tidak
// membutuhkannya. Kesamaannya dijaga uji, bukan oleh kompilator.
const sequenceWidth = 10

// Repo menyimpan master bengkel satu portal beserta ketiga tabel acuannya.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// ADR-0030 dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi. Permintaan HTTP dilayani beberapa goroutine
	// sekaligus, dan penambahan yang menolak kunci ganda harus berjalan satu per satu —
	// persis seperti transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []masterbengkel.Workshop
	failure error

	branch []masterbengkel.Branch
	city   []masterbengkel.City
	bank   []masterbengkel.Bank

	// site dan sequence meniru kode situs dan sequence basis data. Keduanya di sini
	// supaya kunci yang diterbitkan saat pengembangan berbentuk sama dengan yang
	// diterbitkan di produksi — bukan angka berurut yang terlihat berbeda.
	site     string
	sequence int64
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows     []masterbengkel.Workshop
	Branches []masterbengkel.Branch
	Cities   []masterbengkel.City
	Banks    []masterbengkel.Bank

	// Site adalah kode situs yang dipakai menerbitkan ID. Kosong berarti SampleSite.
	Site string

	// Sequence adalah nomor urut terakhir yang sudah dipakai.
	Sequence int64
}

// NewRepo membentuk repo berisi baris dan acuan yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{site: strings.TrimSpace(o.Site), sequence: o.Sequence}
	if r.site == "" {
		r.site = SampleSite
	}
	r.rows = append(r.rows, o.Rows...)
	r.branch = append(r.branch, o.Branches...)
	r.city = append(r.city, o.Cities...)
	r.bank = append(r.bank, o.Banks...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:     SampleList(),
		Branches: SampleBranches(),
		Cities:   SampleCities(),
		Banks:    SampleBanks(),
		Site:     SampleSite,
		Sequence: SampleSequence,
	})
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan baris yang cocok dengan penyaring.
//
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada APPROVAL
// dan pencocokan kata kunci pada ketiga kolom yang sama.
func (r *Repo) List(_ context.Context, filter masterbengkel.Filter) ([]masterbengkel.Workshop, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []masterbengkel.Workshop
	for _, w := range r.rows {
		if w.Status != filter.Status {
			continue
		}
		if keyword != "" && !matchesKeyword(w, keyword) {
			continue
		}
		result = append(result, w)
	}

	// `ORDER BY ID_BENGKEL DESC` pada basis data adalah pengurutan TEKS bila kolomnya
	// bertipe teks. Ditiru apa adanya — termasuk arah menurunnya, yang meniru
	// `pySortType=DESC` pada kolom pertama grid Pega — supaya urutan yang terlihat saat
	// pengembangan sama dengan urutan yang terlihat di produksi.
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan ID_BENGKEL-nya.
func (r *Repo) Get(_ context.Context, id string) (masterbengkel.Workshop, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterbengkel.Workshop{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, w := range r.rows {
		if w.ID == wanted {
			return w, nil
		}
	}
	return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
}

// FindByName mencari baris menurut NAMA_BENGKEL-nya.
//
// Pencocokannya mengabaikan besar-kecil huruf dan spasi tepi, meniru
// `upper(trim(nama_bengkel))` pada kueri aslinya.
func (r *Repo) FindByName(_ context.Context, name string) (masterbengkel.Workshop, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterbengkel.Workshop{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))
	if wanted == "" {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}
	for _, w := range r.rows {
		if strings.ToUpper(strings.TrimSpace(w.Name)) == wanted {
			return w, nil
		}
	}
	return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
}

// FindByLogin mencari baris menurut LOGIN_APLIKASI-nya.
//
// Login kosong TIDAK PERNAH cocok — sama seperti penyaring `IS NOT NULL` pada kuerinya.
// Tanpa itu, bengkel non-rekanan kedua akan ditolak karena "login kosong sudah dipakai".
func (r *Repo) FindByLogin(_ context.Context, login string) (masterbengkel.Workshop, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterbengkel.Workshop{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(login))
	if wanted == "" {
		return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
	}
	for _, w := range r.rows {
		stored := strings.ToUpper(strings.TrimSpace(w.Login))
		if stored != "" && stored == wanted {
			return w, nil
		}
	}
	return masterbengkel.Workshop{}, masterbengkel.ErrNotFound
}

// Insert menolak nama dan login yang sudah dipakai, lalu menambahkan barisnya.
func (r *Repo) Insert(_ context.Context, w masterbengkel.Workshop) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	name := strings.ToUpper(strings.TrimSpace(w.Name))
	login := strings.ToUpper(strings.TrimSpace(w.Login))
	for _, exists := range r.rows {
		if strings.ToUpper(strings.TrimSpace(exists.Name)) == name {
			return masterbengkel.ErrNameTaken
		}
		stored := strings.ToUpper(strings.TrimSpace(exists.Login))
		if login != "" && stored == login {
			return masterbengkel.ErrLoginTaken
		}
	}
	r.rows = append(r.rows, w)
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// ID_BENGKEL SENGAJA TIDAK IKUT DITIMPA, persis seperti `bengkel_update` pada adapter
// SQL yang tidak menyebut kolom itu. Tanpa peniruan ini, uji yang membuktikan kunci
// tidak berpindah akan lulus di memori dan gagal di Oracle — atau lebih buruk,
// sebaliknya.
func (r *Repo) Update(_ context.Context, w masterbengkel.Workshop) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(w.ID)
	for i, exists := range r.rows {
		if exists.ID != wanted {
			continue
		}
		kept := r.rows[i].ID
		r.rows[i] = w
		r.rows[i].ID = kept
		return nil
	}
	return masterbengkel.ErrNotFound
}

// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
//
// Barisnya yang sudah berstatus itu TIDAK dihitung sebagai berubah, meniru penyaring
// `AND TRIM(APPROVAL) <> :1` pada kuerinya.
func (r *Repo) SetStatus(_ context.Context, id []string, status masterbengkel.ApprovalStatus) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return 0, r.failure
	}

	changed := 0
	for _, one := range id {
		wanted := strings.TrimSpace(one)
		for i := range r.rows {
			if r.rows[i].ID != wanted || r.rows[i].Status == status {
				continue
			}
			r.rows[i].Status = status
			changed++
		}
	}
	return changed, nil
}

// NextID menerbitkan ID_BENGKEL berikutnya dengan bentuk yang sama seperti adapter SQL.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	r.sequence++
	return masterbengkel.ComposeID(r.site, r.sequence, sequenceWidth), nil
}

// ListBranches mengembalikan seluruh cabang contoh.
func (r *Repo) ListBranches(_ context.Context) ([]masterbengkel.Branch, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterbengkel.Branch, len(r.branch))
	copy(result, r.branch)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// SearchCities mencari kota pada daftar contoh.
func (r *Repo) SearchCities(_ context.Context, keyword string) ([]masterbengkel.City, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	clean := strings.ToUpper(strings.TrimSpace(keyword))
	if clean == "" {
		return nil, nil
	}

	var result []masterbengkel.City
	for _, c := range r.city {
		if !strings.Contains(strings.ToUpper(c.Name), clean) &&
			!strings.EqualFold(strings.TrimSpace(c.ID), clean) {
			continue
		}
		result = append(result, c)
		if len(result) == masterbengkel.MaxLookupRows {
			break
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListBanks mengembalikan seluruh bank contoh.
func (r *Repo) ListBanks(_ context.Context) ([]masterbengkel.Bank, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterbengkel.Bank, len(r.bank))
	copy(result, r.bank)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// matchesKeyword meniru penyaring kata kunci adapter SQL: nama bengkel, nama kota, atau
// nama cabang, dicocokkan sebagian tanpa memandang besar-kecil huruf.
func matchesKeyword(w masterbengkel.Workshop, keyword string) bool {
	for _, field := range []string{w.Name, w.CityName, w.BranchName} {
		if strings.Contains(strings.ToUpper(field), keyword) {
			return true
		}
	}
	return false
}

var _ masterbengkel.Store = (*Repo)(nil)
