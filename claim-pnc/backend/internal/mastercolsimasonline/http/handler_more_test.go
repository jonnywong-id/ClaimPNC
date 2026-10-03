package mastercolsimasonlinehttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/mastercolsimasonline/repo/memory"
	mastercolusecase "claim-pnc/internal/mastercolsimasonline/usecase"
	"claim-pnc/internal/portal"

	mastercolhttp "claim-pnc/internal/mastercolsimasonline/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler merakit handler yang dipanggil langsung, tanpa router, supaya jalur yang
// tidak dapat dicapai lewat middleware ikut teruji.
type directHandler struct {
	handler  *mastercolhttp.Handler
	repo     *memory.Repo
	business *memory.BusinessRepo
	errors   []error
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()

	d := &directHandler{
		repo:     memory.NewRepo(memory.SampleList()...),
		business: memory.NewBusinessRepo(memory.SampleBusinessList()...),
	}
	service, err := mastercolusecase.NewService(mastercolusecase.Options{
		RepoSelector: func(string) (mastercolsimasonline.Repo, error) { return d.repo, nil },
		BusinessSelector: func(string) (mastercolsimasonline.BusinessRepo, error) {
			return d.business, nil
		},
	})
	require.NoError(t, err)

	d.handler, err = mastercolhttp.NewHandler(mastercolhttp.Options{
		Service: service,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.errors = append(d.errors, err)
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return d
}

func withPortal(r *http.Request) *http.Request {
	return r.WithContext(portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: "ASM"}))
}

// withID memasang parameter jalur {id} seperti yang dilakukan router chi.
func withID(r *http.Request, id string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext))
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestNewHandlerRejectsAnIncompleteAssembly(t *testing.T) {
	writeJSON := func(http.ResponseWriter, *http.Request, int, any) {}
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err := mastercolhttp.NewHandler(mastercolhttp.Options{
		WriteResponse: writeJSON, WriteError: writeError})
	require.EqualError(t, err, "mastercolsimasonline/http: Service wajib diisi")

	service, err := mastercolusecase.NewService(mastercolusecase.Options{
		RepoSelector: func(string) (mastercolsimasonline.Repo, error) { return nil, nil },
		BusinessSelector: func(string) (mastercolsimasonline.BusinessRepo, error) {
			return nil, nil
		},
	})
	require.NoError(t, err)
	_, err = mastercolhttp.NewHandler(mastercolhttp.Options{Service: service})
	require.EqualError(t, err,
		"mastercolsimasonline/http: WriteResponse dan WriteError wajib diisi")
}

// Setiap handler yang dipanggil di luar middleware portal menolak dengan galat portal.
func TestEveryHandlerOutsideThePortalMiddlewareIsRejected(t *testing.T) {
	d := newDirectHandler(t)

	for _, call := range []http.HandlerFunc{
		d.handler.List, d.handler.Get, d.handler.Create, d.handler.Update, d.handler.Business,
	} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}

	require.Len(t, d.errors, 5)
	for _, err := range d.errors {
		require.ErrorIs(t, err, portal.ErrNotStated)
	}
}

// Tanpa parameter id di jalur, Get dan Update menjawab tidak ditemukan.
func TestGetAndUpdateWithoutAnIDAreNotFound(t *testing.T) {
	d := newDirectHandler(t)

	for _, call := range []http.HandlerFunc{d.handler.Get, d.handler.Update} {
		recorder := httptest.NewRecorder()
		call(recorder, withPortal(httptest.NewRequest(http.MethodPut, "/", nil)))
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, mastercolhttp.CodeNotFound, decode(t, recorder)["kode"])
	}
}

// Kegagalan penyimpanan tidak dikenali modul ini dan diserahkan ke penulis bersama.
func TestStorageFailuresAreHandedToTheSharedErrorWriter(t *testing.T) {
	d := newDirectHandler(t)
	failure := errors.New("oracle mati")
	d.repo.SetError(failure)
	d.business.SetError(failure)

	for _, call := range []http.HandlerFunc{d.handler.List, d.handler.Business} {
		recorder := httptest.NewRecorder()
		call(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/", nil)))
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}

	require.Len(t, d.errors, 2)
	for _, err := range d.errors {
		require.ErrorIs(t, err, failure)
	}
}

// Badan penyuntingan yang tidak terbaca ditolak 400 sebelum layanan dipanggil.
func TestUpdateWithAMalformedBodyIsRejected(t *testing.T) {
	d := newDirectHandler(t)

	request := withPortal(httptest.NewRequest(http.MethodPut, "/", strings.NewReader("{")))
	request = withID(request, "1001")
	recorder := httptest.NewRecorder()
	d.handler.Update(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, mastercolhttp.CodeMalformedRequest, decode(t, recorder)["kode"])

	row, err := d.repo.Get(request.Context(), "1001")
	require.NoError(t, err)
	require.Equal(t, "KEBAKARAN", row.Description)
}
