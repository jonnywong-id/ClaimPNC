// Package memory adalah pengisi seam mastergroupingsparepart.Store yang hidup di dalam
// memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang membuat
// seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan tanpa Oracle
// saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara penyaring bekerja,
// kolom mana yang TIDAK ikut berubah saat disimpan, dan bentuk kunci yang diterbitkan — kalau
// tidak, uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/mastergroupingsparepart"
)

// Repo menyimpan grouping satu portal beserta keempat daftar acuannya.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data. Menyatukannya
// justru akan menyembunyikan kelas cacat yang paling ingin dicegah ADR-0030 dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penambahan yang menolak kunci ganda harus berjalan satu per satu — persis seperti
	// transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []mastergroupingsparepart.Grouping
	failure error

	// Keempat sumber acuan. Seluruhnya HANYA DIBACA, sama seperti di produksi — tidak ada satu
	// pun operasi di berkas ini yang mengubahnya.
	panel       []mastergroupingsparepart.Panel
	side        map[string][]mastergroupingsparepart.Side
	vehicleType []mastergroupingsparepart.VehicleType
	part        []mastergroupingsparepart.PartRef

	// sequence meniru pencacah `MAX(ID)+1` pada basis data. Ia di sini supaya kunci yang
	// diterbitkan saat pengembangan berbentuk sama dengan yang diterbitkan di produksi.
	sequence int64
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows        []mastergroupingsparepart.Grouping
	Panel       []mastergroupingsparepart.Panel
	Side        map[string][]mastergroupingsparepart.Side
	VehicleType []mastergroupingsparepart.VehicleType
	Part        []mastergroupingsparepart.PartRef

	// Sequence adalah nomor ID terakhir yang sudah dipakai.
	Sequence int64
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{sequence: o.Sequence, side: map[string][]mastergroupingsparepart.Side{}}
	r.rows = append(r.rows, o.Rows...)
	r.panel = append(r.panel, o.Panel...)
	r.vehicleType = append(r.vehicleType, o.VehicleType...)
	r.part = append(r.part, o.Part...)
	for id, list := range o.Side {
		r.side[id] = append([]mastergroupingsparepart.Side(nil), list...)
	}
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:        SampleList(),
		Panel:       SamplePanels(),
		Side:        SampleSides(),
		VehicleType: SampleVehicleTypes(),
		Part:        SampleParts(),
		Sequence:    SampleSequence,
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
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk pencocokan kata kunci pada
// KEEMPAT kolom — nomor sparepart, nama sparepart, nama panel, dan nomor rangka.
func (r *Repo) List(
	_ context.Context,
	filter mastergroupingsparepart.Filter,
) ([]mastergroupingsparepart.Grouping, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []mastergroupingsparepart.Grouping
	for _, g := range r.rows {
		if g.Status != filter.Status {
			continue
		}
		if keyword != "" && !matches(g, keyword) {
			continue
		}
		result = append(result, g)
	}

	// `ORDER BY A.ID` pada basis data adalah pengurutan TEKS bila kolomnya bertipe teks.
	// Ditiru apa adanya supaya urutan yang terlihat saat pengembangan sama dengan urutan yang
	// terlihat di produksi.
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// matches menyatakan sebuah baris cocok dengan kata kunci yang sudah di-uppercase.
//
// Keempat kolomnya sama persis dengan yang dicari `grouping_list_search`. Bila keduanya
// berbeda, uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
func matches(g mastergroupingsparepart.Grouping, keyword string) bool {
	for _, value := range []string{g.PartNumber, g.PartName, g.PanelName, g.ChassisNumber} {
		if strings.Contains(strings.ToUpper(value), keyword) {
			return true
		}
	}
	return false
}

// Get mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo) Get(
	_ context.Context,
	id string,
) (mastergroupingsparepart.Grouping, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastergroupingsparepart.Grouping{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, g := range r.rows {
		if g.ID == wanted {
			return g, nil
		}
	}
	return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
}

// FindByKey mencari baris menurut keempat kunci alaminya.
//
// Pencocokannya mengabaikan besar-kecil huruf dan spasi tepi, meniru `UPPER(TRIM(...))` pada
// kueri aslinya.
func (r *Repo) FindByKey(
	_ context.Context,
	key mastergroupingsparepart.NaturalKey,
) (mastergroupingsparepart.Grouping, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastergroupingsparepart.Grouping{}, r.failure
	}

	wanted := normaliseKey(key)
	if wanted == (mastergroupingsparepart.NaturalKey{}) {
		return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
	}

	for _, g := range r.rows {
		if normaliseKey(mastergroupingsparepart.KeyOf(g)) == wanted {
			return g, nil
		}
	}
	return mastergroupingsparepart.Grouping{}, mastergroupingsparepart.ErrNotFound
}

// normaliseKey menyiapkan kunci alami untuk dibandingkan.
func normaliseKey(key mastergroupingsparepart.NaturalKey) mastergroupingsparepart.NaturalKey {
	upper := func(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }
	return mastergroupingsparepart.NaturalKey{
		PartNumber:    upper(key.PartNumber),
		PanelName:     upper(key.PanelName),
		ChassisNumber: upper(key.ChassisNumber),
		PanelSide:     upper(key.PanelSide),
	}
}

