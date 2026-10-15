// Package inboxmanager adalah inti modul Inbox Manager.
//
// # Layar apa ini
//
// Menu `MENU_ID 58` "Inbox Manager" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk harness
// `UserInbox_Harness`. Ia berada di kelompok menu INBOX (`ParentID 2`), urutan 1148 — tepat
// sesudah Inbox Manager Admin (1147).
//
// Isinya **meja kerja penyelia**: tiga dashboard dan sembilan antrean persetujuan dalam satu
// layar.
//
// # Harness-nya cangkang — isinya satu lapis lebih dalam
//
// `UserInbox_Harness` hanya menyertakan `Section/InboxManager_Sec`. Section itulah yang
// menggambar ringkasan pencacah lalu menyertakan TIGA BELAS kontainer. Ketiga belasnya BUKAN
// tab bawaan Pega melainkan kontainer bersyarat:
//
//	<pyContainerVisibleWhen>FlagManager.AlasanKlaim==1</pyContainerVisibleWhen>
//	…
//	<pyContainerVisibleWhen>FlagManager.AlasanKlaim==13</pyContainerVisibleWhen>
//
// Angka 1..13 itulah KODE TAB yang sesungguhnya, dan kode tab modul ini mengambilnya apa
// adanya — bukan menomori ulang. Dengan begitu penelusuran balik ke export cukup dengan
// mencocokkan angkanya.
//
// # Sumber setiap bagian layar
//
//	Ringkasan pencacah    Activity/CountDashbroardManager       sepuluh kueri Count*
//	1  Outstanding        Section/PNCDashboardOS                GetPICDashboardOS, GetBisnisGroupDashboardOS
//	2  Produktivitas      Section/DisplayInboxProduktivitas_Sect GetSumPICDashboardProduktivitas_SQL, GetSumBusinessDashboardProduktivitas_SQL
//	3  Klaim              Section/DashboardKlaim_sec            BrowseCaseClaim, BrowseCaseClaimPerCauseOfLoss
//	4  Approval Master    Section/InboxManager_Section2          indeks sembilan antrean di bawahnya
//	5  Master Bengkel     Section/ApprovalMasterBengkelHE        POOLDATA.BENGKEL_HE
//	6  Master Panel       Section/ApprovalMasterPanelHE          POOLDATA.PANEL_HE
//	7  Nomor Rangka Beda  Section/InboxKonfirmasiHE_Section      GetDataKonfirmasiHE
//	8  Master Sparepart   Section/ApprovalMasterSparepartHE      POOLDATA.SPAREPART_HE
//	9  Kategori Sparepart Section/ApprovalMasterKategoriSparepartHE
//	10 Tipe Sparepart     Section/ApprovalMasterTipeSparepartHE
//	11 Grouping Sparepart Section/ApprovalPNCMasterGroupingSparepartHE
//	12 Payment Akseptasi  Section/Sec_PaymentAkseptasiKlaimCase1 GetDataAkseptasiLeaderKlaimNonMBU
//	13 Penolakan Klaim    Section/Sec_PenolakanKlaimChecker      BrowseStatusPenolakanKlaim2
//
// # Sumber data dashboard berpindah — ketetapan Work Owner 2026-09-28
//
// Dari SELURUH kueri layar ini, hanya EMPAT yang menyentuh skema DATAPEGA, dan keempatnya
// milik tab Outstanding: `CountOutstandingManager`, `GetPICDashboardOS`,
// `GetBisnisGroupDashboardOS`, dan `GetYearDashboardOS`. Keempatnya menggabungkan
// `PC_ASM_FW_GCNMFW_WORK` dengan `PC_ASSIGN_WORKLIST`.
//
// Keempatnya kini membaca SATU tabel datar `POOLDATA.T_CLAIMLIST_ADMIN`, sehingga modul ini
// TIDAK menyentuh skema DATAPEGA sama sekali. Sembilan kueri lain sudah POOLDATA sejak awal
// dan tidak berubah.
//
// Kesepuluh kolom yang dibutuhkan keempat kueri itu SELURUHNYA SUDAH ADA di tabel tujuan —
// dibaca langsung dari `ALL_TAB_COLUMNS` pada 2026-09-28, bukan disimpulkan dari modul lain.
// Tidak ada satu pun kolom yang perlu diminta ke DBA. Rinciannya di kepala
// repo/sqlstore/inboxmanager.sql.
//
// # Modul ini MENULIS, dan itu membedakannya dari seluruh modul inbox lain
//
// Modul inbox sebelumnya baca-saja karena tabel objek kerja dan tabel penugasan masih
// dimiliki Pega (`P-1`). Yang ditulis di sini adalah kolom PERSETUJUAN pada tabel POOLDATA —
// bukan satu pun tabel DATAPEGA. Lihat decision.go, termasuk satu jalur yang sengaja
// DITAHAN karena tidak dapat diselesaikan di dalam modul ini.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxmanager/          aturan modul + seam          ← paket ini
//	inboxmanager/usecase/  orkestrasi: daftar tab, isi satu tab, keputusan
//	inboxmanager/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxmanager/http/     lapisan transport modul ini  — handler, dto, ekspor, rute
package inboxmanager

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Ia dicatat pada setiap keputusan yang ditulis modul ini. Selama pemeriksaan peran
	// belum ada (`TKT-F3-004`), jejak itu satu-satunya kontrol yang tersisa — dan `D-59`
	// memang menjadikan jejak audit satu-satunya kontrol pengimbang.
	Login string

	// LineBusiness adalah lini bisnis petugas, dibaca dari
	// `POOLDATA.M_LOGIN_PNC.LINE_BUSINESS`.
	//
	// # Ia padanan `OperatorID.pyPosition`, dan itu terbaca dari export
	//
	// `Activity/CountDashbroardManager-Act.xml` memilih penyaring lini bisnis dashboard
	// lewat precondition `OperatorID.pyPosition=="NONMBU"` (:82436, :84988),
	// `=="BONDING"` (:95201), `=="PA"` (:101417), dan `=="TRAVEL"` (:105687).
	//
	// Nilai itu TIDAK boleh diambil dari `auth.User.Position`. Isian itu diisi
	// `auth/identity.go:68` dari HCQ `EmpResponse.Placement.PositionName` — sebuah JABATAN
	// KEPEGAWAIAN seperti "IT SPECIALIST" — dan memakainya sebagai lini bisnis adalah
	// cacat yang sudah pernah membuat modul Inbox Manager Admin kosong bagi setiap
	// pengguna (koreksi 2026-09-27).
	//
	// Padanan yang benar sudah ada dan sudah dipakai dua modul: `inboxoutstanding` dan
	// `inboxmanageradmin`, lewat kueri `line_business_for` yang sama persis.
	LineBusiness string

	// OrgUnit adalah unit organisasi pengguna — padanan `OperatorID.pyOrgUnit`.
	//
	// Satu nilai punya arti khusus, dan itu terbaca dari export: tab Approval Nomor Rangka
	// Beda tampil bila `OperatorID.pyPosition = 'NONMBU' || OperatorID.pyOrgUnit =
	// 'Development'` (`Section/InboxKonfirmasiHE_Section-Section.xml`).
	OrgUnit string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{
		Login:        strings.TrimSpace(c.Login),
		LineBusiness: strings.TrimSpace(c.LineBusiness),
		OrgUnit:      strings.TrimSpace(c.OrgUnit),
	}
}

