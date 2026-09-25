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

// Nama kolom TOMBOL pada grid.
//
// # Kenapa ia kolom, bukan hiasan yang ditambahkan layar sendiri
//
// Karena begitulah bentuknya di layar lama. Kedua tombol berada DI DALAM grid sebagai kolom
// tersendiri — bukan di bilah aksi di atas tabel — dan itu terbaca dari offsetnya sendiri:
// keduanya muncul di dalam rentang kedua grid, dua kali, sekali untuk tiap grid.
//
//	Detail Komunikasi   offset 432.068 (grid Sudah Dijawab) · 569.984 (grid Belum Dijawab)
//	Selesai Komunikasi  offset 443.422                      · 585.877
//
// Menaruhnya di layar alih-alih di sini akan membuat bentuk grid hidup di dua tempat, dan
// yang satu akan tertinggal — persis alasan seluruh kolom lain pun datang dari server.
//
// Keduanya TIDAK punya isian pada baris: yang digambar adalah tombolnya, dan yang dibawanya
// adalah nomor percakapan yang sudah ada di `Conversation.ID`.
const (
	// FieldActionDetail menggambar tombol **"Detail Komunikasi"**.
	//
	// Di Pega ia menjalankan `runDataTransform` atas `DetailKomunikasi_dt` dengan dua
	// parameter, lalu membuka `localAction` `DETAILKOMUNIKASICABANG_11`:
	//
	//	KOMID      = .ClaimNo    <- KOMUNIKASIID, nomor percakapan
	//	KODECABANG = .pzInsKey   <- CASEID
	//
	// PERANGKAP PENAMAAN: parameter bernama `KODECABANG` menerima `CASEID`, yaitu penanda
	// kanal (`CABANG`) — BUKAN kode cabang. Ia satu lagi nama Pega yang menyebut hal lain,
	// dan tidak dibawa (`D-19`). Yang dikirim modul ini hanyalah nomor percakapannya.
	FieldActionDetail = "aksi_detail"

	// FieldActionFinish menggambar tombol **"Selesai Komunikasi"**.
	//
	// Di Pega ia menjalankan `EndKomunikasiCabang` dengan satu parameter —
	// `KOMID = .ClaimNo` — lalu me-refresh grid. Activity itu mengubah `CASEID` menjadi
	// `CABANG SELESAI`, sehingga barisnya HILANG dari kedua tab.
	//
	// Ia MENULIS, dan sejak 2026-09-24 ia benar-benar menulis di sini — bukan lagi ditolak
	// dengan alasan. Akibatnya TIDAK DAPAT DIBATALKAN dari layar mana pun: sistem lama tidak
	// punya satu pun tindakan yang membuka kembali percakapan yang sudah ditutup.
	FieldActionFinish = "aksi_selesai"
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
	return append([]Column{
		{Key: FieldCreatedAt, Title: "Tanggal"},
		{Key: FieldSender, Title: "Pengirim(Dari)"},
		{Key: FieldMessage, Title: "Pesan"},
	}, actionColumns()...)
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
	return append([]Column{
		{Key: FieldCreatedAt, Title: "Tanggal"},
		{Key: FieldSender, Title: "Pengirim(Dari)"},
		{Key: FieldMessage, Title: "Pesan"},
		{Key: FieldReply, Title: "Jawaban Terakhir"},
		{Key: FieldReplier, Title: "Penjawab(Dari)"},
	}, actionColumns()...)
}

// Kedua kolom TOMBOL, IDENTIK di kedua tab.
//
// # Judulnya memang "Button", dan itu bukan kelalaian penyalinan
//
// Kedua kolom menuliskan `pyCaption Button` sebagai judulnya — terbaca pada offset 424.608
// dan 439.774 (grid Sudah Dijawab), 567.606 dan 583.351 (grid Belum Dijawab). Dua kolom
// berjudul sama di satu tabel memang tidak membantu, tetapi `D-13` menetapkan teks yang
// dilihat pengguna mengikuti layar lama apa adanya.
//
// Yang DITAMBAHKAN di sistem baru bukan judulnya melainkan nama yang dibaca pembaca layar:
// setiap tombol membawa label lengkap beserta nomor percakapannya, sehingga kedua kolom
// tetap dapat dibedakan tanpa melihat.
//
// Urutannya mengikuti urutan di section: Detail lebih dulu, Selesai sesudahnya.
func actionColumns() []Column {
	return []Column{
		{Key: FieldActionDetail, Title: "Button"},
		{Key: FieldActionFinish, Title: "Button"},
	}
}

