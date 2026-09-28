package inboxpladla

import "strings"

// Nama field JSON pada baris daftar.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar (`D-80`).
const (
	FieldClaimNo      = "no_klaim"
	FieldPolicyNo     = "no_polis"
	FieldInsured      = "nama_tertanggung"
	FieldBusinessName = "nama_bisnis"
	FieldRegisterDate = "tanggal_register"
	FieldLossDate     = "tanggal_kejadian"
	FieldPICTeknik    = "pic_teknik"
	FieldStatus       = "status"
	FieldAdviceNo     = "no_pla"
	FieldCloseNote    = "catatan_tutup"
)

// Column adalah satu kolom grid.
type Column struct {
	Key   string
	Title string

	// Date menandai kolom yang berisi tanggal, supaya layar memformatnya sebagai tanggal
	// WIB alih-alih menggambar teksnya apa adanya.
	Date bool
}

// Tab adalah satu daftar beserta penyaringnya.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul yang dibaca pengguna.
	Name string

	// Description menjelaskan isi daftarnya dalam satu kalimat.
	//
	// Layar Pega tidak punya keterangan seperti ini, dan ketiadaannya nyata akibatnya:
	// ketiga daftar menampilkan klaim yang sama-sama sudah dikirimi pemberitahuan, dan
	// tidak ada apa pun yang menjelaskan mengapa sebuah klaim ada di yang satu.
	Description string

	// Columns adalah kolom grid, berurutan seperti tampilnya.
	Columns []Column

	// AdviceKindSent menyatakan dokumen JENIS APA yang harus sudah terkirim kepada
	// reasuradur pemanggil supaya klaimnya masuk daftar ini.
	//
	//	tab PLA    "pla"  <- T_PLALIST
	//	tab DLA    "dla"  <- T_DLALIST
	//	tab Close  "pla"  <- T_PLALIST      (!) BUKAN DLA
	//
	// Baris ketiga itu mudah dikira salah ketik. `GetPNCList_PLADLAClose` memang
	// menyaring `t_plalist`, bukan `t_dlalist`, meski ia menampilkan klaim yang sudah
	// selesai. Alasannya tidak tertulis di mana pun, dan ia dibawa apa adanya (`P-5`).
	AdviceKindSent string

	// ExcludeWhenDLASent menyatakan klaim yang DLA-nya sudah terkirim DIKELUARKAN.
	//
	// **Tab PLA saja.** Itu yang membuat ketiga daftarnya tidak saling bertumpuk: begitu
	// DLA terkirim, klaimnya berpindah dari daftar PLA ke daftar DLA.
	ExcludeWhenDLASent bool

	// WorkStatus menyatakan keadaan alur kerja yang diterima daftar ini.
	WorkStatus WorkStatusFilter

	// AllReinsurerCodes menyatakan daftar ini mencocokkan SELURUH kode reasuradur milik
	// login pemanggil, bukan yang tertinggi saja.
	//
	// **Tab Close saja** (`reinscode in (…)`). Kedua tab lain memakai
	// `reinscode = (… FETCH NEXT 1 ROW ONLY)`. Perbedaan itu ada di kueri Pega dan
	// alasannya tidak tertulis; menyeragamkannya akan mengubah isi daftar.
	AllReinsurerCodes bool

	// PendingCloseBecomes1139 menyatakan kode status diganti `1139` ketika klaimnya
	// menunggu penutupan.
	//
	// **Tab DLA saja.** `CASE WHEN z.ISPENDINGCLOSE='true' THEN '1139' ELSE
	// z.statusclaim_1 END`. Kedua tab lain memakai kode aslinya.
	PendingCloseBecomes1139 bool
}

// WorkStatusFilter menyatakan keadaan alur kerja yang diterima sebuah daftar.
type WorkStatusFilter string

const (
	// WorkOpen menerima klaim yang BELUM selesai.
	//
	// `STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')`.
	WorkOpen WorkStatusFilter = "terbuka"

	// WorkOpenOrPendingClose menerima klaim yang belum selesai, DITAMBAH klaim yang
	// sudah `Resolved-Completed` tetapi masih menunggu penutupan.
	//
	// Tab DLA. Klaim menunggu penutupan itulah yang kode statusnya diganti `1139`.
	WorkOpenOrPendingClose WorkStatusFilter = "terbuka-atau-menunggu-tutup"

	// WorkClosed menerima klaim yang sudah `Resolved-Completed` dan TIDAK menunggu
	// penutupan.
	//
	// Perhatikan `Resolved-Rejected` TIDAK termasuk: klaim yang ditolak tidak muncul di
	// daftar mana pun pada layar ini. Itu perilaku Pega, dan akibatnya klaim yang sudah
	// dikirimi PLA lalu ditolak menghilang dari pandangan reasuradur tanpa satu pun
	// pemberitahuan.
	WorkClosed WorkStatusFilter = "tutup"
)

