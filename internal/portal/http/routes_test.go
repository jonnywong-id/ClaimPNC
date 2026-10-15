package portalhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/portal/repo/memory"
)

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fallbackWriter mencatat galat yang diteruskan dan menjawab 500.
type fallbackWriter struct{ got []error }

func (f *fallbackWriter) write(w http.ResponseWriter, r *http.Request, err error) {
	f.got = append(f.got, err)
	writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "galat_lain"})
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body
}

func TestPortalListMarksReadyAndSendsPrimary(t *testing.T) {
	h := portalhttp.NewHandler(portalhttp.Options{
		Repo:          memory.NewRepo(memory.SampleList()...),
		ReadyAliases:  func() []string { return []string{"asm", "SMI"} },
		PrimaryAlias:  "ASM",
		WriteResponse: writeJSON,
	})
	r := chi.NewRouter()
	portalhttp.Mount(r, h)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/portal", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body portalhttp.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ASM", body.Primary)
	require.Len(t, body.Portals, 6)
	require.Equal(t, portalhttp.PortalDTO{ID: "202600101", Name: "ASURANSI SINAR MAS", Alias: "ASM", Ready: true}, body.Portals[0])
	require.False(t, body.Portals[1].Ready)
	require.True(t, body.Portals[3].Ready)

	// Nama field JSON adalah kontrak berbahasa Indonesia.
	raw := decode(t, rec)
	require.Equal(t, "ASM", raw["utama"])
	first := raw["portal"].([]any)[0].(map[string]any)
	require.Equal(t, map[string]any{"id": "202600101", "nama": "ASURANSI SINAR MAS", "alias": "ASM", "siap": true}, first)
}

func TestPortalListWithoutReadyAliasesMarksAllNotReady(t *testing.T) {
	h := portalhttp.NewHandler(portalhttp.Options{
		Repo:          memory.NewRepo(memory.SampleList()...),
		WriteResponse: writeJSON,
	})
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/portal", nil))

	var body portalhttp.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	for _, p := range body.Portals {
		require.False(t, p.Ready, p.Alias)
	}
	require.Equal(t, "", body.Primary)
}

func TestPortalListRepoErrorGoesToErrorWriter(t *testing.T) {
	boom := errors.New("tabel portal tidak terbaca")
	repo := memory.NewRepo()
	repo.SetError(boom)
	fb := &fallbackWriter{}
	h := portalhttp.NewHandler(portalhttp.Options{Repo: repo, WriteResponse: writeJSON, WriteError: fb.write})

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/portal", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, []error{boom}, fb.got)
}

func TestWithPortalErrorMapsKnownErrors(t *testing.T) {
	fb := &fallbackWriter{}
	write := portalhttp.WithPortalError(fb.write, writeJSON)

	cases := []struct {
		err    error
		status int
		code   string
		pesan  string
	}{
		{portal.ErrNotStated, http.StatusBadRequest, portalhttp.CodeNotStated,
			"Portal entitas belum dipilih. Pilih portal lebih dulu sebelum membuka data entitas."},
		{fmt.Errorf("dibungkus: %w", portal.ErrNotFound), http.StatusBadRequest, portalhttp.CodeUnknown,
			"Portal entitas tidak dikenal."},
		{portal.ErrNotReady, http.StatusServiceUnavailable, portalhttp.CodeNotReady,
			"Basis data portal entitas ini belum tersedia. Hubungi administrator Claim PNC."},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		write(rec, httptest.NewRequest(http.MethodGet, "/", nil), c.err)
		require.Equal(t, c.status, rec.Code, c.code)
		require.Equal(t, map[string]any{"kode": c.code, "pesan": c.pesan}, decode(t, rec))
	}
	require.Empty(t, fb.got, "galat portal tidak diteruskan ke penulis galat lain")

	// Galat lain diteruskan apa adanya.
	other := errors.New("lain")
	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/", nil), other)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, []error{other}, fb.got)
}

// newMiddleware memasang ActivePortal di depan handler yang mencatat portal terpilih.
func newMiddleware(t *testing.T, repo portal.Repo, ready func() []string) (http.Handler, *fallbackWriter, *bytes.Buffer, *portal.Portal) {
	t.Helper()
	fb := &fallbackWriter{}
	logs := &bytes.Buffer{}
	var seen portal.Portal
	mw := portalhttp.ActivePortal(portalhttp.ActivePortalDeps{
		Repo:         repo,
		ReadyAliases: ready,
		Logger:       slog.New(slog.NewJSONHandler(logs, nil)),
		WriteError:   portalhttp.WithPortalError(fb.write, writeJSON),
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := portalhttp.ActivePortalFrom(r.Context())
		require.True(t, ok)
		seen = p
		w.WriteHeader(http.StatusNoContent)
	})
	return mw(next), fb, logs, &seen
}

func TestActivePortalPutsSelectedPortalInContext(t *testing.T) {
	handler, _, _, seen := newMiddleware(t, memory.NewRepo(memory.SampleList()...),
		func() []string { return []string{"ASM", "ASI"} })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(portalhttp.HeaderPortal, "asi")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "ASI", seen.Alias)
	require.Equal(t, "ASURANSI SIMAS INSURTECH", seen.Name)
}

func TestActivePortalRejectsAndLogsWarning(t *testing.T) {
	cases := []struct {
		header string
		status int
		code   string
	}{
		{"", http.StatusBadRequest, portalhttp.CodeNotStated},
		{"TIDAKADA", http.StatusBadRequest, portalhttp.CodeUnknown},
		{"SMAS", http.StatusServiceUnavailable, portalhttp.CodeNotReady},
	}
	for _, c := range cases {
		handler, _, logs, _ := newMiddleware(t, memory.NewRepo(memory.SampleList()...),
			func() []string { return []string{"ASM"} })

		req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
		if c.header != "" {
			req.Header.Set(portalhttp.HeaderPortal, c.header)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(t, c.status, rec.Code, c.header)
		require.Equal(t, c.code, decode(t, rec)["kode"])
		require.Contains(t, logs.String(), "permintaan portal ditolak")
		require.Contains(t, logs.String(), `"jalur":"/api/x"`)
	}
}

func TestActivePortalWithoutReadyFuncTreatsAllAsNotReady(t *testing.T) {
	handler, _, _, _ := newMiddleware(t, memory.NewRepo(memory.SampleList()...), nil)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestActivePortalRepoErrorIsForwarded(t *testing.T) {
	boom := errors.New("daftar portal gagal dibaca")
	repo := memory.NewRepo()
	repo.SetError(boom)
	handler, fb, _, _ := newMiddleware(t, repo, func() []string { return []string{"ASM"} })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, []error{boom}, fb.got)
}

func TestActivePortalFromEmptyContext(t *testing.T) {
	_, ok := portalhttp.ActivePortalFrom(context.Background())
	require.False(t, ok)

	ctx := portalhttp.WithActivePortal(context.Background(), portal.Portal{Alias: "SPK"})
	p, ok := portalhttp.ActivePortalFrom(ctx)
	require.True(t, ok)
	require.Equal(t, "SPK", p.Alias)
}
