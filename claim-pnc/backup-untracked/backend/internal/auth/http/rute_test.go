package authhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/auth/repo/memori"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/platform/waktu"

	authhttp "claim-pnc/internal/auth/http"
)

type peladen struct {
	server    *httptest.Server
	logBuffer *bytes.Buffer
	identitas *provider.Tiruan
	pengguna  *memori.PenggunaRepo
	jam       *waktu.JamTetap
}

func siapkanPeladen(t *testing.T) *peladen {
	t.Helper()

	sistemIdentitas, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	penggunaRepo := memori.PenggunaRepoBaru()
	jamUji := waktu.JamTetapPada(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC))

	layanan, err := usecase.LayananBaru(usecase.Opsi{
		Identitas:       sistemIdentitas,
		PenggunaRepo:    penggunaRepo,
		SesiRepo:        memori.SesiRepoBaru(),
		Jam:             jamUji,
		MasaBerlakuSesi: 30 * time.Minute,
	})
	require.NoError(t, err)

	// Log ditangkap ke buffer supaya isinya dapat diperiksa: token dan kata sandi
	// tidak boleh muncul di sana.
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := authhttp.HandlerBaru(layanan, logger)
	server := httptest.NewServer(httpserver.Router(httpserver.Bahan{
		Logger: logger,
		PasangAPI: func(api chi.Router) {
			authhttp.Pasang(api, handler, layanan, logger)
		},
	}))
	t.Cleanup(server.Close)

	return &peladen{server: server, logBuffer: buffer, identitas: sistemIdentitas, pengguna: penggunaRepo, jam: jamUji}
}

func (p *peladen) masuk(t *testing.T, namaPengguna, kataSandi string) (*http.Response, map[string]any) {
	t.Helper()
	badan := strings.NewReader(`{"nama_pengguna":"` + namaPengguna + `","kata_sandi":"` + kataSandi + `"}`)
	resp, err := http.Post(p.server.URL+"/api/masuk", "application/json", badan)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	var isi map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&isi))
	return resp, isi
}

