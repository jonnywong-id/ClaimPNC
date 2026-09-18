// Package statusprogreshttp adalah lapisan transport modul Master Status Progres.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `statusprogreshttp` supaya tidak menutupi `net/http`.
package statusprogreshttp

import "claim-pnc/internal/statusprogres"

// StatusProgresDTO adalah bentuk satu baris master yang dikirim ke peramban.
//
// Terpisah dari statusprogres.StatusProgres supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type StatusProgresDTO struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`

	// KodePosisi adalah nilai yang tersimpan, dipakai saat menyunting.
	KodePosisi string `json:"kode_posisi"`

	// NamaPosisi adalah label yang dibaca pengguna.
	//
	// Ia dikirim bersama kodenya supaya layar tidak perlu memetakan sendiri — dan
	// karena itu tidak perlu menyimpan salinan keempat posisi di frontend. Satu daftar,
	// satu tempat.
	NamaPosisi string `json:"nama_posisi"`
}

// PosisiDTO adalah satu pilihan pada dropdown Posisi.
type PosisiDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// ResponsDaftar adalah jawaban GET /api/master/status-progres-1.
type ResponsDaftar struct {
	StatusProgres []StatusProgresDTO `json:"status_progres"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna, bukan entitas lain. Pada aplikasi yang
	// melayani empat badan hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// ResponsSatu adalah jawaban penambahan dan penyuntingan.
type ResponsSatu struct {
	StatusProgres StatusProgresDTO `json:"status_progres"`
	Portal        string           `json:"portal"`
}

// ResponsPosisi adalah jawaban GET /api/master/posisi-klaim.
type ResponsPosisi struct {
	Posisi []PosisiDTO `json:"posisi"`
}

// PermintaanSimpan adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini, dan itu disengaja. Pada penambahan ia diturunkan dari isi tabel;
// pada penyuntingan ia diambil dari jalur, bukan dari badan — dua sumber untuk satu
// nilai berarti keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan
// yang tidak perlu ada.
type PermintaanSimpan struct {
	Nama       string `json:"nama"`
	KodePosisi string `json:"kode_posisi"`
}

// ResponsGalat adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — `{kode, pesan}` — ditambah `detail` untuk
// pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ResponsGalat struct {
	Kode   string        `json:"kode"`
	Pesan  string        `json:"pesan"`
	Detail []DetailGalat `json:"detail,omitempty"`
}

// DetailGalat adalah satu isian yang tidak lolos pemeriksaan.
type DetailGalat struct {
	Kolom string `json:"kolom"`
	Pesan string `json:"pesan"`
}

// keDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func keDTO(sp statusprogres.StatusProgres) StatusProgresDTO {
	return StatusProgresDTO{
		ID:         sp.ID,
		Nama:       sp.Nama,
		KodePosisi: sp.KodePosisi,
		NamaPosisi: statusprogres.NamaPosisi(sp.KodePosisi),
	}
}

// keDaftarDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func keDaftarDTO(daftar []statusprogres.StatusProgres) []StatusProgresDTO {
	hasil := make([]StatusProgresDTO, 0, len(daftar))
	for _, sp := range daftar {
		hasil = append(hasil, keDTO(sp))
	}
	return hasil
}
