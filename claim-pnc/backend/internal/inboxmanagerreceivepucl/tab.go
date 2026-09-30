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
// # DUA tab, sama dengan layar lama
//
// `Section/InboxManagerReceive_Section-Section.xml` memuat tepat dua judul tab —
// `<pyTitle>Receive</pyTitle>` dan `<pyTitle>RCL/PUCL</pyTitle>` — dan modul ini mengikutinya
// (`D-13`).
//
// # Kenapa tab "Receive" SATU, padahal di baliknya ada dua grid
//
// Di layar lama, tab "Receive" menempatkan grid `ManagementRecieveView` DUA KALI: yang
// pertama dijalankan dengan `<Position1>"PA"</Position1>`, yang kedua dengan
// `<Position1>"NONMBU"</Position1>`. Keduanya ber-`pyVisible` `ALWAYS`, sehingga keduanya
// tampil bersamaan, bertumpuk ke bawah, dan TIDAK punya satu pun judul di antaranya.
//
// Versi pertama modul ini memecahnya menjadi dua tab bernama "Receive PA" dan
// "Receive NONMBU". Itu dicabut atas keputusan Work Owner 2026-09-30: yang dibaca pengguna
// adalah satu tab "Receive", dan memecahnya membuat petugas yang membandingkan kedua layar
// berdampingan mencari tab yang tidak ada di Pega.
//
// Yang menggantikan pembedaannya adalah kolom **Jenis Klaim**, yang memang sudah digambar
// grid dan memang berisi "PA" atau "NONMBU". Jadi pembedaannya tidak hilang — ia pindah dari
// "tabel yang mana" menjadi "isi kolom yang mana", dan justru dapat diurutkan.
//
// Himpunan barisnya TIDAK berubah karenanya; lihat catatan `list_receive` di
// repo/sqlstore/inboxmanagerreceivepucl.sql.
//
// # Kenapa angka, dan kenapa bukan nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Yang dipakainya adalah nilai
// parameter Report Definition dan judul kontainer, dan keduanya bukan kontrak yang layak
// dibawa ke API. Kodenya karena itu ditetapkan di sini, berurutan, dan diperlakukan sebagai
// kontrak modul ini sendiri. Penelusuran balik ke export ditempuh lewat nama Report
// Definition-nya — disebut lengkap pada setiap tab di bawah — bukan lewat kodenya.
const (
	TabReceive = "1"
	TabRCLPUCL = "2"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// "Receive", karena itulah tab PERTAMA pada layar lama — ia berada di posisi paling atas
// kontainer tabnya, dan itulah yang lebih dulu terbaca pengguna.
const DefaultTab = TabReceive

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

	// OpensReceiveDocument menyatakan mengklik nomor case pada tab ini membuka LAYAR KERJA
	// penerimaan dokumen — flow action `InputReceiveDocument`.
	//
	// Hanya tab Receive begitu, dan pembedaannya bukan kerapian: kedua tab membuka layar
	// kerja yang BERBEDA, karena kelas objek kerjanya berbeda. Lihat ReceiveDocument.
	OpensReceiveDocument bool

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

// tabs adalah kedua tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutannya mengikuti urutan kontainer tab di section: "Receive" lebih dulu, lalu "RCL/PUCL".
var tabs = []Tab{
	{
		Code: TabReceive,

		// Judul ini ADA di layar lama, apa adanya:
		// `Section/InboxManagerReceive_Section-Section.xml` — `<pyTitle>Receive</pyTitle>`.
		Name: "Receive",

		Description: "Berkas penerimaan dokumen klaim yang masih punya penugasan terbuka. " +
			"Ditampilkan untuk seluruh petugas, bukan hanya milik Anda. Kolom " +
			"\"Jenis Klaim\" memisahkan Personal Accident dari lini lainnya.",

		Columns:              receiveColumns,
		OpensReceiveDocument: true,
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

	"Kedua daftar Receive digabung menjadi SATU tabel di dalam satu tab \"Receive\", sama " +
		"dengan judul tab di layar lama. Di Pega keduanya adalah dua grid bertumpuk yang " +
		"tampil bersamaan dan tidak punya judul satu pun, sehingga tidak ada tanda mana " +
		"yang Personal Accident dan mana yang bukan; yang membedakannya kini kolom " +
		"\"Jenis Klaim\", yang memang sudah ada di kedua grid itu. Himpunan barisnya sama " +
		"persis: penyaring gabungannya adalah gabungan tepat dari kedua penyaring lama, " +
		"sehingga berkas tanpa Group Panel tetap tidak muncul — sama seperti di Pega. Yang " +
		"berubah hanya satu hal yang terlihat: baris PA dan NONMBU kini berbagi satu " +
		"penomoran halaman, bukan dua.",

	"Mengklik nomor case di tab \"Receive\" membuka layar kerja penerimaan dokumen sebagai " +
		"halaman tersendiri, dan tab serta nomor halaman antrean ikut ke alamatnya supaya " +
		"tombol kembali mendarat di tempat yang sama. Itu yang terjadi di Pega: sel nomor " +
		"case memang TAUTAN, dan mengkliknya menjalankan `SetAssignmentInboxReceive_act` " +
		"dengan `kunci = .pzInsKey` lalu Open Assignment — yang membuka flow action " +
		"`InputReceiveDocument`. Versi pertama modul ini keliru di sini: nomor case digambar " +
		"sebagai teks biasa, dan di ujung baris ditambahkan kolom tombol \"Lihat Detail\" " +
		"yang tidak ada di Pega sama sekali. Kolom itu dihapus.",

	"Tab \"RCL/PUCL\" TIDAK punya tautan pada nomor case-nya. Di layar lama pun tidak: " +
		"perilaku klik hanya dipasang pada kedua grid Receive. Nomor case di tab itu karena " +
		"itu digambar sebagai teks biasa.",

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
