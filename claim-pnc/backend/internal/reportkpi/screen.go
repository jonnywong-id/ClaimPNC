package reportkpi

// Berkas ini memuat KETERANGAN LAYAR: tab apa saja yang ada, grid apa saja yang digambar,
// dan selisih apa saja yang sudah diketahui terhadap Pega.
//
// Ia lapisan domain, bukan lapisan transport, karena isinya fakta tentang layar lama —
// bukan bentuk JSON-nya. Yang menyusunnya menjadi jawaban adalah http/dto.go.

// Column adalah satu kolom grid.
type Column struct {
	// Key adalah nama field JSON pada baris. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul kolom. Ia mengikuti judul layar Pega apa adanya (`D-13`).
	Title string

	// OnlyOnCombinedType menyatakan kolom ini HANYA digambar pada tipe report `ALL`.
	//
	// # Kenapa ada kolom yang muncul-hilang, dan kenapa itu MENIRU Pega
	//
	// Karena Pega memakai DUA rule untuk grid Summary, dan keduanya tidak mengembalikan
	// kolom yang sama:
	//
	//	GetSummaryKPIAdjuster-SQL.xml     SELECT adjuster …                 tanpa kolom tipe
	//	GetSummaryKPIAdjusterALL-SQL.xml  SELECT adjuster, 'OUTSTANDING' …  DENGAN kolom tipe
	//	                                  UNION ALL
	//	                                  SELECT adjuster, 'FINAL' …
	//
	// Kolom tipe di sana ditulis sebagai LITERAL, bukan dibaca dari tabel — ia ada justru
	// karena `UNION ALL` menggabungkan dua kelompok, dan tanpa penanda itu kedua baris
	// milik satu adjuster tidak dapat dibedakan.
	//
	// Pada tipe tunggal kolom itu tidak berarti apa-apa: seluruh barisnya bernilai sama.
	// Menggambarnya di sana akan menambah kolom yang tidak ada di layar lama (`D-13`).
	OnlyOnCombinedType bool

	// OnlyOnGroup membatasi kolom pada satu kelompok tab KPI Admin.
	//
	// Kosong berarti berlaku di kedua kelompok. Ia dibutuhkan karena kedua kelompok
	// mengukur hal yang berbeda — NON-MBU membedakan leader dan member pada SATU tahap,
	// PA mengukur DUA tahap tanpa pembedaan orang — sehingga gridnya memang tidak sama.
	//
	// Perhatikan "Tgl Terima Dokumen" muncul DUA KALI di daftar kolom, satu per kelompok,
	// dan keduanya menunjuk kolom basis data yang BERBEDA: `TRANSFERPIC_DATE` pada NON-MBU,
	// `RECEIVEDATE` pada PA. Judulnya sama karena begitulah layar lama menamainya (`D-13`),
	// dan menyatukannya akan menampilkan tanggal yang salah pada salah satu kelompok.
	OnlyOnGroup AdminGroup
}

// Nama field JSON pada satu baris.
//
// Dikumpulkan sebagai konstanta supaya judul kolom di berkas ini, penyusun DTO di
// http/dto.go, susunan berkas ekspor di http/export.go, dan penggambar sel di layar tidak
// dapat berselisih tanpa ketahuan — keempatnya merujuk nama yang sama.
const (
	FieldAdjusterName = "adjuster"
	FieldCaseID       = "no_case"
	FieldType         = "tipe"
	FieldScoredOn     = "tanggal"
)

// Nama field JSON pada satu baris rincian tab KPI Admin.
const (
	FieldClaimNumber    = "no_klaim"
	FieldPolicyNumber   = "no_polis"
	FieldBusinessName   = "business"
	FieldRegisterDate   = "tgl_regist_klaim"
	FieldTransferDate   = "tgl_terima_dokumen"
	FieldTeamFlag       = "flag"
	FieldRegisterAging  = "aging_regist_klaim"
	FieldReceiveDate    = "tgl_terima_dokumen_pa"
	FieldLODReceiveDate = "tgl_terima_lod"
	FieldAcceptanceDate = "tgl_pembayaran"
	FieldPaymentAging   = "aging_pembayaran_klaim"
	FieldRegisterSLA    = "status_sla_regist"
	FieldPaymentSLA     = "status_sla_pembayaran"
)

// Kode kedua grid pada tab KPI Adjuster.
const (
	GridSummary = "ringkasan"
	GridDetail  = "rincian"
)

