package inboxmanager

import "strings"

// Column adalah satu kolom pada sebuah grid.
//
// Key menyebut ISIAN mana pada baris yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah supaya judul dapat mengikuti layar Pega apa adanya (`D-13`)
// tanpa memaksa nama isian ikut menyesatkan — dan di layar ini alias kolom sistem lama memang
// menyesatkan secara luar biasa. Lihat catatan pada dashboardProduktivitasColumns.
type Column struct {
	// Key adalah nama isian pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom.
	Title string

	// Money menandai kolom yang berisi NILAI UANG, bukan pencacah.
	//
	// Ia dibutuhkan layar untuk memilih perataan dan pemformatan, dan dibutuhkan berkas
	// ekspor supaya nilainya tidak ikut diformat sebagai bilangan bulat.
	Money bool
}

// TabKind membedakan tiga bentuk tab yang isinya berbeda jenis.
//
// Ia ada supaya transport dan layar tidak perlu menebak dari kode tab: yang menentukan
// bentuk jawaban adalah jenisnya, bukan nomornya.
type TabKind string

const (
	// KindDashboard adalah tab berisi grid ringkasan yang barisnya tidak dapat diputuskan.
	KindDashboard TabKind = "dashboard"

	// KindOverview adalah tab "Approval Master" — indeks kesembilan antrean di bawahnya.
	KindOverview TabKind = "ringkasan"

	// KindQueue adalah tab berisi antrean persetujuan.
	KindQueue TabKind = "antrean"
)

// Kode tab.
//
// # Angkanya BUKAN karangan modul ini
//
// Ia diambil apa adanya dari `Section/InboxManager_Sec-Section.xml`, yang menggambar ketiga
// belas bagiannya sebagai kontainer bersyarat `FlagManager.AlasanKlaim==1` sampai `==13`.
// Angka yang sama ditulis `Activity/CountDashbroardManager` ke isian `ALASAN` setiap
// pencacah, yakni nilai yang dipasang saat pencacahnya diklik.
//
// Karena itu penelusuran balik ke export cukup dengan mencocokkan angkanya — tidak perlu
// tabel pemetaan, dan tidak ada kesempatan bagi dua penomoran untuk menyimpang.
const (
	TabOutstanding       = "1"
	TabProduktivitas     = "2"
	TabKlaim             = "3"
	TabApprovalMaster    = "4"
	TabMasterBengkel     = "5"
	TabMasterPanel       = "6"
	TabNomorRangka       = "7"
	TabMasterSparepart   = "8"
	TabKategoriSparepart = "9"
	TabTipeSparepart     = "10"
	TabGroupingSparepart = "11"
	TabPaymentAkseptasi  = "12"
	TabPenolakanKlaim    = "13"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Outstanding, karena itulah kontainer pertama di section — dan karena ia satu-satunya tab
// yang menjawab pertanyaan "apa yang sedang berjalan", yang lebih dulu ditanyakan penyelia
// daripada "apa yang menunggu persetujuan saya".
const DefaultTab = TabOutstanding

// Lini bisnis yang dikenal penyaring dashboard.
//
// Keempatnya dibaca dari precondition `Activity/CountDashbroardManager-Act.xml`:
// `OperatorID.pyPosition=="NONMBU"` (:82436, :84988), `=="BONDING"` (:95201), `=="PA"`
// (:101417), dan `=="TRAVEL"` (:105687).
//
// Ejaannya sama persis dengan nilai yang dipakai `inboxoutstanding` dan `inboxmanageradmin`
// terhadap kolom `M_LOGIN_PNC.LINE_BUSINESS` yang sama.
const (
	LineNonMBU  = "NONMBU"
	LineBonding = "BONDING"
	LinePA      = "PA"
	LineTravel  = "TRAVEL"
)

// DevelopmentOrgUnit adalah unit organisasi yang membuka tab yang dibatasi lini bisnis.
//
// Ia bukan aturan bisnis melainkan pintu pengembang, dan di sistem lama pun begitu:
// `Section/InboxKonfirmasiHE_Section-Section.xml` menyebutnya sebagai alternatif kedua —
// `OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'`.
const DevelopmentOrgUnit = "Development"

// Nama isian pada baris dashboard dan baris antrean.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	// Dashboard Outstanding.
	FieldPIC         = "pic"
	FieldGrupBisnis  = "grup_bisnis"
	FieldJumlahKlaim = "jumlah"

	// FieldKategoriDOL dan FieldReinsurer adalah kedua kolom tetap grid ketiga tab
	// Outstanding. Kolom sisanya adalah TAHUN, dan tahun mana saja ditentukan data —
	// karena itu ia diisi repo, bukan diumumkan di sini.
	FieldKategoriDOL = "kategori_dol"
	FieldReinsurer   = "reinsurer"

	// FilterReinsurer dan FilterCategoryOS adalah kunci kedua penyaring dashboard
	// Outstanding, dipakai bersama oleh repo, transport, dan layar.
	FilterReinsurer  = "reinsurer"
	FilterCategoryOS = "kategori_os"

	// Dashboard Produktivitas dan Dashboard Klaim.
	FieldDimensi         = "dimensi"
	FieldPenyebab        = "penyebab_kerugian"
	FieldTotalPeriodeIni = "total_periode_ini"
	FieldTotalPeriodeLTY = "total_periode_tahun_lalu"
	FieldAksepPeriodeIni = "akseptasi_periode_ini"
	FieldAksepPeriodeLTY = "akseptasi_periode_tahun_lalu"
	FieldTolakPeriodeIni = "ditolak_periode_ini"
	FieldTolakPeriodeLTY = "ditolak_periode_tahun_lalu"
	FieldOSPeriodeIni    = "outstanding_periode_ini"
	FieldOSPeriodeLTY    = "outstanding_periode_tahun_lalu"

	FieldTotalKlaim   = "total_klaim"
	FieldJumlahAksep  = "jumlah_akseptasi"
	FieldJumlahTolak  = "jumlah_ditolak"
	FieldJumlahOS     = "jumlah_outstanding"
	FieldNilaiAksep   = "nilai_akseptasi"
	FieldNilaiTolak   = "nilai_ditolak"
	FieldNilaiOS      = "nilai_outstanding"
	FieldNilaiKlaim   = "nilai_klaim"
	FieldNamaBisnisDK = "nama_bisnis"

	// Antrean persetujuan.
	FieldID             = "id"
	FieldNama           = "nama"
	FieldKeterangan     = "keterangan"
	FieldCabang         = "cabang"
	FieldKota           = "kota"
	FieldNoKlaim        = "no_klaim"
	FieldPengirim       = "pengirim"
	FieldMerk           = "merk"
	FieldModel          = "model"
	FieldTipe           = "tipe"
	FieldRangkaUser     = "no_rangka_user"
	FieldRangkaBengkel  = "no_rangka_bengkel"
	FieldNoAkseptasi    = "no_akseptasi"
	FieldTanggalInput   = "tanggal_input"
	FieldStatusPenolak1 = "status_penolakan_1"
	FieldStatusPenolak2 = "status_penolakan_2"
	FieldPetugas        = "petugas"
	FieldKategori       = "kategori"

	// Kolom yang BARU dibawa setelah judul kolom disamakan dengan Pega (2026-10-07).
	// Seluruhnya ada di tabelnya dan memang digambar layar lama; yang sebelumnya kami
	// tampilkan adalah kolom lain yang dipilih sendiri.
	FieldTelepon      = "telepon"
	FieldNoHP         = "no_hp"
	FieldLoginApl     = "login_aplikasi"
	FieldStsRepair    = "status_repair"
	FieldStsEditQty   = "status_edit_quantity"
	FieldStsPremium   = "status_premium_repair"
	FieldStsPecah     = "status_pecah"
	FieldStsSticker   = "status_sticker"
	FieldStsSisi      = "status_sisi"
	FieldStsRusak     = "status_rusak_parah"
	FieldHarga        = "harga"
	FieldUserUpdate   = "user_update"
	FieldNoSparepart  = "no_sparepart"
	FieldNamaPanel    = "nama_panel"
	FieldSisiPanel    = "sisi_panel"
	FieldNoRangka     = "no_rangka"
	FieldIDKategoriSP = "id_kategori_sparepart"

	// Dua kolom Master Panel yang sempat terlewat (2026-10-08).
	FieldStsAktif   = "status_aktif"
	FieldExclusionC = "exclusion_c"
)

