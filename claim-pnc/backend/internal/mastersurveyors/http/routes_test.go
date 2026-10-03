package mastersurveyorshttp_test

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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/usecase"

	mastersurveyorshttp "claim-pnc/internal/mastersurveyors/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// stubService merekam permintaan dan menjawab dengan nilai yang ditentukan.
type stubService struct {
	filters    []mastersurveyors.Filter
	submitted  []usecase.Submission
	submitters []usecase.Submitter
	ids        []string
	decisions  []usecase.Decision
	committees []usecase.Committee

	surveyor mastersurveyors.Surveyor
	err      error
}

func (s *stubService) List(_ context.Context, _ string, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error) {
	s.filters = append(s.filters, f)
	if s.err != nil {
		return nil, 0, s.err
	}
	return []mastersurveyors.Surveyor{s.surveyor}, 9, nil
}

func (s *stubService) Get(_ context.Context, _, id string) (mastersurveyors.Surveyor, error) {
	s.ids = append(s.ids, id)
	return s.surveyor, s.err
}

func (s *stubService) Submit(_ context.Context, _ string, in usecase.Submission, by usecase.Submitter) (mastersurveyors.Surveyor, error) {
	s.submitted = append(s.submitted, in)
	s.submitters = append(s.submitters, by)
	return s.surveyor, s.err
}

func (s *stubService) Update(_ context.Context, _, id string, in usecase.Submission, by usecase.Submitter) (mastersurveyors.Surveyor, error) {
	s.ids = append(s.ids, id)
	s.submitted = append(s.submitted, in)
	s.submitters = append(s.submitters, by)
	return s.surveyor, s.err
}

func (s *stubService) Decide(_ context.Context, _, id string, d usecase.Decision, by usecase.Committee) (mastersurveyors.Surveyor, error) {
	s.ids = append(s.ids, id)
	s.decisions = append(s.decisions, d)
	s.committees = append(s.committees, by)
	return s.surveyor, s.err
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fallback(w http.ResponseWriter, r *http.Request, err error) {
	portalhttp.WithPortalError(func(w http.ResponseWriter, r *http.Request, err error) {
		writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
	}, writeJSON)(w, r, err)
}

type fixture struct {
	router  http.Handler
	handler *mastersurveyorshttp.Handler
	service *stubService
	logs    *bytes.Buffer
}

func newFixture(t *testing.T, withCaller bool) fixture {
	t.Helper()
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	decided := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	service := &stubService{surveyor: mastersurveyors.Surveyor{
		ID: "SVY-1", TypeCode: "1", Name: "PT Survei", Status: mastersurveyors.ApprovalStatus("1"),
		Committee: "KOMITE", DecidedAt: &decided,
	}}

	handler, err := mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{
		Service: service,
		Caller: func(context.Context) (mastersurveyorshttp.Caller, bool) {
			return mastersurveyorshttp.Caller{Identity: "K1", Name: "Komite Satu"}, withCaller
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    fallback,
	})
	require.NoError(t, err)

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		mastersurveyorshttp.Mount(api, handler, portalhttp.ActivePortalDeps{
			Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
			ReadyAliases: func() []string { return []string{"ASM"} },
			Logger:       logger,
			WriteError:   fallback,
		})
	})
	return fixture{router: router, handler: handler, service: service, logs: logs}
}

func (f fixture) do(t *testing.T, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	content := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &content)
	return recorder, content
}

const route = "/api/master/surveyor"

func TestNewHandlerRequiresItsDependencies(t *testing.T) {
	service := &stubService{}
	caller := func(context.Context) (mastersurveyorshttp.Caller, bool) { return mastersurveyorshttp.Caller{}, true }

	_, err := mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")
	_, err = mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{Service: service})
	require.ErrorContains(t, err, "Caller wajib diisi")
	_, err = mastersurveyorshttp.NewHandler(mastersurveyorshttp.Options{Service: service, Caller: caller})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestListPassesTheFilter(t *testing.T) {
	f := newFixture(t, true)

	recorder, body := f.do(t, http.MethodGet,
		route+"/?nama=budi&login=bd&kode_tipe=1&status=0&antrean_saya=1&limit=5&offset=10", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, float64(9), body["total"])
	require.Equal(t, "ASM", body["portal"])
	first := body["surveyor"].([]any)[0].(map[string]any)
	require.Equal(t, "SVY-1", first["id"])
	require.NotEmpty(t, first["status_label"])

	require.Equal(t, mastersurveyors.Filter{
		Name: "budi", AppLogin: "bd", TypeCode: "1", Status: mastersurveyors.ApprovalStatus("0"),
		MyCommitteeOnly: true, CommitteeIdentity: "K1", Limit: 5, Offset: 10,
	}, f.service.filters[0])
}

func TestListRejectsAMalformedFilter(t *testing.T) {
	f := newFixture(t, true)
	for _, query := range []string{"status=9", "limit=-1", "limit=x", "offset=-1", "offset=x"} {
		recorder, body := f.do(t, http.MethodGet, route+"/?"+query, "")
		require.Equal(t, http.StatusBadRequest, recorder.Code, query)
		require.Equal(t, mastersurveyorshttp.CodeMalformedRequest, body["kode"], query)
	}

	anonymous := newFixture(t, false)
	recorder, body := anonymous.do(t, http.MethodGet, route+"/?antrean_saya=1", "")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, body["pesan"], "setelah masuk")
}

