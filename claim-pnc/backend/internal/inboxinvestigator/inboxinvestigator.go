// Package inboxinvestigator adalah inti modul Inbox Investigator.
//
// # Layar apa ini
//
// Menu `MENU_ID 48` "Inbox Investigator" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `InboxInvestigator_Harness`. Judul yang dibaca pengguna di sistem lama adalah
// **"Inbox Investigator"**.
//
// Isinya **daftar pekerjaan yang menunggu Investigator** — klaim yang penugasannya berada
// di workbasket `InvestigatorPNC` dan belum selesai dikerjakan. Ia INBOX menurut keempat
// ciri `D-79`: barisnya pekerjaan, baris hilang setelah selesai, "hanya milik saya" adalah
// aturan kewenangan, dan barisnya punya tenggat.
//
// Modul INBOX pertama di aplikasi ini. Modul sebelumnya seluruhnya master data atau
// pencarian; pembedaan keduanya ditetapkan `D-79`, dan itulah yang menentukan bentuk modul
// ini — terutama kenapa penyaring "hanya workbasket InvestigatorPNC" bukan pilihan pengguna
// melainkan bagian dari kuerinya.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxInvestigator_Harness-Harness.xml   pembungkus layar; judul; 2 activity
//	Section/InputInvestigator_Section-Section.xml   grid, kolom, pyPageSize=50, Operator
//	Report Definition/InboxRegisterCompliance_RD-RD.xml  20 kolom, 2 penyaring, MaxRecords
//	When/IsInvestigator-When.xml                    kewenangan membuka menu
//	RDB List/ExportDatainvestigator-SQL.xml         INVESTIGATOR_TF_DATE, rentang tanggal
//	RDB List/CountKlaimPUCL-SQL.xml                 bentuk gabungan work + workbasket
//	RDB List/ReminderPUCL-SQL.xml                   nama kolom fisik PC_ASM_FW_GCNMFW_WORK
//	Database/INSERT_SURVEYORLIST.prc                kolom T_SURVEYORLIST, INDEX_SURVEY
//	Database/m_menu_aplikasi_pnc.csv                MENU_ID 48 "Inbox Investigator"
//
// # Tiga hal dari layar lama yang TIDAK dibawa
//
//  1. **Grid kedua**. Section-nya memuat DUA grid (`…BBBB` dan `…BBBBB`) dengan kolom dan
//     parameter IDENTIK — sisa Save-As dari Inbox Compliance; yang kedua bahkan mengeja
//     "Nama Bisinis".
//
//  2. **Batas `pyMaxRecords = 500` yang senyap**. Lihat MaxRows.
//
//  3. **Export Data Investigation** beserta ketiga kendalinya — dropdown "Pilih
//     Investigation", isian "Dari" dan "Sampai". **Dihapus atas keputusan Work Owner
//     2026-09-24.**
//
//     Keempatnya ada dan terlihat di layar lama (`pyVisible = ALWAYS`). Yang menghalangi
//     bukan lingkup melainkan pemetaan: berkas CSV-nya disusun dari 13 kolom milik 12
//     properti `SurveyList(1).*`, dan tiga di antaranya tidak dapat ditelusuri ke kolom
//     basis data mana pun — terutama `NoRekapMedis`, yang tidak punya kolom sama sekali di
//     `POOLDATA.INVESTIGATIONREPORT`.
//
//     Analisis lengkapnya disimpan di `docs/permintaan-artefak-pega.md` §2.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxinvestigator/          aturan modul + seam          ← paket ini
//	inboxinvestigator/usecase/  orkestrasi: buka daftar
//	inboxinvestigator/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxinvestigator/http/     lapisan transport modul ini  — handler, dto, rute
//
// # Yang BUKAN urusan paket ini
//
// Apa yang terjadi setelah sebuah baris dibuka. Di sistem lama baris menjalankan
// `SetAssignmentInboxPUCL_act` lalu `openAssignment` — mengambil penugasan dan membuka layar
// kerjanya. Layar kerja itu modul tersendiri yang belum ada, dan pencatatan hasil
// investigasi (`SetStatusInvestigator_Act`: `SurveyStatus=5`, `StatusClaim=1151`, kronologi
// TAT) ada di sana, bukan di sini. Modul ini mengetahui KEBERADAAN pekerjaannya — ia
// menyusun rujukannya, lihat Task.Reference — tetapi tidak mengubah satu baris pun.
package inboxinvestigator

import (
	"context"
	"strings"
	"time"
)

