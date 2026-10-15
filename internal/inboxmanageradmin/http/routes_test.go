package inboxmanageradminhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/repo/memory"
	"claim-pnc/internal/inboxmanageradmin/usecase"
	"claim-pnc/internal/portal"

	inboxmanageradminhttp "claim-pnc/internal/inboxmanageradmin/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

var (
	errStore = errors.New("penyimpanan mati")
	now      = time.Date(2026, time.September, 26, 8, 0, 0, 0, time.UTC)
)

type fixedClock struct{}

func (fixedClock) Now() time.Time { return now }

// staticRepo mengembalikan baris yang sama setiap kali, tanpa menyalin maupun mengurutkan,
// supaya uji ekspor berjumlah besar tetap cepat.
type staticRepo struct{ rows []inboxmanageradmin.WorkItem }

func (r staticRepo) List(context.Context, inboxmanageradmin.Query) ([]inboxmanageradmin.WorkItem, error) {
	out := make([]inboxmanageradmin.WorkItem, len(r.rows))
	copy(out, r.rows)
	return out, nil
}

// flakyRepo berhasil pada pemanggilan pertama lalu gagal, meniru koneksi yang putus di
// tengah ekspor.
type flakyRepo struct {
	mu    sync.Mutex
	calls int
	rows  []inboxmanageradmin.WorkItem
}

func (r *flakyRepo) List(context.Context, inboxmanageradmin.Query) ([]inboxmanageradmin.WorkItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.calls > 1 {
		return nil, errStore
	}
	return append([]inboxmanageradmin.WorkItem(nil), r.rows...), nil
}

type fixture struct {
	server *httptest.Server
	caller *inboxmanageradminhttp.Caller
	known  *bool
	logs   *bytes.Buffer
}

type fixtureOptions struct {
	repos       map[string]inboxmanageradmin.Repo
	lines       inboxmanageradmin.LineBusinessRepo
	noCaller    bool
	nilLogger   bool
	readyExtras []string
}

func numbered(count int) []inboxmanageradmin.WorkItem {
	rows := make([]inboxmanageradmin.WorkItem, 0, count)
	for i := 0; i < count; i++ {
		rows = append(rows, inboxmanageradmin.WorkItem{CaseID: fmt.Sprintf("PNC-%05d", i)})
	}
	return rows
}

func newFixture(t *testing.T, o fixtureOptions) *fixture {
	t.Helper()

	store := memory.NewStrictStore()
	store.SetLineBusiness("petugas.pa", inboxmanageradmin.LinePA)
	store.SetLineBusiness("petugas.travel", inboxmanageradmin.LineTravel)
	lines := o.lines
	if lines == nil {
		lines = store
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanageradmin.Repo, error) {
			if repo, listed := o.repos[alias]; listed {
				return repo, nil
			}
			if alias == "ASM" {
				return store, nil
			}
			return nil, portal.ErrNotReady
		},
		LineBusinessSelector: func(alias string) (inboxmanageradmin.LineBusinessRepo, error) {
			if alias == "ASI" {
				// Galat yang tidak dikenali modul ini maupun penulis portal.
				return nil, errStore
			}
			return lines, nil
		},
		Clock: fixedClock{},
	})
	require.NoError(t, err)

	logs := &bytes.Buffer{}
	var logger *slog.Logger
	if !o.nilLogger {
		logger = slog.New(slog.NewTextHandler(logs, nil))
	}

	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
	fallback := portalhttp.WithPortalError(func(w http.ResponseWriter, r *http.Request, err error) {
		writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "cadangan"})
	}, writeJSON)

	caller := &inboxmanageradminhttp.Caller{Login: "petugas.pa"}
	known := true
	var reader inboxmanageradminhttp.CallerReader
	if !o.noCaller {
		reader = func(context.Context) (inboxmanageradminhttp.Caller, bool) { return *caller, known }
	}

	handler := inboxmanageradminhttp.NewHandler(inboxmanageradminhttp.Options{
		Service:             service,
		GetCaller:           reader,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: inboxmanageradminhttp.ErrorWriter(fallback),
	})

	router := chi.NewRouter()
	inboxmanageradminhttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return append([]string{"ASM", "ASI"}, o.readyExtras...) },
		Logger:       slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		WriteError:   fallback,
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &fixture{server: server, caller: caller, known: &known, logs: logs}
}

func (f *fixture) get(t *testing.T, path, portalAlias string) (*http.Response, []byte) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, f.server.URL+path, nil)
	require.NoError(t, err)
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	var body bytes.Buffer
	_, err = body.ReadFrom(response.Body)
	require.NoError(t, err)
	return response, body.Bytes()
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	content := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &content), string(raw))
	return content
}

