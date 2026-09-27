package reportklaim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func daftarContoh() *DominantFactors {
	return NewDominantFactors([][2]string{
		{"KLAIM-1", "Kelalaian"},
		{"KLAIM-1", "Cuaca"},
		{"KLAIM-2", "Kebakaran"},
	})
}

// Urutan JSON — yaitu urutan baris kueri — dipertahankan apa adanya.
func TestNamesMerangkaiMenurutUrutanBaris(t *testing.T) {
	require.Equal(t, "Kelalaian Cuaca", daftarContoh().Names("KLAIM-1"))
	require.Equal(t, "Kebakaran", daftarContoh().Names("KLAIM-2"))
}

// Klaim tanpa faktor dominan menghasilkan teks kosong, bukan tanda apa pun.
func TestNamesKlaimTanpaFaktorKosong(t *testing.T) {
	require.Empty(t, daftarContoh().Names("KLAIM-9"))
	require.Empty(t, daftarContoh().Names(""))
}

// Pemisahnya satu spasi, dan TIDAK ada spasi di depan — berbeda dari akumulator sistem
// lama yang menambahkan spasi sebelum setiap nama. Perbedaan itu disengaja dan dicatat.
func TestNamesTidakBerawalanSpasi(t *testing.T) {
	hasil := daftarContoh().Names("KLAIM-1")

	require.NotEqual(t, ' ', rune(hasil[0]))
	require.Equal(t, "Kelalaian Cuaca", hasil)
}

// Daftar yang sumbernya tidak terbaca menghasilkan teks kosong — bukan panik, dan bukan
// nama separuh.
func TestDaftarTidakTersediaMenghasilkanTeksKosong(t *testing.T) {
	require.Empty(t, UnavailableDominantFactors().Names("KLAIM-1"))
	require.False(t, UnavailableDominantFactors().Available())
	require.Zero(t, UnavailableDominantFactors().Count())
}

// Daftar yang terbaca tetapi KOSONG berbeda dari yang tidak terbaca: keduanya menghasilkan
// teks kosong, tetapi hanya yang pertama menyatakan "klaim ini memang tidak punya faktor".
func TestDaftarKosongBerbedaDariTidakTersedia(t *testing.T) {
	kosong := NewDominantFactors(nil)

	require.True(t, kosong.Available())
	require.Zero(t, kosong.Count())
	require.False(t, UnavailableDominantFactors().Available())
}

// Baris cacat — kunci atau nama kosong — dilewati, bukan menghasilkan spasi menggantung.
func TestBarisCacatDilewati(t *testing.T) {
	d := NewDominantFactors([][2]string{
		{"KLAIM-1", "Kelalaian"},
		{"KLAIM-1", ""},
		{"", "Cuaca"},
		{"KLAIM-1", "Banjir"},
	})

	require.Equal(t, "Kelalaian Banjir", d.Names("KLAIM-1"))
	require.Equal(t, 1, d.Count())
}
