package crudhttp

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

type routesRecorder struct{ hit string }

func (r *routesRecorder) List(http.ResponseWriter, *http.Request)   { r.hit = "list" }
func (r *routesRecorder) Create(http.ResponseWriter, *http.Request) { r.hit = "create" }
func (r *routesRecorder) Get(_ http.ResponseWriter, q *http.Request) {
	r.hit = "get:" + chi.URLParam(q, "kode")
}
func (r *routesRecorder) Update(_ http.ResponseWriter, q *http.Request) {
	r.hit = "update:" + chi.URLParam(q, "kode")
}

func TestMountRegistersFourRoutesBehindPortal(t *testing.T) {
	router := chi.NewRouter()
	rec := &routesRecorder{}
	Mount(router, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteError:   func(w http.ResponseWriter, _ *http.Request, _ error) { w.WriteHeader(http.StatusBadRequest) },
	}, "/master/x", "kode", rec)

	for method, want := range map[string]string{
		http.MethodGet + " /master/x/":  "list",
		http.MethodPost + " /master/x/": "create",
		http.MethodGet + " /master/x/7": "get:7",
		http.MethodPut + " /master/x/7": "update:7",
	} {
		var verb, path string
		for i := range method {
			if method[i] == ' ' {
				verb, path = method[:i], method[i+1:]
				break
			}
		}
		req := httptest.NewRequest(verb, path, nil)
		req.Header.Set(portalhttp.HeaderPortal, "ASM")
		router.ServeHTTP(httptest.NewRecorder(), req)
		require.Equal(t, want, rec.hit, method)
	}

	rec.hit = ""
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/master/x/", nil))
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Empty(t, rec.hit)
}
