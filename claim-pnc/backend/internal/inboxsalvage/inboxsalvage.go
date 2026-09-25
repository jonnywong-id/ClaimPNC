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
	"errors"
	"strings"
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

	// Create menyimpan satu pengajuan salvage beserta detail itemnya.
	//
	// Mengembalikan ID salvage yang terbit. Di sistem lama nilai itu keluar lewat parameter
	// `ErrMsg OUT` yang SEKALIGUS membawa pesan galat (`Database/INSERT_SALVAGE.prc:26`) —
	// kontrak yang `D-68` nyatakan tidak dibawa. Di sini keduanya dipisah: ID pada nilai
	// balik, kegagalan pada galat.
	Create(ctx context.Context, form Form) (string, error)
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
