package inboxpladla

import "strings"

// Nama field JSON pada baris daftar.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar (`D-80`).
const (
	FieldClaimNo      = "no_klaim"
	FieldPolicyNo     = "no_polis"
	FieldInsured      = "nama_tertanggung"
	FieldBusinessName = "nama_bisnis"
	FieldRegisterDate = "tanggal_register"
	FieldLossDate     = "tanggal_kejadian"
	FieldPICTeknik    = "pic_teknik"
	FieldStatus       = "status"
	FieldAdviceNo     = "no_pla"
	FieldCloseNote    = "catatan_tutup"
)

// Column adalah satu kolom grid.
type Column struct {
	Key   string
	Title string

	// Date menandai kolom yang berisi tanggal, supaya layar memformatnya sebagai tanggal
	// WIB alih-alih menggambar teksnya apa adanya.
	Date bool
}

// Source menyatakan TABEL ASAL sebuah daftar klaim.
//
// Keenam daftar klaim tidak berangkat dari penyaring yang sama. Tiga berangkat dari
// dokumen pemberitahuan yang sudah terkirim; tiga lagi berangkat dari percakapan pada
// `POOLDATA.M_KOMUNIKASI_PNC`. Memaksakan keduanya ke dalam satu rangkaian penanda akan
// menghasilkan kombinasi yang tidak pernah ada di Pega — misalnya "daftar komunikasi yang
// juga menuntut PLA terkirim".
type Source string

const (
	// SourceAdvice menyaring lewat dokumen pemberitahuan yang SUDAH terkirim.
	//
	//	GetPNCList_PLA1 · GetPNCList_PLADLA · GetPNCList_PLADLAClose
	SourceAdvice Source = "pemberitahuan"

	// SourceCommunication menyaring lewat percakapan pada `M_KOMUNIKASI_PNC`.
	//
	//	BrowseCommunicationReas — dipakai tipe 4, 5, dan 6
	SourceCommunication Source = "komunikasi"
)

// CommunicationRole menyatakan SISI MANA percakapan yang dihitung milik pemanggil.
//
// `Activity/SetDataPLADLA-Act.xml` menyusunnya sebagai potongan SQL yang berbeda per tipe:
//
//	tipe 4  "and c.COMMUNICATE_TO='" + <pemanggil> + "'"
//	tipe 5  "and c.sender='"         + <pemanggil> + "'"
//	tipe 6  "and c.sender='"         + <pemanggil> + "'"
//
// Di sini ia menjadi penanda, dan nilainya DIIKAT — bukan dirangkai.
type CommunicationRole string

const (
	// RoleRecipient — percakapan yang DITUJUKAN kepada pemanggil (`COMMUNICATE_TO`).
	RoleRecipient CommunicationRole = "penerima"

	// RoleSender — percakapan yang DIKIRIM pemanggil (`SENDER`).
	RoleSender CommunicationRole = "pengirim"
)

// Nilai `M_KOMUNIKASI_PNC.KOMUNIKASISTATUS`.
//
// Artinya diambil dari modul `inboxkomunikasicabang`, yang membaca tabel yang SAMA dan
// sudah menetapkannya lebih dulu: `0` belum dijawab, `1` sudah dijawab.
const (
	CommunicationNotAnswered = "0"
	CommunicationAnswered    = "1"
)

