// Package memori adalah pengisi seam statusprogres.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara nomor
// baru diturunkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun
// tentang adapter SQL.
package memori

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/statusprogres"
)

// Repo menyimpan master status progres satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// ADR-0030 dan R-20.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus,
	// dan penambahan yang menurunkan nomor dari isi tabel harus berjalan satu per satu
	// — persis seperti FOR UPDATE pada adapter SQL.
	mutex sync.Mutex
	baris []statusprogres.StatusProgres
	galat error
}

// RepoBaru membentuk repo berisi baris yang diberikan.
func RepoBaru(baris ...statusprogres.StatusProgres) *Repo {
	salinan := make([]statusprogres.StatusProgres, len(baris))
	copy(salinan, baris)
	return &Repo{baris: salinan}
}

// SetGalat membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetGalat(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.galat = err
}

// Daftar mengembalikan seluruh baris, terurut menurut ID seperti kueri lama.
func (r *Repo) Daftar(_ context.Context) ([]statusprogres.StatusProgres, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return nil, r.galat
	}

	hasil := make([]statusprogres.StatusProgres, len(r.baris))
	copy(hasil, r.baris)
	// `ORDER BY ID_PROGRESS ASC` pada basis data adalah pengurutan TEKS bila kolomnya
	// bertipe VARCHAR2. Ditiru apa adanya di sini supaya urutan yang terlihat saat
	// pengembangan sama dengan urutan yang terlihat di produksi — termasuk keanehannya,
	// yaitu "010" mendahului "09".
	sort.SliceStable(hasil, func(i, j int) bool { return hasil[i].ID < hasil[j].ID })
	return hasil, nil
}

// Ambil mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo) Ambil(_ context.Context, id string) (statusprogres.StatusProgres, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return statusprogres.StatusProgres{}, r.galat
	}

	dicari := strings.TrimSpace(id)
	for _, sp := range r.baris {
		if sp.ID == dicari {
			return sp, nil
		}
	}
	return statusprogres.StatusProgres{}, statusprogres.ErrTidakDitemukan
}

// SisipBaru menurunkan ID dari isi yang tersimpan lalu menambahkan barisnya.
func (r *Repo) SisipBaru(_ context.Context, isian statusprogres.Isian) (statusprogres.StatusProgres, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return statusprogres.StatusProgres{}, r.galat
	}

	sudahAda := make(map[string]bool, len(r.baris))
	tertinggi := 0
	for _, sp := range r.baris {
		sudahAda[sp.ID] = true
		if angka, err := strconv.Atoi(sp.ID); err == nil && angka > tertinggi {
			tertinggi = angka
		}
	}

	id := ""
	for nomor := tertinggi + 1; ; nomor++ {
		kandidat := statusprogres.FormatNomor(nomor)
		if !sudahAda[kandidat] {
			id = kandidat
			break
		}
	}

	baru := statusprogres.StatusProgres{ID: id, Nama: isian.Nama, KodePosisi: isian.KodePosisi}
	r.baris = append(r.baris, baru)
	return baru, nil
}

// Perbarui menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Perbarui(_ context.Context, sp statusprogres.StatusProgres) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return r.galat
	}

	for i, ada := range r.baris {
		if ada.ID == sp.ID {
			// ID tidak ikut ditimpa dari luar: ia kunci baris, bukan isian.
			r.baris[i].Nama = sp.Nama
			r.baris[i].KodePosisi = sp.KodePosisi
			return nil
		}
	}
	return statusprogres.ErrTidakDitemukan
}

// DaftarContoh adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi sebenarnya
// POOLDATA.GCNM_MST_PROGRESS_KLAIM tidak ada di export: tidak ada berkas CSV-nya di
// `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`, dan DDL-nya pun
// belum diterima (R-08).
//
// Nama-nama di bawah karena itu SUSUNAN SENDIRI, dipilih agar tiap posisi klaim terwakili
// sehingga dropdown dan penyaringan dapat dicoba. Ia tidak boleh dipakai sebagai dasar
// uji kesetaraan gerbang 1, dan harus diganti isi tabel yang sebenarnya begitu DBA
// mengirimkannya — sama seperti DaftarContoh pada modul portal yang disalin langsung
// dari `Database/m_portal_pnc.csv`.
func DaftarContoh() []statusprogres.StatusProgres {
	return []statusprogres.StatusProgres{
		{ID: "01", Nama: "DOKUMEN DITERIMA", KodePosisi: "002"},
		{ID: "02", Nama: "DOKUMEN BELUM LENGKAP", KodePosisi: "002"},
		{ID: "03", Nama: "MENUNGGU JADWAL SURVEI", KodePosisi: "004"},
		{ID: "04", Nama: "SURVEI SELESAI", KodePosisi: "004"},
		{ID: "05", Nama: "MENUNGGU PERSETUJUAN KOMITE", KodePosisi: "006"},
		{ID: "06", Nama: "MENUNGGU AKSEPTASI", KodePosisi: "007"},
	}
}

var _ statusprogres.Repo = (*Repo)(nil)
