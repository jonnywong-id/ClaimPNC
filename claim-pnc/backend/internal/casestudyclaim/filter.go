package casestudyclaim

import (
	"strings"
	"time"
)

// BusinessScope adalah pilihan dropdown **Bisnis**.
//
// # Nilainya bukan tebakan
//
// Kodenya terbaca dari `Activity/StudyClaim_act-Act.xml`, yang mencocokkan
// `TempLaporan.StatusReceiver` lalu menyisipkan potongan SQL yang berbeda untuk tiap
// nilai. Labelnya terbaca dari `Property/StatusReceiver_property.xml`, yang memuat
// `pyPromptTableList` lengkap:
//
//	kode   label      potongan SQL di Pega
//	-----  ---------  -------------------------------------------------------------
//	002    PA         and b.grouppanel in ('002')
//	005    TRAVEL     and b.grouppanel in ('005')
//	346    NONMBU     and b.grouppanel in ('003','004','006','009')
//	                  and b.businesscode NOT IN ('10145','10168','10165','10164','10053')
//	003    BONDING    and b.GROUPPANEL = '003'
//	                  and b.BUSINESSCODE IN ('10076','10077','10007','10011','10083',
//	                                         '10141','10131','10126','10055','10075')
//	(kosong)          tanpa penyaring sama sekali
//
// Keempat label itu sama persis dengan nilai `TYPE_BUSINESS` pada master ambang komite
// (`Database/emailkomite.csv`) — PA, TRAVEL, NONMBU, BONDING. Kesamaan itu bukan
// kebetulan, dan ia yang menguatkan bahwa pembacaannya benar.
//
// # Kode `346` dan `003` mudah salah dibaca
//
// `346` BUKAN kode Group Panel; ia singkatan dari gabungan panel 3, 4, dan 6 — ditambah
// 9, yang tidak ikut disebut namanya. Dan `003` di sini BUKAN "Aneka" melainkan BONDING:
// yang membedakannya dari bagian Aneka pada NONMBU adalah kesepuluh `BUSINESSCODE`-nya.
//
// Kedua daftar kode bisnis itu di-hardcode di dalam rule Pega. `D-15` menetapkannya
// menjadi master data (`F-4`), yang belum ada; sampai itu tiba keduanya tinggal di berkas
// .sql, tempat pemetaannya ke kueri lama dapat dibaca berdampingan.
//
// # NONMBU dan BONDING BERTUMPANG TINDIH
//
// Keempat pilihan terbaca seperti pembagian yang saling lepas. Ia tidak:
//
//	NONMBU   panel 003/004/006/009 MINUS lima kode bisnis
//	BONDING  panel 003 DAN sepuluh kode bisnis tertentu
//
// Kesepuluh kode BONDING tidak ada di dalam daftar lima yang dikecualikan NONMBU, sehingga
// **setiap klaim Bonding memenuhi kedua cakupan**. Memilih NONMBU menampilkannya juga.
//
// Akibat yang perlu disadari: menjumlahkan hasil keempat pilihan menghitung sebagian klaim
// DUA KALI. Itu perilaku kueri lama, direplikasi apa adanya (`P-5`), dan dicatat sebagai
// temuan — bukan diperbaiki, karena memperbaikinya menghilangkan baris yang di layar lama
// terlihat. Diuji di `repo/memory/memory_test.go`.
type BusinessScope string

const (
	// ScopeAll adalah pilihan `----- Pilih -----` di Pega — tanpa penyaring apa pun.
	//
	// Ia sengaja bernilai kosong supaya "tidak dipilih" dan "semua bisnis" adalah hal
	// yang sama, persis seperti di Pega: `TempLaporan.StatusReceiver == ""` menyetel
	// potongan SQL-nya menjadi teks kosong.
	ScopeAll BusinessScope = ""

	ScopePA      BusinessScope = "002"
	ScopeTravel  BusinessScope = "005"
	ScopeNonMBU  BusinessScope = "346"
	ScopeBonding BusinessScope = "003"
)

// businessOption adalah satu pilihan dropdown Bisnis beserta labelnya.
type businessOption struct {
	scope BusinessScope
	label string
}

