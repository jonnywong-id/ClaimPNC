package reportkpi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Dropdown berisi 24 bulan: tahun lalu penuh, lalu tahun ini penuh.
func TestPeriodeKPIBerisiDuaTahunPenuh(t *testing.T) {
	options := reportkpi.BuildAdminPeriods(
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))

	require.Len(t, options, 24)
	require.Equal(t, "202501", options[0].Code, "bawaan adalah Januari tahun LALU")
	require.Equal(t, "202512", options[11].Code)
	require.Equal(t, "202601", options[12].Code)
	require.Equal(t, "202612", options[23].Code)

	// Label sama dengan kodenya — layar lama menampilkan YYYYMM apa adanya.
	for _, option := range options {
		require.Equal(t, option.Code, option.Label)
	}
}

// Bulan tahun ini yang BELUM terjadi tetap muncul.
//
// Perulangan Pega 12 kali tanpa memeriksa bulan berjalan. Memotongnya akan menghilangkan
// pilihan yang di Pega ada — selisih yang tidak diminta siapa pun.
func TestPeriodeKPIMemuatBulanYangBelumTerjadi(t *testing.T) {
	options := reportkpi.BuildAdminPeriods(
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))

	codes := make([]string, 0, len(options))
	for _, option := range options {
		codes = append(codes, option.Code)
	}
	require.Contains(t, codes, "202612", "Desember tahun ini tetap dapat dipilih")
}

// Pergantian tahun menggeser kedua tahunnya, bukan salah satu.
func TestPeriodeKPIBergeserSaatTahunBerganti(t *testing.T) {
	akhir := reportkpi.BuildAdminPeriods(
		time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC))
	awal := reportkpi.BuildAdminPeriods(
		time.Date(2027, 1, 1, 0, 1, 0, 0, time.UTC))

	require.Equal(t, "202501", akhir[0].Code)
	require.Equal(t, "202601", awal[0].Code)
	require.Equal(t, "202712", awal[23].Code)
}

// Satu bulan menjadi rentang penuh, dan batasnya INKLUSIF di kedua ujung.
func TestRentangPeriodeKPISatuBulanPenuh(t *testing.T) {
	from, to, err := reportkpi.AdminPeriodRange("202603")
	require.NoError(t, err)
	require.Equal(t, "2026-03-01", from.Format("2006-01-02"))
	require.Equal(t, "2026-03-31", to.Format("2006-01-02"))
}

// Februari tahun kabisat berakhir di tanggal 29.
//
// Inilah sebab hari terakhir DIHITUNG, bukan didaftar.
func TestRentangPeriodeKPIMenanganiTahunKabisat(t *testing.T) {
	_, to, err := reportkpi.AdminPeriodRange("202402")
	require.NoError(t, err)
	require.Equal(t, "2024-02-29", to.Format("2006-01-02"))

	_, to, err = reportkpi.AdminPeriodRange("202502")
	require.NoError(t, err)
	require.Equal(t, "2025-02-28", to.Format("2006-01-02"))
}

// Kode yang tidak berbentuk YYYYMM DITOLAK, bukan diam-diam menjadi rentang asal.
//
// Rentang yang salah tidak terlihat salah: layar tetap menggambar angka, hanya angka
// milik bulan yang bukan diminta.
func TestRentangPeriodeKPIMenolakKodeTakSah(t *testing.T) {
	for _, code := range []string{"", "2026", "2026-03", "202613", "abcdef", "2026031"} {
		_, _, err := reportkpi.AdminPeriodRange(code)
		require.Errorf(t, err, "kode %q seharusnya ditolak", code)
	}
}
