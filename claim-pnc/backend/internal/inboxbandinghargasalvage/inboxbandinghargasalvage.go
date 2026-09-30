// Package inboxbandinghargasalvage adalah inti modul Inbox Banding Harga Salvage.
//
// # Layar apa ini
//
// Menu `MENU_ID 72` "Inbox Banding Harga Salvage" pada `POOLDATA.M_MENU_APLIKASI_PNC`
// (`Database/m_menu_aplikasi_pnc.csv:67`), yang menunjuk harness `InboxRequestSalvage`. Ia
// berada di kelompok menu `2` (Proses Produksi), urutan 1162 — tepat sesudah `MENU_ID 71`
// "Inbox Salvage".
//
// Isinya **antrean banding harga barang salvage**: balai lelang (SimasBid) menilai sebuah
// barang tidak layak dijual pada harga yang diajukan PIC, lalu mengajukan harga tandingan.
// Layar ini tempat komite ASM membacanya dan memutuskan.
//
// Per `D-79` ia benar-benar Inbox, bukan layar data acuan: barisnya pekerjaan, ia HILANG
// dari antrean begitu diputuskan (`TGLAPPROVE` terisi), "hanya milik saya" adalah aturan
// kewenangan (`NAMAKOMITE` = pemanggil), dan barisnya punya kolom Aging. Karena itu modul ini
// milik `U-3`, bukan `U-6`.
//
// # Bedakan dari Inbox Salvage — keduanya menyentuh salvage, tetapi bukan layar yang sama
//
// `MENU_ID 71` "Inbox Salvage" mengelola pengajuan salvage dari awal sampai lelang: tiga
// belas daftar di atas `PNC_SALVAGE` dan `DETAIL_PNC_SALVAGE`. Layar INI mengerjakan satu
// hal saja — banding harga — dan tabel intinya pun berbeda:
// `POOLDATA.T_CLAIM_CHEKER_SALVAGE`.
//
// Keduanya sempat hampir disatukan, dan `frontend/src/app/menu/registry.ts` mencatat
// peringatannya: menunjuk kedua butir menu ke satu rute akan menyatukan dua layar yang di
// Pega memang terpisah.
//
// # Dari mana banding harga itu datang
//
// Bukan dari petugas ASM. `Activity/PengajuanRequestBalaiLelang-Act.xml` adalah layanan yang
// dipanggil SimasBid dari luar, dan ia menempuh lima langkah berurutan:
//
//	RDB List/GetobjectdetailSalvagePerobject-SQL.xml  ambil rincian barangnya
//	RDB List/UpdateRequestSalvage-SQL.xml             tulis NOTE_REQUEST + NILAI_REQUEST
//	RDB List/UpdateStatusRequestSalvage-SQL.xml       PNC_SALVAGE.STSTRANSFER := '7'
//	RDB List/InsertHistoryKomunikasiSalvage-SQL.xml   catat MESSAGE_SIMASBID
//	Call ASMSendsEmailAttachments                     beri tahu PIC lewat surel
//
// Jadi layar ini adalah sisi PENERIMA percakapan itu. Balasan ke SimasBid (`MESSAGE_PNC`
// pada `HISTORY_KOMUNIKASI_SALVAGE`) belum dapat dibangun — lihat Limitations.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxRequestSalvage-Harness.xml        pembungkus layar; judul; tombol Tambah
//	Section/InboxReqSalvageASM-Section.xml         DUA grid, kolomnya, dan kotak Cari
//	Activity/SetReqSalvage_Act-Act.xml             pemasok kedua grid (param `tipe` 1/2)
//	Activity/GCNMCountRequestSalvage_act-Act.xml   tabel ringkas "Status Salvage / Jumlah"
//	RDB List/DataReqSalvage_SQL-SQL.xml            kueri grid Request Banding Harga
//	RDB List/HistoryReqSalvage_SQL-SQL.xml         kueri grid History Cheker
//	RDB List/CountRequestSalvage_Sql-SQL.xml       pencacah grid Request
//	RDB List/CountHistoryReqSalvage_Sql-SQL.xml    pencacah grid History
//	Navigation/pyCaseWorkerNavigation-Navigation.xml  butir menunya, penjaga `IsGCNMUser`
//
// # Artefak yang MASIH kurang, dan apa yang tertahan karenanya
//
// Artefaknya datang dalam LIMA putaran, dan tiap berkas yang tiba menunjuk berkas
// berikutnya. Sepuluh sudah diterima; enam masih kurang, dan seluruhnya rule TERDALAM —
// yakni yang benar-benar menyentuh basis data:
//
//	RDB List/UpdateDataReqSalvage          bagian tulis Approve / Reject
//	RDB List/UpdateDokReqSalvage           idem, jalur Reject
//	RDB List/UpdateHargaSalvage            idem, harga pada detail salvage
//	Connect REST/SendData_SalvageSimasBid  mengirim keputusan KEMBALI ke balai lelang
//	RDB List/DetailHistReqSalvage_SQL      isi panel rincian History
//	DokumenBandingSalvage                  local action tombol "Lihat File"
//
// Ditambah DDL `POOLDATA.T_CLAIM_CHEKER_SALVAGE` dari DBA — tipe kolom uang dan tanggalnya.
//
// Seluruhnya dinyatakan lewat Limitations, bukan ditambal dengan tebakan.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxbandinghargasalvage/          aturan modul + seam          ← paket ini
//	inboxbandinghargasalvage/usecase/  orkestrasi: bentuk layar, isi satu tab, ringkasan
//	inboxbandinghargasalvage/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxbandinghargasalvage/http/     lapisan transport modul ini  — handler, dto, rute
package inboxbandinghargasalvage

