package inboxmanageradmin

import "strings"

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah supaya judul dapat mengikuti layar Pega apa adanya (`D-13`)
// tanpa memaksa nama isian ikut menyesatkan.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom, mengikuti `<pyCaption …>` pada section apa adanya.
	Title string
}

// Nama field JSON pada satu baris antrean.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldCaseID         = "id"
	FieldPolicyNumber   = "no_polis"
	FieldInsuredName    = "nama_tertanggung"
	FieldBusinessName   = "nama_bisnis"
	FieldBusinessSource = "nama_sumber_bisnis"
	FieldClaimStatus    = "status_klaim"
	FieldRegisteredAt   = "tanggal_pendaftaran"
	FieldClaimElapsed   = "lama_waktu_klaim"
	FieldAdminName      = "admin_pnc"
)

// Unit organisasi penugasan yang menjadi parameter `OrgUnit` Report Definition.
//
// # Nilainya kontrak, bukan gaya penulisan
//
// Ketiganya dibandingkan langsung dengan isi kolom `PXASSIGNEDORGUNIT` pada
// POOLDATA.T_CLAIMLIST_ADMIN — kolom yang diminta ditambahkan lewat `migrations/0005`
// tahap 1 saat modul ini pindah ke tabel itu — dan ejaannya diambil apa adanya dari
// `Section/InboxManagerAdmin_Section-Section.xml`: `<OrgUnit>"AdminPNC"</OrgUnit>`,
// `"AdminPA"`, dan `"AdminTRAVEL"`. Perhatikan ketidakseragamannya — dua yang pertama
// memakai huruf besar hanya di awal kata, yang ketiga seluruhnya huruf besar. Merapikannya
// akan membuat ketiga tab mengembalikan nol baris tanpa satu pun pesan galat.
const (
	OrgUnitNonMBU = "AdminPNC"
	OrgUnitPA     = "AdminPA"
	OrgUnitTravel = "AdminTRAVEL"
)

// Lini bisnis yang membuka sebuah tab — padanan `OperatorID.pyPosition` sistem lama.
//
// Nilainya diambil dari `pyContainerVisibleWhen` ketiga kontainer grid di section:
//
//	OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'
//	OperatorID.pyPosition = 'PA'     || OperatorID.pyOrgUnit = 'Development'
//	OperatorID.pyPosition = 'TRAVEL' || OperatorID.pyOrgUnit = 'Development'
//
// Ketiganya dibandingkan dengan `M_LOGIN_PNC.LINE_BUSINESS`, dan ejaannya sama persis
// dengan nilai yang dipakai modul Inbox Outstanding (`inboxoutstanding.LineNonMBU` dan
// seterusnya) — kolom yang sama, domain nilai yang sama.
const (
	LineNonMBU = "NONMBU"
	LinePA     = "PA"
	LineTravel = "TRAVEL"
)

// DevelopmentOrgUnit adalah unit organisasi yang membuka KETIGA tab sekaligus.
//
// Ia bukan aturan bisnis melainkan pintu pengembang, dan di sistem lama pun begitu: ketiga
// `pyContainerVisibleWhen` menyebutnya sebagai alternatif kedua setelah jabatan.
const DevelopmentOrgUnit = "Development"

// Kode tab.
//
// # Kenapa angka, dan kenapa bukan nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Yang dipakainya adalah nilai
// parameter Report Definition (`OrgUnit`) dan kondisi tampil per kontainer, dan keduanya
// bukan kontrak yang layak dibawa ke API — yang pertama bahkan nama unit organisasi yang
// dapat berubah tanpa mengubah arti tabnya.
//
// Kodenya karena itu ditetapkan di sini, berurutan seperti urutan kontainer di section, dan
// diperlakukan sebagai kontrak modul ini sendiri. Penelusuran balik ke export ditempuh lewat
// OrgUnit yang disebut pada setiap tab di bawah, bukan lewat kodenya.
const (
	TabNonMBU = "1"
	TabPA     = "2"
	TabTravel = "3"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Non-MBU, karena itulah kontainer PERTAMA di section — ia berada di posisi paling atas
// (offset ~88.000, mendahului PA di ~212.000 dan Travel di ~322.000), dan itulah yang lebih
// dulu terbaca pengguna.
//
// Catatan: tab bawaan ini dapat saja TIDAK terlihat bagi pengguna, karena visibilitas tab
// mengikuti jabatannya. Layar karena itu membuka tab pertama yang BOLEH ia lihat, bukan
// memaksa yang ini — lihat VisibleTabs.
const DefaultTab = TabNonMBU

// Tab adalah satu antrean pada layar Inbox Manager Admin.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `<pyTitle>` pada section apa
	// adanya (`D-13`).
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena ketiga judul tab hanya berbeda pada
	// kata terakhirnya.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	Columns []Column

	// OrgUnit adalah unit organisasi penugasan yang disaring tab ini.
	//
	// Nilainya BUKAN sekadar keterangan: ia yang dikirim sebagai bind ke kueri dan yang
	// menyaring di repo/memory, sehingga keduanya tidak dapat berselisih.
	OrgUnit string

	// LineBusiness adalah lini bisnis yang membuka tab ini.
	LineBusiness string
}

