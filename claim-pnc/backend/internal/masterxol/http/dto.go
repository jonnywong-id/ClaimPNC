// Package masterxolhttp adalah lapisan transport modul Master XOL: bentuk permintaan dan
// respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola modul lain:
// foldernya `http` supaya letaknya seragam antarmodul, nama paketnya `masterxolhttp`
// supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterxolhttp

// ReinsurerDTO adalah satu baris reas pada sebuah lapisan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK, bukan nama internal (`D-80`).
type ReinsurerDTO struct {
	// ID memetakan kolom IDREAS.
	ID string `json:"id"`

	// Nama memetakan kolom NAMA. Layar lama melabelinya "Reasuransi".
	Nama string `json:"nama"`

	// Share memetakan kolom PERCENTSHARE, dalam persen utuh.
	Share int64 `json:"share"`
}

// LayerDTO adalah satu lapisan treaty.
type LayerDTO struct {
	// ID memetakan kolom IDLAYER. Kosong pada lapisan yang belum pernah disimpan —
	// nomornya diterbitkan server.
	ID string `json:"id"`

	// Nama memetakan kolom NAMA.
	Nama string `json:"nama"`

	// Limit dan Excess memetakan kolom "LIMIT" dan EXCESS, dalam DOLAR. Layar lama
	// melabelinya "Limit (USD)" dan "Excess (USD)".
	Limit  int64 `json:"limit"`
	Excess int64 `json:"excess"`

	// LimitIDR memetakan kolom CONVERT_LIMIT, dalam RUPIAH.
	//
	// HANYA DIKIRIM, tidak pernah diterima: server menghitungnya dari Limit dan kurs
	// induk. Menerimanya dari klien berarti membiarkan limit rupiah dan limit dolar
	// berbeda diam-diam.
	LimitIDR int64 `json:"limit_idr"`

	Reas []ReinsurerDTO `json:"reas"`
}

// BusinessDTO adalah satu grup bisnis yang dicakup sebuah induk.
type BusinessDTO struct {
	// ID memetakan kolom IDBUSINESS. Boleh kosong — baris "TREATY INWARD" memang tidak
	// punya ID, dan dua baris produksi menyimpan NULL.
	ID string `json:"id"`

	// Nama memetakan kolom GROUPBUSINESS.
	Nama string `json:"nama"`
}

// MasterDTO adalah bentuk satu induk XOL yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterxol.Master. Memakai tipe modul langsung sebagai
// bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
type MasterDTO struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`

	// Tahun memetakan kolom TAHUN.
	Tahun string `json:"tahun"`

	// Kurs memetakan kolom KURSVALUE — rupiah per satu dolar.
	Kurs int64 `json:"kurs"`

	// Tipe memetakan kolom TYPEXOL. Boleh kosong: dua dari delapan induk produksi
	// menyimpan NULL.
	Tipe string `json:"tipe"`

	// TipeLabel adalah label Type XOL yang ditampilkan. Dikirim server supaya layar tidak
	// perlu menyalin pemetaan kode ke label — dan supaya keduanya tidak dapat berbeda.
	TipeLabel string `json:"tipe_label"`

	// RemarkPIC memetakan kolom REMARKPIC — catatan PIC saat mengajukan ke komite.
	RemarkPIC string `json:"remark_pic"`

	// Keempat berikut HANYA DIKIRIM, tidak pernah diterima dari klien. PIC dan
	// StatusKomite diisi server dari sesi saat menyimpan; Komite dan RemarkKomite diisi
	// layar persetujuan komite, yang belum dibangun.
	PIC          string `json:"pic"`
	StatusKomite string `json:"status_komite"`
	Komite       string `json:"komite"`
	RemarkKomite string `json:"remark_komite"`

	Bisnis []BusinessDTO `json:"bisnis"`
	Layer  []LayerDTO    `json:"layer"`
}

// ListResponse adalah jawaban GET /api/master/xol.
//
// Induk di dalamnya dikirim TANPA anaknya — grid hanya menampilkan kolom induk.
type ListResponse struct {
	XOL []MasterDTO `json:"xol"`

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

// SingleResponse adalah jawaban untuk satu induk: ambil, tambah, dan ubah.
type SingleResponse struct {
	XOL    MasterDTO `json:"xol"`
	Portal string    `json:"portal"`

	// Peringatan memuat pesan yang TIDAK menggagalkan penyimpanan — hari ini hanya satu
	// jenis: total share sebuah lapisan yang belum 100%.
	//
	// Ia dipisahkan tegas dari galat, dan itu inti perilaku modul ini. Layar lama pun
	// menyimpan lebih dulu baru menampilkan pesannya, dan Work Owner memutuskan perilaku
	// itu ditiru (`P-5`). Bila keduanya disatukan, layar tidak punya cara membedakan
	// "tersimpan, tetapi perhatikan ini" dari "tidak tersimpan".
	Peringatan []string `json:"peringatan,omitempty"`
}

// TypeOptionDTO adalah satu pilihan Type XOL.
type TypeOptionDTO struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`
}

// FormResponse adalah bekal awal layar: pilihan tahun dan pilihan Type XOL.
type FormResponse struct {
	Tahun  []string        `json:"tahun"`
	Tipe   []TypeOptionDTO `json:"tipe"`
	Portal string          `json:"portal"`
}

// BusinessGroupResponse adalah jawaban GET /api/master/xol/bisnis.
type BusinessGroupResponse struct {
	Bisnis []BusinessDTO `json:"bisnis"`
	Total  int           `json:"total"`
	Portal string        `json:"portal"`
}

// SaveRequest adalah isian form tambah dan ubah.
//
// ID TIDAK pernah datang dari badan permintaan: pada penambahan ia dibuat penyimpanan,
// dan pada pengubahan ia berada di jalur URL. Menerimanya dari badan berarti membiarkan
// klien memindahkan satu induk ke nomor lain.
//
// Kolom komite pun tidak diterima. PIC dan StatusKomite diisi server dari sesi saat
// menyimpan — membiarkan klien mengirimnya berarti membiarkan siapa pun mengaku sebagai
// pengaju, dan `D-59` menjadikan jejak itu satu-satunya kontrol pengimbang yang tersisa.
type SaveRequest struct {
	Nama      string `json:"nama"`
	Tahun     string `json:"tahun"`
	Kurs      int64  `json:"kurs"`
	Tipe      string `json:"tipe"`
	RemarkPIC string `json:"remark_pic"`

	Bisnis []BusinessDTO `json:"bisnis"`
	Layer  []LayerDTO    `json:"layer"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta bagian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai bagian yang salah, bukan sekadar menampilkan
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
