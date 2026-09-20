// Package memory adalah pengisi seam mastersupplier.Store yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara penyaring
// bekerja, kunci mana yang TIDAK ikut berubah saat disimpan, penurunan JENIS_STATUS saat
// membaca, dan bentuk kunci yang diterbitkan — kalau tidak, uji yang lulus di sini tidak
// membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/mastersupplier"
)

// Repo menyimpan master supplier satu portal beserta keempat tabel acuannya dan antrean
// persetujuannya.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah ADR-0030
// dan R-20.
type Repo struct {
	// mutex melindungi seluruh isi. Permintaan HTTP dilayani beberapa goroutine
	// sekaligus, dan penambahan yang menolak nama ganda harus berjalan satu per satu —
	// persis seperti transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []mastersupplier.Supplier
	failure error

	branch  []mastersupplier.Branch
	city    []mastersupplier.City
	country []mastersupplier.Country
	bank    []mastersupplier.Bank

	// approval merekam permintaan persetujuan yang tercipta.
	//
	// Ia disimpan, bukan dibuang, supaya uji dapat membuktikan barisnya benar-benar
	// terbentuk — dan terutama supaya uji dapat membuktikan ia TIDAK terbentuk saat
	// sebuah supplier dinonaktifkan, yang merupakan satu-satunya jalur di modul ini yang
	// mengubah keadaan tanpa melewati persetujuan.
	approval []mastersupplier.ApprovalRequest

	// site dan sequence meniru kode situs dan sequence basis data. Keduanya di sini
	// supaya kunci yang diterbitkan saat pengembangan berbentuk sama dengan yang
	// diterbitkan di produksi — bukan angka berurut yang terlihat berbeda.
	site     string
	sequence int64
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows      []mastersupplier.Supplier
	Branches  []mastersupplier.Branch
	Cities    []mastersupplier.City
	Countries []mastersupplier.Country
	Banks     []mastersupplier.Bank

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
	r.country = append(r.country, o.Countries...)
	r.bank = append(r.bank, o.Banks...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:      SampleList(),
		Branches:  SampleBranches(),
		Cities:    SampleCities(),
		Countries: SampleCountries(),
		Banks:     SampleBanks(),
		Site:      SampleSite,
		Sequence:  SampleSequence,
	})
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// Approval mengembalikan salinan permintaan persetujuan yang sudah tercipta.
func (r *Repo) Approval() []mastersupplier.ApprovalRequest {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]mastersupplier.ApprovalRequest{}, r.approval...)
}

// List mengembalikan baris yang cocok dengan penyaring.
//
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk ketiga kolom yang
// dicocokkan kata kunci dan urutan menurut nama.
func (r *Repo) List(_ context.Context, filter mastersupplier.Filter) ([]mastersupplier.Supplier, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var result []mastersupplier.Supplier
	for _, one := range r.rows {
		if keyword != "" && !matches(one, keyword) {
			continue
		}
		result = append(result, read(one))
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// Get mengembalikan satu baris.
func (r *Repo) Get(_ context.Context, id string) (mastersupplier.Supplier, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastersupplier.Supplier{}, r.failure
	}

	key := strings.TrimSpace(id)
	for _, one := range r.rows {
		if strings.TrimSpace(one.ID) == key {
			return read(one), nil
		}
	}
	return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
}

// FindByName mencari baris menurut namanya, tanpa memandang besar-kecil huruf.
func (r *Repo) FindByName(_ context.Context, name string) (mastersupplier.Supplier, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastersupplier.Supplier{}, r.failure
	}

	clean := strings.ToUpper(strings.TrimSpace(name))
	if clean == "" {
		return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
	}
	for _, one := range r.rows {
		if strings.ToUpper(strings.TrimSpace(one.Name)) == clean {
			return read(one), nil
		}
	}
	return mastersupplier.Supplier{}, mastersupplier.ErrNotFound
}

// Insert menyisipkan baris baru, menolak nama yang sudah dipakai.
func (r *Repo) Insert(_ context.Context, s mastersupplier.Supplier) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	clean := strings.ToUpper(strings.TrimSpace(s.Name))
	for _, one := range r.rows {
		if strings.ToUpper(strings.TrimSpace(one.Name)) == clean {
			return mastersupplier.ErrNameTaken
		}
	}

	r.rows = append(r.rows, s)
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(_ context.Context, s mastersupplier.Supplier) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	key := strings.TrimSpace(s.ID)
	for index, one := range r.rows {
		if strings.TrimSpace(one.ID) != key {
			continue
		}
		// OldID dipertahankan dari baris yang tersimpan, sama seperti adapter SQL yang
		// tidak pernah menyebut kolom itu pada pernyataan tulis mana pun.
		s.OldID = one.OldID
		r.rows[index] = s
		return nil
	}
	return mastersupplier.ErrNotFound
}

