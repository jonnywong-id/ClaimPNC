// Package inboxoutstanding adalah inti modul Inbox Outstanding.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `Navigation/pyCaseWorkerNavigation-Navigation.xml` berbunyi **"Inbox Outstanding"**, dan
// `Section/InboxRegister_Section-Section.xml:2150` memuat judul yang sama di dalam
// layarnya. `D-81` menetapkan nama modul mengikuti nama yang dipakai Work Owner.
//
// # Yang dimigrasikan
//
//	Harness/InboxRegister_Harness-Harness.xml    layar rujukan
//	Section/InboxRegister_Section-Section.xml    kolom beserta judulnya
//	RDB List/BrowseInboxOutstanding1-SQL.xml     kueri inti
//	Activity/InboxOutstanding_Act-Act.xml        batas data + derivasi tampilan
//
// Yang mengikat section rujukan dengan kueri itu: properti selnya PERSIS alias kuerinya —
// `.District` untuk "Policy no", `.CountryID` untuk "Insured name", `.City` untuk
// "Branch name", `.ReporterName` untuk "Admin name".
//
// Butir menu "Inbox Outstanding" sendiri menunjuk `InboxOutstanding_Harness`, yang tidak
// ada di export (`K-33`). Section bernama serupa — `InboxOutstandingClaim_Section` —
// dipakai dashboard dengan sumber data berbeda, dan BUKAN rujukan modul ini.
//
// # Apa itu "Outstanding"
//
// Definisinya mengikat dan terbaca dari satu baris —
// `RDB List/BrowseInboxOutstanding1-SQL.xml:125`:
//
//	AND pystatuswork NOT IN ('Resolved-Completed', 'Resolved-Rejected')
//
// Jadi Outstanding adalah SELURUH klaim yang masih berjalan, beserta siapa yang sedang
// memegang tugasnya. Ia BUKAN Inbox menurut `D-79`: isinya bukan "pekerjaan saya"
// melainkan "semua klaim yang belum tuntas", sehingga barisnya tidak hilang setelah
// seseorang mengerjakannya dan tidak punya tombol "Ambil".
//
// Bedakan dari inbox modul `registrasi`, yang memang berisi pekerjaan pemanggil.
//
// # Sumber datanya: POOLDATA.T_CLAIMLIST_ADMIN
//
// Work Owner menetapkan tabel itulah yang menggantikan
// `datapega.pc_asm_fw_gcnmfw_work` (2026-09-21), dan modul ini TIDAK membuat tabel
// sendiri.
//
// Ia tabel DATAR: satu baris per klaim, memuat seluruh yang dibutuhkan layar tanpa satu
// pun join — termasuk `AGING` yang sudah berupa angka, `SOBNAME`, `GROUPPANEL_1`, dan
// `BUSINESSGROUPID` yang di sistem lama harus ditarik lewat empat tabel.
//
// Modul ini hanya MEMBACA. Yang mengisi tabel itu adalah sistem lama.
//
// > Sempat dibangun di atas `CPNC_KLAIM` — tabel rancangan modul `registrasi` yang belum
// > pernah dibuat dan modulnya belum dipasang. Itu salah tafsir atas arahan "tabel baru",
// > dikoreksi Work Owner. Riwayatnya di `docs/catatan-pengembangan.md`.
//
// # Alias Pega tidak dibawa masuk
//
// Kueri lama mengalias kolom secara MENYESATKAN — `a.policyno AS "District"`,
// `a.qqname AS "CountryID"`, `a.branchname AS "City"`, dan
// `to_char(a.pxCreateDateTime,…) AS "StatusWork"` yang sebenarnya tanggal pendaftaran.
// Tiga belas alias semacam itu ada di satu kueri. Tidak satu pun dibawa; nama di sini
// mengikuti padanan Inggris dari `CONTEXT.md` (`D-19`, `D-80`).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxoutstanding

import (
	"context"
	"strings"
	"time"
)

