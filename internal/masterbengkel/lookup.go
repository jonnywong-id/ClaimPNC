package masterbengkel

import "context"

// # Tiga tabel acuan yang dibaca modul ini
//
// Ketiganya milik sistem lain dan aplikasi ini HANYA MEMBACA (ADR-0004, penulis tunggal
// per tabel). Tidak satu pun disentuh oleh penyimpanan master bengkel:
//
//	GENERAL.LST_USER_ASURANSI + LST_DET_CABANG  Cabang  — asal CABANG_ID dan NAMA_CABANG
//	CITY                                        Kota    — asal CITY_ID dan NAMA_KABUPATEN
//	GENERAL.LST_BANK_GROUP                      Bank    — asal BANK_ID dan NAMA_BANK
//
// Ketiganya menyuapi isian yang di Pega berupa `pxAutoComplete` dan `pxDropdown` pada
// `Section/BrowseMasterHEApprove-Section.xml`.

// Branch adalah satu cabang pada lookup Cabang.
//
// Asalnya `RDB List/BrowseCabangBengkelHE-SQL.xml`:
//
//	select ldc_nama as BRANCHNAME, cab_id as BRANCH
//	  from general.lst_user_asuransi a, lst_det_cabang b
//	 where b.ldc_id = a.cab_id
//	   and b.ldc_id = a.rep_cab_id
//	   and a.ldi_id = '0076'
//	   and a.sts_aktif = '1'
//
// # Penyaring `ldi_id = '0076'` direplikasi apa adanya
//
// Ia kode aplikasi yang tertanam di dalam kueri, persis bentuk hardcode yang `D-15`
// perintahkan menjadi konfigurasi. Tidak ada satu pun keterangan di export tentang apa
// arti `0076`, sehingga mengangkatnya menjadi konfigurasi berarti menebak nilainya untuk
// entitas selain yang sekarang. Dibiarkan sebagai literal — ia bukan masukan pengguna,
// sehingga bukan celah injeksi — dan dicatat sebagai utang.
//
// # Penyaring `b.ldc_id = a.rep_cab_id` juga direplikasi
//
// Digabung dengan `b.ldc_id = a.cab_id` pada baris sebelumnya, keduanya menuntut
// `cab_id = rep_cab_id`: hanya cabang yang MEWAKILI DIRINYA SENDIRI yang muncul. Cabang
// yang diwakili cabang lain tersaring habis. Itu tampak disengaja, dan dipertahankan.
type Branch struct {
	// ID adalah kolom CAB_ID, disimpan sebagai BENGKEL_HE.CABANG_ID.
	ID string

	// Name adalah kolom LDC_NAMA, disimpan sebagai BENGKEL_HE.NAMA_CABANG.
	Name string
}

// City adalah satu kota pada lookup Kota.
//
// Asalnya `Report Definition/BrowseCity_RD-RD.xml` atas kelas `ASM-FW-GISFW-Int-CITY`,
// yang dipakai kontrol autocomplete dengan nilai `.Note`.
//
// PERHATIKAN NAMA KOLOMNYA: nama kota tersimpan di kolom bernama **NOTE**, bukan NAME
// maupun CITYNAME. Itu terbaca dari kueri lain yang membaca tabel yang sama, misalnya
// `RDB List/BrowseRW_SQL-SQL.xml`:
//
//	(select distinct a.note from city a where a.id = b.cityid)
//
// Salah membaca kolomnya sekali saja berarti lookup Kota menampilkan daftar kosong.
type City struct {
	// ID adalah kolom ID, disimpan sebagai BENGKEL_HE.CITY_ID.
	ID string

	// Name adalah kolom NOTE, disimpan sebagai BENGKEL_HE.NAMA_KABUPATEN.
	Name string
}

// Bank adalah satu bank pada `GENERAL.LST_BANK_GROUP`.
//
// Asalnya `Report Definition/BrowseBankGroup-RD.xml`, yang dipakai autocomplete "NAMA
// BANK" dengan nilai `.BANK_GROUP`.
//
// Tabel yang sama dibaca modul Master Rekening dan Master Auto Claim. Tipe ini sengaja
// TIDAK dipakai bersama: modul tidak saling mengimpor, dan tipe bersama akan membuat
// perubahan di satu modul menyeret modul lain. Yang dipakai bersama adalah tabelnya,
// bukan kodenya.
type Bank struct {
	// Code adalah kolom LBG_ID, disimpan sebagai BENGKEL_HE.BANK_ID.
	//
	// Berbeda dari Master Auto Claim yang hanya menyimpan nama banknya, tabel bengkel
	// punya kolom kodenya sendiri — sehingga keduanya disimpan berpasangan.
	Code string

	// Name adalah kolom BANK_GROUP, disimpan sebagai BENGKEL_HE.NAMA_BANK.
	Name string
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan sebuah pencarian lookup.
//
// Sistem lama tidak membatasinya: ketiga sumbernya dimuat penuh ke klipboard lalu
// disaring di peramban. Batasnya di sini, bukan di frontend — memotong di peramban
// berarti barisnya sudah terlanjur dibaca, dikirim, dan diurai.
const MaxLookupRows = 50

// MinLookupKeyword adalah panjang minimum kata kunci pencarian Kota.
//
// Hanya Kota yang memakainya: daftar kota berbaris ribuan, sementara daftar Cabang dan
// Bank pendek dan dimuat penuh — sama seperti di Pega, yang menyuapi keduanya dari satu
// muatan penuh.
const MinLookupKeyword = 2

// LookupRepo adalah seam ke ketiga tabel acuan, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini" — dan karena ketiga tabelnya dimiliki sistem
// lain. Keduanya tetap dipilih bersama lewat RepoSelector, karena keduanya selalu
// berasal dari koneksi entitas yang sama.
type LookupRepo interface {
	// ListBranches mengembalikan seluruh cabang yang sah.
	//
	// Tanpa pencarian: daftarnya pendek, dan `ChooseCabangBengkelHE` pun memuatnya penuh
	// sekali saat layar dibuka.
	ListBranches(ctx context.Context) ([]Branch, error)

	// SearchCities mencari kota menurut namanya.
	//
	// Berbeda dari kedua lookup lain, ia MEMAKAI kata kunci: tabel CITY berbaris ribuan,
	// dan memuatnya penuh ke peramban untuk daftar yang hanya akan dipilih satu adalah
	// biaya yang tidak perlu dibayar berulang kali.
	SearchCities(ctx context.Context, keyword string) ([]City, error)

	// ListBanks mengembalikan seluruh bank.
	ListBanks(ctx context.Context) ([]Bank, error)
}