// Nama field JSON pada satu baris penilaian tab KPI PIC Teknik.
const (
	FieldPICName     = "pic"
	FieldPICMetric   = "kpi"
	FieldPICTotal    = "total"
	FieldPICAchieved = "tercapai"
	FieldPICPercent  = "persentase"
	FieldPICValue    = "nilai"
)

// Kode grid tab KPI PIC Teknik.
//
// Hanya SATU grid, dan itu berbeda dari kedua tab lain. Layar lama menggambar satu tabel per
// PIC, empat baris masing-masing; di sini bentuknya sama tetapi dirakit sebagai satu daftar
// kartu, satu kartu per PIC, ditutup kartu rekapitulasi Leader.
const GridPICScorecard = "kartu-skor-pic"

// Kode kedua grid pada tab KPI Admin.
//
// `GridScorecard` bukan tabel: ia satu kartu berisi metrik berurutan, karena kuerinya
// mengembalikan TEPAT SATU baris dengan belasan kolom.
const (
	GridScorecard   = "kartu-skor"
	GridAdminDetail = "rincian-klaim"
)

// Kode ketiga tab layar.
//
// Nilainya dipakai kontrak API dan disimpan sebagai konstanta supaya tab yang BELUM
// dibangun tetap punya nama yang stabil — layar menyebutnya, dan penyebutan itulah yang
// membuat kemajuan migrasi terbaca pengguna alih-alih tampak sebagai menu yang hilang.
const (
	TabAdjuster   = "adjuster"
	TabPICTeknik  = "pic-teknik"
	TabAdmin      = "admin"
	DefaultTabKPI = TabAdjuster
)

// Grid adalah satu grid pada sebuah tab, beserta kolomnya.
type Grid struct {
	// Code adalah kunci grid pada kontrak API.
	Code string

	// Title adalah judul grid, mengikuti `pyTitle` layout group layar lama.
	Title string

	// Columns adalah kolom TETAP grid ini — yang bukan komponen KPI.
	//
	// Kesembilan kolom komponen TIDAK ada di sini, dan itu disengaja: keduanya memakai
	// kesembilan komponen yang SAMA (lihat component.go), dan menuliskannya dua kali di
	// sini berarti dua daftar yang harus dijaga kesamaannya. Layar merakit kolom tetap ini
	// lebih dulu, lalu menambahkan kesembilan komponen di belakangnya.
	Columns []Column
}

// ColumnsFor mengembalikan kolom tetap yang benar-benar digambar pada sebuah tipe report.
//
// Ia dipakai penyusun berkas ekspor supaya isi berkas dan isi layar TIDAK dapat berbeda:
// keduanya bertanya kepada fungsi yang sama. Layar memakai penanda `OnlyOnCombinedType`
// yang dikirim bersama metadata, karena metadata tidak tahu tipe report yang sedang
// dipilih — ia keterangan layar, bukan jawaban permintaan.
func (g Grid) ColumnsFor(reportType ReportType) []Column {
	result := make([]Column, 0, len(g.Columns))
	for _, column := range g.Columns {
		if column.OnlyOnCombinedType && reportType != TypeAll {
			continue
		}
		result = append(result, column)
	}
	return result
}

// ColumnsForGroup mengembalikan kolom yang berlaku pada satu kelompok tab KPI Admin.
//
// Pasangan ColumnsFor untuk tab yang lain. Keduanya dipisah — bukan disatukan menjadi satu
// fungsi bersyarat dua — karena keduanya menjawab pertanyaan yang berbeda: yang satu
// "tipe report mana", yang lain "kelompok mana". Menyatukannya akan memaksa setiap
// pemanggil menyediakan keduanya, termasuk yang tidak punya salah satunya.
func (g Grid) ColumnsForGroup(group AdminGroup) []Column {
	result := make([]Column, 0, len(g.Columns))
	for _, column := range g.Columns {
		if column.OnlyOnGroup != "" && column.OnlyOnGroup != group {
			continue
		}
		result = append(result, column)
	}
	return result
}

// AdminTab mengembalikan tab KPI Admin.
func AdminTab() Tab {
	tab, _ := FindTab(TabAdmin)
	return tab
}

// AdminGridFor mengambil keterangan satu grid pada tab KPI Admin.
func AdminGridFor(code string) Grid {
	for _, grid := range AdminTab().Grids {
		if grid.Code == code {
			return grid
		}
	}
	return Grid{}
}

