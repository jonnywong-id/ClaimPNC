// Package inboxoutstanding adalah inti modul **My Inbox** (MENU_ID 51).
//
// # Nama paketnya belum diganti, dan itu disengaja
//
// Modul ini semula dinamai "Inbox Outstanding" — judul yang memang ada di dalam
// `Section/InboxRegister_Section-Section.xml:2150`. Tetapi itu judul SECTION, bukan nama
// butir menu. `POOLDATA.M_MENU_APLIKASI_PNC` memuat keduanya sebagai butir BERBEDA:
//
//	MENU_ID 51 · "My Inbox"          · InboxRegister_Harness      <- modul ini
//	MENU_ID 79 · "Inbox Outstanding" · InboxOutstanding_Harness   <- layar LAIN
//
// MENU_ID 79 belum punya layar: harness-nya tidak ada di export (`K-33`). Penggantian nama
// paket menyentuh backend, frontend, dan tiket sekaligus (`D-81`) sehingga ditunda sampai
// Work Owner memutuskan; yang sudah diperbaiki adalah kunci menunya.
//
// # Yang dimigrasikan
//
//	Harness/InboxRegister_Harness-Harness.xml       layar rujukan
//	Section/InboxRegister_Section-Section.xml       kolom beserta judulnya
//	Report Definition/InboxRegister_RD-RD.xml       kueri dan kelima penyaringnya
//	Activity/SetClaimPNC-Act.xml                    tab status dokumen (BELUM dibangun)
//
// > Kueri dan batas data sempat diambil dari `BrowseInboxOutstanding1-SQL.xml` dan
// > `InboxOutstanding_Act-Act.xml`. Keduanya TIDAK pernah dipanggil harness maupun section
// > ini — pemanggilnya `SetDashboardClaim`, `ExportOutstanding`, dan `AlertPendingPLADLA`.
// > Akibatnya layar ini sempat berperilaku sebagai dashboard, bukan sebagai inbox.
//
// # Apa yang membuatnya "My"
//
// `InboxRegister_RD` menyaring lima syarat ber-AND, dan yang pertama menentukan segalanya:
//
//	A  Operator ID   =  Param.assign   <- OperatorID.pyUserIdentifier
//	B  GroupPanel    =  Param.panel
//	C  RCV_ID        =  Param.idRCV
//	D  Work Status  !=  "Resolved-Completed"
//	E  Work Status  !=  "Resolved-Rejected"
//
// Syarat D dan E itulah definisi "outstanding"; syarat A yang membuat daftarnya milik
// pemanggil. Tanpa A, layar ini menampilkan pekerjaan SELURUH operator — terisi, tampak
// wajar, dan salah tanpa satu pun galat.
//
// # Sumber datanya: POOLDATA.T_CLAIMLIST_ADMIN
//
// Work Owner menetapkan tabel itulah yang menggantikan `datapega.pc_asm_fw_gcnmfw_work`
// beserta `pc_assign_worklist`. Ia tabel DATAR: satu baris per klaim, tanpa satu pun join.
//
// Modul ini hanya MEMBACA. Yang mengisi tabel itu adalah sistem lama.
//
// # Alias Pega tidak dibawa masuk
//
// Kueri lama mengalias kolom secara MENYESATKAN — `a.policyno AS "District"`,
// `a.qqname AS "CountryID"`, `a.branchname AS "City"`. Tidak satu pun dibawa; nama di sini
// mengikuti padanan Inggris dari `CONTEXT.md` (`D-19`, `D-80`).
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxoutstanding

