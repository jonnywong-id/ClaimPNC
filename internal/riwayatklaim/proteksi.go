package riwayatklaim

import (
	"context"
	"strings"
	"time"
)

// ModuleKey adalah nama modul yang dipakai gerbang proteksi data untuk mengenali layar
// ini di POOLDATA.MST_PROTEKSI_DATA_PNC.
//
// Nilainya adalah nama HARNESS sistem lama, dan itu disengaja: kolom MODUL pada tabel
// proteksi berisi nama harness, bukan nama menu. `Activity/InsertLogProteksiDataKlaimMasking`
// mengisinya dari `SETTINGMASINGDATA.pyCaseID`, yang nilai bawaannya disetel
// `@if(SETTINGMASINGDATA.pyCaseID=="","PNCSearchKlaim",…)`.
//
// Ia karena itu TIDAK dinamai ulang mengikuti `D-19`: ia bukan istilah domain melainkan
// nilai data yang sudah tersimpan di baris-baris milik sistem lama, dan menggantinya
// berarti tidak satu pun pengguna yang terdaftar hari ini dikenali.
const ModuleKey = "PNCSearchKlaim"

// Protection adalah baris proteksi data milik satu pengguna untuk satu modul.
//
// Sumbernya POOLDATA.MST_PROTEKSI_DATA_PNC, dibaca dengan kueri yang di sistem lama
// berbunyi — perhatikan aliasnya, yang sekali lagi menyesatkan:
//
//	select LOGSEEN    as "City",        -- jatah LIHAT data (layar rincian)
//	       LOGSEARCH  as "CityID",      -- jatah PENCARIAN  (layar ini)
//	       STS_NOTELP as "Country",     -- masking nomor telepon
//	       STS_EMAIL  as "CountryID",   -- masking surel
//	       STS_KTP    as "Province",    -- masking nomor KTP
//	       SUBMODUL   as "ProvinceID"
//	  from POOLDATA.MST_PROTEKSI_DATA_PNC
//	 where LOGIN = <operator> and MODUL = 'PNCSearchKlaim'
//
// Arti kelima alias itu tidak dapat ditebak dari namanya; ia hanya terbaca dari komentar
// langkah 3 activity-nya, yang menyebut "1: log seen(city), 2: log search (CityID),
// 3 STS_telp(Country), 4 sts_email(CountryID), 5: stsktp (Province)".
type Protection struct {
	// Login adalah pemilik baris ini.
	Login string

	// SearchQuota adalah jatah PENCARIAN yang tersisa menurut master — kolom LOGSEARCH.
	//
	// Inilah jatah yang dipakai layar ini. Jatah LIHAT data (LOGSEEN) milik layar
	// rincian klaim, dan tidak disentuh di sini.
	SearchQuota int

	// ViewQuota adalah jatah LIHAT data — kolom LOGSEEN.
	//
	// Ia dibaca dan dibawa apa adanya meski layar ini tidak memakainya, supaya keadaan
	// proteksi seorang pengguna dapat ditampilkan utuh tanpa membaca tabelnya dua kali.
	ViewQuota int

	// SubModules adalah daftar submodul yang boleh dibuka pengguna ini — kolom SUBMODUL,
	// tersimpan sebagai teks berpemisah koma.
	SubModules []string

	// Tiga penanda masking. Keduabelasnya dibaca, tetapi TIDAK SATU PUN berlaku di layar
	// ini: grid riwayat klaim tidak punya kolom nomor telepon, surel, maupun nomor KTP.
	//
	// Masking berlaku di layar RINCIAN klaim (`setDataViewKlaim_Act`, yang menyetel
	// "nomor telp Visivility"), dan itu modul tersendiri. Ketiganya dibawa di sini
	// supaya modul itu kelak membaca keadaan yang sama, bukan menafsirkannya ulang.
	MaskPhone  bool
	MaskEmail  bool
	MaskIDCard bool
}

// Access adalah keadaan gerbang setelah layar dibuka.
//
// Ia yang dikirim ke layar, bukan Protection mentah: yang perlu diketahui pengguna adalah
// berapa jatah pencariannya yang tersisa, bukan isi kolom master.
type Access struct {
	// QuotaTotal adalah jatah menurut master proteksi.
	QuotaTotal int

	// QuotaUsed adalah jumlah pencarian yang sudah dicatat aplikasi ini.
	QuotaUsed int

	// QuotaRemaining adalah sisa jatah, tidak pernah negatif.
	QuotaRemaining int
}

