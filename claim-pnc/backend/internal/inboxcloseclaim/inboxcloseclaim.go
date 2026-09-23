// Package inboxcloseclaim adalah inti modul Inbox Close Claim.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `MENU_ID 59` pada `Database/m_menu_aplikasi_pnc.csv` berbunyi **"Inbox Close Claim"**,
// dan `Section/InboxManagerReopen1_Sec-Section.xml` memuat judul yang sama persis di dalam
// layarnya. `D-81` menetapkan nama modul mengikuti nama yang dipakai Work Owner.
//
// # HARNESS-NYA TIDAK ADA DI EXPORT
//
// `InboxCloseClaim_Harness` dirujuk butir menu
// (`Navigation/pyCaseWorkerNavigation-Navigation.xml:16695`) tetapi **tidak ada satu pun
// berkasnya** di antara 74 harness yang diekspor. Ia salah satu dari sembilan yang sudah
// tercatat hilang di `frontend/src/app/menu/registry.ts` dan diperjelas `K-33`/`R-16`.
//
// Preseden penanganannya sudah ada: `InboxOutstanding_Harness` juga hilang, dan modulnya
// tetap dibangun dari artefak yang memang ada. Modul ini menempuh jalan yang sama —
// **tidak ada satu pun perilaku yang dikarang**. Yang dipakai:
//
//	Database/m_menu_aplikasi_pnc.csv:59              nama dan kelompok butir menu
//	Navigation/pyCaseWorkerNavigation-Navigation.xml butir menu -> activity
//	Activity/GCNMGetManagerReopenCase_Act-Act.xml    SELURUH penyaring dan batas lini
//	RDB List/GcnmBrowseReopenCase_SQL-SQL.xml        kueri daftar
//	RDB List/GCNMCountCloseClaim-SQL.xml             kueri hitung
//	Section/InboxManagerReopen1_Sec-Section.xml      judul layar, 11 kolom, 6 tombol
//	Activity/ExportDataCloseClaim-Act.xml            Export to Excel
//	When/IsManagerPNC_CLOSE-When.xml                 siapa yang melihat menunya
//
// # Apa itu "Close Claim"
//
// Definisinya mengikat dan terbaca dari satu baris —
// `RDB List/GcnmBrowseReopenCase_SQL-SQL.xml`:
//
//	AND a.pystatuswork IN ('Resolved-Completed', 'Resolved-Rejected')
//
// Jadi ia kebalikan TEPAT dari Inbox Outstanding, yang menyaring `NOT IN` atas dua nilai
// yang sama. Isinya **klaim yang sudah tutup**, supaya penyelia dapat membukanya kembali.
//
// Ia BUKAN Inbox menurut `D-79`: barisnya bukan pekerjaan pemanggil, tidak hilang setelah
// ditindaklanjuti, dan tidak punya tenggat. Namanya tetap "Inbox Close Claim" karena
// itulah nama butir menunya, dan `D-13` menetapkan teks yang dilihat pengguna mengikuti
// layar lama.
//
// # Sumber datanya: DATAPEGA.PC_ASM_FW_GCNMFW_WORK
//
// Ditetapkan Work Owner 2026-09-23, sama dengan kueri lamanya. Tiga tabel, tanpa join ke
// worklist:
//
//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK  objek kerja klaim
//	POOLDATA.BUSINESS               lini bisnis
//	POOLDATA.BUSINESSGROUP          kelompok bisnis
//
// Pola ini sudah dipakai `inboxadmin`, `inboxlaporanklaim`, `inboxmanagerreceivepucl`, dan
// `komite`. `POOLDATA.T_CLAIMLIST_ADMIN` sengaja TIDAK dipakai di sini: ia baru terisi
// 1.014 dari 7.703 klaim (13%) dan belum memiliki `STATUSCLAIM_1` sampai migrasi `0005`
// tahap 1 dijalankan DBA — padahal kolom itulah yang dibutuhkan penyaring Status Bayar.
//
// # Alias Pega tidak dibawa masuk
//
// Kueri lama mengalias kolom secara MENYESATKAN, dan di sini lebih parah daripada di modul
// mana pun sebelumnya:
//
//	a.userteknis_1      AS "ReinsurerName"   bukan reasuradur — PIC Teknik
//	a.pxCreateOpName    AS "MOName"          bukan Marketing Officer — Admin PNC
//	dateofloss_1        AS "CoverNo"         bukan nomor cover — TANGGAL kejadian
//	a.pzinskey          AS "CaseID"          kunci teknis
//	qqname              AS "CustomerName"
//
// Tidak satu pun dibawa; nama di sini mengikuti padanan Inggris dari `CONTEXT.md`
// (`D-19`, `D-80`).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxcloseclaim

