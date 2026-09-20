package mastersupplier

import "context"

// # Empat tabel acuan yang dibaca modul ini
//
// Keempatnya milik sistem lain dan aplikasi ini HANYA MEMBACA (ADR-0004, penulis tunggal
// per tabel). Tidak satu pun disentuh oleh penyimpanan master supplier:
//
//	M_BRANCH                Cabang  — asal NAMA_CABANG
//	CITY                    Kota    — asal KOTA
//	COUNTRY                 Negara  — asal NEGARA
//	GENERAL.LST_BANK_GROUP  Bank    — asal BANK
//
// Keempatnya menyuapi isian yang di Pega berupa `pxDropdown` dan `pxAutoComplete` pada
// `Section/CreateMasterSupplier_Sec-Section.xml`.
//
// # Yang disimpan adalah NAMA, bukan kode
//
// Itu berbeda dari Master Bengkel, dan bukan pilihan melainkan bentuk datanya:
// `RDB List/GetDataEditMasterSupller-SQL.xml` membaca `KOTA`, `NAMA_CABANG`, `NEGARA`,
// dan `BANK` — dan **tidak ada satu pun kunci berpasangan yang memuat kodenya**. Tidak
// ada `KOTA_ID`, tidak ada `CABANG_ID`, tidak ada `BANK_ID`.
//
// Akibat yang harus disadari: bila sebuah cabang berganti nama di M_BRANCH, baris
// supplier yang menyebut nama lamanya TIDAK ikut berubah dan tidak lagi cocok dengan
// satu pun pilihan di dropdown. Itu keadaan sistem lama, dan membetulkannya menuntut
// kolom kode yang tabelnya tidak punya — perubahan skema (`D-63`), bukan keputusan modul
// ini. Yang dikerjakan di sini: nilai tersimpan SELALU ikut ditawarkan sebagai pilihan,
// supaya menyunting baris lama tidak diam-diam mengosongkan cabangnya.

// Branch adalah satu cabang pada dropdown Cabang.
//
// Asalnya `Report Definition/BrowseBranchForGKM_RD-RD.xml` atas kelas
// `ASM-FW-GISFW-Int-BRANCH`.
//
// KELASNYA MEMETAKAN KE TABEL JSON, bukan tabel berkolom. Itu terbaca dari kueri lain
// yang membaca nama cabang dari ID-nya, misalnya `RDB List/GetBranchName-SQL.xml`:
//
//	select b.JSONDATA.Name as LCA_NOTE from M_BRANCH b where b.id = {…}
//
// dan dipakai dengan bentuk yang sama di tujuh rule `SearchKlaimBy*_RDB`. Salah membaca
// tabelnya sekali saja berarti dropdown Cabang menampilkan daftar kosong.
type Branch struct {
	// ID adalah kolom ID. Ia TIDAK disimpan ke dokumen supplier — lihat catatan di atas.
	ID string

	// Name adalah kunci JSON `Name`, disimpan sebagai kunci NAMA_CABANG.
	Name string
}

// City adalah satu kota pada dropdown Kota.
//
// Asalnya `Report Definition/BrowseCity_RD-RD.xml` atas kelas `ASM-FW-GISFW-Int-CITY`.
//
// PERHATIKAN NAMA KOLOMNYA: nama kota tersimpan di kolom bernama **NOTE**, bukan NAME
// maupun CITYNAME. Itu terbaca dari `RDB List/BrowseRW_SQL-SQL.xml`:
//
//	(select distinct a.note from city a where a.id = b.cityid)
//
// Tabel dan kolom yang sama dibaca Master Bengkel lewat kueri miliknya sendiri. Keduanya
// sengaja TIDAK dipakai bersama: modul tidak saling mengimpor, dan kueri bersama akan
// membuat perubahan di satu modul menyeret modul lain.
type City struct {
	// ID adalah kolom ID.
	ID string

	// Name adalah kolom NOTE, disimpan sebagai kunci KOTA.
	Name string
}

// Country adalah satu negara pada lookup Negara.
//
// Asalnya `Report Definition/BrowseCountry_RD-RD.xml` atas kelas
// `ASM-FW-GISFW-Int-COUNTRY`, yang memilih `.ID` dan `.Country`.
//
// Tabelnya terbaca dari `Database/UPDATEREAS.prc:41,55`:
//
//	SELECT ID INTO NEGARA_ID FROM COUNTRY WHERE COUNTRY = tCOUNTRY;
//
// Perhatikan bahwa TABEL dan KOLOM-nya sama-sama bernama COUNTRY — kueri yang menyebut
// `COUNTRY` tanpa alias akan terbaca sebagai salah satunya tergantung tempatnya.
//
// Isiannya di Pega adalah `pxAutoComplete`, satu-satunya di form ini. Di sini ia
// diperlakukan sama dengan dropdown lain: daftarnya pendek — report definition-nya
// membatasi diri di 500 baris — sehingga dimuat sekali dan disaring di peramban.
type Country struct {
	// ID adalah kolom ID.
	ID string

	// Name adalah kolom COUNTRY, disimpan sebagai kunci NEGARA.
	Name string
}

