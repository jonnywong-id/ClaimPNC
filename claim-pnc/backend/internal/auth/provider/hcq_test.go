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

// katalogTiruan menjawab dengan satu alamat, atau dengan galat yang diminta.
type katalogTiruan struct {
	alamat       string
	galat        error
	appDiminta   string
	jenisDiminta string
}

func (k *katalogTiruan) AlamatLayanan(_ context.Context, app, jenis string) (string, error) {
	k.appDiminta, k.jenisDiminta = app, jenis
	if k.galat != nil {
		return "", k.galat
	}
	return k.alamat, nil
}

// responsBerhasil adalah bentuk respons nyata HCC/HCQ, disalin dari contoh yang
// diberikan Work Owner 2026-09-16 — termasuk blok Placement dan EmpLeader yang tidak
// seluruhnya dipakai.
const responsBerhasil = `{
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

// responsGagal adalah contoh kegagalan yang diberikan Work Owner.
const responsGagal = `{
  "Response": {"pyStatusMessage":"Failed","Alias":"INVALID","pyErrorCode":"403","Result":"INVALID, Username/Password Kosong"},
  "Login": "JONNYWONG8@YAHOO.COM",
  "Password": ""
}`

func peladenHCQ(t *testing.T, status int, badan string, rekam func(*http.Request, []byte)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isi := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(isi)
		if rekam != nil {
			rekam(r, isi)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(badan))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestHCQMenerimaKredensialSah(t *testing.T) {
	var (
		header http.Header
		badan  []byte
	)
	server := peladenHCQ(t, http.StatusOK, responsBerhasil, func(r *http.Request, isi []byte) {
		header = r.Header.Clone()
		badan = isi
	})

	katalog := &katalogTiruan{alamat: server.URL}
	hcq, err := provider.HCQBaru(provider.OpsiHCQ{
		Katalog: katalog, PortalAlias: "ASM", Pengguna: "aplikasi", KataSandi: "sandiaplikasi",
	})
	require.NoError(t, err)

	profil, err := hcq.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNYWONG8@YAHOO.COM", KataSandi: "sandipengguna"})
	require.NoError(t, err)

	t.Run("profil dipetakan dari Person dan Placement", func(t *testing.T) {
		require.Equal(t, "99091100", profil.Identitas)
		require.Equal(t, "JONNY", profil.Nama)
		require.Equal(t, auth.Karyawan, profil.Jenis)
		require.Equal(t, "JONNYWONG8@YAHOO.COM", profil.Login)
		require.Equal(t, "contoh@example.invalid", profil.Email)
		require.Equal(t, "ASM", profil.Perusahaan)
		require.Equal(t, "KANTOR PUSAT", profil.Cabang, "dari Placement.BranchName")
		require.Equal(t, "001", profil.KodeCabang, "dari Placement.BranchCode")
		require.Equal(t, "IT SPECIALIST", profil.Jabatan, "dari Placement.PositionName")
		require.NotNil(t, profil.AktifDiSumber)
		require.True(t, *profil.AktifDiSumber)
	})

	// Kolom APP diisi alias portal, bukan 'ASM' yang ditulis tetap — tiap entitas boleh
	// punya endpoint HCQ sendiri (koreksi Work Owner 2026-09-16).
	t.Run("alamat dicari dengan alias portal", func(t *testing.T) {
		require.Equal(t, "ASM", katalog.appDiminta)
		require.Equal(t, "HCQ-LOGIN", katalog.jenisDiminta)
	})

	t.Run("Basic Auth memakai kredensial aplikasi", func(t *testing.T) {
		sandi, ada := strings.CutPrefix(header.Get("Authorization"), "Basic ")
		require.True(t, ada, "header Authorization harus memakai skema Basic")
		terurai, err := base64.StdEncoding.DecodeString(sandi)
		require.NoError(t, err)
		require.Equal(t, "aplikasi:sandiaplikasi", string(terurai))
		require.NotContains(t, string(terurai), "sandipengguna",
			"kredensial pengguna tidak boleh dipakai sebagai Basic Auth")
	})

	t.Run("badan permintaan memakai bentuk Login dan Password", func(t *testing.T) {
		require.Contains(t, string(badan), `"Login":"JONNYWONG8@YAHOO.COM"`)
		require.Contains(t, string(badan), `"Password":"sandipengguna"`)
	})
}

// pyErrorCode selain "200" berarti gagal, dan dilaporkan sebagai kredensial salah supaya
// rantai meneruskannya ke jalur kedua.
func TestHCQMenolakKredensialSalah(t *testing.T) {
	server := peladenHCQ(t, http.StatusOK, responsGagal, nil)
	hcq := hcqUji(t, server.URL)

	_, err := hcq.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNYWONG8@YAHOO.COM", KataSandi: ""})
	require.ErrorIs(t, err, auth.ErrKredensialSalah)
	require.NotErrorIs(t, err, auth.ErrSistemTidakTerhubung)
}

// Teks Result dari HCQ dapat membedakan "username tidak ditemukan" dari "password
// salah". Ia tidak boleh sampai ke pengguna.
func TestHCQTidakMeneruskanTeksResult(t *testing.T) {
	server := peladenHCQ(t, http.StatusOK, responsGagal, nil)
	hcq := hcqUji(t, server.URL)

	_, err := hcq.Verifikasi(context.Background(), auth.Kredensial{NamaPengguna: "a", KataSandi: "b"})
	require.NotContains(t, err.Error(), "Username/Password Kosong")
}

func TestHCQMembedakanSistemTidakTerhubung(t *testing.T) {
	t.Run("respons bukan JSON", func(t *testing.T) {
		server := peladenHCQ(t, http.StatusBadGateway, "<html>gateway error</html>", nil)
		_, err := hcqUji(t, server.URL).Verifikasi(context.Background(),
			auth.Kredensial{NamaPengguna: "a", KataSandi: "b"})
		require.ErrorIs(t, err, auth.ErrSistemTidakTerhubung)
		require.NotErrorIs(t, err, auth.ErrKredensialSalah)
	})

	t.Run("respons tanpa pyErrorCode", func(t *testing.T) {
		server := peladenHCQ(t, http.StatusOK, `{"Response":{}}`, nil)
		_, err := hcqUji(t, server.URL).Verifikasi(context.Background(),
			auth.Kredensial{NamaPengguna: "a", KataSandi: "b"})
		require.ErrorIs(t, err, auth.ErrSistemTidakTerhubung)
	})

	t.Run("baris GCNM_CONNECT_REST belum ada", func(t *testing.T) {
		katalog := &katalogTiruan{galat: provider.ErrLayananTidakTerdaftar}
		hcq, err := provider.HCQBaru(provider.OpsiHCQ{
			Katalog: katalog, PortalAlias: "SPKS", Pengguna: "u", KataSandi: "p",
		})
		require.NoError(t, err)

		_, err = hcq.Verifikasi(context.Background(), auth.Kredensial{NamaPengguna: "a", KataSandi: "b"})
		require.ErrorIs(t, err, auth.ErrSistemTidakTerhubung)
		// Galatnya menyebut persis baris mana yang kurang, supaya DBA tahu apa yang
		// harus dibuat tanpa menebak.
		require.Contains(t, err.Error(), "GCNM_CONNECT_REST")
		require.Contains(t, err.Error(), `"SPKS"`)
		require.Contains(t, err.Error(), `"HCQ-LOGIN"`)
	})
}

// HCQ menjawab berhasil tetapi tanpa NIK: meneruskannya berarti pengguna masuk tanpa
// dapat dikenali data klaimnya sendiri.
func TestHCQMenolakProfilTanpaIdentitas(t *testing.T) {
	const tanpaNIK = `{"Response":{"pyErrorCode":"200"},"EmpResponse":{"Person":{"Name":"JONNY"}}}`
	server := peladenHCQ(t, http.StatusOK, tanpaNIK, nil)

	_, err := hcqUji(t, server.URL).Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "a", KataSandi: "b"})
	var bolong *auth.GalatProfilTidakLengkap
	require.ErrorAs(t, err, &bolong)
	require.Equal(t, []string{"identitas"}, bolong.FieldKosong)
}

func TestHCQMenolakBahanTidakLengkap(t *testing.T) {
	_, err := provider.HCQBaru(provider.OpsiHCQ{PortalAlias: "ASM", Pengguna: "u", KataSandi: "p"})
	require.Error(t, err)

	_, err = provider.HCQBaru(provider.OpsiHCQ{Katalog: &katalogTiruan{}, PortalAlias: "ASM", KataSandi: "p"})
	require.ErrorContains(t, err, "HCQ_LOGIN_USER")

	_, err = provider.HCQBaru(provider.OpsiHCQ{Katalog: &katalogTiruan{}, PortalAlias: "ASM", Pengguna: "u"})
	require.ErrorContains(t, err, "HCQ_LOGIN_PASSWORD")
}

func hcqUji(t *testing.T, alamat string) *provider.HCQ {
	t.Helper()
	hcq, err := provider.HCQBaru(provider.OpsiHCQ{
		Katalog: &katalogTiruan{alamat: alamat}, PortalAlias: "ASM",
		Pengguna: "aplikasi", KataSandi: "sandiaplikasi",
	})
	require.NoError(t, err)
	return hcq
}