// Tab adalah satu tab layar.
type Tab struct {
	// Code adalah kunci tab pada kontrak API.
	Code string

	// Title adalah judul tab, mengikuti `pyTitle` layar lama apa adanya.
	Title string

	// Grids adalah grid yang digambar tab ini. Kosong bila tabnya belum dibangun.
	Grids []Grid

	// Blocked menyatakan tabnya BELUM dibangun.
	Blocked bool

	// BlockedReason menyebut apa yang menghalanginya, dengan nama artefaknya.
	//
	// Ia wajib terisi bila Blocked — "belum tersedia" tanpa sebab hanya memindahkan
	// pekerjaan menebak kepada pembacanya, dan yang membacanya adalah penguji yang sedang
	// memutuskan apakah ini cacat atau bukan.
	BlockedReason string
}

// tabs adalah ketiga tab layar, dalam urutan layout group section lama.
var tabs = []Tab{
	{
		Code:  TabPICTeknik,
		Title: "KPI PIC Teknik",
		// Urutannya mengikuti offset `pyTitle` pada
		// `Section/ReportKPI_Section-Section.xml`: KPI PIC Teknik (:170000),
		// KPI Adjuster (:399910), KPI Admin (:869593).
		Grids: []Grid{
			{
				Code:  GridPICScorecard,
				Title: "Data KPI PIC Teknik",
				Columns: []Column{
					{Key: FieldPICName, Title: "PIC"},
					{Key: FieldPICMetric, Title: "KPI"},
					{Key: FieldPICTotal, Title: "Total"},
					{Key: FieldPICAchieved, Title: "Tercapai"},
					{Key: FieldPICPercent, Title: "Persentase"},
					{Key: FieldPICValue, Title: "Nilai"},
				},
			},
		},
	},
	{
		Code:  TabAdjuster,
		Title: "KPI Adjuster",
		Grids: []Grid{
			{
				Code:  GridSummary,
				Title: "Summary KPI Adjuster",
				Columns: []Column{
					{Key: FieldAdjusterName, Title: "ADJUSTER"},
					// Hanya pada tipe ALL — lihat Column.OnlyOnCombinedType.
					//
					// Judulnya "TIPE", bukan alias Pega-nya. Di sana kolom literal itu
					// dialiaskan `StatusWork`, dan alias itu menyesatkan: isinya bukan
					// status kerja klaim melainkan kelompok mana barisnya berasal
					// (`D-19`).
					{Key: FieldType, Title: "TIPE", OnlyOnCombinedType: true},
				},
			},
			{
				Code:  GridDetail,
				Title: "Detail KPI Adjuster",
				Columns: []Column{
					{Key: FieldAdjusterName, Title: "ADJUSTER"},
					{Key: FieldCaseID, Title: "NO CASE"},
					{Key: FieldType, Title: "TIPE"},
					{Key: FieldScoredOn, Title: "TANGGAL"},
				},
			},
		},
	},
	{
		Code:  TabAdmin,
		Title: "KPI Admin",
		Grids: []Grid{
			{
				Code:  GridScorecard,
				Title: "Data KPI",
				// Kartu skor TIDAK punya kolom tetap: ia bukan tabel berbaris-baris
				// melainkan satu kartu berisi metrik berurutan, dan metriknya berbeda
				// antar kelompok. Yang menyusunnya adalah BuildScorecard.
			},
			{
				Code:  GridAdminDetail,
				Title: "Rincian Klaim",
				Columns: []Column{
					{Key: FieldClaimNumber, Title: "No Klaim"},
					{Key: FieldPolicyNumber, Title: "No Polis"},
					{Key: FieldBusinessName, Title: "BUSINESS", OnlyOnGroup: AdminGroupNonMBU},
					{Key: FieldRegisterDate, Title: "Tgl Regist Klaim"},
					{Key: FieldTransferDate, Title: "Tgl Terima Dokumen", OnlyOnGroup: AdminGroupNonMBU},
					{Key: FieldTeamFlag, Title: "Flag", OnlyOnGroup: AdminGroupNonMBU},
					{Key: FieldRegisterAging, Title: "Aging Regist Klaim"},

					{Key: FieldReceiveDate, Title: "Tgl Terima Dokumen", OnlyOnGroup: AdminGroupPA},
					{Key: FieldRegisterSLA, Title: "Status SLA Regist Klaim", OnlyOnGroup: AdminGroupPA},
					{Key: FieldLODReceiveDate, Title: "Tgl Terima LOD", OnlyOnGroup: AdminGroupPA},
					{Key: FieldAcceptanceDate, Title: "Tgl Pembayaran", OnlyOnGroup: AdminGroupPA},
					{Key: FieldPaymentAging, Title: "Aging Pembayaran klaim", OnlyOnGroup: AdminGroupPA},
					{Key: FieldPaymentSLA, Title: "Status SLA Pembayaran Klaim", OnlyOnGroup: AdminGroupPA},
				},
			},
		},
	},
}