// businessOrder adalah urutan pilihan PERSIS seperti `pyPromptTableList` pada
// `Property/StatusReceiver_property.xml` — PA, TRAVEL, NONMBU, BONDING.
//
// Menata ulangnya, misalnya menaruh NONMBU di depan karena ia yang terbesar, akan
// memindahkan pilihan yang sudah dihafal petugas.
//
// `ScopeAll` TIDAK ada di daftar ini: di Pega ia bukan baris pada tabel nilai melainkan
// `pyNoSelectionText` pada kontrolnya. Layar menggambarnya sebagai pilihan kosong di
// paling atas, sama seperti Pega.
var businessOrder = []businessOption{
	{ScopePA, "PA"},
	{ScopeTravel, "TRAVEL"},
	{ScopeNonMBU, "NONMBU"},
	{ScopeBonding, "BONDING"},
}

// BusinessOptions mengembalikan isi dropdown Bisnis dalam urutan layar.
func BusinessOptions() []Option {
	options := make([]Option, 0, len(businessOrder))
	for _, item := range businessOrder {
		options = append(options, Option{Code: string(item.scope), Label: item.label})
	}
	return options
}

// FindBusinessScope mencari pilihan dari kodenya.
//
// Kosong berarti ScopeAll dan itu SAH — lihat komentar pada ScopeAll. Kode yang tidak
// dikenal DITOLAK, tidak diam-diam diartikan "semua": salah ketik yang jatuh ke "semua"
// menghasilkan daftar yang tampak wajar tetapi jauh lebih luas daripada yang diminta, dan
// pada layar berisi klaim di atas Rp 5 miliar itu bukan selisih yang pantas diam.
func FindBusinessScope(code string) (BusinessScope, bool) {
	clean := strings.TrimSpace(code)
	if clean == "" {
		return ScopeAll, true
	}
	for _, item := range businessOrder {
		if string(item.scope) == clean {
			return item.scope, true
		}
	}
	return ScopeAll, false
}

// ClaimStatusFilter adalah pilihan dropdown **Status**.
//
// # Nilainya pun bukan tebakan
//
// `Activity/FilterStudyClaim_act-Act.xml` mengisi daftarnya dengan tepat dua nilai —
// `"CLAIM ON PROGRESS/ ACCEPT"` dan `"REJECT"` — dan `Activity/StudyClaim_act-Act.xml`
// mencocokkan keduanya lalu menyisipkan potongan SQL:
//
//	pilihan di layar             potongan SQL di Pega
//	---------------------------  ------------------------
//	(kosong)                     tanpa penyaring
//	CLAIM ON PROGRESS/ ACCEPT    and a.stsklaim != '3'
//	REJECT                       and a.stsklaim  = '3'
//
// # Kodenya TIDAK memakai teks panjang itu
//
// Di Pega, nilai yang dibandingkan adalah teks yang ditampilkan — termasuk spasi
// menggantung pada `"CLAIM ON PROGRESS/ ACCEPT"`. Teks tampilan yang dipakai sebagai kode
// berarti memperbaiki satu spasi akan mematikan penyaringnya. Di sini keduanya
// dipisahkan: kode stabil, label mengikuti Pega apa adanya termasuk spasinya.
type ClaimStatusFilter string

const (
	// StatusAny adalah "tidak memilih" — tanpa penyaring status.
	StatusAny ClaimStatusFilter = ""

	// StatusInProgressOrAccepted adalah `a.stsklaim != '3'`.
	//
	// Perhatikan: ia SEMUA YANG BUKAN ditolak, bukan "yang bernilai 0 atau 1". Nilai
	// asing apa pun ikut masuk ke sini, dan itu direplikasi apa adanya.
	StatusInProgressOrAccepted ClaimStatusFilter = "progress-accept"

	// StatusRejected adalah `a.stsklaim = '3'`.
	StatusRejected ClaimStatusFilter = "reject"
)

type statusOption struct {
	status ClaimStatusFilter
	label  string
}

// statusOrder adalah urutan pilihan Status PERSIS seperti `FilterStudyClaim_act` menyusun
// `TempStatus.pxResults`: yang berjalan/diterima lebih dulu, lalu yang ditolak.
//
// Labelnya disalin apa adanya, **termasuk spasi setelah garis miring** pada yang pertama.
// Itu bukan salah ketik di sini; ia ada di
// `Activity/FilterStudyClaim_act-Act.xml` dan `D-13` menetapkan teks layar mengikuti Pega.
var statusOrder = []statusOption{
	{StatusInProgressOrAccepted, "CLAIM ON PROGRESS/ ACCEPT"},
	{StatusRejected, "REJECT"},
}

