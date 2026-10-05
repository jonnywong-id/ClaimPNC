// Package daftardetaildokumentravel adalah inti modul Daftar Detail Dokumen Travel
// (`F-4`, MENU_ID 39, MENU_PROGRAM `ListDocumentTravel`).
//
// # Apa yang dimodelkan di sini
//
// Satu baris adalah satu **aturan kelengkapan dokumen** pada klaim lini Travel: dokumen
// mana yang diminta, wajib atau tidak, dan paling sedikit berapa berkas yang harus
// diunggah. Aturan itulah yang dibaca `Activity/TravelDocument_act-Act.xml` saat klaim
// Travel diregistrasi, lalu diubahnya menjadi daftar dokumen yang harus dilampirkan.
//
// Di sistem lama layarnya adalah `Harness/ListDocumentTravel-Harness.xml`, dan datanya
// dibaca dari satu view:
//
//	V_LST_DOC_TRAVEL   ID · DOCID · DOCUMENTNAME · STSWAJIB · MINUNGGAH
//
// Kelima kolom itulah seluruh isinya — baik grid (`BrowseLstDocTravel_RD-RD.xml`) maupun
// form `Section/BrowseDocumentTravel-Section.xml` tidak menampilkan yang lain.
//
// # Pembatasan per Plan dan Jaminan TIDAK ada di layar ini
//
// Dicatat eksplisit karena ia sempat dibangun lalu dicabut, dan karena export rule
// menyesatkan di titik ini.
//
// `Section/BrowseDocumentTravel-Section.xml:3731` memuat grid berulang
// `TempDTDocTravel.COVERAGELIST` berkelas `ASM-FW-GCNMFW-Int-V_LST_DOC_TRAVEL_COVERAGE`,
// tanpa `pyDisplayWhen` yang menyembunyikannya. Dibaca dari XML saja, grid itu tampak
// menjadi bagian layar.
//
// **Di aplikasi Pega yang berjalan, grid itu tidak ada.** Work Owner memeriksa layarnya
// langsung dan menetapkan 2026-10-03 bahwa form tambah dan ubah hanya memuat kelima
// isian di atas. Yang dipercaya adalah layar yang berjalan, bukan XML — selisihnya
// kemungkinan besar karena versi rule di export berbeda dari yang ter-deploy, dan itu
// dicatat sebagai temuan di docs/catatan-pengembangan.md, bukan diperdebatkan di kode.
//
// Akibatnya modul ini **tidak menyentuh** V_LST_DOC_TRAVEL_COVERAGE maupun
// POOLDATA.M_PLANTRAVEL sama sekali. Tabel pertama tetap dibaca jalur registrasi klaim
// lewat `Activity/BrowseDocTravel-Act.xml`; yang berubah hanyalah bahwa layar master ini
// bukan penulisnya.
//
// # Batas modul: ini master DETAIL, bukan master induknya
//
// Ada master lain yang mudah tertukar dengannya, dan ia BUKAN lingkup paket ini:
//
//	M_DOCTRAVEL          <- internal/masterdokumentravel   MENU_ID 22
//	  └─ V_LST_DOC_TRAVEL     <- paket ini                 MENU_ID 39
//
// Yang di atas hanya menyimpan DOCID dan judul dokumen. Paket ini merujuk DOCID itu dan
// menambahkan aturannya. Akibat yang mengikat: paket ini TIDAK PERNAH menulis
// M_DOCTRAVEL — ia hanya membacanya sebagai daftar pilihan, lewat seam DocumentRepo yang
// sengaja tidak punya satu pun operasi tulis.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ListDocumentTravel-Harness.xml          layar "Detail Dokumen Travel"
//	Section/LSTDocumentTravel-Section.xml            judul, tombol Tambah dan Refresh
//	Section/BrowseDocumentTravel-Section.xml         grid, form, tombol Simpan dan Ubah
//	Report Definition/BrowseLstDocTravel_RD-RD.xml   kolom grid dan urutannya
//	Activity/BrowseDocTravel-Act.xml                 arti STSWAJIB
//	Activity/TravelDocument_act-Act.xml              siapa yang memakai aturan ini
//
// # Yang TIDAK ada di export, dan bagaimana ketiadaannya diperlakukan
//
// Jalur tulis layar ini hilang dari export (`R-16`): tombol Simpan menunjuk
// `CNMInsertDocumentTravel_act` dan tombol Ubah menunjuk `CNMSetDetailTravelDocument_act`
// (`Section/BrowseDocumentTravel-Section.xml`), dan **tidak satu pun ada di export**.
// Tidak ada pula procedure penggantinya — `Database/DOCTRAVEL_CVG.prc` hanya melayani
// M_DOCTRAVEL, yakni master induknya.
//
// Akibatnya bentuk INSERT dan UPDATE-nya TIDAK DAPAT ditiru; yang dapat ditiru adalah
// APA yang disimpan, dan itu terbaca lengkap dari form beserta view-nya. Bentuk
// pernyataannya karena itu disusun di modul ini dan diisolasi seluruhnya di
// repo/sqlstore/daftardetaildokumentravel.sql, dengan nama objek yang MASIH HARUS
// DIVERIFIKASI DBA — lihat migrations/0006_detail_dokumen_travel.up.sql.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package daftardetaildokumentravel

