package inboxprogressclaim

// Column adalah satu kolom pada grid sebuah region.
//
// Ketiga isiannya dipisah karena ketiganya memang berbeda, dan di layar ini perbedaannya
// nyata — bukan kerapian belaka:
//
//	Key    identitas kolom, unik di dalam satu region
//	Field  isian mana pada baris yang digambar
//	Title  judul yang dibaca pengguna
//
// # Kenapa Key terpisah dari Field
//
// Karena region Outstanding menggambar SATU isian di DUA kolom: `DateForAging` muncul dua
// kali, keduanya terikat `tglklaim`. Bila identitas kolom diambil dari nama isiannya, kedua
// kolom itu bertabrakan. Lihat catatan pada outstandingColumns.
type Column struct {
	// Key adalah identitas kolom, unik di dalam satu region.
	Key string

	// Field adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga
	// tetap berbahasa Indonesia (`D-80`).
	Field string

	// Title adalah judul kolom yang dibaca pengguna.
	//
	// # Kenapa judulnya alias Pega, padahal nama di kode tidak
	//
	// Keputusan Work Owner 2026-09-21. Judul kolom layar ini TIDAK ADA di export: setiap
	// sel grid ber-`pyHeaderTitle` kosong, judulnya datang dari label properti Pega, dan
	// folder `Property` tidak ikut dikirim. Tiga pilihan diajukan — menamai sesuai isi
	// sebenarnya, meminta judul dari layar Pega yang berjalan, atau memakai alias apa
	// adanya — dan yang dipilih adalah yang ketiga.
	//
	// Akibatnya sebagian judul menyatakan hal yang bukan isinya: kolom berjudul `District`
	// berisi nama tertanggung, dan `ClaimNo` berisi nomor polis. Itu diterima secara sadar
	// demi `D-13` — pengguna membaca judul yang sama dengan yang selama ini dibacanya.
	//
	// Arti sebenarnya tidak hilang: ia ada di Description, yang digambar layar sebagai
	// keterangan kolom.
	Title string

	// Description menyatakan isi kolom yang sebenarnya, dalam satu frasa.
	//
	// Ia ada KARENA judulnya memakai alias yang menyesatkan. Tanpa ini, keputusan memakai
	// alias apa adanya akan membuat layar baru sama tidak terbacanya dengan layar lama —
	// dan satu-satunya tempat artinya tercatat adalah kode backend.
	Description string
}

// Nama field JSON pada satu baris klaim.
//
// Dikumpulkan sebagai konstanta supaya susunan kolom di berkas ini, penyusun DTO di
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan —
// ketiganya merujuk nama yang sama.
const (
	FieldClaimNumber      = "no_klaim"
	FieldPolicyNumber     = "no_polis"
	FieldInsuredName      = "nama_tertanggung"
	FieldRegisterDate     = "tanggal_registrasi"
	FieldLossDate         = "tanggal_kejadian"
	FieldLGBNote          = "catatan_lgb"
	FieldTechnicalPIC     = "pic_klaim"
	FieldNextFollowUp     = "next_follow_up"
	FieldProgressStatus1  = "status_progres_1"
	FieldProgressStatus2  = "status_progres_2"
	FieldPosition         = "posisi"
	FieldEarliestFollowUp = "follow_up_terawal"
	FieldProcessDate      = "tanggal_proses"
	FieldProdKe           = "prod_ke"
)

// Nama field JSON pada satu baris rekap per PIC.
const (
	FieldPIC           = "pic"
	FieldClaimCount    = "jumlah_klaim"
	FieldUpdateCount   = "jumlah_pembaruan"
	FieldDueTodayCount = "jatuh_tempo_hari_ini"
	FieldOnTimeCount   = "tepat_waktu"
	FieldLateCount     = "terlambat"
)

// Kode region, dipakai layar sebagai parameter `bagian`.
//
// # Kenapa kodenya kata, bukan angka seperti di Inbox Admin
//
// Karena di sini tidak ada nomor yang perlu dipertahankan. Inbox Admin memakai kode `3`,
// `7`, `9` … karena angka itu benar-benar ada di sistem lama sebagai nilai
// `TempView.CityID`, dan penelusuran balik ke export menempuh angka itu. Layar ini tidak
// punya pemilih region sama sekali — kelima bagiannya ditumpuk dan masing-masing memuat
// datanya sendiri, sehingga tidak ada nilai warisan yang dapat dipertahankan.
const (
	ViewOutstanding  = "outstanding"
	ViewNextFollowUp = "next-fu"
	ViewPerPIC       = "per-pic"
	ViewEvaluation   = "evaluasi"
)

