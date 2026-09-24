package inboxadmin

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang
// dibaca pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke
// tab: `CaseID` berjudul "Case ID" pada enam tab dan "No Klaim" pada dua tab lain, persis
// seperti di `Section/PNCInboxAdmin-Section.xml`.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`),
	// termasuk ketika judul di satu layar bercampur dua bahasa — "Business source"
	// bersebelahan dengan "Cabang Klaim" memang begitu di sistem lama.
	Title string
}

// Nama field JSON pada satu baris pekerjaan.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di tab.go, penyusun DTO di http/dto.go,
// dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan — ketiganya merujuk
// nama yang sama.
const (
	FieldCaseID          = "case_id"
	FieldPolicyNumber    = "no_polis"
	FieldInsuredName     = "nama_tertanggung"
	FieldBusinessName    = "nama_bisnis"
	FieldBusinessSource  = "sumber_bisnis"
	FieldBranchName      = "nama_cabang"
	FieldClaimBranch     = "cabang_klaim"
	FieldCreator         = "pembuat"
	FieldLossDate        = "tanggal_kejadian"
	FieldReportDate      = "tanggal_lapor"
	FieldInputDate       = "tanggal_input"
	FieldNote            = "catatan"
	FieldClaimPosition   = "posisi_klaim"
	FieldClaimStatus     = "status_klaim"
	FieldLODStatus       = "status_lod"
	FieldRequestDate     = "tanggal_request"
	FieldPolicyBranch    = "cabang_polis"
	FieldSurveyBranch    = "cabang_survei"
	FieldTechnicalPIC    = "pic_klaim"
	FieldSurveyor        = "surveyor"
	FieldSurveyNumber    = "no_survei"
	FieldInboxDate       = "tanggal_masuk_inbox"
	FieldAnalystNote     = "deskripsi_analis"
	FieldRCLPUCLStatus   = "status_rcl_pucl"
	FieldLetterPrintDate = "tanggal_cetak_surat"
	FieldClaimAge        = "lama_klaim"
	FieldExpiryStatus    = "status_kadaluarsa"

	FieldReportAging  = "aging_lapor"
	FieldTotalAging   = "aging_total"
	FieldLODAging     = "aging_lod"
	FieldRequestAging = "aging_request"
)

