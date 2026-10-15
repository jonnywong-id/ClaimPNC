package daftardetailtipedokumenhttp

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

	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/repo/memory"
	"claim-pnc/internal/daftardetailtipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

var errStorage = errors.New("penyimpanan mati")

// unitHandler merakit Handler tanpa server supaya cabang yang dicegat middleware di
// aplikasi sungguhan (portal tidak ada di context, id kosong) dapat diuji langsung.
type unitHandler struct {
	handler   *Handler
	repo      *memory.Repo
	lastError error
}

func newUnitHandler(t *testing.T, caller func(context.Context) (Caller, bool), referenceFails bool) *unitHandler {
	t.Helper()
	u := &unitHandler{repo: memory.NewRepo(memory.SampleList()...)}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (daftardetailtipedokumen.Repo, error) { return u.repo, nil },
		ReferenceSelector: func(string) (daftardetailtipedokumen.ReferenceRepo, error) {
			if referenceFails {
				return nil, errStorage
			}
			return memory.NewSampleReferenceRepo(), nil
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	u.handler, err = NewHandler(Options{
		Service: service,
		Caller:  caller,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			u.lastError = err
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	require.NoError(t, err)
	return u
}

func (u *unitHandler) serve(h http.HandlerFunc, body string, withPortal bool, id string) (*httptest.ResponseRecorder, map[string]any) {
	request := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
	ctx := request.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{ID: "1", Alias: "ASM"})
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("id", id)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, route)

	recorder := httptest.NewRecorder()
	h(recorder, request.WithContext(ctx))
	content := map[string]any{}
	_ = json.NewDecoder(recorder.Body).Decode(&content)
	return recorder, content
}

func TestNewHandlerRejectsMissingParts(t *testing.T) {
	writeResponse := func(http.ResponseWriter, *http.Request, int, any) {}
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err := NewHandler(Options{WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Service wajib diisi")
	_, err = NewHandler(Options{Service: &usecase.Service{}, WriteError: writeError})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
	_, err = NewHandler(Options{Service: &usecase.Service{}, WriteResponse: writeResponse})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestHandlersWithoutActivePortalGoToGenericError(t *testing.T) {
	u := newUnitHandler(t, nil, false)
	for name, h := range map[string]http.HandlerFunc{
		"list": u.handler.List, "get": u.handler.Get, "create": u.handler.Create,
		"update": u.handler.Update, "references": u.handler.References,
	} {
		t.Run(name, func(t *testing.T) {
			u.lastError = nil
			recorder, _ := u.serve(h, `{}`, false, "100001")
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.ErrorIs(t, u.lastError, portal.ErrNotStated)
		})
	}
}

func TestEmptyIDIsNotFound(t *testing.T) {
	u := newUnitHandler(t, nil, false)
	for name, h := range map[string]http.HandlerFunc{"get": u.handler.Get, "update": u.handler.Update} {
		t.Run(name, func(t *testing.T) {
			recorder, content := u.serve(h, `{}`, true, "")
			require.Equal(t, http.StatusNotFound, recorder.Code)
			require.Equal(t, CodeNotFound, content["kode"])
		})
	}
}

func TestStorageFailuresGoToGenericError(t *testing.T) {
	u := newUnitHandler(t, nil, false)
	u.repo.SetError(errStorage)
	for name, h := range map[string]http.HandlerFunc{
		"list": u.handler.List, "get": u.handler.Get, "create": u.handler.Create, "update": u.handler.Update,
	} {
		t.Run(name, func(t *testing.T) {
			u.lastError = nil
			recorder, _ := u.serve(h, `{}`, true, "100001")
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.ErrorIs(t, u.lastError, errStorage)
		})
	}
}

func TestReferencesPortalFailureGoesToGenericError(t *testing.T) {
	u := newUnitHandler(t, nil, true)
	recorder, _ := u.serve(u.handler.References, "", true, "")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.ErrorIs(t, u.lastError, errStorage)
}

func TestGetMissingRowIsNotFound(t *testing.T) {
	u := newUnitHandler(t, nil, false)
	recorder, content := u.serve(u.handler.Get, "", true, "999")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, content["pesan"], "Detail tipe dokumen")
}

func TestIdentityFallsBackToEmptyWhenCallerUnknown(t *testing.T) {
	// Tanpa fungsi pemanggil sama sekali.
	u := newUnitHandler(t, nil, false)
	require.Equal(t, "", u.handler.identity(httptest.NewRequest(http.MethodGet, "/", nil)))

	// Pemanggil ada tetapi tidak dikenal.
	u = newUnitHandler(t, func(context.Context) (Caller, bool) { return Caller{Identity: "x"}, false }, false)
	require.Equal(t, "", u.handler.identity(httptest.NewRequest(http.MethodGet, "/", nil)))

	u = newUnitHandler(t, func(context.Context) (Caller, bool) { return Caller{Identity: "adminpnc"}, true }, false)
	require.Equal(t, "adminpnc", u.handler.identity(httptest.NewRequest(http.MethodGet, "/", nil)))
}

func TestTrailingJSONIsRejected(t *testing.T) {
	u := newUnitHandler(t, nil, false)
	recorder, content := u.serve(u.handler.Create, `{} {}`, true, "")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, CodeMalformedRequest, content["kode"])

	recorder, _ = u.serve(u.handler.Update, `{`, true, "100001")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
