package inboxsalvage

// Column adalah satu kolom pada grid sebuah tab.
//
// Key menyebut ISIAN mana pada Row yang digambar, dan Title menyebut JUDUL yang dibaca
// pengguna. Keduanya dipisah karena satu isian berjudul berbeda dari tab ke tab — kolom
// `PNC_SALVAGE.ESTIMASINILAI` berjudul "Estimasi" pada satu grid dan "Nilai Pengajuan PIC"
// pada grid checker, persis seperti di section-nya.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`).
	Title string

	// Numeric menandai kolom yang berisi nilai uang atau bilangan, supaya layar dapat
	// meratakannya ke kanan dan memformatnya sebagai angka.
	//
	// Ia dinyatakan di sini, bukan ditebak layar dari isinya: nomor klaim juga terlihat
	// seperti angka, dan meratakannya ke kanan akan membuat tabel terbaca keliru.
	Numeric bool
}

// Nama field JSON pada satu baris.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldClaimNo         = "no_klaim"
	FieldSalvageID       = "id_salvage"
	FieldInputDate       = "tanggal_input"
	FieldLossDate        = "tanggal_kejadian"
	FieldPIC             = "pic"
	FieldBusinessName    = "nama_bisnis"
	FieldObjectName      = "nama_objek"
	FieldSalvageType     = "jenis_salvage"
	FieldSalvageLocation = "lokasi_salvage"
	FieldAuctionStatus   = "status_lelang"
	FieldEstimateValue   = "nilai_pengajuan_pic"
	FieldEmail           = "email"
	FieldRemark          = "keterangan_pic"
	FieldRequestValue    = "nilai_request_balai_lelang"
	FieldSubmissionType  = "tipe_pengajuan"
	FieldAging           = "aging"
	FieldNote            = "catatan"
)

// Family adalah keluarga kueri yang memasok sebuah tab.
//
// Ia BUKAN kerapian melainkan isi yang menentukan: ketiga keluarga membaca tabel yang
// berbeda dan mengembalikan kolom yang berbeda. Repo memilih kueri dari sini, dan layar
// memilih kolom dari Tab.Columns — keduanya tidak dapat berselisih karena keduanya
// diturunkan dari daftar tab yang sama.
type Family string

const (
	// FamilyClaim membaca `POOLDATA.T_CLAIM_PNC` dan menyaring `STSSALVAGE`.
	// Satu tab: Salvage Outstanding. Kueri lama `GcnmSalvageData_OS_SQL`.
	FamilyClaim Family = "klaim"

	// FamilyClaimObject sama seperti di atas ditambah nama objek pertama.
	// Lima tab. Kueri lama `GcnmSalvageData_ekonomisdanTba`.
	FamilyClaimObject Family = "klaim-objek"

	// FamilySalvage membaca `POOLDATA.PNC_SALVAGE` dan menyaring `STSTRANSFER`.
	// Tujuh tab. Kueri lama `GcnmSalvageData_CloseOs_SQL`.
	FamilySalvage Family = "salvage"
)

// Kode tab.
//
// # Kenapa kode berbentuk kata, sementara Pega memakai angka
//
// Karena angkanya TIDAK unik di Pega: `Param.tipe` dan `Param.tipe2` adalah dua ruang kode
// yang berbeda, dan keduanya memuat nilai `1`, `2`, `3`, `4`, dan `5`. Memakai angka telanjang
// sebagai kode tab berarti `1` dapat berarti "Salvage Outstanding" atau "Ekonomis"
// tergantung ruang mana yang dimaksud — kekeliruan yang menampilkan daftar yang salah tanpa
// satu pun galat.
//
// Angka aslinya TIDAK dibuang; ia tetap dibawa sebagai Tab.LegacyTipe dan Tab.LegacyTipe2,
// dan dipakai memetakan baris pencacah ke tab (lihat TabForLegacy). Dengan begitu
// penelusuran balik ke export tetap mungkin tanpa mewariskan ambiguitasnya.
const (
	TabOutstanding     = "outstanding"
	TabEkonomis        = "ekonomis"
	TabTBA             = "tba"
	TabTidakEkonomis   = "tidak-ekonomis"
	TabTidakAdaSalvage = "tidak-ada-salvage"
	TabBuyback         = "buyback"
	TabHistori         = "histori"
	TabBalaiLelang     = "balai-lelang"
	TabChecker         = "checker"
	TabRejectedChecker = "rejected-checker"
	TabRequestBalai    = "request-balai-lelang"
	TabSalvageDiterima = "salvage-diterima"
	TabSalvageDitolak  = "salvage-ditolak"
)

