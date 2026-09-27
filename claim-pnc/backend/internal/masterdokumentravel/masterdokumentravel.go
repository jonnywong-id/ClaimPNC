// Package masterdokumentravel adalah inti modul Master Dokumen Travel (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Satu baris adalah satu **jenis dokumen** yang dapat diminta pada klaim lini Travel —
// paspor, tiket, boarding pass, dan seterusnya. Daftar inilah yang kemudian dipakai
// layar unggah dokumen klaim untuk menyebut dokumen apa saja yang perlu dilampirkan.
//
// Di sistem lama ia tinggal di POOLDATA.M_DOCTRAVEL, dan hanya punya dua kolom:
//
//	DOCID        kunci baris, dibuat sistem
//	NAMADOKUMEN  judul dokumen yang dibaca petugas
//
// # Batas modul: ini master INDUK, bukan master detailnya
//
// Ada master kedua yang mudah tertukar dengannya, dan ia BUKAN lingkup paket ini:
//
//	M_DOCTRAVEL        <- paket ini              menu "Master Dokumen Travel" (MENU_ID 22)
//	  └─ V_LST_DOC_TRAVEL   menu "Daftar Detail Dokumen Travel" (MENU_ID 39)
//	       DOCID · DOCUMENTNAME · MINUNGGAH · STSWAJIB
//
// Yang kedua merujuk DOCID milik yang pertama dan menambahkan aturan per dokumen —
// wajib atau tidak, dan jumlah unggahan minimum. Ia layar tersendiri dengan harness
// tersendiri (`Harness/ListDocumentTravel-Harness.xml`), dan sejak 2026-09-22 sudah
// dibangun sebagai internal/daftardetaildokumentravel.
//
// Akibat yang mengikat paket ini: DOCID yang sudah terbit TIDAK PERNAH BERUBAH dan
// TIDAK PERNAH DIHAPUS — baris detail dan dokumen klaim yang sudah diunggah merujuknya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/BrowseMasterDocumentTravel_Harness-Harness.xml  layar "Master Dokumen Travel"
//	Section/BrowseMasterDocumentTravel-Section.xml           grid 2 kolom, tombol Tambah/Ubah/Simpan
//	Report Definition/BrowseMstDocTravel_RD-RD.xml           daftar, ORDER BY DOCID ASC
//	Activity/SetMstDocTravelValue_act-Act.xml                memuat satu baris ke form
//	Activity/CNMInsertMstDocTravel_act-Act.xml               urutan langkah simpan
//	RDB List/UpdateMstDocTravel-SQL.xml                      pemanggilan procedure penyimpan
//	Database/DOCTRAVEL_CVG.prc                               isi procedure itu
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterdokumentravel

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// TravelDocument adalah satu jenis dokumen Travel.
//
// Hanya dua field, dan itu memang seluruh isinya: baik grid maupun form di layar Pega
// hanya menampilkan DOCID dan NAMADOKUMEN, dan Report Definition-nya pun hanya memilih
// kedua kolom itu.
type TravelDocument struct {
	// ID adalah DOCID. Dibuat sistem saat dokumen ditambahkan dan TIDAK PERNAH berubah
	// sesudahnya — layar Pega pun menandainya read-only, dan baris detail pada
	// V_LST_DOC_TRAVEL merujuknya.
	ID string

	// Name adalah NAMADOKUMEN, teks yang dibaca petugas. Di layar Pega ia berlabel
	// "Judul Dokumen".
	Name string
}

// Input adalah nilai yang dikirim pengguna dari layar.
//
// Terpisah dari TravelDocument karena ID tidak pernah berasal dari pengguna: pada
// penambahan ia diterbitkan penyimpanan, pada penyuntingan ia diambil dari jalur URL.
type Input struct {
	Name string
}

// Clean memangkas spasi di kedua ujung judul dokumen.
//
// # Ini SATU-SATUNYA perlakuan atas isian, dan ia bukan validasi
//
// Work Owner menetapkan 2026-09-21: layar ini meniru Pega apa adanya, TANPA VALIDASI.
// Judul kosong diterima, judul ganda diterima, dan panjangnya tidak dibatasi aplikasi —
// persis seperti `Section/BrowseMasterDocumentTravel-Section.xml`, yang tidak memuat
// satu pun `pyRequired=true` maupun `pyMaxLength`.
//
// Pemangkasan spasi tetap dilakukan, dan alasannya bukan kerapian: kolomnya dibaca
// kembali dengan pemangkasan (kolom CHAR berlebar tetap memadatkan nilainya dengan
// spasi tanpa memberi tanda apa pun). Tanpa memangkas saat menulis, apa yang disimpan
// dan apa yang dibaca kembali dapat berbeda — dan selisih itu tidak terlihat di layar
// karena spasi tidak tampak.
//
// Perlakuan yang sama dipakai kedua modul master yang sudah ada.
func (i Input) Clean() Input {
	return Input{Name: strings.TrimSpace(i.Name)}
}