// DisplayStatus adalah status yang DILIHAT pengguna pada kolom "Claim status".
//
// Ia dihitung, tidak disimpan. Sistem lama menurunkannya di
// `Activity/InboxOutstanding_Act-Act.xml` dengan memeriksa properti `.StatusClaim` — yang,
// karena alias menyesatkan pada kueri, sebenarnya berisi `pyStatusWork`:
//
//	.StatusClaim == "New"                 -> "On Progress"
//	.StatusClaim == "Resolved-Completed"  -> "Close"
//	.StatusClaim == "Resolved-Rejected"   -> "Reject"
//
// Nilainya berbahasa Inggris karena itulah yang tertulis di layar lama, dan `D-13`
// menetapkan teks yang dilihat pengguna mengikuti layar Pega apa adanya.
type DisplayStatus string

const (
	DisplayOnProgress DisplayStatus = "On Progress"
	DisplayClose      DisplayStatus = "Close"
	DisplayReject     DisplayStatus = "Reject"
)

// Nilai `PYSTATUSWORK` sebagaimana benar-benar tersimpan di `T_CLAIMLIST_ADMIN`.
//
// Dua di antaranya MESTINYA tidak pernah terbaca: kueri sudah menyaring
// `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`. Keduanya tetap
// diturunkan lengkap supaya derivasinya setara dengan sistem lama, dan supaya baris yang
// lolos lewat jalur lain tetap berlabel benar.
//
// # Kekeliruan yang pernah ada di sini
//
// Ketiganya sempat ditulis sebagai `"BERJALAN"`, `"SELESAI"`, `"DITOLAK"` — nilai kolom
// `CPNC_KLAIM.STATUS_PROSES`, tabel yang modul ini sempat salah pakai (§17.15). Data
// contoh memakai nilai yang sama, sehingga seluruh uji lulus: kode diperiksa terhadap data
// yang dikarang dari kode itu sendiri.
//
// Yang membongkarnya adalah SATU BARIS data produksi dari Work Owner, ber-`PYSTATUSWORK`
// bernilai `'New'`. Terhadap kode lama, 'New' jatuh ke cabang default dan kebetulan
// menghasilkan label yang benar — tetapi `'Resolved-Completed'` juga jatuh ke default dan
// menghasilkan **"On Progress" untuk klaim yang sudah ditutup**.
//
// Pemetaannya diambil dari `RDB List/BrowseClaimALL-SQL.xml`, yang menurunkan hal yang
// sama di dalam SQL:
//
//	WHEN a.pystatuswork = 'New'                THEN 'On Progress'
//	WHEN a.pystatuswork = 'Resolved-Rejected'  THEN 'Reject'
//	WHEN a.pystatuswork = 'Resolved-Completed' THEN 'Close'
const (
	statusKerjaBerjalan = "New"
	statusKerjaSelesai  = "Resolved-Completed"
	statusKerjaDitolak  = "Resolved-Rejected"
)

