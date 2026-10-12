// Package inboxsalvage adalah inti modul Inbox Salvage.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `MENU_ID 71` berbunyi **"Inbox Salvage"** dengan `MENU_PROGRAM` `InboxSalvage`, dan
// `Harness/InboxSalvage-Harness.xml` memuat judul yang sama di dalam layarnya. `D-81`
// menetapkan nama modul mengikuti nama yang dipakai Work Owner.
//
// # Artefak Pega yang dibaca
//
//	Harness/InboxSalvage-Harness.xml              rangka layar, judul bergantung portal
//	Section/InboxSalvage-Section.xml              tombol Tambah/Refresh, tabel ringkas, pyPageSize
//	Section/InboxSalvageASM-Section.xml           isi portal ASM — 13 grid
//	Section/InboxSalvageInsurtech-Section.xml     isi portal Insurtech — BELUM dibangun
//	Section/TambahData_Salvage-Section.xml        form Tambah
//	Activity/GCNMCountSalvage_act-Act.xml         pencacah "Status Salvage / Jumlah" (51 langkah)
//	Activity/SetDataSalavage_act-Act.xml          pemuat daftar (52 langkah) — Param.tipe/tipe2
//	Activity/SetStsSalvagePNC_act-Act.xml         tombol Submit (31 langkah)
//	Activity/GCNMNewSalvage_act-Act.xml           pemetaan form -> parameter procedure
//	Activity/UploadDetailSalvage-Act.xml          unggah CSV detail item
//	Data Transform/CNMShowInsertSalvage_dt-DT.xml pembersih form "Tambah"
//	Flow Action/UploadDetailSalvage-FA.xml        pembungkus pxUploadCSVResults
//	RDB List/GcnmSalvageData_OS_SQL-SQL.xml       kueri keluarga A
//	RDB List/GcnmSalvageData_ekonomisdanTba-*.xml kueri keluarga B
//	RDB List/GcnmSalvageData_CloseOs_SQL-SQL.xml  kueri keluarga C
//	RDB List/PNCSalvageGetChekerDataKlaimAllData  agregat detail per baris (checker)
//	RDB List/InsertNewSalvage-SQL.xml             pemanggil INSERT_SALVAGE
//	Database/INSERT_SALVAGE.prc                   penyimpan PNC_SALVAGE
//	Database/INSERT_SALVAGE_DETAILS.prc           penyimpan DETAIL_PNC_SALVAGE
//
// # Apa itu Salvage
//
// `CONTEXT.md`: **Salvage** adalah nilai sisa barang rusak yang bisa dijual kembali,
// termasuk lewat balai lelang, dan ia MENGURANGI nilai bersih klaim. Layar ini adalah
// antrean kerja pengelolaannya, dari barang yang baru ditandai ada sampai terjual.
//
// # TIGA KELUARGA KUERI, BUKAN TIGA BELAS
//
// Layar lama menggambar 13 daftar, tetapi ketiga belasnya hanya memakai TIGA kueri. Itu
// temuan yang menentukan bentuk modul ini — tanpanya, parity penuh akan terbaca seperti
// tiga belas pekerjaan alih-alih tiga:
//
//	keluarga A  GcnmSalvageData_OS_SQL           T_CLAIM_PNC            1 daftar   4 kolom
//	keluarga B  GcnmSalvageData_ekonomisdanTba   T_CLAIM_PNC + objek    5 daftar   4 kolom
//	keluarga C  GcnmSalvageData_CloseOs_SQL      PNC_SALVAGE + klaim    7 daftar   6-12 kolom
//
// Yang membedakan daftar di dalam satu keluarga hanyalah PENYARING, dan penyaringnya
// disisipkan `SetDataSalavage_act` ke dalam `{ASIS:…}` menurut `Param.tipe` dan
// `Param.tipe2`. Lihat tab.go.
//
// # ALIAS KOLOM DI LAYAR INI MENYESATKAN, DAN ITU HARUS DISADARI LEBIH DULU
//
// `GcnmSalvageData_CloseOs_SQL` mengalihnamakan seluruh kolomnya menjadi nama properti
// klaim yang sudah ada — utang teknis §4.2 pada `03-CURRENT-ARCHITECTURE.md`. Tidak satu
// pun aliasnya menyatakan isinya:
//
//	NOKLAIM          -> "CaseID"        SALVAGEID       -> "ClaimNo"
//	TGLINPUT         -> "DateOfLoss"    NOAKSEPTASI     -> "NIK"
//	JENISSALVAGE     -> "NewEmail"      REMARK          -> "AlasanKlaim"
//	QUANTITYSALVAGE  -> "KomiteCount"   STSTRANSFER     -> "Password"
//	ESTIMASINILAI    -> "District"      PIC             -> "UserTeknis"
//	LOKASISALVAGE    -> "Location"      CASE NILAIAKSEP -> "TreatyName"
//
// `D-19` melarang membawanya. Nama di berkas ini karena itu menyebut ISINYA, dan
// penelusuran balik ke export ditempuh lewat tabel dan kolom sebenarnya — disebut pada
// setiap isian di bawah.
//
// Dua yang paling mudah menjebak: `"DateOfLoss"` pada keluarga C berarti **tanggal input
// salvage**, sementara pada keluarga A dan B alias yang SAMA berarti **tanggal kejadian**.
// Dan `"ClaimNo"` pada keluarga C berarti **ID salvage**, bukan nomor klaim — nomor
// klaimnya justru beralias `"CaseID"`.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxsalvage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

