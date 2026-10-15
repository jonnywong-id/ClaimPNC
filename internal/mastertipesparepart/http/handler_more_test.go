package mastertipespareparthttp_test

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

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesparepart/repo/memory"
	mastertipesparepartusecase "claim-pnc/internal/mastertipesparepart/usecase"
	"claim-pnc/internal/portal"

	mastertipespareparthttp "claim-pnc/internal/mastertipesparepart/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler memanggil method handler secara langsung, untuk keadaan yang tidak dapat
// dicapai lewat middleware: portal atau identitas pemanggil yang tidak ada di konteks.
type directHandler struct {
	handler   *mastertipespareparthttp.Handler
	repo      *memory.Repo
	lastError error
	hasCaller bool
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := mastertipesparepartusecase.NewService(mastertipesparepartusecase.Options{
		RepoSelector: func(string) (mastertipesparepart.Store, error) { return repo, nil },
	})
	require.NoError(t, err)

	d := &directHandler{repo: repo, hasCaller: true}
	d.handler, err = mastertipespareparthttp.NewHandler(mastertipespareparthttp.Options{
		Service: service,
		Caller: func(context.Context) (mastertipespareparthttp.Caller, bool) {
			return mastertipespareparthttp.Caller{Login: "PETUGAS"}, d.hasCaller
		},
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.lastError = err
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return d
}

func directRequest(method, body, id string, withPortal bool) *http.Request {
	r := httptest.NewRequest(method, "/master/tipe-sparepart", strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, routeContext))
}

func (d *directHandler) methods() map[string]func(http.ResponseWriter, *http.Request) {
	h := d.handler
	return map[string]func(http.ResponseWriter, *http.Request){
		"list": h.List, "get": h.Get, "choices": h.Choices, "create": h.Create, "save": h.Save, "decide": h.Decide,
	}
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := mastertipespareparthttp.NewHandler(mastertipespareparthttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := mastertipesparepartusecase.NewService(mastertipesparepartusecase.Options{
		RepoSelector: func(string) (mastertipesparepart.Store, error) { return nil, nil },
	})
	require.NoError(t, err)
	_, err = mastertipespareparthttp.NewHandler(mastertipespareparthttp.Options{Service: service})
	require.ErrorContains(t, err, "Caller wajib diisi")

	_, err = mastertipespareparthttp.NewHandler(mastertipespareparthttp.Options{
		Service: service,
		Caller: func(context.Context) (mastertipespareparthttp.Caller, bool) {
			return mastertipespareparthttp.Caller{}, false
		},
	})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

// Tanpa portal di konteks, setiap method meneruskan portal.ErrNotStated.
func TestEveryMethodRequiresActivePortal(t *testing.T) {
	d := newDirectHandler(t)
	for name, call := range d.methods() {
		t.Run(name, func(t *testing.T) {
			d.lastError = nil
			recorder := httptest.NewRecorder()
			call(recorder, directRequest(http.MethodGet, `{}`, "1", false))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, portal.ErrNotStated)
		})
	}
}

// Kegagalan penyimpanan diteruskan apa adanya ke penulis galat bersama.
func TestEveryMethodForwardsStorageFailure(t *testing.T) {
	boom := errors.New("basis data mati")
	bodies := map[string]string{
		"create": `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"2"}`,
		"save":   `{"nama_tipe_sparepart":"FUEL FILTER","id_kategori_sparepart":"2"}`,
		"decide": `{"id_tipe_sparepart":["4"],"status":"1"}`,
	}
	for name := range newDirectHandler(t).methods() {
		t.Run(name, func(t *testing.T) {
			d := newDirectHandler(t)
			d.repo.SetError(boom)
			recorder := httptest.NewRecorder()
			d.methods()[name](recorder, directRequest(http.MethodPost, bodies[name], "1", true))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, boom)
		})
	}
}

// Penambahan, penyimpanan, dan keputusan tanpa identitas pemanggil gagal keras.
func TestWritesRequireCaller(t *testing.T) {
	d := newDirectHandler(t)
	d.hasCaller = false
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"create": d.handler.Create, "save": d.handler.Save, "decide": d.handler.Decide,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			call(recorder, directRequest(http.MethodPost, `{}`, "1", true))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorContains(t, d.lastError, "identitas pemanggil tidak ada di konteks")
		})
	}
}

// ID kosong pada Get dan Save menjawab 404.
func TestEmptyIDAnswersNotFound(t *testing.T) {
	d := newDirectHandler(t)
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"get": d.handler.Get, "save": d.handler.Save,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			call(recorder, directRequest(http.MethodPut, `{}`, "", true))
			require.Equal(t, http.StatusNotFound, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"tidak_ditemukan"`)
		})
	}
}

// Badan cacat dan badan dua dokumen pada penyimpanan dan keputusan ditolak 400.
func TestMalformedAndTrailingBodiesRejected(t *testing.T) {
	d := newDirectHandler(t)
	for _, call := range []func(http.ResponseWriter, *http.Request){d.handler.Save, d.handler.Decide} {
		for _, body := range []string{`{`, `{}{}`} {
			recorder := httptest.NewRecorder()
			call(recorder, directRequest(http.MethodPost, body, "1", true))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"permintaan_cacat"`)
		}
	}
}
