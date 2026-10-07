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
// Empat nilai layar berasal dari objek kerja `Work-SurveyClaim`, yang di Pega diratakan ke
// `PC_ASM_FW_GCNMFW_WORK` dengan akhiran `_1`. Di `T_SURVEYORLIST` akhiran itu TIDAK dipakai,
// dan penamaan itulah yang berlaku di sini:
//
//	Pega                 T_SURVEYORLIST    keadaan per 2026-09-30
//	ADJUSTERSTATUS_1     STS_SURVEY        ADA dan TERISI — tidak lagi menghalangi
//	ADJUSTERACCEPT_1     ADJUSTERACCEPT    ADA, seluruh barisnya masih KOSONG
//	REFNO_1              REFNO             ADA, seluruh barisnya masih KOSONG
//	PYSTATUSWORK         PYSTATUSWORK      ADA, seluruh barisnya masih KOSONG
//	ADJUSTERPIC_1        —                 belum ditambahkan
//
// **Kolom yang ADA tetapi KOSONG tidak lebih siap daripada kolom yang tidak ada**, dan pada tab
// Outstanding ia justru lebih berbahaya: penyaringnya `ADJUSTERACCEPT IS NULL` bernilai benar
// untuk SELURUH antrean, sehingga tabnya terisi wajar dan isinya salah. Karena itu keduanya
// sama-sama menahan tab — lihat UnavailableReason.
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
	"strings"
	"time"

	"claim-pnc/internal/platform/pagination"
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
func UnavailableReason(t Tab) string {
	switch t {
	case TabOutstanding, TabAll, TabInvoice:
		return "Kolom ADJUSTERACCEPT sudah ada di POOLDATA.T_SURVEYORLIST tetapi seluruh " +
			"barisnya masih kosong. Menghitung tab ini sekarang akan menampilkan seluruh " +
			"antrean sebagai belum dikonfirmasi adjuster."

	case TabClose:
		return "Kolom PYSTATUSWORK sudah ada di POOLDATA.T_SURVEYORLIST tetapi seluruh " +
			"barisnya masih kosong, sehingga berkas yang sudah tutup belum dapat dibedakan."

	default:
		return ""
	}
}

// Available menyatakan apakah tab ini dapat dihitung dari data yang ada hari ini.
func (t Tab) Available() bool { return UnavailableReason(t) == "" }

