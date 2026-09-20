package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterstatusprogres"
)

// Repo2 menyimpan master status progres tingkat 2 satu portal di memory.
//
// Ia adapter kedua yang membuat seam masterstatusprogres.Repo2 nyata, bukan hipotetis —
// sekaligus yang memungkinkan layar tingkat 2 dijalankan dan diuji tanpa Oracle.
//
// # Kenapa ia memegang repo tingkat 1, bukan daftar induknya sendiri
//
// Penambahan tingkat 2 WAJIB membaca baris induk: untuk memastikan induknya ada, dan
// untuk menyalin namanya ke ParentName. Adapter SQL melakukannya dengan membaca tabel
// induk di dalam transaksi yang sama. Kalau di sini induknya disalin menjadi daftar
// terpisah, kedua adapter akan berbeda pada hal yang justru paling ingin diuji —
// misalnya induk yang baru saja ditambahkan lewat layar tingkat 1 tidak akan terlihat.
//
// Dengan memegang repo tingkat 1 milik portal yang sama, keduanya membaca sumber yang
// sama persis.
type Repo2 struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penambahan yang menurunkan nomor dari isi tabel harus berjalan satu per satu —
	// persis seperti FOR UPDATE pada adapter SQL.
	mutex   sync.Mutex
	rows    []masterstatusprogres.ProgressStatus2
	failure error

	// induk adalah repo tingkat 1 milik portal yang sama.
	parent *Repo
}

// NewRepo2 membentuk repo tingkat 2 yang membaca induknya dari repo tingkat 1 tertentu.
func NewRepo2(parent *Repo, rows ...masterstatusprogres.ProgressStatus2) *Repo2 {
	copied := make([]masterstatusprogres.ProgressStatus2, len(rows))
	copy(copied, rows)
	return &Repo2{rows: copied, parent: parent}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo2) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh baris, terurut menurut ID seperti kueri lama.
