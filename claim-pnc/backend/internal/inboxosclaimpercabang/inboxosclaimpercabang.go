// Package inboxosclaimpercabang adalah inti modul **Inbox OS Claim per Cabang**
// (`MENU_ID 69`).
//
// # Yang dimigrasikan
//
//	Harness/OutstandingKlaimperCabang_Harness-Harness.xml   layar rujukan
//	Section/InboxOutstandingperCabang_Section-Section.xml   kolom beserta judulnya
//	Activity/OutstandingperCabang_PreAct-Act.xml            pemuat isi layar
//	RDB List/GetDataOutstandingperCabang-SQL.xml            kueri grid
//	RDB List/GetDataOutstandingperCabangExport-SQL.xml      kueri ekspor
//	RDB List/GetNamaCabangTelepon-SQL.xml                   nama cabang pada judul
//	RDB List/GetProgress1Sama-SQL.xml                       penanda baris merah
//	Activity/ExportDataOSCabang-Act.xml                     tombol Export To Excel
//
// Butir menunya terverifikasi di `Database/m_menu_aplikasi_pnc.csv:64` —
// `MENU_ID 69 · "Inbox OS Claim per Cabang" · OutstandingKlaimperCabang_Harness`.
//
// # OS berarti Outstanding, bukan Operating System
//
// `CONTEXT.md` mendefinisikannya: **OS = Outstanding**, klaim yang sudah diakui tetapi belum
// selesai dibayar. Di layar ini "outstanding" punya arti yang lebih sempit dan sangat tegas,
// yaitu satu penyaring pada kuerinya:
//
//	w.pystatuswork NOT IN ('Resolved-Rejected', 'Resolved-Completed')
//
// Klaim yang belum tuntas maupun belum ditolak. Tidak ada kaitannya dengan status akseptasi.
//
// # Modul ini hanya MEMBACA
//
// Seluruh tabel yang disentuhnya milik sistem lama, dan selama masa paralel setiap tabel
// hanya boleh ditulis satu sistem (`P-1`). Layar lamanya pun tidak punya satu pun aksi tulis:
// dua tombolnya adalah ekspor berkas dan pembuka layar rincian.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxosclaimpercabang

import (
	"strings"
	"time"

	"claim-pnc/internal/platform/money"
	"claim-pnc/internal/platform/pagination"
)