// IsAction menyatakan sebuah kolom menggambar TOMBOL, bukan isian baris.
//
// Ia dipakai berkas ekspor dan uji: kolom tombol tidak punya nilai yang dapat ditulis ke
// berkas, dan menuliskannya sebagai sel kosong akan menggeser seluruh kolom sesudahnya.
func IsAction(key string) bool {
	return key == FieldActionDetail || key == FieldActionFinish
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

	"Tindakan \"Balas\" dan \"Selesai Komunikasi\" SUDAH dapat dikerjakan di sini sejak " +
		"24 September 2026. Sejak itu tabel POOLDATA.M_KOMUNIKASI_PNC dan " +
		"M_KOMUNIKASI_CABANG ditulis sistem baru, dan `P-1` menuntut layar Pega yang sama " +
		"BERHENTI menulis ke keduanya — dua sistem yang sama-sama menulis satu tabel " +
		"menghasilkan konflik yang hampir mustahil dilacak.",

	"Tindakan \"Kirim Pesan\" dan \"Tambah\" SUDAH dapat dikerjakan di sini. Formnya " +
		"ternyata tidak hilang dari export melainkan tersembunyi: ia blok bersyarat di dalam " +
		"section daftar, bukan section tersendiri. Dengan itu KEEMPAT tindakan tulis layar " +
		"lama bekerja di sistem baru.",

	"Daftar cabang pada pemilih tujuan disusun dari seluruh baris POOLDATA.V_D_SURVEYORS " +
		"yang punya kode dan nama, tanpa duplikat, urut menurut nama. Kueri yang sebenarnya " +
		"MENGISI daftar itu tidak ada di export mana pun — yang terbaca hanyalah kelas " +
		"halamannya dan ketiga kolom yang dipakainya. Bila daftar yang tampil berbeda dari " +
		"Pega, inilah tempat pertama yang harus diperiksa.",

	"Surel \"Notifikasi Komunikasi Cabang Baru\" TIDAK dikirim. Sistem lama mengirimkannya " +
		"ke alamat cabang tujuan pada langkah terakhir, dan activity-nya memuat satu alamat " +
		"surel PRIBADI yang ter-hardcode di jalur produksi — yang tidak boleh dibawa apa pun " +
		"yang terjadi (`D-15`, `D-67`). Penggantinya menuntut seam Notifier beserta master " +
		"Penerima Notifikasi, dan keduanya belum ada di modul ini. Penerima tetap melihat " +
		"pesannya di kotak masuk; yang hilang adalah pemberitahuan lewat surel.",

	"Kolom KODECABANG pada tabel riwayat menerima DUA jenis nilai, tergantung tombol yang " +
		"menulisnya: dari \"Balas\" ia berisi penanda kanal, dari \"Kirim Pesan\" ia berisi " +
		"kode cabang tujuan. Keduanya direplikasi apa adanya — tidak ada satu pun layar yang " +
		"membaca kolom itu, sehingga tidak ada yang dapat membuktikan mana yang benar.",

	"Balasan atas percakapan yang SUDAH DITUTUP ditolak, sementara pernyataan SQL lama " +
		"tidak punya syarat itu. Di Pega syaratnya memang tidak dibutuhkan — layar balasan " +
		"hanya dapat dicapai dari grid, yang sudah menyaring percakapan berjalan. Di sistem " +
		"baru alamatnya dapat dipanggil langsung, dan tanpa penjagaan ini balasannya akan " +
		"\"berhasil\" pada baris yang tetap tidak muncul di kedua tab.",

	"Menyimpan balasan memperbarui percakapan DAN mencatat riwayatnya dalam SATU " +
		"transaksi. Sistem lama menjalankan keduanya sebagai dua langkah terpisah, sehingga " +
		"kegagalan di antara keduanya meninggalkan percakapan yang terbarui tanpa riwayat. " +
		"Selisihnya hanya muncul SAAT GAGAL, dan karena itu tidak akan terlihat pada uji " +
		"kesetaraan yang jalannya mulus (`D-68`).",

	"Panjang balasan dibatasi 4.000 karakter. Sistem lama tidak memeriksa apa pun dan " +
		"menyerahkan penolakannya ke basis data, yang menjawab dengan galat mentah. Lebar " +
		"kolom REPLYMESSAGE yang sebenarnya belum diketahui karena DDL-nya belum ada " +
		"(`R-08`); angka ini penjaga terhadap kiriman yang jelas tidak masuk akal, bukan " +
		"tebakan atas lebar kolomnya.",

	"Layar \"Detail Komunikasi\" menampilkan UTAS percakapan — setiap pesan dan setiap " +
		"balasan sebagai barisnya sendiri, urut dari yang paling awal. Ia dibaca dari tabel " +
		"riwayat komunikasi cabang, bukan dari tabel percakapan yang hanya menyimpan pesan " +
		"dan balasan TERAKHIR.",

	"Kueri pemasok layar \"Detail Komunikasi\" TIDAK ADA di export mana pun. Yang terbaca " +
		"adalah ketiga kolom yang digambar section-nya — Tanggal, Pengirim, Pesan — beserta " +
		"nama kueri yang dipanggil activity-nya. Penyaring dan urutannya karena itu disusun " +
		"dari bentuk itu: nomor percakapan, urut tanggal. Bila utas yang tampil berbeda dari " +
		"Pega, inilah tempat pertama yang harus diperiksa.",

	"Satu nama kolom pada tabel riwayat DITEBAK: kolom tanggalnya. Tidak satu pun INSERT " +
		"mengisinya — ia diisi basis data — dan DDL-nya belum ada. Namanya disamakan dengan " +
		"konvensi tabel saudaranya di skema yang sama. Bila tebakannya salah, layar detail " +
		"menampilkan galat basis data, bukan baris yang keliru.",

	"Kolom \"Pengirim\" pada layar detail menampilkan Operator ID APA ADANYA, berbeda dari " +
		"kolom \"Pengirim(Dari)\" pada grid yang dirakit menjadi `asal (operator)`. Itu bukan " +
		"pilihan tampilan: tabel riwayat tidak memuat kolom asal sama sekali.",

	"Percakapan yang belum punya satu pun baris riwayat tetap DAPAT DIBUKA, dengan utas " +
		"kosong dan keterangannya. Ia keadaan yang nyata untuk percakapan yang dibuat lewat " +
		"layar lain. Menjawabnya \"tidak ditemukan\" akan menyatakan percakapan yang nyata " +
		"itu tidak ada.",

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