// WithLineBusiness mengembalikan salinan identitas yang lini bisnisnya sudah terisi.
//
// Ia ada supaya lapisan transport tidak perlu tahu dari mana nilai itu dibaca: transport
// menyusun Login dan OrgUnit, usecase yang melengkapinya dari seam.
func (c Caller) WithLineBusiness(line string) Caller {
	filled := c
	filled.LineBusiness = line
	return filled
}

// Counter adalah satu baris ringkasan pencacah di kepala layar.
//
// # Kenapa ia berjenjang
//
// Karena di Pega pun begitu, dan itu terbaca dari cara activity mengisinya.
// `CountDashbroardManager` menulis empat pencacah pertama ke `TempCountDashboard.pxResults`
// langsung, lalu SEMBILAN sisanya ke `.pxResults(4).pxResults(<APPEND>)` — yakni sebagai
// ANAK dari baris keempat, "Approval Master". Gridnya pun ber-`pyRepeatDirection` TreeGrid.
//
// Karena itu Parent diisi hanya pada kesembilan anak, dan layar menggambarnya bersarang.
type Counter struct {
	// TabCode adalah kode tab yang dibuka saat pencacah ini diklik — nilai `ALASAN` yang
	// ditulis activity, yakni angka 1..13 yang sama dengan `FlagManager.AlasanKlaim`.
	TabCode string

	// Label adalah teks yang dibaca pengguna, diambil apa adanya dari `pyLabel` yang
	// ditulis activity (`D-13`).
	Label string

	// Count adalah angka pencacahnya.
	Count int

	// Parent adalah kode tab induk, kosong pada pencacah tingkat atas.
	Parent string

	// Unavailable menjelaskan kenapa pencacah ini TIDAK dapat dihitung, bila memang tidak.
	//
	// Ia teks kosong pada keadaan normal. Ia terisi — dan Count tidak bermakna — ketika
	// sumbernya sendiri sedang tidak dapat dibaca; hari ini itu terjadi pada Master
	// Sparepart, yang view-nya rusak di basis data. Lihat repo/sqlstore.
	//
	// Angka nol yang sesungguhnya berarti "tidak terbaca" adalah kebohongan kecil yang
	// membuat penyelia mengira antreannya kosong.
	Unavailable string
}

