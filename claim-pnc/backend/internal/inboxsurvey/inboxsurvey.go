// Package inboxsurvey adalah inti modul **My Work** (MENU_ID 50) — antrean kerja
// Surveyor dan Loss Adjuster.
//
// # Nama modul ini
//
// `Database/m_menu_aplikasi_pnc.csv` memuat butirnya sebagai:
//
//	APP_ID 1 · MENU_ID 50 · "My Work" · InboxSurvey_Harness · MENU_ID_LEADER 2 (INBOX)
//
// Work Owner menyebutnya **"Inbox Survey/Loss Adjuster"**, dan `D-81` menetapkan nama modul
// mengikuti nama yang dipakai Work Owner. Nama paketnya karena itu `inboxsurvey`; judul yang
// dilihat pengguna tetap "My Work" apa adanya dari master menu (`D-13`).
//
// JANGAN tertukar dengan MENU_ID 80 "Lost Adjuster" (`LostAdjuster_harness`) — butir menu
// BERBEDA yang harness-nya tidak ada di export sama sekali (`K-33`), dan belum punya layar.
//
// # Artefak Pega yang dibaca
//
//	Harness/InboxSurvey_Harness-Harness.xml         2 tab besar: INBOX dan KPI
//	Section/InboxSurvey_section-Section.xml         13 kolom · 7 tab status · page size 15
//	Activity/SetTempLostAdjuster-Act.xml            mesin tab: kueri mana untuk tab mana
//	Activity/GetLostAdjuster_act-Act.xml            pemuatan awal
//	RDB List/BrowseLossAdjuster-SQL.xml             bentuk baris Loss Adjuster (type 2)
//	RDB List/BrowseInternalSurveyor-SQL.xml         bentuk baris Internal Surveyor (type 1)
//	RDB List/CountOSLostAdjuster-SQL.xml            SELURUH predikat tab, satu kueri
//	RDB List/GetDataCaseSurveyALL-SQL.xml           domain nilai ADJUSTERSTATUS_1
//	RDB List/GetLoginLeaderSurveyor-SQL.xml         jembatan login -> surveyor
//	RDB List/GetSummaryKPIAdjuster*-SQL.xml         ketiga bentuk ringkasan KPI
//	Database/INSERT_SURVEYORLIST.prc                kolom POOLDATA.T_SURVEYORLIST
//
// # Apa itu My Work
//
// Antrean kerja atas case type **`ASM-FW-GCNMFW-Work-SurveyClaim`** — bukan `Work-PNC`.
// Itu case type KELIMA, dan `03-CURRENT-ARCHITECTURE.md` §1 hanya menyebut empat
// (`Work-PNC`, `Work-Komite`, `Work-OpenProtection`, `Work-ReceiveDocument`). Ketiadaannya
// di Steering adalah kekurangan dokumen, bukan bukti ia tidak ada: 778 activity custom
// merujuknya, dan `POOLDATA.T_SURVEYORLIST` menyimpan barisnya.
//
// Ia benar-benar Inbox menurut `D-79`. Keempat cirinya terpenuhi: barisnya **pekerjaan**
// (satu janji survei yang menunggu dikerjakan), barisnya **hilang** setelah ditutup, "hanya
// milik saya" adalah **aturan kewenangan** (disaring identitas surveyor), dan barisnya punya
// **tenggat** — kolom Aging ada justru untuk itu.
//
// # Ia melayani DUA populasi sekaligus
//
// `Activity/SetTempLostAdjuster-Act.xml` menyetel penyaringnya berbeda menurut siapa yang
// masuk — terbaca dari dua Property-Set pada `tempQuery.UserTeknisEmail`:
//
//	"AND KODECABANG_1 IN (" + Local.IDCabang + ")"   petugas internal, dibatasi cabangnya
//	"and SurveyorType_1='1'"                          Internal Surveyor
//
// sementara `BrowseLossAdjuster` mematok `SURVEYORTYPE_1 = '2'` untuk Loss Adjuster.
// Keputusan Work Owner 2026-09-28: **keduanya dilayani, dipilih dari identitas yang masuk**.
//
// # DUA HAL YANG HARUS DISADARI SEBELUM MEMBACA SISA BERKAS INI
//
// ## 1. Sumber datanya BUKAN tabel yang dipakai inbox lain
//
// Seluruh kueri layar ini di Pega membaca `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dengan
// `PXOBJCLASS = 'ASM-FW-GCNMFW-Work-SurveyClaim'`. Tabel itu dicabut dari pemakaian
// (keputusan Work Owner 2026-09-28), dan `POOLDATA.T_CLAIMLIST_ADMIN` — penggantinya yang
// pertama — **tidak memuat satu pun baris `Work-SurveyClaim`**; isinya 870 `Work-PNC` dan
// 142 `Work-ReceiveDocument`, diverifikasi langsung ke basis data.
//
// Keputusan Work Owner 2026-09-29 menggeser sumbernya sekali lagi:
//
//	POOLDATA.T_SURVEYORLIST   menggerakkan baris  (satu baris per JANJI SURVEI)
//	POOLDATA.T_CLAIM_PNC      menyediakan header  (satu baris per KLAIM)
//
// disambung `c.CLAIMID = s.PNCCASEID`. Kunci itu bukan tebakan —
// `RDB List/BroswseKlaimByNoSurvey-SQL.xml` menyambungkan keduanya persis begitu.
//
// Kenapa bukan tetap di `T_CLAIMLIST_ADMIN`: tabel datar itu hanya memuat klaim yang tugasnya
// berada di antrean Admin — empat label, salah satunya Choose Surveyor. Survei yang SEDANG
// BERJALAN berarti klaimnya sudah MELEWATI tahap itu, sehingga `INNER JOIN` ke sana membuang
// justru baris yang dicari layar ini. Layarnya akan terisi sebagian, dan tampak wajar.
//
// ### Akibat yang harus terlihat: isian yang belum tersedia
//
// Beberapa nilai layar berasal dari objek kerja `Work-SurveyClaim`, yang di Pega diratakan ke
// `PC_ASM_FW_GCNMFW_WORK` dengan akhiran `_1`. Di `T_SURVEYORLIST` akhiran itu TIDAK dipakai,
// dan penamaan itulah yang berlaku di sini:
//
//	Pega                 T_SURVEYORLIST    keadaan per 2026-10-03
//	ADJUSTERSTATUS_1     STS_SURVEY        ADA dan TERISI — tidak menghalangi
//	ADJUSTERACCEPT_1     ADJUSTERACCEPT    ADA, seluruh barisnya masih KOSONG
//	PYSTATUSWORK         PYSTATUSWORK      ADA, seluruh barisnya masih KOSONG
//	REFNO_1              REFNO             ADA & kosong — tetapi BUKAN asal "Reference No"
//
// **Kolom yang ADA tetapi KOSONG tidak lebih siap daripada kolom yang tidak ada**, dan pada tab
// Outstanding ia justru lebih berbahaya: penyaringnya `ADJUSTERACCEPT IS NULL` bernilai benar
// untuk SELURUH antrean, sehingga tabnya terisi wajar dan isinya salah. Karena itu keduanya
// sama-sama menahan tab — lihat UnavailableReason.
//
// ### Dua dugaan yang DICABUT 2026-10-03
//
// Keduanya dipatahkan Work Owner yang melihat layar Pega berjalan, bukan oleh pembacaan ulang.
// Dicatat supaya tidak dihidupkan kembali oleh pembaca berikutnya:
//
//	"Appointment No" berisi nama adjuster dari ADJUSTERPIC_1
//	    SALAH. Isinya `SRV-xxxxx` — nomor berkas survei. Sudah di tangan modul ini sebagai
//	    CASEID, dan `ADJUSTER_PIC` TIDAK perlu diminta sama sekali. Lihat AppointmentNo.
//
//	"Reference No" berasal dari REFNO_1
//	    SALAH. `REFNO_1` dialiaskan "Province" di kedua kueri rujukan, dan properti itu tidak
//	    termasuk 13 sel data grid — ia diambil lalu tidak pernah digambar.
//
// Akar kedua kekeliruan ini sama: pemetaan judul-ke-kolom disusun dari kueri **rujukan**
// (`BrowseLossAdjuster`, `BrowseInternalSurveyor`), padahal keempat kueri yang BENAR-BENAR
// dipakai tab hilang dari export (`R-16`, lihat §2 di bawah). Rujukan terdekat bukan sumber.
//
// ## 2. EMPAT kueri tab HILANG dari export
//
// `SetTempLostAdjuster` memanggil `BrowseOSLostAdjuster`, `BrowseConfirmLostAdjuster`,
// `BrowseCommunicationLostAdjuster`, dan `BrowseCloseLostAdjuster`. Tidak satu pun ada —
// diperiksa lewat isi `pyRuleName`, bukan lewat nama berkas (`R-16`).
//
// Yang menyelamatkan modul ini: **predikat keempatnya tetap terbaca**, karena
// `CountOSLostAdjuster` menghitung ketujuh keranjang yang sama dalam satu kueri. Yang TIDAK
// terbaca hanyalah daftar SELECT dan urutan masing-masing, dan untuk itu `BrowseLossAdjuster`
// menjadi rujukan terdekat. Perbedaannya dinyatakan ke pengguna, bukan disamarkan.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxsurvey

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Jenis surveyor — kolom `SURVEYORTYPE_1`.
//
// Satu digit ini menentukan POPULASI yang dilihat pengguna, dan salah satu digit menukar
// antrean Loss Adjuster dengan antrean Internal Surveyor tanpa satu pun galat: kedua layar
// berkolom sama, terisi wajar, dan hanya isinya yang milik orang lain.
const (
	// SurveyorTypeInternal adalah petugas survei INTERNAL ASM.
	//
	// `BrowseInternalSurveyor-SQL.xml` mematoknya, dan `SetTempLostAdjuster` menyetel
	// `"and SurveyorType_1='1'"` untuk jalur ini.
	SurveyorTypeInternal = "1"

	// SurveyorTypeLossAdjuster adalah adjuster EKSTERNAL.
	//
	// `BrowseLossAdjuster-SQL.xml` dan `GetDataCaseSurveyALL-SQL.xml` keduanya mematoknya.
	SurveyorTypeLossAdjuster = "2"

	// SurveyorTypeThree dan SurveyorTypeFour ADA, dan itu ditemukan terlambat.
	//
	// Dua rule ekspor menyaring `SURVEYORTYPE_1 IN ('2','3','4')` lalu mengambil
	// `Adjusterpic_1` — artinya ketiganya diperlakukan sebagai adjuster, bukan hanya `'2'`:
	//
	//	RDB List/ExportDataDetailKlaim-SQL.xml
	//	RDB List/ExportDataKomitesKlaimNONMBU-SQL.xml
	//
	// Katalog kolom mencatat `SURVEYORTYPE_1` punya **4 nilai berbeda**, sejalan dengan itu.
	//
	// ARTI keduanya TIDAK terbaca dari export — tidak satu pun rule memberinya label. Karena
	// itu keduanya dinamai menurut nilainya, bukan menurut dugaan artinya: nama yang
	// mengarang akan dipercaya pembaca berikutnya.
	//
	// Modul ini TIDAK menyaring dengan jenis surveyor — ia hanya menampilkannya — sehingga
	// ketiadaan arti keduanya belum berakibat. Ia akan berakibat pada modul pertama yang
	// menyaring dengannya, dan keempatnya ditulis di sini supaya model dua-populasi tidak
	// terlanjur dipakai.
	SurveyorTypeThree = "3"
	SurveyorTypeFour  = "4"
)

