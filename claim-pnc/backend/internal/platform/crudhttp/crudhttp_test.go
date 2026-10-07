package crudhttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

type req struct{ Name string }

type out struct {
	status int
	body   any
	err    error
}

var errMissing = errors.New("tidak ada")

func setup(fail error, readOK bool) (*Handler[req], *out, *[]string) {
	o := &out{}
	calls := &[]string{}
	h := New(Spec[req]{
		WriteResponse:    func(_ http.ResponseWriter, _ *http.Request, s int, b any) { o.status, o.body = s, b },
		WriteModuleError: func(_ http.ResponseWriter, _ *http.Request, err error) { o.err = err },
		NotFound:         errMissing,
		Read:             func(http.ResponseWriter, *http.Request) (req, bool) { return req{"a"}, readOK },
		List: func(_ *http.Request, alias string) (any, error) {
			*calls = append(*calls, "list:"+alias)
			return "daftar", fail
		},
		Get: func(_ *http.Request, alias, id string) (any, error) {
			*calls = append(*calls, "get:"+id)
			return "satu", fail
		},
		Create: func(_ *http.Request, alias string, r req) (any, error) {
			*calls = append(*calls, "create:"+r.Name)
			return "baru", fail
		},
		Update: func(_ *http.Request, alias, id string, r req) (any, error) {
			*calls = append(*calls, "update:"+id+":"+r.Name)
			return "ubah", fail
		},
	})
	return h, o, calls
}

func request(id string, withPortal bool) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rc))
}

func TestEveryHandlerRequiresPortal(t *testing.T) {
	h, o, calls := setup(nil, true)
	for _, call := range []func(http.ResponseWriter, *http.Request){h.List, h.Get, h.Create, h.Update} {
		o.err = nil
		call(httptest.NewRecorder(), request("1", false))
		require.ErrorIs(t, o.err, portal.ErrNotStated)
	}
	require.Empty(t, *calls)
}

func TestStatusesAndArguments(t *testing.T) {
	h, o, calls := setup(nil, true)
	h.List(httptest.NewRecorder(), request("", true))
	require.Equal(t, []any{http.StatusOK, "daftar"}, []any{o.status, o.body})
	h.Get(httptest.NewRecorder(), request("7", true))
	require.Equal(t, http.StatusOK, o.status)
	h.Create(httptest.NewRecorder(), request("", true))
	require.Equal(t, http.StatusCreated, o.status)
	h.Update(httptest.NewRecorder(), request("9", true))
	require.Equal(t, []any{http.StatusOK, "ubah"}, []any{o.status, o.body})
	require.Equal(t, []string{"list:ASM", "get:7", "create:a", "update:9:a"}, *calls)
}

func TestMissingIDAndUnreadableBody(t *testing.T) {
	h, o, _ := setup(nil, true)
	h.Get(httptest.NewRecorder(), request("", true))
	require.ErrorIs(t, o.err, errMissing)
	o.err = nil
	h.Update(httptest.NewRecorder(), request("", true))
	require.ErrorIs(t, o.err, errMissing)

	h, _, calls := setup(nil, false)
	h.Create(httptest.NewRecorder(), request("", true))
	h.Update(httptest.NewRecorder(), request("9", true))
	require.Empty(t, *calls)
}

func TestWorkErrorsGoToModuleWriter(t *testing.T) {
	broken := errors.New("basis data mati")
	h, o, _ := setup(broken, true)
	h.List(httptest.NewRecorder(), request("", true))
	require.ErrorIs(t, o.err, broken)
	require.Zero(t, o.status)
}
