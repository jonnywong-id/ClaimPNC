package reportklaim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Keadaan AWAL kotak centang "Treaty" dikunci per kartu, dan ketiganya TIDAK sama.
//
// # Kenapa ini layak diuji
//
// Karena kotak ini memilih SUSUNAN KOLOM berkas, bukan menyaring baris. Keadaan awal yang
// keliru menghasilkan berkas bersusunan kolom berbeda dari Pega — tanpa pengguna menyentuh
// apa pun, dan tanpa satu pun tanda sampai kedua berkasnya dibandingkan berdampingan.
//
// Sebelum 2026-10-13 ketiganya mulai KOSONG, dan dua di antaranya salah.
//
// Nilainya dibaca dari layar Pega produksi, bukan dari harness: harness menyimpan keadaan
// awal kotak centang di tempat yang tidak terbaca dari XML-nya.
func TestKeadaanAwalTreatyDikunciPerKartu(t *testing.T) {
	harapan := map[Code]bool{
		CodeAkseptasi:     true,
		CodeOSKomite:      false, // satu-satunya yang kosong
		CodeOSBelumKomite: true,
	}

	punyaKotak := map[Code]bool{}
	for _, r := range CatalogInPegaOrder() {
		if !r.Uses.Has(FilterDetail) {
			require.Falsef(t, r.DetailDefault,
				"laporan %s tidak punya kotak Treaty tetapi DetailDefault-nya true", r.Code)
			continue
		}
		punyaKotak[r.Code] = r.DetailDefault
	}

	require.Equal(t, harapan, punyaKotak,
		"daftar kartu ber-Treaty ATAU keadaan awalnya berubah — cocokkan dulu ke layar Pega")
}
