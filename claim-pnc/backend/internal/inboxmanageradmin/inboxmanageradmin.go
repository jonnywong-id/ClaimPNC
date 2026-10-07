// Package inboxmanageradmin adalah inti modul Inbox Manager Admin.
//
// # Layar apa ini
//
// Menu `MENU_ID 57` "Inbox Manager Admin" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `InboxManagerAdmin_Harness`. Ia berada di kelompok menu INBOX (`ParentID 2`),
// urutan 1147 — tepat sesudah Inbox Manager Receive / PUCL (1146).
//
// Isinya **pandangan penyelia admin klaim atas antrean registrasi**, dipecah menjadi tiga
// tab menurut lini bisnis:
//
//	Manajemen Admin - Non MBU    unit organisasi AdminPNC
//	Manajemen Admin - PA         unit organisasi AdminPA
//	Manajemen Admin - Travel     unit organisasi AdminTRAVEL
//
// Per `D-79` ia benar-benar Inbox: barisnya PEKERJAAN — disaring menurut unit organisasi
// yang memegang penugasannya — hilang begitu klaimnya selesai
// (`PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`), dan punya tenggat
// berupa kolom "Lama Waktu Klaim". Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// # Sumber datanya berpindah — ketetapan Work Owner 2026-09-27
//
// Semula gabungan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dan `DATAPEGA.PC_ASSIGN_WORKLIST`; kini
// SATU tabel datar `POOLDATA.T_CLAIMLIST_ADMIN`. Modul ini tidak lagi menyentuh skema
// DATAPEGA sama sekali.
//
// Dua akibatnya dicatat di PlannedDifferences karena keduanya terlihat pengguna: join
// `INNER` yang lama membuat klaim tanpa assignment LENYAP dan klaim dengan banyak
// assignment TAMPIL BERKALI-KALI — keduanya hilang bersama join itu. Sebaliknya, tabel
// datarnya belum memuat seluruh klaim, sehingga daftarnya untuk sementara lebih pendek
// daripada di Pega.
//
// Tiga kolom yang dibutuhkannya belum ada di tabel itu dan DIMINTA ditambahkan lewat
// `migrations/0005` tahap 1 — `PXASSIGNEDORGUNIT`, `PXCREATEOPNAME`, `STATUSCLAIM_1`.
// Rinciannya, termasuk kenapa tidak satu pun boleh diakali, ada di kepala
// repo/sqlstore/inboxmanageradmin.sql.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxManagerAdmin_Harness-Harness.xml        pembungkus layar; klon UserInbox_Harness
//	Section/InboxManagerAdmin_Section-Section.xml        judul "My Inbox", 3 kontainer grid
//	Report Definition/ManagementAdminView-RD.xml         kolom, penyaring, parameter OrgUnit
//	Activity/SetAssignmentInboxReg_act-Act.xml           tombol per baris
//	Activity/ExportExcelManagerAdminPA-Act.xml           tombol Export To Excel
//
// # Modul ini BACA-SAJA, dan itu terbaca dari activity tombolnya
//
// Tombol pada kolom terakhir setiap baris menjalankan `SetAssignmentInboxReg_act`. Activity
// itu punya **satu langkah saja**, dan langkah itu Property-Set:
//
//	TempIns.pyNote := "ASSIGN-WORKLIST " + param.inskey + "!Register_Flow"
//
// Ia tidak menugaskan apa pun, tidak menulis satu baris pun, dan tidak mengubah status
// klaim. Yang disusunnya adalah KUNCI ASSIGNMENT untuk dibuka — pola yang sama sudah
// tercatat pada `SetAssignmentCompliance` di modul Inbox Compliance. Karena itu seam Repo di
// bawah hanya punya satu operasi, dan tidak satu pun yang menulis.
//
// # Deskripsi Report Definition-nya menyesatkan, dan itu bukan kelalaian pembacaan
//
// `ManagementAdminView-RD.xml:13` berbunyi **"ONLY SHOWS VALUE IF THE USERTEKNIS IS NULL"**.
// Pembacaan seluruh berkas RD membantahnya: `pyFilterLogic` berbunyi `A AND B AND C`, dan
// ketiganya adalah unit organisasi penugasan ditambah dua penyaring status kerja. Properti
// `.ClaimData.UserTeknis` muncul HANYA sebagai kolom tampilan (`pyListFields` nomor 9,
// berjudul "Nama PIC Teknik"), tidak sekali pun sebagai penyaring.
//
// Deskripsi itu karena itu **usang atau penyaringnya pernah dihapus**, dan tidak diterapkan
// di sini. Menerapkannya berarti menambah penyaring yang tidak dijalankan sistem lama, dan
// akibatnya baris yang hari ini terlihat akan hilang tanpa satu pun pesan galat. Ia dicatat
// sebagai pertanyaan terbuka di docs/keputusan-implementasi.md alih-alih diputuskan sendiri.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxmanageradmin/          aturan modul + seam          ← paket ini
//	inboxmanageradmin/usecase/  orkestrasi: daftar tab, isi satu tab
//	inboxmanageradmin/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxmanageradmin/http/     lapisan transport modul ini  — handler, dto, ekspor, rute
package inboxmanageradmin

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/platform/pagination"
)

