package inboxclaimtreatynonprop

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena judul dan isi tidak selalu sejalan di layar ini:
// kolom berjudul "Status" pada grid Teknik berisi WAKTU objek kerja dibuat, dan kolom master
// id berjudul "MasterID" pada grid Admin tetapi "ID Master" pada grid Teknik — sekaligus
// membaca sumber yang berbeda. Ketiganya persis seperti di
// `Section/InboxClaimNonProp_Harness-Section.xml`.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`),
	// termasuk salah ejanya — lihat TabTechnical.
	Title string
}

// Nama field JSON pada satu baris pekerjaan.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldClaimID            = "no_klaim"
	FieldMasterID           = "master_id"
	FieldJSONMasterID       = "id_master"
	FieldPolicyNumber       = "no_polis"
	FieldLossDate           = "tanggal_kejadian"
	FieldBusinessName       = "nama_bisnis"
	FieldBusinessSource     = "sumber_bisnis"
	FieldCedingCompany      = "ceding_co"
	FieldInsuredName        = "nama_tertanggung"
	FieldCreatedAt          = "dibuat_pada"
	FieldAgingDays          = "aging"
	FieldCreateOperator     = "operator_pembuat"
	FieldLastUpdateOperator = "operator_pengubah"
)

// Kode tab.
//
// # Kenapa angka, dan kenapa TIDAK memakai nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Yang dipakainya adalah nama
// HALAMAN KLIPBOARD yang diisi tiap grid — `ListCaseInbox`, `ListCaseInboxTeknik`,
// `ListCaseInboxKomite` — dan nama halaman klipboard adalah detail mesin Pega, bukan
// kontrak yang layak dibawa ke API.
//
// Karena itu kodenya ditetapkan di sini, berurutan, dan diperlakukan sebagai kontrak modul
// ini sendiri. Penelusuran balik ke export ditempuh lewat nama kuerinya — yang disebut
// lengkap pada setiap tab di bawah — bukan lewat kodenya.
//
// Nilainya sengaja SAMA dengan modul Prop (`1`, `2`, `3`) supaya kedua layar saudara ini
// tidak menuntut dua cara membaca kode tab. Ia tetap kontrak modul masing-masing: kesamaan
// ini kemudahan, bukan ketergantungan.
const (
	TabAdmin     = "1"
	TabTechnical = "2"
	TabCommittee = "3"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Tab Admin, karena hanya tab inilah yang isinya MILIK pemanggil. Membuka layar pada antrean
// bersama akan menampilkan pekerjaan orang lain lebih dulu.
const DefaultTab = TabAdmin

// Tab adalah satu antrean kerja pada layar Inbox Claim Treaty Non Prop.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah teks PILIHAN pada dropdown pemilih antrean.
	//
	// Ia diambil dari `Activity/GetWorkCNP_Act-Act.xml`, yang menyusun daftar pilihan
	// dropdown itu sebagai `ListWorkBasket.pxResults(<APPEND>).CARI1` bernilai
	// `"Treaty-In Admin"` dan `"Treaty-In Teknik"`. Di layar ini label dan nilainya SATU
	// properti yang sama — prakondisi langkah-langkahnya membandingkan
	// `ParamInboxCNP.CARI22 == "Treaty-In Teknik"`, yaitu teks yang dibaca pengguna itu
	// sendiri.
	//
	// Jangan tertukar dengan GridTitle di bawah: keduanya teks berbeda yang tampil
	// BERSAMAAN di layar yang sama.
	Name string

	// GridTitle adalah judul KONTAINER grid yang digambar sesudah dropdown.
	//
	// Diambil dari `<pyValue>` kontainer di `Section/InboxClaimNonProp_Harness-Section.xml`.
	// Dropdown-nya bertuliskan "Treaty-In Teknik", sementara grid di bawahnya berjudul
	// "Work Treatyin Non Propotional Teknik" — dan itu memang dua teks yang berbeda.
	//
	// KOSONG pada tab yang Blocked: tidak ada grid yang digambar di sana.
	GridTitle string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena judul seperti "Work Treatyin Non
	// Propotional Teknik" tidak memberi tahu apa pun tentang isinya.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	//
	// KOSONG pada tab yang Blocked: tidak ada grid yang digambar di sana.
	Columns []Column

	// ScopedToCaller menyatakan kuerinya menyaring menurut pengguna yang login.
	//
	// Hanya tab Admin begitu, dan itulah yang membuatnya "antrean saya". Penyaringnya
	// `PXASSIGNEDOPERATORID = {Inputdata.CARI10}` pada
	// `RDB List/GetKlaimNonPropAdmin_SQL-SQL.xml`.
	ScopedToCaller bool

	// SupportsSeeAll menyatakan checkbox "See All Claim" berlaku pada tab ini.
	//
	// Di sistem lama ia properti `SearchWorkbasket.CARI55`, dan mencentangnya mengosongkan
	// `Inputdata.CARI10` sehingga kueri bertukar dari `GetKlaimNonPropAdmin_SQL` menjadi
	// `GetKlaimNonPropAdminALL_SQL` — yang menghapus penyaring operator. Ia HANYA berlaku
	// pada tab Admin.
	SupportsSeeAll bool

	// SupportsTBAOnly menyatakan checkbox "See TBA Claim" berlaku pada tab ini.
	//
	// Di sistem lama ia properti `SearchWorkbasket.CARI54`, dan mencentangnya mengosongkan
	// `Inputdata.CARI11` sehingga kueri `GetKlaimNonPropAdminTBA_SQL` ikut berjalan —
	// kueri yang menambahkan `c.NOPOLIS IS NULL`.
	//
	// "TBA" berarti *to be advised*: klaim treaty yang sudah masuk tetapi nomor polisnya
	// belum terbit. Ia HANYA berlaku pada tab Admin.
	//
	// Perilakunya di sini BERBEDA dari sistem lama — lihat PlannedDifferences.
	SupportsTBAOnly bool

	// Blocked menyatakan tab ini digambar tetapi belum dapat diisi.
	//
	// Ia BUKAN tab yang disembunyikan. Tabnya tetap tampak supaya pengguna tahu fiturnya
	// ada dan terhalang, bukan mengira ia hilang — perlakuan yang sama dengan tab komite
	// pada modul Prop.
	Blocked bool

	// BlockedReason menjelaskan penghalangnya dalam kalimat yang dapat langsung
	// ditampilkan ke pengguna. Kosong bila tab tidak terhalang.
	BlockedReason string

	// BlockedOwner menyebut siapa yang dapat menghilangkan penghalangnya.
	//
	// Ia disebut supaya penghalang punya alamat. Penghalang tanpa pemilik tidak pernah
	// hilang — itu pelajaran yang sudah tercatat di `D-36`.
	BlockedOwner string
}

