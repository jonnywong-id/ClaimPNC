package memori

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/statusprogres"
)

// Repo2 menyimpan master status progres tingkat 2 satu portal di memori.
//
// Ia adapter kedua yang membuat seam statusprogres.Repo2 nyata, bukan hipotetis —
// sekaligus yang memungkinkan layar tingkat 2 dijalankan dan diuji tanpa Oracle.
//
// # Kenapa ia memegang repo tingkat 1, bukan daftar induknya sendiri
//
// Penambahan tingkat 2 WAJIB membaca baris induk: untuk memastikan induknya ada, dan
// untuk menyalin namanya ke NamaInduk. Adapter SQL melakukannya dengan membaca tabel
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
	mutex sync.Mutex
	baris []statusprogres.StatusProgres2
	galat error

	// induk adalah repo tingkat 1 milik portal yang sama.
	induk *Repo
}

// Repo2Baru membentuk repo tingkat 2 yang membaca induknya dari repo tingkat 1 tertentu.
func Repo2Baru(induk *Repo, baris ...statusprogres.StatusProgres2) *Repo2 {
	salinan := make([]statusprogres.StatusProgres2, len(baris))
	copy(salinan, baris)
	return &Repo2{baris: salinan, induk: induk}
}

// SetGalat membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo2) SetGalat(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.galat = err
}

// Daftar mengembalikan seluruh baris, terurut menurut ID seperti kueri lama.
func (r *Repo2) Daftar(_ context.Context) ([]statusprogres.StatusProgres2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return nil, r.galat
	}

	hasil := make([]statusprogres.StatusProgres2, len(r.baris))
	copy(hasil, r.baris)
	// `ORDER BY ID_MST ASC` pada basis data adalah pengurutan TEKS bila kolomnya bertipe
	// teks. Ditiru apa adanya supaya urutan yang terlihat saat pengembangan sama dengan
	// urutan yang terlihat di produksi — termasuk keanehannya, yaitu "10" mendahului "9".
	sort.SliceStable(hasil, func(i, j int) bool { return hasil[i].ID < hasil[j].ID })
	return hasil, nil
}

// Ambil mengembalikan satu baris berdasarkan ID-nya.
func (r *Repo2) Ambil(_ context.Context, id string) (statusprogres.StatusProgres2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return statusprogres.StatusProgres2{}, r.galat
	}

	dicari := strings.TrimSpace(id)
	for _, sp := range r.baris {
		if sp.ID == dicari {
			return sp, nil
		}
	}
	return statusprogres.StatusProgres2{}, statusprogres.ErrTidakDitemukan
}

// SisipBaru membaca induknya, menurunkan ID, lalu menambahkan barisnya.
//
// Urutannya sengaja sama dengan adapter SQL: induk diperiksa LEBIH DULU, sebelum nomor
// diturunkan. Uji yang lulus di sini karena itu membuktikan sesuatu tentang adapter SQL,
// bukan hanya tentang dirinya sendiri.
func (r *Repo2) SisipBaru(ctx context.Context, isian statusprogres.Isian2) (statusprogres.StatusProgres2, error) {
	// Induk dibaca SEBELUM mutex dikunci: Repo tingkat 1 punya mutex-nya sendiri, dan
	// mengunci keduanya dalam urutan yang berbeda di tempat lain adalah cara paling umum
	// membuat kebuntuan. Di sini hanya satu kunci yang dipegang pada satu waktu.
	induk, err := r.induk.Ambil(ctx, isian.IDInduk)
	if err != nil {
		return statusprogres.StatusProgres2{}, fmt.Errorf("%w: %q", statusprogres.ErrIndukTidakDitemukan, isian.IDInduk)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.galat != nil {
		return statusprogres.StatusProgres2{}, r.galat
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
		kandidat := statusprogres.FormatNomor2(nomor)
		if !sudahAda[kandidat] {
			id = kandidat
			break
		}
	}

	baru := statusprogres.StatusProgres2{
		ID:      id,
		Nama:    isian.Nama,
		IDInduk: induk.ID,
		// Disalin, bukan dirujuk — meniru sistem lama apa adanya. Baris yang sudah
		// tersimpan karena itu TIDAK ikut berubah bila nama induknya kelak diganti,
		// dan itulah yang membuat perilakunya setara.
		NamaInduk: induk.Nama,
		// Tipe dibiarkan kosong: sistem lama pun tidak mengisinya saat menyisipkan.
	}
	r.baris = append(r.baris, baru)
	return baru, nil
}

// DaftarContoh2 adalah isi awal untuk pengembangan dan pengujian tanpa basis data.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi sebenarnya POOLDATA.GCNM_MST_PROGRESS tidak
// ada di export: tidak ada berkas CSV-nya di `Database/` seperti halnya `v_sts_claim.csv`
// dan `m_portal_pnc.csv`, dan DDL-nya pun belum diterima (R-08).
//
// Nama-nama di bawah karena itu SUSUNAN SENDIRI. Induknya sengaja menunjuk ID yang ada
// pada DaftarContoh tingkat 1, supaya tautan induk–anak dapat dicoba di layar. Ia tidak
// boleh dipakai sebagai dasar uji kesetaraan gerbang 1, dan harus diganti isi tabel yang
// sebenarnya begitu DBA mengirimkannya.
//
// Kolom Tipe dibiarkan kosong pada seluruh contoh — sama seperti baris yang disisipkan
// sistem lama, yang juga tidak pernah mengisinya.
func DaftarContoh2() []statusprogres.StatusProgres2 {
	return []statusprogres.StatusProgres2{
		{ID: "1", Nama: "SURAT PERMINTAAN DOKUMEN DIKIRIM", IDInduk: "01", NamaInduk: "DOKUMEN DITERIMA"},
		{ID: "2", Nama: "DOKUMEN SUSULAN DITERIMA", IDInduk: "01", NamaInduk: "DOKUMEN DITERIMA"},
		{ID: "3", Nama: "MENUNGGU BALASAN TERTANGGUNG", IDInduk: "02", NamaInduk: "DOKUMEN BELUM LENGKAP"},
		{ID: "4", Nama: "SURVEYOR DITUNJUK", IDInduk: "03", NamaInduk: "MENUNGGU JADWAL SURVEI"},
		{ID: "5", Nama: "LAPORAN SURVEI DITERIMA", IDInduk: "04", NamaInduk: "SURVEI SELESAI"},
		{ID: "6", Nama: "BERKAS DIKIRIM KE KOMITE", IDInduk: "05", NamaInduk: "MENUNGGU PERSETUJUAN KOMITE"},
	}
}

var _ statusprogres.Repo2 = (*Repo2)(nil)
