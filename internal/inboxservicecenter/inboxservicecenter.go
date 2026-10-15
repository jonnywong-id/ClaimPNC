// Package inboxservicecenter adalah inti modul Inbox Service Center.
//
// # Layar apa ini
//
// Menu `MENU_ID 46` "Inbox Service Center" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `InboxServiceCenter`. Ia berada di kelompok menu `2` (Proses Produksi), urutan
// 1136.
//
// Isinya **daftar klaim portal rekanan** — klaim perbaikan perangkat (gawai) yang masuk dari
// portal mitra, dipecah menjadi empat tab menurut status persetujuannya. Per `D-79` ia
// benar-benar Inbox, bukan layar data acuan: barisnya pekerjaan, ia berpindah tab begitu
// diputuskan, dan "hanya milik saya" adalah aturan kewenangan — bukan sekadar penyaring.
// Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// # Kenapa isinya IMEI, Brand, dan Model — bukan objek pertanggungan biasa
//
// Karena lini yang dilayaninya memang perbaikan perangkat. Tabel intinya,
// `POOLDATA.T_KLAIM_PORTAL_REKANAN`, memuat kolom `IMEI`, `BRAND`, `MODEL`, `COLOUR`,
// `SIMCARD`, `BATTERY`, dan `LCD_TEXT`. Ini bukan salah baca: rule tampilnya,
// `When/IsServiceCenterPNC-When.xml`, ikut menyalakan menu ini pada host **entitas
// Insurtech**, dan itulah lini yang menjual asuransi gawai.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxServiceCenter-Harness.xml             pembungkus layar; panel CENTER
//	Section/BrowseServiceCenter-Section.xml            wadah TABBED, 4 tab + judulnya
//	Section/ClaimServiceCenterOSnotTransfer-Section.xml tab "Registrasi SC"
//	Section/BrowseServiceCenterWaitingApproval-Section.xml tab "Waiting Approval"
//	Section/BrowseServiceCenterApprove-Section.xml     tab "Approved"
//	Section/BrowseServiceCenterReject-Section.xml      tab "Rejected"
//	Section/ButtonPagingInbox-Section.xml              First/Previous/Next/Last
//	Activity/DataServiceCenter-Act.xml                 61 langkah: penyaring per tab + paginasi
//	Activity/CariDataSCUntukCaseBaseOnStatus-Act.xml   pencarian per ID / awalan Repair ID
//	RDB List/CountDataServiceCenter-SQL.xml            jumlah baris yang cocok
//	RDB List/ExportDataServiceCenter-SQL.xml           kolom yang tersedia di tabel yang sama
//	RDB List/GetDataServiceCenter_Update-SQL.xml       peta kolom -> alias, lengkap
//	RDB List/GetListDataServiceCenter-SQL.xml          riwayat catatan progres satu baris
//	When/IsServiceCenterPNC-When.xml                   kapan menunya tampil
//
// # Satu rule yang TIDAK ada di export, dan bagaimana ia direkonstruksi
//
// Activity `DataServiceCenter` memanggil RDB rule `GetDataServiceCenter` pada kelas
// `ASM-FW-GCNMFW-Data-ClaimData`. Yang ikut terekspor adalah rule BERNAMA SAMA pada kelas
// `ASM-FW-GCNMFW-Int-T_GENERAL` — kueri lain, yang membaca `service_log_nonmbu`. Kueri grid
// yang sebenarnya karena itu tidak ada di export (`R-16`).
//
// Ia TIDAK dikarang, melainkan disusun ulang dari tiga rule sekelas yang ada:
//
//	CountDataServiceCenter        tabelnya, dan KETIGA penyaringnya persis
//	ExportDataServiceCenter       kolom yang dibaca dari tabel yang sama
//	GetDataServiceCenter_Update   peta kolom -> alias untuk seluruh kolom
//
// Keenam kolom yang digambar grid ada di kedua kueri terakhir, sehingga rekonstruksinya
// menyentuh bentuk — bukan isi. Yang tetap menjadi dugaan hanyalah urutan `ORDER BY`-nya;
// lihat catatan di inboxservicecenter.sql.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxservicecenter/          aturan modul + seam          ← paket ini
//	inboxservicecenter/usecase/  orkestrasi: daftar tab, isi satu tab
//	inboxservicecenter/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxservicecenter/http/     lapisan transport modul ini  — handler, dto, rute
package inboxservicecenter

