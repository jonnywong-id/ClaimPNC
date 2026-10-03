package inputacceptationhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/inputacceptation/repo/memory"
	"claim-pnc/internal/inputacceptation/usecase"
	"claim-pnc/internal/portal"

	inputacceptationhttp "claim-pnc/internal/inputacceptation/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const headerLogin = "X-Uji-Login"

type callerKey struct{}

// brokenRepo selalu gagal membaca.
type brokenRepo struct{ *memory.Store }

func (brokenRepo) Find(context.Context, inputacceptation.Query) (inputacceptation.Detail, error) {
	return inputacceptation.Detail{}, errors.New("basis data rusak")
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type fixture struct {
	router  http.Handler
	handler *inputacceptationhttp.Handler
	logs    *bytes.Buffer
}

func newFixture(t *testing.T, withFallback bool) fixture {
	t.Helper()
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inputacceptation.Repo, error) {
			switch alias {
			case "ASM":
				return store, nil
			case "ASI":
				return brokenRepo{memory.NewStore()}, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	options := inputacceptationhttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (inputacceptationhttp.Caller, bool) {
			login, ok := ctx.Value(callerKey{}).(string)
			return inputacceptationhttp.Caller{Login: login}, ok
		},
		Logger:    logger,
		WriteJSON: writeJSON,
	}
	if withFallback {
		options.FallbackErrorWriter = func(w http.ResponseWriter, r *http.Request, err error) {
			portalhttp.WithPortalError(func(w http.ResponseWriter, r *http.Request, err error) {
				writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
			}, writeJSON)(w, r, err)
		}
	}
	handler := inputacceptationhttp.NewHandler(options)

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if login, ok := r.Header[headerLogin]; ok {
				r = r.WithContext(context.WithValue(r.Context(), callerKey{}, login[0]))
			}
			next.ServeHTTP(w, r)
		})
	})
	router.Route("/api", func(api chi.Router) {
		inputacceptationhttp.Mount(api, handler, portalhttp.ActivePortalDeps{
			Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
			ReadyAliases: func() []string { return []string{"ASM", "ASI", "SMI"} },
			Logger:       logger,
			WriteError: func(w http.ResponseWriter, r *http.Request, err error) {
				writeJSON(w, r, http.StatusBadRequest, map[string]string{"kode": "portal"})
			},
		})
	})
	return fixture{router: router, handler: handler, logs: logs}
}

func (f fixture) do(t *testing.T, method, path, portalAlias, login, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	if login != "-" {
		request.Header.Set(headerLogin, login)
	}
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	content := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &content)
	return recorder, content
}

const route = "/api/input-acceptation/CLMNP-1001"

func TestDetailRendersEveryGroupWithValues(t *testing.T) {
	recorder, body := newFixture(t, true).do(t, http.MethodGet, route, "ASM", "ADMIN", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "CLMNP-1001", body["no_klaim"])
	require.Equal(t, "ASM", body["portal"])
	require.NotEmpty(t, body["selisih_terencana"])

	groups := body["kelompok"].([]any)
	require.NotEmpty(t, groups)

	filled, gridRows := 0, 0
	for _, raw := range groups {
		group := raw.(map[string]any)
		for _, f := range group["isian"].([]any) {
			if f.(map[string]any)["nilai"] != "" {
				filled++
			}
		}
		for _, g := range group["tabel"].([]any) {
			grid := g.(map[string]any)
			require.NotEmpty(t, grid["kolom"])
			gridRows += len(grid["baris"].([]any))
		}
	}
	require.Positive(t, filled, "nilai isian ikut dikirim")
	require.Positive(t, gridRows, "baris tabel ikut dikirim")
}

func TestDetailErrors(t *testing.T) {
	f := newFixture(t, true)

	recorder, body := f.do(t, http.MethodGet, route, "ASM", "-", "")
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeCallerUnknown, body["kode"])

	recorder, body = f.do(t, http.MethodGet, route, "ASM", " ", "")
	require.Equal(t, http.StatusConflict, recorder.Code, "login kosong sama dengan tidak dikenal")
	require.Equal(t, inputacceptationhttp.CodeCallerUnknown, body["kode"])

	recorder, body = f.do(t, http.MethodGet, "/api/input-acceptation/CLMNP-9999", "ASM", "ADMIN", "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeNotFound, body["kode"])

	recorder, body = f.do(t, http.MethodGet, "/api/input-acceptation/CLMP-70", "ASM", "ADMIN", "")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeValidationFail, body["kode"])
	require.NotEmpty(t, body["detail"])

	recorder, _ = f.do(t, http.MethodGet, route, "SMI", "ADMIN", "")
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, "galat portal lewat cadangan")

	recorder, body = f.do(t, http.MethodGet, route, "ASI", "ADMIN", "")
	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.Equal(t, "cadangan", body["kode"])
}

func TestAnUnknownErrorWithoutFallbackBecomes500(t *testing.T) {
	f := newFixture(t, false)
	recorder, body := f.do(t, http.MethodGet, route, "ASI", "ADMIN", "")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeInternalError, body["kode"])
	require.Contains(t, f.logs.String(), "permintaan gagal")
}

func TestSubmitIsValidatedThenRefusedForOwnership(t *testing.T) {
	f := newFixture(t, true)

	recorder, body := f.do(t, http.MethodPost, route, "ASM", "ADMIN", `{"isian":{"dla_no_ceding":"DLA/1"},`+
		`"tabel":{"adjustment_list":[{"type":"Interim"}]}}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeWriteNotOwned, body["kode"])

	recorder, body = f.do(t, http.MethodPost, route, "ASM", "ADMIN", `{"isian":{"treaty_id":"x"}}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, inputacceptationhttp.CodeValidationFail, body["kode"])

	recorder, body = f.do(t, http.MethodPost, route, "ASM", "ADMIN", `{"karangan":1}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code, "isian tak dikenal ditolak")
	require.Equal(t, "badan", body["detail"].([]any)[0].(map[string]any)["field"])

	recorder, _ = f.do(t, http.MethodPost, route, "ASM", "-", `{}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestHandlersWithoutAPortalAreRejected(t *testing.T) {
	f := newFixture(t, true)
	for _, handler := range []http.HandlerFunc{f.handler.Detail, f.handler.Submit} {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestACallerReaderIsRequired(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inputacceptation.Repo, error) { return memory.NewSampleStore(), nil },
	})
	require.NoError(t, err)
	handler := inputacceptationhttp.NewHandler(inputacceptationhttp.Options{
		Service: service, WriteJSON: writeJSON,
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(), portal.Portal{Alias: "ASM"}))
	recorder := httptest.NewRecorder()
	handler.Detail(recorder, request)
	require.Equal(t, http.StatusConflict, recorder.Code)
}
