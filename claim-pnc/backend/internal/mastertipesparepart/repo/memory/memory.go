// Package memory adalah pengisi seam mastertipesparepart.Store yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga URUTAN barisnya, cara penyaring
// bekerja, cakupan pemeriksaan nama ganda, bentuk kunci yang diterbitkan, dan **perilaku
// LEFT JOIN** — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/mastertipesparepart"
)

// Repo menyimpan master tipe sparepart satu portal, beserta acuan kategorinya.
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
	mutex sync.Mutex
	rows  []mastertipesparepart.PartType

	// category adalah acuan yang di basis data dimiliki modul masterkategorisparepart.
	//
	// Ia disimpan di sini karena modus memori tidak punya basis data bersama: setiap modul
	// memegang salinannya sendiri. Akibatnya kategori yang ditambahkan di layar Master
	// Kategori Sparepart TIDAK muncul di dropdown layar ini saat berjalan tanpa Oracle.
	// Terhadap Oracle keduanya membaca tabel yang sama dan tautannya bekerja — keterbatasan
	// modus memori, bukan cacat modul. Catatan yang sama ada pada sample.go.
	category []mastertipesparepart.Category

	failure error
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows []mastertipesparepart.PartType

	// Category adalah kategori yang dianggap SUDAH DISETUJUI.
	//
	// Hanya yang disetujui yang disimpan di sini, meniru `type_category_list` yang menyaring
	// APPROVAL = '1' di sisi basis data. Menyimpan seluruh status lalu menyaringnya di sini
	// akan memindahkan penyaring dari tempat yang benar ke tempat yang kebetulan mudah.
	Category []mastertipesparepart.Category
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{}
	r.rows = append(r.rows, o.Rows...)
	r.category = append(r.category, o.Category...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{Rows: SampleList(), Category: SampleCategoryList()})
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan baris yang cocok dengan penyaring, beserta nama kategori induknya.
//
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada APPROVAL dan
// pencarian yang menyentuh nama tipe SEKALIGUS nama kategori.
func (r *Repo) List(
	_ context.Context,
	filter mastertipesparepart.Filter,
) ([]mastertipesparepart.PartType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []mastertipesparepart.PartType
	for _, t := range r.rows {
		if strings.TrimSpace(string(t.Status)) != string(filter.Status) {
			continue
		}
		// Nama kategori diisi di sini, bukan disimpan pada barisnya — meniru LEFT JOIN pada
		// adapter SQL. Kategori yang berganti nama karena itu langsung terlihat baru di
		// seluruh barisnya, dan kategori yang tidak ada meninggalkan nama kosong alih-alih
		// membuang barisnya.
		joined := t
		joined.CategoryName = r.categoryNameOf(t.CategoryID)

		if keyword != "" &&
			!strings.Contains(strings.ToUpper(joined.Name), keyword) &&
			!strings.Contains(strings.ToUpper(joined.CategoryName), keyword) {
			continue
		}
		result = append(result, joined)
	}

	sortByID(result)
	return result, nil
}

// categoryNameOf mencari nama kategori menurut kuncinya; kosong bila tidak ada.
//
// Kosong BUKAN galat: ia padanan sisi kanan LEFT JOIN yang tidak menemukan pasangan. Lihat
// banner pada berkas .sql.
//
// Pemanggil WAJIB sudah memegang mutex.
func (r *Repo) categoryNameOf(id string) string {
	wanted := strings.TrimSpace(id)
	if wanted == "" {
		return ""
	}
	for _, c := range r.category {
		if strings.TrimSpace(c.ID) == wanted {
			return c.Name
		}
	}
	return ""
}

// sortByID mengurutkan seperti `ORDER BY PART_SECTION_ID` pada kolom bertipe ANGKA.
//
// Bukan pengurutan teks — dan itu berbeda dari Master Sparepart, yang ID-nya memang teks
// sehingga "10" berada sebelum "9" di sana. Di sini kolomnya angka (lihat banner berkas
// .sql), jadi 9 mendahului 10.
//
// Kunci yang tidak dapat dibaca sebagai angka ditaruh di BELAKANG seluruh yang dapat, lalu
// diurutkan sebagai teks di antara sesamanya. Ia keadaan yang tidak seharusnya ada, dan
// menaruhnya di belakang membuatnya terlihat alih-alih tersebar di tengah daftar.
func sortByID(list []mastertipesparepart.PartType) {
	sort.SliceStable(list, func(i, j int) bool {
		left, leftErr := strconv.ParseInt(strings.TrimSpace(list[i].ID), 10, 64)
		right, rightErr := strconv.ParseInt(strings.TrimSpace(list[j].ID), 10, 64)
		switch {
		case leftErr == nil && rightErr == nil:
			return left < right
		case leftErr == nil:
			return true
		case rightErr == nil:
			return false
		default:
			return list[i].ID < list[j].ID
		}
	})
}

// Get mengembalikan satu baris berdasarkan kuncinya, beserta nama kategori induknya.
func (r *Repo) Get(
	_ context.Context,
	id string,
) (mastertipesparepart.PartType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastertipesparepart.PartType{}, r.failure
	}

	index := r.indexOf(id)
	if index < 0 {
		return mastertipesparepart.PartType{}, mastertipesparepart.ErrNotFound
	}

	found := r.rows[index]
	found.CategoryName = r.categoryNameOf(found.CategoryID)
	return found, nil
}

