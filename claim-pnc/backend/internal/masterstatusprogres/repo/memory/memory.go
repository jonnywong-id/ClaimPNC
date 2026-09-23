// Package memory adalah pengisi seam masterstatusprogres.Repo yang hidup di dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara nomor
// baru diturunkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun
// tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterstatusprogres"
)

// Repo menyimpan master status progres satu portal di memory.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// ADR-0030 dan R-20.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus,
	// dan penambahan yang menurunkan nomor dari isi tabel harus berjalan satu per satu
	// — persis seperti FOR UPDATE pada adapter SQL.
	mutex   sync.Mutex
	rows    []masterstatusprogres.ProgressStatus
	failure error
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(rows ...masterstatusprogres.ProgressStatus) *Repo {
	copied := make([]masterstatusprogres.ProgressStatus, len(rows))
	copy(copied, rows)
	return &Repo{rows: copied}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh baris, terurut menurut ID seperti kueri lama.
func (r *Repo) List(_ context.Context) ([]masterstatusprogres.ProgressStatus, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterstatusprogres.ProgressStatus, len(r.rows))
	copy(result, r.rows)
	// `ORDER BY ID_PROGRESS ASC` pada basis data adalah pengurutan TEKS bila kolomnya
	// bertipe VARCHAR2. Ditiru apa adanya di sini supaya urutan yang terlihat saat
	// pengembangan sama dengan urutan yang terlihat di produksi — termasuk keanehannya,
	// yaitu "010" mendahului "09".
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo) Get(_ context.Context, id string) (masterstatusprogres.ProgressStatus, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterstatusprogres.ProgressStatus{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, sp := range r.rows {
		if sp.ID == wanted {
			return sp, nil
		}
	}
	return masterstatusprogres.ProgressStatus{}, masterstatusprogres.ErrNotFound
}

// InsertNew menurunkan ID dari isi yang tersimpan lalu menambahkan barisnya.
func (r *Repo) InsertNew(_ context.Context, input masterstatusprogres.Input) (masterstatusprogres.ProgressStatus, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterstatusprogres.ProgressStatus{}, r.failure
	}

	taken := make(map[string]bool, len(r.rows))
	highest := 0
	for _, sp := range r.rows {
		taken[sp.ID] = true
		if number, err := strconv.Atoi(sp.ID); err == nil && number > highest {
			highest = number
		}
	}

	id := ""
	for number := highest + 1; ; number++ {
		candidate := masterstatusprogres.FormatID(number)
		if !taken[candidate] {
			id = candidate
			break
		}
	}

	fresh := masterstatusprogres.ProgressStatus{ID: id, Name: input.Name, PositionCode: input.PositionCode}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(_ context.Context, sp masterstatusprogres.ProgressStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	for i, exists := range r.rows {
		if exists.ID == sp.ID {
			// ID tidak ikut ditimpa dari luar: ia kunci baris, bukan isian.
			r.rows[i].Name = sp.Name
			r.rows[i].PositionCode = sp.PositionCode
			return nil
		}
	}
	return masterstatusprogres.ErrNotFound
}

// SampleList adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi sebenarnya
// POOLDATA.GCNM_MST_PROGRESS_KLAIM tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, dan DDL-nya pun
// belum diterima (R-08).
//
// Nama-nama di bawah karena itu SUSUNAN SENDIRI, dipilih agar tiap posisi klaim terwakili
// sehingga dropdown dan penyaringan dapat dicoba. Ia tidak boleh dipakai sebagai dasar
// uji kesetaraan gerbang 1, dan harus diganti isi tabel yang sebenarnya begitu DBA
// mengirimkannya — sama seperti SampleList pada modul portal yang disalin langsung
// dari `Database/m_portal_pnc.csv`.
func SampleList() []masterstatusprogres.ProgressStatus {
	return []masterstatusprogres.ProgressStatus{
		{ID: "01", Name: "DOKUMEN DITERIMA", PositionCode: "REGISTER"},
		{ID: "02", Name: "DOKUMEN BELUM LENGKAP", PositionCode: "REGISTER"},
		{ID: "03", Name: "MENUNGGU JADWAL SURVEI", PositionCode: "SURVEY"},
		{ID: "04", Name: "SURVEI SELESAI", PositionCode: "SURVEY"},
		{ID: "05", Name: "MENUNGGU PERSETUJUAN KOMITE", PositionCode: "KOMITE"},
		{ID: "06", Name: "MENUNGGU AKSEPTASI", PositionCode: "AKSEPTASI"},
	}
}

var _ masterstatusprogres.Repo = (*Repo)(nil)