func TestMetadataListsOnlyTheCallersTab(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	response, raw := f.get(t, "/inbox-manager-admin/tab", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	content := decode(t, raw)
	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, inboxmanageradmin.TabPA, content["tab_bawaan"])
	require.Equal(t, inboxmanageradmin.LinePA, content["lini_bisnis_anda"])
	require.Len(t, content["tab"], 1)
	require.Len(t, content["semua_tab"], 3)
	require.Len(t, content["lini_bisnis_yang_diharapkan"], 3)
	require.NotContains(t, content, "selisih_terencana")

	tab := content["tab"].([]any)[0].(map[string]any)
	require.Equal(t, inboxmanageradmin.OrgUnitPA, tab["unit_organisasi"])
	require.NotEmpty(t, tab["kolom"])
}

func TestRequestsWithoutPortalAreRejectedByMiddleware(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	for _, path := range []string{"/inbox-manager-admin/tab", "/inbox-manager-admin", "/inbox-manager-admin/ekspor"} {
		response, raw := f.get(t, path, "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode, path)
		require.Equal(t, portalhttp.CodeNotStated, decode(t, raw)["kode"], path)
	}
}

func TestUnreadableCallerAnswersConflict(t *testing.T) {
	cases := map[string]func(f *fixture){
		"caller not known": func(f *fixture) { *f.known = false },
		"blank login":      func(f *fixture) { f.caller.Login = "  " },
	}
	for name, setup := range cases {
		f := newFixture(t, fixtureOptions{})
		setup(f)
		for _, path := range []string{"/inbox-manager-admin/tab", "/inbox-manager-admin", "/inbox-manager-admin/ekspor"} {
			response, raw := f.get(t, path, "ASM")
			require.Equal(t, http.StatusConflict, response.StatusCode, name+path)
			require.Equal(t, inboxmanageradminhttp.CodeCallerUnknown, decode(t, raw)["kode"], name+path)
		}
	}

	noReader := newFixture(t, fixtureOptions{noCaller: true})
	response, raw := noReader.get(t, "/inbox-manager-admin", "ASM")
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, inboxmanageradminhttp.CodeCallerUnknown, decode(t, raw)["kode"])
}

func TestListReturnsPageWithDefaultsForUnreadableNumbers(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	response, raw := f.get(t, "/inbox-manager-admin?halaman=abc&ukuran=-5", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	content := decode(t, raw)

	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, inboxmanageradmin.TabPA, content["tab"].(map[string]any)["kode"])
	require.Equal(t, map[string]any{
		"halaman": float64(1), "ukuran": float64(inboxmanageradmin.DefaultPageSize),
		"total": float64(2), "total_halaman": float64(1),
	}, content["paginasi"])

	rows := content["baris"].([]any)
	require.Len(t, rows, 2)
	first := rows[0].(map[string]any)
	require.Equal(t, "PNC-9101", first["id"])
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-9101", first["referensi"])
	require.Equal(t, "2026-09-25", first["tanggal_pendaftaran"])
	require.Equal(t, "1 day ago", first["lama_waktu_klaim"])
	require.Equal(t, inboxmanageradmin.DisplayStatusFor("New"), first["status_klaim"])
}

func TestListHonoursExplicitPageNumbers(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	response, raw := f.get(t, "/inbox-manager-admin?tab=2&halaman=2&ukuran=1", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	content := decode(t, raw)
	rows := content["baris"].([]any)
	require.Len(t, rows, 1)
	require.Equal(t, "PNC-9102", rows[0].(map[string]any)["id"])
	require.Equal(t, float64(2), content["paginasi"].(map[string]any)["total_halaman"])
}

func TestListMapsDomainErrors(t *testing.T) {
	t.Run("tab not allowed", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{})
		response, raw := f.get(t, "/inbox-manager-admin?tab=3", "ASM")
		require.Equal(t, http.StatusForbidden, response.StatusCode)
		require.Equal(t, inboxmanageradminhttp.CodeTabNotAllowed, decode(t, raw)["kode"])
	})
	t.Run("no tab allowed", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{})
		f.caller.Login = "tanpa.lini"
		response, raw := f.get(t, "/inbox-manager-admin", "ASM")
		require.Equal(t, http.StatusForbidden, response.StatusCode)
		content := decode(t, raw)
		require.Equal(t, inboxmanageradminhttp.CodeNoTabAllowed, content["kode"])
		require.Contains(t, content["pesan"], strings.Join(inboxmanageradmin.ExpectedLineBusinesses(), ", "))
	})
	t.Run("unknown tab", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{})
		*f.caller = inboxmanageradminhttp.Caller{Login: "dev", OrgUnit: inboxmanageradmin.DevelopmentOrgUnit}
		response, raw := f.get(t, "/inbox-manager-admin?tab=9", "ASM")
		require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
		content := decode(t, raw)
		require.Equal(t, inboxmanageradminhttp.CodeValidationFail, content["kode"])
		require.Equal(t, []any{map[string]any{"field": "tab", "pesan": "Tab tidak dikenal."}}, content["detail"])
	})
	t.Run("source column missing", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{
			repos: map[string]inboxmanageradmin.Repo{"SMI": missingColumnRepo{}}, readyExtras: []string{"SMI"},
		})
		response, raw := f.get(t, "/inbox-manager-admin", "SMI")
		require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
		require.Equal(t, inboxmanageradminhttp.CodeSourceColumnMissing, decode(t, raw)["kode"])
		require.Contains(t, f.logs.String(), "permintaan gagal")
	})
	t.Run("unrecognised error goes to fallback", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{})
		for _, path := range []string{"/inbox-manager-admin/tab", "/inbox-manager-admin", "/inbox-manager-admin/ekspor"} {
			response, raw := f.get(t, path, "ASI")
			require.Equal(t, http.StatusInternalServerError, response.StatusCode, path)
			require.Equal(t, "cadangan", decode(t, raw)["kode"], path)
		}
	})
}

