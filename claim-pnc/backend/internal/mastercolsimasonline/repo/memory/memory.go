// Package memory adalah pengisi seam mastercolsimasonline.Repo dan BusinessRepo yang
// hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat kedua seam itu nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara kode baru
// diterbitkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/mastercolsimasonline"
)

// defaultSiteCode meniru POOLDATA.M_SITE_DATABASE.ID pada situs yang aktif.
//
// Nilainya "1" karena itulah yang terbaca dari kode yang benar-benar ada di master
// status klaim: dengan `id_site = 1` dan urutan 134…166, kodenya menjadi 1134…1166
// (`internal/masterstatus/masterstatus.go`). Pola pembentukan kode yang sama dipakai
// SELURUH master di sistem lama, termasuk PEGA_M_CAUSE_OF_LOSS.
const defaultSiteCode = "1"

// Repo menyimpan master COL satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus,
	// dan penambahan yang menerbitkan kode dari isi tabel harus berjalan satu per satu
	// — persis seperti transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []mastercolsimasonline.CauseOfLoss
	failure error

	// business dipakai hanya untuk mengisi nama bisnis saat baris dibaca kembali,
	// meniru join ke POOLDATA.BUSINESS pada adapter SQL. Nil berarti nama dibiarkan
	// kosong — itu tetap benar, hanya kurang informatif di layar.
	business *BusinessRepo
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(rows ...mastercolsimasonline.CauseOfLoss) *Repo {
	copied := make([]mastercolsimasonline.CauseOfLoss, len(rows))
	copy(copied, rows)
	return &Repo{rows: copied}
}

