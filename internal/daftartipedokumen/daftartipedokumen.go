// Package daftartipedokumen adalah inti modul Daftar Tipe Dokumen (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Satu baris adalah satu **tipe dokumen klaim** — kategori paling atas dari dokumen yang
// dapat dilampirkan pada sebuah klaim. Daftar inilah yang dipakai layar unggah dokumen
// untuk menyebut golongan berkas yang sedang diunggah, dan yang dirujuk layar Arsip
// Dokumen saat menelusuri berkas klaim lama.
//
// Di sistem lama ia tinggal di POOLDATA.LST_DOC_TYPE, dibaca lewat view
// POOLDATA.V_LST_DOC_TYPE yang mengeluarkan enam kolom:
//
//	ID             kunci baris, dibuat sistem
//	OLD_ID         penomoran warisan; tidak pernah diisi baris baru
//	TYPE_DOCUMENT  nama tipe dokumen yang dibaca petugas
//	STS_PROSES     catatan bebas — lihat di bawah, ia BUKAN penanda aktif
//	USER_EDIT      siapa yang terakhir menyimpan
//	TGL_EDIT       kapan terakhir disimpan
//
// # STS_PROSES bukan status, dan itu temuan yang mudah salah dibaca
//
// Namanya menyiratkan penanda aktif/non-aktif, dan dugaan itu KELIRU. Dua bukti:
//
//  1. Di form Pega ia isian teks biasa — `Section/BrowseListDocumentType-Section.xml`
//     mengikatnya ke `TempDcol.STS_PROSES` dengan `pyEditOptions=Auto`, TANPA satu pun
//     daftar pilihan, tanpa `pyRequired`, dan tanpa `pyMaxLength`.
//  2. Layar Arsip Dokumen membacanya sebagai catatan bebas, bahkan mengalias-namakannya
//     `"NoteKasir"` — `Activity/SearchDataArchiveFilling-Act.xml` dan
//     `Activity/GetDataArchiveCabangKlaim-Act.xml`.
//
// Mengubahnya menjadi sakelar aktif akan memutus kedua pembaca itu. Work Owner
// menetapkan 2026-09-21: ia **ditiru apa adanya sebagai teks bebas**.
//
// # Batas modul: ini master INDUK, bukan master detailnya
//
// Ada dua master lain yang mudah tertukar dengannya, dan keduanya BUKAN lingkup paket ini:
//
//	LST_DOC_TYPE            <- paket ini      menu "Daftar Tipe Dokumen" (MENU_ID 40)
//	  ├─ V_LST_DET_TYPE_DOC                   menu "Daftar Detail Tipe Dokumen"
//	  └─ LST_TYPE_DOC_BUSINESS                menu "Detail Tipe Dokumen per Bisnis"
//
// Keduanya merujuk ID milik yang pertama dan menambahkan aturan per tipe dokumen. Masing-
// masing punya harness tersendiri (`ListDetTypeDocument`, `DetTypeDocumenBisnis`), dan
// keduanya belum dibangun.
//
// Akibat yang mengikat paket ini: ID yang sudah terbit TIDAK PERNAH BERUBAH dan TIDAK
// PERNAH DIHAPUS — kedua master turunan itu, ditambah sekurang-kurangnya sepuluh rule
// pembaca lain, merujuknya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ListDocumentTypeInbox-Harness.xml         layar "Daftar Tipe Dokumen"
//	Section/BrowseListDocumentType-Section.xml         grid 3 kolom + form Tambah/Ubah
//	Section/ListDocumentType-Section.xml               bingkai layar, tombol Tambah/Refresh
//	Report Definition/BrowseLstDocType_RD-RD.xml       daftar beserta kolom yang dipilih
//	Activity/CNMSetListDocumentType_act-Act.xml        memuat satu baris ke form
//	Activity/CNMInsertListDocumentType_act-Act.xml     urutan langkah simpan
//	RDB List/UpdateLstDocType-SQL.xml                  pemanggilan procedure penyimpan
//	Database/PEGA_LST_DOC_TYPE.prc                     isi procedure itu
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package daftartipedokumen

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