// DashboardCell adalah satu sel pada grid dashboard.
//
// # Kenapa Count dan Amount dipisah
//
// Karena Dashboard Klaim mencampur keduanya dalam satu baris: empat kolom pertamanya
// pencacah klaim, empat berikutnya jumlah UANG (`ttlos`, `TTLAKSEP`). Menyatukannya ke satu
// isian bertipe float akan membuat nilai uang menjadi bilangan pecahan biner — yang
// `09-DATABASE-STRATEGY.md` §5 dan invarian `I-12` larang.
//
// Amount karena itu teks: ia datang dari basis data sebagai angka presisi penuh dan diteruskan
// apa adanya, tanpa pernah melewati float.
type DashboardCell struct {
	// Count terisi pada kolom pencacah.
	Count int

	// Amount terisi pada kolom nilai uang, sebagai teks presisi penuh.
	Amount string

	// Text terisi pada kolom dimensi — nama PIC, nama grup bisnis, penyebab kerugian.
	Text string
}

// DashboardRow adalah satu baris grid dashboard.
//
// # Kenapa ia TERPISAH dari QueueRow
//
// Karena barisnya tidak punya kunci, tidak dapat diputuskan, dan tidak hilang setelah
// ditindaklanjuti. Menyatukannya akan membuat layar menggambar tombol keputusan pada baris
// yang tidak punya kunci untuk diputuskan.
type DashboardRow struct {
	// Cells dikunci dengan nama kolom yang diumumkan Panel.Columns.
	Cells map[string]DashboardCell
}