type missingColumnRepo struct{}

func (missingColumnRepo) List(context.Context, inboxmanageradmin.Query) ([]inboxmanageradmin.WorkItem, error) {
	return nil, fmt.Errorf("ORA-00904 (%w)", inboxmanageradmin.ErrSourceColumnMissing)
}

func TestExportWritesLegacyHeaderAndRows(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	response, raw := f.get(t, "/inbox-manager-admin/ekspor?tab=2", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/csv; charset=utf-8", response.Header.Get("Content-Type"))
	require.Equal(t, `attachment; filename="Data AdminPA.csv"`, response.Header.Get("Content-Disposition"))
	require.Equal(t, "no-store", response.Header.Get("Cache-Control"))

	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{
		"ID", "No Polis", "Nama Tertanggung", "Nama Bisnis", "Nama Sumber Bisnis",
		"Status Klaim", "Tanggal Pendaftaran", "AdminPNC",
	}, records[0])
	require.Len(t, records, 3)
	require.Equal(t, []string{
		"PNC-9101", "00.002.2026.00011", "Tertanggung Contoh A", "Personal Accident",
		"Keagenan Contoh", inboxmanageradmin.DisplayStatusFor("New"), "2026-09-25", "Admin Contoh Dua",
	}, records[1])
}

func TestExportOfEmptyDateLeavesCellBlank(t *testing.T) {
	f := newFixture(t, fixtureOptions{})
	*f.caller = inboxmanageradminhttp.Caller{Login: "dev", OrgUnit: inboxmanageradmin.DevelopmentOrgUnit}

	response, raw := f.get(t, "/inbox-manager-admin/ekspor?tab=1", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, `attachment; filename="Data AdminPNC.csv"`, response.Header.Get("Content-Disposition"))
	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)
	// PNC-9002 tanggal pendaftarannya kosong dan berada di baris terakhir.
	require.Equal(t, "PNC-9002", records[2][0])
	require.Equal(t, "", records[2][6])
}

func TestExportRejectsForbiddenTabBeforeWritingFile(t *testing.T) {
	f := newFixture(t, fixtureOptions{})

	response, raw := f.get(t, "/inbox-manager-admin/ekspor?tab=1", "ASM")
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.NotEqual(t, "text/csv; charset=utf-8", response.Header.Get("Content-Type"))
	require.Equal(t, inboxmanageradminhttp.CodeTabNotAllowed, decode(t, raw)["kode"])
}

func TestExportStreamsSeveralChunks(t *testing.T) {
	rows := numbered(inboxmanageradmin.MaxPageSize + 50)
	f := newFixture(t, fixtureOptions{
		repos: map[string]inboxmanageradmin.Repo{"SMI": staticRepo{rows: rows}}, readyExtras: []string{"SMI"},
	})

	response, raw := f.get(t, "/inbox-manager-admin/ekspor", "SMI")
	require.Equal(t, http.StatusOK, response.StatusCode)
	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+len(rows))
	require.Equal(t, "PNC-00000", records[1][0])
	require.Equal(t, fmt.Sprintf("PNC-%05d", len(rows)-1), records[len(records)-1][0])
}

func TestExportLogsFailureOfLaterChunk(t *testing.T) {
	repo := &flakyRepo{rows: numbered(inboxmanageradmin.MaxPageSize + 1)}
	f := newFixture(t, fixtureOptions{
		repos: map[string]inboxmanageradmin.Repo{"SMI": repo}, readyExtras: []string{"SMI"},
	})

	response, raw := f.get(t, "/inbox-manager-admin/ekspor?tab=2", "SMI")
	// Header sudah terkirim, sehingga berkasnya berhenti di tengah dan galatnya hanya masuk log.
	require.Equal(t, http.StatusOK, response.StatusCode)
	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+inboxmanageradmin.MaxPageSize)
	require.Contains(t, f.logs.String(), "ekspor Inbox Manager Admin terputus")
	require.Contains(t, f.logs.String(), "tab=2")
}