// DocumentType adalah satu tipe dokumen klaim.
//
// Ketiga field memetakan ke kolom yang dibaca Pega lewat POOLDATA.V_LST_DOC_TYPE, dan
// namanya mengikuti `CONTEXT.md` — bukan nama kolomnya. Pemetaannya ada di repo/sqlstore,
// satu tempat saja.
//
// OLD_ID, USER_EDIT, dan TGL_EDIT sengaja TIDAK ada di sini. Ketiganya memang tidak
// pernah tampil di layar Pega — baik grid maupun formnya hanya menampilkan tiga field di
// bawah. OLD_ID adalah jejak sejarah yang tidak dikelola siapa pun, dan kedua field jejak
// simpan diisi otomatis saat menyimpan (lihat Editor).
type DocumentType struct {
	// ID adalah kolom ID. Dibuat sistem saat tipe dokumen ditambahkan dan TIDAK PERNAH
	// berubah sesudahnya — layar Pega pun menandainya read-only
	// (`Section/BrowseListDocumentType-Section.xml`, `pyEditOptions=Read-only`).
	ID string

	// Type adalah TYPE_DOCUMENT, nama yang dibaca petugas. Di grid Pega ia berlabel
	// "Tipe Dokumen", dan di formnya "Jenis Dokumen" — kedua teks itu memang berbeda di
	// layar lama, dan keduanya ditiru apa adanya (`D-13`).
	Type string

	// ProcessStatus adalah STS_PROSES. **Teks bebas, bukan penanda aktif** — lihat
	// penjelasan di kepala paket ini. Di layar Pega ia berlabel "Status Proses".
	ProcessStatus string
}

// Input adalah nilai yang dikirim pengguna dari layar.
//
// Terpisah dari DocumentType karena ID tidak pernah berasal dari pengguna: pada
// penambahan ia diterbitkan penyimpanan, pada penyuntingan ia diambil dari jalur URL.
type Input struct {
	Type          string
	ProcessStatus string
}

// Clean memangkas spasi di kedua ujung kedua isian.
//
// # Ini SATU-SATUNYA perlakuan atas isian, dan ia bukan validasi
//
// Work Owner menetapkan 2026-09-21: layar ini meniru Pega apa adanya, TANPA VALIDASI.
// Nama kosong diterima, nama ganda diterima, dan panjangnya tidak dibatasi aplikasi —
// persis seperti `Section/BrowseListDocumentType-Section.xml`, yang tidak memuat satu pun
// `pyRequired=true` maupun `pyMaxLength` pada isian mana pun, dan seperti
// `Database/PEGA_LST_DOC_TYPE.prc` yang menyisipkan tanpa memeriksa apa pun.
//
// Pemangkasan spasi tetap dilakukan, dan alasannya bukan kerapian: kolomnya dibaca
// kembali dengan pemangkasan (kolom CHAR berlebar tetap memadatkan nilainya dengan spasi
// tanpa memberi tanda apa pun). Tanpa memangkas saat menulis, apa yang disimpan dan apa
// yang dibaca kembali dapat berbeda — dan selisih itu tidak terlihat di layar karena
// spasi tidak tampak.
//
// Perlakuan yang sama dipakai seluruh modul master yang sudah ada.
func (i Input) Clean() Input {
	return Input{
		Type:          strings.TrimSpace(i.Type),
		ProcessStatus: strings.TrimSpace(i.ProcessStatus),
	}
}

// Editor adalah jejak siapa yang menyimpan dan kapan.
//
// Ia BUKAN isian pengguna dan tidak pernah datang dari badan permintaan: sistem lama pun
// mengisinya sendiri saat menyimpan (`Activity/CNMInsertListDocumentType_act-Act.xml`
// menetapkan `TempDcol.USER_EDIT := OperatorID.pyUserIdentifier` dan
// `TempDcol.TGL_EDIT := @getCurrentTimeStamp()`).
//
// Nilainya dibawa sampai ke repo, bukan dibentuk di sana, karena repo tidak tahu siapa
// yang sedang masuk dan tidak boleh tahu — identitas datang dari sesi, dan sesi adalah
// urusan lapisan transport.
type Editor struct {
	// Identity mengisi USER_EDIT. Sama dengan `OperatorID.pyUserIdentifier` di Pega.
	Identity string

	// At mengisi TGL_EDIT.
	//
	// Waktunya dibawa dari luar, bukan diambil `time.Now()` di dalam repo, supaya seluruh
	// modul membaca jam lewat satu seam yang sama (`F-5`) dan supaya penyimpanan dapat
	// diuji secara deterministik.
	At time.Time
}

// ErrNotFound: baris yang diminta tidak ada di master.
//
// Hanya satu galat domain, dan itu konsekuensi langsung dari "tanpa validasi": tidak ada
// aturan isian yang dapat dilanggar, dan tidak ada keunikan yang dapat bentrok. Yang
// masih mungkin gagal hanyalah menunjuk baris yang sudah tidak ada.
var ErrNotFound = errors.New("daftartipedokumen: tipe dokumen tidak ditemukan")