// DefaultTab adalah tab yang terbuka saat layar pertama dibuka.
//
// "Salvage Outstanding", karena itulah grid pertama pada `Section/InboxSalvageASM-Section.xml`
// (judulnya di offset ~1.594.000, sebelum seluruh grid lain) dan karena ia awal perjalanan
// sebuah salvage: barang yang klaimnya belum ditandai punya salvage sama sekali.
const DefaultTab = TabOutstanding

// Tab adalah satu daftar pada layar Inbox Salvage.
type Tab struct {
	// Code adalah kode tab, kontrak modul ini sendiri.
	Code string

	// Name adalah judul yang dibaca pengguna, mengikuti `D-13`.
	Name string

	// Description menjelaskan isi daftarnya dalam satu kalimat.
	//
	// Sistem lama tidak punya keterangan seperti ini, dan di layar ini ketiadaannya nyata
	// akibatnya: tiga belas daftar dengan judul yang mirip-mirip — "Checker", "Salvage
	// Diterima", "Balai Lelang", "Request Balai Lelang" — dan tidak ada apa pun yang
	// menjelaskan mengapa sebuah salvage ada di yang satu dan tidak di yang lain.
	Description string

	// Family adalah keluarga kueri yang memasok tab ini.
	Family Family

	// Columns adalah kolom grid tab ini, berurutan seperti tampilnya.
	Columns []Column

	// SalvageStatus adalah nilai `T_CLAIM_PNC.STSSALVAGE` yang disaring.
	//
	// Hanya berlaku pada FamilyClaim dan FamilyClaimObject. Kosong berarti tab ini tidak
	// menyaring kolom itu dengan nilai — lihat SalvageStatusIsNull dan CustomClaimFilter.
	SalvageStatus string

	// SalvageStatusIsNull menyatakan tab ini menyaring `STSSALVAGE IS NULL`.
	//
	// Ia terpisah dari SalvageStatus karena "kosong berarti tidak menyaring" dan "menyaring
	// yang kosong" adalah dua hal yang berbeda, dan keduanya muncul di layar ini.
	SalvageStatusIsNull bool

	// BuybackFilter menyatakan tab ini memakai penyaring buyback alih-alih `STSSALVAGE`.
	//
	// Satu tab saja: Salvage Buyback menyaring klaim yang punya baris adjustment
	// ber-`NILAI_SALVAGE_A` terisi, bukan menyaring `STSSALVAGE`. Lihat langkah 22
	// `Activity/SetDataSalavage_act-Act.xml`.
	BuybackFilter bool

	// TransferStatus adalah nilai `PNC_SALVAGE.STSTRANSFER` yang disaring.
	//
	// Hanya berlaku pada FamilySalvage. Kosong berarti tab ini TIDAK menyaringnya sama
	// sekali — dan itu memang terjadi pada tab Histori Salvage, yang menampilkan seluruh
	// baris salvage.
	TransferStatus string

	// OwnedByCaller menyatakan tab ini hanya menampilkan baris milik pemanggil.
	//
	// Satu tab saja: Request Balai Lelang menyaring `PIC = <pemanggil>`
	// (`Activity/SetDataSalavage_act-Act.xml` langkah 23). Inilah satu-satunya daftar di
	// layar ini yang isinya berbeda dari petugas ke petugas.
	OwnedByCaller bool

	// SearchLabel adalah judul kotak pencarian tab ini, kosong bila tab tidak punya.
	SearchLabel string

	// SearchExact menyatakan pencarian tab ini COCOK PERSIS, bukan mengandung.
	//
	// Satu tab saja: Checker membandingkan `a.noklaim = '…' or a.pic = '…'` — sama dengan,
	// bukan `LIKE '%…%'` seperti keempat tab pencari lainnya. Perbedaannya dibawa apa
	// adanya (`P-5`): pengguna yang mengetik separuh nomor klaim di tab ini tidak
	// mendapatkan apa-apa, dan itu memang yang terjadi di Pega.
	SearchExact bool

	// SearchByPIC menyatakan pencarian tab ini juga mencocokkan kolom PIC.
	//
	// Menyertai SearchExact pada tab Checker: satu kotak mencari DUA kolom sekaligus.
	SearchByPIC bool

	// LegacyTipe dan LegacyTipe2 adalah `Param.tipe` dan `Param.tipe2` sistem lama.
	//
	// Keduanya dibawa untuk dua hal: memetakan baris pencacah ke tab (lihat TabForLegacy),
	// dan menjaga penelusuran balik ke `Activity/SetDataSalavage_act-Act.xml` tetap mungkin.
	//
	// Biasanya tepat satu yang terisi, tetapi TIDAK SELALU — tab Request Balai Lelang
	// mengisi keduanya (`tipe` 11 dan `tipe2` 7), karena langkah 23 activity itu memeriksa
	// keduanya sekaligus.
	//
	// Satu nilai tidak sama dengan yang terbaca di dalam activity: tab Histori Salvage
	// masuk dengan `tipe` 6, lalu langkah 15 MENULIS ULANG `Param.tipe` menjadi 2 —
	// sehingga seluruh percabangan sesudahnya menyebut 2. Yang disimpan di sini adalah
	// kode MASUKNYA, yakni yang benar-benar dikirim tabel ringkas.
	LegacyTipe  string
	LegacyTipe2 string

	// Notice adalah keterangan yang berlaku pada TAB INI SAJA, ditampilkan di atas grid.
	//
	// Ia berbeda dari PlannedDifferences, yang berlaku untuk seluruh layar. Kosong berarti
	// tidak ada keterangan khusus.
	Notice string
}

