package inboxsurveyhttp

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

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/repo/memory"
	"claim-pnc/internal/inboxsurvey/usecase"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

func writeTestJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// brokenRepo selalu gagal dengan galat yang bukan milik modul.
type brokenRepo struct{}

func (brokenRepo) List(context.Context, inboxsurvey.SurveyorIdentity, inboxsurvey.Filter, inboxsurvey.Readiness) (inboxsurvey.Page, error) {
	return inboxsurvey.Page{}, errors.New("rahasia basis data")
}

func (brokenRepo) Counts(context.Context, inboxsurvey.SurveyorIdentity, inboxsurvey.Readiness) ([]inboxsurvey.TabCount, error) {
	return nil, errors.New("rahasia basis data")
}

func (brokenRepo) KPI(context.Context, inboxsurvey.SurveyorIdentity, inboxsurvey.KPIFilter) ([]inboxsurvey.KPIRow, error) {
	return nil, errors.New("rahasia basis data")
}

// Readiness pada repo yang rusak mengembalikan kesiapan KOSONG, bukan galat.
//
// Itu memang kontraknya: kegagalan membaca ketersediaan tidak boleh menjatuhkan layar,
// melainkan jatuh ke perilaku paling berhati-hati.
func (brokenRepo) KPIYears(context.Context, inboxsurvey.SurveyorIdentity) ([]string, error) {
	return nil, errors.New("rahasia basis data")
}

func (brokenRepo) Readiness(context.Context) inboxsurvey.Readiness {
	return inboxsurvey.Readiness{}
}

type fixture struct {
	router chi.Router
	logs   *bytes.Buffer
}

func newFixture(t *testing.T, repo inboxsurvey.Repo, login string, withFallback bool) *fixture {
	t.Helper()

	store := memory.NewSampleStore()
	if repo == nil {
		repo = store
	}
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector:      func(string) (inboxsurvey.Repo, error) { return repo, nil },
		DirectorySelector: func(string) (inboxsurvey.Directory, error) { return store, nil },
	})
	require.NoError(t, err)

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeTestJSON(w, r, http.StatusTeapot, ErrorResponse{Code: "cadangan"})
		}, writeTestJSON)

	var fallback ErrorWriter
	if withFallback {
		fallback = ErrorWriter(portalError)
	}

	h := NewHandler(Options{
		Service: svc,
		GetCaller: func(context.Context) (Caller, bool) {
			return Caller{Login: login}, login != ""
		},
		Location:            time.UTC,
		Clock:               fixedClock{at: time.Date(2026, 9, 30, 3, 0, 0, 0, time.UTC)},
		Logger:              logger,
		WriteJSON:           writeTestJSON,
		FallbackErrorWriter: fallback,
	})

	router := chi.NewRouter()
	Mount(router, h, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		Logger:       logger,
		WriteError:   portalError,
	})
	return &fixture{router: router, logs: logs}
}