// WorkItem adalah satu baris pada grid layar.
//
// # Nama di sini menyatakan ISI, bukan alias Pega
//
// Kueri lama mengalias kolomnya secara menyesatkan, dan dua di antaranya cukup parah untuk
// disebut satu per satu (`D-19`, `D-80`):
//
//	alias Pega          isi sebenarnya                       nama di sini
//	------------------- ------------------------------------ ---------------------
//	CABANG              c.branchname                         BranchName
//	Sources             c.sobname (source of business)       BusinessSource
//	BusinessName        grouppanel diterjemahkan ke lini      BusinessName
//	EstimationValue     SUM(estimationvalue) seluruh estimasi EstimationValue
//	TanggalTerlambat    p.tgl_input progres TERAKHIR          LastProgressAt
//	PICRekanan          c.picteknik (PIC Teknik, bukan rekanan) TechnicalPIC
//	NamaSurveyor        surveyor bertipe '2' (adjuster)      AdjusterName
//	Medicare            penanda progres mandek               ProgressStalled
//
// `TanggalTerlambat` dan `Medicare` adalah dua yang paling jauh dari isinya. Yang pertama
// bukan tanggal keterlambatan apa pun melainkan kapan progres terakhir dicatat; yang kedua
// tidak ada hubungannya dengan pengobatan — lihat ProgressStalled.
type WorkItem struct {
	// BranchName adalah nama cabang pemilik klaim — `c.branchname`, berjudul "Cabang".
	//
	// Isinya kadang nama ORANG, dan itu bukan data rusak: `POOLDATA.BRANCH` memang menamai
	// sebagian kantor pemasaran dengan nama kepalanya. Terverifikasi — dari 95 kode cabang
	// pada `T_CLAIM_PNC`, 94 resolve ke `BRANCH.ID` dengan nama yang sama persis.
	BranchName string

	// BranchCode adalah kode cabang pemilik klaim — `c.branchcode`.
	//
	// Ia TIDAK digambar sebagai kolom; kueri lama ikut memilihnya dan nilainya berguna untuk
	// penelusuran ketika daftar tampak berisi cabang yang salah.
	BranchCode string

	// BusinessSource adalah sumber bisnis — `c.sobname`, berjudul "Sumbis".
	BusinessSource string

	// BusinessName adalah lini bisnis, berjudul "COB" (Class of Business).
	//
	// Ia BUKAN kolom melainkan terjemahan `c.grouppanel` di dalam kueri:
	//
	//	002 -> PA · 003 -> Aneka · 004 -> Marine Cargo · 005 -> Travel · 006 -> Fire
	//
	// Kode yang tidak dikenali dikembalikan apa adadanya. Perhatikan `009` — yang
	// `CONTEXT.md` sebut sebagai varian Aneka — TIDAK punya cabang di sini, sehingga ia
	// tampil sebagai "009" di layar. Itu perilaku sistem lama dan dibawa apa adanya (`P-5`).
	BusinessName string

	// PolicyNumber adalah nomor polis — `c.nopolis`, berjudul "Policy No".
	PolicyNumber string

	// InsuredName adalah nama tertanggung — `t_general.theinsured`, berjudul "Nama Insured".
	//
	// # Kolom ini KOSONG di layar lama, dan di sini diisi
	//
	// `Section/InboxOutstandingperCabang_Section-Section.xml` menggambar kolom berjudul
	// "Nama Insured" yang terikat pada `.InsuredName`, tetapi **tidak satu pun rule yang
	// mengisinya**: `GetDataOutstandingperCabang` tidak mengembalikannya (nol kemunculan),
	// dan `OutstandingperCabang_PreAct` tidak menyetelnya (nol kemunculan). Kolom itu karena
	// itu selalu kosong di Pega.
	//
	// Nilainya diisi di sini karena sumbernya sudah terbukti ada dan sudah dipakai: kueri
	// ekspor layar yang SAMA membacanya dari `t_general.theinsured` pada perpanjangan polis
	// terbaru. Kolom yang selalu kosong tidak melayani siapa pun, sedangkan berkas ekspornya
	// selama ini memuat nama itu — sehingga layar dan berkasnya justru saling bertentangan.
	//
	// Ini selisih terhadap Pega, dan dinyatakan lewat PlannedDifferences (`D-54`). Bila Work
	// Owner memilih mengosongkannya demi kesetaraan, yang berubah satu ekspresi di .sql.
	//
	// Diambil dari perpanjangan polis TERBARU — `prodke` terbesar sebagai ANGKA — dengan
	// urutan yang IDENTIK dengan subkueri PolicyBusinessName pada baris ekspor. Mengubah
	// salah satunya memasangkan nama bisnis satu perpanjangan dengan nama tertanggung
	// perpanjangan lain, dan tidak ada satu pun gejala yang menandainya.
	InsuredName string

	// ClaimNumber adalah nomor klaim — `c.claimno`, berjudul "Claim No".
	//
	// Ia kunci baris dan yang dikirim tombol "Detail" sebagai parameter `Inskey`.
	ClaimNumber string

	// RegisterDate adalah tanggal registrasi klaim — `c.registerdate`, berjudul
	// "Registration Date".
	//
	// Ia juga dasar AgingDays, dan kueri menyaring `IS NOT NULL` atasnya.
	RegisterDate *time.Time

	// LossDate adalah tanggal kejadian — `c.dateofloss`, berjudul "DOL".
	LossDate *time.Time

	// RemarkRecommendation adalah catatan rekomendasi — `c.remarkrecomendation`.
	//
	// Salah ketik "recomendation" ada di nama kolomnya, bukan di sini. Ia tidak digambar
	// sebagai kolom grid tetapi ikut pada berkas ekspor.
	RemarkRecommendation string

	// EstimationValue adalah nilai yang digambar pada kolom berjudul
	// **"Reserve Claim ASM Share"**, dan judul itu TIDAK menggambarkan isinya.
	//
	// # Satu selisih yang menyangkut uang, direplikasi dengan sadar
	//
	// Kedua kueri layar ini menghitungnya BERBEDA untuk klaim yang sama:
	//
	//	grid    SUM(estimationvalue)                              tanpa kurs, tanpa share
	//	ekspor  SUM(estimationvalue * kursvalue) * SHAREASM/100   dengan keduanya
	//
	// Artinya angka di layar dan angka di berkas ekspor memang berbeda, dan yang di layar
	// bukan share ASM sama sekali. `P-5` menetapkan perilaku dipertahankan lebih dulu,
	// sehingga keduanya dibawa apa adanya — tetapi selisihnya dinyatakan kepada pengguna
	// lewat PlannedDifferences, bukan hanya dicatat di sini.
	//
	// Bila Work Owner kelak memutuskan menyamakan keduanya, yang berubah adalah satu ekspresi
	// di berkas .sql — bukan aturan modul ini.
	//
	// Tipenya money.Money, TIDAK PERNAH float64. `I-12` dan
	// `09-DATABASE-STRATEGY.md` §5 menyebutnya tidak bisa ditawar: pembulatan floating
	// point membuat perbandingan nilai uang gagal secara acak dan tidak dapat direproduksi.
	EstimationValue money.Money

	// LastProgressAt adalah kapan progres TERAKHIR dicatat — `p.tgl_input`, berjudul
	// "Tgl Update Progress Terakhir".
	//
	// Alias Pega-nya `TanggalTerlambat`, dan nama itu tidak dibawa: tidak ada satu pun
	// perhitungan keterlambatan di balik nilai ini.
	//
	// Kosong berarti klaim itu belum punya satu pun catatan progres ber-`status_progress1`.
	LastProgressAt *time.Time

	// ProgressStatus1 adalah label progres tahap 1 — `GCNM_MST_PROGRESS.sts_progress1`,
	// berjudul "Status Progress 1".
	ProgressStatus1 string

	// ProgressStatus2 adalah label progres tahap 2 — `GCNM_MST_PROGRESS.sts_progress2`,
	// berjudul "Status Progress2".
	//
	// Judulnya memang tanpa spasi sebelum angka, berbeda dari kolom di sebelahnya. Ejaan
	// layar lama dibawa apa adanya (`D-13`).
	ProgressStatus2 string

	// TechnicalPIC adalah PIC Teknik yang memegang klaim — `c.picteknik`, berjudul "PIC".
	//
	// Alias Pega-nya `PICRekanan`. Ia bukan rekanan: `CONTEXT.md` menyebut peran ini
	// **PIC Teknik / User Teknis**, penanggung jawab teknis dari sisi lini bisnis.
	TechnicalPIC string

	// ProgressNote adalah keterangan pada catatan progres terakhir — `p.keterangan`.
	//
	// Ia TIDAK digambar sebagai kolom grid, tetapi dikirim tombol "Detail" sebagai parameter
	// `notepic` dan ikut pada berkas ekspor.
	ProgressNote string

	// AdjusterName adalah nama adjuster — `T_SURVEYORLIST.surveyor_name`, berjudul
	// "Adjuster".
	//
	// Hanya baris ber-`SURVEYTYPE = '2'` yang diambil. Satu klaim dapat punya beberapa,
	// dan kueri lama mengambil `MAX(surveyor_name)` — bukan yang terbaru, melainkan yang
	// terbesar menurut urutan teks. Itu dipertahankan (`P-5`).
	AdjusterName string

	// CauseOfLoss adalah penyebab kerugian — `T_CLAIM_OBJECTCOVERAGE.causeofloss`,
	// berjudul "COL".
	//
	// Diambil dari coverage yang `createdatetime`-nya paling akhir. Satu klaim dapat punya
	// banyak coverage dengan penyebab berbeda; yang tampil hanya satu.
	CauseOfLoss string

	// Chronology adalah kronologi kejadian — `c.kronologi`.
	//
	// Tidak digambar sebagai kolom grid; ikut pada berkas ekspor.
	Chronology string

	// AgingDays adalah umur klaim dalam HARI KALENDER sejak registrasi, berjudul
	// "Aging (Hari)".
	//
	// # Kenapa ia dihitung di Go, bukan di SQL
	//
	// Kueri lama memakai `TRUNC(SYSDATE) - TRUNC(c.registerdate)`, dan `TRUNC` tidak portabel
	// (`09-DATABASE-STRATEGY.md` §4). Padanan yang dianjurkan dokumen itu —
	// `CAST(x AS DATE)` — TIDAK memangkas jam di Oracle, dan itu terukur: klaim yang
	// terdaftar kemarin siang menghasilkan `0.8758`, bukan `1`. Dipindai ke bilangan bulat,
	// ia menjadi **0**.
	//
	// Menghitungnya di Go menyelesaikan tiga hal sekaligus: portabel tanpa perkecualian,
	// benar di sekitar tengah malam karena batas harinya WIB dan bukan jam server basis data
	// (`F-5`), dan bebas dari pembulatan diam-diam.
	//
	// Urutan layar tidak ikut berpindah ke Go. Aging menurun seiring registrasi menaik,
	// sehingga `ORDER BY registerdate ASC` di basis data menghasilkan urutan yang persis sama
	// dengan `ORDER BY "AgingKlaim" DESC` milik kueri lama.
	AgingDays int

	// ProgressStalled menyatakan **progres klaim ini mandek**, dan itulah yang membuat
	// barisnya digambar MERAH.
	//
	// # Namanya di Pega `Medicare`, dan itu menyesatkan sepenuhnya
	//
	// `ASM-FW-GCNMFW-Data-ClaimData.Medicare` berlabel "Data Pengobatan" di layar registrasi,
	// dan di layar INI ia dipakai untuk hal yang sama sekali berbeda.
	// `Activity/OutstandingperCabang_PreAct-Act.xml` langkah 5 menyetelnya `"1"` bila klaim
	// itu muncul pada hasil `GetProgress1Sama` — yakni bila **tiga catatan progres
	// terakhirnya bernilai `status_progress1` sama**: progres dilaporkan tiga kali berturut
	// tanpa bergerak.
	//
	// Kolom `MEDICARE` tidak ada di `T_CLAIM_PNC` — terverifikasi ke katalog Oracle. Ia
	// memang properti clipboard yang hidup hanya selama layar terbuka, bukan data tersimpan.
	//
	// Namanya karena itu TIDAK dibawa. Nama di sini menyatakan apa yang ditanyakan.
	ProgressStalled bool
}

