// Package memory adalah pengisi seam masterkategorisparepart.Store yang hidup di dalam
// memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga URUTAN barisnya, cara penyaring
// bekerja, cakupan pemeriksaan nama ganda, dan bentuk kunci yang diterbitkan — kalau tidak,
// uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterkategorisparepart"
)

// Repo menyimpan master kategori sparepart satu portal.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah ADR-0030
// dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi.
	//
	// Ia padanan `LOCK TABLE ... IN EXCLUSIVE MODE` pada adapter SQL: penambahan menerbitkan
	// ID dengan `max+1`, dan dua penambahan bersamaan yang membaca maksimum yang sama akan
	// menerbitkan kunci kembar. Mutex di sini menutupnya dengan cara yang sama — satu
	// penulis pada satu saat.
	mutex   sync.Mutex
	rows    []masterkategorisparepart.PartCategory
	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows []masterkategorisparepart.PartCategory
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
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada APPROVAL dan
// pencarian yang hanya menyentuh nama.
func (r *Repo) List(
	_ context.Context,
	filter masterkategorisparepart.Filter,
) ([]masterkategorisparepart.PartCategory, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []masterkategorisparepart.PartCategory
	for _, c := range r.rows {
		if strings.TrimSpace(string(c.Status)) != string(filter.Status) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToUpper(c.Name), keyword) {
			continue
		}
		result = append(result, c)
	}

	sortByID(result)
	return result, nil
}

// sortByID mengurutkan dengan aturan yang SAMA PERSIS dengan adapter SQL.
//
// Keduanya memakai masterkategorisparepart.SortKey, yang di sisi SQL berbentuk
// `LPAD(TRIM(PART_CATEGORY_ID), 10, '0')`. Kolomnya `VARCHAR2(10)`, sehingga
// `ORDER BY PART_CATEGORY_ID` apa adanya akan menaruh "10" sebelum "9" — dan kalau adapter
// memori mengurutkan secara numerik sementara SQL mengurutkan secara teks, uji yang lulus
// di sini tidak membuktikan apa pun tentang yang berjalan di produksi.
//
// Kunci yang bentuknya janggal terurut paling belakang pada kedua adapter, karena huruf
// berada di atas angka pada perbandingan teks.
func sortByID(list []masterkategorisparepart.PartCategory) {
	sort.SliceStable(list, func(i, j int) bool {
		return masterkategorisparepart.SortKey(list[i].ID) <
			masterkategorisparepart.SortKey(list[j].ID)
	})
}

// Get mengembalikan satu baris berdasarkan kuncinya.
func (r *Repo) Get(
	_ context.Context,
	id string,
) (masterkategorisparepart.PartCategory, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterkategorisparepart.PartCategory{}, r.failure
	}

	index := r.indexOf(id)
	if index < 0 {
		return masterkategorisparepart.PartCategory{}, masterkategorisparepart.ErrNotFound
	}
	return r.rows[index], nil
}

// FindByName mencari baris menurut namanya.
//
// TIDAK menyaring status, meniru `ValidationSparepartCat` apa adanya. Bila dua baris
// bernama sama — keadaan yang mungkin ada pada data lama karena tidak ada constraint unik
// (R-08) — yang dikembalikan adalah yang ID-nya terkecil, sama seperti kueri SQL yang
// memakai `ORDER BY PART_CATEGORY_ID FETCH FIRST 1 ROW ONLY`.
func (r *Repo) FindByName(
	_ context.Context,
	name string,
) (masterkategorisparepart.PartCategory, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterkategorisparepart.PartCategory{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))

	var found []masterkategorisparepart.PartCategory
	for _, c := range r.rows {
		if strings.ToUpper(strings.TrimSpace(c.Name)) == wanted {
			found = append(found, c)
		}
	}
	if len(found) == 0 {
		return masterkategorisparepart.PartCategory{}, masterkategorisparepart.ErrNotFound
	}

	sortByID(found)
	return found[0], nil
}

// Insert menerbitkan ID, memeriksa keunikan nama, lalu menyisipkan.
//
// Ketiganya di bawah satu mutex, meniru transaksi bertahan kunci tabel pada adapter SQL.
// Urutan langkahnya pun sama: nama diperiksa lebih dulu, supaya nomor urut tidak terpakai
// oleh percobaan yang memang akan ditolak.
//
// ID pada argumen DIABAIKAN, sama seperti pada adapter SQL.
func (r *Repo) Insert(
	_ context.Context,
	c masterkategorisparepart.PartCategory,
) (masterkategorisparepart.PartCategory, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterkategorisparepart.PartCategory{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(c.Name))
	for _, existing := range r.rows {
		if strings.ToUpper(strings.TrimSpace(existing.Name)) == wanted {
			return masterkategorisparepart.PartCategory{},
				masterkategorisparepart.ErrNameTaken
		}
	}

	id, err := r.nextID()
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}

	fresh := masterkategorisparepart.PartCategory{
		ID:     id,
		Name:   c.Name,
		Status: c.Status,
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// nextID menerbitkan kunci berikutnya lewat aturan domain yang sama dengan adapter SQL.
//
// Perhitungannya TIDAK disalin ke sini: masterkategorisparepart.NextKey yang memilikinya,
// sehingga maksimum numerik, pelewatan kunci janggal, dan batas lebar kolom berperilaku
// identik pada kedua adapter. Lihat catatan pada NextKey untuk alasan ia tidak meniru
// `nvl(max(...),0)+1` milik Pega.
func (r *Repo) nextID() (string, error) {
	existing := make([]string, 0, len(r.rows))
	for _, c := range r.rows {
		existing = append(existing, c.ID)
	}
	return masterkategorisparepart.NextKey(existing)
}

// NextID menerbitkan ID berikutnya tanpa menyisipkan apa pun.
//
// Dipakai pemeriksaan, bukan jalur simpan; lihat masterkategorisparepart.IDSource.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}
	return r.nextID()
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Hanya NAMA dan STATUS yang ditulis, sama persis dengan `category_update`. Kuncinya tidak
// pernah berubah.
func (r *Repo) Update(_ context.Context, c masterkategorisparepart.PartCategory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	index := r.indexOf(c.ID)
	if index < 0 {
		return masterkategorisparepart.ErrNotFound
	}

	r.rows[index].Name = c.Name
	r.rows[index].Status = c.Status
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
//
// Baris yang tidak ada dilewati tanpa galat, dan baris yang sudah berstatus itu TIDAK
// dihitung berubah — keduanya meniru `RowsAffected` pada adapter SQL, yang melaporkan nol
// untuk UPDATE yang tidak mengubah apa pun.
func (r *Repo) SetStatus(
	_ context.Context,
	id []string,
	status masterkategorisparepart.ApprovalStatus,
) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return 0, r.failure
	}

	changed := 0
	for _, one := range id {
		index := r.indexOf(one)
		if index < 0 || r.rows[index].Status == status {
			continue
		}
		r.rows[index].Status = status
		changed++
	}
	return changed, nil
}

// indexOf mencari posisi baris menurut kuncinya; -1 bila tidak ada.
//
// Pembandingnya dipangkas di kedua sisi, meniru `TRIM(...)` pada kueri SQL.
func (r *Repo) indexOf(id string) int {
	wanted := strings.TrimSpace(id)
	for i, c := range r.rows {
		if strings.TrimSpace(c.ID) == wanted {
			return i
		}
	}
	return -1
}

var _ masterkategorisparepart.Store = (*Repo)(nil)