// Tab adalah satu tampilan beserta penyaringnya.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul yang dibaca pengguna.
	//
	// Ia diambil APA ADANYA dari layar Pega (`D-13`). Ketiga judul daftar pemberitahuan
	// terbaca dari `Section/InboxDLAReas_sect-Section.xml` sebagai label bersyarat:
	//
	//	TempView.CityID=='1'  ->  "PLA"
	//	TempView.CityID=='2'  ->  "PLA & DLA"
	//	TempView.CityID=='3'  ->  "CLOSE CLAIM"
	//
	// Ketiga daftar komunikasi TIDAK punya label di export — lihat catatan pada tabs.
	Name string

	// Description menjelaskan isi daftarnya dalam satu kalimat.
	//
	// Layar Pega tidak punya keterangan seperti ini, dan ketiadaannya nyata akibatnya:
	// keenam daftar menampilkan klaim yang saling bertumpang tindih, dan tidak ada apa pun
	// yang menjelaskan mengapa sebuah klaim ada di yang satu dan tidak di yang lain.
	Description string

	// Columns adalah kolom grid, berurutan seperti tampilnya.
	Columns []Column

	// Source menyatakan tabel asal daftarnya.
	Source Source

	// AdviceKindSent menyatakan dokumen JENIS APA yang harus sudah terkirim kepada
	// reasuradur pemanggil supaya klaimnya masuk daftar ini.
	//
	//	tab PLA          "pla"  <- T_PLALIST
	//	tab PLA & DLA    "dla"  <- T_DLALIST
	//	tab CLOSE CLAIM  "pla"  <- T_PLALIST      (!) BUKAN DLA
	//
	// Baris ketiga itu mudah dikira salah ketik. `GetPNCList_PLADLAClose` memang
	// menyaring `t_plalist`, bukan `t_dlalist`, meski ia menampilkan klaim yang sudah
	// selesai. Alasannya tidak tertulis di mana pun, dan ia dibawa apa adanya (`P-5`).
	//
	// KOSONG pada ketiga daftar komunikasi: `BrowseCommunicationReas` tidak memeriksa
	// dokumen pemberitahuan sama sekali.
	AdviceKindSent string

	// ExcludeWhenDLASent menyatakan klaim yang DLA-nya sudah terkirim DIKELUARKAN.
	//
	// **Tab PLA saja.** Itu yang membuat ketiga daftar pemberitahuan tidak saling
	// bertumpuk: begitu DLA terkirim, klaimnya berpindah dari daftar PLA ke daftar
	// PLA & DLA.
	ExcludeWhenDLASent bool

	// WorkStatus menyatakan keadaan alur kerja yang diterima daftar ini.
	WorkStatus WorkStatusFilter

	// ExcludeGroupPanels adalah lini bisnis yang DIKELUARKAN daftar ini.
	//
	// Ketiga daftar pemberitahuan mengecualikan `002` (Personal Accident) dan `005`
	// (Travel). Ketiga daftar komunikasi TIDAK mengecualikan apa pun — dan itu BUKAN
	// kelalaian pembacaan: `BrowseCommunicationReas` memang tidak memuat satu pun syarat
	// `grouppanel`.
	//
	// Akibatnya nyata dan harus disadari: sebuah klaim PA dapat muncul di daftar
	// komunikasi sementara ia tidak akan pernah muncul di ketiga daftar pemberitahuan.
	ExcludeGroupPanels []string

	// CommunicationStatus adalah nilai `KOMUNIKASISTATUS` yang diterima daftar ini.
	// Kosong pada daftar yang bukan daftar komunikasi.
	CommunicationStatus string

	// CommunicationRole menyatakan sisi mana percakapan yang dihitung milik pemanggil.
	CommunicationRole CommunicationRole

	// AllReinsurerCodes menyatakan daftar ini mencocokkan SELURUH kode reasuradur milik
	// login pemanggil, bukan yang tertinggi saja.
	//
	// **Tab CLOSE CLAIM saja** (`reinscode in (…)`). Kedua tab pemberitahuan lain memakai
	// `reinscode = (… FETCH NEXT 1 ROW ONLY)`. Perbedaan itu ada di kueri Pega dan
	// alasannya tidak tertulis; menyeragamkannya akan mengubah isi daftar.
	AllReinsurerCodes bool

	// PendingCloseBecomes1139 menyatakan kode status diganti `1139` ketika klaimnya
	// menunggu penutupan.
	//
	// **Tab PLA & DLA saja.** `CASE WHEN z.ISPENDINGCLOSE='true' THEN '1139' ELSE
	// z.statusclaim_1 END` di Pega; kini kedua kolom dibaca dari `T_CLAIM_PNC` (SUMBER BARU
	// 2026-10-08). Daftar lain memakai kode aslinya.
	PendingCloseBecomes1139 bool

	// HasDetailAction menyatakan barisnya punya tombol **"Detail Claim"**.
	//
	// Ia datang dari SERVER, bukan dicocokkan di layar, dengan alasan yang sama seperti
	// senarai kolom: inventaris tombol adalah hasil pembacaan export.
	HasDetailAction bool
}

// WorkStatusFilter menyatakan keadaan alur kerja yang diterima sebuah daftar.
type WorkStatusFilter string

