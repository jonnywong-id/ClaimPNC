package clock

import "time"

// ZoneWIB adalah satu-satunya tempat pergeseran zona waktu terjadi di aplikasi ini.
//
// # Kenapa FixedZone, bukan LoadLocation
//
// `time.LoadLocation("Asia/Jakarta")` membaca basis data zona waktu sistem operasi.
// Pada Windows basis data itu tidak selalu ada, dan kegagalannya muncul sebagai galat
// saat proses berjalan — bukan saat kompilasi. WIB adalah offset tetap UTC+7 tanpa
// daylight saving sejak 1964, sehingga offset tetap adalah wakil yang TEPAT, bukan
// penyederhanaan.
//
// # Kenapa ini bukan "penambahan 7 jam manual"
//
// Larangan pada `docs/Steering/08-TECHNICAL-STRATEGY.md` §4.4 adalah larangan
// menambahkan tujuh jam ke NILAI WAKTU — persis yang dilakukan sistem lama lewat 101
// panggilan `addCalendar(...,7,0,0)` yang tersebar, sebagian pada satu sisi sebuah
// perbandingan dan tidak pada sisi lainnya. Yang dilakukan di sini berbeda: nilai waktu
// tidak diubah sama sekali; yang dipilih hanyalah zona untuk membacanya.
var ZoneWIB = time.FixedZone("WIB", 7*60*60)

// DateWIB memotong sebuah waktu menjadi tanggal kalendernya menurut WIB.
//
// Hasilnya adalah tengah malam WIB pada tanggal tersebut. Seluruh perbandingan tanggal
// pada aturan bisnis memakai fungsi ini, sehingga dua nilai yang jatuh pada hari kerja
// yang sama selalu dianggap sama — berapa pun jamnya, dan dari sisi mana pun
// perbandingan itu ditulis.
func DateWIB(t time.Time) time.Time {
	w := t.In(ZoneWIB)
	return time.Date(w.Year(), w.Month(), w.Day(), 0, 0, 0, 0, ZoneWIB)
}

// DaysBetween menghitung jarak hari kalender WIB dari a ke b.
//
// Positif berarti b jatuh setelah a. Karena kedua sisi dipotong lebih dulu oleh
// DateWIB, hasilnya tidak terpengaruh jam maupun menit.
func DaysBetween(a, b time.Time) int {
	dayA, db := DateWIB(a), DateWIB(b)
	return int(db.Sub(dayA).Hours() / 24)
}

// AddDays menggeser sebuah tanggal WIB sebanyak n hari kalender.
func AddDays(t time.Time, n int) time.Time {
	return DateWIB(t).AddDate(0, 0, n)
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
	return t.In(ZoneWIB).Year() % 100
}