// MessageRangeSameYear adalah pesan penolakan rentang lintas tahun, disalin APA ADANYA dari
// `Activity/DashboardKlaim_act` — `local.message` pada langkah 4.
const MessageRangeSameYear = "Periode Up To hanya untuk periode tahun yang sama"

// Tab adalah satu bagian layar Inbox Manager.
type Tab struct {
	// Code adalah kode tab — angka 1..13 yang sama dengan `FlagManager.AlasanKlaim`.
	Code string

	// Name adalah judul tab yang dibaca pengguna.
	//
	// Diambil apa adanya dari `pyLabel` yang ditulis `Activity/CountDashbroardManager`
	// untuk pencacah tab ini (`D-13`) — bukan dari `<pyTitle>` kontainer, yang pada dua tab
	// terakhir justru SAMA ("Payment Klaim Akseptasi") padahal isinya berbeda.
	Name string

	// Description menjelaskan isi tab dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena tujuh dari tiga belas judulnya berawalan
	// kata yang sama.
	Description string

	// Kind menentukan bentuk isinya.
	Kind TabKind

	// Panels adalah grid pada tab dashboard, berurutan seperti tampilnya. Kosong pada tab
	// lain.
	Panels []Panel

	// Columns adalah kolom antrean pada tab antrean. Kosong pada tab lain.
	Columns []Column

	// Decision menyatakan apa yang boleh diputuskan pada tab antrean.
	Decision DecisionRule

	// LineBusiness membatasi tab ini pada satu lini bisnis. Kosong berarti tidak dibatasi.
	//
	// Hanya SATU tab yang dibatasi, dan batas itu dibaca dari export — bukan ditambahkan
	// sendiri. Lihat tab Approval Nomor Rangka Beda.
	LineBusiness string

	// SectionTitle adalah judul SUB-TAB di dalam tab ini, bila Pega menggambarnya.
	//
	// Hanya satu tab punya: kontainer ke-12 di Pega bukan sebuah grid melainkan sebuah bilah
	// sub-tab (`Section/Sec_PaymentAkseptasiKlaimCase1`), dan sub-tab yang TAMPIL hanya satu
	// — "Approval Payment Akseptasi".
	//
	// Sub-tab keduanya, "Approval Progress Klaim", bersyarat `pyContainerVisibleWhen = 1==2`
	// (`:67064`) — syarat yang tidak pernah benar. Ia mati di sistem lama dan tidak dibawa;
	// lihat TestSubTabApprovalProgressKlaimTidakDibawa.
	SectionTitle string

	// InParentTabStrip menyatakan tab ini muncul di BILAH TAB induknya.
	//
	// # Kenapa ini BUKAN hal yang sama dengan Parent()
	//
	// Pega menyimpan dua fakta yang berbeda, dan keduanya benar:
	//
	//   pohon pencacah   `CountDashbroardManager` menulis SEMBILAN pencacah sebagai anak
	//                    baris "Approval Master" — termasuk Penolakan Klaim
	//   bilah tab        `Section/InboxManager_Section2` menyertakan DELAPAN section, dan
	//                    Penolakan Klaim BUKAN salah satunya
	//
	// Penolakan Klaim karena itu terhitung di bawah Approval Master tetapi tidak dapat
	// dibuka dari bilah tabnya — ia dibuka dengan mengeklik pencacahnya. Menyamakan kedua
	// fakta itu akan menambah satu tab yang Pega tidak punya, atau menghilangkan satu
	// pencacah yang Pega punya.
	InParentTabStrip bool

	// PeriodApplyLabel adalah label tombol yang MENERAPKAN penyaring periode.
	//
	// Di Pega periode TIDAK berlaku saat isiannya diubah — ia berlaku saat tombolnya
	// ditekan, dan labelnya berbeda tiap tab: `pyButtonLabel Cari` pada
	// `DisplayInboxProduktivitas_Sect`, `pyButtonLabel Lihat Data` pada `DashboardKlaim_sec`.
	//
	// Perbedaan itu bukan kerapian. Isian tanggal diisi sepotong demi sepotong, dan tanpa
	// tombol setiap potongan menjadi permintaan tersendiri — termasuk keadaan setengah jadi
	// seperti tahun "0020" saat pengguna baru mengetik dua angka.
	PeriodApplyLabel string

	// RangeSameYearOnly menolak rentang yang kedua tanggalnya berbeda tahun.
	//
	// Hanya tab Klaim yang punya batas ini, dan itu BUKAN penyederhanaan:
	// `Activity/DashboardKlaim_act` langkah 4 memotong tahun dari kedua tanggal
	// (`@substring(…,6,10)`) lalu langkah 5 melompat ke blok galat bila keduanya berbeda.
	// `PNCGetDashboardProduktivitasInbox_Act` tidak punya langkah serupa — tidak ada
	// `Page-Set-Messages`, tidak ada blok galat, dan tidak ada pesan apa pun.
	RangeSameYearOnly bool

	// PageSize adalah ukuran halaman antrean ini, disalin dari `pyPageSize` pada grid
	// section-nya. Nol berarti memakai DefaultPageSize.
	//
	// Angkanya BERBEDA-BEDA per tab di Pega — 20, 50, dan 15 — dan itu bukan kebetulan
	// susunan kolomnya: antrean berkolom banyak diberi halaman lebih kecil.
	PageSize int

	// HasDetailExport menyatakan tab ini punya blok "Export Data Detail Klaim" — ekspor
	// berentang tanggal yang berdiri sendiri, terpisah dari ekspor antrean di kepala layar.
	//
	// Hanya tab Outstanding yang punya, dan itu bukan penyederhanaan: hanya kontainernya
	// yang memuat tombol `pyButtonLabel Export To Excel`.
	HasDetailExport bool

	// HasPeriodFilter menyatakan tab ini punya penyaring periode.
	//
	// Ia BUKAN seragam: Dashboard Outstanding tidak punya penyaring tanggal sama sekali —
	// ketiga kuerinya tidak menyaring tanggal, karena yang dihitungnya klaim yang SEDANG
	// berjalan, bukan klaim pada suatu periode. Menambahkan penyaring periode di sana akan
	// menjadi kemampuan baru yang mengubah arti angkanya.
	HasPeriodFilter bool
}

