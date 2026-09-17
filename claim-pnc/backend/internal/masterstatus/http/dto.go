// Package masterstatushttp adalah lapisan transport modul Master Status Klaim: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterstatushttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterstatushttp

// StatusKlaimDTO adalah bentuk satu status klaim yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterstatus.StatusKlaim. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
type StatusKlaimDTO struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`

	// KodeLama adalah penomoran 01–11 yang melekat pada sebelas kode pertama. Ia
	// dikirim supaya layar dapat menampilkannya, dan kosong untuk 22 kode sisanya.
	KodeLama string `json:"kode_lama"`
}

// ResponsDaftar adalah jawaban GET /api/master/status-klaim.
type ResponsDaftar struct {
	StatusKlaim []StatusKlaimDTO `json:"status_klaim"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Salah satu acceptance criteria TKT-F4-005 berbunyi "memuat TEPAT 33 kode,
	// dihitung dan dilaporkan angkanya" — angka itu harus datang dari server.
	Total int `json:"total"`
}

// ResponsSatu adalah jawaban untuk satu status klaim: ambil, tambah, dan ubah.
type ResponsSatu struct {
	StatusKlaim StatusKlaimDTO `json:"status_klaim"`
}

// PermintaanSimpan adalah isian form tambah dan ubah.
//
// Hanya label yang diterima. Kode TIDAK pernah datang dari klien: pada penambahan ia
// dibuat penyimpanan, dan pada perubahan ia diambil dari jalur URL. Menerimanya dari
// badan permintaan akan membuat klien dapat memindahkan satu status ke kode lain.
type PermintaanSimpan struct {
	Label string `json:"label"`
}

// PelanggaranDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form.
type PelanggaranDTO struct {
	Field string `json:"field"`
	Pesan string `json:"pesan"`
}

// ResponsGalat adalah bentuk galat modul ini.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuknya sengaja dibuat sama persis dengan authhttp.ResponsGalat, ditambah satu field
// opsional. Menyatukan keduanya menjadi satu tipe bersama adalah lingkup TKT-F1-004,
// kontrak galat yang mengikat seluruh aplikasi — dan tiket itu masih terhalang keputusan
// Work Owner. Sampai itu diputuskan, dua tipe yang berbentuk sama lebih jujur daripada
// satu tipe bersama yang menyiratkan kontraknya sudah ada.
type ResponsGalat struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []PelanggaranDTO `json:"detail,omitempty"`
}
