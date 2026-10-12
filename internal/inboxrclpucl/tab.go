package inboxrclpucl

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada WorkItem yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian dapat berjudul berbeda dari tab ke tab —
// `RCL_PUCL_1` berjudul "Status RCL/PUCL" pada dua tab pertama dan "Status" pada tab Klaim
// MSIG, persis seperti di ketiga section-nya.
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
	FieldCaseID          = "no_case"
	FieldPolicyNumber    = "no_polis"
	FieldInsuredName     = "nama_tertanggung"
	FieldInboxEntryAt    = "tanggal_masuk_inbox"
	FieldAnalystNote     = "deskripsi_analyst"
	FieldTrack           = "status_rcl_pucl"
	FieldLetterPrintedAt = "tanggal_cetak_surat"
	FieldClaimAge        = "lama_klaim"
	FieldExpiryStatus    = "status_kadaluarsa"
	FieldCreatedAt       = "tanggal_dibuat"
)

// Nama field JSON pada satu baris LAPORAN HARIAN.
//
// Terpisah dari yang di atas karena laporan memang bukan grid: dua isiannya tidak ada di
// grid mana pun (`STATUSCLAIM_1`), dan satu isian grid tidak ada di laporan
// (`LAMAKLAIM_1`). Menyatukan keduanya akan membuat berkas ekspor tampak seperti salinan
// layar, padahal isinya berbeda — lihat DailyReportRow.
const (
	FieldReportSentAt      = "tanggal_kirim_rcl_pucl"
	FieldReportClaimStatus = "status_klaim"
)

// Kode tab.
//
// # Kenapa angka, dan kenapa bukan nilai sistem lama
//
// Karena sistem lama tidak punya kode tab untuk layar ini. Ketiga tabnya dipisahkan oleh
// SUB_SECTION yang berbeda di dalam `Section/InputPUCL-RCL_Section-Section.xml`, dan yang
// membedakannya hanyalah nama section — bukan kontrak yang layak dibawa ke API.
//
// Kodenya karena itu ditetapkan di sini, berurutan seperti tampilnya, dan diperlakukan
// sebagai kontrak modul ini sendiri. Penelusuran balik ke export ditempuh lewat nama
// section dan Report Definition-nya — disebut lengkap pada setiap tab di bawah — bukan lewat
// kodenya.
const (
	TabCetakSurat         = "1"
	TabKelengkapanDokumen = "2"
	TabKlaimMSIG          = "3"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// "Cetak Surat", karena itulah SUB_SECTION pertama pada kontainer tabnya — ia berada di
// posisi paling atas `InputPUCL-RCL_Section`, dan itulah yang lebih dulu terbaca pengguna.
// Ia pula tab yang menuntut tindakan paling awal dalam perjalanan surat PUCL.
const DefaultTab = TabCetakSurat

// Tab adalah satu partisi antrean RCL/PUCL.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul tab yang dibaca pengguna, mengikuti `D-13`.
	Name string

	// Description menjelaskan isi partisinya dalam satu kalimat.
	//
	// Sistem lama tidak punya keterangan seperti ini, dan di layar ini ketiadaannya nyata
	// akibatnya: ketiga tab punya kolom yang IDENTIK, sehingga tanpa keterangan tidak ada
	// apa pun yang menjelaskan mengapa satu klaim ada di tab yang satu dan tidak di yang
	// lain.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	Columns []Column

	// LetterPrinted menyatakan penyaring `TANGGALCETAKDOKUMENPUCL_1` tab ini.
	//
	//	false  IS NULL     — tab Cetak Surat
	//	true   IS NOT NULL — tab Kelengkapan Dokumen dan Klaim MSIG
	//
	// Nilainya BUKAN sekadar keterangan: ia yang memilih kueri di repo/sqlstore dan yang
	// menyaring di repo/memory, sehingga keduanya tidak dapat berselisih.
	LetterPrinted bool

	// MSIG menyatakan penyaring `MSIG_1` tab ini.
	//
	//	false  IS NULL   — tab Cetak Surat (tidak menyaringnya sama sekali) dan Kelengkapan
	//	true   = 'MSIG'  — tab Klaim MSIG
	//
	// Perhatikan tab Cetak Surat TIDAK menyaring `MSIG_1` sama sekali — Report
	// Definition-nya memang tidak punya penyaring itu. Pembedaan itu ditangani kueri
	// masing-masing, bukan oleh isian ini sendiri.
	MSIG bool

	// HasDateRangeReport menyatakan tab ini punya LAPORAN HARIAN berbasis rentang tanggal
	// di balik tombol ekspornya, bukan ekspor grid biasa.
	//
	// Hanya tab Cetak Surat begitu. Perbedaannya bukan kehalusan: berkas yang dihasilkan
	// punya kolom yang berbeda DAN isi yang berbeda dari grid yang sedang dilihat — lihat
	// DailyReportRow.
	HasDateRangeReport bool

	// Blocked menyatakan tab ini digambar tetapi belum dapat diisi.
	//
	// Tidak ada tab yang terhalang di layar ini. Tab "Klaim MSIG" SANGAT MUNGKIN kosong di
	// produksi, tetapi itu bukan hal yang sama: ia tidak terhalang artefak mana pun,
	// kuerinya dapat dijalankan, dan kosongnya adalah JAWABAN — bukan ketidakmampuan
	// menjawab. Menandainya terhalang akan menyembunyikan baris yang mungkin memang ada.
	Blocked bool

	// BlockedReason menjelaskan penghalangnya dalam kalimat yang dapat langsung
	// ditampilkan ke pengguna. Kosong bila tab tidak terhalang.
	BlockedReason string

	// BlockedOwner menyebut siapa yang dapat menghilangkan penghalangnya.
	//
	// Ia disebut supaya penghalang punya alamat. Penghalang tanpa pemilik tidak pernah
	// hilang — itu pelajaran yang sudah tercatat di `D-36`.
	BlockedOwner string

	// Notice adalah keterangan yang berlaku pada TAB INI SAJA, ditampilkan di atas grid.
	//
	// Ia berbeda dari PlannedDifferences, yang berlaku untuk seluruh layar. Yang ditaruh di
	// sini adalah hal yang hanya benar pada satu tab — dan di layar ini ada dua: kolom
	// "Tanggal Cetak Surat" yang SELALU kosong di tab Cetak Surat, dan tab Klaim MSIG yang
	// kemungkinan selalu kosong.
	//
	// Kosong berarti tidak ada keterangan khusus.
	Notice string
}