// StatusOptions mengembalikan isi dropdown Status dalam urutan layar.
func StatusOptions() []Option {
	options := make([]Option, 0, len(statusOrder))
	for _, item := range statusOrder {
		options = append(options, Option{Code: string(item.status), Label: item.label})
	}
	return options
}

// FindClaimStatus mencari pilihan Status dari kodenya; kosong berarti StatusAny.
func FindClaimStatus(code string) (ClaimStatusFilter, bool) {
	clean := strings.TrimSpace(strings.ToLower(code))
	if clean == "" {
		return StatusAny, true
	}
	for _, item := range statusOrder {
		if string(item.status) == clean {
			return item.status, true
		}
	}
	return StatusAny, false
}

// Option adalah satu pilihan dropdown: kode yang dikirim, label yang dibaca.
//
// Dipisahkan dengan sengaja. Kode stabil dan boleh dipakai di URL; label adalah teks layar
// yang mengikuti Pega dan boleh berubah tanpa merusak klien mana pun.
type Option struct {
	Code  string
	Label string
}

// Filter adalah penyaring dan paginasi yang diminta layar.
type Filter struct {
	// FromYear dan ToYear adalah rentang **TAHUN REGISTRASI**, empat digit.
	//
	// # Kenapa tahun, padahal isiannya tanggal
	//
	// Karena itulah yang dilakukan kueri lama. Pengguna memilih dua TANGGAL, lalu
	// `RDB List/BrowseClaimStudy-SQL.xml` membuang hari dan bulannya:
	//
	//	A.THNREGIS BETWEEN TO_CHAR(TO_DATE(awal),'yyyy')
	//	               AND TO_CHAR(TO_DATE(akhir),'yyyy')
	//
	// Memilih 01/01/2024 sampai 31/01/2024 karena itu mengembalikan **seluruh tahun
	// 2024**, bukan bulan Januari saja. Perilakunya direplikasi (`P-5`, keputusan Work
	// Owner 2026-09-26); yang ditambahkan hanyalah keterangan di layar, supaya pengguna
	// tidak mengira sedang menyaring per hari.
	//
	// Pembuangan hari dan bulan terjadi di lapisan transport, bukan di sini: yang sampai
	// ke domain sudah berupa tahun. Dengan begitu penyaringnya dapat diuji tanpa
	// menyentuh zona waktu sama sekali.
	FromYear string
	ToYear   string

	Business BusinessScope
	Status   ClaimStatusFilter

	Limit  int
	Offset int
}

// Batas paginasi.
//
// DefaultLimit 20 mengikuti `pyPageSize` pada `Section/PNCStudyClaim-Section.xml` — bukan
// angka yang dipilih di sini. Ukuran halaman di sistem lama berbeda-beda per layar, dan
// yang berlaku untuk layar ini adalah 20.
//
// MaxLimit mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipangkas diam-diam. Normalize memangkasnya karena di sini nilainya sudah melewati
// pemeriksaan transport; penolakannya terjadi di sana.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// MaxExportBatch adalah banyaknya baris sekali ambil saat mengekspor.
//
// Lebih longgar dari MaxLimit karena unduhan tidak dilihat manusia sebagai halaman — ia
// dibaca sekumpulan demi sekumpulan lalu langsung dialirkan ke berkas.
const MaxExportBatch = 500

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
func (f Filter) Normalize() Filter {
	f.FromYear = strings.TrimSpace(f.FromYear)
	f.ToYear = strings.TrimSpace(f.ToYear)

	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxExportBatch {
		f.Limit = MaxExportBatch
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}

// YearOf mengambil tahun sebuah tanggal sebagai teks empat digit.
//
// Ia ada di paket domain supaya aturan "hanya tahunnya yang dipakai" hidup di SATU tempat
// dan dapat diuji, alih-alih tersebar di handler daftar dan handler unduhan.
//
// Zona waktu diserahkan pemanggil, tidak dibaca dari jam sistem. Tanpa itu, tanggal
// 1 Januari yang dipilih pengguna WIB akan terbaca sebagai 31 Desember tahun sebelumnya
// begitu waktunya sempat melewati UTC — kelas kesalahan yang melahirkan 101 penyesuaian
// tujuh jam di sistem lama (`R-12`).
func YearOf(at time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return at.In(loc).Format("2006")
}
