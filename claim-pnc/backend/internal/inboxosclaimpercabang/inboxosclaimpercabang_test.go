package inboxosclaimpercabang_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/clock"
)

// wibNoon adalah satu titik waktu tetap, dipakai seluruh uji umur.
//
// Ia dipatok, bukan diambil dari jam berjalan: umur yang bergerak setiap hari membuat uji
// lulus hari ini dan gagal bulan depan tanpa satu baris pun berubah.
func wibNoon(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, clock.ZoneWIB)
}

func at(t time.Time) *time.Time { return &t }

func TestAgingCountsCalendarDaysInWIB(t *testing.T) {
	now := wibNoon(2026, time.September, 28)

	cases := []struct {
		label      string
		registered time.Time
		want       int
	}{
		{"hari ini", wibNoon(2026, time.September, 28), 0},
		{"kemarin", wibNoon(2026, time.September, 27), 1},
		{"seminggu", wibNoon(2026, time.September, 21), 7},
		{"setahun", wibNoon(2025, time.September, 28), 365},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			require.Equal(t, c.want,
				inboxosclaimpercabang.AgingDaysSince(at(c.registered), now))
		})
	}
}

func TestAgingCountsYesterdayAsOneEvenWhenRegisteredInTheAfternoon(t *testing.T) {
	// Inilah kasus yang membuat hitungan ini pindah dari SQL ke Go.
	//
	// Bentuk portabel yang dianjurkan `09-DATABASE-STRATEGY.md` §4 —
	// `CAST(CURRENT_TIMESTAMP AS DATE) - CAST(registerdate AS DATE)` — TIDAK memangkas jam di
	// Oracle. Diukur langsung terhadap basis data dev, klaim yang terdaftar kemarin siang
	// menghasilkan `0.8758`, yang menjadi NOL hari begitu dipindai ke bilangan bulat.
	registered := time.Date(2026, time.September, 27, 13, 0, 0, 0, clock.ZoneWIB)
	now := time.Date(2026, time.September, 28, 10, 0, 0, 0, clock.ZoneWIB)

	require.Equal(t, 1, inboxosclaimpercabang.AgingDaysSince(at(registered), now),
		"klaim yang terdaftar kemarin siang harus berumur 1 hari, bukan 0")
}

func TestAgingUsesWIBDayBoundaryNotUTC(t *testing.T) {
	// Klaim yang terdaftar pukul 23.00 WIB masih tanggal 27 di WIB, tetapi sudah pukul 16.00
	// UTC pada tanggal yang sama — dan pada kasus lain pergeseran tujuh jam itu memindahkan
	// harinya. Batas hari yang dipakai aturan bisnis adalah tengah malam WIB (`F-5`).
	registered := time.Date(2026, time.September, 27, 23, 30, 0, 0, clock.ZoneWIB)
	now := time.Date(2026, time.September, 28, 0, 30, 0, 0, clock.ZoneWIB)

	require.Equal(t, 1, inboxosclaimpercabang.AgingDaysSince(at(registered), now),
		"pergantian hari WIB harus terhitung satu hari meski selisihnya satu jam")
}

func TestAgingIsZeroWhenUnknownOrInTheFuture(t *testing.T) {
	now := wibNoon(2026, time.September, 28)

	require.Equal(t, 0, inboxosclaimpercabang.AgingDaysSince(nil, now),
		"klaim tanpa tanggal registrasi tidak punya umur")

	future := wibNoon(2026, time.October, 5)
	require.Equal(t, 0, inboxosclaimpercabang.AgingDaysSince(at(future), now),
		"umur negatif bukan jawaban yang berarti bagi pembacanya")
}

func TestRedRuleFollowsTheTwoConditionsFromTheSection(t *testing.T) {
	// Aturannya di `pyInlineStyle` setiap sel grid:
	//
	//	<pega:when test='.AgingKlaim > 180 || .Medicare == 1'> color: red; </pega:when>
	cases := []struct {
		label   string
		aging   int
		stalled bool
		want    bool
	}{
		{"muda dan bergerak", 10, false, false},
		{"tepat di ambang", inboxosclaimpercabang.AgingThreshold, false, false},
		{"satu hari di atas ambang", inboxosclaimpercabang.AgingThreshold + 1, false, true},
		{"muda tetapi mandek", 3, true, true},
		{"tua dan mandek", 400, true, true},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			item := inboxosclaimpercabang.WorkItem{
				AgingDays:       c.aging,
				ProgressStalled: c.stalled,
			}
			require.Equal(t, c.want, item.NeedsAttention())
		})
	}
}

