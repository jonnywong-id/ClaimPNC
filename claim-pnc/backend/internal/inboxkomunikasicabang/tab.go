package inboxkomunikasicabang

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada baris yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena kedua tab layar ini menggambar JUMLAH kolom yang
// berbeda dari baris yang bentuknya sama.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`).
	Title string
}

// Nama field JSON pada satu baris percakapan.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldCreatedAt = "tanggal"
	FieldSender    = "pengirim"
	FieldMessage   = "pesan"
	FieldReply     = "jawaban_terakhir"
	FieldReplier   = "penjawab"
	FieldStatus    = "status_register"
	FieldRepliedAt = "tanggal_jawaban"
	FieldRecipient = "tujuan"
)

// Kode tab.
//
// # Kenapa angka, dan kenapa bukan nilai sistem lama
//
// Karena nilai sistem lama TIDAK layak dijadikan kontrak. `PNCGetInboxKomunikasiCabang_Act`
// memilih kuerinya dengan parameter `TYPE`, dan pemetaannya terbalik dari urutan tampil:
//
//	Local.TYPE == "1"  ->  GetInboxKomunikasiCabangAnswered  (tab KEDUA)
//	Local.TYPE == "2"  ->  GetInboxKomunikasiCabang          (tab PERTAMA)
//
// Membawa angka itu apa adanya berarti kontrak API yang `tab=2` berarti tab pertama. Kode
// di bawah karena itu ditetapkan berurutan seperti tampilnya, dan penelusuran balik ke
// export ditempuh lewat nama kuerinya — disebut lengkap pada setiap tab — bukan lewat
// kodenya.
const (
	TabNotAnswered = "1"
	TabAnswered    = "2"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// "Belum Dijawab", karena itulah pekerjaan yang menunggu tindakan. Grid-nya pula yang
// diurutkan dari yang PALING LAMA menunggu — sebuah antrean, dan antrean dibaca dari
// depannya.
const DefaultTab = TabNotAnswered

// Tab adalah satu partisi kotak percakapan.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul tab yang dibaca pengguna.
	//
	// # Kenapa judulnya TIDAK diambil dari layar lama
	//
	// Karena layar lama TIDAK punya judul tab. Kedua grid-nya digambar bertumpuk di satu
	// halaman tanpa kontainer tab sama sekali — yang membedakannya hanya urutan dan jumlah
	// kolomnya. `Harness/InboxKomunikasiCabang-Harness.xml` hanya memuat satu judul,
	// "Inbox Komunikasi".
	//
	// Judul di bawah karena itu DIBUAT, dan diambil dari kata yang sudah dipakai sistem
	// lama pada pencacahnya sendiri — "Answered" dan "Not Answered" pada
	// `PNCCountKomunikasiCabang_Act` langkah 10 dan 13 — diterjemahkan ke bahasa yang
	// dipakai seluruh layar ini.
	Name string

	// Description menjelaskan isi partisinya dalam satu kalimat.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	Columns []Column

	// Answered menyatakan penyaring `REPLYMESSAGE` tab ini.
	//
	//	false  IS NULL      — tab Belum Dijawab
	//	true   IS NOT NULL  — tab Sudah Dijawab
	//
	// Nilainya BUKAN sekadar keterangan: ia yang memilih kueri di repo/sqlstore dan yang
	// menyaring di repo/memory, sehingga keduanya tidak dapat berselisih.
	Answered bool

	// Notice adalah keterangan yang berlaku pada TAB INI SAJA, ditampilkan di atas grid.
	//
	// Kosong berarti tidak ada keterangan khusus.
	Notice string
}

