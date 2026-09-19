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
	"claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"

	authhttp "claim-pnc/internal/auth/http"
)

type testServer struct {
	http      *httptest.Server
	logBuffer *bytes.Buffer
	identity  *provider.Fake
	user      *memory.UserRepo
	clock     *clock.Fixed
}

func startServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	userRepo := memory.NewUserRepo()
	testClock := clock.FixedAt(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC))

	service, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        userRepo,
		SessionRepo:     memory.NewSessionRepo(),
		Clock:           testClock,
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	// Log ditangkap ke buffer supaya isinya dapat diperiksa: token dan kata sandi
	// tidak boleh muncul di sana.
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := authhttp.NewHandler(service, logger)
	httpSrv := httptest.NewServer(httpserver.Router(httpserver.Deps{
		Logger: logger,
		MountAPI: func(api chi.Router) {
			authhttp.Mount(api, handler, service, logger)
		},
	}))
	t.Cleanup(httpSrv.Close)

	return &testServer{http: httpSrv, logBuffer: buffer, identity: identitySystem, user: userRepo, clock: testClock}
}

func (p *testServer) login(t *testing.T, username, password string) (*http.Response, map[string]any) {
	t.Helper()
	body := strings.NewReader(`{"nama_pengguna":"` + username + `","kata_sandi":"` + password + `"}`)
	resp, err := http.Post(p.http.URL+"/api/masuk", "application/json", body)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	var content map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&content))
	return resp, content
}

func (p *testServer) call(t *testing.T, metode, filePath, token string) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), metode, p.http.URL+filePath, nil)
	require.NoError(t, err)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestSuccessfulLoginReturnsTokenAndProfile(t *testing.T) {
	p := startServer(t)

	resp, content := p.login(t, "adminpnc", "rahasia123")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, content["token"])
	require.Equal(t, "Bearer", content["tipe_token"])
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))

	profile, ok := content["pengguna"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "90000001", profile["identitas"])
	require.NotContains(t, profile, "kata_sandi")
}

// Ketiga jenis galat masuk dibedakan lewat kode, bukan lewat teks pesan, dan
// masing-masing memakai status HTTP yang berbeda (TKT-U1-002).
func TestThreeLoginErrorKindsAreDistinguished(t *testing.T) {
	t.Run("kredensial salah", func(t *testing.T) {
		p := startServer(t)
		resp, content := p.login(t, "adminpnc", "salah")
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, authhttp.CodeWrongCredential, content["kode"])
	})

	t.Run("pengguna tidak aktif", func(t *testing.T) {
		p := startServer(t)
		resp, content := p.login(t, "penggunanonaktif", "rahasia123")
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.Equal(t, authhttp.CodeUserInactive, content["kode"])
	})

	t.Run("sistem identitas tidak dapat dihubungi", func(t *testing.T) {
		p := startServer(t)
		p.identity.SetSimulateOutage(true)
		resp, content := p.login(t, "adminpnc", "rahasia123")
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		require.Equal(t, authhttp.CodeIdentityDown, content["kode"])
	})
}

// Pesan galat untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah
// harus SAMA PERSIS — termasuk kode dan statusnya.
func TestLoginErrorDoesNotLeakAccountExistence(t *testing.T) {
	p := startServer(t)

	respTidakAda, notFoundBody := p.login(t, "tidakpernahada", "apa saja")
	wrongPasswordResponse, wrongPasswordBody := p.login(t, "adminpnc", "bukan sandinya")

	require.Equal(t, respTidakAda.StatusCode, wrongPasswordResponse.StatusCode)
	require.Equal(t, notFoundBody["kode"], wrongPasswordBody["kode"])
	require.Equal(t, notFoundBody["pesan"], wrongPasswordBody["pesan"])
}