// Kolom grid, per keluarga.
//
// Urutan dan judulnya dibaca dari SEL GRID `Section/InboxSalvageASM-Section.xml`, bukan
// dari kueri. Itu penting: `GcnmSalvageData_CloseOs_SQL` mengambil 14 isian sementara grid
// terlebar hanya menggambar 12, dan dua yang tidak digambar (`QUANTITYSALVAGE`,
// `NOAKSEPTASI`) tetap ditarik karena tombol Detail Salvage memakainya.
//
// Pasangan judul dan properti, diverifikasi satu per satu dari offset sel grid-nya:
//
//	offset ~1.646.000  PIC .UserTeknis · No Klaim .CaseID · COB .Location · Tgl Kejadian .DateOfLoss
//	offset ~1.866.000  Tanggal Input .DateOfLoss · PIC .UserTeknis · No Klaim .CaseID ·
//	                   Status Lelang .TreatyName · Jenis Salvage .NewEmail · Lokasi .Location
//	offset ~2.438.000  + Nilai Pengajuan PIC .District · Keterangan PIC .AlasanKlaim ·
//	                   Nilai Request Balai Lelang .ClaimAmountAdjust · Email .Email ·
//	                   Catatan .ResponseNote · Tipe Pengajuan · Aging · Action
//	offset ~2.975.000  No Klaim .CaseID · Object Name .NewEmail · Lokasi .Location ·
//	                   User Input .UserTeknis
func outstandingColumns() []Column {
	return []Column{
		{Key: FieldPIC, Title: "PIC"},
		{Key: FieldClaimNo, Title: "No Klaim"},

		// Judulnya "COB" — class of business — dan kolomnya memang `BUSINESSNAME`. Pada
		// kelima tab keluarga berikutnya kolom yang SAMA berjudul "Lokasi". Lihat
		// Row.BusinessName.
		{Key: FieldBusinessName, Title: "COB"},
		{Key: FieldLossDate, Title: "Tgl Kejadian"},
	}
}

func claimObjectColumns() []Column {
	return []Column{
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldObjectName, Title: "Object Name"},

		// Judul "Lokasi" untuk kolom `BUSINESSNAME`. Keliru di sistem lama, dibawa apa
		// adanya — lihat Row.BusinessName.
		{Key: FieldBusinessName, Title: "Lokasi"},
		{Key: FieldPIC, Title: "User Input"},
	}
}

// salvageColumns adalah enam kolom yang dipakai grid keluarga C yang SEDERHANA —
// Histori Salvage dan Balai Lelang.
func salvageColumns() []Column {
	return []Column{
		{Key: FieldInputDate, Title: "Tanggal Input"},
		{Key: FieldPIC, Title: "PIC"},
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldAuctionStatus, Title: "Status Lelang"},
		{Key: FieldSalvageType, Title: "Jenis Salvage"},
		{Key: FieldSalvageLocation, Title: "Lokasi"},
	}
}

// checkerColumns adalah grid terlebar di layar ini — dipakai tab Checker dan Salvage
// Diterima.
//
// Kolom "Action" pada layar lama berisi tombol Approve/Reject (section
// `ButtonApproveRejectedChecker`). Ia TIDAK digambar sebagai kolom data di sini; tombolnya
// digambar layar dan aksinya ditolak beralasan — lihat PlannedDifferences.
func checkerColumns() []Column {
	return []Column{
		{Key: FieldInputDate, Title: "Tanggal Input"},
		{Key: FieldPIC, Title: "PIC"},
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldSalvageType, Title: "Jenis Salvage"},
		{Key: FieldEstimateValue, Title: "Nilai Pengajuan PIC", Numeric: true},
		{Key: FieldRemark, Title: "Keterangan PIC"},
		{Key: FieldRequestValue, Title: "Nilai Request Balai Lelang", Numeric: true},
		{Key: FieldEmail, Title: "Email"},
		{Key: FieldNote, Title: "Catatan"},
		{Key: FieldSubmissionType, Title: "Tipe Pengajuan"},
		{Key: FieldAging, Title: "Aging"},
	}
}