import (
	"context"
	"strings"
	"time"
)

// AppealRow adalah satu baris pada layar Inbox Banding Harga Salvage.
//
// # Kenapa SATU tipe untuk DUA grid yang tabelnya berbeda
//
// Karena yang dibaca layar adalah "baris beserta kolom mana yang digambar", dan kolomnya
// sudah dinyatakan Tab.Columns. Memecahnya menjadi dua tipe berarti dua bentuk JSON, dua
// jalur DTO, dan dua komponen tabel — sementara yang berbeda hanyalah isian mana yang terisi.
//
// Isian yang tidak berlaku pada sebuah tab dibiarkan kosong, dan itu tidak pernah terlihat
// pengguna: kolom yang tidak ada di Tab.Columns tidak digambar sama sekali.
//
// # Alias Pega TIDAK dibawa
//
// Di layar ini aliasnya nyaris seluruhnya menyesatkan — `.Email` berisi HARGA, `.AgentID`
// berisi harga lain lagi, `.BranchID` berisi catatan. Menyalinnya akan menampilkan kolom yang
// keliru tanpa satu pun galat (`D-19`). Pemetaan lengkapnya ada di kepala berkas
// repo/sqlstore/inboxbandinghargasalvage.sql.
type AppealRow struct {
	// ClaimNo adalah No Klaim — `NOKLAIM`. Satu-satunya isian yang digambar KEDUA grid.
	ClaimNo string

	// ---------------------------------------------------------------- grid Request

	// RequestDate adalah Tanggal Request — `TGLREQUEST`, saat balai lelang mengajukan
	// harga tandingannya.
	//
	// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari
	// "belum diisi" saat ditampilkan.
	RequestDate *time.Time

	// DetailObject adalah Detail Object — `IDDETAILSALVAGE`.
	//
	// Judul kolomnya menjanjikan rincian objek, tetapi isinya ID baris detail salvage.
	// Judul itu dipertahankan apa adanya (`D-13`); yang tidak dipertahankan adalah
	// menyembunyikan kenyataannya dari pembaca kode.
	DetailObject string

	// ItemName adalah Nama Barang — `NAMABARANG`.
	ItemName string

	// ItemPrice adalah Harga Barang — `HARGABARANG`, harga yang diajukan PIC.
	//
	// Teks, bukan angka pecahan. Nilai uang disimpan presisi penuh dan pembulatan hanya
	// terjadi saat ditampilkan (`I-12`); memindainya ke float64 membuang presisi sebelum
	// ia sempat sampai ke layar.
	ItemPrice string

	// RequestPrice adalah Harga Request — `HARGAREQUEST`, harga tandingan balai lelang.
	//
	// Inilah angka yang menjadi pokok seluruh layar ini: selisihnya terhadap ItemPrice yang
	// harus diputuskan komite.
	RequestPrice string

	// RequestNote adalah Note Request — `ALASANREQUEST`, alasan balai lelang mengajukan
	// harga berbeda.
	RequestNote string

	// AgingDays adalah Aging — berapa HARI sebuah banding sudah menunggu keputusan.
	//
	// Angka, bukan teks. Sistem lama menyusunnya sebagai teks (`… || ' days'`) lalu
	// mengurutkannya sebagai teks pula, sehingga `'9 days'` didahulukan dari `'30 days'` —
	// justru baris yang paling lama menunggu yang tenggelam. Lihat PlannedDifferences.
	//
	// Nol bila RequestDate kosong. Ia tidak dapat tertukar dengan "baru masuk hari ini",
	// karena lapisan transport mengirim kolomnya KOSONG saat tanggalnya tidak ada.
	//
	// Tampilannya tetap `<n> days`, dibentuk lapisan transport.
	AgingDays int

	// CheckerNote adalah Note Checker — `NOTEAPPROVE`, catatan komite atas banding ini.
	//
	// Nama kolomnya BUKAN `NOTEKOMITE` meski propertinya di Pega bernama `.NoteKomite`;
	// `NOTEKOMITE` ada pada tabel lain (`T_CLAIM_KOMITE_LIST`). Lihat kepala berkas .sql.
	//
	// Di layar lama kolom ini SELALU KOSONG: headernya digambar tetapi kueri grid tidak
	// pernah memilih kolomnya. Lihat PlannedDifferences.
	CheckerNote string

	// SalvageID adalah `IDSALVAGE`. TIDAK digambar sebagai kolom.
	//
	// Ia ikut ditarik kuerinya karena tombol Approve/Reject membutuhkannya — pasangan
	// (NOKLAIM, IDSALVAGE) itulah yang dipakai `UpdateStatusRequestSalvage` menyetel
	// `STSTRANSFER`.
	SalvageID string

	// CommitteeName adalah `NAMAKOMITE`. TIDAK digambar sebagai kolom.
	//
	// Ia penyaring kepemilikan baris ini, dan dibawa supaya jejak audit maupun penelusuran
	// selisih dapat menyebut milik siapa barisnya tanpa menanyakannya ulang.
	CommitteeName string

	// ---------------------------------------------------------------- grid History

	// SalvageType adalah kolom berjudul **"Object Name"** — `PNC_SALVAGE.JENISSALVAGE`.
	//
	// Judulnya menyebut nama objek; isinya JENIS SALVAGE. Salah satu alias menyesatkan yang
	// paling mudah terbawa, karena keduanya sama-sama masuk akal di layar antrean salvage.
	SalvageType string

	// SalvageLocation adalah Lokasi Salvage — `PNC_SALVAGE.LOKASISALVAGE`.
	SalvageLocation string

	// PIC adalah PIC — `PNC_SALVAGE.PIC`, petugas yang mengajukan salvage-nya.
	//
	// Ia BUKAN pemilik antrean ini. Yang menentukan baris mana yang Anda lihat adalah
	// `NAMAKOMITE`, bukan kolom ini.
	PIC string
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Karena itu yang dipakai `Activity/SetReqSalvage_Act-Act.xml` langkah 2, yang menyusun
// halaman lewat `.FirstRow = ((.CurrentIndex-1) * .PageSize) + 1` dan
// `.LastRow = .CurrentIndex * .PageSize` pada kelas `ASM-FW-GISFW-Data-Pagination`. Ia tidak
// dikarang dan tidak disamakan dengan modul lain yang memakai 20.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxPageSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
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
	Items []AppealRow
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

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
	// di sistem lama (`Activity/SetReqSalvage_Act-Act.xml` langkah 7).
	//
	// Itulah yang dicocokkan ke `T_CLAIM_CHEKER_SALVAGE.NAMAKOMITE`. Memakai NIK di sini
	// akan membuat kedua tab tampak KOSONG bagi setiap pengguna — dan antrean kosong tidak
	// pernah dilaporkan siapa pun sebagai kerusakan.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Summary adalah tabel ringkas "Status Salvage / Jumlah" di kepala layar.
//
// Sistem lama menggambarnya sebagai grid DAN diagram sekaligus, keduanya dari halaman
// `TempALLSalvage` yang disusun `Activity/GCNMCountRequestSalvage_act-Act.xml` langkah 9–12.
type Summary struct {
	Rows []SummaryRow
}

// SummaryRow adalah satu baris tabel ringkas.
type SummaryRow struct {
	// Label adalah teks kolom "Status Salvage", disalin harfiah dari `Local.LOOP` pada
	// activity pencacahnya — termasuk salah ketik "Cheker".
	Label string

	// Tab adalah kode tab yang dibuka bila barisnya diklik.
	Tab string

	// Count adalah isi kolom "Jumlah".
	Count int
}

// Repo adalah seam ke antrean banding harga salvage SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
//
// # Kenapa tidak ada satu pun operasi yang menulis
//
// Bukan karena layarnya memang hanya membaca — di Pega ia punya kolom "Action" berisi tombol
// Approve dan Reject. Melainkan karena jalur tulisnya BELUM DAPAT dibangun.
//
// Section tombolnya dan activity penulisnya sudah diterima, sehingga sebagian besar alurnya
// kini terbaca. Yang MASIH menahan adalah keempat rule yang dipanggil activity itu.
//
// # Yang sudah PASTI, dan itu menutup pertanyaan terbesar
//
// Arti kedua kode statusnya. `Activity/ApprovalCheckerSalvage` langkah 11 menyusunnya begini:
//
//	SalvagePrice = @if(statusapprove == "1", hargarequest, hargasalvage)
//
// Artinya `1` menerima harga tandingan balai lelang, dan `0` mempertahankan harga semula.
// Jadi **`1` = Approve, `0` = Reject** — BUKAN `0` = menunggu seperti pada
// `T_CLAIM_KOMITE_LIST` (`RDB List/CountAIDiterima_SQL`). Dugaan itu sempat mungkin, dan
// kini gugur.
//
// Kelima isian yang dikirim tombolnya, beserta kolom sebenarnya:
//
//	statusapprove     `1` Approve · `0` Reject   -> STATUSAPPROVE
//	noteapprove       `.NoteKomite`              -> NOTEAPPROVE  (!)
//	iddetailsalvage   `.ClientName`              -> IDDETAILSALVAGE
//	idsalvage         `.ClaimNo`                 -> IDSALVAGE
//	hargarequest      `.Email`                   -> HARGAREQUEST
//
// Ketiga pemetaan terakhir MENEGUHKAN peta kolom modul ini dari sumber yang berbeda.
//
// # Yang TETAP menahan
//
// Empat rule yang dipanggil activity itu belum ada: `UpdateDataReqSalvage`,
// `UpdateDokReqSalvage`, `UpdateHargaSalvage`, dan Connect REST `SendData_SalvageSimasBid`.
//
// Dua di antaranya bukan sekadar detail:
//
//   - **Siapa yang menyetel `TGLAPPROVE`.** Satu-satunya UPDATE yang terbaca langsung —
//     dirangkai sebagai teks di langkah 5 — hanya menyetel `STATUSAPPROVE`, dan justru
//     menyaring `TGLAPPROVE IS NULL`. Padahal grid Request menyaring `TGLAPPROVE IS NULL`
//     dan grid History menuntutnya TERISI. Bila tidak ada yang mengisinya, baris yang sudah
//     diputus TIDAK pindah ke History dan TIDAK hilang dari Request.
//   - **`SendData_SalvageSimasBid` mengirim keputusannya KEMBALI ke balai lelang.** Menulis
//     ke basis data tanpa memanggilnya berarti keputusan yang tidak pernah sampai ke pihak
//     yang mengajukannya — setengah tindakan yang tidak menghasilkan satu pun galat.
//
// Menambalnya dengan tebakan berarti menulis keputusan atas NILAI UANG tanpa mengetahui
// aturannya. Lihat Limitations.
type Repo interface {
	// List mengembalikan satu halaman baris beserta jumlah seluruh baris yang cocok.
	//
	// Pemotongan halaman terjadi di BASIS DATA, bukan di aplikasi — lihat catatan paginasi
	// di kepala inboxbandinghargasalvage.sql.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// Count menghitung baris yang cocok dengan penyaring sebuah tab, tanpa paginasi.
	//
	// Ia dipakai tabel ringkas, dan penyaringnya WAJIB sama persis dengan List — itulah
	// yang membuat angka ringkas sama dengan jumlah baris gridnya (`D-73`, keputusan Work
	// Owner 2026-09-29). Lihat PlannedDifferences.
	Count(ctx context.Context, query Query) (int, error)

	// ListDecisions mengambil keputusan banding harga satu klaim — panel rincian pada grid
	// History Cheker.
	//
	// Daftar KOSONG bukan galat: klaim yang belum punya keputusan apa pun memang tidak
	// punya barisnya, dan pada panel ini keadaan itu sah. Ia tidak dapat tertukar dengan
	// "klaim tidak ditemukan", karena panel ini hanya dibuka dari baris yang sudah
	// tergambar di grid.
	ListDecisions(ctx context.Context, query DecisionQuery) ([]Decision, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan banding harga satu
// badan hukum kepada komite badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Kolom Aging dihitung BASIS DATA, bukan di sini — dengan bentuk portabel
// `CAST(CURRENT_TIMESTAMP AS DATE) - CAST(TGLREQUEST AS DATE)` yang menggantikan
// `TRUNC(SYSDATE) - TRUNC(…)` milik kueri lama (`D-20`). Clock tetap ada karena lapisan
// transport membentuk tanggal darinya, dan supaya modul ini tidak perlu dibongkar bila Aging
// kelak dipindahkan ke Go.
//
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
