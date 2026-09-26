// Package inboxcompliance adalah inti modul Inbox Compliance.
//
// # Layar apa ini
//
// Menu `MENU_ID 47` "Inbox Compliance" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `inboxCompliance_Harness`. Ia berada di kelompok menu `2` (Proses Produksi),
// urutan 1137 — tepat sebelum Inbox Investigator (1138) dan jauh sebelum Inbox Admin (1153).
//
// Isinya **daftar pekerjaan milik petugas Compliance**. Per `D-79` ia benar-benar Inbox,
// bukan layar data acuan: barisnya pekerjaan yang diambil dari workbasket
// `DATAPEGA.PC_ASSIGN_WORKBASKET`, hilang begitu klaimnya selesai
// (`PYSTATUSWORK <> 'Resolved-Completed'`), dan punya tenggat berupa kolom Aging. Karena itu
// modul ini milik `U-3`, bukan `U-6`.
//
// # Koreksi atas inventaris harness
//
// `22-INVENTARIS-HARNESS.md` menandai harness ini **JANGGAL** dengan alasan `RD 0` —
// "antrean kerja tanpa sumber data tidak masuk akal". Premis itu KELIRU, dan koreksinya
// dicatat di sini karena ia akan ditanyakan lagi:
//
//	Report Definition/InboxRegisterCompliance_RD-RD.xml   tab Compliance
//	Report Definition/InboxCompliance_RD-RD.xml           tab Post Audit
//
// Keduanya ADA. Ia tidak terhitung pada tingkat harness karena harness ini cangkang tipis —
// RD-nya tinggal di dalam section, bukan di harness. Sidik jari 145 KB · RD 0 karena itu
// tidak menyatakan "layar tanpa data", melainkan "layar yang isinya ada di tempat lain".
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/inboxCompliance_Harness-Harness.xml         pembungkus; judul "Inbox Compliance"
//	Section/InputCompliance_Section-Section.xml         MURNI pembungkus dua tab; tanpa field
//	Section/InputComplianceDtl_Section-Section.xml      grid tab Compliance, 9 kolom
//	Section/InputPostAuditDtl_Section-Section.xml       grid tab Post Audit, 8 kolom
//	Report Definition/InboxRegisterCompliance_RD-RD.xml sumber tab Compliance
//	Report Definition/InboxCompliance_RD-RD.xml         sumber tab Post Audit
//	Activity/GetInboxRegisterCompliance-Act.xml         pengisi kolom Aging
//	RDB List/GetSelisihJam_sql-SQL.xml                  pemformat Aging
//	Database/GETSELISIHJAM.fnc                          hitungan jam Aging
//	Flow/Register_Flow.xml:3769                         nama workbasket: CompliancePNC
//	When/IsCompliance-When.xml                          gerbang alur: isComplianceTransfer = "1"
//
// # Layar ini TIDAK punya penyaring
//
// Dicatat karena mudah salah duga. `GetInboxRegisterCompliance-Act` memuat parameter
// bernama `CARI1` dan `CARI2`, dan nama itu di modul lain memang berarti kotak cari. Di sini
// TIDAK: keduanya adalah dua argumen tanggal untuk `GETSELISIHJAM` —
// `CARI1 = "SYSDATE"` dan `CARI2 = TanggalBuatCompliance + 7 jam`
// (`Activity/GetInboxRegisterCompliance-Act.xml:750`, `:815`).
//
// `Section/InputCompliance_Section-Section.xml` yang membungkus kedua tab juga tidak memuat
// satu pun field masukan, tombol, maupun dropdown — hanya dua kontainer `TABBED` berjudul
// "Compliance" dan "Post Audit". Menambahkan kotak cari di sini berarti mengarang kemampuan
// yang tidak ada di sistem lama.
//
// # Kenapa paginasinya BERBEDA dari modul Inbox Admin
//
// Ini perbedaan yang disengaja, bukan kelalaian, dan alasannya perlu tercatat.
//
// Modul Inbox Admin menarik SELURUH baris lalu memotongnya di aplikasi. Itu keputusan Work
// Owner 2026-09-20 yang berlaku KHUSUS untuk layar itu — replikasi apa adanya atas perilaku
// Pega yang memang menarik semuanya lebih dulu.
//
// Tidak ada keputusan serupa untuk layar ini, sehingga yang berlaku adalah standar koding:
// **paginasi selalu server-side** (`08-TECHNICAL-STRATEGY.md` §5 dan
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 butir 1). Baris dipotong di basis data lewat
// `OFFSET … FETCH NEXT`, sehingga memori aplikasi tidak bergantung pada besarnya antrean.
//
// Satu akibatnya harus dinyatakan di muka sebagai selisih terencana pada gerbang 1: RD lama
// memasang `pyMaxRecords=500`, sehingga antrean yang lebih panjang dari 500 baris
// **terpotong** di Pega. Sistem baru tidak memotongnya. Ini penambahan kemampuan yang sudah
// diperkirakan `ADR-0011`, bukan cacat.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxcompliance/          aturan modul + seam          ← paket ini
//	inboxcompliance/usecase/  orkestrasi: daftar tab, isi satu tab
//	inboxcompliance/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxcompliance/http/     lapisan transport modul ini  — handler, dto, rute
package inboxcompliance

