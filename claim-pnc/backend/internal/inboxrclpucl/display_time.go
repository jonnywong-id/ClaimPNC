package inboxrclpucl

import (
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// DisplayTimeLayout adalah bentuk waktu yang digambar SELURUH isian tanggal modul ini.
//
// # Kenapa bentuk ini, dan kenapa ia tidak dikarang di sini
//
// Karena penyimpanan memori modul ini SUDAH memakainya sejak awal — baik untuk kolom
// pengurut (`decorate`) maupun untuk baris contohnya (`sample.go`). Yang berubah bukan
// pilihan bentuknya melainkan siapa yang mematuhinya: sebelum ini penyimpanan SQL
// mengembalikan apa adanya dari driver, sehingga kedua pengisi seam menggambar hal yang
// sama dengan dua bentuk yang berbeda.
//
// Bentuk persis yang digambar Pega TIDAK dapat dibaca dari export: ketiga sel tanggal
// memakai kontrol `pxDateTime` dengan `pyDateFormat` dan `pyDateTimeFormat` KOSONG
// (`Section/InboxCetakSuratPUCLRCL_Section-Section.xml`), sehingga Pega memakai bentuk
// bawaan locale-nya. Yang dapat dipastikan hanyalah bahwa ia bentuk TANGGAL — bukan teks
// ISO ber-`T` dan ber-offset zona. Lihat selisih terencana.
const DisplayTimeLayout = "2006-01-02 15:04:05"

// DisplayTime menggambar satu titik waktu sebagai teks WIB.
//
// Waktu NOL menghasilkan teks KOSONG, bukan "0001-01-01 00:00:00": baris yang waktunya
// tidak diketahui berarti tidak ada tanggalnya, dan tanggal tahun 1 di layar terbaca
// sebagai data alih-alih sebagai ketiadaan data.
//
// Konversi zonanya menempuh `clock.ZoneWIB` — satu-satunya tempat pergeseran zona terjadi
// di aplikasi ini (`F-5`).
func DisplayTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.In(clock.ZoneWIB).Format(DisplayTimeLayout)
}

// zonedTimeLayouts adalah bentuk yang MEMBAWA zona waktunya sendiri.
//
// Nilai yang cocok dengan salah satunya dikonversi ke WIB, karena zonanya diketahui.
// Driver Oracle yang dipakai aplikasi ini mengembalikan kolom `TIMESTAMP(6)` dalam bentuk
// pertama — terverifikasi langsung terhadap basis data ASM pada 2026-09-30.
var zonedTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
}

// wallClockTimeLayouts adalah bentuk yang TIDAK membawa zona.
//
// Nilai yang cocok dengan salah satunya dibaca sebagai jam dinding WIB dan TIDAK digeser.
// Membacanya sebagai UTC lalu mengubahnya ke WIB akan memajukan setiap tanggal tujuh jam —
// dan pada nilai menjelang tengah malam itu memindahkannya ke hari berikutnya, tanpa satu
// pun galat. `keputusan-implementasi.md` §49 mencatat kelas kekeliruan yang sama pada arah
// sebaliknya.
var wallClockTimeLayouts = []string{
	DisplayTimeLayout,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02",
}

// DisplayTimeText menggambar nilai tanggal yang sampai ke sini sebagai TEKS.
//
// # Kenapa teks, bukan `time.Time`
//
// Karena DDL tabel Pega tidak pernah diterima (`R-08`), dan yang terverifikasi barulah satu
// basis data dari enam. Pada portal ASM ketiga kolom tanggal modul ini bertipe
// `TIMESTAMP(6)` — diperiksa langsung ke `ALL_TAB_COLUMNS` pada 2026-09-30 — tetapi kelima
// portal lain belum diperiksa, dan tabel yang sama di sana boleh jadi menyimpannya sebagai
// teks.
//
// Nilai yang TIDAK dikenali karena itu dikembalikan APA ADANYA, bukan dikosongkan maupun
// ditolak. Kosong akan menghapus isi kolom pada seluruh baris di portal yang bentuknya
// berbeda, dan itu kegagalan yang tidak menghasilkan satu pun galat.
func DisplayTimeText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	for _, layout := range zonedTimeLayouts {
		if at, err := time.Parse(layout, trimmed); err == nil {
			return DisplayTime(at)
		}
	}

	for _, layout := range wallClockTimeLayouts {
		if at, err := time.ParseInLocation(layout, trimmed, clock.ZoneWIB); err == nil {
			return DisplayTime(at)
		}
	}

	return trimmed
}