// rejectedColumns adalah grid Rejected Checker dan Salvage Ditolak — sembilan kolom.
//
// Ia checkerColumns TANPA "Nilai Request Balai Lelang" dan TANPA "Tipe Pengajuan". Keduanya
// memang tidak digambar di sel grid offset ~2.639.000, dan ketiadaannya masuk akal: baris
// yang sudah ditolak tidak lagi menunggu keputusan balai lelang.
func rejectedColumns() []Column {
	return []Column{
		{Key: FieldInputDate, Title: "Tanggal Input"},
		{Key: FieldPIC, Title: "PIC"},
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldSalvageType, Title: "Jenis Salvage"},
		{Key: FieldEstimateValue, Title: "Nilai Pengajuan PIC", Numeric: true},
		{Key: FieldRemark, Title: "Keterangan PIC"},
		{Key: FieldNote, Title: "Catatan"},
		{Key: FieldAging, Title: "Aging"},
	}
}

// requestColumns adalah grid Request Balai Lelang — sel grid offset ~3.237.000.
//
// Ia checkerColumns TANPA "Keterangan PIC". Perbedaan satu kolom itu dibawa apa adanya.
func requestColumns() []Column {
	return []Column{
		{Key: FieldInputDate, Title: "Tanggal Input"},
		{Key: FieldPIC, Title: "PIC"},
		{Key: FieldClaimNo, Title: "No Klaim"},
		{Key: FieldSalvageType, Title: "Jenis Salvage"},
		{Key: FieldEstimateValue, Title: "Nilai Pengajuan PIC", Numeric: true},
		{Key: FieldRequestValue, Title: "Nilai Request Balai Lelang", Numeric: true},
		{Key: FieldEmail, Title: "Email"},
		{Key: FieldNote, Title: "Catatan"},
		{Key: FieldSubmissionType, Title: "Tipe Pengajuan"},
		{Key: FieldAging, Title: "Aging"},
	}
}

// Label kotak pencarian, mengikuti teks layar lama.
//
// `pyCaption CARI NO KLAIM` ada di kedua section utama. Tab Checker mencari dua kolom
// sekaligus, dan labelnya menyebutkan keduanya supaya pengguna tahu ia boleh mengetik PIC.
const (
	searchByClaimNo = "CARI NO KLAIM"
	searchChecker   = "CARI NO KLAIM ATAU PIC"
)

