package inboxpladlapredla

import "strings"

// Nama field JSON pada baris antrean.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar — sama halnya dengan
// nama field JSON lain (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	FieldClaimNo      = "no_klaim"
	FieldPolicyNo     = "no_polis"
	FieldInsured      = "nama_tertanggung"
	FieldRegisterDate = "tanggal_register"
	FieldLossDate     = "tanggal_kejadian"
	FieldPICTeknik    = "pic_teknik"
	FieldAdviceDate   = "tanggal_advice"
)

// FieldAttachmentKey adalah kunci berkas lampiran pada baris panel "Print Pre DLA".
//
// Ia satu-satunya field yang tidak punya pasangan di grid rincian; sisanya memakai nama
// yang sama karena artinya memang sama.
const FieldAttachmentKey = "kunci_lampiran"

// Nama field JSON pada baris grid rincian.
const (
	FieldAdviceNo     = "no_advice"
	FieldReinsurer    = "reasuradur"
	FieldAdviceType   = "tipe"
	FieldRevision     = "revisi"
	FieldDocDate      = "tanggal_dokumen"
	FieldSent         = "terkirim"
	FieldSentDate     = "tanggal_kirim"
	FieldReceivedDate = "tanggal_terima"
	FieldNotes        = "catatan"
	FieldEmail        = "email"
	FieldAcceptanceNo = "no_akseptasi"
)

// Column adalah satu kolom grid.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom, mengikuti judul layar Pega apa adanya (`D-13`).
	Title string

	// Date menandai kolom yang berisi TANGGAL, supaya layar memformatnya sebagai tanggal
	// WIB alih-alih menggambar cap waktu UTC mentah.
	//
	// Ia dinyatakan di sini, bukan ditebak layar dari isinya: dua kolom tanggal di grid
	// rincian beralias `"POLICY_NO"` dan `"CUSTOMER"` di Pega, dan menebak dari namanya
	// akan salah.
	Date bool
}

// AdviceKind adalah jenis dokumen yang diurus sebuah tab.
//
// Ia dipakai penyimpanan untuk memilih tabel dan penyaringnya. Ia BUKAN sekadar salinan
// kode tab: kode tab adalah kontrak ke layar dan boleh berubah kata-katanya, sedangkan
// ini menunjuk tabel di basis data.
type AdviceKind string

const (
	// KindPLA membaca `POOLDATA.T_PLALIST`.
	KindPLA AdviceKind = "pla"

	// KindDLA membaca `POOLDATA.T_DLALIST`.
	KindDLA AdviceKind = "dla"

	// KindPreDLA membaca `POOLDATA.T_PREDLALIST`.
	KindPreDLA AdviceKind = "pre-dla"
)

// SendVisibility adalah syarat tampilnya tombol **"SEND"** pada satu baris grid rincian.
//
// # Kenapa ia dinyatakan, bukan disimpulkan dari penanda terkirim
//
// Karena kedua tab TIDAK memakai syarat yang sama, dan perbedaannya terbaca langsung dari
// sectionnya masing-masing:
//
//	tab PLA  `Section/InboxPLA_sect-Section.xml:13238`  `.MARKETING != '1'`
//	tab DLA  `Section/InboxDLA_sect-Section.xml:11972`  `.MARKETING == ''`
//
// `MARKETING` adalah alias Pega untuk `ISKIRIM` — `GetPLAList-SQL.xml` menuliskannya
// `iskirim AS MARKETING`, dan `GetDLAList-SQL.xml` sama. Aliasnya tidak menyatakan isinya
// sama sekali: ia penanda terkirim, bukan apa pun tentang pemasaran.
//
// Keduanya berbeda pada SATU nilai, dan nilai itu benar-benar ada di data: `ISKIRIM = '0'`.
// Di tab PLA barisnya masih bertombol; di tab DLA tidak. Perbedaan itu dibawa apa adanya
// (`P-5`) — lihat catatan pada SendWhenSentFlagEmpty.
type SendVisibility string

