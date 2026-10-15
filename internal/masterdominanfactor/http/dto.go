// Package masterdominanfactorhttp adalah lapisan transport modul Master Dominan Factor:
// bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// masterstatushttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterdominanfactorhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterdominanfactorhttp

// DominantFactorDTO adalah bentuk satu faktor dominan yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterdominanfactor.DominantFactor. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
type DominantFactorDTO struct {
	// ID dibuat sistem saat faktor ditambahkan; tidak pernah berubah sesudahnya.
	ID string `json:"id"`

	// Nama memetakan kolom NAME.
	//
	// Nama fieldnya `nama`, bukan `keterangan`, meski layar Pega melabelinya
	// "Keterangan". Alasannya: field JSON adalah KONTRAK yang mencerminkan isinya, dan
	// isinya adalah kolom NAME — sementara label layar mengikuti Pega apa adanya
	// (`D-13`). Keduanya memang berbeda peruntukan, dan perbedaan itu dicatat supaya
	// tidak dibaca sebagai ketidakkonsistenan.
	Nama string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/dominan-factor.
type ListResponse struct {
	DominantFactor []DominantFactorDTO `json:"dominan_factor"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Angkanya adalah isi master ENTITAS YANG MENJAWAB — bukan angka yang sama untuk
	// seluruh aplikasi.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu faktor dominan: ambil, tambah, dan ubah.
type SingleResponse struct {
	DominantFactor DominantFactorDTO `json:"dominan_factor"`
	Portal         string            `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Hanya nama yang diterima. ID TIDAK pernah datang dari klien: pada penambahan ia dibuat
// penyimpanan, dan pada perubahan ia diambil dari jalur URL. Menerimanya dari badan
// permintaan akan membuat klien dapat memindahkan satu faktor ke nomor lain — dan
// memutus setiap baris T_CLAIM_DOMINANFACTOR yang menyimpan nomor lamanya.
type SaveRequest struct {
	Nama string `json:"nama"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuknya sengaja dibuat sama persis dengan modul master lain. Menyatukan seluruhnya
// menjadi satu tipe bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat
// seluruh aplikasi — dan tiket itu masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