// DefaultView adalah region yang dimuat pertama kali.
//
// Outstanding, karena ia yang paling atas di layar lama dan satu-satunya yang memuat
// seluruh klaim berjalan.
const DefaultView = ViewOutstanding

// Kind menyatakan bentuk baris sebuah region.
//
// Dua region memuat baris KLAIM, satu memuat baris PETUGAS, dan satu tidak memuat apa pun.
// Layar menggambar ketiganya berbeda, dan repo memanggil operasi yang berbeda pula.
type Kind string

// Ketiga bentuk baris.
const (
	// KindClaim berarti barisnya klaim — ClaimRow.
	KindClaim Kind = "klaim"

	// KindPIC berarti barisnya petugas beserta pencacahnya — PICSummary.
	KindPIC Kind = "pic"

	// KindEmpty berarti region itu tidak memuat data sama sekali. Lihat EvaluationView.
	KindEmpty Kind = "kosong"
)

// View adalah satu region pada layar Inbox Progress Claim.
type View struct {
	// Code adalah kode region.
	Code string

	// Name adalah judul region, mengikuti judul di Pega apa adanya (`D-13`).
	Name string

	// Description menjelaskan isi region dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini.
	Description string

	// Kind menyatakan bentuk barisnya.
	Kind Kind

	// Columns adalah kolom grid region ini, berurutan seperti di Pega.
	Columns []Column

	// Paginated menyatakan region ini dipaginasi.
	//
	// Dua region klaim dipaginasi 15 baris seperti di Pega; rekap per PIC tidak, karena
	// `Activity/GetProgressPerPIC-Act.xml` tidak memuat satu pun langkah Pagination.
	Paginated bool

	// SupportsSearch menyatakan kotak cari berlaku pada region ini.
	SupportsSearch bool

	// SupportsDateRange menyatakan penyaring rentang tanggal registrasi berlaku.
	//
	// Hanya region per PIC. Ia satu-satunya yang benar-benar menyaring tanggal — lihat
	// DeadControls untuk kotak tanggal yang ADA di layar lama tetapi tidak menyaring
	// apa pun.
	SupportsDateRange bool

	// SupportsBusinessFilter menyatakan penyaring lini bisnis BENAR-BENAR menyaring.
	//
	// Hanya region per PIC. Pada dua region klaim, dropdown-nya ada di layar tetapi
	// hasilnya dibuang — lihat DeadControls.
	SupportsBusinessFilter bool

	// ScopedToCaller menyatakan barisnya dibatasi ke petugas yang login.
	ScopedToCaller bool
}

// Kolom region klaim, disusun sekali supaya judulnya tidak dapat berbeda antarregion tanpa
// disengaja.
var (
	colClaimNumber = Column{
		Key: FieldClaimNumber, Field: FieldClaimNumber,
		Title: "CaseID", Description: "Nomor Klaim",
	}
	colPolicyNumber = Column{
		Key: FieldPolicyNumber, Field: FieldPolicyNumber,
		Title: "ClaimNo", Description: "Nomor Polis",
	}
	colInsuredName = Column{
		Key: FieldInsuredName, Field: FieldInsuredName,
		Title: "District", Description: "Nama Tertanggung",
	}
	colRegisterDate = Column{
		Key: FieldRegisterDate, Field: FieldRegisterDate,
		Title: "DateForAging", Description: "Tanggal Registrasi",
	}
	colLossDate = Column{
		Key: FieldLossDate, Field: FieldLossDate,
		Title: "DateOfLoss", Description: "Tanggal Kejadian",
	}
	colLGBNote = Column{
		Key: FieldLGBNote, Field: FieldLGBNote,
		Title: "Country", Description: "Catatan LGB",
	}
	colTechnicalPIC = Column{
		Key: FieldTechnicalPIC, Field: FieldTechnicalPIC,
		Title: "UserTeknis", Description: "PIC Klaim",
	}
	colNextFollowUp = Column{
		Key: FieldNextFollowUp, Field: FieldNextFollowUp,
		Title: "AnalystTransferDate", Description: "Next Follow Up per posisi",
	}
	colProgressStatus1 = Column{
		Key: FieldProgressStatus1, Field: FieldProgressStatus1,
		Title: "CityID", Description: "Status Progres 1",
	}
	colProgressStatus2 = Column{
		Key: FieldProgressStatus2, Field: FieldProgressStatus2,
		Title: "CountryID", Description: "Status Progres 2",
	}
	colPosition = Column{
		Key: FieldPosition, Field: FieldPosition,
		Title: "City", Description: "Posisi klaim yang sedang berjalan",
	}

	// colRegisterDateAgain adalah kolom KEDUA yang menggambar isian yang sama dengan
	// colRegisterDate.
	//
	// # Ini bukan salah salin
	//
	// `Section/ProgressClaim_Section-Section.xml` benar-benar mengikat `.DateForAging` di
	// dua sel grid terpisah pada region yang sama — sel ke-4 dan sel ke-12 dari 13 sel
	// yang ada. Keduanya menggambar `tglklaim`, sehingga dua kolom bersebelahan
	// menampilkan tanggal yang sama persis dengan judul yang sama pula.
	//
	// # Kenapa tetap digambar
	//
	// Keputusan Work Owner 2026-09-21: perilaku layar ini direplikasi apa adanya.
	// Menghapus kolom kedua akan mengubah jumlah kolom yang dibandingkan pada uji
	// kesetaraan gerbang 1.
	//
	// # Dugaan yang TIDAK diterapkan
	//
	// Kueri `DataProgressClaim` mengembalikan `tgl_proses` (alias `TanggalAnalystSendRCL`)
	// yang TIDAK terikat ke satu sel pun. Sangat mungkin sel ke-12 seharusnya menggambar
	// kolom itu dan salah diikat. Itu dugaan, bukan bukti — ia dicatat di sini dan
	// diajukan ke Work Owner, bukan diam-diam diperbaiki.
	colRegisterDateAgain = Column{
		Key: FieldRegisterDate + "_2", Field: FieldRegisterDate,
		Title: "DateForAging", Description: "Tanggal Registrasi — kolom kedua, sama dengan sebelumnya",
	}

	colEarliestFollowUp = Column{
		Key: FieldEarliestFollowUp, Field: FieldEarliestFollowUp,
		Title: "KomiteApproveDate", Description: "Follow Up terawal pada klaim ini",
	}
)