func TestMeReturnsCallerIdentity(t *testing.T) {
	p := startServer(t)
	_, content := p.login(t, "pictekniks", "rahasia123")
	token, _ := content["token"].(string)

	resp := p.call(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	profile, ok := body["pengguna"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "90000002", profile["identitas"])
}

func TestMeWithoutTokenRejected(t *testing.T) {
	p := startServer(t)

	resp := p.call(t, http.MethodGet, "/api/saya", "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, authhttp.CodeInvalidSession, body["kode"])
}

// Logout mencabut sesi di server: memakai token lama setelah keluar harus DITOLAK
// (TKT-U1-002).
func TestLogoutMakesOldTokenRejected(t *testing.T) {
	p := startServer(t)
	_, content := p.login(t, "adminpnc", "rahasia123")
	token, _ := content["token"].(string)

	require.Equal(t, http.StatusOK, p.call(t, http.MethodGet, "/api/saya", token).StatusCode)

	outageResponse := p.call(t, http.MethodPost, "/api/keluar", token)
	require.Equal(t, http.StatusNoContent, outageResponse.StatusCode)

	respSetelah := p.call(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusUnauthorized, respSetelah.StatusCode)
}

// Sesi yang habis di tengah pekerjaan dijawab dengan kode yang BERBEDA dari token tidak
// sah, supaya frontend dapat menyelamatkan isian yang belum tersimpan.
func TestExpiredSessionHasItsOwnCode(t *testing.T) {
	p := startServer(t)
	_, content := p.login(t, "adminpnc", "rahasia123")
	token, _ := content["token"].(string)

	p.clock.Advance(31 * time.Minute)

	resp := p.call(t, http.MethodGet, "/api/saya", token)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, authhttp.CodeSessionExpired, body["kode"])
}

func TestRenewSessionShiftsExpiry(t *testing.T) {
	p := startServer(t)
	_, content := p.login(t, "adminpnc", "rahasia123")
	token, _ := content["token"].(string)

	p.clock.Advance(25 * time.Minute)
	resp := p.call(t, http.MethodPost, "/api/sesi/perpanjang", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	p.clock.Advance(20 * time.Minute)
	require.Equal(t, http.StatusOK, p.call(t, http.MethodGet, "/api/saya", token).StatusCode)
}

// Kredensial dan token tidak pernah masuk log, termasuk pada jalur galat
// (TKT-F3-001, TKT-F3-003).
func TestCredentialAndTokenNeverEnterLog(t *testing.T) {
	p := startServer(t)

	_, content := p.login(t, "adminpnc", "rahasia123")
	token, _ := content["token"].(string)
	require.NotEmpty(t, token)

	p.call(t, http.MethodGet, "/api/saya", token)
	p.login(t, "adminpnc", "sandi-yang-salah-sekali")
	p.call(t, http.MethodPost, "/api/keluar", token)

	logContent := p.logBuffer.String()
	require.NotEmpty(t, logContent, "log harus terisi; kalau kosong uji ini tidak membuktikan apa pun")
	require.NotContains(t, logContent, token, "token tidak boleh muncul di log")
	require.NotContains(t, logContent, "rahasia123", "kata sandi tidak boleh muncul di log")
	require.NotContains(t, logContent, "sandi-yang-salah-sekali", "kata sandi salah pun tidak boleh masuk log")
}

// Setiap permintaan membawa ID yang dapat dipakai menelusuri log satu keluhan pengguna.
func TestEveryResponseCarriesRequestID(t *testing.T) {
	p := startServer(t)
	resp, _ := p.login(t, "adminpnc", "rahasia123")
	require.NotEmpty(t, resp.Header.Get("X-Request-Id"))
}

func TestMalformedRequestAnsweredWithoutEchoingBody(t *testing.T) {
	p := startServer(t)

	resp, err := http.Post(p.http.URL+"/api/masuk", "application/json",
		strings.NewReader(`{"nama_pengguna": "adminpnc", "kata_sandi": `))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, authhttp.CodeMalformedRequest, body["kode"])
	require.NotContains(t, body["pesan"], "adminpnc")
}