// Tabs mengembalikan salinan ketiga tab.
func Tabs() []Tab {
	result := make([]Tab, len(tabs))
	copy(result, tabs)
	return result
}

// FindTab mencari tab menurut kodenya.
func FindTab(code string) (Tab, bool) {
	for _, t := range tabs {
		if t.Code == code {
			return t, true
		}
	}
	return Tab{}, false
}

// AdjusterTab mengembalikan tab yang dibangun di tahap ini.
//
// Ia fungsi tersendiri, bukan indeks ke dalam tabs, supaya pemanggil tidak bergantung pada
// urutan — urutan itu mengikuti layar lama dan dapat berubah bila layar lama dibaca ulang.
func AdjusterTab() Tab {
	tab, _ := FindTab(TabAdjuster)
	return tab
}

// PlannedDifferences adalah selisih terhadap Pega yang SUDAH diputuskan, bukan ditemukan.
//
// Ia dikirim ke layar dan ditampilkan di bawah tabel. Alasannya sederhana: yang tidak
// dinyatakan di muka akan dilaporkan sebagai kerusakan, dan menelusurinya kembali jauh
// lebih mahal daripada menuliskannya sekarang (`D-54` menuntut setiap selisih
// diklasifikasikan, bukan sekadar muncul).
var PlannedDifferences = []string{
	"Layar ini MEMBACA saja. Di Pega, menekan Cari MENGHITUNG ULANG penilaian tiap kasus " +
		"survei lalu MENYIMPANNYA ke " + SourceTable + " lewat INSERT_KPIADJUSTER. Di " +
		"sini tabel itu hanya dibaca — selama masa paralel tepat satu sistem yang boleh " +
		"menulis sebuah tabel (P-1), dan penulisnya masih Pega.",

	"Akibat langsung dari butir di atas: kasus survei yang BELUM pernah dihitung Pega " +
		"belum muncul di sini. Yang tampil adalah penilaian yang sudah tersimpan, bukan " +
		"penilaian yang dihitung saat layar dibuka.",

	"Isi dropdown Adjuster diambil dari kolom ADJUSTER pada " + SourceTable + ", bukan " +
		"dari master surveyor. Rule pengisinya di Pega (BrowseAdjsuterExternal) tidak ada " +
		"di export (R-16). Akibatnya adjuster yang terdaftar di master tetapi belum punya " +
		"satu pun penilaian TIDAK muncul di dropdown — dan sebagai gantinya, setiap " +
		"pilihan yang muncul pasti menghasilkan baris.",

	"Periode WAJIB diisi untuk ketiga tipe report. Di Pega kedua tanggal disisipkan " +
		"langsung ke teks SQL, sehingga mengosongkannya menghasilkan galat basis data " +
		"mentah — bukan hasil yang lebih luas. Yang berubah adalah cara galatnya " +
		"disampaikan; rentang yang sah menghasilkan baris yang sama.",

	"Pada tipe report ALL, satu adjuster tampil DUA baris di grid Summary — satu " +
		"OUTSTANDING, satu FINAL. Itu bukan baris ganda: kueri lamanya memang " +
		"menggabungkan dua kelompok dengan UNION ALL, dan kolom TIPE yang membedakan " +
		"keduanya ada di kueri itu pula. Pada tipe tunggal kolom TIPE TIDAK digambar, " +
		"persis seperti di Pega — di sana kueri tipe tunggal memang tidak " +
		"mengembalikannya.",

	"Kolom NILAI adalah kolom TERSENDIRI di basis data, bukan jumlah kedelapan komponen " +
		"di sebelahnya. Ia dibaca apa adanya dan tidak pernah dihitung ulang di sini.",

	"Nilai komponen yang kosong ditampilkan sebagai tanda hubung, bukan sebagai 0. " +
		"Kolomnya bertipe teks di basis data, dan komponen yang belum dinilai berbeda " +
		"artinya dari komponen yang dinilai nol.",

	"Grid Detail dipaginasi di server. Layar lama memuat seluruh baris sekaligus ke " +
		"clipboard Pega; laporan lamanya pun terpotong di 500 baris (ADR-0011). Jumlah " +
		"baris yang terlihat karena itu dapat berbeda.",
}