// outstandingColumns adalah ketiga belas kolom region Outstanding, berurutan seperti urutan
// sel di `Section/ProgressClaim_Section-Section.xml`.
var outstandingColumns = []Column{
	colClaimNumber, colPolicyNumber, colInsuredName, colRegisterDate, colLossDate,
	colLGBNote, colTechnicalPIC, colNextFollowUp, colProgressStatus1, colProgressStatus2,
	colPosition, colRegisterDateAgain, colEarliestFollowUp,
}

// nextFollowUpColumns adalah kedua belas kolom region Next Follow Up.
//
// Sama dengan Outstanding kecuali `KomiteApproveDate`, yang memang nol kemunculan di
// region itu.
var nextFollowUpColumns = []Column{
	colClaimNumber, colPolicyNumber, colInsuredName, colRegisterDate, colLossDate,
	colLGBNote, colTechnicalPIC, colNextFollowUp, colProgressStatus1, colProgressStatus2,
	colPosition, colRegisterDateAgain,
}

// perPICColumns adalah keenam kolom rekap per PIC.
//
// Empat dari enam judulnya menyebut atribut klaim padahal isinya pencacah. Judulnya tetap
// dibawa apa adanya; artinya ada di Description.
var perPICColumns = []Column{
	{Key: FieldPIC, Field: FieldPIC,
		Title: "PIC", Description: "Nama petugas"},
	{Key: FieldClaimCount, Field: FieldClaimCount,
		Title: "NOKLAIM", Description: "Jumlah klaim yang ditangani"},
	{Key: FieldUpdateCount, Field: FieldUpdateCount,
		Title: "NOAKSEP", Description: "Jumlah pembaruan progres, di luar yang bertanda AUTO"},
	{Key: FieldDueTodayCount, Field: FieldDueTodayCount,
		Title: "REINSURER", Description: "Tindak lanjut yang jatuh tempo hari ini"},
	{Key: FieldOnTimeCount, Field: FieldOnTimeCount,
		Title: "STSKLAIM", Description: "Tindak lanjut yang tepat waktu"},
	{Key: FieldLateCount, Field: FieldLateCount,
		Title: "NOPOLIS", Description: "Tindak lanjut yang terlambat"},
}

