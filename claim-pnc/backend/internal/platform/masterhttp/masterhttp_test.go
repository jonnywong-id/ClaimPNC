package masterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

type status string

type row struct{ ID, Name string }

type input struct{ Name string }

type actor struct{ Login string }

type saveReq struct {
	Name string `json:"nama"`
}

type decideReq struct {
	ID     []string `json:"id"`
	Status string   `json:"status"`
}

var errMissing = errors.New("tidak ada")

type fakeService struct {
	fail   error
	listed status
	by     actor
	ids    []string
}

func (s *fakeService) List(_ context.Context, _ string, st status, _ string) ([]row, error) {
	s.listed = st
	return []row{{"1", "satu"}}, s.fail
}
func (s *fakeService) Get(_ context.Context, _, id string) (row, error) {
	if id == "x" {
		return row{}, errMissing
	}
	return row{id, "satu"}, s.fail
}
func (s *fakeService) Create(_ context.Context, _ string, in input, by actor, _ *slog.Logger) (row, error) {
	s.by = by
	return row{"baru", in.Name}, s.fail
}
func (s *fakeService) Save(_ context.Context, _, id string, in input, by actor, _ *slog.Logger) (row, error) {
	s.by = by
	return row{id, in.Name}, s.fail
}
func (s *fakeService) Decide(_ context.Context, _ string, id []string, _ status, by actor, _ *slog.Logger) (int, error) {
	s.ids, s.by = id, by
	return len(id), s.fail
}

type out struct {
	status int
	body   any
	err    error
	shared error
}

func setup(svc *fakeService, known bool) (*Handler[row, input, status, actor, saveReq, decideReq], *out) {
	o := &out{}
	h := New(Spec[row, input, status, actor, saveReq, decideReq]{
		Module:           "uji/http",
		Service:          svc,
		Caller:           func(context.Context) (string, bool) { return "PETUGAS", known },
		Actor:            func(login string) actor { return actor{login} },
		WriteResponse:    func(_ http.ResponseWriter, _ *http.Request, s int, b any) { o.status, o.body = s, b },
		WriteError:       func(_ http.ResponseWriter, _ *http.Request, err error) { o.shared = err },
		WriteModuleError: func(_ http.ResponseWriter, _ *http.Request, err error) { o.err = err },
		NotFound:         errMissing,
		DefaultStatus:    "1",
		MaxBody:          1 << 10,
		Malformed:        "cacat",
		ToInput:          func(r saveReq) input { return input{r.Name} },
		ListBody:         func(_ *http.Request, rows []row, s status, alias string) any { return []any{rows, s, alias} },
		SingleBody:       func(_ *http.Request, r row, alias string) any { return []any{r, alias} },
		Decider:          svc,
		DecisionIDs:      func(d decideReq) []string { return d.ID },
		DecisionStatus:   func(d decideReq) string { return d.Status },
		MaxDecisionRows:  2,
		TooManyRows:      errors.New("terlalu banyak"),
		DecisionBody:     func(n int, s status, alias string) any { return []any{n, s, alias} },
	})
	return h, o
}

// request membangun permintaan dengan portal aktif "ASM" dan, bila id tidak kosong, kunci jalur.
func request(method, target, body, id string, withPortal bool) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := r.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	}
	rc := chi.NewRouteContext()
	if id != "" {
		rc.URLParams.Add("id", id)
	}
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rc))
}

func TestEveryHandlerRequiresPortal(t *testing.T) {
	h, o := setup(&fakeService{}, true)
	for _, call := range []func(http.ResponseWriter, *http.Request){h.List, h.Get, h.Create, h.Save, h.Decide} {
		o.err = nil
		call(httptest.NewRecorder(), request(http.MethodGet, "/", "{}", "1", false))
		require.ErrorIs(t, o.err, portal.ErrNotStated)
	}
}