func TestGetCreateUpdateAndDecide(t *testing.T) {
	f := newFixture(t, true)

	recorder, body := f.do(t, http.MethodGet, route+"/SVY-1", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "SVY-1", body["surveyor"].(map[string]any)["id"])

	recorder, _ = f.do(t, http.MethodPost, route+"/",
		`{"kode_tipe":"1","nama":"PT Survei","email":"a@contoh.co.id","login_aplikasi":"svy"}`)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "PT Survei", f.service.submitted[0].Name)
	require.Equal(t, "svy", f.service.submitted[0].AppLogin)
	require.Equal(t, usecase.Submitter{Identity: "K1", Name: "Komite Satu"}, f.service.submitters[0])

	recorder, _ = f.do(t, http.MethodPut, route+"/SVY-1", `{"nama":"PT Baru"}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "PT Baru", f.service.submitted[1].Name)

	recorder, _ = f.do(t, http.MethodPost, route+"/SVY-1/keputusan", `{"status":" 1 ","catatan":"ok"}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, usecase.Decision{Status: mastersurveyors.ApprovalStatus("1"), Note: "ok"},
		f.service.decisions[0])
	require.Equal(t, usecase.Committee{Identity: "K1", Name: "Komite Satu"}, f.service.committees[0])
	require.Equal(t, []string{"SVY-1", "SVY-1", "SVY-1"}, f.service.ids)
}

func TestWriteBodiesAreValidated(t *testing.T) {
	f := newFixture(t, true)
	for _, tc := range []struct{ method, path, body, message string }{
		{http.MethodPost, route + "/", "", "Badan permintaan kosong."},
		{http.MethodPost, route + "/", `{"tak_dikenal":1}`, "Badan permintaan bukan JSON yang dikenali."},
		{http.MethodPut, route + "/SVY-1", "{", "Badan permintaan bukan JSON yang dikenali."},
		{http.MethodPost, route + "/SVY-1/keputusan", "", "Badan permintaan kosong."},
	} {
		recorder, body := f.do(t, tc.method, tc.path, tc.body)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, tc.message, body["pesan"])
	}
}

func TestWritesWithoutACallerGoToTheFallback(t *testing.T) {
	f := newFixture(t, false)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, route + "/"},
		{http.MethodPut, route + "/SVY-1"},
		{http.MethodPost, route + "/SVY-1/keputusan"},
	} {
		recorder, body := f.do(t, tc.method, tc.path, `{}`)
		require.Equal(t, http.StatusTeapot, recorder.Code)
		require.Equal(t, "cadangan", body["kode"])
	}
}

func TestDomainErrorsAreMapped(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&mastersurveyors.ValidationError{Field: map[string]string{"nama": "wajib", "email": "salah"}},
			http.StatusUnprocessableEntity, mastersurveyorshttp.CodeValidationFailed},
		{mastersurveyors.ErrNameTaken, http.StatusConflict, mastersurveyorshttp.CodeNameTaken},
		{mastersurveyors.ErrLoginTaken, http.StatusConflict, mastersurveyorshttp.CodeLoginTaken},
		{mastersurveyors.ErrAlreadyDecided, http.StatusConflict, mastersurveyorshttp.CodeAlreadyDecided},
		{mastersurveyors.ErrNotAssignedCommittee, http.StatusForbidden, mastersurveyorshttp.CodeNotAssignedCommittee},
		{mastersurveyors.ErrUnknownStatus, http.StatusUnprocessableEntity, mastersurveyorshttp.CodeUnknownDecisionStatus},
		{mastersurveyors.ErrNoSite, http.StatusInternalServerError, mastersurveyorshttp.CodeSiteMissing},
		{mastersurveyors.ErrNotFound, http.StatusNotFound, mastersurveyorshttp.CodeNotFound},
		{errors.New("tak dikenal"), http.StatusTeapot, "cadangan"},
	}
	for _, tc := range cases {
		f := newFixture(t, true)
		f.service.err = tc.err
		for _, route := range []struct{ method, path, body string }{
			{http.MethodGet, route + "/", ""},
			{http.MethodGet, route + "/SVY-1", ""},
			{http.MethodPost, route + "/", `{}`},
			{http.MethodPut, route + "/SVY-1", `{}`},
			{http.MethodPost, route + "/SVY-1/keputusan", `{}`},
		} {
			recorder, body := f.do(t, route.method, route.path, route.body)
			require.Equal(t, tc.status, recorder.Code, tc.err.Error())
			require.Equal(t, tc.code, body["kode"], tc.err.Error())
		}
		if tc.status == http.StatusUnprocessableEntity && tc.code == mastersurveyorshttp.CodeValidationFailed {
			recorder, body := f.do(t, http.MethodGet, route+"/SVY-1", "")
			require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			detail := body["detail"].([]any)
			require.Equal(t, "email", detail[0].(map[string]any)["field"], "diurutkan menurut nama field")
		}
		if tc.status == http.StatusInternalServerError {
			require.Contains(t, f.logs.String(), "permintaan gagal")
		}
	}
}

func TestHandlersWithoutAPortalAreRejected(t *testing.T) {
	f := newFixture(t, true)
	for _, handler := range []http.HandlerFunc{
		f.handler.List, f.handler.Get, f.handler.Create, f.handler.Update, f.handler.Decide,
	} {
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}")))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}