// ClosedWorkStatuses adalah kedua nilai `STATUSWORK` yang berarti klaimnya sudah selesai.
//
// Keduanya ditulis di sini, bukan di dalam SQL, supaya penyimpanan memori dan penyimpanan
// SQL tidak dapat memakai daftar yang berbeda.
var ClosedWorkStatuses = []string{"Resolved-Completed", "Resolved-Rejected"}

// CompletedWorkStatus adalah nilai `STATUSWORK` yang dipakai tab Close.
const CompletedWorkStatus = "Resolved-Completed"

// PendingCloseStatusCode adalah kode status yang menggantikan kode asli pada tab DLA
// ketika klaimnya menunggu penutupan.
const PendingCloseStatusCode = "1139"

// DefaultTab adalah tab yang terbuka pertama kali.
//
// PLA — tahap paling awal dari sudut pandang reasuradur: ia baru menerima pemberitahuan
// estimasi dan belum menerima pemberitahuan akseptasi.
const DefaultTab = "pla"

// listColumnsFor menyusun kolom daftar.
//
// Ia fungsi, bukan variabel bersama, supaya setiap tab memegang senarai kolomnya sendiri.
// Senarai bersama yang dipakai tiga tab dapat diubah salah satu pemakainya dan diam-diam
// mengubah dua yang lain.
func listColumnsFor() []Column {
	return []Column{
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldAdviceNo, Title: "No PLA"},
		{Key: FieldPolicyNo, Title: "No Polis"},
		{Key: FieldInsured, Title: "Nama Tertanggung"},
		{Key: FieldBusinessName, Title: "Bisnis"},
		{Key: FieldRegisterDate, Title: "Tanggal Register", Date: true},
		{Key: FieldLossDate, Title: "Tanggal Kejadian", Date: true},
		{Key: FieldPICTeknik, Title: "PIC Teknik"},
		{Key: FieldStatus, Title: "Status"},
	}
}

// tabs adalah ketiga daftar yang digambar layar.
var tabs = []Tab{
	{
		Code: "pla",
		Name: "PLA",
		Description: "Klaim yang PLA-nya sudah dikirimkan kepada Anda, tetapi DLA-nya " +
			"belum. Anda sudah diberi tahu nilai estimasinya dan belum diberi tahu " +
			"nilai akseptasinya.",
		Columns:            listColumnsFor(),
		AdviceKindSent:     "pla",
		ExcludeWhenDLASent: true,
		WorkStatus:         WorkOpen,
	},
	{
		Code: "dla",
		Name: "DLA",
		Description: "Klaim yang DLA-nya sudah dikirimkan kepada Anda. Termasuk klaim " +
			"yang sudah selesai tetapi masih menunggu penutupan — status klaim itu " +
			"digambar sebagai 1139.",
		Columns:                 listColumnsFor(),
		AdviceKindSent:          "dla",
		WorkStatus:              WorkOpenOrPendingClose,
		PendingCloseBecomes1139: true,
	},
	{
		Code: "close",
		Name: "Close",
		Description: "Klaim yang sudah selesai dan tidak lagi menunggu penutupan. " +
			"Disaring oleh PLA yang terkirim — bukan DLA — dan mencocokkan SELURUH " +
			"kode reasuradur milik login Anda.",
		Columns:           listColumnsFor(),
		AdviceKindSent:    "pla",
		WorkStatus:        WorkClosed,
		AllReinsurerCodes: true,
	},
}

// Tabs mengembalikan salinan ketiga daftar.
func Tabs() []Tab {
	out := make([]Tab, len(tabs))
	copy(out, tabs)
	return out
}

// FindTab mencari tab menurut kodenya.
//
// Kode dicocokkan tanpa memedulikan huruf besar-kecil dan spasi di ujung: ia datang dari
// alamat.
func FindTab(code string) (Tab, bool) {
	clean := strings.ToLower(strings.TrimSpace(code))
	for _, tab := range tabs {
		if tab.Code == clean {
			return tab, true
		}
	}
	return Tab{}, false
}

// XOLColumns adalah kolom grid "DATA PLA DLA XOL KLAIM".
//
// Ia konstanta layar, bukan milik tab: gridnya sama di ketiga tab, dan di Pega pun ia
// digambar sekali di luar ketiga daftarnya.
func XOLColumns() []Column {
	return []Column{
		{Key: "tahun", Title: "Tahun"},
		{Key: "penyebab_kerugian", Title: "Penyebab Kerugian"},
		{Key: "jenis", Title: "Jenis"},
		{Key: "tanggal_terakhir", Title: "Tanggal Terakhir", Date: true},
	}
}
