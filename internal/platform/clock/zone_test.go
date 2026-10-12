package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
)

// WIB selalu tujuh jam di depan UTC, tanpa daylight saving.
//
// Zona bergeser tetap dipakai, bukan time.LoadLocation("Asia/Jakarta"), karena basis data
// zona waktu sistem operasi TIDAK selalu ada di Windows — pemanggilannya gagal dengan
// galat, bukan jatuh ke nilai baku. Aplikasi yang tanggalnya bergantung pada paket sistem
// operasi akan berperilaku berbeda di mesin pengembang dan di peladen produksi.
func TestWIBIsSevenHoursAheadOfUTC(t *testing.T) {
	_, offset := time.Date(2026, time.September, 18, 0, 0, 0, 0, clock.ZoneWIB).Zone()
	require.Equal(t, 7*60*60, offset)

	// Juga di bulan yang di belahan bumi lain berlaku daylight saving.
	_, januaryOffset := time.Date(2026, time.January, 15, 0, 0, 0, 0, clock.ZoneWIB).Zone()
	require.Equal(t, 7*60*60, januaryOffset, "WIB tidak mengenal daylight saving")
}

// Tanggal WIB dihitung dari waktu Jakarta, bukan dari waktu UTC.
//
// Selisih tujuh jam itu MENGUBAH HASIL pada aturan berbasis hari kalender — batas 7 hari,
// 30 hari, dan 90 hari pada validasi registrasi. Ini kegagalan yang sama dengan yang
// diwarisi sistem lama lewat 118 titik `+7 jam` manual, hanya dari arah sebaliknya.
func TestDateWIBComputedFromJakartaTime(t *testing.T) {
	// 17 September 2026 pukul 20:00 UTC = 18 September pukul 03:00 WIB.
	night := time.Date(2026, time.September, 17, 20, 0, 0, 0, time.UTC)

	date := clock.DateWIB(night)
	require.Equal(t, 2026, date.Year())
	require.Equal(t, time.September, date.Month())
	require.Equal(t, 18, date.Day(), "pukul 03:00 WIB sudah hari berikutnya")

	// Jam, menit, dan detiknya dibuang.
	require.Zero(t, date.Hour())
	require.Zero(t, date.Minute())
	require.Zero(t, date.Second())
	require.Zero(t, date.Nanosecond())
}

// Dua waktu pada hari yang sama di Jakarta menghasilkan tanggal yang sama persis, berapa
// pun selisih jamnya.
func TestTwoTimesSameJakartaDayGiveSameDate(t *testing.T) {
	// 17 September 17:00 UTC = 18 September 00:00 WIB.
	veryEarly := time.Date(2026, time.September, 17, 17, 0, 0, 0, time.UTC)
	// 18 September 16:59 UTC = 18 September 23:59 WIB.
	veryLate := time.Date(2026, time.September, 18, 16, 59, 0, 0, time.UTC)

	require.Equal(t, clock.DateWIB(veryEarly), clock.DateWIB(veryLate))
}

// Tahun dua digit diambil dari zona WIB.
//
// Keduanya berbeda selama tujuh jam setiap pergantian tahun, dan nomor yang salah tahun
// tidak dapat diperbaiki setelah terbit — ia sudah tercetak di dokumen dan dirujuk
// klaimnya.
func TestTwoDigitYearWIBAroundNewYear(t *testing.T) {
	// 31 Desember 2026 pukul 22:00 UTC = 1 Januari 2027 pukul 05:00 WIB.
	require.Equal(t, 27, clock.TwoDigitYearWIB(
		time.Date(2026, time.December, 31, 22, 0, 0, 0, time.UTC)))

	// 31 Desember 2026 pukul 16:00 UTC = 31 Desember pukul 23:00 WIB.
	require.Equal(t, 26, clock.TwoDigitYearWIB(
		time.Date(2026, time.December, 31, 16, 0, 0, 0, time.UTC)))

	// Tahun 2007 menjadi 7, bukan 2007 — pemformatan dua digitnya urusan pemanggil.
	require.Equal(t, 7, clock.TwoDigitYearWIB(
		time.Date(2007, time.June, 1, 0, 0, 0, 0, time.UTC)))
}