import (
	"context"
	"errors"
	"strings"
)

// Detail adalah satu aturan kelengkapan dokumen Travel.
//
// Lima field, dan itu memang seluruh isinya: baik grid maupun form di layar Pega hanya
// menampilkan kelimanya, dan Report Definition-nya pun hanya memilih kelima kolom itu.
type Detail struct {
	// ID adalah kunci baris V_LST_DOC_TRAVEL. Diterbitkan penyimpanan dan tidak pernah
	// diisi pengguna — di layar Pega pun isiannya hanya ditampilkan, tidak disunting.
	//
	// Ia kolom pengurut pertama grid (`BrowseLstDocTravel_RD-RD.xml`, pySortOrder 1).
	ID string

	// DocumentID adalah DOCID, rujukan ke POOLDATA.M_DOCTRAVEL.
	//
	// Di layar ia isian autocomplete yang sumbernya `BrowseMstDocTravel_RD` dan
	// menampilkan DOCID beserta NAMADOKUMEN-nya — jadi petugas memilih dari master
	// induk, tidak mengetik kode sendiri. Di sini ia tetap TEKS BEBAS, karena Work Owner
	// menetapkan layar ini tanpa validasi: kode yang tidak ada di master pun diterima,
	// persis seperti Pega yang tidak memeriksanya sebelum menyimpan.
	DocumentID string

	// DocumentName adalah DOCUMENTNAME, judul dokumen yang dibaca petugas.
	//
	// Ia TERSIMPAN DI BARIS INI, bukan diambil dari M_DOCTRAVEL saat ditampilkan —
	// `BrowseLstDocTravel_RD` memilihnya sebagai kolom view, dan `BrowseDocTravel-Act`
	// membacanya dari baris yang sama. Akibatnya judul di sini dapat berbeda dari judul
	// di master induknya, dan itu perilaku sistem lama yang dipertahankan (`P-5`).
	DocumentName string

	// Mandatory adalah STSWAJIB: dokumen ini wajib dilampirkan atau tidak.
	//
	// Di basis data ia ANGKA, bukan teks. `Activity/BrowseDocTravel-Act.xml` membuktikan
	// kedua nilainya secara langsung lewat precondition langkahnya:
	//
	//	.STSWAJIB==1  ->  Local.StatusWajib := "Ya"
	//	.STSWAJIB==0  ->  Local.StatusWajib := "Tidak"
	//
	// Teks "Ya"/"Tidak" itu hanya tampilan; yang tersimpan tetap 1 atau 0.
	Mandatory bool

	// MinUpload adalah MINUNGGAH, jumlah berkas minimum yang harus diunggah.
	//
	// Nol berarti tidak ada tuntutan jumlah. Perhatikan `TravelDocument_act` menuliskan
	// `MIN_DOC := 1` saat menyusun daftar dokumen klaim — angka tetap, bukan nilai dari
	// baris ini — sehingga MINUNGGAH hari ini TIDAK terbaca oleh jalur registrasi klaim.
	// Itu keadaan sistem lama apa adanya; modul ini menyimpannya dengan benar dan tidak
	// mengubah pembacanya.
	MinUpload int
}

// Input adalah nilai yang dikirim pengguna dari layar.
//
// Terpisah dari Detail karena ID tidak pernah berasal dari pengguna: pada penambahan ia
// diterbitkan penyimpanan, pada penyuntingan ia diambil dari jalur URL.
type Input struct {
	DocumentID   string
	DocumentName string
	Mandatory    bool
	MinUpload    int
}

