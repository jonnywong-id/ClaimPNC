package masterpenyebabkerugian_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
)

// Spasi tepi dibuang dari ketiga field. Ini bukan kerapian kosmetik: keterangan yang
// membawa spasi tepi menghasilkan KELOMPOK TERSENDIRI pada `GROUP BY COL_DESC` di laporan
// XOL per bisnis, dan ID berpadding tidak dikenali sama dengan pasangannya yang bersih.
func TestCleanRemovesEdgeSpaces(t *testing.T) {
	clean := masterpenyebabkerugian.CauseOfLoss{
		ID:          " 1001 ",
		LegacyID:    " 01 ",
		Description: "  Kebakaran  ",
	}.Clean()

	require.Equal(t, "1001", clean.ID)
	require.Equal(t, "01", clean.LegacyID)
	require.Equal(t, "Kebakaran", clean.Description)
}

// Keterangan kosong DITERIMA — keputusan Work Owner 2026-09-20, meniru layar Pega apa
// adanya. `Section/BrowseCauseOfLoss-Section.xml` tidak memasang `pyRequired`, dan
// procedure lama tidak memeriksa apa pun sebelum menyisipkan.
func TestEmptyDescriptionAccepted(t *testing.T) {
	require.Empty(t, masterpenyebabkerugian.CheckDescription(""))
	require.Empty(t, masterpenyebabkerugian.CheckDescription("   "),
		"yang hanya berisi spasi sama dengan kosong")
}

// Batas panjang ada, dan ia penjaga basis data — bukan aturan bisnis. Tanpa batas, isian
// yang melebihi lebar kolom sampai ke pengguna sebagai 500 beserta nomor galat Oracle.
func TestDescriptionLengthGuard(t *testing.T) {
	atLimit := strings.Repeat("a", masterpenyebabkerugian.MaxDescriptionLength)
	require.Empty(t, masterpenyebabkerugian.CheckDescription(atLimit),
		"tepat pada batas masih diterima")

	tooLong := masterpenyebabkerugian.CheckDescription(atLimit + "a")
	require.Len(t, tooLong, 1)
	require.Equal(t, masterpenyebabkerugian.FieldDescription, tooLong[0].Field,
		"field disebut supaya layar dapat menandai kolom yang salah")
}

// Dihitung dalam rune, bukan byte. Satu huruf beraksen memakan dua byte, dan menghitung
// byte akan membuat batas terasa berubah-ubah bagi pengguna.
func TestDescriptionLengthCountedInRunes(t *testing.T) {
	accented := strings.Repeat("é", masterpenyebabkerugian.MaxDescriptionLength)
	require.Empty(t, masterpenyebabkerugian.CheckDescription(accented),
		"100 huruf beraksen adalah 100 karakter, bukan 200")
}

// Spasi tepi tidak ikut dihitung — isian yang panjangnya pas tetapi diketik dengan spasi
// di ujung tidak boleh ditolak.
func TestDescriptionLengthIgnoresEdgeSpaces(t *testing.T) {
	padded := "  " + strings.Repeat("a", masterpenyebabkerugian.MaxDescriptionLength) + "  "
	require.Empty(t, masterpenyebabkerugian.CheckDescription(padded))
}

// Pengurutan teks BENAR untuk modul ini justru karena bentuk ID-nya tetap: kode situs
// ditambah tiga digit ber-nol di depan. Inilah yang membedakannya dari Master Dominan
// Factor, yang ID-nya `max+1` tanpa nol di depan sehingga harus diurutkan numerik.
func TestSortByIDKeepsSequenceOrder(t *testing.T) {
	list := []masterpenyebabkerugian.CauseOfLoss{
		{ID: "1010"}, {ID: "1002"}, {ID: "1001"}, {ID: "1009"},
	}
	masterpenyebabkerugian.SortByID(list)

	require.Equal(t, []string{"1001", "1002", "1009", "1010"},
		[]string{list[0].ID, list[1].ID, list[2].ID, list[3].ID},
		"nol di depan membuat urutan teks sama dengan urutan numerik")
}

// Urutannya stabil: baris ber-ID sama mempertahankan urutan masuknya, bukan bertukar
// tempat setiap kali daftar dimuat.
func TestSortByIDIsStable(t *testing.T) {
	list := []masterpenyebabkerugian.CauseOfLoss{
		{ID: "1001", Description: "pertama"},
		{ID: "1001", Description: "kedua"},
	}
	masterpenyebabkerugian.SortByID(list)

	require.Equal(t, "pertama", list[0].Description)
	require.Equal(t, "kedua", list[1].Description)
}

// ValidationError memuat seluruh pelanggaran sekaligus, dan nil-nya benar-benar nil —
// bukan pointer nil terbungkus interface, yang akan membuat `if err != nil` di pemanggil
// berperilaku berbeda dari yang terbaca.
func TestNewValidationErrorReturnsTrueNil(t *testing.T) {
	require.NoError(t, masterpenyebabkerugian.NewValidationError(nil))
	require.Nil(t, masterpenyebabkerugian.NewValidationError([]masterpenyebabkerugian.Violation{}))

	err := masterpenyebabkerugian.NewValidationError([]masterpenyebabkerugian.Violation{
		{Field: masterpenyebabkerugian.FieldDescription, Message: "terlalu panjang"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), masterpenyebabkerugian.FieldDescription)
}