// Kolom yang dipakai lebih dari satu tab, disusun sekali supaya judulnya tidak dapat
// berbeda antar tab tanpa disengaja.
// Judulnya ditulis PERSIS seperti `<pyValue>` sel kepala di section, termasuk yang
// janggal: "Claim.ID" bertitik di tengah, dan "List Update Operator" yang jelas maksudnya
// "Last". Section yang sama memuat kedua ejaan — "List Update Operator" pada grid Admin dan
// Teknik, "Last Update Operator" pada grid Komite — dan masing-masing dibawa ke gridnya
// sendiri. `D-13` menetapkan tampilan meniru Pega; merapikannya berarti pengguna mencari
// kolom yang tidak ada.
var (
	colClaimID            = Column{Key: FieldClaimID, Title: "Claim.ID"}
	colPolicyNumber       = Column{Key: FieldPolicyNumber, Title: "Policy No"}
	colLossDate           = Column{Key: FieldLossDate, Title: "Date of Loss"}
	colBusinessName       = Column{Key: FieldBusinessName, Title: "Business Name"}
	colBusinessSource     = Column{Key: FieldBusinessSource, Title: "Source of Business"}
	colCedingCompany      = Column{Key: FieldCedingCompany, Title: "Ceding Co Name"}
	colInsuredName        = Column{Key: FieldInsuredName, Title: "Insured Name"}
	colAgingDays          = Column{Key: FieldAgingDays, Title: "Aging"}
	colCreateOperator     = Column{Key: FieldCreateOperator, Title: "Create Operator"}
	colLastUpdateOperator = Column{Key: FieldLastUpdateOperator, Title: "List Update Operator"}
)