// Clean memangkas spasi di kedua ujung setiap isian teks.
//
// # Ini SATU-SATUNYA perlakuan atas isian, dan ia bukan validasi
//
// Work Owner menetapkan 2026-09-21: layar ini tanpa validasi, sama seperti Master
// Dokumen Travel. Nama dokumen kosong diterima dan DOCID yang tidak ada di master pun
// diterima — persis seperti `Section/BrowseDocumentTravel-Section.xml`, yang tidak memuat
// satu pun `pyRequired` bernilai true maupun Validate rule.
//
// Pemangkasan spasi tetap dilakukan, dan alasannya bukan kerapian: kolomnya dibaca
// kembali dengan pemangkasan, karena kolom CHAR berlebar tetap memadatkan nilainya
// dengan spasi tanpa memberi tanda apa pun. Tanpa memangkas saat menulis, apa yang
// disimpan dan apa yang dibaca kembali dapat berbeda — dan selisih itu tidak terlihat di
// layar karena spasi tidak tampak.
func (i Input) Clean() Input {
	clean := Input{
		DocumentID:   strings.TrimSpace(i.DocumentID),
		DocumentName: strings.TrimSpace(i.DocumentName),
		Mandatory:    i.Mandatory,
		MinUpload:    i.MinUpload,
	}

	// MINUNGGAH negatif tidak punya arti apa pun — "paling sedikit minus satu berkas"
	// bukan aturan yang dapat dipenuhi maupun dilanggar. Ia diratakan menjadi nol, bukan
	// ditolak, supaya perlakuannya tetap sejalan dengan "tanpa validasi".
	if clean.MinUpload < 0 {
		clean.MinUpload = 0
	}
	return clean
}

// ErrNotFound: baris yang diminta tidak ada.
//
// Hanya satu galat domain, dan itu konsekuensi langsung dari "tanpa validasi": tidak ada
// aturan isian yang dapat dilanggar, dan tidak ada keunikan yang dapat bentrok. Yang
// masih mungkin gagal hanyalah menunjuk baris yang sudah tidak ada.
var ErrNotFound = errors.New("daftardetaildokumentravel: detail dokumen travel tidak ditemukan")

// Document adalah satu pilihan pada isian ID Dokumen, dibaca dari master induk.
//
// Ia sengaja BUKAN masterdokumentravel.TravelDocument: modul tidak saling mengimpor
// tipenya, sehingga perubahan di satu modul tidak merambat ke modul lain. Yang dibagi
// adalah tabelnya, bukan kodenya.
type Document struct {
	ID   string
	Name string
}

// Repo adalah seam ke penyimpanan detail dokumen travel SATU portal.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Satu instans Repo selalu terikat pada satu basis data entitas —
// pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri (`ADR-0030`
// Opsi 1). Yang memilih instans mana yang melayani satu permintaan adalah RepoSelector.
//
// Tidak ada Delete, dan itu bukan kelalaian: layar Pega tidak punya tombol hapus, dan
// `D-66` melarang penghapusan fisik data bernilai bisnis. Baris ini menentukan dokumen
// apa yang diminta pada klaim yang sedang berjalan — menghapusnya mengubah kelengkapan
// klaim yang sudah telanjur dinilai.
type Repo interface {
	// List mengembalikan seluruh aturan dokumen, terurut seperti grid lama: ID menaik,
	// lalu DOCID menaik.
	List(ctx context.Context) ([]Detail, error)

	// Get mengembalikan satu aturan; ErrNotFound bila barisnya tidak ada.
	Get(ctx context.Context, id string) (Detail, error)

	// InsertNew menerbitkan ID lalu menyisipkan barisnya, dan mengembalikan baris yang
	// benar-benar tersimpan.
	//
	// Penerbitan ID berada DI DALAM satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi: memecahnya melebarkan jarak antara
	// mengambil nomor urut dan memakainya, dan memaksa lapisan aplikasi mengetahui
	// bentuk kunci yang seharusnya hanya diketahui penyimpanan.
	InsertNew(ctx context.Context, input Input) (Detail, error)

	// Update mengganti isi satu aturan; ErrNotFound bila barisnya hilang di antara
	// pemuatan layar dan penyimpanan.
	Update(ctx context.Context, id string, input Input) (Detail, error)
}

// DocumentRepo adalah seam BACA-SAJA ke master dokumen travel (POOLDATA.M_DOCTRAVEL).
//
// Ketiadaan operasi tulis di sini disengaja: tabel itu dimiliki modul Master Dokumen
// Travel (`P-1` — satu tabel satu penulis), dan batas itu ditegakkan oleh bentuk
// antarmuka, bukan oleh ingatan orang yang menulis kode berikutnya.
type DocumentRepo interface {
	// List mengembalikan seluruh dokumen, terurut menurut DOCID seperti kueri lama.
	List(ctx context.Context) ([]Document, error)
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

// DocumentRepoSelector memilih DocumentRepo milik satu portal entitas.
//
// Terpisah dari RepoSelector meski keduanya selalu dipilih bersamaan, karena keduanya
// mengisi seam yang berbeda: yang satu tabel milik modul ini, yang lain tabel milik modul
// master induknya yang HANYA DIBACA.
type DocumentRepoSelector func(portalAlias string) (DocumentRepo, error)
