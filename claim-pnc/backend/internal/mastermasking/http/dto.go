// Package mastermaskinghttp adalah lapisan transport modul Master Masking: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `mastermaskinghttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package mastermaskinghttp

// MaskingDTO adalah bentuk satu baris masking yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari mastermasking.Masking. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
//
// # Satu kolom yang sengaja TIDAK ada di sini
//
// PASSWORD. Ia tidak pernah dikirim ke peramban, tidak pernah tampil di layar, dan tidak
// pernah diterima dari klien. Layar Pega pun tidak memilikinya. Alasan lengkapnya ada di
// repo/sqlstore (konstanta legacyPassword): isinya literal coba-coba yang tertinggal, dan
// tidak pernah diperiksa siapa pun.
type MaskingDTO struct {
	ID string `json:"id"`

	BranchID string `json:"cabang"`

	// BranchName ikut dikirim supaya grid tidak perlu memanggil daftar cabang hanya untuk
	// menampilkan namanya. Ia BACA-SAJA — apa pun yang dikirim klien di field ini diabaikan.
	BranchName string `json:"nama_cabang"`

	Login     string `json:"login"`
	Module    string `json:"modul"`
	SubModule string `json:"sub_modul"`

	// Dua kuota. Namanya menyebut "maks" karena itulah yang tertulis di layar lama:
	// "MAX CARI DATA" dan "MAX LIHAT DATA".
	SearchQuota int `json:"maks_cari"`
	ViewQuota   int `json:"maks_lihat"`

	// Tiga kewenangan melihat data pribadi tidak tersamar.
	//
	// Dikirim sebagai boolean, bukan sebagai teks 'Ya'/'Tidak' yang tersimpan di kolom.
	// Bentuk penyimpanan adalah urusan repo; kontrak ke peramban menyebut keputusannya.
	ViewIDCard bool `json:"lihat_ktp"`
	ViewEmail  bool `json:"lihat_email"`
	ViewPhone  bool `json:"lihat_notelp"`

	Active bool `json:"aktif"`

	InputBy string `json:"dicatat_oleh"`

	// InputAt dikirim dalam RFC 3339 dan zona UTC, sama dengan modul lain. Penyesuaian ke
	// waktu setempat dilakukan peramban, bukan server — `docs/Steering/07` §4.4 menetapkan
	// konversi zona waktu hanya terjadi di satu tempat.
	InputAt string `json:"dicatat_pada"`
}

// BranchDTO adalah satu pilihan cabang.
type BranchDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/masking.
type ListResponse struct {
	Masking []MaskingDTO `json:"masking"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Angkanya harus datang dari server supaya yang dilaporkan layar adalah yang
	// benar-benar cocok dengan penyaring di basis data entitas itu.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`). Pada modul
	// ini taruhannya lebih tinggi daripada master lain — isinya adalah daftar siapa yang
	// boleh membuka data pribadi nasabah.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu baris: ambil, tambah, ubah, dan ubah status.
type SingleResponse struct {
	Masking MaskingDTO `json:"masking"`
	Portal  string     `json:"portal"`
}

// BranchListResponse adalah jawaban GET /api/master/masking/cabang.
type BranchListResponse struct {
	Branch []BranchDTO `json:"cabang"`
	Total  int         `json:"total"`
	Portal string      `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// Isinya sepadan dengan form layar lama, TERMASUK `aktif`: `MasterProteksi_Sec:22449`
// memuat isian berlabel "STATUS" yang terikat `InputData.BranchID`, dan
// `InsermaskingDataKlaimPnc_` memetakan properti itu ke parameter `T_STSAKTF`. Menyimpan
// form memang mengubah status di sistem lama.
//
// Yang menjaga status tidak berubah tanpa sengaja bukanlah ketiadaan field ini, melainkan
// LAYAR: tombol Ubah hanya muncul pada baris aktif, meniru `ActionMaskingData_Sec` yang
// bersyarat `.STS_AKTF=='AKTIF'`.
//
// Tiga hal yang TIDAK diterima dari klien, dan masing-masing punya alasannya sendiri:
//
//   - `id`          dibuat penyimpanan saat menambah, dan diambil dari jalur URL saat
//     mengubah. Menerimanya berarti klien dapat memindahkan satu baris ke
//     ID lain.
//   - `nama_cabang` hasil penggabungan ke POOLDATA.BRANCH, dihitung ulang setiap kali
//     dibaca. Menerimanya membuat layar dapat menampilkan nama cabang yang
//     berbeda dari kodenya.
//   - `dicatat_oleh` diisi dari sesi. Membiarkan klien menyebut pelakunya sendiri membuat
//     jejak audit tidak membuktikan apa pun — dan `D-59` menjadikan jejak
//     itu satu-satunya kontrol pengimbang yang tersisa.
//
// Ketiganya bukan sekadar diabaikan: handler menolak badan permintaan yang memuat field
// yang tidak dikenal, sehingga klien yang mencoba mengirimnya mendapat penolakan yang
// jelas alih-alih keheningan.
type SaveRequest struct {
	BranchID    string `json:"cabang"`
	Login       string `json:"login"`
	Module      string `json:"modul"`
	SubModule   string `json:"sub_modul"`
	SearchQuota int    `json:"maks_cari"`
	ViewQuota   int    `json:"maks_lihat"`
	ViewIDCard  bool   `json:"lihat_ktp"`
	ViewEmail   bool   `json:"lihat_email"`
	ViewPhone   bool   `json:"lihat_notelp"`

	// Active WAJIB dikirim pada penyuntingan.
	//
	// Ia bertipe bool biasa, bukan pointer, sehingga menghilangkannya berarti mengirim
	// `false` — dan itu akan menonaktifkan baris tanpa diminta. Layar selalu mengirimnya;
	// pemanggil langsung wajib melakukan hal yang sama. Pada penambahan nilainya
	// DIABAIKAN: baris baru selalu aktif, sama seperti layar lama yang tidak punya cara
	// membuat baris nonaktif.
	Active bool `json:"aktif"`
}

// StatusRequest adalah badan permintaan penonaktifan lewat tombol pada baris.
//
// Ia memuat satu field saja, dan itu disengaja: permintaannya menggantikan keadaan, bukan
// menyuruh "balikkan". Menyuruh membalik akan membuat dua klik yang tiba bersamaan saling
// meniadakan, dan pengguna tidak tahu kewenangan itu akhirnya hidup atau mati.
type StatusRequest struct {
	Active bool `json:"aktif"`
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
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