// Nilai `ADJUSTERSTATUS_1` — kolom yang digambar sebagai **"Status ASM"** sekaligus yang
// menggerakkan dua tab.
//
// Ketiganya terbaca berdampingan pada satu klausa di
// `RDB List/GetDataCaseSurveyALL-SQL.xml`:
//
//	and ADJUSTERSTATUS_1 not in ('Final Report','Close Case','Invoice Fee')
//
// Kolomnya punya 19 nilai berbeda di produksi (`docs/kolom-t-claimlist-admin.md`), sehingga
// ketiga di bawah ini adalah yang TERBUKTI dipakai sebagai penyaring — bukan seluruh
// domainnya. Ketiadaan enam belas sisanya di sini bukan kelalaian: tidak satu pun dipakai
// menyaring di export, dan menuliskannya berarti mengarang.
const (
	// StatusFinalReport — laporan akhir sudah masuk.
	//
	// TIDAK dipakai menyaring tab mana pun di sini; ia ada supaya ketiganya terbaca sebagai
	// satu keluarga, dan supaya pembaca berikutnya tidak menduga `Close Case` berdiri
	// sendiri.
	StatusFinalReport = "Final Report"

	// StatusCloseCase menggerakkan tab **Close**.
	//
	// # Ini SELISIH TERENCANA, dan bukan kesetaraan
	//
	// Di Pega tab Close menyaring `PYSTATUSWORK = 'Resolved-Completed'` milik objek kerja
	// SurveyClaim — status ALUR KERJA. `T_SURVEYORLIST` tidak punya kolom itu, dan
	// `T_CLAIMLIST_ADMIN.PYSTATUSWORK` adalah status KLAIM, bukan status survei.
	//
	// Yang dipakai karena itu `ADJUSTERSTATUS_1 = 'Close Case'` — status yang dicatat
	// adjuster. Keduanya berkorelasi tetapi tidak sama: survei ber-`Close Case` yang objek
	// kerjanya belum ditutup akan muncul di sini sementara di Pega tidak.
	//
	// Ia dinyatakan ke pengguna lewat PlannedDifferences (`D-54`), bukan disamarkan sebagai
	// layar yang sudah setara.
	StatusCloseCase = "Close Case"

	// StatusInvoiceFee menggerakkan tab **Invoice**.
	//
	// Ini satu-satunya predikat tab yang terbaca UTUH dan apa adanya dari export:
	// `SetTempLostAdjuster` menyetel `tempQuery.UserAdmin` menjadi
	// `"and a.AdjusterStatus_1='Invoice Fee'"`.
	StatusInvoiceFee = "Invoice Fee"
)

// AdjusterConfirmed adalah nilai `ADJUSTERACCEPT` — `ADJUSTERACCEPT_1` di Pega — yang berarti
// adjuster SUDAH menerima penugasan.
//
// Ia membelah antrean menjadi dua tab yang berlawanan, dan pembelahannya terbaca dari
// `CountOSLostAdjuster`:
//
//	AdjusterAccept_1 IS NULL   -> Outstanding  (belum dikonfirmasi)
//	AdjusterAccept_1 = '1'     -> ALL/Confirm  (sudah dikonfirmasi)
//
// Perhatikan lawannya `IS NULL`, BUKAN `<> '1'`. Keduanya berbeda pada baris bernilai `'0'`
// atau teks kosong — dan kolomnya punya 2 nilai berbeda di produksi, sehingga baris seperti
// itu mungkin ada. Yang dibawa adalah predikat Pega apa adanya (`P-5`).
const AdjusterConfirmed = "1"

// Status alur kerja berkas survei — kolom `PYSTATUSWORK` milik objek `Work-SurveyClaim`.
//
// # Kenapa dua nilai ini yang paling penting di seluruh berkas ini
//
// Karena ia **penentu baris mana yang sedang berjalan**, bukan sekadar penggerak satu tab.
//
// Satu berkas survei dapat memiliki beberapa baris di `POOLDATA.T_SURVEYORLIST`: kolom
// `INDEX_SURVEY` adalah nomor urut survei ke-berapa, dan nilainya BERTAMBAH setiap kali survei
// baru ditambahkan — terbaca dari `Activity/SetSurveyorList-Act.xml`:
//
//	TempSurvey.IdxSurveyResults := @if(Param.IndexSurvey=="", local.index+1, Param.IndexSurvey)
//	childPageSurveyClaim.SurveyData.SurveyList(<LAST>).IdxSurveyResults := local.index+1
//
// dan dibaca kembali sebagai riwayat oleh `RDB List/GetDataProgressSurvey-SQL.xml`
// (`order by to_number(index_survey) asc`).
//
// Work Owner menegaskan 2026-09-29: **hanya SATU survei yang berjalan**; bila ada lebih dari
// satu, sisanya sudah dibatalkan — dan yang berjalan dikenali dari **status Pega-nya yang
// bukan `Resolved-Completed` maupun `Resolved-Rejected`**.
//
// # Ini yang menutup cacat penggandaan baris
//
// Kedua Browse rule yang tersisa memasang penyaring itu di tingkat teratas:
//
//	BrowseLossAdjuster-SQL.xml:    A.PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')
//	BrowseInternalSurveyor-SQL.xml: idem
//	CountOSLostAdjuster-SQL.xml:   idem pada ENAM tab, dan `= 'Resolved-Completed'` pada tab Close
//
// Tanpa kolom itu, antrean menampilkan **seluruh riwayat survei**, termasuk yang sudah
// dibatalkan — satu berkas survei tampil beberapa kali dengan Claim No yang sama berulang.
// Itu keadaan yang berlaku hari ini, dan dinyatakan lewat Limitations (`D-54`).
//
// Kolomnya BELUM ADA di `POOLDATA.T_SURVEYORLIST`. Kedua nilai di bawah ditulis sekarang supaya
// predikatnya terekam dan siap dipasang begitu kolomnya tiba — bukan digali ulang dari awal.
const (
	// StatusWorkCompleted — berkas survei selesai. Menggerakkan tab Close di Pega.
	StatusWorkCompleted = "Resolved-Completed"

	// StatusWorkRejected — berkas survei ditolak atau dibatalkan.
	StatusWorkRejected = "Resolved-Rejected"
)