const (
	// WorkOpen menerima klaim yang BELUM selesai.
	//
	// `STATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')`.
	WorkOpen WorkStatusFilter = "terbuka"

	// WorkOpenOrPendingClose menerima klaim yang belum selesai, DITAMBAH klaim yang
	// sudah `Resolved-Completed` tetapi masih menunggu penutupan.
	//
	// Tab PLA & DLA. Klaim menunggu penutupan itulah yang kode statusnya diganti `1139`.
	WorkOpenOrPendingClose WorkStatusFilter = "terbuka-atau-menunggu-tutup"

	// WorkClosed menerima klaim yang sudah `Resolved-Completed` dan TIDAK menunggu
	// penutupan.
	//
	// Perhatikan `Resolved-Rejected` TIDAK termasuk: klaim yang ditolak tidak muncul di
	// daftar mana pun pada layar ini. Itu perilaku Pega, dan akibatnya klaim yang sudah
	// dikirimi PLA lalu ditolak menghilang dari pandangan reasuradur tanpa satu pun
	// pemberitahuan.
	WorkClosed WorkStatusFilter = "tutup"
)

// ClosedWorkStatuses adalah kedua nilai `STATUSWORK` yang berarti klaimnya sudah selesai.
//
// Keduanya ditulis di sini, bukan di dalam SQL, supaya penyimpanan memori dan penyimpanan
// SQL tidak dapat memakai daftar yang berbeda.
var ClosedWorkStatuses = []string{"Resolved-Completed", "Resolved-Rejected"}

// CompletedWorkStatus adalah nilai `STATUSWORK` yang dipakai tab CLOSE CLAIM.
const CompletedWorkStatus = "Resolved-Completed"

// PendingCloseStatusCode adalah kode status yang menggantikan kode asli pada tab PLA & DLA
// ketika klaimnya menunggu penutupan.
const PendingCloseStatusCode = "1139"

// ExcludedGroupPanels adalah lini bisnis yang dikecualikan ketiga daftar pemberitahuan.
//
//	002  Personal Accident
//	005  Travel
var ExcludedGroupPanels = []string{"002", "005"}

// Kode keenam daftar klaim.
//
// Kodenya mengikuti JUDULNYA, bukan penyaringnya. Kode yang menggambarkan penyaring
// (`komunikasi-masuk`) dan judul yang menggambarkan keadaan (`NOT ANSWERED`) akan selalu
// terbaca seperti dua hal yang berbeda — dan yang tertulis di alamat adalah kodenya.
const (
	TabPLA      = "pla"
	TabPLADLA   = "dla"
	TabClose    = "close"
	TabInbound  = "not-answered"
	TabOutbound = "not-replied-from-asm"
	TabAnswered = "replied-from-asm"
)

// DefaultTab adalah tampilan yang terbuka pertama kali.
//
// PLA — tahap paling awal dari sudut pandang reasuradur: ia baru menerima pemberitahuan
// estimasi dan belum menerima pemberitahuan akseptasi.
//
// # Ini SELISIH terhadap Pega, dan disengaja
//
// Di Pega `TempView.CityID` mulai KOSONG, dan seluruh wadah isinya bersyarat:
//
//	TempView.CityID!=''
//
// Artinya layar Pega mula-mula menggambar tabel ringkas saja —
// tidak satu pun daftar — sampai pengguna menekan salah satu angkanya.
//
// Membuka tab pertama secara langsung menghemat satu klik yang tidak menyampaikan apa pun,
// dan tidak menyembunyikan apa pun: keenam tab tetap terlihat di bilahnya.
const DefaultTab = TabPLA