import (
	"context"
	"strings"
	"time"
)

// ServiceClaim adalah satu baris klaim portal rekanan di Inbox Service Center.
//
// # Kenapa isiannya lebih banyak daripada kolom yang digambar
//
// Grid Pega hanya menggambar ENAM kolom — ID, Tanggal Input, No Polis, Nasabah, Tipe, dan
// PIC. Isian selebihnya dibawa karena dibutuhkan hal lain: `RepairID` untuk membuka rincian,
// `IMEI` dan `ClaimNumber` karena keduanya ikut dicari kotak "Cari", serta kedua status
// karena keempat tab justru dibentuk olehnya.
//
// Tidak ada satu pun dari isian tambahan itu yang menjadi kolom baru. `D-13` menetapkan
// tampilan meniru Pega; menambah kolom yang tidak ada di sana akan membuat uji kesetaraan
// gerbang 1 melaporkan selisih yang bukan berasal dari data.
type ServiceClaim struct {
	// ID adalah nomor yang dibaca pengguna di kolom pertama — `T_KLAIM_PORTAL_REKANAN.ID`,
	// di grid lama dibawa alias `CaseID`.
	ID string

	// RepairID adalah nomor perbaikan dari portal mitra — `REPAIRID`, di grid lama dibawa
	// alias `NoClaim`.
	//
	// Ia dibawa karena dua hal membutuhkannya dan keduanya bukan kolom: riwayat catatan
	// progres dikunci dengannya (`PROGRESS_SERVICECENTER_CLAIM.REPAIRID`), dan pencarian
	// per status mencocokkan KARAKTER PERTAMANYA
	// (`Activity/CariDataSCUntukCaseBaseOnStatus-Act.xml`).
	RepairID string

	// ClaimNumber adalah nomor klaim PNC — `CLAIMNO`.
	//
	// Ia tidak digambar sebagai kolom, tetapi ikut dicari kotak "Cari".
	ClaimNumber string

	// PolicyNumber adalah No Polis — `NOPOLIS`, di grid lama alias `PolicyNo`.
	PolicyNumber string

	// CustomerName adalah Nasabah — `QQNAME`, di grid lama dibawa alias `NamaDokumen`
	// yang namanya menyebut nama dokumen padahal isinya nama tertanggung.
	CustomerName string

	// Type adalah Tipe — `TYPE`, di grid lama dibawa alias `RefNo` yang namanya menyebut
	// nomor referensi.
	Type string

	// TechnicalPIC adalah PIC — `PIC`, di grid lama dibawa alias `PICRekanan`.
	//
	// Ia bukan sekadar kolom: keempat tab MENYARING menurut isian ini terhadap pengguna
	// yang login. Lihat Query.
	TechnicalPIC string

	// InputDate adalah Tanggal Input — `INPUTDATE`, di grid lama dibawa alias `DateOfLoss`
	// yang namanya menyebut tanggal kejadian.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan.
	InputDate *time.Time

	// IMEI adalah nomor IMEI perangkat — `IMEI`, di grid lama dibawa alias `ClientID` pada
	// satu kueri dan `NoKTP` pada kueri lain.
	//
	// Ia tidak digambar sebagai kolom, tetapi ikut dicari kotak "Cari".
	IMEI string

	// ApprovalStatus adalah kode status persetujuan — `STS_APPROVAL`.
	//
	// Kosong berarti NULL, dan itu keadaan yang bermakna: baris yang belum pernah diajukan
	// ke komite. Lihat ApprovalStatusLabel.
	ApprovalStatus string

	// RepairStatus adalah kode status perbaikan — `STATUS`. Lihat RepairStatusLabel.
	RepairStatus string

	// Owner adalah login yang membuat barisnya — `LOGIN`, di grid lama alias `Resources`.
	//
	// Sistem lama menyaring menurut isian ini HANYA bagi access group
	// `GCNMFW:PNCServiceCenter`. Penyaring itu belum dapat diterapkan; lihat Limitations.
	Owner string

	// CommitteeApprover adalah petugas komite yang ditunjuk memutuskan — `KOMITEAPPROVE`.
	CommitteeApprover string
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Karena itu yang tertulis di `Activity/DataServiceCenter-Act.xml` langkah 21 sebagai
// `.PageSize`, berdampingan dengan `.FirstRow = ((.CurrentIndex-1) * .PageSize) + 1` dan
// `.LastRow = .CurrentIndex * .PageSize`. Ia tidak dikarang dan tidak disamakan dengan modul
// lain yang memakai 20.
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
	Items []ServiceClaim
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination

	// Paginated menyatakan hasilnya memang dipotong per halaman.
	//
	// Ia bernilai false saat pengguna sedang mencari, dan itu BUKAN kelalaian: sistem lama
	// mematikan paginasinya sendiri begitu kotak cari terisi. Lihat Query.Paginated.
	Paginated bool
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
	if !p.Paginated {
		return 1
	}

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

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Keempat tab menyaring `PIC` menurut nilai ini, dan itulah yang membuat layar ini
	// menjadi "inbox saya" alih-alih daftar seluruh klaim portal rekanan.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Repo adalah seam ke daftar klaim portal rekanan SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
//
// # Kenapa tidak ada satu pun operasi yang menulis
//
// Bukan karena layarnya memang hanya membaca — di Pega ia punya tombol simpan dan keputusan
// komite. Melainkan karena jalur tulisnya BELUM DAPAT dibangun: seluruhnya bermuara pada
// `POOLDATA.PEGA_PORTAL_REKANAN`, satu stored procedure ber-90 parameter yang dipanggil
// `RDB List/CallProcServiceCenter-SQL.xml`, dan sumbernya TIDAK ADA di `Database/` (`R-01`).
//
// `D-02` menetapkan logikanya ditulis ulang di Go, dan itu tidak dapat dilakukan tanpa
// membaca isinya. Menambalnya dengan tebakan berarti menulis ke kolom uang tanpa mengetahui
// aturannya. Lihat Limitations.
type Repo interface {
	// List mengembalikan satu halaman baris beserta jumlah seluruh baris yang cocok.
	//
	// Berbeda dari Inbox Admin, pemotongan halaman terjadi di BASIS DATA — bukan di
	// aplikasi. Sistem lama pun memotongnya di sana, lewat `WHERE rn >= :awal AND rn <=
	// :akhir` atas kolom `ROW_NUMBER()`; tidak ada alasan menarik seluruh baris ke memori
	// hanya untuk menirunya.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// FindDetail mengambil satu klaim beserta seluruh isian layar rinciannya.
	//
	// Mengembalikan ErrNotFound bila klaimnya tidak ada ATAU bukan milik pemanggil —
	// keduanya sengaja tidak dibedakan (lihat ErrNotFound).
	FindDetail(ctx context.Context, query DetailQuery) (ClaimDetail, error)

	// ListProgress mengambil riwayat catatan progres satu klaim.
	//
	// Ia menerima RepairID, bukan ID: tabel `PROGRESS_SERVICECENTER_CLAIM` dikunci
	// `REPAIRID`, dan keduanya kolom yang berbeda.
	//
	// Riwayat kosong BUKAN galat — klaim yang belum pernah dicatat progresnya memang
	// tidak punya barisnya.
	ListProgress(ctx context.Context, repairID string) ([]ProgressNote, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan klaim satu badan
// hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Modul ini tidak menghitung tenggat apa pun — grid Pega tidak punya kolom Aging. Clock tetap
// ada karena lapisan transport membentuk tanggal dari sini, dan supaya modul ini tidak perlu
// dibongkar saat kolom Aging kelak diminta.
//
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
