package reportklaim

import (
	"context"
	"time"
)

// Caller adalah identitas petugas yang menjalankan laporan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
//
// # Kenapa modul ini membutuhkannya sama sekali
//
// Bukan untuk menyaring baris. Tidak satu pun dari 28 kueri menyaring menurut petugas
// yang menjalankannya — laporan ini memang laporan lintas cabang, dan itulah gunanya.
//
// Ia dibutuhkan untuk **jejak audit**: berkas yang keluar dari sini memuat data nasabah
// lintas cabang, dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena
// tidak ada pemisahan tugas. Siapa mengunduh laporan apa, kapan, dengan penyaring apa —
// itu yang dicatat.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
	// di sistem lama.
	Login string

	// Name adalah nama petugas sebagaimana ditampilkan.
	Name string
}

// Row adalah satu baris hasil laporan.
//
// # Kenapa peta teks, bukan struct
//
// Katalog ini memuat 28 laporan dengan **34 susunan kolom berbeda dan sekitar 700 kolom
// seluruhnya**, dan tidak ada dua laporan yang susunannya sama. Sebuah struct per laporan
// berarti 34 struct yang masing-masing hanya dipakai sekali, dan setiap penambahan kolom
// menyentuh tiga lapisan sekaligus.
//
// Kuncinya adalah nama properti pada `CSVProperties` milik langkah `pxConvertResultsToCSV`
// — sama persis dengan yang dipakai sistem lama, sehingga satu kolom dapat ditelusuri dari
// judul di berkas sampai ke alias di SQL tanpa tabel penerjemah.
//
// # Kenapa nilainya string
//
// Keluarannya CSV, dan seluruh pemformatan sudah terjadi di SQL — tanggal keluar sebagai
// `to_char(...,'dd/mm/yyyy')` dan angka sebagai teks. Mengubahnya menjadi tipe lalu
// mengembalikannya menjadi teks hanya menambah satu tempat baru untuk salah format.
//
// Nilai yang tidak ada pada baris ditulis sebagai sel kosong — lihat Value.
type Row map[string]string

// Value mengembalikan isi satu kolom, atau string kosong bila tidak ada.
func (r Row) Value(field string) string {
	if r == nil {
		return ""
	}
	return r[field]
}

// Repo membaca baris laporan dari basis data satu portal.
type Repo interface {
	// Stream memanggil emit sekali untuk setiap baris hasil, sesuai urutan kueri.
	//
	// # Kenapa dialirkan, bukan dikembalikan sebagai slice
	//
	// Laporan ini berjalan atas data historis puluhan juta baris (`D-10`), dan berkasnya
	// ditulis langsung ke jawaban HTTP. Mengumpulkan seluruh baris lebih dulu membuat
	// memori tumbuh sebanding dengan hasil — persis yang dilarang
	// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6.
	//
	// Galat yang dikembalikan emit menghentikan pengaliran dan diteruskan apa adanya.
	// Itulah cara pemanggil menghentikan kueri yang sudah melewati batas barisnya tanpa
	// harus membaca sisanya.
	Stream(ctx context.Context, report Report, filter Filter, emit func(Row) error) error

	// ListBusinessOptions mengembalikan isi autocomplete "Bisnis".
	//
	// Asalnya `BrowseBusiness_RD`, yang di harness mengisi `TempLaporan.CountryID`.
	// Ia berada di Repo — bukan di seam tersendiri — karena isinya milik basis data
	// entitas: daftar bisnis satu badan hukum bukan daftar badan hukum lain (`R-20`).
	ListBusinessOptions(ctx context.Context) ([]BusinessOption, error)
}

// BusinessOption adalah satu baris pilihan pada autocomplete "Bisnis".
type BusinessOption struct {
	// Code adalah nilai yang dikirim kembali sebagai penyaring — `TempLaporan.CountryID`.
	Code string

	// Name adalah yang dibaca pengguna — `TempLaporan.Country`.
	Name string
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti membaca data satu badan
// hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}

// AuditSink mencatat siapa mengunduh laporan apa.
//
// # Kenapa ia seam, bukan pemanggilan log biasa
//
// Karena yang dicatat bukan peristiwa teknis melainkan **peristiwa bisnis**: satu berkas
// berisi data nasabah lintas cabang keluar dari sistem. `S-5` adalah modul tersendiri dan
// belum ada; sampai ia ada, pencatatnya diisi penulis log terstruktur.
//
// Ia dibuat seam sejak sekarang supaya penggantiannya kelak tidak menyentuh satu baris pun
// aturan di modul ini — persis alasan seam ada (`04-FUTURE-ARCHITECTURE.md` §3).
//
// OPSIONAL: nil berarti tidak ada yang dicatat. Ia dibiarkan opsional supaya modul tetap
// dapat dirakit di lingkungan yang belum punya pencatatnya, bukan supaya pencatatannya
// boleh dilewatkan di produksi.
type AuditSink interface {
	// ReportDownloaded dipanggil SETELAH berkas selesai terkirim, beserta jumlah barisnya.
	//
	// Sesudah, bukan sebelum: unduhan yang gagal di tengah bukan unduhan, dan mencatatnya
	// sebagai berhasil membuat jejak audit berbohong ke arah yang paling merugikan —
	// menyatakan data keluar padahal tidak, atau sebaliknya.
	ReportDownloaded(ctx context.Context, caller Caller, portalAlias string, report Report, filter Filter, rows int)
}