// Kolom grid — IDENTIK di ketiga tab kecuali judul kolom keenam.
//
// # Urutan dan judulnya dari mana
//
// Keduanya dibaca dari SEL GRID ketiga section, bukan dari `pyListFields` Report
// Definition-nya. Itu penting dan bukan kerapian: Report Definition ketiga tab mengambil
// 17–25 isian, sementara yang benar-benar DIGAMBAR hanya sembilan. `D-13` menetapkan
// tampilan mengikuti layar lama, sehingga yang dibawa adalah yang digambar.
//
// Pasangan judul dan properti, diverifikasi satu per satu dari
// `Section/InboxCetakSuratPUCLRCL_Section-Section.xml`:
//
//	Nomor Case         .pyID
//	No Polis           .Policy.PolicyNo
//	Nama Tertanggung   .Policy.QQName
//	Tanggal Masuk Inbox .ClaimData.PUCLStatus.TanggalKirimPUCL
//	Deskripsi Analyst  .ClaimData.PUCLStatus.KomentarAnalisator
//	Status RCL/PUCL    .ClaimData.PUCLStatus.RCL_PUCL
//	Tanggal Cetak Surat .ClaimData.PUCLStatus.TanggalCetakDokumenPUCL
//	Lama Klaim         .ClaimData.PUCLStatus.LamaKlaim
//	Status Kadaluarsa  .ClaimData.PUCLStatus.StatusKlaim
//
// # Isian Report Definition yang TIDAK digambar, dan itu disengaja
//
// Ketiga Report Definition mengambil antara lain `.ClaimData.UserTeknis` ("Nama PIC
// Teknik"), `.ClaimData.PUCLStatus.KomentarPUCL`, `.ClaimData.StatusClaim`, dan
// `.ClaimData.PUCLStatus.StatusCase`. Tidak satu pun punya sel di section mana pun, dan
// tidak satu pun dibawa — menggambarnya berarti menambah kolom yang tidak pernah ada.
//
// `STATUSCASE_1` patut diperhatikan khusus: ia MENYARING tab Cetak Surat tetapi tidak
// digambar di mana pun, sementara judul "Status Kadaluarsa" justru menunjuk `STATUSKLAIM_1`.
// Lihat WorkItem.ExpiryStatus.
func gridColumns(trackTitle string) []Column {
	return []Column{
		{Key: FieldCaseID, Title: "Nomor Case"},
		{Key: FieldPolicyNumber, Title: "No Polis"},
		{Key: FieldInsuredName, Title: "Nama Tertanggung"},
		{Key: FieldInboxEntryAt, Title: "Tanggal Masuk Inbox"},
		{Key: FieldAnalystNote, Title: "Deskripsi Analyst"},
		{Key: FieldTrack, Title: trackTitle},
		{Key: FieldLetterPrintedAt, Title: "Tanggal Cetak Surat"},
		{Key: FieldClaimAge, Title: "Lama Klaim"},
		{Key: FieldExpiryStatus, Title: "Status Kadaluarsa"},

		// Kolom kesepuluh TIDAK ada di layar lama — ia kolom yang MENGURUTKAN tabel ini.
		//
		// Ditaruh paling kanan dengan sengaja: kesembilan kolom sebelumnya berada persis
		// pada urutan layar lama, sehingga petugas yang membandingkan kedua layar
		// berdampingan membaca kolom yang sama di tempat yang sama. Menyisipkannya di
		// tengah akan menggeser seluruhnya demi satu kolom tambahan.
		//
		// Lihat WorkItem.CreatedAt untuk alasan ia ditambahkan (Work Owner, 2026-09-30).
		{Key: FieldCreatedAt, Title: "Tanggal Dibuat"},
	}
}

// Judul kolom keenam — satu-satunya yang berbeda antartab.
//
// Tab Klaim MSIG menuliskannya "Status" saja, tanpa "RCL/PUCL". Ia dibawa apa adanya
// (`D-13`) meski kolom sumbernya sama persis: pengguna yang membandingkan kedua layar
// berdampingan membaca teks yang sama.
const (
	trackTitleDefault = "Status RCL/PUCL"
	trackTitleMSIG    = "Status"
)