// Workbasket adalah antrean bersama yang isinya ditampilkan layar ini.
//
// Nilainya terbaca dari `Section/InputInvestigator_Section-Section.xml`, yang mengirimkannya
// ke Report Definition sebagai parameter:
//
//	<pyReportDefParams>
//	  <pyName>Operator</pyName>
//	  <pyValue>"InvestigatorPNC"</pyValue>
//
// Report Definition memakainya sebagai penyaring `newAssignPage.pxAssignedOperatorID =
// Param.Operator`.
//
// # Ia KONSTANTA, bukan penyaring yang dapat dipilih pengguna
//
// Di sistem lama nilainya tertanam di dalam section, sehingga layar ini hanya pernah
// menampilkan satu workbasket. Menjadikannya parameter permintaan berarti menyediakan cara
// membaca antrean peran lain — Compliance, RCL/PUCL, Komite — lewat endpoint Investigator.
// Itu bukan kesetaraan perilaku, melainkan kewenangan baru yang tidak pernah ada.
//
// Bahwa ia tertanam di kode dan bukan di master data adalah hal yang DISADARI dan
// bertentangan dengan `D-15`. Yang membuatnya tetap di sini: ia bukan nilai bisnis yang
// berubah menurut kebijakan, melainkan IDENTITAS layar ini — mengubahnya berarti layar ini
// menjadi layar lain. Bila kelak antrean investigator dipecah per lini bisnis, ia naik
// menjadi master data; sampai itu terjadi, memindahkannya hanya menambah satu tabel yang
// isinya satu baris dan tidak pernah berubah.
const Workbasket = "InvestigatorPNC"

// ResolvedWorkStatus adalah nilai `pyStatusWork` yang menandai pekerjaan sudah tuntas.
//
// Report Definition menyaring `.pyStatusWork != "Resolved-Completed"`, dan itulah yang
// membuat baris HILANG dari inbox setelah dikerjakan — ciri kedua Inbox pada `D-79`.
//
// Perhatikan bahwa penyaringnya "bukan selesai", bukan "sedang berjalan". Klaim yang
// berstatus `Resolved-Rejected` KARENA ITU TETAP TAMPIL. Itu perilaku sistem lama dan
// direplikasi apa adanya (`P-5`); apakah ia disengaja tidak dapat dibuktikan dari export.
const ResolvedWorkStatus = "Resolved-Completed"

// MaxRows membatasi jumlah baris yang dikembalikan satu permintaan.
//
// # Angkanya diwarisi, tetapi artinya berubah
//
// `pyMaxRecords = 500` pada `InboxRegisterCompliance_RD` MEMOTONG hasil tanpa memberi tahu
// siapa pun: baris ke-501 tidak pernah tampil, dan tidak ada apa pun di layar yang
// menyatakannya. Steering mencatat pola itu pada 54 dari 56 laporan
// (`15-NFR-PERFORMANCE-SCALABILITY.md`).
//
// Di sini angkanya dipertahankan — Work Owner memilih penyaringan dan paginasi dikerjakan
// PERAMBAN, dan mengirim seluruh antrean tanpa batas ke peramban akan mengubah layar
// menjadi tidak dapat dipakai jauh sebelum basis datanya keberatan. Yang BERUBAH adalah
// pemotongannya **dinyatakan**: Page.Truncated memberi tahu layar bahwa masih ada baris
// yang tidak terkirim, dan layar menyebutkannya kepada pengguna.
//
// Itu perbedaan yang menentukan. Batas yang diketahui adalah batas; batas yang senyap
// adalah data yang hilang.
const MaxRows = 500