// OutstandingClaim adalah satu baris pada layar — sebuah klaim yang masih berjalan.
//
// Field di sini adalah apa yang DITAMPILKAN, bukan salinan utuh klaim. Modul ini layar
// pemantauan; isi klaim yang sesungguhnya milik modul `registrasi` dan modul nilai.
type OutstandingClaim struct {
	// ClaimID adalah kunci teknis `PZINSKEY`. Dipakai untuk menautkan baris ke layar
	// detail, TIDAK ditampilkan — dan memang tidak boleh.
	//
	// Isinya berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`: nama kelas internal Pega tertanam
	// di dalam kunci data bisnis — utang teknis §4.1 yang `D-22` dan `D-71` hapus untuk
	// klaim baru. Ia dibaca apa adanya karena baris warisan memang memilikinya, tetapi
	// tidak pernah sampai ke layar maupun CSV.
	ClaimID string

	// ClaimNumber — kolom "No Klaim". Berformat PNCN.YY.xxxx (`D-71`).
	//
	// KOSONG untuk klaim yang belum lolos Input Register: nomor baru terbit di ujung tahap
	// itu (`ADR-0009`). Layar menampilkannya sebagai "belum bernomor", bukan sebagai sel
	// kosong yang tampak rusak.
	ClaimNumber string

	PolicyNumber   string // "Policy no"       <- POLICYNO      (alias lama: District)
	InsuredName    string // "Insured name"    <- QQNAME        (alias lama: CountryID)
	BusinessName   string // "Business Name"   <- BUSINESSNAME  (alias lama: Country)
	BusinessSource string // "Business source" <- SOBNAME       (alias lama: CityID)
	BranchName     string // "Branch name"     <- BRANCHNAME    (alias lama: City)

	// GroupPanel adalah lini bisnis: 002 PA · 003/009 Aneka · 004 Marine · 005 Travel ·
	// 006 Fire. Ia yang dipakai batas data — lihat LineScope.
	GroupPanel string

	// BusinessGroupID dipakai batas data untuk MENGECUALIKAN kelompok bisnis tertentu.
	//
	// Di sistem lama ia datang lewat join ke `pooldata.businessgroup`; di tabel datar ini
	// ia kolom tersendiri, sehingga aturan pengecualian dapat diterapkan tanpa join —
	// termasuk aturan BONDING yang seluruhnya bersandar padanya.
	BusinessGroupID string

	// RegisteredAt — kolom "Register Date" <- REGISTERDATE_1.
	//
	// Tabel ini punya DUA tanggal yang bisa disalahartikan sebagai tanggal registrasi:
	// `REGISTERDATE_1` dan `PXCREATEDATETIME`. Yang dipakai adalah yang pertama, karena
	// namanya memang menyebut registrasi; yang kedua adalah waktu baris dibuat di Pega.
	// Bila REGISTERDATE_1 kosong, PXCREATEDATETIME dipakai sebagai cadangan.
	RegisteredAt time.Time

	// LossDate — kolom "Date of loss" <- DATEOFLOSS_1.
	//
	// Aliasnya di kueri lama "RW", salah satu alias yang paling tidak berhubungan dengan
	// isinya. Boleh kosong.
	LossDate *time.Time

	// ReportDate <- REPORTDATE_1. TIDAK ditampilkan: section rujukan tidak punya kolomnya.
	ReportDate *time.Time

	// AgingDays — kolom "Aging" <- kolom AGING, bertipe NUMBER.
	//
	// Ia SUDAH DIHITUNG sistem lama dan dibaca apa adanya, bukan dihitung ulang di sini.
	// Menghitungnya ulang berarti menebak dari tanggal mana ia dihitung — dan tabel ini
	// punya `DATEFORAGING_1` justru sebagai acuannya, yang artinya belum dipastikan.
	//
	// Dibedakan dari AgeInDays, yang menghitung "Total Aging" sejak registrasi.
	AgingDays *int

	// ProcessStatus adalah `PYSTATUSWORK` — status ALUR KERJA, salah satu dari empat
	// konsep status yang `ADR-0018` larang digabung. Dari sinilah DisplayStatus
	// diturunkan; nilainya `New`, `Resolved-Completed`, atau `Resolved-Rejected`.
	//
	// Kolom layar "Claim status"; di section lama ia properti `.StatusClaim`, yang karena
	// alias kueri (`pyStatusWork AS "StatusClaim"`) sebenarnya berisi status ALUR KERJA.
	ProcessStatus string

	// ClaimStatus — kolom layar "Status ASM" <- STATUSLOCK_1.
	//
	// Konsep status yang BERBEDA dari ProcessStatus, dan `ADR-0018` melarang keduanya
	// digabung: yang satu posisi dalam alur kerja, yang satu keadaan bisnis klaim.
	//
	// # Kenapa kolom ini, dan apa yang masih belum dikonfirmasi
	//
	// Sistem lama tidak menyimpan labelnya: `BrowseInboxOutstanding1-SQL.xml` menempuh
	// subkueri `SELECT b.lsc_note FROM v_sts_claim b WHERE a.STATUSCLAIM_1 = b.LSC_ID`.
	// Jadi `STATUSCLAIM_1` menyimpan KODE, dan labelnya dicari saat itu juga.
	//
	// `T_CLAIMLIST_ADMIN` **tidak punya `STATUSCLAIM_1`** — kolom itu tidak ada di DDL.
	// Yang ada `STATUSLOCK_1`, dan lebarnya `VARCHAR2(100)`. Kode status hanya butuh
	// empat karakter; seratus karakter adalah lebar untuk LABEL — `1143` berlabel
	// "Close Claim for this object", 26 karakter.
	//
	// Bacaan yang paling sesuai bukti: tabel datar ini sudah MENYELESAIKAN pencarian itu
	// di muka, menyimpan labelnya, dan membuang kodenya. Itulah yang memang dilakukan
	// tabel pelaporan yang didatarkan.
	//
	// Yang belum dikonfirmasi: baris contoh dari Work Owner **tidak menyertakan kolom
	// ini**, sehingga isinya belum pernah terlihat. Bila ternyata ia menyimpan kode dan
	// bukan label, yang dibutuhkan adalah master status (`33 kode 1134`–`1166`) — bukan
	// kolom lain. Tercatat di `docs/keputusan-implementasi.md` §18.20.
	ClaimStatus string

	// ProgressStatus — kolom "Progress Klaim" <- STATUSPROGRESS1.
	//
	// Di sistem lama ia hasil `GET_POSISI_PROGRESS_PNC(claimno,'sts_prg1')`; di tabel
	// datar ini ia sudah tersimpan sebagai kolom. TIDAK ditampilkan pada layar ini —
	// section rujukan tidak punya kolomnya — tetapi dibaca karena tersedia tanpa biaya.
	ProgressStatus string

	// TechnicalPIC — kolom "ASM PIC" <- USERTEKNIS_1.
	TechnicalPIC string

	// RecordedBy — kolom "Admin name" <- PXCREATEOPERATOR, petugas yang membuat klaim.
	//
	// Bukan pemegang tugas saat ini; keduanya orang yang berbeda dan tabel ini memuat
	// keduanya (`PXCREATEOPERATOR` dan `PXASSIGNEDOPERATORID`).
	RecordedBy string

	// CurrentHolder adalah pemegang tugas <- PXASSIGNEDOPERATORID.
	//
	// Dialias "NamaSurveyor" pada kueri lama padahal bukan surveyor. TIDAK ditampilkan
	// pada layar ini; section rujukan tidak punya kolomnya.
	CurrentHolder string

	// CurrentStage adalah tahap tempat klaim berada <- PXTASKLABEL. Tidak ditampilkan.
	CurrentStage string
}

