package masterdokumentravelhttp

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

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/repo/memory"
	"claim-pnc/internal/masterdokumentravel/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler merakit handler tanpa middleware, supaya cabang penjaga di dalam handler
// (portal tidak ada di context, ID kosong, galat tak dikenal) dapat diuji langsung.
type directHandler struct {
	handler *Handler
	repo    *memory.Repo
	// unknownErrors mencatat galat yang diteruskan ke ErrorWriter bersama.
	unknownErrors []error
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()
	d := &directHandler{repo: memory.NewRepo(memory.SampleList()...)}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterdokumentravel.Repo, error) { return d.repo, nil },
	})
	require.NoError(t, err)

	d.handler, err = NewHandler(Options{
		Service: service,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json")
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

// request membentuk permintaan; withPortal menaruh portal ASM ke context, id mengisi
// parameter rute chi.
func request(method, body string, withPortal bool, id string) *http.Request {
	r := httptest.NewRequest(method, "/master/dokumen-travel", strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	route := chi.NewRouteContext()
	if id != "" {
		route.URLParams.Add("id", id)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, route)
	return r.WithContext(ctx)
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := NewHandler(Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterdokumentravel.Repo, error) { return nil, nil },
	})
	require.NoError(t, err)
	_, err = NewHandler(Options{Service: service})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestHandlersWithoutActivePortalAreRejected(t *testing.T) {
	d := newDirectHandler(t)
	handlers := map[string]http.HandlerFunc{
		"list": d.handler.List, "get": d.handler.Get, "create": d.handler.Create, "update": d.handler.Update,
	}
	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			before := len(d.unknownErrors)
			w := httptest.NewRecorder()
			h(w, request(http.MethodGet, `{}`, false, "100001"))
			// Galat portal tidak dikenal modul ini sehingga diteruskan ke ErrorWriter.
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.Len(t, d.unknownErrors, before+1)
			require.ErrorIs(t, d.unknownErrors[before], portal.ErrNotStated)
		})
	}
}

func TestRepoFailureIsForwardedToSharedErrorWriter(t *testing.T) {
	d := newDirectHandler(t)
	boom := errors.New("basis data mati")
	d.repo.SetError(boom)

	calls := []struct {
		name string
		h    http.HandlerFunc
		r    *http.Request
	}{
		{"list", d.handler.List, request(http.MethodGet, "", true, "")},
		{"get", d.handler.Get, request(http.MethodGet, "", true, "100001")},
		{"create", d.handler.Create, request(http.MethodPost, `{"judul":"x"}`, true, "")},
		{"update", d.handler.Update, request(http.MethodPut, `{"judul":"x"}`, true, "100001")},
	}
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			before := len(d.unknownErrors)
			w := httptest.NewRecorder()
			c.h(w, c.r)
			require.Equal(t, http.StatusInternalServerError, w.Code)
			require.ErrorIs(t, d.unknownErrors[before], boom)
		})
	}
}

func TestUpdateWithoutIDIsReportedAsNotFound(t *testing.T) {
	d := newDirectHandler(t)
	w := httptest.NewRecorder()
	d.handler.Update(w, request(http.MethodPut, `{"judul":"x"}`, true, ""))

	require.Equal(t, http.StatusNotFound, w.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, CodeNotFound, body.Code)
}

func TestMalformedBodiesAreRejected(t *testing.T) {
	d := newDirectHandler(t)
	for _, body := range []string{`{`, `{"judul":"a"}{"judul":"b"}`} {
		w := httptest.NewRecorder()
		d.handler.Update(w, request(http.MethodPut, body, true, "100001"))
		require.Equal(t, http.StatusBadRequest, w.Code, body)
		var got ErrorResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Equal(t, CodeMalformedRequest, got.Code)
	}
}

func TestMapErrorOnlyKnowsNotFound(t *testing.T) {
	status, body, known := mapError(masterdokumentravel.ErrNotFound)
	require.True(t, known)
	require.Equal(t, http.StatusNotFound, status)
	require.Equal(t, CodeNotFound, body.Code)

	_, _, known = mapError(errors.New("lain"))
	require.False(t, known)
}