// Usage adalah satu pemakaian jatah yang dicatat aplikasi ini.
//
// # Kenapa ia tabel milik aplikasi, bukan tabel milik sistem lama
//
// Sistem lama mengurangi jatah dengan `update POOLDATA.MST_PROTEKSI_DATA_PNC set
// LOGSEARCH = <sisa-1>`. Menirunya berarti aplikasi ini dan Pega sama-sama menulis satu
// tabel selama masa paralel — tepat yang dilarang `P-1`, dan akibatnya bukan galat
// melainkan jatah yang saling menimpa tanpa jejak.
//
// Work Owner memutuskan 2026-09-20 penulisannya diarahkan ke tabel baru milik aplikasi
// (`migrations/0004`), mengikuti pola yang sama dengan modul Pelaporan Klaim. Master
// tetap DIBACA apa adanya; sisa jatah dihitung sebagai jatah master dikurangi pemakaian
// yang tercatat di sini.
//
// # Kenapa nilai pencariannya ikut dicatat
//
// Karena itulah gunanya jejak ini. `D-59` menetapkan tidak ada pemisahan tugas, sehingga
// jejak audit menjadi satu-satunya kontrol pengimbang — dan "siapa mencari data siapa"
// tidak terjawab bila yang tercatat hanya "seseorang melakukan pencarian".
//
// Nilainya tinggal di basis data dan TIDAK PERNAH ikut ke log aplikasi maupun ke dokumen
// yang di-commit (`D-69`).
type Usage struct {
	// Login adalah pengguna yang memakai jatahnya.
	Login string

	// Module adalah layar yang dipakai — selalu ModuleKey di modul ini.
	Module string

	// SearchTypeCode adalah kode tipe pencarian yang dipilih, kosong bila jatah dipakai
	// saat layar dibuka tanpa pencarian.
	SearchTypeCode string

	// SearchValue adalah nilai yang dicari, apa adanya.
	SearchValue string

	// ConsumesQuota menyatakan baris ini MENGURANGI jatah, bukan sekadar mencatat.
	//
	// Keduanya perlu dibedakan karena sistem lama hanya mengurangi jatah SEKALI, saat
	// layar dibuka — prakondisi langkahnya `TempSearch.SearchType==""`, yang hanya benar
	// sebelum tipe pencarian dipilih. Pencarian yang dijalankan sesudahnya tidak
	// mengurangi jatah sama sekali.
	//
	// Pencariannya tetap DICATAT di sini meski tidak mengurangi jatah, karena jejak
	// auditnya justru ada di situ: tanpa itu yang tercatat hanyalah "seseorang membuka
	// layar", bukan apa yang ia cari.
	ConsumesQuota bool

	// At adalah saat pemakaian, UTC.
	At time.Time
}

// ProtectionRepo adalah seam ke gerbang proteksi data SATU portal.
//
// Pembacaan dan penulisannya sengaja terpisah menjadi dua sumber: Find membaca tabel
// milik sistem lama, CountUsage dan RecordUsage menyentuh tabel milik aplikasi ini.
// Pemisahan itu bukan detail teknis melainkan bentuk nyata `P-1` — lihat Usage.
type ProtectionRepo interface {
	// Find mengembalikan baris proteksi milik satu pengguna untuk satu modul.
	//
	// Nilai kedua false bila barisnya tidak ada. Ketiadaan baris BUKAN galat: ia keadaan
	// yang sah dan sering, dan pemanggilnyalah yang memutuskan artinya.
	Find(ctx context.Context, login, module string) (Protection, bool, error)

	// CountUsage mengembalikan jumlah pemakaian yang MENGURANGI jatah.
	//
	// Baris yang hanya mencatat pencarian tidak ikut terhitung — lihat
	// Usage.ConsumesQuota.
	CountUsage(ctx context.Context, login, module string) (int, error)

	// RecordUsage mencatat satu pemakaian jatah.
	RecordUsage(ctx context.Context, usage Usage) error
}

// ProtectionRepoSelector memilih ProtectionRepo milik satu portal entitas.
//
// Alasannya sama persis dengan RepoSelector: portal yang tidak dikenal atau koneksinya
// belum hidup WAJIB menghasilkan galat, tidak pernah dialihkan ke portal utama sebagai
// cadangan (`R-20`). Jatah proteksi seorang pengguna di satu badan hukum bukan jatahnya
// di badan hukum lain.
type ProtectionRepoSelector func(portalAlias string) (ProtectionRepo, error)

// Check menghitung keadaan gerbang TANPA memakai jatah.
//
// Ia fungsi murni: seluruh bahannya diberikan pemanggil, dan ia tidak menyentuh
// penyimpanan mana pun. Itulah yang membuat aturan gerbangnya dapat diuji tanpa basis
// data — dan aturan inilah yang menentukan siapa boleh membuka layar berisi nama
// tertanggung dan tanggal lahir.
//
// Urutan pemeriksaannya mengikuti sistem lama:
//
//  1. tidak terdaftar       → ErrNotRegistered
//  2. jatah pencarian habis → ErrQuotaExhausted
//  3. selain itu            → boleh
func Check(protection Protection, registered bool, used int) (Access, error) {
	if !registered {
		return Access{}, ErrNotRegistered
	}

	if used < 0 {
		used = 0
	}

	remaining := protection.SearchQuota - used
	if remaining <= 0 {
		return Access{
			QuotaTotal:     protection.SearchQuota,
			QuotaUsed:      used,
			QuotaRemaining: 0,
		}, ErrQuotaExhausted
	}

	return Access{
		QuotaTotal:     protection.SearchQuota,
		QuotaUsed:      used,
		QuotaRemaining: remaining,
	}, nil
}

// Grant menghitung keadaan gerbang bagi pemakaian yang SEDANG berjalan.
//
// Bedanya dari Check hanya satu: sisa yang dilaporkan sudah memperhitungkan pemakaian
// yang sedang berjalan. Pemanggil mencatat pemakaiannya tepat setelah ini, dan melaporkan
// sisa sebelum pengurangan akan membuat angka di layar selalu satu lebih besar daripada
// kenyataannya.
func Grant(protection Protection, registered bool, used int) (Access, error) {
	access, err := Check(protection, registered, used)
	if err != nil {
		return access, err
	}
	return Access{
		QuotaTotal:     access.QuotaTotal,
		QuotaUsed:      access.QuotaUsed + 1,
		QuotaRemaining: access.QuotaRemaining - 1,
	}, nil
}

// SplitSubModules memecah isi kolom SUBMODUL menjadi daftar.
//
// Sistem lama memecahnya dengan `@pxPageListFromStringCSV`, sehingga pemisahnya koma.
// Potongan kosong dibuang supaya "A,,B" tidak menghasilkan submodul tanpa nama.
func SplitSubModules(raw string) []string {
	parts := strings.Split(raw, ",")
	list := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.TrimSpace(part)
		if clean != "" {
			list = append(list, clean)
		}
	}
	return list
}