// dashboardOutstandingPanels adalah kedua grid tab Outstanding.
//
// Keduanya dibaca dari `Section/PNCDashboardOS-Section.xml`, yang mengikat TEPAT DUA page
// list: `GetPICDashboardOS.pxResults` dan `GetBisnisDashboardOS.pxResults`.
//
// Activity pemasoknya `PNCGetDashboardOSInbox_Act` menjalankan LIMA kueri — dua di atas
// ditambah `GetYearDashboardOS`, `GetProgressAllYearDashboarOS`, dan `BrowseMstProgress1`.
// Ketiga sisanya mengisi penyaring dan daftar pilihan, bukan grid. Membawanya sebagai grid
// akan menggambar tiga tabel yang tidak pernah ada di layar lama.
// Judul kolomnya `PIC`, `OS`, `COB`, `OS` — disalin apa adanya dari section (`D-13`),
// tersimpan di sana sebagai `<pyValue>` pada offset 191893, 197557, 257353, dan 260459.
// Judul yang lebih panjang seperti "Nama PIC" dan "Jumlah Klaim" sempat dipakai di sini, dan
// itu karangan: tidak satu pun ada di export.
//
// Kedua grid pertama juga TIDAK berjudul di Pega — yang membedakannya adalah kepala
// kolomnya. Judul per grid karena itu dikosongkan, dan layar tidak menggambar apa pun di
// tempatnya.
var dashboardOutstandingPanels = []Panel{
	{
		Key: "pic",
		Columns: []Column{
			{Key: FieldPIC, Title: "PIC"},
			{Key: FieldJumlahKlaim, Title: "OS"},
		},
	},
	{
		Key: "grup_bisnis",
		Columns: []Column{
			{Key: FieldGrupBisnis, Title: "COB"},
			{Key: FieldJumlahKlaim, Title: "OS"},
		},
	},

	// Grid ketiga — "Kategori/DOL × Reinsurer × tahun".
	//
	// # Kenapa ia sempat tidak ada di sini, dan apa yang membuatnya terlewat
	//
	// Catatan sebelumnya menyatakan `GetYearDashboardOS` dan `GetProgressAllYearDashboarOS`
	// "mengisi penyaring dan daftar pilihan, bukan grid". Itu SALAH, dan Work Owner
	// menunjukkannya dari layar Pega yang berjalan.
	//
	// Sebabnya: `Section/PNCDashboardOS` hanya mengikat DUA page list, dan grid ketiga
	// tidak diikat page list sama sekali — ia `Rule-HTML-Property` bernama
	// `PNCSummaryDashboardOS`, dipasang sebagai `pyFormat` pada satu sel baca-saja
	// (`:355876`). Kontrol HTML tidak terlihat oleh pencarian page list.
	//
	// Rule itu sendiri TIDAK ADA di export meski direktori `HTML/` terkirim dengan tiga
	// puluh rule lain — gap `R-16`. Bentuk datanya tetap ditentukan penuh oleh kedua kueri
	// pemasoknya, sehingga gridnya dapat dibangun tanpa rule itu; yang tidak dapat
	// dipulihkan hanyalah detail tampilannya.
	//
	// Kolom tahun TIDAK diumumkan di sini karena tahun mana saja yang tampil ditentukan
	// data — di Pega pun begitu: activity-nya merangkai satu `SUM(CASE WHEN … )` per tahun
	// yang dikembalikan `GetYearDashboardOS`, lalu menyisipkannya ke daftar SELECT.
	{
		Key: "kategori_os",
		Columns: []Column{
			{Key: FieldKategoriDOL, Title: "Kategori/DOL"},
			{Key: FieldReinsurer, Title: "Reinsurer"},
		},
	},
}

// dashboardProduktivitasColumns adalah kesembilan kolom kedua grid tab Produktivitas Klaim.
//
// # Alias kolom sistem lama tidak berarti apa-apa, dan itu bukan kelalaian pembacaan
//
// Kueri pemasoknya mengembalikan delapan pencacah beralias `BRANCHNAME`, `BUSINESSCODE`,
// `BUSINESSNAME`, `CLIENTID`, `EDMNO`, `FLAGEDMBATAL`, `FOLLOWEDPOLICY`, dan `IDPEGA` —
// tidak satu pun menyebut isinya. Alias itu dipaksa agar cocok dengan properti klipboard Pega
// yang sudah ada, persis utang teknis yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat.
//
// Arti sesungguhnya dibaca dari `Activity/PNCGetDashboardProduktivitasInbox_Act`, yang
// menyusun kedua predikatnya:
//
//	TempLaporan.LokasiSurveyor -> periode yang DIPILIH
//	TempLaporan.NamaSurveyor   -> periode yang sama SATU TAHUN SEBELUMNYA
//	                              (`- INTERVAL '1' YEAR`)
//
// Jadi kedelapan pencacah itu adalah empat keranjang × dua periode, dan dashboard ini
// sesungguhnya membandingkan periode berjalan dengan periode yang sama tahun lalu. Tidak ada
// satu pun judul di layar lama yang menyatakannya; itu hanya terbaca dari predikatnya.
// Kedelapan judul pencacah disalin APA ADANYA dari section, yang menyimpannya sebagai
// `<pyValue><b>…</b></pyValue>` pada offset 220362…245764 dan 356549…383444.
//
// # Satu judul lama bukan sekadar beda kata, melainkan beda ARTI
//
// Pasangan kedua sempat diberi judul "Akseptasi — Periode Ini" dan "Akseptasi — Tahun Lalu".
// Section-nya menyebutnya **Close**, bukan akseptasi. Keduanya tahapan yang berbeda, dan
// judul karangan itu membuat penyelia membaca angka penutupan klaim sebagai angka akseptasi.
var dashboardProduktivitasCounterColumns = []Column{
	{Key: FieldTotalPeriodeIni, Title: "Total register tahun sama"},
	{Key: FieldTotalPeriodeLTY, Title: "Total register tahun sebelumnya"},
	{Key: FieldAksepPeriodeIni, Title: "Total Close tahun sama"},
	{Key: FieldAksepPeriodeLTY, Title: "Total close tahun sebelumnya"},
	{Key: FieldTolakPeriodeIni, Title: "Total reject tahun sama"},
	{Key: FieldTolakPeriodeLTY, Title: "Total reject tahun sebelumnya"},
	{Key: FieldOSPeriodeIni, Title: "Total OS tahun sama"},
	{Key: FieldOSPeriodeLTY, Title: "Total OS tahun sebelumnya"},
}