import (
	"context"
	"strings"
	"time"
)

// Nilai `PYSTATUSWORK` yang menandai klaim SUDAH TUTUP.
//
// Keduanya inilah yang disaring kueri — `IN`, bukan `NOT IN`. Satu huruf yang salah di
// sini membalik seluruh isi layar tanpa menghasilkan galat apa pun: yang tampil menjadi
// klaim yang masih berjalan, dan tidak ada apa pun di layar yang menandakannya.
const (
	StatusKerjaSelesai = "Resolved-Completed"
	StatusKerjaDitolak = "Resolved-Rejected"
)

// DisplayStatus adalah label yang DILIHAT pengguna pada kolom status.
//
// Pemetaannya diambil dari `RDB List/BrowseClaimALL-SQL.xml`, yang menurunkan hal yang
// sama di dalam SQL — bukan dari tebakan:
//
//	WHEN a.pystatuswork = 'Resolved-Completed' THEN 'Close'
//	WHEN a.pystatuswork = 'Resolved-Rejected'  THEN 'Reject'
//
// Nilainya berbahasa Inggris karena itulah yang tertulis di layar lama (`D-13`).
type DisplayStatus string

const (
	DisplayClose  DisplayStatus = "Close"
	DisplayReject DisplayStatus = "Reject"
)

