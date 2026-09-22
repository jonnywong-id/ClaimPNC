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
// dibaca dari DUA view:
//
//	V_LST_DOC_TRAVEL            ID · DOCID · DOCUMENTNAME · STSWAJIB · MINUNGGAH
//	V_LST_DOC_TRAVEL_COVERAGE   + PLANID · COVERAGEID · COVERAGENAME
//
// Yang pertama mengisi grid layar (`Report Definition/BrowseLstDocTravel_RD-RD.xml`).
// Yang kedua menyimpan pembatasan per **Plan** dan **Jaminan** — satu aturan dokumen
// dapat dibatasi hanya berlaku pada kombinasi plan dan jaminan tertentu. Keduanya
// dirangkai satu form: `Section/BrowseDocumentTravel-Section.xml` memuat isian dokumen
// di bagian atas dan satu grid berulang `TempDTDocTravel.COVERAGELIST` di bawahnya.
//
// # Batas modul: ini master DETAIL, bukan master induknya
//
// Ada master lain yang mudah tertukar dengannya, dan ia BUKAN lingkup paket ini:
//
//	M_DOCTRAVEL          <- internal/masterdokumentravel   MENU_ID 22
//	  └─ V_LST_DOC_TRAVEL     <- paket ini                 MENU_ID 39
//	       └─ V_LST_DOC_TRAVEL_COVERAGE
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
//	Activity/BrowseDocTravel-Act.xml                 arti STSWAJIB dan penyaring per plan
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
// APA yang disimpan, dan itu terbaca lengkap dari form beserta kedua view-nya. Bentuk
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

	// Coverages adalah pembatasan per Plan dan Jaminan, isi
	// V_LST_DOC_TRAVEL_COVERAGE untuk baris ini.
	//
	// KOSONG berarti aturan dokumen berlaku TANPA pembatasan plan maupun jaminan — dan
	// itu keadaan yang sah, bukan data yang belum lengkap. Grid utama layar Pega pun
	// hanya membaca V_LST_DOC_TRAVEL, sehingga baris tanpa coverage tetap tampil utuh.
	//
	// Senarai ini hanya terisi pada pembacaan SATU baris (Get), tidak pada daftar.
	// Daftar tidak membutuhkannya — gridnya lima kolom dan tidak satu pun menyebut plan
	// atau jaminan — dan menariknya untuk seluruh baris berarti satu kueri yang hasilnya
	// tidak pernah dilihat siapa pun.
	Coverages []Coverage
}

// Coverage adalah satu pembatasan plan dan jaminan pada sebuah aturan dokumen.
//
// Bentuknya mengikuti `Section/BrowseDocumentTravel-Section.xml` pada grid
// `TempDTDocTravel.COVERAGELIST`: dua isian yang terlihat petugas — Nama Plan dan Nama
// Jaminan — masing-masing menyimpan kode tersembunyi di sampingnya.
type Coverage struct {
	// ID adalah kunci baris V_LST_DOC_TRAVEL_COVERAGE. Diterbitkan penyimpanan.
	ID string

	// PlanID adalah PLANID, kode plan travel.
	//
	// Di layar ia diisi otomatis saat petugas memilih Nama Plan — autocomplete
	// `BrowsePlanTravelMaster_RD` menyalin `.ID` ke `.PLANID` (`pyPropertyTarget`).
	PlanID string

	// PlanName adalah PLANNAME, nama plan yang dilihat petugas.
	PlanName string

	// CoverageID adalah COVERAGEID, kode jaminan.
	//
	// Diisi otomatis dari autocomplete `SearchCoverageTravel_RD`, yang DISARING oleh
	// plan yang sudah dipilih — parameter `plan` diisi `.PLANID` pada baris yang sama.
	// Urutan pengisiannya karena itu mengikat: plan dulu, jaminan menyusul.
	CoverageID string

	// CoverageName adalah COVERAGENAME, nama jaminan yang dilihat petugas.
	CoverageName string
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
	Coverages    []CoverageInput
}

// CoverageInput adalah satu baris grid plan dan jaminan yang dikirim layar.
//
// Tanpa ID: seluruh daftar coverage DIGANTI setiap kali disimpan (lihat Repo.Update),
// sehingga kunci baris lamanya tidak berguna bagi pemanggil.
type CoverageInput struct {
	PlanID       string
	PlanName     string
	CoverageID   string
	CoverageName string
}