const (
	// SendNever berarti tab ini tidak punya tombol "SEND" sama sekali.
	//
	// Ia nilai NOL tipe ini, sehingga tab yang tidak menyebutkannya — Pre DLA — tidak
	// diam-diam mewarisi syarat tab lain.
	SendNever SendVisibility = ""

	// SendWhenNotMarkedSent menampilkan tombol selama `ISKIRIM` BUKAN `'1'`.
	//
	// **Tab PLA.** Kosong, `'0'`, dan nilai lain apa pun tetap bertombol.
	SendWhenNotMarkedSent SendVisibility = "bukan-satu"

	// SendWhenSentFlagEmpty menampilkan tombol HANYA ketika `ISKIRIM` kosong.
	//
	// **Tab DLA.** `ISKIRIM = '0'` TIDAK bertombol di sini, padahal penyaring antreannya
	// sendiri menerima `'0'` sebagai "belum terkirim" — sehingga baris seperti itu masuk
	// antrean tetapi tidak dapat dikirim dari layar ini.
	//
	// Itu kejanggalan Pega, bukan kekeliruan pembacaan. Ia dibawa apa adanya karena
	// menyeragamkannya mengubah dokumen mana yang dapat dikirim ke pihak LUAR perusahaan
	// — perubahan yang menuntut keputusan Work Owner, bukan penyeragaman sepihak.
	SendWhenSentFlagEmpty SendVisibility = "kosong"
)

