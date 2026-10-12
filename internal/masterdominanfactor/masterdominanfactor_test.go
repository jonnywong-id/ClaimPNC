package masterdominanfactor_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdominanfactor"
)

// Dua aturan yang SENGAJA TIDAK ADA di modul ini, dan keduanya dikunci di sini supaya
// tidak "diperbaiki" tanpa keputusan baru.
//
// Work Owner memutuskannya pada 2026-09-20 setelah akibat keduanya dijelaskan: layar
// Pega menerima keduanya (`Section/DetailDominanFactor_Sec-Section.xml` —
// `pyRequired=false`, tanpa constraint keunikan), dan `P-5` menetapkan perilaku
// dipertahankan lebih dulu.
//
// Ini berbeda dari Master Status Klaim dan Master Tipe Surveyors, yang justru menolak
// keduanya. Perbedaan itu disengaja; bila uji ini gagal, artinya seseorang menyamakannya
// tanpa membaca keputusannya.
func TestEmptyNameAccepted(t *testing.T) {
	require.Empty(t, masterdominanfactor.CheckName(""),
		"nama kosong DITERIMA di modul ini — keputusan Work Owner 2026-09-20")
	require.Empty(t, masterdominanfactor.CheckName("   "),
		"nama berisi spasi saja sama dengan kosong, dan juga diterima")
}

func TestNameWithinLimitAccepted(t *testing.T) {
	require.Empty(t, masterdominanfactor.CheckName("Kelalaian pihak ketiga"))
	require.Empty(t, masterdominanfactor.CheckName(strings.Repeat("a", masterdominanfactor.MaxNameLength)),
		"tepat pada batas masih diterima")
}

// Batas panjang BUKAN aturan bisnis melainkan penjaga terhadap penolakan basis data:
// tanpa ini, isian yang melebihi lebar kolom sampai ke pengguna sebagai galat 500
// beserta nomor galat Oracle.
func TestNameBeyondLimitRejected(t *testing.T) {
	violations := masterdominanfactor.CheckName(strings.Repeat("a", masterdominanfactor.MaxNameLength+1))

	require.Len(t, violations, 1)
	require.Equal(t, masterdominanfactor.FieldName, violations[0].Field,
		"field disebut supaya layar dapat menandai kolom yang salah")
	require.Contains(t, violations[0].Message, "100")
}

// Panjang dihitung dalam rune, bukan byte. Satu huruf beraksen memakan dua byte, dan
// menghitungnya sebagai dua akan membuat batas terasa berubah-ubah bagi pengguna.
func TestLengthCountedInRunesNotBytes(t *testing.T) {
	// 100 huruf beraksen = 200 byte, tetapi tetap 100 rune.
	require.Empty(t, masterdominanfactor.CheckName(strings.Repeat("é", masterdominanfactor.MaxNameLength)))
}

// Perapian spasi tepi nyata gunanya di modul ini, bukan kerapian kosmetik: nama yang
// membawa spasi tepi muncul apa adanya di LISTAGG laporan Outstanding per Cabang, di
// antara koma pemisahnya.
func TestCleanRemovesEdgeSpaces(t *testing.T) {
	clean := masterdominanfactor.DominantFactor{ID: " 7  ", Name: "  Faktor Alam  "}.Clean()

	require.Equal(t, "7", clean.ID)
	require.Equal(t, "Faktor Alam", clean.Name)
}

// ID dibentuk `max+1` TANPA nol di depan, sehingga pengurutan sebagai teks menempatkan
// `10` sebelum `9`. Uji ini mengunci bahwa urutannya numerik.
func TestSortByIDIsNumericNotTextual(t *testing.T) {
	list := []masterdominanfactor.DominantFactor{
		{ID: "10", Name: "sepuluh"},
		{ID: "2", Name: "dua"},
		{ID: "9", Name: "sembilan"},
		{ID: "1", Name: "satu"},
	}
	masterdominanfactor.SortByID(list)

	require.Equal(t, []string{"1", "2", "9", "10"}, idsOf(list),
		"diurutkan sebagai teks, `10` akan mendahului `2` dan `9`")
}

// Baris ber-ID bukan bilangan ditempatkan di BELAKANG. Menaruhnya di depan akan
// menyembunyikan baris normal di bawah keanehan warisan.
func TestNonNumericIDsSortedLast(t *testing.T) {
	list := []masterdominanfactor.DominantFactor{
		{ID: "X1"},
		{ID: "3"},
		{ID: "A"},
		{ID: "1"},
	}
	masterdominanfactor.SortByID(list)

	require.Equal(t, []string{"1", "3", "A", "X1"}, idsOf(list))
}

// NextID meniru `select nvl(max(to_number(ID)),0)+1` pada
// Database/PEGA_M_DOMINAN_FACTOR.prc:11, termasuk hasilnya saat tabel kosong.
func TestNextIDMimicsProcedure(t *testing.T) {
	require.Equal(t, "1", masterdominanfactor.NextID(nil),
		"nvl(max(...),0)+1 pada tabel kosong menghasilkan 1")
	require.Equal(t, "1", masterdominanfactor.NextID([]string{}))
	require.Equal(t, "4", masterdominanfactor.NextID([]string{"1", "2", "3"}))

	// Yang diambil adalah nilai TERBESAR, bukan jumlah baris. Bila deretnya berlubang —
	// misalnya karena baris pernah dihapus langsung di basis data — memakai jumlah baris
	// akan menghasilkan nomor yang sudah dipakai.
	require.Equal(t, "12", masterdominanfactor.NextID([]string{"1", "11", "5"}),
		"lubang di tengah deret tidak boleh membuat nomor terbitkan ulang")
}

// Satu baris warisan yang ID-nya bukan bilangan tidak boleh membuat penambahan faktor
// baru mustahil. Procedure lama justru sebaliknya: `to_number(ID)` gagal dengan ORA-01722
// dan membatalkan seluruh penambahan.
func TestNextIDSkipsNonNumericIDs(t *testing.T) {
	require.Equal(t, "8", masterdominanfactor.NextID([]string{"7", "AB", "", "  "}),
		"baris cacat dilewati, bukan menjatuhkan penambahan")
}

func TestNumericIDRejectsNonNumbers(t *testing.T) {
	for _, id := range []string{"", "  ", "A", "1A", "-3", "1.5"} {
		_, isNumber := masterdominanfactor.NumericID(id)
		require.Falsef(t, isNumber, "ID %q bukan bilangan", id)
	}

	n, isNumber := masterdominanfactor.NumericID("  42 ")
	require.True(t, isNumber, "spasi tepi dibuang sebelum dibaca")
	require.Equal(t, int64(42), n)
}

func idsOf(list []masterdominanfactor.DominantFactor) []string {
	result := make([]string, 0, len(list))
	for _, f := range list {
		result = append(result, f.ID)
	}
	return result
}