// Row adalah satu baris pada grid mana pun di layar ini.
//
// # Kenapa SATU tipe untuk tiga keluarga yang kolomnya berbeda
//
// Karena yang menentukan kolom mana yang digambar adalah Tab.Columns, bukan tipe barisnya.
// Membuat tiga tipe akan memaksa tiga jalur di seam Repo, tiga penyusun DTO, dan tiga
// penggambar di layar — sementara yang benar-benar berbeda hanyalah daftar kolomnya.
//
// Preseden yang sama sudah ada di modul Inbox Manager Receive / PUCL, tempat sembilan dari
// enam belas isian hanya berlaku pada salah satu tab.
//
// Isian yang TIDAK diisi keluarga tertentu bernilai kosong, dan itu tidak pernah terlihat
// pengguna: kolomnya memang tidak digambar pada tab itu. Setiap isian di bawah menyebut
// keluarga mana yang mengisinya.
type Row struct {
	// Reference adalah kunci baris untuk tautan dan aksi.
	//
	// Isinya BERBEDA menurut keluarga, dan perbedaannya bukan kelalaian:
	//
	//	keluarga A, B  nomor klaim   — belum ada baris salvage sama sekali
	//	keluarga C     ID salvah     — `PNC_SALVAGE.IDSALVAGE`
	//
	// Keluarga A dan B membaca `T_CLAIM_PNC`, dan klaim di sana BELUM punya baris salvage.
	// Memaksakan ID salvage sebagai kunci di sana berarti kunci yang selalu kosong.
	Reference string

	// ClaimNo — kolom **"No Klaim"** / **"Nomor Klaim"**.
	//
	//	keluarga A, B  `POOLDATA.T_CLAIM_PNC.CLAIMNO`
	//	keluarga C     `POOLDATA.PNC_SALVAGE.NOKLAIM`  (alias menyesatkan "CaseID")
	ClaimNo string

	// SalvageID — `PNC_SALVAGE.IDSALVAGE`, keluarga C saja.
	//
	// Ia TIDAK digambar sebagai kolom di grid mana pun; yang memakainya adalah tombol
	// "Detail Salvage" dan agregat detail per baris.
	//
	// Aliasnya di kueri lama `"ClaimNo"` — nama yang justru menunjuk hal lain. Lihat
	// catatan alias di kepala berkas ini.
	SalvageID string

	// InputDate — kolom **"Tanggal Input"** <- `PNC_SALVAGE.TGLINPUT`. Keluarga C.
	//
	// Aliasnya di kueri lama `"DateOfLoss"`, yang berarti tanggal kejadian di seluruh
	// modul lain. Menyalin pemetaan itu akan menampilkan tanggal yang salah TANPA satu pun
	// galat.
	InputDate string

	// LossDate — kolom **"Tgl Kejadian"** <- `T_CLAIM_PNC.DATEOFLOSS`. Keluarga A.
	LossDate string

	// PIC — kolom **"PIC"** (keluarga C) dan **"User Input"** (keluarga B).
	//
	//	keluarga A, B  `T_CLAIM_PNC.PICTEKNIK`
	//	keluarga C     `PNC_SALVAGE.PIC`
	//
	// Keduanya beralias `"UserTeknis"` di kueri lama, dan kebetulan keduanya memang PIC
	// Teknik — ini satu-satunya alias pada layar ini yang menyatakan isinya.
	PIC string

	// BusinessName <- `T_CLAIM_PNC.BUSINESSNAME`. Keluarga A dan B.
	//
	// Judulnya BERBEDA di kedua keluarga meski kolomnya sama persis: **"COB"** pada daftar
	// Salvage Outstanding, **"Lokasi"** pada kelima daftar keluarga B. Keduanya dibawa apa
	// adanya (`D-13`).
	//
	// Judul "Lokasi" itu keliru di sistem lama — `BUSINESSNAME` adalah nama lini bisnis,
	// bukan lokasi. Ia tetap tidak diperbaiki: mengganti judul mengubah apa yang dibaca
	// pengguna, dan itu selisih yang belum diputuskan siapa pun.
	BusinessName string

	// ObjectName — kolom **"Object Name"** <- sub-kueri ke
	// `POOLDATA.T_CLAIM_OBJECTLIST.OBJECTNAME`, objek pertama saja. Keluarga B.
	//
	// Kueri lama memakai `fetch next 1 row only` tanpa `ORDER BY`, sehingga objek mana yang
	// terpilih TIDAK ditentukan saat sebuah klaim punya lebih dari satu objek. Perilakunya
	// dibawa apa adanya (`P-5`); lihat PlannedDifferences.
	ObjectName string

	// SalvageType — kolom **"Jenis Salvage"** <- `PNC_SALVAGE.JENISSALVAGE`. Keluarga C.
	// Aliasnya di kueri lama `"NewEmail"`.
	SalvageType string

	// SalvageLocation — kolom **"Lokasi"** <- `PNC_SALVAGE.LOKASISALVAGE`. Keluarga C.
	//
	// Berbeda dari BusinessName, INI benar-benar lokasi.
	SalvageLocation string

	// AuctionStatus — kolom **"Status Lelang"**. Keluarga C.
	//
	// Ia BUKAN kolom, melainkan hasil hitungan di dalam kueri:
	//
	//	NILAIAKSEP IS NOT NULL AND NILAIAKSEP != 0  -> "Terjual"
	//	selain itu                                  -> "Belum Terjual"
	//
	// Perhatikan ia tidak membaca `DETAIL_PNC_SALVAGE.STATUSTERJUAL`, yang punya lima
	// keadaan berbeda. Kedua sumber dapat berselisih, dan yang digambar di grid adalah yang
	// ini (`P-5`).
	AuctionStatus string

	// Quantity — `PNC_SALVAGE.QUANTITYSALVAGE`, beralias `"KomiteCount"`. Keluarga C.
	//
	// Diambil kueri tetapi TIDAK digambar di grid mana pun. Ia dibawa karena tombol
	// "Detail Salvage" memakainya.
	Quantity string

	// EstimateValue — kolom **"Nilai Pengajuan PIC"** <- `PNC_SALVAGE.ESTIMASINILAI`,
	// beralias `"District"`. Keluarga C.
	//
	// Judulnya "Estimasi" pada sebagian section dan "Nilai Pengajuan PIC" pada grid
	// checker. Keduanya kolom yang sama.
	EstimateValue string

	// Email — kolom **"Email"** <- `PNC_SALVAGE.EMAIL`. Keluarga C.
	Email string

	// Remark — kolom **"Keterangan PIC"** dan **"Remark"** <- `PNC_SALVAGE.REMARK`,
	// beralias `"AlasanKlaim"`. Keluarga C.
	Remark string

	// AcceptanceNo — kolom **"No Akseptasi"** <- `PNC_SALVAGE.NOAKSEPTASI`, beralias
	// `"NIK"`. Keluarga C.
	AcceptanceNo string

	// TransferStatus <- `PNC_SALVAGE.STSTRANSFER`, beralias `"Password"`. Keluarga C.
	//
	// Inilah kolom yang MEMISAHKAN ketujuh daftar keluarga C satu sama lain. Ia tidak
	// digambar sebagai kolom — yang digambar adalah tabnya sendiri.
	TransferStatus string

	// RequestValue — kolom **"Nilai Request Balai Lelang"**. Keluarga C, tiga daftar saja.
	//
	// Sumbernya BUKAN `PNC_SALVAGE` melainkan agregat atas `DETAIL_PNC_SALVAGE`:
	// `max(NILAI_REQUEST)` untuk pasangan `NOKLAIM` + `IDSALVAGE` yang sama, dengan
	// penyaring `statusterjual is null or statusterjual = '3'`.
	//
	// Aliasnya di kueri lama `"ClaimAmountAdjust"`.
	RequestValue string

	// RequestNote adalah `max(NOTE_REQUEST)` dari agregat yang sama.
	//
	// Ia TIDAK digambar sebagai kolom. Yang memakainya adalah SubmissionType di bawah —
	// terisi atau tidaknya isian inilah yang menentukan tipe pengajuan.
	RequestNote string

	// SubmissionType — kolom **"Tipe Pengajuan"**. Keluarga C, dua daftar saja.
	//
	// Ia hasil hitungan, bukan kolom:
	//
	//	NOTE_REQUEST kosong      -> "Pengajuan Baru"
	//	NOTE_REQUEST terisi      -> "Request Balai Lelang"
	//
	// Terbaca dari `Activity/PNCSalvageHistorySemuaKlaim_checker-Act.xml` langkah 2.4, yang
	// menuliskannya ke `.AlasanDokterRejectRCL` — alias yang tidak ada hubungannya sama
	// sekali dengan isinya.
	SubmissionType string

	// Aging — kolom **"Aging"**, berbentuk `"N day"`. Keluarga C, empat daftar.
	//
	// Ia selisih HARI KERJA antara tanggal input salvage dan hari ini, dihitung
	// `GCNMTimeDifferenceWorkCalender_Act` di sistem lama. `D-50` menetapkan perhitungan
	// jam kerja dan kalender libur ditulis ulang di Go karena ia aturan bisnis, bukan
	// pengambilan data.
	//
	// Sampai kalender libur menjadi master data (`F-4`), isian ini berisi selisih hari
	// kalender. Selisihnya dinyatakan di PlannedDifferences, bukan disamarkan.
	Aging string

	// Note — kolom **"Catatan"**. Keluarga C, tiga daftar.
	//
	// SELALU KOSONG, dan itu bukan cacat modul ini melainkan keadaan di Pega. Section
	// menggambar `.ResponseNote`, dan penelusuran ke seluruh jalur pemuat daftar tidak
	// menemukan satu pun penulisnya:
	//
	//   - `GcnmSalvageData_CloseOs_SQL` tidak mengambilnya.
	//   - `PNCSalvageGetChekerDataKlaimAllData` tidak mengambilnya.
	//   - `PNCSalvageHistorySemuaKlaim_checker` tidak menulisnya.
	//
	// Alias `"ResponseNote"` MEMANG ada di export — pada `GetDataSalvagefromPNC_salvage`,
	// kueri tombol Export Data — tetapi di sana ia menunjuk `b.pic`, hal yang sama sekali
	// berbeda. Menyambungkan keduanya akan mengisi kolom "Catatan" dengan nama PIC.
	//
	// Kolomnya tetap digambar karena layar lama menggambarnya (`D-13`), dan kekosongannya
	// dinyatakan di PlannedDifferences.
	Note string
}