// Kolom tab "Belum Dijawab" — TIGA kolom.
//
// Diambil dari sel grid kedua pada `Section/InboxKomunikasi-Section.xml`, dibaca berurutan
// dari offset ~527.000:
//
//	Tanggal         -> .CloseClaimDate   <- CREATEDDATE
//	Pengirim(Dari)  -> section PengirimKomunikasi
//	Pesan           -> .Email            <- MESSAGE
//
// Kolom balasan dan penjawab TIDAK digambar di sini, dan itu bukan kelalaian: penyaring tab
// ini `REPLYMESSAGE IS NULL`, sehingga keduanya dijamin kosong pada setiap baris.
func notAnsweredColumns() []Column {
	return []Column{
		{Key: FieldCreatedAt, Title: "Tanggal"},
		{Key: FieldSender, Title: "Pengirim(Dari)"},
		{Key: FieldMessage, Title: "Pesan"},
	}
}

// Kolom tab "Sudah Dijawab" — LIMA kolom.
//
// Diambil dari sel grid pertama pada section yang sama, dibaca berurutan dari offset
// ~363.000:
//
//	Tanggal           -> .CloseClaimDate   <- CREATEDDATE
//	Pengirim(Dari)    -> section PengirimKomunikasi
//	Pesan             -> .Email            <- MESSAGE
//	Jawaban Terakhir  -> .CloseClaimNote   <- REPLYMESSAGE
//	Penjawab(Dari)    -> section PenjawabKomunikasi
//
// Perhatikan urutannya: balasan lebih dulu, penjawabnya belakangan. Itu urutan section
// aslinya dan dibawa apa adanya (`D-13`), meski membaca "siapa" sesudah "apa" terasa
// terbalik.
func answeredColumns() []Column {
	return []Column{
		{Key: FieldCreatedAt, Title: "Tanggal"},
		{Key: FieldSender, Title: "Pengirim(Dari)"},
		{Key: FieldMessage, Title: "Pesan"},
		{Key: FieldReply, Title: "Jawaban Terakhir"},
		{Key: FieldReplier, Title: "Penjawab(Dari)"},
	}
}