// Tab adalah satu daftar beserta kolom dan penyaringnya.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul yang dibaca pengguna, mengikuti caption Pega apa adanya (`D-13`).
	Name string

	// Description menjelaskan isi daftarnya dalam satu kalimat.
	//
	// Sistem lama tidak punya keterangan seperti ini, dan ketiadaannya nyata akibatnya di
	// layar ini: tiga tab bernama PLA, DLA, dan Pre DLA, dan tidak ada apa pun yang
	// menjelaskan mengapa sebuah klaim ada di yang satu dan tidak di yang lain — padahal
	// satu klaim memang bisa ada di ketiganya sekaligus.
	Description string

	// Kind menunjuk tabel dokumen yang dibaca tab ini.
	Kind AdviceKind

	// Columns adalah kolom grid antrean, berurutan seperti tampilnya.
	Columns []Column

	// DocumentColumns adalah kolom grid rincian di bawahnya.
	//
	// Kosong berarti tab ini TIDAK punya grid rincian — dan itu memang terjadi pada tab
	// Pre DLA. Lihat HasDocuments.
	DocumentColumns []Column

	// SearchLabel adalah judul kotak pencarian, sama di ketiga tab (`pyCaption No Klaim`).
	SearchLabel string

	// DateLabel adalah judul rentang tanggal yang disaring tab ini.
	//
	// Ia BERBEDA per tab meski kotaknya sama-sama berjudul "Dari"/"Sampai" di Pega, karena
	// kolom yang disaringnya berbeda: `TGLPLA` pada PLA, `TGLDLA` pada DLA dan Pre DLA.
	// Menyebutnya membuat pengguna tahu tanggal APA yang sedang ia batasi — di Pega ia
	// harus menebaknya.
	DateLabel string

	// ExcludeASNET menyatakan tab ini mengecualikan `BRANCHNAME = 'ASNET'`.
	//
	// **Tab DLA saja.** Penyaring ini tidak punya pasangan di tab lain, dan alasan
	// bisnisnya tidak tertulis di mana pun di export. Ia dibawa apa adanya (`P-5`).
	ExcludeASNET bool

	// ExcludeBusinessGroup adalah `BUSINESSGROUPID` yang dikecualikan lewat
	// `POOLDATA.BUSINESS`.
	//
	// **Tab DLA saja**, bernilai `10008`. Kosong berarti tab ini tidak mengecualikan
	// kelompok bisnis mana pun.
	ExcludeBusinessGroup string

	// SentFilterApplies menyatakan keanggotaan baris ditentukan penanda TERKIRIM.
	//
	// Benar pada PLA dan DLA (`ISKIRIM IS NULL OR ISKIRIM = '0'`). **Salah pada Pre DLA**,
	// yang keanggotaannya ditentukan `NOAKSEP IS NULL` dan tidak menyinggung `ISKIRIM`
	// sama sekali — lihat catatan di kepala inboxpladlapredla.go.
	SentFilterApplies bool

	// RequiresReinsurer menyatakan dokumennya wajib sudah punya kode reasuradur.
	//
	// **Tab PLA saja** (`a.reinscode is not null`). PLA yang belum menunjuk reasuradur
	// belum dapat dikirim ke siapa pun, sehingga ia belum menjadi pekerjaan.
	RequiresReinsurer bool

	// HasPrintAction menyatakan setiap BARIS daftar ini punya tombol
	// **"Print Pre DLA"**.
	//
	// **Tab Pre DLA saja**, dan ia tombol PER BARIS — bukan satu tombol di bawah grid.
	// Itu terbaca dari letaknya di dalam berkas section: labelnya berada di dalam blok
	// `pyBodyType = REPEATING` milik gridnya, berdampingan dengan kolom `BRANCH_NAME`
	// (kunci klaim) dan `pyLocalAction = PNCInboxPrintPreDLA`.
	//
	// Baris itulah yang memasok kunci klaim ke modalnya; tanpa baris, panelnya tidak
	// punya klaim untuk ditampilkan.
	HasPrintAction bool

	// PrintColumns adalah kolom panel "Print Pre DLA".
	//
	// Kosong berarti daftar ini tidak punya panel itu. Ia dipisahkan dari
	// DocumentColumns karena keduanya membaca TABEL yang berbeda dan dibuka tombol yang
	// berbeda — menyatukannya akan membuat satu senarai kolom melayani dua panel yang
	// tidak pernah tampil bersamaan.
	PrintColumns []Column

	// RowActionLabel adalah judul tombol aksi pada setiap baris.
	//
	// **KOSONG pada PLA dan DLA**: keduanya tidak punya kolom aksi sama sekali —
	// rinciannya dibuka dengan MENGKLIK NOMOR KLAIM.
	//
	// Itu perilaku Pega, dan terbaca dari `Section/InboxPLA_sect-Section.xml:171096`:
	// sel `.BRANCH_CODE` membawa `pyAction = refresh` beserta parameter `caseId`
	// bernilai `.BRANCH_NAME` — menyegarkan grid rincian dengan kunci klaim baris itu.
	// Nomor klaimnya memang tautan di sana, bukan teks biasa.
	//
	// Terisi hanya pada Pre DLA, yang punya tombol bernama di dalam barisnya.
	//
	// Layar mengetahui nomor klaim dapat diklik dari HasDocuments, BUKAN dari medan ini:
	// yang membuka rincian adalah keberadaan rinciannya, bukan ada-tidaknya tombol.
	RowActionLabel string

	// SendVisible adalah syarat tampilnya tombol "SEND" pada baris grid rincian.
	//
	// Kosong (SendNever) berarti daftar ini tidak punya tombol itu — Pre DLA, yang
	// gridnya pun tidak ada.
	SendVisible SendVisibility

	// RequiresNoAcceptance menyatakan keanggotaan baris ditentukan `NOAKSEP IS NULL`.
	//
	// **Tab Pre DLA saja.** Ia dinyatakan sebagai isian TERSENDIRI, bukan disimpulkan
	// dari SentFilterApplies yang kebetulan salah, karena keduanya menjawab pertanyaan
	// yang berbeda: yang satu "apakah surat sudah dikirim", yang lain "apakah
	// akseptasinya sudah terbit". Menyimpulkan yang satu dari yang lain akan membuat tab
	// berikutnya yang tidak menyaring `ISKIRIM` diam-diam mewarisi penyaring akseptasi
	// yang tidak pernah dimintanya.
	RequiresNoAcceptance bool
}

