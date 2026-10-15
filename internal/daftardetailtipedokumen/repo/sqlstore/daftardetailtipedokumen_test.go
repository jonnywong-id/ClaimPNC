package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TGL_EDIT bertipe VARCHAR2, bukan DATE, dan isinya berformat Pega.
//
// Bentuknya dibaca langsung dari baris yang sudah tersimpan di Oracle pada 2026-09-23:
//
//	"20231030T075651.051 GMT"
//
// Uji ini mengunci bentuknya. Menulis `time.Time` apa adanya — yang sempat dilakukan —
// menghasilkan bentuk yang berbeda dari seluruh baris lain, dan perbedaan itu TIDAK
// menimbulkan galat: kolomnya menerima teks apa pun. Yang rusak hanyalah kemampuan
// mengurutkan dan membandingkan jejak simpan, dan itu baru terasa jauh di kemudian hari.
func TestEditTimestampFollowsThePegaTextFormat(t *testing.T) {
	at := time.Date(2023, 10, 30, 7, 56, 51, 51_000_000, time.UTC)

	require.Equal(t, "20231030T075651.051 GMT", pegaTimestamp(at))
}

// Waktu SELALU diubah ke UTC lebih dulu.
//
// Akhiran "GMT" pada baris lama menyatakan demikian. Menulis waktu lokal dengan akhiran
// GMT akan menggeser seluruh jejak tujuh jam tanpa satu pun tanda — persis kelas cacat
// yang `R-12` catat dan yang seam Clock (`F-5`) ada untuk mencegahnya.
func TestEditTimestampIsAlwaysConvertedToUTC(t *testing.T) {
	jakarta := time.FixedZone("WIB", 7*60*60)
	at := time.Date(2023, 10, 30, 14, 56, 51, 51_000_000, jakarta)

	require.Equal(t, "20231030T075651.051 GMT", pegaTimestamp(at))
}

// MIN_DOC dibaca dari kolom TEKS, dan isi yang bukan angka dijawab NOL.
//
// Pada view anak ia `VARCHAR2(10)` hasil `JSON_TABLE ... PATH '$.MIN_DOC'`, sehingga isinya
// dapat berupa apa saja yang pernah masuk ke dokumen JSON. Menggagalkan seluruh daftar
// karena satu baris warisan jauh lebih buruk daripada menampilkannya sebagai nol.
//
// Perlakuannya sengaja sama dengan MandatoryFrom: longgar saat membaca, tegas saat menulis.
func TestMinimumIsReadLenientlyFromATextColumn(t *testing.T) {
	require.Equal(t, 1, minimumFrom("1"))
	require.Equal(t, 12, minimumFrom(" 12 "))

	for _, stored := range []string{"", "   ", "-", "dua", "1.5"} {
		require.Equal(t, 0, minimumFrom(stored), "nilai %q seharusnya dibaca nol", stored)
	}
}