func (f *fixture) get(t *testing.T, target, alias string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if alias != "" {
		req.Header.Set(portalhttp.HeaderPortal, alias)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec, body
}

func TestMetadataEndpoint(t *testing.T) {
	f := newFixture(t, nil, memory.SampleLeaderLogin, true)

	rec, body := f.get(t, "/inbox-survey/keterangan", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ASM", body["portal"])
	require.Equal(t, string((inboxsurvey.Readiness{}).DefaultAvailableTab()), body["tab_bawaan"])
	// Nilainya dikirim APA ADANYA, persis isi TipeData dan TipeExport di Pega — bukan kode
	// pendek. Nilai yang sama itulah yang dibandingkan kueri terhadap kolom `tipe`.
	require.Equal(t, []any{"ALL", "OUTSTANDING", "FINAL"}, body["status_survei"])
	require.Equal(t, []any{"DATA SUMMARY", "DATA DETAIL"}, body["tipe_report"])
	require.Equal(t, []any{"1", "2", "3", "4"}, body["kuartal"])
	require.EqualValues(t, inboxsurvey.DefaultLimit, body["ukuran_halaman"])
	require.Len(t, body["kolom"], 13)
	require.Len(t, body["kolom_kpi"], 9)
	first := body["tab"].([]any)[0].(map[string]any)
	require.Equal(t, false, first["tersedia"])
	require.NotEmpty(t, first["alasan_tak_tersedia"])
}

func TestMissingPortalIsRejected(t *testing.T) {
	f := newFixture(t, nil, memory.SampleLeaderLogin, true)

	rec, body := f.get(t, "/inbox-survey", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, portalhttp.CodeNotStated, body["kode"])
}

func TestListEndpointReturnsTasksAndIdentity(t *testing.T) {
	f := newFixture(t, nil, memory.SampleLeaderLogin, true)

	rec, body := f.get(t,
		"/inbox-survey?tab=belum-dijawab&batas=5&lewati=-2&cari=%20", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "belum-dijawab", body["tab"])
	require.EqualValues(t, 5, body["batas"])
	require.EqualValues(t, 0, body["lewati"])
	identity := body["identitas"].(map[string]any)
	require.Equal(t, memory.SampleLeaderLogin, identity["login"])
	require.Equal(t, true, identity["leader"])
	require.GreaterOrEqual(t, identity["jumlah_tim"].(float64), float64(1))
	require.NotNil(t, body["data"])
}

func TestCountsAndKPIEndpoints(t *testing.T) {
	f := newFixture(t, nil, memory.SampleLeaderLogin, true)

	rec, body := f.get(t, "/inbox-survey/jumlah-tab", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, body["tab"])

	rec, body = f.get(t,
		"/inbox-survey/kpi?status_survei=FINAL&tipe_report=DATA+SUMMARY&kuartal=1&tahun=2026",
		"ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "FINAL", body["status_survei"])
	require.Equal(t, "DATA SUMMARY", body["tipe_report"])
	require.Equal(t, "1", body["kuartal"])
	require.Equal(t, "2026", body["tahun"])
}

// TestPanelKPIMenolakIsianYangBelumLengkap mengunci jawaban 422, bukan tabel kosong.
func TestPanelKPIMenolakIsianYangBelumLengkap(t *testing.T) {
	f := newFixture(t, nil, memory.SampleLeaderLogin, true)

	rec, body := f.get(t, "/inbox-survey/kpi", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, CodeKPIFilterIncomplete, body["kode"])

	// DATA DETAIL menghasilkan BENTUK yang berbeda, bukan penolakan — rule-nya sudah ada.
	rec, body = f.get(t,
		"/inbox-survey/kpi?status_survei=ALL&tipe_report=DATA+DETAIL", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "detail", body["bentuk"])
}

func TestUnknownCallerAnswersConflict(t *testing.T) {
	f := newFixture(t, nil, "", true)

	for _, path := range []string{"/inbox-survey", "/inbox-survey/jumlah-tab",
		"/inbox-survey/kpi?status_survei=ALL&tipe_report=DATA+SUMMARY"} {
		rec, body := f.get(t, path, "ASM")
		require.Equal(t, http.StatusConflict, rec.Code, path)
		require.Equal(t, CodeCallerUnknown, body["kode"], path)
	}
}

func TestOutsiderAnswersForbidden(t *testing.T) {
	f := newFixture(t, nil, memory.SampleOutsiderLogin, true)

	rec, body := f.get(t, "/inbox-survey", "ASM")
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, CodeNotSurveyor, body["kode"])
}

func TestRepoFailureUsesFallbackThenInternalError(t *testing.T) {
	withFallback := newFixture(t, brokenRepo{}, memory.SampleLeaderLogin, true)
	for _, path := range []string{"/inbox-survey", "/inbox-survey/jumlah-tab",
		"/inbox-survey/kpi?status_survei=ALL&tipe_report=DATA+SUMMARY"} {
		rec, body := withFallback.get(t, path, "ASM")
		require.Equal(t, http.StatusTeapot, rec.Code, path)
		require.Equal(t, "cadangan", body["kode"], path)
	}

	without := newFixture(t, brokenRepo{}, memory.SampleLeaderLogin, false)
	rec, body := without.get(t, "/inbox-survey", "ASM")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, CodeInternalError, body["kode"])
	require.NotContains(t, rec.Body.String(), "rahasia basis data")
	require.Contains(t, without.logs.String(), "rahasia basis data")
}

func TestHandlersWithoutPortalContext(t *testing.T) {
	store := memory.NewSampleStore()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector:      func(string) (inboxsurvey.Repo, error) { return store, nil },
		DirectorySelector: func(string) (inboxsurvey.Directory, error) { return store, nil },
	})
	require.NoError(t, err)
	var logs bytes.Buffer
	h := NewHandler(Options{
		Service: svc, WriteJSON: writeTestJSON,
		Logger: slog.New(slog.NewJSONHandler(&logs, nil)),
	})

	for name, handle := range map[string]http.HandlerFunc{
		"metadata": h.Metadata, "list": h.List, "counts": h.Counts, "kpi": h.KPI,
	} {
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code, name)
	}
}

func TestNewHandlerDefaultsClockAndLocation(t *testing.T) {
	h := NewHandler(Options{WriteJSON: writeTestJSON})
	require.NotNil(t, h.location)
	require.IsType(t, systemClock{}, h.clock)
	require.WithinDuration(t, time.Now(), h.clock.Now(), time.Minute)
	_, known := h.readCaller(httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, known)
}

func TestJakartaLocationIsSevenHoursAhead(t *testing.T) {
	_, offset := time.Date(2026, 1, 1, 0, 0, 0, 0, jakarta()).Zone()
	require.Equal(t, 7*3600, offset)
}

func TestToTaskDTOFormatsDateAndAging(t *testing.T) {
	now := time.Date(2026, 9, 30, 3, 0, 0, 0, time.UTC)
	task := inboxsurvey.SurveyTask{
		ClaimNumber: "PNCN.26.1",
		DateOfLoss:  time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC),
		CreatedAt:   time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
	}
	dto := toTaskDTO(task, now, nil)
	require.Equal(t, "2026-09-01", dto.DateOfLoss)
	require.Equal(t, 2, *dto.Aging)

	dto = toTaskDTO(task, now, time.FixedZone("WIB", 7*3600))
	require.Equal(t, "2026-09-02", dto.DateOfLoss)

	require.Empty(t, toTaskDTO(inboxsurvey.SurveyTask{}, now, nil).DateOfLoss)
	require.Nil(t, toTaskDTO(inboxsurvey.SurveyTask{}, now, nil).Aging)
}

func TestToIdentityDTOWithoutScope(t *testing.T) {
	dto := toIdentityDTO(inboxsurvey.SurveyorIdentity{Login: "X"})
	require.Equal(t, 0, dto.JumlahTim)
	require.Empty(t, dto.Cakupan)
}

func TestNonNegativeNumber(t *testing.T) {
	require.Equal(t, 4, nonNegativeNumber("4"))
	require.Equal(t, 0, nonNegativeNumber("-4"))
	require.Equal(t, 0, nonNegativeNumber("x"))
}