// ClosedClaim adalah satu baris pada layar — sebuah klaim yang sudah tutup.
//
// Field di sini adalah apa yang DITAMPILKAN, bukan salinan utuh klaim. Modul ini layar
// pemantauan beserta dua aksi permintaan; isi klaim yang sesungguhnya milik modul
// `registrasi` dan modul nilai.
type ClosedClaim struct {
	// ClaimID adalah kunci teknis `PZINSKEY` — dialias `CaseID` pada kueri lama.
	//
	// Isinya berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`: nama kelas internal Pega tertanam di
	// dalam kunci data bisnis — utang teknis §4.1 yang `D-22` dan `D-71` hapus untuk klaim
	// baru.
	//
	// Berbeda dari modul Inbox Outstanding, di sini ia TIDAK hanya dipakai menautkan baris
	// ke layar detail: kedua tombol aksi mengirimnya sebagai `casePNC`, persis seperti
	// `Section/InboxManagerReopen1_Sec-Section.xml` mengirim `.CaseID` ke `SaveReOpenAct`.
	ClaimID string

	// ClaimNumber — kolom "No Klaim" <- PYID, dialias `CaseIDView`.
	ClaimNumber string

	PolicyNumber   string // "No Polis"          <- POLICYNO
	InsuredName    string // "Nama Tertanggung"  <- QQNAME       (alias lama: CustomerName)
	BusinessName   string // "Nama Bisnis"       <- BUSINESSNAME
	BusinessSource string // "Sumber Bisnis"     <- SOBNAME      (alias lama: SourceOfBusinessName)
	BranchName     string // "Nama Cabang"       <- BRANCHNAME
	TechnicalPIC   string // "PIC Teknik"        <- USERTEKNIS_1 (alias lama: ReinsurerName)
	AdminPNC       string // "Admin PNC"         <- PXCREATEOPNAME (alias lama: MOName)

	// GroupPanel adalah lini bisnis: 002 PA · 003/004/006 Non-MBU · 005 Travel.
	//
	// Ia yang dipakai penyaring lini bisnis — lihat BusinessLine.
	GroupPanel string

	// BusinessGroupID adalah kelompok bisnis dari `POOLDATA.BUSINESS.BUSINESSGROUPID`.
	//
	// Dipakai penyaring BONDING dan NONMBU, dan arah keduanya BERLAWANAN — lihat
	// BusinessLine.
	BusinessGroupID string

	// RegisteredAt — kolom "Tanggal Pendaftaran" <- PXCREATEDATETIME.
	//
	// Kueri lama memakai `pxCreateDateTime`, bukan `REGISTERDATE_1`. Keduanya ada di tabel
	// dan mudah tertukar; yang dipakai di sini adalah yang benar-benar dibaca layar lama.
	RegisteredAt time.Time

	// LossDate — tanggal kejadian <- DATEOFLOSS_1, dialias `CoverNo`.
	//
	// Kueri lama sudah memformatnya menjadi teks `dd-mm-yyyy` di dalam SQL
	// (`to_char(TRUNC(dateofloss_1),'dd-mm-yyyy')`). Itu TIDAK dibawa: `D-20` menetapkan
	// pemformatan tanggal keluar dari SQL, karena tanggal yang dikembalikan sebagai teks
	// membuat pengurutan menjadi pengurutan teks — `01/12/2024` terbaca lebih kecil dari
	// `02/01/2020`.
	//
	// TIDAK ditampilkan sebagai kolom: section rujukan tidak punya kolomnya. Dibaca karena
	// kueri lama menyediakannya dan berguna saat menelusuri satu klaim.
	LossDate *time.Time

	// ClosedAt adalah tanggal klaim ditutup <- CLOSECLAIMDATE_1.
	//
	// # Kolom ini TIDAK ada di kueri lama, dan penambahannya disengaja
	//
	// Ia dibutuhkan kolom "Lama Waktu Klaim" — lihat DurationDays. Kolomnya memang ADA di
	// `PC_ASM_FW_GCNMFW_WORK` dengan 514 nilai berbeda (`migrations/0005`), hanya tidak
	// pernah diambil kueri lama.
	ClosedAt *time.Time

	// ResolvedAt <- PYRESOLVEDTIMESTAMP, cadangan bagi ClosedAt.
	//
	// Ia terisi pada 3.142 klaim, jauh lebih banyak daripada `CLOSECLAIMDATE_1` yang 514.
	// Bedanya: yang satu tanggal penutupan menurut BISNIS, yang satu waktu objek kerjanya
	// diselesaikan Pega. Yang pertama lebih tepat; yang kedua lebih sering ada.
	ResolvedAt *time.Time

	// ProcessStatus adalah `PYSTATUSWORK` — status ALUR KERJA, salah satu dari empat
	// konsep status yang `ADR-0018` larang digabung. Dari sinilah DisplayStatus diturunkan.
	ProcessStatus string

	// ClaimStatusCode adalah KODE status klaim <- STATUSCLAIM_1, bukan labelnya.
	//
	// Ia konsep status yang BERBEDA dari ProcessStatus. Di sini ia lebih dari sekadar
	// tampilan: penyaring Status Bayar membandingkannya dengan `'1163'` (Paid) —
	// `Activity/GCNMGetManagerReopenCase_Act-Act.xml`, properti `TempFilter.DistrictID`.
	ClaimStatusCode string

	// ClaimStatusLabel adalah label kode di atas, dicari ke `POOLDATA.V_STS_CLAIM`.
	//
	// Sistem lama TIDAK menampilkannya di layar ini — section rujukan tidak punya
	// kolomnya. Ia dibaca supaya penyaring Status Bayar dapat MENYATAKAN apa yang
	// disaringnya, alih-alih memperlihatkan kode telanjang `1163` kepada pengguna.
	ClaimStatusLabel string

	// TransferredToCashier menandai klaim pernah ditransfer ke kasir.
	//
	// Diturunkan dari keberadaan baris ber-`TRANSFER_CASHIER_DATE` pada
	// `POOLDATA.T_CLAIM_ADJUSTMENT`, persis seperti penyaring Status Transfer sistem lama.
	// Dihitung di SQL, bukan lewat kueri kedua per baris — kueri di dalam perulangan adalah
	// hambatan peringkat ketiga pada `15-NFR` §3.2.
	TransferredToCashier bool
}

// DisplayStatus menurunkan label status klaim.
func (c ClosedClaim) DisplayStatus() DisplayStatus {
	switch strings.TrimSpace(c.ProcessStatus) {
	case StatusKerjaSelesai:
		return DisplayClose
	case StatusKerjaDitolak:
		return DisplayReject
	default:
		// Nilai yang tidak dikenali ditampilkan APA ADANYA, tidak dipaksa menjadi salah
		// satu dari keduanya.
		//
		// Kueri sudah menyaring `IN` atas dua nilai itu, sehingga cabang ini mestinya tidak
		// pernah tercapai. Bila ia tercapai, sesuatu yang tidak diduga sedang terjadi — dan
		// status asing yang tampil mentah akan segera ditanyakan pengguna, sedangkan status
		// asing yang menyamar sebagai "Close" tidak pernah ditanyakan.
		return DisplayStatus(strings.TrimSpace(c.ProcessStatus))
	}
}

