package daftartipedokumenbisnishttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/daftartipedokumenbisnis/repo/memory"
	daftartipedokumenbisnisusecase "claim-pnc/internal/daftartipedokumenbisnis/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	daftartipedokumenbisnishttp "claim-pnc/internal/daftartipedokumenbisnis/http"
	portalhttp "claim-pnc/internal/portal/http"
)

var errStorage = errors.New("penyimpanan mati")

// directFixture merakit handler tanpa server, supaya galat yang BUKAN milik modul dapat
// ditangkap langsung dari penulis cadangan.
type directFixture struct {
	handler  *daftartipedokumenbisnishttp.Handler
	repo     *memory.Repo
	fallback []error
}

func newDirectFixture(
	t *testing.T,
	caller func(context.Context) (daftartipedokumenbisnishttp.Caller, bool),
	referencesFail bool,
) *directFixture {
	t.Helper()

	f := &directFixture{repo: memory.NewRepo(memory.SampleList()...)}
	fail := func() error {
		if referencesFail {
			return errStorage
		}
		return nil
	}
	service, err := daftartipedokumenbisnisusecase.NewService(daftartipedokumenbisnisusecase.Options{
		RepoSelector: func(string) (daftartipedokumenbisnis.Repo, error) { return f.repo, nil },
		BusinessSelector: func(string) (daftartipedokumenbisnis.BusinessRepo, error) {
			return memory.NewBusinessRepo(memory.SampleBusinessList()...), fail()
		},
		DocumentTypeSelector: func(string) (daftartipedokumenbisnis.DocumentTypeRepo, error) {
			return memory.NewReferenceRepo(), fail()
		},
		DetailTypeDocSelector: func(string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) {
			return memory.NewReferenceRepo(), fail()
		},
		ObjectDocSelector: func(string) (daftartipedokumenbisnis.ObjectDocRepo, error) {
			return memory.NewReferenceRepo(), fail()
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	f.handler, err = daftartipedokumenbisnishttp.NewHandler(daftartipedokumenbisnishttp.Options{
		Service: service,
		Caller:  caller,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			f.fallback = append(f.fallback, err)
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return f
}

// request menyusun permintaan; withPortal menentukan apakah portal aktif ditanam di konteks,
// dan params mengisi parameter jalur chi.
func request(method, body string, withPortal bool, params map[string]string) *http.Request {
	r := httptest.NewRequest(method, "/", strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	route := chi.NewRouteContext()
	for key, value := range params {
		route.URLParams.Add(key, value)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, route)
	return r.WithContext(ctx)
}

func TestNewHandlerRejectsMissingDependencies(t *testing.T) {
	_, err := daftartipedokumenbisnishttp.NewHandler(daftartipedokumenbisnishttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := daftartipedokumenbisnisusecase.NewService(daftartipedokumenbisnisusecase.Options{
		RepoSelector:          func(string) (daftartipedokumenbisnis.Repo, error) { return nil, nil },
		BusinessSelector:      func(string) (daftartipedokumenbisnis.BusinessRepo, error) { return nil, nil },
		DocumentTypeSelector:  func(string) (daftartipedokumenbisnis.DocumentTypeRepo, error) { return nil, nil },
		DetailTypeDocSelector: func(string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) { return nil, nil },
		ObjectDocSelector:     func(string) (daftartipedokumenbisnis.ObjectDocRepo, error) { return nil, nil },
		Clock:                 clock.FixedAt(time.Time{}),
	})
	require.NoError(t, err)
	_, err = daftartipedokumenbisnishttp.NewHandler(daftartipedokumenbisnishttp.Options{Service: service})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestEveryHandlerWithoutPortalHandsErrNotStatedToFallback(t *testing.T) {
	f := newDirectFixture(t, nil, false)
	handlers := map[string]http.HandlerFunc{
		"ListBusinesses":       f.handler.ListBusinesses,
		"ListByBusiness":       f.handler.ListByBusiness,
		"Get":                  f.handler.Get,
		"Create":               f.handler.Create,
		"Update":               f.handler.Update,
		"AddCoverage":          f.handler.AddCoverage,
		"BusinessChoices":      f.handler.BusinessChoices,
		"DocumentTypeChoices":  f.handler.DocumentTypeChoices,
		"DetailTypeDocChoices": f.handler.DetailTypeDocChoices,
		"ObjectDocChoices":     f.handler.ObjectDocChoices,
	}
	for name, handle := range handlers {
		f.fallback = nil
		recorder := httptest.NewRecorder()
		handle(recorder, request(http.MethodGet, "", false, nil))
		require.Equal(t, http.StatusTeapot, recorder.Code, name)
		require.Len(t, f.fallback, 1, name)
		require.ErrorIs(t, f.fallback[0], portal.ErrNotStated, name)
	}
}

func TestRepoFailuresAreHandedToFallback(t *testing.T) {
	f := newDirectFixture(t, nil, false)
	f.repo.SetError(errStorage)

	cases := []struct {
		name   string
		handle http.HandlerFunc
		body   string
		params map[string]string
	}{
		{"ListBusinesses", f.handler.ListBusinesses, "", nil},
		{"ListByBusiness", f.handler.ListByBusiness, "", map[string]string{"businessID": "001"}},
		{"Get", f.handler.Get, "", map[string]string{"id": "10001"}},
		{"Create", f.handler.Create, `{"bisnis":["001"],"dokumen":[]}`, nil},
		{"Update", f.handler.Update, `{"id_tipe_dokumen":"1"}`, map[string]string{"id": "10001"}},
		{"AddCoverage", f.handler.AddCoverage, `{"id_jenis_klaim":"1"}`, map[string]string{"id": "10001"}},
	}
	for _, c := range cases {
		f.fallback = nil
		recorder := httptest.NewRecorder()
		c.handle(recorder, request(http.MethodPost, c.body, true, c.params))
		require.Equal(t, http.StatusTeapot, recorder.Code, c.name)
		require.Len(t, f.fallback, 1, c.name)
		require.ErrorIs(t, f.fallback[0], errStorage, c.name)
	}
}

func TestReferenceAndBusinessChoiceFailuresAreHandedToFallback(t *testing.T) {
	f := newDirectFixture(t, nil, true)

	for name, handle := range map[string]http.HandlerFunc{
		"BusinessChoices":      f.handler.BusinessChoices,
		"DocumentTypeChoices":  f.handler.DocumentTypeChoices,
		"DetailTypeDocChoices": f.handler.DetailTypeDocChoices,
		"ObjectDocChoices":     f.handler.ObjectDocChoices,
	} {
		f.fallback = nil
		recorder := httptest.NewRecorder()
		handle(recorder, request(http.MethodGet, "", true, nil))
		require.Equal(t, http.StatusTeapot, recorder.Code, name)
		require.ErrorIs(t, f.fallback[0], errStorage, name)
	}
}

func TestUpdateWithoutIDAnswersNotFound(t *testing.T) {
	f := newDirectFixture(t, nil, false)

	recorder := httptest.NewRecorder()
	f.handler.Update(recorder, request(http.MethodPut, `{"id_tipe_dokumen":"1"}`, true, nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), daftartipedokumenbisnishttp.CodeNotFound)
	require.Empty(t, f.fallback)
}

func TestMalformedBodiesAnswerBadRequest(t *testing.T) {
	f := newDirectFixture(t, nil, false)

	for name, c := range map[string]struct {
		handle http.HandlerFunc
		body   string
	}{
		"trailing document": {f.handler.Create, `{"bisnis":["001"],"dokumen":[]}{}`},
		"coverage not json": {f.handler.AddCoverage, `bukan json`},
		"update unknown":    {f.handler.Update, `{"x":1}`},
	} {
		recorder := httptest.NewRecorder()
		c.handle(recorder, request(http.MethodPost, c.body, true, map[string]string{"id": "10001"}))
		require.Equal(t, http.StatusBadRequest, recorder.Code, name)
		require.Contains(t, recorder.Body.String(), daftartipedokumenbisnishttp.CodeMalformedRequest, name)
	}
}

func TestCallerAbsentGivesEmptyPositionAndIdentity(t *testing.T) {
	// Pembaca identitas yang menyatakan tidak ada pemanggil: tombol "Pilih semua"
	// disembunyikan, dan penyimpanan tetap berjalan.
	absent := func(context.Context) (daftartipedokumenbisnishttp.Caller, bool) {
		return daftartipedokumenbisnishttp.Caller{Identity: "x", Position: "NONMBU"}, false
	}
	for name, caller := range map[string]func(context.Context) (daftartipedokumenbisnishttp.Caller, bool){
		"nil reader":    nil,
		"absent caller": absent,
	} {
		f := newDirectFixture(t, caller, false)

		recorder := httptest.NewRecorder()
		f.handler.BusinessChoices(recorder, request(http.MethodGet, "", true, nil))
		require.Equal(t, http.StatusOK, recorder.Code, name)
		var content map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &content), name)
		require.Equal(t, false, content["boleh_pilih_semua"], name)

		recorder = httptest.NewRecorder()
		f.handler.Create(recorder, request(http.MethodPost,
			`{"bisnis":["005"],"dokumen":[{"id_tipe_dokumen":"20001","id_object_dokumen":"",
			"id_detail_dokumen":"40001","detail_dokumen":"L","status_wajib":true,"minimum_dokumen":1}]}`,
			true, nil))
		require.Equal(t, http.StatusCreated, recorder.Code, name)
	}
}
