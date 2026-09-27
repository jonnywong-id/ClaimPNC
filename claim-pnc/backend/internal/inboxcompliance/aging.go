package inboxcompliance

import (
	"fmt"
	"math"
	"time"

	"claim-pnc/internal/platform/clock"
)

// Kolom Aging: hitungan lama sebuah klaim menunggu di antrean Compliance.
//
// # Apa yang digantikan berkas ini
//
// Tiga artefak sistem lama sekaligus:
//
//	Database/GETSELISIHJAM.fnc            hitungan jamnya
//	RDB List/GetSelisihJam_sql-SQL.xml    pemformatan teksnya
//	Activity/GetInboxRegisterCompliance-Act.xml  perakitan keduanya per baris
//
// Ia ditulis ulang di Go, bukan dipanggil lewat basis data. Dua keputusan menuntutnya:
// `D-02` (aplikasi tidak memanggil stored procedure) dan `D-50` (perhitungan jam kerja
// adalah ATURAN BISNIS, sehingga ditulis ulang di Go alih-alih diambil lewat pemanggilan).
//
// # Apa yang dilakukan fungsi aslinya, apa adanya
//
//	cnt := weekends2(TRUNC(tglawal), TRUNC(tglakhir))
//	wkt := datediff('SS', tglawal, tglakhir)
//	IF cnt > 0 THEN wkt := wkt - (86400 * cnt)
//	RETURN ROUND(wkt / 3600, 2)
//
// Jadi: selisih DETIK antara kedua waktu, dikurangi satu hari penuh untuk setiap hari akhir
// pekan yang terlewati, dibagi 3600, dibulatkan dua desimal.
//
// # Yang ia TIDAK lakukan, meski namanya "selisih jam kerja"
//
// Ia tidak mengenal hari libur nasional, dan tidak mengenal jam kerja. Hari Sabtu dan Minggu
// dipotong sebagai hari penuh 24 jam, sedangkan malam hari kerja tetap dihitung. Ini sejalan
// dengan `D-50`, yang menetapkan `GETSELISIHJAM` dan `GET_WORKING_HOURS@ASMD` adalah DUA
// basis berbeda untuk keperluan berbeda — dan layar inilah satu-satunya pemakai
// `GETSELISIHJAM` di seluruh export.
//
// Karena itu angka di sini TIDAK boleh dibandingkan dengan angka TAT pada laporan KPI. Yang
// kedua memakai `GET_WORKING_HOURS@ASMD` dan mengenal kalender libur.
//
// # Satu asumsi yang harus dibuka
//
// Source `weekends2` dan `POOLDATA.datediff` TIDAK ikut dikirim — keduanya bagian dari 12
// dependensi yang dipanggil 62 procedure tetapi belum diterima (`R-01`). Yang diketahui
// hanya tanda tangannya: `weekends2(TRUNC(tglawal), TRUNC(tglakhir))`.
//
// Yang diterapkan di sini: jumlah hari Sabtu dan Minggu pada rentang SETENGAH TERBUKA
// `[tanggal WIB tglawal, tanggal WIB tglakhir)`. Tafsir itu dipilih karena ia satu-satunya
// yang tidak pernah memotong lebih banyak daripada waktu yang benar-benar berlalu — tafsir
// inklusif akan memotong 24 jam penuh dari selisih delapan jam pada satu hari Sabtu yang
// sama, dan menghasilkan Aging negatif.
//
// Bila source `weekends2` kelak diterima dan tafsirnya berbeda, tempat mengubahnya satu:
// weekendDaysWIB di bawah. Selisihnya wajib diperiksa pada gerbang 1.