// DisplayStatus menurunkan label kolom "Claim status".
//
// Lihat komentar pada tipe DisplayStatus untuk asal pemetaannya.
func (c OutstandingClaim) DisplayStatus() DisplayStatus {
	switch strings.TrimSpace(c.ProcessStatus) {
	case statusKerjaBerjalan:
		return DisplayOnProgress
	case statusKerjaSelesai:
		return DisplayClose
	case statusKerjaDitolak:
		return DisplayReject
	default:
		// Nilai yang tidak dikenali ditampilkan APA ADANYA, bukan dipaksa menjadi
		// "On Progress".
		//
		// Inilah cabang yang dulu menyembunyikan cacatnya: setiap nilai jatuh ke sini
		// dan keluar sebagai "On Progress", termasuk klaim yang sudah ditutup. Sebuah
		// status asing yang tampil mentah di layar akan segera ditanyakan pengguna;
		// status asing yang menyamar sebagai "On Progress" tidak pernah ditanyakan.
		return DisplayStatus(strings.TrimSpace(c.ProcessStatus))
	}
}

// AgeInDays adalah kolom **"Total Aging"** — umur klaim dalam hari kalender sejak
// didaftarkan.
//
// # Kolom "Aging" yang bersebelahan TIDAK dapat dihitung di sini
//
// `Section/InboxRegister_Section-Section.xml` menampilkan DUA ukuran umur bersebelahan:
// "Total Aging" dan "Aging". Yang kedua terikat properti `.DateForAging` — sebuah tanggal
// acuan tersendiri yang **tidak disediakan kueri** `BrowseInboxOutstanding1`.
//
// Karena itu hanya Total Aging yang dibangun. Menghitung "Aging" dari tanggal pendaftaran
// akan membuat kedua kolom selalu bernilai sama, dan itu lebih menyesatkan daripada tidak
// menampilkannya.
//
// # Kenapa dihitung terhadap TANGGAL, bukan selisih jam dibagi 24
//
// Klaim yang didaftarkan pukul 23.00 dan dilihat pukul 01.00 keesokan harinya sudah
// berumur SATU HARI bagi pengguna, meski selisihnya dua jam. Membagi selisih jam akan
// mengembalikan nol, dan petugas akan membaca klaim kemarin sebagai klaim hari ini.
//
// # Kenapa zona waktu ikut masuk
//
// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4) sedangkan "hari" yang dimaksud
// pengguna adalah hari WIB. Tanpa konversi, seluruh klaim yang dibuat antara pukul 00.00
// dan 07.00 WIB akan dihitung satu hari lebih tua. Konversinya diserahkan pemanggil lewat
// parameter, bukan dibaca dari jam sistem — supaya dapat diuji tanpa bergantung mesin.
func (c OutstandingClaim) AgeInDays(now time.Time, location *time.Location) int {
	if c.RegisteredAt.IsZero() {
		return 0
	}
	if location == nil {
		location = time.UTC
	}
	start := c.RegisteredAt.In(location)
	end := now.In(location)

	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location)

	days := int(endDay.Sub(startDay).Hours() / 24)
	if days < 0 {
		// Klaim bertanggal masa depan tidak berumur negatif; ia berumur nol hari.
		return 0
	}
	return days
}