// RequestApproval merekam satu permintaan persetujuan.
func (r *Repo) RequestApproval(_ context.Context, request mastersupplier.ApprovalRequest) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	r.approval = append(r.approval, request)
	return nil
}

// NextID menerbitkan ID supplier berikutnya, berbentuk sama dengan adapter SQL.
func (r *Repo) NextID(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}

	r.sequence++
	return mastersupplier.ComposeID(r.site, r.sequence, mastersupplier.SequenceWidth), nil
}

// ListBranches mengembalikan seluruh cabang.
func (r *Repo) ListBranches(_ context.Context) ([]mastersupplier.Branch, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	return append([]mastersupplier.Branch{}, r.branch...), nil
}

// SearchCities mencari kota menurut namanya, atau menurut ID persisnya.
//
// Batas MaxLookupRows ditegakkan di sini juga, bukan hanya di adapter SQL: uji yang lulus
// terhadap repo yang tidak membatasinya tidak membuktikan apa pun tentang yang membatasi.
func (r *Repo) SearchCities(_ context.Context, keyword string) ([]mastersupplier.City, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	clean := strings.ToUpper(strings.TrimSpace(keyword))
	if len(clean) < mastersupplier.MinLookupKeyword {
		return nil, nil
	}

	var result []mastersupplier.City
	for _, one := range r.city {
		if !strings.Contains(strings.ToUpper(one.Name), clean) &&
			strings.ToUpper(strings.TrimSpace(one.ID)) != clean {
			continue
		}
		result = append(result, one)
		if len(result) == mastersupplier.MaxLookupRows {
			break
		}
	}
	return result, nil
}

// ListCountries mengembalikan seluruh negara.
func (r *Repo) ListCountries(_ context.Context) ([]mastersupplier.Country, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	return append([]mastersupplier.Country{}, r.country...), nil
}

// ListBanks mengembalikan seluruh bank.
func (r *Repo) ListBanks(_ context.Context) ([]mastersupplier.Bank, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	return append([]mastersupplier.Bank{}, r.bank...), nil
}

// ListCodes mengumpulkan sandi yang benar-benar dipakai baris yang ada.
//
// Sama seperti adapter SQL, ia TIDAK menggabungkannya dengan sandi yang artinya terbukti
// — itu urusan lapisan aplikasi. Labelnya pun diisi sandinya sendiri, bukan tebakan
// artinya.
func (r *Repo) ListCodes(_ context.Context) (mastersupplier.CodeSet, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastersupplier.CodeSet{}, r.failure
	}

	var result mastersupplier.CodeSet
	for _, one := range r.rows {
		result.PartnerStatus = addCode(result.PartnerStatus, one.PartnerStatus)
		result.SupplyType = addCode(result.SupplyType, one.SupplyType)
		result.SupplierType = addCode(result.SupplierType, one.SupplierType)
		result.Active = addCode(result.Active, one.ActiveRequested)
		result.AutoPayment = addCode(result.AutoPayment, one.AutoPayment)
	}
	return result, nil
}

// read menyiapkan sebuah baris untuk dibaca pemanggil.
//
// Ia menurunkan JENIS_STATUS dari SUPPLIER_HE persis seperti scanRow pada adapter SQL,
// yang keduanya meniru `Activity/GetDataSupplier_pre` step 6.3. Tanpa ini, uji yang lulus
// terhadap repo memori tidak membuktikan apa pun tentang perilaku yang sebenarnya.
func read(s mastersupplier.Supplier) mastersupplier.Supplier {
	s.SupplyType = mastersupplier.DeriveSupplyType(s.HeavyEquipment)
	return s
}

// matches menyatakan sebuah baris cocok dengan kata kunci.
//
// Ketiga kolomnya sama dengan `supplier_list_search` pada adapter SQL: nama, kota, dan
// contact person.
func matches(s mastersupplier.Supplier, keyword string) bool {
	for _, field := range []string{s.Name, s.City, s.ContactPerson} {
		if strings.Contains(strings.ToUpper(field), keyword) {
			return true
		}
	}
	return false
}

// addCode menambahkan satu sandi bila belum ada dan tidak kosong.
func addCode(list []mastersupplier.CodeOption, value string) []mastersupplier.CodeOption {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return list
	}
	for _, one := range list {
		if one.Value == clean {
			return list
		}
	}
	return append(list, mastersupplier.CodeOption{Value: clean, Label: clean})
}
