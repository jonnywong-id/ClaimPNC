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
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan — ketiganya
// merujuk nama yang sama.
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

	// Kedua isian berikut TIDAK bersumber dari export — lihat kepala
	// inboxclaimtreatyprop.go. Namanya mengikuti ARTI kolomnya, bukan judulnya: judul
	// "Status Claim ID" menyesatkan (isinya status alur kerja Pega, bukan Status Klaim
	// berkode `1134`–`1166`), dan nama kontrak yang ikut menyesatkan akan menularkan salah
	// arti itu ke setiap pemakainya.
	FieldLastUpdate  = "operator_pengubah"
	FieldClaimStatus = "status_kerja"
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
// "Prop Treaty-in Admin", karena itulah satu-satunya pilihan yang SELALU ada di dropdown
// Pega — `FilterWorkBasket_Act` menambahkannya tanpa syarat, sedangkan pilihan Teknik hanya
// menyusul bila pemanggil anggota antreannya.
const DefaultTab = TabWorkList

// Tab adalah satu antrean kerja pada layar Inbox Claim Treaty Prop.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah teks PILIHAN pada dropdown pemilih antrean.
	//
	// Ia diambil dari `Data Transform/FilterWorkBasket_Act-DT.xml`, yang menyusun daftar
	// pilihan dropdown itu: `CARI2` adalah labelnya, `CARI1` nilainya. Jangan tertukar
	// dengan GridTitle di bawah — keduanya teks yang berbeda di layar yang sama.
	Name string

	// GridTitle adalah judul KONTAINER grid yang digambar sesudah dropdown.
	//
	// Ia teks yang sama sekali berbeda dari Name, dan keduanya tampil bersamaan:
	// dropdown-nya bertuliskan "Prop Treaty-in Admin", sementara grid di bawahnya berjudul
	// "Work List Treatyin Propotional". Menyamakan keduanya — yang sempat dilakukan di
	// sini — membuat judul grid hilang dan pilihan dropdown tidak dikenali pengguna.
	//
	// KOSONG pada tab yang Blocked: tidak ada grid yang digambar di sana.
	GridTitle string

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
	// SELURUHNYA false sejak sumber datanya berpindah ke Report Definition: tidak satu pun
	// dari kedua RD punya filter `pxAssignedOperatorID`. Isian ini DIPERTAHANKAN — bukan
	// dihapus — karena ia yang menjaga agar penyaring itu tidak kembali diam-diam: sekali
	// ada tab yang menyatakannya true, kueri dan penyimpanan memori wajib menyaring.
	//
	// Di ketiga rule Connect-SQL yang dulu dipakai, penyaringnya
	// `PXASSIGNEDOPERATORID = {OperatorID.pyUserIdentifier}`.
	ScopedToCaller bool

	// SupportsSeeAll menyatakan checkbox "See All Claim" berlaku pada tab ini.
	//
	// SELURUHNYA false, dan alasannya mengikuti ScopedToCaller: checkbox yang melepas
	// penyaring tidak punya apa pun untuk dilepas pada tab yang memang tidak menyaring.
	//
	// Di sistem lama ia properti `SearchWorkbasket.CARI43`, dan mencentangnya menukar kueri
	// dari `GetClaimTreaty_SQL` menjadi `GetClaimTreatyAllAdmin_SQL`.
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

	// Judul kedua kolom terakhir ditulis PERSIS seperti di Pega produksi, termasuk
	// "Last update" yang huruf besarnya tidak konsisten dengan tujuh judul di kirinya, dan
	// "Status Claim ID" yang menyebut ID padahal isinya status. `D-13` menetapkan tampilan
	// meniru Pega; merapikan judulnya berarti pengguna mencari kolom yang tidak ada.
	colLastUpdate  = Column{Key: FieldLastUpdate, Title: "Last update"}
	colClaimStatus = Column{Key: FieldClaimStatus, Title: "Status Claim ID"}
)

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutan kolom tiap tab diambil dari urutan sel di
// `Section/InboxClaimTreaty_Section-Section.xml` — bukan dari urutan kolom SQL-nya.
// Keduanya berbeda, dan yang dilihat pengguna adalah yang pertama.
var tabs = []Tab{
	{
		Code: TabWorkList,

		// Teks pilihan dropdown, dari `FilterWorkBasket_Act` langkah 2:
		// `Operator.pxResults(<APPEND>).CARI2 := "Prop Treaty-in Admin"`, dengan
		// `CARI1 := ""` sebagai nilainya.
		Name: "Prop Treaty-in Admin",

		// Salah eja "Propotional" DIPERTAHANKAN. Ia tertulis begitu di section
		// (`<pyValue>Work List Treatyin Propotional</pyValue>`), dan `D-13` menetapkan
		// tampilan meniru Pega supaya pengguna tidak perlu belajar ulang. Membetulkannya
		// menjadi "Proportional" akan membuat grid ini tidak dikenali pengguna yang
		// mencarinya, dan itu harga yang lebih mahal daripada satu huruf yang hilang.
		GridTitle: "Work List Treatyin Propotional",

		Description: "Seluruh penugasan klaim treaty proporsional di entitas ini — " +
			"bukan hanya milik Anda. Report Definition InboxKlaimPropAdmin tidak " +
			"menyaring menurut petugas.",
		Columns: []Column{
			colClaimID, colMasterID, colPolicyNumber, colLossDate,
			colBusinessName, colBusinessSource, colCedingCompany, colInsuredName,
			colLastUpdate, colClaimStatus,
		},
		// KEDUANYA false sejak sumbernya berpindah ke Report Definition.
		//
		// `InboxKlaimPropAdmin` tidak punya satu pun filter selain kondisi join: tidak ada
		// `pxAssignedOperatorID = <pemanggil>`, dan keenam parameternya dideklarasikan
		// tetapi tidak dirujuk di mana pun. Tab ini karena itu TIDAK menyaring pemanggil.
		//
		// Dan karena tidak menyaring, checkbox "See All Claim" tidak punya apa pun untuk
		// dilepas. Menggambarnya tetap akan menjadi kontrol yang tidak mengubah apa pun —
		// lebih buruk daripada kontrol yang tidak ada.
		ScopedToCaller: false,
		SupportsSeeAll: false,
	},
	{
		Code: TabTechnical,

		// Teks pilihan dropdown, dari `FilterWorkBasket_Act` langkah terakhir: baris yang
		// `CARI1`-nya bernilai `"TreatyinPNCTeknik"` dilabeli
		// `CARI2 := "Prop Treaty-in Teknik"`.
		Name: "Prop Treaty-in Teknik",

		// Judul kontainernya di section berakhir dengan satu spasi
		// (`<pyValue>Work Teknik Treatyin </pyValue>`). Spasi itu TIDAK dibawa: ia
		// artefak pengetikan, bukan teks yang dibaca pengguna, dan membawanya hanya
		// membuat pembandingan judul gagal tanpa alasan yang terbaca.
		GridTitle: "Work Teknik Treatyin",

		Description: "Seluruh antrean bersama klaim treaty proporsional — belum diambil " +
			"siapa pun. Report Definition InboxKlaimPropTeknik tidak menyaring menurut " +
			"nama antrean. Di Pega PILIHANNYA hanya muncul bagi petugas yang terdaftar " +
			"sebagai anggota antrean TreatyinPNCTeknik; di sini ia terlihat oleh semua " +
			"(TKT-F3-004).",
		Columns: []Column{
			colClaimID, colMasterID, colPolicyNumber, colLossDate,
			colBusinessName, colBusinessSource, colCedingCompany, colInsuredName,
			// HANYA tab ini yang punya kolom Subjectivity: hanya kueri Teknik yang
			// membawanya.
			{Key: FieldSubjectivity, Title: "Subjectivity"},

			// Kedua kolom terakhir tetap di UJUNG, sesudah Subjectivity, supaya urutan
			// delapan kolom pertama sama persis di kedua tab. Di Pega produksi tab ini
			// tidak terlihat, jadi urutannya di sini ditetapkan agar konsisten — bukan
			// ditiru dari layar yang tidak ada gambarnya.
			colLastUpdate, colClaimStatus,
		},
	},
	{
		Code: TabCommittee,

		// Judul kontainernya di section. Ia TIDAK punya teks pilihan dropdown yang dapat
		// dibaca — lihat BlockedReason.
		Name: "Komite Treaty ASM",

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
			"sehingga tidak ada cara membacanya tanpa menebak. Di Pega ia juga bukan " +
			"pilihan pada dropdown ini melainkan pada DROPDOWN KEDUA yang hanya tampil " +
			"bagi anggota komite, dan daftar pilihan dropdown itu (Operator1.pxResults) " +
			"pun tidak punya rule pembangun di export (R-16).",

		BlockedOwner: "DBA — dibutuhkan DDL DATAPEGA.PC_ASM_FW_GCNMFW_WORK beserta " +
			"pemetaan properti KomiteClaimData ke kolomnya. Ditambah Tim Pega, untuk " +
			"rule yang mengisi Operator1.pxResults.",
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
	"Kolom bisnis dibaca dari KOLOM TEREKSPOS pada objek kerja, bukan dari dokumen JSON " +
		"POOLDATA.JSON_KLAIM. Itu mengikuti Report Definition InboxKlaimPropAdmin dan " +
		"InboxKlaimPropTeknik yang memasok kedua grid di Pega produksi. Kedua sumber itu " +
		"dapat berbeda isinya, sehingga baris yang dulu terisi dapat menjadi kosong dan " +
		"sebaliknya.",

	"Antrean \"Prop Treaty-in Admin\" menampilkan SELURUH penugasan klaim treaty di " +
		"entitas ini, bukan hanya milik Anda. Report Definition-nya tidak menyaring " +
		"menurut petugas, dan karena itu checkbox \"See All Claim\" dihapus — ia tidak " +
		"lagi punya penyaring untuk dilepas.",

	"Antrean \"Prop Treaty-in Teknik\" menampilkan seluruh antrean bersama klaim treaty, " +
		"tidak lagi disaring ke akun TreatyinPNCTeknik saja. Report Definition-nya tidak " +
		"menyaring menurut nama antrean.",

	"Kolom \"Subjectivity\" pada antrean Teknik SELALU kosong. Report Definition " +
		"mengambilnya dari properti IsSubjectivity, tetapi nama kolom terekspos properti " +
		"itu tidak dapat ditemukan di export — dan menebaknya akan menggagalkan seluruh " +
		"kueri, bukan satu kolom.",

	"Kolom \"Date Of Loss\" pada tab Work Teknik Treatyin kini TERISI. Di sistem lama ia " +
		"selalu kosong: kueri antrean teknik menaruh tanggalnya di kolom CARI13 " +
		"sementara sel gridnya membaca CARI10. Perbaikan ini disetujui Work Owner " +
		"2026-09-21 sebagai selisih terencana P-5.",

	"Tombol \"Create Claim Treaty Prop\" belum membuat klaim. Selama Pega dan sistem " +
		"baru berjalan berdampingan, tabel objek kerja hanya boleh ditulis satu sistem " +
		"(P-1), dan tabel itu masih dimiliki Pega. Buat klaim treaty baru lewat Pega.",

	"Kolom \"Last update\" dan \"Status Claim ID\" dibaca dari PXUPDATEOPERATOR dan " +
		"PYSTATUSWORK pada tabel objek kerja — persis seperti Report Definition " +
		"menyebutnya (WorkPage.pxUpdateOperator dan WorkPage.pyStatusWork).",

	"\"Status Claim ID\" berisi status ALUR KERJA Pega (\"New\", \"Pending\", …), bukan " +
		"Status Klaim berkode 1134–1166 milik master V_STS_CLAIM. Judulnya menyesatkan " +
		"sejak di Pega dan dipertahankan apa adanya (D-13); yang tidak dipertahankan " +
		"adalah salah artinya — nilainya tidak diterjemahkan ke label master mana pun.",

	"Pilihan \"Prop Treaty-in Teknik\" SELALU terlihat. Di Pega ia hanya muncul bagi " +
		"petugas yang terdaftar sebagai anggota antrean TreatyinPNCTeknik — " +
		"FilterWorkBasket_Act menyusunnya dari OperatorID.pyWorkBasketList. Penugasan " +
		"operator ke antrean itu tidak ada di basis data maupun di export, sehingga " +
		"keanggotaannya belum dapat diperiksa (TKT-F3-004). Melihat pilihannya bukan " +
		"berarti dapat mengambil pekerjaannya: layar ini membaca saja.",

	"Pilihan \"Choose\" menampilkan antrean yang sama dengan \"Prop Treaty-in Admin\", " +
		"bukan layar kosong. Itu perilaku Pega, bukan cacat: keduanya bernilai sama " +
		"(CARI1 kosong), sehingga pemeriksaan yang memilih antrean teknik gagal pada " +
		"keduanya dan yang tampil adalah antrean admin.",

	"Nomor klaim pada kolom pertama menjadi TAUTAN ke layar Outstanding Claim, " +
		"menggantikan tombol \"Lihat Detail Klaim\" di kolom terakhir. Itu mengikuti Pega " +
		"produksi, tempat Claim ID sendirilah yang menjalankan Flow Action " +
		"OutstandingClaim.",

	"Urutan baris ditetapkan tegas menurut tanggal penugasan terbaru. Kueri antrean " +
		"teknik di sistem lama tidak mengurutkan hasilnya sama sekali — dapat dibiarkan " +
		"selama seluruh baris ditarik sekaligus, tetapi membuat satu baris muncul di dua " +
		"halaman begitu hasilnya dipotong per halaman.",
}