// managerAdminColumns adalah kedelapan kolom grid, dipakai KETIGA tab.
//
// # Kenapa satu senarai untuk tiga tab
//
// Karena ketiganya dilayani Report Definition yang sama dengan satu-satunya perbedaan pada
// nilai parameter `OrgUnit`. Menyalinnya tiga kali hanya menciptakan tiga tempat yang dapat
// menyimpang tanpa alasan.
//
// # Urutan dan judulnya dari mana
//
// Urutannya dari urutan sel di `Section/InboxManagerAdmin_Section-Section.xml`, bukan dari
// urutan `pyListFields` Report Definition-nya — keduanya berbeda, dan yang dilihat pengguna
// adalah yang pertama.
//
// Judulnya dari `<pyCaption …>` pada section, BUKAN dari `pyFieldLabel` pada RD. Keduanya
// berbeda di tiga tempat:
//
//	isian                            pyFieldLabel (RD)       pyCaption (section)
//	.pyID                            Case ID                 ID
//	.Policy.Quotation.SobName        Nama Sumbis             Nama Sumber Bisnis
//	.pxCreateOpName                  Create Operator Name    Admin PNC
//
// # Satu kolom RD yang TIDAK digambar, dan tiga yang tidak dibawa sama sekali
//
// RD memuat 13 isian; section menggambar 8. Yang tidak digambar: `BranchName`,
// `.ClaimData.DateOfLoss`, `.ClaimData.StatusClaim`, `.ClaimData.UserTeknis`, dan
// `.pyOrigUserID`. Empat yang pertama tidak punya satu pun sel di section.
//
// `.ClaimData.StatusClaim` tetap DIBAWA sebagai data meski tidak digambar, karena berkas
// ekspor memuatnya — lihat WorkItem.ClaimStatus. Keempat sisanya tidak dibawa sama sekali.
var managerAdminColumns = []Column{
	{Key: FieldCaseID, Title: "ID"},
	{Key: FieldPolicyNumber, Title: "No Polis"},
	{Key: FieldInsuredName, Title: "Nama Tertanggung"},
	{Key: FieldBusinessName, Title: "Nama Bisnis"},
	{Key: FieldBusinessSource, Title: "Nama Sumber Bisnis"},
	{Key: FieldRegisteredAt, Title: "Tanggal Pendaftaran"},
	{Key: FieldClaimElapsed, Title: "Lama Waktu Klaim"},
	{Key: FieldAdminName, Title: "Admin PNC"},
}

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya di section.
var tabs = []Tab{
	{
		Code: TabNonMBU,

		// Judul ini ADA di layar lama, apa adanya: `pyCaption Manajemen Admin - Non MBU`.
		Name: "Manajemen Admin - Non MBU",

		Description: "Antrean registrasi klaim di luar Personal Accident dan Travel, " +
			"pada unit organisasi AdminPNC. Menampilkan pekerjaan seluruh petugas unit " +
			"itu, bukan hanya milik Anda.",

		Columns:      managerAdminColumns,
		OrgUnit:      OrgUnitNonMBU,
		LineBusiness: LineNonMBU,
	},
	{
		Code:        TabPA,
		Name:        "Manajemen Admin - PA",
		Description: "Antrean registrasi klaim Personal Accident, pada unit organisasi AdminPA.",

		Columns:      managerAdminColumns,
		OrgUnit:      OrgUnitPA,
		LineBusiness: LinePA,
	},
	{
		Code:        TabTravel,
		Name:        "Manajemen Admin - Travel",
		Description: "Antrean registrasi klaim Travel, pada unit organisasi AdminTRAVEL.",

		Columns:      managerAdminColumns,
		OrgUnit:      OrgUnitTravel,
		LineBusiness: LineTravel,
	},
}