// listColumnsFor menyusun kolom daftar klaim.
//
// Ia fungsi, bukan variabel bersama, supaya setiap tab memegang senarai kolomnya sendiri.
// Senarai bersama yang dipakai enam tab dapat diubah salah satu pemakainya dan diam-diam
// mengubah lima yang lain.
//
// # Judul dan urutannya diambil APA ADANYA dari Pega
//
// Dibaca dari header grid pada `Section/InboxDLAReas_sect-Section.xml`:
//
//	Claim No · Insured · Policy No · Business Name · DOL · Register Date ·
//	PIC ASM · PLA No · Claim Progress
//
// Judulnya berbahasa Inggris, dan itu bukan kelalaian melainkan `D-13`: pengguna layar ini
// sudah mengenalnya. Menerjemahkannya berarti menambah satu hal yang harus dipelajari
// ulang tanpa menambah kejelasan — dan pembacanya pihak LUAR, yang tidak dapat kita latih.
//
// Perhatikan dua judul yang menyesatkan dan tetap dibawa:
//
//	"DOL"           berisi TANGGAL KEJADIAN — singkatan Date Of Loss, bukan kolom `DOL`
//	"Claim Progress" berisi STATUS KLAIM, bukan tahapan progres (`GCNM_PROGRESS_CLAIM`)
//
// Yang kedua bertabrakan dengan istilah **Status Posisi Progres** di `CONTEXT.md`. Ia
// tetap ditulis begitu di kolomnya, dan bedanya dijelaskan di keterangan tab.
func listColumnsFor() []Column {
	return []Column{
		{Key: FieldClaimNo, Title: "Claim No"},
		{Key: FieldInsured, Title: "Insured"},
		{Key: FieldPolicyNo, Title: "Policy No"},
		{Key: FieldBusinessName, Title: "Business Name"},
		{Key: FieldLossDate, Title: "DOL", Date: true},
		{Key: FieldRegisterDate, Title: "Register Date", Date: true},
		{Key: FieldPICTeknik, Title: "PIC ASM"},
		{Key: FieldAdviceNo, Title: "PLA No"},
		{Key: FieldStatus, Title: "Claim Progress"},
	}
}

