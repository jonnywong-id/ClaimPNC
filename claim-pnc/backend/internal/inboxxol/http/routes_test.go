package inboxxolhttp_test

import (
	"bytes"
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

	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/inboxxol/repo/memory"
	"claim-pnc/internal/inboxxol/usecase"
	"claim-pnc/internal/portal"

	inboxxolhttp "claim-pnc/internal/inboxxol/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

var errBroken = errors.New("penyimpanan rusak")

// brokenRepo adalah penyimpanan yang setiap operasinya gagal — untuk jalur galat handler.
type brokenRepo struct{}

func (brokenRepo) ListMasterXOL(context.Context) ([]inboxxol.MasterXOL, error) {
	return nil, errBroken
}
func (brokenRepo) SummarizeClaims(context.Context, inboxxol.ClaimFilter) ([]inboxxol.ClaimSummary, error) {
	return nil, errBroken
}
func (brokenRepo) BreakdownByBusiness(context.Context, inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	return nil, errBroken
}
func (brokenRepo) BreakdownTreatyInward(context.Context, inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	return nil, errBroken
}
func (brokenRepo) SearchAdvice(context.Context, inboxxol.AdviceFilter) ([]inboxxol.Advice, error) {
	return nil, errBroken
}
func (brokenRepo) ListPendingAdviceApproval(context.Context) ([]inboxxol.ApprovalItem, error) {
	return nil, errBroken
}
func (brokenRepo) ListPendingMasterApproval(context.Context) ([]inboxxol.MasterXOL, error) {
	return nil, errBroken
}
func (brokenRepo) ListCauseOfLoss(context.Context) ([]inboxxol.CauseOfLoss, error) {
	return nil, errBroken
}

func (brokenRepo) SummarizeBusiness(context.Context, inboxxol.SummaryFilter) ([]inboxxol.SummaryBusiness, error) {
	return nil, errBroken
}

type serverOptions struct {
	caller   inboxxolhttp.CallerReader
	fallback bool
	logger   *slog.Logger
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func newServer(t *testing.T, options serverOptions) http.Handler {
	t.Helper()
	asm := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxxol.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return brokenRepo{}, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	logger := options.logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "cadangan"})
		},
		writeJSON,
	)
	var fallback inboxxolhttp.ErrorWriter
	if options.fallback {
		fallback = inboxxolhttp.ErrorWriterFrom(portalError)
	}

	handler := inboxxolhttp.NewHandler(inboxxolhttp.Options{
		Service:             service,
		GetCaller:           options.caller,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: fallback,
	})
	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   portalError,
	}
	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) { inboxxolhttp.Mount(api, handler, portalDeps) })
	return router
}

func knownCaller(context.Context) (inboxxolhttp.Caller, bool) {
	return inboxxolhttp.Caller{Login: " PICTEKNIK01 "}, true
}

func call(t *testing.T, handler http.Handler, method, path, alias string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	if alias != "" {
		request.Header.Set(portalhttp.HeaderPortal, alias)
	}
	record := httptest.NewRecorder()
	handler.ServeHTTP(record, request)
	return record
}

func decode(t *testing.T, record *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
	return body
}

func TestListMasters(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet, "/api/inbox-xol/perjanjian", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	masters := decode(t, record)["perjanjian"].([]any)
	require.Len(t, masters, 3)
	require.Equal(t, "XOL-001", masters[0].(map[string]any)["id"])
}

func TestSummarizeClaimsDividesByRate(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet, "/api/inbox-xol/klaim?id_master=XOL-001", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	body := decode(t, record)
	require.Equal(t, "XOL-001", body["perjanjian"].(map[string]any)["id"])
	rows := body["baris"].([]any)
	require.Len(t, rows, 2)
	require.InDelta(t, 4_650_000_000.0/15500, rows[0].(map[string]any)["nilai_outstanding"], 0.001)
}

// Tanpa `id_master`, grid mengakumulasi SELURUH perjanjian — bukan menolak permintaan.
//
// Itu perilaku `GetClaimXOL` step 4, yang me-loop `MstXOL.pxResults` dan meng-APPEND
// hasil tiap perjanjian ke satu daftar.
func TestSummarizeClaimsTanpaMasterMengakumulasiSeluruhnya(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})

	record := call(t, server, http.MethodGet, "/api/inbox-xol/klaim", "ASM")
	require.Equal(t, http.StatusOK, record.Code)

	rows := decode(t, record)["baris"].([]any)
	require.NotEmpty(t, rows)
	// Tiap baris menyebut perjanjian asalnya, supaya rincian di baliknya dapat dibuka.
	require.NotEmpty(t, rows[0].(map[string]any)["id_master"])
}