func (p *peladen) panggil(t *testing.T, metode, jalur, token string) *http.Response {
	t.Helper()
	permintaan, err := http.NewRequestWithContext(context.Background(), metode, p.server.URL+jalur, nil)
	require.NoError(t, err)
	if token != "" {
		permintaan.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(permintaan)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestMasukBerhasilMengembalikanTokenDanProfil(t *testing.T) {
	p := siapkanPeladen(t)

	resp, isi := p.masuk(t, "adminpnc", "rahasia123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, isi["token"])
	require.Equal(t, "Bearer", isi["tipe_token"])
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

	profil, ok := isi["pengguna"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "90000001", profil["identitas"])
	require.NotContains(t, profil, "kata_sandi")
}

// Ketiga jenis galat masuk dibedakan lewat kode, bukan lewat teks pesan, dan
// masing-masing memakai status HTTP yang berbeda (TKT-U1-002).
func TestTigaJenisGalatMasukDibedakan(t *testing.T) {
	t.Run("kredensial salah", func(t *testing.T) {
		p := siapkanPeladen(t)
		resp, isi := p.masuk(t, "adminpnc", "salah")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, authhttp.KodeKredensialSalah, isi["kode"])
	})

	t.Run("pengguna tidak aktif", func(t *testing.T) {
		p := siapkanPeladen(t)
		resp, isi := p.masuk(t, "penggunanonaktif", "rahasia123")
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.Equal(t, authhttp.KodePenggunaTidakAktif, isi["kode"])
	})

	t.Run("sistem identitas tidak dapat dihubungi", func(t *testing.T) {
		p := siapkanPeladen(t)
		p.identitas.SetSimulasiPutus(true)
		resp, isi := p.masuk(t, "adminpnc", "rahasia123")
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		require.Equal(t, authhttp.KodeIdentitasPutus, isi["kode"])
	})
}

// Pesan galat untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah
// harus SAMA PERSIS — termasuk kode dan statusnya.
func TestGalatMasukTidakMembocorkanKeberadaanAkun(t *testing.T) {
	p := siapkanPeladen(t)

	respTidakAda, isiTidakAda := p.masuk(t, "tidakpernahada", "apa saja")
	respSandiSalah, isiSandiSalah := p.masuk(t, "adminpnc", "bukan sandinya")

	require.Equal(t, respTidakAda.StatusCode, respSandiSalah.StatusCode)
	require.Equal(t, isiTidakAda["kode"], isiSandiSalah["kode"])
	require.Equal(t, isiTidakAda["pesan"], isiSandiSalah["pesan"])
}

func TestSayaMengembalikanIdentitasPemanggil(t *testing.T) {
	p := siapkanPeladen(t)
	_, isi := p.masuk(t, "pictekniks", "rahasia123")
	token, _ := isi["token"].(string)

	resp := p.panggil(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var badan map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&badan))
	profil, ok := badan["pengguna"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "90000002", profil["identitas"])
}

func TestSayaTanpaTokenDitolak(t *testing.T) {
	p := siapkanPeladen(t)

	resp := p.panggil(t, http.MethodGet, "/api/saya", "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var badan map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&badan))
	require.Equal(t, authhttp.KodeSesiTidakSah, badan["kode"])
}

// Keluar mencabut sesi di server: memakai token lama setelah keluar harus DITOLAK
// (TKT-U1-002).
func TestKeluarMembuatTokenLamaDitolak(t *testing.T) {
	p := siapkanPeladen(t)
	_, isi := p.masuk(t, "adminpnc", "rahasia123")
	token, _ := isi["token"].(string)

	require.Equal(t, http.StatusOK, p.panggil(t, http.MethodGet, "/api/saya", token).StatusCode)

	respKeluar := p.panggil(t, http.MethodPost, "/api/keluar", token)
	require.Equal(t, http.StatusNoContent, respKeluar.StatusCode)

	respSetelah := p.panggil(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusUnauthorized, respSetelah.StatusCode)
}

// Sesi yang habis di tengah pekerjaan dijawab dengan kode yang BERBEDA dari token tidak
// sah, supaya frontend dapat menyelamatkan isian yang belum tersimpan.
func TestSesiKedaluwarsaPunyaKodeSendiri(t *testing.T) {
	p := siapkanPeladen(t)
	_, isi := p.masuk(t, "adminpnc", "rahasia123")
	token, _ := isi["token"].(string)

	p.jam.Maju(31 * time.Minute)

	resp := p.panggil(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var badan map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&badan))
	require.Equal(t, authhttp.KodeSesiKedaluwarsa, badan["kode"])
}

func TestPerpanjangSesiMenggeserBatasBerlaku(t *testing.T) {
	p := siapkanPeladen(t)
	_, isi := p.masuk(t, "adminpnc", "rahasia123")
	token, _ := isi["token"].(string)

	p.jam.Maju(25 * time.Minute)
	resp := p.panggil(t, http.MethodPost, "/api/sesi/perpanjang", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	p.jam.Maju(20 * time.Minute)
	require.Equal(t, http.StatusOK, p.panggil(t, http.MethodGet, "/api/saya", token).StatusCode)
}

// Kredensial dan token tidak pernah masuk log, termasuk pada jalur galat
// (TKT-F3-001, TKT-F3-003).
func TestKredensialDanTokenTidakPernahMasukLog(t *testing.T) {
	p := siapkanPeladen(t)

	_, isi := p.masuk(t, "adminpnc", "rahasia123")
	token, _ := isi["token"].(string)
	require.NotEmpty(t, token)

	p.panggil(t, http.MethodGet, "/api/saya", token)
	p.masuk(t, "adminpnc", "sandi-yang-salah-sekali")
	p.panggil(t, http.MethodPost, "/api/keluar", token)

	isiLog := p.logBuffer.String()
	require.NotEmpty(t, isiLog, "log harus terisi; kalau kosong uji ini tidak membuktikan apa pun")
	require.NotContains(t, isiLog, token, "token tidak boleh muncul di log")
	require.NotContains(t, isiLog, "rahasia123", "kata sandi tidak boleh muncul di log")
	require.NotContains(t, isiLog, "sandi-yang-salah-sekali", "kata sandi salah pun tidak boleh masuk log")
}

// Setiap permintaan membawa ID yang dapat dipakai menelusuri log satu keluhan pengguna.
func TestSetiapResponsMembawaIDPermintaan(t *testing.T) {
	p := siapkanPeladen(t)
	resp, _ := p.masuk(t, "adminpnc", "rahasia123")
	require.NotEmpty(t, resp.Header.Get("X-Request-Id"))
}

func TestPermintaanCacatDijawabTanpaMemantulkanIsinya(t *testing.T) {
	p := siapkanPeladen(t)

	resp, err := http.Post(p.server.URL+"/api/masuk", "application/json",
		strings.NewReader(`{"nama_pengguna": "adminpnc", "kata_sandi": `))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var badan map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&badan))
	require.Equal(t, authhttp.KodePermintaanCacat, badan["kode"])
	require.NotContains(t, badan["pesan"], "adminpnc")
}