// views adalah keempat region dalam urutan tampilnya di layar lama.
var views = []View{
	{
		Code:           ViewOutstanding,
		Name:           "Outstanding",
		Description:    "Seluruh klaim yang masih berjalan, beserta posisi dan status progresnya.",
		Kind:           KindClaim,
		Columns:        outstandingColumns,
		Paginated:      true,
		SupportsSearch: true,
	},
	{
		Code: ViewNextFollowUp,
		Name: "Next Follow Up",
		Description: "Klaim yang tindak lanjutnya jatuh tempo hari ini atau sudah " +
			"terlewat — di layar lama berjudul \"CLAIM YANG HARUS DI FOLLOW UP HARI INI\".",
		Kind:           KindClaim,
		Columns:        nextFollowUpColumns,
		Paginated:      true,
		SupportsSearch: true,
	},
	{
		Code: ViewPerPIC,
		Name: "Progress Klaim per PIC",
		Description: "Rekap beban dan ketepatan tindak lanjut setiap petugas, " +
			"disaring menurut tanggal registrasi dan lini bisnis.",
		Kind:                   KindPIC,
		Columns:                perPICColumns,
		SupportsDateRange:      true,
		SupportsBusinessFilter: true,
		ScopedToCaller:         true,
	},
	{
		Code: ViewEvaluation,
		Name: "Evaluasi Progress Klaim",
		Description: "Bagian ini kosong di sistem lama — lihat keterangan di bawah " +
			"tabel.",
		Kind:    KindEmpty,
		Columns: []Column{},
	},
}

// Views mengembalikan keempat region dalam urutan tampilnya.
//
// Salinan, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah daftar region
// dengan menulisi hasilnya.
func Views() []View {
	result := make([]View, len(views))
	copy(result, views)
	return result
}

// FindView mencari region menurut kodenya.
func FindView(code string) (View, bool) {
	for _, view := range views {
		if view.Code == code {
			return view, true
		}
	}
	return View{}, false
}

// DeadControl adalah kontrol yang ADA di layar lama tetapi tidak menyaring apa pun.
//
// Ia ditulis sebagai data, bukan sebagai komentar yang hilang saat berkas ini dibaca
// sebagian, karena pertanyaan "kenapa penyaring ini tidak bekerja" pasti diajukan lagi.
type DeadControl struct {
	// View adalah kode region tempat kontrol itu berada.
	View string

	// Name adalah nama kontrol sebagaimana dibaca pengguna.
	Name string

	// Reason menjelaskan mengapa ia tidak menyaring, dalam kalimat yang dapat langsung
	// ditampilkan.
	Reason string
}

// DeadControls adalah kontrol yang digambar tetapi tidak berpengaruh pada hasil.
//
// # Kenapa tetap digambar
//
// Keputusan Work Owner 2026-09-21: kontrol ini direplikasi apa adanya — tampil, tetapi
// mati. Alternatifnya, membuatnya benar-benar bekerja, adalah PERUBAHAN PERILAKU yang akan
// memunculkan selisih pada uji kesetaraan gerbang 1 dan menuntut persetujuan tertulis
// sebagai butir `P-5` baru (`D-54`).
//
// # Bukti bahwa keduanya memang mati
//
//   - Kotak Tanggal Kejadian mengisi `TempRefresh.DateOfLoss`, dan properti itu **nol
//     kemunculan** di seluruh direktori `RDB List/`. Langkah terakhir
//     `GetDataProgressClaim` hanya menuliskannya kembali ke dirinya sendiri supaya isiannya
//     bertahan setelah layar disegarkan.
//   - Dropdown lini bisnis pada kedua region klaim mengisi `tempgetpic.CaseID`, dan
//     properti itu tidak dibaca satu pun kueri. Yang benar-benar dibaca
//     `DataProgressClaim` hanyalah `tempgetpic.MCL_NAME` dan `TempCabang.District`.
//
// Pada region per PIC, dropdown yang sama BENAR-BENAR menyaring — lihat
// `Activity/GetProgressPerPIC-Act.xml`, yang menuliskannya ke `TempBisnis.GROUP_PANEL`.
var DeadControls = []DeadControl{
	{
		View: ViewOutstanding,
		Name: "Tanggal Kejadian",
		Reason: "tidak menyaring apa pun di sistem lama — isiannya tidak pernah sampai " +
			"ke kueri",
	},
	{
		View: ViewOutstanding,
		Name: "Business",
		Reason: "tidak menyaring apa pun di sistem lama pada bagian ini; ia hanya " +
			"berlaku pada Progress Klaim per PIC dan pada Export",
	},
	{
		View: ViewNextFollowUp,
		Name: "Business",
		Reason: "tidak menyaring apa pun di sistem lama pada bagian ini; ia hanya " +
			"berlaku pada Progress Klaim per PIC dan pada Export",
	},
}

// DeadControlsFor mengembalikan kontrol mati pada satu region.
func DeadControlsFor(view string) []DeadControl {
	result := make([]DeadControl, 0, len(DeadControls))
	for _, control := range DeadControls {
		if control.View == view {
			result = append(result, control)
		}
	}
	return result
}