// AgingThreshold adalah ambang umur yang membuat satu baris digambar merah.
//
// Nilainya dari `pyInlineStyle` pada setiap sel grid
// (`Section/InboxOutstandingperCabang_Section-Section.xml`):
//
//	<pega:when test='.AgingKlaim > 180 || .Medicare == 1'> color: red; </pega:when>
//
// Perhatikan `>` , bukan `>=`: umur tepat 180 hari TIDAK merah. Batas yang salah arah adalah
// tempat paling mudah bergeser saat aturan disalin, jadi ia diuji tersendiri.
//
// # Kenapa angkanya di sini, bukan di master data
//
// `D-15` menetapkan nilai bisnis menjadi master data yang dapat diubah tanpa deploy, dan
// ambang ini memenuhi syaratnya. Ia BELUM dipindahkan ke sana karena master "Ambang Layar"
// belum ada, dan membuat satu master berisi satu baris demi satu modul akan mendahului
// keputusan bentuk masternya. Yang dikerjakan sekarang adalah menaruhnya di SATU tempat yang
// bernama, sehingga pemindahannya kelak menyentuh satu konstanta.
const AgingThreshold = 180

// NeedsAttention menyatakan baris ini digambar merah.
//
// Kedua syaratnya ber-OR, sama dengan aturan layar lama. Ia method domain, bukan perhitungan
// di frontend: aturan yang hidup di kode layar akan berbeda antara grid dan berkas ekspor
// tanpa ada yang menyadarinya.
func (w WorkItem) NeedsAttention() bool {
	return w.AgingDays > AgingThreshold || w.ProgressStalled
}

