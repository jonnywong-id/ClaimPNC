package inboxinvestigatorhttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxinvestigator/repo/memory"
	inboxinvestigatorusecase "claim-pnc/internal/inboxinvestigator/usecase"
	"claim-pnc/internal/portal"

	inboxinvestigatorhttp "claim-pnc/internal/inboxinvestigator/http"
)

func plainService(t *testing.T) *inboxinvestigatorusecase.Service {
	t.Helper()
	service, err := inboxinvestigatorusecase.NewService(inboxinvestigatorusecase.Options{
		RepoSelector: func(string) (inboxinvestigator.Repo, error) {
			return memory.NewSampleRepo(), nil
		},
		InvestigationSelector: func(string) (inboxinvestigator.InvestigationRepo, error) {
			return memory.NewInvestigationRepo(), nil
		},
	})
	require.NoError(t, err)
	return service
}

// Handler menolak dirakit tanpa layanan maupun penulis jawaban.
func TestNewHandlerRejectsAnIncompleteAssembly(t *testing.T) {
	writeJSON := func(http.ResponseWriter, *http.Request, int, any) {}
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err := inboxinvestigatorhttp.NewHandler(inboxinvestigatorhttp.Options{
		WriteResponse: writeJSON, WriteError: writeError,
	})
	require.EqualError(t, err, "inboxinvestigator/http: Service wajib diisi")

	_, err = inboxinvestigatorhttp.NewHandler(inboxinvestigatorhttp.Options{
		Service: plainService(t), WriteError: writeError,
	})
	require.EqualError(t, err,
		"inboxinvestigator/http: WriteResponse dan WriteError wajib diisi")
}

// Handler yang dipanggil di luar middleware portal menolak dengan galat portal, bukan
// melayani portal utama.
func TestListOutsideThePortalMiddlewareIsRejected(t *testing.T) {
	var received error
	handler, err := inboxinvestigatorhttp.NewHandler(inboxinvestigatorhttp.Options{
		Service: plainService(t),
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			received = err
			w.WriteHeader(http.StatusBadRequest)
		},
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/inbox/investigator", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.ErrorIs(t, received, portal.ErrNotStated)
}
