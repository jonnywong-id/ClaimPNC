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
		Notice: "Tab ini kemungkinan besar kosong. Kolom penanda jalur MSIG " +
			"(`MSIG_1`) tidak muncul di inventaris kolom terisi yang dibaca langsung dari " +
			"katalog Oracle pada 2026-09-22 — artinya kolomnya ada tetapi tampaknya " +
			"belum pernah diisi. Bila memang begitu, seluruh klaim yang suratnya sudah " +
			"dicetak berada di tab Kelengkapan Dokumen. Menunggu pemastian DBA.",
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
	"Tab \"Klaim MSIG\" kemungkinan besar kosong, dan itu bukan kerusakan. Penyaringnya " +
		"membandingkan kolom penanda jalur MSIG, dan kolom itu TIDAK muncul di inventaris " +
		"kolom terisi yang dibaca langsung dari katalog Oracle pada 2026-09-22 — artinya " +
		"ia ada tetapi tampaknya belum pernah diisi. Keputusan Work Owner 2026-09-23: " +
		"tabnya dibangun apa adanya mengikuti layar lama, dan temuannya dinyatakan " +
		"alih-alih disamarkan. Bila kolomnya memang kosong, seluruh klaim yang suratnya " +
		"sudah dicetak berada di tab Kelengkapan Dokumen. Menunggu pemastian DBA.",

	"Isian tanggal di tab \"Cetak Surat\" TIDAK menyaring tabel di bawahnya. Keduanya " +
		"hanya dipakai tombol ekspor, dan itu perilaku layar lama apa adanya: grid-nya " +
		"dipasok Report Definition yang tidak menyaring tanggal sama sekali, sementara " +
		"tombol ekspornya menjalankan kueri yang BERBEDA. Keputusan Work Owner " +
		"2026-09-23 untuk mereplikasinya (`P-5`).",

	"Berkas ekspor tab \"Cetak Surat\" berisi LAPORAN HARIAN, bukan salinan tabel yang " +
		"sedang dilihat. Isinya berbeda dalam empat hal: ia disaring rentang tanggal " +
		"pengiriman RCL/PUCL, ia memuat klaim yang suratnya SUDAH dicetak, ia memuat " +
		"klaim yang sudah selesai, dan ia menambahkan seluruh klaim Personal Accident " +
		"pada rentang yang sama tanpa melihat antrean bersamanya. Kolomnya pun berbeda: " +
		"ada \"Status Klaim\" yang tidak ada di tabel, dan tidak ada \"Lama Klaim\" yang " +
		"ada di tabel. Ini perilaku layar lama apa adanya.",

	"Kode jalur pada berkas laporan harian diterjemahkan menjadi \"RCL\" dan \"PUCL\". " +
		"Kueri lama menuliskannya sebagai angka mentah 1 dan 2 ke dalam berkas, padahal " +
		"kueri lain pada domain yang sama sudah menerjemahkannya. Angka mentah di dalam " +
		"berkas Excel tidak berarti apa pun bagi pembacanya.",

	"Tindakan \"Cetak Surat\" dan \"Reminder PUCL\" belum tersedia. Keduanya MENULIS — " +
		"yang pertama mengisi tanggal cetak sehingga klaimnya berpindah tab, yang kedua " +
		"mengirim pengingat. Selama Pega dan sistem baru berjalan berdampingan, tabel " +
		"objek kerja dan tabel penugasan hanya boleh ditulis satu sistem, dan keduanya " +
		"masih dimiliki Pega. Tombolnya tetap digambar supaya keberadaannya terlihat, dan " +
		"penekanannya menjawab alasan — bukan halaman kosong.",

	"Urutan baris MENGIKUTI layar lama apa adanya: menurut tanggal pembuatan objek " +
		"kerja, yang terbaru lebih dulu. Yang perlu disadari, kolom itu TIDAK ditampilkan " +
		"di layar ini — yang tampil sebagai \"Tanggal Masuk Inbox\" adalah tanggal " +
		"pengiriman RCL/PUCL, dan keduanya dapat terpaut berbulan-bulan karena sebuah " +
		"klaim lahir jauh sebelum ia masuk antrean ini. Akibatnya tabel dapat terbaca " +
		"tidak urut. Ia tetap tidak diubah: mengganti kunci urut mengubah baris mana yang " +
		"ada di halaman pertama, dan itu selisih yang belum diputuskan siapa pun.",

	"Daftar dipotong per halaman di basis data. Grid Pega memotongnya di 500 baris " +
		"setelah seluruh barisnya ditarik, lalu menomori halamannya di memori; di sini " +
		"halamannya dipotong sebelum baris meninggalkan basis data, dan jumlah seluruhnya " +
		"tetap dihitung tepat. Ukuran halamannya tetap 50, sama dengan layar lama.",

	"Klaim yang penanda persetujuan PUCL-nya belum pernah diisi TIDAK muncul di tab " +
		"\"Kelengkapan Dokumen\" maupun \"Klaim MSIG\". Penyaring layar lama membandingkan " +
		"kolom itu dengan tanda \"tidak sama dengan\", dan perbandingan semacam itu tidak " +
		"pernah bernilai benar untuk nilai yang kosong — di Oracle maupun PostgreSQL. " +
		"Perilakunya dibawa apa adanya karena memperbaikinya akan MENAMBAH baris yang di " +
		"Pega tidak pernah terlihat, dan itu bukan kesetaraan. Bila Anda merasa ada klaim " +
		"yang hilang dari kedua tab itu, laporkan — kemungkinan inilah sebabnya.",

	"Judul kolom \"Status RCL/PUCL\" dan \"Status Kadaluarsa\" di layar ini menunjuk " +
		"kolom yang BERBEDA dari layar Inbox Manager Receive / PUCL, meski judulnya sama " +
		"persis dan sumbernya tabel yang sama. Itu keadaan di Pega, bukan kekeliruan " +
		"penyalinan, dan masing-masing layar membawa pemetaannya sendiri supaya yang " +
		"terbaca pengguna tidak berubah.",
}
