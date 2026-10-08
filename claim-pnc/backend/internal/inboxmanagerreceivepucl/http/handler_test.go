package inboxmanagerreceivepuclhttp

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
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
	"claim-pnc/internal/inboxmanagerreceivepucl/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// writeTestJSON menuliskan badan JSON apa adanya.
func writeTestJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// scriptedRepo mengembalikan halaman menurut urutan pemanggilan, lalu galat bila habis.
type scriptedRepo struct {
	pages []inboxmanagerreceivepucl.Page
	err   error
	calls int
}

func (s *scriptedRepo) List(
	context.Context, inboxmanagerreceivepucl.Query, inboxmanagerreceivepucl.Pagination,
) (inboxmanagerreceivepucl.Page, error) {
	s.calls++
	if s.calls <= len(s.pages) {
		return s.pages[s.calls-1], nil
	}
	return inboxmanagerreceivepucl.Page{}, s.err
}

func (s *scriptedRepo) Document(context.Context, string) (inboxmanagerreceivepucl.ReceiveDocument, error) {
	return inboxmanagerreceivepucl.ReceiveDocument{}, s.err
}

type fixture struct {
	router chi.Router
	logs   *bytes.Buffer
}

func newFixture(t *testing.T, repo inboxmanagerreceivepucl.Repo, login string) *fixture {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanagerreceivepucl.Repo, error) {
			if alias == "ASM" {
				return repo, nil
			}
			return nil, portal.ErrNotReady
		},
		Logger: logger,
	})
	require.NoError(t, err)

	fallback := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeTestJSON(w, r, http.StatusTeapot, ErrorResponse{Code: "cadangan"})
		},
		writeTestJSON,
	)

	h := NewHandler(Options{
		Service: svc,
		GetCaller: func(context.Context) (Caller, bool) {
			return Caller{Login: login}, login != ""
		},
		Logger:              logger,
		WriteJSON:           writeTestJSON,
		FallbackErrorWriter: ErrorWriter(fallback),
	})

	router := chi.NewRouter()
	Mount(router, h, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   fallback,
	})
	return &fixture{router: router, logs: logs}
}

func (f *fixture) do(t *testing.T, method, target, alias string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if alias != "" {
		req.Header.Set(portalhttp.HeaderPortal, alias)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestMetadataReturnsTabsWithPortalAlias(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/tab", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, "ASM", body["portal"])
	require.Equal(t, inboxmanagerreceivepucl.DefaultTab, body["tab_bawaan"])
	require.Len(t, body["tab"], len(inboxmanagerreceivepucl.Tabs()))
	require.NotContains(t, body, "selisih_terencana")
}

func TestRoutesRejectMissingPortal(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, portalhttp.CodeNotStated, decode(t, rec)["kode"])
}

func TestListReturnsRowsAndPagination(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl?tab=2&halaman=abc&ukuran=-3", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, "2", body["tab"].(map[string]any)["kode"])
	pagination := body["paginasi"].(map[string]any)
	require.EqualValues(t, 1, pagination["halaman"])
	require.EqualValues(t, inboxmanagerreceivepucl.DefaultPageSize, pagination["ukuran"])
	rows := body["baris"].([]any)
	require.NotEmpty(t, rows)
	require.Contains(t, rows[0].(map[string]any)["no_case"], "PNC-")
}

func TestListRejectsUnknownTabWithValidationError(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl?tab=9", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := decode(t, rec)
	require.Equal(t, CodeValidationFail, body["kode"])
	require.Equal(t, inboxmanagerreceivepucl.FieldTab,
		body["detail"].([]any)[0].(map[string]any)["field"])
}

func TestListRejectsUnknownCaller(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl", "ASM")
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, CodeCallerUnknown, decode(t, rec)["kode"])
}

func TestListPortalNotReadyGoesToFallback(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	// ASI lolos middleware tetapi selector menjawab "belum siap".
	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl", "ASI")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, portalhttp.CodeNotReady, decode(t, rec)["kode"])
}

func TestListUnknownErrorUsesFallbackWriter(t *testing.T) {
	f := newFixture(t, &scriptedRepo{err: errors.New("boom")}, "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl", "ASM")
	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Equal(t, "cadangan", decode(t, rec)["kode"])
}