// Caller adalah identitas pemanggil, diturunkan dari sesi.
//
// # Kenapa ia dibawa meski tidak satu pun kueri daftar menyaringnya
//
// Karena satu daftar MEMANG menyaringnya, dan itu mudah terlewat: daftar "Request Balai
// Lelang" menyaring `PIC = <pemanggil>` — `Activity/SetDataSalavage_act-Act.xml` langkah 23
// menyusun `and a.STSTRANSFER ='7' AND PIC='"+TempClaimAttach.UserAdmin+"'`, dan
// `TempClaimAttach.UserAdmin` berisi kode cabang pemanggil hasil `GetOperatorID`.
//
// Di luar itu ia dipakai jejak log. Seluruh baris memuat nomor klaim dan nilai uang, dan
// selama pemeriksaan peran belum ada (`TKT-F3-004`), catatan siapa yang membukanya adalah
// satu-satunya kontrol yang tersisa (`D-59`).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama. Bukan NIK.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Lini bisnis yang ditangani layar ini.
//
// Ketiga kueri daftar DAN seluruh kueri pencacah menyaring dengan pasangan yang sama, dan
// pasangan itu muncul kata demi kata di sepuluh rule berbeda:
//
//	GROUPPANEL IN ('003','004','006','009')
//	BUSINESSCODE NOT IN ('10145','10168')
//
// `CONTEXT.md` menerjemahkan Group Panel-nya: `003` dan `009` Aneka, `004` Marine Cargo,
// `006` Fire/Property. Perhatikan `002` Personal Accident dan `005` Travel TIDAK termasuk —
// salvage memang tidak berlaku pada klaim orang.
//
// Arti kedua `BUSINESSCODE` yang dikecualikan TIDAK diketahui: tidak ada master yang
// menerjemahkannya di export mana pun. Keduanya dibawa apa adanya sebagai angka (`P-5`).
var (
	GroupPanels     = []string{"003", "004", "006", "009"}
	ExcludedBizCode = []string{"10145", "10168"}
)

// Status kerja klaim yang DIKELUARKAN daftar Salvage Outstanding.
//
// Hanya keluarga A yang memakainya — keluarga B dan C tidak menyaring status kerja sama
// sekali, sehingga keduanya memuat klaim yang sudah selesai. Itu perilaku sistem lama apa
// adanya.
var ClosedWorkStatuses = []string{"Resolved-Completed", "Resolved-Rejected"}

// Penanda salvage yang DIMUAT daftar Salvage Outstanding.
//
// # Kenapa nilainya diambil dari layar Pega, bukan dari export
//
// Export membaca sebaliknya: `Activity/GCNMInboxSalvage_act-Act.xml` menyusun penyaring
// `(3 atau 5)` pada satu langkah, lalu MENIMPANYA dengan `STSSALVAGE IS NULL` pada langkah
// berikutnya — keduanya berprasyarat sama — sebelum kuerinya dijalankan. Dibaca apa adanya,
// daftar ini memuat klaim yang BELUM ditandai sama sekali.
//
// Pengukuran terhadap layar Pega sungguhan pada 2026-10-08 membantahnya:
//
//	daftar Outstanding di Pega           145 baris
//	"3 atau 5" + masih terbuka           145 baris   <- cocok
//	"belum ditandai" + masih terbuka     465 baris
//
// Work Owner memastikannya sekali lagi dari arah yang berbeda: satu klaim yang tampil di
// daftar Outstanding Pega juga tampil di daftar TBA — dan TBA menyaring `STSSALVAGE='5'`.
// Klaim ber-penanda tidak akan pernah lolos penyaring "belum ditandai".
//
// Kesimpulannya export untuk layar ini SUDAH BASI — persis yang diperingatkan `R-09`:
// 124 activity berubah pada 2026 dan rule-nya terus bergerak selama migrasi berjalan.
// Yang dipegang adalah perilaku sistem berjalan, bukan berkas XML-nya.
var OutstandingSalvageStatuses = []string{"3", "5"}

