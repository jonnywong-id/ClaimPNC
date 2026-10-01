package mastertipesurveyorshttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/mastertipesurveyors/usecase"
	"claim-pnc/internal/portal"

	mastertipesurveyorshttp "claim-pnc/internal/mastertipesurveyors/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// be4Repo adalah repo yang galatnya diatur per uji.
type be4Repo struct{ err error }

func (r be4Repo) List(context.Context) ([]mastertipesurveyors.SurveyorType, error) {
	return nil, r.err
}

func (r be4Repo) Get(context.Context, string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{}, r.err
}

func (r be4Repo) Insert(context.Context, string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{}, r.err
}

func (r be4Repo) Update(context.Context, string, string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{}, r.err
}

// be4Recorder menangkap jawaban penulis respons dan penulis galat.
type be4Recorder struct {
	status int
	body   any
	err    error
}

func be4NewHandler(t *testing.T, repo mastertipesurveyors.Repo) (*mastertipesurveyorshttp.Handler, *be4Recorder) {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastertipesurveyors.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	rec := &be4Recorder{}
	handler, err := mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{
		Service: service,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			rec.status, rec.body = status, body
			w.WriteHeader(status)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			rec.err = err
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return handler, rec
}

func be4WithPortal(r *http.Request) *http.Request {
	return r.WithContext(portalhttp.WithActivePortal(r.Context(),
		portal.Portal{ID: "1", Name: "ASM", Alias: "ASM"}))
}

// TestNewHandlerRequiresWriters membuktikan penulis respons dan galat wajib ada.
func TestNewHandlerRequiresWriters(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastertipesurveyors.Repo, error) { return be4Repo{}, nil },
	})
	require.NoError(t, err)
	_, err = mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{Service: service})
	require.ErrorContains(t, err, "WriteResponse")
	_, err = mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{})
	require.ErrorContains(t, err, "Service")
}

// TestEveryHandlerRequiresActivePortal membuktikan tiap handler menolak tanpa portal aktif.
func TestEveryHandlerRequiresActivePortal(t *testing.T) {
	handler, rec := be4NewHandler(t, be4Repo{})
	for _, call := range []http.HandlerFunc{handler.List, handler.Get, handler.Create, handler.Update} {
		rec.err = nil
		w := httptest.NewRecorder()
		call(w, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusTeapot, w.Code)
		require.ErrorIs(t, rec.err, portal.ErrNotStated)
	}
}

// TestUnknownErrorsGoToSharedWriter membuktikan galat penyimpanan diserahkan ke penulis bersama.
func TestUnknownErrorsGoToSharedWriter(t *testing.T) {
	boom := errors.New("basis data mati")
	handler, rec := be4NewHandler(t, be4Repo{err: boom})

	w := httptest.NewRecorder()
	handler.List(w, be4WithPortal(httptest.NewRequest(http.MethodGet, "/", nil)))
	require.Equal(t, http.StatusTeapot, w.Code)
	require.ErrorIs(t, rec.err, boom)

	rec.err = nil
	w = httptest.NewRecorder()
	handler.Get(w, be4WithPortal(httptest.NewRequest(http.MethodGet, "/", nil)))
	require.ErrorIs(t, rec.err, boom)
}

// TestUpdateWithoutCodeIsNotFound membuktikan jalur tanpa kode dijawab 404.
func TestUpdateWithoutCodeIsNotFound(t *testing.T) {
	handler, rec := be4NewHandler(t, be4Repo{})
	w := httptest.NewRecorder()
	handler.Update(w, be4WithPortal(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{}`))))
	require.Equal(t, http.StatusNotFound, w.Code)
	body, ok := rec.body.(mastertipesurveyorshttp.ErrorResponse)
	require.True(t, ok)
	require.Equal(t, mastertipesurveyorshttp.CodeNotFound, body.Code)
}

// be4Save menjalankan Create (kode kosong) atau Update dengan badan permintaan.
func be4Save(handler *mastertipesurveyorshttp.Handler, code, body string) *httptest.ResponseRecorder {
	request := be4WithPortal(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	w := httptest.NewRecorder()
	if code == "" {
		handler.Create(w, request)
		return w
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("kode", code)
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
	handler.Update(w, request)
	return w
}

// TestCodeTakenAndNoSiteAreMapped memeriksa pemetaan ErrCodeTaken (409) dan ErrNoSite (500).
func TestCodeTakenAndNoSiteAreMapped(t *testing.T) {
	// Repo sukses untuk List (pemeriksaan keunikan) dan gagal pada Insert.
	handler, rec := be4NewHandler(t, be4ListOK{err: mastertipesurveyors.ErrCodeTaken})
	w := be4Save(handler, "", `{"deskripsi":"BARU"}`)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Equal(t, mastertipesurveyorshttp.CodeCodeTaken, rec.body.(mastertipesurveyorshttp.ErrorResponse).Code)

	handler, rec = be4NewHandler(t, be4ListOK{err: mastertipesurveyors.ErrNoSite})
	w = be4Save(handler, "", `{"deskripsi":"BARU"}`)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, mastertipesurveyorshttp.CodeSiteMissing, rec.body.(mastertipesurveyorshttp.ErrorResponse).Code)
}

// TestUpdateBodyAndServiceFailures memeriksa badan cacat dan galat layanan pada Update.
func TestUpdateBodyAndServiceFailures(t *testing.T) {
	handler, rec := be4NewHandler(t, be4ListOK{})
	w := be4Save(handler, "1001", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, mastertipesurveyorshttp.CodeMalformedRequest, rec.body.(mastertipesurveyorshttp.ErrorResponse).Code)

	w = be4Save(handler, "1001", `{"deskripsi":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)

	w = be4Save(handler, "1001", `{"deskripsi":"BARU"}`)
	require.Equal(t, http.StatusOK, w.Code)
	single, ok := rec.body.(mastertipesurveyorshttp.SingleResponse)
	require.True(t, ok)
	encoded, err := json.Marshal(single)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"BARU"`)

	handler, rec = be4NewHandler(t, be4ListOK{err: mastertipesurveyors.ErrDescriptionTaken})
	w = be4Save(handler, "1001", `{"deskripsi":"BARU"}`)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Equal(t, mastertipesurveyorshttp.CodeDescriptionTaken, rec.body.(mastertipesurveyorshttp.ErrorResponse).Code)
}

// be4ListOK berhasil untuk List/Get dan mengembalikan err untuk Insert/Update.
type be4ListOK struct{ err error }

func (r be4ListOK) List(context.Context) ([]mastertipesurveyors.SurveyorType, error) {
	return nil, nil
}

func (r be4ListOK) Get(_ context.Context, code string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: code}, nil
}

func (r be4ListOK) Insert(_ context.Context, d string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: "1005", Description: d}, r.err
}

func (r be4ListOK) Update(_ context.Context, c, d string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: c, Description: d}, r.err
}