// Panel adalah satu grid di dalam sebuah tab dashboard.
//
// Satu tab dashboard punya DUA panel, dan angka dua itu dibaca dari section — bukan dari
// activity. `Section/PNCDashboardOS` mengikat `GetPICDashboardOS.pxResults` dan
// `GetBisnisDashboardOS.pxResults`, meski activity pemasoknya `PNCGetDashboardOSInbox_Act`
// menjalankan LIMA kueri. Tiga sisanya mengisi penyaring, bukan grid.
type Panel struct {
	// Key adalah nama panel pada kontrak API.
	Key string

	// Title adalah judul panel, mengikuti `<pyTitle>` pada section apa adanya (`D-13`).
	Title string

	// Columns adalah kolom panel ini, berurutan seperti tampilnya.
	Columns []Column

	// Rows adalah isinya.
	Rows []DashboardRow
}

// DashboardView adalah isi satu tab dashboard.
//
// # Kenapa RefreshedAt ikut, dan kenapa ia BUKAN jam aplikasi
//
// Karena layar lama menampilkannya. Kedua tab yang membaca `POOLDATA.PEGA_DASHBOARDPNC`
// memanggil `Activity/LastRefresh_act` sebagai `pyDeferLoadRetrievalActivity`, dan activity
// itu menjalankan `GetDASHBOARDPNCrefresh_sql`:
//
//	select refreshdate as "DateOfLoss" from pega_DASHBOARDPNC_refresh
//
// Nilainya karena itu datang dari BASIS DATA, bukan dari jam aplikasi. Bedanya menentukan:
// `PEGA_DASHBOARDPNC` adalah tabel cuplikan yang disegarkan berkala, sehingga angka di kedua
// dashboard itu berumur sampai penyegaran berikutnya. Menampilkan jam sekarang akan
// menyiratkan angkanya mutakhir padahal belum tentu.
//
// Ia kosong pada tab Outstanding, yang membaca `T_CLAIMLIST_ADMIN` langsung dan karena itu
// tidak punya penyegaran untuk dilaporkan.
// FilterOption adalah satu pilihan pada penyaring dashboard.
type FilterOption struct {
	Value string
	Label string
}

// FilterView adalah satu penyaring dashboard beserta pilihannya.
//
// # Kenapa pilihannya datang bersama DATA, bukan bersama keterangan layar
//
// Karena salah satunya dibaca dari master yang dapat berubah tanpa deploy — "Kategori OS"
// berasal dari `POOLDATA.GCNM_MST_PROGRESS_KLAIM`, dan di Pega pun begitu
// (`BrowseMstProgress1`). Menaruhnya di keterangan layar yang di-cache lima menit berarti
// kategori baru tidak muncul sampai cache-nya kedaluwarsa.
type FilterView struct {
	Key     string
	Label   string
	Options []FilterOption
}

type DashboardView struct {
	Panels []Panel

	// Filters adalah penyaring yang digambar di bawah kedua grid pertama, mengikuti layar
	// lama. Kosong pada tab yang tidak punya penyaring.
	Filters []FilterView

	// RefreshedAt adalah waktu cuplikan sumbernya terakhir disegarkan, apa adanya dari
	// basis data. Pointer supaya "belum pernah disegarkan" dapat dibedakan dari tanggal
	// nol.
	RefreshedAt *time.Time
}

