package inboxcompliance

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke tab.
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
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan —
// ketiganya merujuk nama yang sama.
const (
	FieldCaseID             = "nomor_case"
	FieldClaimNumber        = "no_klaim"
	FieldPolicyNumber       = "no_polis"
	FieldInsuredName        = "nama_tertanggung"
	FieldBusinessName       = "nama_bisnis"
	FieldBranchName         = "nama_cabang"
	FieldAdminName          = "nama_admin"
	FieldComplianceSentDate = "tanggal_kirim_compliance"
	FieldPostAuditSentDate  = "tanggal_kirim_post_audit"
	FieldComplianceRemarks  = "catatan_compliance"
	FieldAging              = "aging"
	FieldOutstanding        = "outstanding"
)

// Kode tab.
//
// # Kenapa kodenya kata, bukan angka
//
// Berbeda dengan modul Inbox Admin — yang mempertahankan angka `3`, `7`, `9`, `11` karena
// angka itu muncul di prakondisi 34 langkah activity dan menjadi jalan telusur balik ke
// export — layar ini TIDAK punya properti pemilih tab sama sekali. Kedua tabnya adalah dua
// kontainer `TABBED` di dalam `Section/InputCompliance_Section-Section.xml` (`:985` dan
// `:2315`), dipilih peramban tanpa satu pun nilai yang dikirim ke server.
//
// Karena tidak ada angka yang perlu dipertahankan, yang dipakai adalah kata yang menyatakan
// isinya — dan ia ikut menjadi nilai parameter query yang terbaca di log.
const (
	TabCompliance = "compliance"
	TabPostAudit  = "post-audit"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Compliance, karena ia tab pertama di `InputCompliance_Section` (`:985`, sebelum Post Audit
// di `:2315`) dan satu-satunya yang sudah dapat dilayani.
const DefaultTab = TabCompliance

// Tab adalah satu antrean kerja pada layar Inbox Compliance.
type Tab struct {
	// Code adalah kode tab.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`. Nilainya diambil dari
	// `pyTitle` kontainer TABBED-nya.
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena judul sependek "Post Audit" tidak
	// memberi tahu apa pun tentang isinya.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti di Pega.
	Columns []Column

	// Available menyatakan tab ini sudah dapat dilayani.
	//
	// Tab yang belum dapat dilayani TETAP dikirim ke layar beserta Blocker-nya — bukan
	// disembunyikan. Menyembunyikannya membuat pengguna yang mencari "Post Audit" menduga
	// modulnya belum selesai, sementara yang sebenarnya kurang adalah satu artefak yang
	// pemiliknya jelas.
	Available bool

	// Blocker menyebut apa yang kurang dan siapa pemiliknya. Kosong bila Available.
	Blocker string
}

// tabs adalah kedua tab beserta kolomnya, berurutan seperti di
// `Section/InputCompliance_Section-Section.xml`.
//
// Urutan kolom tiap tab diambil dari urutan sel di section grid-nya, bukan dari urutan
// kolom Report Definition-nya — keduanya berbeda, dan yang dilihat pengguna adalah yang
// pertama.
var tabs = []Tab{
	{
		Code: TabCompliance,
		Name: "Compliance",
		Description: "Klaim yang menunggu pemeriksaan kepatuhan di antrean " +
			WorkbasketCompliance + " dan belum selesai.",

		// Kesembilan kolom `Section/InputComplianceDtl_Section-Section.xml`. Ketujuh
		// judul di bawah adalah `pyLabelFieldValue` section itu apa adanya; kolom
		// kedelapan adalah Aging, dan kolom kesembilan adalah tombol buka detail yang
		// tidak digambar sebagai kolom data.
		Columns: []Column{
			{Key: FieldCaseID, Title: "Nomor Case"},
			{Key: FieldPolicyNumber, Title: "No Polis"},
			{Key: FieldInsuredName, Title: "Nama Tertanggung"},
			{Key: FieldBusinessName, Title: "Nama Bisnis"},
			{Key: FieldBranchName, Title: "Nama Cabang"},
			{Key: FieldAdminName, Title: "Nama Admin"},
			{Key: FieldComplianceSentDate, Title: "Tanggal Kirim Compliance"},
			{Key: FieldAging, Title: "Aging"},
		},
		Available: true,
	},
	{
		Code: TabPostAudit,
		Name: "Post Audit",
		Description: "Pemeriksaan Post Audit yang baru dibuat dan belum " +
			"ditindaklanjuti.",

		// Ketujuh judul di bawah adalah nama `pyCaption` pada
		// `Section/InputPostAuditDtl_Section-Section.xml` APA ADANYA, dan urutannya urutan
		// sel di section itu (`:1694` … `:2480`). Keduanya cocok persis dengan layar Pega
		// yang berjalan.
		//
		// # Koreksi atas versi pertama berkas ini
		//
		// Versi pertama hanya memuat LIMA kolom, dan judulnya diambil dari label Report
		// Definition — "Case ID", "Compliance Remarks", "Tanggal Kirim Post Audit".
		// Ketiganya SALAH: yang dibaca pengguna adalah judul di section, bukan di RD.
		//
		// Sebabnya kesalahan itu layak dicatat: `pyLabelFieldValue` memang kosong di
		// section ini, sehingga pencarian pertama tidak menemukan apa pun dan jatuh ke RD
		// sebagai gantinya. Judulnya ternyata tersimpan sebagai `pyCaption`, dan baru
		// terlihat setelah dicari sebagai teks biasa.
		Columns: []Column{
			{Key: FieldCaseID, Title: "Nomor Case"},
			{Key: FieldClaimNumber, Title: "No Klaim"},
			{Key: FieldInsuredName, Title: "Nama Tertanggung"},
			{Key: FieldPolicyNumber, Title: "No Polis"},
			{Key: FieldComplianceRemarks, Title: "Catatan"},
			{Key: FieldPostAuditSentDate, Title: "Tanggal Kirim Audit Compliance"},
			{Key: FieldOutstanding, Title: "OutStanding"},
		},
		Available: true,
	},
}

// Tabs mengembalikan kedua tab dalam urutan tampilnya.
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