// AgingHoursBetween menghitung lama antrean dalam jam, sudah dipotong akhir pekan.
//
// Mengembalikan nil bila tanggal dasarnya kosong, sehingga kolom Aging pada baris yang
// belum punya Tanggal Kirim Compliance tampil KOSONG — bukan "0 hours ago" yang menyatakan
// hal yang tidak benar.
//
// Pembedaan itu adalah perbaikan yang sudah diputuskan, bukan pilihan gaya:
// `Database/GETSELISIHJAM.fnc:22` mengembalikan `0` pada setiap kegagalan, sehingga
// kegagalan tidak dapat dibedakan dari nol jam. `D-49` butir 10 memutuskan cacat itu
// diperbaiki, dan ia tercatat sebagai butir ke-13 daftar perbaikan eksplisit `P-5`.
func AgingHoursBetween(sentAt *time.Time, now time.Time) *float64 {
	if sentAt == nil || sentAt.IsZero() {
		return nil
	}

	seconds := now.Sub(*sentAt).Seconds()
	seconds -= float64(86400 * weekendDaysWIB(*sentAt, now))

	if seconds < 0 {
		// Antrean tidak dapat berumur negatif. Baris bertanggal di masa depan memang ada
		// di data warisan — dan setelah pemotongan akhir pekan, selisih yang wajar pun
		// dapat menjadi negatif ketika keduanya jatuh di akhir pekan yang sama.
		//
		// Nol, bukan nil: tanggalnya ADA dan terbaca, hanya belum menghasilkan jam yang
		// berarti. Mengembalikan nil di sini akan mengaburkannya dengan "tanggal belum
		// diisi", yang justru baru saja dibedakan di atas.
		seconds = 0
	}

	hours := math.Round(seconds/3600*100) / 100
	return &hours
}

// weekendDaysWIB menghitung jumlah hari Sabtu dan Minggu pada rentang setengah terbuka
// [tanggal WIB a, tanggal WIB b).
//
// # Kenapa WIB, bukan UTC
//
// Karena begitulah sistem lama menghitungnya, dan perbedaannya nyata. Activity pemanggil
// mengubah kedua sisi ke WIB lebih dulu — `CARI1 = "SYSDATE"` (jam server, WIB) dan
// `CARI2 = TanggalBuatCompliance + (7/24)`
// (`Activity/GetInboxRegisterCompliance-Act.xml:750`, `:815`) — lalu `weekends2` memotongnya
// dengan `TRUNC`, yakni tengah malam WIB.
//
// Memakai tanggal UTC akan menggeser batas harinya tujuh jam, sehingga klaim yang masuk
// Jumat pukul 23.00 WIB akan terhitung masih hari Jumat pukul 16.00 UTC — dan jumlah hari
// akhir pekan yang terlewati dapat meleset satu hari, yakni 24 jam pada kolom Aging.
//
// Penambahan tujuh jam pada `CARI2` itu sendiri TIDAK direplikasi: ia bentuk penyesuaian
// manual yang `F-5` hapus. Yang diambil di sini hanyalah akibat yang benar darinya — bahwa
// batas harinya adalah tengah malam WIB — lewat clock.DateWIB, satu-satunya tempat
// pergeseran zona terjadi di aplikasi ini.
func weekendDaysWIB(a, b time.Time) int {
	from, to := clock.DateWIB(a), clock.DateWIB(b)
	if !to.After(from) {
		return 0
	}

	days := int(to.Sub(from).Hours() / 24)

	// Setiap tujuh hari penuh memuat tepat satu Sabtu dan satu Minggu, berapa pun hari
	// mulainya. Hanya sisa harinya yang perlu diperiksa satu per satu — sehingga rentang
	// bertahun-tahun pun tidak menuntut perulangan sepanjang itu.
	weekends := (days / 7) * 2

	for i := days - days%7; i < days; i++ {
		switch from.AddDate(0, 0, i).Weekday() {
		case time.Saturday, time.Sunday:
			weekends++
		}
	}

	return weekends
}

// FormatAging menyusun teks kolom Aging persis seperti sistem lama menampilkannya.
//
// Bentuknya diambil apa adanya dari `RDB List/GetSelisihJam_sql-SQL.xml`:
//
//	WHEN hours_diff < 24 THEN TRUNC(hours_diff) || ' hours ago'
//	ELSE FLOOR(hours_diff / 24) || ' days ' || MOD(TRUNC(hours_diff), 24) || ' hours ago'
//
// Dua hal pada bentuk itu sengaja dipertahankan meski tampak janggal, karena keduanya
// terlihat pengguna dan mengubahnya akan memunculkan selisih di gerbang 1:
//
//   - Jamnya DIPOTONG, bukan dibulatkan. 23,9 jam tampil "23 hours ago", bukan 24.
//   - Bagian hari dan bagian jam dihitung dari nilai yang sama tetapi dengan cara berbeda —
//     `FLOOR(h/24)` untuk harinya dan `MOD(TRUNC(h), 24)` untuk jamnya. Pada jam positif
//     keduanya konsisten, sehingga 49,5 jam tampil "2 days 1 hours ago".
//
// Bentuk jamaknya pun tidak diperbaiki: satu jam tetap tampil "1 hours ago", dan satu hari
// tetap "1 days". Itu yang ditampilkan sistem lama.
//
// Teksnya tetap berbahasa Inggris. `D-13` menetapkan tampilan meniru Pega supaya pengguna
// tidak perlu belajar ulang, dan "hours ago" adalah yang selama ini mereka baca di kolom
// ini.
//
// Mengembalikan teks kosong bila Aging tidak dapat dihitung.
func FormatAging(hours *float64) string {
	if hours == nil {
		return ""
	}

	whole := math.Trunc(*hours)

	if *hours < 24 {
		return fmt.Sprintf("%d hours ago", int64(whole))
	}

	days := int64(math.Floor(*hours / 24))
	rest := int64(whole) % 24

	return fmt.Sprintf("%d days %d hours ago", days, rest)
}

