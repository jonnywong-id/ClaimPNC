package inboxmanagerreceivepucl

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke tab —
// `CaseID` berjudul "CaseID" pada kedua tab Receive dan "Nomor Case" pada tab RCL/PUCL,
// persis seperti di `Section/InboxManagerReceive_Section-Section.xml`.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`).
	Title string
}

// Nama field JSON pada satu baris pekerjaan.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldCaseID               = "no_case"
	FieldPolicyNumber         = "no_polis"
	FieldClaimNumber          = "no_klaim_pnc"
	FieldInsuredName          = "nama_tertanggung"
	FieldLossDate             = "tanggal_kejadian"
	FieldClaimType            = "jenis_klaim"
	FieldSenderName           = "nama_pengirim"
	FieldDocumentReceivedDate = "tanggal_terima_dokumen"
	FieldDocumentSheetCount   = "jumlah_lembar_dokumen"
	FieldInboxEntryAt         = "tanggal_masuk_inbox"
	FieldAnalystNote          = "deskripsi_analyst"
	FieldTrack                = "rcl_pucl"
	FieldTrackStatus          = "status_rcl_pucl"
	FieldLetterPrintedAt      = "tanggal_cetak_surat"
	FieldClaimAge             = "lama_klaim"
	FieldExpiryStatus         = "status_kadaluarsa"
)