func TestSummarizeClaimsNotFound(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})

	record := call(t, server, http.MethodGet, "/api/inbox-xol/klaim?id_master=TIDAK-ADA", "ASM")
	require.Equal(t, http.StatusNotFound, record.Code)
	require.Equal(t, inboxxolhttp.CodeMasterNotFound, decode(t, record)["kode"])
}

func TestBreakdown(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet,
		"/api/inbox-xol/klaim/rincian?id_master=XOL-001&tanggal_kejadian=12/03/2024&sebab_kerugian=BANJIR", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	rows := decode(t, record)["baris"].([]any)
	require.NotEmpty(t, rows)
	last := rows[len(rows)-1].(map[string]any)
	require.Equal(t, string(inboxxol.SourceTreatyInward), last["sumber"])

	record = call(t, server, http.MethodGet, "/api/inbox-xol/klaim/rincian", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, record.Code)
	require.Len(t, decode(t, record)["detail"].([]any), 3)
}

func TestSearchAdvice(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet, "/api/inbox-xol/pla-dla?tahun=2024&sebab_kerugian=BANJIR&tipe=PLA", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	advices := decode(t, record)["pemberitahuan"].([]any)
	require.Len(t, advices, 2)

	record = call(t, server, http.MethodGet, "/api/inbox-xol/pla-dla?tipe=XX", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, record.Code)
	require.Len(t, decode(t, record)["detail"].([]any), 3)
}

func TestDownloadAdvice(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet,
		"/api/inbox-xol/pla-dla/unduh?tahun=2024&sebab_kerugian=BANJIR%22x&tipe=PLA", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	require.Equal(t, "text/csv; charset=utf-8", record.Header().Get("Content-Type"))
	require.Equal(t, "no-store", record.Header().Get("Cache-Control"))
	// Kutip ganda di nilai penyaring diganti garis bawah, tidak menyisip ke header.
	require.Equal(t, `attachment; filename="perhitungan-pla-2024-BANJIR_x.csv"`,
		record.Header().Get("Content-Disposition"))
	lines := strings.Split(strings.TrimSpace(record.Body.String()), "\n")
	require.Len(t, lines, 1, "penyaring sebab tidak cocok — hanya judul kolom")
	require.True(t, strings.HasPrefix(lines[0], "NO PLA / DLA,Tipe,"))

	record = call(t, server, http.MethodGet,
		"/api/inbox-xol/pla-dla/unduh?tahun=2024&sebab_kerugian=BANJIR&tipe=PLA", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	lines = strings.Split(strings.TrimSpace(record.Body.String()), "\n")
	require.Len(t, lines, 3)
	require.Contains(t, lines[2], "PLA/XOL/2024/0002 / 1,PLA,Reasuransi Contoh Kedua")
	require.Contains(t, lines[1], ",15500,35.5,")

	record = call(t, server, http.MethodGet, "/api/inbox-xol/pla-dla/unduh?tipe=PLA", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, record.Code)
}

// failingWriter menolak setiap penulisan badan — meniru koneksi yang putus di tengah unduhan.
type failingWriter struct {
	header http.Header
	status int
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(status int)    { f.status = status }
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("koneksi putus") }

func TestDownloadAdviceLogsBrokenConnection(t *testing.T) {
	// Isi banyak baris supaya penyangga csv melampaui kapasitasnya dan galat tulis terlihat.
	advices := make([]inboxxol.Advice, 0, 200)
	for i := 0; i < 200; i++ {
		advices = append(advices, inboxxol.Advice{Number: strings.Repeat("N", 40), Type: inboxxol.AdvicePLA,
			Year: "2024", CauseOfLoss: "BANJIR", ReinsurerName: strings.Repeat("R", 40)})
	}
	repo := memory.NewRepo(memory.WithAdvices(advices...))
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxxol.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	var logs bytes.Buffer
	handler := inboxxolhttp.NewHandler(inboxxolhttp.Options{
		Service:   service,
		GetCaller: knownCaller,
		Logger:    slog.New(slog.NewTextHandler(&logs, nil)),
		WriteJSON: writeJSON,
	})
	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM"} },
		WriteError:   func(w http.ResponseWriter, r *http.Request, err error) { writeJSON(w, r, 500, nil) },
	}
	router := chi.NewRouter()
	inboxxolhttp.Mount(router, handler, portalDeps)

	request := httptest.NewRequest(http.MethodGet, "/inbox-xol/pla-dla/unduh?tahun=2024&sebab_kerugian=BANJIR&tipe=PLA", nil)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	writer := &failingWriter{header: http.Header{}}
	router.ServeHTTP(writer, request)

	require.Contains(t, logs.String(), "unduhan perhitungan XOL terputus")
	require.Contains(t, logs.String(), "koneksi putus")
}