func (r *Repo2) List(_ context.Context) ([]masterstatusprogres.ProgressStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterstatusprogres.ProgressStatus2, len(r.rows))
	copy(result, r.rows)
	// `ORDER BY ID_MST ASC` pada basis data adalah pengurutan TEKS bila kolomnya bertipe
	// teks. Ditiru apa adanya supaya urutan yang terlihat saat pengembangan sama dengan
	// urutan yang terlihat di produksi — termasuk keanehannya, yaitu "10" mendahului "9".
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo2) Get(_ context.Context, id string) (masterstatusprogres.ProgressStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterstatusprogres.ProgressStatus2{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, sp := range r.rows {
		if sp.ID == wanted {
			return sp, nil
		}
	}
	return masterstatusprogres.ProgressStatus2{}, masterstatusprogres.ErrNotFound
}

// InsertNew membaca induknya, menurunkan ID, lalu menambahkan barisnya.
//
// Urutannya sengaja sama dengan adapter SQL: induk diperiksa LEBIH DULU, sebelum nomor
// diturunkan. Uji yang lulus di sini karena itu membuktikan sesuatu tentang adapter SQL,
// bukan hanya tentang dirinya sendiri.
func (r *Repo2) InsertNew(ctx context.Context, input masterstatusprogres.Input2) (masterstatusprogres.ProgressStatus2, error) {
	// Induk dibaca SEBELUM mutex dikunci: Repo tingkat 1 punya mutex-nya sendiri, dan
	// mengunci keduanya dalam urutan yang berbeda di tempat lain adalah cara paling umum
	// membuat kebuntuan. Di sini hanya satu kunci yang dipegang pada satu waktu.
	parent, err := r.parent.Get(ctx, input.ParentID)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("%w: %q", masterstatusprogres.ErrParentNotFound, input.ParentID)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterstatusprogres.ProgressStatus2{}, r.failure
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
		candidate := masterstatusprogres.FormatID2(number)
		if !taken[candidate] {
			id = candidate
			break
		}
	}

	fresh := masterstatusprogres.ProgressStatus2{
		ID:       id,
		Name:     input.Name,
		ParentID: parent.ID,
		// Disalin, bukan dirujuk — meniru sistem lama apa adanya. Baris yang sudah
		// tersimpan karena itu TIDAK ikut berubah bila nama induknya kelak diganti,
		// dan itulah yang membuat perilakunya setara.
		ParentName: parent.Name,
		// Tipe dibiarkan kosong: sistem lama pun tidak mengisinya saat menyisipkan.
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan nama dan induk pada baris yang sudah ada.
//
// Induk dibaca dari repo tingkat 1 yang SAMA dengan yang dipakai InsertNew, sehingga
// adapter ini dan adapter SQL memperlakukan perpindahan induk dengan cara yang sama:
// namanya disalin ulang, tidak dibiarkan menyimpang.
//
// TIPE dan ID tidak pernah tersentuh — yang pertama karena aplikasi ini tidak pernah
// menulisnya, yang kedua karena ia kunci baris yang dirujuk data klaim berjalan.
func (r *Repo2) Update(ctx context.Context, id string, input masterstatusprogres.Input2) (masterstatusprogres.ProgressStatus2, error) {
	// Induk dibaca SEBELUM kunci diambil, sama seperti InsertNew: repo tingkat 1 punya
	// kuncinya sendiri, dan mengambil keduanya bersarang membuka jalan ke deadlock.
	parent, err := r.parent.Get(ctx, input.ParentID)
	if err != nil {
		return masterstatusprogres.ProgressStatus2{}, fmt.Errorf("%w: %q", masterstatusprogres.ErrParentNotFound, input.ParentID)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterstatusprogres.ProgressStatus2{}, r.failure
	}

	for i, sp := range r.rows {
		if strings.TrimSpace(sp.ID) != strings.TrimSpace(id) {
			continue
		}

		r.rows[i].Name = input.Name
		r.rows[i].ParentID = parent.ID
		r.rows[i].ParentName = parent.Name
		// Kind (TIPE) sengaja dibiarkan apa adanya — baris lama tidak boleh kehilangan
		// nilainya hanya karena namanya disunting.
		return r.rows[i], nil
	}

	return masterstatusprogres.ProgressStatus2{}, masterstatusprogres.ErrNotFound
}

// SampleList2 adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi sebenarnya POOLDATA.GCNM_MST_PROGRESS tidak
// ada di export: tidak ada berkas CSV-nya di `Database/` seperti halnya `v_sts_claim.csv`
// dan `m_portal_pnc.csv`, dan DDL-nya pun belum diterima (R-08).
//
// Nama-nama di bawah karena itu SUSUNAN SENDIRI. Induknya sengaja menunjuk ID yang ada
// pada SampleList tingkat 1, supaya tautan induk–anak dapat dicoba di layar. Ia tidak
// boleh dipakai sebagai dasar uji kesetaraan gerbang 1, dan harus diganti isi tabel yang
// sebenarnya begitu DBA mengirimkannya.
//
// Kolom Tipe dibiarkan kosong pada seluruh contoh — sama seperti baris yang disisipkan
// sistem lama, yang juga tidak pernah mengisinya.
func SampleList2() []masterstatusprogres.ProgressStatus2 {
	return []masterstatusprogres.ProgressStatus2{
		{ID: "1", Name: "SURAT PERMINTAAN DOKUMEN DIKIRIM", ParentID: "01", ParentName: "DOKUMEN DITERIMA"},
		{ID: "2", Name: "DOKUMEN SUSULAN DITERIMA", ParentID: "01", ParentName: "DOKUMEN DITERIMA"},
		{ID: "3", Name: "MENUNGGU BALASAN TERTANGGUNG", ParentID: "02", ParentName: "DOKUMEN BELUM LENGKAP"},
		{ID: "4", Name: "SURVEYOR DITUNJUK", ParentID: "03", ParentName: "MENUNGGU JADWAL SURVEI"},
		{ID: "5", Name: "LAPORAN SURVEI DITERIMA", ParentID: "04", ParentName: "SURVEI SELESAI"},
		{ID: "6", Name: "BERKAS DIKIRIM KE KOMITE", ParentID: "05", ParentName: "MENUNGGU PERSETUJUAN KOMITE"},
	}
}

var _ masterstatusprogres.Repo2 = (*Repo2)(nil)
