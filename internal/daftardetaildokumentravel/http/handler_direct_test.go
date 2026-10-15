package daftardetaildokumentravelhttp

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

	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/repo/memory"
	"claim-pnc/internal/daftardetaildokumentravel/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

var errBoom = errors.New("basis data mati")

// directHandler merakit handler tanpa middleware, supaya cabang penjaga di dalam handler
// dapat diuji langsung.
type directHandler struct {
	handler       *Handler
	repo          *memory.Repo
	documents     *memory.DocumentRepo
	unknownErrors []error
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()
	d := &directHandler{
		repo:      memory.NewRepo(memory.SampleList()...),
		documents: memory.NewDocumentRepo(memory.SampleDocumentList()...),
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector:     func(string) (daftardetaildokumentravel.Repo, error) { return d.repo, nil },
		DocumentSelector: func(string) (daftardetaildokumentravel.DocumentRepo, error) { return d.documents, nil },
	})
	require.NoError(t, err)

	d.handler, err = NewHandler(Options{
		Service: service,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.unknownErrors = append(d.unknownErrors, err)
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	require.NoError(t, err)
	return d
}

func request(method, body string, withPortal bool, id string) *http.Request {
	r := httptest.NewRequest(method, "/master/daftar-detail-dokumen-travel", strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	route := chi.NewRouteContext()
	if id != "" {
		route.URLParams.Add("id", id)
	}
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, route))
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := NewHandler(Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	d := newDirectHandler(t)
	_, err = NewHandler(Options{Service: d.handler.service})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestHandlersWithoutActivePortalAreRejected(t *testing.T) {
	d := newDirectHandler(t)
	handlers := map[string]http.HandlerFunc{
		"list": d.handler.List, "get": d.handler.Get, "create": d.handler.Create,
		"update": d.handler.Update, "documents": d.handler.Documents,
	}
	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			before := len(d.unknownErrors)
			w := httptest.NewRecorder()
			h(w, request(http.MethodGet, `{}`, false, "00001"))
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.ErrorIs(t, d.unknownErrors[before], portal.ErrNotStated)
		})
	}
}

func TestRepoFailuresAreForwardedToSharedErrorWriter(t *testing.T) {
	d := newDirectHandler(t)
	d.repo.SetError(errBoom)
	d.documents.SetError(errBoom)

	calls := map[string]struct {
		h http.HandlerFunc
		r *http.Request
	}{
		"list":      {d.handler.List, request(http.MethodGet, "", true, "")},
		"get":       {d.handler.Get, request(http.MethodGet, "", true, "00001")},
		"create":    {d.handler.Create, request(http.MethodPost, `{"nama_dokumen":"x"}`, true, "")},
		"update":    {d.handler.Update, request(http.MethodPut, `{"nama_dokumen":"x"}`, true, "00001")},
		"documents": {d.handler.Documents, request(http.MethodGet, "", true, "")},
	}
	for name, c := range calls {
		t.Run(name, func(t *testing.T) {
			before := len(d.unknownErrors)
			w := httptest.NewRecorder()
			c.h(w, c.r)
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.ErrorIs(t, d.unknownErrors[before], errBoom)
		})
	}
}

func TestGetAndUpdateWithoutIDAreReportedAsNotFound(t *testing.T) {
	d := newDirectHandler(t)
	for name, h := range map[string]http.HandlerFunc{"get": d.handler.Get, "update": d.handler.Update} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h(w, request(http.MethodPut, `{}`, true, ""))
			require.Equal(t, http.StatusNotFound, w.Code)
			var body ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, CodeNotFound, body.Code)
		})
	}
}

func TestMalformedBodiesAreRejected(t *testing.T) {
	d := newDirectHandler(t)
	for _, body := range []string{`{`, `{"nama_dokumen":"a"} {}`} {
		w := httptest.NewRecorder()
		d.handler.Create(w, request(http.MethodPost, body, true, ""))
		require.Equal(t, http.StatusBadRequest, w.Code, body)
		var got ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Equal(t, CodeMalformedRequest, got.Code)
	}
}