// QueueRow adalah satu baris antrean persetujuan.
//
// # Kenapa SATU bentuk untuk sembilan antrean
//
// Karena yang berbeda di antara kesembilannya hanyalah KOLOMNYA, dan kolom itu sudah
// diumumkan Tab.Columns. Bentuk yang dipecah sembilan akan memaksa transport, penyimpanan,
// dan layar bercabang sembilan kali — padahal percabangan yang sesungguhnya hanya satu: tab
// mana yang terbuka.
//
// Ini berbeda dari modul Inbox Admin, yang kedelapan tabnya berbagi satu bentuk klaim
// sehingga struct bermedan tetap masih masuk akal. Di sini kesembilan antrean berada di atas
// tabel yang sama sekali tidak berhubungan — bengkel, panel, sparepart, nomor rangka,
// akseptasi pembayaran, dan penolakan klaim — sehingga struct bermedan tetap akan menjadi
// gabungan puluhan isian yang sebagian besarnya kosong pada setiap baris.
type QueueRow struct {
	// Key adalah kunci baris, dipakai saat keputusan dikirim balik.
	//
	// Ia OPAK bagi layar: layar mengirimkannya kembali apa adanya tanpa pernah menyusunnya
	// sendiri. Bentuknya berbeda tiap antrean — sebagian satu kolom, satu di antaranya
	// gabungan EMPAT kolom (lihat decision.go) — dan menyusunnya di layar berarti aturan
	// kuncinya hidup di dua tempat.
	Key string

	// Cells dikunci dengan nama kolom yang diumumkan Tab.Columns.
	Cells map[string]string
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// DefaultPageSize 25 hanya berlaku pada antrean yang TIDAK menyebut ukurannya sendiri.
//
// Catatan di sini sebelumnya menyatakan "tidak ada angka Pega yang dapat disalin untuk
// antreannya". Itu KELIRU: setiap section antrean menyimpan `pyPageSize` pada gridnya, dan
// angkanya berbeda-beda per tab — 20, 50, atau 15. Angkanya kini disalin ke Tab.PageSize,
// dan nilai baku ini tinggal menjadi cadangan.
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

// QueuePage adalah satu halaman antrean beserta jumlah seluruh baris yang cocok.
type QueuePage struct {
	Rows  []QueueRow
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p QueuePage) TotalPages() int {
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

// SliceQueue memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// # Kenapa dipotong di aplikasi, bukan di basis data
//
// Karena sistem lama pun begitu: kesembilan antrean dilayani RDB List yang menyerahkan
// seluruh hasilnya ke grid klipboard, dan gridnya yang memotong halaman. Tidak ada satu pun
// `OFFSET` di seluruh export (`ADR-0011`).
//
// Yang diterima secara sadar: antrean yang panjang ditarik seluruhnya ke memori aplikasi.
// Risikonya kecil di sini dan itu dapat diperiksa, bukan diandaikan — kesembilan antrean
// adalah antrean PERSETUJUAN yang isinya baris menunggu, dan saat diperiksa 2026-09-28
// seluruhnya berjumlah dua digit. Yang terbesar `MST_PENOLAKAN_KLAIM_2` dengan 11 baris.
//
// Peringatan LargeResultWarning tetap dipasang: yang kecil hari ini tidak dijamin kecil
// selamanya, dan menghapusnya sekarang berarti menghapusnya tepat sebelum ia dibutuhkan.
func SliceQueue(all []QueueRow, page Pagination) QueuePage {
	clean := page.Normalize()

	result := QueuePage{Total: len(all), Pagination: clean, Rows: []QueueRow{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Rows = all[offset:end]
	return result
}

// LargeResultWarning adalah jumlah baris yang, begitu terlampaui, dicatat sebagai peringatan
// di log.
//
// Ia TIDAK memotong hasil dan tidak mengubah perilaku apa pun. Yang dilakukannya hanya
// membuat akibat "potong di aplikasi" terlihat operator sebelum ia terlihat sebagai aplikasi
// yang kehabisan memori.
const LargeResultWarning = 5000

// Repo adalah seam ke SATU portal entitas.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
type Repo interface {
	// Counters mengembalikan kesepuluh pencacah di kepala layar.
	//
	// Pencacah yang sumbernya tidak dapat dibaca dikembalikan dengan Unavailable terisi,
	// BUKAN dengan galat: satu sumber yang rusak tidak boleh membuat sembilan pencacah lain
	// ikut hilang dari layar.
	Counters(ctx context.Context, caller Caller) ([]Counter, error)

	// Dashboard mengembalikan kedua panel sebuah tab dashboard.
	Dashboard(ctx context.Context, q Query) (DashboardView, error)

	// Queue mengembalikan SELURUH baris sebuah antrean, belum dipaginasi.
	//
	// Ia sengaja tidak menerima Pagination: pemotongan halaman terjadi di aplikasi
	// (lihat SliceQueue), dan menaruhnya di sini akan menyembunyikan bahwa seluruh baris
	// memang ditarik.
	Queue(ctx context.Context, q Query) ([]QueueRow, error)

	// Decide menuliskan keputusan atas sejumlah baris sekaligus.
	//
	// Ia mengembalikan jumlah baris yang BENAR-BENAR berubah, dan angka itu boleh lebih
	// kecil daripada jumlah kunci yang dikirim — lihat DecisionResult.
	Decide(ctx context.Context, d Decision) (int, error)
}

// LineBusinessRepo adalah seam ke lini bisnis petugas.
//
// Ia dipisah dari Repo dengan sengaja, meski keduanya membaca basis data yang sama: Repo
// membaca ANTREAN, seam ini membaca KEWENANGAN. Menggabungkannya membuat satu interface
// menjawab dua pertanyaan yang berbeda umurnya — antrean berubah setiap saat, lini bisnis
// petugas nyaris tidak pernah.
//
// Bentuknya mengikuti `inboxoutstanding.Repo.LineBusinessFor` dan
// `inboxmanageradmin.LineBusinessRepo` yang sudah ada, termasuk perlakuannya terhadap petugas
// tanpa baris.
type LineBusinessRepo interface {
	// LineBusinessFor membaca lini bisnis seorang petugas dari `M_LOGIN_PNC`.
	//
	// Petugas tanpa baris, atau yang kolomnya kosong, mengembalikan teks kosong TANPA
	// galat. Di Pega `pyPosition` yang tidak cocok satu pun sekadar tidak membuka kontainer
	// mana pun — ia tidak menggagalkan layarnya.
	LineBusinessFor(ctx context.Context, loginID string) (string, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan — dan di modul ini
// bahkan MEMUTUSKAN — pekerjaan satu badan hukum atas nama badan hukum lain (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// LineBusinessRepoSelector memilih LineBusinessRepo milik satu portal entitas.
//
// `M_LOGIN_PNC` adalah tabel PER ENTITAS: petugas yang sama dapat punya lini bisnis berbeda di
// badan hukum yang berbeda, dan membacanya dari koneksi portal yang salah berarti membuka tab
// atas dasar kewenangan entitas lain.
type LineBusinessRepoSelector func(portalAlias string) (LineBusinessRepo, error)

// Clock adalah seam ke waktu.
//
// # Apa yang dipakainya, dan apa yang TIDAK
//
// Ia dipakai untuk SATU hal saja: menentukan bulan bawaan tab Produktivitas Klaim ketika
// pemanggil tidak memilih periode. Tab itu tidak dapat berjalan tanpa periode, sehingga
// bawaannya harus ada — dan mengambilnya dari jam sistem secara langsung akan membuat
// perilakunya tidak dapat diuji.
//
// Ia TIDAK dipakai untuk waktu penyegaran dashboard. Nilai itu datang dari basis data
// (`PEGA_DASHBOARDPNC_REFRESH.REFRESHDATE`), bukan dari jam aplikasi — lihat DashboardView.
//
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}

// SortCounters mengurutkan pencacah seperti urutan penulisannya di
// `Activity/CountDashbroardManager`: tingkat atas lebih dulu menurut kode tab, lalu anaknya.
//
// Ia ada supaya urutan di layar tidak bergantung pada urutan selesainya kueri — sepuluh kueri
// pencacah dijalankan berurutan di Pega, tetapi tidak ada jaminan urutan yang sama bertahan
// bila kelak dijalankan bersamaan.
func SortCounters(counters []Counter) []Counter {
	result := make([]Counter, len(counters))
	copy(result, counters)

	sort.SliceStable(result, func(i, j int) bool {
		return tabOrder(result[i].TabCode) < tabOrder(result[j].TabCode)
	})
	return result
}
