package waktu

import "time"

// WIB adalah SATU-SATUNYA tempat waktu UTC diubah menjadi waktu Jakarta.
//
// # Kenapa berkas ini ada
//
// `04-FUTURE-ARCHITECTURE.md` §3.2 menetapkan seluruh waktu disimpan UTC dan dikonversi
// ke WIB hanya di satu tempat. Sistem lama melakukan sebaliknya: `+7 jam` ditambahkan
// manual di **118 titik pada 36 activity** lewat
// `@DateTime.addCalendar(..., 0,0,0,0, 7, 0,0)` dan activity `Set7Hours`. Satu tempat
// yang lupa memanggilnya menggeser tanggal tujuh jam tanpa ada yang menyadarinya — dan
// pada aturan seperti "Tanggal Lapor ≤ DOL + 7 hari", pergeseran itu MENGUBAH HASIL
// VALIDASI.
//
// Aturan yang berlaku sejak berkas ini ada: tidak ada `.Add(7 * time.Hour)` di mana pun
// selain di sini.
//
// # Kenapa FixedZone, bukan LoadLocation("Asia/Jakarta")
//
// `time.LoadLocation` membaca basis data zona waktu sistem operasi, dan pada Windows
// basis data itu TIDAK selalu ada — pemanggilannya gagal dengan galat, bukan jatuh ke
// nilai baku. Aplikasi yang tanggalnya bergantung pada paket sistem operasi akan
// berperilaku berbeda di mesin pengembang dan di peladen produksi, dan bedanya baru
// terlihat pada tanggal yang salah.
//
// WIB juga tidak mengenal daylight saving dan tidak pernah berubah sejak 1964, sehingga
// zona bergeser tetap adalah gambaran yang benar — bukan penyederhanaan.
//
// # Yang BELUM ada di sini
//
// Hari libur dan jam kerja. Keduanya milik `F-5` selengkapnya (`D-50`), dan menuntut
// master kalender libur yang belum ada. Yang di sini hanya bagian terkecil yang
// dibutuhkan modul-modul yang sudah berjalan.
func WIB() *time.Location {
	return wib
}

var wib = time.FixedZone("WIB", 7*60*60)

// DateWIB mengembalikan tanggal kalender WIB dari sebuah waktu.
//
// Dipakai aturan bisnis yang berbasis HARI, bukan berbasis detik: batas 7 hari, 30 hari,
// dan 90 hari pada validasi registrasi, serta tahun yang masuk ke dalam nomor terbitan.
//
// Jam, menit, dan detiknya dibuang, dan hasilnya tetap berada di zona WIB — sehingga
// dua waktu yang jatuh pada hari yang sama di Jakarta menghasilkan nilai yang sama
// persis, berapa pun selisih jamnya.
func DateWIB(t time.Time) time.Time {
	local := t.In(wib)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, wib)
}

// TwoDigitYearWIB mengembalikan dua digit terakhir tahun WIB dari sebuah waktu.
//
// Ia dipakai generator nomor terbitan. Dua digit, bukan empat, mengikuti bentuk nomor
// klaim yang `D-71` tetapkan: `PNCN.YY.xxxx`.
//
// Yang dipakai adalah tahun WIB, bukan tahun UTC. Keduanya berbeda selama tujuh jam
// setiap pergantian tahun — dari pukul 07:00 WIB tanggal 1 Januari ke belakang — dan
// laporan yang masuk pada rentang itu akan bernomor tahun lalu bila UTC yang dipakai.
func TwoDigitYearWIB(t time.Time) int {
	return t.In(wib).Year() % 100
}