import (
	"context"
	"errors"
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
	// 006 Fire. Dipakai penyaring opsional `Param.panel`; layar ini TIDAK punya batas data
	// per lini — lihat catatan pada Filter.GroupPanel.
	GroupPanel string

	// RCVID — nomor register dokumen (`PNCCASEID`), padanan `.ClaimData.RCV_ID` pada RD.
	//
	// Dipakai penyaring opsional `Param.idRCV`. TIDAK ditampilkan sebagai kolom; section
	// rujukan tidak punya kolomnya.
	RCVID string

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

	// DocumentComplete menandai dokumen klaim sudah lengkap <- DOKUMENLENGKAP_1 = '1'.
	//
	// Bertipe bool, bukan tiga keadaan, karena aturan Pega memang hanya mengenal dua:
	// `= '1'` lengkap, `= '0' ATAU NULL` belum (`SetClaimPNC-Act.xml:1630`, `:1799`). NULL
	// bukan "tidak diketahui" di sana melainkan "belum lengkap", dan itulah yang membuat
	// donut tetap bermakna meski kolomnya belum pernah diisi.
	//
	// TIDAK ditampilkan sebagai kolom; section rujukan tidak punya kolomnya. Ia dibaca
	// untuk ringkasan dan untuk menyaring lewat irisan donut.
	DocumentComplete bool
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

// Filter adalah penyaring dan paginasi yang diminta layar.
type Filter struct {
	// Search mencari pada No Klaim, No Polis, dan PIC Teknik sekaligus.
	//
	// Ketiganya dalam SATU kotak, meniru layar lama yang berlabel
	// "No Klaim / No Polis / PIC". Pencarian dikerjakan SERVER, bukan peramban: tabel klaim
	// berisi puluhan juta baris (`D-10`), dan menyaring satu halaman dari sepuluh akan
	// memberi tahu pengguna bahwa sesuatu tidak ada padahal ia ada di halaman lain.
	Search string

	// AssignedTo adalah pemilik pekerjaan — inti layar ini, dan satu-satunya penyaring
	// yang WAJIB terisi.
	//
	// Ia BUKAN pilihan pengguna dan tidak pernah datang dari badan permintaan maupun query
	// string: ia diturunkan dari identitas pemanggil di server. Membiarkannya datang dari
	// klien berarti siapa pun dapat membaca pekerjaan orang lain dengan mengubah satu
	// parameter.
	//
	// Asalnya `Param.assign` pada `Report Definition/InboxRegister_RD-RD.xml`, yang diisi
	// `OperatorID.pyUserIdentifier` oleh section rujukan.
	AssignedTo string

	// AssignedToLegacy adalah identitas LAMA orang yang sama; kosong berarti tidak ada.
	//
	// # Kenapa satu orang punya dua identitas
	//
	// Login baru memakai **email** lewat HCC, sedangkan klaim warisan tertugas ke **nama
	// operator Pega**. Keduanya tidak akan pernah cocok, dan jembatannya adalah
	// `POOLDATA.T_ACCESS_GROUP_PNC` yang memetakan `OPERATOR_ID` → `OLD_OPERATOR_ID`.
	//
	// Sistem lama melakukan hal yang sama: `BrowseInboxPicTeknik-SQL.xml:21` menyaring
	// `pxassignedoperatorid IN {ASIS:TempOperator.CityID}`, dan daftar itu dirangkai
	// `Activity/SetClaimPNC-Act.xml:554` dari identitas lama DAN identitas sekarang.
	//
	// # Kenapa ini bukan kenyamanan, melainkan syarat agar layarnya berfungsi
	//
	// Terukur pada data ASM: **20 dari 29 operator** punya identitas lama yang berbeda,
	// dan salah satunya memegang **79 klaim berjalan yang seluruhnya tersimpan di
	// identitas lamanya** — nol di identitas barunya. Tanpa field ini, ia membuka My Inbox
	// dan melihat layar kosong yang tampak rapi, tanpa satu pun galat.
	AssignedToLegacy string

	// GroupPanel menyaring satu lini bisnis; kosong berarti seluruh lini.
	//
	// Di sistem lama ia `Param.panel` — PARAMETER, bukan pita tetap per pengguna. Layar ini
	// karena itu tidak punya batas data per lini, dan tidak membaca LINEBUSINESS.
	GroupPanel string

	// RCVID menyaring satu nomor register dokumen; kosong berarti seluruhnya.
	// Asalnya `Param.idRCV`.
	RCVID string

	// DocumentStatus menyaring satu status kelengkapan dokumen; kosong berarti seluruhnya.
	//
	// Ia diisi saat pengguna mengeklik irisan donut, dan itulah satu-satunya cara mengisinya
	// — sama seperti panel ringkasan Inbox Auto Claim yang menjadi satu-satunya penyaring
	// perusahaannya.
	DocumentStatus DocumentStatus

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
	f.AssignedTo = strings.TrimSpace(f.AssignedTo)
	f.AssignedToLegacy = strings.TrimSpace(f.AssignedToLegacy)
	f.GroupPanel = strings.TrimSpace(f.GroupPanel)
	f.RCVID = strings.TrimSpace(f.RCVID)
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

// DocumentStatus adalah status kelengkapan dokumen sebuah klaim.
//
// # Ia hanya DUA, padahal Pega punya sebelas tab
//
// `Section/InboxRegister_Section-Section.xml` memuat sebelas tab di bawah judul
// "Document status", disetel ke `TempVisibility.Email` 0…10. Tetapi kesebelasnya **tidak
// menghitung populasi yang sama**, sehingga tidak dapat menjadi irisan satu donut:
//
//	tab                          sumber                          dapat dihitung di sini?
//	---------------------------  ------------------------------  -----------------------
//	Complete / Not complete      Work-PNC + DOKUMENLENGKAP_1      ya — membagi habis
//	Loss Adjuster                Work-SurveyClaim SURVEYORTYPE=2  tidak — case type lain
//	Internal Surveyor            Work-SurveyClaim SURVEYORTYPE=1  tidak — case type lain
//	Temporary Close              Resolved-Completed + PendingClose  di LUAR himpunan ini
//	Not Answered / Replied       tabel komunikasi                 tidak
//
// `POOLDATA.T_CLAIMLIST_ADMIN` hanya memuat `Work-PNC` (870 baris) dan
// `Work-ReceiveDocument` (142) — tidak ada satu pun `Work-SurveyClaim`.
//
// Yang dijadikan irisan karena itu hanya kedua status yang **membagi habis** inbox
// pemanggil. Menggambar yang lain sebagai irisan bernilai nol akan membuat donutnya
// berbohong: bagian-bagiannya tidak menjumlah menjadi keseluruhan.
type DocumentStatus string

const (
	// StatusComplete — tab "Complete documents" (Email 1).
	StatusComplete DocumentStatus = "lengkap"

	// StatusIncomplete — tab "Documents not complete" (Email 0).
	//
	// NULL ikut dihitung "belum lengkap", persis seperti fragmen Pega
	// (`Activity/SetClaimPNC-Act.xml:1630`). Bukan penyederhanaan: kolomnya NULL pada
	// SELURUH 1.012 baris hari ini, sehingga tanpa aturan itu donutnya kosong sama sekali.
	StatusIncomplete DocumentStatus = "belum-lengkap"

	// StatusTemporaryClose — tab "Temporary Close" (Email 8).
	StatusTemporaryClose DocumentStatus = "temporary-close"

	// StatusDeadlineTemporaryClose — tab "Deadline To Temporary Close" (Email 9, non-PA).
	StatusDeadlineTemporaryClose DocumentStatus = "deadline-temporary-close"

	// StatusLossAdjuster — tab "Loss Adjuster" (Email 2).
	StatusLossAdjuster DocumentStatus = "loss-adjuster"

	// StatusInternalSurveyor — tab "Internal Surveyor" (Email 4).
	StatusInternalSurveyor DocumentStatus = "internal-surveyor"

	// StatusAll — tab "ALL Case" (Email 3).
	StatusAll DocumentStatus = "semua"

	// StatusCommunication — tab "Communication" (Email 5).
	//
	// SATU tab, bukan tiga. Nilai 6 dan 7 adalah keadaan DI DALAMNYA — "Not Replied From
	// Receiver" dan "Replied From Receiver" — dan section menyatakannya begitu:
	// `TempVisibility.Email=5 || 6 || 7` pada satu blok yang sama (`:27503`).
	StatusCommunication DocumentStatus = "komunikasi"

	// StatusTKA — tab "TKA" (Email 9, tetapi hanya bila `OperatorID.pyPosition == 'PA'`).
	//
	// Nilai 9 karena itu berarti DUA hal berbeda tergantung jabatan pemakainya —
	// `Section/InboxRegister_Section-Section.xml:12129` dan `:13312`. Di sistem baru
	// keduanya kode tersendiri, supaya artinya tidak bergantung pada siapa yang melihat.
	StatusTKA DocumentStatus = "tka"
)

// statusDefinition memuat seluruh yang membedakan satu tab dari tab lain.
//
// Ia satu tabel data, bukan rangkaian switch yang tersebar — bentuk yang sama dengan
// `categoryOrder` pada modul Inbox Laporan Klaim, dan dipilih dengan alasan yang sama:
// definisi baru dan tab lama dapat dibandingkan baris per baris saat `S-8` dijalankan.
type statusDefinition struct {
	status DocumentStatus
	title  string

	// legacy adalah nilai `TempVisibility.Email` di Pega. Ia disimpan supaya uji
	// kesetaraan dapat memanggil cabang yang sama persis.
	legacy string

	// countable menandai tab yang jumlahnya dapat dihitung dari `T_CLAIMLIST_ADMIN`.
	//
	// Yang tidak dapat dihitung TIDAK diberi angka nol — ia dikirim tanpa jumlah, dan
	// layar tidak menggambar lencananya. Alasannya sama persis dengan tab "Data rejected"
	// pada Inbox Laporan Klaim: lencana bertuliskan 0 menyatakan "tidak ada", dan itu
	// tidak benar — yang benar adalah "belum dihitung".
	countable bool
}

// statusOrder adalah urutan tab PERSIS seperti di layar Pega.
//
// Dibaca dari label tebal `Section/InboxRegister_Section-Section.xml` berurutan:
// `:11438`, `:11626`, `:11841`, `:12023`, `:12170`, `:12632`, `:12825`, `:12971`,
// `:13184`. Menata ulangnya — misalnya menaruh "ALL Case" di depan karena terasa wajar —
// akan memindahkan tab yang sudah dihafal petugas.
//
// # Kenapa hanya tiga yang dapat dihitung
//
//	Complete / Not complete   Work-PNC + DOKUMENLENGKAP_1        dapat
//	ALL Case                  seluruh inbox                      dapat
//	Temporary Close           Resolved-Completed + PendingClose  populasi LAIN
//	Deadline To Temp. Close   idem                               populasi LAIN
//	Loss Adjuster             Work-SurveyClaim SURVEYORTYPE=2    case type lain
//	Internal Surveyor         Work-SurveyClaim SURVEYORTYPE=1    case type lain
//	Communication             tabel komunikasi                   bukan klaim
//	TKA                       kolom TKA_1                        kolomnya TIDAK ADA
//
// `T_CLAIMLIST_ADMIN` hanya memuat `Work-PNC` (870 baris) dan `Work-ReceiveDocument`
// (142) — nol `Work-SurveyClaim`.
var statusOrder = []statusDefinition{
	{StatusComplete, "Complete documents", "1", true},
	{StatusIncomplete, "Documents not complete", "0", true},
	{StatusTemporaryClose, "Temporary Close", "8", false},
	{StatusDeadlineTemporaryClose, "Deadline To Temporary Close", "9", false},
	{StatusLossAdjuster, "Loss Adjuster", "2", false},
	{StatusInternalSurveyor, "Internal Surveyor", "4", false},
	{StatusAll, "ALL Case", "3", true},
	{StatusCommunication, "Communication", "5", false},
	{StatusTKA, "TKA", "9", false},
}

// FindDocumentStatus mencari tab dari kodenya. Nilai kedua false bila kodenya tidak dikenal.
//
// Kode tak dikenal DITOLAK, tidak diam-diam diartikan "semua" — pola yang sama dengan
// `FindCategory` pada Inbox Laporan Klaim, dan alasannya sama: salah ketik yang jatuh ke
// "semua" menghasilkan daftar yang tampak wajar tetapi bukan yang diminta.
func FindDocumentStatus(code string) (DocumentStatus, bool) {
	clean := DocumentStatus(strings.ToLower(strings.TrimSpace(code)))
	for _, d := range statusOrder {
		if d.status == clean {
			return d.status, true
		}
	}
	return "", false
}

// Title mengembalikan judul tab sebagaimana tertulis di layar Pega (`D-13`).
func (s DocumentStatus) Title() string {
	for _, d := range statusOrder {
		if d.status == s {
			return d.title
		}
	}
	return string(s)
}

// Countable menyatakan apakah jumlah tab ini dapat dihitung dari tabel yang ada.
func (s DocumentStatus) Countable() bool {
	for _, d := range statusOrder {
		if d.status == s {
			return d.countable
		}
	}
	return false
}

// StatusCount adalah satu tab beserta jumlahnya.
type StatusCount struct {
	Status DocumentStatus
	Label  string

	// Count nil berarti **belum dihitung**, dan itu BERBEDA dari nol.
	//
	// Layar tidak menggambar lencana untuk yang nil. Memberinya angka nol akan menyatakan
	// "tidak ada satu pun", padahal yang benar "sumber datanya belum dimigrasikan" —
	// perbedaan yang menentukan bagi petugas yang mencari pekerjaannya.
	Count *int
}

// Summary adalah ringkasan inbox pemanggil.
//
// Total sengaja IKUT dikembalikan, bukan dijumlahkan di peramban: ia mengisi tab
// "ALL Case", dan ia wajib cocok dengan total paginasi grid. Menjumlahkan tab di frontend
// akan diam-diam salah, karena sebagian tab tidak punya angka sama sekali.
type Summary struct {
	Status []StatusCount
	Total  int
}

// BuildSummary menyusun kesembilan tab dalam urutan layar.
//
// Penyimpanan cukup menyerahkan angka yang BERHASIL dihitungnya; sisanya otomatis
// dikirim tanpa jumlah. Bentuk ini dipilih supaya daftar tab hidup di SATU tempat —
// `statusOrder` — dan bukan tersebar di tiap adapter. Adapter yang lupa satu tab akan
// menghilangkannya dari layar tanpa satu pun galat.
func BuildSummary(counted map[DocumentStatus]int, total int) Summary {
	status := make([]StatusCount, 0, len(statusOrder))
	for _, d := range statusOrder {
		item := StatusCount{Status: d.status, Label: d.title}
		if d.countable {
			if n, ada := counted[d.status]; ada {
				angka := n
				item.Count = &angka
			}
		}
		status = append(status, item)
	}
	return Summary{Status: status, Total: total}
}

// LineBusiness adalah lini bisnis seorang petugas.
//
// # Ia hanya berlaku pada EXPORT, tidak pada daftar
//
// Daftar menyaring `PXASSIGNEDOPERATORID` dan tidak mengenal lini bisnis sama sekali
// (`Report Definition/InboxRegister_RD-RD.xml` memperlakukan panel sebagai PARAMETER).
// Export berbeda: `RDB List/ExportDataDetailKlaim-SQL.xml` **tidak menyaring operator
// sama sekali**, dan sebagai gantinya menyuntikkan cakupan lini bisnis lewat
// `{ASIS:TempBisnis.BUSINESSTYPE}`.
//
// Nilainya dipilih `Activity/ExportDataDetailKlaim-Act.xml` dari `OperatorID.pyPosition`.
// Di sistem baru padanannya `M_LOGIN_PNC.LINE_BUSINESS`.
type LineBusiness string

const (
	LinePA      LineBusiness = "PA"
	LineTravel  LineBusiness = "TRAVEL"
	LineNonMBU  LineBusiness = "NONMBU"
	LineBonding LineBusiness = "BONDING"

	// LineUnknown adalah petugas yang lini bisnisnya belum terisi.
	//
	// Sistem lama memperlakukannya sebagai **tanpa cakupan** — step terakhir
	// `ExportDataDetailKlaim-Act` menyetel `TempBisnis.BUSINESSTYPE` menjadi `""`,
	// sehingga export mengembalikan seluruh klaim yang masih berjalan.
	//
	// Perilaku itu DIPERTAHANKAN (`P-5`), dan konsekuensinya dicatat, bukan disembunyikan:
	// petugas tanpa lini bisnis mengunduh lebih banyak daripada petugas yang punya. Ini
	// bukan `R-20` — keempat cakupan berada di dalam SATU badan hukum, dan pemisahan antar
	// entitas tetap dijaga `RepoSelector`. Yang tepat memperbaikinya adalah mengisi
	// `M_LOGIN_PNC.LINE_BUSINESS`, bukan mengubah perilaku export di sini.
	LineUnknown LineBusiness = ""
)

// NormalizeLineBusiness membakukan nilai yang dibaca dari basis data.
//
// Nilai yang tidak dikenali menjadi LineUnknown — bukan ditolak. Kolomnya bebas isi dan
// baru terisi pada sebagian petugas; menolak nilai asing akan membuat export GAGAL bagi
// mereka, padahal sistem lama justru melayaninya tanpa cakupan.
func NormalizeLineBusiness(raw string) LineBusiness {
	switch LineBusiness(strings.ToUpper(strings.TrimSpace(raw))) {
	case LinePA:
		return LinePA
	case LineTravel:
		return LineTravel
	case LineNonMBU:
		return LineNonMBU
	case LineBonding:
		return LineBonding
	default:
		return LineUnknown
	}
}

// ExportFilter adalah penyaring unduhan CSV.
//
// # Kenapa ia TIPE TERSENDIRI, bukan Filter dengan satu field tambahan
//
// Keduanya menyaring hal yang berbeda, dan menyatukannya sempat membuat export salah:
// export dulu memanggil ulang daftar, sehingga ikut terkena penyaring
// `PXASSIGNEDOPERATORID` dan **mengembalikan berkas kosong bagi petugas yang inbox-nya
// kosong** — padahal di Pega ia tetap berisi.
//
//	                  daftar (RD)          export (ExportDataDetailKlaim)
//	operator          WAJIB disaring       TIDAK disaring sama sekali
//	lini bisnis       tidak dikenal        menentukan cakupannya
//	rentang tanggal   tidak ada            ada, opsional
//	alur & tugas      tidak disaring       disaring
//
// Tipe terpisah membuat perbedaan itu tidak mungkin tertukar lagi.
type ExportFilter struct {
	// LineBusiness menentukan cakupan baris. LineUnknown berarti tanpa cakupan.
	LineBusiness LineBusiness

	// From dan To menyaring tanggal pembuatan klaim; nil berarti tidak menyaring.
	//
	// Asalnya `{ASIS:TempBisnis.REGISTERID}`, yang dirangkai dari `TempBisnis.EDMDATE` dan
	// `TempBisnis.ENDDATE`; bila keduanya kosong, fragmennya kosong dan tidak ada penyaring
	// tanggal sama sekali.
	//
	// Keduanya TANGGAL, bukan timestamp. Pemanggil mengirim awal hari WIB pada From dan
	// awal hari BERIKUTNYA pada To; lihat catatan batas atas pada berkas SQL.
	From *time.Time
	To   *time.Time

	Limit  int
	Offset int
}

// Normalize mengembalikan penyaring export dengan nilai yang dijamin masuk akal.
func (f ExportFilter) Normalize() ExportFilter {
	f.LineBusiness = NormalizeLineBusiness(string(f.LineBusiness))
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

// MaxExportBatch adalah batas baris sekali ambil saat mengekspor.
//
// Ia lebih longgar dari MaxLimit karena export TIDAK dilihat manusia sebagai halaman —
// ia dibaca sekumpulan demi sekumpulan lalu langsung dialirkan ke berkas. Batas jumlah
// baris seluruhnya ada di lapisan transport, bukan di sini.
const MaxExportBatch = 1000

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

	// Export membaca baris untuk unduhan CSV — TANPA menyaring pemilik pekerjaan.
	//
	// Ia sengaja method tersendiri, bukan List dengan penyaring dikosongkan: List MENOLAK
	// filter tanpa pemilik (ErrAssigneeRequired), dan penolakan itu justru pengaman yang
	// tidak boleh dilemahkan demi export.
	Export(ctx context.Context, f ExportFilter) (Page, error)

	// LineBusinessFor membaca lini bisnis seorang petugas dari `M_LOGIN_PNC`.
	//
	// Petugas yang tidak punya baris, atau punya baris tetapi kolomnya kosong,
	// mengembalikan LineUnknown TANPA galat — keduanya keadaan yang sah, dan sistem lama
	// melayaninya sebagai "tanpa cakupan".
	LineBusinessFor(ctx context.Context, loginID string) (LineBusiness, error)

	// LegacyOperatorFor membaca identitas LAMA seorang petugas dari
	// `POOLDATA.T_ACCESS_GROUP_PNC`.
	//
	// Petugas yang tidak punya baris mengembalikan string kosong TANPA galat: ia berarti
	// identitasnya tidak pernah berganti, bukan bahwa ada yang salah.
	LegacyOperatorFor(ctx context.Context, loginID string) (string, error)

	// SummarizeDocumentStatus menghitung isi inbox per status kelengkapan dokumen.
	//
	// Penyaring status pada filter DIABAIKAN di sini — ringkasan harus tetap memuat seluruh
	// status supaya irisan yang sedang dipilih tetap terlihat. Menghormatinya akan membuat
	// donut menyusut menjadi satu irisan begitu pengguna mengeklik salah satunya, dan tidak
	// ada jalan kembali selain memuat ulang halaman.
	SummarizeDocumentStatus(ctx context.Context, f Filter) (Summary, error)
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

// ErrAssigneeRequired dikembalikan bila pemilik pekerjaan tidak terisi.
//
// Ia sengaja GAGAL, bukan jatuh ke "tampilkan semuanya". Layar ini bernama "My Inbox" dan
// satu-satunya yang membuatnya "My" adalah penyaring `PXASSIGNEDOPERATORID`; menjalankan
// kueri tanpa penyaring itu akan menampilkan pekerjaan SELURUH operator — terisi, tampak
// wajar, dan salah tanpa satu pun galat.
//
// Pola yang sama dipakai `TKT-F6-002` untuk portal: permintaan tanpa portal ditolak, tidak
// dilayani portal utama sebagai cadangan.
var ErrAssigneeRequired = errors.New("inboxoutstanding: pemilik pekerjaan wajib terisi")