import (
	"context"
	"time"
)

// WorkbasketCompliance adalah nama workbasket yang menampung antrean Compliance.
//
// Nilainya dibaca dari `Flow/Register_Flow.xml:3769` — shape assignment "Compliance"
// ber-`pyImplementation` WorkBasket dengan `pyWorkBasket` bernilai `CompliancePNC`. Ia
// BUKAN nama orang dan bukan nama peran.
//
// # Kenapa ia konstanta di sini, padahal `D-15` melarang nilai bisnis di-hardcode
//
// Karena yang dilarang `D-15` adalah nilai yang BERUBAH menurut kebijakan — ambang uang,
// penerima notifikasi, pemetaan peran. Nama workbasket bukan salah satunya: ia identitas
// antrean yang menjadi bagian dari bentuk alur kerja, dan mengubahnya berarti mengubah
// flow-nya, bukan mengubah kebijakan.
//
// Preseden yang sama sudah ada dan sengaja diikuti: modul Inbox Admin menuliskan
// `wb.PXASSIGNEDOPERATORID = 'RCLPUCL'` di dalam kuerinya.
//
// Yang tetap dicatat sebagai utang: di Pega nilai ini sampai ke RD lewat `Param.Operator`,
// yakni PARAMETER — sehingga secara mekanisme ia memang dapat berbeda per pemanggil. Section
// yang menyetel parameter itu tidak memuat satu pun pilihan bagi pengguna, jadi satu nilai
// tetap adalah pembacaan yang benar hari ini. Bila kelak terbukti ada pemanggil lain dengan
// workbasket berbeda, tempat mengubahnya satu: konstanta ini.
const WorkbasketCompliance = "CompliancePNC"

