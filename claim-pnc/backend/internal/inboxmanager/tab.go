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
)

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
var dashboardOutstandingPanels = []Panel{
	{
		Key:   "pic",
		Title: "Outstanding per PIC",
		Columns: []Column{
			{Key: FieldPIC, Title: "Nama PIC"},
			{Key: FieldJumlahKlaim, Title: "Jumlah Klaim"},
		},
	},
	{
		Key:   "grup_bisnis",
		Title: "Outstanding per Grup Bisnis",
		Columns: []Column{
			{Key: FieldGrupBisnis, Title: "Grup Bisnis"},
			{Key: FieldJumlahKlaim, Title: "Jumlah Klaim"},
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
var dashboardProduktivitasColumns = []Column{
	{Key: FieldDimensi, Title: "Grup Bisnis / PIC"},
	{Key: FieldTotalPeriodeIni, Title: "Total — Periode Ini"},
	{Key: FieldTotalPeriodeLTY, Title: "Total — Tahun Lalu"},
	{Key: FieldAksepPeriodeIni, Title: "Akseptasi — Periode Ini"},
	{Key: FieldAksepPeriodeLTY, Title: "Akseptasi — Tahun Lalu"},
	{Key: FieldTolakPeriodeIni, Title: "Ditolak — Periode Ini"},
	{Key: FieldTolakPeriodeLTY, Title: "Ditolak — Tahun Lalu"},
	{Key: FieldOSPeriodeIni, Title: "Outstanding — Periode Ini"},
	{Key: FieldOSPeriodeLTY, Title: "Outstanding — Tahun Lalu"},
}

// dashboardKlaimColumns adalah kolom grid "Total Klaim Bisnis" pada tab Klaim.
//
// Empat pencacah dan tiga nilai uang, dibaca dari `RDB List/BrowseCaseClaim-SQL.xml`.
// Aliasnya sama menyesatkannya dengan tab Produktivitas — `BUSINESSNAME` untuk jumlah klaim,
// `NOPOLIS` untuk sebuah jumlah uang — dan diabaikan dengan cara yang sama.
var dashboardKlaimColumns = []Column{
	{Key: FieldNamaBisnisDK, Title: "Grup Bisnis"},
	{Key: FieldTotalKlaim, Title: "Total Klaim"},
	{Key: FieldJumlahAksep, Title: "Jumlah Akseptasi"},
	{Key: FieldJumlahTolak, Title: "Jumlah Ditolak"},
	{Key: FieldJumlahOS, Title: "Jumlah Outstanding"},
	{Key: FieldNilaiAksep, Title: "Nilai Akseptasi", Money: true},
	{Key: FieldNilaiTolak, Title: "Nilai Ditolak", Money: true},
	{Key: FieldNilaiOS, Title: "Nilai Outstanding", Money: true},
}

// dashboardKlaimCauseColumns adalah kolom grid "Total Klaim CauseOfLoss" pada tab Klaim.
//
// Ia grid KEDUA yang diikat `Section/DashboardKlaim_sec` (`TempBrowseCase2.pxResults`),
// dipasok `BrowseCaseClaimPerCauseOfLoss` — kueri yang sama dengan grid pertama ditambah satu
// dimensi `COL_DESC`, yakni penyebab kerugian.
var dashboardKlaimCauseColumns = append(
	[]Column{
		{Key: FieldNamaBisnisDK, Title: "Grup Bisnis"},
		{Key: FieldPenyebab, Title: "Penyebab Kerugian"},
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
	},
	{
		Code: TabProduktivitas,
		Name: "Produktivitas Klaim",
		Description: "Perbandingan jumlah klaim periode berjalan dengan periode yang sama " +
			"tahun lalu, per grup bisnis dan per PIC.",
		Kind: KindDashboard,
		Panels: []Panel{
			{Key: "grup_bisnis", Title: "Produktivitas per Grup Bisnis", Columns: dashboardProduktivitasColumns},
			{Key: "pic", Title: "Produktivitas per PIC", Columns: dashboardProduktivitasColumns},
		},
		HasPeriodFilter: true,
	},
	{
		Code:        TabKlaim,
		Name:        "Klaim",
		Description: "Jumlah dan nilai klaim per grup bisnis, dan rinciannya per penyebab kerugian.",
		Kind:        KindDashboard,
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
		Code:        TabMasterBengkel,
		Name:        "Master Bengkel",
		Description: "Pengajuan data bengkel yang menunggu persetujuan.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Bengkel"},
			{Key: FieldNama, Title: "Nama Bengkel"},
			{Key: FieldCabang, Title: "Nama Cabang"},
			{Key: FieldKota, Title: "Nama Kota"},
			{Key: FieldKeterangan, Title: "Alamat Bengkel"},
		},
		Decision: DecisionRule{
			Decidable:              true,
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Alasan Status Bengkel",
		},
	},
	{
		Code:        TabMasterPanel,
		Name:        "Master Panel",
		Description: "Pengajuan data panel kendaraan yang menunggu persetujuan.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Panel"},
			{Key: FieldNama, Title: "Nama Panel"},
		},
		Decision: DecisionRule{
			Decidable:              true,
			ReasonRequiredOnReject: true,
			ReasonLabel:            "Alasan Tolak",
		},
	},
	{
		Code: TabNomorRangka,
		Name: "Approval Nomor Rangka Beda",
		Description: "Klaim yang nomor rangka dari bengkel berbeda dengan nomor rangka yang " +
			"diinput pengguna.",
		Kind: KindQueue,
		Columns: []Column{
			{Key: FieldNoKlaim, Title: "No Klaim"},
			{Key: FieldPengirim, Title: "Pemilik Kendaraan"},
			{Key: FieldMerk, Title: "Merk Kendaraan"},
			{Key: FieldModel, Title: "Model Kendaraan"},
			{Key: FieldTipe, Title: "Tipe Kendaraan"},
			{Key: FieldRangkaUser, Title: "No Rangka (Inputan User)"},
			{Key: FieldRangkaBengkel, Title: "No Rangka (Dari Bengkel)"},
		},

		// Tabelnya TIDAK punya kolom alasan — delapan kolomnya habis untuk identitas
		// kendaraan dan status. Isian alasan karena itu tidak digambar sama sekali, bukan
		// digambar lalu isinya dibuang.
		Decision: DecisionRule{Decidable: true},

		// SATU-SATUNYA tab yang dibatasi lini bisnis, dan batas itu dibaca dari export:
		// `Section/InboxKonfirmasiHE_Section-Section.xml` memasang
		// `OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'`.
		LineBusiness: LineNonMBU,
	},
	{
		Code:        TabMasterSparepart,
		Name:        "Master Sparepart",
		Description: "Pengajuan data sparepart yang menunggu persetujuan.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Sparepart"},
			{Key: FieldNama, Title: "Nama Sparepart"},
			{Key: FieldKategori, Title: "Kategori Sparepart"},
			{Key: FieldKeterangan, Title: "Nomor Sparepart"},
		},
		Decision: DecisionRule{Decidable: true},
	},
	{
		Code:        TabKategoriSparepart,
		Name:        "Master Kategori Sparepart",
		Description: "Pengajuan kategori sparepart yang menunggu persetujuan.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Kategori Sparepart"},
			{Key: FieldNama, Title: "Sparepart Kategori"},
		},
		Decision: DecisionRule{Decidable: true},
	},
	{
		Code:        TabTipeSparepart,
		Name:        "Master Tipe Sparepart",
		Description: "Pengajuan tipe sparepart yang menunggu persetujuan.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID Tipe Sparepart"},
			{Key: FieldNama, Title: "Nama Tipe Sparepart"},
			{Key: FieldKategori, Title: "Nama Kategori Sparepart"},
		},
		Decision: DecisionRule{Decidable: true},
	},
	{
		Code:        TabGroupingSparepart,
		Name:        "Master Grouping Sparepart",
		Description: "Pengajuan pengelompokan sparepart menurut nomor rangka.",
		Kind:        KindQueue,
		Columns: []Column{
			{Key: FieldID, Title: "ID"},
			{Key: FieldRangkaUser, Title: "No Rangka"},
			{Key: FieldNama, Title: "Nama Sparepart"},
			{Key: FieldKeterangan, Title: "No Sparepart"},
		},
		Decision: DecisionRule{Decidable: true},
	},
	{
		Code: TabPaymentAkseptasi,
		Name: "Payment Klaim Akseptasi",
		Description: "Nomor akseptasi yang menunggu persetujuan atasan sebelum pembayaran " +
			"diteruskan ke kasir.",
		Kind: KindQueue,
		Columns: []Column{
			{Key: FieldNoKlaim, Title: "No Klaim"},
			{Key: FieldNoAkseptasi, Title: "No Akseptasi"},
			{Key: FieldPIC, Title: "PIC"},
			{Key: FieldTanggalInput, Title: "Tanggal Input"},
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
		Columns: []Column{
			{Key: FieldID, Title: "ID"},
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

	"Dua dashboard berjudul \"Outstanding\" kini benar-benar hanya menghitung klaim yang " +
		"berjalan. Dua dari tiga kueri Pega yang memasoknya — per grup bisnis dan per " +
		"tahun — TIDAK menyaring status kerja sama sekali, sehingga klaim yang sudah selesai " +
		"ikut terhitung dan angkanya tidak pernah cocok dengan pencacah di kepala layar yang " +
		"sama. Penyaringnya ditambahkan.",

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
