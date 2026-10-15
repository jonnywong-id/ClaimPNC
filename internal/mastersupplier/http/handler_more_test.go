package mastersupplierhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/repo/memory"
	mastersupplierusecase "claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	mastersupplierhttp "claim-pnc/internal/mastersupplier/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler memanggil method handler secara langsung, untuk keadaan yang tidak dapat
// dicapai lewat middleware: portal atau identitas pemanggil yang tidak ada di konteks.
type directHandler struct {
	handler   *mastersupplierhttp.Handler
	repo      *memory.Repo
	lastError error
	hasCaller bool
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: func(string) (mastersupplier.Store, error) { return repo, nil },
		Clock:        clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	d := &directHandler{repo: repo, hasCaller: true}
	d.handler, err = mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: service,
		Caller: func(context.Context) (mastersupplierhttp.Caller, bool) {
			return mastersupplierhttp.Caller{Login: "PETUGAS"}, d.hasCaller
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
	r := httptest.NewRequest(method, "/master/supplier?cari=x", strings.NewReader(body))
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
		"list": h.List, "get": h.Get, "create": h.Create, "save": h.Save,
		"branches": h.Branches, "cities": h.Cities, "countries": h.Countries, "banks": h.Banks, "codes": h.Codes,
	}
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := mastersupplierhttp.NewHandler(mastersupplierhttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := mastersupplierusecase.NewService(mastersupplierusecase.Options{
		RepoSelector: func(string) (mastersupplier.Store, error) { return nil, nil },
		Clock:        clock.FixedAt(time.Now()),
	})
	require.NoError(t, err)
	_, err = mastersupplierhttp.NewHandler(mastersupplierhttp.Options{Service: service})
	require.ErrorContains(t, err, "Caller wajib diisi")

	_, err = mastersupplierhttp.NewHandler(mastersupplierhttp.Options{
		Service: service,
		Caller:  func(context.Context) (mastersupplierhttp.Caller, bool) { return mastersupplierhttp.Caller{}, false },
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
			call(recorder, directRequest(http.MethodGet, `{}`, "x", false))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, portal.ErrNotStated)
		})
	}
}

// Kegagalan penyimpanan diteruskan apa adanya ke penulis galat bersama.
func TestEveryMethodForwardsStorageFailure(t *testing.T) {
	boom := errors.New("basis data mati")
	for name := range newDirectHandler(t).methods() {
		t.Run(name, func(t *testing.T) {
			d := newDirectHandler(t)
			d.repo.SetError(boom)
			body := `{"nama":"Supplier Contoh Utama","alamat":"a","kota":"k","nama_cabang":"c","negara":"n",` +
				`"telepon":"1","contact_person":"p","status_rekanan":"1","status_supply":"1",` +
				`"term_of_payment":"1","term_of_delivery":"1","bank":"b","no_account":"1",` +
				`"jenis_supplier":"1","status_aktif":"1"}`
			request := directRequest(http.MethodPost, body, "0100000000001", true)
			if name == "cities" {
				request.URL.RawQuery = "cari=bandung"
			}
			recorder := httptest.NewRecorder()
			d.methods()[name](recorder, request)
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorIs(t, d.lastError, boom)
		})
	}
}

// Penambahan dan penyimpanan tanpa identitas pemanggil gagal keras, bukan menulis jejak kosong.
func TestCreateAndSaveRequireCaller(t *testing.T) {
	d := newDirectHandler(t)
	d.hasCaller = false
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"create": d.handler.Create, "save": d.handler.Save,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			call(recorder, directRequest(http.MethodPost, `{}`, "x", true))
			require.Equal(t, http.StatusTeapot, recorder.Code)
			require.ErrorContains(t, d.lastError, "identitas pemanggil tidak ada di konteks")
		})
	}
	require.Empty(t, d.repo.Approval())
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

// Badan yang memuat dua dokumen JSON, atau badan cacat pada penyimpanan, ditolak 400.
func TestSaveRejectsMalformedAndTrailingBody(t *testing.T) {
	d := newDirectHandler(t)
	for name, body := range map[string]string{"malformed": `{`, "trailing": `{}{}`} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			d.handler.Save(recorder, directRequest(http.MethodPut, body, "0100000000001", true))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), `"permintaan_cacat"`)
		})
	}
}
