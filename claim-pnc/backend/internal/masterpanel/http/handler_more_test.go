package masterpanelhttp_test

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

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/repo/memory"
	"claim-pnc/internal/masterpanel/usecase"
	"claim-pnc/internal/portal"

	masterpanelhttp "claim-pnc/internal/masterpanel/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler memanggil handler langsung, tanpa router, supaya jalur yang dijaga
// middleware ikut teruji.
type directHandler struct {
	handler *masterpanelhttp.Handler
	store   *memory.Repo
	errors  []error
	known   bool
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()

	d := &directHandler{store: memory.NewSampleRepo(), known: true}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return d.store, nil },
	})
	require.NoError(t, err)

	d.handler, err = masterpanelhttp.NewHandler(masterpanelhttp.Options{
		Service: service,
		Caller: func(context.Context) (masterpanelhttp.Caller, bool) {
			return masterpanelhttp.Caller{Login: "penguji"}, d.known
		},
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

func request(method, body, id string) *http.Request {
	r := httptest.NewRequest(method, "/", strings.NewReader(body))
	ctx := portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: "ASM"})
	if id != "" {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", id)
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeContext)
	}
	return r.WithContext(ctx)
}

func codeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"kode"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Code
}

func TestNewHandlerRejectsAnIncompleteAssembly(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpanel.Store, error) { return nil, nil },
	})
	require.NoError(t, err)
	caller := func(context.Context) (masterpanelhttp.Caller, bool) {
		return masterpanelhttp.Caller{}, true
	}

	_, err = masterpanelhttp.NewHandler(masterpanelhttp.Options{Caller: caller})
	require.EqualError(t, err, "masterpanel/http: Service wajib diisi")

	_, err = masterpanelhttp.NewHandler(masterpanelhttp.Options{Service: service})
	require.EqualError(t, err, "masterpanel/http: Caller wajib diisi")

	_, err = masterpanelhttp.NewHandler(masterpanelhttp.Options{Service: service, Caller: caller})
	require.EqualError(t, err, "masterpanel/http: WriteResponse dan WriteError wajib diisi")
}

// Setiap handler yang dipanggil di luar middleware portal menolak dengan galat portal.
func TestEveryHandlerWithoutAPortalIsRejected(t *testing.T) {
	d := newDirectHandler(t)

	for _, call := range []http.HandlerFunc{
		d.handler.List, d.handler.Get, d.handler.Create, d.handler.Save, d.handler.Decide,
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

// Tanpa identitas pemanggil, jalur tulis menolak alih-alih menulis tanpa jejak.
func TestWritesWithoutACallerFailLoudly(t *testing.T) {
	d := newDirectHandler(t)
	d.known = false

	for _, call := range []http.HandlerFunc{d.handler.Create, d.handler.Save, d.handler.Decide} {
		recorder := httptest.NewRecorder()
		call(recorder, request(http.MethodPost, "{}", "01000001"))
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}
	require.Len(t, d.errors, 3)
	for _, err := range d.errors {
		require.EqualError(t, err, "masterpanel/http: identitas pemanggil tidak ada di konteks")
	}
}

// Tanpa parameter id, Get dan Save menjawab tidak ditemukan.
func TestGetAndSaveWithoutAnIDAreNotFound(t *testing.T) {
	d := newDirectHandler(t)

	for _, call := range []http.HandlerFunc{d.handler.Get, d.handler.Save} {
		recorder := httptest.NewRecorder()
		call(recorder, request(http.MethodPut, "{}", ""))
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, masterpanelhttp.CodeNotFound, codeOf(t, recorder))
	}
}

// Badan yang memuat dua dokumen JSON ditolak; badan penyimpanan dan keputusan yang cacat
// pun ditolak sebelum layanan dipanggil.
func TestMalformedBodiesAreRejected(t *testing.T) {
	d := newDirectHandler(t)

	cases := []struct {
		call http.HandlerFunc
		body string
	}{
		{d.handler.Create, `{"nama_panel":"A"} {"nama_panel":"B"}`},
		{d.handler.Save, `{`},
		{d.handler.Decide, `{"tidak_dikenal":1}`},
	}
	for _, c := range cases {
		recorder := httptest.NewRecorder()
		c.call(recorder, request(http.MethodPost, c.body, "01000001"))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, masterpanelhttp.CodeMalformedRequest, codeOf(t, recorder))
	}
}

// Penyimpanan atas baris yang hilang dijawab 404.
func TestSaveOnAMissingRowIsNotFound(t *testing.T) {
	d := newDirectHandler(t)

	body := `{"nama_panel":"Spion","status_repair":"1","status_edit_quantity":"1",` +
		`"status_premium_repair":"0","status_pecah":"0","status_sticker":"1",` +
		`"status_sisi":"-","status_rusak_parah":"0","status_aktif":"1","exclusion_c":"0",` +
		`"lokasi":[{"lokasi_panel":"DEPAN","sisi_panel":"-"}]}`
	recorder := httptest.NewRecorder()
	d.handler.Save(recorder, request(http.MethodPut, body, "99999999"))
	require.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
}

// Kegagalan penyimpanan tidak dikenali modul ini dan diserahkan ke penulis bersama.
func TestStorageFailuresGoToTheSharedWriter(t *testing.T) {
	d := newDirectHandler(t)
	failure := errors.New("oracle mati")
	d.store.SetError(failure)

	recorder := httptest.NewRecorder()
	d.handler.List(recorder, request(http.MethodGet, "", ""))
	require.Equal(t, http.StatusTeapot, recorder.Code)

	recorder = httptest.NewRecorder()
	d.handler.Decide(recorder, request(http.MethodPost,
		`{"id_panel":["01000002"],"status":"1"}`, ""))
	require.Equal(t, http.StatusTeapot, recorder.Code)

	require.Len(t, d.errors, 2)
	for _, err := range d.errors {
		require.ErrorIs(t, err, failure)
	}
}
