// Package inboxprogressclaim adalah inti modul Inbox Progress Claim.
//
// # Layar apa ini
//
// Menu `MENU_ID 65` "Inbox Progress Claim" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `ProgressClaim_Harness`. Ia berada di kelompok menu `2` (Proses Produksi),
// urutan 1155.
//
// Isinya **pemantauan progres klaim yang masih berjalan**: sudah sampai posisi mana sebuah
// klaim, apa status progresnya, dan kapan ia harus ditindaklanjuti berikutnya. Per `D-79`
// ia benar-benar Inbox — barisnya pekerjaan yang menunggu ditindaklanjuti, hilang begitu
// klaimnya tutup (`stsklaim NOT IN ('1','2','3')`), dan punya tenggat berupa Next Follow Up.
// Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ProgressClaim_Harness-Harness.xml        pembungkus layar
//	Section/ProgressClaim_Section-Section.xml        lima region, kolom grid, tombol
//	Activity/GetDataProgressClaim-Act.xml            region Outstanding: 14 langkah penyaring
//	Activity/GetNextFUdata_act-Act.xml               region Next Follow Up
//	Activity/GetProgressPerPIC-Act.xml               region Progress Klaim per PIC
//	Activity/StatusProgress_act11-Act.xml            aksi baris: buka klaim
//	RDB List/DataProgressClaim-SQL.xml               kueri grid, berpaginasi
//	RDB List/GcnmCountProgressClaim_SQL-SQL.xml      pencacah total baris
//	RDB List/GetProgressPIC-SQL.xml                  rekap per PIC
//	Database/GET_POSISI_PROGRESS_PNC.fnc             posisi & status progres per klaim
//
// # Lima region di Pega, empat yang dibawa
//
// Layar lama menumpuk lima bagian dalam satu halaman, masing-masing dengan kueri sendiri:
//
//	Outstanding                 dibawa  — View OutstandingView
//	Next Follow Up              dibawa  — View NextFollowUpView
//	Progress Klaim per PIC      dibawa  — View PerPICView
//	Evaluasi Progress Klaim     dibawa  — View EvaluationView, KOSONG di Pega (lihat view.go)
//	Approval Progress Klaim     TIDAK   — di luar lingkup, keputusan Work Owner 2026-09-21
//
// Region Approval beserta tombol "Input Progress Claim" adalah satu-satunya bagian layar
// ini yang MENULIS. Keduanya sengaja ditinggalkan: tabel yang akan ditulisnya
// (`POOLDATA.GCNM_PROGRESS_CLAIM`) masih dimiliki Pega selama masa paralel, dan `P-1`
// menetapkan satu tabel hanya boleh ditulis satu sistem. Akibatnya modul ini **tidak punya
// satu pun operasi tulis**, sama seperti Inbox Admin.
//
// # Kenapa nama field di sini tidak mirip nama properti Pega
//
// Karena hampir setiap alias di layar ini menyebut hal yang bukan isinya:
//
//	Alias Pega             Kolom sebenarnya        Artinya
//	---------------------- ----------------------- --------------------------
//	CaseID                 noklaim                 Nomor Klaim
//	ClaimNo                nopolis                 Nomor Polis
//	District               T_CLAIM_PNC.QQNAME      Nama Tertanggung
//	DateForAging           tglklaim                Tanggal Registrasi
//	Country                lgb_note                Catatan LGB
//	UserTeknis             pic                     PIC Klaim
//	AnalystTransferDate    nextfu                  Next Follow Up
//	City / CityID /        posisi / sts_prg1 /     Posisi & Status Progres 1 dan 2
//	CountryID              sts_prg2
//	KomiteApproveDate      MIN(next_followup)      Follow Up terawal
//
// Perhatikan `ClaimNo` yang berisi nomor POLIS sementara `CaseID` berisi nomor KLAIM —
// tertukar persis. Membawa nama seperti itu ke sistem baru berarti mewariskan kekacauan
// yang justru menjadi alasan migrasi (`03-CURRENT-ARCHITECTURE.md` §4.2). Nama di kode ini
// memakai padanan Inggris yang benar sesuai `D-19` dan `D-80`.
//
// **Judul KOLOM adalah perkecualian yang disengaja.** Keputusan Work Owner 2026-09-21:
// judul yang dibaca pengguna tetap memakai alias Pega apa adanya. Alasannya di view.go.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxprogressclaim/          aturan modul + seam          ← paket ini
//	inboxprogressclaim/usecase/  orkestrasi: bentuk layar, isi satu region
//	inboxprogressclaim/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxprogressclaim/http/     lapisan transport modul ini  — handler, dto, rute
package inboxprogressclaim

import (
	"context"
	"strings"
	"time"
)

