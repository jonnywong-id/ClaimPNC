package premium_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/premium"
)

// katalog meniru POOLDATA.GCNM_CONNECT_REST.
type katalog struct {
	address   string
	err       error
	app, kind string
}

func (k *katalog) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	k.app, k.kind = app, kind
	return k.address, k.err
}

func pemeriksa(t *testing.T, address string) (*premium.Pega, *katalog) {
	t.Helper()
	cat := &katalog{address: address}
	checker, err := premium.NewPega(premium.Options{
		Catalog: cat,
		Now:     func() time.Time { return time.Date(2026, 9, 29, 3, 4, 5, 0, time.UTC) },
	})
	require.NoError(t, err)
	return checker, cat
}

func TestCekPremiMengirimPermintaanSepertiPega(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		_, _ = w.Write([]byte(`{"AgingAmount":"0"}`))
	}))
	defer server.Close()

	checker, cat := pemeriksa(t, server.URL+"/premi")
	answer, err := checker.CheckPremium(context.Background(), "asm",
		inboxautoclaim.PremiumQuery{PolicyNo: "0100120260500", ProductSeq: "2"})
	require.NoError(t, err)
	require.Equal(t, "0", answer.AgingAmount)

	// Alamat dicari per portal dengan TYPESERVICE PREMI.
	require.Equal(t, "ASM", cat.app)
	require.Equal(t, premium.DefaultServiceKind, cat.kind)

	// getPremiumPaidOn_before: POST dengan empat parameter kueri.
	require.Equal(t, http.MethodPost, received.Method)
	query := received.URL.Query()
	require.Equal(t, "0100120260500", query.Get("noPolis"))
	require.Equal(t, "2", query.Get("prodKe"))
	require.Equal(t, "20260929T030405.000 GMT", query.Get("tglRequest"))
	require.True(t, query.Has("caseId"))
}

func TestCekPremiMembacaAgingAmountDalamBerbagaiBentuk(t *testing.T) {
	for body, want := range map[string]string{
		`{"AgingAmount": 12.5}`:               "12.5",
		`{"agingamount":"3"}`:                 "3",
		`{"PaymentData":{"AgingAmount":"0"}}`: "0",
		`{"Status":"OK"}`:                     "", // kunci tidak ada -> kosong -> belum lunas
		`{"AgingAmount": null}`:               "",
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		checker, _ := pemeriksa(t, server.URL)
		answer, err := checker.CheckPremium(context.Background(), "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P"})
		server.Close()
		require.NoError(t, err, body)
		require.Equal(t, want, answer.AgingAmount, body)
	}
}

func TestCekPremiGagalBilaLayananBermasalah(t *testing.T) {
	t.Run("status HTTP bukan 2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		checker, _ := pemeriksa(t, server.URL)
		_, err := checker.CheckPremium(context.Background(), "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P"})
		require.Error(t, err)
	})

	t.Run("jawaban bukan JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>maintenance</html>"))
		}))
		defer server.Close()
		checker, _ := pemeriksa(t, server.URL)
		_, err := checker.CheckPremium(context.Background(), "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P"})
		require.Error(t, err)
	})

	t.Run("layanan tidak dapat dihubungi, dan alamatnya tidak bocor ke pesan", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		address := server.URL + "/rahasia-internal"
		server.Close() // layanan mati

		checker, _ := pemeriksa(t, address)
		_, err := checker.CheckPremium(context.Background(), "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P"})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "rahasia-internal")
		require.NotContains(t, err.Error(), server.Listener.Addr().String())
	})

	t.Run("alamat belum terdaftar untuk portal", func(t *testing.T) {
		checker, cat := pemeriksa(t, "")
		cat.err = errors.New("baris tidak ada")
		_, err := checker.CheckPremium(context.Background(), "SMI", inboxautoclaim.PremiumQuery{PolicyNo: "P"})
		require.Error(t, err)
	})
}

func TestTotalPremiMengirimPermintaanSepertiGetPremiumPaidSPK(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		_, _ = w.Write([]byte(`{"PaymentData":{"TotalPremiumPaid":125000000.5}}`))
	}))
	defer server.Close()

	checker, cat := pemeriksa(t, server.URL+"/getPaymentDataSumbis")
	total, err := checker.PremiumPaidBySource(context.Background(), "asm",
		inboxautoclaim.PremiumCheckQuery{BusinessCode: "10104", SourceOfBusiness: "KRDU"})
	require.NoError(t, err)
	require.Equal(t, "125000000.5", total)

	// Katalog dicari dengan TYPESERVICE PREMI-API, bukan PREMI.
	require.Equal(t, "ASM", cat.app)
	require.Equal(t, premium.DefaultTotalServiceKind, cat.kind)

	require.Equal(t, http.MethodPost, received.Method)
	require.Equal(t, "KRDU", received.URL.Query().Get("SourceOfBizCode"))
	require.Equal(t, "10104", received.URL.Query().Get("BizCode"))
}

func TestTotalPremiGagalTanpaMembocorkanAlamat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address := server.URL + "/rahasia-internal"
	server.Close()

	checker, _ := pemeriksa(t, address)
	_, err := checker.PremiumPaidBySource(context.Background(), "ASM",
		inboxautoclaim.PremiumCheckQuery{BusinessCode: "B", SourceOfBusiness: "S"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "rahasia-internal")
}