// Kode tab.
//
// # Kenapa TIGA, padahal layar lama punya DUA judul tab
//
// Karena tab "Receive" pada layar lama memuat DUA grid bertumpuk, bukan satu.
// `Section/InboxManagerReceive_Section-Section.xml` menempatkan grid `ManagementRecieveView`
// dua kali di dalam satu tab — yang pertama dijalankan dengan `<Position1>"PA"</Position1>`
// dan yang kedua dengan `<Position1>"NONMBU"</Position1>` — dan keduanya ber-`pyVisible`
// `ALWAYS`, sehingga keduanya tampil bersamaan.
//
// Kedua grid itu TIDAK punya judul apa pun di sana: penelusuran seluruh section tidak
// menemukan satu pun label di antara keduanya. Pengguna karena itu melihat dua tabel yang
// kolomnya identik, berurutan ke bawah, tanpa satu pun tanda mana yang mana.
//
// Di sini keduanya menjadi tab yang dapat dipilih dan DIBERI JUDUL. Ini perubahan yang
// disadari, dan pola yang sama sudah ditempuh modul Inbox Claim Treaty Non Prop — di sana
// tiga kontainer yang dipilih oleh keadaan pemanggil menjadi tiga tab yang dapat diklik.
// Dua alasannya:
//
//   - Paginasi. Masing-masing grid punya halamannya sendiri di sistem lama
//     (`pyPageSize` 50, penomoran Numeric). Dua tabel berhalaman yang bertumpuk pada satu
//     layar menghasilkan dua penomoran yang mudah tertukar.
//   - Kejujuran. Tabel tanpa judul yang isinya berbeda adalah cacat tampilan yang dibawa
//     tanpa perlu — dan justru di layar ini akibatnya nyata, karena yang membedakan keduanya
//     adalah lini bisnis klaimnya.
//
// # Kenapa angka, dan kenapa bukan nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Yang dipakainya adalah nilai
// parameter Report Definition dan judul kontainer, dan keduanya bukan kontrak yang layak
// dibawa ke API. Kodenya karena itu ditetapkan di sini, berurutan, dan diperlakukan sebagai
// kontrak modul ini sendiri. Penelusuran balik ke export ditempuh lewat nama Report
// Definition-nya — disebut lengkap pada setiap tab di bawah — bukan lewat kodenya.
const (
	TabReceivePA     = "1"
	TabReceiveNonMBU = "2"
	TabRCLPUCL       = "3"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Grid PA, karena itulah grid PERTAMA pada tab "Receive" di layar lama — ia berada di
// posisi paling atas section, dan itulah yang lebih dulu terbaca pengguna.
const DefaultTab = TabReceivePA

// Tab adalah satu antrean kerja pada layar Inbox Manager Receive / PUCL.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`.
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena dua dari tiga grid di sana bahkan tidak
	// punya judul sama sekali.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	Columns []Column

	// ClaimType menyatakan Jenis Klaim yang disaring tab ini.
	//
	// Kosong pada tab RCL/PUCL, yang tidak menyaring menurut lini bisnis sama sekali.
	// Nilainya BUKAN sekadar keterangan: ia yang memilih kueri di repo/sqlstore dan yang
	// menyaring di repo/memory, sehingga keduanya tidak dapat berselisih.
	ClaimType string

	// FromWorkbasket menyatakan barisnya diambil dari antrean BERSAMA
	// (DATAPEGA.PC_ASSIGN_WORKBASKET), bukan dari penugasan per orang
	// (DATAPEGA.PC_ASSIGN_WORKLIST).
	//
	// Hanya tab RCL/PUCL begitu. Pembedaan ini bukan kerapian: kedua tabel itu berbeda, dan
	// kueri yang membaca tabel yang salah mengembalikan nol baris tanpa satu pun galat.
	FromWorkbasket bool

	// Blocked menyatakan tab ini digambar tetapi belum dapat diisi.
	//
	// Tidak ada tab yang terhalang pada layar ini hari ini. Isian tetap disediakan karena
	// bentuk tab dipakai bersama lapisan transport dan layar, dan ketiadaannya akan membuat
	// penambahan tab terhalang kelak menuntut perubahan kontrak API.
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

// Kolom kedua tab Receive, disusun sekali supaya judulnya tidak dapat berbeda antar keduanya
// tanpa disengaja.
//
// # Urutan dan judulnya dari mana
//
// Urutannya dari `pyListFields` pada `Report Definition/ManagementRecieveView-RD.xml`,
// nomor 1 sampai 9. Kedua isian terakhir RD itu — `newAssignPage.pxAssignedOrgUnit` dan
// `newAssignPage.pyPosition` — TIDAK ikut: keduanya tidak digambar satu sel pun di section,
// dan keduanya hanya ada di RD karena filter unit organisasi yang tidak pernah dipakai.
//
// Judulnya dari `<pyCaption …>` pada section, BUKAN dari `pyFieldLabel` pada RD. Keduanya
// berbeda di tiga tempat, dan yang dibaca pengguna adalah yang di section (`D-13`):
//
//	isian                          pyFieldLabel (RD)        pyCaption (section)
//	.ReceiveDocument.PolicyNo      PolicyNo                 No Polis
//	.ReceiveDocument.TypeOfClaim   Tipe Klaim               Jenis Klaim
//	.ReceiveDocument.NumberOfDoc…  Jumlah Dokumen           Jumlah Lembar Dokumen
var receiveColumns = []Column{
	{Key: FieldCaseID, Title: "CaseID"},
	{Key: FieldPolicyNumber, Title: "No Polis"},
	{Key: FieldClaimNumber, Title: "PNC CaseID"},
	{Key: FieldInsuredName, Title: "Nama Tertanggung"},
	{Key: FieldLossDate, Title: "Tanggal Kejadian"},
	{Key: FieldClaimType, Title: "Jenis Klaim"},
	{Key: FieldSenderName, Title: "Nama Pengirim"},
	{Key: FieldDocumentReceivedDate, Title: "Tanggal Terima Dokumen"},
	{Key: FieldDocumentSheetCount, Title: "Jumlah Lembar Dokumen"},
}

// Kolom tab RCL/PUCL.
//
// Urutannya dari `pyListFields` pada `Report Definition/InboxRCLPUCL_RD-RD.xml`; judulnya
// dari `<pyCaption …>` pada section.
//
// # Satu isian RD yang TIDAK digambar, dan kenapa
//
// RD memuat `.ClaimData.StatusClaim` (`STATUSCLAIM_1`) berdampingan dengan
// `.ClaimData.PUCLStatus.StatusKlaim` (`STATUSKLAIM_1`). Section tidak punya caption untuk
// yang pertama, dan tidak ada sel yang menggambarnya. Ia karena itu tidak dibawa.
//
// Perhatikan kedua nama kolom itu hanya berbeda satu huruf dan artinya berbeda jauh: yang
// digambar adalah status jalur RCL/PUCL, bukan Status Klaim ber-33 kode `1134`–`1166`
// (`R-06`). Menukarnya tidak menghasilkan satu pun galat.
var rclpuclColumns = []Column{
	{Key: FieldCaseID, Title: "Nomor Case"},
	{Key: FieldPolicyNumber, Title: "No Polis"},
	{Key: FieldInsuredName, Title: "Nama Tertanggung"},
	{Key: FieldInboxEntryAt, Title: "Tanggal Masuk Inbox"},
	{Key: FieldAnalystNote, Title: "Deskripsi Analyst"},
	{Key: FieldTrack, Title: "RCL/PUCL"},
	{Key: FieldTrackStatus, Title: "Status RCL/PUCL"},
	{Key: FieldLetterPrintedAt, Title: "Tanggal Cetak Surat"},
	{Key: FieldClaimAge, Title: "Lama Klaim"},
	{Key: FieldExpiryStatus, Title: "Status Kadaluarsa"},
}

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutannya mengikuti urutan grid di section: grid PA (offset ~37.000), grid NON-MBU
// (~170.000), lalu grid RCL/PUCL (~317.000).
var tabs = []Tab{
	{
		Code: TabReceivePA,

		// Judulnya tidak ada di layar lama — kedua grid Receive di sana tanpa label.
		// Yang dipakai adalah judul tabnya ("Receive", dari `<pyTitle>Receive</pyTitle>`)
		// ditambah lini bisnis yang disaringnya, dalam ejaan yang sama dengan nilai
		// parameter Report Definition-nya.
		Name: "Receive PA",

		Description: "Berkas penerimaan dokumen klaim Personal Accident yang masih punya " +
			"penugasan terbuka. Ditampilkan untuk seluruh petugas, bukan hanya milik Anda.",

		Columns:   receiveColumns,
		ClaimType: ClaimTypePA,
	},
	{
		Code: TabReceiveNonMBU,
		Name: "Receive NONMBU",

		Description: "Berkas penerimaan dokumen klaim di luar Personal Accident yang masih " +
			"punya penugasan terbuka. Ditampilkan untuk seluruh petugas, bukan hanya milik " +
			"Anda.",

		Columns:   receiveColumns,
		ClaimType: ClaimTypeNonMBU,
	},
	{
		Code: TabRCLPUCL,

		// Judul ini ADA di layar lama, apa adanya:
		// `Section/InboxManagerReceive_Section-Section.xml` — `<pyTitle>RCL/PUCL</pyTitle>`.
		Name: "RCL/PUCL",

		Description: "Klaim yang ditolak (RCL) atau diproses ulang (PUCL) dan masih " +
			"menunggu keputusan. Antrean bersama, bukan penugasan per orang.",

		Columns:        rclpuclColumns,
		FromWorkbasket: true,
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
// di komentar akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan
// layar baru dengan Pega berdampingan.
//
// Ia juga yang dipakai saat uji kesetaraan gerbang 1: setiap selisih WAJIB dapat dipetakan
// ke salah satu butir `P-5`, atau dinyatakan sebagai bug (`D-54`). Butir di bawah adalah
// pemetaan itu, sudah tertulis di muka alih-alih dicari setelah selisihnya muncul.
var PlannedDifferences = []string{
	"Kolom \"Jenis Klaim\" diturunkan dari Group Panel, bukan dibaca dari isian aslinya. " +
		"Di Pega, kedua tabel Receive dipisahkan properti `.ReceiveDocument.TypeOfClaim`, " +
		"yang ditandai `unexposed` di Report Definition-nya — ia hidup di dalam blob Pega " +
		"dan tidak punya kolom basis data, sehingga tidak dapat disaring SQL. Penggantinya " +
		"Group Panel: `002` berarti PA, selain itu NONMBU. Berkas yang Group Panel-nya " +
		"kosong tidak muncul di tab mana pun — sama seperti berkas tanpa Jenis Klaim di " +
		"layar lama. Keputusan Work Owner 2026-09-22 sebagai selisih terencana P-5.",

	"Kedua daftar Receive menjadi DUA TAB, bukan dua tabel bertumpuk. Di Pega keduanya " +
		"berada di dalam satu tab \"Receive\", tampil bersamaan, dan tidak punya judul " +
		"satu pun — sehingga tidak ada tanda mana yang PA dan mana yang bukan. Isi, kolom, " +
		"dan urutan kolomnya sama persis; yang berubah hanya cara keduanya dipisahkan di " +
		"layar.",

	"Tab RCL/PUCL menyaring antrean bersama `RCLPUCL`. Report Definition yang memasok " +
		"grid itu di Pega (`InboxRCLPUCL_RD`) tidak punya gabungan sama sekali dan hanya " +
		"menyaring status kerja, sementara parameternya bernama `assign` dideklarasikan " +
		"tetapi tidak dipakai. Ditiru apa adanya, tab ini akan menampilkan SELURUH klaim " +
		"yang belum selesai. Penyaring yang dipakai di sini diambil dari dua kueri Pega " +
		"pada domain yang sama — `CountKlaimPUCL` dan `ReminderPUCL` — yang keduanya " +
		"menyaring `PXASSIGNEDOPERATORID = 'RCLPUCL'`.",

	"Penyaring unit organisasi tidak dibawa. Report Definition Receive menyaring " +
		"`pxAssignedOrgUnit` menurut parameter `OrgUnit`, tetapi layar Pega mengirimkannya " +
		"KOSONG dan tidak ada satu pun activity di export yang mengisinya — sehingga " +
		"penyaring itu tidak pernah benar-benar berlaku. Keputusan Work Owner 2026-09-22: " +
		"ikuti export apa adanya.",

	"Kolom \"Jumlah Lembar Dokumen\" selalu kosong. Tidak ada satu pun kolom basis data " +
		"untuknya di seluruh export, dan tabel penerimaan dokumen tidak menyimpannya. " +
		"Kolomnya tetap digambar, bukan dihilangkan, supaya isian yang belum terbawa " +
		"terlihat alih-alih tersamar sebagai layar yang sudah setara.",

	"\"Nama Pengirim\" dan \"Tanggal Terima Dokumen\" dibaca dari tabel " +
		"POOLDATA.T_CLAIM_RECIVEDCLAIM. Keduanya tidak punya kolom pada objek kerja Pega. " +
		"Tabel itu diisi procedure PROCINSERTDATARECIVEDKLAIM dengan kunci yang sama, " +
		"tetapi TIDAK PERNAH DIBACA sistem lama — sehingga kelengkapan isinya belum " +
		"terverifikasi. Baris tanpa pasangan di sana tetap muncul dengan kedua kolom " +
		"kosong, bukan hilang dari daftar.",

	"Urutan baris ditetapkan tegas: yang terbaru masuk lebih dulu. Ketiga grid di sistem " +
		"lama tidak menetapkan urutan sama sekali — dapat dibiarkan selama seluruh baris " +
		"ditarik sekaligus, tetapi membuat satu baris muncul di dua halaman begitu hasilnya " +
		"dipotong per halaman.",

	"Daftar dipotong per halaman di basis data. Grid Pega memotongnya di `pyMaxRecords` " +
		"500 setelah seluruh barisnya ditarik; di sini halamannya dipotong sebelum baris " +
		"meninggalkan basis data, dan jumlah seluruhnya tetap dihitung tepat.",
}