// LineScope adalah batas data: lini bisnis mana yang boleh dilihat seorang pengguna.
//
// # Asalnya di sistem lama
//
// `Activity/InboxOutstanding_Act-Act.xml` memilih potongan WHERE berdasarkan
// `OperatorID.pyPosition`, lalu menyisipkannya ke kueri lewat `{ASIS:TempView.pyNote}`:
//
//	pyPosition  potongan yang dipasang                                   baris
//	NONMBU      AND GROUPPANEL_1 in ('003','004','006') AND c.business…  :905
//	BONDING     AND c.businessgroupid NOT IN (…)                         :1081
//	PA          AND GROUPPANEL_1='002'                                   :1235
//	TRAVEL      AND GROUPPANEL_1='005'                                   :1374
//
// Penyisipan `{ASIS:…}` itu adalah PERANGKAIAN SQL dan tidak dibawa. Yang dipakai di sini:
// cabang ditentukan di Go, nilainya lewat parameter binding.
//
// # Penggantinya di sistem baru
//
// `OperatorID.pyPosition` diganti kolom BARU `POOLDATA.M_LOGIN_PNC.LINEBUSINESS`
// (keputusan Work Owner 2026-09-19). Tabel itu dipakai untuk karyawan juga, bukan hanya
// non-karyawan seperti sekarang.
type LineScope struct {
	// Unrestricted menandai pengguna melihat SELURUH lini.
	//
	// Terjadi bila LINEBUSINESS-nya kosong atau bernilai di luar yang dikenali. Work Owner
	// menetapkan perilaku ini ditiru dari Pega apa adanya: di sana potongan WHERE-nya tidak
	// terbentuk, sehingga kueri berjalan tanpa penyaring dan pengguna melihat semuanya —
	// tanpa satu pun pesan.
	//
	// RISIKO YANG DITERIMA SADAR: petugas yang datanya belum dilengkapi admin ikut melihat
	// lini yang bukan haknya, dan kegagalan itu tidak terlihat sebagai galat. Dicatat di
	// docs/keputusan-implementasi.md.
	Unrestricted bool

	// GroupPanels adalah daftar GROUPPANEL yang boleh dilihat. Kosong berarti tidak
	// menyaring berdasarkan lini — perhatikan bahwa itu hanya sah bila Unrestricted,
	// ATAU bila ExcludedBusinessGroups terisi (aturan BONDING).
	GroupPanels []string

	// ExcludedBusinessGroups adalah kelompok bisnis yang DIKECUALIKAN.
	//
	// Aturan NONMBU dan BONDING keduanya memuat `c.businessgroupid NOT IN (…)`. Di sistem
	// lama nilai itu datang lewat join ke `pooldata.businessgroup`; di tabel datar
	// `T_CLAIMLIST_ADMIN` ia kolom `BUSINESSGROUPID`, sehingga aturannya dapat diterapkan
	// apa adanya tanpa join.
	ExcludedBusinessGroups []string
}

