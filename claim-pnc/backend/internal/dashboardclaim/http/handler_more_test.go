package dashboardclaimhttp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/memory"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"

	dashboardhttp "claim-pnc/internal/dashboardclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

func plainWriteJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func sampleService(t *testing.T) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) {
			return memory.NewRepo(memory.SampleOutstanding(), memory.SampleSurveys()), nil
		},
		ClosedClaim: memory.NewClosedReader(memory.SampleClosed()),
	})
	require.NoError(t, err)
	return service
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	service := sampleService(t)

	_, err := dashboardhttp.NewHandler(dashboardhttp.Options{})
	require.EqualError(t, err, "dashboardclaim/http: Service wajib diisi")

	_, err = dashboardhttp.NewHandler(dashboardhttp.Options{Service: service})
	require.EqualError(t, err, "dashboardclaim/http: WriteResponse wajib diisi")

	_, err = dashboardhttp.NewHandler(dashboardhttp.Options{Service: service, WriteResponse: plainWriteJSON})
	require.EqualError(t, err, "dashboardclaim/http: WriteError wajib diisi")
}

// Tanpa zona waktu, handler memakai WIB: 2026-09-01 20:00 UTC tampil sebagai 2 September.
func TestNewHandlerDefaultsToJakartaTime(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) {
			return memory.NewRepo([]memory.ClaimRecord{{Row: dashboardclaim.ClaimRow{
				ClaimNumber:  "PNCN.26.9999",
				RegisteredAt: mustUTC(t, "2026-09-01T20:00:00Z"),
			}}}, nil), nil
		},
		ClosedClaim: memory.NewClosedReader(nil),
	})
	require.NoError(t, err)

	handler, err := dashboardhttp.NewHandler(dashboardhttp.Options{
		Service:       service,
		WriteResponse: plainWriteJSON,
		WriteError:    dashboardhttp.WriteError(nil, plainWriteJSON, nil),
	})
	require.NoError(t, err)

	recorder := serveDirect(handler, "/api/dashboard-claim/outstanding")
	require.Equal(t, http.StatusOK, recorder.Code)
	first := decode(t, recorder)["klaim"].([]any)[0].(map[string]any)
	require.Equal(t, "2026-09-02", first["tanggal_register"])
	require.Equal(t, "", first["tanggal_kejadian"], "tanggal kejadian kosong tidak menjadi tahun satu")
}

func mustUTC(t *testing.T, raw string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, raw)
	require.NoError(t, err)
	return parsed
}

func activePortal() portal.Portal { return portal.Portal{Alias: portalUtama} }

// serveDirect memasang handler di router dengan portal aktif sudah tertanam di konteks.
func serveDirect(handler *dashboardhttp.Handler, path string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Get("/api/dashboard-claim/{tile}", handler.List)
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(), activePortal()))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// Setiap handler menolak konteks tanpa portal aktif, meski dipasang tanpa middleware.
func TestHandlersRejectMissingPortal(t *testing.T) {
	writeJSON := plainWriteJSON
	writeError := portalhttp.WithPortalError(dashboardhttp.WriteError(nil, writeJSON, nil), writeJSON)

	handler, err := dashboardhttp.NewHandler(dashboardhttp.Options{
		Service:       sampleService(t),
		Location:      wib,
		WriteResponse: writeJSON,
		WriteError:    dashboardhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	for label, handle := range map[string]http.HandlerFunc{
		"ringkasan": handler.Summary,
		"telusur":   handler.List,
		"penyaring": handler.Metadata,
	} {
		recorder := httptest.NewRecorder()
		handle(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equalf(t, http.StatusBadRequest, recorder.Code, label)
		require.Equalf(t, "portal_tidak_disebut", decode(t, recorder)["kode"], label)
	}
}

// Tanpa cadangan, galat asing dijawab 500 umum dan rinciannya hanya masuk log.
func TestUnrecognizedErrorWithoutFallbackIsLoggedNotLeaked(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) {
			return nil, errors.New("rahasia-koneksi")
		},
		ClosedClaim: memory.NewClosedReader(nil),
	})
	require.NoError(t, err)

	writeError := portalhttp.WithPortalError(dashboardhttp.WriteError(logger, plainWriteJSON, nil), plainWriteJSON)
	handler, err := dashboardhttp.NewHandler(dashboardhttp.Options{
		Service: service, Logger: logger, Location: wib,
		WriteResponse: plainWriteJSON, WriteError: dashboardhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	deps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{portalUtama} },
		Logger:       logger,
		WriteError:   writeError,
	}
	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) { dashboardhttp.Mount(api, handler, deps) })

	for _, path := range []string{"/api/dashboard-claim/ringkasan", "/api/dashboard-claim/outstanding"} {
		recorder := get(t, router, path)
		require.Equal(t, http.StatusInternalServerError, recorder.Code, path)
		require.Equal(t, "galat_internal", decode(t, recorder)["kode"], path)
		require.NotContains(t, recorder.Body.String(), "rahasia-koneksi", path)
	}
	require.Contains(t, logs.String(), "rahasia-koneksi")
}

// Isian paginasi yang salah pada telusur dijawab 422, bukan diabaikan.
func TestListInvalidPagingIs422(t *testing.T) {
	server := testServer(t)

	recorder := get(t, server, "/api/dashboard-claim/loss-adjuster?halaman=nol")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, "validasi_gagal", decode(t, recorder)["kode"])
}