// DurationDays adalah kolom **"Lama Waktu Klaim"** — berapa hari klaim itu berjalan.
//
// # Kenapa dihitung, padahal Pega menampilkan tanggal
//
// Di sistem lama kolom ini terikat `.pxCreateDateTime` dengan format `pxDateTime`
// (`Section/InboxManagerReopen1_Sec-Section.xml`, sel 55) — yaitu **TANGGAL YANG SAMA
// PERSIS** dengan kolom "Tanggal Pendaftaran" di sebelahnya. Sebuah kolom berjudul durasi
// yang isinya tanggal.
//
// Work Owner memutuskan 2026-09-23 kolom itu diisi umur dalam hari. Ia **selisih
// terencana** terhadap Pega, dinyatakan di layar, bukan disembunyikan sebagai detail
// tampilan (`D-54`).
//
// # Dihitung sampai KAPAN
//
// Sampai klaim ditutup — bukan sampai hari ini. Klaim yang tutup tiga tahun lalu bukan
// klaim berumur seribu hari; ia klaim yang dulu memakan sekian hari. Urutan sumbernya:
//
//  1. ClosedAt     CLOSECLAIMDATE_1      tanggal penutupan menurut bisnis
//  2. ResolvedAt   PYRESOLVEDTIMESTAMP   waktu objek kerjanya diselesaikan Pega
//  3. now          bila keduanya kosong
//
// Cadangan ketiga bukan kesalahan: baris warisan yang kedua kolomnya kosong tetap harus
// menampilkan sesuatu yang bermakna, dan "sampai sekarang" adalah bacaan terbaik yang
// tersisa.
//
// # Kenapa dihitung terhadap TANGGAL, bukan selisih jam dibagi 24
//
// Klaim yang didaftarkan pukul 23.00 dan ditutup pukul 01.00 keesokan harinya sudah
// berumur SATU HARI bagi pengguna, meski selisihnya dua jam. Membagi selisih jam akan
// mengembalikan nol.
//
// # Kenapa zona waktu ikut masuk
//
// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4) sedangkan "hari" yang dimaksud
// pengguna adalah hari WIB. Tanpa konversi, klaim yang dibuat antara pukul 00.00 dan 07.00
// WIB dihitung satu hari lebih tua. Konversinya diserahkan pemanggil lewat parameter,
// bukan dibaca dari jam sistem — supaya dapat diuji tanpa bergantung mesin.
func (c ClosedClaim) DurationDays(now time.Time, location *time.Location) int {
	if c.RegisteredAt.IsZero() {
		return 0
	}
	if location == nil {
		location = time.UTC
	}

	end := now
	switch {
	case c.ClosedAt != nil && !c.ClosedAt.IsZero():
		end = *c.ClosedAt
	case c.ResolvedAt != nil && !c.ResolvedAt.IsZero():
		end = *c.ResolvedAt
	}

	start := c.RegisteredAt.In(location)
	finish := end.In(location)

	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDay := time.Date(finish.Year(), finish.Month(), finish.Day(), 0, 0, 0, 0, location)

	days := int(endDay.Sub(startDay).Hours() / 24)
	if days < 0 {
		// Tanggal tutup yang mendahului tanggal daftar adalah data yang cacat, bukan durasi
		// negatif. Ia ditampilkan nol; yang memperbaikinya adalah datanya, bukan layar ini.
		return 0
	}
	return days
}

// BusinessLine adalah penyaring lini bisnis pada bilah atas layar.
//
// # Dari mana nilainya, dan kenapa hanya lima
//
// Dari `Activity/GCNMGetManagerReopenCase_Act-Act.xml`, yang bercabang atas
// `TempView2.Remark` — properti yang diisi dropdown "Business" di layar. Kelima cabangnya
// menyusun potongan SQL yang berbeda, dan tidak ada cabang keenam.
//
// Predikat aslinya, APA ADANYA:
//
//	NONMBU   GROUPPANEL_1 in ('003','004','006')
//	         AND c.businessgroupid NOT IN ('10008','10010','10015','10023')
//	BONDING  AND c.businessgroupid IN  ('10008','10010','10015','10023')
//	PA       and GroupPanel_1 in ('002')
//	TRAVEL   and GroupPanel_1 in ('005')
//	ALL      "" — tanpa saringan
//
// # DUA PERBEDAAN YANG TIDAK BOLEH DISERAGAMKAN
//
// Modul lain punya daftar yang MIRIP tetapi tidak sama, dan menyamakannya akan menampilkan
// lini yang salah tanpa satu pun galat:
//
//	                  BONDING                  NONMBU
//	modul ini         businessgroupid IN       ('003','004','006')
//	inboxadmin        businessgroupid IN       ('003','004','006','009')
//	inboxoutstanding  businessgroupid NOT IN   ('003','004','006')
//
// `inboxoutstanding` memakai arah yang BERLAWANAN pada BONDING
// (`Activity/InboxOutstanding_Act-Act.xml`), dan `inboxadmin` memuat `'009'` yang layar ini
// tidak punya (`Activity/SetTempClaimRegistandNotRegist-Act.xml`). Ketiganya disalin dari
// activity-nya masing-masing, dan `P-5` menetapkan perilaku dipertahankan lebih dulu.
//
// Karena itu modul ini TIDAK memakai ulang `inboxoutstanding.ScopeFor` maupun
// `inboxadmin.BusinessLine`. Kesamaan namanya kebetulan; isinya tidak sama.
type BusinessLine string

