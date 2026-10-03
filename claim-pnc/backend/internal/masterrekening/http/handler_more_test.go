package masterrekeninghttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/portal"

	masterrekeninghttp "claim-pnc/internal/masterrekening/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// Berkas ini melengkapi routes_test.go: alur penuh tiap rute pada satu entitas, dan
// pemetaan galat layanan menjadi status HTTP.

func TestFullFlowListGetUpdateDecideAndBanks(t *testing.T) {
	server := testServer(t)

	recorder, body := call(t, server, http.MethodPost, routeAccount, "ASM", contohPengajuan)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "0", body["status"])

	recorder, body = call(t, server, http.MethodGet, routeAccount+"/014/1234567890", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "BENGKEL CONTOH SEJAHTERA", body["nama_pemilik"])

	recorder, body = call(t, server, http.MethodGet, routeAccount+"/014/9999", "ASM", "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, masterrekeninghttp.CodeNotFound, body["kode"])

	updated := strings.Replace(contohPengajuan, "BENGKEL CONTOH SEJAHTERA", "BENGKEL BARU", 1)
	recorder, body = call(t, server, http.MethodPut, routeAccount+"/014/1234567890", "ASM", updated)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "BENGKEL BARU", body["nama_pemilik"])

	// Isian wajib yang dikosongkan ditolak dengan rincian per kolom.
	empty := strings.Replace(contohPengajuan, `"nik":"3171000000000000"`, `"nik":""`, 1)
	recorder, body = call(t, server, http.MethodPut, routeAccount+"/014/1234567890", "ASM", empty)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, masterrekeninghttp.CodeInvalidInput, body["kode"])
	require.NotEmpty(t, body["detail"])
	require.Contains(t, body["field"], "nik")

	recorder, body = call(t, server, http.MethodPost, routeAccount+"/014/1234567890/keputusan",
		"ASM", `{"status":"3"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, masterrekeninghttp.CodeMalformedRequest, body["kode"])

	recorder, body = call(t, server, http.MethodPost, routeAccount+"/014/1234567890/keputusan",
		"ASM", `{"status":"1","catatan":"Disetujui atasan"}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "1", body["status"])
	require.Equal(t, "3171999", body["komite_approval"], "pemutus dicatat sebagai komite")

	recorder, body = call(t, server, http.MethodPost, routeAccount+"/014/1234567890/keputusan",
		"ASM", `{"status":"2"}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, masterrekeninghttp.CodeAlreadyDecided, body["kode"])

	recorder, body = call(t, server, http.MethodGet,
		routeAccount+"?status=1&nomor_rekening=1234&batas=999&lewati=-1&komite_saya=1", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, float64(1), body["jumlah"])
	require.Equal(t, float64(200), body["batas"], "batas dipangkas ke maksimum klien")
	require.Equal(t, float64(0), body["lewati"], "nilai negatif jatuh ke bawaan")

	recorder, body = call(t, server, http.MethodGet, routeAccount+"?status=7", "ASM", "")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, masterrekeninghttp.CodeMalformedRequest, body["kode"])

	recorder, body = call(t, server, http.MethodGet, routeAccount+"/bank", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEmpty(t, body["bank"])
}

func TestMalformedBodiesAreRejectedOnEveryWriteRoute(t *testing.T) {
	server := testServer(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, routeAccount},
		{http.MethodPut, routeAccount + "/014/1"},
		{http.MethodPost, routeAccount + "/014/1/keputusan"},
	} {
		recorder, body := call(t, server, tc.method, tc.path, "ASM", `{"tidak_dikenal":1}`)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, masterrekeninghttp.CodeMalformedRequest, body["kode"])
	}
}

// stubService adalah layanan tiruan yang selalu gagal dengan galat yang ditentukan.
type stubService struct{ err error }

func (s stubService) List(context.Context, masterrekening.Filter) ([]masterrekening.Account, int, error) {
	return nil, 0, s.err
}
func (s stubService) Get(context.Context, masterrekening.Key) (masterrekening.Account, error) {
	return masterrekening.Account{}, s.err
}
func (s stubService) ListBanks(context.Context) ([]masterrekening.Bank, error) { return nil, s.err }
func (s stubService) Submit(context.Context, usecase.Submission, usecase.Submitter) (masterrekening.Account, error) {
	return masterrekening.Account{}, s.err
}
func (s stubService) Update(context.Context, masterrekening.Key, usecase.Submission, usecase.Submitter) (masterrekening.Account, error) {
	return masterrekening.Account{}, s.err
}
func (s stubService) Decide(context.Context, masterrekening.Key, usecase.Decision, usecase.Committee, *slog.Logger) (masterrekening.Account, error) {
	return masterrekening.Account{}, s.err
}

// stubRouter memasang handler tanpa middleware portal; portal disisipkan langsung ke
// context supaya kegagalan layanan dan ketiadaan pemanggil dapat diuji.
func stubRouter(t *testing.T, service masterrekeninghttp.Service, selectorErr error,
	withCaller bool, logs *bytes.Buffer,
) http.Handler {
	t.Helper()
	var caller func(context.Context) (masterrekeninghttp.Caller, bool)
	if withCaller {
		caller = func(context.Context) (masterrekeninghttp.Caller, bool) {
			return masterrekeninghttp.Caller{Identity: "K1"}, true
		}
	}
	handler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		ServiceSelector: func(string) (masterrekeninghttp.Service, error) {
			return service, selectorErr
		},
		Caller: caller,
		Logger: slog.New(slog.NewJSONHandler(logs, nil)),
	})

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: "ASM"})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Get("/r", handler.List)
	router.Post("/r", handler.Submit)
	router.Get("/r/bank", handler.ListBanks)
	router.Get("/r/{kodeBank}/{nomorRekening}", handler.Get)
	router.Put("/r/{kodeBank}/{nomorRekening}", handler.Update)
	router.Post("/r/{kodeBank}/{nomorRekening}/keputusan", handler.Decide)
	return router
}

func hit(t *testing.T, router http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
	content := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &content)
	return recorder, content
}

var everyRoute = []struct{ method, path, body string }{
	{http.MethodGet, "/r", ""},
	{http.MethodPost, "/r", "{}"},
	{http.MethodGet, "/r/bank", ""},
	{http.MethodGet, "/r/014/1", ""},
	{http.MethodPut, "/r/014/1", "{}"},
	{http.MethodPost, "/r/014/1/keputusan", "{}"},
}

func TestServiceErrorsAreMappedOnEveryRoute(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{masterrekening.ErrAlreadyExists, http.StatusConflict, masterrekeninghttp.CodeAlreadyExists},
		{masterrekening.ErrNotFound, http.StatusNotFound, masterrekeninghttp.CodeNotFound},
		{errors.New("putus"), http.StatusInternalServerError, masterrekeninghttp.CodeInternalError},
	}
	for _, tc := range cases {
		logs := &bytes.Buffer{}
		router := stubRouter(t, stubService{err: tc.err}, nil, true, logs)
		for _, route := range everyRoute {
			recorder, body := hit(t, router, route.method, route.path, route.body)
			require.Equal(t, tc.status, recorder.Code, route.path)
			require.Equal(t, tc.code, body["kode"], route.path)
		}
		if tc.status == http.StatusInternalServerError {
			require.Contains(t, logs.String(), "permintaan master rekening gagal")
		}
	}
}

func TestASelectorFailureIsWritten(t *testing.T) {
	router := stubRouter(t, nil, errors.New("tidak siap"), true, &bytes.Buffer{})
	for _, route := range everyRoute {
		recorder, body := hit(t, router, route.method, route.path, route.body)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		require.Equal(t, masterrekeninghttp.CodeInternalError, body["kode"])
	}
}

func TestWritesWithoutACallerAreRefused(t *testing.T) {
	router := stubRouter(t, stubService{}, nil, false, &bytes.Buffer{})
	for _, route := range []struct{ method, path, body string }{
		{http.MethodGet, "/r?komite_saya=1", ""},
		{http.MethodPost, "/r", "{}"},
		{http.MethodPut, "/r/014/1", "{}"},
		{http.MethodPost, "/r/014/1/keputusan", "{}"},
	} {
		recorder, _ := hit(t, router, route.method, route.path, route.body)
		require.Equal(t, http.StatusInternalServerError, recorder.Code, route.path)
	}
}

func TestHandlersWithoutAPortalAreRejected(t *testing.T) {
	handler := masterrekeninghttp.NewHandler(masterrekeninghttp.Options{
		Logger: slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	})
	recorder := httptest.NewRecorder()
	handler.Get(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	// Penulis galat bawaan modul tidak mengenal galat portal, sehingga ia 500.
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

// brokenWriter menolak penulisan badan jawaban.
type brokenWriter struct{ header http.Header }

func (b *brokenWriter) Header() http.Header       { return b.header }
func (b *brokenWriter) WriteHeader(int)           {}
func (b *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("putus") }

func TestWriteJSONLogsAFailedWriteAndSkipsANilBody(t *testing.T) {
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	writer := &brokenWriter{header: http.Header{}}
	masterrekeninghttp.WriteJSON(writer, httptest.NewRequest(http.MethodGet, "/x", nil),
		http.StatusOK, map[string]string{"a": "b"}, logger)
	require.Contains(t, logs.String(), "gagal menulis respons master rekening")
	require.Equal(t, "no-store", writer.header.Get("Cache-Control"))

	recorder := httptest.NewRecorder()
	masterrekeninghttp.WriteJSON(recorder, httptest.NewRequest(http.MethodGet, "/x", nil),
		http.StatusNoContent, nil, logger)
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Body.String())
}
