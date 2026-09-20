// Package memory adalah pengisi seam mastersparepart.Store yang hidup di dalam memory.
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

	"claim-pnc/internal/mastersparepart"
)

// sequenceWidth disalin dari adapter SQL, dan harus sama dengannya.
//
// Ia TIDAK diimpor dari sana: paket memory tidak boleh bergantung pada paket sqlstore, yang
// menyeret driver basis data ke dalam uji yang justru dibuat agar tidak membutuhkannya.
// Kesamaannya dijaga uji, bukan oleh kompilator.
const sequenceWidth = 10

// Repo menyimpan master sparepart satu portal beserta kedua daftar acuannya.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah ADR-0030
// dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi. Permintaan HTTP dilayani beberapa goroutine sekaligus,
	// dan penambahan yang menolak kunci ganda harus berjalan satu per satu — persis seperti
	// transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []mastersparepart.Sparepart
	failure error

	// category dan partType meniru kedua tabel acuan. Keduanya HANYA DIBACA, sama seperti
	// di produksi — tidak ada satu pun operasi di berkas ini yang mengubahnya.
	category []mastersparepart.Category
	partType []mastersparepart.PartType

	// site dan sequence meniru kode situs dan sequence basis data. Keduanya di sini supaya
	// kunci yang diterbitkan saat pengembangan berbentuk sama dengan yang diterbitkan di
	// produksi — bukan angka berurut yang terlihat berbeda.
	site     string
	sequence int64
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows     []mastersparepart.Sparepart
	Category []mastersparepart.Category
	Type     []mastersparepart.PartType

	// Site adalah kode situs yang dipakai menerbitkan ID. Kosong berarti SampleSite.
	Site string

	// Sequence adalah nomor urut terakhir yang sudah dipakai.
	Sequence int64
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{site: strings.TrimSpace(o.Site), sequence: o.Sequence}
	if r.site == "" {
		r.site = SampleSite
	}
	r.rows = append(r.rows, o.Rows...)
	r.category = append(r.category, o.Category...)
	r.partType = append(r.partType, o.Type...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:     SampleList(),
		Category: SampleCategories(),
		Type:     SampleTypes(),
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
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada APPROVAL dan
// pencocokan kata kunci pada KETIGA kunci alami — nama, nomor, dan kode.
func (r *Repo) List(
	_ context.Context,
	filter mastersparepart.Filter,
) ([]mastersparepart.Sparepart, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []mastersparepart.Sparepart
	for _, s := range r.rows {
		if s.Status != filter.Status {
			continue
		}
		if keyword != "" && !matches(s, keyword) {
			continue
		}
		result = append(result, s)
	}

	// `ORDER BY ID` pada basis data adalah pengurutan TEKS bila kolomnya bertipe teks.
	// Ditiru apa adanya — termasuk arahnya — supaya urutan yang terlihat saat pengembangan
	// sama dengan urutan yang terlihat di produksi.
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// matches menyatakan sebuah baris cocok dengan kata kunci yang sudah di-uppercase.
//
// Ketiga kolomnya sama persis dengan yang dicari `sparepart_list_search`. Bila keduanya
// berbeda, uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
func matches(s mastersparepart.Sparepart, keyword string) bool {
	for _, value := range []string{s.Name, s.Number, s.Code} {
		if strings.Contains(strings.ToUpper(value), keyword) {
			return true
		}
	}
	return false
}

// Get mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo) Get(_ context.Context, id string) (mastersparepart.Sparepart, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastersparepart.Sparepart{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, s := range r.rows {
		if s.ID == wanted {
			return s, nil
		}
	}
	return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
}

// FindByName mencari baris menurut NAMA_SPART-nya.
//
// Pencocokannya mengabaikan besar-kecil huruf dan spasi tepi, meniru
// `upper(trim(nama_spart))` pada kueri aslinya.
func (r *Repo) FindByName(_ context.Context, name string) (mastersparepart.Sparepart, error) {
	return r.findBy(name, func(s mastersparepart.Sparepart) string { return s.Name })
}

// FindByNumber mencari baris menurut NO_SPART-nya.
func (r *Repo) FindByNumber(_ context.Context, number string) (mastersparepart.Sparepart, error) {
	return r.findBy(number, func(s mastersparepart.Sparepart) string { return s.Number })
}