// Kolom dimensinya BERBEDA antara kedua grid — `Nama PIC` dan `COB` — meski kedelapan
// pencacahnya sama. Judul gabungan "Grup Bisnis / PIC" yang sempat dipakai tidak ada di
// export sama sekali; ia karangan yang menutupi perbedaan itu.
var dashboardProduktivitasPICColumns = append(
	[]Column{{Key: FieldDimensi, Title: "Nama PIC"}},
	dashboardProduktivitasCounterColumns...,
)

var dashboardProduktivitasCOBColumns = append(
	[]Column{{Key: FieldDimensi, Title: "COB"}},
	dashboardProduktivitasCounterColumns...,
)

// dashboardKlaimColumns adalah kolom grid "Total Klaim Bisnis" pada tab Klaim.
//
// Empat pencacah dan tiga nilai uang, dibaca dari `RDB List/BrowseCaseClaim-SQL.xml`.
// Aliasnya sama menyesatkannya dengan tab Produktivitas — `BUSINESSNAME` untuk jumlah klaim,
// `NOPOLIS` untuk sebuah jumlah uang — dan diabaikan dengan cara yang sama.
// Judulnya disalin apa adanya dari section (offset 144686…174780), dan URUTANNYA ikut:
// jumlah dan nilai berselang-seling per tahapan, bukan seluruh jumlah lalu seluruh nilai.
//
// Urutan itu bukan kerapian — ia yang membuat sepasang angka tahapan yang sama terbaca
// berdampingan. Mengelompokkannya memaksa mata melompat delapan kolom untuk memasangkannya.
//
// `NILAI Klaim (Rp)` DIBAWA, dan rumusnya direplikasi apa adanya (`P-5`). Ia sempat tidak
// dibawa dengan alasan "tidak diketahui artinya"; Work Owner meminta kolomnya disamakan
// dengan Pega (2026-10-07), dan alasan itu tidak cukup untuk menghilangkan satu kolom.
//
// Yang perlu diketahui saat membacanya — rumus lamanya bercabang dua, dan cabang pertamanya
// menyederhana menjadi dua kali nilai outstanding:
//
//	stsklaim = '1'              -> (TTLAKSEP+TTLOS)+(TTLOS-TTLAKSEP)  ==  2 x TTLOS
//	stsklaim bukan '1','2','3'  -> TTLAKSEP+TTLOS
//
// Cabang pertama itu patut dicurigai sebagai cacat di Pega, tetapi memperbaikinya di sini
// akan mengubah angka tanpa dasar keputusan. Ia direplikasi, dan kecurigaannya dicatat.
var dashboardKlaimColumns = []Column{
	{Key: FieldNamaBisnisDK, Title: "COB"},
	{Key: FieldTotalKlaim, Title: "Total Klaim"},
	{Key: FieldNilaiKlaim, Title: "NILAI Klaim (Rp)", Money: true},
	{Key: FieldJumlahAksep, Title: "CASE Akseptasi"},
	{Key: FieldNilaiAksep, Title: "NILAI Akseptasi (Rp)", Money: true},
	{Key: FieldJumlahOS, Title: "CASE OS"},
	{Key: FieldNilaiOS, Title: "NILAI OS (Rp)", Money: true},
	{Key: FieldJumlahTolak, Title: "CASE Reject"},
	{Key: FieldNilaiTolak, Title: "NILAI Reject (Rp)", Money: true},
}

// dashboardKlaimCauseColumns adalah kolom grid "Total Klaim CauseOfLoss" pada tab Klaim.
//
// Ia grid KEDUA yang diikat `Section/DashboardKlaim_sec` (`TempBrowseCase2.pxResults`),
// dipasok `BrowseCaseClaimPerCauseOfLoss` — kueri yang sama dengan grid pertama ditambah satu
// dimensi `COL_DESC`, yakni penyebab kerugian.
var dashboardKlaimCauseColumns = append(
	[]Column{
		{Key: FieldNamaBisnisDK, Title: "COB"},
		{Key: FieldPenyebab, Title: "Cause Of Loss"},
	},
	dashboardKlaimColumns[1:]...,
)