// ClaimRow adalah satu baris klaim pada region Outstanding dan Next Follow Up.
//
// Kedua region memakai bentuk yang sama karena kuerinya memang sama — `GetNextFUdata_act`
// hanya menambah satu saringan di atas kueri yang dipakai `GetDataProgressClaim`. Yang
// berbeda hanya kolom mana yang digambar, dan itu ditentukan View.Columns.
type ClaimRow struct {
	// ClaimNumber adalah nomor klaim — `pega_dashboardpnc.noklaim`.
	//
	// Di grid lama ia dialiaskan `CaseID`, dan sel pertama grid memanggil
	// `StatusProgress_act11` dengan nomor ini untuk membuka klaimnya.
	ClaimNumber string

	// PolicyNumber adalah nomor polis — `pega_dashboardpnc.nopolis`.
	//
	// Di grid lama ia dialiaskan `ClaimNo`. Nama itu menyebut nomor klaim padahal isinya
	// nomor polis, dan nomor klaim justru ada di alias `CaseID` — keduanya tertukar.
	PolicyNumber string

	// InsuredName adalah nama tertanggung — `POOLDATA.T_CLAIM_PNC.QQNAME`, dicari
	// berdasarkan nomor klaim. Di grid lama dialiaskan `District`.
	InsuredName string

	// RegisterDate adalah tanggal registrasi klaim — `pega_dashboardpnc.tglklaim`.
	//
	// Di grid lama dialiaskan `DateForAging`, dan ia digambar DUA KALI sebagai dua kolom
	// terpisah pada region yang sama. Lihat catatan di view.go.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan.
	RegisterDate *time.Time

	// LossDate adalah tanggal kejadian — `pega_dashboardpnc.dateofloss`. Ini satu-satunya
	// alias di layar ini yang namanya benar.
	LossDate *time.Time

	// LGBNote adalah catatan LGB — `pega_dashboardpnc.lgb_note`, di grid lama dialiaskan
	// `Country`.
	//
	// Artinya tidak terbaca dari export: tidak ada satu pun rule yang mengisinya, dan
	// kolomnya hanya dibaca. Ia dibawa apa adanya, dan namanya di sini menyebut kolom
	// sumbernya — bukan menebak artinya.
	LGBNote string

	// TechnicalPIC adalah PIC klaim — `pega_dashboardpnc.pic`, di grid lama dialiaskan
	// `UserTeknis`.
	//
	// Ia juga yang dicocokkan penyaring "Progress Claim per User" pada region per PIC.
	TechnicalPIC string

	// Positions adalah posisi klaim yang sedang berjalan, beserta status progres dan
	// tenggat tindak lanjut masing-masing.
	//
	// # Kenapa senarai, bukan satu nilai
	//
	// Karena satu klaim dapat berada di BEBERAPA posisi sekaligus. Sistem lama
	// menyatakannya dengan menggabungkan nilai memakai koma di dalam
	// `GET_POSISI_PROGRESS_PNC` — kursornya berputar atas setiap baris
	// `GCNM_PROGRESS_POSISI_PNC` yang `statusposisi`-nya `'On Progress'`.
	//
	// Penggabungan itu tidak dibawa ke sini. Yang dibawa adalah barisnya, dan
	// penggabungan menjadi satu teks terjadi di lapisan transport tempat ia memang
	// dibutuhkan — sehingga jumlah posisi tetap dapat dihitung, diurutkan, dan diuji.
	Positions []Position

	// EarliestFollowUp adalah tenggat tindak lanjut paling awal pada klaim ini —
	// `MIN(next_followup)` atas seluruh `GCNM_PROGRESS_CLAIM` miliknya.
	//
	// Di grid lama dialiaskan `KomiteApproveDate`, nama yang menyebut persetujuan komite
	// padahal tidak ada hubungannya dengan komite sama sekali. Ia hanya digambar di
	// region Outstanding.
	EarliestFollowUp *time.Time

	// ProcessDate adalah tanggal proses — `pega_dashboardpnc.tgl_proses`.
	//
	// Kueri lama MENGEMBALIKANNYA (alias `TanggalAnalystSendRCL`) tetapi tidak ada satu
	// pun sel grid yang mengikatnya, sehingga ia tidak pernah tampil. Ia tetap dibawa
	// karena ia kunci urutan bawaan grid — `ORDER BY tgl_proses ASC`.
	ProcessDate *time.Time

	// ProdKe adalah nomor perpanjangan polis — `pega_dashboardpnc.prod_ke`.
	//
	// Ia bukan kolom grid melainkan isi baris rincian yang terbuka saat baris diklik,
	// bersama nomor klaim dan nomor polis.
	ProdKe string
}

