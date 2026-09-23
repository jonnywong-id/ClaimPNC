// Package mastertipesurveyorshttp adalah lapisan transport modul Master Tipe Surveyors:
// bentuk permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastertipesurveyorshttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package mastertipesurveyorshttp

// SurveyorTypeDTO adalah bentuk satu tipe surveyor yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari mastertipesurveyors.SurveyorType. Memakai tipe modul
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
type SurveyorTypeDTO struct {
	Code        string `json:"kode"`
	Description string `json:"deskripsi"`

	// LegacyCode adalah jejak penomoran sistem sebelumnya (OLD_M_SURVEY_ID).
	//
	// Ia dikirim meski KOSONG pada seluruh empat baris portal ASM: ia bagian dari kontrak
	// view yang dibaca Pega, dan basis data entitas lain belum tentu sama. Layar yang
	// memutuskan menampilkannya atau tidak.
	LegacyCode string `json:"kode_lama"`
}

// ListResponse adalah jawaban GET /api/master/tipe-surveyor.
type ListResponse struct {
	SurveyorType []SurveyorTypeDTO `json:"tipe_surveyor"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Angkanya harus datang dari server supaya yang dilaporkan layar adalah yang
	// benar-benar ada di basis data entitas itu.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu tipe surveyor: ambil, tambah, dan ubah.
type SingleResponse struct {
	SurveyorType SurveyorTypeDTO `json:"tipe_surveyor"`
	Portal       string          `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Hanya deskripsi yang diterima. Kode TIDAK pernah datang dari klien: pada penambahan ia
// dibuat penyimpanan, dan pada perubahan ia diambil dari jalur URL. Menerimanya dari badan
// permintaan akan membuat klien dapat memindahkan satu tipe ke kode lain — dan kode itu
// dipatok langsung di tiga kueri Pega.
type SaveRequest struct {
	Description string `json:"deskripsi"`
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
// Kode dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Message.
//
// Bentuknya sengaja dibuat sama dengan modul master lain. Menyatukannya menjadi satu tipe
// bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat seluruh aplikasi — dan
// tiket itu masih terhalang keputusan Work Owner. Sampai itu diputuskan, tipe yang
// berbentuk sama lebih jujur daripada satu tipe bersama yang menyiratkan kontraknya sudah
// ada.
//
// `field` dipakai sebagai nama kunci pelanggaran, mengikuti masterstatus — bukan `kolom`
// seperti masterstatusprogres. Keduanya sudah ditampung `APIError.violations()` di
// frontend, sehingga layar tidak perlu memilih sendiri.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
