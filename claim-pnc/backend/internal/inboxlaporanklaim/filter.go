package inboxlaporanklaim

import "strings"

// BusinessLine adalah pilihan dropdown "Bisnis" di atas daftar.
//
// # Asalnya, apa adanya
//
// `Activity/SetListRCV_Act-Act.xml` mengisi `tempQuery.pyNote` dengan salah satu dari
// lima potongan SQL berikut, lalu menempelkannya ke kesembilan kueri lewat `{ASIS:}`:
//
//	""                                                                     seluruhnya
//	"AND GROUPPANEL_1='002'"                                               PA
//	"AND GROUPPANEL_1='005'"                                               Travel
//	"AND GROUPPANEL_1 in ('003','004','006','009')
//	   AND c.businessgroupid NOT IN ('10008','10010','10015','10023')"      Non-MBU umum
//	"AND c.businessgroupid IN ('10008','10010','10015','10023')"            kelompok khusus
//
// Kode Group Panel-nya cocok dengan `CONTEXT.md`: 002 Personal Accident, 005 Travel,
// dan 003/004/006/009 sisa lini Non-MBU (Aneka, Marine Cargo, Fire).
//
// # Satu label yang TIDAK saya karang
//
// Keempat `businessgroupid` itu — 10008, 10010, 10015, 10023 — dipakai sebagai kelompok
// tersendiri yang dikecualikan dari Non-MBU umum, tetapi **tidak ada satu pun rule di
// export yang menyebut namanya**, dan POOLDATA.BUSINESSGROUP tidak ikut dikirim. Label
// tab itu karena itu dibiarkan menyebut apa adanya dan ditandai terbuka, bukan ditebak
// menjadi "Kredit" atau "Bonding" yang kebetulan masuk akal — tebakan yang salah di
// dropdown penyaring akan membuat petugas mengira sedang melihat lini yang bukan.
// Pertanyaannya ditujukan ke Work Owner.
type BusinessLine string

const (
	// BusinessLineAll: tanpa penyaring lini bisnis.
	BusinessLineAll BusinessLine = ""

	// BusinessLinePA: Group Panel 002.
	BusinessLinePA BusinessLine = "pa"

	// BusinessLineTravel: Group Panel 005.
	BusinessLineTravel BusinessLine = "travel"

	// BusinessLineNonMBU: Group Panel 003, 004, 006, 009 di luar kelompok khusus.
	BusinessLineNonMBU BusinessLine = "non-mbu"

	// BusinessLineSpecialGroup: kelompok businessgroupid 10008, 10010, 10015, 10023.
	BusinessLineSpecialGroup BusinessLine = "kelompok-khusus"
)

// businessLineDefinition memetakan pilihan ke kode yang benar-benar disaring.
type businessLineDefinition struct {
	line BusinessLine
	name string

	// groupPanel adalah kode Group Panel yang diterima. Kosong berarti tidak disaring.
	groupPanel []string

	// businessGroupIn menyaring businessgroupid yang HARUS termasuk.
	businessGroupIn []string

	// businessGroupNotIn menyaring businessgroupid yang harus DIKECUALIKAN.
	businessGroupNotIn []string
}

// specialBusinessGroup adalah keempat kode kelompok khusus, disebut satu kali saja.
//
// Ia dipakai DUA arah — sebagai penyaring masuk pada kelompok khusus, dan sebagai
// pengecualian pada Non-MBU umum. Menuliskannya dua kali berarti keduanya dapat
// berbeda saat salah satu diubah, dan baris yang jatuh di antaranya akan hilang dari
// kedua tab tanpa satu pun tanda.
var specialBusinessGroup = []string{"10008", "10010", "10015", "10023"}

var businessLineOrder = []businessLineDefinition{
	{BusinessLineAll, "Semua bisnis", nil, nil, nil},
	{BusinessLineNonMBU, "Non-MBU (Aneka, Marine, Fire)", []string{"003", "004", "006", "009"}, nil, specialBusinessGroup},
	{BusinessLineSpecialGroup, "Kelompok bisnis khusus", nil, specialBusinessGroup, nil},
	{BusinessLinePA, "Personal Accident", []string{"002"}, nil, nil},
	{BusinessLineTravel, "Travel", []string{"005"}, nil, nil},
}

// ListBusinessLines mengembalikan seluruh pilihan dropdown "Bisnis".
func ListBusinessLines() []BusinessLineInfo {
	result := make([]BusinessLineInfo, 0, len(businessLineOrder))
	for _, d := range businessLineOrder {
		result = append(result, BusinessLineInfo{Line: d.line, Name: d.name})
	}
	return result
}

// BusinessLineInfo adalah satu pilihan dropdown "Bisnis".
type BusinessLineInfo struct {
	Line BusinessLine
	Name string
}

// FindBusinessLine mencari pilihan dari kodenya; false bila tidak dikenal.
func FindBusinessLine(code string) (BusinessLine, bool) {
	clean := BusinessLine(strings.ToLower(strings.TrimSpace(code)))
	for _, d := range businessLineOrder {
		if d.line == clean {
			return d.line, true
		}
	}
	return "", false
}

// Criteria menyebut kode yang disaring sebuah pilihan lini bisnis.
//
// Repo memakainya untuk menyusun penyaring; ia sengaja mengembalikan DAFTAR KODE, bukan
// potongan SQL. Potongan SQL yang lewat batas modul adalah persis pola `{ASIS:}` yang
// membuka celah injeksi di sistem lama (utang teknis §4.5), dan modul domain tidak boleh
// mengetahui SQL sama sekali.
func (b BusinessLine) Criteria() (groupPanel, businessGroupIn, businessGroupNotIn []string) {
	for _, d := range businessLineOrder {
		if d.line == b {
			return d.groupPanel, d.businessGroupIn, d.businessGroupNotIn
		}
	}
	return nil, nil, nil
}

