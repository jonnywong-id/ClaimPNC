// Package masterpenyebabkerugianhttp adalah lapisan transport modul Master Penyebab
// Kerugian: bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran
// rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// masterstatushttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterpenyebabkerugianhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterpenyebabkerugianhttp

// CauseOfLossDTO adalah bentuk satu golongan penyebab kerugian yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterpenyebabkerugian.CauseOfLoss. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
type CauseOfLossDTO struct {
	// ID dibuat sistem saat golongan ditambahkan; tidak pernah berubah sesudahnya.
	// Bentuknya kode situs ditambah tiga digit, mis. `1001`.
	ID string `json:"id"`

	// Deskripsi memetakan kolom COL_DESC — teks yang dibaca pengguna, dan yang menjadi
	// kolom pengelompokan pada laporan yang memakai `GROUP BY COL_DESC`.
	//
	// Nama fieldnya `deskripsi`, mengikuti label yang terpasang di layar Pega: `pyCaption`
	// **"Deskripsi Kerugian"** pada kolom grid maupun isian formnya. Field JSON adalah
	// kontrak yang mencerminkan isinya, dan di sini isinya memang itu.
	Deskripsi string `json:"deskripsi"`

	// IDLama memetakan kolom OLD_M_COL_ID: penomoran sebelum sistem ini dibangun.
	//
	// Ia tidak pernah diisi maupun diubah aplikasi, dan kosong pada golongan yang tidak
	// pernah punya nomor lama.
	//
	// # Dikirim, tetapi TIDAK ditampilkan layar
	//
	// Grid Pega hanya memuat dua kolom — `ID` dan `Deskripsi Kerugian`. Layar baru
	// mengikutinya apa adanya, sehingga kolom ini tidak tampil di mana pun.
	//
	// Ia tetap dikirim karena lapisan data mengikuti `BrowseVMCauseOfLoss_RD`, yang
	// MEMUATNYA — sama seperti lapisan layar mengikuti section-nya. Keduanya artefak Pega
	// yang berbeda, dan keduanya diikuti pada tempatnya masing-masing. Dengan begitu modul
	// Rincian Penyebab Kerugian (`MENU_ID 38`) kelak tidak perlu mengubah kontrak ini
	// untuk menautkan data historisnya.
	IDLama string `json:"id_lama"`
}

// ListResponse adalah jawaban GET /api/master/penyebab-kerugian.
type ListResponse struct {
	PenyebabKerugian []CauseOfLossDTO `json:"penyebab_kerugian"`

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

// SingleResponse adalah jawaban untuk satu golongan: ambil, tambah, dan ubah.
type SingleResponse struct {
	PenyebabKerugian CauseOfLossDTO `json:"penyebab_kerugian"`
	Portal           string         `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Hanya deskripsi yang diterima. ID TIDAK pernah datang dari klien: pada penambahan ia
// dibuat penyimpanan, dan pada perubahan ia diambil dari jalur URL. Menerimanya dari badan
// permintaan akan membuat klien dapat memindahkan satu golongan ke nomor lain — dan
// memutus setiap baris D_CAUSE_OF_LOSS yang bernaung di bawah nomor lamanya.
//
// ID lama pun tidak diterima: ia jejak sejarah yang ditulis sebelum sistem ini ada, bukan
// isian yang boleh disunting. Layar Pega pun tidak menyediakan isiannya.
//
// # Satu isian Pega yang sengaja TIDAK dibawa
//
// Form Pega sebenarnya memuat DUA isian berlabel "Deskripsi Kerugian": satu terikat
// `TempCauseOfLoss.COL_DESC`, satu lagi terikat `TempCauseOfLoss.Description`. Keduanya
// berbagi `pyAutomationID` yang sama — tanda salin-tempel.
//
// Yang kedua **mati**: `Activity/SetCauseOfLossValue_act-Act.xml` mengisi form dengan
// `M_COL_ID`, `COL_DESC`, dan `pyNote` saja, sehingga `.Description` tidak pernah terisi
// saat mengubah; dan view `V_M_CAUSE_OF_LOSS` tidak punya kolom untuk membacanya kembali.
// Apa pun yang diketik di sana hanya menumpang di dokumen JSON lalu hilang dari pandangan.
//
// Membawanya berarti menyalin isian yang sejak semula tidak berfungsi. Ia ditinggalkan,
// dan alasannya dicatat di sini supaya ketiadaannya tidak dibaca sebagai kelalaian.
type SaveRequest struct {
	Deskripsi string `json:"deskripsi"`
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
// menjadi satu tipe bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat seluruh
// aplikasi — dan tiket itu masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
