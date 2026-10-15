package virtualaccount

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
)

// catalogStub adalah katalog alamat layanan tiruan.
type catalogStub struct {
	address     string
	err         error
	gotApp      string
	gotKind     string
	callCounter int
}

func (c *catalogStub) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	c.callCounter++
	c.gotApp = app
	c.gotKind = kind
	return c.address, c.err
}

var errNotRegistered = errors.New("baris katalog tidak ada")

func newPega(t *testing.T, catalog ServiceCatalog, client *http.Client) *Pega {
	t.Helper()
	p, err := NewPega(Options{
		Catalog:       catalog,
		User:          " petugas ",
		Password:      "rahasia",
		NotRegistered: errNotRegistered,
		Client:        client,
	})
	require.NoError(t, err)
	return p
}

func TestNewPegaRejectsIncompleteOptions(t *testing.T) {
	_, err := NewPega(Options{User: "u", Password: "p"})
	require.ErrorContains(t, err, "katalog layanan wajib diisi")

	_, err = NewPega(Options{Catalog: &catalogStub{}, User: "  ", Password: "p"})
	require.ErrorContains(t, err, "VIRTUAL_ACCOUNT_PENGGUNA")

	_, err = NewPega(Options{Catalog: &catalogStub{}, User: "u", Password: " "})
	require.ErrorContains(t, err, "VIRTUAL_ACCOUNT_SANDI")
}

func TestNewPegaAppliesDefaults(t *testing.T) {
	// Tanpa Client dan Timeout: batas waktu baku 30 detik, jenis layanan baku.
	p, err := NewPega(Options{Catalog: &catalogStub{}, User: " u ", Password: "p"})
	require.NoError(t, err)
	require.Equal(t, 30*time.Second, p.client.Timeout)
	require.Equal(t, DefaultServiceKind, p.serviceKind)
	require.Equal(t, "u", p.user)

	// Timeout yang disebut dipakai, ServiceKind menimpa nilai baku.
	p, err = NewPega(Options{Catalog: &catalogStub{}, User: "u", Password: "p", Timeout: 5 * time.Second, ServiceKind: "LAIN"})
	require.NoError(t, err)
	require.Equal(t, 5*time.Second, p.client.Timeout)
	require.Equal(t, "LAIN", p.serviceKind)

	// Client yang dipasok dipakai apa adanya.
	client := &http.Client{}
	p, err = NewPega(Options{Catalog: &catalogStub{}, User: "u", Password: "p", Client: client})
	require.NoError(t, err)
	require.Same(t, client, p.client)
}

func TestPegaIssueSendsRequestAndReadsFlatResponse(t *testing.T) {
	var gotBody request
	var gotUser, gotPass string
	var gotAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		gotUser, gotPass, gotAuth = r.BasicAuth()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = io.WriteString(w, `{"VirtualAccountNumber":" 8800123 ","Status":"OK","Message":"terbit"}`)
	}))
	defer server.Close()

	catalog := &catalogStub{address: server.URL}
	p := newPega(t, catalog, server.Client())

	va, err := p.Issue(context.Background(), " asm ", masterrecovery.VirtualAccountRequest{
		ClientID: " CL-1 ", PrincipalName: " PT A ",
	})
	require.NoError(t, err)
	require.Equal(t, masterrecovery.VirtualAccount{Number: "8800123", Status: "OK", Message: "terbit"}, va)

	require.Equal(t, "ASM", catalog.gotApp)
	require.Equal(t, DefaultServiceKind, catalog.gotKind)
	require.True(t, gotAuth)
	require.Equal(t, "petugas", gotUser)
	require.Equal(t, "rahasia", gotPass)
	require.Equal(t, request{SourceID: "LELANG", NoRef: "CL-1", CustomerName: "PT A", PaymentAmount: 0}, gotBody)
}

func TestPegaIssuePrefersNestedClaimData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"VANumber":"111","StatusCode":"X","ErrorMessage":"e",
			"ClaimData":{"VirtualAccountNumber":"222","Status":"S","Message":"M"}}`)
	}))
	defer server.Close()

	va, err := newPega(t, &catalogStub{address: server.URL}, server.Client()).
		Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.NoError(t, err)
	require.Equal(t, "222", va.Number)
	require.Equal(t, "S", va.Status)
	require.Equal(t, "M", va.Message)
}

func TestPegaIssueFallsBackToAlternativeKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"VirtualAccount":"333","StatusCode":"C","ErrorMessage":"EM"}`)
	}))
	defer server.Close()

	va, err := newPega(t, &catalogStub{address: server.URL}, server.Client()).
		Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.NoError(t, err)
	require.Equal(t, masterrecovery.VirtualAccount{Number: "333", Status: "C", Message: "EM"}, va)
}