// WorkItem adalah satu baris pekerjaan di Inbox Compliance.
//
// # Kenapa satu bentuk untuk dua tab
//
// Karena keduanya memang berbagi sebagian besar kolomnya — nomor kasus, nomor polis, nama
// tertanggung — dan hanya berbeda pada ujungnya: tab Compliance membawa Nama Bisnis, Nama
// Cabang, Nama Admin, dan Aging; tab Post Audit membawa Catatan Compliance dan Tanggal
// Kirim Post Audit.
//
// Kolom yang tidak dibawa tab yang sedang terbuka bernilai kosong, dan layar
// menyembunyikannya mengikuti Tab.Columns — bukan menampilkan kolom kosong yang membuat
// pengguna menduga datanya hilang.
type WorkItem struct {
	// CaseID adalah Nomor Case yang dibaca pengguna di kolom pertama — `.pyID`, berformat
	// `PNC-xxxx` pada baris warisan dan `PNCN.YY.xxxx` pada terbitan sistem baru (`D-71`).
	CaseID string

	// Reference adalah kunci teknis Pega — `PZINSKEY`, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada baris warisan.
	//
	// Ia dibawa karena tombol buka detail klaim membutuhkannya, BUKAN untuk ditampilkan.
	// `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas Pega ke data bisnis
	// sebagai utang teknis, dan `D-22` menetapkan klaim terbitan sistem baru tidak pernah
	// menulis awalan itu lagi.
	Reference string

	// ClaimNumber adalah kolom "No Klaim" pada tab Post Audit — `.pxCoverInsKey`.
	//
	// Namanya menyesatkan, dan itu bentuk sistem lama: isinya BUKAN nomor klaim melainkan
	// kunci teknis Pega berbentuk "ASM-FW-GCNMFW-WORK PNC-2114". Layar Pega menampilkannya
	// apa adanya di kolom berjudul "No Klaim", dan `D-13` menetapkan tampilan ditiru —
	// sehingga ia ikut ditampilkan, bukan diperbaiki diam-diam.
	//
	// Tab Compliance tidak memakainya: di sana nomor kasus dan kunci teknis memang dua
	// kolom yang berbeda, dan kuncinya tidak pernah digambar.
	ClaimNumber string

	// PolicyNumber adalah No Polis — `.Policy.PolicyNo`, kolom `POLICYNO`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — `.Policy.QQName`, kolom `QQNAME`.
	InsuredName string

	// BusinessName adalah Nama Bisnis — `.Policy.Quotation.BusinessName`, kolom
	// `BUSINESSNAME`. Hanya tab Compliance yang membawanya.
	BusinessName string

	// BranchName adalah Nama Cabang — `.Policy.Quotation.BranchName`, kolom `BRANCHNAME`.
	// Hanya tab Compliance yang membawanya.
	BranchName string

	// AdminName adalah Nama Admin — `.pyOrigUserID`.
	//
	// Perhatikan: BUKAN `PXCREATEOPERATOR` yang dipakai modul Inbox Admin untuk kolom
	// "Creator". `InboxRegisterCompliance_RD` secara eksplisit memberi label "Nama Admin"
	// pada `.pyOrigUserID`, dan keduanya dapat berbeda ketika baris dibuat job terjadwal
	// atau layanan REST masuk alih-alih oleh orang.
	AdminName string

	// ComplianceSentDate adalah Tanggal Kirim Compliance —
	// `.ClaimData.TanggalBuatCompliance`. Ia dasar hitungan Aging.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan, dan perbedaan itu menentukan apakah kolom Aging
	// terisi.
	ComplianceSentDate *time.Time

	// PostAuditSentDate adalah Tanggal Kirim Post Audit — `.TanggalKirimPostAudit`.
	// Hanya tab Post Audit yang membawanya.
	PostAuditSentDate *time.Time

	// ComplianceRemarks adalah catatan petugas Compliance — `.ComplianceRemarks`.
	// Hanya tab Post Audit yang membawanya.
	//
	// Namanya mudah tertukar dengan alias `"ComplianceRemark"` (tanpa `s`) pada
	// `RDB List/BrowseClaimStudy-SQL.xml:85`, yang isinya sama sekali bukan catatan
	// melainkan `SUM(TOTAL_CLAIM * CURRENCYVALUE)` — satu lagi contoh alias menyesatkan
	// `03-CURRENT-ARCHITECTURE.md` §4.2. Keduanya tidak berhubungan.
	ComplianceRemarks string

	// AgingHours adalah lama antrean dalam JAM, sudah dipotong akhir pekan.
	//
	// Pointer supaya "tidak dapat dihitung" dapat dibedakan dari "nol jam" — dan keduanya
	// memang berbeda: nol jam berarti baru saja masuk antrean, sedangkan tidak dapat
	// dihitung berarti Tanggal Kirim Compliance belum terisi.
	//
	// Pembedaan itu WAJIB, bukan pilihan gaya: `Database/GETSELISIHJAM.fnc:22`
	// mengembalikan `0` pada setiap kegagalan, sehingga kegagalan tak terbedakan dari nol
	// jam. `D-49` butir 10 memutuskan cacat itu DIPERBAIKI, dan ia masuk daftar 13
	// perbaikan eksplisit `P-5`.
	AgingHours *float64

	// Outstanding adalah kolom "OutStanding" pada tab Post Audit: lama sebuah pemeriksaan
	// menggantung, dihitung dari PostAuditSentDate.
	//
	// # Kenapa ia BUKAN kolom Aging dengan nama lain
	//
	// Karena dasarnya berbeda, dan itu terbukti dari angkanya. Layar Pega menampilkan
	// "1 year 5 months ago" untuk baris bertanggal 22/04/25 — dan 22 April 2025 sampai
	// hari ini memang tepat 1 tahun 5 bulan secara KALENDER. Bila akhir pekan dipotong
	// seperti pada kolom Aging, selisihnya menyusut sekitar 149 hari dan angkanya akan
	// berbunyi "1 year 0 months".
	//
	// Jadi Outstanding adalah waktu kalender apa adanya, sedangkan Aging memotong akhir
	// pekan. Keduanya sengaja tidak berbagi satu perhitungan.
	Outstanding string
}