// Task adalah satu baris inbox — satu klaim yang menunggu dikerjakan Investigator.
//
// # Kenapa namanya Task, bukan Claim
//
// Karena yang didaftar layar ini adalah PEKERJAAN, bukan klaim. Klaim yang sama dapat
// muncul di beberapa inbox pada waktu yang berbeda, dan yang membedakannya adalah penugasan
// — bukan klaimnya. `CONTEXT.md` menyebut satuan itu **Tugas**, dan `D-26` menetapkan ia
// selalu berada di Worklist atau di Workbasket. Yang ini di Workbasket.
//
// # Kesembilan kolom grid layar lama, pada urutannya
//
// Dibaca dari `Section/InputInvestigator_Section-Section.xml` beserta caption-nya. Kolom
// yang ADA di Report Definition tetapi TIDAK digambar grid — `SobName`, `GroupPanel`,
// `UserTeknis`, `PNCStatus`, `StatusClaim`, `isComplianceTransfer`, `TanggalBuatCompliance`
// — tidak dibawa: mengambil kolom yang tidak ada pembacanya hanya menambah lalu lintas,
// dan menampilkannya berarti mengarang kegunaan.
type Task struct {
	// Reference adalah `pzInsKey` — kunci teknis Pega, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada klaim warisan.
	//
	// Ia dibawa karena membuka pekerjaannya membutuhkannya, BUKAN untuk ditampilkan.
	// `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas Pega ke data bisnis
	// sebagai utang teknis, dan `D-22` menetapkan klaim terbitan sistem baru tidak pernah
	// menulis awalan itu lagi.
	//
	// Ia juga kunci yang menghubungkan baris ini ke T_CLAIM_OBJECTLIST dan
	// T_SURVEYORLIST; lihat berkas .sql.
	Reference string

	// CaseNumber adalah kolom "Nomor Case" — `pyID`.
	//
	// Inilah nomor yang disebut pengguna saat membicarakan sebuah pekerjaan. Dua format
	// hidup berdampingan selama masa paralel: `PNC-xxxx` dari Pega dan `PNCN.YY.xxxx` dari
	// sistem baru (`D-71`).
	CaseNumber string

	// PolicyNumber adalah kolom "No Polis" — `POLICYNO`.
	PolicyNumber string

	// InsuredName adalah kolom "Nama Tertanggung" — `QQNAME`.
	InsuredName string

	// ParticipantName adalah kolom "Nama Peserta" — `.ClaimData.ObjectList(1).ObjectName`,
	// yaitu nama objek pertanggungan PERTAMA pada klaim itu.
	//
	// Caption layar lamanya berbunyi "Nama Peserta", bukan "Nama Objek", dan itu dipakai
	// apa adanya (`D-13`). Penyebabnya terbaca: investigasi paling sering menyangkut lini
	// Personal Accident, tempat objek pertanggungan memang seorang peserta.
	//
	// # Satu klaim dapat punya banyak objek, dan hanya yang pertama yang tampil
	//
	// Itu perilaku sistem lama — Pega mengambil `ObjectList(1)`, elemen pertama page list.
	// Klaim berobjek banyak karena itu tampil seolah berobjek satu. Direplikasi apa adanya
	// (`P-5`); yang ditambahkan hanya kepastian URUTAN, lihat berkas .sql.
	ParticipantName string

	// BusinessName adalah kolom "Nama Bisnis" — `BUSINESSNAME`, lini bisnis klaim.
	//
	// Caption-nya muncul DUA KALI di section dengan ejaan berbeda — "Nama Bisnis" pada
	// judul kolom dan "Nama Bisinis" pada penyaring di atasnya. Salah ketik itu TIDAK
	// dibawa; yang dipakai ejaan yang benar.
	BusinessName string

	// BranchName adalah kolom "Nama Cabang" — `BRANCHNAME`, cabang yang menangani.
	BranchName string

	// AdminName adalah kolom "Nama Admin" — `pyOrigUserID`.
	//
	// Ia PEMBUAT kasus, bukan petugas yang sedang memegangnya. Pekerjaan di workbasket
	// memang belum bertuan (`D-26`) — itulah yang membuatnya antrean bersama — sehingga
	// tidak ada "sedang dikerjakan siapa" untuk ditampilkan.
	AdminName string

	// RegisteredAt adalah kolom "Tanggal Pendaftaran" — `pxCreateDateTime`.
	//
	// Nil bila kolomnya kosong. Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak
	// dapat dibedakan dari "belum diisi" saat ditampilkan, dan layar akan menuliskan
	// "01/01/0001" alih-alih tanda hubung.
	RegisteredAt *time.Time

	// SurveyDate adalah kolom KESEMBILAN grid, yang captionnya berbunyi
	// **"Lama Masuk Inbox"** — `.ClaimData.SurveyResults(1).SurveyDate`.
	//
	// # Captionnya menyebut durasi, isinya TANGGAL
	//
	// Ini bukan salah baca. Penelusuran sel per sel pada
	// `Section/InputInvestigator_Section-Section.xml` memasangkan kesembilan caption dengan
	// kesembilan sel datanya satu lawan satu, dan yang kesembilan berpasangan dengan
	// properti di atas:
	//
	//	Nomor Case          <-> .pyID                                   (tautan)
	//	No Polis            <-> .Policy.PolicyNo
	//	Nama Tertanggung    <-> .Policy.QQName
	//	Nama Peserta        <-> .ClaimData.ObjectList(1).ObjectName
	//	Nama Bisnis         <-> .Policy.Quotation.BusinessName
	//	Nama Cabang         <-> .Policy.Quotation.BranchName
	//	Nama Admin          <-> .pyOrigUserID
	//	Tanggal Pendaftaran <-> .pxCreateDateTime
	//	Lama Masuk Inbox    <-> .ClaimData.SurveyResults(1).SurveyDate   <-- INI
	//
	// Sebabnya terbaca: section ini Save-As dari inbox Compliance, tempat kolom bernama
	// sama memang berisi lama menunggu yang dirangkai `RDB List/GetSelisihJam_sql-SQL.xml`.
	// Di sini seseorang mengikat ulang selnya ke tanggal survei dan **captionnya tidak ikut
	// diganti** — bentuk yang sama dengan alias menyesatkan pada
	// `03-CURRENT-ARCHITECTURE.md` §4.2.
	//
	// Yang direplikasi adalah **isinya**: layar menampilkan tanggal survei di bawah caption
	// itu, persis sistem lama (`P-5`). Nama field di sini mengikuti ISI, bukan caption,
	// supaya pembaca kode berikutnya tidak menduga ada durasi yang harus dihitung.
	//
	// Nil bila klaimnya belum punya baris survei.
	SurveyDate *time.Time
}

