package dashboardclaim

import (
	"strings"

	"claim-pnc/internal/platform/businessline"
	"claim-pnc/internal/platform/pagination"
)

// BusinessLine adalah penyaring lini bisnis layar ini.
//
// Asalnya `Activity/SetDashboardClaim-Act.xml`, yang bercabang pada `TempView2.Remark`
// menjadi lima langkah Property-Set. Setiap cabang menyusun satu potongan klausa WHERE:
//
//	NONMBU   AND GROUPPANEL_1 IN ('003','004','006')
//	         AND c.businessgroupid NOT IN ('10008','10010','10015','10023')
//	BONDING  AND c.businessgroupid IN ('10008','10010','10015','10023')
//	PA       AND GROUPPANEL_1 = '002'
//	TRAVEL   AND GROUPPANEL_1 = '005'
//	ALL      (tanpa saringan)
//
// # Group Panel 009 tidak termasuk NONMBU, dan itu DIREPLIKASI
//
// `CONTEXT.md` mendaftar `009` sebagai varian Aneka, tetapi cabang NONMBU di atas hanya
// menyebut `003`, `004`, dan `006`. Klaim ber-Group Panel `009` karena itu TIDAK tampil pada
// pilihan Non-MBU, dan juga tidak pada pilihan mana pun selain ALL.
//
// Perilaku itu dipertahankan apa adanya sesuai `P-5`: hasil yang benar selama migrasi adalah
// hasil yang sama dengan Pega. Menambahkan `009` di sini akan mengubah angka pada layar
// manajerial tanpa ada keputusan yang mendasarinya, dan selisihnya akan muncul pada gerbang 1
// sebagai cacat yang tidak dapat dijelaskan.
//
// Apakah `009` seharusnya ikut adalah pertanyaan untuk Work Owner, bukan sesuatu yang
// diputuskan sambil menulis kode.
//
// # Kenapa modul ini punya salinannya sendiri
//
// `inboxcloseclaim.BusinessLine` isinya identik hari ini, dan justru karena itu menggodanya
// dipakai ulang. Ia tidak dipakai ulang dengan alasan yang sama yang sudah dicatat modul itu
// sendiri terhadap `inboxoutstanding`: kelimanya berasal dari activity yang BERBEDA, dan
// kesamaannya kebetulan. Bila salah satu activity kelak diubah, yang berbagi tipe akan ikut
// berubah tanpa ada yang memintanya.
type BusinessLine = businessline.Line

const (
	BusinessAll     = businessline.All
	BusinessNonMBU  = businessline.NonMBU
	BusinessBonding = businessline.Bonding
	BusinessPA      = businessline.PA
	BusinessTravel  = businessline.Travel
)

// BusinessLines mengembalikan isi dropdown lini bisnis.
func BusinessLines() []BusinessLine { return businessline.Lines() }

// ParseBusinessLine membaca pilihan lini bisnis dari isian layar.
//
// Isian kosong berarti ALL, bukan galat: layar yang baru dibuka belum memilih apa pun, dan
// "belum memilih" di sistem lama memang berarti tanpa saringan.
func ParseBusinessLine(raw string) (BusinessLine, bool) { return businessline.Parse(raw) }

// Filter adalah penyaring yang berlaku untuk keempat tile sekaligus.
//
// Ia satu bentuk, bukan satu per tile, dan itu disengaja: keempat angka pada kartu WAJIB
// menghitung populasi yang disaring sama. Bila ringkasan dan telusur dapat menyaring
// berbeda, pengguna membaca "247" lalu menemukan jumlah baris yang lain saat menelusurinya —
// persis cacat yang sudah ditemukan pada kueri hitung sistem lama.
type Filter struct {
	// Business adalah pilihan lini bisnis. Kosong diperlakukan sebagai ALL oleh Normalize.
	Business BusinessLine

	// Search adalah kotak cari gabungan atas No Klaim dan No Polis.
	//
	// Pencarian dikerjakan SERVER, bukan peramban: tabel klaim berisi puluhan juta baris
	// (`D-10`), dan menyaring satu halaman dari sepuluh akan memberi tahu pengguna bahwa
	// sesuatu tidak ada padahal ia ada di halaman lain.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi.
//
// Bawaannya 25 mengikuti `.PageSize := 25` pada activity sistem lama. Batas maksimumnya
// mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan dipenuhi.
const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
//
// Ia dipanggil SEKALI di usecase, bukan di setiap repo: kalau setiap repo menormalkan
// sendiri, dua repo dapat menormalkan berbeda dan angka ringkasan menyimpang dari isi
// telusurnya.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)

	if f.Business == "" {
		f.Business = BusinessAll
	}
	f.Limit, f.Offset = pagination.LimitOffset(f.Limit, f.Offset, DefaultLimit, MaxLimit)
	return f
}

// CountFilter mengembalikan salinan filter tanpa paginasi.
//
// Kueri hitung tidak boleh membawa LIMIT/OFFSET — ia menghitung SELURUH baris yang cocok,
// bukan baris pada halaman ini. Menyediakannya sebagai method membuat kekeliruan itu sulit
// dilakukan tanpa sengaja.
func (f Filter) CountFilter() Filter {
	f.Limit = 0
	f.Offset = 0
	return f
}