// Name mengembalikan label pilihan.
func (b BusinessLine) Name() string {
	for _, d := range businessLineOrder {
		if d.line == b {
			return d.name
		}
	}
	return string(b)
}

// Filter adalah seluruh penyaring yang dipakai satu permintaan daftar.
type Filter struct {
	// Category menentukan tab; wajib terisi.
	Category Category

	// RegionCode menyaring menurut kanwil — dropdown "Pilih Kanwil".
	//
	// Asalnya `tempQuery.Remark`:
	//   AND a.KODECABANG_1 IN (SELECT ID FROM pooldata.branch WHERE basterritory=<kode>)
	//
	// Kosong berarti seluruh kanwil.
	RegionCode string

	// BranchCode menyaring menurut satu cabang.
	//
	// Asalnya `tempQuery.NoKTP`, yang di sistem lama SELALU diisi cabang milik pemanggil
	// sendiri (hasil `GetIDCabang` atas OperatorID). Ia karena itu bukan sekadar
	// kenyamanan: ia BATAS DATA — petugas cabang hanya melihat berkas cabangnya.
	//
	// Lihat catatan pada Filter.Clean tentang kenapa ia tidak boleh diisi dari layar.
	BranchCode string

	// BusinessLine menyaring menurut lini bisnis polis.
	BusinessLine BusinessLine

	// Keyword mencari satu berkas menurut nomor registernya.
	//
	// Asalnya `TempContents.Keyword` := " And a.pyid = '<Param.Search>'" — perhatikan
	// `=`, bukan `LIKE`. Pencariannya PERSIS, bukan sebagian, dan perilaku itu
	// dipertahankan: mengubahnya menjadi pencarian sebagian akan membuat kueri berhenti
	// memakai index pada tabel berpuluh juta baris (`D-10`).
	Keyword string

	// Operator adalah identitas pemanggil, dipakai ketiga tab komunikasi.
	//
	// `ViewRejectKomunikasiUser` menyaring `a.pxcreateoperator = <saya>` dan
	// membandingkan `b.sender` terhadap nilai yang sama. Tanpa ini, tab komunikasi
	// menampilkan percakapan milik orang lain.
	Operator string
}

// Clean memangkas spasi dan menyeragamkan besar-kecil huruf setiap penyaring.
func (f Filter) Clean() Filter {
	return Filter{
		Category:     f.Category,
		RegionCode:   strings.TrimSpace(f.RegionCode),
		BranchCode:   strings.TrimSpace(f.BranchCode),
		BusinessLine: f.BusinessLine,
		Keyword:      strings.TrimSpace(f.Keyword),
		Operator:     strings.TrimSpace(f.Operator),
	}
}

// DefaultPageSize adalah banyaknya baris per halaman bila pemanggil tidak menyebutnya.
//
// Sistem lama memakai `pyRDLPageSize = 10` pada grid-nya
// (`Section/ViewStatusReceiveDocument-Section.xml`), dan angka itu dipertahankan supaya
// jumlah baris per layar sama seperti yang sudah dikenal petugas.
const DefaultPageSize = 10

// MaxPageSize membatasi permintaan halaman raksasa.
//
// `10-API-STRATEGY.md` §4 menetapkan batas 100 dan menolak permintaan yang lebih besar,
// bukan memenuhinya. Batas itu bukan kerapian: tanpa batas, satu permintaan dapat
// menarik puluhan juta baris ke memori aplikasi (`D-10`).
const MaxPageSize = 100

// Pagination adalah halaman yang diminta.
//
// Sistem lama menghitungnya begini (`SetListRCV_Act`):
//
//	.FirstRow := ((.CurrentIndex-1) * .PageSize) + 1
//	.LastRow  := (.CurrentIndex * .PageSize)
//
// lalu menempelkan " WHERE rn >= <FirstRow> AND rn <= <LastRow>" ke kueri yang sudah
// memberi nomor barisnya lewat ROWNUM. Perhitungannya dipertahankan; cara memotongnya
// TIDAK — `ROWNUM` diganti `OFFSET … FETCH NEXT … ROWS ONLY` karena `D-20` menuntut satu
// set SQL yang berjalan di Oracle 19c maupun PostgreSQL 17+.
type Pagination struct {
	// Page dihitung mulai 1, seperti `CurrentIndex` di sistem lama.
	Page int

	// Size adalah banyaknya baris per halaman.
	Size int
}

// Clean menyehatkan halaman yang diminta.
//
// Nilai yang tidak masuk akal DIPERBAIKI, bukan ditolak: halaman adalah kendali
// tampilan, dan menolak permintaan karena nomor halaman nol hanya memindahkan kerepotan
// ke layar tanpa melindungi apa pun. Ukuran halaman yang melewati batas dipotong ke
// MaxPageSize — batasnya melindungi peladen, dan memotong sudah cukup untuk itu.
func (p Pagination) Clean() Pagination {
	page, size := p.Page, p.Size
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return Pagination{Page: page, Size: size}
}

// Offset mengembalikan banyaknya baris yang dilewati sebelum halaman ini.
func (p Pagination) Offset() int {
	clean := p.Clean()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman hasil beserta keterangan letaknya.
type Page struct {
	Report []ClaimReport

	// Total adalah banyaknya baris yang cocok dengan penyaring, BUKAN banyaknya baris
	// pada halaman ini. Ia yang membuat layar dapat menggambar nomor halaman.
	Total int

	Pagination Pagination
}

// TotalPages menghitung banyaknya halaman.
func (p Page) TotalPages() int {
	size := p.Pagination.Clean().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}