// ClosedWorkStatuses menyerahkan status yang berarti berkas survei SUDAH tidak berjalan.
//
// Dikembalikan sebagai daftar, bukan dua konstanta terpisah, supaya pemanggil tidak dapat
// memasang salah satunya dan lupa yang lain — kelalaian yang menghasilkan antrean yang
// memuat survei batal tanpa satu pun galat.
func ClosedWorkStatuses() []string {
	return []string{StatusWorkCompleted, StatusWorkRejected}
}

// Status komunikasi — kolom `KOMUNIKASISTATUS` pada `POOLDATA.M_KOMUNIKASI_PNC`.
//
// Ketiga tab komunikasi seluruhnya dibentuk dari kombinasi status ini dengan `SENDER`, dan
// kombinasinya terbaca dari `CountOSLostAdjuster` berdampingan dengan Property-Set pada
// `SetTempLostAdjuster`.
const (
	// CommunicationOpen — pesan belum dijawab.
	CommunicationOpen = "0"

	// CommunicationAnswered — pesan sudah dijawab.
	CommunicationAnswered = "1"
)

// Tab adalah satu keranjang pada bilah status layar.
//
// # Kenapa tab menjadi tipe, bukan teks bebas
//
// Karena masing-masing menjawab pertanyaan yang BERBEDA atas tabel yang SAMA, dan tab yang
// salah menghasilkan layar yang terisi wajar dengan isi yang keliru — bukan galat. Menjadikan
// nilainya tertutup membuat tab yang tidak dikenal DITOLAK, bukan diam-diam jatuh ke ALL.
type Tab string

// Ketujuh tab, dalam urutan tampilnya di `Section/InboxSurvey_section-Section.xml`.
//
// Urutan di section: Outstanding · Invoice · Close · ALL · Not answered communication ·
// Not replied from ASM · Replied from ASM. Urutan itu dibawa apa adanya (`D-13`).
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang muncul di URL dan dibaca layar
// (`D-80`); yang berbahasa Inggris hanyalah nama konstantanya.
const (
	// TabOutstanding — penugasan yang BELUM dikonfirmasi adjuster.
	TabOutstanding Tab = "outstanding"

	// TabInvoice — sudah dikonfirmasi DAN berstatus `Invoice Fee`.
	//
	// Ia irisan, bukan keranjang tersendiri: `SetTempLostAdjuster` menambahkan penyaring
	// `AdjusterStatus_1` DI ATAS penyaring Confirm, tidak menggantikannya.
	TabInvoice Tab = "invoice"

	// TabClose — berstatus `Close Case`. Lihat StatusCloseCase soal selisihnya.
	TabClose Tab = "close"

	// TabAll — sudah dikonfirmasi adjuster.
	//
	// Namanya "ALL" di layar lama, tetapi ia BUKAN seluruh baris: `CountOSLostAdjuster`
	// menghitungnya sebagai `AdjusterAccept_1 = '1'`. Penamaannya menyesatkan sejak di Pega,
	// dan nama itu dibawa apa adanya (`D-13`) sementara artinya dijelaskan di sini.
	TabAll Tab = "all"

	// TabNotAnswered — ada pesan terbuka dari pihak LAIN yang belum dijawab pemanggil.
	//
	//	KOMUNIKASISTATUS = '0'  AND  SENDER <> pemanggil
	TabNotAnswered Tab = "belum-dijawab"

	// TabNotReplied — pemanggil sudah mengirim, ASM belum membalas.
	//
	//	KOMUNIKASISTATUS = '0'  AND  SENDER = pemanggil
	TabNotReplied Tab = "belum-dibalas-asm"

	// TabReplied — pesan pemanggil SUDAH dibalas.
	//
	//	KOMUNIKASISTATUS = '1'  AND  SENDER = pemanggil
	TabReplied Tab = "sudah-dibalas-asm"
)

// DefaultTab adalah keranjang yang terbuka saat layar dibuka pertama kali.
//
// Outstanding, bukan ALL: ia yang berisi pekerjaan yang BELUM disentuh siapa pun, dan itulah
// alasan seseorang membuka layar bernama My Work.
const DefaultTab = TabOutstanding

// tabOrder mengunci urutan tampil sekaligus menjadi daftar tab yang sah.
var tabOrder = []Tab{
	TabOutstanding,
	TabInvoice,
	TabClose,
	TabAll,
	TabNotAnswered,
	TabNotReplied,
	TabReplied,
}

// Tabs menyerahkan salinan urutan tab.
func Tabs() []Tab {
	result := make([]Tab, len(tabOrder))
	copy(result, tabOrder)
	return result
}

// UnavailableReason menyatakan kenapa sebuah tab BELUM dapat dihitung, atau kosong bila ia
// dapat dihitung.
//
// Readiness menyatakan kolom mana yang BENAR-BENAR dapat dipakai pada satu portal.
//
// # Kenapa ia ada, dan kenapa ia menggantikan konstanta
//
// Sampai 2026-10-07, ketersediaan tab adalah **konstanta di dalam kode**: sebuah `switch` yang
// selalu mengembalikan kalimat "kolomnya masih kosong". Akibatnya layar menyatakan hal yang
// tidak lagi benar begitu kolomnya terisi — dan tidak ada cara mengetahuinya selain menyunting
// kode.
//
// Itu benar-benar terjadi: `pega_dev83` sudah terisi penuh pada 2026-10-07, dan layar tetap
// menyatakan seluruh barisnya kosong.
//
// # Kenapa PER PORTAL, bukan sekali saat aplikasi menyala
//
// `D-75` menetapkan **satu basis data per entitas**. Kolom yang sudah ada di satu portal belum
// tentu ada di portal lain — dan itu bukan kemungkinan teoretis: per 2026-10-07 kelimanya ada
// di dev dan **belum ada di produksi**.
//
// Satu nilai global karena itu akan salah pada salah satu portal, dan salahnya tidak kelihatan:
// portal yang kolomnya belum ada akan mencoba membacanya dan menjatuhkan seluruh layar dengan
// ORA-00904, atau sebaliknya menahan tab yang sebenarnya sudah siap.
//
// # Nol berarti TIDAK SIAP, dan itu sisi yang aman
//
// Bila pemeriksaannya gagal — hak akses kurang, basis data tidak terjangkau — seluruh field
// bernilai `false`, dan layar kembali ke perilaku hari ini: tab ditahan beserta sebabnya.
// Tidak pernah sebaliknya, karena kesalahan ke arah "siap" berarti membaca kolom yang mungkin
// tidak ada.
type Readiness struct {
	// AdjusterAccept menggerakkan tab Outstanding, ALL, dan Invoice.
	AdjusterAccept bool

	// WorkStatus menggerakkan tab Close dan penyaring berkas yang sudah tutup.
	WorkStatus bool

	// Reference menggerakkan kolom "Reference No" dan setengah kotak cari.
	Reference bool

	// AdjusterPIC menentukan kolom "PIC Loss Adjuster" memakai sumber sebenarnya atau pengganti.
	AdjusterPIC bool

	// SurveyLocation menentukan hal yang sama untuk kolom "Location".
	SurveyLocation bool
}

// Complete menyatakan KELIMA kolom siap, sehingga kueri varian penuh dapat dipakai.
//
// # Kenapa satu bendera menentukan kuerinya, bukan lima
//
// Kueri varian penuh menyebut kelima kolom sekaligus. Satu saja yang belum ada menghasilkan
// **ORA-00904 saat parse** — sebelum satu baris pun dibaca, dan tidak dapat dihindari dengan
// percabangan apa pun di dalam SQL. Jadi pilihannya memang biner: pakai varian penuh, atau
// tidak sama sekali.
//
// Kelimanya pun tiba bersama-sama — satu `ALTER`, satu backfill — sehingga keadaan "sebagian
// siap" bersifat sementara dan tidak layak dilayani kueri tersendiri. Pada keadaan itu modul
// bertahan di varian terbatas, yang selalu benar.
func (r Readiness) Complete() bool {
	return r.AdjusterAccept && r.WorkStatus && r.Reference &&
		r.AdjusterPIC && r.SurveyLocation
}