// Pagination adalah permintaan satu halaman.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// DefaultPageSize 20 BUKAN kebiasaan modul lain melainkan angka layar ini sendiri:
// `<pyPageSize>20</pyPageSize>` pada `Section/InboxSalvage-Section.xml`, dan muncul 12 kali
// dengan nilai yang sama pada `Section/InboxSalvageASM-Section.xml`. Modul inbox lain
// memakai 50; perbedaannya tidak diseragamkan.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipenuhi diam-diam.
const (
	DefaultPageSize = 20
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
	Items []Row

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan.
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
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat kepala inboxsalvage.sql.
func Slice(all []Row, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []Row{}}

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

// StatusCount adalah satu baris tabel ringkas **"Status Salvage / Jumlah"** di atas grid.
//
// # Ia BUKAN sekadar hiasan — ia navigasi
//
// `Activity/GCNMCountSalvage_act-Act.xml` menulis TIGA isian per baris, dan dua di antaranya
// adalah kode:
//
//	.CauseOfLoss  label yang dibaca pengguna       -> Label
//	.CityID       Param.tipe daftar yang dituju    -> Tab
//	.RW           Param.tipe2 daftar yang dituju   -> Tab
//	.City         angkanya                         -> Total
//
// Artinya mengklik satu baris pencacah MEMBUKA daftar yang bersangkutan. Tanpa membawa
// kedua kode itu, tabel ringkasnya menjadi angka yang tidak dapat ditindaklanjuti.
type StatusCount struct {
	// Label — kolom **"Status Salvage"**, teks layar lama apa adanya (`D-13`).
	Label string

	// Tab adalah kode tab yang dibuka bila barisnya diklik. Kosong berarti barisnya tidak
	// menuju daftar mana pun.
	Tab string

	// Total — kolom **"Jumlah"**.
	Total int
}

// ErrRowNotFound berarti baris yang diminta tidak ada.
var ErrRowNotFound = errors.New("inboxsalvage: baris salvage tidak ditemukan")

// Penolakan unggahan dokumen.
//
// Keduanya dipisah dari galat validasi per berkas karena keduanya menyangkut PERMINTAAN,
// bukan isi satu berkas — dan transport memetakannya ke kode HTTP yang berbeda.
var (
	ErrNoDocument       = errors.New("inboxsalvage: tidak ada berkas yang diunggah")
	ErrTooManyDocuments = errors.New("inboxsalvage: berkas melebihi batas satu unggahan")
	ErrDocumentTooLarge = errors.New("inboxsalvage: ukuran berkas melebihi batas")
)

// Detail adalah isi panel **"Detail Salvage"**, yang di Pega terbuka lewat tombol bernama
// sama pada grid.
//
// # Apa yang digantikan
//
//	Section/DataDetail_Salvage-Section.xml   lima belas isian di bawah
//	RDB List/GcnmSetSalvageData_SQL-SQL.xml  kueri kepalanya
//	RDB List/GetDetailSalvage-SQL.xml        kueri daftar barangnya
//
// # Kenapa ia tipe TERSENDIRI, bukan Row yang diperlebar
//
// Karena isinya memang berbeda, dan enam isian di sini TIDAK ADA di grid mana pun: tanggal
// transfer GA, tanggal akseptasi, mata uang, nama pemenang lelang, tanggal lelang, dan
// nilai penawaran. Memperlebar Row berarti menarik keenamnya pada setiap baris daftar —
// dua puluh kali per halaman — untuk sesuatu yang hanya dibaca saat satu baris dibuka.
type Detail struct {
	SalvageID string
	ClaimNo   string

	// HasSubmission menyatakan klaim ini benar-benar punya pengajuan salvage.
	//
	// Selalu benar bila panel dibuka dari baris PENGAJUAN. Dapat salah bila dibuka dari
	// baris KLAIM: enam daftar berbasis klaim menampilkan klaim yang salvage-nya belum
	// tentu pernah diajukan — daftar Salvage Outstanding bahkan menampilkan klaim yang
	// justru BELUM ditandai punya salvage sama sekali.
	//
	// Bila salah, seluruh isian yang berasal dari `PNC_SALVAGE` kosong dan hanya isian
	// klaimnya yang terisi. Layar menyatakannya sebagai kalimat, bukan sebagai panel
	// yang tampak gagal dimuat.
	HasSubmission bool

	// History adalah SELURUH pengajuan salvage milik klaim ini, terbaru lebih dulu.
	//
	// Ia digambar sebagai grid "Detail History Salvage" pada form "Menambahkan Data
	// Salvage" — bukan pada panel rincian. Isinya menjawab pertanyaan yang tidak dapat
	// dijawab daftar mana pun: klaim ini sudah pernah diajukan berapa kali, dan
	// masing-masing berakhir di mana.
	//
	// Kosong berarti klaim ini belum pernah diajukan salvage sama sekali.
	History []HistoryRow

	// ObjectChoices dan CoverageChoices adalah ISI KEDUA AUTOCOMPLETE pada form
	// "Menambahkan Data Salvage" — kolom "Nama Object" dan "Nama Coverage".
	//
	// # Kenapa keduanya menempel pada rincian KLAIM, bukan pada pengajuan
	//
	// Karena di layar lama pun begitu. Ketiga kolom teratas form itu bekerja sebagai satu
	// rangkaian: mengetik **Nomor Klaim** memicu `SetDataDetailSalvage_act` dengan
	// `tipe=1` dan `CaseeID` berisi nomor yang diketik
	// (`Section/TambahData_Salvage-Section.xml:2165-2190`), dan kedua autocomplete di
	// bawahnya membaca hasilnya. Satu nomor klaim, satu pemuatan, dua daftar pilihan.
	//
	// Keduanya karena itu dibaca pada jalur DetailByClaim — jalur yang parameternya sama
	// persis dengan `CaseeID` di sistem lama.
	//
	// # Activity pemasoknya
	//
	// `pyListPreActivity` kedua autocomplete itu adalah **`GetDataSalavageCovCurObj_act`**
	// (`:2973` untuk objek, `:3832` untuk coverage). Berkasnya diterima 2026-10-03, dan
	// isinya empat langkah: satu Property-Set yang menyusun kunci klaim, lalu tiga
	// RDB-List. Dua di antaranya mengisi kedua daftar di bawah; yang ketiga mengisi
	// `TempCurrencySalvage` dan TIDAK dipakai form ini — lihat catatan pada
	// `claim_coverages` di inboxsalvage.sql.
	//
	// Kedua kuerinya kini disalin apa adanya, bukan direkonstruksi lagi.
	//
	// # Kosong BUKAN galat
	//
	// Klaim yang objeknya belum terisi memang ada — modul Inbox Investigator mencatat
	// keadaan yang sama. Form tetap dapat diisi: kedua kolomnya menerima ketikan bebas,
	// persis seperti `pyAllowFreeFormInput=true` di layar lama.
	ObjectChoices   []ObjectChoice
	CoverageChoices []CoverageChoice

	// PIC dan LossDate berasal dari KLAIM, bukan dari pengajuan.
	//
	// Keduanya hanya terisi bila panel dibuka dari baris klaim — pada jalur itu keduanya
	// satu-satunya isian yang pasti ada, dan tanpanya panel klaim tanpa pengajuan akan
	// kosong seluruhnya.
	PIC      string
	LossDate string

	// BusinessName adalah lini bisnis klaimnya, diambil sub-kueri ke `T_CLAIM_PNC`.
	//
	// Ia satu-satunya isian panel ini yang TIDAK berasal dari `PNC_SALVAGE`.
	BusinessName string

	InputDate     string // "Tanggal Input Salvage" <- TGLINPUT
	SalvageType   string // "Jenis Salvage"         <- JENISSALVAGE
	Quantity      string // "Quantity Salvage"      <- QUANTITYSALVAGE
	EstimateValue string // "Estimasi"            <- ESTIMASINILAI
	Location      string // "Lokasi Salvage"        <- LOKASISALVAGE

	// TransferGADate — "Tanggal Transfer GA" <- TGLTRANSFERGA.
	//
	// Kosong pada pengajuan yang dibuat modul ini: kolomnya sengaja tidak diisi saat
	// menyimpan, karena ia menandai pengajuan yang sudah BENAR-BENAR dikirim ke bagian
	// umum — bukan yang baru dibuat. Lihat catatan pada insert_salvage.
	TransferGADate string

	// TransferStatus <- STSTRANSFER, dan Position adalah labelnya.
	//
	// Keduanya dibawa: kodenya untuk penelusuran, labelnya untuk dibaca. Label diturunkan
	// dari tab yang menyaring kode itu — lihat PositionLabelOf.
	TransferStatus string
	Position       string

	AcceptanceDate string // "Tanggal Akseptasi" <- TGLAKSEPTASI
	AcceptanceNo   string // "No Akseptasi"      <- NOAKSEPTASI
	Remark         string // "Remark"            <- REMARK
	Currency       string // "Mata Uang"         <- CURRENCY

	ObjectID     string
	ObjectName   string // "Nama Object"   <- OBJECTNAME
	CoverageID   string
	CoverageName string // "Nama Coverage" <- COVERAGENAME

	// AcceptedValue — "Nilai Salvage" <- NILAIAKSEP.
	//
	// Kolom yang sama menentukan "Status Lelang" pada grid. Di panel ini ia digambar
	// sebagai nilai, bukan sebagai status.
	AcceptedValue string

	Email       string
	OfferValue  string // NILAIPENAWARAN
	WinnerName  string // PEMENANGNAME
	AuctionDate string // TANGGALLELANG

	SurveyorName  string // PICSURVEY
	SurveyorPhone string // NOTELP
	SurveyorEmail string // EMAILSURVEY

	InJabodetabek bool // ISJABODATABEK

	// LegacyBeforeJuly2023 menandai pengajuan yang dibuat SEBELUM 17 Juli 2023.
	//
	// Kueri lama menghitungnya dengan `case when trunc(b.tglinput) < to_date('17/07/2023')`.
	// Apa yang dibedakannya tidak terbaca dari export mana pun — tidak ada rule yang
	// memakainya selain menggambarnya. Ia dibawa apa adanya, dan artinya ditanyakan bila
	// kelak ada yang membutuhkannya.
	LegacyBeforeJuly2023 bool

	// Items adalah daftar barang pada pengajuan ini.
	Items []DetailBarang
}

// DetailBarang adalah satu baris grid barang pada panel Detail Salvage.
//
// Barisnya DIKELOMPOKKAN menurut nama barang dan satuan — `GetDetailSalvage` memakai
// `GROUP BY`, sehingga dua baris `DETAIL_PNC_SALVAGE` bernama sama menyatu menjadi satu
// baris berjumlah dua. Itu perilaku layar lama apa adanya.
type DetailBarang struct {
	// Name — "Nama Barang" <- NAMABARANG.
	Name string

	// Quantity adalah teks berbentuk `"<jumlah> <satuan>"`, hasil
	// `count(namabarang) || ' ' || satuan` pada kueri lama.
	//
	// Ia dirangkai DI BASIS DATA di sistem lama; di sini keduanya diambil terpisah lalu
	// dirangkai di Go, supaya jumlah dan satuannya tetap dapat dibaca sendiri-sendiri.
	Count int
	Unit  string

	// TotalValue adalah `sum(HARGAITEM)` — jumlah kolom yang namanya menyebut HARGA tetapi
	// isinya JUMLAH ITEM. Lihat catatan pada DetailItem.Quantity.
	TotalValue string

	// SoldStatus adalah label `STATUSTERJUAL`.
	SoldStatus string

	WinnerName    string // PEMENANGSALVAGE
	AcceptanceNo  string // NOAKSEPTASI
	AcceptedValue string // NILAIAKSEPTASI
	Remark        string // REMARK
}

// ObjectChoice adalah satu pilihan pada autocomplete **"Nama Object"**.
//
// Di layar lama ia satu baris `TempObjectData.pxResults`: yang DIBACA pengguna adalah
// `.City`, dan yang ikut tersimpan diam-diam saat barisnya dipilih adalah `.CaseID` —
// disalin ke `TempInsert.NewNoKTP` lewat `pyAdditionalFields` ber-`pyShow=false`
// (`Section/TambahData_Salvage-Section.xml:2944-2948`).
//
// Kedua nama properti itu MENYESATKAN dan tidak dibawa: `.City` bukan kota dan `.CaseID`
// bukan nomor kasus. Isian di bawah menyebut isinya (`D-19`).
type ObjectChoice struct {
	// ID <- `POOLDATA.T_CLAIM_OBJECTLIST.OBJECTID`.
	//
	// Inilah yang berakhir di `PNC_SALVAGE.IDOBJECT`, parameter `tIDOBJ` pada
	// `INSERT_SALVAGE`. Tanpanya pengajuan tersimpan dengan nama objek tetapi tanpa
	// penunjuk ke objeknya — persis keadaan modul ini sebelum perbaikan ini.
	ID string

	// Name <- `OBJECTNAME`, satu-satunya isian yang dibaca pengguna.
	Name string
}

// CoverageChoice adalah satu pilihan pada autocomplete **"Nama Coverage"**.
//
// Bentuknya sama persis dengan ObjectChoice, dan itu bukan kebetulan: kedua kuerinya di
// layar lama memang kembar — `COVERAGEID as "CaseID", COVERAGENAME as "City"` berbanding
// `OBJECTID as "CaseID", OBJECTNAME as "City"`.
type CoverageChoice struct {
	// ID <- `POOLDATA.T_CLAIM_OBJECTCOVERAGE.COVERAGEID`.
	//
	// # Catatan koreksi: ia KODE JENIS JAMINAN, bukan penunjuk satu baris
	//
	// `COVERAGEID` berisi kode seperti `'10003'`, dan kode yang sama berulang di banyak
	// klaim — bahkan dapat berulang DI DALAM satu klaim bila dua objek punya jaminan yang
	// sama. Ia karena itu tidak menunjuk satu baris coverage tertentu.
	//
	// Sempat disimpulkan bahwa yang dipakai adalah `OBJECTCOVERAGEID`, karena tujuh kueri
	// PLA/DLA memperlakukan "IDCoverage" sebagai kolom itu
	// (`GetDataDLA-SQL.xml:49`, `GetDataPLA-SQL.xml:108`, `GetDataPreDLA-SQL.xml:47`).
	// Kesimpulan itu **salah untuk layar ini**:
	// `RDB List/GetDataSalvageObjectCoverageForOs-SQL.xml` menyebut `COVERAGEID` apa
	// adanya. Dua layar memakai kata yang sama untuk kolom yang berbeda, dan yang berlaku
	// di sini adalah kuerinya sendiri.
	//
	// Akibat yang harus disadari: `PNC_SALVAGE.IDCOVERAGE` karena itu TIDAK dapat dipakai
	// menunjuk baris coverage mana yang dimaksud bila klaimnya punya dua objek berjaminan
	// sama. Itu keadaan di sistem lama, bukan sesuatu yang modul ini perkenalkan.
	ID string

	// Name <- `COVERAGENAME`.
	Name string
}

// CurrencyOption adalah satu pilihan dropdown **"Mata Uang"** pada form Tambah.
//
// # Dari mana isinya
//
// Dropdown itu di layar lama bersumber Report Definition `SelectCurrency_RD`
// (`Section/TambahData_Salvage-Section.xml:5251`), dan RD itu membaca kelas
// `ASM-FW-GISFW-Int-CURRENCY` — tabel `POOLDATA.CURRENCY`. Kolom yang dibacanya
// terkonfirmasi dari tempat kedua: `RDB List/Gcnmgetdatacurrencysalvage_SQL-SQL.xml`
// menggabungkan `A.CURRENCY = B.ID` lalu mengambil `b.currency` sebagai teks yang dibaca
// orang.
//
// # Kode DAN labelnya sama, dan itu disengaja
//
// Yang tersimpan di `PNC_SALVAGE.CURRENCY` adalah teks mata uangnya ("IDR"), bukan
// `ID`-nya — panel rincian menggambar kolom itu apa adanya sebagai "Mata Uang", dan ia
// terbaca sebagai kode tiga huruf. Mengirim `ID` sebagai nilai akan menyimpan angka di
// kolom yang selama ini berisi huruf.
type CurrencyOption struct {
	// Code adalah nilai yang tersimpan DAN yang dibaca — `POOLDATA.CURRENCY.CURRENCY`.
	Code string

	// Label dibedakan dari Code supaya keduanya dapat berpisah kelak tanpa mengubah
	// kontrak — misalnya bila nama negaranya kelak ikut ditampilkan.
	Label string
}

// Batas unggahan dokumen, sebagaimana tertulis MERAH pada modal
// `Section/SalvageUploadDocumentAll-Section.xml`.
//
// Keduanya ditegakkan DUA KALI — di layar dan di sini. Pemeriksaan layar menghemat
// perjalanan kirim yang sudah pasti ditolak; pemeriksaan di sini yang benar-benar
// mengikat, karena layar dapat dilewati.
const (
	MaxDocumentPerUpload = 5
	MaxDocumentSizeBytes = 1024 * 1024
)

// DocumentCommand adalah parameter `tCOMMAND` kedua procedure lampiran.
//
// Hanya `"INSERT"` yang dipakai modul ini. Cabang satunya menghapus lampiran, dan
// penghapusan dokumen belum ditawarkan layar mana pun.
const DocumentCommand = "INSERT"

// DocumentUpload adalah satu berkas yang diunggah lewat modal "UploadDocument_Salvage".
//
// # Isinya base64, bukan byte mentah
//
// `GCNMUploadResult64` — langkah pertama `SaveFilePenunjangBySalvage` — mengubah berkas
// menjadi base64 sebelum apa pun terjadi, dan kolom `TEMP_DATA_ATTACHFILE.ATTACHFILE`
// menyimpannya dalam bentuk itu. Pengubahannya dikerjakan lapisan transport, supaya
// seam ini menerima bentuk yang benar-benar disimpan.
type DocumentUpload struct {
	// ClaimNo dan SalvageID menyatakan pengajuan mana yang dilampiri.
	//
	// SalvageID boleh kosong: pada form pengajuan BARU, pengajuannya memang belum punya
	// ID. Lampirannya tetap tersimpan terhadap klaimnya — penaut salvage yang dilewati.
	ClaimNo   string
	SalvageID string

	// FileName adalah nama ASLI berkas dari peramban.
	//
	// Bukan nama yang tersimpan: yang tersimpan disusun `SetFileNameSalvage` menjadi
	// `<awalan>-<nomor klaim>-<urutan>.<ekstensi>`. Nama asli tetap dibawa karena
	// ekstensinya diambil dari sini.
	FileName string

	// MimeType adalah EKSTENSI berkas, bukan media type HTTP.
	//
	// `SetFileNameSalvage` merangkainya langsung di belakang titik — `"." +
	// Param.FileMimeType` — sehingga yang diharapkannya `pdf`, bukan `application/pdf`.
	MimeType string

	// Content64 adalah isi berkas yang sudah di-base64.
	Content64 string

	// Category dan SubCategory adalah `DOC_TYPE_ID` dan `DOC_TYPE_DT_ID` pada
	// `LST_TYPE_DOC_BUSINESS`.
	//
	// Keduanya BOLEH kosong, dan pada modal salvage memang begitu: modalnya tidak punya
	// pemilih jenis dokumen sama sekali. Akibatnya awalan nama berkas tidak terbaca dan
	// nama asli yang dipakai — lihat Repo.AttachDocument.
	Category    string
	SubCategory string

	// Operator adalah pengunggahnya — `OperatorID.pyUserIdentifier` di sistem lama.
	Operator string
}

// Clean memangkas spasi setiap isian dan menormalkan ekstensinya.
func (d DocumentUpload) Clean() DocumentUpload {
	cleaned := DocumentUpload{
		ClaimNo:     strings.ToUpper(strings.TrimSpace(d.ClaimNo)),
		SalvageID:   strings.TrimSpace(d.SalvageID),
		FileName:    strings.TrimSpace(d.FileName),
		MimeType:    strings.ToLower(strings.TrimSpace(d.MimeType)),
		Content64:   strings.TrimSpace(d.Content64),
		Category:    strings.TrimSpace(d.Category),
		SubCategory: strings.TrimSpace(d.SubCategory),
		Operator:    strings.TrimSpace(d.Operator),
	}

	// Titik di depan ekstensi dibuang: `SetFileNameSalvage` sudah menambahkannya sendiri,
	// dan membiarkannya menghasilkan nama berakhiran `..pdf`.
	cleaned.MimeType = strings.TrimPrefix(cleaned.MimeType, ".")

	return cleaned
}

// Validate memeriksa satu berkas sebelum ia menyentuh basis data.
func (d DocumentUpload) Validate() error {
	switch {
	case d.ClaimNo == "":
		return errors.New("inboxsalvage: nomor klaim wajib diisi")
	case d.FileName == "":
		return errors.New("inboxsalvage: nama berkas wajib diisi")
	case d.Content64 == "":
		return errors.New("inboxsalvage: isi berkas kosong")
	}
	return nil
}

// AttachedDocument adalah satu dokumen yang BERHASIL tersimpan.
type AttachedDocument struct {
	// DataID adalah kunci baris `DATA_ATTACHFILE` — yang KETERANGANNYA, bukan isinya.
	//
	// Kedua tabel membangkitkan DATAID sendiri-sendiri, dan yang dipakai penaut salvage
	// adalah yang ini. Lihat catatan di kepala bagian unggahan pada inboxsalvage.sql.
	DataID string

	// ImageID menautkan keterangan dengan isinya.
	ImageID string

	// StoredName adalah nama berkas yang BENAR-BENAR tersimpan.
	StoredName string

	// LinkedToSalvage menyatakan baris penaut ke pengajuan salvage ikut ditulis.
	//
	// Salah bila pengajuannya belum punya ID — pada form pengajuan baru, lampirannya
	// menempel ke klaimnya saja.
	LinkedToSalvage bool
}

// NewImageID membangkitkan `IMAGEID` satu berkas lampiran.
//
// # Ia menggantikan kueri, bukan menirunya sembarangan
//
// `RDB List/GenerateimageID-SQL.xml` menyusunnya begini:
//
//	STANDARD_HASH('ASMPP' || TO_CHAR(SYSTIMESTAMP, 'DD/MM/YYYY HH24:MI:SS.FF3'), 'MD5')
//
// Bentuk yang dihasilkan di sini SAMA PERSIS — MD5 atas teks yang sama, dirender sebagai
// heksadesimal huruf besar, seperti yang Oracle lakukan terhadap RAW.
//
// # Kenapa pindah ke Go
//
// Karena `STANDARD_HASH` dan `SYSTIMESTAMP` keduanya khas Oracle, sementara `D-20`
// menuntut satu set SQL yang berjalan di Oracle DAN PostgreSQL. Dan karena pembangkit yang
// hidup di Go dapat diuji tanpa basis data — termasuk dibuktikan bahwa dua berkas yang
// diunggah pada milidetik berbeda menerima ID yang berbeda.
//
// # Yang dibawa serta, termasuk kelemahannya
//
// ID ini bergantung pada WAKTU saja. Dua berkas yang diunggah pada milidetik yang sama
// menerima ID yang sama, dan karena `IMAGEID` yang menautkan isi berkas dengan
// keterangannya, tabrakan itu menautkan keduanya secara silang. Itu keadaan sistem lama
// apa adanya; memperbaikinya mengubah bentuk ID, dan bentuknya dibaca sistem lain.
func NewImageID(now time.Time) string {
	stamp := now.Format("02/01/2006 15:04:05.000")
	sum := md5.Sum([]byte("ASMPP" + stamp))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// SafeFileName membuang karakter yang tidak aman dari sebuah nama berkas.
//
// Aturannya disalin dari `SaveFilePenunjangBySalvage`, yang menempuh dua langkah:
// membuang SPASI, lalu membuang segala yang bukan huruf, angka, titik, atau tanda hubung.
//
//	@pxReplaceAllViaRegex(@replaceAll(nama," ",""), "[^a-zA-Z0-9 .-]+", "")
//
// Ia dipakai saat awalan dari master jenis dokumen tidak terbaca — lihat storedNameOf.
func SafeFileName(name string) string {
	var builder strings.Builder
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z',
			char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9',
			char == '.', char == '-':
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

// PositionLabelOf menerjemahkan `STSTRANSFER` menjadi teks "Posisi Salvage".
//
// # Dari mana labelnya
//
// Dari TAB yang menyaring kode itu — bukan dari master, karena tidak ada master yang
// menerjemahkan `STSTRANSFER` di export mana pun. Dengan begitu label di panel detail dan
// nama daftar tempat barisnya muncul selalu menyebut hal yang sama, dan keduanya tidak
// dapat berselisih.
//
// Kode yang tidak punya tab dikembalikan APA ADANYA, bukan diganti teks kosong maupun
// "tidak dikenal". Satu kode memang begitu — `6`, yang ikut terhitung pencacah "Histori
// Salvage" tetapi tidak punya daftar sendiri. Menyembunyikannya akan membuat panelnya
// tampak kehilangan data.
func PositionLabelOf(status string) string {
	clean := strings.TrimSpace(status)
	if clean == "" {
		return ""
	}
	for _, tab := range tabs {
		if tab.TransferStatus == clean {
			return tab.Name
		}
	}
	return clean
}

// Teks kolom "Status Terjual" pada grid barang.
//
// Keempatnya DISALIN HARFIAH dari `RDB List/GetDetailSalvage-SQL.xml`.
//
// # Peringatan: kolom yang sama dipetakan BERBEDA di kueri lain
//
// `GetDataSalvagefromPNC_salvage` — kueri tombol Export Data — memetakan kolom yang sama
// menjadi LIMA keadaan, dan salah satunya bertabrakan:
//
//	kode   panel Detail Salvage   kueri ekspor
//	----   --------------------   -----------------------
//	'1'    Terjual                Terjual
//	'0'    Tidak terjual          Tidak terjual
//	'2'    (tidak dipetakan)      Waiting approval waive
//	'3'    Waiting approval       Rejected waive
//	NULL   Belum terjual          Belum terjual
//
// Kode `'3'` berarti hal yang BERLAWANAN di kedua tempat. Yang dipakai di sini adalah
// pemetaan panel detail, karena itulah kueri panel ini. Selisihnya dinyatakan.
func SoldStatusOf(code string) string {
	switch strings.TrimSpace(code) {
	case "1":
		return "Terjual"
	case "0":
		return "Tidak terjual"
	case "3":
		return "Waiting approval"
	case "":
		return "Belum terjual"
	default:
		// `ELSE '-'` pada kueri lama. Kode `'2'` jatuh ke sini, dan itu memang yang
		// terjadi di Pega.
		return "-"
	}
}

// Repo adalah seam ke penyimpanan.
//
// # Kenapa ia MENULIS, berbeda dari modul inbox lain
//
// Karena Work Owner memutuskannya pada 2026-09-25: "Tambah" menjalankan
// `CNMShowInsertSalvage_dt` dan section `TambahData_Salvage`, dan Submit "dibangun di sesi
// section TambahData_Salvage".
//
// Yang ditulis adalah `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE`. Keduanya
// dimiliki modul ini selama masa paralel — tidak ada layar Pega lain yang menulisinya, dan
// `P-1` karena itu terpenuhi.
//
// Empat langkah `SetStsSalvagePNC_act` yang TIDAK ditulis di seam ini, dan ketiadaannya
// disengaja — lihat PlannedDifferences:
//
//	langkah 14  UploadDocumentToGoogleStorage   penyimpanan berkas eksternal
//	langkah 19  Insert_salvageToSimasBid        sistem luar balai lelang
//	langkah 23  UploadingFileUntukSendByEmail   kirim email
//	langkah 26  procedure penyisip JSON         sinkronisasi JSON_KLAIM
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// Counts mengembalikan tabel ringkas "Status Salvage / Jumlah".
	//
	// Ia terpisah dari List karena memang kueri yang berbeda — sepuluh kueri di sistem
	// lama, satu di sini. Lihat CountsQuery.
	Counts(ctx context.Context, caller Caller) ([]StatusCount, error)

	// Detail mengembalikan isi panel "Detail Salvage" untuk satu pengajuan.
	//
	// Kuncinya ID salvage — parameter yang sama dengan `TempInsert.Password` di sistem
	// lama.
	//
	// ID yang tidak ditemukan menghasilkan ErrRowNotFound, BUKAN nilai kosong: pengajuan
	// yang tidak ada dan pengajuan yang seluruh isiannya kosong terlihat sama di layar,
	// dan hanya yang pertama yang merupakan kekeliruan.
	Detail(ctx context.Context, salvageID string) (Detail, error)

	// DetailByClaim membaca rincian lewat NOMOR KLAIM, bukan lewat ID pengajuan.
	//
	// Dibutuhkan keenam daftar berbasis klaim: barisnya tidak membawa ID pengajuan sama
	// sekali — kueri yang memasoknya (`GcnmSalvageData_OS_SQL` dan
	// `GcnmSalvageData_ekonomisdanTba`) membaca `T_CLAIM_PNC` dan tidak mengambil satu
	// pun kolom dari `PNC_SALVAGE`.
	//
	// Pengajuan yang dipilih adalah yang TERAKHIR milik klaim itu, mengikuti
	// `SetDataDetailSalvage_act` langkah 18 pada jalur `param.tipe == 1`.
	//
	// Klaim tanpa pengajuan BUKAN galat: ia menghasilkan Detail ber-HasSubmission salah,
	// bukan ErrRowNotFound. Yang menghasilkan ErrRowNotFound hanyalah nomor klaim yang
	// tidak ada.
	DetailByClaim(ctx context.Context, claimNo string) (Detail, error)

	// Currencies mengembalikan pilihan dropdown "Mata Uang".
	//
	// Ia TIDAK bergantung pada klaim mana pun — isinya master, sama untuk seluruh layar.
	//
	// Kegagalannya diperlakukan pemanggil sebagai keadaan yang dapat ditoleransi, bukan
	// sebagai galat yang menutup layar: kolom Mata Uang tetap digambar, hanya tanpa isi.
	// Satu master yang tidak terbaca tidak boleh berubah menjadi layar yang tidak dapat
	// dipakai sama sekali.
	Currencies(ctx context.Context) ([]CurrencyOption, error)

	// AttachDocument menyimpan SATU berkas lampiran beserta penautnya.
	//
	// Ia menggantikan pasca-proses `SaveFilePenunjangBySalvage`, dan satu pemanggilan
	// menempuh enam langkah yang HARUS atomik: membangkitkan ID gambar, menyusun nama
	// berkas, menyimpan isinya, menyimpan keterangannya, mencatat histori, lalu
	// menautkannya ke pengajuan salvage.
	//
	// Kegagalan di tengah tidak boleh meninggalkan berkas tanpa keterangan — atau
	// sebaliknya. Di sistem lama tiap langkah `COMMIT` sendiri; di sini keenamnya satu
	// transaksi (`D-68`).
	AttachDocument(ctx context.Context, doc DocumentUpload) (AttachedDocument, error)

	// Create menyimpan satu pengajuan salvage beserta detail itemnya.
	//
	// Mengembalikan ID salvage yang terbit. Di sistem lama nilai itu keluar lewat parameter
	// `ErrMsg OUT` yang SEKALIGUS membawa pesan galat (`Database/INSERT_SALVAGE.prc:26`) —
	// kontrak yang `D-68` nyatakan tidak dibawa. Di sini keduanya dipisah: ID pada nilai
	// balik, kegagalan pada galat.
	Create(ctx context.Context, form Form) (string, error)

	// MarkSentToAuction mencatat JAWABAN balai lelang pada baris pengajuan.
	//
	// Dua kolom yang ditulisnya: `STSTRANSFER` — yang menentukan di daftar mana baris ini
	// muncul — dan `IDSIMASBID`, nomor pengajuan di sisi balai lelang.
	//
	// # Kenapa ia TERPISAH dari Create, bukan satu transaksi
	//
	// Karena `09-API-STRATEGY.md` §8.2 melarang pemanggilan sistem luar berada di dalam
	// transaksi basis data: kegagalan jaringan tidak boleh menahan kunci baris. Urutannya
	// karena itu simpan dulu, kirim kemudian, lalu catat jawabannya.
	//
	// Akibat yang diterima secara sadar: pengiriman yang berhasil tetapi pencatatannya
	// gagal meninggalkan baris yang `STSTRANSFER`-nya menyatakan "belum dikirim" padahal
	// balai lelang sudah menerimanya. Itu lebih baik daripada kebalikannya — baris yang
	// mengaku terkirim padahal tidak — karena yang pertama membuat petugas mengirim ulang,
	// sementara yang kedua membuatnya menunggu lelang yang tidak pernah terjadi.
	//
	// ID yang tidak ditemukan menghasilkan ErrRowNotFound.
	MarkSentToAuction(ctx context.Context, salvageID string, receipt AuctionReceipt) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas. Salvage milik Asuransi Sinar Mas dan
// salvage milik Simas Insurtech karena itu tidak pernah berada di tabel yang sama.
//
// Layar lama pun sudah memisahkannya, hanya dengan cara yang tidak terlihat: harness
// memilih section menurut `TempGetApp.LSC_ID != 'SIMASNET'`, dan `SetDataSalavage_act`
// langkah 4 memanggil `SetDataSalavage_act_ASI` untuk portal yang lain.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan salvage satu
// badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)
