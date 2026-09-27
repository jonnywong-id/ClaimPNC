package inboxcompliance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/platform/clock"
)

// wib menyusun sebuah waktu dalam zona WIB.
//
// Uji ini memakai WIB, bukan UTC, karena batas hari yang menentukan jumlah akhir pekan
// memang batas WIB — lihat weekendDaysWIB. Menyusun kasus uji dalam UTC akan membuat kasus
// "Jumat malam" diam-diam menjadi kasus "Jumat sore", dan uji yang lulus tidak menyatakan
// apa pun tentang yang dihitung di produksi.
func wib(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, clock.ZoneWIB)
}

func TestAgingHoursBetween(t *testing.T) {
	t.Parallel()

	// 2026-09-21 adalah Senin; 26 Jumat; 27 Sabtu; 28 Minggu.
	tests := []struct {
		name  string
		from  time.Time
		to    time.Time
		hours float64
	}{
		{
			name:  "beberapa jam di hari yang sama",
			from:  wib(2026, time.September, 21, 8, 0),
			to:    wib(2026, time.September, 21, 11, 30),
			hours: 3.5,
		},
		{
			name:  "melewati satu malam hari kerja, malamnya IKUT dihitung",
			from:  wib(2026, time.September, 21, 20, 0),
			to:    wib(2026, time.September, 22, 8, 0),
			hours: 12,
		},
		{
			name: "melewati Sabtu dan Minggu, keduanya dipotong penuh",
			// Jumat 08:00 ke Senin 08:00 adalah 72 jam kalender; dua hari akhir pekan
			// dipotong 24 jam masing-masing, sehingga tersisa 24 jam.
			from:  wib(2026, time.September, 25, 8, 0),
			to:    wib(2026, time.September, 28, 8, 0),
			hours: 24,
		},
		{
			name: "dua pekan penuh memotong empat hari akhir pekan",
			// 14 hari kalender = 336 jam; empat hari akhir pekan = 96 jam.
			from:  wib(2026, time.September, 21, 9, 0),
			to:    wib(2026, time.October, 5, 9, 0),
			hours: 240,
		},
		{
			name:  "pembulatan dua desimal, mengikuti ROUND(wkt/3600, 2)",
			from:  wib(2026, time.September, 21, 8, 0),
			to:    wib(2026, time.September, 21, 8, 20),
			hours: 0.33,
		},
		{
			name: "dua waktu pada satu hari Sabtu tidak menghasilkan angka negatif",
			// Tafsir setengah terbuka membuat hari yang sama tidak terpotong sama sekali,
			// sehingga hasilnya selisih apa adanya — bukan minus enam belas jam.
			from:  wib(2026, time.September, 26, 9, 0),
			to:    wib(2026, time.September, 26, 17, 0),
			hours: 8,
		},
		{
			name:  "tanggal di masa depan dijepit ke nol, bukan negatif",
			from:  wib(2026, time.September, 22, 9, 0),
			to:    wib(2026, time.September, 21, 9, 0),
			hours: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := inboxcompliance.AgingHoursBetween(&tc.from, tc.to)

			require.NotNil(t, got, "Aging seharusnya dapat dihitung")
			require.InDelta(t, tc.hours, *got, 0.001)
		})
	}
}

// Tanggal dasar yang kosong TIDAK boleh menghasilkan nol jam.
//
// Ini butir ke-13 daftar perbaikan eksplisit `P-5` (`D-49` butir 10):
// `Database/GETSELISIHJAM.fnc:22` mengembalikan `0` pada setiap kegagalan, sehingga
// kegagalan tidak dapat dibedakan dari klaim yang baru saja masuk antrean.
func TestAgingHoursBetweenTanpaTanggalDasar(t *testing.T) {
	t.Parallel()

	now := wib(2026, time.September, 21, 9, 0)

	require.Nil(t, inboxcompliance.AgingHoursBetween(nil, now),
		"tanggal kosong seharusnya tidak menghasilkan angka")

	var zero time.Time
	require.Nil(t, inboxcompliance.AgingHoursBetween(&zero, now),
		"tanggal nol seharusnya tidak menghasilkan angka")
}

// Batas hari yang dipakai menghitung akhir pekan adalah tengah malam WIB, bukan UTC.
//
// Kasusnya dipilih tepat pada rentang tujuh jam yang membedakan keduanya: Jumat pukul 23.00
// WIB masih Jumat pukul 16.00 UTC. Bila batas hari UTC yang dipakai, jumlah hari akhir pekan
// yang terlewati meleset satu — yakni 24 jam pada kolom Aging.
func TestAgingMemakaiBatasHariWIB(t *testing.T) {
	t.Parallel()

	// Jumat 2026-09-25 pukul 23.00 WIB sampai Senin 2026-09-28 pukul 08.00 WIB.
	from := wib(2026, time.September, 25, 23, 0)
	to := wib(2026, time.September, 28, 8, 0)

	got := inboxcompliance.AgingHoursBetween(&from, to)
	require.NotNil(t, got)

	// 57 jam kalender, dikurangi Sabtu dan Minggu (48 jam), tersisa 9 jam.
	require.InDelta(t, 9.0, *got, 0.001)
}

func TestFormatAging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		hours *float64
		want  string
	}{
		{name: "tidak dapat dihitung tampil kosong", hours: nil, want: ""},
		{name: "nol jam tetap tampil, bukan kosong", hours: hoursPtr(0), want: "0 hours ago"},
		{name: "di bawah 24 jam dipotong, bukan dibulatkan", hours: hoursPtr(23.9), want: "23 hours ago"},
		{name: "tepat 24 jam berpindah ke bentuk hari", hours: hoursPtr(24), want: "1 days 0 hours ago"},
		{name: "lebih dari sehari", hours: hoursPtr(49.5), want: "2 days 1 hours ago"},
		{name: "sisa jam terbesar", hours: hoursPtr(47.99), want: "1 days 23 hours ago"},
		{name: "beberapa pekan", hours: hoursPtr(240.75), want: "10 days 0 hours ago"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, inboxcompliance.FormatAging(tc.hours))
		})
	}
}