// Query adalah permintaan isi layar yang sudah tervalidasi.
//
// # Kenapa hanya ada cabang di sini
//
// Karena layar lamanya memang tidak punya satu pun penyaring. `GetDataOutstandingperCabang`
// menyaring dua hal saja — `registerdate IS NOT NULL` yang tetap, dan kode cabang pemanggil —
// dan sectionnya tidak memuat satu pun kotak cari maupun dropdown. Menambahkan penyaring di
// sini berarti menambah kemampuan yang tidak pernah ada, dan pada layar yang sedang diuji
// kesetaraannya, kemampuan tambahan adalah selisih yang harus dipertanggungjawabkan.
type Query struct {
	// Branch adalah cabang yang barisnya ditampilkan.
	//
	// Ia TIDAK PERNAH datang dari badan permintaan maupun query string. Ia diturunkan dari
	// kode cabang rinci pada sesi pemanggil lewat `Repo.BranchOf`, dan itulah satu-satunya
	// batas data layar ini.
	Branch Branch
}

// Branch adalah satu cabang, sebagaimana dikenali data klaim.
//
// Kode dan nama disimpan bersama karena keduanya datang dari baris yang sama di
// `POOLDATA.BRANCH`. Memisahkannya membuka kemungkinan judul layar menyebut cabang yang
// berbeda dari cabang barisnya.
type Branch struct {
	// Code adalah `POOLDATA.BRANCH.ID` — ruang kode yang dipakai `T_CLAIM_PNC.BRANCHCODE`.
	//
	// Ia BUKAN kode yang dikirim HCQ. Yang dikirim HCQ adalah `DetailBranchCode`, yang
	// setara `BRANCH.OLDID`; terjemahannya di `Repo.BranchOf`.
	Code string

	// Name adalah `POOLDATA.BRANCH.BRANCHNAME`, dipakai pada judul layar.
	//
	// Isinya kadang nama ORANG, dan itu bukan data rusak: sebagian kantor pemasaran memang
	// dinamai menurut kepalanya.
	Name string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk.
	Login string

	// DetailBranchCode adalah kode cabang RINCI dari profil sesi.
	//
	// Sumbernya `EmpResponse.Placement.DetailBranchCode` pada jawaban API HCQ saat masuk —
	// keputusan Work Owner 2026-09-28.
	//
	// # Ia BUKAN kode yang dipakai menyaring klaim
	//
	// Ia setara `POOLDATA.BRANCH.OLDID` (3 digit), sedangkan `T_CLAIM_PNC.BRANCHCODE`
	// memakai `BRANCH.ID` (6 digit). Memakainya langsung sebagai penyaring membuat layar
	// KOSONG untuk setiap pengguna — terukur: 94 dari 95 kode cabang klaim cocok lewat
	// `ID`, dan **nol** lewat `OLDID`. Terjemahannya di `Repo.BranchOf`.
	//
	// Ia boleh kosong: `POOLDATA.M_LOGIN_PNC` tidak punya satu pun kolom cabang, sehingga
	// pengguna non-karyawan tidak membawanya. Kosong BUKAN galat pemrograman — ia keadaan
	// yang dijawab ErrBranchUnknown beserta pesannya.
	DetailBranchCode string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{
		Login:            strings.TrimSpace(c.Login),
		DetailBranchCode: strings.TrimSpace(c.DetailBranchCode),
	}
}