// tabs adalah ketiga antrean beserta kolomnya, berurutan seperti tampilnya.
//
// # Dari mana urutan kolomnya
//
// Dari urutan SEL di `Section/InboxClaimNonProp_Harness-Section.xml`, bukan dari urutan
// kolom kuerinya. Keduanya berbeda — kueri menaruh nomor polis di tengah, section
// menaruhnya sesudah Ceding Co — dan yang dilihat pengguna adalah yang pertama.
//
// Versi sebelumnya di sini memakai urutan KUERI, dengan alasan bahwa sel section terikat
// alias `CARI` bernomor sehingga sulit ditelusuri. Alasan itu tidak lagi berlaku: setiap
// sel data grid membawa `<pyValue>.CARInn</pyValue>` yang dapat dibaca berurutan, dan
// hasilnya di bawah sudah dicocokkan dengan layar Pega produksi.
//
//	                   grid Admin (11 kolom)     grid Teknik (12 kolom)
//	1  Claim.ID        CARI11                    CARI11
//	2  master id       CARI23 = JSON IDMaster    CARI19 = b.MASTERID
//	3  Insured Name    CARI17                    CARI17
//	4  Date of Loss    CARI22                    CARI22
//	5  Business Name   CARI15                    CARI15
//	6  Source of Bus.  CARI14                    CARI14
//	7  Ceding Co Name  CARI16                    CARI16
//	8  Policy No       CARI18                    CARI18
//	9  Aging           CARI20                    CARI20
//	10 Create Operator CARI12                    CARI12
//	11 List Update Op. CARI24                    CARI24
//	12 Status          —                         CARI21 = b.PXCREATEDATETIME
//
// Baris ke-2 itu yang paling mudah salah: KEDUA grid menggambar SATU kolom master id, dan
// keduanya mengambilnya dari sumber yang BERBEDA — grid Admin dari blob JSON, grid Teknik
// dari kolom objek kerja. Judulnya pun berbeda, dan keduanya dibawa apa adanya.
//
// Judul setiap kolom diambil dari `<pyValue>` sel kepalanya — itulah teks yang selama ini
// dibaca pengguna (`D-13`).
var tabs = []Tab{
	{
		Code: TabAdmin,

		// Teks pilihan dropdown, dari `GetWorkCNP_Act` langkah yang mengisi
		// `ListWorkBasket.pxResults(<APPEND>).CARI1 := "Treaty-In Admin"`.
		Name: "Treaty-In Admin",

		// Salah eja "Propotional" DIPERTAHANKAN. Ia tertulis begitu di section
		// (`<pyValue>Work Treatyin Non Propotional Admin</pyValue>`), dan `D-13`
		// menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang.
		// Membetulkannya akan membuat grid ini tidak dikenali pengguna yang mencarinya,
		// dan itu harga yang lebih mahal daripada satu huruf yang hilang.
		GridTitle: "Work Treatyin Non Propotional Admin",

		Description: "Klaim treaty non-proporsional yang ditugaskan kepada Anda. " +
			"Centang \"See All Claim\" untuk melihat penugasan seluruh petugas, atau " +
			"\"See TBA Claim\" untuk menyaring klaim yang nomor polisnya belum terbit.",

		Columns: []Column{
			colClaimID,
			// Grid Admin membaca master id dari BLOB JSON (`CARI23`), dan judulnya
			// "MasterID" dirangkai tanpa spasi. Grid Teknik memakai kolom objek kerja
			// dengan judul "ID Master" — dua hal yang berbeda, dan keduanya dibawa apa
			// adanya. Lihat WorkItem.JSONMasterID.
			{Key: FieldJSONMasterID, Title: "MasterID"},
			colInsuredName, colLossDate, colBusinessName,
			colBusinessSource, colCedingCompany, colPolicyNumber,
			colAgingDays, colCreateOperator, colLastUpdateOperator,
			// TIDAK ada kolom "Status" di sini. Ketiga kueri Admin tidak memilih
			// `b.PXCREATEDATETIME` sama sekali, dan grid Admin di section memang berhenti
			// di sel ke-11.
		},

		ScopedToCaller:  true,
		SupportsSeeAll:  true,
		SupportsTBAOnly: true,
	},
	{
		Code: TabTechnical,

		// Teks pilihan dropdown, dari `GetWorkCNP_Act`:
		// `ListWorkBasket.pxResults(<APPEND>).CARI1 := "Treaty-In Teknik"`.
		Name: "Treaty-In Teknik",

		// Salah eja "Propotional" DIPERTAHANKAN, alasan yang sama dengan grid di atas.
		// Perhatikan section memakai DUA ejaan berbeda untuk kata yang sama pada layar
		// ini — "Propotional" pada kedua grid pertama dan "Proportional" pada grid komite.
		// Keduanya dibawa apa adanya; menyeragamkannya berarti memilih salah satu yang
		// benar, dan tidak ada dasar untuk memilih.
		GridTitle: "Work Treatyin Non Propotional Teknik",

		Description: "Antrean bersama PIC Teknik treaty — belum diambil siapa pun.",

		Columns: []Column{
			colClaimID,
			// Master id di sini dari KOLOM objek kerja (`CARI19`), berjudul dengan spasi.
			{Key: FieldMasterID, Title: "ID Master"},
			colInsuredName, colLossDate, colBusinessName,
			colBusinessSource, colCedingCompany, colPolicyNumber,
			colAgingDays, colCreateOperator, colLastUpdateOperator,
			// Kolom ke-12, HANYA ada di grid ini. Judulnya "Status" tetapi isinya waktu
			// objek kerja dibuat — lihat WorkItem.CreatedAt.
			{Key: FieldCreatedAt, Title: "Status"},
		},
	},
	{
		Code: TabCommittee,

		// Grid ini memakai ejaan "Proportional" yang BENAR, tidak seperti kedua grid di
		// atas. Dibawa apa adanya.
		//
		// Ia TIDAK punya teks pilihan dropdown yang dapat dibaca: `GetWorkCNP_Act` hanya
		// menambahkan dua pilihan, dan grid komite ditampilkan lewat pemeriksaan keadaan
		// (`InputData.CARI13 == "tampil"`), bukan lewat dropdown. Judul kontainernya yang
		// dipakai sebagai teks pilihan di sini — lihat BlockedReason.
		Name:        "Work List Treatyin Non Proportional Komite",
		Description: "Antrean persetujuan komite klaim treaty non-proporsional.",

		// Tidak ada kolom, karena tidak ada grid yang digambar. Menyebut kolomnya di sini
		// akan membuat layar menggambar tabel kosong yang terbaca sebagai "tidak ada
		// pekerjaan" — padahal yang benar adalah "belum dapat dibaca".
		Blocked: true,

		BlockedReason: "Antrean komite treaty non-proporsional belum dapat dibaca sistem " +
			"baru. Sumbernya di Pega adalah kueri `KmtGetInboxListCNP_SQL`, yang " +
			"DIPANGGIL oleh Activity/GetWorkCNP_Act-Act.xml tetapi TIDAK ADA di export " +
			"rule — sehingga tidak diketahui tabel mana yang dibacanya maupun kolom apa " +
			"yang dikembalikannya. Menyusunnya sendiri dari pola kedua kueri lain berarti " +
			"menebak aturan yang menentukan persetujuan nilai uang.",

		BlockedOwner: "Tim Pega — dibutuhkan export rule `KmtGetInboxListCNP_SQL` " +
			"(kelas Assign-WorkBasket, ruleset GCNMFW). Ia bagian dari permintaan export " +
			"ulang berbasis Product rule (`D-39`, `R-16`).",
	},
}

