package inboxclaimtreatynonprop

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke tab —
// `BusinessName` berjudul "Business Name" pada tab Admin dan "Class of Business" pada tab
// Teknik, persis seperti di `Section/InboxClaimNonProp_Harness-Section.xml`.
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
	FieldStatus             = "status"
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

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`.
	Name string

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
var (
	colClaimID            = Column{Key: FieldClaimID, Title: "No Klaim"}
	colMasterID           = Column{Key: FieldMasterID, Title: "MasterID"}
	colPolicyNumber       = Column{Key: FieldPolicyNumber, Title: "Policy No"}
	colLossDate           = Column{Key: FieldLossDate, Title: "Date of Loss"}
	colBusinessSource     = Column{Key: FieldBusinessSource, Title: "Source of Business"}
	colCedingCompany      = Column{Key: FieldCedingCompany, Title: "Ceding Co Name"}
	colInsuredName        = Column{Key: FieldInsuredName, Title: "Insured Name"}
	colStatus             = Column{Key: FieldStatus, Title: "Status"}
	colAgingDays          = Column{Key: FieldAgingDays, Title: "Aging"}
	colCreateOperator     = Column{Key: FieldCreateOperator, Title: "Create Operator"}
	colLastUpdateOperator = Column{Key: FieldLastUpdateOperator, Title: "Last Update Operator"}
)

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya.
//
// # Dari mana urutan kolomnya
//
// Dari urutan kolom KUERI-nya, bukan dari urutan sel di section. Itu berbeda dari modul
// Prop, dan alasannya konkret: di layar ini ketiga grid membaca halaman klipboard yang sama
// dengan alias `CARI` bernomor, sehingga urutan sel di section tidak dapat dibaca sebagai
// urutan kolom tanpa menelusuri tiap sel satu per satu ke nomor aliasnya. Urutan kueri
// terbaca utuh dan tidak dapat salah tafsir.
//
// Judulnya tetap diambil dari `<pyCaption …>` pada section — itulah teks yang selama ini
// dibaca pengguna (`D-13`).
var tabs = []Tab{
	{
		Code: TabAdmin,

		// Salah eja "Propotional" DIPERTAHANKAN. Ia tertulis begitu di section
		// (`pyCaption Work Treatyin Non Propotional Admin`), dan `D-13` menetapkan
		// tampilan meniru Pega supaya pengguna tidak perlu belajar ulang. Membetulkannya
		// akan membuat tab ini tidak dikenali pengguna yang mencarinya, dan itu harga yang
		// lebih mahal daripada satu huruf yang hilang.
		Name: "Work Treatyin Non Propotional Admin",

		Description: "Klaim treaty non-proporsional yang ditugaskan kepada Anda. " +
			"Centang \"See All Claim\" untuk melihat penugasan seluruh petugas, atau " +
			"\"See TBA Claim\" untuk menyaring klaim yang nomor polisnya belum terbit.",

		Columns: []Column{
			colClaimID,
			// Kedua kolom master id memang berdampingan, dan memang dari dua sumber yang
			// berbeda — lihat WorkItem.JSONMasterID.
			colMasterID,
			{Key: FieldJSONMasterID, Title: "ID Master"},
			colPolicyNumber, colLossDate,
			{Key: FieldBusinessName, Title: "Business Name"},
			colBusinessSource, colCedingCompany, colInsuredName,
			colStatus, colAgingDays,
			colCreateOperator, colLastUpdateOperator,
		},

		ScopedToCaller:  true,
		SupportsSeeAll:  true,
		SupportsTBAOnly: true,
	},
	{
		Code: TabTechnical,

		// Salah eja "Propotional" DIPERTAHANKAN, alasan yang sama dengan tab di atas.
		// Perhatikan section memakai DUA ejaan berbeda untuk kata yang sama pada layar
		// ini — "Propotional" pada kedua tab pertama dan "Proportional" pada tab komite.
		// Keduanya dibawa apa adanya; menyeragamkannya berarti memilih salah satu yang
		// benar, dan tidak ada dasar untuk memilih.
		Name: "Work Treatyin Non Propotional Teknik",

		Description: "Antrean bersama PIC Teknik treaty — belum diambil siapa pun.",

		Columns: []Column{
			colClaimID,
			// TIDAK ada kolom "ID Master" di sini: `GetInboxListCNP_SQL` tidak membawa
			// `c.data_json.IDMaster` sama sekali. Menggambarnya akan menghasilkan kolom
			// yang kosong di seluruh baris — terbaca sebagai data hilang, padahal memang
			// tidak pernah diambil.
			colMasterID,
			colPolicyNumber, colLossDate,
			// Judulnya BERBEDA dari tab Admin untuk isi yang sama, mengikuti section.
			{Key: FieldBusinessName, Title: "Class of Business"},
			colBusinessSource, colCedingCompany, colInsuredName,
			colStatus, colAgingDays,
			colCreateOperator, colLastUpdateOperator,
		},
	},
	{
		Code: TabCommittee,

		// Tab ini memakai ejaan "Proportional" yang BENAR, tidak seperti kedua tab di
		// atas. Dibawa apa adanya.
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

	"Kolom \"Status\" menyatakan ANTREAN, bukan status klaim. Ia teks tetap di dalam " +
		"kuerinya — \"Estimation\" untuk seluruh baris tab Admin dan \"Acceptation\" untuk " +
		"seluruh baris tab Teknik — dan tidak membaca satu pun kolom status. Perilakunya " +
		"dipertahankan apa adanya (P-5); yang ditambahkan hanyalah keterangan ini, karena " +
		"kolom bernama \"Status\" yang tidak menyatakan status adalah hal yang wajar " +
		"disalahpahami.",

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
}
