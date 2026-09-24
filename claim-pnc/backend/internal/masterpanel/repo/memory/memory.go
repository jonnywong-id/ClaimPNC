// Package memory adalah pengisi seam masterpanel.Store yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara penyaring
// bekerja, kolom mana yang TIDAK ikut berubah saat disimpan, bentuk kunci yang
// diterbitkan, dan cara baris ANAK diganti seluruhnya saat penyimpanan — kalau tidak, uji
// yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterpanel"
)

// sequenceWidth disalin dari adapter SQL, dan harus sama dengannya.
//
// Ia TIDAK diimpor dari sana: paket memory tidak boleh bergantung pada paket sqlstore,
// yang menyeret driver basis data ke dalam uji yang justru dibuat agar tidak
// membutuhkannya. Kesamaannya dijaga uji, bukan oleh kompilator.
const sequenceWidth = 6

// Repo menyimpan master panel satu portal beserta baris anaknya.
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
	rows    []masterpanel.Panel
	failure error

	// site dan sequence meniru kode situs dan sequence basis data. Keduanya di sini supaya
	// kunci yang diterbitkan saat pengembangan berbentuk sama dengan yang diterbitkan di
	// produksi — bukan angka berurut yang terlihat berbeda.
	site     string
	sequence int64
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows []masterpanel.Panel

	// Site adalah kode situs yang dipakai menerbitkan ID. Kosong berarti SamplePanelSite.
	Site string

	// Sequence adalah nomor urut terakhir yang sudah dipakai.
	Sequence int64
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{site: strings.TrimSpace(o.Site), sequence: o.Sequence}
	if r.site == "" {
		r.site = SamplePanelSite
	}
	for _, one := range o.Rows {
		r.rows = append(r.rows, clone(one))
	}
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:     SampleList(),
		Site:     SamplePanelSite,
		Sequence: SamplePanelSequence,
	})
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan baris yang cocok dengan penyaring, lengkap dengan lokasinya.
//
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada APPROVAL
// dan pencocokan kata kunci pada nama panel saja.
func (r *Repo) List(_ context.Context, filter masterpanel.Filter) ([]masterpanel.Panel, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []masterpanel.Panel
	for _, p := range r.rows {
		if p.Status != filter.Status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToUpper(p.Name), keyword) {
			continue
		}
		result = append(result, clone(p))
	}

	// `ORDER BY ID_PANEL DESC` pada basis data adalah pengurutan TEKS bila kolomnya
	// bertipe teks. Ditiru apa adanya — termasuk arahnya — supaya urutan yang terlihat
	// saat pengembangan sama dengan urutan yang terlihat di produksi, dan sama dengan
	// urutan yang dilihat petugas di Pega hari ini.
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan ID_PANEL-nya, beserta lokasinya.
func (r *Repo) Get(_ context.Context, id string) (masterpanel.Panel, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpanel.Panel{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, p := range r.rows {
		if p.ID == wanted {
			return clone(p), nil
		}
	}
	return masterpanel.Panel{}, masterpanel.ErrNotFound
}

// FindByName mencari baris menurut NAME-nya.
//
// Pencocokannya mengabaikan besar-kecil huruf dan spasi tepi, meniru
// `upper(trim(name))` pada kueri aslinya.
//
// Lokasinya TIDAK ikut dikembalikan, meniru adapter SQL yang tidak membacanya pada jalur
// ini. Tanpa peniruan itu, uji yang memeriksa isi jawabannya akan lulus di memori dan
// gagal di Oracle.
func (r *Repo) FindByName(_ context.Context, name string) (masterpanel.Panel, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpanel.Panel{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))
	if wanted == "" {
		return masterpanel.Panel{}, masterpanel.ErrNotFound
	}
	for _, p := range r.rows {
		if strings.ToUpper(strings.TrimSpace(p.Name)) == wanted {
			found := clone(p)
			found.Location = nil
			return found, nil
		}
	}
	return masterpanel.Panel{}, masterpanel.ErrNotFound
}

// Insert menolak nama yang sudah dipakai, lalu menambahkan barisnya beserta lokasinya.
func (r *Repo) Insert(_ context.Context, p masterpanel.Panel) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	name := strings.ToUpper(strings.TrimSpace(p.Name))
	for _, exists := range r.rows {
		if strings.ToUpper(strings.TrimSpace(exists.Name)) == name {
			return masterpanel.ErrNameTaken
		}
	}
	r.rows = append(r.rows, clone(p))
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// ID_PANEL SENGAJA TIDAK IKUT DITIMPA, persis seperti `panel_update` pada adapter SQL
// yang tidak menyebut kolom itu. Tanpa peniruan ini, uji yang membuktikan kunci tidak
// berpindah akan lulus di memori dan gagal di Oracle — atau lebih buruk, sebaliknya.
//
// Lokasinya DIGANTI seluruhnya, meniru `panel_location_clear` yang diikuti penyisipan
// ulang.
func (r *Repo) Update(_ context.Context, p masterpanel.Panel) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(p.ID)
	for i, exists := range r.rows {
		if exists.ID != wanted {
			continue
		}
		kept := r.rows[i].ID
		r.rows[i] = clone(p)
		r.rows[i].ID = kept
		return nil
	}
	return masterpanel.ErrNotFound
}

// SetStatus menetapkan APPROVAL sejumlah baris sekaligus, beserta alasannya.
//
// Baris yang sudah berstatus itu TIDAK dihitung sebagai berubah, meniru penyaring
// `AND TRIM(APPROVAL) <> :1` pada kuerinya.
//
// Alasan ditulis pada setiap baris yang berubah — termasuk ketika ia kosong, persis
// seperti `panel_set_status` yang selalu menyebut kolom itu. Pemanggil yang memutuskan
// kapan ia kosong; lihat usecase.Service.Decide.
func (r *Repo) SetStatus(
	_ context.Context,
	id []string,
	status masterpanel.ApprovalStatus,
	reason string,
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
			r.rows[i].RejectReason = reason
			changed++
		}
	}
	return changed, nil
}

// NextID menerbitkan ID_PANEL berikutnya dengan bentuk yang sama seperti adapter SQL.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	r.sequence++
	return masterpanel.ComposeID(r.site, r.sequence, sequenceWidth), nil
}

// clone menyalin satu panel beserta daftar lokasinya.
//
// Tanpa ini, pemanggil yang menyimpan hasil List lalu mengubah salah satu lokasinya akan
// ikut mengubah isi repo — dan adapter SQL tidak berperilaku begitu. Slice adalah
// referensi; menyalin struct saja tidak cukup.
func clone(p masterpanel.Panel) masterpanel.Panel {
	if p.Location == nil {
		return p
	}
	copied := make([]masterpanel.PanelLocation, len(p.Location))
	copy(copied, p.Location)
	p.Location = copied
	return p
}

var _ masterpanel.Store = (*Repo)(nil)
