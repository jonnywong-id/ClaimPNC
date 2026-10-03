package daftarobjekdokumenhttp_test

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

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/repo/memory"
	daftarobjekdokumenusecase "claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/portal"

	daftarobjekdokumenhttp "claim-pnc/internal/daftarobjekdokumen/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler merakit handler tanpa router, supaya setiap method dapat dipanggil langsung
// dengan konteks yang disusun sendiri — termasuk keadaan yang tidak dapat dicapai lewat
// middleware, seperti portal yang tidak ada di konteks.
type directHandler struct {
	handler   *daftarobjekdokumenhttp.Handler
	repo      *memory.Repo
	lastError error
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()
	repo := memory.NewRepo(memory.SampleList()...)
	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	service, err := daftarobjekdokumenusecase.NewService(daftarobjekdokumenusecase.Options{
		RepoSelector:     func(string) (daftarobjekdokumen.Repo, error) { return repo, nil },
		BusinessSelector: func(string) (daftarobjekdokumen.BusinessRepo, error) { return business, nil },
	})
	require.NoError(t, err)

	d := &directHandler{repo: repo}
	d.handler, err = daftarobjekdokumenhttp.NewHandler(daftarobjekdokumenhttp.Options{
		Service: service,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		// Penulis galat bersama dicatat galatnya supaya uji dapat memastikan galat apa yang
		// diteruskan ke sana.
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.lastError = err
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return d
}

// request menyusun permintaan dengan portal ASM (bila withPortal) dan parameter id chi.
func request(method, body, id string, withPortal bool) *http.Request {
	r := httptest.NewRequest(method, "/master/objek-dokumen", strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", id)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, routeContext)
	return r.WithContext(ctx)
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := daftarobjekdokumenhttp.NewHandler(daftarobjekdokumenhttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := daftarobjekdokumenusecase.NewService(daftarobjekdokumenusecase.Options{
		RepoSelector:     func(string) (daftarobjekdokumen.Repo, error) { return nil, nil },
		BusinessSelector: func(string) (daftarobjekdokumen.BusinessRepo, error) { return nil, nil },
	})
	require.NoError(t, err)
	_, err = daftarobjekdokumenhttp.NewHandler(daftarobjekdokumenhttp.Options{Service: service})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

// Tanpa portal di konteks, keempat method meneruskan portal.ErrNotStated ke penulis bersama.
func TestEveryMethodRequiresActivePortal(t *testing.T) {
	d := newDirectHandler(t)
	calls := map[string]func(http.ResponseWriter, *http.Request){
		"list": d.handler.List, "get": d.handler.Get, "create": d.handler.Create, "update": d.handler.Update,
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			d.lastError = nil
			recorder := httptest.NewRecorder()
			call(recorder, request(http.MethodGet, `{}`, "10001", false))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, portal.ErrNotStated)
		})
	}
}

// ID kosong pada Get dan Update menjawab 404 tanpa menyentuh repo.
func TestEmptyIDAnswersNotFound(t *testing.T) {
	d := newDirectHandler(t)
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"get": d.handler.Get, "update": d.handler.Update,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			call(recorder, request(http.MethodPut, `{}`, "", true))
			require.Equal(t, http.StatusNotFound, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"tidak_ditemukan"`)
		})
	}
}

// Galat penyimpanan yang tidak dikenali modul diteruskan ke penulis bersama apa adanya.
func TestRepositoryFailureIsForwarded(t *testing.T) {
	boom := errors.New("basis data mati")
	for name, run := range map[string]func(d *directHandler, w http.ResponseWriter){
		"list": func(d *directHandler, w http.ResponseWriter) {
			d.handler.List(w, request(http.MethodGet, "", "", true))
		},
		"get": func(d *directHandler, w http.ResponseWriter) {
			d.handler.Get(w, request(http.MethodGet, "", "10001", true))
		},
		"create": func(d *directHandler, w http.ResponseWriter) {
			d.handler.Create(w, request(http.MethodPost, `{}`, "", true))
		},
		"update": func(d *directHandler, w http.ResponseWriter) {
			d.handler.Update(w, request(http.MethodPut, `{}`, "10001", true))
		},
	} {
		t.Run(name, func(t *testing.T) {
			d := newDirectHandler(t)
			d.repo.SetError(boom)
			recorder := httptest.NewRecorder()
			run(d, recorder)
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, boom)
		})
	}
}

// Penyuntingan atas baris yang tidak ada menjawab 404, dan isian tidak sah menjawab 422.
func TestUpdateNotFoundAndValidation(t *testing.T) {
	d := newDirectHandler(t)
	recorder := httptest.NewRecorder()
	d.handler.Update(recorder, request(http.MethodPut, `{"objek_dokumen":"x"}`, "99999", true))
	require.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	long := strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1)
	d.handler.Update(recorder, request(http.MethodPut, `{"objek_dokumen":"`+long+`"}`, "10001", true))
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"validasi_gagal"`)
}

// Badan cacat pada penyuntingan, dan badan berisi dua dokumen JSON, sama-sama ditolak 400.
func TestUpdateRejectsMalformedAndTrailingBody(t *testing.T) {
	d := newDirectHandler(t)
	for name, body := range map[string]string{
		"malformed": `{bukan`,
		"trailing":  `{"objek_dokumen":"a"}{"objek_dokumen":"b"}`,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			d.handler.Update(recorder, request(http.MethodPut, body, "10001", true))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"permintaan_cacat"`)
		})
	}
	// Tidak satu pun yang tersimpan.
	got, err := d.repo.Get(t.Context(), "10001")
	require.NoError(t, err)
	require.Equal(t, "KTP Tertanggung", got.Description)
}