// Clean memangkas spasi di kedua ujung setiap isian teks, dan membuang baris coverage
// yang seluruhnya kosong.
//
// # Ini SATU-SATUNYA perlakuan atas isian, dan ia bukan validasi
//
// Work Owner menetapkan 2026-09-21: layar ini tanpa validasi, sama seperti Master
// Dokumen Travel. Judul kosong diterima, DOCID yang tidak ada di master diterima, dan
// baris coverage yang hanya berisi plan tanpa jaminan pun diterima — persis seperti
// `Section/BrowseDocumentTravel-Section.xml`, yang tidak memuat satu pun `pyRequired`
// bernilai true maupun Validate rule.
//
// Pemangkasan spasi tetap dilakukan, dan alasannya bukan kerapian: kolomnya dibaca
// kembali dengan pemangkasan, karena kolom CHAR berlebar tetap memadatkan nilainya
// dengan spasi tanpa memberi tanda apa pun. Tanpa memangkas saat menulis, apa yang
// disimpan dan apa yang dibaca kembali dapat berbeda — dan selisih itu tidak terlihat di
// layar karena spasi tidak tampak.
//
// Baris coverage yang SELURUH isiannya kosong dibuang, dan itu pun bukan validasi
// melainkan pembacaan maksud: grid di layar selalu menyisakan baris kosong yang baru
// ditambahkan tetapi belum diisi. Menyimpannya berarti menulis pembatasan plan yang
// tidak membatasi apa pun, lalu membacanya kembali sebagai baris hantu di form
// berikutnya.
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

	clean.Coverages = make([]CoverageInput, 0, len(i.Coverages))
	for _, row := range i.Coverages {
		row = CoverageInput{
			PlanID:       strings.TrimSpace(row.PlanID),
			PlanName:     strings.TrimSpace(row.PlanName),
			CoverageID:   strings.TrimSpace(row.CoverageID),
			CoverageName: strings.TrimSpace(row.CoverageName),
		}
		if row.PlanID == "" && row.PlanName == "" && row.CoverageID == "" && row.CoverageName == "" {
			continue
		}
		clean.Coverages = append(clean.Coverages, row)
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

// Plan adalah satu pilihan pada isian Nama Plan.
type Plan struct {
	ID   string
	Name string
}

// CoverageOption adalah satu pilihan pada isian Nama Jaminan.
//
// PlanID ikut dibawa supaya layar dapat menyaring jaminan menurut plan yang sudah
// dipilih tanpa menembak server lagi untuk setiap baris grid — persis penyaringan yang
// dilakukan `SearchCoverageTravel_RD` lewat parameter `plan`.
type CoverageOption struct {
	ID     string
	Name   string
	PlanID string
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
	// List mengembalikan seluruh aturan dokumen TANPA coverage-nya, terurut seperti
	// grid lama: ID menaik, lalu DOCID menaik.
	List(ctx context.Context) ([]Detail, error)

	// Get mengembalikan satu aturan LENGKAP dengan daftar coverage-nya; ErrNotFound
	// bila barisnya tidak ada.
	Get(ctx context.Context, id string) (Detail, error)

	// InsertNew menerbitkan ID lalu menyisipkan barisnya beserta seluruh coverage-nya,
	// dan mengembalikan baris yang benar-benar tersimpan.
	//
	// Penerbitan ID berada DI DALAM satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi: memecahnya melebarkan jarak antara
	// mengambil nomor urut dan memakainya, dan memaksa lapisan aplikasi mengetahui
	// bentuk kunci yang seharusnya hanya diketahui penyimpanan.
	InsertNew(ctx context.Context, input Input) (Detail, error)

	// Update mengganti isi satu aturan beserta SELURUH daftar coverage-nya;
	// ErrNotFound bila barisnya hilang di antara pemuatan layar dan penyimpanan.
	//
	// Daftar coverage diganti seluruhnya, bukan ditambal baris demi baris. Alasannya
	// ada di layar: grid di form memang mengirim susunan akhir yang dikehendaki
	// petugas, dan tidak ada satu pun penanda di sana yang menyatakan baris mana yang
	// baru, mana yang berubah, dan mana yang dibuang. Penggantian menyeluruh adalah
	// satu-satunya tafsiran yang tidak menebak.
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

// PlanRepo adalah seam BACA-SAJA ke master plan dan jaminan Travel milik GISFW.
//
// Penegakan `D-03`: data plan dan jaminan dimiliki tim lain, dan modul ini tidak boleh
// menulisnya.
//
// # Kenapa plan dan jaminan berada di SATU seam
//
// Karena keduanya berasal dari SATU tabel. `Activity/SetspreadingtoCoverage-Act.xml`
// menyimpan pembenaran peringatan Pega yang menyebutkannya apa adanya:
//
//	"ngambil data coverage bukan dari coverage travel tapi dari m_plantravel"
//
// dan rule yang membacanya bernama `GetDataMasterCoverageTravel_m_plantravel`. Kedua
// kelas Pega yang tampak berbeda — `ASM-FW-GISFW-Int-PLANTRAVEL` dan
// `ASM-FW-GISFW-Int-COVERAGETRAVEL` — karena itu dua sudut pandang atas tabel yang sama.
// Memisahkannya menjadi dua seam akan menyiratkan dua sumber yang sebenarnya satu.
type PlanRepo interface {
	// ListPlans mengembalikan plan yang dapat dipilih, terurut menurut namanya.
	ListPlans(ctx context.Context) ([]Plan, error)

	// ListCoverages mengembalikan jaminan yang dapat dipilih beserta plan pemiliknya.
	//
	// Seluruhnya dikembalikan sekaligus, tidak disaring per plan di server seperti
	// `SearchCoverageTravel_RD` yang menerima parameter `plan`. Sebabnya bentuk layarnya
	// berbeda: grid di form dapat memuat banyak baris dengan plan berbeda-beda, dan
	// menyaring di server berarti satu permintaan per baris grid setiap kali plannya
	// berganti. Penyaringannya dikerjakan layar atas daftar yang sudah di tangan.
	ListCoverages(ctx context.Context) ([]CoverageOption, error)
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
type DocumentRepoSelector func(portalAlias string) (DocumentRepo, error)

// PlanRepoSelector memilih PlanRepo milik satu portal entitas.
//
// Terpisah dari kedua selector lain meski ketiganya selalu dipilih bersamaan, karena
// ketiganya mengisi seam yang berbeda: yang satu tabel milik modul ini, yang kedua tabel
// milik modul master induk, dan yang ketiga tabel milik GISFW.
type PlanRepoSelector func(portalAlias string) (PlanRepo, error)
