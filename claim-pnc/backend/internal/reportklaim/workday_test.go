package reportklaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

// wib menyusun tanggal WIB, supaya uji tidak bergantung pada zona mesin yang menjalankannya.
func wib(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, clock.ZoneWIB)
}

// 1 September 2026 jatuh pada Selasa. Sepekan penuh berisi lima hari kerja.
func TestHariKerjaSepekanPenuhAdaLima(t *testing.T) {
	kalender := reportklaim.NewHolidayCalendar(nil)

	// Selasa 1 Sep -> Selasa 8 Sep: tujuh hari sesudahnya, dua di antaranya akhir pekan.
	require.Equal(t, 5, reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 8), kalender))
}

// Hari awal tidak dihitung — itu arti "selisih" pada rumus aslinya. Klaim yang
// diregistrasi dan ditransfer pada hari yang sama menghasilkan 0, bukan 1.
func TestHariYangSamaMenghasilkanNol(t *testing.T) {
	kalender := reportklaim.NewHolidayCalendar(nil)
	require.Equal(t, 0, reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 1), kalender))
}

func TestAkhirPekanTidakDihitung(t *testing.T) {
	kalender := reportklaim.NewHolidayCalendar(nil)

	// Jumat 4 Sep -> Senin 7 Sep: Sabtu dan Minggu dilewati, tersisa Senin.
	require.Equal(t, 1, reportklaim.WorkingDaysBetween(wib(2026, 9, 4), wib(2026, 9, 7), kalender))

	// Jumat -> Minggu: tidak ada satu pun hari kerja di antaranya.
	require.Equal(t, 0, reportklaim.WorkingDaysBetween(wib(2026, 9, 4), wib(2026, 9, 6), kalender))
}

func TestHariLiburTidakDihitung(t *testing.T) {
	// Rabu 2 September ditetapkan libur.
	kalender := reportklaim.NewHolidayCalendar([]time.Time{wib(2026, 9, 2)})

	// Selasa 1 -> Kamis 3: tanpa libur ada dua hari kerja, dengan libur tinggal satu.
	require.Equal(t, 1, reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 3), kalender))
}

// Hari libur yang jatuh pada akhir pekan TIDAK boleh mengurangi dua kali.
//
// Inilah sebab kueri aslinya menyaring `NOT IN ('SABTU','MINGGU','SATURDAY','SUNDAY')`.
// Tanpa penyaring itu, hari libur pada hari Sabtu dikurangkan sekali sebagai akhir pekan
// dan sekali lagi sebagai libur — dan hasilnya lebih kecil dari yang sebenarnya.
func TestLiburPadaAkhirPekanTidakMengurangiDuaKali(t *testing.T) {
	// Sabtu 5 September ditetapkan libur; ia sudah bukan hari kerja.
	dengan := reportklaim.NewHolidayCalendar([]time.Time{wib(2026, 9, 5)})
	tanpa := reportklaim.NewHolidayCalendar(nil)

	require.Equal(t,
		reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 8), tanpa),
		reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 8), dengan),
	)
}

// Jam pada nilai waktu tidak boleh mengubah hasil: perbandingannya hari kalender WIB.
func TestJamTidakMengubahHasil(t *testing.T) {
	kalender := reportklaim.NewHolidayCalendar(nil)

	pagi := time.Date(2026, 9, 1, 1, 0, 0, 0, clock.ZoneWIB)
	malam := time.Date(2026, 9, 8, 23, 59, 0, 0, clock.ZoneWIB)
	require.Equal(t, 5, reportklaim.WorkingDaysBetween(pagi, malam, kalender))

	// Nilai UTC yang menunjuk hari WIB yang sama menghasilkan angka yang sama pula.
	utc := time.Date(2026, 8, 31, 17, 30, 0, 0, time.UTC) // 1 Sep 00:30 WIB
	require.Equal(t, 5, reportklaim.WorkingDaysBetween(utc, malam, kalender))
}

// Tiga keadaan yang TIDAK boleh menghasilkan angka, karena angka apa pun akan terbaca
// sebagai hasil perhitungan yang sah.
func TestKeadaanYangTidakDapatDihitung(t *testing.T) {
	kalender := reportklaim.NewHolidayCalendar(nil)

	t.Run("tanggal awal kosong", func(t *testing.T) {
		require.Equal(t, reportklaim.WorkingDaysUnknown,
			reportklaim.WorkingDaysBetween(time.Time{}, wib(2026, 9, 8), kalender))
	})

	t.Run("tanggal akhir kosong", func(t *testing.T) {
		require.Equal(t, reportklaim.WorkingDaysUnknown,
			reportklaim.WorkingDaysBetween(wib(2026, 9, 1), time.Time{}, kalender))
	})

	t.Run("akhir mendahului awal", func(t *testing.T) {
		require.Equal(t, reportklaim.WorkingDaysUnknown,
			reportklaim.WorkingDaysBetween(wib(2026, 9, 8), wib(2026, 9, 1), kalender))
	})

	// Kalender yang TIDAK TERSEDIA berbeda dari kalender tanpa hari libur. Yang pertama
	// berarti sumbernya tidak terbaca; yang kedua berarti memang tidak ada hari libur.
	t.Run("kalender tidak tersedia", func(t *testing.T) {
		require.Equal(t, reportklaim.WorkingDaysUnknown,
			reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 8), reportklaim.UnavailableCalendar()))
	})
}

func TestKalenderKosongBerbedaDariKalenderTidakTersedia(t *testing.T) {
	require.True(t, reportklaim.NewHolidayCalendar(nil).Available())
	require.False(t, reportklaim.UnavailableCalendar().Available())
	require.Zero(t, reportklaim.NewHolidayCalendar(nil).Count())
}

// Kalender nil tidak boleh membuat aplikasi berhenti; ia diperlakukan sebagai tidak
// tersedia. Nilai nol sebuah pointer adalah keadaan yang pasti terjadi suatu saat.
func TestKalenderNilAman(t *testing.T) {
	var kalender *reportklaim.HolidayCalendar
	require.False(t, kalender.Available())
	require.False(t, kalender.IsHoliday(wib(2026, 9, 1)))
	require.Zero(t, kalender.Count())
	require.Equal(t, reportklaim.WorkingDaysUnknown,
		reportklaim.WorkingDaysBetween(wib(2026, 9, 1), wib(2026, 9, 8), kalender))
}
