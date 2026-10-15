package masterpasalaihttp_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
	masterpasalaiusecase "claim-pnc/internal/masterpasalai/usecase"
	"claim-pnc/internal/portal"

	masterpasalaihttp "claim-pnc/internal/masterpasalai/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// be4FailingRepo selalu gagal membaca.
type be4FailingRepo struct{ err error }

func (r be4FailingRepo) List(context.Context, masterpasalai.Filter) (masterpasalai.Page, error) {
	return masterpasalai.Page{}, r.err
}

// be4Handler merakit handler langsung (tanpa middleware) dan merekam galat yang ditulis.
func be4Handler(t *testing.T, repo masterpasalai.Repo) (*masterpasalaihttp.Handler, *error) {
	t.Helper()
	service, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{
		RepoSelector: func(string) (masterpasalai.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	var written error
	handler, err := masterpasalaihttp.NewHandler(masterpasalaihttp.Options{
		Service: service,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, _ any) {
			w.WriteHeader(status)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			written = err
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return handler, &written
}

// TestListWithoutActivePortalWritesNotStated membuktikan handler tanpa portal aktif menolak.
func TestListWithoutActivePortalWritesNotStated(t *testing.T) {
	handler, written := be4Handler(t, be4FailingRepo{})

	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/master/pasal-ai", nil))

	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.ErrorIs(t, *written, portal.ErrNotStated)
}

// TestListServiceFailureIsForwarded membuktikan galat repo diteruskan ke penulis galat.
func TestListServiceFailureIsForwarded(t *testing.T) {
	boom := errors.New("basis data mati")
	handler, written := be4Handler(t, be4FailingRepo{err: boom})

	request := httptest.NewRequest(http.MethodGet, "/master/pasal-ai?cari=x&halaman=2", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(),
		portal.Portal{ID: "1", Name: "Asuransi Sinar Mas", Alias: "ASM"}))

	recorder := httptest.NewRecorder()
	handler.List(recorder, request)

	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.ErrorIs(t, *written, boom)
}