// # Kenapa ini ada, dan kenapa tabnya tetap digambar
//
// Keempat tab di bawah bergantung pada kolom yang BELUM DAPAT DIPERCAYA. Per 2026-09-30
// kolomnya sudah ditambahkan ke `POOLDATA.T_SURVEYORLIST` lewat `ALTER`, tetapi **belum ada
// yang mengisinya**: seluruh 17.641 baris bernilai NULL.
//
// Tabnya TETAP digambar, dan itu disengaja. `D-13` menetapkan bentuk layar mengikuti Pega;
// menghapus empat dari tujuh tab akan membuat pengguna yang hafal layarnya mengira fiturnya
// hilang. Yang berubah hanyalah: tab itu MENJELASKAN kenapa ia kosong, alih-alih
// menampilkan daftar kosong yang terbaca sebagai "tidak ada pekerjaan".
//
// Perbedaan itu yang menentukan. Daftar kosong tidak pernah dilaporkan siapa pun sebagai
// kerusakan; tab yang menyebut sebabnya akan.
//
// # Kenapa teksnya menyebut "belum terisi", bukan "belum ada"
//
// Karena keduanya diperbaiki orang yang berbeda. Kolom ditambahkan DBA lewat satu `ALTER`;
// isinya ditulis Tim Pega lewat jalur pemutakhiran saat status objek kerja berubah. Pesan yang
// menyebut sebab yang salah akan mengirim orang yang memperbaikinya ke arah yang keliru — dan
// pernah persis begitu: teks lama masih menyebut `ADJUSTERACCEPT_1` (nama Pega, berakhiran
// `_1`) padahal kolom yang ditambahkan bernama `ADJUSTERACCEPT`, sehingga kolom yang SUDAH ada
// terbaca sebagai belum ada.
func (r Readiness) UnavailableReason(t Tab) string {
	switch t {
	case TabOutstanding, TabAll, TabInvoice, TabClose:
		if r.Complete() {
			return ""
		}
		return "Belum dapat dihitung. " + r.missing() +
			" Menghitungnya sekarang akan menghasilkan angka yang terlihat wajar dan salah."

	default:
		// Ketiga tab komunikasi tidak menyentuh satu pun kolom itu. Ia berjalan sejak hari
		// pertama dan tidak pernah ditahan.
		return ""
	}
}