// AgingLabel adalah kolom Aging sebagaimana dibaca pengguna.
//
// Kosong bila Aging tidak dapat dihitung — bukan "0 hours ago", sesuai `P-5` butir 13.
func (w WorkItem) AgingLabel() string {
	return FormatAging(w.AgingHours)
}

// WithElapsed mengembalikan salinan baris yang kedua kolom hitungan waktunya sudah terisi.
//
// Keduanya dihitung di sini, bukan di SQL, karena tiga alasan: hasilnya dapat diuji secara
// deterministik lewat seam Clock, kueri tetap portabel tanpa memanggil fungsi basis data
// (`D-02`), dan hitungan waktu tunggu adalah ATURAN BISNIS yang `D-50` tetapkan ditulis
// ulang di Go — bukan pengambilan data.
//
// Satu baris hanya akan punya salah satunya: tab Compliance membawa Tanggal Kirim
// Compliance dan tidak membawa Tanggal Kirim Audit Compliance, dan sebaliknya.
func (w WorkItem) WithElapsed(now time.Time) WorkItem {
	filled := w
	filled.AgingHours = AgingHoursBetween(w.ComplianceSentDate, now)
	filled.Outstanding = FormatElapsed(w.PostAuditSentDate, now)
	return filled
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Menyamai modul Inbox Admin, yang mengambilnya dari ukuran halaman grid di
// `Section/PNCInboxAdmin-Section.xml`. Grid layar ini tidak menyatakan ukuran halamannya
// sendiri — yang dinyatakannya hanya `pyMaxRecords=500` di tingkat RD, yakni batas seluruh
// hasil, bukan batas satu halaman. Karena tidak ada angka yang dapat dibaca, yang dipakai
// adalah angka yang sudah berlaku di layar antrean kerja lain.
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

// Repo adalah seam ke antrean Compliance SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Satu-satunya operasi yang MENULIS, dan tabel mana yang ditulisnya
//
// `CreatePostAudit` adalah satu-satunya. Ia menulis ke `POOLDATA.T_CLAIM_COMPLIANCE_H`,
// yakni tabel BARU yang tidak dikenal Pega — bukan ke tabel warisan mana pun.
//
// Pembedaan itu yang membuatnya tidak melanggar `P-1`: selama masa paralel setiap tabel
// hanya boleh ditulis satu sistem, dan untuk tabel ini sistem itu adalah aplikasi baru.
// Ketiga tabel warisan yang dibaca modul ini — `PC_ASM_FW_GCNMFW_WORK`,
// `PC_ASSIGN_WORKBASKET`, `T_CLAIM_PNC` — tetap DIBACA saja, dan tidak ada operasi di seam
// ini yang dapat mengubahnya.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris beserta jumlah seluruh baris yang cocok.
	//
	// Berbeda dengan modul Inbox Admin, ia menerima Pagination dan memotong halamannya di
	// basis data. Alasannya ada di dokumentasi paket ini, bagian "Kenapa paginasinya
	// BERBEDA dari modul Inbox Admin".
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// FindInQueue mencari satu klaim yang SEDANG menunggu di antrean Compliance.
	//
	// Nilai kedua bernilai salah bila klaimnya tidak ada di antrean itu — termasuk ketika
	// klaimnya ada tetapi sudah selesai, atau sudah berpindah antrean. Pemanggil TIDAK
	// boleh memperlakukan keduanya sebagai galat teknis.
	//
	// Ia ada supaya pengiriman ke Post Audit hanya dapat dilakukan atas klaim yang memang
	// sedang diperiksa — meniru Pega, yang menuntut assignment-nya ada sebelum flow-nya
	// dapat dijalankan.
	FindInQueue(ctx context.Context, query Query, reference string) (WorkItem, bool, error)

	// CreatePostAudit menulis satu baris Post Audit dan mengembalikannya beserta nomor
	// yang terbit.
	//
	// Nomornya dibentuk DI SINI dari sequence basis data, bukan diterima dari pemanggil.
	// Nomor yang dikarang lapisan di atasnya akan bertabrakan diam-diam: tabelnya tidak
	// punya primary key maupun constraint unik, sehingga basis data tidak akan menolaknya.
	CreatePostAudit(ctx context.Context, entry PostAuditEntry) (PostAuditEntry, error)
}

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
// Ia ada supaya hitungan Aging dapat diuji secara deterministik. Seluruh waktu yang
// dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan tidak pernah dengan
// menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