// DailyReportColumns adalah kolom berkas LAPORAN HARIAN RCL/PUCL.
//
// Urutannya mengikuti `RDB List/GetDataPUCLRCLForDailyReport-SQL.xml` apa adanya, kecuali
// `PZINSKEY` yang tidak digambar — ia kunci teknis, bukan isian yang dibaca orang.
//
// Judulnya TIDAK diambil dari alias kueri lama, dan alasannya terbaca sendiri begitu
// aliasnya dibaca: `POLICYNO` dialiaskan "ProdKe", `QQNAME` dialiaskan "NamaSurveyor",
// `TANGGALKIRIMPUCL_1` dialiaskan "City", `KOMENTARANALISATOR_1` dialiaskan
// "CloseClaimNote", dan `TANGGALCETAKDOKUMENPUCL_1` dialiaskan "CityID". Tidak satu pun
// menyatakan isinya, dan `D-19` melarang membawanya.
//
// Yang dipakai adalah judul kolom grid untuk isian yang sama, supaya berkas dan layar
// menyebut hal yang sama dengan kata yang sama.
var DailyReportColumns = []Column{
	{Key: FieldCaseID, Title: "Nomor Case"},
	{Key: FieldPolicyNumber, Title: "No Polis"},
	{Key: FieldInsuredName, Title: "Nama Tertanggung"},
	{Key: FieldReportSentAt, Title: "Tanggal Kirim RCL/PUCL"},
	{Key: FieldAnalystNote, Title: "Deskripsi Analyst"},
	{Key: FieldLetterPrintedAt, Title: "Tanggal Cetak Surat"},
	{Key: FieldTrack, Title: "Status RCL/PUCL"},
	{Key: FieldReportClaimStatus, Title: "Status Klaim"},
}