// tabs adalah ketiga belas tab, berurutan seperti kode kontainernya di section.
//
// Kode 1..13 tidak berlubang dan tidak diloncati — kecuali satu SUB-tab di dalam tab 12, yang
// memang mati di Pega. Lihat catatan pada tab Payment Klaim Akseptasi.
var tabs = []Tab{
	{
		Code:        TabOutstanding,
		Name:        "Outstanding",
		Description: "Klaim yang sedang berjalan dan sudah punya PIC Teknik, dihitung per petugas dan per grup bisnis.",
		Kind:        KindDashboard,
		Panels:      dashboardOutstandingPanels,

		// TIDAK punya penyaring periode, dan itu diperiksa uji. Ketiga kueri Pega yang
		// memasoknya tidak menyaring tanggal sama sekali.
		HasPeriodFilter: false,

		// Blok "Export Data Detail Klaim" — isian Dari dan Sampai beserta tombolnya. Ia
		// ekspor TERSENDIRI, berbeda dari ekspor antrean di kepala layar: di Pega ia
		// tombol `pyButtonLabel Export To Excel` yang memanggil activity
		// `ExportDataDetailKlaim`, dan hanya tab ini yang punya.
		HasDetailExport: true,
	},
	{
		Code:             TabProduktivitas,
		PeriodApplyLabel: "Cari",
		Name:             "Produktivitas Klaim",
		Description: "Perbandingan jumlah klaim periode berjalan dengan periode yang sama " +
			"tahun lalu, per grup bisnis dan per PIC.",
		Kind: KindDashboard,
		// Urutannya PIC LEBIH DULU, mengikuti urutan page list di section:
		// `GetPICDashboardProduktivitas` pada offset 200374, `GetBusinessDashboardProduktivitas`
		// pada 350800. Urutan terbalik yang sempat dipakai membuat grid yang di Pega berada di
		// kiri tergambar di bawah.
		//
		// Judul per grid dikosongkan — section tidak memberi judul pada keduanya; yang
		// membedakannya kepala kolom `Nama PIC` versus `COB`.
		Panels: []Panel{
			{Key: "pic", Columns: dashboardProduktivitasPICColumns},
			{Key: "grup_bisnis", Columns: dashboardProduktivitasCOBColumns},
		},
		HasPeriodFilter: true,
	},
	{
		Code:        TabKlaim,
		Name:        "Klaim",
		Description: "Jumlah dan nilai klaim per grup bisnis, dan rinciannya per penyebab kerugian.",
		Kind:        KindDashboard,

		// Hanya tab INI yang menolak rentang lintas tahun — lihat Tab.RangeSameYearOnly.
		RangeSameYearOnly: true,
		PeriodApplyLabel:  "Lihat Data",
		// Kedua grid tab ini PUNYA judul di Pega — `pyTitle` pada
		// `Section/DashboardKlaim_sec` offset 128772 dan 267644 — berbeda dari tab
		// Outstanding dan Produktivitas yang tidak punya. Judulnya sempat ikut dikosongkan
		// saat kedua tab itu dirapikan; itu keliru.
		Panels: []Panel{
			{Key: "bisnis", Title: "Total Klaim Bisnis", Columns: dashboardKlaimColumns},
			{Key: "penyebab", Title: "Total Klaim CauseOfLoss", Columns: dashboardKlaimCauseColumns},
		},
		HasPeriodFilter: true,
	},
	{
		Code: TabApprovalMaster,
		Name: "Approval Master",
		Description: "Ringkasan seluruh antrean yang menunggu persetujuan Anda, beserta " +
			"jumlahnya masing-masing.",
		Kind: KindOverview,
	},

	{
		Code:             TabMasterBengkel,
		InParentTabStrip: true,
		PageSize:         20,
		Name:             "Master Bengkel",
		Description:      "Pengajuan data bengkel yang menunggu persetujuan.",
		Kind:             KindQueue,
		// Judul DAN kolomnya disalin dari `Section/ApprovalMasterBengkelHE`, yang
		// menyimpannya sebagai teks `pyValue` sesudah page list-nya. Nama kolom basis
		// datanya dibaca dari `Report Definition/BrowseBengkelHE_RD`, yang dipanggil
		// `Activity/GetDataMaster` lewat `pxRetrieveReportData`.
		//
		// "Nama Cabang" dan "Nama Kota" yang sempat kami tampilkan TIDAK ada di layar lama;
		// keduanya kolom lain pada tabel yang sama yang kami pilih sendiri.
		Columns: []Column{
			{Key: FieldID, Title: "ID Bengkel"},
			{Key: FieldNama, Title: "Nama Bengkel"},
			{Key: FieldKeterangan, Title: "Alamat Bengkel"},
			{Key: FieldTelepon, Title: "Telp Bengkel"},
			{Key: FieldNoHP, Title: "No HP Bengkel"},
			{Key: FieldLoginApl, Title: "Login Aplikasi"},
		},
		// Satu dari tiga antrean yang punya `Select All` / `Deselect All` di atas gridnya,
		// dan tombol keputusannya memanggil `SetApprovalAllMaster` yang mengulang baris.
		Decision: DecisionRule{
			Decidable:              true,
			Bulk:                   true,
			ApproveLabel:           "APPROVE",
			RejectLabel:            "REJECT",
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Alasan Status Bengkel",
		},
	},
	{
		Code:             TabMasterPanel,
		InParentTabStrip: true,
		PageSize:         20,
		Name:             "Master Panel",
		Description:      "Pengajuan data panel kendaraan yang menunggu persetujuan.",
		Kind:             KindQueue,
		// SEBELAS kolom. Judul dan urutannya dibaca dari `<pyValue>` pada
		// `Section/ApprovalMasterPanelHE` — offset 145578 sampai 190972 — dan propertinya
		// dari blok sesudahnya, offset 210365 sampai 276060:
		//
		//	ID → .ID_PANEL · NAMA PANEL → .NAME · STATUS REPAIR → .STS_REPAIR
		//	STATUS EDIT QUANTITY → .STS_EDIT_QTY · STATUS PREMIUM REPAIR → .STS_PREMIUM_REPAIR
		//	STATUS PECAH → .STS_PECAH · STATUS STICKER → .STS_STICKER · STATUS SISI → .STS_SISI
		//	STATUS RUSAK PARAH → .STS_RUSAK_PARAH · STATUS AKTIF → .STS_AKTIF
		//	Exclusion C → .EXCLUSION_C
		//
		// Dua yang terakhir sempat terlewat. Keduanya ada di tabelnya dan terisi — pemeriksaan
		// `POOLDATA.PANEL_HE` pada 2026-10-08: `STS_AKTIF` berisi `1` (105 baris), `0` (16),
		// NULL (3); `EXCLUSION_C` berisi `0` (100), NULL (20), `1` (4).
		//
		// `Exclusion C` ber-`pyReadOnly=false` di Pega — ia dapat DIUBAH di dalam grid. Di
		// sini ia digambar baca-saja seperti kolom lain; penyuntingan di dalam grid adalah
		// bentuk layar yang belum dibangun modul ini.
		Columns: []Column{
			{Key: FieldID, Title: "ID"},
			{Key: FieldNama, Title: "NAMA PANEL"},
			{Key: FieldStsRepair, Title: "STATUS REPAIR"},
			{Key: FieldStsEditQty, Title: "STATUS EDIT QUANTITY"},
			{Key: FieldStsPremium, Title: "STATUS PREMIUM REPAIR"},
			{Key: FieldStsPecah, Title: "STATUS PECAH"},
			{Key: FieldStsSticker, Title: "STATUS STICKER"},
			{Key: FieldStsSisi, Title: "STATUS SISI"},
			{Key: FieldStsRusak, Title: "STATUS RUSAK PARAH"},
			{Key: FieldStsAktif, Title: "STATUS AKTIF"},
			{Key: FieldExclusionC, Title: "Exclusion C"},
		},
		Decision: DecisionRule{
			Decidable:              true,
			Bulk:                   true,
			ApproveLabel:           "APPROVE",
			RejectLabel:            "REJECT",
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Alasan Tolak",
		},
	},
	{
		Code:             TabNomorRangka,
		InParentTabStrip: true,
		PageSize:         50,
		Name:             "Approval Nomor Rangka Beda",
		Description: "Klaim yang nomor rangka dari bengkel berbeda dengan nomor rangka yang " +
			"diinput pengguna.",
		Kind: KindQueue,
		Columns: []Column{
			{Key: FieldNoKlaim, Title: "No Klaim"},
			{Key: FieldPengirim, Title: "Pemilik Kendaraan"},
			{Key: FieldModel, Title: "Model Kendaraan"},
			{Key: FieldMerk, Title: "Merk Kendaraan"},
			{Key: FieldTipe, Title: "Tipe Kendaraan"},
			{Key: FieldRangkaUser, Title: "No Rangka (Inputan User)"},
			{Key: FieldRangkaBengkel, Title: "No Rangka (Dari Bengkel)"},
		},

		// Tabelnya TIDAK punya kolom alasan — delapan kolomnya habis untuk identitas
		// kendaraan dan status. Isian alasan karena itu tidak digambar sama sekali, bukan
		// digambar lalu isinya dibuang.
		//
		// Satu-satunya antrean yang label tombolnya berbahasa Indonesia di Pega, dan
		// keduanya memanggil activity yang berbeda — `UpdateStatusAksepRangka_HE` dan
		// `UpdateStatusRejectRangka_HE`, satu baris per panggilan.
		Decision: DecisionRule{
			Decidable:    true,
			ApproveLabel: "Setuju",
			RejectLabel:  "Tidak Setuju",
		},

		// SATU-SATUNYA tab yang dibatasi lini bisnis, dan batas itu dibaca dari export:
		// `Section/InboxKonfirmasiHE_Section-Section.xml` memasang
		// `OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'`.
		LineBusiness: LineNonMBU,
	},
	{
		Code:             TabMasterSparepart,
		InParentTabStrip: true,
		PageSize:         20,
		Name:             "Master Sparepart",
		Description:      "Pengajuan data sparepart yang menunggu persetujuan.",
		Kind:             KindQueue,
		// Nama kolomnya dari `Report Definition/BrowseSparepartHE_RD`. "Kategori Sparepart"
		// dan "Nomor Sparepart" yang sempat kami tampilkan tidak ada di layar lama.
		Columns: []Column{
			{Key: FieldID, Title: "ID Sparepart"},
			{Key: FieldNama, Title: "Nama Sparepart"},
			{Key: FieldHarga, Title: "Harga (Rp)"},
			{Key: FieldUserUpdate, Title: "User Update"},
		},
		Decision: DecisionRule{
			Decidable:    true,
			Bulk:         true,
			ApproveLabel: "APPROVE",
			RejectLabel:  "REJECT",
		},
	},
	{
		Code:             TabKategoriSparepart,
		InParentTabStrip: true,
		PageSize:         50,
		Name:             "Master Kategori Sparepart",
		Description:      "Pengajuan kategori sparepart yang menunggu persetujuan.",
		Kind:             KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Sparepart Kategori"},
			{Key: FieldNama, Title: "Sparepart Kategori"},
		},
		// TANPA jalur massal: sectionnya tidak menggambar `Select All`, dan tombolnya
		// memanggil `UpdateKategoriSparepart_act` untuk satu baris.
		Decision: DecisionRule{
			Decidable:    true,
			ApproveLabel: "APPROVE",
			RejectLabel:  "REJECT",
		},
	},
	{
		Code:             TabTipeSparepart,
		InParentTabStrip: true,
		PageSize:         50,
		Name:             "Master Tipe Sparepart",
		Description:      "Pengajuan tipe sparepart yang menunggu persetujuan.",
		Kind:             KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Tipe Sparepart"},
			{Key: FieldNama, Title: "Nama Tipe Sparepart"},
			{Key: FieldKategori, Title: "ID Kategori Sparepart"},
		},
		Decision: DecisionRule{
			Decidable:    true,
			ApproveLabel: "APPROVE",
			RejectLabel:  "REJECT",
		},
	},
	{
		Code:             TabGroupingSparepart,
		InParentTabStrip: true,
		PageSize:         15,
		Name:             "Master Grouping Sparepart",
		Description:      "Pengajuan pengelompokan sparepart menurut nomor rangka.",
		Kind:             KindQueue,
		// Enam kolom, dan urutannya dari `RDB List/GetDataMasterGrouping-SQL.xml` — kueri
		// Pega untuk grid yang sama.
		Columns: []Column{
			{Key: FieldID, Title: "ID"},
			{Key: FieldNoSparepart, Title: "No Sparepart"},
			{Key: FieldNama, Title: "Nama Sparepart"},
			{Key: FieldNamaPanel, Title: "Nama Panel"},
			{Key: FieldSisiPanel, Title: "Sisi Panel"},
			{Key: FieldNoRangka, Title: "No Rangka"},
		},
		// Huruf besarnya BERBEDA dari lima antrean master lain — "Approve" dan "Reject",
		// bukan "APPROVE" dan "REJECT". Itu apa adanya dari sectionnya, dan tidak
		// diseragamkan di sini (`D-13`).
		Decision: DecisionRule{
			Decidable:    true,
			ApproveLabel: "Approve",
			RejectLabel:  "Reject",
		},
	},
	{
		Code:             TabPaymentAkseptasi,
		InParentTabStrip: true,
		SectionTitle:     "Approval Payment Akseptasi",
		PageSize:         15,
		Name:             "Payment Klaim Akseptasi",
		Description: "Nomor akseptasi yang menunggu persetujuan atasan sebelum pembayaran " +
			"diteruskan ke kasir.",
		Kind: KindQueue,
		// Tanggal Input lebih dulu, mengikuti urutan kolom di `Akseptasi_PaymentLeader1`.
		Columns: []Column{
			{Key: FieldTanggalInput, Title: "Tanggal Input"},
			{Key: FieldNoKlaim, Title: "No Klaim"},
			{Key: FieldNoAkseptasi, Title: "No Akseptasi"},
			{Key: FieldPIC, Title: "PIC Klaim"},
		},

		// # Kenapa jalur SETUJU ditahan, dan jalur TOLAK tidak
		//
		// Keduanya menulis tabel yang sama lewat `SaveApproveAkseptasiPaymentLeader_Sql`.
		// Yang membedakannya ada di pemanggilnya,
		// `Activity/SaveApprovalAkseptasiPaymentLeader_Act-Act.xml:89539`: pada
		// precondition `TEMPADJUSTMENT.AcceptanceStatus=="1"` — yakni HANYA pada
		// persetujuan — ia memanggil `TransferToKasir_act_Leader`.
		//
		// Activity itu bukan langkah kecil. Ia memanggil `InsertDataAkseptasiToLeader`,
		// `InsertLogKasir_act`, dan `TransferCashierDataASM_act`; yang terakhir menembak
		// Connect REST `SendDataPaidASMtoCashier_2` ke sistem Kasir. Dua rule SQL yang
		// dibutuhkannya masih hilang dari export — `UpdateChasierIDTablePembayaran` dan
		// `GetDataMSTDetailSales` — dan integrasi keluarnya sendiri adalah lingkup modul
		// `S-4`, yang belum dibangun.
		//
		// Menuliskan `STSAPP='1'` tanpa langkah itu menandai pembayaran sebagai DISETUJUI
		// padahal ia tidak pernah sampai ke kasir, dan tidak ada apa pun di layar yang
		// menandakannya. Itu selisih yang menyangkut uang yang benar-benar dibayarkan.
		//
		// Jalur TOLAK tidak menyentuh satu pun dari itu: ia berhenti di UPDATE tabel
		// checker. Karena itu ia dibangun penuh, dan hanya jalur setuju yang ditahan.
		//
		// # Kenapa tanpa label dan tanpa jalur massal
		//
		// Pega tidak punya tombol APPROVE/REJECT di sini. `Section/Akseptasi_PaymentLeader`
		// — section yang digambar sub-tab "Approval Payment Akseptasi" — hanya punya
		// `Refresh`, `SIMPAN`, `Dokumen`, dan `DETAILS`; keputusannya diisi DI DALAM grid
		// lalu disimpan sekali. Dan `SaveApprovalAkseptasiPaymentLeader_Act` menyentuh satu
		// baris saja: Obj-Open-By-Handle → Obj-Save → Commit.
		Decision: DecisionRule{
			Decidable: true,
			ApproveBlockedReason: "Menyetujui pembayaran di Pega ikut menjalankan transfer ke " +
				"kasir (TransferToKasir_act_Leader). Dua rule yang dibutuhkannya belum ada — " +
				"UpdateChasierIDTablePembayaran dan GetDataMSTDetailSales — dan integrasi " +
				"keluar ke sistem Kasir belum dibangun (modul S-4). Menyetujui tanpa langkah " +
				"itu akan menandai pembayaran sudah disetujui padahal tidak pernah sampai ke " +
				"kasir. Penolakan tidak terpengaruh dan tetap dapat dikerjakan di sini.",
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Catatan Atasan",
		},
	},
	{
		Code:        TabPenolakanKlaim,
		Name:        "Penolakan Klaim",
		Description: "Pengajuan pasal penolakan klaim yang menunggu persetujuan checker.",
		Kind:        KindQueue,
		// Kolom "ID" yang sempat kami tampilkan TIDAK ada di layar lama — `ID_ST` di sana
		// dipakai sebagai kunci baris, bukan digambar.
		//
		// Dua kolom terakhir layar lama, "Approval" dan "Note Approval", juga tidak dibawa
		// sebagai kolom: di Pega keduanya ISIAN di dalam grid, bukan tampilan. Di sini
		// keduanya menjadi tombol Setujui/Tolak beserta isian alasan di bilah keputusan.
		//
		// Karena itu pula tab ini TANPA label Pega dan tanpa jalur massal:
		// `Section/Sec_PenolakanKlaimChecker` hanya punya satu tombol, `Simpan Data
		// Penolakan`, dan `SaveDataCheckerPenolakan` berisi Page-New → Property-Set →
		// RDB-List atas satu baris.
		Columns: []Column{
			{Key: FieldStatusPenolak1, Title: "Status Penolakan 1"},
			{Key: FieldStatusPenolak2, Title: "Status Penolakan 2"},
			{Key: FieldPetugas, Title: "PIC"},
		},
		Decision: DecisionRule{
			Decidable:              true,
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Note Approval",
		},
	},
}