func TestDocumentReturnsValuesGroupsAndActions(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet,
		"/inbox-manager-receive-pucl/dokumen/ASM-FW-GCNMFW-WORK%20RCV-900001", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, "RCV-900001", body["no_case"])
	require.Equal(t, "ASM", body["portal"])
	require.NotEmpty(t, body["kelompok"])
	require.NotEmpty(t, body["tindakan"])
	require.NotEmpty(t, body["nilai"])
	require.Contains(t, f.logs.String(), "layar kerja penerimaan dokumen dibuka")
}

func TestDocumentNotFoundAndBlankReference(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/dokumen/TIDAK-ADA", "ASM")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, CodeDocumentNotFound, decode(t, rec)["kode"])

	rec = f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/dokumen/%20%20", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, CodeReferenceRequired, decode(t, rec)["kode"])
}

func TestDocumentRejectsUnknownCaller(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/dokumen/K", "ASM")
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestRejectWriteAnswersNotImplementedAndLogsAction(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodPost,
		"/inbox-manager-receive-pucl/tindakan?tindakan=CetakPUCL&berkas=PNC-1", "ASM")
	require.Equal(t, http.StatusNotImplemented, rec.Code)
	require.Equal(t, CodeWriteNotAvailable, decode(t, rec)["kode"])
	require.Contains(t, f.logs.String(), `"tindakan":"CetakPUCL"`)
	require.Contains(t, f.logs.String(), `"berkas":"PNC-1"`)
}

func TestHandlersWithoutPortalContextAnswerNotStated(t *testing.T) {
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxmanagerreceivepucl.Repo, error) {
			return memory.NewSampleStore(), nil
		},
	})
	require.NoError(t, err)
	// Tanpa cadangan: galat portal bukan milik modul ini sehingga menjadi 500 umum.
	h := NewHandler(Options{Service: svc, WriteJSON: writeTestJSON})

	for name, handle := range map[string]http.HandlerFunc{
		"metadata": h.Metadata, "list": h.List, "document": h.Document,
		"export": h.Export, "reject": h.RejectWrite,
	} {
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code, name)
		require.Equal(t, CodeInternalError, decode(t, rec)["kode"], name)
	}
}

func TestReadCallerWithoutReaderIsUnknown(t *testing.T) {
	h := &Handler{}
	_, known := h.readCaller(httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, known)

	h.caller = func(context.Context) (Caller, bool) { return Caller{Login: "   "}, true }
	_, known = h.readCaller(httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, known)
}

func TestExportWritesCSVOfCurrentTab(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor?tab=1", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Header().Get("Content-Disposition"), `filename="receive.csv"`)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	records, err := csv.NewReader(rec.Body).ReadAll()
	require.NoError(t, err)
	tab, _ := inboxmanagerreceivepucl.FindTab(inboxmanagerreceivepucl.TabReceive)
	require.Equal(t, exportHeader(tab), records[0])
	require.Greater(t, len(records), 1)
	require.Contains(t, records[1][0], "RCV-")
}

func TestExportRCLPUCLFilename(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor?tab=2", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Disposition"), `filename="rcl-pucl.csv"`)
}

func TestExportFirstPageErrorIsAnsweredAsJSON(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor?tab=7", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, CodeValidationFail, decode(t, rec)["kode"])
}

func TestExportPagesUntilTotalAndStopsOnLaterError(t *testing.T) {
	item := inboxmanagerreceivepucl.WorkItem{CaseID: "RCV-1"}
	repo := &scriptedRepo{
		pages: []inboxmanagerreceivepucl.Page{
			{Items: []inboxmanagerreceivepucl.WorkItem{item}, Total: 3},
			{Items: []inboxmanagerreceivepucl.WorkItem{item}, Total: 3},
		},
		err: errors.New("halaman ketiga gagal"),
	}
	f := newFixture(t, repo, "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 3, repo.calls)

	records, err := csv.NewReader(rec.Body).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3, "judul + dua baris yang sempat terbaca")
	require.Contains(t, f.logs.String(), "ekspor inbox manager receive/PUCL terputus")
}

func TestExportStopsWhenPageIsEmpty(t *testing.T) {
	repo := &scriptedRepo{pages: []inboxmanagerreceivepucl.Page{{Total: 10}}}
	f := newFixture(t, repo, "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, repo.calls)
	records, err := csv.NewReader(rec.Body).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1)
}

