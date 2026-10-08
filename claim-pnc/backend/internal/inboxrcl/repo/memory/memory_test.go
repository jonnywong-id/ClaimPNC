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

// TestAntreanMengikutiKeempatPenyaringDanUrutannya adalah uji utama modul ini: dari delapan
// baris contoh tepat empat lolos, urut TGL_CREATE_PUCL DESC lalu CLAIMID DESC.
func TestAntreanMengikutiKeempatPenyaringDanUrutannya(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxrcl.Filter{Limit: 100})

	require.Equal(t, []string{
		"PNCN.26.0412", "PNCN.26.0405", "PNCN.26.0404", "PNCN.26.0380",
	}, numbers(page))
	require.Equal(t, 4, page.Total)
}

func TestPenyaringMenolakBarisYangTidakBerhak(t *testing.T) {
	got := numbers(list(t, memory.SampleOperator, inboxrcl.Filter{Limit: 100}))

	require.NotContains(t, got, "PNCN.26.0371", "B: Resolved-Completed keluar")
	require.NotContains(t, got, "PNCN.26.0366", "C: belum dikirim analis")
	require.NotContains(t, got, "PNCN.26.0359", "D: jalur PUCL tidak melewati dokter")
	require.NotContains(t, got, "PNCN.26.0350", "A: milik dokter lain")
	require.Contains(t, got, "PNCN.26.0380", "Resolved-Rejected tetap di antrean")
}

func TestIdentitasTidakPekaHurufBesarKecil(t *testing.T) {
	require.Len(t, list(t, "adminpnc", inboxrcl.Filter{Limit: 100}).Tasks, 4)
}

func TestOperatorKosongTidakMencocokkanApaPun(t *testing.T) {
	require.Empty(t, list(t, "", inboxrcl.Filter{Limit: 100}).Tasks)
}

func TestPencarianMenyentuhNomorCaseDanNoPolis(t *testing.T) {
	require.Equal(t, []string{"PNCN.26.0404"},
		numbers(list(t, memory.SampleOperator, inboxrcl.Filter{Search: "26.005", Limit: 100})))
	require.Equal(t, []string{"PNCN.26.0412"},
		numbers(list(t, memory.SampleOperator, inboxrcl.Filter{Search: "pncn.26.0412", Limit: 100})))
}

func TestHalamanDiLuarJangkauanTetapMembawaTotal(t *testing.T) {
	page := list(t, memory.SampleOperator, inboxrcl.Filter{Offset: 50, Limit: 25})

	require.Empty(t, page.Tasks)
	require.Equal(t, 4, page.Total)
}

// TestDetailHanyaUntukKlaimDiAntreanPemanggil — layar kerja tidak terbuka bagi klaim milik
// dokter lain, klaim PUCL, atau klaim yang sudah keluar antrean.
func TestDetailHanyaUntukKlaimDiAntreanPemanggil(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	detail, err := store.Detail(ctx, memory.SampleOperator, "pncn.26.0412")
	require.NoError(t, err)
	require.Equal(t, inboxrcl.ModeRCL, detail.Mode)
	require.Equal(t, "Penyakit tidak dijamin polis.", detail.Reason)

	detail, err = store.Detail(ctx, memory.SampleOperator, "PNCN.26.0405")
	require.NoError(t, err)
	require.Equal(t, inboxrcl.ModeMSIG, detail.Mode)

	for _, number := range []string{"PNCN.26.0350", "PNCN.26.0359", "PNCN.26.0371", "TIDAKADA"} {
		_, err := store.Detail(ctx, memory.SampleOperator, number)
		require.ErrorIsf(t, err, inboxrcl.ErrClaimNotFound, "klaim %s", number)
	}
}

func TestOperatorHanyaDariLoginAktif(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	operator, err := store.OperatorFor(ctx, memory.SampleLogin)
	require.NoError(t, err)
	require.Equal(t, memory.SampleOperator, operator)

	operator, err = store.OperatorFor(ctx, memory.SampleLoginInactive)
	require.NoError(t, err)
	require.Empty(t, operator, "login tidak aktif")

	operator, err = store.OperatorFor(ctx, "tidakdikenal")
	require.NoError(t, err)
	require.Empty(t, operator, "login tidak ada")
}