// FindGroupByChassis mengembalikan nomor grup yang dipakai sebuah nomor rangka.
//
// Bila nomor rangkanya terdaftar pada beberapa nomor grup, yang menang adalah nomor grup
// TERKECIL — meniru `ORDER BY NO_GROUP_RANGKA` ditambah pengambilan satu baris pada kueri
// adapter SQL.
func (r *Repo) FindGroupByChassis(_ context.Context, chassis string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(chassis))
	if wanted == "" {
		return "", mastergroupingsparepart.ErrGroupChassisNotFound
	}

	found := ""
	for _, g := range r.rows {
		group := strings.TrimSpace(g.GroupNumber)
		if group == "" || strings.ToUpper(strings.TrimSpace(g.ChassisNumber)) != wanted {
			continue
		}
		if found == "" || group < found {
			found = group
		}
	}
	if found == "" {
		return "", mastergroupingsparepart.ErrGroupChassisNotFound
	}
	return found, nil
}

// ListPanels mengembalikan seluruh panel contoh.
//
// Salinan, bukan slice yang sama: pemanggil yang mengurutkannya tidak boleh ikut mengubah isi
// repo. Adapter SQL selalu menyusun slice baru setiap dipanggil.
func (r *Repo) ListPanels(_ context.Context) ([]mastergroupingsparepart.Panel, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastergroupingsparepart.Panel, len(r.panel))
	copy(result, r.panel)
	return result, nil
}

// ListSides mengembalikan sandi sisi yang tersedia pada sebuah panel.
//
// Penyaringnya ID_PANEL saja; lihat catatan pada SampleSides. Panel yang tidak dikenal
// mengembalikan daftar KOSONG, bukan galat — sama seperti adapter SQL yang mengembalikan nol
// baris.
func (r *Repo) ListSides(
	_ context.Context,
	key mastergroupingsparepart.SideKey,
) ([]mastergroupingsparepart.Side, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	list := r.side[strings.TrimSpace(key.PanelID)]
	result := make([]mastergroupingsparepart.Side, len(list))
	copy(result, list)
	return result, nil
}

// ListVehicleTypes mengembalikan seluruh tipe kendaraan contoh.
func (r *Repo) ListVehicleTypes(
	_ context.Context,
) ([]mastergroupingsparepart.VehicleType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastergroupingsparepart.VehicleType, len(r.vehicleType))
	copy(result, r.vehicleType)
	return result, nil
}

// FindPart mencari satu sparepart menurut nomornya.
func (r *Repo) FindPart(
	_ context.Context,
	number string,
) (mastergroupingsparepart.PartRef, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastergroupingsparepart.PartRef{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(number))
	if wanted == "" {
		return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.ErrPartNotFound
	}
	for _, p := range r.part {
		if strings.ToUpper(strings.TrimSpace(p.Number)) == wanted {
			return p, nil
		}
	}
	return mastergroupingsparepart.PartRef{}, mastergroupingsparepart.ErrPartNotFound
}

// Insert menolak kunci alami yang sudah dipakai, lalu menambahkan barisnya.
func (r *Repo) Insert(_ context.Context, g mastergroupingsparepart.Grouping) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := normaliseKey(mastergroupingsparepart.KeyOf(g))
	if wanted != (mastergroupingsparepart.NaturalKey{}) {
		for _, exists := range r.rows {
			if normaliseKey(mastergroupingsparepart.KeyOf(exists)) == wanted {
				return mastergroupingsparepart.ErrDuplicate
			}
		}
	}

	r.rows = append(r.rows, g)
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// ID SENGAJA TIDAK IKUT DITIMPA, persis seperti `grouping_update` pada adapter SQL yang tidak
// menyebut kolom itu. Tanpa peniruan ini, uji yang membuktikan kunci tidak berpindah akan
// lulus di memori dan gagal di Oracle — atau lebih buruk, sebaliknya.
func (r *Repo) Update(_ context.Context, g mastergroupingsparepart.Grouping) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(g.ID)
	for i, exists := range r.rows {
		if exists.ID != wanted {
			continue
		}
		kept := r.rows[i].ID
		r.rows[i] = g
		r.rows[i].ID = kept
		return nil
	}
	return mastergroupingsparepart.ErrNotFound
}

// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
//
// Baris yang sudah berstatus itu TIDAK dihitung sebagai berubah, meniru penyaring
// `AND TRIM(APPROVAL) <> :1` pada kuerinya.
//
// TIDAK ada kolom lain yang disentuh.
func (r *Repo) SetStatus(
	_ context.Context,
	id []string,
	status mastergroupingsparepart.ApprovalStatus,
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
//
// Angka polos, tanpa kode situs dan tanpa pengisian nol.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	r.sequence++
	return strconv.FormatInt(r.sequence, 10), nil
}

// NextGroupNumber menerbitkan nomor grup berikutnya.
//
// Nomor terbesarnya dihitung SECARA ANGKA atas seluruh baris yang ada, persis seperti adapter
// SQL — bukan lewat perbandingan teks yang pada bentuk berawalan nol menghasilkan nomor grup
// ganda. Nilai yang tidak dapat diurai dilewati, dengan alasan yang sama.
func (r *Repo) NextGroupNumber(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	var highest int64
	for _, g := range r.rows {
		number, err := mastergroupingsparepart.ParseGroupNumber(g.GroupNumber)
		if err != nil {
			continue
		}
		if number > highest {
			highest = number
		}
	}
	return mastergroupingsparepart.ComposeGroupNumber(highest + 1), nil
}

var _ mastergroupingsparepart.Store = (*Repo)(nil)