func TestExportMarksTruncationAtLimit(t *testing.T) {
	items := make([]inboxmanagerreceivepucl.WorkItem, exportLimit+1)
	for i := range items {
		items[i] = inboxmanagerreceivepucl.WorkItem{CaseID: fmt.Sprintf("RCV-%d", i)}
	}
	// Satu halaman raksasa cukup untuk menyentuh batas pada potongan pertama.
	repo := &scriptedRepo{pages: []inboxmanagerreceivepucl.Page{{Items: items, Total: len(items)}}}
	f := newFixture(t, repo, "penyelia")

	rec := f.do(t, http.MethodGet, "/inbox-manager-receive-pucl/ekspor", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	records, err := csv.NewReader(rec.Body).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, exportLimit+2)
	require.Equal(t,
		fmt.Sprintf("-- Terpotong pada %d baris dari %d yang cocok. Persempit penyaringnya. --",
			exportLimit, exportLimit+1),
		records[len(records)-1][0])
}

// failingWriter menolak setiap penulisan badan.
type failingWriter struct {
	header http.Header
	status int
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(status int)    { f.status = status }
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("klien terputus") }

func TestExportLogsFailureWhenClientDisconnects(t *testing.T) {
	f := newFixture(t, memory.NewSampleStore(), "penyelia")

	req := httptest.NewRequest(http.MethodGet, "/inbox-manager-receive-pucl/ekspor", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	w := &failingWriter{header: http.Header{}}
	f.router.ServeHTTP(w, req)

	require.Contains(t, f.logs.String(), "ekspor inbox manager receive/PUCL terputus")
	require.Contains(t, f.logs.String(), "klien terputus")
}

func TestExportHelpers(t *testing.T) {
	require.Equal(t, "inbox-manager-receive-pucl.csv",
		exportFilename(inboxmanagerreceivepucl.Tab{Code: "9"}))
	require.Empty(t, exportTruncationNotice(0, 5))
	require.Len(t, exportTruncationNotice(3, 5), 3)

	item := inboxmanagerreceivepucl.WorkItem{
		CaseID: "a", PolicyNumber: "b", ClaimNumber: "c", InsuredName: "d", LossDate: "e",
		ClaimType: "f", SenderName: "g", DocumentReceivedDate: "h", DocumentSheetCount: "i",
		InboxEntryAt: "j", AnalystNote: "k", Track: "l", TrackStatus: "m",
		LetterPrintedAt: "n", ClaimAge: "o", ExpiryStatus: "p",
	}
	keys := []string{
		inboxmanagerreceivepucl.FieldCaseID, inboxmanagerreceivepucl.FieldPolicyNumber,
		inboxmanagerreceivepucl.FieldClaimNumber, inboxmanagerreceivepucl.FieldInsuredName,
		inboxmanagerreceivepucl.FieldLossDate, inboxmanagerreceivepucl.FieldClaimType,
		inboxmanagerreceivepucl.FieldSenderName, inboxmanagerreceivepucl.FieldDocumentReceivedDate,
		inboxmanagerreceivepucl.FieldDocumentSheetCount, inboxmanagerreceivepucl.FieldInboxEntryAt,
		inboxmanagerreceivepucl.FieldAnalystNote, inboxmanagerreceivepucl.FieldTrack,
		inboxmanagerreceivepucl.FieldTrackStatus, inboxmanagerreceivepucl.FieldLetterPrintedAt,
		inboxmanagerreceivepucl.FieldClaimAge, inboxmanagerreceivepucl.FieldExpiryStatus,
	}
	var got []string
	for _, key := range keys {
		got = append(got, cellValue(item, key))
	}
	require.Equal(t, strings.Split("a b c d e f g h i j k l m n o p", " "), got)
	require.Empty(t, cellValue(item, "tidak-dikenal"))
}

func TestPositiveNumber(t *testing.T) {
	require.Equal(t, 7, positiveNumber("7"))
	require.Equal(t, 0, positiveNumber("-1"))
	require.Equal(t, 0, positiveNumber("x"))
}

func TestWriteErrorWithoutFallbackLogsInternalError(t *testing.T) {
	var logs bytes.Buffer
	write := WriteError(slog.New(slog.NewJSONHandler(&logs, nil)), writeTestJSON, nil)

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/jalur", nil), errors.New("rahasia internal"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "rahasia internal")
	require.Contains(t, logs.String(), "rahasia internal")
}