// Bank adalah satu bank pada dropdown Bank.
//
// Asalnya `Report Definition/BrowseBankGroup-RD.xml` atas `GENERAL.LST_BANK_GROUP`.
//
// Di Pega dropdown ini disuapi page list `DataBank.pxResults`, bukan report definition
// langsung. Halaman itu diisi `Activity/GetDataMasterBank-Act.xml` — yang isinya ternyata
// pemuat Master Rekening lengkap dengan penyaring persetujuan dan komite, bukan pemuat
// daftar bank. Yang dibutuhkan isian ini hanyalah nama banknya, sehingga yang dipakai di
// sini adalah sumber bank yang sebenarnya, sama dengan tiga modul master lain.
//
// Tabel yang sama dibaca Master Rekening, Master Auto Claim, dan Master Bengkel lewat
// kueri masing-masing. Keempatnya sengaja tidak dipakai bersama; yang dipakai bersama
// adalah tabelnya, bukan kodenya.
type Bank struct {
	// Code adalah kolom LBG_ID.
	//
	// Ia TIDAK disimpan ke dokumen supplier — dokumennya tidak punya kunci BANK_ID. Ia
	// dikirim ke layar hanya sebagai pembeda bila ada dua bank bernama mirip.
	Code string

	// Name adalah kolom BANK_GROUP, disimpan sebagai kunci BANK.
	Name string
}

// CodeOption adalah satu pilihan pada kelima dropdown bersandi.
//
// # Kenapa pilihannya dibaca dari DATA, bukan dari daftar tetap
//
// Kelima isian — STS_REKANAN, JENIS_STATUS, JENIS_SUPPLIER, STS_AKTIF_PROMLIST, dan
// STS_AUTOPAYMENT — dirender `pxDropdown` bersumber `associated` di Pega, artinya daftar
// pilihannya ada di rule **Field Value**. Tidak satu pun rule Field Value ikut di export
// (`R-16`), sehingga daftarnya tidak diketahui.
//
// Ada tiga jalan, dan hanya satu yang tidak menebak:
//
//  1. Mengarang daftarnya. DITOLAK — nilai yang salah tersimpan ke master yang menentukan
//     ke rekening siapa uang berpindah, dan salahnya tidak terlihat di layar mana pun.
//  2. Mengubah isiannya menjadi teks bebas. DITOLAK — layar lamanya dropdown, dan salah
//     ketik pada kolom bersandi tersimpan diam-diam.
//  3. Menawarkan nilai yang BENAR-BENAR ADA di data. Itu yang dipakai.
//
// Ketiga tidak sempurna dan keterbatasannya harus dinyatakan: nilai sah yang belum pernah
// dipakai satu baris pun TIDAK akan muncul. Karena itu tiga di antaranya diberi nilai
// dasar yang terbukti dari percabangan activity — lihat DefaultCodeOption — dan
// daftarnya selalu digabung dengan nilai yang sedang dipakai baris yang sedang disunting.
//
// Begitu daftar Field Value diterima Work Owner, yang berubah hanyalah sumber daftarnya;
// bentuk isian dan yang tersimpan tidak berubah sama sekali.
type CodeOption struct {
	// Value adalah sandi yang benar-benar tersimpan di dokumen.
	Value string

	// Label adalah sebutan yang dibaca pengguna.
	//
	// Untuk sandi yang artinya terbukti dari export ia berisi artinya; untuk sandi yang
	// hanya ditemukan di data ia berisi sandinya sendiri. Keduanya sengaja dibedakan
	// supaya petugas dapat melihat mana yang artinya diketahui dan mana yang tidak,
	// alih-alih membaca tebakan yang tampak meyakinkan.
	Label string
}

// CodeSet adalah kelima daftar pilihan bersandi, dikirim sekaligus.
//
// Satu perjalanan untuk lima daftar, bukan lima. Kelimanya dibaca dari tabel yang SAMA
// dalam satu kali pemindaian, sehingga memecahnya menjadi lima endpoint berarti lima kali
// memindai tabel yang sama untuk mengisi satu form.
type CodeSet struct {
	PartnerStatus []CodeOption
	SupplyType    []CodeOption
	SupplierType  []CodeOption
	Active        []CodeOption
	AutoPayment   []CodeOption
}