// WorkItem adalah satu baris antrean di Inbox Manager Admin.
//
// # Kenapa satu bentuk untuk tiga tab, dan seluruh isiannya terisi
//
// Berbeda dari modul Inbox Admin yang delapan tabnya berbagi satu halaman klipboard dengan
// kolom berbeda-beda, ketiga tab di sini dilayani **Report Definition yang SAMA**
// (`ManagementAdminView`) dengan satu-satunya perbedaan pada nilai parameter `OrgUnit`.
// Kolomnya karena itu identik, dan tidak ada satu pun isian yang kosong karena "tab ini
// tidak membawanya".
//
// Satu isian memang tidak digambar grid: ClaimStatus. Lihat catatan di sana.
type WorkItem struct {
	// Reference adalah kunci teknis Pega — `PZINSKEY`, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada baris warisan.
	//
	// Ia dibawa karena tombol pada kolom terakhir membutuhkannya — activity lamanya
	// menyusun `"ASSIGN-WORKLIST " + pzInsKey + "!Register_Flow"` dari nilai ini — BUKAN
	// untuk ditampilkan. `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas
	// Pega ke data bisnis sebagai utang teknis, dan `D-22` menetapkan klaim terbitan
	// sistem baru tidak pernah menulis awalan itu lagi.
	Reference string

	// CaseID adalah kolom pertama yang dibaca pengguna, berjudul "ID" — `PYID`.
	//
	// Judulnya memang sependek itu di `Section/InboxManagerAdmin_Section-Section.xml`,
	// dan dipertahankan apa adanya (`D-13`) meski di layar inbox lain kolom yang sama
	// berjudul "Case ID".
	CaseID string

	// PolicyNumber adalah No Polis — `POLICYNO`, di RD `.Policy.PolicyNo`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — `QQNAME`, di RD `.Policy.QQName`.
	InsuredName string

	// BusinessName adalah Nama Bisnis — `BUSINESSNAME`, di RD
	// `.Policy.Quotation.BusinessName`.
	BusinessName string

	// BusinessSource adalah Nama Sumber Bisnis — `SOBNAME`, di RD
	// `.Policy.Quotation.SobName`.
	//
	// Perhatikan judulnya: di layar ini "Nama Sumber Bisnis", sedangkan Report Definition
	// yang memasoknya memberi label "Nama Sumbis". Yang dibaca pengguna adalah yang di
	// section (`D-13`).
	BusinessSource string

	// RegisteredAt adalah Tanggal Pendaftaran — `PXCREATEDATETIME`.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan, dan kolom Lama Waktu Klaim yang dihitung darinya
	// akan berbunyi dalam ribuan tahun.
	RegisteredAt *time.Time

	// AdminName adalah Admin PNC — `PXCREATEOPNAME`, di RD berlabel
	// "Create Operator Name".
	//
	// Ia petugas yang MEMBUAT barisnya, bukan petugas yang sedang memegangnya. Judul
	// "Admin PNC" karena itu lebih sempit artinya daripada bunyinya, dan itu dipertahankan
	// apa adanya (`D-13`).
	AdminName string

	// ClaimStatus adalah Status Klaim, diturunkan dari `PYSTATUSWORK`.
	//
	// # Kenapa ia ada padahal grid tidak menggambarnya
	//
	// Karena BERKAS EKSPOR memuatnya. `Activity/ExportExcelManagerAdminPA-Act.xml`
	// menyusun berkasnya dengan delapan judul kolom, dan yang keenam berbunyi
	// "Status Klaim" — kolom yang tidak ada di grid mana pun pada layar yang sama.
	//
	// Work Owner memutuskan 2026-09-26 berkas ekspor mengikuti kedelapan kolom Pega apa
	// adanya, sehingga isian ini wajib ada meski layar tidak menampilkannya.
	//
	// # Sumbernya berubah, dan akibatnya perlu disadari
	//
	// Semula ia label dari `V_STS_CLAIM.LSC_NOTE` yang dicari berdasarkan `STATUSCLAIM_1` —
	// domain 33 kode (`1134`–`1166`). Koreksi Work Owner 2026-09-27: kolom ini diturunkan
	// dari `PYSTATUSWORK`, sama seperti modul Inbox Outstanding.
	//
	// Akibat yang TIDAK menguntungkan dan tidak disembunyikan: kueri layar ini sudah
	// menyaring `PYSTATUSWORK NOT IN ('Resolved-Completed','Resolved-Rejected')`, sehingga
	// hampir setiap baris bernilai sama. Kolom yang dulu membedakan 33 keadaan bisnis kini
	// praktis berisi satu nilai. Ia dicatat sebagai selisih terencana, bukan diperbaiki
	// sepihak — lihat PlannedDifferences.
	ClaimStatus string

	// ClaimElapsed adalah Lama Waktu Klaim — waktu relatif sejak RegisteredAt, misalnya
	// "1 year 5 months ago".
	//
	// # Ia BUKAN properti tersendiri di sistem lama
	//
	// Ia properti yang SAMA dengan Tanggal Pendaftaran — `.pxCreateDateTime`, yang karena
	// itu muncul dua kali di section — hanya digambar dengan mode kontrol kedua
	// ber-`pyDateTimeFormat` **`DateTime-Frame`**. Bentuk itu muncul tepat tiga kali di
	// section ini, satu untuk tiap tab.
	//
	// Dihitung di Go, bukan di SQL: hasilnya dapat diuji secara deterministik lewat seam
	// Clock, dan kueri tetap portabel tanpa fungsi tanggal khas Oracle
	// (`08-TECHNICAL-STRATEGY.md` §4.3).
	ClaimElapsed string
}