// tabs adalah ketiga belas daftar, berurutan seperti tampilnya di layar lama.
//
// Urutannya mengikuti urutan sel judul pada `Section/InboxSalvageASM-Section.xml`, dan
// kelima tab keluarga B disisipkan tepat setelah Salvage Outstanding karena di pencacah
// keduanya bersebelahan (`CityID` 1 diikuti `RW` 1..5).
var tabs = []Tab{
	{
		Code: TabOutstanding,
		Name: "Salvage Outstanding",
		Description: "Klaim yang BELUM ditandai punya salvage sama sekali — `STSSALVAGE` " +
			"masih kosong. Inilah awal perjalanannya: begitu petugas menandainya " +
			"Ekonomis, TBA, Tidak Ekonomis, atau Tidak Ada, klaimnya berpindah sendiri ke " +
			"daftar yang bersangkutan.",
		Family:              FamilyClaim,
		Columns:             outstandingColumns(),
		SalvageStatusIsNull: true,
		SearchLabel:         searchByClaimNo,
		LegacyTipe:          "1",

		Notice: "Angka pada baris \"Outstanding\" di tabel ringkas TIDAK sama dengan " +
			"jumlah baris di sini, dan itu bukan kerusakan. Pencacahnya menghitung klaim " +
			"ber-`STSSALVAGE` 3 atau 5 — yakni yang sudah ditandai Ekonomis atau TBA — " +
			"sementara daftar ini justru menampilkan yang penandanya masih KOSONG. " +
			"Keduanya menghitung populasi yang berbeda di Pega pula; lihat keterangan " +
			"selisih terencana.",
	},
	{
		Code: TabEkonomis,
		Name: "Ekonomis",
		Description: "Klaim yang salvage-nya dinilai MASIH BERNILAI JUAL — `STSSALVAGE` = 3. " +
			"Dari sini pengajuannya dibuat lewat tombol Tambah.",
		Family:        FamilyClaimObject,
		Columns:       claimObjectColumns(),
		SalvageStatus: "3",
		SearchLabel:   searchByClaimNo,
		LegacyTipe2:   "1",
	},
	{
		Code: TabTBA,
		Name: "TBA",
		Description: "Klaim yang keputusan salvage-nya DITUNDA — `STSSALVAGE` = 5. " +
			"TBA singkatan dari to be advised.",
		Family:        FamilyClaimObject,
		Columns:       claimObjectColumns(),
		SalvageStatus: "5",
		SearchLabel:   searchByClaimNo,
		LegacyTipe2:   "2",
	},
	{
		Code: TabTidakEkonomis,
		Name: "Tidak Ekonomis",
		Description: "Klaim yang salvage-nya dinilai TIDAK BERNILAI JUAL — `STSSALVAGE` = 4. " +
			"Barangnya ada, tetapi biaya menjualnya melebihi hasilnya.",
		Family:        FamilyClaimObject,
		Columns:       claimObjectColumns(),
		SalvageStatus: "4",
		SearchLabel:   searchByClaimNo,
		LegacyTipe2:   "3",
	},
	{
		Code: TabTidakAdaSalvage,
		Name: "Tidak Ada Salvage",
		Description: "Klaim yang dinyatakan TIDAK punya barang sisa sama sekali — " +
			"`STSSALVAGE` = 1.",
		Family:        FamilyClaimObject,
		Columns:       claimObjectColumns(),
		SalvageStatus: "1",
		SearchLabel:   searchByClaimNo,
		LegacyTipe2:   "4",
	},
	{
		Code: TabBuyback,
		Name: "Salvage Buyback",
		Description: "Klaim yang salvage-nya DIBELI KEMBALI tertanggung. Penyaringnya " +
			"berbeda dari kelima daftar sekelasnya: ia tidak melihat `STSSALVAGE` sama " +
			"sekali, melainkan mencari klaim yang punya baris adjustment ber-nilai " +
			"salvage terisi.",
		Family:        FamilyClaimObject,
		Columns:       claimObjectColumns(),
		BuybackFilter: true,
		SearchLabel:   searchByClaimNo,
		LegacyTipe2:   "5",
	},
	{
		Code: TabHistori,
		Name: "Histori Salvage",
		Description: "SELURUH pengajuan salvage yang pernah dibuat, tanpa penyaring status " +
			"apa pun. Ia satu-satunya daftar yang menampilkan baris dari semua tahap " +
			"sekaligus.",
		Family:      FamilySalvage,
		Columns:     salvageColumns(),
		SearchLabel: searchByClaimNo,

		// 6, bukan 2. Tabel ringkas mengirim 6; langkah 15 activity pemuat daftar yang
		// menulis ulangnya menjadi 2 sebelum percabangan berikutnya membacanya.
		LegacyTipe: "6",
	},
	{
		Code: TabBalaiLelang,
		Name: "Salvage Balai Lelang",
		Description: "Pengajuan yang SUDAH dikirim ke balai lelang — `STSTRANSFER` = 1. " +
			"Kolom \"Status Lelang\" di sini menyatakan apakah barangnya sudah laku.",
		Family:         FamilySalvage,
		Columns:        salvageColumns(),
		TransferStatus: "1",
		SearchLabel:    searchByClaimNo,
		LegacyTipe:     "3",
	},
	{
		Code: TabChecker,
		Name: "Checker",
		Description: "Pengajuan yang menunggu KEPUTUSAN checker — `STSTRANSFER` = 3. " +
			"Inilah antrean tempat nilai pengajuan PIC dibandingkan dengan nilai request " +
			"balai lelang sebelum disetujui atau ditolak.",
		Family:         FamilySalvage,
		Columns:        checkerColumns(),
		TransferStatus: "3",
		SearchLabel:    searchChecker,
		SearchExact:    true,
		SearchByPIC:    true,
		LegacyTipe:     "4",

		Notice: "Pencarian di daftar ini COCOK PERSIS, bukan mengandung — mengetik " +
			"separuh nomor klaim tidak menghasilkan apa-apa. Itu perilaku layar lama apa " +
			"adanya. Satu kotak ini mencari nomor klaim DAN nama PIC sekaligus.",
	},
	{
		Code: TabRejectedChecker,
		Name: "Rejected Checker",
		Description: "Pengajuan yang DIKEMBALIKAN checker kepada PIC — `STSTRANSFER` = 4. " +
			"Ia menunggu PIC memperbaiki pengajuannya, bukan menunggu keputusan.",
		Family:         FamilySalvage,
		Columns:        rejectedColumns(),
		TransferStatus: "4",
		SearchLabel:    searchByClaimNo,
		LegacyTipe:     "5",
	},
	{
		Code: TabRequestBalai,
		Name: "Request Balai Lelang",
		Description: "Pengajuan yang balai lelangnya MEMINTA nilai berbeda — " +
			"`STSTRANSFER` = 7. Hanya baris milik Anda sendiri yang tampil di sini.",
		Family:         FamilySalvage,
		Columns:        requestColumns(),
		TransferStatus: "7",
		OwnedByCaller:  true,
		SearchLabel:    searchByClaimNo,

		// Satu-satunya tab yang mengisi KEDUA kode. Langkah 23 activity pemuat daftar
		// memeriksa `tipe2 == "7"` dan `tipe == 11` sekaligus.
		LegacyTipe:  "11",
		LegacyTipe2: "7",

		Notice: "Daftar ini HANYA menampilkan baris yang PIC-nya Anda sendiri. Ia " +
			"satu-satunya daftar di layar ini yang isinya berbeda dari petugas ke " +
			"petugas, dan itu perilaku layar lama apa adanya.",
	},
	{
		Code: TabSalvageDiterima,
		Name: "Salvage Diterima",
		Description: "Pengajuan yang DISETUJUI checker — `STSTRANSFER` = 3. Penyaringnya " +
			"sama persis dengan daftar Checker; di layar lama keduanya memang dua pintu " +
			"masuk ke baris yang sama.",
		Family:         FamilySalvage,
		Columns:        checkerColumns(),
		TransferStatus: "3",
		SearchLabel:    searchChecker,
		SearchExact:    true,
		SearchByPIC:    true,
		LegacyTipe:     "15",

		Notice: "Daftar ini berisi baris yang SAMA PERSIS dengan daftar \"Checker\" — " +
			"keduanya menyaring `STSTRANSFER` = 3. Itu bukan kekeliruan penyalinan: di " +
			"layar lama keduanya memang dua pintu masuk ke satu kumpulan baris yang sama.",
	},
	{
		Code: TabSalvageDitolak,
		Name: "Salvage Ditolak",
		Description: "Pengajuan yang DITOLAK checker — `STSTRANSFER` = 5. Berbeda dari " +
			"Rejected Checker, penolakan di sini menutup pengajuannya.",
		Family:         FamilySalvage,
		Columns:        rejectedColumns(),
		TransferStatus: "5",
		SearchLabel:    searchChecker,
		SearchExact:    true,
		SearchByPIC:    true,
		LegacyTipe:     "16",

		Notice: "Daftar ini SELALU KOSONG di Pega, dan di sini tidak. Lihat keterangan " +
			"selisih terencana: di layar lama tidak ada satu pun kueri yang dijalankan " +
			"untuknya, sehingga barisnya tidak pernah muncul meski datanya ada.",
	},
}

