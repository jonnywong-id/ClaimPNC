package dashboardclaim

import "strings"

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
type BusinessLine string

const (
	BusinessAll     BusinessLine = "ALL"
	BusinessNonMBU  BusinessLine = "NONMBU"
	BusinessBonding BusinessLine = "BONDING"
	BusinessPA      BusinessLine = "PA"
	BusinessTravel  BusinessLine = "TRAVEL"
)

// businessLines adalah kelimanya dalam urutan tampilnya di dropdown.
var businessLines = []BusinessLine{
	BusinessAll, BusinessNonMBU, BusinessBonding, BusinessPA, BusinessTravel,
}

// BusinessLines mengembalikan isi dropdown lini bisnis.
func BusinessLines() []BusinessLine {
	result := make([]BusinessLine, len(businessLines))
	copy(result, businessLines)
	return result
}

// Label adalah teks yang dibaca pengguna pada dropdown.
func (b BusinessLine) Label() string {
	switch b {
	case BusinessAll:
		return "Semua Lini Bisnis"
	case BusinessNonMBU:
		return "Non-MBU"
	case BusinessBonding:
		return "Bonding"
	case BusinessPA:
		return "Personal Accident"
	case BusinessTravel:
		return "Travel"
	default:
		return string(b)
	}
}

// ParseBusinessLine membaca pilihan lini bisnis dari isian layar.
//
// Isian kosong berarti ALL, bukan galat: layar yang baru dibuka belum memilih apa pun, dan
// "belum memilih" di sistem lama memang berarti tanpa saringan.
func ParseBusinessLine(raw string) (BusinessLine, bool) {
	value := BusinessLine(strings.ToUpper(strings.TrimSpace(raw)))
	if value == "" {
		return BusinessAll, true
	}
	for _, known := range businessLines {
		if known == value {
			return value, true
		}
	}
	return "", false
}

// CashierStatus adalah penyaring "Status Transfer" pada panel penyaring.
//
// Namanya BUKAN `TransferStatus`, dan itu disengaja dua kali. Pertama, nama itu sudah
// dipakai `transfer.go` untuk keadaan sebuah permintaan pemindahan penugasan — hal yang
// sama sekali berbeda. Kedua, yang disaring di sini adalah **transfer ke kasir**, yang di
// `CONTEXT.md` memang bernama `TransferCashierStatus`.
//
// Ia TIDAK dibaca dari sebuah kolom. `Activity/GCNMGetManagerCase_Act-Act.xml:6491` dan
// `:6862` menyusun klausanya sebagai sub-kueri atas tabel adjustment:
//
//	SUDAH   AND EXISTS     (SELECT CLAIMID FROM POOLDATA.T_CLAIM_ADJUSTMENT B
//	                         WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
//	                           AND B.CLAIMID = A.PZINSKEY)
//	BELUM   AND not EXISTS (… sama persis …)
//
// Artinya "sudah transfer" berarti **ada sekurang-kurangnya satu baris settlement** yang
// sudah ditransfer ke kasir — bukan bahwa seluruh baris sudah. Satu klaim dapat memuat
// banyak baris settlement, dan klaim yang baru satu barisnya ditransfer tetap terhitung
// SUDAH. Itu perilaku Pega dan direplikasi apa adanya (`P-5`).
type CashierStatus string

const (
	CashierAny  CashierStatus = ""
	CashierDone CashierStatus = "SUDAH"
	CashierTodo CashierStatus = "BELUM"
)

// PaymentStatus adalah penyaring "Status Pembayaran".
//
// Asalnya `Activity/GCNMGetManagerCase_Act-Act.xml:7197` dan `:7485`:
//
//	LUNAS       AND A.STATUSCLAIM_1  = '1163'
//	BELUM LUNAS AND A.STATUSCLAIM_1 != '1163'
//
// `1163` adalah **Paid** pada master status klaim (`R-06`, 33 kode `1134`–`1166`). Jadi
// "lunas" di sini berarti status klaimnya Paid — bukan hasil penjumlahan nilai yang dibayar.
type PaymentStatus string

const (
	PaymentAny    PaymentStatus = ""
	PaymentPaid   PaymentStatus = "LUNAS"
	PaymentUnpaid PaymentStatus = "BELUM"
)

var (
	cashierStatuses = []CashierStatus{CashierAny, CashierDone, CashierTodo}
	paymentStatuses = []PaymentStatus{PaymentAny, PaymentPaid, PaymentUnpaid}
)

// CashierStatuses dan PaymentStatuses mengembalikan isi kedua dropdown.
func CashierStatuses() []CashierStatus {
	result := make([]CashierStatus, len(cashierStatuses))
	copy(result, cashierStatuses)
	return result
}