// WithElapsed mengembalikan salinan baris yang kolom Lama Waktu Klaim-nya sudah terisi.
func (w WorkItem) WithElapsed(now time.Time) WorkItem {
	filled := w
	filled.ClaimElapsed = FormatElapsed(w.RegisteredAt, now)
	return filled
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 50
//
// Karena itu yang tertulis di `Section/InboxManagerAdmin_Section-Section.xml` sebagai
// ukuran halaman ketiga gridnya (`pyPageSize` bernilai 50, tiga kali). Ia tidak dikarang,
// dan tidak disamakan dengan modul lain yang memakai 25.
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
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
// diminta. Layar menggambar penomoran halamannya dari sini.
type Page = pagination.Page[WorkItem, pageSizes]

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// # Kenapa dipotong di aplikasi, bukan di basis data
//
// Keputusan Work Owner 2026-09-26: paginasi layar ini **direplikasi apa adanya**, pilihan
// yang sama dengan modul Inbox Admin. Sistem lama menjalankan Report Definition-nya dengan
// `pyMaxRecords=500` lalu menyerahkan seluruh hasilnya ke grid, yang memotong halamannya
// sendiri di klipboard — bukan di basis data. Tidak ada satu pun `OFFSET` di seluruh export
// (`ADR-0011`).
//
// Keberatan atas konsekuensinya sudah disampaikan sebelum keputusan diambil dan keputusan
// ditegaskan; ia dicatat di sini supaya menjadi keputusan yang tercatat, bukan kelalaian.
//
// Yang diterima secara sadar: satu unit organisasi yang antreannya panjang dapat menarik
// seluruh barisnya ke memori aplikasi. Usecase karena itu MEMPERINGATKAN lewat log begitu
// satu permintaan melampaui LargeResultWarning.
//
// Risiko itu untuk sementara LEBIH KECIL daripada saat sumbernya masih tabel kerja Pega:
// `POOLDATA.T_CLAIMLIST_ADMIN` baru memuat 1.014 baris, bukan puluhan juta (`D-10`). Tetapi
// ia dirancang untuk akhirnya memuat seluruh klaim, sehingga peringatan itu tetap ada —
// menghapusnya sekarang berarti menghapusnya tepat sebelum ia dibutuhkan.
//
// Satu hal yang TIDAK direplikasi: batas 500 baris. Menyalinnya berarti baris ke-501
// seterusnya tidak pernah terlihat — cacat yang `ADR-0011` nyatakan sebagai pertanyaan
// terbuka, bukan perilaku yang layak dibawa. Ia dicatat sebagai selisih terencana.
func Slice(all []WorkItem, page Pagination) Page { return pagination.Slice(all, page) }

// LargeResultWarning adalah jumlah baris yang, begitu terlampaui, dicatat sebagai
// peringatan di log.
//
// Ia TIDAK memotong hasil dan tidak mengubah perilaku apa pun — memotongnya akan menyalahi
// keputusan "replikasi apa adanya". Yang dilakukannya hanya membuat akibat keputusan itu
// terlihat operator sebelum ia terlihat sebagai aplikasi yang kehabisan memori.
const LargeResultWarning = 5000

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Tidak satu pun kueri di modul ini menyaring menurut nilai ini: layar ini pandangan
	// penyelia atas pekerjaan orang lain. Yang memakainya adalah jejak log, dan selama
	// pemeriksaan peran belum ada (`TKT-F3-004`) itulah satu-satunya kontrol yang tersisa.
	Login string

	// LineBusiness adalah lini bisnis petugas — padanan `OperatorID.pyPosition` sistem
	// lama, dibaca dari `POOLDATA.M_LOGIN_PNC.LINE_BUSINESS`.
	//
	// Ia MENENTUKAN tab mana yang terlihat, dan itu keputusan Work Owner 2026-09-26:
	// visibilitas tab mengikuti Pega apa adanya. Lihat VisibleTabs.
	//
	// # Kenapa ia BUKAN jabatan, dan kenapa itu perlu ditegaskan
	//
	// Sampai 2026-09-27 isian ini bernama `Position` dan diisi dari
	// `auth.User.Position`, yang bersumber dari HCQ `EmpResponse.Placement.PositionName` —
	// sebuah JABATAN KEPEGAWAIAN, misalnya "IT SPECIALIST". Nilai itu tidak pernah berisi
	// `NONMBU`, `PA`, maupun `TRAVEL`, sehingga **tidak seorang pun melihat satu tab pun**.
	//
	// Itu cacat, bukan kesetaraan dengan Pega. Di Pega `pyPosition` adalah field pada
	// rekaman operator yang diisi administrator dengan KODE LINI BISNIS — terbaca dari
	// perbandingannya di export: `'NONMBU'`, `'PA'`, `'TRAVEL'`
	// (`Section/InboxManagerAdmin_Section-Section.xml:2618`, `:6755`, `:10531`), dan
	// `'BROKER'` di layar lain. Karena terisi, kontainernya tampil.
	//
	// Padanannya di sistem baru adalah `M_LOGIN_PNC.LINE_BUSINESS` — kolom yang SUDAH ADA,
	// terverifikasi dari `ALL_TAB_COLUMNS`, dan sudah dibaca modul Inbox Outstanding lewat
	// kueri `line_business_for`. Nilainya diisi lewat seam LineBusinessRepo di bawah, bukan
	// diambil dari sesi.
	LineBusiness string

	// OrgUnit adalah unit organisasi pengguna — padanan `OperatorID.pyOrgUnit`.
	//
	// Satu nilai punya arti khusus di layar ini: `Development` membuka ketiga tab
	// sekaligus. Lihat DevelopmentOrgUnit.
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

// Repo adalah seam ke antrean SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut
// entitas, dan memang tidak boleh ada.
//
// # Kenapa hanya ada satu operasi, dan tidak ada satu pun yang menulis
//
// Karena layar ini tidak mengubah apa pun — lihat catatan tentang
// `SetAssignmentInboxReg_act` di kepala paket. Operasi yang tidak tersedia di seam ini tidak
// dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar, dan pada masa paralel itu
// penting: seluruh tabel yang dibaca modul ini adalah milik Pega (`P-1`).
type Repo interface {
	// List mengembalikan SELURUH baris yang cocok, belum dipaginasi.
	//
	// Ia sengaja tidak menerima Pagination: pemotongan halaman terjadi di aplikasi
	// (lihat Slice), dan menaruhnya di sini akan menyembunyikan bahwa seluruh baris
	// memang ditarik.
	List(ctx context.Context, query Query) ([]WorkItem, error)
}

// LineBusinessRepo adalah seam ke lini bisnis petugas.
//
// Ia dipisah dari Repo dengan sengaja, meski keduanya membaca basis data yang sama: Repo
// membaca ANTREAN, seam ini membaca KEWENANGAN. Menggabungkannya membuat satu interface
// menjawab dua pertanyaan yang berbeda umurnya — antrean berubah setiap saat, lini bisnis
// petugas nyaris tidak pernah.
//
// Bentuknya mengikuti `inboxoutstanding.Repo.LineBusinessFor` yang sudah ada, termasuk
// perlakuannya terhadap petugas tanpa baris: lihat catatan di pengisinya.
type LineBusinessRepo interface {
	// LineBusinessFor membaca lini bisnis seorang petugas dari `M_LOGIN_PNC`.
	//
	// Petugas tanpa baris, atau yang kolomnya kosong, mengembalikan teks kosong TANPA
	// galat. Itu bukan kelonggaran: di Pega `pyPosition` yang tidak cocok satu pun
	// sekadar tidak membuka kontainer mana pun — ia tidak menggagalkan layarnya. Menjadikan
	// ketiadaannya galat akan mengubah layar yang seharusnya menjelaskan diri menjadi layar
	// yang rusak.
	LineBusinessFor(ctx context.Context, loginID string) (string, error)
}

// LineBusinessRepoSelector memilih LineBusinessRepo milik satu portal entitas.
//
// Ia ada dan tidak digabung ke sesi karena `M_LOGIN_PNC` adalah tabel PER ENTITAS: petugas
// yang sama dapat punya lini bisnis berbeda di badan hukum yang berbeda, dan membacanya dari
// koneksi portal yang salah berarti membuka tab atas dasar kewenangan entitas lain
// (`ADR-0030`, `R-20`).
type LineBusinessRepoSelector func(portalAlias string) (LineBusinessRepo, error)

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan antrean kerja
// satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Ia ada supaya kolom Lama Waktu Klaim dapat diuji secara deterministik. Seluruh waktu yang
// dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan tidak pernah dengan
// menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
