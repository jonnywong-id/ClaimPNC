package inboxanalystdoctorhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/usecase"

	analystdoctorhttp "claim-pnc/internal/inboxanalystdoctor/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

var errBroken = errors.New("penyimpanan rusak")

// brokenRepo selalu gagal membaca antrean.
type brokenRepo struct{}

func (brokenRepo) List(context.Context, string, inboxanalystdoctor.Filter) (inboxanalystdoctor.Page, error) {
	return inboxanalystdoctor.Page{}, errBroken
}

func plainJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// bareServer membentuk handler TANPA cadangan galat, tanpa jam, dan tanpa zona waktu —
// jalur bawaan yang tidak disentuh routes_test.go.
func bareServer(t *testing.T, logs *bytes.Buffer, caller analystdoctorhttp.CallerReader) (*analystdoctorhttp.Handler, http.Handler) {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxanalystdoctor.Repo, error) { return brokenRepo{}, nil },
	})
	require.NoError(t, err)
	logger := slog.New(slog.NewTextHandler(logs, nil))
	handler := analystdoctorhttp.NewHandler(analystdoctorhttp.Options{
		Service:   service,
		GetCaller: caller,
		Logger:    logger,
		WriteJSON: plainJSON,
	})
	router := chi.NewRouter()
	analystdoctorhttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   func(w http.ResponseWriter, r *http.Request, err error) { plainJSON(w, r, http.StatusBadRequest, nil) },
	})
	return handler, router
}

func knownLogin(context.Context) (analystdoctorhttp.Caller, bool) {
	return analystdoctorhttp.Caller{Login: "DOKTER1"}, true
}

func TestHandlersWithoutActivePortalAnswerInternalError(t *testing.T) {
	var logs bytes.Buffer
	handler, _ := bareServer(t, &logs, knownLogin)

	for name, serve := range map[string]http.HandlerFunc{"metadata": handler.Metadata, "list": handler.List} {
		record := httptest.NewRecorder()
		serve(record, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equal(t, http.StatusInternalServerError, record.Code, name)
		var body map[string]any
		require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
		require.Equal(t, analystdoctorhttp.CodeInternalError, body["kode"], name)
	}
	require.Contains(t, logs.String(), "permintaan gagal")
}

func TestListRepoFailureWithoutFallbackIsInternalError(t *testing.T) {
	var logs bytes.Buffer
	_, router := bareServer(t, &logs, knownLogin)
	record := get(t, router, "/inbox-analyst-doctor?batas=-5&lewati=abc")
	require.Equal(t, http.StatusInternalServerError, record.Code)
	require.NotContains(t, record.Body.String(), errBroken.Error(), "rincian galat tidak dikirim ke peramban")
	require.Contains(t, logs.String(), errBroken.Error())
}

func TestListWithoutCallerReaderIsCallerUnknown(t *testing.T) {
	var logs bytes.Buffer
	_, router := bareServer(t, &logs, nil)
	record := get(t, router, "/inbox-analyst-doctor")
	require.Equal(t, http.StatusConflict, record.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
	require.Equal(t, analystdoctorhttp.CodeCallerUnknown, body["kode"])
}

func TestMetadataUsesDefaultLocation(t *testing.T) {
	var logs bytes.Buffer
	_, router := bareServer(t, &logs, knownLogin)
	record := get(t, router, "/inbox-analyst-doctor/keterangan")
	require.Equal(t, http.StatusOK, record.Code)
}