// Kelima nilai penyaring lini bisnis.
//
// Nilainya huruf besar persis seperti yang dibandingkan activity lama, dan itu bukan gaya
// penulisan melainkan kontrak: ia dikirim layar sebagai parameter query.
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

// TransferStatus adalah penyaring "Status Transfer Kasir".
//
// Asalnya `Activity/GCNMGetManagerReopenCase_Act-Act.xml`, properti `TempFilter.District`:
//
//	"SUDAH TRANSFER"  AND EXISTS (SELECT CLAIMID FROM POOLDATA.T_CLAIM_ADJUSTMENT B
//	                              WHERE B.TRANSFER_CASHIER_DATE IS NOT NULL
//	                                AND B.CLAIMID = A.PZINSKEY)
//	"BELUM TRANSFER"  AND NOT EXISTS (… kueri yang sama …)
//	""                tanpa saringan
type TransferStatus string

const (
	TransferAny  TransferStatus = ""
	TransferDone TransferStatus = "SUDAH TRANSFER"
	TransferNone TransferStatus = "BELUM TRANSFER"
)

// ParseTransferStatus membaca pilihan Status Transfer; kosong berarti tanpa saringan.
func ParseTransferStatus(raw string) (TransferStatus, bool) {
	value := TransferStatus(strings.ToUpper(strings.TrimSpace(raw)))
	switch value {
	case TransferAny, TransferDone, TransferNone:
		return value, true
	default:
		return "", false
	}
}

// PaymentStatus adalah penyaring "Status Bayar".
//
// Asalnya `TempFilter.DistrictID` pada activity yang sama:
//
//	"LUNAS"            AND A.STATUSCLAIM_1 =  '1163'
//	selain itu, terisi AND A.STATUSCLAIM_1 != '1163'
//	""                 tanpa saringan
//
// `1163` adalah kode status **Paid** pada master 33 kode `1134`–`1166` (`R-06` tertutup).
type PaymentStatus string

const (
	PaymentAny  PaymentStatus = ""
	PaymentPaid PaymentStatus = "LUNAS"

	// PaymentUnpaid mewakili cabang "terisi tetapi bukan LUNAS" pada activity lama.
	//
	// Di sana cabang itu dipicu nilai apa pun yang bukan "LUNAS" dan bukan kosong; di sini
	// ia satu nilai tegas, supaya layar tidak dapat mengirim teks sembarang yang diam-diam
	// berarti "belum lunas".
	PaymentUnpaid PaymentStatus = "BELUM LUNAS"
)

// KodeStatusLunas adalah kode status klaim "Paid".
//
// Ia konstanta di sini, bukan angka yang tersebar di dalam SQL — `D-15` menetapkan tidak
// ada nilai bisnis yang boleh di-hardcode berulang kali. Bahwa ia masih konstanta kode dan
// belum master data dicatat sebagai pertanyaan terbuka: master status sudah ada
// (`POOLDATA.V_STS_CLAIM`, 33 kode), tetapi PENANDA "kode mana yang berarti lunas" tidak
// ada di dalamnya.
const KodeStatusLunas = "1163"

// ParsePaymentStatus membaca pilihan Status Bayar; kosong berarti tanpa saringan.
func ParsePaymentStatus(raw string) (PaymentStatus, bool) {
	value := PaymentStatus(strings.ToUpper(strings.TrimSpace(raw)))
	switch value {
	case PaymentAny, PaymentPaid, PaymentUnpaid:
		return value, true
	default:
		return "", false
	}
}