// ErrNotFound: baris yang diminta tidak ada di master.
//
// Hanya satu galat domain, dan itu konsekuensi langsung dari "tanpa validasi": tidak ada
// aturan isian yang dapat dilanggar, dan tidak ada keunikan yang dapat bentrok. Yang
// masih mungkin gagal hanyalah menunjuk baris yang sudah tidak ada.
var ErrNotFound = errors.New("masterdokumentravel: dokumen travel tidak ditemukan")

// SequenceDigits adalah lebar nomor urut pada DOCID.
//
// Angkanya diambil langsung dari `Database/DOCTRAVEL_CVG.prc:20`:
//
//	id_docTravel := id_site || lpad(to_Char(DOCTRAVEL_SEQ.nextval),5,'0');
const SequenceDigits = 5

// FormatID menyusun DOCID dari kode situs dan nomor urut, meniru procedure lama persis.
//
// # Kenapa bentuknya direplikasi, bukan diperbaiki
//
// DOCID dirujuk baris V_LST_DOC_TRAVEL dan dokumen klaim yang sudah terunggah. Mengganti
// bentuknya akan memutus baris baru dari data historis. `P-5` berlaku — bentuk ini tidak
// ada di daftar 13 perbaikan eksplisit `D-49`.
//
// # Batas yang nyata, dan cacat yang ikut terbawa
//
// Nomor di atas 99999 dikembalikan APA ADANYA, tanpa dipotong, sama seperti LPAD Oracle.
// Kode ke-100000 karena itu menjadi satu karakter lebih panjang. `DOCTRAVEL_CVG.prc:5`
// mendeklarasikan penampungnya `varchar(8)`, sehingga dengan kode situs satu karakter
// masih tersisa ruang, tetapi dengan kode situs tiga karakter penyisipannya akan DITOLAK
// basis data (ORA-12899).
//
// Dipotong menjadi lima digit? Tidak. Itu menghasilkan DOCID GANDA, yang jauh lebih
// buruk daripada penyisipan yang gagal dengan pesan jelas — dan DOCID ganda berarti dua
// jenis dokumen berbeda berbagi satu kunci yang dirujuk tabel lain.
//
// Lebar kolom DOCID yang sebenarnya belum dapat diperiksa: DDL tabel tidak ada di export
// (`R-08`), dan `varchar(8)` di atas adalah lebar VARIABEL LOKAL di dalam procedure,
// bukan lebar kolomnya.
func FormatID(site string, sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < SequenceDigits {
		digits = "0" + digits
	}
	return strings.TrimSpace(site) + digits
}

// Repo adalah seam ke penyimpanan master dokumen travel SATU portal.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Satu instans Repo selalu terikat pada satu basis data entitas —
// pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri (`ADR-0030`
// Opsi 1). Yang memilih instans mana yang melayani satu permintaan adalah RepoSelector.
//
// Tidak ada Delete, dan itu bukan kelalaian: layar Pega tidak punya tombol hapus, dan
// `Database/DOCTRAVEL_CVG.prc` hanya mengenal INSERT dan UPDATE. Menghapus satu baris
// akan membuat setiap dokumen klaim yang menyimpan DOCID itu kehilangan artinya —
// persis alasan `ADR-0012` menetapkan master tidak dihapus permanen.
type Repo interface {
	// List mengembalikan seluruh dokumen, terurut menurut DOCID seperti kueri lama.
	List(ctx context.Context) ([]TravelDocument, error)

	// Get mengembalikan satu dokumen; ErrNotFound bila DOCID-nya tidak ada.
	Get(ctx context.Context, id string) (TravelDocument, error)

	// InsertNew menerbitkan DOCID lalu menyisipkan barisnya, dan mengembalikan baris
	// yang benar-benar tersimpan beserta ID-nya.
	//
	// Penerbitan ID berada DI DALAM satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi — keduanya memang satu langkah pada
	// procedure lama, dan memecahnya melebarkan jarak antara mengambil nomor urut dan
	// memakainya.
	InsertNew(ctx context.Context, input Input) (TravelDocument, error)

	// Update mengganti judul dokumen yang sudah ada; ErrNotFound bila barisnya hilang
	// di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, doc TravelDocument) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca saat
// permintaan datang — bukan diputuskan sekali ketika aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
