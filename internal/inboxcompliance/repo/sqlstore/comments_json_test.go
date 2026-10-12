package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

// Grid yang dirakit lalu diurai kembali harus utuh — nomor, tanggal, dan isinya.
//
// Ini uji yang paling sering disangka berlebihan dan paling sering menangkap sesuatu:
// kolom ini satu-satunya tempat komentar petugas tinggal, dan kehilangannya tidak
// menimbulkan galat apa pun — hanya grid yang tiba-tiba kosong.
func TestKomentarUtuhSetelahBolakBalik(t *testing.T) {
	saat := time.Date(2026, 10, 7, 9, 30, 0, 0, time.UTC)

	asli := []inboxcompliance.Comment{
		{Index: 1, Date: saat, Text: "Dokumen sudah lengkap."},
		{Index: 2, Date: saat.Add(2 * time.Hour), Text: "Menunggu konfirmasi cabang."},
	}

	encoded, err := encodeComments(asli)
	require.NoError(t, err)
	require.IsType(t, "", encoded, "grid berisi harus menghasilkan teks, bukan NULL")

	kembali, err := decodeComments(encoded.(string))
	require.NoError(t, err)
	require.Equal(t, asli, kembali)
}

// Grid kosong menghasilkan NULL, bukan "[]".
//
// Keduanya berbeda bagi pembaca SQL, dan kolomnya memang boleh NULL. Menyimpan "[]" akan
// membuat setiap keputusan tanpa komentar tetap menempati ruang dan tetap harus diurai.
func TestGridKosongMenghasilkanNull(t *testing.T) {
	encoded, err := encodeComments(nil)
	require.NoError(t, err)
	require.Nil(t, encoded)

	encoded, err = encodeComments([]inboxcompliance.Comment{})
	require.NoError(t, err)
	require.Nil(t, encoded)
}

// Kolom NULL atau kosong dibaca sebagai grid kosong, TANPA galat.
//
// Itu keadaan normal setiap klaim yang diputuskan tanpa komentar — bukan kerusakan.
func TestKolomKosongBukanGalat(t *testing.T) {
	kembali, err := decodeComments("")
	require.NoError(t, err)
	require.Empty(t, kembali)
}

// Teks rusak DILAPORKAN sebagai galat, bukan dibulatkan menjadi grid kosong.
//
// Ini butir terpenting di berkas ini. Membulatkannya akan menyembunyikan kehilangan data
// dengan cara yang paling mahal: petugas membuka form, melihat grid kosong, mengira belum
// pernah berkomentar, lalu menulis ulang — dan komentar lamanya tertimpa tanpa seorang pun
// tahu ia pernah ada.
func TestTeksRusakDilaporkan(t *testing.T) {
	for _, rusak := range []string{
		`[{"urutan":1,`,          // terpotong
		`bukan json sama sekali`, // bukan JSON
		`{"urutan":1}`,           // objek, bukan senarai
	} {
		_, err := decodeComments(rusak)
		require.Errorf(t, err, "teks rusak %q diterima diam-diam", rusak)
		require.ErrorContainsf(t, err, "KOMENTAR_JSON",
			"pesan galat tidak menyebut kolomnya, sehingga sulit ditelusuri")
	}
}