// WithElapsed mengisi kolom Aging satu baris, dan AgingLabel membacanya.
func TestWorkItemWithElapsed(t *testing.T) {
	t.Parallel()

	sent := wib(2026, time.September, 21, 8, 0)
	now := wib(2026, time.September, 22, 8, 0)

	filled := inboxcompliance.WorkItem{ComplianceSentDate: &sent}.WithElapsed(now)

	require.NotNil(t, filled.AgingHours)
	require.InDelta(t, 24.0, *filled.AgingHours, 0.001)
	require.Equal(t, "1 days 0 hours ago", filled.AgingLabel())
}

func hoursPtr(v float64) *float64 { return &v }

// FormatElapsed menyusun kolom OutStanding pada tab Post Audit.
//
// Kasus pertama adalah SATU-SATUNYA yang teramati langsung dari layar Pega yang berjalan:
// baris bertanggal 22/04/25 menampilkan "1 year 5 months ago" pada 24 September 2026.
// Selebihnya rekonstruksi — lihat catatan di FormatElapsed.
func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	now := wib(2026, time.September, 24, 9, 57)

	tests := []struct {
		name string
		from time.Time
		want string
	}{
		{
			name: "teramati di layar Pega: 22/04/25 13:46 tampil sebagai 1 year 5 months ago",
			from: wib(2025, time.April, 22, 13, 46),
			want: "1 year 5 months ago",
		},
		{
			name: "tepat dua tahun: sisa nol bulan tidak ditulis",
			from: wib(2024, time.September, 24, 9, 57),
			want: "2 years ago",
		},
		{
			name: "beberapa bulan",
			from: wib(2026, time.June, 24, 9, 0),
			want: "3 months ago",
		},
		{
			name: "belum genap sebulan tampil sebagai hari",
			from: wib(2026, time.September, 4, 9, 0),
			want: "20 days ago",
		},
		{
			name: "belum genap sehari tampil sebagai jam",
			from: wib(2026, time.September, 24, 3, 57),
			want: "6 hours ago",
		},
		{
			name: "belum genap sejam tampil sebagai menit",
			from: wib(2026, time.September, 24, 9, 30),
			want: "27 minutes ago",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, inboxcompliance.FormatElapsed(&tc.from, now))
		})
	}
}

// OutStanding memakai waktu KALENDER, bukan jam kerja — dan pembedaannya terbukti dari
// angka di layar Pega.
//
// Dari 22 April 2025 ke 24 September 2026 ada 520 hari kalender. Bila akhir pekan dipotong
// seperti pada kolom Aging, tersisa sekitar 371 hari — yang akan berbunyi "1 year 0 months",
// bukan "1 year 5 months" seperti yang benar-benar ditampilkan Pega.
func TestOutstandingMemakaiWaktuKalenderBukanJamKerja(t *testing.T) {
	t.Parallel()

	from := wib(2025, time.April, 22, 13, 46)
	now := wib(2026, time.September, 24, 9, 57)

	require.Equal(t, "1 year 5 months ago", inboxcompliance.FormatElapsed(&from, now))

	// Pembandingnya: hitungan yang MEMOTONG akhir pekan menghasilkan angka yang jauh lebih
	// kecil, sehingga keduanya memang tidak boleh berbagi satu perhitungan.
	hours := inboxcompliance.AgingHoursBetween(&from, now)
	require.NotNil(t, hours)
	require.Less(t, *hours/24, 400.0, "hitungan jam kerja jauh lebih pendek dari 520 hari kalender")
}

// Tanggal kosong tidak menghasilkan "0 minutes ago" — sama seperti kolom Aging.
func TestFormatElapsedTanpaTanggal(t *testing.T) {
	t.Parallel()

	now := wib(2026, time.September, 24, 9, 57)

	require.Empty(t, inboxcompliance.FormatElapsed(nil, now))

	var zero time.Time
	require.Empty(t, inboxcompliance.FormatElapsed(&zero, now))

	future := wib(2026, time.October, 1, 9, 0)
	require.Empty(t, inboxcompliance.FormatElapsed(&future, now),
		"tanggal di masa depan tidak punya lama menggantung")
}

// Bentuk tunggal ditulis tanpa "s", dan itu terbaca dari satu contoh yang teramati.
//
// Layar Pega menampilkan "1 year 5 months ago" — tunggal dan jamak berdampingan dalam satu
// kalimat. Ia tidak menulis "1 years", sehingga pemformat ini pun tidak boleh.
func TestFormatElapsedBentukTunggal(t *testing.T) {
	t.Parallel()

	now := wib(2026, time.September, 24, 12, 0)

	tests := []struct {
		name string
		from time.Time
		want string
	}{
		{"satu tahun tepat", wib(2025, time.September, 24, 12, 0), "1 year ago"},
		{"satu tahun satu bulan", wib(2025, time.August, 24, 12, 0), "1 year 1 month ago"},
		{"satu bulan", wib(2026, time.August, 24, 12, 0), "1 month ago"},
		{"satu hari", wib(2026, time.September, 23, 12, 0), "1 day ago"},
		{"satu jam", wib(2026, time.September, 24, 11, 0), "1 hour ago"},
		{"satu menit", wib(2026, time.September, 24, 11, 59), "1 minute ago"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, inboxcompliance.FormatElapsed(&tc.from, now))
		})
	}
}