// SequenceDigits adalah lebar nomor urut pada ID.
//
// Angkanya diambil langsung dari `Database/PEGA_LST_DOC_TYPE.prc:21`:
//
//	id_lst_doc_type := id_site || lpad(to_Char(SET_LST_DOC_TYPE.nextval),4,'0');
//
// EMPAT, bukan tiga seperti Master Status Klaim dan bukan lima seperti Master Dokumen
// Travel. Ketiganya memakai pola yang sama dengan lebar yang berbeda-beda, dan menyalin
// lebar dari modul tetangga akan menghasilkan ID yang tidak dikenali data historis.
const SequenceDigits = 4

// FormatID menyusun ID dari kode situs dan nomor urut, meniru procedure lama persis.
//
// # Kenapa bentuknya direplikasi, bukan diperbaiki
//
// ID ini dirujuk dua master turunan (`V_LST_DET_TYPE_DOC`, `LST_TYPE_DOC_BUSINESS`) dan
// sekurang-kurangnya sepuluh rule pembaca lain — di antaranya
// `RDB List/BrowseRegisterCvg-SQL.xml` dan `Section/SecArchiveDokumen-Section.xml`.
// Mengganti bentuknya akan memutus baris baru dari data historis. `P-5` berlaku — bentuk
// ini tidak ada di daftar 13 perbaikan eksplisit `D-49`.
//
// # Batas yang nyata, dan cacat yang ikut terbawa
//
// Nomor di atas 9999 dikembalikan APA ADANYA, tanpa dipotong, sama seperti LPAD Oracle.
// ID ke-10000 karena itu menjadi satu karakter lebih panjang.
//
// Dipotong menjadi empat digit? Tidak. Itu menghasilkan ID GANDA, yang jauh lebih buruk
// daripada penyisipan yang gagal dengan pesan jelas — dan ID ganda di sini berarti dua
// tipe dokumen berbeda berbagi satu kunci yang dirujuk dua master turunan.
//
// Lebar kolom ID yang sebenarnya belum dapat diperiksa: DDL tabelnya tidak ada di export
// (`R-08`). Bila kolomnya ternyata sempit, penyisipan ke-10000 akan ditolak basis data
// (ORA-12899) — dan keputusan memperlebar kolom sebaiknya diambil sebelum, bukan sesudah,
// penyisipan pertama yang gagal. Batas ini dicatat di migrasi 0005 langkah 0.
func FormatID(site string, sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < SequenceDigits {
		digits = "0" + digits
	}
	return strings.TrimSpace(site) + digits
}

// Repo adalah seam ke penyimpanan daftar tipe dokumen SATU portal.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Satu instans Repo selalu terikat pada satu basis data entitas —
// pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri (`ADR-0030`
// Opsi 1). Yang memilih instans mana yang melayani satu permintaan adalah RepoSelector.
//
// Tidak ada Delete, dan itu bukan kelalaian: layar Pega tidak punya tombol hapus, dan
// `Database/PEGA_LST_DOC_TYPE.prc` hanya mengenal INSERT dan UPDATE. Menghapus satu baris
// akan membuat setiap dokumen klaim dan setiap baris master turunan yang menyimpan ID itu
// kehilangan artinya — persis alasan `ADR-0012` menetapkan master tidak dihapus permanen,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis.
type Repo interface {
	// List mengembalikan seluruh tipe dokumen, terurut seperti kueri lama.
	List(ctx context.Context) ([]DocumentType, error)

	// Get mengembalikan satu tipe dokumen; ErrNotFound bila ID-nya tidak ada.
	Get(ctx context.Context, id string) (DocumentType, error)

	// InsertNew menerbitkan ID lalu menyisipkan barisnya, dan mengembalikan baris yang
	// benar-benar tersimpan beserta ID-nya.
	//
	// Penerbitan ID berada DI DALAM satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi — keduanya memang satu langkah pada
	// procedure lama, dan memecahnya melebarkan jarak antara mengambil nomor urut dan
	// memakainya.
	InsertNew(ctx context.Context, input Input, by Editor) (DocumentType, error)

	// Update mengganti isi baris yang sudah ada; ErrNotFound bila barisnya hilang di
	// antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, doc DocumentType, by Editor) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca saat
// permintaan datang — bukan diputuskan sekali ketika aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
//
// # Kenapa modul ini per portal
//
// Dibuktikan dari Pega, bukan diandaikan: `Database/PEGA_LST_DOC_TYPE.prc:12` membentuk
// ID dengan `SELECT ID FROM M_SITE_DATABASE WHERE CURRENT_SITE = '1'` — kode situs yang
// melekat pada basis data tempat procedure berjalan. Mekanismenya identik dengan
// `DOCTRAVEL_CVG.prc:12` dan `PEGA_M_CAUSE_OF_LOSS.prc:11`, dua master yang sudah
// diputuskan per entitas. Tiap entitas karena itu menerbitkan awalan ID-nya sendiri, dan
// tabelnya memang terpisah per entitas.
type RepoSelector func(portalAlias string) (Repo, error)