// DefaultAvailableTab mengembalikan tab bawaan yang BENAR-BENAR dapat dihitung.
//
// # Kenapa bukan langsung DefaultTab
//
// Karena DefaultTab adalah Outstanding, dan Outstanding termasuk yang belum dapat dihitung.
// Membuka layar pada tab yang pasti kosong akan membuat kesan pertama setiap pengguna adalah
// layar tanpa isi — dan kesan itu bertahan meski enam tab lain berisi.
//
// Begitu kolomnya tiba, fungsi ini kembali mengembalikan DefaultTab dengan sendirinya, tanpa
// satu baris pun disunting.
func DefaultAvailableTab() Tab {
	if DefaultTab.Available() {
		return DefaultTab
	}
	for _, t := range tabOrder {
		if t.Available() {
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
	// `'ASM-FW-GCNMFW-WORK ' || {InputData.CARI4}`. Nama kelas internal Pega tertanam di
	// dalam kunci data bisnis — utang teknis §4.1.
	//
	// TIDAK digambar sebagai kolom; ia kunci baris dan tujuan tautan.
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

	// AppointmentNumber dan ReferenceNumber BELUM TERSEDIA, dengan sebab yang BERBEDA.
	//
	//	ReferenceNumber   -> T_SURVEYORLIST.REFNO   kolomnya ADA, isinya masih kosong
	//	AppointmentNumber -> belum ada kolomnya sama sekali
	//
	// Keduanya TIDAK dihapus dari tipe ini. Menghapusnya akan menghilangkan kolomnya dari
	// layar, dan isian yang belum terbawa harus TERLIHAT — bukan tersamar sebagai layar yang
	// sudah setara.
	//
	// Satu catatan untuk AppointmentNumber: judul "Appointment No" di Pega menggambar
	// `AdjusterPIC_1`, dan `ExportDataDetailKlaim-SQL.xml` membuktikan kolom itu berisi **nama
	// adjuster eksternal**, bukan nomor penugasan. Bila itu benar, isinya menggandakan kolom
	// "PIC Loss Adjuster" di sebelahnya dan `T_SURVEYORLIST.SURVEYOR_NAME` sudah membawanya —
	// lihat `docs/permintaan-kolom-t-surveyorlist.md` §4.
	AppointmentNumber string // "Appointment No"  ** BELUM ADA KOLOMNYA **
	ReferenceNumber   string // "Reference No"    <- REFNO ** ADA, MASIH KOSONG **

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

	// ASMStatus — kolom **"Status ASM"** <- `T_SURVEYORLIST.STS_SURVEY` pada langkah terakhir.
	//
	// # Ia TERSEDIA, dan itu berubah pada 2026-09-29
	//
	// `STS_SURVEY` terbukti membawa domain `ADJUSTERSTATUS_1`. Sebaran nilainya di produksi
	// memuat ketiga nilai yang dipakai Pega sebagai penyaring, dengan jumlah yang nyata:
	// `Final Report` 1.466 · `Invoice Fee` 1.069 · `Close Case` 316 — berdampingan dengan
	// seluruh tahapan hidup survei dari `Waiting Claim Document` sampai `Close Case`.
	//
	// Pernyataan sebelumnya bahwa kolom ini hanya berisi `"On Progress"` DICABUT: itu satu
	// dari 22 nilai, dan disimpulkan dari satu-satunya penulis yang kebetulan ada di export.
	// Penulis lainnya berada di luar export (`R-01`, `R-16`).
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
	// manusia: pengguna mengetik `PNC-1865`, bukan `ASM-FW-GCNMFW-WORK PNC-1865`. Bagian
	// `REFNO_1` GUGUR karena kolomnya belum tersedia — lihat SurveyTask.ReferenceNumber. Ia
	// kembali begitu kolomnya tiba, dan sampai saat itu selisihnya dinyatakan lewat
	// Limitations, bukan disamarkan.
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
	f.Limit, f.Offset = pagination.LimitOffset(f.Limit, f.Offset, DefaultLimit, MaxLimit)
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

// KPIKind memilih ringkasan KPI mana yang diminta.
type KPIKind string

// Ketiga bentuk ringkasan, masing-masing dari rule-nya sendiri.
const (
	// KPIOutstanding — `GetSummaryKPIAdjuster-SQL.xml`.
	//
	// Kategorinya diserahkan pemanggil (`where tipe = {TempAdjComp.ASMFull}`), sehingga ia
	// yang paling longgar dari ketiganya.
	KPIOutstanding KPIKind = "outstanding"

	// KPIFinal — `GetSummaryKPIAdjusterALL-SQL.xml`, `where tipe = 'FINAL'`.
	KPIFinal KPIKind = "final"

	// KPIQuarterly — `GetSummaryKPIAdjusterKuartal-SQL.xml`.
	//
	// Sama-sama `tipe = 'FINAL'`, tetapi dikelompokkan `to_char(tanggal,'yyyy')` — per
	// TAHUN, meski namanya menyebut kuartal. Nama rule-nya menyesatkan sejak di Pega;
	// perilakunya yang dibawa, bukan namanya.
	KPIQuarterly KPIKind = "kuartal"
)

// KPITypeFinal adalah nilai `tipe` yang dipatok dua dari tiga ringkasan.
const KPITypeFinal = "FINAL"

// KPIFilter adalah penyaring ringkasan KPI.
type KPIFilter struct {
	Kind KPIKind

	// Category adalah nilai kolom `tipe` untuk KPIOutstanding.
	//
	// Diabaikan oleh KPIFinal dan KPIQuarterly, yang keduanya mematok `'FINAL'`. Kosong
	// berarti seluruh kategori.
	Category string

	// Year menyaring `to_char(tanggal,'yyyy')`. Kosong berarti seluruh tahun.
	Year string
}

// Valid menyatakan apakah jenis ringkasan ini dikenal.
func (k KPIKind) Valid() bool {
	switch k {
	case KPIOutstanding, KPIFinal, KPIQuarterly:
		return true
	default:
		return false
	}
}

// Normalize mengembalikan penyaring KPI dengan nilai yang dijamin masuk akal.
func (f KPIFilter) Normalize() KPIFilter {
	f.Category = strings.TrimSpace(f.Category)
	f.Year = strings.TrimSpace(f.Year)

	if !f.Kind.Valid() {
		f.Kind = KPIOutstanding
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
	List(ctx context.Context, identity SurveyorIdentity, f Filter) (Page, error)

	// Counts menghitung isi KETUJUH tab sekaligus.
	//
	// Satu method, bukan tujuh: `CountOSLostAdjuster` menghitung seluruh keranjang dalam
	// SATU kueri lewat tujuh `SUM(CASE WHEN …)`. Memecahnya menjadi tujuh perjalanan akan
	// membaca tabel yang sama tujuh kali untuk menggambar satu bilah tab.
	Counts(ctx context.Context, identity SurveyorIdentity) ([]TabCount, error)

	// KPI mengambil ringkasan KPI adjuster.
	KPI(ctx context.Context, identity SurveyorIdentity, f KPIFilter) ([]KPIRow, error)
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
