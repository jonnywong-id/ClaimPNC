package portal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/portal/repo/memori"
)

func TestTandaiTersediaMembedakanPortalSiapDanBelum(t *testing.T) {
	daftar := memori.DaftarContoh()
	require.Len(t, daftar, 6, "POOLDATA.M_PORTAL_PNC memuat enam entitas")

	ditandai := portal.TandaiTersedia(daftar, []string{"ASM", "ASI"})
	require.Len(t, ditandai, 6, "portal yang belum siap tetap ditampilkan, bukan disembunyikan")

	siap := map[string]bool{}
	for _, p := range ditandai {
		siap[p.Alias] = p.Siap
	}
	require.True(t, siap["ASM"])
	require.True(t, siap["ASI"])
	require.False(t, siap["SMAS"], "kredensialnya belum diisi")
	require.False(t, siap["SPKS"])
}

// Kapitalisasi tidak boleh menentukan hasil. `11-SECURITY.md` §3.1 mencatat tiga nama
// access group Pega muncul dalam dua kapitalisasi dan perbandingan rule lama tidak
// konsisten soal itu; kesalahan yang sama tidak diulang.
func TestPencocokanAliasAbaiBesarKecilHuruf(t *testing.T) {
	daftar := memori.DaftarContoh()

	ditandai := portal.TandaiTersedia(daftar, []string{"asm", "  smas  "})
	siap := map[string]bool{}
	for _, p := range ditandai {
		siap[p.Alias] = p.Siap
	}
	require.True(t, siap["ASM"])
	require.True(t, siap["SMAS"])

	ditemukan, err := portal.Cari(daftar, "spk")
	require.NoError(t, err)
	require.Equal(t, "SINARMAS PENJAMINAN KREDIT", ditemukan.Nama)
}

func TestCariPortalYangTidakAda(t *testing.T) {
	_, err := portal.Cari(memori.DaftarContoh(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrTidakAda)
}

func TestDaftarContohSesuaiIsiTabel(t *testing.T) {
	// Nilai-nilai ini disalin dari Database/m_portal_pnc.csv yang diekspor Work Owner.
	// Bila tabelnya berubah, daftar contoh harus ikut — dan uji ini yang mengingatkan.
	daftar := memori.DaftarContoh()
	require.Equal(t, "202600101", daftar[0].ID)
	require.Equal(t, "ASURANSI SINAR MAS", daftar[0].Nama)
	require.Equal(t, "ASM", daftar[0].Alias)
	require.Equal(t, "SPKS", daftar[5].Alias)
	require.Equal(t, "SINARMAS PENJAMINAN KREDIT SYARIAH", daftar[5].Nama)
}
