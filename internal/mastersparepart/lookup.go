package mastersparepart

import "context"

// # Dua tabel acuan yang dibaca modul ini
//
// Keduanya milik sistem lain dan aplikasi ini HANYA MEMBACA (ADR-0004, penulis tunggal per
// tabel). Tidak satu pun disentuh oleh penyimpanan master sparepart:
//
//	POOLDATA.GCNM_M_SPAREPART_CATEGORY  Kategori — PART_CATEGORY_ID, PART_CATEGORY_NAME
//	POOLDATA.GCNM_M_SPAREPART_TYPE      Tipe     — PART_SECTION_ID, PART_SECTION_NAME,
//	                                               PART_CATEGORY_ID
//
// Keduanya menyuapi isian yang di Pega berupa `pxAutoComplete` pada
// `Section/BrowseMasterSparepartHEApproval-Section.xml`, dan keduanya punya layar
// masternya sendiri — MENU_ID 33 dan 34 — yang belum dibangun.
//
// # Kenapa penyaringnya APPROVAL = '1' dan bukan '0'
//
// Karena itulah yang dilakukan layar Master Sparepart. Export memuat TIGA rule yang
// membaca kedua tabel ini dengan penyaring yang berbeda-beda, dan hanya SATU yang benar
// dipakai layar ini — `Activity/BrowseTipeKategoriPart-Act.xml`, yang dirujuk ketiga
// section tab Master Sparepart:
//
//	TempStatus.City := "1"
//	BrowseMasterSparepartCategoryClaimHE  ->  where APPROVAL = {TempStatus.City}
//	BrowseTipeSparepart                   ->  where APPROVAL = '1'
//
// Kedua rule lain yang memakai `APPROVAL = '0'` —
// `RDB List/BrowseSparepartCategoryClaimHE-SQL.xml` dan
// `RDB List/BrowseSparepartTipeClaimHE2-SQL.xml` — dipanggil dari layar Master Kategori
// dan Master Tipe, bukan dari sini. Penyaring '0' di sana justru masuk akal: layar itu
// mengurus antrean persetujuan kategori dan tipe.
//
// Akibat yang harus disadari: **kategori atau tipe yang belum disetujui tidak muncul di
// dropdown**, sehingga sparepart tidak dapat ditautkan kepadanya. Itu perilaku sistem
// lama, dan ia konsisten secara bisnis.

// Category adalah satu kategori pada lookup Kategori Sparepart.
//
// Asalnya `RDB List/BrowseMasterSparepartCategoryClaimHE-SQL.xml`:
//
//	select PART_CATEGORY_ID as "CityID", PART_CATEGORY_NAME as "City"
//	  from POOLDATA.gcnm_m_sparepart_category where APPROVAL = {TempStatus.City}
//
// PERHATIKAN ALIAS KOLOMNYA. `PART_CATEGORY_ID` dialiaskan `"CityID"` dan
// `PART_CATEGORY_NAME` dialiaskan `"City"` — nama yang sama sekali tidak ada hubungannya
// dengan kota. Itu persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat:
// developer memaksa nama kolom agar cocok dengan property klipboard Pega yang sudah ada.
// Alias itu TIDAK dibawa; di sini keduanya bernama ID dan Name.
type Category struct {
	// ID adalah kolom PART_CATEGORY_ID, disimpan sebagai SPAREPART_HE.KATEGORI_SPART.
	ID string

	// Name adalah kolom PART_CATEGORY_NAME — yang dilihat pengguna.
	Name string
}

// PartType adalah satu tipe pada lookup Tipe Sparepart.
//
// Asalnya `RDB List/BrowseTipeSparepart-SQL.xml`:
//
//	select PART_SECTION_ID as "CityID", PART_SECTION_NAME as "City",
//	       PART_CATEGORY_ID as "District"
//	  from POOLDATA.gcnm_m_sparepart_type where APPROVAL = '1'
//	 order by PART_SECTION_ID desc
//
// Namanya **PartType**, bukan `Type` — `type` adalah kata kunci Go dan tidak dapat dipakai
// sebagai nama tipe.
//
// # Kenapa CategoryID ikut dibawa
//
// Karena Tipe BERCABANG dari Kategori: `RDB List/BrowseSparepartTypeClaimHE_sql-SQL.xml`
// menggabungkan keduanya lewat `A.PART_CATEGORY_ID = B.PART_CATEGORY_ID`. Layar memakainya
// untuk mempersempit daftar Tipe begitu Kategori dipilih — tanpa itu, petugas memilih dari
// seluruh tipe yang ada dan dapat menautkan tipe milik kategori lain.
//
// Penyaringannya dilakukan di LAYAR, bukan di kueri: daftarnya pendek, dan memuat keduanya
// sekali saat form dibuka persis seperti `BrowseTipeKategoriPart` yang menarik keduanya
// dalam satu jalan.
type PartType struct {
	// ID adalah kolom PART_SECTION_ID, disimpan sebagai SPAREPART_HE.TIPE_SPART.
	ID string

	// Name adalah kolom PART_SECTION_NAME — yang dilihat pengguna.
	Name string

	// CategoryID adalah kolom PART_CATEGORY_ID — induk tipe ini.
	CategoryID string
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan sebuah lookup.
//
// Sistem lama tidak membatasinya: keduanya dimuat penuh ke klipboard lalu disaring di
// peramban. Batasnya di sini, bukan di frontend — memotong di peramban berarti barisnya
// sudah terlanjur dibaca, dikirim, dan diurai.
//
// Lima ratus dipilih karena kedua tabel acuan ini adalah master penggolongan, bukan master
// transaksi: jumlah kategori dan tipe suku cadang berada pada orde puluhan sampai ratusan,
// sehingga batas ini tidak akan tersentuh pemakaian yang wajar. Bila kelak tersentuh,
// daftar akan terpotong diam-diam — dan itulah yang membuat pencacahnya ikut dilaporkan;
// lihat usecase.Service.Options.
const MaxLookupRows = 500

// LookupRepo adalah seam ke kedua tabel acuan, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini" — dan karena kedua tabelnya dimiliki sistem
// lain. Keduanya tetap dipilih bersama lewat RepoSelector, karena keduanya selalu berasal
// dari koneksi entitas yang sama.
type LookupRepo interface {
	// ListCategories mengembalikan kategori yang sudah disetujui.
	//
	// Tanpa pencarian: daftarnya pendek, dan `BrowseTipeKategoriPart` pun memuatnya penuh
	// sekali saat layar dibuka.
	ListCategories(ctx context.Context) ([]Category, error)

	// ListTypes mengembalikan tipe yang sudah disetujui, beserta kategori induknya.
	//
	// Tanpa pencarian, dengan alasan yang sama. Penyaringan menurut kategori dilakukan di
	// layar; lihat PartType.
	ListTypes(ctx context.Context) ([]PartType, error)
}