// Nilai LINEBUSINESS yang dikenali.
//
// Keempatnya terbukti dipakai layar ini. Nilai lain mungkin ada — `pyPosition` di rule
// lain memuat sekurangnya satu nilai lagi — dan daftar pastinya BELUM DIPUTUSKAN.
//
// Karena itu tidak ada constraint maupun enum yang mengunci di basis data: nilai yang
// tidak dikenali jatuh ke Unrestricted, bukan ditolak. Mengunci daftarnya sekarang berarti
// mendahului keputusan yang sengaja ditinggalkan terbuka.
const (
	LineNonMBU  = "NONMBU"
	LineBonding = "BONDING"
	LinePA      = "PA"
	LineTravel  = "TRAVEL"
)

// Group Panel per lini, dari potongan WHERE sistem lama.
var (
	groupPanelsNonMBU = []string{"003", "004", "006"}
	groupPanelsPA     = []string{"002"}
	groupPanelsTravel = []string{"005"}
)

// excludedBusinessGroups adalah kelompok bisnis yang dikecualikan NONMBU dan BONDING.
//
// Keempat nilainya disalin apa adanya dari potongan WHERE sistem lama
// (`Activity/InboxOutstanding_Act-Act.xml:905` dan `:1081`). Artinya tidak diketahui — ia
// kode di master `businessgroup` yang tidak ada di export — sehingga nilainya
// dipertahankan tanpa ditafsirkan.
var excludedBusinessGroups = []string{"10008", "10010", "10015", "10023"}

// ScopeFor menerjemahkan nilai LINEBUSINESS menjadi batas data.
//
// Keempat cabang di bawah adalah salinan apa adanya dari potongan WHERE sistem lama —
// termasuk BONDING, yang seluruh aturannya bersandar pada pengecualian kelompok bisnis dan
// karena itu TIDAK menyaring Group Panel sama sekali.
func ScopeFor(lineBusiness string) LineScope {
	switch strings.ToUpper(strings.TrimSpace(lineBusiness)) {
	case LinePA:
		return LineScope{GroupPanels: groupPanelsPA}
	case LineTravel:
		return LineScope{GroupPanels: groupPanelsTravel}
	case LineNonMBU:
		return LineScope{
			GroupPanels:            groupPanelsNonMBU,
			ExcludedBusinessGroups: excludedBusinessGroups,
		}
	case LineBonding:
		// BONDING melihat SELURUH Group Panel, dikurangi kelompok bisnis yang
		// dikecualikan. Ia karena itu bukan Unrestricted — pengecualiannya nyata.
		return LineScope{ExcludedBusinessGroups: excludedBusinessGroups}
	default:
		// Termasuk nilai kosong. Lihat LineScope.Unrestricted.
		return LineScope{Unrestricted: true}
	}
}