// Tabs mengembalikan ketiga belas tab dalam urutan tampilnya.
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

// TabForLegacy memetakan pasangan `Param.tipe` / `Param.tipe2` sistem lama ke kode tab.
//
// # Kenapa ia ada
//
// Karena tabel ringkas "Status Salvage / Jumlah" MENYIMPAN pasangan itu, bukan nama tab:
// `Activity/GCNMCountSalvage_act-Act.xml` menulis `.CityID` dan `.RW` pada setiap barisnya,
// dan mengklik baris berarti membuka daftar dengan pasangan tersebut. Tanpa pemetaan ini,
// tabel ringkasnya menjadi angka yang tidak dapat ditindaklanjuti.
//
// # Kenapa tipe2 diperiksa LEBIH DULU
//
// Karena pencacah mengisi KEDUANYA pada baris keluarga B: baris "Ekonomis" misalnya
// bernilai `CityID = 1` dan `RW = 1` sekaligus. Memeriksa `tipe` lebih dulu akan
// memetakannya ke "Salvage Outstanding" — daftar yang sama sekali berbeda, dan tanpa satu
// pun galat.
//
// Ini persis kelas kekeliruan yang sudah tercatat pada modul Inbox Komunikasi Cabang:
// dua kode yang keduanya sah tetapi hidup di ruang yang berbeda.
func TabForLegacy(tipe, tipe2 string) (Tab, bool) {
	if tipe2 != "" {
		for _, tab := range tabs {
			if tab.LegacyTipe2 == tipe2 {
				return tab, true
			}
		}
		return Tab{}, false
	}

	if tipe == "" {
		return Tab{}, false
	}
	for _, tab := range tabs {
		if tab.LegacyTipe == tipe {
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
	"Angka pada baris \"Outstanding\" di tabel ringkas TIDAK sama dengan jumlah baris " +
		"daftar Salvage Outstanding. Pencacahnya menghitung `STSSALVAGE` 3 atau 5, " +
		"sementara daftarnya menampilkan `STSSALVAGE` yang masih kosong. Keduanya " +
		"memang menghitung populasi yang berbeda di Pega: penyaring lama ditulis dua " +
		"kali pada activity yang sama — sekali sebagai 3-atau-5, lalu ditimpa menjadi " +
		"kosong — dan hanya yang kedua yang sampai ke daftar. Keputusan Work Owner " +
		"2026-09-25: direplikasi apa adanya (`P-5`), karena angkanya dibaca orang " +
		"setiap hari.",

	"Jumlah halaman pada daftar Salvage Outstanding lebih banyak daripada barisnya. " +
		"Kueri penghitung lama tidak memakai penyaring `STSSALVAGE` sama sekali, " +
		"sehingga ia menghitung SELURUH klaim pada keempat Group Panel. Keputusan Work " +
		"Owner 2026-09-25: direplikasi apa adanya (`P-5`).",

	"Baris \"Salvage Diterima\" dan \"Salvage Ditolak\" MUNCUL di tabel ringkas, dan di " +
		"Pega tidak. Activity pencacah lama menuliskan kedua baris itu ke dalam daftar " +
		"BERSARANG — ke sub-daftar milik baris pertama, bukan ke daftar utamanya — " +
		"sehingga keduanya tidak pernah tergambar. Keputusan Work Owner 2026-09-25: " +
		"diperbaiki, karena kode itu tidak pernah berjalan sebagaimana dimaksud " +
		"penulisnya dan tidak ada perilaku yang perlu dijaga.",

	"Baris \"Checker\", \"Salvage Diterima\", dan \"Salvage Ditolak\" pada tabel ringkas " +
		"tergambar untuk SEMUA pengguna. Di Pega, baris mana yang tergambar bergantung " +
		"pada DUA NAMA ORANG yang tertanam di dalam activity pencacah: kedua orang itu " +
		"melihat \"Salvage Diterima\" dan \"Salvage Ditolak\", semua orang lain melihat " +
		"\"Checker\". `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode, dan " +
		"nama orang sebagai penentu perilaku adalah tepat yang dilarangnya — ia salah satu " +
		"dari 24 Operator ID yang `F-4` hapus.",

	"Angka pada baris \"Histori Salvage\" di tabel ringkas TIDAK sama dengan jumlah baris " +
		"daftarnya. Pencacahnya menghitung `STSTRANSFER` 1 atau 6, sementara daftarnya " +
		"tidak menyaring status apa pun sehingga memuat SELURUH pengajuan. Ia selisih " +
		"ketiga sejenis pada layar ini, dan direplikasi dengan alasan yang sama (`P-5`).",

	"Grafik lingkaran di samping tabel ringkas TIDAK dibangun. Activity pencacah lama " +
		"menyusun dua daftar sekaligus — satu untuk tabel, satu untuk grafik — dan " +
		"keduanya berbeda: baris bernilai nol dibuang dari grafik tetapi tetap tergambar " +
		"di tabel. Yang dibangun adalah tabelnya, karena itulah yang memuat angka dan " +
		"tautan ke daftarnya.",

	"Daftar \"Salvage Ditolak\" BERISI baris, dan di Pega selalu kosong. Activity " +
		"pemuat daftar lama menyebutkan tipe 3, 4, 5, 2, 11, dan 15 pada langkah " +
		"pengambilan datanya — tetapi TIDAK 16 — dan langkah cadangannya pun " +
		"melewatinya. Akibatnya tabnya tergambar tanpa satu pun kueri yang pernah " +
		"dijalankan untuknya. Penyaringnya sendiri tidak ambigu: `STSTRANSFER` = 5, " +
		"ditulis dua kali di activity yang sama. Ia diperlakukan sama dengan baris " +
		"pencacah bersarang di atas — kode yang tidak pernah berjalan, sehingga tidak " +
		"ada perilaku yang perlu dijaga.",

	"Kolom \"Catatan\" pada daftar Checker, Rejected Checker, Salvage Diterima, Salvage " +
		"Ditolak, dan Request Balai Lelang SELALU KOSONG. Itu bukan data yang hilang: " +
		"tidak ada satu pun kueri maupun activity pada jalur pemuat daftar yang pernah " +
		"mengisinya di Pega. Kolomnya tetap digambar karena layar lama menggambarnya " +
		"(`D-13`).",

	"Kolom \"Aging\" berisi selisih HARI KALENDER, bukan hari kerja. Sistem lama " +
		"menghitungnya lewat kalender hari kerja yang hidup di basis data lain " +
		"(`GET_WORKING_HOURS` lewat DB Link). `D-50` menetapkan perhitungannya ditulis " +
		"ulang di Go karena ia aturan bisnis, dan `D-25` mengganti DB Link dengan " +
		"pemanggilan API — keduanya menunggu kalender libur menjadi master data " +
		"(`F-4`). Sampai itu tiba, angkanya LEBIH BESAR daripada di Pega untuk setiap " +
		"baris yang melewati akhir pekan atau hari libur.",

	"Kolom \"Object Name\" pada kelima daftar Ekonomis, TBA, Tidak Ekonomis, Tidak Ada " +
		"Salvage, dan Salvage Buyback menampilkan SATU objek saja untuk klaim yang punya " +
		"banyak objek. Kueri lama mengambil objek pertama tanpa menyatakan urutan, " +
		"sehingga objek mana yang terpilih tidak ditentukan. Perilakunya dibawa apa " +
		"adanya (`P-5`); yang ditambahkan hanyalah urutan yang pasti, supaya baris yang " +
		"sama menampilkan objek yang sama pada setiap pembukaan.",

	"Daftar \"Checker\" dan \"Salvage Diterima\" berisi baris yang SAMA PERSIS — " +
		"keduanya menyaring `STSTRANSFER` = 3. Itu keadaan di Pega, bukan kekeliruan " +
		"penyalinan.",

	"Pencarian di daftar Checker, Salvage Diterima, dan Salvage Ditolak COCOK PERSIS, " +
		"sementara di daftar lain MENGANDUNG. Mengetik separuh nomor klaim di ketiga " +
		"daftar itu tidak menghasilkan apa-apa. Itu perilaku layar lama apa adanya " +
		"(`P-5`).",

	"Daftar dipotong per halaman di basis data. Grid Pega menariknya lebih dulu lalu " +
		"menomori halamannya lewat nomor baris yang disisipkan ke dalam teks SQL; di " +
		"sini halamannya dipotong sebelum baris meninggalkan basis data, dan jumlah " +
		"seluruhnya tetap dihitung tepat. Ukuran halamannya tetap 20, sama dengan layar " +
		"lama.",

	"Nilai pencarian dikirim sebagai PARAMETER, bukan dirangkai ke dalam teks SQL. " +
		"Kelima kotak pencarian layar lama menyisipkan apa yang diketik pengguna " +
		"langsung ke dalam kueri lewat pola `{ASIS:…}` — celah yang `11-SECURITY.md` §5 " +
		"tutup tanpa perkecualian. Hasil untuk pencarian yang sah tidak berubah.",

	"Kolom \"Nilai Request Balai Lelang\" dan \"Tipe Pengajuan\" diambil dalam SATU " +
		"kueri bersama barisnya, bukan satu kueri tambahan per baris. Layar lama " +
		"menjalankan kueri agregat sekali untuk SETIAP baris pada daftar Checker, " +
		"Salvage Diterima, dan Salvage Ditolak — dua puluh satu kueri untuk satu " +
		"halaman. Angkanya sama persis; yang berubah hanya jumlah perjalanan ke basis " +
		"data.",

	"Tombol \"Export Data\" mengunduh SALINAN daftar yang sedang dilihat. Di Pega ia " +
		"mengunduh berkas yang berbeda: kueri ekspornya menggabungkan pengajuan dengan " +
		"setiap barang di dalamnya, sehingga satu pengajuan menjadi banyak baris, dan " +
		"kolomnya memuat nama barang, nama pemenang lelang, status terjual berlima " +
		"keadaan, serta nama surveyor — tidak satu pun ada di tabel. Ekspor rincian itu " +
		"belum dibangun.",

	"Tombol \"Approve\", \"Reject\", dan \"Send To BalaiLelang\" belum tersedia. " +
		"Ketiganya MENULIS, dan dua di antaranya menembak sistem di luar aplikasi ini — " +
		"balai lelang SimasBid dan penyimpanan berkas. Tombolnya tetap digambar supaya " +
		"keberadaannya terlihat, dan penekanannya menjawab alasan, bukan halaman kosong.",

	"Tombol \"Submit\" pada form Tambah menyimpan ke basis data, tetapi TIDAK " +
		"menjalankan empat langkah lain yang dijalankan layar lama: mengunggah berkas ke " +
		"penyimpanan eksternal, mengirim data ke balai lelang SimasBid, mengirim email, " +
		"dan menyisipkan salinan JSON klaim. Akibatnya pengajuan yang dibuat di sini " +
		"TIDAK sampai ke SimasBid, dan tidak ada email yang terkirim.",

	"Cacat `IDSALVAGE` yang tertimpa kosong saat pengajuan DIUBAH sudah diperbaiki. " +
		"Procedure lama menulis `IDSALVAGE = idsalvage` pada cabang pembaruan, " +
		"sementara `idsalvage` tidak pernah diisi di cabang itu — sehingga kunci " +
		"barisnya sendiri menjadi kosong. Ia butir 12 pada daftar perbaikan eksplisit " +
		"`P-5` (`D-49` #9), sudah diputuskan sebelum sesi ini.",

	"Portal Insurtech BELUM dibangun. Layar lama punya isi tersendiri untuknya " +
		"(`Section/InboxSalvageInsurtech`) yang berbeda dalam beberapa hal — antara " +
		"lain tidak punya daftar Request Balai Lelang. Keputusan Work Owner 2026-09-25: " +
		"portal ASM dibangun lebih dulu supaya perbedaan keduanya terbaca sebagai " +
		"perbedaan, bukan sebagai kerusakan.",
}