// Filter adalah penyaring dan paginasi yang diminta layar.
//
// Keenam penyaringnya disalin dari `Activity/GCNMGetManagerReopenCase_Act-Act.xml`, bukan
// dikarang — panel penyaringnya sendiri (`FilterDashboardClaimclose`) tidak ada di export,
// tetapi activity yang membacanya menyebut keenamnya satu per satu.
type Filter struct {
	// Search adalah kotak cari gabungan — `param.filter2` di sistem lama.
	//
	// Ia mencari pada No Polis DAN No Klaim sekaligus:
	//
	//	(policyno like '%…%' or pyid like '%…%')
	//
	// Pencarian dikerjakan SERVER, bukan peramban: tabel klaim berisi puluhan juta baris
	// (`D-10`), dan menyaring satu halaman dari sepuluh akan memberi tahu pengguna bahwa
	// sesuatu tidak ada padahal ia ada di halaman lain.
	Search string

	// PolicyNumber, ClaimNumber, dan TechnicalPIC adalah tiga penyaring TERPISAH —
	// `filterNOPOLIS`, `filterNOKLAIM`, `filterPIC`.
	//
	// Ketiganya hidup berdampingan dengan Search dan tidak menggantikannya: di sistem lama
	// keempatnya disisipkan ke kueri yang sama lewat penanda `{ASIS:…}` yang berbeda.
	PolicyNumber string
	ClaimNumber  string
	TechnicalPIC string

	Business BusinessLine
	Transfer TransferStatus
	Payment  PaymentStatus

	Limit  int
	Offset int
}

// Batas paginasi.
//
// Sistem lama memakai `.PageSize := 25` (`Activity/GCNMGetManagerReopenCase_Act-Act.xml`),
// dan angka itulah yang dipakai sebagai bawaan. Batas maksimumnya mengikuti
// `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan dipenuhi.
const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
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

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// # Total di sini BERBEDA dari total Pega, dan itu disengaja
//
// `RDB List/GCNMCountCloseClaim-SQL.xml` kehilangan satu penanda yang ada di kueri
// daftarnya — `{ASIS:TempFilter.DistrictID}`, yaitu penyaring **Status Bayar**. Akibatnya
// di sistem lama: begitu penyaring itu dipakai, jumlah total yang ditampilkan TIDAK cocok
// dengan baris yang benar-benar dapat ditelusuri, dan tidak ada galat yang muncul.
//
// Work Owner memutuskan 2026-09-23 keduanya disamakan. Syarat WHERE kueri hitung dibuat
// sama persis dengan kueri daftar dan dijaga uji — selisih terencana yang dicatat, bukan
// perbaikan diam-diam (`D-54`).
type Page struct {
	Claims []ClosedClaim
	Total  int
}

// Repo adalah seam ke penyimpanan klaim.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya.
// Diisi `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// Ia hanya MEMBACA, dan ketiadaan method tulis di sini disengaja: klaim masih ditulis Pega
// selama masa paralel, dan `P-1` menetapkan satu tabel hanya ditulis satu sistem. Aksi
// ReOpen dan Copy Klaim tidak menulis klaim — lihat RequestRepo di request.go.
type Repo interface {
	List(ctx context.Context, f Filter) (Page, error)

	// ClosedClaimNumber mengembalikan nomor klaim bila klaim itu ADA dan SUDAH TUTUP.
	//
	// Mengembalikan ErrClaimNotFound bila tidak ada — termasuk bila klaimnya ada tetapi
	// masih berjalan.
	//
	// # Kenapa pemeriksaan ini ada, dan kenapa ia MEMBACA
	//
	// Kedua aksi layar ini mengirim kunci klaim dari peramban. Tanpa memeriksanya lebih
	// dulu, permintaan ReOpen dapat diajukan atas klaim yang MASIH BERJALAN — yang justru
	// tidak boleh dibuka kembali karena belum pernah tutup.
	//
	// Syarat tutupnya sama persis dengan kueri daftar, dan itulah yang membuat "ada di
	// layar ini" dan "boleh diajukan" tidak dapat berselisih.
	ClosedClaimNumber(ctx context.Context, claimID string) (string, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas, bukan satu database bersama dengan
// penanda entitas. Klaim milik Asuransi Sinar Mas dan klaim milik Simas Insurtech karena
// itu tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan klaim satu
// badan hukum di layar badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)