// Filter mempersempit daftar yang dibaca layar.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
// **Tidak ada penyaring workbasket.** Ia konstanta; lihat Workbasket.
//
// **Tidak ada penyaring rentang tanggal**, meski layar lama punya isian "Dari" dan "Sampai".
// Keduanya milik tombol **Export Data Investigation**, bukan milik grid: kueri yang
// memakainya adalah `RDB List/ExportDatainvestigator-SQL.xml`, dan ia menyaring
// `INVESTIGATOR_TF_DATE` — kolom yang tidak dibaca grid sama sekali. Menerapkannya pada
// daftar berarti menyaring yang tampil dengan penyaring yang dibuat untuk hal lain.
//
// **Tidak ada penyaring status.** Report Definition menyaring `pyStatusWork` dengan nilai
// tetap, bukan dengan pilihan pengguna; lihat ResolvedWorkStatus.
type Filter struct {
	// Keyword mempersempit daftar pada Nomor Case, No Polis, Nama Tertanggung, Nama
	// Peserta, Nama Bisnis, Nama Cabang, dan Nama Admin.
	//
	// DITAMBAHKAN terhadap sistem lama, yang menyaring di peramban lewat kotak isian per
	// kolom pada kepala grid. Kosong berarti tanpa penyaring.
	//
	// # Layar TIDAK memakainya hari ini
	//
	// Work Owner memilih penyaringan dikerjakan peramban (2026-09-23), sehingga layar
	// memakai pencarian bawaan `DataTable` atas baris yang sudah di tangan — sama seperti
	// seluruh layar master. Penyaring ini tetap disediakan supaya perpindahan ke
	// penyaringan sisi server kelak (`TKT-U2-001`) tidak menuntut perubahan kontrak.
	Keyword string
}

// Clean memangkas spasi di kedua ujung penyaring.
func (f Filter) Clean() Filter {
	return Filter{Keyword: strings.TrimSpace(f.Keyword)}
}

// Page adalah satu halaman inbox beserta keterangan pemotongannya.
//
// Keduanya dikembalikan bersama, bukan lewat dua panggilan terpisah, supaya keterangan
// "terpotong" tidak dapat berasal dari saat yang berbeda dengan barisnya.
type Page struct {
	// Tasks adalah baris yang terkirim, sebanyak-banyaknya MaxRows.
	Tasks []Task

	// Truncated menyatakan masih ada baris yang cocok tetapi TIDAK terkirim.
	//
	// Inilah satu-satunya perbedaan yang disengaja terhadap `pyMaxRecords = 500` sistem
	// lama, yang memotong tanpa memberi tahu siapa pun. Lihat MaxRows.
	Truncated bool
}

// Repo adalah seam ke penyimpanan inbox investigator SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Hanya List, dan tidak ada satu pun yang menulis
//
// Layar ini tidak mengubah apa pun. Mengambil pekerjaan dari antrean, mencatat hasil
// investigasi, dan memindahkan klaim ke Analyst seluruhnya terjadi di layar kerja yang
// belum dibangun — lihat banner paket. Operasi yang tidak tersedia di seam ini tidak dapat
// dipakai kode yang ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan pekerjaan yang menunggu di workbasket Investigator, terpotong
	// pada MaxRows.
	List(ctx context.Context, filter Filter) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan antrean
// pekerjaan satu badan hukum — lengkap dengan nama tertanggung dan nama peserta — kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// TANPA seam Clock, dan ketiadaannya adalah hasil koreksi.
//
// Modul ini sempat memilikinya, karena kolom "Lama Masuk Inbox" dibaca sebagai durasi yang
// dihitung terhadap sekarang. Penelusuran sel per sel membuktikan kolom itu **menampilkan
// tanggal survei apa adanya** (lihat Task.SurveyDate), sehingga tidak ada satu pun nilai di
// modul ini yang bergantung pada jam dinding.
//
// Seam yang tidak ada yang bervariasi di baliknya bukan seam — ia hanya bahan rakitan yang
// harus diisi tanpa pernah dipakai.