func TestPegaIssueTreatsMissingNumberAsRejection(t *testing.T) {
	// Status 200 dengan pesan penolakan di badan — wajib dibaca sebagai penolakan.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"Status":"FAILED","Message":"principal tidak dikenal"}`)
	}))
	defer server.Close()

	_, err := newPega(t, &catalogStub{address: server.URL}, server.Client()).
		Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerRejected)
	require.ErrorContains(t, err, "principal tidak dikenal")
}

func TestPegaIssueRejectionWithoutMessageMentionsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	_, err := newPega(t, &catalogStub{address: server.URL}, server.Client()).
		Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerRejected)
	require.ErrorContains(t, err, "status HTTP 502")
}

func TestPegaIssueRejectsNonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `<html>error</html>`)
	}))
	defer server.Close()

	_, err := newPega(t, &catalogStub{address: server.URL}, server.Client()).
		Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
	require.ErrorContains(t, err, "status HTTP 500")
}

func TestPegaIssueCatalogErrors(t *testing.T) {
	// Baris katalog tidak ada → belum dikonfigurasi, tidak memanggil jaringan.
	p := newPega(t, &catalogStub{err: errNotRegistered}, nil)
	_, err := p.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnconfigured)
	require.ErrorContains(t, err, `TYPESERVICE="GENERATEDVA"`)

	// Galat lain → tidak dapat dihubungi.
	p = newPega(t, &catalogStub{err: errors.New("db mati")}, nil)
	_, err = p.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
	require.ErrorContains(t, err, "membaca alamat layanan")

	// Tanpa NotRegistered, ketiadaan baris terbaca sebagai tidak dapat dihubungi.
	p2, err := NewPega(Options{Catalog: &catalogStub{err: errNotRegistered}, User: "u", Password: "p"})
	require.NoError(t, err)
	_, err = p2.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
}

func TestPegaIssueInvalidAddress(t *testing.T) {
	// Alamat yang tidak dapat diurai gagal saat menyusun permintaan.
	p := newPega(t, &catalogStub{address: "://bukan-url"}, nil)
	_, err := p.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
	require.ErrorContains(t, err, "menyusun permintaan")
}

func TestPegaIssueNetworkFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address := server.URL
	server.Close()

	p := newPega(t, &catalogStub{address: address}, &http.Client{Timeout: time.Second})
	_, err := p.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
}

// errReader adalah badan respons yang gagal dibaca.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("putus") }
func (errReader) Close() error             { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPegaIssueBodyReadFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: errReader{}, Header: http.Header{}}, nil
	})}
	p := newPega(t, &catalogStub{address: "http://contoh.invalid/va"}, client)
	_, err := p.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, masterrecovery.ErrIssuerUnreachable)
	require.ErrorContains(t, err, "membaca respons")
}

func TestFirstNonEmpty(t *testing.T) {
	require.Equal(t, "b", firstNonEmpty("", "  ", " b ", "c"))
	require.Equal(t, "", firstNonEmpty("", " "))
}

func TestFakeIssuesStableNumberPerPrincipal(t *testing.T) {
	f := NewFake()
	first, err := f.Issue(context.Background(), "asm", masterrecovery.VirtualAccountRequest{ClientID: "C1", PrincipalName: "PT A"})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(first.Number, Prefix))
	require.Len(t, first.Number, len(Prefix)+12)
	require.Equal(t, "OK", first.Status)
	require.Contains(t, first.Message, "tiruan")

	// Principal yang sama (beda besar-kecil/spasi portal) → nomor yang sama.
	again, err := f.Issue(context.Background(), " ASM ", masterrecovery.VirtualAccountRequest{ClientID: "C1", PrincipalName: "PT A"})
	require.NoError(t, err)
	require.Equal(t, first.Number, again.Number)

	// Principal lain → nomor lain.
	other, err := f.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{ClientID: "C2", PrincipalName: "PT B"})
	require.NoError(t, err)
	require.NotEqual(t, first.Number, other.Number)
}

func TestFakeSetErrorFailsIssue(t *testing.T) {
	f := NewFake()
	want := errors.New("gagal")
	f.SetError(want)
	_, err := f.Issue(context.Background(), "ASM", masterrecovery.VirtualAccountRequest{})
	require.ErrorIs(t, err, want)
}
