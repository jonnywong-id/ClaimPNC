package menuhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/menu/repo/memory"
	menuusecase "claim-pnc/internal/menu/usecase"

	menuhttp "claim-pnc/internal/menu/http"
)

func plainWriter(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func newHandler(t *testing.T, repo *memory.Repo, caller func(context.Context) (menuhttp.Caller, bool),
	writeError menuhttp.ErrorWriter) http.Handler {
	t.Helper()
	service, err := menuusecase.NewService(menuusecase.Options{Repo: repo})
	require.NoError(t, err)
	handler, err := menuhttp.NewHandler(menuhttp.Options{Service: service, Caller: caller,
		WriteResponse: plainWriter, WriteError: writeError, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	require.NoError(t, err)
	router := chi.NewRouter()
	menuhttp.Mount(router, handler)
	return router
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	service, err := menuusecase.NewService(menuusecase.Options{Repo: memory.NewSampleRepo()})
	require.NoError(t, err)
	caller := func(context.Context) (menuhttp.Caller, bool) { return menuhttp.Caller{}, true }
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err = menuhttp.NewHandler(menuhttp.Options{Caller: caller, WriteResponse: plainWriter, WriteError: writeError, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	require.ErrorContains(t, err, "Service wajib diisi")
	_, err = menuhttp.NewHandler(menuhttp.Options{Service: service, WriteResponse: plainWriter, WriteError: writeError, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	require.ErrorContains(t, err, "Caller wajib diisi")
	_, err = menuhttp.NewHandler(menuhttp.Options{Service: service, Caller: caller, WriteResponse: plainWriter})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestListWithoutCallerContextGoesToErrorWriter(t *testing.T) {
	var captured error
	router := newHandler(t, memory.NewSampleRepo(),
		func(context.Context) (menuhttp.Caller, bool) { return menuhttp.Caller{}, false },
		func(w http.ResponseWriter, _ *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusUnauthorized)
		})
	record := httptest.NewRecorder()
	router.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/menu", nil))
	require.Equal(t, http.StatusUnauthorized, record.Code)
	require.ErrorContains(t, captured, "konteks pemanggil tidak ada")
}

func TestListAnswersMissingAppWithItsOwnCode(t *testing.T) {
	repo := memory.NewSampleRepo()
	repo.SetError(menu.ErrAppNotFound)
	router := newHandler(t, repo,
		func(context.Context) (menuhttp.Caller, bool) { return menuhttp.Caller{Login: "admin"}, true },
		func(http.ResponseWriter, *http.Request, error) { t.Fatal("galat aplikasi tidak boleh ke penulis umum") })
	record := httptest.NewRecorder()
	router.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/menu", nil))
	require.Equal(t, http.StatusInternalServerError, record.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
	require.Equal(t, menuhttp.KodeMenuTidakTerkonfigurasi, body["kode"])
}

func TestListPassesOtherFailuresToErrorWriter(t *testing.T) {
	repo := memory.NewSampleRepo()
	failure := errors.New("rusak")
	repo.SetError(failure)
	var captured error
	router := newHandler(t, repo,
		func(context.Context) (menuhttp.Caller, bool) { return menuhttp.Caller{Login: "admin"}, true },
		func(w http.ResponseWriter, _ *http.Request, err error) {
			captured = err
			w.WriteHeader(http.StatusBadGateway)
		})
	record := httptest.NewRecorder()
	router.ServeHTTP(record, httptest.NewRequest(http.MethodGet, "/menu", nil))
	require.Equal(t, http.StatusBadGateway, record.Code)
	require.ErrorIs(t, captured, failure)
}