// FormatElapsed menyusun kolom "OutStanding" pada tab Post Audit.
//
// # Kenapa ia terpisah dari FormatAging, padahal keduanya "sudah berapa lama"
//
// Karena keluarannya memang berbeda, dan itu terbaca langsung dari layar Pega yang
// berjalan: baris bertanggal `22/04/25 13:46` menampilkan **"1 year 5 months ago"**,
// sedangkan kolom Aging pada tab sebelah berbunyi "N days M hours ago".
//
// Perbedaannya bukan hanya satuan. Dari 22 April 2025 ke 24 September 2026 ada tepat
// 1 tahun 5 bulan 2 hari secara KALENDER. Bila akhir pekan dipotong seperti pada
// `GETSELISIHJAM`, selisihnya menyusut sekitar 149 hari dan angkanya akan berbunyi
// "1 year 0 months". Karena Pega menampilkan yang pertama, kolom ini **tidak memotong akhir
// pekan** — ia waktu kalender apa adanya.
//
// # Dari mana bentuknya, dan apa yang masih rekonstruksi
//
// Kolom ini BUKAN properti tersendiri. Ia properti yang SAMA dengan kolom Tanggal Kirim
// Audit Compliance — `.TanggalKirimPostAudit`, yang karena itu muncul dua kali di section —
// hanya digambar dengan mode kontrol kedua ber-`pyDateTimeFormat` **`DateTime-Frame`**
// (`Section/InputPostAuditDtl_Section-Section.xml:4036`).
//
// `DateTime-Frame` adalah format waktu relatif **bawaan Pega**, bukan rule buatan sendiri.
// Itu sebabnya pencarian teks "months ago" di `RDB List/`, `Database/`, `Activity/`,
// `Section/`, `HTML/`, dan `Function/` tidak menemukan satu pun penyusunnya: kodenya ada di
// platform, yang memang tidak ikut dalam export rule aplikasi.
//
// Format yang sama dipakai belasan section Inbox lain di aplikasi ini — antara lain
// `DashboardClaim_Section1`, `InboxAnalystDoctor_Section`, `InboxManagerAdmin_Section`, dan
// `InboxOutstandingClaim_Section` — sehingga penulisan ulangnya di sini akan dipakai ulang
// oleh modul-modul itu kelak.
//
// Yang TERAMATI langsung dari layar Pega yang berjalan: `"1 year 5 months ago"`. Dari satu
// contoh itu dua hal dapat dipastikan, dan keduanya diterapkan:
//
//   - Dua satuan ditampilkan sekaligus begitu melewati satu tahun.
//   - Bentuk jamaknya BENAR per satuan — "1 year" tunggal berdampingan dengan "5 months"
//     jamak. Ia tidak menulis "1 years".
//
// Yang masih REKONSTRUKSI: bunyi cabang bulan, hari, jam, dan menit. Tidak ada baris yang
// cukup baru di layar Pega untuk membandingkannya, sehingga tangga di bawah mengikuti
// bentuk baku `DateTime-Frame` dan **belum diverifikasi**.
//
// Mengembalikan teks kosong bila tanggalnya kosong — bukan "0 minutes ago", dengan alasan
// yang sama seperti kolom Aging (`P-5` butir 13).
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
// Bentuk jamak bahasa Inggris di sini cukup dengan menambahkan "s": keenam satuan yang
// dipakai — minute, hour, day, month, year — seluruhnya beraturan. Tidak ada gunanya
// menarik pustaka pluralisasi untuk enam kata.
func plural(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", count, unit)
}