func PaymentStatuses() []PaymentStatus {
	result := make([]PaymentStatus, len(paymentStatuses))
	copy(result, paymentStatuses)
	return result
}

// Label adalah teks yang dibaca pengguna.
//
// Teks cabang default — pilihan "belum dipilih" — disalin dari elemen `pyNoSelectionText`
// pada `Section/FilterDashboardClaim_sec-Section.xml` (`D-13`), termasuk huruf besar dan
// tanda hubungnya. Section-nya diterima 2026-10-08.
//
// Layar MENYARING pilihan bernilai kosong ini dan menggambar baris kosongnya sendiri, jadi
// teks di sini tidak pernah terbaca pengguna hari ini. Ia tetap disamakan supaya jawaban
// `/penyaring` tidak membantah apa yang tergambar di layar.
func (t CashierStatus) Label() string {
	switch t {
	case CashierDone:
		return "Sudah Transfer"
	case CashierTodo:
		return "Belum Transfer"
	default:
		return "---PILIH STATUS TRANSFER---"
	}
}

func (p PaymentStatus) Label() string {
	switch p {
	case PaymentPaid:
		return "Lunas"
	case PaymentUnpaid:
		return "Belum Lunas"
	default:
		return "---PILIH STATUS PEMBAYARAN---"
	}
}

// ParseCashierStatus dan ParsePaymentStatus membaca pilihan dropdown.
//
// Keduanya MENOLAK nilai yang tidak dikenal, sementara Pega memperlakukan nilai apa pun
// selain `"LUNAS"` dan kosong sebagai "belum lunas". Perbedaan itu disengaja dan tidak
// menyentuh perilaku bisnis: nilai di layar lama hanya dapat berasal dari dropdown, sehingga
// nilai lain hanya mungkin muncul dari URL yang dirakit tangan. Menerimanya diam-diam akan
// menyaring sesuatu yang tidak diminta siapa pun.
func ParseCashierStatus(raw string) (CashierStatus, bool) {
	value := CashierStatus(strings.ToUpper(strings.TrimSpace(raw)))
	for _, known := range cashierStatuses {
		if known == value {
			return value, true
		}
	}
	return "", false
}

func ParsePaymentStatus(raw string) (PaymentStatus, bool) {
	value := PaymentStatus(strings.ToUpper(strings.TrimSpace(raw)))
	for _, known := range paymentStatuses {
		if known == value {
			return value, true
		}
	}
	return "", false
}

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

	// Ketiga penyaring berikut adalah isian teks pada panel penyaring grid Outstanding —
	// local action `FilterDashboardClaim`.
	//
	// Section-nya DITERIMA 2026-10-08, dan kelima label beserta ikatannya kini terbaca
	// langsung dari `Section/FilterDashboardClaim_sec-Section.xml`:
	//
	//	Nopolis            TempInputFilter.CaseID
	//	No Klaim           TempInputFilter.ClaimNo
	//	PIC                TempInputFilter.ClaimID
	//	Status Transfer    TempInputFilter.CityID
	//	Status Pembayaran  TempInputFilter.City
	//
	// Kelimanya cocok dengan yang sudah dibangun di sini, termasuk pemetaan Cashier→CityID
	// dan Payment→City yang sebelumnya hanya DISIMPULKAN dari urutan bidang.
	//
	// `Flow Action/FilterDashboardClaim-FA.xml` juga diterima, dan ia membuka **section yang
	// sama** dengan `FilterDashboardClaimclose`. Panel penyaring Outstanding dan Close Claim
	// memang SATU, bukan dua — itu membenarkan satu bentuk Filter untuk keduanya.
	//
	// Nama propertinya di layar lama MENYESATKAN — tidak mencerminkan isinya sama sekali,
	// persis utang teknis 4.2. Di sini ketiganya dinamai menurut ISINYA (`D-19`).
	PolicyNumber string
	ClaimNumber  string
	TechnicalPIC string

	// Kedua dropdown pada panel yang sama — `TempInputFilter.CityID` (Status Transfer) dan
	// `TempInputFilter.City` (Status Pembayaran). Kosong berarti tanpa saringan.
	//
	// Opsinya tidak diambil dari section itu melainkan dari klausa SQL yang dibangunnya di
	// `Activity/GCNMGetManagerCase_Act-Act.xml` — rinciannya ada di CashierStatus serta
	// PaymentStatus di atas, lengkap dengan nomor baris.
	Cashier CashierStatus
	Payment PaymentStatus

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
	f.PolicyNumber = strings.TrimSpace(f.PolicyNumber)
	f.ClaimNumber = strings.TrimSpace(f.ClaimNumber)
	f.TechnicalPIC = strings.TrimSpace(f.TechnicalPIC)

	if f.Business == "" {
		f.Business = BusinessAll
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
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