// HasDocuments menyatakan tab ini menggambar grid rincian di bawah antreannya.
//
// Tab Pre DLA TIDAK, dan itu bukan kelalaian pembacaan:
// `Section/PNCInboxPreDLA_sect-Section.xml` hanya memuat SATU grid
// (`TempPNCList.pxResults`), sementara kedua section lain memuat dua. Rincian Pre-DLA ada
// di panel tersendiri yang dibuka tombol "Print Pre DLA" pada barisnya — lihat
// PrintColumns.
func (t Tab) HasDocuments() bool {
	return len(t.DocumentColumns) > 0
}

// ShowSendButton menyatakan baris ber-`ISKIRIM` ini digambar bertombol "SEND".
//
// # Ia DIHITUNG peladen, bukan disimpulkan layar
//
// Syaratnya hasil pembacaan section Pega, dan tempat pembacaan itu tercatat adalah paket
// ini — sama halnya dengan daftar kolom. Layar yang menyimpulkannya sendiri dari
// `terkirim` akan menyamakan kedua tab, dan penyamaan itu tidak akan menghasilkan satu
// pun galat: tombolnya muncul pada baris DLA yang di Pega tidak pernah bertombol.
//
// # Ia BUKAN pemeriksaan boleh-tidaknya mengirim
//
// Tombol yang tampil masih dapat ditolak — alamat reasuradur yang kosong dijawab
// ErrReinsurerEmailEmpty, persis seperti Pega menjawab "Email Reinsurer Kosong". Pega pun
// tetap menggambar tombolnya pada baris beralamat kosong, dan itu benar: yang harus
// dilakukan pengguna adalah melengkapi alamatnya, dan ia tidak akan tahu itu dari tombol
// yang menghilang.
func (t Tab) ShowSendButton(sent string) bool {
	switch t.SendVisible {
	case SendWhenNotMarkedSent:
		return strings.TrimSpace(sent) != "1"
	case SendWhenSentFlagEmpty:
		return strings.TrimSpace(sent) == ""
	default:
		return false
	}
}

// DefaultTab adalah tab yang terbuka pertama kali.
//
// PLA, mengikuti urutan tab di `Section/InboxPLADLA_sect-Section.xml` — dan mengikuti
// urutan proses: PLA terbit lebih dulu, Pre-DLA menyusul, DLA terakhir.
const DefaultTab = "pla"

// queueColumns adalah kolom grid antrean, SAMA di ketiga tab kecuali judul kolom tanggal
// advice-nya.
//
// Ia fungsi, bukan variabel, supaya setiap tab memegang senarai kolomnya sendiri. Senarai
// bersama yang dipakai tiga tab dapat diubah salah satu pemakainya dan diam-diam mengubah
// dua yang lain.
func queueColumns(adviceTitle string) []Column {
	return []Column{
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldPolicyNo, Title: "No Polis"},
		{Key: FieldInsured, Title: "Nama Tertanggung"},
		{Key: FieldRegisterDate, Title: "Tanggal Register", Date: true},
		{Key: FieldLossDate, Title: "Tanggal Kejadian", Date: true},
		{Key: FieldPICTeknik, Title: "PIC Teknik"},
		{Key: FieldAdviceDate, Title: adviceTitle, Date: true},
	}
}

