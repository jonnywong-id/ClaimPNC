package sqlstore_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/pelaporanklaim/repo/sqlstore"
)

// Nomor laporan berbentuk LPK.YY.xxxx, mengikuti bentuk nomor klaim yang `D-71` tetapkan.
func TestBuildNumberHasThreeSegments(t *testing.T) {
	recordedAt := time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)

	require.Equal(t, "LPK.26.0001", sqlstore.BuildNumber(recordedAt, 1))
	require.Equal(t, "LPK.26.0148", sqlstore.BuildNumber(recordedAt, 148))
	require.Equal(t, "LPK.26.9999", sqlstore.BuildNumber(recordedAt, 9999))
}

// Lebar segmen terakhir TETAP, dengan nol di depan.
//
// Ini memperbaiki cacat yang `D-71` catat pada nomor klaim: sintaks yang ditetapkan di
// sana memakai TO_CHAR tanpa format mask, sehingga lebarnya berubah-ubah dan pengurutan
// sebagai TEKS tidak sesuai urutan penerbitan.
//
// Uji ini membuktikan perbaikannya dengan cara yang paling langsung: mengurutkan nomor
// sebagai teks harus menghasilkan urutan penerbitan yang sama.
func TestTextSortMatchesIssueOrder(t *testing.T) {
	recordedAt := time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)

	nine := sqlstore.BuildNumber(recordedAt, 9)
	ten := sqlstore.BuildNumber(recordedAt, 10)
	thousand := sqlstore.BuildNumber(recordedAt, 1000)

	require.Less(t, nine, ten,
		"nomor ke-9 harus mendahului ke-10 saat diurutkan sebagai teks")
	require.Less(t, ten, thousand)
}

// Di atas 9.999 nomornya melebar, dan pengurutan teks kembali menyimpang pada tahun itu.
//
// Perilakunya dibiarkan apa adanya dan batasnya dicatat — memotongnya menjadi empat digit
// akan menghasilkan nomor GANDA, yang jauh lebih buruk daripada pengurutan yang melenceng.
func TestAboveTenThousandTheNumberWidensOnPurpose(t *testing.T) {
	recordedAt := time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)

	require.Equal(t, "LPK.26.10000", sqlstore.BuildNumber(recordedAt, 10000),
		"nomor melebar, bukan terpotong; nomor ganda jauh lebih berbahaya")
}

// Tahun diambil dari zona WIB, bukan UTC.
//
// Keduanya berbeda selama tujuh jam setiap pergantian tahun. Laporan yang dicatat pada
// 1 Januari pukul 05:00 WIB akan bernomor tahun LALU bila UTC yang dipakai — dan nomor
// yang salah tahun tidak dapat diperbaiki setelah terbit, karena ia sudah dicetak di
// dokumen dan dirujuk klaimnya.
func TestYearTakenFromWIBZone(t *testing.T) {
	// 31 Desember 2026 pukul 22:00 UTC = 1 Januari 2027 pukul 05:00 WIB.
	newYearEve := time.Date(2026, time.December, 31, 22, 0, 0, 0, time.UTC)

	require.Equal(t, "LPK.27.0001", sqlstore.BuildNumber(newYearEve, 1),
		"pukul 05:00 WIB tanggal 1 Januari sudah tahun baru bagi pengguna di Jakarta")

	// 31 Desember 2026 pukul 16:00 UTC = 31 Desember 2026 pukul 23:00 WIB — masih tahun
	// lama di kedua zona.
	stillOldYear := time.Date(2026, time.December, 31, 16, 0, 0, 0, time.UTC)
	require.Equal(t, "LPK.26.0001", sqlstore.BuildNumber(stillOldYear, 1))
}

// Tahun 2000-an ditulis dua digit dengan nol di depan.
//
// Tanpa pemformatan dua digit, tahun 2007 akan menghasilkan "LPK.7.0001" — tiga segmen
// yang lebarnya tidak seragam dan tidak dapat diurai kembali dengan pasti.
func TestSingleDigitYearWrittenAsTwoDigits(t *testing.T) {
	earlyYear := time.Date(2007, time.March, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, "LPK.07.0001", sqlstore.BuildNumber(earlyYear, 1))
}