// tabs adalah ketujuh tampilan yang digambar layar.
//
// # Urutannya mengikuti `param.tipe` pada `SetDataPLADLA`
//
//	tipe 1  PLA                 GetPNCList_PLA1
//	tipe 2  PLA & DLA           GetPNCList_PLADLA
//	tipe 3  CLOSE CLAIM         GetPNCList_PLADLAClose
//	tipe 4  komunikasi masuk    BrowseCommunicationReas  status 0, COMMUNICATE_TO
//	tipe 5  komunikasi keluar   BrowseCommunicationReas  status 0, SENDER
//	tipe 6  komunikasi dijawab  BrowseCommunicationReas  status 1, SENDER
//
// tipe 7 TIDAK dibawa: ia tampilan XOL, dan wadahnya di Pega bersyarat
// `TempView.CityID==7` yang tidak pernah dapat tercapai. Lihat Tabs().
//
// # Judul ketiga daftar komunikasi datang dari WORK OWNER, bukan dari export
//
// Ketiganya dipilih lewat tabel ringkas yang berupa TreeGrid (`TempPLADLA.pxResults`), dan
// rule yang MENGISI halaman itu tidak ada di export mana pun — tidak di section, tidak di
// harness, tidak di `SetDataPLADLA`. Yang terbaca dari export hanyalah nomor tipenya.
//
// Judulnya karena itu sempat DISUSUN dari penyaringnya sendiri, dan selisih itu dinyatakan
// di PlannedDifferences. Work Owner menyebutkan ketiganya pada 2026-09-28 —
// **NOT ANSWERED · NOT REPLIED FROM ASM · REPLIED FROM ASM** — sehingga judulnya kini
// sama dengan layar lama dan selisih itu dicabut.
//
// Urutan yang disebutkan Work Owner cocok dengan urutan tipe pada `SetDataPLADLA`, dan
// artinya cocok dengan penyaring masing-masing:
//
//	tipe 4  NOT ANSWERED          pesan UNTUK saya, belum SAYA jawab
//	tipe 5  NOT REPLIED FROM ASM  pesan DARI saya, belum dijawab ASM
//	tipe 6  REPLIED FROM ASM      pesan DARI saya, sudah dijawab ASM
//
// Kecocokan itu diperiksa, bukan diterima begitu saja: "NOT ANSWERED" tanpa keterangan
// pihak berarti yang belum menjawab adalah PEMBACANYA, sementara kedua judul lain menyebut
// ASM secara eksplisit. Itu persis pembagian `COMMUNICATE_TO` versus `SENDER`.
var tabs = []Tab{
	{
		Code: TabPLA,
		Name: "PLA",
		Description: "Klaim yang PLA-nya sudah dikirimkan kepada Anda, tetapi DLA-nya " +
			"belum. Anda sudah diberi tahu nilai estimasinya dan belum diberi tahu " +
			"nilai akseptasinya.",
		Columns:            listColumnsFor(),
		Source:             SourceAdvice,
		AdviceKindSent:     "pla",
		ExcludeWhenDLASent: true,
		WorkStatus:         WorkOpen,
		ExcludeGroupPanels: ExcludedGroupPanels,
		HasDetailAction:    true,
	},
	{
		Code: TabPLADLA,
		Name: "PLA & DLA",
		Description: "Klaim yang DLA-nya sudah dikirimkan kepada Anda. Termasuk klaim " +
			"yang sudah selesai tetapi masih menunggu penutupan — status klaim itu " +
			"digambar sebagai 1139.",
		Columns:                 listColumnsFor(),
		Source:                  SourceAdvice,
		AdviceKindSent:          "dla",
		WorkStatus:              WorkOpenOrPendingClose,
		ExcludeGroupPanels:      ExcludedGroupPanels,
		PendingCloseBecomes1139: true,
		HasDetailAction:         true,
	},
	{
		Code: TabClose,
		Name: "CLOSE CLAIM",
		Description: "Klaim yang sudah selesai dan tidak lagi menunggu penutupan. " +
			"Disaring oleh PLA yang terkirim — bukan DLA — dan mencocokkan SELURUH " +
			"kode reasuradur milik login Anda.",
		Columns:            listColumnsFor(),
		Source:             SourceAdvice,
		AdviceKindSent:     "pla",
		WorkStatus:         WorkClosed,
		ExcludeGroupPanels: ExcludedGroupPanels,
		AllReinsurerCodes:  true,
		HasDetailAction:    true,
	},
	{
		Code: TabInbound,
		Name: "NOT ANSWERED",
		Description: "Klaim yang punya pesan DITUJUKAN kepada Anda dan belum ANDA " +
			"jawab. Bukalah rinciannya untuk membaca dan membalas pesannya. Berbeda " +
			"dari ketiga daftar di sebelah kiri, daftar ini TIDAK mengecualikan lini " +
			"Personal Accident maupun Travel.",
		Columns:             listColumnsFor(),
		Source:              SourceCommunication,
		WorkStatus:          WorkOpen,
		CommunicationStatus: CommunicationNotAnswered,
		CommunicationRole:   RoleRecipient,
		HasDetailAction:     true,
	},
	{
		Code: TabOutbound,
		Name: "NOT REPLIED FROM ASM",
		Description: "Klaim yang punya pesan yang ANDA kirim dan belum dijawab pihak " +
			"Asuransi Sinar Mas.",
		Columns:             listColumnsFor(),
		Source:              SourceCommunication,
		WorkStatus:          WorkOpen,
		CommunicationStatus: CommunicationNotAnswered,
		CommunicationRole:   RoleSender,
		HasDetailAction:     true,
	},
	{
		Code: TabAnswered,
		Name: "REPLIED FROM ASM",
		Description: "Klaim yang punya pesan yang Anda kirim dan SUDAH dijawab " +
			"Asuransi Sinar Mas. Jawabannya terbaca pada rincian klaim.",
		Columns:             listColumnsFor(),
		Source:              SourceCommunication,
		WorkStatus:          WorkOpen,
		CommunicationStatus: CommunicationAnswered,
		CommunicationRole:   RoleSender,
		HasDetailAction:     true,
	},
}

// Tabs mengembalikan salinan ketujuh tampilan.
func Tabs() []Tab {
	out := make([]Tab, len(tabs))
	copy(out, tabs)
	return out
}

// FindTab mencari tab menurut kodenya.
//
// Kode dicocokkan tanpa memedulikan huruf besar-kecil dan spasi di ujung: ia datang dari
// alamat.
func FindTab(code string) (Tab, bool) {
	clean := strings.ToLower(strings.TrimSpace(code))
	for _, tab := range tabs {
		if tab.Code == clean {
			return tab, true
		}
	}
	return Tab{}, false
}

// ExcludesGroupPanel menyatakan sebuah lini bisnis dikecualikan tab ini.
func (t Tab) ExcludesGroupPanel(groupPanel string) bool {
	for _, excluded := range t.ExcludeGroupPanels {
		if excluded == groupPanel {
			return true
		}
	}
	return false
}
