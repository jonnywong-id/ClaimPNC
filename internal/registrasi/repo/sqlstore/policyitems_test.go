package sqlstore

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Coverage dan spreading tidak lagi dibaca dari dokumen JSON di dalam kolom BLOB, melainkan
// dari T_COVERAGELIST_* dan T_SPREADINGLIST (Work Owner, 2026-10-06). Yang tersisa untuk
// diuji di sini adalah dua hal yang menyatukan baris ketiga tabel itu — dan keduanya tidak
// menyentuh basis data sama sekali.

// TestJoinKeyNormalisesNumericIndex menjaga penggabungan tetap bekerja meski kolom kuncinya
// bertipe berbeda antartabel.
//
// INDEXCOVERAGE bertipe VARCHAR2(10) pada T_COVERAGELIST_CARGO dan NUMBER pada
// T_SPREADINGLIST. Nilai yang sama karena itu dapat datang sebagai "01" dan "1". Bila
// keduanya tidak dianggap sama, spreading sebuah coverage hilang TANPA galat — dan
// gejalanya muncul jauh dari sebabnya, sebagai klaim yang ditolak gerbang validasi karena
// total spreading-nya tidak genap seratus.
func TestJoinKeyNormalisesNumericIndex(t *testing.T) {
	key := func(value string) string { return joinKey(sql.NullString{String: value, Valid: true}) }

	require.Equal(t, key("1"), key("01"), "nol di depan tidak boleh membuat kunci berbeda")
	require.Equal(t, key("1"), key(" 1 "), "spasi tidak boleh membuat kunci berbeda")
	require.Equal(t, "0", key("000"), "nol tetap nol, bukan teks kosong")
	require.Equal(t, "", key(""), "nilai kosong tetap kosong")

	// Kunci yang BUKAN angka dibiarkan apa adanya: memangkas nol di depan sebuah kode
	// mengubah kodenya menjadi kode lain.
	require.Equal(t, "0A1", key("0A1"))
}

// TestSpreadingIndexFallsBackToCoverageCode menjaga jalur cadangan tetap ada.
//
// Pencocokan utama memakai INDEXOBJECT + INDEXCOVERAGE. Bila sebuah polis tidak mengisi
// INDEXCOVERAGE, seluruh spreading-nya akan hilang — karena itu pencocokan diulang lewat
// kode coverage. Uji ini juga membuktikan cadangan TIDAK dipakai saat kunci utama berisi,
// supaya baris yang sama tidak terhitung dua kali.
func TestSpreadingIndexFallsBackToCoverageCode(t *testing.T) {
	byIndex := []registrasi.SourceSpreading{{TreatyType: "10007", Share: registrasi.PercentFull}}
	byCode := []registrasi.SourceSpreading{{TreatyType: "10015", Share: registrasi.PercentFull}}

	index := spreadingIndex{
		byIndex: map[string][]registrasi.SourceSpreading{},
		byCode:  map[string][]registrasi.SourceSpreading{},
	}
	index.byIndex[spreadingKey("1", "2")] = byIndex
	index.byCode[spreadingKey("1", "11022")] = byCode

	require.Equal(t, byIndex, index.of("1", "2", "11022"), "kunci utama menang atas cadangan")
	require.Equal(t, byCode, index.of("1", "", "11022"), "tanpa INDEXCOVERAGE, kode coverage dipakai")
	require.Nil(t, index.of("1", "", ""), "tanpa keduanya, coverage memang tanpa spreading")
	require.Nil(t, index.of("9", "2", "11022"), "objek lain tidak boleh ikut terbawa")
}

// TestSpreadingKeySeparatesParts menjaga dua bagian kunci tetap terpisah.
//
// Perangkaian polos membuat pasangan ("1","23") dan ("12","3") bertabrakan — dua coverage
// milik objek berbeda akan berbagi spreading yang sama.
func TestSpreadingKeySeparatesParts(t *testing.T) {
	require.NotEqual(t, spreadingKey("1", "23"), spreadingKey("12", "3"))
}

// TestIsFlaggedReadsOnlyTrueMarkers menjaga penanda hapus tidak menyala karena nilai lain.
//
// Kolom bendera bertipe teks dan diisi bermacam-macam oleh sistem polis. Menganggap
// "apa pun selain kosong" sebagai menyala akan membuang coverage yang sah.
func TestIsFlaggedReadsOnlyTrueMarkers(t *testing.T) {
	flag := func(value string) bool { return isFlagged(sql.NullString{String: value, Valid: true}) }

	require.True(t, flag("1"))
	require.True(t, flag("y"))
	require.True(t, flag(" Y "))

	require.False(t, flag("0"))
	require.False(t, flag(""))
	require.False(t, flag("N"))
	require.False(t, isFlagged(sql.NullString{}))
}
