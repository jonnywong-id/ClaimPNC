package inboxprogressclaimhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/repo/memory"
	"claim-pnc/internal/inboxprogressclaim/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

var errForeign = errors.New("galat milik modul lain")

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// writeJSONTo menulis jawaban sebagai JSON apa adanya.
func writeJSONTo(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// newHandler merakit Handler di atas penyimpanan contoh. caller nil berarti fungsi
// pemanggil tidak dipasang.
func newHandler(t *testing.T, caller CallerReader, fallback ErrorWriter, logs *bytes.Buffer) *Handler {
	t.Helper()
	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxprogressclaim.Repo, error) {
			if alias != "ASM" {
				return nil, errForeign
			}
			return store, nil
		},
		Clock: fixedClock{at: time.Date(2026, time.September, 21, 9, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)

	return NewHandler(Options{
		Service:             service,
		GetCaller:           caller,
		Logger:              slog.New(slog.NewJSONHandler(logs, nil)),
		WriteJSON:           writeJSONTo,
		FallbackErrorWriter: fallback,
	})
}

func ownerCaller(context.Context) (Caller, bool) { return Caller{Login: memory.SampleOwner}, true }

func serve(h http.HandlerFunc, target, alias string) (*httptest.ResponseRecorder, map[string]any) {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	if alias != "" {
		request = request.WithContext(portalhttp.WithActivePortal(request.Context(), portal.Portal{ID: "1", Alias: alias}))
	}
	recorder := httptest.NewRecorder()
	h(recorder, request)
	content := map[string]any{}
	_ = json.NewDecoder(recorder.Body).Decode(&content)
	return recorder, content
}

func TestMetadataAnswersForActivePortal(t *testing.T) {
	h := newHandler(t, ownerCaller, nil, &bytes.Buffer{})
	recorder, content := serve(h.Metadata, "/inbox-progress-claim/bagian", "ASM")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, inboxprogressclaim.DefaultView, content["bagian_bawaan"])
	require.EqualValues(t, inboxprogressclaim.DefaultPageSize, content["ukuran_halaman"])

	lines := content["lini_bisnis"].([]any)
	require.Len(t, lines, 4)
	require.Equal(t, map[string]any{"kode": "PA", "label": "Personal Accident"}, lines[3])
}

func TestMetadataAndListWithoutPortalUseInternalError(t *testing.T) {
	logs := &bytes.Buffer{}
	h := newHandler(t, ownerCaller, nil, logs)

	for _, handler := range []http.HandlerFunc{h.Metadata, h.List} {
		recorder, content := serve(handler, "/inbox-progress-claim", "")
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		require.Equal(t, CodeInternalError, content["kode"])
	}
	// Galat 5xx dicatat ke log.
	require.Contains(t, logs.String(), "permintaan gagal")
}

func TestUnrecognizedErrorGoesToFallbackWhenProvided(t *testing.T) {
	var handed error
	fallback := func(w http.ResponseWriter, _ *http.Request, err error) {
		handed = err
		w.WriteHeader(http.StatusTeapot)
	}
	h := newHandler(t, ownerCaller, fallback, &bytes.Buffer{})

	recorder, _ := serve(h.List, "/inbox-progress-claim", "ASI")
	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.ErrorIs(t, handed, errForeign)
}

func TestListWithoutCallerIsConflict(t *testing.T) {
	cases := map[string]CallerReader{
		"no reader":   nil,
		"not present": func(context.Context) (Caller, bool) { return Caller{}, false },
		"empty login": func(context.Context) (Caller, bool) { return Caller{}, true },
	}
	for name, caller := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHandler(t, caller, nil, &bytes.Buffer{})
			recorder, content := serve(h.List, "/inbox-progress-claim", "ASM")
			require.Equal(t, http.StatusConflict, recorder.Code)
			require.Equal(t, CodeCallerUnknown, content["kode"])
		})
	}
}

func TestListPassesQueryAndPagination(t *testing.T) {
	h := newHandler(t, ownerCaller, nil, &bytes.Buffer{})
	recorder, content := serve(h.List, "/inbox-progress-claim?bagian=outstanding&halaman=1&ukuran=2&cari=PNCN", "ASM")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "ASM", content["portal"])
	body, err := json.Marshal(content)
	require.NoError(t, err)
	require.Contains(t, string(body), "PNCN.26.0101")
}

func TestListValidationErrorIs422(t *testing.T) {
	h := newHandler(t, ownerCaller, nil, &bytes.Buffer{})
	recorder, content := serve(h.List, "/inbox-progress-claim?bagian=tidak-ada", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, CodeValidationFail, content["kode"])
}

func TestPositiveNumber(t *testing.T) {
	require.Equal(t, 0, positiveNumber(""))
	require.Equal(t, 0, positiveNumber("abc"))
	require.Equal(t, 0, positiveNumber("-3"))
	require.Equal(t, 7, positiveNumber("7"))
}

func TestMountRegistersBothRoutesBehindPortal(t *testing.T) {
	h := newHandler(t, ownerCaller, nil, &bytes.Buffer{})
	router := chi.NewRouter()
	Mount(router, h, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		Logger:       slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
		WriteError: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	for _, path := range []string{"/inbox-progress-claim/bagian", "/inbox-progress-claim"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set(portalhttp.HeaderPortal, "ASM")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, path)
	}

	// Tanpa header portal, middleware menolak sebelum handler.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/inbox-progress-claim", nil))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