// tabs adalah ketiga tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutannya mengikuti urutan SUB_SECTION pada `Section/InputPUCL-RCL_Section-Section.xml`:
// Cetak Surat (offset ~42.000), Kelengkapan Dokumen (~76.000), Klaim MSIG (~116.000).
var tabs = []Tab{
	{
		Code: TabCetakSurat,

		// Judul ini ADA di layar lama apa adanya: `pyCaption Cetak Surat` pada
		// `Harness/RCLPUCL_Harness-Harness.xml` dan pada kontainer tabnya.
		Name: "Cetak Surat",

		Description: "Klaim RCL/PUCL yang suratnya BELUM dicetak. Inilah antrean yang " +
			"menunggu tindakan paling awal: begitu suratnya dicetak, klaimnya berpindah " +
			"sendiri ke tab Kelengkapan Dokumen.",

		Columns:            gridColumns(trackTitleDefault),
		LetterPrinted:      false,
		HasDateRangeReport: true,

		Notice: "Kolom \"Tanggal Cetak Surat\" selalu kosong di tab ini, dan itu bukan " +
			"data yang hilang — justru itulah arti tab ini: suratnya belum dicetak. " +
			"Kolomnya tetap digambar karena layar lama menggambarnya.",
	},
	{
		Code: TabKelengkapanDokumen,

		// `pyCaption Kelengkapan Dokumen` pada harness dan kontainer tabnya.
		Name: "Kelengkapan Dokumen",

		Description: "Klaim RCL/PUCL yang suratnya SUDAH dicetak tetapi belum disetujui, " +
			"dan bukan jalur MSIG. Antrean ini menunggu kelengkapan dokumen dari " +
			"tertanggung.",

		Columns:       gridColumns(trackTitleDefault),
		LetterPrinted: true,
		MSIG:          false,
	},
	{
		Code: TabKlaimMSIG,

		// `pyCaption Klaim MSIG` pada harness dan kontainer tabnya.
		Name: "Klaim MSIG",

		Description: "Klaim RCL/PUCL jalur MSIG yang suratnya sudah dicetak tetapi belum " +
			"disetujui. Penyaringnya sama dengan tab Kelengkapan Dokumen, kecuali " +
			"penanda jalurnya.",

		Columns:       gridColumns(trackTitleMSIG),
		LetterPrinted: true,
		MSIG:          true,

		// Keterangan ini ADA di layar, bukan hanya di kode, karena tanpanya tab yang
		// kosong akan dilaporkan berulang kali sebagai kerusakan.
		Notice: "Tab ini nyaris selalu kosong, dan itu bukan kerusakan. Penanda jalur " +
			"MSIG hanya terisi pada segelintir klaim — dihitung langsung ke basis data " +
			"pada 2026-09-30: satu baris dari lebih dari tujuh ribu. Seluruh klaim " +
			"bersurat lainnya berada di tab Kelengkapan Dokumen. Penyaringnya sama " +
			"persis dengan penyaring layar lama, sehingga isi tab ini sama dengan isinya " +
			"di Pega.",
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

// Difference adalah satu selisih terhadap sistem lama yang DIPUTUSKAN, bukan cacat.
//
// # Kenapa ia dua bagian, bukan satu kalimat panjang
//
// Karena ia punya DUA pembaca, dan keduanya membutuhkan hal yang berbeda.
//
//	Summary  petugas klaim  — apa akibatnya bagi pekerjaan saya hari ini
//	Detail   penguji & DBA  — kenapa begitu, dan ke butir `P-5` mana ia dipetakan
//
// Sebelum 2026-09-30 keduanya ditulis menjadi satu, dan hasilnya panel sepanjang **1.114
// kata dalam 15 butir** — lebih panjang daripada tabel yang dijelaskannya. Panel yang
// terlalu panjang untuk dibaca TIDAK mencegah laporan kerusakan palsu yang menjadi alasan
// keberadaannya; ia hanya memindahkan kegagalannya dari "tidak ada penjelasan" menjadi
// "penjelasannya tidak dibaca".
//
// Yang dipisah hanya penyajiannya. Tidak satu kata pun dibuang: seluruh teks lama ada di
// Detail, dan `D-54` tetap terlayani.
type Difference struct {
	// Summary adalah satu kalimat dalam bahasa petugas klaim.
	//
	// Ia menyebut AKIBATNYA, bukan sebabnya, dan tidak memuat nama artefak Pega, nomor
	// keputusan, tanggal, maupun nama tabel. Pembacanya tidak mengenal satu pun dari itu —
	// dan istilah yang tidak dikenali membuat kalimatnya dilewati, bukan dipelajari.
	Summary string

	// Detail adalah alasan lengkapnya, dibuka hanya bila diminta.
	//
	// Di sinilah nama artefak, nomor keputusan, dan tanggalnya tinggal. Ia yang dibaca saat
	// uji kesetaraan gerbang 1, ketika pertanyaannya bukan lagi "apa akibatnya bagi saya"
	// melainkan "selisih ini dipetakan ke butir `P-5` yang mana" (`D-54`).
	Detail string
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
//
// # URUTANNYA BUKAN URUTAN PENULISAN
//
// Ia diurutkan menurut seberapa sering petugas MENABRAKNYA, bukan menurut kapan butirnya
// ditulis. Sembilan butir pertama menyangkut hal yang ditemui saat memakai layar; sisanya
// menjelaskan hal yang hanya terlihat bila kedua layar dibandingkan berdampingan.
//
// Sebelum diurutkan begini, dua hal yang paling sering ditanyakan — isian tanggal yang tidak
// menyaring tabel, dan berkas unduhan yang bukan salinan tabel — berada di urutan kedua dan
// ketiga di bawah butir terpanjang di seluruh daftar.
var PlannedDifferences = []Difference{
	// ---- Yang ditemui saat MEMAKAI layar -------------------------------------------

	{
		Summary: "\"Download Dokumen\" memindahkan klaim ke tab \"Kelengkapan Dokumen\", " +
			"tetapi TIDAK menerbitkan berkas PDF suratnya.",
		Detail: "Rangkaian aslinya tiga langkah: `InsertMitraPA(tipe=\"cetak\")`, " +
			"`PUCLPost(Status=\"\", statusCase=\"1\")`, lalu `InsertHistoryClaimPNC`.\n\n" +
			"YANG DIKERJAKAN DI SINI:\n" +
			"  · TGL_CETAK_DOKUMEN_PUCL = kini (langkah 4 dan 51) — INILAH yang " +
			"memindahkan klaimnya, karena penyaring tab \"Cetak Surat\" adalah kolom ini " +
			"masih kosong\n" +
			"  · STATUS_CASE = 1               (langkah 17)\n" +
			"  · STATUS_CLAIM = 1157           (langkah 17) — \"Document Waiting RCL/PUCL\"\n\n" +
			"YANG TIDAK DIKERJAKAN:\n" +
			"  · `AttachAsPDFC` — membuat PDF surat dan melampirkannya. DUA penghalang, " +
			"bukan satu: templat suratnya (`SuratPUCL`) TIDAK ADA di export — folder " +
			"`HTML/` memuat 30 berkas dan tidak satu pun surat RCL/PUCL — dan lampiran " +
			"Pega tidak disimpan di basis data melainkan di layanan penyimpanan luar.\n" +
			"  · `InsertHistoryClaimPNC` — riwayat \"Wait for Complete PUCL Document\".\n" +
			"  · `IsCetakSurat = 1` dari `InsertMitraPA`.\n\n" +
			"Perpindahan tabnya tetap dikerjakan, karena itulah yang menghambat petugas: " +
			"tanpa itu klaimnya tertahan di tab pertama selamanya. Suratnya dicetak di Pega.",
	},

	{
		Summary: "Kedua tombol \"Kirim\" MEMINDAHKAN klaim ke tahap Send To Analis, " +
			"beserta isian, surat, dan riwayatnya — tetapi baris antrean lama di Pega " +
			"tidak ikut ditutup.",
		Detail: "Rangkaian aslinya tiga langkah: `InsertMitraPA(tipe=\"dokumen\")`, " +
			"`PUCLPost(Status=\"1\")`, lalu penyerahan flow action `SendtoRCLPUCL` " +
			"(di layar: Finish Assignment). Nomor langkah di bawah mengikuti " +
			"`REPEATINGINDEX` pada `PUCLPost`, yaitu nomor yang terbaca di designer Pega.\n\n" +
			"YANG DIKERJAKAN DI SINI:\n" +
			"  · KOMENTAR_PUCL dan TGL_TERIMA_DOKUMEN_PUCL disimpan lebih dulu — " +
			"`PUCLPost` langkah 10 menulis `KomentarPUCL` bersama tindakannya, dan " +
			"keduanya `pyRequired` sehingga kosong DITOLAK, seperti Finish Assignment.\n" +
			"  · PUCL_APPROVE = 1             (langkah 11)\n" +
			"  · STATUS_CLAIM = 1151          (langkah 11) — \"Analyst\" menurut master\n" +
			"  · TGL_CETAK_DOKUMEN_PUCL = kini (langkah 4 dan 30)\n" +
			"  · PDF surat diterbitkan dan dilampirkan — `AttachAsPDFC` langkah 27 " +
			"berprekondisi `1==1`, jadi ia berjalan untuk SETIAP tombol, bukan hanya " +
			"\"Download Dokumen\". Namanya mengikuti jalur klaim (langkah 20-22): " +
			"PUCL.pdf · RCL.pdf · Notification.pdf.\n" +
			"  · Satu baris riwayat \"Send by PUCL to Analyst\" pada " +
			"`LIST_HISTORY_CLAIM_PNC` — `InsertHistoryClaimPNC` langkah 35.\n" +
			"  · PERPINDAHAN TAHAP: tugas lama ditutup dan tugas baru dibuka pada tahap " +
			"`kirim-analis` di `CPNC_TUGAS`, bertuan PIC Teknik klaim. Ini meniru " +
			"`SetTicket(SendtoAnalysator)` langkah 17 ditambah Finish Assignment — dan " +
			"inilah yang membuat klaim menjadi pekerjaan orang lain, bukan sekadar " +
			"berubah penanda.\n\n" +
			"YANG TIDAK DIKERJAKAN, masing-masing dengan alasannya:\n" +
			"  · `SendtoAnalystDate` (langkah 30) — tidak punya kolom di tabel datar.\n" +
			"  · `PUCLApprove = \"Setuju\"` pada baris AdjustmentList klaim (langkah 10).\n" +
			"  · `IsDokLengkap = 1` — ia properti clipboard objek kerja Pega, tersimpan " +
			"di `PZPVSTREAM` dan TIDAK punya kolom tabel di mana pun.\n" +
			"  · `PNCInsertMitraLog_Act` — catatan TAT ke `PNC_CHRONOLOGYTAT` lewat " +
			"procedure `INSERT_PNCCHRONOLOGYTAT` (163 baris, memakai jam kerja lewat DB " +
			"Link). Menulis ulangnya adalah pekerjaan modul TAT, bukan modul ini. Ia pun " +
			"hanya berjalan bila `IsDokLengkap` SUDAH bernilai 1 sebelumnya.\n" +
			"  · Baris antrean `RCLPUCL` milik PEGA tidak dihapus. Menulis tabel " +
			"penugasan Pega sudah terbukti merusak satu klaim, sehingga baris itu " +
			"dibiarkan. Akibatnya klaim yang sama masih tergambar di antrean RCL/PUCL " +
			"Pega — tidak menghalangi apa pun, dan tidak perlu dikerjakan ulang di sana.\n\n" +
			"Satu lampiran PUCL.pdf TAMBAHAN muncul bila petugas sudah menekan " +
			"\"Download Dokumen\" lebih dulu. Pega membuang yang senama (langkah 25-26); " +
			"itu TIDAK ditiru karena `D-66` melarang penghapusan fisik. Daftarnya terurut " +
			"terbaru di atas, jadi yang berlaku tetap yang teratas.\n\n" +
			"Akibat yang harus disadari selama masa paralel: klaimnya KELUAR dari antrean " +
			"RCL/PUCL layar ini, tetapi MASIH TERLIHAT di antrean RCL/PUCL milik Pega — " +
			"baris `PC_ASSIGN_WORKBASKET` sengaja tidak dihapus, karena penghapusan tabel " +
			"Pega tidak dapat dipulihkan dan tidak dibutuhkan agar layar ini benar.\n\n" +
			"Email ke cabang TIDAK terkirim — tetapi itu BUKAN selisih untuk klaim PA: " +
			"keempat langkah emailnya (31-34) berprekondisi `IsTravel`.",
	},

	{
		Summary: "Kedua isian tanggal di tab \"Cetak Surat\" tidak menyaring tabel di " +
			"bawahnya. Keduanya hanya dipakai tombol unduh.",

		Detail: "Itu perilaku layar lama apa adanya: grid-nya dipasok Report Definition " +
			"yang tidak menyaring tanggal sama sekali, sementara tombol ekspornya " +
			"menjalankan kueri yang BERBEDA. Keputusan Work Owner 2026-09-23 untuk " +
			"mereplikasinya (`P-5`).",
	},

	{
		Summary: "Berkas unduhan tab \"Cetak Surat\" BUKAN salinan tabel yang sedang " +
			"dilihat. Baris dan kolomnya berbeda.",

		Detail: "Isinya laporan harian, dan ia berbeda dalam empat hal: disaring rentang " +
			"tanggal pengiriman RCL/PUCL, memuat klaim yang suratnya SUDAH dicetak, memuat " +
			"klaim yang sudah selesai, dan menambahkan seluruh klaim Personal Accident pada " +
			"rentang yang sama tanpa melihat antrean bersamanya. Kolomnya pun berbeda: ada " +
			"\"Status Klaim\" yang tidak ada di tabel, dan tidak ada \"Lama Klaim\" yang ada " +
			"di tabel. Ini perilaku layar lama apa adanya.",
	},

	{
		Summary: "Tab \"Klaim MSIG\" nyaris selalu kosong, dan itu normal — bukan " +
			"kerusakan.",

		Detail: "Tab ini berisi klaim yang datanya DARI atau UNTUK perusahaan MSIG " +
			"(dijelaskan Work Owner 2026-09-30). Penandanya hanya terisi pada segelintir " +
			"klaim: dihitung langsung ke basis data pada 2026-09-30, satu baris dari lebih " +
			"dari tujuh ribu. Seluruh klaim bersurat lainnya berada di tab Kelengkapan " +
			"Dokumen. Nilai pembandingnya disalin dari penyaring Pega, bukan ditebak — " +
			"sehingga isi tab ini sama persis dengan isinya di Pega.",
	},

	{
		Summary: "Kolom \"Lama Klaim\" berisi TANGGAL, bukan lamanya klaim.",

		Detail: "Isinya tanggal kirim untuk proses PUCL (dijelaskan Work Owner 2026-09-30). " +
			"Judulnya menyesatkan sejak di Pega, dan judul itu tetap dibawa apa adanya " +
			"supaya yang terbaca pengguna tidak berubah (`D-13`).",
	},

	{
		Summary: "Kolom \"Lama Klaim\" dan \"Tanggal Masuk Inbox\" sering terbaca sama " +
			"persis. Keduanya memang tetap digambar.",

		Detail: "Pemeriksaan langsung ke basis data pada 2026-09-30 menunjukkan keduanya " +
			"ditulis pada langkah yang sama dan hanya terpaut milidetik, sehingga setelah " +
			"digambar sampai satuan detik keduanya kerap identik — meski tidak selalu. " +
			"Duplikasinya ada di Pega, dan keputusan Work Owner 2026-09-30 adalah mengikuti " +
			"Pega apa adanya.",
	},

	{
		Summary: "Seluruh tombol layar kerja kini berjalan di sini; tidak ada lagi yang " +
			"menunggu layanan Pega.",

		Detail: "\"Tolak Klaim\" adalah yang terakhir berpindah, pada 2026-10-06. Ia " +
			"menjalankan activity yang SAMA dengan kedua tombol Kirim — PUCLPost — hanya " +
			"dengan Status \"0\" alih-alih \"1\", dan perbedaan satu karakter itu membalik " +
			"tujuannya: Status \"1\" melepas ticket SendtoAnalysator sehingga klaim " +
			"diteruskan ke Analyst, sedangkan Status \"0\" memanggil ASMForceCaseClose " +
			"sehingga kasusnya DITUTUP dengan status kerja Resolved-Rejected. Ketiga " +
			"tulisannya seluruhnya pada tabel milik aplikasi ini, sehingga tidak ada yang " +
			"dibutuhkan dari Pega. Yang TIDAK terjadi: penugasan di Pega tidak disentuh " +
			"(P-1), sehingga klaim yang lahir di Pega masih terlihat di antrean Pega sampai " +
			"Pega sendiri menutupnya — sama seperti kedua tombol Kirim.",
	},

	{
		Summary: "Kedua tombol Kirim TIDAK memindahkan penugasan di Pega; klaim tetap " +
			"terlihat di antrean RCL/PUCL milik Pega.",

		Detail: "Pemindahan penugasan sempat dipasang 2026-10-02 dan DICABUT hari itu juga: " +
			"ia membuat satu klaim tidak dapat dibuka lagi di Pega. Sebabnya baris " +
			"penugasan yang kami tulis tidak memuat PZPVSTREAM, dan dari 105.616 baris " +
			"penugasan Pega tidak satu pun berbentuk begitu — kolomnya nullable di katalog, " +
			"tetapi Pega tidak menerimanya. Klaim yang terdampak sudah dipulihkan. " +
			"Yang dikerjakan tombol ini sekarang: menandai klaim selesai di TC_PNC_PUCL " +
			"saja, sehingga ia keluar dari antrean layar ini tetapi masih terlihat di " +
			"antrean RCL/PUCL milik Pega — bergabung dengan 16 klaim yang sudah " +
			"berkeadaan sama sebelum modul ini ada. Perpindahan yang sebenarnya menunggu " +
			"Service REST ActionClaimPUCL dari Tim Pega.",
	},

	{
		Summary: "Surat RCL/PUCL terbit dengan tiga baris KOSONG, dan surat lama tidak " +
			"dibuang saat dicetak ulang.",

		Detail: "Susunan dan seluruh kalimat tetapnya disalin dari templat Pega " +
			"HTML/SuratPUCL yang dikirim Work Owner 2026-10-02. Tiga baris tergambar kosong " +
			"karena datanya tidak ada di basis data mana pun — Nomor Kontrak, Unit Bisnis / " +
			"Seksi, dan nama kedua penanda tangan; ketiganya hidup di clipboard Pega. " +
			"Barisnya tetap digambar supaya kekosongannya terlihat. Kop logo juga belum " +
			"tergambar: berkas gambarnya tidak ada di export. Selisih kedua: Pega membuang " +
			"lampiran bernama sama sebelum melampirkan yang baru, sehingga satu klaim hanya " +
			"punya satu PUCL.pdf. Itu tidak ditiru karena D-66 melarang penghapusan fisik " +
			"dan tabel lampirannya tidak punya kolom penanda hapus; mencetak ulang karena " +
			"itu menambah baris, dan yang berlaku adalah yang teratas.",
	},

	{
		Summary: "Dialog unggah menyimpan kategori lampiran yang dipilih; nama kategorinya " +
			"ditampilkan apa adanya, bukan diberi label seperti di Pega.",

		Detail: "Kolom \"Category\" berisi KATEGORI LAMPIRAN — AcceptanceNote, " +
			"ClaimFaceSheet, LOD, dan seterusnya — bukan jenis dokumen. Yang " +
			"mendefinisikannya rule Rule-Obj-AttachmentCategory, dan tipe rule itu tidak ada " +
			"di export sama sekali (R-16), sehingga daftarnya diturunkan dari kategori yang " +
			"benar-benar dipakai lampiran klaim PNC: 30 baris. Dua akibatnya nyata. " +
			"Pertama, kategori yang sudah didefinisikan tetapi belum pernah dipakai TIDAK " +
			"muncul. Kedua, yang digambar adalah NAMA kategorinya (AcceptanceNote), " +
			"sedangkan Pega menggambar labelnya (\"Acceptance Note\") — teks label itu " +
			"hidup di rule yang tidak diekspor, dan mengarangnya dengan memecah huruf besar " +
			"akan menghasilkan teks yang tidak pernah ada di layar mana pun. Bawaannya " +
			"\"File\", mengikuti Pega. Lampiran yang telanjur tersimpan tanpa kategori " +
			"dibiarkan apa adanya.",
	},

	{
		Summary: "Tombol yang muncul BERBEDA-BEDA menurut jalur klaim dan lini bisnisnya, " +
			"persis seperti di Pega.",

		Detail: "Tab Lampiran Surat menampilkan \"Download Dokumen\" untuk klaim non-MSIG, " +
			"dan \"Tutup Klaim\" untuk klaim MSIG berstatus Notification — tidak pernah " +
			"keduanya, dan untuk klaim MSIG di luar Notification tidak satu pun. Tab " +
			"Penerimaan Dokumen selalu menampilkan Unggah Dokumen, Lihat Dokumen, dan Save; " +
			"\"Tolak Klaim\" hanya pada jalur RCL; \"Kirim Ke Analyst\" hanya pada jalur PUCL " +
			"lini PA, dan \"Kirim ke PIC Teknik\" hanya pada jalur PUCL lini Travel. Jadi " +
			"tombol yang tidak Anda lihat belum tentu hilang — ia mungkin memang bukan " +
			"tindakan untuk klaim itu.",
	},

	{
		Summary: "Tombol \"Download Dokumen\" melakukan DUA hal: menerbitkan PDF suratnya, " +
			"DAN menandainya sudah dicetak sehingga klaimnya berpindah tab.",

		Detail: "Di Pega tombol itu menjalankan aksi bertipe \"cetak\". Surat memang terbit — " +
			"PDF-nya dibentuk, dilampirkan ke klaim, lalu dibuka. Tetapi pada langkah yang " +
			"sama tanggal cetak dokumen ikut terisi, dan kolom itulah yang menentukan klaim " +
			"berada di tab \"Cetak Surat\" atau sudah pindah ke \"Kelengkapan Dokumen\". Jadi " +
			"ia bukan tombol lihat-lihat: sekali ditekan, klaimnya berpindah. Yang TIDAK " +
			"terjadi padanya adalah penutupan klaim — " +
			"langkah itu dilewati untuk jalur cetak, dan hanya berlaku pada \"Tolak Klaim\" " +
			"di jalur RCL.",
	},

	{
		Summary: "Tombol \"Reminder PUCL\" tidak ada di layar kerja, dan di Pega pun " +
			"sebenarnya tidak pernah muncul.",

		Detail: "Syarat tampilnya di layar lama ditulis \"1==2\" — syarat yang tidak pernah " +
			"benar. Ia tombol yang dimatikan dengan cara dikarang syaratnya alih-alih " +
			"dihapus. Yang ditiru adalah perilakunya yang nyata, yaitu tidak muncul; " +
			"menggambarnya di sini akan menawarkan tindakan yang tidak pernah tersedia. Hal " +
			"yang sama berlaku pada satu tombol \"Tolak Klaim\" kedua di layar lama.",
	},

	{
		Summary: "Lima isian di layar kerja tidak dapat diisi dari sini, dan itu bukan " +
			"data yang kosong.",

		Detail: "Kelimanya adalah properti clipboard pada objek kerja Pega — No Kontrak, " +
			"Business Unit / Seksi, Email Tertanggung, Tanggal Kelengkapan Dokumen, dan " +
			"daftar Tanggal terima Dokumen. Ia tidak diekspos sebagai kolom tabel, sehingga " +
			"tidak dapat dibaca dengan kueri biasa selama objek kerjanya masih dimiliki Pega. " +
			"Isian itu tetap digambar di tempatnya, bertanda, supaya ketiadaannya terlihat " +
			"alih-alih tersamar sebagai isian yang memang belum diisi. Daftarnya semula " +
			"sembilan; empat di antaranya — Perihal dan ketiga Keterangan — ternyata SUDAH " +
			"ada sebagai kolom dan kini terbaca.",
	},

	{
		Summary: "Klaim yang suratnya sudah dicetak tetapi penanda PUCL-nya kosong tidak " +
			"muncul di tab mana pun — di sini maupun di Pega. Laporkan bila Anda merasa ada " +
			"klaim yang hilang.",

		Detail: "Kedua tab bersurat menampilkan klaim yang MASIH di tangan PUCL. Penandanya " +
			"dijelaskan Work Owner 2026-09-30: nilai 0 berarti klaimnya dikirim ke PUCL, " +
			"nilai 1 berarti PUCL sudah mengembalikannya ke Analyst — dan yang bernilai 1 " +
			"keluar dari kedua tab. Klaim yang penandanya BELUM PERNAH DIISI juga tidak " +
			"muncul, karena penyaring layar lama memakai tanda \"tidak sama dengan\" dan " +
			"perbandingan semacam itu tidak pernah bernilai benar untuk nilai kosong. Untuk " +
			"klaim yang belum pernah dikirim ke PUCL, itu memang benar. Keputusan Work Owner " +
			"2026-09-30: ikuti Pega apa adanya, penyaring tidak diubah. Jumlahnya dapat " +
			"dihitung saat dibutuhkan.",
	},

	{
		Summary: "Klaim berstatus Notification hanya punya tab \"Lampiran Surat\" — tab " +
			"\"Penerimaan Dokumen\" memang tidak ada untuknya, sama seperti di layar lama.",

		Detail: "Ketiga kode jalurnya: 1 RCL, 2 PUCL, 3 Notification (dijelaskan Work " +
			"Owner 2026-09-30). Klaim Notification bukan pekerjaan RCL maupun PUCL " +
			"melainkan pemberitahuan, sehingga tidak ada dokumen yang ditunggu dan tidak " +
			"ada yang dikirim kembali ke Analyst — dan layar lama menyembunyikan tab itu " +
			"justru karena itu. Syaratnya dibaca dari kontainer tab kedua di berkas " +
			"layarnya, dan ditegakkan di sini pula. Klaimnya sendiri TETAP dapat dibuka, " +
			"sama seperti di layar lama; yang disembunyikan hanya satu tab.",
	},

	// ---- Yang hanya terlihat bila kedua layar DIBANDINGKAN --------------------------

	{
		Summary: "Ada satu kolom tambahan di paling kanan, \"Tanggal Dibuat\", yang tidak " +
			"ada di layar lama.",

		Detail: "Urutan baris MENGIKUTI layar lama apa adanya: menurut tanggal pembuatan " +
			"objek kerja, yang terbaru lebih dulu. Di layar lama kolom itu tidak " +
			"ditampilkan, sehingga tabelnya terbaca tidak urut — yang tampil sebagai " +
			"\"Tanggal Masuk Inbox\" adalah tanggal pengiriman RCL/PUCL, dan keduanya dapat " +
			"terpaut berbulan-bulan karena sebuah klaim lahir jauh sebelum ia masuk antrean " +
			"ini. Keputusan Work Owner 2026-09-30: kolom pengurutnya DITAMPILKAN, sebagai " +
			"kolom terakhir. Urutan, isi, dan pembagian halamannya tidak berubah sedikit pun.",
	},

	{
		Summary: "Tanggal ditulis \"tahun-bulan-tanggal jam:menit:detik\" waktu WIB. Yang " +
			"berbeda bentuk penulisannya, bukan isinya.",

		Detail: "Bentuk persis yang dipakai layar lama TIDAK dapat dibaca dari artefaknya: " +
			"ketiga selnya memakai kontrol tanggal bawaan Pega tanpa menyebutkan bentuknya " +
			"sama sekali, sehingga Pega memakai bentuk bawaan tempatnya dijalankan. Yang " +
			"dapat dipastikan hanyalah bahwa ia bentuk TANGGAL — bukan teks teknis ber-huruf " +
			"T dan ber-akhiran zona, yang justru itulah yang dikembalikan basis data apa " +
			"adanya. Yang dipakai adalah bentuk yang sudah dipakai modul ini sejak awal, dan " +
			"judul kolomnya tidak berubah sedikit pun.",
	},

	{
		Summary: "Judul \"Status RCL/PUCL\" dan \"Status Kadaluarsa\" di layar ini menunjuk " +
			"kolom yang BERBEDA dari layar Inbox Manager Receive / PUCL.",

		Detail: "Judulnya sama persis dan sumbernya tabel yang sama, tetapi kolom yang " +
			"digambar berbeda. Itu keadaan di Pega, bukan kekeliruan penyalinan, dan " +
			"masing-masing layar membawa pemetaannya sendiri supaya yang terbaca pengguna " +
			"tidak berubah.",
	},

	{
		Summary: "Di berkas unduhan, kode jalur ditulis \"RCL\" dan \"PUCL\" — bukan angka " +
			"1 dan 2.",

		Detail: "Kueri lama menuliskannya sebagai angka mentah ke dalam berkas, padahal " +
			"kueri lain pada domain yang sama sudah menerjemahkannya. Angka mentah di dalam " +
			"berkas Excel tidak berarti apa pun bagi pembacanya.",
	},

	{
		Summary: "Daftar dipotong per halaman di basis data. Ukuran halamannya tetap 50, " +
			"sama dengan layar lama.",

		Detail: "Grid Pega memotongnya di 500 baris setelah seluruh barisnya ditarik, lalu " +
			"menomori halamannya di memori; di sini halamannya dipotong sebelum baris " +
			"meninggalkan basis data, dan jumlah seluruhnya tetap dihitung tepat.",
	},

	{
		Summary: "Mengklik Nomor Case membuka layar kerja klaim sebagai halaman tersendiri.",

		Detail: "Tab dan nomor halaman antrean ikut ke alamatnya supaya tombol kembali " +
			"mendarat di tempat yang sama. Isinya mengikuti kedua bagian layar lama — " +
			"\"Lampiran Surat\" dan \"Penerimaan Dokumen\" — dan ketiga isian turunannya " +
			"diambil dari anak klaim, sama seperti di Pega. Yang belum ada adalah TINDAKAN " +
			"pada layar itu: di Pega, membuka baris berarti mengambil penugasannya untuk " +
			"dikerjakan, dan itu menulis ke tabel penugasan yang masih dimiliki Pega selama " +
			"kedua sistem berjalan berdampingan.",
	},
}
