// Package memory adalah pengisi seam masterautoclaim.Store yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi dijalankan
// tanpa Oracle saat pengembangan, mengikuti pola modul auth, portal, dan master lainnya.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara penyaring
// bekerja, dan kolom mana yang TIDAK ikut berubah saat disimpan — kalau tidak, uji yang
// lulus di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterautoclaim"
)

// Repo menyimpan master auto claim satu portal beserta keempat tabel acuannya.
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
	rows    []masterautoclaim.AutoClaim
	failure error

	source    []masterautoclaim.BusinessSource
	client    []masterautoclaim.Client
	bank      []masterautoclaim.Bank
	committee string
}

// Options adalah isi awal repo memori.
type Options struct {
	Rows      []masterautoclaim.AutoClaim
	Sources   []masterautoclaim.BusinessSource
	Clients   []masterautoclaim.Client
	Banks     []masterautoclaim.Bank
	Committee string
}

// NewRepo membentuk repo berisi baris dan acuan yang diberikan.
func NewRepo(o Options) *Repo {
	r := &Repo{committee: o.Committee}
	r.rows = append(r.rows, o.Rows...)
	r.source = append(r.source, o.Sources...)
	r.client = append(r.client, o.Clients...)
	r.bank = append(r.bank, o.Banks...)
	return r
}

// NewSampleRepo membentuk repo berisi contoh lengkap untuk pengembangan.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Rows:      SampleList(),
		Sources:   SampleBusinessSources(),
		Clients:   SampleClients(),
		Banks:     SampleBanks(),
		Committee: SampleCommittee,
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
// Penyaringnya ditiru persis dari kedua kueri adapter SQL, termasuk TRIM pada kolom
// KOMITE: baris yang nilainya dipadatkan spasi harus tetap cocok di sini, karena di
// basis data pun ia cocok.
func (r *Repo) List(_ context.Context, filter masterautoclaim.Filter) ([]masterautoclaim.AutoClaim, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	var result []masterautoclaim.AutoClaim
	for _, ac := range r.rows {
		if ac.Status != filter.Status {
			continue
		}
		if filter.CommitteeOnly && !strings.EqualFold(strings.TrimSpace(ac.Committee), strings.TrimSpace(filter.CommitteeID)) {
			continue
		}
		result = append(result, ac)
	}

	// `ORDER BY INISIALID` pada basis data adalah pengurutan TEKS bila kolomnya bertipe
	// teks. Ditiru apa adanya supaya urutan yang terlihat saat pengembangan sama dengan
	// urutan yang terlihat di produksi.
	sort.SliceStable(result, func(i, j int) bool { return result[i].Initial < result[j].Initial })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan INISIALID-nya.
func (r *Repo) Get(_ context.Context, initial string) (masterautoclaim.AutoClaim, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterautoclaim.AutoClaim{}, r.failure
	}

	wanted := strings.TrimSpace(initial)
	for _, ac := range r.rows {
		if ac.Initial == wanted {
			return ac, nil
		}
	}
	return masterautoclaim.AutoClaim{}, masterautoclaim.ErrNotFound
}

