package mastertipesparepart

import "context"

// # Satu tabel acuan yang dibaca modul ini
//
// Ia milik modul LAIN dan aplikasi ini HANYA MEMBACA dari sini (ADR-0004, penulis tunggal
// per tabel). Penulisnya adalah `masterkategorisparepart`:
//
//	POOLDATA.GCNM_M_SPAREPART_CATEGORY  Kategori — PART_CATEGORY_ID, PART_CATEGORY_NAME
//
// Ia menyuapi dropdown `---PILIH KATEGORI---` pada
// `Section/MasterTipeSparepartHEApproval-Section.xml`, dan nilainya menjadi
// `PART_CATEGORY_ID` setiap baris tipe.
//
// # Kenapa tidak mengimpor paket masterkategorisparepart saja
//
// Karena yang dibutuhkan di sini bukan modulnya melainkan **dua kolom dari tabelnya**, dan
// mengimpor paket lain untuk itu akan mengikat dua modul master yang seharusnya dapat
// berpindah sendiri-sendiri. Pola yang sama dipakai `mastersparepart` dan
// `mastergroupingsparepart`, yang masing-masing mendeklarasikan tipe acuannya sendiri.
//
// Harganya disadari: tipe `Category` di sini, `mastersparepart.Category`, dan
// `masterkategorisparepart.PartCategory` adalah tiga tipe berbeda atas satu tabel.
// Ketiganya memang membawa kolom yang berbeda-beda — yang ketiga membawa status, dua yang
// pertama tidak — dan menyatukannya akan memaksa dropdown membawa kolom yang tidak
// dipakainya.

// Category adalah satu pilihan pada dropdown Kategori.
//
// Asalnya `RDB List/BrowseMasterSparepartCategoryClaimHE-SQL.xml`:
//
//	select PART_CATEGORY_ID as "CityID", PART_CATEGORY_NAME as "City"
//	  from POOLDATA.gcnm_m_sparepart_category where APPROVAL = {TempStatus.City}
//
// PERHATIKAN ALIAS KOLOMNYA. `PART_CATEGORY_ID` dialiaskan `"CityID"` dan
// `PART_CATEGORY_NAME` dialiaskan `"City"` — nama yang sama sekali tidak ada hubungannya
// dengan kota. Pada grid layar ini aliasnya bahkan berbeda lagi: kolom kategori dialiaskan
// `"District"` dan namanya `"DistrictID"`, sehingga ID dan NAMA tertukar posisi terhadap
// pola "…ID" yang biasa. Itu persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2
// catat. Alias itu TIDAK dibawa; di sini keduanya bernama ID dan Name.
type Category struct {
	// ID adalah kolom PART_CATEGORY_ID, disimpan sebagai
	// GCNM_M_SPAREPART_TYPE.PART_CATEGORY_ID.
	ID string

	// Name adalah kolom PART_CATEGORY_NAME — yang dilihat pengguna pada dropdown.
	//
	// Ia TIDAK ikut disimpan ke tabel tipe. Nama kategori pada setiap baris dibaca lewat
	// JOIN saat daftar dimuat, sehingga kategori yang berganti nama langsung terlihat baru
	// di seluruh barisnya — perilaku yang sama dengan sistem lama, yang juga menggabungkan
	// kedua tabel alih-alih menyalin namanya.
	Name string
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan lookup.
//
// Sistem lama tidak membatasinya: daftarnya dimuat penuh ke klipboard lalu disaring di
// peramban. Batasnya di sini, bukan di frontend — memotong di peramban berarti barisnya
// sudah terlanjur dibaca, dikirim, dan diurai.
//
// Lima ratus, sama dengan `mastersparepart.MaxLookupRows`. Tabel kategori adalah master
// penggolongan, bukan master transaksi: jumlahnya berada pada orde puluhan sampai ratusan,
// sehingga batas ini tidak akan tersentuh pemakaian yang wajar. Bila kelak tersentuh,
// daftar akan terpotong diam-diam — dan itulah yang membuat pencacahnya ikut dilaporkan;
// lihat usecase.Service.Options.
const MaxLookupRows = 500

// LookupRepo adalah seam ke tabel acuan kategori, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini" — dan karena tabelnya dimiliki modul lain.
// Keduanya tetap dipilih bersama lewat RepoSelector, karena keduanya selalu berasal dari
// koneksi entitas yang sama.
type LookupRepo interface {
	// ListCategories mengembalikan kategori yang sudah DISETUJUI.
	//
	// # Penyaring APPROVAL = '1' adalah REKONSTRUKSI, bukan pembacaan
	//
	// Rule yang mengisi dropdown layar ini tidak dapat ditelusuri di export. Dropdown-nya
	// terikat pada page `TempSparepartTypeClaimHE2` lewat `pyListSource=pageList`, dan page
	// itu **tidak dimuat oleh satu pun rule di antara 2.634 berkas export** (`R-16`) —
	// ketiga activity layar ini sudah diperiksa satu per satu:
	//
	//	Activity/SetMasterTipeSparepart_act        memuat TempInputTipeSparepart, TempSparepartTypeClaimHE
	//	Activity/SetMasterTipeSparepartReject_act  memuat TempReject, TempKategori, TempSparepartTypeClaimHE
	//	Activity/GCNMBrowseMasterSparepartType_act memuat TempSparepartTypeClaimHE
	//
	// Tidak satu pun menyentuh `...HE2`.
	//
	// Yang DAPAT dipastikan adalah pola Pega sendiri: **setiap dropdown yang menawarkan
	// master sebagai pilihan menyaring APPROVAL = '1'**, dan penyaring '0' hanya dipakai
	// layar ANTREAN PERSETUJUAN. Dua rule membuktikannya:
	//
	//	RDB List/BrowseTipeSparepart-SQL.xml               where APPROVAL = '1'   dipakai pemilih
	//	Activity/GCNMBrowseMasterSparepartCategory_act2    approval = 1           dipakai pemilih
	//	Activity/GCNMBrowseMasterSparepartCategory_act     approval = 0           dipakai antrean
	//
	// Dropdown di layar ini adalah pemilih, sehingga '1' yang dipakai. Keputusan Work Owner
	// 2026-09-21: "sesuai Pega".
	//
	// Dinyatakan di sini supaya ia dapat diuji ulang begitu rule aslinya tiba — bukan
	// tersamar sebagai fakta.
	//
	// # Akibat yang harus disadari
	//
	// **Kategori yang belum disetujui tidak muncul di dropdown**, sehingga tipe tidak dapat
	// ditautkan kepadanya. Lebih jauh: kategori yang sudah disetujui lalu DISUNTING kembali
	// menunggu — `masterkategorisparepart` menetapkan APPROVAL := "0" pada setiap
	// penyimpanan — dan selama menunggu ia hilang dari dropdown ini. Tipe yang sudah
	// menunjuknya tetap menyimpan ID-nya dan tetap menampilkan namanya lewat JOIN; yang
	// tidak dapat dilakukan hanyalah MEMILIHNYA untuk tipe baru.
	//
	// Tanpa pencarian: daftarnya pendek, dan layar lama pun memuatnya penuh sekali saat
	// form dibuka.
	ListCategories(ctx context.Context) ([]Category, error)
}
