// Package masterpicteknikhttp adalah lapisan transport modul Master PIC Teknik: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp, portalhttp,
// dan mastertipesurveyorshttp: foldernya `http` supaya letaknya seragam antarmodul, nama
// paketnya `masterpicteknikhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterpicteknikhttp

// TechnicianDTO adalah bentuk satu petugas teknik yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterpicteknik.Technician. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
type TechnicianDTO struct {
	OperatorID string `json:"id_operator"`

	// Name berasal dari direktori pegawai, bukan dari isian. Ia dikirim supaya layar
	// dapat menampilkannya, tetapi tidak pernah diterima kembali.
	Name string `json:"nama"`

	Email        string `json:"email"`
	BusinessLine string `json:"lini_bisnis"`
	Group        string `json:"grup"`
	Supervisor   string `json:"atasan"`
	Quota        int    `json:"kuota"`

	// ExternalQuota adalah COUNTER_QUOTA2 — beban kerja petugas yang sama di sistem lain.
	// Namanya di sistem lama, alias "OLD_OPERATOR_ID", menyesatkan: isinya angka.
	ExternalQuota int `json:"kuota_luar"`

	// Workload adalah TOTAL_JOB, beban pekerjaan yang sebenarnya. HANYA DIBACA, dan hanya
	// terisi pada daftar: ia kolom milik view dan tidak ada di tabelnya.
	Workload int `json:"beban_kerja"`

	// PanelGroup hanya dibaca: procedure penulis lama tidak pernah menulis kolom ini, dan
	// form Pega tidak memuatnya.
	PanelGroup string `json:"grup_panel"`

	Active bool `json:"aktif"`
}

// ListResponse adalah jawaban GET /api/master/pic-teknik.
type ListResponse struct {
	Technician []TechnicianDTO `json:"pic_teknik"`

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

// SingleResponse adalah jawaban untuk satu petugas: ambil, tambah, dan ubah.
type SingleResponse struct {
	Technician TechnicianDTO `json:"pic_teknik"`
	Portal     string        `json:"portal"`
}

// EmployeeDTO adalah jawaban pencarian direktori pegawai.
//
// Atasan ikut dikirim karena respons direktori memang memuatnya dalam blok `EmpLeader` —
// satu pencarian menjawab dua isian sekaligus, dan layar tidak perlu menembak dua kali.
//
// `nama_atasan` dikirim untuk DITAMPILKAN saja; yang disimpan hanyalah `atasan` yang
// berisi ID. Tabelnya pun hanya punya kolom ATASAN berisi ID, bukan nama.
type EmployeeDTO struct {
	OperatorID     string `json:"id_operator"`
	Name           string `json:"nama"`
	Email          string `json:"email"`
	Supervisor     string `json:"atasan"`
	SupervisorName string `json:"nama_atasan"`
}

// EmployeeResponse adalah jawaban GET /api/master/pic-teknik/direktori/{id}.
type EmployeeResponse struct {
	Employee EmployeeDTO `json:"pegawai"`
	Portal   string      `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Empat hal yang TIDAK diterima, dan itu disengaja:
//
//   - Nama. Ia dimiliki direktori pegawai dan dicari server setiap kali disimpan.
//     Menerimanya berarti master ini dapat memuat nama yang tidak cocok dengan direktori,
//     dan setiap layar penugasan akan menyebut orang yang berbeda dari yang sesungguhnya
//     bertugas.
//   - PanelGroup. Procedure lama tidak pernah menulisnya pada cabang mana pun.
//   - Workload. Ia milik view, dihitung, dan tidak ada di tabel yang ditulis modul ini.
//   - Portal. Ia datang dari header, bukan dari badan permintaan — badan yang dapat
//     menyebut entitas berarti satu permintaan punya dua sumber kebenaran.
//
// OperatorID hanya dipakai pada penambahan. Pada perubahan ia diambil dari jalur URL —
// dua sumber untuk satu nilai berarti keduanya dapat berbeda.
type SaveRequest struct {
	OperatorID    string `json:"id_operator"`
	Email         string `json:"email"`
	BusinessLine  string `json:"lini_bisnis"`
	Group         string `json:"grup"`
	Supervisor    string `json:"atasan"`
	Quota         int    `json:"kuota"`
	ExternalQuota int    `json:"kuota_luar"`
	Active        bool   `json:"aktif"`
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
// `field` dipakai sebagai nama kunci pelanggaran, mengikuti masterstatus dan
// mastertipesurveyors — bukan `kolom` seperti masterstatusprogres. Keduanya sudah
// ditampung `APIError.violations()` di frontend.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