func TestListUsesDefaultOrRequestedStatus(t *testing.T) {
	svc := &fakeService{}
	h, o := setup(svc, true)
	h.List(httptest.NewRecorder(), request(http.MethodGet, "/", "", "", true))
	require.Equal(t, status("1"), svc.listed)
	require.Equal(t, http.StatusOK, o.status)
	h.List(httptest.NewRecorder(), request(http.MethodGet, "/?status=2", "", "", true))
	require.Equal(t, status("2"), svc.listed)
}

func TestGetWithoutIDIsNotFound(t *testing.T) {
	h, o := setup(&fakeService{}, true)
	h.Get(httptest.NewRecorder(), request(http.MethodGet, "/", "", "", true))
	require.ErrorIs(t, o.err, errMissing)
	o.err = nil
	h.Get(httptest.NewRecorder(), request(http.MethodGet, "/", "", "x", true))
	require.ErrorIs(t, o.err, errMissing)
	h.Get(httptest.NewRecorder(), request(http.MethodGet, "/", "", "7", true))
	require.Equal(t, []any{row{"7", "satu"}, "ASM"}, o.body)
}

func TestWritesRequireCallerAndCarryActor(t *testing.T) {
	svc := &fakeService{}
	h, o := setup(svc, false)
	h.Create(httptest.NewRecorder(), request(http.MethodPost, "/", `{"nama":"a"}`, "", true))
	require.EqualError(t, o.shared, "uji/http: identitas pemanggil tidak ada di konteks")

	h, o = setup(svc, true)
	h.Create(httptest.NewRecorder(), request(http.MethodPost, "/", `{"nama":"a"}`, "", true))
	require.Equal(t, http.StatusCreated, o.status)
	require.Equal(t, actor{"PETUGAS"}, svc.by)
	h.Save(httptest.NewRecorder(), request(http.MethodPut, "/", `{"nama":"b"}`, "9", true))
	require.Equal(t, []any{row{"9", "b"}, "ASM"}, o.body)
	h.Save(httptest.NewRecorder(), request(http.MethodPut, "/", `{"nama":"b"}`, "", true))
	require.ErrorIs(t, o.err, errMissing)
}

func TestMalformedBodiesAreRejected(t *testing.T) {
	h, o := setup(&fakeService{}, true)
	for _, body := range []string{`{`, `{"tidak_dikenal":1}`, `{"nama":"a"}{}`} {
		o.status = 0
		h.Create(httptest.NewRecorder(), request(http.MethodPost, "/", body, "", true))
		require.Equal(t, http.StatusBadRequest, o.status, body)
		require.Equal(t, "cacat", o.body)
	}
}

func TestDecideLimitsRowsAndReportsChanged(t *testing.T) {
	svc := &fakeService{}
	h, o := setup(svc, true)
	h.Decide(httptest.NewRecorder(), request(http.MethodPost, "/", `{"id":["a","b","c"],"status":"2"}`, "", true))
	require.EqualError(t, o.err, "terlalu banyak")
	h.Decide(httptest.NewRecorder(), request(http.MethodPost, "/", `{"id":["a","b"],"status":"2"}`, "", true))
	require.Equal(t, []any{2, status("2"), "ASM"}, o.body)
	require.Equal(t, []string{"a", "b"}, svc.ids)
}

func TestServiceFailuresGoToModuleWriter(t *testing.T) {
	broken := errors.New("basis data mati")
	h, o := setup(&fakeService{fail: broken}, true)
	h.List(httptest.NewRecorder(), request(http.MethodGet, "/", "", "", true))
	require.ErrorIs(t, o.err, broken)
	o.err = nil
	h.Create(httptest.NewRecorder(), request(http.MethodPost, "/", `{"nama":"a"}`, "", true))
	require.ErrorIs(t, o.err, broken)
	o.err = nil
	h.Decide(httptest.NewRecorder(), request(http.MethodPost, "/", `{"id":["a"],"status":"1"}`, "", true))
	require.ErrorIs(t, o.err, broken)
}
