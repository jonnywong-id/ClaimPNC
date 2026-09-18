package portal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/portal/repo/memory"
)

func TestMarkAvailableDistinguishesReadyFromNotReady(t *testing.T) {
	list := memory.SampleList()
	require.Len(t, list, 6, "POOLDATA.M_PORTAL_PNC memuat enam entitas")

	marked := portal.MarkAvailable(list, []string{"ASM", "ASI"})
	require.Len(t, marked, 6, "portal yang belum siap tetap ditampilkan, bukan disembunyikan")

	ready := map[string]bool{}
	for _, p := range marked {
		ready[p.Alias] = p.Ready
	}
	require.True(t, ready["ASM"])
	require.True(t, ready["ASI"])
	require.False(t, ready["SMAS"], "kredensialnya belum diisi")
	require.False(t, ready["SPKS"])
}

// Kapitalisasi tidak boleh menentukan hasil. `11-SECURITY.md` §3.1 mencatat tiga nama
// access group Pega muncul dalam dua kapitalisasi dan perbandingan rule lama tidak
// konsisten soal itu; kesalahan yang sama tidak diulang.
func TestAliasMatchingIgnoresCase(t *testing.T) {
	list := memory.SampleList()

	marked := portal.MarkAvailable(list, []string{"asm", "  smas  "})
	ready := map[string]bool{}
	for _, p := range marked {
		ready[p.Alias] = p.Ready
	}
	require.True(t, ready["ASM"])
	require.True(t, ready["SMAS"])

	found, err := portal.Find(list, "spk")
	require.NoError(t, err)
	require.Equal(t, "SINARMAS PENJAMINAN KREDIT", found.Name)
}

func TestFindPortalThatDoesNotExist(t *testing.T) {
	_, err := portal.Find(memory.SampleList(), "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotFound)
}

func TestSampleListMatchesTableContent(t *testing.T) {
	// Nilai-nilai ini disalin dari Database/m_portal_pnc.csv yang diekspor Work Owner.
	// Bila tabelnya berubah, daftar contoh harus ikut — dan uji ini yang mengingatkan.
	list := memory.SampleList()
	require.Equal(t, "202600101", list[0].ID)
	require.Equal(t, "ASURANSI SINAR MAS", list[0].Name)
	require.Equal(t, "ASM", list[0].Alias)
	require.Equal(t, "SPKS", list[5].Alias)
	require.Equal(t, "SINARMAS PENJAMINAN KREDIT SYARIAH", list[5].Name)
}