// PICTeknikPlannedDifferences adalah selisih terencana khusus tab KPI PIC Teknik.
//
// Tiga butir pertama BUKAN selisih yang saya buat — ketiganya adalah perilaku sistem lama
// yang direplikasi apa adanya (`P-5`) dan tampak seperti kerusakan bila dibaca tanpa
// keterangan.
//
// # Ketiganya SUDAH diputuskan, bukan menunggu jawaban
//
// Ketiganya diangkat ke Work Owner pada 2026-09-25 beserta gejala yang dapat diperiksa
// langsung di Pega, dan jawabannya: **"kalau semuanya sudah sesuai Pega, biarkan saja apa
// adanya"**.
//
// Jadi ini bukan daftar tunggu. Ketiganya BERLAKU, dan yang mengubahnya kelak memerlukan
// keputusan baru — bukan sekadar merapikan kode. Uji di `picteknik_test.go` menjaga
// ketiganya tetap begitu.
var PICTeknikPlannedDifferences = []string{
	"Ketiga butir di bawah sudah DIPUTUSKAN tetap mengikuti Pega (2026-09-25). Ia " +
		"dinyatakan di sini supaya angkanya tidak dilaporkan sebagai kerusakan — bukan " +
		"karena akan diubah.",

	"NILAI BERLAWANAN ARAH PADA DUA KOMPONEN — direplikasi dari Pega. Tangga nilai " +
		"UPDATE PROGRESS KLAIM dan SLA KLAIM pada M_KPI_PNC tersusun MENURUN (0–20 bernilai " +
		"5; 35–100 bernilai 1), yang berarti tabelnya disusun untuk persentase TERLAMBAT. " +
		"Tetapi activity lama mengirim persentase TEPAT WAKTU. Akibatnya PIC yang 95% tepat " +
		"waktu bernilai 1, sedangkan yang 10% tepat waktu bernilai 5. Dua komponen lain " +
		"(ANALISA, AKSEPTASI) tangganya MENAIK dan tidak terdampak.",

	"TAMBALAN 100% pada komponen Update Status Progress — direplikasi dari Pega. Ketika " +
		"persentasenya mencapai 100, activity lama memaksa nilainya menjadi 5 dan " +
		"MENULISKAN persentasenya sebagai 99. Akibatnya dua PIC dapat sama-sama tampil " +
		"\"99%\" dengan nilai 5 dan 1. Tambalan itu sendiri adalah petunjuk terkuat bahwa " +
		"butir pertama di atas memang cacat.",

	"BARIS SLA LEADER SELALU 100% — direplikasi dari Pega. Agregasinya menghitung " +
		"pembagian sebuah angka dengan DIRINYA SENDIRI, sehingga hasilnya selalu 100 berapa " +
		"pun isinya. Digabung dengan tangga SLA yang menurun, baris itu praktis selalu " +
		"bernilai 1.",

	"TITIK BATAS TANGGA menjadi dapat ditentukan. Pita bertetangga bertindih di batasnya " +
		"(nilai 20 memenuhi 0–20 dan 20–25 sekaligus), dan kueri lama tidak punya ORDER BY " +
		"sehingga jawabannya bergantung urutan baris yang kebetulan dikembalikan Oracle. Di " +
		"sini pita dibaca terurut menurut ID dan yang pertama cocok yang dipakai — salah " +
		"satu dari dua jawaban yang sama-sama mungkin di Pega menjadi satu-satunya jawaban " +
		"di sini.",

	"DUA LABEL YANG BERSELISIH untuk komponen yang sama dibawa apa adanya: baris PIC " +
		"berbunyi \"max terlambat 25%\" (dari GetReportKPI_Progress) sedangkan baris Leader " +
		"di Pega berbunyi \"max terlambat 20%\" (dari PNCReportKPI_act). Di sini keduanya " +
		"memakai teks baris PIC, karena itulah yang menilai orangnya.",

	"AGREGASI PER KELOMPOK TIM TIDAK DIBANGUN. Sistem lama masih punya satu lapis " +
		"rekapitulasi lagi di atas baris Leader, dengan pembagi 2, 4, dan 5 pada tempat yang " +
		"berbeda. Syarat kapan masing-masing pembagi berlaku tidak dapat ditentukan dari " +
		"export, sehingga lapis itu ditinggalkan alih-alih ditebak.",

	"DUA PETUGAS DIKECUALIKAN dari penilaian SLA — direplikasi dari Pega, yang menyaringnya " +
		"dengan mencocokkan NAMA ORANG di dalam rule (D-15). Keduanya tetap muncul pada " +
		"ketiga komponen lain.",

	"Perhitungan hari kerja ditulis ulang di Go (D-50), bukan dipanggil lewat DB link. " +
		"Hari liburnya tetap dibaca dari sumber yang sama dan akhir pekan dikecualikan di " +
		"sisi kueri persis seperti CheckHoliday_SQL, supaya libur yang jatuh pada Sabtu atau " +
		"Minggu tidak terpotong dua kali.",

	"Ambang hari diambil dari kolom DAY pada M_KPI_PNC — 10 hari untuk Analisa, 1 hari " +
		"untuk Akseptasi, 390 hari untuk SLA Leader dan 399 untuk SLA Member. Satu " +
		"pengecualian: jalur Akseptasi baris LEADER di Pega menuliskan angka 1 langsung di " +
		"dalam rule alih-alih membacanya dari tabel; keduanya kebetulan bernilai sama hari " +
		"ini, dan yang dipakai di sini adalah yang dari tabel.",
}