// WithBusiness menyambungkan master bisnis sebagai sumber nama.
//
// Ia terpisah dari NewRepo supaya repo tetap dapat dipakai sendirian di uji yang tidak
// peduli nama bisnis.
func (r *Repo) WithBusiness(business *BusinessRepo) *Repo {
	r.business = business
	return r
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh baris terurut menurut kode, TANPA pemetaan bisnisnya.
//
// Pemetaan sengaja tidak ikut, sama seperti adapter SQL: grid di layar hanya menampilkan
// ID dan Description.
func (r *Repo) List(_ context.Context) ([]mastercolsimasonline.CauseOfLoss, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastercolsimasonline.CauseOfLoss, 0, len(r.rows))
	for _, row := range r.rows {
		result = append(result, mastercolsimasonline.CauseOfLoss{
			Code:        row.Code,
			Description: row.Description,
			MasterCode:  row.MasterCode,
		})
	}
	// `ORDER BY M_COL_ID` pada basis data adalah pengurutan TEKS bila kolomnya bertipe
	// VARCHAR2. Ditiru apa adanya supaya urutan saat pengembangan sama dengan urutan di
	// produksi — termasuk keanehannya, yaitu "1010" mendahului "109".
	sort.SliceStable(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

// Get mengembalikan satu baris lengkap dengan pemetaan bisnisnya.
func (r *Repo) Get(_ context.Context, code string) (mastercolsimasonline.CauseOfLoss, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastercolsimasonline.CauseOfLoss{}, r.failure
	}

	wanted := strings.TrimSpace(code)
	for _, row := range r.rows {
		if row.Code == wanted {
			return mastercolsimasonline.CauseOfLoss{
				Code:        row.Code,
				Description: row.Description,
				MasterCode:  row.MasterCode,
				Businesses:  copyBusinesses(row.Businesses),
			}, nil
		}
	}
	return mastercolsimasonline.CauseOfLoss{}, mastercolsimasonline.ErrNotFound
}

// Insert menerbitkan kode baru lalu menambahkan barisnya.
func (r *Repo) Insert(_ context.Context, data mastercolsimasonline.SaveData) (mastercolsimasonline.CauseOfLoss, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastercolsimasonline.CauseOfLoss{}, r.failure
	}

	fresh := mastercolsimasonline.CauseOfLoss{
		Code:        r.nextCode(),
		Description: data.Description,
		MasterCode:  data.MasterCode,
		Businesses:  copyBusinesses(data.Businesses),
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(_ context.Context, code string, data mastercolsimasonline.SaveData) (mastercolsimasonline.CauseOfLoss, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return mastercolsimasonline.CauseOfLoss{}, r.failure
	}

	wanted := strings.TrimSpace(code)
	for i, row := range r.rows {
		if row.Code != wanted {
			continue
		}
		// Code tidak ikut ditimpa dari luar: ia kunci baris, bukan isian.
		r.rows[i].Description = data.Description
		r.rows[i].MasterCode = data.MasterCode
		// Pemetaan bisnis DIGANTI seluruhnya, bukan digabung. Layar mengirim keadaan
		// akhir grid apa adanya, sehingga baris yang dihapus pengguna memang harus
		// hilang — menggabungkannya akan membuat baris yang dihapus muncul kembali.
		r.rows[i].Businesses = copyBusinesses(data.Businesses)
		return r.rows[i], nil
	}
	return mastercolsimasonline.CauseOfLoss{}, mastercolsimasonline.ErrNotFound
}

// nextCode meniru `id_site || lpad(to_char(M_CAUSE_SEQ.nextval), 3, '0')` pada
// `Database/PEGA_M_CAUSE_OF_LOSS.prc:19`.
//
// Nomor urutnya diturunkan dari isi yang tersimpan, bukan dari pencacah tersendiri,
// supaya repo yang dibentuk dengan SampleList melanjutkan deret yang sudah ada.
func (r *Repo) nextCode() string {
	taken := make(map[string]bool, len(r.rows))
	highest := 0
	for _, row := range r.rows {
		taken[row.Code] = true
		// Nomor urut adalah kode tanpa satu digit situs di depannya.
		if len(row.Code) > 1 {
			if number, err := strconv.Atoi(row.Code[len(defaultSiteCode):]); err == nil && number > highest {
				highest = number
			}
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk isi
	// tabel yang sudah memuat kode berbentuk lain, supaya baris baru tidak menabraknya.
	for number := highest + 1; ; number++ {
		candidate := defaultSiteCode + threeDigits(number)
		if !taken[candidate] {
			return candidate
		}
	}
}

// copyBusinesses menyalin pemetaan bisnis apa adanya, termasuk URUTANNYA.
//
// Disalin, bukan dibagikan, supaya pemanggil yang mengubah senarai hasilnya tidak
// diam-diam mengubah isi yang tersimpan — cacat yang hanya muncul di adapter memori dan
// karena itu paling mudah terlewat.
//
// Nama dan ID disimpan apa adanya, TIDAK dicari ulang ke master: nama bisnis sudah
// diselesaikan lapisan usecase sebelum sampai ke sini, dan nama yang diketik bebas
// memang tidak punya ID (`pyAllowFreeFormInput=true`).
func copyBusinesses(list []mastercolsimasonline.Business) []mastercolsimasonline.Business {
	result := make([]mastercolsimasonline.Business, len(list))
	copy(result, list)
	return result
}

// threeDigits meniru lpad(to_char(seq), 3, '0'). Bilangan di atas 999 dikembalikan apa
// adanya, sama seperti LPAD Oracle — lihat catatan panjang pada
// `internal/masterstatus/repo/sqlstore/masterstatus.go` fungsi ThreeDigits.
func threeDigits(n int) string {
	digits := strconv.Itoa(n)
	for len(digits) < 3 {
		digits = "0" + digits
	}
	return digits
}

// BusinessRepo menyimpan master bisnis satu portal di memori.
type BusinessRepo struct {
	mutex   sync.Mutex
	rows    []mastercolsimasonline.Business
	failure error
}

// NewBusinessRepo membentuk master bisnis berisi baris yang diberikan.
func NewBusinessRepo(rows ...mastercolsimasonline.Business) *BusinessRepo {
	copied := make([]mastercolsimasonline.Business, len(rows))
	copy(copied, rows)
	return &BusinessRepo{rows: copied}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *BusinessRepo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh bisnis, terurut menurut namanya.
func (r *BusinessRepo) List(_ context.Context) ([]mastercolsimasonline.Business, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]mastercolsimasonline.Business, len(r.rows))
	copy(result, r.rows)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (r *BusinessRepo) nameOf(id string) (string, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	wanted := strings.ToUpper(strings.TrimSpace(id))
	for _, b := range r.rows {
		if strings.ToUpper(strings.TrimSpace(b.ID)) == wanted {
			return b.Name, true
		}
	}
	return "", false
}

// SampleBusinessList adalah isi awal master bisnis untuk pengembangan tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi POOLDATA.BUSINESS tidak ada di export:
// tidak ada berkas CSV-nya di `Database/` seperti halnya `v_sts_claim.csv`, dan DDL-nya
// pun belum diterima (`R-08`).
//
// Yang di bawah disusun dari **Group Panel** yang memang terbaca di export dan sudah
// tercatat di `CONTEXT.md` — Personal Accident, Aneka, Marine Cargo, Travel, dan
// Fire/Property. Kodenya mengikuti kode Group Panel itu. Ia cukup untuk mencoba layar,
// dan HARUS diganti isi tabel yang sebenarnya begitu DBA mengirimkannya.
func SampleBusinessList() []mastercolsimasonline.Business {
	return []mastercolsimasonline.Business{
		{ID: "002", Name: "PERSONAL ACCIDENT"},
		{ID: "003", Name: "ANEKA"},
		{ID: "004", Name: "MARINE CARGO"},
		{ID: "005", Name: "TRAVEL"},
		{ID: "006", Name: "FIRE / PROPERTY"},
	}
}

// SampleList adalah isi awal master COL untuk pengembangan dan pengujian tanpa basis
// data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI, dengan alasan yang sama seperti
// SampleBusinessList: isi POOLDATA.M_CAUSE_OF_LOSS tidak ada di export.
//
// Nama-nama di bawah SUSUNAN SENDIRI, dipilih agar bentuk kodenya benar
// (`id_site` + tiga digit) dan agar pemetaan ke lebih dari satu bisnis dapat dicoba di
// layar. Ia tidak boleh dipakai sebagai dasar uji kesetaraan gerbang 1.
// MST_COL_ID pada contoh di bawah menunjuk Code baris LAIN di senarai yang sama —
// `1002` dan `1003` bernaung di bawah `1001`. Itu memang artinya sejak Work Owner
// menegaskannya pada 2026-09-21; ia bukan kode dari sistem sebelah.
//
// Baris `1004` sengaja punya satu bisnis yang TIDAK ada di SampleBusinessList
// ("KENDARAAN BERMOTOR"), supaya keadaan "nama diketik bebas, tanpa ID" ikut terlihat
// saat pengembangan — itu keadaan sah yang harus ditangani setiap layar dan setiap
// pembaca, bukan data rusak.
func SampleList() []mastercolsimasonline.CauseOfLoss {
	return []mastercolsimasonline.CauseOfLoss{
		{
			Code:        "1001",
			Description: "KEBAKARAN",
			MasterCode:  "",
			Businesses: []mastercolsimasonline.Business{
				{ID: "006", Name: "FIRE / PROPERTY"},
				{ID: "003", Name: "ANEKA"},
			},
		},
		{
			Code:        "1002",
			Description: "KEBAKARAN AKIBAT PETIR",
			MasterCode:  "1001",
			Businesses: []mastercolsimasonline.Business{
				{ID: "006", Name: "FIRE / PROPERTY"},
			},
		},
		{
			Code:        "1003",
			Description: "KECELAKAAN DIRI",
			MasterCode:  "",
			Businesses: []mastercolsimasonline.Business{
				{ID: "002", Name: "PERSONAL ACCIDENT"},
			},
		},
		{
			Code:        "1004",
			Description: "KERUSAKAN DALAM PENGANGKUTAN",
			MasterCode:  "",
			Businesses: []mastercolsimasonline.Business{
				{ID: "004", Name: "MARINE CARGO"},
				{ID: "", Name: "KENDARAAN BERMOTOR"},
			},
		},
	}
}

var (
	_ mastercolsimasonline.Repo         = (*Repo)(nil)
	_ mastercolsimasonline.BusinessRepo = (*BusinessRepo)(nil)
)