// Position adalah satu posisi tempat sebuah klaim sedang berjalan.
//
// # Dari mana bentuk ini, dan apa yang digantikannya
//
// Ia menggantikan `Database/GET_POSISI_PROGRESS_PNC.fnc` — fungsi basis data yang `D-02`
// tetapkan tidak boleh dipanggil lagi dari aplikasi. Fungsi itu dipanggil **empat kali
// untuk setiap baris grid**, masing-masing dengan kategori berbeda (`POSISI`, `sts_prg1`,
// `sts_prg2`, `nextfu`), dan setiap panggilan MENGULANG kursor yang sama persis. Satu
// pembacaan baris posisi di sini menggantikan keempatnya.
type Position struct {
	// Name adalah nama posisi — `GCNM_PROGRESS_POSISI_PNC.posisi`, misalnya REGISTER atau
	// SURVEY. Daftar lengkapnya ada di modul Master Status Progres.
	Name string

	// Status1 adalah Status Progres 1 — `GCNM_MST_PROGRESS_KLAIM.STS_PROGRESS1`.
	Status1 string

	// Status2 adalah Status Progres 2 — `GCNM_MST_PROGRESS.STS_PROGRESS2`.
	Status2 string

	// NextFollowUp adalah tenggat tindak lanjut terakhir yang dicatat pada posisi ini.
	//
	// # Satu cacat sistem lama yang TIDAK dibawa
	//
	// `GET_POSISI_PROGRESS_PNC` memformatnya dengan `to_char(..., 'DD-MM-YYYY hh:mm:ss')`.
	// Pada Oracle, `mm` di dalam bagian jam berarti **bulan**, bukan menit — menitnya
	// seharusnya `mi`. Jam yang selama ini tampil karena itu berbunyi `jam:BULAN:detik`.
	//
	// Di sini ia tanggal sungguhan, bukan teks, dan pemformatannya terjadi di satu tempat
	// di lapisan transport. Cacat itu hilang dengan sendirinya — dan karena ia mengubah
	// apa yang terbaca di layar, ia dicatat sebagai selisih terencana pada uji kesetaraan.
	NextFollowUp *time.Time
}

// PositionNames mengembalikan nama seluruh posisi, berurutan.
func (c ClaimRow) PositionNames() []string {
	return collect(c.Positions, func(p Position) string { return p.Name })
}

// Status1List mengembalikan Status Progres 1 seluruh posisi, berurutan.
func (c ClaimRow) Status1List() []string {
	return collect(c.Positions, func(p Position) string { return p.Status1 })
}

// Status2List mengembalikan Status Progres 2 seluruh posisi, berurutan.
func (c ClaimRow) Status2List() []string {
	return collect(c.Positions, func(p Position) string { return p.Status2 })
}

// collect memetakan setiap posisi menjadi satu teks.
//
// Nilai kosong TETAP disertakan, tidak dibuang. Sistem lama menggabungkan apa adanya,
// sehingga posisi yang belum punya status progres muncul sebagai celah di antara dua koma
// — dan celah itu menyatakan sesuatu: posisi keberapa yang belum diisi.
func collect(positions []Position, pick func(Position) string) []string {
	result := make([]string, 0, len(positions))
	for _, position := range positions {
		result = append(result, pick(position))
	}
	return result
}

