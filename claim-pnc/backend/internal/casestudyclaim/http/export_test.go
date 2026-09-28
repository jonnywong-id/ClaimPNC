package casestudyclaimhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/logging"
)

// Uji INTERNAL — ia menyentuh exportRow dan decimalText yang tidak diekspor.
//
// Berkas uji lain modul ini berada di paket `_test` dan menembak lewat HTTP; yang di sini
// menjaga dua hal yang tidak dapat dilihat dari luar.

// Jumlah sel satu baris CSV WAJIB sama dengan jumlah judul kolomnya.
//
// # Kenapa uji ini ada
//
// `exportRow` menyusun selnya SATU PER SATU dan urutannya dijaga tangan, sedangkan
// judulnya datang dari `casestudyclaim.ColumnTitles()`. Menambah kolom di `columns.go`
// tanpa menambahnya di sini menghasilkan berkas CSV yang judulnya bergeser satu kolom
// terhadap isinya — **tanpa satu pun galat**, dan tanpa ada yang menyadarinya sampai
// seseorang membandingkan berkasnya dengan grid.
//
// Ia tidak dapat menangkap kolom yang TERTUKAR posisinya; yang menangkap itu adalah uji
// rute, yang memeriksa isi sel pada indeks tertentu.
func TestJumlahSelCSVSamaDenganJumlahJudulKolom(t *testing.T) {
	require.Len(t, exportRow(rowDTO{}), len(casestudyclaim.ColumnTitles()))
}

// Nilai uang ditulis sebagai ANGKA berdesimal titik, bukan "Rp 1.000.000,00".
//
// Berkas ini dibuka di pengolah angka, dan teks bersatuan tidak dapat dijumlahkan di sana.
func TestNilaiUangDitulisSebagaiAngka(t *testing.T) {
	amount := int64(550_000_000_000) // Rp 5.500.000.000,00
	require.Equal(t, "5500000000", decimalText(&amount, 2))

	withCents := int64(123_456) // Rp 1.234,56
	require.Equal(t, "1234.56", decimalText(&withCents, 2))

	// Nol di belakang koma dipangkas supaya nilai bulat tidak tampil sebagai "1234.50".
	half := int64(123_450)
	require.Equal(t, "1234.5", decimalText(&half, 2))
}

// Persentase memakai skala 4 — empat desimal, sesuai `D-51`.
func TestPersentaseDitulisEmpatDesimal(t *testing.T) {
	whole := int64(400_000) // 40%
	require.Equal(t, "40", decimalText(&whole, 4))

	fraction := int64(999_999) // 99,9999%
	require.Equal(t, "99.9999", decimalText(&fraction, 4))
}

// Nilai KOSONG menjadi sel kosong, BUKAN "0".
//
// Pada berkas yang dibuka di pengolah angka, nol adalah angka yang ikut terhitung dalam
// rata-rata dan penjumlahan. "Belum ada nilainya" bukan nol.
func TestNilaiKosongMenjadiSelKosong(t *testing.T) {
	require.Equal(t, "", decimalText(nil, 2))
	require.Equal(t, "", decimalText(nil, 4))
}

// Nilai negatif tetap terbaca sebagai angka negatif.
//
// Ia bukan kemustahilan di layar ini: kolom NILAI KLAIM NET adalah hasil pengurangan
// berlapis, dan salvage yang melampaui nilai klaim membuatnya negatif.
func TestNilaiNegatifTetapTerbaca(t *testing.T) {
	negative := int64(-123_456)
	require.Equal(t, "-1234.56", decimalText(&negative, 2))
}

// Penolakan lebar kolom dijawab 422 menunjuk isian "catatan", bukan 500.
//
// # Kenapa pemetaan ini dikunci uji
//
// Ia satu-satunya hal yang membuat keputusan "jangan kejar DDL, jalan seperti Pega" aman.
// Bila galat ini jatuh ke cabang bawaan, pengguna menerima "terjadi kesalahan pada sistem"
// setelah mengetik satu halaman penuh — dan tidak ada apa pun yang memberi tahu bahwa yang
// salah hanyalah panjang catatannya.
func TestPenolakanLebarKolomDijawab422(t *testing.T) {
	var status int
	var body ErrorResponse

	write := JSONWriter(func(_ http.ResponseWriter, _ *http.Request, code int, payload any) {
		status = code
		body = payload.(ErrorResponse)
	})

	// Cadangan sengaja diisi supaya uji ini membuktikan galatnya DIKENALI di sini — bukan
	// sekadar lolos karena tidak ada yang menanganinya.
	fallbackCalled := false
	fallback := ErrorWriter(func(http.ResponseWriter, *http.Request, error) {
		fallbackCalled = true
	})

	request := httptest.NewRequest(http.MethodPut, "/api/case-study-claim/STD-1/catatan", nil)
	WriteError(logging.New(0), write, fallback)(
		httptest.NewRecorder(), request,
		fmt.Errorf("%w: ORA-12899", casestudyclaim.ErrRemarkRejectedByColumn))

	require.False(t, fallbackCalled, "galat ini milik modul ini, bukan milik cadangan")
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.Equal(t, CodeValidation, body.Code)
	require.Equal(t, "catatan", body.Field)
	require.Contains(t, body.Message, "terlalu panjang")

	// Nomor galat Oracle TIDAK boleh sampai ke peramban — ia hanya masuk log.
	require.NotContains(t, body.Message, "ORA-")
}