// DefaultCodeOption mengembalikan nilai dasar yang artinya TERBUKTI dari export.
//
// Ketiganya, dan hanya ketiganya:
//
//	JENIS_STATUS        "1" = Heavy Equipment, "0" = selain itu
//	                    Activity/CreateNewMasterSupplier_post step 7 dan
//	                    Activity/GetDataSupplier_pre step 6.3 saling membalik keduanya
//	STS_AKTIF_PROMLIST  "1" = aktif, "0" = tidak aktif
//	                    Activity/CreateNewMasterSupplier_post step 6 menetapkan "0" pada
//	                    supplier baru, dan EditMasterSupplier_post step 12 hanya meminta
//	                    persetujuan bila nilainya "1" atau kosong
//	STS_AUTOPAYMENT     isiannya pxCheckbox, sehingga hanya punya dua keadaan
//
// STS_REKANAN dan JENIS_SUPPLIER TIDAK ada di sini. Master Bengkel punya bukti bahwa
// `STATUS_REKANAN = "0"` berarti bukan rekanan, tetapi itu kolom tabel LAIN pada modul
// lain — memindahkan artinya ke sini berarti mengandaikan kedua master memakai sandi yang
// sama, dan tidak ada satu pun bukti untuk itu. Keduanya diisi dari data saja.
func DefaultCodeOption() CodeSet {
	return CodeSet{
		SupplyType: []CodeOption{
			{Value: SupplyTypeHeavyEquipment, Label: "Heavy Equipment"},
			{Value: SupplyTypeOther, Label: "Selain Heavy Equipment"},
		},
		Active: []CodeOption{
			{Value: ActiveYes, Label: "Aktif"},
			{Value: ActiveNo, Label: "Tidak aktif"},
		},
		AutoPayment: []CodeOption{
			{Value: ActiveYes, Label: "Ya"},
			{Value: ActiveNo, Label: "Tidak"},
		},
	}
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan sebuah lookup.
//
// Sistem lama membatasinya berbeda-beda: `BrowseCountry_RD` dan `BrowseBankGroup` di 500
// baris, sedangkan `BrowseCity_RD` dan `BrowseBranchForGKM_RD` **tidak membatasinya sama
// sekali** (`pyMaxRecords=0`) sehingga keduanya dimuat penuh ke klipboard lalu disaring di
// peramban.
//
// Batasnya di sini, bukan di frontend — memotong di peramban berarti barisnya sudah
// terlanjur dibaca, dikirim, dan diurai.
const MaxLookupRows = 50

// MinLookupKeyword adalah panjang minimum kata kunci pencarian Kota.
//
// Hanya Kota yang memakainya: tabel CITY berbaris ribuan, sementara Cabang, Negara, dan
// Bank berdaftar pendek dan dimuat penuh — sama seperti di Pega.
const MinLookupKeyword = 2

// LookupRepo adalah seam ke keempat tabel acuan dan kelima daftar sandi, SATU portal.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini". Keduanya tetap dipilih bersama lewat
// RepoSelector, karena keduanya selalu berasal dari koneksi entitas yang sama.
type LookupRepo interface {
	// ListBranches mengembalikan seluruh cabang yang sah.
	//
	// Tanpa pencarian: report definition-nya pun memuatnya penuh sekali saat layar
	// dibuka.
	ListBranches(ctx context.Context) ([]Branch, error)

	// SearchCities mencari kota menurut namanya.
	//
	// Berbeda dari ketiga lookup lain, ia MEMAKAI kata kunci: tabel CITY berbaris ribuan,
	// dan memuatnya penuh ke peramban untuk daftar yang hanya akan dipilih satu adalah
	// biaya yang tidak perlu dibayar berulang kali.
	SearchCities(ctx context.Context, keyword string) ([]City, error)

	// ListCountries mengembalikan seluruh negara.
	ListCountries(ctx context.Context) ([]Country, error)

	// ListBanks mengembalikan seluruh bank.
	ListBanks(ctx context.Context) ([]Bank, error)

	// ListCodes mengembalikan sandi yang benar-benar dipakai baris yang ada.
	//
	// Hasilnya digabung dengan DefaultCodeOption oleh lapisan aplikasi, bukan di sini:
	// adapter menjawab "apa yang ada di data", dan yang memutuskan bagaimana itu
	// dilengkapi adalah aturan, bukan penyimpanan.
	ListCodes(ctx context.Context) (CodeSet, error)
}