// Insert menolak INISIALID yang sudah dipakai, lalu menambahkan barisnya.
func (r *Repo) Insert(_ context.Context, ac masterautoclaim.AutoClaim) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(ac.Initial)
	for _, exists := range r.rows {
		if exists.Initial == wanted {
			return masterautoclaim.ErrInitialTaken
		}
	}
	r.rows = append(r.rows, ac)
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// NAMA_PENERIMA SENGAJA TIDAK IKUT DITIMPA, persis seperti `auto_claim_update` pada
// adapter SQL yang tidak menyebut kolom itu. Tanpa peniruan ini, uji yang membuktikan
// nama penerima tidak berubah akan lulus di memori dan gagal di Oracle — atau lebih
// buruk, sebaliknya.
func (r *Repo) Update(_ context.Context, ac masterautoclaim.AutoClaim) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	wanted := strings.TrimSpace(ac.Initial)
	for i, exists := range r.rows {
		if exists.Initial != wanted {
			continue
		}
		r.rows[i].BankName = ac.BankName
		r.rows[i].AccountNumber = ac.AccountNumber
		r.rows[i].MaxPercent = ac.MaxPercent
		r.rows[i].ReporterPIC = ac.ReporterPIC
		r.rows[i].ReporterEmail = ac.ReporterEmail
		r.rows[i].ClaimAllowed = ac.ClaimAllowed
		r.rows[i].ReceiverAddress = ac.ReceiverAddress
		r.rows[i].Status = ac.Status
		r.rows[i].SubmittedBy = ac.SubmittedBy
		r.rows[i].Committee = ac.Committee
		r.rows[i].ClientID = ac.ClientID
		r.rows[i].ClientName = ac.ClientName
		return nil
	}
	return masterautoclaim.ErrNotFound
}

// SearchBusinessSources mencari Sumber Bisnis pada daftar contoh.
func (r *Repo) SearchBusinessSources(_ context.Context, keyword string) ([]masterautoclaim.BusinessSource, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	clean := normalise(keyword)
	var result []masterautoclaim.BusinessSource
	for _, s := range r.source {
		if !matches(s.Name, s.ID, clean) {
			continue
		}
		result = append(result, s)
		if len(result) == masterautoclaim.MaxLookupRows {
			break
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// SearchClients mencari tertanggung pada daftar contoh.
func (r *Repo) SearchClients(_ context.Context, keyword string) ([]masterautoclaim.Client, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	clean := normalise(keyword)
	var result []masterautoclaim.Client
	for _, c := range r.client {
		if !matches(c.Name, c.ID, clean) {
			continue
		}
		result = append(result, c)
		if len(result) == masterautoclaim.MaxLookupRows {
			break
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListBanks mengembalikan seluruh bank contoh.
func (r *Repo) ListBanks(_ context.Context) ([]masterautoclaim.Bank, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterautoclaim.Bank, len(r.bank))
	copy(result, r.bank)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// FindBankByName mencari satu bank menurut namanya.
func (r *Repo) FindBankByName(_ context.Context, name string) (masterautoclaim.Bank, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterautoclaim.Bank{}, r.failure
	}

	wanted := strings.ToUpper(strings.TrimSpace(name))
	if wanted == "" {
		return masterautoclaim.Bank{}, masterautoclaim.ErrBankNotFound
	}
	for _, b := range r.bank {
		if strings.ToUpper(strings.TrimSpace(b.Name)) == wanted {
			return b, nil
		}
	}
	return masterautoclaim.Bank{}, masterautoclaim.ErrBankNotFound
}

// Committee mengembalikan penyetuju komite contoh.
//
// Teks kosong bukan galat, sama seperti adapter SQL: bila tidak ada baris komite yang
// aktif, KOMITE tersimpan kosong. Repo memori dapat dibuat tanpa komite justru supaya
// jalur itu dapat diuji.
func (r *Repo) Committee(_ context.Context) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", r.failure
	}
	return r.committee, nil
}

// normalise menyamakan perlakuan kata kunci dengan adapter SQL: huruf besar, tanpa titik.
func normalise(keyword string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(keyword), ".", ""))
}

// matches meniru penyaring kedua kueri lookup: nama yang titiknya dibuang lalu
// dicocokkan sebagian, ATAU ID yang cocok persis.
//
// Bentuk SQL-nya tidak dikutip di sini karena dua tanda petik tunggal berurutan pada
// komentar Go diubah gofmt menjadi tanda kutip tipografis, sehingga cuplikannya tidak
// lagi sama dengan kueri yang sebenarnya. Kuerinya ada di berkas .sql.
func matches(name, id, keyword string) bool {
	if keyword == "" {
		return false
	}
	if strings.Contains(normalise(name), keyword) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(id), keyword)
}

var _ masterautoclaim.Store = (*Repo)(nil)