// PlannedDifferences adalah selisih terhadap layar Pega yang sudah diputuskan.
//
// Ia dikirim ke layar, bukan disimpan sebagai komentar, supaya pengguna yang membandingkan
// kedua layar berdampingan memperoleh jawaban alih-alih melaporkannya sebagai kerusakan
// (`D-54`: setiap selisih wajib terpetakan ke butir `P-5`, atau menunggu persetujuan).
var PlannedDifferences = []string{
	"Kolom \"Reserve Claim ASM Share\" menampilkan jumlah estimasi apa adanya — tanpa " +
		"kurs dan tanpa porsi ASM. Berkas ekspor menghitungnya dengan keduanya, sehingga " +
		"angkanya berbeda dari yang di layar. Keduanya sama dengan sistem lama.",
	"Daftar dibagi per halaman di server. Layar lama memuat seluruh baris sekaligus, " +
		"sehingga jumlah baris yang tampak sekali layar berbeda.",
	"Umur klaim dihitung terhadap tanggal WIB, bukan terhadap jam server basis data. " +
		"Klaim yang terdaftar menjelang tengah malam karena itu dapat berbeda satu hari " +
		"dari layar lama.",
	"Penanda progres mandek memakai tiga catatan progres terakhir menurut waktunya saja. " +
		"Sistem lama memakai urutan penyimpanan fisik baris sebagai pemisah ketika dua " +
		"catatan berwaktu sama, dan urutan itu tidak dapat direproduksi di luar Oracle.",

	// Selisih yang paling besar akibatnya, dan karena itu ditulis paling tegas. Ia bukan
	// perbedaan angka melainkan perbedaan ARTI layar, dan pengguna yang tidak diberi tahu akan
	// menyimpulkan klaimnya hilang.
	"Sumber klaim outstanding dipindahkan ke POOLDATA.T_CLAIMLIST_ADMIN, dan tabel itu " +
		"memuat kumpulan klaim yang BERBEDA dari sebelumnya — bukan lebih sedikit, melainkan " +
		"berbeda. Diukur pada data hari ini: 443 dari 953 klaim outstanding yang tampil. " +
		"Dari 517 yang tidak tampil, 167 sedang di tahap Send To Analis, 136 di Estimation, " +
		"81 di Choose Surveyor, 45 tanpa penugasan terbuka, 37 di View Polis, dan 19 di " +
		"Send To PIC Teknik.",

	"Kolom \"Nama Insured\" terisi di sini. Di layar lama kolom itu digambar tetapi SELALU " +
		"kosong — tidak satu pun rule mengisinya, meski berkas ekspor layar yang sama " +
		"memuat nama tertanggung dari sumber yang sekarang dipakai kolom ini.",

	// Dinyatakan supaya petugas yang tahu cabangnya tetapi tertolak tidak menyimpulkan
	// haknya dicabut.
	"Cabang ditentukan dari kode cabang rinci yang dikirim sistem autentikasi HCQ, bukan dari " +
		"kolom telepon operator seperti sistem lama. Petugas yang di HCQ belum punya kode " +
		"cabang rinci karena itu ditolak dengan pesan, bukan diberi daftar kosong.",
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Paginasi ini KEMAMPUAN BARU, bukan pemindahan
//
// `GetDataOutstandingperCabang` dipanggil tanpa `MaxRecords` sama sekali, sehingga layar lama
// menarik seluruh baris cabang itu sekaligus ke klipboard. Dengan puluhan juta baris data
// historis (`D-10`) itu bukan pola yang dibawa (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2
// aturan 1). Selisihnya dinyatakan lewat PlannedDifferences.
//
// Page dimulai dari 1.
// Size adalah jumlah baris per halaman.
type Pagination = pagination.Request[pageSizes]

// pageSizes membawa ukuran halaman layar ini ke tipe generik pagination.
type pageSizes struct{}

func (pageSizes) Default() int { return DefaultPageSize }
func (pageSizes) Max() int     { return MaxPageSize }

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
// diminta. Layar menggambar penomoran halamannya dari sini.
type Page = pagination.Page[WorkItem, pageSizes]

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja: menarik seluruh baris ke memori
// aplikasi lebih dulu adalah persis yang dihindari paginasi server.
//
// Keduanya tetap menghasilkan Page dengan arti yang sama, sehingga uji aturan modul yang
// berjalan di atas memori menyatakan hal yang benar tentang yang berjalan di Oracle.
func Slice(all []WorkItem, page Pagination) Page { return pagination.Slice(all, page) }