// Kode tab, persis seperti nilai `TempView.CityID` di sistem lama.
//
// # Kenapa kodenya dipertahankan, padahal namanya tidak
//
// Properti pemilih tab di Pega bernama `CityID` — nama yang sama sekali tidak menyatakan
// isinya, sama seperti belasan alias lain di layar ini. Namanya karena itu TIDAK dibawa.
//
// Tetapi NILAINYA dipertahankan, dan itu keputusan yang berbeda: kode `3`, `7`, `9`, `11`
// muncul di prakondisi 34 langkah activity dan di kondisi tampil 8 kontainer grid. Menomori
// ulang tabnya berarti setiap penelusuran balik ke export Pega harus menempuh satu tabel
// terjemahan — dan penelusuran balik itulah yang dipakai saat uji kesetaraan gerbang 1
// menemukan selisih.
//
// Kode `4`, `5`, dan `6` — tab Komunikasi — sengaja TIDAK ada di sini. Lihat DisabledTabs.
const (
	TabAll             = "3"
	TabUnregisteredRCV = "7"
	TabRCVOnline       = "8"
	TabRequestSurvey   = "9"
	TabRequestDocument = "10"
	TabAllCaseAdmin    = "11"
	TabBranchClaim     = "12"
	TabRCLPUCL         = "13"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Keputusan Work Owner 2026-09-20: **All Case Admin**, bukan ALL. Pilihan itu punya akibat
// yang bagus dan perlu disebut — kueri tab ini menyaring `PXCREATEOPNAME` ke pengguna yang
// login, sehingga layar terbuka pada pekerjaan MILIK petugas itu sendiri, bukan pada
// seluruh klaim yang sedang berjalan di perusahaan.
const DefaultTab = TabAllCaseAdmin

// Tab adalah satu antrean kerja pada layar Inbox Admin.
type Tab struct {
	// Code adalah kode tab, sama dengan nilai `TempView.CityID` sistem lama.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`.
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak
	// punya keterangan seperti ini; ia ditambahkan karena delapan tab dengan judul
	// sependek "ALL" tidak memberi tahu apa pun tentang isinya.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti di Pega.
	Columns []Column

	// ScopedToCaller menyatakan kuerinya menyaring menurut pengguna yang login.
	//
	// Tiga tab melakukannya, dan itulah yang membuat mereka "milik saya": Request Survey
	// menyaring `PXASSIGNEDOPERATORID`, sedangkan Request Dokumen dan All Case Admin
	// menyaring `PXCREATEOPNAME`. Lima tab lain menampilkan antrean bersama.
	ScopedToCaller bool

	// SupportsSearch menyatakan kotak cari berlaku pada tab ini.
	//
	// Lima kueri lama memuat `{ASIS:TempContents.Keyword}`; dua tidak. Menampilkan kotak
	// cari yang tidak menyaring apa pun lebih buruk daripada tidak menampilkannya.
	SupportsSearch bool

	// SupportsBusinessFilter menyatakan penyaring lini bisnis berlaku pada tab ini.
	//
	// Hanya kueri yang bergabung ke POOLDATA.BUSINESS dan BUSINESSGROUP yang dapat
	// menyaringnya — empat dari tujuh.
	SupportsBusinessFilter bool
}

// Kolom yang dipakai berulang, disusun sekali supaya judulnya tidak dapat berbeda antar tab
// tanpa disengaja.
var (
	colCaseID       = Column{Key: FieldCaseID, Title: "Case ID"}
	colClaimNumber  = Column{Key: FieldCaseID, Title: "No Klaim"}
	colPolicyNumber = Column{Key: FieldPolicyNumber, Title: "Policy no"}
	colInsuredName  = Column{Key: FieldInsuredName, Title: "Insured name"}
	colBusinessName = Column{Key: FieldBusinessName, Title: "Business name"}
	colBusinessSrc  = Column{Key: FieldBusinessSource, Title: "Business source"}
	colBranchName   = Column{Key: FieldBranchName, Title: "Branch Name"}
	colLossDate     = Column{Key: FieldLossDate, Title: "Date of loss"}
	colReportDate   = Column{Key: FieldReportDate, Title: "Report Date"}
	colInputDate    = Column{Key: FieldInputDate, Title: "Input Date"}
	colReportAging  = Column{Key: FieldReportAging, Title: "Aging Report"}
	colTotalAging   = Column{Key: FieldTotalAging, Title: "Total Aging"}
	colCreator      = Column{Key: FieldCreator, Title: "Creator"}
)

// tabs adalah kedelapan tab beserta kolomnya, berurutan seperti dikehendaki Work Owner:
// All Case Admin lebih dulu karena ia yang terbuka pertama.
//
// Urutan kolom tiap tab diambil dari urutan sel di `Section/PNCInboxAdmin-Section.xml`,
// bukan dari urutan kolom SQL-nya — keduanya berbeda, dan yang dilihat pengguna adalah
// yang pertama.
var tabs = []Tab{
	{
		Code:        TabAllCaseAdmin,
		Name:        "All Case Admin",
		Description: "Klaim yang Anda buat dan masih berjalan.",
		Columns: []Column{
			colClaimNumber, colPolicyNumber, colInsuredName, colBusinessName,
			colBusinessSrc, colLossDate, colReportDate, colInputDate,
		},
		ScopedToCaller: true,
		SupportsSearch: true,
	},
	{
		Code:        TabAll,
		Name:        "ALL",
		Description: "Seluruh klaim PNC yang masih berjalan pada tahap Register, Estimasi, atau Estimation.",
		Columns: []Column{
			colCaseID, colPolicyNumber, colInsuredName, colBusinessName, colBusinessSrc,
			colBranchName, colLossDate, colReportDate, colInputDate,
			colReportAging, colTotalAging, colCreator,
			{Key: FieldClaimBranch, Title: "Branch Claim"},
			{Key: FieldClaimPosition, Title: "Claim Position"},
			{Key: FieldClaimStatus, Title: "Claim Status"},
		},
		SupportsSearch:         true,
		SupportsBusinessFilter: true,
	},
	{
		Code:        TabUnregisteredRCV,
		Name:        "Unregistered RCV",
		Description: "Dokumen yang sudah diterima tetapi klaimnya belum diregistrasi.",
		Columns: []Column{
			colCaseID, colPolicyNumber, colInsuredName, colBusinessName, colBusinessSrc,
			colBranchName, colLossDate, colInputDate,
			{Key: FieldNote, Title: "Note"},
			colTotalAging, colCreator,
			{Key: FieldClaimBranch, Title: "Cabang Klaim"},
		},
		SupportsSearch:         true,
		SupportsBusinessFilter: true,
	},
	{
		Code: TabRCVOnline,
		Name: "Unregistered RCV Online",
		Description: "Sama dengan Unregistered RCV, tetapi hanya yang masuk lewat " +
			"Auto Service.",
		Columns: []Column{
			colCaseID, colPolicyNumber, colInsuredName, colBusinessName, colBusinessSrc,
			colBranchName, colLossDate, colInputDate,
			{Key: FieldNote, Title: "Note"},
			colTotalAging, colCreator,
			{Key: FieldClaimBranch, Title: "Cabang Klaim"},
		},
		SupportsSearch:         true,
		SupportsBusinessFilter: true,
	},
	{
		Code:        TabRequestSurvey,
		Name:        "Request Survey",
		Description: "Permintaan survei yang ditugaskan kepada Anda dan belum ada adjustment-nya.",
		Columns: []Column{
			colCaseID, colPolicyNumber,
			{Key: FieldRequestDate, Title: "Tanggal Request"},
			{Key: FieldRequestAging, Title: "Aging"},
			{Key: FieldPolicyBranch, Title: "Cabang Polis"},
			{Key: FieldSurveyBranch, Title: "Cabang Survey"},
			{Key: FieldTechnicalPIC, Title: "PIC Klaim"},
			{Key: FieldSurveyor, Title: "Surveyor"},
			{Key: FieldSurveyNumber, Title: "No Survey"},
		},
		ScopedToCaller: true,
		SupportsSearch: true,
	},
	{
		Code:        TabRequestDocument,
		Name:        "Request Dokumen",
		Description: "Permintaan dokumen yang Anda kirim dan belum dijawab penerimanya.",
		Columns: []Column{
			colClaimNumber, colPolicyNumber, colInsuredName, colBusinessName,
			colBusinessSrc, colLossDate, colReportDate, colInputDate,
		},
		ScopedToCaller: true,
	},
	{
		Code:        TabBranchClaim,
		Name:        "Branch Claim",
		Description: "Klaim personal accident yang menunggu unggahan LOD dari cabang.",
		Columns: []Column{
			colCaseID, colPolicyNumber, colInsuredName, colBusinessSrc, colBranchName,
			colLossDate, colReportDate, colInputDate,
			colReportAging, colTotalAging, colCreator,
			{Key: FieldClaimBranch, Title: "Branch Claim"},
			// Judulnya "Claim Status", tetapi isinya PROSES — `CASE PYSTATUSWORK` yang
			// sama dengan kolom berjudul "Claim Position" pada tab ALL. Kueri
			// `GetKlaimCabang` tidak membawa V_STS_CLAIM sama sekali, sehingga status
			// bisnis klaim memang tidak tersedia di tab ini. Judul lama dipertahankan
			// (`D-13`); yang tidak dipertahankan adalah menduga ia berisi hal lain.
			{Key: FieldClaimPosition, Title: "Claim Status"},
			{Key: FieldLODStatus, Title: "LOD Status"},
			{Key: FieldLODAging, Title: "Aging LOD"},
		},
		SupportsSearch:         true,
		SupportsBusinessFilter: true,
	},
	{
		Code:        TabRCLPUCL,
		Name:        "Status RCL/PUCL",
		Description: "Klaim yang suratnya sudah dicetak dan menunggu persetujuan PUCL.",
		Columns: []Column{
			colCaseID,
			{Key: FieldPolicyNumber, Title: "No Polis"},
			{Key: FieldInsuredName, Title: "Nama Tertanggung"},
			{Key: FieldInboxDate, Title: "Tanggal Masuk Inbox"},
			{Key: FieldAnalystNote, Title: "Deskripsi Analyst"},
			{Key: FieldRCLPUCLStatus, Title: "Status RCL/PUCL"},
			{Key: FieldLetterPrintDate, Title: "Tanggal Cetak Surat"},
			{Key: FieldClaimAge, Title: "Lama Klaim"},
			{Key: FieldExpiryStatus, Title: "Status Kadaluarsa"},
		},
	},
}

// Tabs mengembalikan kedelapan tab dalam urutan tampilnya.
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

// DisabledTabs adalah tab yang ADA di export Pega tetapi TIDAK dibangun, beserta alasannya.
//
// Ia ditulis sebagai data, bukan sebagai komentar yang hilang saat berkas ini dibaca
// sebagian, karena pertanyaan "kenapa tab Komunikasi tidak ada" pasti diajukan lagi.
//
// Ketiganya dinyatakan Work Owner 2026-09-20 sudah tidak dipakai — sudah di-*remark* di
// aplikasi Pega. Pembacaan export mendukungnya dari dua sisi:
//
//   - `Section/PNCInboxAdmin-Section.xml` memuat EMPAT elemen ber-`pyCondition` `1==2`,
//     yaitu kondisi yang tidak pernah benar — cara Pega menyembunyikan elemen tanpa
//     menghapusnya. Satu di antaranya berada di bilah tab.
//   - `RDB List/BrowseClaimALLKomunikasi-SQL.xml` yang melayani ketiganya adalah `UNION`
//     dengan JUMLAH KOLOM BERBEDA — 15 pada cabang pertama, 14 pada cabang kedua yang
//     kehilangan `KODECABANG_1`. Oracle menolaknya dengan ORA-01789, sehingga ketiga tab
//     itu memang tidak dapat berjalan.
//
// Urutan tag di dalam XML section ini acak, sehingga pasangan label-ke-kode tidak dapat
// dibuktikan dari urutan markup-nya; yang dipakai adalah pernyataan Work Owner.
var DisabledTabs = []DisabledTab{
	{Code: "4", Name: "Not Answered", Reason: "tidak dipakai — sudah di-remark di Pega"},
	{Code: "5", Name: "Not replied from Receiver", Reason: "tidak dipakai — sudah di-remark di Pega"},
	{Code: "6", Name: "Replied from ASM", Reason: "tidak dipakai — sudah di-remark di Pega"},
}

// DisabledTab adalah satu tab sistem lama yang sengaja tidak dibangun.
type DisabledTab struct {
	Code   string
	Name   string
	Reason string
}