// tabs adalah ketiga daftar yang digambar layar.
//
// Urutannya mengikuti urutan tab di `Section/InboxPLADLA_sect-Section.xml`: PLA, DLA,
// Pre DLA. Perhatikan bahwa urutan itu BUKAN urutan prosesnya — Pre-DLA terbit sebelum
// DLA. Urutan layar dibawa apa adanya (`D-13`).
var tabs = []Tab{
	{
		Code: "pla",
		Name: "PLA",
		Description: "Preliminary Loss Advice yang sudah terbit, sudah menunjuk " +
			"reasuradur, tetapi belum dikirim. PLA memberitahukan nilai ESTIMASI klaim " +
			"kepada koasuransi/reasuransi.",
		Kind:              KindPLA,
		Columns:           queueColumns("Tanggal PLA"),
		SearchLabel:       "No Klaim",
		DateLabel:         "Tanggal PLA",
		SentFilterApplies: true,
		RequiresReinsurer: true,

		// `Section/InboxPLA_sect-Section.xml:13238` -> `.MARKETING != '1'`.
		SendVisible: SendWhenNotMarkedSent,

		// Urutannya mengikuti grid "Detail PLA List" di Pega apa adanya (`D-13`):
		//
		//	No PLA · PLA REINSURER · PLA TYPE · REVISION · PLA DATE ·
		//	TGL Terima PLA · SENT DATE · Email · File Penunjang
		//
		// Perhatikan TGL Terima mendahului SENT DATE, bukan sebaliknya. Judulnya
		// diterjemahkan karena seluruh judul modul ini berbahasa Indonesia, tetapi
		// URUTANNYA tidak diubah — petugas membaca grid ini berdampingan dengan layar
		// lama selama masa paralel.
		DocumentColumns: []Column{
			{Key: FieldAdviceNo, Title: "No PLA"},
			{Key: FieldReinsurer, Title: "Reasuradur"},
			{Key: FieldAdviceType, Title: "Tipe"},
			{Key: FieldRevision, Title: "Revisi"},
			{Key: FieldDocDate, Title: "Tanggal PLA", Date: true},
			{Key: FieldReceivedDate, Title: "Tanggal Terima", Date: true},
			{Key: FieldSentDate, Title: "Tanggal Kirim", Date: true},

			// DITAMBAHKAN — Pega tidak punya kolom ini; penanda terkirimnya hanya
			// terbaca dari ada-tidaknya SENT DATE. Letaknya tepat di sebelah
			// "Tanggal Kirim" karena keduanya menyatakan fakta yang sama.
			{Key: FieldSent, Title: "Terkirim"},

			{Key: FieldEmail, Title: "Email"},
			{Key: FieldNotes, Title: "Catatan"},
		},
	},
	{
		Code: "dla",
		Name: "DLA",
		Description: "Definite Loss Advice yang sudah terbit tetapi belum dikirim. DLA " +
			"memberitahukan nilai AKSEPTASI klaim. Daftar ini mengecualikan cabang ASNET " +
			"dan kelompok bisnis 10008 — dua pengecualian yang tidak berlaku di tab lain.",
		Kind:                 KindDLA,
		Columns:              queueColumns("Tanggal DLA"),
		SearchLabel:          "No Klaim",
		DateLabel:            "Tanggal DLA",
		ExcludeASNET:         true,
		ExcludeBusinessGroup: "10008",
		SentFilterApplies:    true,

		// `Section/InboxDLA_sect-Section.xml:11972` -> `.MARKETING == ''`.
		//
		// Syaratnya LEBIH KETAT daripada tab PLA, dan itu bukan salah salin. Lihat
		// SendWhenSentFlagEmpty.
		SendVisible: SendWhenSentFlagEmpty,

		// Urutannya mengikuti grid "Detail DLA List" di Pega apa adanya (`D-13`):
		//
		//	NO DLA · DLA REINSURER · DLA TYPE · DLA DATE · Tgl Terima DLA ·
		//	SENT DATE · EMAIL · File Penunjang
		//
		// "No Akseptasi" DITAMBAHKAN — `GetDLAList` mengambil `noaksep` tetapi grid Pega
		// tidak menggambarnya. Ia ditaruh sebelum tanggal karena ia pengenal, bukan
		// keterangan waktu.
		DocumentColumns: []Column{
			{Key: FieldAdviceNo, Title: "No DLA"},
			{Key: FieldReinsurer, Title: "Reasuradur"},
			{Key: FieldAdviceType, Title: "Tipe"},
			{Key: FieldAcceptanceNo, Title: "No Akseptasi"},
			{Key: FieldDocDate, Title: "Tanggal DLA", Date: true},
			{Key: FieldReceivedDate, Title: "Tanggal Terima", Date: true},
			{Key: FieldSentDate, Title: "Tanggal Kirim", Date: true},
			{Key: FieldSent, Title: "Terkirim"},
			{Key: FieldEmail, Title: "Email"},
			{Key: FieldNotes, Title: "Catatan"},
		},
	},
	{
		Code: "pre-dla",
		Name: "Pre DLA",
		Description: "Pre-DLA yang belum punya Nomor Akseptasi. Ia memberitahukan nilai " +
			"yang AKAN diakseptasi. Berbeda dari dua tab lain, yang mengeluarkan baris " +
			"dari daftar ini adalah terbitnya Nomor Akseptasi — bukan terkirimnya surat.",
		Kind:                 KindPreDLA,
		Columns:              queueColumns("Tanggal Pre DLA"),
		SearchLabel:          "No Klaim",
		DateLabel:            "Tanggal Pre DLA",
		RequiresNoAcceptance: true,
		HasPrintAction:       true,
		RowActionLabel:       "Print Pre DLA",

		// Keempat kolom panel "Print Pre DLA", mengikuti caption
		// `Section/PrintPreDLA-Section.xml` apa adanya — termasuk yang ditulis KAPITAL
		// di sana (`D-13`).
		//
		// Kolom "Tgl Kirim" memakai FieldSentDate, BUKAN FieldDocDate. Keduanya pernah
		// tertukar di sini, dan tertukarnya tidak menghasilkan satu pun galat:
		// PreDLADocumentDTO memang tidak punya `tanggal_dokumen`, sehingga selnya
		// mengambil medan yang tidak ada dan menggambar tanda hubung pada SETIAP baris.
		// Panelnya terlihat utuh; hanya tanggalnya yang tidak pernah muncul.
		//
		// Kuerinya sendiri mengambil `TGLKIRIM` (beralias `"TglDLA"` di Pega — alias
		// yang menyesatkan, dan justru itu yang membuat keduanya mudah tertukar).
		PrintColumns: []Column{
			{Key: FieldAdviceNo, Title: "NO DLA"},
			{Key: FieldReinsurer, Title: "DLA REINSURER"},
			{Key: FieldAdviceType, Title: "TIPE DLA"},
			{Key: FieldSentDate, Title: "Tgl Kirim", Date: true},
			{Key: FieldSent, Title: "Terkirim"},
		},

		// SentFilterApplies sengaja SALAH, dan DocumentColumns sengaja kosong. Keduanya
		// perbedaan nyata tab ini terhadap dua tab lain; lihat catatan di kepala
		// inboxpladlapredla.go.
	},
}

// Tabs mengembalikan salinan ketiga daftar.
//
// Salinan, bukan senarai aslinya: pemanggil yang mengubah isinya tidak boleh mengubah
// bentuk layar bagi pemanggil berikutnya.
func Tabs() []Tab {
	out := make([]Tab, len(tabs))
	copy(out, tabs)
	return out
}

// FindTab mencari tab menurut kodenya.
//
// Kode dicocokkan tanpa memedulikan huruf besar-kecil dan spasi di ujung: ia datang dari
// alamat, dan `?daftar=PLA` maupun `?daftar=pla ` adalah permintaan yang sama.
func FindTab(code string) (Tab, bool) {
	clean := strings.ToLower(strings.TrimSpace(code))
	for _, tab := range tabs {
		if tab.Code == clean {
			return tab, true
		}
	}
	return Tab{}, false
}