// Filter adalah penyaring dan paginasi yang diminta layar.
type Filter struct {
	// Search mencari pada No Klaim, No Polis, dan PIC Teknik sekaligus.
	//
	// Ketiganya dalam SATU kotak, meniru layar lama yang berlabel
	// "No Klaim / No Polis / PIC". Pencarian dikerjakan SERVER, bukan peramban: tabel klaim
	// berisi puluhan juta baris (`D-10`), dan menyaring satu halaman dari sepuluh akan
	// memberi tahu pengguna bahwa sesuatu tidak ada padahal ia ada di halaman lain.
	Search string

	// Scope adalah batas data pemanggil. Ia BUKAN pilihan pengguna dan tidak pernah datang
	// dari badan permintaan — ia diturunkan dari identitasnya di server.
	Scope LineScope

	// Stage menyaring satu tahap tertentu; kosong berarti seluruh tahap.
	Stage string

	// BranchCode menyaring satu cabang; kosong berarti seluruh cabang.
	BranchCode string

	Limit  int
	Offset int
}

// Batas paginasi.
//
// `10-API-STRATEGY.md` §4 menetapkan limit maksimum 100 dan permintaan yang lebih besar
// DITOLAK, bukan dipenuhi. Normalize memangkasnya karena di sini ia berasal dari query
// string yang sudah divalidasi transport; penolakannya terjadi di sana.
const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)
	f.Stage = strings.TrimSpace(f.Stage)
	f.BranchCode = strings.TrimSpace(f.BranchCode)

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
// Total ikut dikembalikan meski `10-API-STRATEGY.md` §4 menyarankan menghindari COUNT(*)
// pada tabel besar. Alasannya: layar ini layar PEMANTAUAN, dan "berapa banyak klaim yang
// masih berjalan" adalah angka yang dicari pengguna — bukan sekadar hiasan paginasi.
// Bila kelak terbukti mahal, yang diganti adalah kueri hitungnya, bukan bentuk halaman.
type Page struct {
	Claims []OutstandingClaim
	Total  int
}

// Repo adalah seam ke penyimpanan klaim.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya.
// Diisi `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// Modul ini hanya MEMBACA. Tidak ada method tulis, dan ketiadaannya disengaja: klaim
// dimiliki modul `registrasi`, dan `P-1` menetapkan satu tabel hanya ditulis satu pemilik.
type Repo interface {
	List(ctx context.Context, f Filter) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `T_CLAIMLIST_ADMIN` ada di basis data SETIAP entitas — `ADR-0030` menetapkan satu database per
// portal, bukan satu database bersama dengan penanda entitas. Klaim milik Asuransi Sinar
// Mas dan klaim milik Simas Insurtech karena itu tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan klaim satu
// badan hukum di layar badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)

// LineBusinessRepo adalah seam ke POOLDATA.M_LOGIN_PNC.LINEBUSINESS.
//
// Kolom itu BELUM ADA; ia ditambahkan migrasi 0004 yang menempuh `D-63` dan belum
// dijalankan DBA di lingkungan mana pun. Sampai itu terjadi, adapter Oracle akan gagal
// membacanya — dan kegagalannya ditangani sebagai "tidak diketahui", yang jatuh ke
// Unrestricted persis seperti pengguna tanpa lini.
type LineBusinessRepo interface {
	// LineBusinessFor mengembalikan nilai LINEBUSINESS seorang pengguna, atau string
	// kosong bila barisnya tidak ada maupun kolomnya belum terisi.
	//
	// Ia TIDAK mengembalikan galat untuk "tidak ditemukan": pengguna yang tidak punya baris
	// di M_LOGIN_PNC adalah keadaan biasa hari ini, bukan kegagalan.
	LineBusinessFor(ctx context.Context, loginID string) (string, error)
}
