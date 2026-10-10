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

	// Search menyaring baris menurut NOMOR KLAIM atau NOMOR POLIS. Kosong berarti seluruh
	// baris cabang itu.
	//
	// Ia TIDAK pernah mempersempit batas data — `Branch` tetap berlaku lebih dulu. Mencari
	// nomor klaim cabang lain tetap menghasilkan nol baris, dan itu disengaja (`R-20`).
	//
	// Ia juga TIDAK dipakai ekspor maupun panel ringkasan. Keduanya menjawab pertanyaan
	// "bagaimana keadaan cabang saya", bukan "di mana klaim ini" — ringkasan yang ikut
	// tersaring akan selalu menyebut satu berkas dan berhenti berarti.
	Search string
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

// PlannedDifferences adalah selisih terhadap layar Pega yang ditampilkan kepada pengguna.
//
// # Ia KOSONG sejak 2026-10-10, dan itu keputusan — bukan kelalaian
//
// Kedelapan butir yang pernah ada dibahas Work Owner satu per satu, dan seluruhnya dicabut.
// Alasan tiap pencabutan ditulis di bawah, berurut, supaya tidak ada yang menambahkannya
// kembali mengira butirnya terlewat.
//
// # Yang TIDAK hilang karena daftar ini kosong
//
// Selisihnya sendiri. `D-54` menuntut setiap selisih **diklasifikasikan pada uji kesetaraan**
// terhadap 13 butir `P-5` — ia tidak pernah menuntut selisih itu ditampilkan di layar.
// Menampilkannya adalah kebiasaan proyek ini (19 dari 77 modul), bukan kewajiban.
//
// Rincian tiap selisih beserta angkanya tetap hidup di dua tempat: komentar di berkas .sql
// dan `docs/catatan-pengembangan.md`.
//
// # Mekanismenya sengaja TIDAK dibongkar
//
// Senarai, DTO, dan komponen layarnya dibiarkan utuh. Komponen layar menggambar NOL ketika
// senarainya kosong, sehingga menambahkan satu butir kelak cukup satu baris di sini.
var PlannedDifferences = []string{
	// 0. "Kolom Reserve Claim ASM Share kini benar-benar porsi ASM." DICABUT 2026-10-10 —
	//    Work Owner menilai kolomnya sudah benar, sehingga tidak perlu diumumkan di layar.
	//
	//    Ia tetap SELISIH terhadap Pega, dan selisih itu tidak hilang karena catatannya
	//    dicabut: layar lama menampilkan `SUM(estimationvalue)` apa adanya, di sini
	//    `SUM(estimationvalue * kursvalue) * SHAREASM/100`. Pada data 2026-10-10 ia
	//    mengubah angka 74 klaim bermata uang asing dan 305 klaim ber-SHAREASM bukan 100%.
	//
	//    Yang dicabut hanya TAMPILANNYA. Pencatatannya tetap: `D-54` menuntut selisih
	//    diklasifikasikan pada uji kesetaraan `S-8`, bukan ditampilkan kepada pengguna, dan
	//    rinciannya ada di kepala berkas .sql serta di catatan pengembangan.

	// ENAM butir lain dicabut atas keputusan Work Owner 2026-10-10, dan seluruhnya dicatat
	// di sini supaya tidak ditambahkan kembali oleh orang yang mengira butirnya terlewat.
	//
	// 1. "Daftar dibagi per halaman di server." DICABUT — ia bukan selisih hasil.
	//    Layar lama PUN berhalaman: `pyPageMode = Next Previous`, `pyPageSize = 20`. Yang
	//    berbeda hanya TEMPAT pemotongannya — klipboard Pega versus basis data — dan ukuran
	//    halamannya, 20 versus 25. Work Owner menilai keduanya urusan tampilan semata, dan
	//    menetapkan ukuran 25 tetap dipakai.
	//
	// 2. "Umur klaim dihitung terhadap tanggal WIB." DICABUT — ia TIDAK BENAR. Diukur
	//    langsung ke Oracle pada 2026-10-10: `SYSTIMESTAMP = +07:00`, `DBTIMEZONE = +07:00`.
	//    Jadi `TRUNC(SYSDATE)` milik kueri lama memang sudah tanggal WIB, sama persis dengan
	//    yang dihitung `AgingDaysSince`. WIB tidak mengenal waktu musim panas, sehingga
	//    keduanya sama sepanjang tahun — tidak ada klaim yang dapat berbeda satu hari.
	//
	//    Butir itu lahir dari kehati-hatian yang tidak pernah diperiksa. Menyatakan selisih
	//    yang tidak ada sama buruknya dengan menyembunyikan selisih yang ada: pembacanya
	//    berhenti memercayai daftar ini.
	// 7. "Penanda progres mandek." DICABUT 2026-10-10 — ia tidak pernah terlihat pengguna.
	//
	//    Baris digambar merah bila umurnya melewati ambang ATAU progresnya mandek. Setelah
	//    pemisahnya diganti `ID_UPDATE DESC`, selisih terhadap Pega tinggal 2 klaim — dan
	//    keduanya SUDAH merah karena umur. Diukur: 0 baris yang warnanya berbeda, dan 0
	//    baris merah palsu (turun dari 23).
	//
	//    Pada kedua klaim itu pilihan kita justru lebih tepat: `ROWID` adalah alamat
	//    penyimpanan fisik yang dapat berpindah dan tidak punya arti bisnis, sedangkan
	//    `ID_UPDATE` adalah nomor urut pencatatan per klaim — dan "tiga catatan terakhir"
	//    memang berarti urutan pencatatan, bukan urutan penyimpanan.
	//
	//    Rincian pengukuran dan alasan teknisnya ada di kepala berkas .sql, bagian
	//    PROGRESS_STALLED.

	// 3. "Sumber klaim outstanding dipindahkan ke POOLDATA.T_CLAIMLIST_ADMIN." DICABUT —
	//    Work Owner menyatakan perpindahannya memang dikehendaki dan bukan persoalan
	//    (2026-10-10). Selisih jumlah klaim yang pernah diukur — 443 dari 953 — adalah
	//    keadaan tabel yang BELUM terisi penuh, bukan perilaku tetap; pengisiannya disusun
	//    di `docs/backfill-t-claimlist-admin.md` dan menunggu DBA (`D-63`). Angka
	//    pengukurannya tetap tersimpan di sana dan di catatan pengembangan.
	//
	// 4. "Kotak cari nomor klaim dan nomor polis adalah kemampuan baru." DICABUT — Work
	//    Owner menyatakan penambahannya memang diminta dan tidak perlu diumumkan di layar.
	//
	// 5. "Cabang ditentukan dari kode cabang rinci HCQ." DICABUT — idem; perubahan sumber
	//    cabang memang dikehendaki. Penolakan bagi pengguna tanpa kode cabang tetap dijawab
	//    dengan pesan tersendiri (`ErrBranchUnknown`), bukan dengan daftar kosong, sehingga
	//    pengguna tetap tahu sebabnya tanpa perlu daftar ini.

	// 6. "Kolom Nama Insured terisi di sini." DICABUT — Work Owner menyatakan kolom yang
	//    terisi memang yang dikehendaki (2026-10-10). Sumbernya sama dengan yang sudah
	//    dipakai berkas ekspor layar ini sejak dulu: `POOLDATA.T_GENERAL.theinsured`.
	//    Kueri daftar Pega tidak memilihnya sama sekali — nol kemunculan `theinsured`,
	//    `InsuredName`, maupun `qqname` — sehingga kolomnya digambar tetapi selalu kosong.

}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Paginasi ini KEMAMPUAN BARU, bukan pemindahan
//
// `GetDataOutstandingperCabang` dipanggil tanpa `MaxRecords` sama sekali, sehingga layar lama
// menarik seluruh baris cabang itu sekaligus ke klipboard. Dengan puluhan juta baris data
// historis (`D-10`) itu bukan pola yang dibawa (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2
// aturan 1). Selisihnya dinyatakan lewat PlannedDifferences.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari parameter
// query yang mudah salah ketik, dan menolak seluruh permintaan karena `halaman=0` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna.
func (p Pagination) Normalize() Pagination {
	clean := p
	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Items []WorkItem

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
	size := p.Pagination.Normalize().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja: menarik seluruh baris ke memori
// aplikasi lebih dulu adalah persis yang dihindari paginasi server.
//
// Keduanya tetap menghasilkan Page dengan arti yang sama, sehingga uji aturan modul yang
// berjalan di atas memori menyatakan hal yang benar tentang yang berjalan di Oracle.
func Slice(all []WorkItem, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []WorkItem{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Items = all[offset:end]
	return result
}