// tabs adalah kedua tab beserta kolomnya, berurutan seperti tampilnya.
//
// Urutannya mengikuti urutan grid di dalam section: grid "Sudah Dijawab" digambar LEBIH
// DULU di berkasnya (offset ~363.000) daripada grid "Belum Dijawab" (~527.000).
//
// Urutan di sini SENGAJA dibalik dari itu, dan itu satu-satunya tempat modul ini tidak
// mengikuti section apa adanya. Alasannya: layar lama menggambar keduanya BERTUMPUK
// sekaligus, sehingga tidak ada satu pun tab yang "pertama" bagi penggunanya — yang ada
// hanyalah urutan gulir. Begitu keduanya menjadi tab, salah satunya harus terbuka lebih
// dulu, dan yang dipilih adalah yang memuat pekerjaan yang menunggu.
//
// Ia dinyatakan lewat PlannedDifferences, bukan disamarkan.
var tabs = []Tab{
	{
		Code: TabNotAnswered,
		Name: "Belum Dijawab",

		Description: "Percakapan yang pesannya belum dibalas sama sekali. Diurutkan dari " +
			"yang PALING LAMA menunggu, sehingga baris teratas adalah yang paling perlu " +
			"dijawab.",

		Columns:  notAnsweredColumns(),
		Answered: false,
	},
	{
		Code: TabAnswered,
		Name: "Sudah Dijawab",

		Description: "Percakapan yang sudah dibalas. Diurutkan dari balasan TERBARU, " +
			"berlawanan dengan tab sebelah — dan itu perilaku layar lama apa adanya.",

		Columns:  answeredColumns(),
		Answered: true,

		Notice: "Tab ini menampilkan BALASAN TERAKHIR pada setiap percakapan, bukan " +
			"seluruh isinya. Satu baris di tabel ini menyimpan satu pesan beserta satu " +
			"balasan; utas lengkapnya dibuka lewat tombol \"Detail Komunikasi\".",
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

// ExportColumns adalah kolom berkas ekspor.
//
// # Kenapa ia SATU daftar untuk kedua tab, padahal grid-nya berbeda
//
// Karena berkas yang kolomnya berubah-ubah menurut tab yang kebetulan terbuka tidak dapat
// digabungkan maupun dibandingkan oleh penerimanya. Yang diekspor adalah baris yang sama
// dari tabel yang sama; yang berbeda hanyalah kolom mana yang DIGAMBAR di layar.
//
// Ia karena itu memuat SELURUH isian, termasuk tiga yang tidak digambar grid mana pun:
// nomor percakapan, status register, dan tanggal balasan. Ketiganya ada di kueri lama dan
// dibuang layar; di berkas ekspor ia berguna justru karena berkas dibaca di luar layar.
//
// Sistem lama TIDAK punya tombol ekspor di layar ini, sehingga seluruh daftar ini adalah
// KEMAMPUAN BARU — dinyatakan lewat PlannedDifferences.
var ExportColumns = []Column{
	{Key: FieldCreatedAt, Title: "Tanggal"},
	{Key: FieldSender, Title: "Pengirim(Dari)"},
	{Key: FieldRecipient, Title: "Tujuan"},
	{Key: FieldMessage, Title: "Pesan"},
	{Key: FieldReply, Title: "Jawaban Terakhir"},
	{Key: FieldReplier, Title: "Penjawab(Dari)"},
	{Key: FieldRepliedAt, Title: "Tanggal Jawaban"},
	{Key: FieldStatus, Title: "Status Register"},
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
	"Kedua daftar dipisahkan menjadi TAB. Layar lama menggambar keduanya bertumpuk pada " +
		"satu halaman — daftar yang sudah dijawab di atas, yang belum dijawab di bawah — " +
		"tanpa kontainer tab sama sekali. Isinya, penyaringnya, dan urutannya tidak " +
		"berubah; yang berubah hanya cara berpindah di antara keduanya. Tab yang terbuka " +
		"lebih dulu adalah \"Belum Dijawab\", karena itulah pekerjaan yang menunggu.",

	"Urutan kedua tab BERLAWANAN, dan itu perilaku layar lama apa adanya: yang belum " +
		"dijawab diurutkan dari tanggal pesan TERLAMA, yang sudah dijawab dari tanggal " +
		"balasan TERBARU. Bila tabel terbaca tidak konsisten saat Anda berpindah tab, " +
		"inilah sebabnya.",

	"Petugas yang kode cabangnya TIDAK DAPAT DITURUNKAN melihat percakapan KANTOR PUSAT. " +
		"Layar lama menyatukan dua keadaan yang berbeda — petugas kantor pusat, dan " +
		"petugas yang cabangnya tidak terbaca — lalu memperlakukan keduanya sama. " +
		"Keputusan Work Owner 2026-09-24: direplikasi apa adanya. Yang perlu disadari, " +
		"petugas non-karyawan tidak terdaftar di HRD sehingga cabangnya memang tidak " +
		"pernah terbaca; bila Anda melihat percakapan yang bukan urusan cabang Anda, " +
		"laporkan — kemungkinan inilah sebabnya.",

	"Kolom \"Pengirim(Dari)\" menampilkan KODE CABANG apa adanya untuk pengirim dari " +
		"cabang, sementara kolom tujuan hanya berbunyi \"CABANG\" tanpa menyebut yang " +
		"mana. Asimetri itu ada di layar lama dan tidak diperbaiki: memperbaikinya berarti " +
		"menampilkan kode cabang di tempat pengguna hari ini membaca kata \"CABANG\".",

	"Tindakan \"Kirim Pesan\", \"Balas\", \"Selesai Komunikasi\", dan \"Tambah\" belum " +
		"tersedia. Keempatnya MENULIS ke tabel komunikasi, dan selama Pega dan sistem baru " +
		"berjalan berdampingan tabel itu hanya boleh ditulis satu sistem — hari ini Pega. " +
		"Satu di antaranya bahkan tidak dapat direplikasi sama sekali: activity di balik " +
		"tombol \"Balas\" TIDAK ADA di export mana pun, sehingga tidak ada yang dapat " +
		"dibaca untuk ditulis ulang. Tombolnya tetap digambar supaya keberadaannya " +
		"terlihat, dan penekanannya menjawab alasan — bukan halaman kosong.",

	"Layar \"Detail Komunikasi\" dibangun dari bentuk section-nya sendiri, karena kueri " +
		"pemasoknya HILANG dari export. Yang terbaca dari section adalah ketiga kolomnya " +
		"— Tanggal, Pengirim, Pesan — beserta kotak balasan; yang TIDAK terbaca adalah " +
		"penyaring persisnya. Yang dipakai di sini hanyalah nomor percakapan, satu-satunya " +
		"parameter yang benar-benar dikirim tombolnya. Bila utas yang tampil berbeda dari " +
		"Pega, inilah tempat pertama yang harus diperiksa.",

	"Balasan digambar sebagai baris tersendiri di bawah pesannya pada layar Detail " +
		"Komunikasi. Section lama hanya menggambar tiga kolom dan tidak menampilkan " +
		"balasan sama sekali, padahal satu baris tabel menyimpan pesan DAN balasannya — " +
		"sehingga utasnya terbaca separuh. Ketiga kolomnya tetap utuh; yang ditambahkan " +
		"adalah barisnya, bukan kolomnya.",

	"Daftar lampiran percakapan dapat DIBACA, tetapi berkasnya belum dapat diunduh dan " +
		"lampiran baru belum dapat ditambahkan. Yang ditampilkan adalah jenis dokumen, " +
		"rincian, catatan, dan tanggal unggahnya.",

	"Status \"Sudah Upload\" pada lampiran disimpulkan dari tanggal unggah setiap " +
		"lampiran. Kueri lama menyimpulkannya dari sebuah pencacah yang membandingkan " +
		"hasil hitung dengan nol secara terbalik, sehingga jawabannya SELALU \"Belum " +
		"Upload\" berapa pun isinya. Itu cacat yang tidak dibawa.",

	"Tombol unduh menghasilkan berkas berisi DELAPAN kolom — termasuk tiga yang tidak " +
		"digambar tabel mana pun: tujuan pesan, tanggal jawaban, dan status register. " +
		"Layar lama tidak punya tombol unduh sama sekali, sehingga seluruh berkas ini " +
		"adalah kemampuan baru. Ketiga kolom tambahan diambil dari kueri lama yang memang " +
		"sudah mengambilnya lalu membuangnya di layar.",

	"Kedua pencacah di atas tabel menghitung dengan penyaring yang SEDIKIT BERBEDA dari " +
		"tabelnya sendiri: pencacah memeriksa dua kolom balasan, tabel hanya memeriksa " +
		"satu. Akibatnya jumlah pada pencacah dapat tidak sama persis dengan jumlah baris " +
		"di tabel bila ada percakapan yang salah satu kolom balasannya terisi sendirian. " +
		"Keduanya dibawa apa adanya karena keduanya memang begitu di layar lama.",

	"Daftar dipotong per halaman di basis data. Grid Pega menarik seluruh barisnya lebih " +
		"dulu lalu menomori halamannya di memori; di sini halamannya dipotong sebelum " +
		"baris meninggalkan basis data, dan jumlah seluruhnya tetap dihitung tepat. " +
		"Ukuran halamannya tetap 20, sama dengan layar lama.",

	"Nama pengirim yang tersimpan di kolom nama TIDAK ditampilkan, dan itu bukan data " +
		"yang hilang. Kueri lama memberi DUA kolom nama alias yang sama persis di dalam " +
		"satu perintah, sehingga yang sampai ke layar hanyalah yang terakhir — kode asal, " +
		"bukan nama orang. Yang tampil di layar lama karena itu juga bukan nama pengirim.",
}