// Tabs mengembalikan ketiga belas tab dalam urutan tampilnya, tanpa memandang siapa
// pemanggilnya.
//
// Salinan, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah daftar tab dengan
// menulisi hasilnya.
func Tabs() []Tab {
	result := make([]Tab, len(tabs))
	copy(result, tabs)
	return result
}

// FindTab mencari tab menurut kodenya.
func FindTab(code string) (Tab, bool) {
	for _, tab := range tabs {
		if tab.Code == code {
			return tab, true
		}
	}
	return Tab{}, false
}

// Parent mengembalikan kode tab induk, kosong bila tab ini berada di tingkat atas.
//
// # Kenapa hanya ada EMPAT tab tingkat atas
//
// Karena di Pega pun hanya empat. `Activity/CountDashbroardManager` menulis empat pencacah
// pertama ke `TempCountDashboard.pxResults(<APPEND>)` langsung, lalu kesembilan sisanya ke
// `.pxResults(4).pxResults(<APPEND>)` — yakni sebagai ANAK baris keempat, "Approval Master" —
// dan gridnya ber-`pyRepeatDirection` TreeGrid. Kesembilan antrean persetujuan karena itu
// BUKAN tab sejajar; ia isi dari satu tab.
//
// Jenjangnya diturunkan di sini, bukan dituliskan satu per satu pada kesembilan tab, supaya
// ia dan `Counter.Parent` tidak dapat menyimpang: keduanya berangkat dari fakta yang sama,
// yakni `Kind == KindQueue`.
func (t Tab) Parent() string {
	if t.Kind == KindQueue {
		return TabApprovalMaster
	}
	return ""
}

