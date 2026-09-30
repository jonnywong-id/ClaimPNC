package premiumlink_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/premiumlink"
)

type catalog string

func (c catalog) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	if app != "ASM" || kind != premiumlink.ServiceKind {
		return "", nil
	}
	return string(c), nil
}

func TestParseAcceptsTextAndNumbers(t *testing.T) {
	t.Parallel()
	s, err := premiumlink.Parse([]byte(`{"AgingAmount":"2.5","Payment":{"ListInstallment":[
		{"DueDate":"20260113","PaymentDate":"","PaymentAmount":null},
		{"DueDate":"20260213T000000.000 GMT","PaymentDate":"20260210","PaymentAmount":1500000}]}}`))
	require.NoError(t, err)
	require.Equal(t, "5/2", s.AgingAmount.RatString())
	require.Len(t, s.Installments, 2)
	require.Equal(t, time.Date(2026, 1, 13, 0, 0, 0, 0, time.UTC), s.Installments[0].DueDate)
	require.Empty(t, s.Installments[0].PaymentAmount)
	require.Equal(t, "1500000", s.Installments[1].PaymentAmount)

	empty, err := premiumlink.Parse([]byte(`{}`))
	require.NoError(t, err)
	require.Nil(t, empty.AgingAmount)
}

// Permintaan: POST tanpa badan, empat parameter query seperti getPremiumPaidOn_before.
func TestStatementSendsQueryParameters(t *testing.T) {
	t.Parallel()
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(`{"AgingAmount":0}`))
	}))
	defer srv.Close()

	h := premiumlink.New(catalog(srv.URL+"/premi"), nil)
	end := time.Date(2026, 12, 31, 17, 0, 0, 0, time.UTC)
	_, err := h.Statement(context.Background(), "ASM", registrasi.PremiumQuery{PolicyNumber: "POLIS-UJI", ProdKe: "1", RequestedAt: end, CaseID: "-"})
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, got.Method)
	require.Equal(t, "/premi", got.URL.Path)
	q := got.URL.Query()
	require.Equal(t, "POLIS-UJI", q.Get("noPolis"))
	require.Equal(t, "1", q.Get("prodKe"))
	require.Equal(t, "20261231T170000.000 GMT", q.Get("tglRequest"))
	require.Equal(t, "-", q.Get("caseId"))

	_, err = h.Statement(context.Background(), "ASI", registrasi.PremiumQuery{})
	require.ErrorIs(t, err, premiumlink.ErrServiceAddress)
}