// missing menyebut kolom mana yang menahan, supaya pesannya menunjuk ke pekerjaan yang nyata.
//
// Pesan "belum tersedia" tanpa menyebut sebabnya akan membuat pembacanya menebak — dan pada
// modul ini sebabnya bisa dua hal yang DIPERBAIKI ORANG BERBEDA: kolom ditambahkan DBA lewat
// `ALTER`, isinya ditulis Tim Pega lewat jalur pemutakhiran.
func (r Readiness) missing() string {
	var names []string
	for _, c := range []struct {
		ready bool
		name  string
	}{
		{r.AdjusterAccept, "ADJUSTERACCEPT"},
		{r.WorkStatus, "PYSTATUSWORK"},
		{r.Reference, "REFNO"},
		{r.AdjusterPIC, "ADJUSTER_PIC"},
		{r.SurveyLocation, "RESCHEDULE_LOCATION"},
	} {
		if !c.ready {
			names = append(names, c.name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Kolom berikut pada POOLDATA.T_SURVEYORLIST belum dapat dipakai — belum ada, atau " +
		"ada tetapi seluruh barisnya masih kosong: " + strings.Join(names, ", ") + "."
}

// TabAvailable menyatakan apakah tab ini dapat dihitung dari data yang ada HARI INI.
func (r Readiness) TabAvailable(t Tab) bool { return r.UnavailableReason(t) == "" }

// DefaultAvailableTab mengembalikan tab bawaan yang BENAR-BENAR dapat dihitung.
//
// # Kenapa bukan langsung DefaultTab
//
// Karena DefaultTab adalah Outstanding, dan Outstanding termasuk yang belum dapat dihitung.
// Membuka layar pada tab yang pasti kosong akan membuat kesan pertama setiap pengguna adalah
// layar tanpa isi — dan kesan itu bertahan meski enam tab lain berisi.
//
// Begitu kolomnya terisi, fungsi ini kembali mengembalikan DefaultTab dengan sendirinya — dan
// sejak 2026-10-07 kalimat itu benar secara harfiah, karena Readiness dibaca dari basis data.
// Sebelumnya ia konstanta, dan "dengan sendirinya" berarti "setelah seseorang menyunting kode".
func (r Readiness) DefaultAvailableTab() Tab {
	if r.TabAvailable(DefaultTab) {
		return DefaultTab
	}
	for _, t := range tabOrder {
		if r.TabAvailable(t) {
			return t
		}
	}
	return DefaultTab
}

// Valid menyatakan apakah tab ini dikenal.
func (t Tab) Valid() bool {
	for _, known := range tabOrder {
		if known == t {
			return true
		}
	}
	return false
}

// SurveyTask adalah satu baris pada layar — satu janji survei yang menunggu.
//
// # Satu baris per JANJI SURVEI, bukan per klaim
//
// `T_SURVEYORLIST` punya `INDEX_SURVEY`, dan `INSERT_SURVEYORLIST.prc` memakai pasangan
// `(CASEID, INDEX_SURVEY)` sebagai kunci idempotensinya. Satu klaim karena itu dapat punya
// beberapa janji survei, dan masing-masing adalah barisnya sendiri di sini — sama seperti di
// Pega, tempat tiap janji adalah satu objek `Work-SurveyClaim`.
type SurveyTask struct {
	// SurveyID adalah `T_SURVEYORLIST.CASEID` — kunci teknis objek SurveyClaim.
	//
	// Bentuknya `ASM-FW-GCNMFW-WORK <pyID>`, terbaca dari
	// `BroswseKlaimByNoSurvey-SQL.xml` yang merangkainya sebagai
	// `'ASM-FW-GCNMFW-WORK ' || {InputData.CARI4}`, dan **dikonfirmasi Work Owner 2026-10-03**
	// bahwa isinya memang kunci utuh berprefix. Nama kelas internal Pega tertanam di dalam
	// kunci data bisnis — utang teknis §4.1.
	//
	// Tidak digambar apa adanya; ia kunci baris, tujuan tautan, dan **asal kolom
	// "Appointment No"** lewat AppointmentNo.
	SurveyID string

	// ClaimID adalah `T_SURVEYORLIST.PNCCASEID` — kunci klaim induknya.
	//
	// Inilah yang menyambung ke `POOLDATA.T_CLAIM_PNC.CLAIMID`. Tidak digambar; dipakai tautan
	// baris untuk membuka klaimnya.
	ClaimID string

	// SurveyIndex adalah `INDEX_SURVEY` — janji keberapa pada klaim yang sama.
	//
	// Tidak digambar sebagai kolom. Ia dibawa karena tanpanya dua baris milik satu klaim
	// tidak dapat dibedakan di layar maupun saat menelusuri keluhan.
	SurveyIndex string

	// ReferenceNumber adalah kolom layar **"Reference No"**, dari `T_SURVEYORLIST.REFNO`.
	//
	// Asalnya TERBUKTI sejak keempat kueri tab diterima 2026-10-03 — keempatnya memakai
	// `a.REFNO_1 AS "UserName"`, dan `.UserName` adalah sel kedua grid.
	//
	// Kolomnya SUDAH ADA tetapi seluruh barisnya masih kosong, sehingga ia tetap dinyatakan
	// belum tersedia. Ia TIDAK dihapus dari tipe ini: menghapusnya akan menghilangkan kolomnya
	// dari layar, dan isian yang belum terbawa harus TERLIHAT.
	//
	// # Catatan untuk pembaca yang menemukan jejak dugaan yang dicabut
	//
	// Pada 2026-10-03 kolom ini sempat dinyatakan "asalnya belum diketahui", karena kedua kueri
	// RUJUKAN (`BrowseLossAdjuster`, `BrowseInternalSurveyor`) mengaliaskan `REFNO_1` menjadi
	// `"Province"` — properti yang tidak digambar. Kesimpulan itu salah sebabnya sama dengan
	// kekeliruan lain di modul ini: **rujukan terdekat bukan sumber**. Begitu kueri tab yang
	// sebenarnya tiba, `REFNO_1` terbukti memang asal kolom ini.
	ReferenceNumber string // "Reference No"  <- REFNO ** ADA, MASIH KOSONG **

	ClaimNumber     string // "Claim No"      <- T_CLAIM_PNC.CLAIMNO
	PolicyNumber    string // "Policy No"     <- T_CLAIM_PNC.NOPOLIS
	InsuredName     string // "Insured Name"  <- T_CLAIM_PNC.QQNAME
	ClassOfBusiness string // "COB"           <- T_CLAIM_PNC.BUSINESSNAME

	// CauseOfLoss — kolom **"Cause Of Loss"**.
	//
	// Belum terkonfirmasi. `BrowseLossAdjuster` tidak mengambilnya sama sekali; yang
	// mengisinya adalah salah satu dari empat Browse rule yang HILANG. Kandidatnya
	// `T_SURVEYORLIST.LOSSTYPE`, dan itulah yang dipakai — lihat kepala berkas .sql.
	CauseOfLoss string

	// Location — kolom **"Location"** <- `T_SURVEYORLIST.LOCATION_SURVEY`.
	//
	// Di Pega ia `RescheduleLocation_1`, yaitu lokasi pada objek SurveyClaim. Keduanya
	// menyatakan hal yang sama — tempat survei dilakukan — dan yang tersedia di tabel
	// penggeraknya adalah `LOCATION_SURVEY`.
	Location string

	TechnicalPIC string // "PIC ASM"            <- T_CLAIM_PNC.PICTEKNIK
	AdjusterPIC  string // "PIC Loss Adjuster"  <- T_SURVEYORLIST.SURVEYOR_NAME

	// DateOfLoss — kolom **"Date of Loss"** <- T_CLAIM_PNC.DATEOFLOSS.
	DateOfLoss time.Time

	// CreatedAt adalah `T_SURVEYORLIST.TGLINPUT` — kapan janji survei ini dicatat.
	//
	// Ia kunci urutan antrean, DAN dasar kolom Aging. Tidak digambar sendiri.
	CreatedAt time.Time

	// ASMStatus — kolom **"Status ASM"** <- `T_CLAIM_PNC.LEADER_MEMBER`.
	//
	// # Ia BUKAN status, melainkan PERAN KOASURANSI
	//
	// Judulnya menyesatkan sejak di Pega. Kolom yang digambarnya, `ASMSTATUS_1`, hanya
	// bernilai `LEADER`, `MEMBER`, atau kosong — dan `SetTempLostAdjuster` menggambarnya
	// lewat `@If(.UserAdmin=="", "LEADER", .UserAdmin)`. Jadi yang dijawabnya adalah: **ASM
	// bertindak sebagai leader atau member pada klaim ini.**
	//
	// # Kenapa dari T_CLAIM_PNC, bukan dari tabel survei
	//
	// `T_SURVEYORLIST` tidak punya padanannya. `T_CLAIM_PNC.LEADER_MEMBER` punya, dan
	// kesetaraannya diukur di produksi 2026-10-03 dengan **nol pertentangan** pada 17.633
	// baris: setiap `ASMSTATUS_1` yang terisi selalu sama dengan `LEADER_MEMBER`.
	//
	// Selisihnya **10 baris (0,06%)** — ber-`LEADER_MEMBER = MEMBER` tetapi `ASMSTATUS_1`
	// kosong, sehingga Pega menggambarnya LEADER. Di sana modul ini justru lebih tepat.
	//
	// # Dua dugaan yang DICABUT, dicatat supaya tidak dihidupkan kembali
	//
	//	"kolom ini berisi STS_SURVEY"      salah — itu status perkembangan adjuster, dan
	//	                                   Pega TIDAK menggambarnya di grid sama sekali
	//	"butuh kolom ASMSTATUS baru"       tidak perlu — LEADER_MEMBER sudah ada
	//
	// Usulan memakai `LEADER_MEMBER` datang dari Work Owner. Ia menghapus satu permintaan
	// kolom yang sudah sempat diajukan — dan pola yang pantas ditiru: **cari kolom yang sudah
	// ada sebelum meminta yang baru.**
	ASMStatus string

	// SurveyorType adalah `SURVEYORTYPE_1` — `"1"` internal, `"2"` loss adjuster.
	//
	// Tidak digambar sebagai kolom. Dibawa supaya jawaban API dapat menjelaskan dirinya
	// sendiri: antrean yang tampak salah isi hampir selalu salah di sini.
	SurveyorType string
}

// AgingDays adalah kolom **"Aging"** — sudah berapa hari janji survei ini menunggu.
//
// # DIHITUNG, bukan dibaca — dan itu berubah dari sebelumnya
//
// Semula kolom ini dibaca dari `T_CLAIMLIST_ADMIN.AGING`. Sesudah pindah ke
// `POOLDATA.T_CLAIM_PNC`, kolom itu tidak ada lagi — `AGING` adalah kolom tabel datar, bukan
// kolom tabel klaim.
//
// Yang dipakai sekarang: umur janji survei sejak `TGLINPUT`. Preseden menghitungnya sudah ada
// dan sudah disetujui Work Owner — `inboxcloseclaim.DurationDays` dan
// `inboxanalystdoctor.DurationDays`.
//
// Ia juga LEBIH TEPAT untuk layar ini daripada `AGING`: yang ditanyakan seorang adjuster
// adalah "sudah berapa lama SURVEI ini menunggu saya", bukan "sudah berapa lama KLAIMNYA
// berjalan". Keduanya berbeda jauh pada klaim yang surveinya baru ditugaskan kemarin.
//
// # Kenapa terhadap TANGGAL, bukan selisih jam dibagi 24
//
// Janji yang masuk pukul 23.00 dan dilihat pukul 01.00 keesokan harinya sudah berumur SATU
// HARI bagi pengguna, meski selisihnya dua jam. Membagi selisih jam akan mengembalikan nol.
//
// # Kenapa zona waktu ikut masuk
//
// Waktu disimpan UTC sedangkan "hari" yang dimaksud pengguna adalah hari WIB. Tanpa konversi,
// janji yang masuk antara pukul 00.00 dan 07.00 WIB dihitung satu hari lebih tua. Keduanya
// diserahkan pemanggil lewat parameter, bukan dibaca dari jam sistem — supaya dapat diuji
// tanpa bergantung mesin (`F-5`).
// pegaWorkKeyPrefix adalah prefix kelas Pega yang tertanam di dalam `T_SURVEYORLIST.CASEID`.
//
// Panjangnya **tepat 19 karakter** — dan angka itu bukan hitungan sendiri melainkan terbaca
// langsung dari `Activity/SetTempLostAdjuster-Act.xml:6197`, yang memotongnya dengan
// `@substring(.CaseID,19,30)`.
const pegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// AppointmentNo adalah kolom layar **"Appointment No"** — nomor berkas survei, `SRV-xxxxx`.
//
// # Kenapa ia diturunkan, bukan dibaca dari kolom tersendiri
//
// Karena memang begitu di Pega. `SetTempLostAdjuster-Act.xml:6197` tidak membaca kolom mana
// pun untuk ini; ia **memotong prefix** dari kunci objek kerja yang sudah di tangan:
//
//	TempDataLostAdjuster.pxResults(<LAST>).UserName  <-  @substring(.CaseID,19,30)
//
// `T_SURVEYORLIST.CASEID` menyimpan kunci utuh yang sama (dikonfirmasi Work Owner 2026-10-03),
// sehingga nomornya **sudah ada di tangan modul ini** — tidak menunggu kolom baru dari siapa
// pun.
//
// # Dugaan yang dicabut, dan kenapa dicatat di sini
//
// Sampai 2026-10-03 kolom ini dinyatakan menunggu `ADJUSTER_PIC` dari Tim Pega, atas dugaan
// bahwa "Appointment No" menggambar `AdjusterPIC_1`. Dugaan itu **salah**, dan yang
// mematahkannya bukan pembacaan ulang melainkan **Work Owner yang melihat layarnya**: isinya
// `SRV-xxx`, bukan nama orang.
//
// Dicatat supaya permintaan `ADJUSTER_PIC` tidak dihidupkan kembali oleh pembaca berikutnya
// yang menemukan jejak dugaan lama.
//
// # Kenapa prefiksnya dipotong dengan TrimPrefix, bukan substring(19)
//
// `@substring(.CaseID,19,30)` memotong **membabi buta** pada posisi 19 — pada nilai yang
// bentuknya tidak terduga ia memotong di tengah. TrimPrefix hanya memotong bila prefiksnya
// memang ada, dan mengembalikan nilainya apa adanya bila tidak. Selisihnya hanya muncul pada
// data yang menyimpang, dan di situlah ia justru dibutuhkan.
func (t SurveyTask) AppointmentNo() string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t.SurveyID), pegaWorkKeyPrefix))
}