// TopLevelTabs menyaring tab tingkat atas dari sebuah daftar.
//
// Ia menerima daftar — bukan membaca `tabs` langsung — supaya penyaringan lini bisnis yang
// sudah dijalankan VisibleTabs tidak terbuang.
func TopLevelTabs(list []Tab) []Tab {
	result := []Tab{}
	for _, tab := range list {
		if tab.Parent() == "" {
			result = append(result, tab)
		}
	}
	return result
}

// ChildTabs menyaring anak sebuah tab dari sebuah daftar.
func ChildTabs(list []Tab, parent string) []Tab {
	result := []Tab{}
	for _, tab := range list {
		if tab.Parent() == parent {
			result = append(result, tab)
		}
	}
	return result
}

// QueueTabs mengembalikan kesembilan tab antrean — isi tab "Approval Master".
func QueueTabs() []Tab {
	result := []Tab{}
	for _, tab := range tabs {
		if tab.Kind == KindQueue {
			result = append(result, tab)
		}
	}
	return result
}

// VisibleTabs mengembalikan tab yang BOLEH dilihat seorang pemanggil.
//
// # Dua belas tab terbuka bagi siapa pun yang membuka layar ini, dan itu mengikuti Pega
//
// `Section/InboxManager_Sec` tidak memasang satu pun syarat jabatan pada dua belas
// kontainernya — yang dipasangnya hanyalah `FlagManager.AlasanKlaim==n`, yakni tab mana yang
// sedang dipilih. Yang menjaga siapa boleh membuka layarnya adalah butir menunya, bukan
// section-nya.
//
// SATU tab dibatasi, dan batas itu ada di section-nya sendiri:
// `Section/InboxKonfirmasiHE_Section` tampil hanya bila
// `OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'`.
//
// Perbandingannya di sini TIDAK memandang huruf besar-kecil dan memangkas spasi, berbeda dari
// Pega yang membandingkan persis. Itu satu-satunya pelonggaran, dan ia hanya dapat MENAMBAH
// tab yang terlihat — tidak pernah menghilangkan. Alasannya: kolom
// `M_LOGIN_PNC.LINE_BUSINESS` bebas isi, dan ejaannya diketik manusia.
func VisibleTabs(caller Caller) []Tab {
	clean := caller.Clean()
	development := strings.EqualFold(clean.OrgUnit, DevelopmentOrgUnit)

	result := []Tab{}
	for _, tab := range tabs {
		if tab.LineBusiness == "" ||
			development ||
			strings.EqualFold(clean.LineBusiness, tab.LineBusiness) {
			result = append(result, tab)
		}
	}
	return result
}

