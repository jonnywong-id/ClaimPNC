package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/repo/memory"
)

func list(t *testing.T, operator string, f inboxrcl.Filter) inboxrcl.Page {
	t.Helper()
	page, err := memory.NewSampleStore().List(context.Background(), operator, f)
	require.NoError(t, err)
	return page
}

func numbers(page inboxrcl.Page) []string {
	result := make([]string, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		result = append(result, task.ClaimNumber)
	}
	return result
}

// TestAntreanMengikutiKeempatPenyaringDanUrutannya adalah uji utama modul ini.
//
// Dari delapan baris contoh, tepat empat lolos — dan urutannya mengikuti
// `pxCreateDateTime DESC, pyID DESC`, termasuk dua baris yang waktunya sama persis.
func TestAntreanMengikutiKeempatPenyaringDanUrutannya(t *testing.T) {
	page := list(t, memory.SampleLegacyID, inboxrcl.Filter{Limit: 100})

	require.Equal(t, []string{
		"PNCN.26.0412", "PNCN.26.0405", "PNCN.26.0404", "PNCN.26.0380",
	}, numbers(page))
	require.Equal(t, 4, page.Total)
}

func TestKlaimDitolakTetapMunculYangTuntasKeluar(t *testing.T) {
	got := numbers(list(t, memory.SampleLegacyID, inboxrcl.Filter{Limit: 100}))

	require.Contains(t, got, "PNCN.26.0380", "Resolved-Rejected tetap di antrean")
	require.NotContains(t, got, "PNCN.26.0371", "Resolved-Completed keluar")
}

func TestKlaimYangBelumDikirimAnalisTidakMuncul(t *testing.T) {
	require.NotContains(t, numbers(list(t, memory.SampleLegacyID, inboxrcl.Filter{Limit: 100})),
		"PNCN.26.0366")
}

func TestKlaimDokterLainTidakMunculMeskiPenugasannyaMilikPemanggil(t *testing.T) {
	require.NotContains(t, numbers(list(t, memory.SampleLegacyID, inboxrcl.Filter{Limit: 100})),
		"PNCN.26.0359", "penyaring D harus menolaknya")
}

func TestIdentitasTidakPekaHurufBesarKecil(t *testing.T) {
	require.Len(t, list(t, "dokterrcl01", inboxrcl.Filter{Limit: 100}).Tasks, 4)
}

func TestIdentitasKosongTidakMencocokkanApaPun(t *testing.T) {
	require.Empty(t, list(t, "", inboxrcl.Filter{Limit: 100}).Tasks)
}

func TestPencarianMenyentuhNomorCaseDanNoPolis(t *testing.T) {
	require.Equal(t, []string{"PNCN.26.0404"},
		numbers(list(t, memory.SampleLegacyID, inboxrcl.Filter{Search: "26.005", Limit: 100})))
	require.Equal(t, []string{"PNCN.26.0412"},
		numbers(list(t, memory.SampleLegacyID, inboxrcl.Filter{Search: "pncn.26.0412", Limit: 100})))
}

func TestHalamanDiLuarJangkauanTetapMembawaTotal(t *testing.T) {
	page := list(t, memory.SampleLegacyID, inboxrcl.Filter{Offset: 50, Limit: 25})

	require.Empty(t, page.Tasks)
	require.Equal(t, 4, page.Total)
}

func TestIdentitasLamaHanyaDariGrupYangDiizinkan(t *testing.T) {
	store := memory.NewSampleStore()

	legacy, err := store.LegacyOperatorFor(context.Background(), memory.SampleLogin)
	require.NoError(t, err)
	require.Equal(t, memory.SampleLegacyID, legacy)

	// Grupnya tidak diizinkan, dan baris pada grup yang diizinkan tidak aktif.
	legacy, err = store.LegacyOperatorFor(context.Background(), memory.SampleLoginNoLegacy)
	require.NoError(t, err)
	require.Empty(t, legacy)

	legacy, err = store.LegacyOperatorFor(context.Background(), "tidakdikenal")
	require.NoError(t, err)
	require.Empty(t, legacy)
}