// FindByCode mencari baris menurut KODE_SPART-nya.
func (r *Repo) FindByCode(_ context.Context, code string) (mastersparepart.Sparepart, error) {
	return r.findBy(code, func(s mastersparepart.Sparepart) string { return s.Code })
}

func (r *Repo) findBy(
	value string,
	read func(mastersparepart.Sparepart) string,
) (mastersparepart.Sparepart, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastersparepart.Sparepart{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(value))
	if wanted == "" {
		return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
	}
	for _, s := range r.rows {
		if strings.ToUpper(strings.TrimSpace(read(s))) == wanted {
			return s, nil
		}
	}
	return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
}

// ListCategories mengembalikan seluruh kategori contoh.
//
// Salinan, bukan slice yang sama: pemanggil yang mengurutkannya tidak boleh ikut mengubah
// isi repo. Adapter SQL selalu menyusun slice baru setiap dipanggil.
func (r *Repo) ListCategories(_ context.Context) ([]mastersparepart.Category, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastersparepart.Category, len(r.category))
	copy(result, r.category)
	return result, nil
}

// ListTypes mengembalikan seluruh tipe contoh beserta kategori induknya.
func (r *Repo) ListTypes(_ context.Context) ([]mastersparepart.PartType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastersparepart.PartType, len(r.partType))
	copy(result, r.partType)
	return result, nil
}

// Insert menolak ketiga kunci alami yang sudah dipakai, lalu menambahkan barisnya.
//
// Ketiga galat sentinel dikembalikan satu per satu — yang pertama bentrok yang menang —
// berbeda dari adapter SQL yang melaporkan seluruhnya sekaligus lewat ValidationError.
//
// Perbedaan itu DISENGAJA dan disadari: adapter SQL dapat memeriksa ketiganya dalam satu
// kueri, sedangkan di sini pemeriksaannya berurutan dan menghentikannya pada yang pertama
// membuat perilakunya lebih mudah ditebak. Yang dijaga sama adalah apa yang dilihat
// pengguna: transport memetakan keduanya ke 409 dengan pesan yang menempel pada isiannya.
func (r *Repo) Insert(_ context.Context, s mastersparepart.Sparepart) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	upper := func(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }
	name, number, code := upper(s.Name), upper(s.Number), upper(s.Code)

	for _, exists := range r.rows {
		switch {
		case number != "" && upper(exists.Number) == number:
			return mastersparepart.ErrNumberTaken
		case name != "" && upper(exists.Name) == name:
			return mastersparepart.ErrNameTaken
		case code != "" && upper(exists.Code) == code:
			return mastersparepart.ErrCodeTaken
		}
	}

	r.rows = append(r.rows, s)
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// ID SENGAJA TIDAK IKUT DITIMPA, persis seperti `sparepart_update` pada adapter SQL yang
// tidak menyebut kolom itu. Tanpa peniruan ini, uji yang membuktikan kunci tidak berpindah
// akan lulus di memori dan gagal di Oracle — atau lebih buruk, sebaliknya.
func (r *Repo) Update(_ context.Context, s mastersparepart.Sparepart) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(s.ID)
	for i, exists := range r.rows {
		if exists.ID != wanted {
			continue
		}
		kept := r.rows[i].ID
		r.rows[i] = s
		r.rows[i].ID = kept
		return nil
	}
	return mastersparepart.ErrNotFound
}

// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
//
// Baris yang sudah berstatus itu TIDAK dihitung sebagai berubah, meniru penyaring
// `AND TRIM(APPROVAL) <> :1` pada kuerinya.
//
// TIDAK ada kolom lain yang disentuh — termasuk USER_UPDATE, yang tetap berisi siapa yang
// mengajukan. Itu meniru `Activity/SetApprovalAllMaster` apa adanya.
func (r *Repo) SetStatus(
	_ context.Context,
	id []string,
	status mastersparepart.ApprovalStatus,
) (int, error) {
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

// NextID menerbitkan ID berikutnya dengan bentuk yang sama seperti adapter SQL.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	r.sequence++
	return mastersparepart.ComposeID(r.site, r.sequence, sequenceWidth), nil
}

var _ mastersparepart.Store = (*Repo)(nil)
