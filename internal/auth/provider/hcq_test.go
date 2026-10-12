package provider_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// fakeCatalog menjawab dengan satu alamat, atau dengan galat yang diminta.
type fakeCatalog struct {
	address    string
	issues     error
	appDiminta string
	wantedKind string
}

func (k *fakeCatalog) ServiceAddress(_ context.Context, app, kind string) (string, error) {
	k.appDiminta, k.wantedKind = app, kind
	if k.issues != nil {
		return "", k.issues
	}
	return k.address, nil
}

// successResponse adalah bentuk respons nyata HCC/HCQ, disalin dari contoh yang
// diberikan Work Owner 2026-09-16 — termasuk blok Placement dan EmpLeader yang tidak
// seluruhnya dipakai.
const successResponse = `{
  "Response": {"pyStatusMessage":"Success","Alias":"VALID","pyErrorCode":"200","Result":"VALID, Data Ditemukan"},
  "EmpResponse": {
    "Placement": {
      "RegionCode":"1","PositionID":31,"pyCompany":"ASM","BranchID":691,
      "NewBranchCode":"100081","Name":"JONNY","RegionName":"KANTOR PUSAT",
      "BranchName":"KANTOR PUSAT","PositionName":"IT SPECIALIST","Login":"JONNYWONG8@YAHOO.COM",
      "BranchCode":"001","DetailBranchCode":"001","IsActive":true,"DivisionName":"IT",
      "NIK":"99091100","Alias":"JONNY"
    },
    "ContractCount":0,
    "Person": {
      "NIK":"99091100","pyCompany":"ASM","Login":"JONNYWONG8@YAHOO.COM",
      "pyEmail1":"contoh@example.invalid","Name":"JONNY"
    }
  },
  "EmpLeader": {"Person":{"NIK":"88880000","Login":"atasan@example.invalid","Name":"ATASAN"}},
  "Login": "JONNYWONG8@YAHOO.COM"
}`

// failureResponse adalah contoh kegagalan yang diberikan Work Owner.
const failureResponse = `{
  "Response": {"pyStatusMessage":"Failed","Alias":"INVALID","pyErrorCode":"403","Result":"INVALID, Username/Password Kosong"},
  "Login": "JONNYWONG8@YAHOO.COM",
  "Password": ""
}`