func (t SurveyTask) AgingDays(now time.Time, location *time.Location) *int {
	if t.CreatedAt.IsZero() {
		// Tanpa tanggal masuk, umurnya tidak dapat dihitung — dan itu BERBEDA dari nol hari.
		// Penunjuk kosong yang membedakannya; menjadikannya 0 akan menampilkan angka yang
		// terlihat sah dan salah.
		return nil
	}
	if location == nil {
		location = time.UTC
	}

	mulai := t.CreatedAt.In(location)
	kini := now.In(location)

	hariMulai := time.Date(mulai.Year(), mulai.Month(), mulai.Day(), 0, 0, 0, 0, location)
	hariKini := time.Date(kini.Year(), kini.Month(), kini.Day(), 0, 0, 0, 0, location)

	hari := int(hariKini.Sub(hariMulai).Hours() / 24)
	if hari < 0 {
		// Tanggal masuk di masa depan adalah data yang cacat, bukan umur negatif.
		hari = 0
	}
	return &hari
}

// SurveyorIdentity adalah hasil jembatan identitas — siapa pemanggil di mata data survei.
//
// # Kenapa ia tipe tersendiri, bukan sekadar sebuah nama
//
// Karena seorang LEADER melihat pekerjaan anggotanya, bukan hanya miliknya sendiri. Itu
// terbaca dari dua tempat yang saling menguatkan: `MST_LOGIN_SURVEYOR.LOGINLEADER` yang
// dibaca `GetLoginLeaderSurveyor-SQL.xml`, dan kueri Pega yang membandingkan dengan
// `IN {ASIS:TempOperator.CityID}` — sebuah DAFTAR, bukan satu nilai.
//
// Menyederhanakannya menjadi satu nama akan membuat leader melihat antrean kosong sementara
// anggotanya sibuk, dan antrean kosong tidak pernah dilaporkan siapa pun sebagai kerusakan.
type SurveyorIdentity struct {
	// Login adalah `MST_LOGIN_SURVEYOR.LOGIN` — yang diketik saat masuk.
	Login string

	// Name adalah `MST_LOGIN_SURVEYOR.NAMA`.
	//
	// Inilah yang dicocokkan ke `T_SURVEYORLIST.SURVEYOR_NAME` dan ke
	// `DETAIL_KPI_ADJUSTER.adjuster`. Pencocokan berbasis NAMA, bukan kode — itu bentuk
	// datanya di sistem lama, dan bukan pilihan yang diambil di sini.
	Name string

	// IsLeader menyatakan pemanggil membawahi surveyor lain.
	IsLeader bool

	// Scope adalah SELURUH nama surveyor yang boleh dilihat pemanggil — dirinya sendiri,
	// ditambah anggotanya bila ia leader.
	//
	// Ia selalu memuat Name. Scope kosong berarti pemanggil bukan surveyor sama sekali, dan
	// itu ditangani sebagai ErrNotSurveyor — bukan sebagai antrean kosong.
	Scope []string
}

// Caller adalah identitas pemanggil sebagaimana dibutuhkan modul ini.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	Login string
}

