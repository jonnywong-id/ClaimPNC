// Package memory adalah pengisi seam daftarobjekdokumen.Repo dan BusinessRepo yang hidup
// di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat kedua seam itu nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara ID baru
// diterbitkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/daftarobjekdokumen"
)

// defaultSiteCode meniru POOLDATA.M_SITE_DATABASE.ID pada situs yang aktif.
//
// Nilainya "1" karena itulah yang terbaca dari ID yang benar-benar ada di master status
// klaim: dengan `id_site = 1` dan urutan 134…166, kodenya menjadi 1134…1166
// (`internal/masterstatus/masterstatus.go`). Pola pembentukan ID yang sama dipakai SELURUH
// master di sistem lama, termasuk rumpun LST_*.
const defaultSiteCode = "1"

// sequenceDigits adalah lebar nomor urut, mengikuti
// `POOLDATA.PEGA_LST_DOC_OBJ` — LIMA. Procedure itu ADA di basis data meski hilang dari
// export, dan ID yang terpakai hari ini (100766..100777, CHAR(6)) menegaskannya.
const sequenceDigits = 5

// Repo menyimpan objek dokumen satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penambahan yang menerbitkan ID dari isi tabel harus berjalan satu per satu — persis
	// seperti transaksi pada adapter SQL.
	mutex   sync.Mutex
	rows    []daftarobjekdokumen.DocumentObject
	failure error
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(rows ...daftarobjekdokumen.DocumentObject) *Repo {
	copied := make([]daftarobjekdokumen.DocumentObject, len(rows))
	copy(copied, rows)
	return &Repo{rows: copied}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh baris terurut menurut ID, TANPA pemetaan bisnisnya.
//
// Pemetaan sengaja tidak ikut, sama seperti adapter SQL: grid di layar hanya menampilkan ID
// dan keterangannya.
func (r *Repo) List(_ context.Context) ([]daftarobjekdokumen.DocumentObject, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftarobjekdokumen.DocumentObject, 0, len(r.rows))
	for _, row := range r.rows {
		result = append(result, daftarobjekdokumen.DocumentObject{
			ID:          row.ID,
			Description: row.Description,
			OldID:       row.OldID,
		})
	}
	// `ORDER BY ID` pada basis data adalah pengurutan TEKS bila kolomnya bertipe VARCHAR2.
	// Ditiru apa adanya supaya urutan saat pengembangan sama dengan urutan di produksi —
	// termasuk keanehannya, yaitu "10010" mendahului "1009".
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu baris lengkap dengan pemetaan bisnisnya.
func (r *Repo) Get(_ context.Context, id string) (daftarobjekdokumen.DocumentObject, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftarobjekdokumen.DocumentObject{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, row := range r.rows {
		if row.ID == wanted {
			return daftarobjekdokumen.DocumentObject{
				ID:          row.ID,
				Description: row.Description,
				OldID:       row.OldID,
				Businesses:  copyBusinesses(row.Businesses),
			}, nil
		}
	}
	return daftarobjekdokumen.DocumentObject{}, daftarobjekdokumen.ErrNotFound
}

// Insert menerbitkan ID baru lalu menambahkan barisnya.
func (r *Repo) Insert(_ context.Context, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftarobjekdokumen.DocumentObject{}, r.failure
	}

	fresh := daftarobjekdokumen.DocumentObject{
		ID:          r.nextID(),
		Description: data.Description,
		Businesses:  copyBusinesses(data.Businesses),
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(_ context.Context, id string, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftarobjekdokumen.DocumentObject{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for i, row := range r.rows {
		if row.ID != wanted {
			continue
		}
		// ID dan OldID tidak ikut ditimpa dari luar: yang pertama kunci baris, yang kedua
		// jejak sejarah. Keduanya bukan isian.
		r.rows[i].Description = data.Description
		// Pemetaan bisnis DIGANTI seluruhnya, bukan digabung — sama seperti adapter SQL
		// yang menonaktifkan seluruh baris lebih dulu. Layar mengirim keadaan akhir grid
		// apa adanya, sehingga baris yang dicabut pengguna memang harus hilang.
		r.rows[i].Businesses = copyBusinesses(data.Businesses)
		return r.rows[i], nil
	}
	return daftarobjekdokumen.DocumentObject{}, daftarobjekdokumen.ErrNotFound
}

// nextID meniru `id_site || lpad(to_char(LST_DOC_OBJ_SEQ.nextval), 5, '0')`.
//
// Nomor urutnya diturunkan dari isi yang tersimpan, bukan dari pencacah tersendiri, supaya
// repo yang dibentuk dengan SampleList melanjutkan deret yang sudah ada.
func (r *Repo) nextID() string {
	taken := make(map[string]bool, len(r.rows))
	highest := 0
	for _, row := range r.rows {
		taken[row.ID] = true
		// Nomor urut adalah ID tanpa satu digit situs di depannya.
		if len(row.ID) > len(defaultSiteCode) {
			if number, err := strconv.Atoi(row.ID[len(defaultSiteCode):]); err == nil && number > highest {
				highest = number
			}
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk isi tabel
	// yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	for number := highest + 1; ; number++ {
		candidate := defaultSiteCode + fiveDigits(number)
		if !taken[candidate] {
			return candidate
		}
	}
}

// copyBusinesses menyalin pemetaan bisnis apa adanya, termasuk URUTANNYA.
//
// Disalin, bukan dibagikan, supaya pemanggil yang mengubah senarai hasilnya tidak diam-diam
// mengubah isi yang tersimpan — cacat yang hanya muncul di adapter memori dan karena itu
// paling mudah terlewat.
//
// Nama dan ID disimpan apa adanya, TIDAK dicari ulang ke master: nama bisnis sudah
// diselesaikan lapisan usecase sebelum sampai ke sini, dan nama yang diketik bebas memang
// tidak punya ID.
func copyBusinesses(list []daftarobjekdokumen.Business) []daftarobjekdokumen.Business {
	result := make([]daftarobjekdokumen.Business, len(list))
	copy(result, list)
	return result
}

// fiveDigits meniru lpad(to_char(seq), 5, '0'). Bilangan di atas 99999 dikembalikan apa
// adanya, sama seperti LPAD Oracle — lihat catatan panjang pada
// `repo/sqlstore/daftarobjekdokumen.go` fungsi FiveDigits.
func fiveDigits(n int) string {
	digits := strconv.Itoa(n)
	for len(digits) < sequenceDigits {
		digits = "0" + digits
	}
	return digits
}

// BusinessRepo menyimpan master bisnis satu portal di memori.
type BusinessRepo struct {
	mutex   sync.Mutex
	rows    []daftarobjekdokumen.Business
	failure error
}

// NewBusinessRepo membentuk master bisnis berisi baris yang diberikan.
func NewBusinessRepo(rows ...daftarobjekdokumen.Business) *BusinessRepo {
	copied := make([]daftarobjekdokumen.Business, len(rows))
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
func (r *BusinessRepo) List(_ context.Context) ([]daftarobjekdokumen.Business, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftarobjekdokumen.Business, len(r.rows))
	copy(result, r.rows)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// SampleBusinessList adalah isi awal master bisnis untuk pengembangan tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi POOLDATA.BUSINESS tidak ada di export: tidak
// ada berkas CSV-nya di `Database/` seperti halnya `v_sts_claim.csv`, dan DDL-nya pun belum
// diterima (`R-08`).
//
// Nama-namanya disusun dari **Group Panel** yang terbaca di export dan tercatat di
// `CONTEXT.md`. BENTUK ID-nya mengikuti data sebenarnya — pembacaan basis data 2026-10-03
// memperlihatkan BUSINESS.ID berupa lima digit seperti `10027`, bukan tiga digit seperti
// yang semula disalin dari modul Master COL Simas Online.
//
// Bentuk yang keliru pada data contoh tidak menimbulkan galat; ia hanya membuat uji lulus
// atas bentuk yang tidak pernah ada di produksi.
func SampleBusinessList() []daftarobjekdokumen.Business {
	return []daftarobjekdokumen.Business{
		{ID: "10002", Name: "PERSONAL ACCIDENT"},
		{ID: "10013", Name: "ANEKA"},
		{ID: "10027", Name: "MARINE CARGO"},
		{ID: "10045", Name: "TRAVEL"},
		{ID: "10052", Name: "FIRE / PROPERTY"},
	}
}

// SampleList adalah isi awal objek dokumen untuk pengembangan dan pengujian tanpa basis
// data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi V_LST_DOC_OBJ tidak ada di export, dan tidak
// ada satu pun berkas CSV-nya di `Database/`.
//
// Nama-nama di bawah SUSUNAN SENDIRI, dipilih agar bentuk ID-nya benar — kode situs "1"
// ditambah LIMA digit, menghasilkan enam karakter persis seperti `100766`..`100777` yang
// terpakai di produksi. Ia TIDAK boleh dipakai sebagai dasar uji kesetaraan gerbang 1.
//
// Panjang keterangannya sengaja di bawah 20 karakter, karena itulah lebar kolomnya.
//
// Dua keadaan sengaja ikut terwakili, karena keduanya sah dan harus ditangani setiap layar:
//
//	100002  pemetaan ke lebih dari satu bisnis
//	100003  punya OLD_ID — baris warisan yang pernah bernomor lain
//
// Keadaan "nama bisnis yang tidak ada di master" TIDAK lagi diwakili di sini, dan itu
// disengaja: sejak pemetaan terbukti hanya menyimpan ID, nama seperti itu tidak dapat
// tersimpan sama sekali. Penolakannya diuji di `usecase`, bukan diwakili data contoh.
func SampleList() []daftarobjekdokumen.DocumentObject {
	return []daftarobjekdokumen.DocumentObject{
		{
			ID:          "100001",
			Description: "KTP Tertanggung",
			Businesses: []daftarobjekdokumen.Business{
				{ID: "10002", Name: "PERSONAL ACCIDENT"},
			},
		},
		{
			ID:          "100002",
			Description: "Polis Asli",
			Businesses: []daftarobjekdokumen.Business{
				{ID: "10052", Name: "FIRE / PROPERTY"},
				{ID: "10013", Name: "ANEKA"},
			},
		},
		{
			ID:          "100003",
			Description: "Surat Dokter",
			OldID:       "0007",
			Businesses: []daftarobjekdokumen.Business{
				{ID: "10002", Name: "PERSONAL ACCIDENT"},
			},
		},
		{
			ID:          "100004",
			Description: "Bill of Lading",
			Businesses: []daftarobjekdokumen.Business{
				{ID: "10027", Name: "MARINE CARGO"},
			},
		},
	}
}

var (
	_ daftarobjekdokumen.Repo         = (*Repo)(nil)
	_ daftarobjekdokumen.BusinessRepo = (*BusinessRepo)(nil)
)
