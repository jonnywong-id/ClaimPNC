package riwayatklaimhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/repo/memory"
	"claim-pnc/internal/riwayatklaim/usecase"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

func writeTestJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC) }

type brokenClaims struct{}

func (brokenClaims) Search(context.Context, riwayatklaim.Criteria, riwayatklaim.Pagination) (riwayatklaim.Page, error) {
	return riwayatklaim.Page{}, errors.New("rahasia basis data")
}

type fixture struct {
	router chi.Router
	gate   *memory.ProtectionRepo
	logs   *bytes.Buffer
}

func newFixture(t *testing.T, login string, claims riwayatklaim.Repo, withFallback bool) *fixture {
	t.Helper()
	if claims == nil {
		claims = memory.NewRepo(memory.SampleClaims()...)
	}
	gate := memory.NewProtectionRepo(memory.SampleProtections()...)

	svc, err := usecase.NewService(usecase.Options{
		RepoSelector:       func(string) (riwayatklaim.Repo, error) { return claims, nil },
		ProtectionSelector: func(string) (riwayatklaim.ProtectionRepo, error) { return gate, nil },
		Clock:              fixedClock{},
	})
	require.NoError(t, err)

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeTestJSON(w, r, http.StatusTeapot, ErrorResponse{Code: "cadangan"})
		}, writeTestJSON)
	var fallback ErrorWriter
	if withFallback {
		fallback = ErrorWriter(portalError)
	}

	h := NewHandler(Options{
		Service: svc,
		GetCaller: func(context.Context) (Caller, bool) {
			return Caller{Login: login}, login != ""
		},
		Logger:              logger,
		WriteJSON:           writeTestJSON,
		FallbackErrorWriter: fallback,
	})
	router := chi.NewRouter()
	Mount(router, h, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		Logger:       logger,
		WriteError:   portalError,
	})
	return &fixture{router: router, gate: gate, logs: logs}
}

func (f *fixture) do(t *testing.T, method, target, alias string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if alias != "" {
		req.Header.Set(portalhttp.HeaderPortal, alias)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec.Code, body
}

func TestOpenGrantsQuotaAndListsSearchTypes(t *testing.T) {
	f := newFixture(t, "adminpnc", nil, true)

	status, body := f.do(t, http.MethodPost, "/riwayat-klaim/buka", "ASM")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "ASM", body["portal"])
	access := body["proteksi"].(map[string]any)
	require.EqualValues(t, 50, access["jatah_total"])
	require.EqualValues(t, 1, access["jatah_terpakai"])
	require.EqualValues(t, 49, access["jatah_sisa"])
	types := body["tipe_pencarian"].([]any)
	require.Len(t, types, len(riwayatklaim.SearchTypes()))
	require.Len(t, f.gate.Usage(), 1)
}

func TestOpenRejectsUnregisteredAndExhausted(t *testing.T) {
	status, body := newFixture(t, "profilbolong", nil, true).
		do(t, http.MethodPost, "/riwayat-klaim/buka", "ASM")
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, CodeNotRegistered, body["kode"])

	status, body = newFixture(t, "penggunanonaktif", nil, true).
		do(t, http.MethodPost, "/riwayat-klaim/buka", "ASM")
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, CodeQuotaExhausted, body["kode"])
}

func TestUnknownCallerAndMissingPortal(t *testing.T) {
	f := newFixture(t, "", nil, true)
	for _, route := range [][2]string{
		{http.MethodPost, "/riwayat-klaim/buka"}, {http.MethodGet, "/riwayat-klaim"},
	} {
		status, body := f.do(t, route[0], route[1], "ASM")
		require.Equal(t, http.StatusConflict, status, route[1])
		require.Equal(t, CodeCallerUnknown, body["kode"], route[1])
	}

	status, body := f.do(t, http.MethodGet, "/riwayat-klaim", "")
	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, portalhttp.CodeNotStated, body["kode"])
}