// Filter adalah penyaring dan paginasi yang diminta layar.
type Filter struct {
	// Tab adalah keranjang yang sedang dibuka. Kosong berarti DefaultTab.
	Tab Tab

	// Search mencari pada Claim No.
	//
	// Di Pega ia mencari pada DUA kolom — `SetTempLostAdjuster` menyusun penyaring carinya
	// sebagai
	//
	//	"AND (a.pzinskey LIKE '%" + TempLaporan.CaseID + "%' OR A.REFNO_1 LIKE '%…%')"
	//
	// Yang dibawa hanya yang pertama, dengan `pzinskey` digantikan nomor klaim yang terbaca
	// manusia: pengguna mengetik `PNC-1865`, bukan `ASM-FW-GCNMFW-WORK PNC-1865`.
	//
	// Bagian `REFNO_1` GUGUR. Kolom `REFNO` di `T_SURVEYORLIST` sudah ada tetapi seluruhnya
	// kosong, sehingga mencarinya tidak akan pernah menemukan apa pun. Perhatikan bahwa ini
	// soal yang BERBEDA dari kolom layar "Reference No": di sini `REFNO_1` memang benar-benar
	// kolom yang dicari Pega, sedangkan sebagai kolom layar ia terbukti bukan asalnya — lihat
	// SurveyTask.ReferenceNumber. Dua pertanyaan berbeda tentang satu nama kolom.
	//
	// Selisihnya dinyatakan lewat Limitations, bukan disamarkan.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi.
//
// # Kenapa 25, sementara Pega memakai 15
//
// `SetTempLostAdjuster` menyetel `.PageSize` menjadi **15**, lalu menghitung halamannya di
// klipboard: `FirstRow = ((CurrentIndex-1) * PageSize) + 1`, `LastRow = CurrentIndex * PageSize`.
// Angka itu ukuran halaman KLIPBOARD — seluruh baris ditarik lebih dulu, baru dinomori.
//
// Di sini halamannya dipotong basis data sebelum baris meninggalkannya, sehingga angkanya
// menjawab pertanyaan yang berbeda. Yang dipakai 25, sama dengan inbox lain di aplikasi ini,
// supaya ukuran halaman tidak berbeda-beda antarlayar tanpa alasan.
const (
	DefaultLimit = 25
	MaxLimit     = 100

	// PegaPageSize adalah `.PageSize` layar lama. Disimpan supaya selisihnya dapat
	// dinyatakan ke pengguna, bukan supaya ditiru.
	PegaPageSize = 15
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
//
// Tab yang TIDAK dikenal dijatuhkan ke DefaultTab, tidak ditolak. Alasannya: tab datang dari
// URL, dan URL yang salah ketik atau tertinggal versi lama sebaiknya membuka halaman yang
// masuk akal — bukan layar galat. Yang ditolak adalah batas paginasi berlebihan, karena itu
// permintaan yang membebani basis data (`10-API-STRATEGY.md` §4).
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)

	if !f.Tab.Valid() {
		f.Tab = DefaultTab
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
type Page struct {
	Tasks []SurveyTask
	Total int
}

// TabCount adalah jumlah baris pada satu tab.
type TabCount struct {
	Tab   Tab
	Total int
}

// KPIRow adalah satu baris ringkasan KPI.
//
// # Kesembilan angkanya, dan dari mana judulnya
//
// Kolomnya datang dari `POOLDATA.DETAIL_KPI_ADJUSTER`; judul yang dilihat pengguna datang
// dari `Section/InboxSurvey_section-Section.xml`, yang memuat kesembilannya sebagai teks
// kapital. Keduanya berpasangan satu-satu:
//
//	PENJADWALAN SURVEY      surveylap
//	IMMEDIATE ADVICE        immediateadvice
//	PRELIMINARY ADVICE      preliminaryadvice
//	INTERIM REPORT          interim
//	UPDATE PROGRESS         progress
//	TANGGAPAN KOMUNIKASI    komunikasi
//	PROPOSE ADJUSTMENT      propose
//	FINAL REPORT            finalreport
//	NILAI                   nilai
//
// Seluruhnya `round(avg(to_number(...)),2)` di sistem lama — rata-rata, dibulatkan dua
// desimal. `to_number` di sana menyiratkan kolomnya bertipe TEKS di basis data, dan itu
// ditangani di lapisan SQL, bukan di sini.
type KPIRow struct {
	// Group adalah kunci pengelompokan.
	//
	// Isinya nama adjuster pada ringkasan biasa (`group by adjuster`), atau TAHUN pada
	// ringkasan kuartal (`group by to_char(tanggal,'yyyy')`). Satu field, bukan dua, karena
	// keduanya tidak pernah muncul bersamaan — dan dua field yang salah satunya selalu
	// kosong adalah dua field yang akan tertukar.
	Group string

	// Status terisi HANYA pada ShapePerAdjusterStatus — label 'OUTSTANDING' atau 'FINAL'
	// yang dibawa masing-masing blok UNION.
	Status string

	// Quarter terisi HANYA pada ShapePerQuarterYear — label '1'…'4'.
	Quarter string

	// Month dan CaseID terisi HANYA pada ShapeDetail. Keduanya tidak pernah muncul pada
	// bentuk ringkasan mana pun, karena di sana barisnya bukan berkas.
	Month  string
	CaseID string

	SurveyScheduling      float64 // PENJADWALAN SURVEY
	ImmediateAdvice       float64 // IMMEDIATE ADVICE
	PreliminaryAdvice     float64 // PRELIMINARY ADVICE
	InterimReport         float64 // INTERIM REPORT
	ProgressUpdate        float64 // UPDATE PROGRESS
	CommunicationResponse float64 // TANGGAPAN KOMUNIKASI
	ProposeAdjustment     float64 // PROPOSE ADJUSTMENT
	FinalReport           float64 // FINAL REPORT
	Value                 float64 // NILAI
}

// SurveyStatus adalah isian **Status Survey** pada panel KPI.
//
// Di Pega ia `TempAdjComp.ASMFull`, dan pilihannya diisi `Activity/GetFilterKPI-Act.xml`
// menjadi `TipeData.pxResults().DESCRIPTION` — tiga nilai, dalam urutan ini.
//
// Nilainya dipakai apa adanya sebagai penyaring kolom `tipe` di
// `POOLDATA.DETAIL_KPI_ADJUSTER`, kecuali `ALL` yang berarti TANPA penyaring.
type SurveyStatus string

// Ketiga pilihan Status Survey, pada urutan yang sama dengan GetFilterKPI.
const (
	SurveyStatusAll         SurveyStatus = "ALL"
	SurveyStatusOutstanding SurveyStatus = "OUTSTANDING"
	SurveyStatusFinal       SurveyStatus = "FINAL"
)

// SurveyStatuses mengembalikan ketiganya untuk mengisi dropdown.
func SurveyStatuses() []SurveyStatus {
	return []SurveyStatus{SurveyStatusAll, SurveyStatusOutstanding, SurveyStatusFinal}
}

// Valid menyatakan apakah nilai ini salah satu dari ketiganya.
func (s SurveyStatus) Valid() bool {
	switch s {
	case SurveyStatusAll, SurveyStatusOutstanding, SurveyStatusFinal:
		return true
	default:
		return false
	}
}

// ReportType adalah isian **Tipe Report** pada panel KPI.
//
// Di Pega ia `TempAdjComp.AcceptedNo`, diisi `GetFilterKPI` menjadi
// `TipeExport.pxResults().DESCRIPTION`.
type ReportType string

// Kedua pilihan Tipe Report.
const (
	// ReportSummary — ringkasan rata-rata penilaian. Inilah yang dapat dibangun hari ini.
	ReportSummary ReportType = "DATA SUMMARY"

	// ReportDetail — laporan baris per BERKAS, dan angkanya MENTAH: tidak dirata-ratakan
	// sama sekali.
	//
	// Ia laporan yang berbeda, bukan tampilan lain dari angka yang sama.
	// `RDB List/GetDetailKPIAdjusterKuartal-SQL.xml` memilih `caseid`, bulan, tahun, dan
	// nama adjuster berikut kesembilan nilainya apa adanya — tanpa `group by` sama sekali.
	//
	// Rule-nya sempat TIDAK ADA di export dan jalur ini ditolak; ia diterima 2026-10-07,
	// bersama `GetSummaryKPIAdjusterALLKuartal`.
	ReportDetail ReportType = "DATA DETAIL"
)

// ReportTypes mengembalikan keduanya untuk mengisi dropdown.
func ReportTypes() []ReportType { return []ReportType{ReportSummary, ReportDetail} }

// Valid menyatakan apakah nilai ini salah satu dari keduanya.
func (r ReportType) Valid() bool {
	return r == ReportSummary || r == ReportDetail
}

// Quarters mengembalikan keempat kuartal untuk mengisi dropdown.
//
// Di Pega ia `TempAdjComp.Initial`, dan `GetReportKPIAdjuster` mencabangkannya dengan
// `@contains(TempAdjComp.Initial,"1")` … `"4"`.
func Quarters() []string { return []string{"1", "2", "3", "4"} }

// KPIFilter adalah penyaring panel KPI, satu field per kendali di layar Pega.
type KPIFilter struct {
	// Status — kendali "Status Survey". WAJIB; layar lama menandainya bintang merah.
	Status SurveyStatus

	// Report — kendali "Tipe Report". WAJIB.
	Report ReportType

	// Quarter — kendali "Kuartal", `"1"`…`"4"`. Kosong berarti seluruh kuartal.
	Quarter string

	// Year — kendali "Tahun Kuartal". Kosong berarti seluruh tahun.
	Year string
}

// QuarterApplies menyatakan apakah kendali Kuartal dan Tahun Kuartal ditampilkan.
//
// Meniru `pyVisibleWhen` panel KPI apa adanya:
//
//	TempAdjComp.ASMFull=='ALL'||TempAdjComp.ASMFull=='FINAL'
//
// Itu sebabnya tangkapan layar Pega hanya memperlihatkan DUA kendali ketika Status Survey
// masih `--Pilih--`: dua lainnya memang belum muncul.
func (s SurveyStatus) QuarterApplies() bool {
	return s == SurveyStatusAll || s == SurveyStatusFinal
}

// QuarterAll adalah pilihan "seluruh kuartal" pada kendali Kuartal.
//
// Terbaca dari percabangan `Activity/GetReportKPIAdjuster-Act.xml`, yang membedakan
// `@contains(Initial,"1")`…`"4"` dari `Initial==""||Initial=="ALL"`. Isi daftar pilihannya
// sendiri — `TempKuartal.pxResults` — TIDAK ADA di export (`R-16`); yang terbaca hanyalah
// nilai yang benar-benar dicabangkan.
const QuarterAll = "ALL"

// CategoryValue adalah nilai yang dikirim ke penyaring kolom `tipe`.
//
// `ALL` mengembalikan string kosong, dan repo menerjemahkannya menjadi kueri yang BERBEDA —
// bukan menjadi "tanpa penyaring". Lihat Shape: `GetSummaryKPIAdjusterALL` menggabungkan dua
// blok `tipe='OUTSTANDING'` dan `tipe='FINAL'` dengan UNION ALL, masing-masing membawa label
// statusnya sendiri.
//
// Perbedaannya bukan akademis. "Tanpa penyaring" menghasilkan SATU baris per adjuster yang
// merata-ratakan kedua kategori menjadi satu angka; Pega menghasilkan DUA baris per adjuster
// dengan angka masing-masing. Angka yang pertama tidak pernah ada di layar lama.
func (f KPIFilter) CategoryValue() string {
	if f.Status == SurveyStatusAll {
		return ""
	}
	return string(f.Status)
}

// QuarterChosen menyatakan kendali Kuartal benar-benar menunjuk satu kuartal.
//
// Kosong dan "ALL" sama-sama berarti BUKAN satu kuartal, dan keduanya menempuh jalur yang
// berbeda — lihat Shape.
func (f KPIFilter) QuarterChosen() bool {
	return f.Quarter != "" && f.Quarter != QuarterAll
}

// KPIShape adalah BENTUK hasil — apa yang menjadi satu baris.
//
// Ia bukan pilihan tampilan melainkan akibat langsung dari kueri mana yang dijalankan, dan
// setiap kuerinya menghasilkan baris yang berbeda artinya.
type KPIShape string

// Kelima bentuk, masing-masing dari rule Pega-nya sendiri.
const (
	// ShapePerAdjuster — `GetSummaryKPIAdjuster`, `where tipe = {ASMFull}`, `group by adjuster`.
	ShapePerAdjuster KPIShape = "per-adjuster"

	// ShapePerAdjusterStatus — `GetSummaryKPIAdjusterALL`, UNION ALL dua blok yang
	// masing-masing membawa label `'OUTSTANDING'` dan `'FINAL'`. DUA baris per adjuster.
	ShapePerAdjusterStatus KPIShape = "per-adjuster-status"

	// ShapePerYear — `GetSummaryKPIAdjusterKuartal`, `group by to_char(tanggal,'yyyy')`,
	// dengan penyaring bulan satu kuartal.
	ShapePerYear KPIShape = "per-tahun"

	// ShapePerQuarterYear — `GetSummaryKPIAdjusterALLKuartal`, empat blok UNION yang
	// masing-masing memberi label kuartal. EMPAT baris per tahun.
	ShapePerQuarterYear KPIShape = "per-kuartal-tahun"

	// ShapeDetail — `GetDetailKPIAdjusterKuartal`. Satu baris per BERKAS, dan angkanya
	// MENTAH — tidak dirata-ratakan sama sekali.
	ShapeDetail KPIShape = "detail"
)

// Shape memilih bentuk hasil dari kombinasi isian, meniru percabangan
// `Activity/GetReportKPIAdjuster-Act.xml`.
//
//	Tipe Report   Kuartal      Status        -> bentuk
//	DATA DETAIL   apa pun      apa pun          ShapeDetail
//	DATA SUMMARY  1..4         apa pun          ShapePerYear
//	DATA SUMMARY  ALL          apa pun          ShapePerQuarterYear
//	DATA SUMMARY  kosong       ALL              ShapePerAdjusterStatus
//	DATA SUMMARY  kosong       OUTSTANDING/FINAL ShapePerAdjuster
//
// # Satu keanehan yang DIPERTAHANKAN
//
// Ketiga jalur berkuartal mematok `tipe='FINAL'` DI DALAM rule-nya masing-masing — Status
// Survey tidak ikut berpengaruh di sana, bahkan ketika dipilih OUTSTANDING. Itu perilaku
// sistem lama, dan `P-5` menetapkan perilakunya yang dibawa, bukan yang masuk akal.
func (f KPIFilter) Shape() KPIShape {
	switch {
	case f.Report == ReportDetail:
		return ShapeDetail
	case f.QuarterChosen():
		return ShapePerYear
	case f.Quarter == QuarterAll:
		return ShapePerQuarterYear
	case f.Status == SurveyStatusAll:
		return ShapePerAdjusterStatus
	default:
		return ShapePerAdjuster
	}
}

// Check memeriksa kedua isian wajib, dan mengumpulkan SELURUH kesalahannya sekaligus.
//
// Dikumpulkan, bukan berhenti pada yang pertama: layar lama menandai kedua kendali dengan
// bintang merah dan menolak keduanya bersamaan, dan mengembalikan satu per satu akan membuat
// pengguna menekan Cari dua kali untuk mengetahui dua hal.
func (f KPIFilter) Check() error {
	var missing []string

	if !f.Status.Valid() {
		missing = append(missing, "Status Survey")
	}
	if !f.Report.Valid() {
		missing = append(missing, "Tipe Report")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrKPIFilterIncomplete, strings.Join(missing, " dan "))
	}
	return nil
}

// Normalize merapikan isian tanpa menebak yang kosong.
//
// Tidak ada nilai bawaan untuk Status Survey maupun Tipe Report — keduanya WAJIB di layar
// lama, dan memilihkan salah satunya berarti menjalankan laporan yang tidak diminta siapa pun.
func (f KPIFilter) Normalize() KPIFilter {
	f.Status = SurveyStatus(strings.ToUpper(strings.TrimSpace(string(f.Status))))
	f.Report = ReportType(strings.ToUpper(strings.TrimSpace(string(f.Report))))
	f.Quarter = strings.TrimSpace(f.Quarter)
	f.Year = strings.TrimSpace(f.Year)

	// Kuartal dan Tahun Kuartal dibuang ketika kendalinya memang tidak ditampilkan. Tanpa
	// ini, nilai yang tertinggal dari pilihan sebelumnya akan ikut menyaring diam-diam —
	// pengguna melihat Status Survey "OUTSTANDING" dan hasil yang tersaring kuartal yang
	// tidak terlihat di mana pun.
	if !f.Status.QuarterApplies() {
		f.Quarter = ""
		f.Year = ""
	}
	return f
}

// Repo adalah seam ke penyimpanan antrean survei.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya. Diisi
// `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// Ia hanya MEMBACA. Menerima penugasan, menjadwal ulang survei, dan mengunggah laporan
// seluruhnya menempuh `Surveyor_Flow` — sebuah flow yang TIDAK ADA di export (`Flow/` hanya
// memuat empat, dan itu bukan salah satunya). Selama masa paralel penugasan tetap milik Pega
// (`P-1`), sehingga ketiadaan method tulis di sini bukan kekurangan melainkan batas.
type Repo interface {
	// List mengambil satu halaman satu tab.
	//
	// Scope adalah daftar nama surveyor yang boleh dilihat pemanggil. Ia diserahkan terpisah
	// dari Filter dengan sengaja: ia bukan penyaring yang dipilih pengguna melainkan batas
	// kewenangan, dan menaruhnya di dalam Filter akan membuatnya terlihat seperti sesuatu
	// yang boleh dikosongkan.
	List(ctx context.Context, identity SurveyorIdentity, f Filter, ready Readiness) (Page, error)

	// Counts menghitung isi KETUJUH tab sekaligus.
	//
	// Satu method, bukan tujuh: `CountOSLostAdjuster` menghitung seluruh keranjang dalam
	// SATU kueri lewat tujuh `SUM(CASE WHEN …)`. Memecahnya menjadi tujuh perjalanan akan
	// membaca tabel yang sama tujuh kali untuk menggambar satu bilah tab.
	Counts(ctx context.Context, identity SurveyorIdentity, ready Readiness) ([]TabCount, error)

	// KPI mengambil ringkasan KPI adjuster.
	KPI(ctx context.Context, identity SurveyorIdentity, f KPIFilter) ([]KPIRow, error)

	// KPIYears mengisi dropdown "Tahun Kuartal".
	//
	// Ia TERPISAH dari KPI karena dibutuhkan SEBELUM tombol Cari ditekan — dropdown yang
	// kosong sampai pencarian pertama bukan dropdown, melainkan kotak teks yang menyamar.
	//
	// Isinya direkonstruksi dari data; lihat kueri kpi_years untuk alasannya.
	KPIYears(ctx context.Context, identity SurveyorIdentity) ([]string, error)

	// Readiness melaporkan kolom mana yang benar-benar dapat dipakai di portal ini.
	//
	// # Kenapa ia di sini, bukan di seam tersendiri
	//
	// Karena jawabannya milik basis data portal yang SAMA dengan yang dibaca List dan Counts.
	// Seam terpisah akan membuka kemungkinan keduanya menunjuk basis data berbeda — dan
	// akibatnya adalah tab yang dinyatakan siap lalu dihitung terhadap tabel yang belum punya
	// kolomnya (ORA-00904, seluruh layar mati).
	//
	// # Galat TIDAK dikembalikan
	//
	// Pemeriksaan yang gagal mengembalikan Readiness kosong, yaitu "tidak ada yang siap" —
	// perilaku yang sama dengan sebelum kolomnya tiba. Mengembalikan galat akan menjatuhkan
	// layar hanya karena pemeriksaan ketersediaan gagal, padahal enam tab lain tetap dapat
	// dihitung tanpa satu pun kolom itu.
	Readiness(ctx context.Context) Readiness
}

// Directory adalah seam jembatan identitas — login menjadi identitas surveyor.
//
// # Kenapa ia seam TERSENDIRI, bukan bagian dari Repo
//
// Karena ia membaca tabel yang BERBEDA dan menjawab pertanyaan yang berbeda:
// `POOLDATA.MST_LOGIN_SURVEYOR`, master login surveyor yang sudah punya modulnya sendiri
// (Master Login, MENU_ID 37). Menyatukannya akan membuat modul ini seolah memiliki master
// itu, padahal ia hanya membacanya.
//
// Pemisahan ini juga yang membuat keputusan Work Owner 2026-09-28 — identitas BERLAPIS,
// `M_LOGIN_PNC` sebagai master pengguna aplikasi dan `MST_LOGIN_SURVEYOR` sebagai data
// surveyor — dapat berubah di satu tempat saja bila kelak lapisannya disederhanakan.
type Directory interface {
	// ResolveSurveyor menerjemahkan login menjadi identitas surveyor beserta cakupannya.
	//
	// Login yang TIDAK terdaftar sebagai surveyor menghasilkan ErrNotSurveyor — bukan
	// identitas kosong. Perbedaannya menentukan: identitas kosong akan menghasilkan antrean
	// kosong, dan pengguna akan membaca itu sebagai "tidak ada pekerjaan" alih-alih "Anda
	// bukan surveyor".
	ResolveSurveyor(ctx context.Context, login string) (SurveyorIdentity, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas. Klaim milik Asuransi Sinar Mas dan klaim
// milik Simas Insurtech karena itu tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama.
// Jatuh ke koneksi default berarti menampilkan pekerjaan survei satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)

// DirectorySelector memilih Directory milik satu portal entitas.
//
// Alasannya sama dengan RepoSelector, dan di sini taruhannya lebih besar: jembatan identitas
// yang membaca portal yang salah akan memetakan pemanggil ke surveyor bernama sama di entitas
// lain — lalu SELURUH antrean yang muncul sesudahnya milik entitas itu.
type DirectorySelector func(portalAlias string) (Directory, error)

// Clock adalah seam ke waktu (`F-5`).
//
// Waktu TIDAK pernah dibaca langsung dari jam sistem di dalam modul:
// `08-TECHNICAL-STRATEGY.md` §4.4 menetapkan pembacaan waktu terpusat.
type Clock interface {
	Now() time.Time
}