func TestExportTruncatesAtLimitWithNotice(t *testing.T) {
	const limit = 50_000
	rows := numbered(limit + 1)
	f := newFixture(t, fixtureOptions{
		repos: map[string]inboxmanageradmin.Repo{"SMI": staticRepo{rows: rows}}, readyExtras: []string{"SMI"},
	})

	response, raw := f.get(t, "/inbox-manager-admin/ekspor", "SMI")
	require.Equal(t, http.StatusOK, response.StatusCode)
	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+limit+1)
	notice := records[len(records)-1]
	require.Equal(t, fmt.Sprintf(
		"-- Berkas dipotong pada %d baris dari %d baris yang cocok. "+
			"Persempit tab atau minta bantuan tim teknis. --", limit, limit+1), notice[0])
	require.Equal(t, []string{"", "", "", "", "", "", ""}, notice[1:])
}

// failingWriter menolak setiap penulisan badan, meniru peramban yang memutus unduhan.
type failingWriter struct {
	header http.Header
	status int
}

func (w *failingWriter) Header() http.Header         { return w.header }
func (w *failingWriter) WriteHeader(status int)      { w.status = status }
func (w *failingWriter) Write([]byte) (int, error)   { return 0, errStore }
func (w *failingWriter) written() (int, http.Header) { return w.status, w.header }

func exportThroughFailingWriter(t *testing.T, f *fixture, path, alias string) *failingWriter {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set(portalhttp.HeaderPortal, alias)
	writer := &failingWriter{header: http.Header{}}
	f.server.Config.Handler.ServeHTTP(writer, request)
	return writer
}

func TestExportLogsWriterFailures(t *testing.T) {
	t.Run("small file fails on flush", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{})
		writer := exportThroughFailingWriter(t, f, "/inbox-manager-admin/ekspor?tab=2", "ASM")
		_, header := writer.written()
		require.Equal(t, "text/csv; charset=utf-8", header.Get("Content-Type"))
		require.Contains(t, f.logs.String(), "ekspor Inbox Manager Admin terputus")
	})
	t.Run("large chunk fails while writing rows", func(t *testing.T) {
		rows := numbered(inboxmanageradmin.MaxPageSize)
		for i := range rows {
			rows[i].InsuredName = strings.Repeat("x", 80)
		}
		f := newFixture(t, fixtureOptions{
			repos: map[string]inboxmanageradmin.Repo{"SMI": staticRepo{rows: rows}}, readyExtras: []string{"SMI"},
		})
		exportThroughFailingWriter(t, f, "/inbox-manager-admin/ekspor", "SMI")
		require.Contains(t, f.logs.String(), "ekspor Inbox Manager Admin terputus")
	})
	t.Run("without logger nothing panics", func(t *testing.T) {
		f := newFixture(t, fixtureOptions{nilLogger: true})
		writer := exportThroughFailingWriter(t, f, "/inbox-manager-admin/ekspor?tab=2", "ASM")
		_, header := writer.written()
		require.Equal(t, "no-store", header.Get("Cache-Control"))
		require.Empty(t, f.logs.String())
	})
}

func TestWriteErrorWithoutFallback(t *testing.T) {
	var recorded struct {
		status int
		body   any
	}
	writeJSON := func(_ http.ResponseWriter, _ *http.Request, status int, body any) {
		recorded.status, recorded.body = status, body
	}
	request := httptest.NewRequest(http.MethodGet, "/x", nil)

	var logs bytes.Buffer
	inboxmanageradminhttp.WriteError(slog.New(slog.NewTextHandler(&logs, nil)), writeJSON, nil)(
		httptest.NewRecorder(), request, errStore)
	require.Equal(t, http.StatusInternalServerError, recorded.status)
	require.Equal(t, inboxmanageradminhttp.ErrorResponse{
		Code: inboxmanageradminhttp.CodeInternalError, Message: "Terjadi kesalahan pada sistem.",
	}, recorded.body)
	require.Contains(t, logs.String(), errStore.Error())

	// Tanpa logger pun jawabannya tetap 500.
	inboxmanageradminhttp.WriteError(nil, writeJSON, nil)(httptest.NewRecorder(), request, errStore)
	require.Equal(t, http.StatusInternalServerError, recorded.status)

	// Galat milik modul yang di bawah 500 tidak dicatat.
	logs.Reset()
	inboxmanageradminhttp.WriteError(slog.New(slog.NewTextHandler(&logs, nil)), writeJSON, nil)(
		httptest.NewRecorder(), request, inboxmanageradmin.ErrTabNotAllowed)
	require.Equal(t, http.StatusForbidden, recorded.status)
	require.Empty(t, logs.String())
}