func TestListApprovalsAndCauseOfLoss(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})

	record := call(t, server, http.MethodGet, "/api/inbox-xol/persetujuan", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	body := decode(t, record)
	require.Len(t, body["pemberitahuan"].([]any), 2)
	require.Len(t, body["perjanjian"].([]any), 1)

	record = call(t, server, http.MethodGet, "/api/inbox-xol/sebab-kerugian", "ASM")
	require.Equal(t, http.StatusOK, record.Code)
	require.Len(t, decode(t, record)["sebab_kerugian"].([]any), 4)
}

func TestWriteRoutesAnswerWithReason(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	for _, path := range []string{"/api/inbox-xol/dol-col", "/api/inbox-xol/persetujuan", "/api/inbox-xol/pengajuan-komite"} {
		record := call(t, server, http.MethodPost, path, "ASM")
		require.Equal(t, http.StatusConflict, record.Code, path)
		require.Equal(t, inboxxolhttp.CodeWriteNotAllowed, decode(t, record)["kode"])
	}
}

func TestCallerMustBeKnown(t *testing.T) {
	for name, reader := range map[string]inboxxolhttp.CallerReader{
		"nil reader":  nil,
		"not known":   func(context.Context) (inboxxolhttp.Caller, bool) { return inboxxolhttp.Caller{}, false },
		"empty login": func(context.Context) (inboxxolhttp.Caller, bool) { return inboxxolhttp.Caller{Login: " "}, true },
	} {
		t.Run(name, func(t *testing.T) {
			server := newServer(t, serverOptions{caller: reader, fallback: true})
			record := call(t, server, http.MethodGet, "/api/inbox-xol/perjanjian", "ASM")
			require.Equal(t, http.StatusConflict, record.Code)
			require.Equal(t, inboxxolhttp.CodeCallerUnknown, decode(t, record)["kode"])
		})
	}
}

func TestPortalIsRequired(t *testing.T) {
	server := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	record := call(t, server, http.MethodGet, "/api/inbox-xol/perjanjian", "")
	require.NotEqual(t, http.StatusOK, record.Code)
	require.GreaterOrEqual(t, record.Code, 400)
}

func TestRepoFailuresGoToFallbackOrInternalError(t *testing.T) {
	paths := []string{
		"/api/inbox-xol/perjanjian",
		"/api/inbox-xol/klaim?id_master=X",
		"/api/inbox-xol/klaim/rincian?id_master=X&tanggal_kejadian=a&sebab_kerugian=b",
		"/api/inbox-xol/pla-dla?tahun=2024&sebab_kerugian=B&tipe=PLA",
		"/api/inbox-xol/pla-dla/unduh?tahun=2024&sebab_kerugian=B&tipe=DLA",
		"/api/inbox-xol/persetujuan",
		"/api/inbox-xol/sebab-kerugian",
	}
	withFallback := newServer(t, serverOptions{caller: knownCaller, fallback: true})
	var logs bytes.Buffer
	withoutFallback := newServer(t, serverOptions{caller: knownCaller,
		logger: slog.New(slog.NewTextHandler(&logs, nil))})
	for _, path := range paths {
		record := call(t, withFallback, http.MethodGet, path, "ASI")
		require.Equal(t, http.StatusInternalServerError, record.Code, path)
		require.Equal(t, "cadangan", decode(t, record)["kode"], path)

		record = call(t, withoutFallback, http.MethodGet, path, "ASI")
		require.Equal(t, http.StatusInternalServerError, record.Code, path)
		body := decode(t, record)
		require.Equal(t, inboxxolhttp.CodeInternalError, body["kode"], path)
		require.NotContains(t, record.Body.String(), errBroken.Error(), "rincian galat tidak dikirim ke peramban")
	}
	require.Contains(t, logs.String(), "permintaan gagal")
}

func TestErrorWriterFromNil(t *testing.T) {
	require.Nil(t, inboxxolhttp.ErrorWriterFrom(nil))
}