// FindByName mencari baris menurut namanya.
//
// TIDAK menyaring status dan TIDAK menyaring kategori, meniru `ValidationSparepartType` apa
// adanya. Bila dua baris bernama sama — keadaan yang mungkin ada pada data lama karena
// tidak ada constraint unik (R-08) — yang dikembalikan adalah yang ID-nya terkecil, sama
// seperti kueri SQL yang memakai `ORDER BY PART_SECTION_ID FETCH FIRST 1 ROW ONLY`.
//
// Nama kategori TIDAK diisi, meniru `type_find_by_name` yang memang tidak ber-JOIN.
func (r *Repo) FindByName(
	_ context.Context,
	name string,
) (mastertipesparepart.PartType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastertipesparepart.PartType{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))

	var found []mastertipesparepart.PartType
	for _, t := range r.rows {
		if strings.ToUpper(strings.TrimSpace(t.Name)) == wanted {
			one := t
			one.CategoryName = ""
			found = append(found, one)
		}
	}
	if len(found) == 0 {
		return mastertipesparepart.PartType{}, mastertipesparepart.ErrNotFound
	}

	sortByID(found)
	return found[0], nil
}

// ListCategories mengembalikan kategori yang dianggap sudah disetujui.
//
// Urutannya menurut NAMA, meniru `ORDER BY PART_CATEGORY_NAME` pada adapter SQL — itulah
// urutan yang dipindai mata pengguna pada dropdown.
//
// Batas MaxLookupRows ikut ditiru supaya kedua adapter memotong pada titik yang sama.
func (r *Repo) ListCategories(
	_ context.Context,
) ([]mastertipesparepart.Category, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastertipesparepart.Category, 0, len(r.category))
	result = append(result, r.category...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })

	if len(result) > mastertipesparepart.MaxLookupRows {
		result = result[:mastertipesparepart.MaxLookupRows]
	}
	return result, nil
}

// Insert menerbitkan ID, memeriksa keunikan nama, lalu menyisipkan.
//
// Ketiganya di bawah satu mutex, meniru transaksi bertahan kunci tabel pada adapter SQL.
// Urutan langkahnya pun sama: nama diperiksa lebih dulu, supaya nomor urut tidak terpakai
// oleh percobaan yang memang akan ditolak.
//
// Keberadaan KATEGORI tidak diperiksa di sini, sama seperti adapter SQL: pemeriksaannya ada
// di lapisan aplikasi.
//
// ID pada argumen DIABAIKAN, sama seperti pada adapter SQL. CategoryName juga tidak
// disimpan — ia milik tabel kategori.
func (r *Repo) Insert(
	_ context.Context,
	t mastertipesparepart.PartType,
) (mastertipesparepart.PartType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastertipesparepart.PartType{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(t.Name))
	for _, existing := range r.rows {
		if strings.ToUpper(strings.TrimSpace(existing.Name)) == wanted {
			return mastertipesparepart.PartType{}, mastertipesparepart.ErrNameTaken
		}
	}

	fresh := mastertipesparepart.PartType{
		ID:         r.nextID(),
		Name:       t.Name,
		CategoryID: t.CategoryID,
		Status:     t.Status,
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// nextID meniru `COALESCE(MAX(PART_SECTION_ID), 0) + 1`.
//
// Kunci yang tidak dapat dibaca sebagai angka DILEWATI, bukan membuat penerbitan gagal —
// sama seperti `MAX` pada basis data yang mengabaikan nilai yang tidak ikut terhitung.
//
// Pemanggil WAJIB sudah memegang mutex.
func (r *Repo) nextID() string {
	var highest int64
	for _, t := range r.rows {
		number, err := strconv.ParseInt(strings.TrimSpace(t.ID), 10, 64)
		if err == nil && number > highest {
			highest = number
		}
	}
	return strconv.FormatInt(highest+1, 10)
}

// NextID menerbitkan ID berikutnya tanpa menyisipkan apa pun.
//
// Dipakai pemeriksaan, bukan jalur simpan; lihat mastertipesparepart.IDSource.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}
	return r.nextID(), nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Hanya NAMA, KATEGORI, dan STATUS yang ditulis, sama persis dengan `type_update`. Kuncinya
// tidak pernah berubah, dan nama kategori tidak pernah disimpan.
func (r *Repo) Update(_ context.Context, t mastertipesparepart.PartType) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	index := r.indexOf(t.ID)
	if index < 0 {
		return mastertipesparepart.ErrNotFound
	}

	r.rows[index].Name = t.Name
	r.rows[index].CategoryID = t.CategoryID
	r.rows[index].Status = t.Status
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
	status mastertipesparepart.ApprovalStatus,
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
//
// Pemanggil WAJIB sudah memegang mutex.
func (r *Repo) indexOf(id string) int {
	wanted := strings.TrimSpace(id)
	for i, t := range r.rows {
		if strings.TrimSpace(t.ID) == wanted {
			return i
		}
	}
	return -1
}

var _ mastertipesparepart.Store = (*Repo)(nil)
