package inboxmanageradmin

import (
	"fmt"
	"time"

	"claim-pnc/internal/platform/clock"
)

// FormatElapsed menyusun kolom "Lama Waktu Klaim".
//
// # Dari mana bentuknya
//
// Kolom ini BUKAN properti tersendiri. Ia properti yang SAMA dengan kolom Tanggal
// Pendaftaran — `.pxCreateDateTime`, yang karena itu muncul dua kali di
// `Section/InboxManagerAdmin_Section-Section.xml` — hanya digambar dengan mode kontrol kedua
// ber-`pyDateTimeFormat` **`DateTime-Frame`**. Bentuk itu muncul tepat tiga kali di section,
// satu untuk tiap tab.
//
// `DateTime-Frame` adalah format waktu relatif **bawaan Pega**, bukan rule buatan sendiri.
// Itu sebabnya pencarian teks "months ago" di `RDB List/`, `Database/`, `Activity/`,
// `Section/`, `HTML/`, dan `Function/` tidak menemukan satu pun penyusunnya: kodenya ada di
// platform, yang memang tidak ikut dalam export rule aplikasi.
//
// # Kenapa ia HARI KALENDER, bukan hari kerja
//
// Karena yang diamati langsung dari layar Pega yang berjalan memang begitu: baris bertanggal
// `22/04/25 13:46` menampilkan "1 year 5 months ago", yang tepat sama dengan selisih
// KALENDER-nya. Bila akhir pekan dipotong seperti pada `GETSELISIHJAM`, angkanya akan
// berbunyi "1 year 0 months".
//
// # Kenapa ditulis ulang, bukan dipakai bersama modul Inbox Compliance
//
// Karena `inboxcompliance.FormatElapsed` adalah fungsi milik PAKET DOMAIN modul lain, dan
// modul tidak saling mengimpor (`08-TECHNICAL-STRATEGY.md` §2). Memindahkannya ke
// `internal/platform/` adalah jalan yang lebih baik dan sudah diantisipasi komentar di sana
// — tetapi itu menyentuh modul yang sudah selesai, sehingga ditulis sebagai usulan di
// docs/keputusan-implementasi.md alih-alih dikerjakan sepihak di sesi ini.
//
// Perilakunya dijaga SAMA PERSIS, dan uji di elapsed_test.go memuat kasus yang sama
// supaya keduanya tidak dapat menyimpang tanpa ketahuan.
//
// Mengembalikan teks kosong bila tanggalnya kosong — bukan "0 minutes ago", dengan alasan
// yang sama seperti kolom Aging modul lain (`P-5` butir 13): nilai gagal yang tidak dapat
// dibedakan dari nol adalah cacat yang justru sedang diperbaiki migrasi ini.
func FormatElapsed(from *time.Time, now time.Time) string {
	if from == nil || from.IsZero() || !now.After(*from) {
		return ""
	}

	start, end := from.In(clock.ZoneWIB), now.In(clock.ZoneWIB)

	// Tahun dan bulan dihitung secara KALENDER, bukan dengan membagi selisih detik.
	// Membaginya akan memakai "bulan" sepanjang 30 hari yang tidak ada di kalender mana
	// pun, dan hasilnya meleset makin jauh seiring rentangnya memanjang.
	months := int(end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month())
	if end.Day() < start.Day() {
		months--
	}

	if months >= 12 {
		years, rest := months/12, months%12

		// Sisa nol bulan tidak ditulis. "1 year 0 months ago" bukan bentuk yang ditulis
		// pemformat waktu relatif mana pun, dan ia hanya muncul tepat pada hari ulang
		// tahun sebuah baris.
		if rest == 0 {
			return plural(years, "year") + " ago"
		}
		return plural(years, "year") + " " + plural(rest, "month") + " ago"
	}

	if months >= 1 {
		return plural(months, "month") + " ago"
	}

	elapsed := end.Sub(start)
	switch {
	case elapsed >= 24*time.Hour:
		return plural(int(elapsed.Hours())/24, "day") + " ago"
	case elapsed >= time.Hour:
		return plural(int(elapsed.Hours()), "hour") + " ago"
	default:
		return plural(int(elapsed.Minutes()), "minute") + " ago"
	}
}

// plural menuliskan sebuah jumlah beserta satuannya dalam bentuk yang benar.
//
// Bentuk jamak bahasa Inggris di sini cukup dengan menambahkan "s": kelima satuan yang
// dipakai — minute, hour, day, month, year — seluruhnya beraturan.
func plural(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", count, unit)
}