// CanSee menyatakan apakah seorang pemanggil boleh membuka sebuah tab.
func CanSee(caller Caller, tab Tab) bool {
	for _, allowed := range VisibleTabs(caller) {
		if allowed.Code == tab.Code {
			return true
		}
	}
	return false
}

// DefaultTabFor adalah tab pertama yang boleh dilihat seorang pemanggil.
//
// Mengembalikan teks kosong bila tidak ada satu pun — keadaan yang secara teori mungkin
// terjadi meski hari ini tidak, karena dua belas tab tidak dibatasi apa pun.
func DefaultTabFor(caller Caller) string {
	visible := VisibleTabs(caller)
	if len(visible) == 0 {
		return ""
	}
	for _, tab := range visible {
		if tab.Code == DefaultTab {
			return tab.Code
		}
	}
	return visible[0].Code
}

// tabOrder mengembalikan urutan sebuah kode tab pada daftar di atas.
//
// Kode tab adalah ANGKA DALAM TEKS ("1".."13"), sehingga mengurutkannya sebagai teks akan
// menaruh "10" sebelum "2". Urutan diambil dari posisinya di senarai tabs, bukan dari
// nilainya, supaya satu-satunya sumber urutan adalah urutan kontainer di section.
func tabOrder(code string) int {
	for i, tab := range tabs {
		if tab.Code == code {
			return i
		}
	}
	return len(tabs)
}

// PlannedDifferences adalah selisih terhadap sistem lama yang DIPUTUSKAN, bukan cacat.
//
// # Kenapa ia data, bukan komentar
//
// Karena ia dikirim ke layar dan ditampilkan kepada pengguna. Selisih yang hanya tercatat di
// komentar akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan
// layar baru dengan Pega berdampingan.
//
// Ia juga yang dipakai saat uji kesetaraan gerbang 1: setiap selisih WAJIB dapat dipetakan ke
// salah satu butir `P-5`, atau dinyatakan sebagai bug (`D-54`).
var PlannedDifferences = []string{
	"Dashboard Outstanding kini dihitung dari satu tabel datar, bukan dari gabungan dua " +
		"tabel Pega (ketetapan Work Owner 2026-09-28). Gabungan `INNER` yang lama membuang " +
		"klaim yang belum punya penugasan terbuka — padahal justru itu pekerjaan yang " +
		"terhenti — dan menghitung klaim berpenugasan banyak berkali-kali. Keduanya hilang " +
		"bersama gabungan itu, sehingga satu klaim kini selalu dihitung sekali.",

	"Untuk sementara angka Dashboard Outstanding LEBIH KECIL daripada di Pega, dan itu " +
		"bukan karena penyaring. Tabel sumbernya baru memuat sebagian klaim — 872 klaim PNC " +
		"saat diperiksa 2026-09-28 — karena proses pengisinya belum mengejar. Selisih ini " +
		"menyusut sendiri begitu tabelnya terisi penuh, tanpa perubahan kode.",

	"Penyaring Reinsurer pada tab Outstanding memakai nilai yang tersimpan di tabel " +
		"datar. Pega menempuh jalur lain: ia membaca tabel cuplikan lebih dulu, lalu jatuh " +
		"ke master sales di basis data lain lewat DB Link — jalur yang penggantinya belum " +
		"ada (D-25, R-03). Klaim yang hanya dikenali lewat jalur itu karena itu tidak ikut " +
		"tersaring.",

	"Grid ketiga tab Outstanding memakai SATU dasar tahun — tanggal klaim dibuat — untuk " +
		"kolom maupun isinya. Pega memakai dua: daftar kolomnya dari tanggal dibuat, " +
		"sedangkan selnya dari tahun registrasi bila ada. Klaim yang kedua tahunnya berbeda " +
		"karena itu jatuh ke kolom yang tidak ada di sana, dan hilang dari tabel tanpa jejak.",

	"Pencacah \"Outstanding\" di kepala layar kini menghitung KLAIM, sesuai namanya. Kueri " +
		"lamanya menggabungkan tabel objek pertanggungan tanpa memilih satu kolom pun " +
		"darinya, sehingga klaim yang berobjek banyak terhitung berkali-kali. Gabungan itu " +
		"tidak dibawa.",

	"Sub-tab \"Approval Progress Klaim\" tidak dibawa. Di Pega ia bersyarat `1==2` — sebuah " +
		"kondisi yang tidak pernah benar — sehingga sudah dimatikan di sana dan tidak pernah " +
		"tampil bagi siapa pun.",

	"Menolak WAJIB menyertakan alasan pada antrean yang tabelnya punya kolom alasan. Pega " +
		"menerima penolakan tanpa alasan apa pun. Kolom itu satu-satunya hal yang memberi " +
		"tahu pengaju kenapa barisnya ditolak, dan baris yang ditolak tanpa keterangan akan " +
		"diajukan ulang apa adanya.",

	"Menyetujui pembayaran pada tab Payment Klaim Akseptasi DITAHAN, sementara menolak " +
		"tetap dapat dikerjakan. Di Pega persetujuan ikut menjalankan transfer ke kasir, dan " +
		"rantai itu belum lengkap. Alasannya ditampilkan di tempat tombolnya, bukan " +
		"disembunyikan.",

	"Keputusan tidak dapat menimpa keputusan orang lain. Setiap pernyataan ikut menyaring " +
		"status menunggu di samping kuncinya, sehingga baris yang sudah diputuskan orang " +
		"lain sementara daftar Anda masih yang lama TIDAK berubah — dan layar menyatakan " +
		"berapa banyak yang demikian. Di Pega keputusan terakhir menang tanpa ada yang tahu.",

	"Ketiga belas kontainer menjadi TAB yang dapat dipilih. Di Pega ketiganya kontainer " +
		"terpisah yang ditampilkan dengan mengubah `FlagManager.AlasanKlaim`. Isi, kolom, " +
		"dan urutan kolomnya sama.",

	"Pada Dashboard Klaim dan Dashboard Produktivitas, ketiga keranjang TIDAK berjumlah " +
		"sama dengan kolom Total, dan itu perilaku sistem lama yang dibawa apa adanya. " +
		"Kuerinya menghitung status klaim `1`, `3`, dan \"selain 1, 2, dan 3\" — sehingga " +
		"baris berstatus `2` masuk Total tetapi tidak masuk satu keranjang pun. Saat " +
		"diperiksa 2026-09-28 ada 252 baris seperti itu dari 66.972.",

	"Penyaring periode dikirim sebagai parameter terikat, bukan sebagai potongan SQL. " +
		"Sembilan kueri layar ini di Pega menyisipkan teks SQL lewat pola `ASIS` — termasuk " +
		"`to_char(tglklaim,'mm-rrrr')` dan `trunc(tglklaim)`, yang menyentuh kolomnya pada " +
		"setiap baris sehingga index tidak terpakai. Penggantinya selang setengah terbuka " +
		"yang kedua batasnya dihitung di aplikasi.",
}