// PICSummary adalah satu baris rekap pada region Progress Klaim per PIC.
//
// Bentuknya berbeda dari ClaimRow karena isinya memang berbeda: barisnya PETUGAS, bukan
// klaim, dan kolomnya PENCACAH, bukan atribut. Menyatukan keduanya menjadi satu bentuk
// akan menghasilkan tipe yang separuh isiannya selalu kosong.
//
// Nama alias di sistem lama pada region ini bahkan lebih jauh dari isinya daripada di
// region lain — empat dari enam kolom dinamai seperti atribut klaim padahal seluruhnya
// hasil `COUNT`:
//
//	Alias Pega   Isi sebenarnya
//	------------ ----------------------------------------------------
//	PIC          nama petugas
//	NOKLAIM      COUNT klaim yang ditangani
//	NOAKSEP      COUNT pembaruan progres, di luar yang bertanda AUTO
//	REINSURER    COUNT tindak lanjut yang jatuh tempo HARI INI
//	STSKLAIM     COUNT tindak lanjut yang TEPAT WAKTU
//	NOPOLIS      COUNT tindak lanjut yang TERLAMBAT
type PICSummary struct {
	// PIC adalah nama petugas — `pega_dashboardpnc.pic`.
	PIC string

	// ClaimCount adalah banyaknya klaim yang ditanganinya — alias `NOKLAIM`.
	ClaimCount int

	// UpdateCount adalah banyaknya pembaruan progres yang ia catat — alias `NOAKSEP`.
	//
	// Baris ber-`keterangan` berawalan `AUTO` tidak dihitung: ia dibuat sistem, bukan
	// petugas, sehingga memasukkannya akan menghitung pekerjaan yang tidak ia kerjakan.
	UpdateCount int

	// DueTodayCount adalah banyaknya tindak lanjut yang jatuh tempo hari ini —
	// alias `REINSURER`.
	DueTodayCount int

	// OnTimeCount adalah banyaknya tindak lanjut yang dipenuhi tepat waktu —
	// alias `STSKLAIM`.
	OnTimeCount int

	// LateCount adalah banyaknya tindak lanjut yang terlambat — alias `NOPOLIS`.
	LateCount int
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 15
//
// Karena itu yang ditetapkan `Activity/GetDataProgressClaim-Act.xml` sebagai `.PageSize`.
// Ia tidak dikarang, dan sengaja tidak disamakan dengan Inbox Admin yang memakai 25 —
// ukuran halaman di sistem lama memang berbeda-beda per layar.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 15
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

// ClaimPage adalah satu halaman baris klaim beserta jumlah seluruh baris yang cocok.
//
// # Kenapa dipotong di basis data, bukan di aplikasi
//
// Karena begitulah sistem lama melakukannya, dan di layar ini ia memang benar-benar
// memaginasi. `RDB List/DataProgressClaim-SQL.xml` memotong dengan `ROW_NUMBER` antara
// `Pagination.FirstRow` dan `Pagination.LastRow`, lalu
// `RDB List/GcnmCountProgressClaim_SQL-SQL.xml` menghitung totalnya dengan `COUNT(1)`
// terhadap penyaring yang sama.
//
// Ini berbeda dari Inbox Admin, yang menarik seluruh baris lebih dulu karena memang itu
// yang dilakukan Pega di sana. Perbedaannya bukan pilihan gaya — keduanya mengikuti
// layarnya masing-masing.
type ClaimPage struct {
	Items []ClaimRow
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p ClaimPage) TotalPages() int {
	return totalPages(p.Total, p.Pagination.Normalize().Size)
}

// totalPages membagi jumlah baris menjadi jumlah halaman, dibulatkan ke atas.
func totalPages(total, size int) int {
	if total <= 0 || size <= 0 {
		return 1
	}
	pages := total / size
	if total%size != 0 {
		pages++
	}
	return pages
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Region Progress Klaim per PIC menyaring `pega_dashboardpnc.pic` menurut nilai ini
	// pada langkah bernama "Progress Claim per User".
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Repo adalah seam ke data progres klaim SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Kenapa tidak ada satu pun operasi yang menulis
//
// Karena region yang menulis — Approval Progress Klaim dan tombol Input Progress Claim —
// berada di luar lingkup atas keputusan Work Owner 2026-09-21. Selama masa paralel,
// `POOLDATA.GCNM_PROGRESS_CLAIM` masih ditulis Pega, dan `P-1` menetapkan satu tabel hanya
// boleh ditulis satu sistem. Operasi yang tidak tersedia di seam ini tidak dapat dipakai
// kode yang ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// ListClaims mengembalikan SATU HALAMAN baris klaim beserta jumlah seluruh baris yang
	// cocok.
	//
	// Berbeda dengan Inbox Admin, ia menerima Pagination: pemotongan halaman benar-benar
	// terjadi di basis data, persis seperti di Pega.
	ListClaims(ctx context.Context, query ClaimQuery, page Pagination) (ClaimPage, error)

	// ListPICSummary mengembalikan seluruh baris rekap per PIC.
	//
	// Ia tidak menerima Pagination karena region ini memang tidak memaginasi di Pega:
	// `Activity/GetProgressPerPIC-Act.xml` tidak memuat satu pun langkah Pagination.
	// Barisnya satu per petugas, bukan satu per klaim, sehingga jumlahnya tidak tumbuh
	// bersama volume klaim.
	ListPICSummary(ctx context.Context, query PICQuery) ([]PICSummary, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan progres klaim
// satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// # Kenapa modul ini membutuhkannya
//
// Region Next Follow Up menyaring "tindak lanjut yang jatuh tempo hari ini atau sudah
// lewat", dan region per PIC menghitung pencacah yang bertumpu pada tanggal hari ini.
// Tanpa seam, keduanya tidak dapat diuji secara deterministik.
//
// Sistem lama menyusun tanggal itu dengan
// `@DateTime.addCalendar(@CurrentDateTime(),0,0,0,0,7,0,0)` —
// penambahan tujuh jam secara manual yang `F-5` larang tanpa perkecualian. Di sini waktu
// yang dihasilkan UTC, dan pengubahan ke WIB terjadi di satu tempat saja.
type Clock interface {
	Now() time.Time
}
