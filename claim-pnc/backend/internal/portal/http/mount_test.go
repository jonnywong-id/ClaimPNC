package portalhttp_test

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

func TestMountGetsServesRoutesBehindPortal(t *testing.T) {
	router := chi.NewRouter()
	hit := ""
	portalhttp.MountGets(router, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteError:   func(w http.ResponseWriter, _ *http.Request, _ error) { w.WriteHeader(http.StatusBadRequest) },
	},
		portalhttp.Route{Path: "/a", Handler: func(http.ResponseWriter, *http.Request) { hit = "a" }},
		portalhttp.Route{Path: "/b", Handler: func(http.ResponseWriter, *http.Request) { hit = "b" }},
	)

	req := httptest.NewRequest(http.MethodGet, "/b", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	router.ServeHTTP(httptest.NewRecorder(), req)
	require.Equal(t, "b", hit)

	hit = ""
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/a", nil))
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Empty(t, hit)
}