// Tabs mengembalikan ketiga tab dalam urutan tampilnya.
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

// PlannedDifferences adalah selisih terhadap sistem lama yang DIPUTUSKAN, bukan cacat.
//
// # Kenapa ia data, bukan komentar
//
// Karena ia dikirim ke layar dan ditampilkan kepada pengguna. Selisih yang hanya tercatat
// di komentar akan dilaporkan berulang kali sebagai kerusakan oleh orang yang
// membandingkan layar baru dengan Pega berdampingan.
//
// Ia juga yang dipakai saat uji kesetaraan gerbang 1: setiap selisih WAJIB dapat dipetakan
// ke salah satu butir `P-5`, atau dinyatakan sebagai bug (`D-54`). Butir di bawah adalah
// pemetaan itu, sudah tertulis di muka alih-alih dicari setelah selisihnya muncul.
var PlannedDifferences = []string{
	"Checkbox \"See TBA Claim\" kini BERDIRI SENDIRI. Di sistem lama ia hanya berpengaruh " +
		"bila \"See All Claim\" ikut dicentang — kueri TBA-nya dijaga dua prakondisi " +
		"ber-AND (Inputdata.CARI10==\"\" DAN Inputdata.CARI11==\"\"), sehingga mencentang " +
		"TBA sendirian tidak mengubah apa pun sama sekali. Di sini keduanya penyaring yang " +
		"saling bebas: TBA sendirian menyaring klaim Anda yang polisnya belum terbit. " +
		"Perbaikan ini disetujui Work Owner 2026-09-22 sebagai selisih terencana P-5.",

	"Kolom \"Status\" pada antrean Teknik berisi WAKTU objek kerja dibuat, bukan status " +
		"klaim. Judulnya menyesatkan sejak di Pega dan dipertahankan apa adanya (D-13): sel " +
		"ke-12 grid itu berjudul Status tetapi terikat CARI21, yang di GetInboxListCNP_SQL " +
		"adalah b.PXCREATEDATETIME. Bentuk teksnya pun dipertahankan " +
		"(20240201T095612.955 GMT), karena itulah yang selama ini terbaca di layar.",

	"Teks \"Estimation\" dan \"Acceptation\" TIDAK lagi digambar. Keduanya memang dipilih " +
		"keempat kueri lama sebagai teks tetap (CARI13), tetapi tidak satu pun sel di " +
		"Section/InboxClaimNonProp_Harness-Section.xml terikat padanya — jadi keduanya tidak " +
		"pernah sampai ke layar Pega. Versi sebelumnya di sini menggambarnya di bawah judul " +
		"\"Status\", dan itu keliru: kolom Status yang sebenarnya berisi hal lain.",

	"Kolom \"Aging\" menghitung HARI KALENDER sejak objek kerja dibuat, bukan hari kerja. " +
		"Akhir pekan dan hari libur ikut terhitung. Ia karena itu bukan TAT — perhitungan " +
		"TAT memotong jam kerja lewat GET_WORKING_HOURS, dan layar ini tidak menyentuhnya.",

	"Tombol \"Create Claim Treaty Non Prop\" belum membuat klaim. Selama Pega dan sistem " +
		"baru berjalan berdampingan, tabel objek kerja hanya boleh ditulis satu sistem " +
		"(P-1), dan tabel itu masih dimiliki Pega. Buat klaim treaty non-prop baru lewat " +
		"Pega.",

	"Berkas ekspor memakai judul kolom yang terbaca. Di sistem lama berkasnya dihasilkan " +
		"MSOGenerateExcelFile atas properti bernama CARI1…CARI7, sehingga baris judulnya " +
		"berisi nama properti itu apa adanya. Isi kolomnya sama persis, urutannya sama " +
		"persis; yang berubah hanya baris judulnya.",

	"Urutan baris ditetapkan tegas. Ketiga kueri tab Admin di sistem lama tidak " +
		"mengurutkan hasilnya sama sekali — dapat dibiarkan selama seluruh baris ditarik " +
		"sekaligus, tetapi membuat satu baris muncul di dua halaman begitu hasilnya " +
		"dipotong per halaman.",

	"Sejak 2026-10-08 tabel objek kerja Pega tidak lagi dibaca (keputusan Work Owner). " +
		"Kolom Business Name, Source of Business, Ceding Co Name, Insured Name, dan MasterID " +
		"kini diambil dari dokumen klaim di POOLDATA.JSON_KLAIM, sehingga dapat terisi pada " +
		"baris yang dulu kosong. Kolom \"Status\" dan \"Last Update Operator\" KOSONG dan " +
		"\"Aging\" bernilai 0 karena tidak punya padanan di tabel pengganti, dan urutan baris kini " +
		"mengikuti waktu penugasan, bukan waktu objek kerja dibuat.",
}