func TestThresholdIsStrictlyGreaterThan(t *testing.T) {
	// `>` bukan `>=`. Batas yang salah arah adalah tempat paling mudah bergeser saat aturan
	// disalin, dan pergeserannya hanya terlihat pada satu nilai saja.
	exact := inboxosclaimpercabang.WorkItem{AgingDays: inboxosclaimpercabang.AgingThreshold}
	require.False(t, exact.NeedsAttention(), "umur tepat 180 hari tidak digambar merah")

	require.Equal(t, 180, inboxosclaimpercabang.AgingThreshold,
		"ambang berubah dari yang tertulis di section")
}

func TestPaginationCorrectsInsteadOfRejecting(t *testing.T) {
	cases := []struct {
		label string
		given inboxosclaimpercabang.Pagination
		want  inboxosclaimpercabang.Pagination
	}{
		{
			"nol dibetulkan ke bawaan",
			inboxosclaimpercabang.Pagination{},
			inboxosclaimpercabang.Pagination{
				Page: 1, Size: inboxosclaimpercabang.DefaultPageSize},
		},
		{
			"halaman negatif menjadi pertama",
			inboxosclaimpercabang.Pagination{Page: -4, Size: 10},
			inboxosclaimpercabang.Pagination{Page: 1, Size: 10},
		},
		{
			"ukuran di atas batas dipotong",
			inboxosclaimpercabang.Pagination{Page: 2, Size: 5_000},
			inboxosclaimpercabang.Pagination{
				Page: 2, Size: inboxosclaimpercabang.MaxPageSize},
		},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			require.Equal(t, c.want, c.given.Normalize())
		})
	}
}

func TestTotalPagesNeverReportsZero(t *testing.T) {
	// "Halaman 1 dari 0" adalah kalimat yang tidak pernah benar, dan ia justru muncul pada
	// keadaan yang paling sering terjadi di layar ini — cabang yang sedang tidak punya
	// pekerjaan.
	empty := inboxosclaimpercabang.Page{
		Pagination: inboxosclaimpercabang.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 1, empty.TotalPages())

	partial := inboxosclaimpercabang.Page{
		Total:      26,
		Pagination: inboxosclaimpercabang.Pagination{Page: 1, Size: 25},
	}
	require.Equal(t, 2, partial.TotalPages())
}

func TestSliceCutsTheRequestedPage(t *testing.T) {
	all := []inboxosclaimpercabang.WorkItem{
		{ClaimNumber: "PNC-1"}, {ClaimNumber: "PNC-2"},
		{ClaimNumber: "PNC-3"}, {ClaimNumber: "PNC-4"},
	}

	second := inboxosclaimpercabang.Slice(all,
		inboxosclaimpercabang.Pagination{Page: 2, Size: 2})

	require.Equal(t, 4, second.Total, "jumlah seluruhnya bukan jumlah yang tampil")
	require.Len(t, second.Items, 2)
	require.Equal(t, "PNC-3", second.Items[0].ClaimNumber)

	beyond := inboxosclaimpercabang.Slice(all,
		inboxosclaimpercabang.Pagination{Page: 9, Size: 2})
	require.Empty(t, beyond.Items, "halaman di luar jangkauan kosong, bukan galat")
	require.Equal(t, 4, beyond.Total)
}

func TestCallerCleanTrimsBothFields(t *testing.T) {
	clean := inboxosclaimpercabang.Caller{
		Login:            "  PETUGAS1  ",
		DetailBranchCode: " 078 ",
	}.Clean()

	require.Equal(t, "PETUGAS1", clean.Login)
	require.Equal(t, "078", clean.DetailBranchCode,
		"spasi di ujung harus dipangkas: kode bersisa spasi tidak akan cocok dengan OLDID")
}

func TestCallerWithoutBranchKeepsAnEmptyCode(t *testing.T) {
	// Pengguna non-karyawan tidak punya cabang sama sekali: `POOLDATA.M_LOGIN_PNC` hanya
	// memuat LOGIN_ID, LOGIN_NAME, HASH_PASSWORD, ACTIVE_STATUS, dan LINE_BUSINESS.
	//
	// Itu keadaan yang mungkin, bukan kerusakan. Clean tidak boleh mengarangkan nilai
	// penggantinya — yang memutuskan artinya adalah penyimpanan, lewat BranchOf.
	clean := inboxosclaimpercabang.Caller{Login: "MITRA1"}.Clean()

	require.Equal(t, "MITRA1", clean.Login)
	require.Empty(t, clean.DetailBranchCode, "cabang kosong tetap kosong")
}

func TestPlannedDifferencesAreStated(t *testing.T) {
	// `D-54` menuntut setiap selisih terhadap Pega terpetakan dan dinyatakan. Daftar kosong
	// berarti selisihnya tidak pernah sampai ke pengguna yang membandingkan kedua layar.
	require.NotEmpty(t, inboxosclaimpercabang.PlannedDifferences)

	for _, difference := range inboxosclaimpercabang.PlannedDifferences {
		require.NotEmpty(t, difference)
	}
}
