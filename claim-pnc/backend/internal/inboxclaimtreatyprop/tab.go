package inboxclaimtreatyprop

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke tab —
// `BusinessName` berjudul "Business Name" pada tab worklist dan "Class Of Business" pada
// tab komite, persis seperti di `Section/InboxClaimTreaty_Section-Section.xml`.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`),
	// termasuk salah ejanya — lihat TabWorkList.
	Title string
}

// Nama field JSON pada satu baris pekerjaan.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan —
// ketiganya merujuk nama yang sama.
const (
	FieldClaimID        = "claim_id"
	FieldMasterID       = "id_master"
	FieldPolicyNumber   = "no_polis"
	FieldLossDate       = "tanggal_kejadian"
	FieldBusinessName   = "nama_bisnis"
	FieldBusinessSource = "sumber_bisnis"
	FieldCedingCompany  = "ceding_co"
	FieldInsuredName    = "nama_tertanggung"
	FieldSubjectivity   = "subjectivity"
)

// Kode tab.
//
// # Kenapa angka, dan kenapa TIDAK memakai nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Yang dipakainya adalah dua
// pemeriksaan yang sama sekali berbeda jenisnya — `InputData.CARI13 == 'tampil'` yang
// berarti "pemanggil anggota komite", dan `SearchWorkbasket.CARI1 == 'TreatyinPNCTeknik'`
// yang berarti "pemanggil memegang akun antrean teknik". Keduanya KEADAAN, bukan pilihan
// pengguna; tidak ada satu pun nilai yang dapat dipertahankan sebagai kode tab.
//
// Karena itu kodenya ditetapkan di sini, berurutan, dan diperlakukan sebagai kontrak modul
// ini sendiri. Penelusuran balik ke export ditempuh lewat nama kuerinya — yang disebut
// lengkap pada setiap tab di bawah — bukan lewat kodenya.
const (
	TabWorkList  = "1"
	TabTechnical = "2"
	TabCommittee = "3"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// "Work List Treatyin Propotional", karena hanya tab inilah yang isinya MILIK pemanggil.
// Membuka layar pada antrean bersama akan menampilkan pekerjaan orang lain lebih dulu.
const DefaultTab = TabWorkList

// Tab adalah satu antrean kerja pada layar Inbox Claim Treaty Prop.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`.
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena judul seperti "Work Teknik Treatyin"
	// tidak memberi tahu apa pun tentang isinya.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti di Pega.
	//
	// KOSONG pada tab yang Blocked: tidak ada grid yang digambar di sana.
	Columns []Column

	// ScopedToCaller menyatakan kuerinya menyaring menurut pengguna yang login.
	//
	// Hanya tab pertama begitu, dan itulah yang membuatnya "antrean saya". Penyaringnya
	// `PXASSIGNEDOPERATORID = {OperatorID.pyUserIdentifier}` pada
	// `RDB List/GetClaimTreaty_SQL-SQL.xml`.
	ScopedToCaller bool

	// SupportsSeeAll menyatakan checkbox "See All Claim" berlaku pada tab ini.
	//
	// Di sistem lama ia properti `SearchWorkbasket.CARI43`, dan mencentangnya menukar
	// kueri dari `GetClaimTreaty_SQL` menjadi `GetClaimTreatyAllAdmin_SQL` — yang
	// menghapus penyaring operator. Ia HANYA berlaku pada tab pertama; pada tab Teknik
	// kedua kueri itu tidak pernah dipanggil.
	SupportsSeeAll bool

	// Blocked menyatakan tab ini digambar tetapi belum dapat diisi.
	//
	// Ia BUKAN tab yang disembunyikan. Keputusan Work Owner 2026-09-21: tabnya tetap
	// tampak supaya pengguna tahu fiturnya ada dan terhalang, bukan mengira ia hilang.
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
	colClaimID        = Column{Key: FieldClaimID, Title: "Claim ID"}
	colMasterID       = Column{Key: FieldMasterID, Title: "ID Master"}
	colPolicyNumber   = Column{Key: FieldPolicyNumber, Title: "Policy No"}
	colLossDate       = Column{Key: FieldLossDate, Title: "Date Of Loss"}
	colBusinessName   = Column{Key: FieldBusinessName, Title: "Business Name"}
	colBusinessSource = Column{Key: FieldBusinessSource, Title: "Source Of Business"}
	colCedingCompany  = Column{Key: FieldCedingCompany, Title: "Ceding Co Name"}
	colInsuredName    = Column{Key: FieldInsuredName, Title: "Insured Name"}
)

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutan kolom tiap tab diambil dari urutan sel di
// `Section/InboxClaimTreaty_Section-Section.xml` — bukan dari urutan kolom SQL-nya.
// Keduanya berbeda, dan yang dilihat pengguna adalah yang pertama.
var tabs = []Tab{
	{
		Code: TabWorkList,

		// Salah eja "Propotional" DIPERTAHANKAN. Ia tertulis begitu di section
		// (`<pyValue>Work List Treatyin Propotional</pyValue>`), dan `D-13` menetapkan
		// tampilan meniru Pega supaya pengguna tidak perlu belajar ulang. Membetulkannya
		// menjadi "Proportional" akan membuat tab ini tidak dikenali pengguna yang
		// mencarinya, dan itu harga yang lebih mahal daripada satu huruf yang hilang.
		Name: "Work List Treatyin Propotional",

		Description: "Klaim treaty proporsional yang ditugaskan kepada Anda. " +
			"Centang \"See All Claim\" untuk melihat penugasan seluruh petugas.",
		Columns: []Column{
			colClaimID, colMasterID, colPolicyNumber, colLossDate,
			colBusinessName, colBusinessSource, colCedingCompany, colInsuredName,
		},
		ScopedToCaller: true,
		SupportsSeeAll: true,
	},
	{
		Code: TabTechnical,

		// Judulnya di section berakhir dengan satu spasi
		// (`<pyValue>Work Teknik Treatyin </pyValue>`). Spasi itu TIDAK dibawa: ia
		// artefak pengetikan, bukan teks yang dibaca pengguna, dan membawanya hanya
		// membuat pembandingan judul gagal tanpa alasan yang terbaca.
		Name: "Work Teknik Treatyin",

		Description: "Antrean bersama PIC Teknik treaty — belum diambil siapa pun.",
		Columns: []Column{
			colClaimID, colMasterID, colPolicyNumber, colLossDate,
			colBusinessName, colBusinessSource, colCedingCompany, colInsuredName,
			// HANYA tab ini yang punya kolom Subjectivity: hanya kueri Teknik yang
			// membawanya.
			{Key: FieldSubjectivity, Title: "Subjectivity"},
		},
	},
	{
		Code:        TabCommittee,
		Name:        "Komite Treaty ASM",
		Description: "Antrean persetujuan komite treaty.",

		// Tidak ada kolom, karena tidak ada grid yang digambar. Menyebut kolomnya di sini
		// akan membuat layar menggambar tabel kosong yang terbaca sebagai "tidak ada
		// pekerjaan" — padahal yang benar adalah "belum dapat dibaca".
		Blocked: true,

		BlockedReason: "Antrean komite treaty belum dapat dibaca sistem baru. Kedua " +
			"sumbernya di Pega (WorkListKomite2 dan InboxKomiteTreaty_RD) mengambil " +
			"nilai dari properti di dalam BLOB objek kerja — antara lain No Klaim, " +
			"Insured, dan Ceding Co — bukan dari kolom tabel. Tidak satu pun nama itu " +
			"muncul di seluruh SQL sistem lama, dan DDL tabelnya belum tersedia (R-08), " +
			"sehingga tidak ada cara membacanya tanpa menebak.",

		BlockedOwner: "DBA — dibutuhkan DDL DATAPEGA.PC_ASM_FW_GCNMFW_WORK beserta " +
			"pemetaan properti KomiteClaimData ke kolomnya.",
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
	"Kolom \"Date Of Loss\" pada tab Work Teknik Treatyin kini TERISI. Di sistem lama ia " +
		"selalu kosong: kueri antrean teknik menaruh tanggalnya di kolom CARI13 " +
		"sementara sel gridnya membaca CARI10. Perbaikan ini disetujui Work Owner " +
		"2026-09-21 sebagai selisih terencana P-5.",

	"Tombol \"Create Claim Treaty Prop\" belum membuat klaim. Selama Pega dan sistem " +
		"baru berjalan berdampingan, tabel objek kerja hanya boleh ditulis satu sistem " +
		"(P-1), dan tabel itu masih dimiliki Pega. Buat klaim treaty baru lewat Pega.",

	"Urutan baris ditetapkan tegas menurut tanggal penugasan terbaru. Kueri antrean " +
		"teknik di sistem lama tidak mengurutkan hasilnya sama sekali — dapat dibiarkan " +
		"selama seluruh baris ditarik sekaligus, tetapi membuat satu baris muncul di dua " +
		"halaman begitu hasilnya dipotong per halaman.",
}