func TestSearchReturnsClaimsAndPagination(t *testing.T) {
	f := newFixture(t, "adminpnc", nil, true)

	status, body := f.do(t, http.MethodGet,
		"/riwayat-klaim?tipe=13&nilai=bl-contoh-77&halaman=x&ukuran=-1", "ASM")
	require.Equal(t, http.StatusOK, status)
	claims := body["klaim"].([]any)
	require.Len(t, claims, 1)
	first := claims[0].(map[string]any)
	require.Equal(t, "PNC-9003", first["nomor_klaim"])
	require.Equal(t, "2025-11-04", first["tanggal_kejadian"])
	require.Equal(t, "BL-CONTOH-77", first["nomor_balai_lelang"])
	require.Nil(t, first["tanggal_lahir"])
	page := body["halaman"].(map[string]any)
	require.EqualValues(t, 1, page["halaman"])
	require.EqualValues(t, 1, page["total"])
	require.EqualValues(t, 1, page["total_halaman"])
}

func TestSearchByLossDate(t *testing.T) {
	f := newFixture(t, "adminpnc", nil, true)

	status, body := f.do(t, http.MethodGet,
		"/riwayat-klaim?tipe=6&tanggal_pencarian=2026-03-12", "ASM")
	require.Equal(t, http.StatusOK, status)
	require.Len(t, body["klaim"], 2)
}

func TestSearchRejectsUnreadableDates(t *testing.T) {
	f := newFixture(t, "adminpnc", nil, true)

	status, body := f.do(t, http.MethodGet, "/riwayat-klaim?tipe=6&tanggal_pencarian=12-03-2026", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, status)
	detail := body["detail"].([]any)[0].(map[string]any)
	require.Equal(t, riwayatklaim.FieldSearchDate, detail["field"])

	status, body = f.do(t, http.MethodGet, "/riwayat-klaim?tipe=9&tanggal_lahir=bukan", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, status)
	detail = body["detail"].([]any)[0].(map[string]any)
	require.Equal(t, riwayatklaim.FieldBirthDate, detail["field"])
}

func TestSearchValidationFailure(t *testing.T) {
	f := newFixture(t, "adminpnc", nil, true)

	status, body := f.do(t, http.MethodGet, "/riwayat-klaim?tipe=1", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.Equal(t, CodeValidationFail, body["kode"])
}

func TestSearchRepoFailureUsesFallbackOrInternalError(t *testing.T) {
	status, body := newFixture(t, "adminpnc", brokenClaims{}, true).
		do(t, http.MethodGet, "/riwayat-klaim?tipe=1&nilai=POL", "ASM")
	require.Equal(t, http.StatusTeapot, status)
	require.Equal(t, "cadangan", body["kode"])

	f := newFixture(t, "adminpnc", brokenClaims{}, false)
	status, body = f.do(t, http.MethodGet, "/riwayat-klaim?tipe=1&nilai=POL", "ASM")
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, CodeInternalError, body["kode"])
	require.Contains(t, f.logs.String(), "rahasia basis data")
}

func TestHandlersWithoutPortalContext(t *testing.T) {
	gate := memory.NewProtectionRepo()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector:       func(string) (riwayatklaim.Repo, error) { return memory.NewRepo(), nil },
		ProtectionSelector: func(string) (riwayatklaim.ProtectionRepo, error) { return gate, nil },
		Clock:              fixedClock{},
	})
	require.NoError(t, err)
	h := NewHandler(Options{
		Service: svc, WriteJSON: writeTestJSON,
		Logger: slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	})

	for name, handle := range map[string]http.HandlerFunc{"open": h.Open, "search": h.Search} {
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code, name)
	}

	_, known := h.readCaller(httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, known)
}

func TestDateHelpers(t *testing.T) {
	parsed, ok := parseDate("")
	require.True(t, ok)
	require.Nil(t, parsed)

	parsed, ok = parseDate("2026-01-02")
	require.True(t, ok)
	require.Equal(t, "2026-01-02", *toDateString(parsed))

	_, ok = parseDate("x")
	require.False(t, ok)
	require.Nil(t, toDateString(nil))

	require.Equal(t, 3, positiveNumber("3"))
	require.Equal(t, 0, positiveNumber("-3"))
}
