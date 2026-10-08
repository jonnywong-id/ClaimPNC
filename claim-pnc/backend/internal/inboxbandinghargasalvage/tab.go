package inboxbandinghargasalvage

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada AppealRow yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah supaya layar tidak perlu tahu nama isian apa pun.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`), diambil
	// dari `pyCaption` pada `Section/InboxReqSalvageASM-Section.xml`.
	Title string

	// Numeric menandai kolom yang isinya angka, supaya layar meratakannya ke kanan dan
	// memformatnya sebagai bilangan.
	//
	// Ia sifat KOLOM, bukan sifat nilainya: nilai uang dibawa sebagai teks demi menjaga
	// presisi (`I-12`), sehingga layar tidak dapat menyimpulkannya dari tipe datanya.
	Numeric bool
}

// Nama field JSON pada satu baris.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan — ketiganya
// merujuk nama yang sama.
const (
	FieldClaimNo      = "no_klaim"
	FieldRequestDate  = "tanggal_request"
	FieldDetailObject = "detail_object"
	FieldItemName     = "nama_barang"
	FieldItemPrice    = "harga_barang"
	FieldRequestPrice = "harga_request"
	FieldRequestNote  = "note_request"
	FieldAging        = "aging"
	FieldCheckerNote  = "note_checker"

	FieldSalvageType     = "object_name"
	FieldSalvageLocation = "lokasi_salvage"
	FieldPIC             = "pic"
)

// Kode tab pada kontrak API.
//
// # Kenapa berbentuk kata, sementara Pega memakai angka
//
// Karena angkanya tidak menyatakan apa pun. `Activity/SetReqSalvage_Act-Act.xml` menerima
// parameter `tipe` bernilai `1` atau `2`, dan nilai itu lalu disetel ke DUA properti sekaligus
// (`tempQuery.FlagReject` dan `tempQuery.FlagASO`) yang kemudian menyalakan kontainer grid
// yang bersangkutan.
//
// Angka aslinya TIDAK dibuang; ia tetap dibawa sebagai Tab.PegaParam, dan itulah yang
// menghubungkan tab di layar dengan langkah activity saat uji kesetaraan gerbang 1 menemukan
// selisih.
const (
	TabRequest = "request-banding-harga"
	TabHistory = "history-cheker"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// Request Banding Harga, karena ia kontainer pertama pada
// `Section/InboxReqSalvageASM-Section.xml` (`FlagASO==1`, sebelum `FlagASO=2`) — dan karena
// isinya justru yang menuntut tindakan: banding yang belum diputus siapa pun.
const DefaultTab = TabRequest

// Tab adalah satu antrean kerja pada layar Inbox Banding Harga Salvage.
type Tab struct {
	// Code adalah kode tab pada kontrak API.
	Code string

	// PegaParam adalah nilai `tipe` yang dipakai sistem lama untuk tab ini.
	//
	// Ia dibawa demi ketelusuran, bukan untuk dipakai layar: tanpa nilai ini, menelusuri
	// balik sebuah selisih ke langkah activity Pega menempuh satu tabel terjemahan yang
	// tidak tertulis di mana pun.
	PegaParam string

	// Name adalah judul tab yang dibaca pengguna.
	//
	// Diambil dari `Local.LOOP` pada `Activity/GCNMCountRequestSalvage_act-Act.xml` langkah
	// 9 dan 11 — yakni teks yang selama ini muncul di kolom "Status Salvage" tabel ringkas.
	// Salah ketik "Cheker" dipertahankan apa adanya, karena itulah yang dibaca pengguna hari
	// ini (`D-13`), sama seperti "Assesment" pada Inbox Service Center.
	Name string

	// Description menjelaskan isi antreannya dalam satu kalimat. Sistem lama tidak punya
	// keterangan seperti ini; ia ditambahkan karena dua judul yang sama-sama menyebut
	// "banding" tidak memberi tahu apa pun tentang baris mana yang masuk ke mana.
	Description string

	// Columns adalah kolom grid tab ini, berurutan seperti di Pega.
	Columns []Column

	// Decided menyatakan tab ini berisi banding yang SUDAH diputus.
	//
	// Ia yang membedakan kedua kueri: tab yang belum diputus menyaring `TGLAPPROVE IS NULL`
	// pada tabel checker, sedangkan yang sudah menyaring klaim yang punya baris checker
	// ber-`STATUSAPPROVE IS NOT NULL AND TGLAPPROVE IS NOT NULL`.
	//
	// Disimpan sebagai sifat tab, bukan disimpulkan dari kodenya, supaya penambahan tab
	// ketiga kelak tidak menuntut percabangan baru di tempat lain.
	Decided bool
}

// requestColumns adalah kesembilan kolom grid "Request Banding Harga".
//
// Urutan dan judulnya dibaca dari sel grid `Section/InboxReqSalvageASM-Section.xml` pada
// offset 135.272–175.772, dan properti yang mengikatnya dari offset 179.785–239.685. Pasangan
// keduanya, satu per satu:
//
//	Tanggal Request  .TglTerimaSalvage  <- TGLREQUEST
//	No Klaim         .ClaimID           <- NOKLAIM
//	Detail Object    .ClientName        <- IDDETAILSALVAGE
//	Nama Barang      .BranchName        <- NAMABARANG
//	Harga Barang     .AgentID           <- HARGABARANG
//	Harga Request    .Email             <- HARGAREQUEST
//	Note Request     .BranchID          <- ALASANREQUEST
//	Aging            .AgingAmount       <- TRUNC(SYSDATE) - TRUNC(TGLREQUEST)
//	Note Checker     .NoteKomite        <- NOTEKOMITE
//
// Kolom kesepuluh di Pega adalah **Action**, berisi section `ButtonApproveRejectedRequest`.
// Ia TIDAK didaftarkan di sini: ia kontrol, bukan data, dan sectionnya pun tidak ada di
// export sehingga tombolnya belum dapat dibangun (lihat Limitations).
func requestColumns() []Column {
	return []Column{
		{Key: FieldRequestDate, Title: "Tanggal Request"},
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldDetailObject, Title: "Detail Object"},
		{Key: FieldItemName, Title: "Nama Barang"},
		{Key: FieldItemPrice, Title: "Harga Barang", Numeric: true},
		{Key: FieldRequestPrice, Title: "Harga Request", Numeric: true},
		{Key: FieldRequestNote, Title: "Note Request"},
		{Key: FieldAging, Title: "Aging"},
		{Key: FieldCheckerNote, Title: "Note Checker"},
	}
}

// historyColumns adalah keempat kolom grid "History Cheker".
//
// Sel gridnya ada pada offset 371.464–383.361, propertinya pada 387.407–405.928:
//
//	No Klaim        .ClaimID     <- PNC_SALVAGE.NOKLAIM
//	Object Name     .Email       <- PNC_SALVAGE.JENISSALVAGE   (!)
//	Lokasi Salvage  .AgentID     <- PNC_SALVAGE.LOKASISALVAGE
//	PIC             .BranchName  <- PNC_SALVAGE.PIC
//
// Tanda (!) menandai judul yang menyebut hal BERBEDA dari isinya: kolom berjudul "Object
// Name" sebenarnya berisi jenis salvage. Judulnya dipertahankan (`D-13`), tetapi nama isiannya
// di kode menyebut apa yang benar-benar ada di dalamnya (`D-19`).
func historyColumns() []Column {
	return []Column{
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldSalvageType, Title: "Object Name"},
		{Key: FieldSalvageLocation, Title: "Lokasi Salvage"},
		{Key: FieldPIC, Title: "PIC"},
	}
}

// SearchLabel adalah judul kotak pencarian, sama pada kedua tab.
//
// `pyCaption Cari No Klaim` muncul DUA kali di section — satu per grid — dan keduanya
// berpasangan dengan tombol `Cari` dan `Clear` serta petunjuk `Contoh : PNC-1234`.
const (
	SearchLabel       = "Cari No Klaim"
	SearchPlaceholder = "Contoh : PNC-1234"
)

// tabs adalah kedua antrean dalam urutan tampilnya.
var tabs = []Tab{
	{
		Code:      TabRequest,
		PegaParam: "1",
		Name:      "Request Banding Harga",
		Description: "Banding harga yang BELUM diputus — balai lelang mengajukan harga " +
			"berbeda dari harga yang diajukan PIC, dan keputusannya menunggu Anda.",
		Columns: requestColumns(),
	},
	{
		Code:      TabHistory,
		PegaParam: "2",
		Name:      "History Cheker",
		Description: "Pengajuan salvage yang banding harganya SUDAH Anda putuskan. " +
			"Barisnya PENGAJUAN, bukan barang — satu klaim dapat muncul beberapa kali bila " +
			"ia punya lebih dari satu pengajuan salvage.",
		Columns: historyColumns(),
		Decided: true,
	},
}

// Tabs mengembalikan kedua tab dalam urutan tampilnya.
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
func copyTab(tab Tab) Tab {
	clone := tab
	clone.Columns = make([]Column, len(tab.Columns))
	copy(clone.Columns, tab.Columns)
	return clone
}

// PlannedDifferences menyatakan hal yang SENGAJA berbeda dari layar lama.
//
// Ia dipisahkan dari Limitations karena keduanya menjawab pertanyaan yang berbeda:
// Limitations menyatakan yang BELUM ada, berkas ini menyatakan yang sengaja TIDAK SAMA. Uji
// kesetaraan gerbang 1 akan melaporkan seluruhnya sebagai selisih, dan selisih yang tidak
// dinyatakan lebih dulu akan diperlakukan sebagai bug (`P-5`, `D-54`).
//
// Kelima butir pertama disetujui Work Owner pada 2026-09-29, dan kedua butir terakhir pada
// 2026-09-30 — seluruhnya setelah dibacakan satu per satu beserta buktinya.
func PlannedDifferences() []string {
	return []string{
		"Kolom Aging diurutkan sebagai ANGKA hari, bukan sebagai teks. Kueri lama menyusunnya " +
			"sebagai `TRUNC(SYSDATE)-TRUNC(tglrequest) || ' days'` lalu mengurutkannya " +
			"`ORDER BY \"AgingAmount\" DESC`, sehingga '9 days' didahulukan dari '30 days' — " +
			"justru banding yang paling lama menunggu yang tenggelam ke halaman belakang. " +
			"Tampilannya tetap '<n> days'.",

		"Kolom Note Checker kini ikut dipilih kuerinya — kolomnya bernama `NOTEAPPROVE`. " +
			"Ia TETAP kosong di daftar ini, dan itu bukan cacat: daftar ini hanya memuat " +
			"banding yang BELUM diputus, sedangkan catatan komite ditulis bersamaan dengan " +
			"tanggal putusan. Baris yang catatannya terisi karena itu sudah tidak ada di " +
			"sini. Keterangan sebelumnya yang menyebut Pega \"lupa memilih kolomnya\" " +
			"KELIRU dan dicabut.",

		"Kueri pencacah tidak lagi memuat `ORDER BY TGL_REQUEST DESC`. Klausa itu tidak " +
			"bermakna pada `SELECT COUNT(1)`, menyebut kolom yang tidak dikenal di mana pun " +
			"selain baris itu, dan pada Oracle berpotensi membuat kuerinya galat — yang " +
			"akibatnya total halaman tidak pernah terisi.",

		"Angka antrean kini SAMA dengan jumlah baris gridnya. Di layar lama pencacah dan " +
			"daftarnya menghitung populasi yang berbeda: daftarnya menyaring " +
			"`HARGAREQUEST IS NOT NULL` pada tabel checker, sementara pencacahnya " +
			"menyaring `NILAI_REQUEST IS NOT NULL` pada tabel detail salvage lewat JOIN yang " +
			"dapat melipatgandakan baris. Angkanya kini digambar sebagai lencana pada tab, " +
			"bukan sebagai tabel ringkas tersendiri — lihat selisih tampilan di bawah.",

		"TAMPILAN — kedua antrean dipilih lewat BILAH TAB. Di Pega keduanya dua kontainer " +
			"bersyarat pada section yang sama (`tempQuery.FlagASO==1` dan `==2`), tanpa " +
			"kendali terlihat di dalam section itu. Sakelarnya setara; bentuknya tidak.",

		"TAMPILAN — jumlah antrean digambar sebagai lencana pada tab, dan HANYA bila " +
			"antreannya berisi. Pega tidak punya penghitung apa pun di layar ini: tabel " +
			"ringkas \"Status Salvage / Jumlah\" milik layar Inbox Salvage, dan " +
			"`GCNMCountRequestSalvage_act` yang memasok angkanya tidak dipanggil harness " +
			"maupun section mana pun.",

		"TAMPILAN — tombol \"Refresh\" di kepala halaman TIDAK ada padanannya di Pega. " +
			"Harness maupun section layar ini tidak memuat satu tombol pun di luar kolom aksi; " +
			"di sana daftar disegarkan dengan memuat ulang layarnya. Tombol ini ditambahkan " +
			"mengikuti kebiasaan aplikasi baru, bukan menyalin Pega.",

		"TAMPILAN — judul kolom tombol \"Aksi\", sedangkan literal Pega-nya \"Action\". Ini " +
			"satu-satunya judul kolom yang sengaja tidak menyalin Pega, berlaku seluruh " +
			"modul (ketetapan Work Owner 2026-10-03). Label TOMBOLNYA tetap literal Pega: " +
			"Approve, Reject, Lihat File.",

		"Kata kunci pencarian dikirim sebagai parameter terikat, bukan dirangkai ke dalam " +
			"teks SQL. Sistem lama menyusunnya sebagai `\"and noklaim='\" + kata kunci + \"'\"` " +
			"— apa yang diketik pengguna, langsung ke dalam kueri. Hasil pencariannya sama " +
			"persis; yang berubah hanya cara nilainya masuk (`11-SECURITY.md` §5).",

		"Approve dan Reject berjalan dalam SATU transaksi. Di layar lama ketiga " +
			"pernyataannya berdiri sendiri dan menyimpan seketika, sehingga kegagalan di " +
			"tengah meninggalkan putusan tanpa harga — atau harga tanpa putusan. Di sini " +
			"kegagalan mengembalikan keadaan seperti semula (`D-68`). Ini mengubah keadaan " +
			"akhir SAAT GAGAL, bukan saat berhasil, sehingga perbandingannya tidak boleh " +
			"dilakukan atas kasus kegagalan (`14-TESTING-STRATEGY.md` §6.4).",

		"Penandaan dokumen saat banding DITOLAK ditiru apa adanya, termasuk perbandingan " +
			"kolom `SALAVAGEDOCUMENT.NOKLAIM` dengan sebuah ID DETAIL salvage. Nama kolom " +
			"itu menyesatkan, bukan penyaringnya: kueri dialog \"Lihat File\" melakukan " +
			"perbandingan yang SAMA, dan ia kueri baca yang hasilnya langsung terlihat " +
			"pengguna — bila tidak pernah cocok, dialognya selalu kosong. Kolom itu karena " +
			"itu berisi id detail salvage meski namanya menyatakan nomor klaim. " +
			"Keterangan sebelumnya yang menyatakan penandaan ini TIDAK PERNAH berjalan " +
			"KELIRU dan dicabut.",
	}
}

// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta alasannya.
//
// Ia data, bukan komentar, supaya dapat dikirim apa adanya ke layar — dan supaya hilang
// dengan sendirinya begitu penghalangnya hilang, tanpa menyunting frontend.
//
// Seluruhnya punya jejak bukti; tidak satu pun ditulis berdasarkan dugaan.
func Limitations() []string {
	return []string{
		"Keputusan Approve dan Reject TERSIMPAN, tetapi BELUM DIKIRIM ke balai lelang. " +
			"Layanan `SendData_SalvageSimasBid` menunjuk host DEV dan berjalan tanpa " +
			"autentikasi, sehingga memanggilnya dari produksi berarti mengirim putusan atas " +
			"nilai uang ke lingkungan yang salah. Work Owner memutuskan membangunnya tanpa " +
			"pemanggilan itu (2026-09-30). Akibatnya harus disadari: balai lelang tidak " +
			"mengetahui putusan Anda sampai alamat produksinya ditetapkan (sekelas `R-18`).",

		"Menyetujui TIDAK selalu mengubah harga barang. Penerapan harga di layar lama " +
			"dijaga syarat bernama satu orang, sehingga bagi komite lain tombol Approve " +
			"hanya MENCATAT putusan. Perilaku itu ditiru apa adanya, dan hasil tiap " +
			"keputusan menyatakan mana yang benar-benar terjadi — bukan \"berhasil disimpan\" " +
			"yang menyiratkan lebih.",

		"Baris pada grid Request tidak dapat dibuka — dan di layar lama pun TIDAK. Grid itu " +
			"berkonfigurasi `pyEditingMode = expandPane` dengan " +
			"`pyEditAction = ShowDetailSalvageInboxOSClose`, tetapi rule itu **tidak ada di " +
			"aplikasi Pega-nya**: ia dirujuk tiga section sebagai `pyEditAction` dan tidak " +
			"terindeks di satu pun `pxRuleReferences`, sementara kedua `pyEditAction` lain " +
			"pada section yang SAMA terindeks sebagai `Rule-Obj-FlowAction`. Jadi ia rujukan " +
			"menggantung, bukan artefak yang kurang dari export — tidak ada kemampuan yang " +
			"hilang, dan tidak ada yang perlu diminta. Keterangan sebelumnya yang menyebutnya " +
			"\"tidak ada di export\" menyesatkan dan dicabut.",

		"Daftar dokumen pada \"Lihat File\" hanya menampilkan yang BELUM ditandai ditolak. " +
			"Itu ditiru dari kueri lama (`IDBALAILELANG IS NULL`), bukan pilihan di sini — " +
			"akibatnya, dokumen banding yang pernah Anda tolak tidak dapat dibuka lagi " +
			"dari layar ini.",

		"Kolom \"Kategori\" pada dialog \"Lihat File\" SELALU bertuliskan " +
			"\"BandingHarga\". Ia konstanta yang ditetapkan activity lama untuk setiap " +
			"baris, bukan isi kolom basis data — nilai `ATTACHNOTE` yang sesungguhnya " +
			"tidak pernah dibaca layar lama.",

		"Balasan ke balai lelang belum dibangun. Percakapannya tersimpan di " +
			"HISTORY_KOMUNIKASI_SALVAGE dengan dua kolom — MESSAGE_SIMASBID untuk pengajuan " +
			"mereka dan MESSAGE_PNC untuk balasan kita — dan satu-satunya rule yang " +
			"menyentuhnya di export hanya menulis kolom yang pertama.",

		"Pengajuan salvage BARU tidak dibuat dari layar ini, melainkan dari layar Inbox " +
			"Salvage. Keterangan sebelumnya yang menyebut layar lama punya tombol " +
			"\"Tambah\" di sini KELIRU dan dicabut: harness maupun section layar ini tidak " +
			"memuat tombol itu sama sekali, dan tidak pula tombol lain. Keduanya hanya " +
			"memuat kotak cari, kedua grid, dan tombol pada kolom aksi grid Request.",

		"Pemeriksaan peran belum ada. Butir menu ini dijaga `When/IsGCNMUser` di Pega, dan " +
			"rule itu berisi `1 = 2` — yakni sakelar \"jangan tampilkan ini\", bukan " +
			"pemeriksaan peran. Yang menentukan siapa melihat butirnya sekarang adalah " +
			"M_OTORISASI_PNC, sama seperti butir lain (TKT-F3-004).",

		"Yang membatasi baris mana yang Anda lihat BUKAN peran, melainkan kepemilikan: " +
			"hanya banding yang NAMAKOMITE-nya Anda sendiri. Dua nama operator di layar lama " +
			"menyimpang dari aturan itu, dan penyimpangannya ditiru apa adanya atas keputusan " +
			"Work Owner — lihat keterangan di kepala layar bila ia berlaku bagi Anda.",
	}
}