// Tabs mengembalikan ketiga tab dalam urutan tampilnya, tanpa memandang siapa pemanggilnya.
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

// VisibleTabs mengembalikan tab yang BOLEH dilihat seorang pemanggil.
//
// # Aturannya mengikuti Pega apa adanya, dan itu keputusan yang tercatat
//
// Keputusan Work Owner 2026-09-26. Di sistem lama ketiga kontainer grid bersyarat:
//
//	OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit = 'Development'
//	OperatorID.pyPosition = 'PA'     || OperatorID.pyOrgUnit = 'Development'
//	OperatorID.pyPosition = 'TRAVEL' || OperatorID.pyOrgUnit = 'Development'
//
// Perbandingannya di sini TIDAK memandang huruf besar-kecil dan memangkas spasi, berbeda
// dari Pega yang membandingkan persis. Itu satu-satunya pelonggaran, dan ia hanya dapat
// MENAMBAH tab yang terlihat — tidak pernah menghilangkan. Alasannya: kolom
// `M_LOGIN_PNC.LINE_BUSINESS` bebas isi, dan ejaannya diketik manusia.
//
// # Koreksi 2026-09-27: sumber nilainya pernah salah, dan itu cacat — bukan kesetaraan
//
// Sampai 2026-09-27 perbandingan ini memakai `auth.User.Position`, yang bersumber dari HCQ
// `EmpResponse.Placement.PositionName` — sebuah JABATAN KEPEGAWAIAN seperti
// "IT SPECIALIST". Nilai itu tidak pernah berisi `NONMBU`, `PA`, maupun `TRAVEL`, sehingga
// **tidak seorang pun melihat satu tab pun**.
//
// Keadaan itu sempat dicatat sebagai konsekuensi yang diterima dari "ikuti Pega apa adanya".
// Pembacaan ulang export membantahnya: di Pega `pyPosition` adalah field pada rekaman
// operator yang diisi administrator dengan KODE LINI BISNIS, dan karena terisi, kontainernya
// TAMPIL. Yang salah bukan aturannya melainkan dari mana nilainya dibaca.
//
// Sejak koreksi itu nilainya dibaca dari `POOLDATA.M_LOGIN_PNC.LINE_BUSINESS` lewat seam
// LineBusinessRepo — kolom yang sudah ada, terverifikasi dari `ALL_TAB_COLUMNS`, dan sudah
// dipakai modul Inbox Outstanding untuk keperluan yang sama.
//
// # Yang MASIH menjadi konsekuensi, dan tidak diperbaiki kode
//
// `LINE_BUSINESS` baru terisi pada sebagian petugas — saat diperiksa 2026-09-24, pada 1 dari
// 1 baris `M_LOGIN_PNC` (`0004_DICABUT.md`). Petugas yang barisnya kosong tetap tidak
// melihat satu tab pun, dan itu benar: di Pega pun `pyPosition` yang tidak cocok satu pun
// tidak membuka kontainer mana pun. Yang menyelesaikannya adalah MENGISI kolom itu, bukan
// mengubah kode di sini.
//
// Padanan `OperatorID.pyOrgUnit` tidak ada sama sekali di sesi pengguna. Ia diisi dari
// variabel lingkungan di cmd, mengikuti preseden `PERAN_PENGGUNA` di registration.go, dan
// seperti preseden itu ia BUKAN otorisasi: ia tidak menjaga apa pun.
func VisibleTabs(caller Caller) []Tab {
	clean := caller.Clean()

	if strings.EqualFold(clean.OrgUnit, DevelopmentOrgUnit) {
		return Tabs()
	}

	result := []Tab{}
	for _, tab := range tabs {
		if strings.EqualFold(clean.LineBusiness, tab.LineBusiness) {
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
// Mengembalikan teks kosong bila tidak ada satu pun — keadaan yang mungkin terjadi, dan
// yang layar harus jelaskan alih-alih membuka tab yang akan ditolak server.
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

// ExpectedLineBusinesses adalah ketiga lini bisnis yang membuka tab, untuk ditampilkan ke
// pengguna yang tidak melihat satu tab pun.
//
// Ia data, bukan kalimat yang ditulis tetap di layar, supaya layar dan aturan penyaringnya
// tidak dapat menyebut nilai yang berbeda.
func ExpectedLineBusinesses() []string {
	return []string{LineNonMBU, LinePA, LineTravel}
}

// PlannedDifferences adalah selisih terhadap sistem lama yang DIPUTUSKAN, bukan cacat.
//
// # Kenapa ia data, bukan komentar
//
// Karena ia dikirim ke layar dan ditampilkan kepada pengguna. Selisih yang hanya tercatat di
// komentar akan dilaporkan berulang kali sebagai kerusakan oleh orang yang membandingkan
// layar baru dengan Pega berdampingan.
//
// Ia juga yang dipakai saat uji kesetaraan gerbang 1: setiap selisih WAJIB dapat dipetakan
// ke salah satu butir `P-5`, atau dinyatakan sebagai bug (`D-54`). Butir di bawah adalah
// pemetaan itu, sudah tertulis di muka alih-alih dicari setelah selisihnya muncul.
var PlannedDifferences = []string{
	"Kolom \"Status Klaim\" pada berkas ekspor kini berisi status PROSES, bukan status " +
		"bisnis. Di Pega ia dicari ke master status dan dapat bernilai salah satu dari 33 " +
		"keadaan — Register, Claim Committee, Paid, dan seterusnya. Sekarang ia " +
		"diturunkan dari status kerja, yang hanya mengenal On Progress, Close, dan " +
		"Reject (ketetapan Work Owner 2026-09-27). Karena layar ini memang hanya " +
		"menampilkan klaim yang masih berjalan, hampir setiap baris akan berbunyi " +
		"\"On Progress\".",

	"Satu klaim kini selalu satu baris, dan klaim yang belum punya penugasan terbuka " +
		"tetap terlihat. Sumber layar ini berpindah dari gabungan dua tabel Pega ke satu " +
		"tabel datar (ketetapan Work Owner 2026-09-27), sehingga gabungan `INNER` yang " +
		"lama ikut hilang. Gabungan itu membuang klaim yang tidak punya penugasan — " +
		"padahal justru itu pekerjaan yang terhenti — dan menampilkan klaim berpenugasan " +
		"banyak berkali-kali. Keduanya tidak pernah menghasilkan pesan galat.",

	"Untuk sementara daftarnya LEBIH PENDEK daripada di Pega, dan itu bukan karena " +
		"penyaring. Tabel sumbernya baru memuat sebagian klaim — 1.014 dari 7.703 saat " +
		"diperiksa 2026-09-22 — karena proses pengisinya belum mengejar. Selisih ini " +
		"menyusut dengan sendirinya begitu tabelnya terisi penuh, dan tidak menuntut " +
		"perubahan kode apa pun.",

	"Daftar tidak lagi terpotong di 500 baris. Report Definition Pega yang memasok " +
		"ketiga grid berjalan dengan `pyMaxRecords=500`, sehingga baris ke-501 dan " +
		"seterusnya tidak pernah terlihat — tanpa satu pun tanda bahwa daftarnya " +
		"terpotong. Batas itu tidak dibawa; berapa baris yang wajib dilayani satu layar " +
		"adalah pertanyaan terbuka `ADR-0011`.",

	"Ketiga kontainer grid menjadi TAB yang dapat dipilih. Di Pega ketiganya adalah " +
		"kontainer terpisah yang tampil menurut jabatan pengguna, sehingga petugas " +
		"berjabatan Development melihat ketiga tabel bertumpuk ke bawah, masing-masing " +
		"dengan penomoran halamannya sendiri. Isi, kolom, dan urutan kolomnya sama persis.",

	"Kolom \"Lama Waktu Klaim\" dihitung dalam waktu KALENDER, termasuk akhir pekan dan " +
		"hari libur. Di Pega ia memakai format bawaan platform `DateTime-Frame`, yang " +
		"kodenya tidak ikut dalam export rule sehingga tidak dapat dibaca. Bentuknya " +
		"direkonstruksi dari satu baris yang teramati langsung di layar Pega " +
		"(\"1 year 5 months ago\"); bunyi cabang hari, jam, dan menit BELUM diverifikasi.",
}
