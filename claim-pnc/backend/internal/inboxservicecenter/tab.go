package inboxservicecenter

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada ServiceClaim yang digambar, dan Title menyebut JUDUL yang
// dibaca pengguna. Keduanya dipisah supaya layar tidak perlu tahu nama isian apa pun.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`).
	Title string
}

// Nama field JSON pada satu baris.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan —
// ketiganya merujuk nama yang sama.
const (
	FieldID             = "id"
	FieldInputDate      = "tanggal_input"
	FieldPolicyNumber   = "no_polis"
	FieldCustomerName   = "nasabah"
	FieldType           = "tipe"
	FieldTechnicalPIC   = "pic"
	FieldRepairID       = "repair_id"
	FieldClaimNumber    = "no_klaim"
	FieldIMEI           = "imei"
	FieldRepairStatus   = "status_perbaikan"
	FieldApprovalStatus = "status_persetujuan"
)

// Kode tab pada kontrak API.
//
// # Kenapa kodenya TIDAK sama dengan nilai Pega, padahal modul lain mempertahankannya
//
// Karena di layar ini nilai Pega untuk tab pertama adalah TEKS KOSONG. Parameter tab pada
// `Activity/DataServiceCenter-Act.xml` bernama `stsapprove`, dan keempat section mengisinya
// begini:
//
//	Section/ClaimServiceCenterOSnotTransfer-Section.xml      <stsapprove/>      (kosong)
//	Section/BrowseServiceCenterWaitingApproval-Section.xml   <stsapprove>"0"</stsapprove>
//	Section/BrowseServiceCenterApprove-Section.xml           <stsapprove>"1"</stsapprove>
//	Section/BrowseServiceCenterReject-Section.xml            <stsapprove>"2"</stsapprove>
//
// Pada parameter query HTTP, "kosong" tidak dapat dibedakan dari "tidak dikirim" — dan
// keduanya di sini berarti hal yang BERBEDA: yang pertama tab Registrasi SC, yang kedua
// "pakai tab bawaan". Mempertahankan teks kosong sebagai kode berarti `?tab=` dan tanpa
// `tab` sama sekali harus berperilaku berbeda, padahal keduanya sampai ke server sebagai
// nilai yang sama.
//
// Nilai Pega-nya tidak hilang: ia tetap tercatat pada Tab.PegaParam, dan itulah yang dipakai
// menelusuri balik ke export saat uji kesetaraan gerbang 1 menemukan selisih.
const (
	TabRegistration    = "registrasi-sc"
	TabWaitingApproval = "waiting-approval"
	TabApproved        = "approved"
	TabRejected        = "rejected"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Registrasi SC, karena ia tab pertama pada wadah TABBED di
// `Section/BrowseServiceCenter-Section.xml` — dan karena isinya justru yang menuntut
// tindakan: baris yang `STS_APPROVAL`-nya masih NULL belum diajukan ke siapa pun.
const DefaultTab = TabRegistration

// Tab adalah satu antrean kerja pada layar Inbox Service Center.
type Tab struct {
	// Code adalah kode tab pada kontrak API.
	Code string

	// PegaParam adalah nilai `stsapprove` yang dipakai sistem lama untuk tab ini.
	//
	// Ia dibawa demi ketelusuran, bukan untuk dipakai layar: tanpa nilai ini, menelusuri
	// balik sebuah selisih ke langkah activity Pega menempuh satu tabel terjemahan yang
	// tidak tertulis di mana pun. Teks kosong pada tab pertama memang begitu adanya.
	PegaParam string

	// Name adalah judul tab yang dibaca pengguna, diambil apa adanya dari `pyTitle` pada
	// `Section/BrowseServiceCenter-Section.xml` (`D-13`).
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena judul sependek "Approved" tidak
	// memberi tahu apa pun tentang baris mana yang masuk ke sana.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti di Pega.
	Columns []Column
}

// Keenam kolom grid.
//
// # Kenapa keempat tab berbagi daftar yang sama
//
// Karena begitulah di Pega: keempat section tab memuat `pyCaption` yang PERSIS sama — ID,
// Tanggal Input, No Polis, Nasabah, Tipe, PIC — dan keempatnya mengikat properti yang sama
// pula (`CaseID`, `DateOfLoss`, `PolicyNo`, `UserName`, `RefNo`, `PICRekanan`). Yang berbeda
// antar tab hanyalah penyaringnya, bukan bentuk gridnya.
//
// Enam kolom juga yang menjadi median seluruh grid sistem lama (`T-11`) — layar ini tidak
// termasuk yang lebar.
var gridColumns = []Column{
	{Key: FieldID, Title: "ID"},
	{Key: FieldInputDate, Title: "Tanggal Input"},
	{Key: FieldPolicyNumber, Title: "No Polis"},
	{Key: FieldCustomerName, Title: "Nasabah"},
	{Key: FieldType, Title: "Tipe"},
	{Key: FieldTechnicalPIC, Title: "PIC"},
}

// columns menyerahkan salinan daftar kolom, supaya satu tab tidak dapat mengubah tab lain.
func columns() []Column {
	result := make([]Column, len(gridColumns))
	copy(result, gridColumns)
	return result
}

// tabs adalah keempat tab dalam urutan tampilnya pada wadah TABBED.
var tabs = []Tab{
	{
		Code:      TabRegistration,
		PegaParam: "",
		Name:      "Registrasi SC",
		Description: "Klaim portal rekanan yang belum pernah diajukan ke komite — " +
			"status persetujuannya masih kosong.",
		Columns: columns(),
	},
	{
		Code:        TabWaitingApproval,
		PegaParam:   "0",
		Name:        "Waiting Approval",
		Description: "Sudah diajukan dan menunggu keputusan komite.",
		Columns:     columns(),
	},
	{
		Code:        TabApproved,
		PegaParam:   "1",
		Name:        "Approved",
		Description: "Sudah disetujui komite.",
		Columns:     columns(),
	},
	{
		Code:      TabRejected,
		PegaParam: "2",
		Name:      "Rejected",
		// Dua kode, bukan satu — lihat ApprovalRejected dan ApprovalTotalLoss.
		Description: "Ditolak komite, termasuk yang diputuskan Total Loss Only (TLO).",
		Columns:     columns(),
	},
}

// Tabs mengembalikan keempat tab dalam urutan tampilnya.
//
// Salinan, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah daftar tab dengan
// menulisi hasilnya.
func Tabs() []Tab {
	result := make([]Tab, 0, len(tabs))
	for _, tab := range tabs {
		result = append(result, copyTab(tab))
	}
	return result
}

// FindTab mencari tab menurut kodenya.
func FindTab(code string) (Tab, bool) {
	for _, tab := range tabs {
		if tab.Code == code {
			return copyTab(tab), true
		}
	}
	return Tab{}, false
}

// copyTab menyalin satu tab BESERTA daftar kolomnya.
//
// # Kenapa penyalinan dangkal tidak cukup
//
// Karena Columns adalah senarai, dan menyalin strukturnya hanya menyalin kepala senarai —
// larik di baliknya tetap yang sama. Tanpa penyalinan ini, pemanggil yang menulisi
// `Tabs()[0].Columns[0].Title` akan mengubah judul kolom bagi SELURUH permintaan berikutnya,
// karena `tabs` adalah nilai tingkat paket yang hidup selama aplikasi berjalan.
//
// Ini bukan kekhawatiran teoretis: uji
// TestDaftarKolomTidakDapatDiubahLewatHasilTabs menangkapnya saat modul ini ditulis.
func copyTab(tab Tab) Tab {
	clone := tab
	clone.Columns = make([]Column, len(tab.Columns))
	copy(clone.Columns, tab.Columns)
	return clone
}

// Kode status persetujuan — kolom `STS_APPROVAL`.
//
// Keempatnya dibaca dari `Activity/DataServiceCenter-Act.xml`, yang menyusun isi dropdown
// `TempApproval` sebagai pasangan `Status` dan `StatusClaim`:
//
//	"0" -> "==PILIH=="   "1" -> "APPROVED"   "2" -> "TLO"   "3" -> "REJECT"
//
// Label `==PILIH==` adalah teks penanda dropdown, bukan nama keadaan; di sini ia diberi nama
// yang menyatakan artinya. Nilai NULL tidak ada di dropdown itu sama sekali, dan memang
// begitu: ia keadaan baris yang BELUM pernah masuk ke dropdown mana pun.
const (
	ApprovalPending   = "0"
	ApprovalApproved  = "1"
	ApprovalTotalLoss = "2"
	ApprovalRejected  = "3"
)

// ApprovalStatusLabel menerjemahkan kode status persetujuan menjadi teks yang dibaca
// pengguna.
//
// Kode yang tidak dikenal dikembalikan APA ADANYA, bukan diganti "Tidak diketahui": kode
// asing berarti ada nilai di basis data yang belum terbaca modul ini, dan menyembunyikannya
// di balik satu label seragam membuat hal itu tidak pernah ketahuan.
func ApprovalStatusLabel(code string) string {
	switch code {
	case "":
		return "Belum diajukan"
	case ApprovalPending:
		return "Menunggu Approval"
	case ApprovalApproved:
		return "APPROVED"
	case ApprovalTotalLoss:
		return "TLO"
	case ApprovalRejected:
		return "REJECT"
	default:
		return code
	}
}

// Kode status perbaikan — kolom `STATUS`.
//
// Kesembilannya dibaca dari `Activity/DataServiceCenter-Act.xml`, yang menyusun daftar
// `StatusRepair` sebagai pasangan `CountryID` dan `Country` — dua nama properti yang tidak
// menyatakan isinya sama sekali, persis kelas alias menyesatkan yang `D-19` tinggalkan.
var repairStatusLabels = map[string]string{
	"1": "Repair Submitted",
	"2": "Repair Assesment",
	"3": "Repair Cancel",
	"4": "Repair Indent",
	"5": "Repair Eligible",
	"6": "Repair Inprogress",
	"7": "Repair Completed",
	"8": "Pick UP",
	"9": "Data SC",
}

// RepairStatusLabel menerjemahkan kode status perbaikan menjadi teks yang dibaca pengguna.
//
// Ejaan "Assesment" dipertahankan apa adanya meski salah ketik, karena itulah yang tertulis
// di rule dan itulah yang dibaca pengguna hari ini (`D-13`).
func RepairStatusLabel(code string) string {
	if label, known := repairStatusLabels[code]; known {
		return label
	}
	if code == "" {
		return ""
	}
	return code
}

// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta alasannya.
//
// Ia data, bukan komentar, supaya dapat dikirim apa adanya ke layar — dan supaya hilang
// dengan sendirinya begitu penghalangnya hilang, tanpa menyunting frontend.
//
// Keempatnya nyata dan masing-masing punya jejak bukti; tidak satu pun ditulis berdasarkan
// dugaan.
func Limitations() []string {
	return []string{
		"Layar ini baru MEMBACA. Menyimpan rincian dan memutuskan komite belum dibangun — " +
			"bukan lagi karena artefaknya kurang (source PEGA_PORTAL_REKANAN sudah diterima " +
			"2026-09-28), melainkan karena tabelnya hari ini masih ditulis Pega. Selama masa " +
			"paralel, satu tabel hanya boleh ditulis satu sistem (P-1).",

		"Rincian klaim hanya terbuka bagi PIC-nya sendiri. Di Pega rincian tidak disaring, " +
			"karena di sana ia hanya dapat dicapai lewat klik pada baris yang sudah tersaring; " +
			"pada API yang dapat dipanggil langsung, jaminan itu hilang dan ID dapat ditebak.",

		"Grid \"Detail Part\" belum ditampilkan. Isinya tersimpan sebagai JSON pada kolom " +
			"DETAILPART, dan bentuk JSON-nya belum pernah dibaca dari data sungguhan.",

		"Kotak Cari menyaring No Polis, Nasabah, dan IMEI. Mencari dengan ID atau No Klaim " +
			"saja tidak menghasilkan baris — sistem lama menggabungkan dua penyaring " +
			"pencarian dengan AND, sehingga yang benar-benar bekerja hanyalah ketiga isian " +
			"yang ada di keduanya. Perilaku itu ditiru apa adanya (P-5).",

		"Saat Cari terisi, paginasi dimatikan dan seluruh baris yang cocok ditampilkan " +
			"sekaligus — persis seperti sistem lama.",

		"Penyaring tambahan bagi peran Service Center (hanya baris yang ia buat sendiri) " +
			"belum aktif karena sumber peran belum ada (TKT-F3-004). Barisnya sudah " +
			"dipersempit menurut PIC, seperti di sistem lama.",

		"Isian \"BIAYA LAINNYA\" kini terisi benar. Di layar lama ia dan \"ALASAN BATAL\" " +
			"berbagi satu nama properti, sehingga yang satu menimpa yang lain dan salah " +
			"satunya selalu hilang.",
	}
}