// AdminPlannedDifferences adalah selisih terencana khusus tab KPI Admin.
//
// Ia TERPISAH dari PlannedDifferences di atas, dan pemisahannya disengaja: keduanya
// menyangkut layar yang berbeda, dan menggabungkannya akan membuat pengguna tab Adjuster
// membaca delapan butir yang tidak berlaku baginya — lalu berhenti membaca seluruhnya.
//
// Tiga butir pertama adalah KEANEHAN SISTEM LAMA yang direplikasi atas ketetapan Work Owner
// 2026-09-24 ("seperti aplikasi PEGA saja"). Ketiganya dapat mengubah angka, dan karena itu
// disebut lebih dulu.
var AdminPlannedDifferences = []string{
	"Kartu skor NON-MBU menghitung Group Panel 009, grid rinciannya TIDAK. Akibatnya " +
		"jumlah baris rincian tidak selalu sama dengan TOTAL KLAIM pada kartu skornya. " +
		"Itu perbedaan penyaring yang memang ada di kedua kueri Pega, direplikasi apa " +
		"adanya.",

	"Pada kartu skor PA, angka TOTAL KLAIM BAYAR terkunci pada rentang 1 Januari – " +
		"10 November 2023 dan TIDAK ikut berubah ketika periode diganti. Rentang itu " +
		"ditulis sebagai nilai tetap di dalam kueri Pega dan tampak seperti sisa uji coba " +
		"yang tertinggal. Direplikasi; menunggu keputusan Work Owner.",

	"Enam Operator ID yang menentukan klaim siapa yang dihitung, nama dan NIK " +
		"koordinator, bobot 0,45 dan 0,40, serta ambang nilainya seluruhnya masih " +
		"tertulis di dalam kode — persis seperti di Pega. D-15 menuntut semuanya menjadi " +
		"master data; masternya belum ada.",

	"Nama koordinator NON-MBU yang ditampilkan diambil dari activity Pega, BUKAN dari " +
		"teks kuerinya. Keduanya menyebut nama yang berbeda, dan activity berjalan " +
		"belakangan sehingga itulah yang selama ini dilihat pengguna. Mana yang benar " +
		"menurut bisnis belum dipastikan.",

	"Kartu skor PA tidak punya baris kesimpulan tercapai/tidak tercapai. Kuerinya memang " +
		"tidak menghitungnya — hanya kartu NON-MBU yang punya.",

	"Periode kosong ditolak dengan menyebut isian mana yang kurang. Layar lama menolaknya " +
		"pula, tetapi dengan satu pesan yang tidak membedakan keduanya.",

	"Bila tidak ada satu pun klaim pada periode yang dipilih, metrik yang pembaginya nol " +
		"ditampilkan kosong — bukan 0, dan bukan galat. Di Pega keadaan itu menghasilkan " +
		"kegagalan basis data ORA-01476 yang sampai ke pengguna apa adanya.",

	"Grid rincian dipaginasi di server. Layar lama memotongnya 25 baris per halaman di " +
		"dalam kueri; ukuran halaman di sini ditentukan layar.",
}