func hcqServer(t *testing.T, status int, body string, rekam func(*http.Request, []byte)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(content)
		if rekam != nil {
			rekam(r, content)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestHCQAcceptsValidCredential(t *testing.T) {
	var (
		header http.Header
		body   []byte
	)
	server := hcqServer(t, http.StatusOK, successResponse, func(r *http.Request, content []byte) {
		header = r.Header.Clone()
		body = content
	})

	katalog := &fakeCatalog{address: server.URL}
	hcq, err := provider.NewHCQ(provider.HCQOptions{
		Katalog: katalog, PortalAlias: "ASM", User: "aplikasi", Password: "sandiaplikasi",
	})
	require.NoError(t, err)

	profile, err := hcq.Verify(context.Background(),
		auth.Credential{Username: "JONNYWONG8@YAHOO.COM", Password: "sandipengguna"})
	require.NoError(t, err)

	t.Run("profil dipetakan dari Person dan Placement", func(t *testing.T) {
		require.Equal(t, "99091100", profile.Identity)
		require.Equal(t, "JONNY", profile.Name)
		require.Equal(t, auth.Employee, profile.Kind)
		require.Equal(t, "JONNYWONG8@YAHOO.COM", profile.Login)
		require.Equal(t, "contoh@example.invalid", profile.Email)
		require.Equal(t, "ASM", profile.Company)
		require.Equal(t, "KANTOR PUSAT", profile.Branch, "dari Placement.BranchName")
		require.Equal(t, "001", profile.BranchCode, "dari Placement.BranchCode")
		require.Equal(t, "IT SPECIALIST", profile.Position, "dari Placement.PositionName")
		require.NotNil(t, profile.ActiveAtSource)
		require.True(t, *profile.ActiveAtSource)
	})

	// Kolom APP diisi alias portal, bukan 'ASM' yang ditulis tetap — tiap entitas boleh
	// punya endpoint HCQ sendiri (koreksi Work Owner 2026-09-16).
	t.Run("alamat dicari dengan alias portal", func(t *testing.T) {
		require.Equal(t, "ASM", katalog.appDiminta)
		require.Equal(t, "HCQ-LOGIN", katalog.wantedKind)
	})

	t.Run("Basic Auth memakai kredensial aplikasi", func(t *testing.T) {
		password, existing := strings.CutPrefix(header.Get("Authorization"), "Basic ")
		require.True(t, existing, "header Authorization harus memakai skema Basic")
		terurai, err := base64.StdEncoding.DecodeString(password)
		require.NoError(t, err)
		require.Equal(t, "aplikasi:sandiaplikasi", string(terurai))
		require.NotContains(t, string(terurai), "sandipengguna",
			"kredensial pengguna tidak boleh dipakai sebagai Basic Auth")
	})

	t.Run("badan permintaan memakai bentuk Login dan Password", func(t *testing.T) {
		require.Contains(t, string(body), `"Login":"JONNYWONG8@YAHOO.COM"`)
		require.Contains(t, string(body), `"Password":"sandipengguna"`)
	})
}

// pyErrorCode selain "200" berarti gagal, dan dilaporkan sebagai kredensial salah supaya
// rantai meneruskannya ke jalur kedua.
func TestHCQRejectsWrongCredential(t *testing.T) {
	server := hcqServer(t, http.StatusOK, failureResponse, nil)
	hcq := testHCQ(t, server.URL)

	_, err := hcq.Verify(context.Background(),
		auth.Credential{Username: "JONNYWONG8@YAHOO.COM", Password: ""})
	require.ErrorIs(t, err, auth.ErrWrongCredential)
	require.NotErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
}

// Teks Result dari HCQ dapat membedakan "username tidak ditemukan" dari "password
// salah". Ia tidak boleh sampai ke pengguna.
func TestHCQDoesNotForwardResultText(t *testing.T) {
	server := hcqServer(t, http.StatusOK, failureResponse, nil)
	hcq := testHCQ(t, server.URL)

	_, err := hcq.Verify(context.Background(), auth.Credential{Username: "a", Password: "b"})
	require.NotContains(t, err.Error(), "Username/Password Kosong")
}

func TestHCQDistinguishesUnreachableSystem(t *testing.T) {
	t.Run("respons bukan JSON", func(t *testing.T) {
		server := hcqServer(t, http.StatusBadGateway, "<html>gateway error</html>", nil)
		_, err := testHCQ(t, server.URL).Verify(context.Background(),
			auth.Credential{Username: "a", Password: "b"})
		require.ErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
		require.NotErrorIs(t, err, auth.ErrWrongCredential)
	})

	t.Run("respons tanpa pyErrorCode", func(t *testing.T) {
		server := hcqServer(t, http.StatusOK, `{"Response":{}}`, nil)
		_, err := testHCQ(t, server.URL).Verify(context.Background(),
			auth.Credential{Username: "a", Password: "b"})
		require.ErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
	})

	t.Run("baris GCNM_CONNECT_REST belum ada", func(t *testing.T) {
		katalog := &fakeCatalog{issues: provider.ErrServiceNotRegistered}
		hcq, err := provider.NewHCQ(provider.HCQOptions{
			Katalog: katalog, PortalAlias: "SPKS", User: "u", Password: "p",
		})
		require.NoError(t, err)

		_, err = hcq.Verify(context.Background(), auth.Credential{Username: "a", Password: "b"})
		require.ErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
		// Galatnya menyebut persis baris mana yang kurang, supaya DBA tahu apa yang
		// harus dibuat tanpa menebak.
		require.Contains(t, err.Error(), "GCNM_CONNECT_REST")
		require.Contains(t, err.Error(), `"SPKS"`)
		require.Contains(t, err.Error(), `"HCQ-LOGIN"`)
	})
}

// HCQ menjawab berhasil tetapi tanpa NIK: meneruskannya berarti pengguna masuk tanpa
// dapat dikenali data klaimnya sendiri.
func TestHCQRejectsProfileWithoutIdentity(t *testing.T) {
	const withoutNIK = `{"Response":{"pyErrorCode":"200"},"EmpResponse":{"Person":{"Name":"JONNY"}}}`
	server := hcqServer(t, http.StatusOK, withoutNIK, nil)

	_, err := testHCQ(t, server.URL).Verify(context.Background(),
		auth.Credential{Username: "a", Password: "b"})
	var bolong *auth.IncompleteProfileError
	require.ErrorAs(t, err, &bolong)
	require.Equal(t, []string{"identitas"}, bolong.EmptyFields)
}

func TestHCQRejectsIncompleteDeps(t *testing.T) {
	_, err := provider.NewHCQ(provider.HCQOptions{PortalAlias: "ASM", User: "u", Password: "p"})
	require.Error(t, err)

	_, err = provider.NewHCQ(provider.HCQOptions{Katalog: &fakeCatalog{}, PortalAlias: "ASM", Password: "p"})
	require.ErrorContains(t, err, "HCQ_LOGIN_USER")

	_, err = provider.NewHCQ(provider.HCQOptions{Katalog: &fakeCatalog{}, PortalAlias: "ASM", User: "u"})
	require.ErrorContains(t, err, "HCQ_LOGIN_PASSWORD")
}

func testHCQ(t *testing.T, address string) *provider.HCQ {
	t.Helper()
	hcq, err := provider.NewHCQ(provider.HCQOptions{
		Katalog: &fakeCatalog{address: address}, PortalAlias: "ASM",
		User: "aplikasi", Password: "sandiaplikasi",
	})
	require.NoError(t, err)
	return hcq
}
