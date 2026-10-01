package inboxrclpuclhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
	"claim-pnc/internal/inboxrclpucl/usecase"

	rclpuclhttp "claim-pnc/internal/inboxrclpucl/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
const testPortal = "ASM"

// fakeRepo adalah pengisi seam yang perilakunya ditentukan tiap uji.
//
// Fungsi yang dibiarkan nil menghasilkan jawaban kosong yang sah.
type fakeRepo struct {
	list   func(q inboxrclpucl.Query, p inboxrclpucl.Pagination) (inboxrclpucl.Page, error)
	report func(rng inboxrclpucl.DateRange, p inboxrclpucl.Pagination) ([]inboxrclpucl.DailyReportRow, int, error)
	detail func(reference string) (inboxrclpucl.ClaimDetail, error)
}

func (f fakeRepo) List(
	_ context.Context, q inboxrclpucl.Query, p inboxrclpucl.Pagination,
) (inboxrclpucl.Page, error) {
	if f.list == nil {
		return inboxrclpucl.Page{Items: []inboxrclpucl.WorkItem{}, Pagination: p.Normalize()}, nil
	}
	return f.list(q, p)
}

func (f fakeRepo) DailyReport(
	_ context.Context, rng inboxrclpucl.DateRange, p inboxrclpucl.Pagination,
) ([]inboxrclpucl.DailyReportRow, int, error) {
	if f.report == nil {
		return []inboxrclpucl.DailyReportRow{}, 0, nil
	}
	return f.report(rng, p)
}

func (f fakeRepo) Detail(_ context.Context, reference string) (inboxrclpucl.ClaimDetail, error) {
	if f.detail == nil {
		return inboxrclpucl.ClaimDetail{}, inboxrclpucl.ErrClaimNotFound
	}
	return f.detail(reference)
}

// serverOptions menyetel perakitan server uji.
type serverOptions struct {
	repo      inboxrclpucl.Repo
	login     string
	noCaller  bool
	noLogger  bool
	logBuffer *bytes.Buffer
}

// writeJSON meniru penulis JSON bersama aplikasi.
func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// buildHandler merakit handler beserta penulis galat cadangan, persis seperti rantai di
// cmd/claimpnc: galat portal dipetakan modul portal, sisanya menjadi 500.
func buildHandler(t *testing.T, o serverOptions) (*rclpuclhttp.Handler, portalhttp.ErrorWriter, *slog.Logger) {
	t.Helper()

	if o.repo == nil {
		o.repo = memory.NewSampleStore()
	}
	if o.login == "" {
		o.login = "PETUGASCONTOH"
	}

	buf := o.logBuffer
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return o.repo, nil },
	})
	require.NoError(t, err)

	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode":  "galat_internal",
				"pesan": err.Error(),
			})
		},
		writeJSON,
	)

	var getCaller rclpuclhttp.CallerReader
	if !o.noCaller {
		login := o.login
		getCaller = func(context.Context) (rclpuclhttp.Caller, bool) {
			return rclpuclhttp.Caller{Login: login}, true
		}
	}

	handlerLogger := logger
	if o.noLogger {
		handlerLogger = nil
	}

	handler := rclpuclhttp.NewHandler(rclpuclhttp.Options{
		Service:             service,
		GetCaller:           getCaller,
		Logger:              handlerLogger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: rclpuclhttp.ErrorWriter(writeError),
	})
	return handler, writeError, logger
}

func testServer(t *testing.T, o serverOptions) http.Handler {
	t.Helper()

	handler, writeError, logger := buildHandler(t, o)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		rclpuclhttp.Mount(api, handler, portalDeps)
	})
	return router
}

// send mengirim permintaan BESERTA header portal.
func send(t *testing.T, server http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("X-Portal", testPortal)

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func errorCode(t *testing.T, res *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"kode"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	return body.Code
}

func readCSV(t *testing.T, body string) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(body))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	require.NoError(t, err)
	return records
}

// ---------------------------------------------------------------------------
// Keterangan layar
// ---------------------------------------------------------------------------

func TestMetadataDescribesTheTabsAndNamesThePortal(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl/tab")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Tab []struct {
			Kode  string `json:"kode"`
			Nama  string `json:"nama"`
			Kolom []struct {
				Kunci string `json:"kunci"`
				Judul string `json:"judul"`
			} `json:"kolom"`
			PunyaLaporan bool `json:"punya_laporan_rentang_tanggal"`
		} `json:"tab"`
		TabBawaan    string `json:"tab_bawaan"`
		KolomLaporan []struct {
			Kunci string `json:"kunci"`
		} `json:"kolom_laporan"`
		Selisih []struct {
			Ringkas string `json:"ringkas"`
			Rincian string `json:"rincian"`
		} `json:"selisih_terencana"`
		Portal string `json:"portal"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Len(t, body.Tab, 3)
	require.Equal(t, inboxrclpucl.TabCetakSurat, body.Tab[0].Kode)
	require.True(t, body.Tab[0].PunyaLaporan)
	require.False(t, body.Tab[1].PunyaLaporan)
	require.NotEmpty(t, body.Tab[0].Kolom)
	require.Equal(t, inboxrclpucl.DefaultTab, body.TabBawaan)
	require.Len(t, body.KolomLaporan, len(inboxrclpucl.DailyReportColumns))
	require.Len(t, body.Selisih, len(inboxrclpucl.PlannedDifferences))
	require.NotEmpty(t, body.Selisih[0].Rincian)
	require.Equal(t, testPortal, body.Portal)
}

func TestEveryRouteRejectsARequestWithoutAPortal(t *testing.T) {
	server := testServer(t, serverOptions{})

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/inbox-rcl-pucl/tab"},
		{http.MethodGet, "/api/inbox-rcl-pucl"},
		{http.MethodGet, "/api/inbox-rcl-pucl/klaim/KUNCI"},
		{http.MethodGet, "/api/inbox-rcl-pucl/ekspor"},
		{http.MethodPost, "/api/inbox-rcl-pucl/tindakan"},
	} {
		recorder := httptest.NewRecorder()
		// Sengaja TANPA header X-Portal.
		server.ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))

		require.Equal(t, http.StatusBadRequest, recorder.Code, route.path)
		require.Equal(t, "portal_tidak_disebut", errorCode(t, recorder), route.path)
	}
}

// Handler tetap memeriksa portal sendiri meski dipasang tanpa middleware — bila rakitan di
// cmd suatu saat keliru, permintaannya tidak jatuh ke portal bawaan.
func TestHandlersRejectAMissingPortalEvenWithoutTheMiddleware(t *testing.T) {
	handler, _, _ := buildHandler(t, serverOptions{})

	for name, serve := range map[string]http.HandlerFunc{
		"metadata": handler.Metadata,
		"list":     handler.List,
		"detail":   handler.Detail,
		"export":   handler.Export,
		"reject":   handler.RejectWrite,
	} {
		recorder := httptest.NewRecorder()
		serve(recorder, httptest.NewRequest(http.MethodGet, "/api/inbox-rcl-pucl", nil))

		require.Equal(t, http.StatusBadRequest, recorder.Code, name)
		require.Equal(t, "portal_tidak_disebut", errorCode(t, recorder), name)
	}
}

// ---------------------------------------------------------------------------
// Daftar
// ---------------------------------------------------------------------------

func TestListReturnsTheDefaultTabWithItsRows(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Tab struct {
			Kode string `json:"kode"`
		} `json:"tab"`
		Baris []struct {
			Referensi string `json:"referensi"`
			NoCase    string `json:"no_case"`
			NoPolis   string `json:"no_polis"`
		} `json:"baris"`
		Paginasi struct {
			Halaman      int `json:"halaman"`
			Ukuran       int `json:"ukuran"`
			Total        int `json:"total"`
			TotalHalaman int `json:"total_halaman"`
		} `json:"paginasi"`
		Portal string `json:"portal"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, inboxrclpucl.TabCetakSurat, body.Tab.Kode)
	require.Len(t, body.Baris, 2)
	require.ElementsMatch(t, []string{"PNC-700001", "PNC-700002"},
		[]string{body.Baris[0].NoCase, body.Baris[1].NoCase})
	require.NotEmpty(t, body.Baris[0].Referensi)
	require.Equal(t, 1, body.Paginasi.Halaman)
	require.Equal(t, inboxrclpucl.DefaultPageSize, body.Paginasi.Ukuran)
	require.Equal(t, 2, body.Paginasi.Total)
	require.Equal(t, 1, body.Paginasi.TotalHalaman)
	require.Equal(t, testPortal, body.Portal)
}

func TestListPassesThePaginationAndReadsUnreadableNumbersAsDefaults(t *testing.T) {
	var seen []inboxrclpucl.Pagination
	repo := fakeRepo{list: func(q inboxrclpucl.Query, p inboxrclpucl.Pagination) (inboxrclpucl.Page, error) {
		seen = append(seen, p)
		return inboxrclpucl.Page{Items: []inboxrclpucl.WorkItem{}, Pagination: p.Normalize()}, nil
	}}
	server := testServer(t, serverOptions{repo: repo})

	require.Equal(t, http.StatusOK,
		send(t, server, http.MethodGet, "/api/inbox-rcl-pucl?halaman=3&ukuran=20").Code)
	require.Equal(t, http.StatusOK,
		send(t, server, http.MethodGet, "/api/inbox-rcl-pucl?halaman=abc&ukuran=-5").Code)

	require.Equal(t, []inboxrclpucl.Pagination{{Page: 3, Size: 20}, {Page: 0, Size: 0}}, seen)
}

func TestListRejectsAnUnknownTab(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl?tab=9")
	require.Equal(t, http.StatusUnprocessableEntity, res.Code)

	var body struct {
		Kode   string `json:"kode"`
		Detail []struct {
			Field string `json:"field"`
			Pesan string `json:"pesan"`
		} `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, rclpuclhttp.CodeValidationFail, body.Kode)
	require.Len(t, body.Detail, 1)
	require.Equal(t, inboxrclpucl.FieldTab, body.Detail[0].Field)
}

func TestListRejectsAnUnreadableCaller(t *testing.T) {
	for name, o := range map[string]serverOptions{
		"tanpa pembaca": {noCaller: true},
		"login kosong":  {login: "   "},
	} {
		server := testServer(t, o)
		res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl")
		require.Equal(t, http.StatusConflict, res.Code, name)
		require.Equal(t, rclpuclhttp.CodeCallerUnknown, errorCode(t, res), name)
	}
}

func TestListRejectsACallerReaderThatReportsNoIdentity(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return memory.NewSampleStore(), nil },
	})
	require.NoError(t, err)

	handler := rclpuclhttp.NewHandler(rclpuclhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (rclpuclhttp.Caller, bool) {
			return rclpuclhttp.Caller{Login: "ADA"}, false
		},
		WriteJSON: writeJSON,
	})

	request := httptest.NewRequest(http.MethodGet, "/api/inbox-rcl-pucl", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(),
		portalmemory.SampleList()[0]))
	recorder := httptest.NewRecorder()
	handler.List(recorder, request)

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, rclpuclhttp.CodeCallerUnknown, errorCode(t, recorder))
}

func TestListAnswersARepoFailureThroughTheFallbackWriter(t *testing.T) {
	repo := fakeRepo{list: func(inboxrclpucl.Query, inboxrclpucl.Pagination) (inboxrclpucl.Page, error) {
		return inboxrclpucl.Page{}, errors.New("koneksi putus")
	}}
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl")
	require.Equal(t, http.StatusInternalServerError, res.Code)
	require.Equal(t, "galat_internal", errorCode(t, res))
}

// ---------------------------------------------------------------------------
// Layar kerja
// ---------------------------------------------------------------------------

func TestDetailReturnsTheWorkScreenOfOneClaim(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/klaim/ASM-FW-GCNMFW-WORK%20PNC-700001")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Referensi     string `json:"referensi"`
		NoCase        string `json:"no_case"`
		LampiranSurat struct {
			NoPolis     string `json:"no_polis"`
			NamaPeserta string `json:"nama_peserta"`
			UP          string `json:"up"`
		} `json:"lampiran_surat"`
		BelumTerpetakan []string `json:"isian_belum_terpetakan"`
		MasihDiPega     bool     `json:"tindakan_masih_di_pega"`
		Portal          string   `json:"portal"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-700001", body.Referensi)
	require.NotEmpty(t, body.NoCase)
	require.NotEmpty(t, body.LampiranSurat.NoPolis)
	// "UP" diisi dari sumber yang sama dengan "Nama Peserta" — perilaku Pega apa adanya.
	require.Equal(t, body.LampiranSurat.NamaPeserta, body.LampiranSurat.UP)
	require.Len(t, body.BelumTerpetakan, 9)
	require.Contains(t, body.BelumTerpetakan, "No Kontrak")
	require.True(t, body.MasihDiPega)
	require.Equal(t, testPortal, body.Portal)
}

func TestDetailAnswersAnUnknownKeyWithAPortalAwareNotFound(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl/klaim/TIDAK-ADA")
	require.Equal(t, http.StatusNotFound, res.Code)
	require.Equal(t, rclpuclhttp.CodeClaimNotFound, errorCode(t, res))
	require.Contains(t, res.Body.String(), "portal")
}

func TestDetailRequiresACaller(t *testing.T) {
	server := testServer(t, serverOptions{noCaller: true})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/klaim/ASM-FW-GCNMFW-WORK%20PNC-700001")
	require.Equal(t, http.StatusConflict, res.Code)
}

// ---------------------------------------------------------------------------
// Aksi tulis
// ---------------------------------------------------------------------------

func TestRejectWriteAnswersNotImplementedAndLogsTheRequest(t *testing.T) {
	var buf bytes.Buffer
	server := testServer(t, serverOptions{logBuffer: &buf})

	res := send(t, server, http.MethodPost, "/api/inbox-rcl-pucl/tindakan?tindakan=%20cetak%20")
	require.Equal(t, http.StatusNotImplemented, res.Code)
	require.Equal(t, rclpuclhttp.CodeWriteNotAvailable, errorCode(t, res))

	require.Contains(t, buf.String(), "aksi tulis diminta pada modul yang belum menulis")
	require.Contains(t, buf.String(), "tindakan=cetak")
}

func TestRejectWriteWorksWithoutALogger(t *testing.T) {
	server := testServer(t, serverOptions{noLogger: true})

	res := send(t, server, http.MethodPost, "/api/inbox-rcl-pucl/tindakan")
	require.Equal(t, http.StatusNotImplemented, res.Code)
	require.Equal(t, rclpuclhttp.CodeWriteNotAvailable, errorCode(t, res))
}

// ---------------------------------------------------------------------------
// Penulis galat
// ---------------------------------------------------------------------------

func TestWriteErrorMapsEveryModuleError(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{inboxrclpucl.NewValidationError([]inboxrclpucl.Violation{{Field: "dari", Message: "x"}}),
			http.StatusUnprocessableEntity, rclpuclhttp.CodeValidationFail},
		{inboxrclpucl.ErrCallerUnknown, http.StatusConflict, rclpuclhttp.CodeCallerUnknown},
		{inboxrclpucl.ErrClaimNotFound, http.StatusNotFound, rclpuclhttp.CodeClaimNotFound},
		{inboxrclpucl.ErrReportNotAvailable, http.StatusNotFound, rclpuclhttp.CodeReportNotAvailable},
		{inboxrclpucl.ErrWriteNotAvailable, http.StatusNotImplemented, rclpuclhttp.CodeWriteNotAvailable},
	} {
		recorder := httptest.NewRecorder()
		write := rclpuclhttp.WriteError(nil, writeJSON, nil)
		write(recorder, httptest.NewRequest(http.MethodGet, "/x", nil), tc.err)

		require.Equal(t, tc.status, recorder.Code, tc.code)
		require.Equal(t, tc.code, errorCode(t, recorder))
	}
}

// Galat yang tidak dikenali dan tanpa cadangan dijawab 500 dengan pesan umum — rinciannya
// hanya masuk log, tidak pernah ke peramban.
func TestWriteErrorWithoutFallbackHidesTheDetailAndLogsIt(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	recorder := httptest.NewRecorder()
	write := rclpuclhttp.WriteError(logger, writeJSON, nil)
	write(recorder, httptest.NewRequest(http.MethodGet, "/api/x", nil),
		errors.New("rahasia basis data"))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, rclpuclhttp.CodeInternalError, errorCode(t, recorder))
	require.NotContains(t, recorder.Body.String(), "rahasia basis data")
	require.Contains(t, buf.String(), "permintaan gagal")
	require.Contains(t, buf.String(), "rahasia basis data")
}

func TestWriteErrorHandsUnknownErrorsToTheFallback(t *testing.T) {
	var handed error
	fallback := func(w http.ResponseWriter, _ *http.Request, err error) {
		handed = err
		w.WriteHeader(http.StatusTeapot)
	}

	recorder := httptest.NewRecorder()
	cause := errors.New("bukan milik modul")
	rclpuclhttp.WriteError(nil, writeJSON, fallback)(
		recorder, httptest.NewRequest(http.MethodGet, "/x", nil), cause)

	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.Equal(t, cause, handed)
}

// ---------------------------------------------------------------------------
// Ekspor grid
// ---------------------------------------------------------------------------

func TestExportOfAGridTabCopiesTheGrid(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "text/csv; charset=utf-8", res.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="rcl-pucl-kelengkapan-dokumen.csv"`,
		res.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", res.Header().Get("Cache-Control"))

	tab, _ := inboxrclpucl.FindTab(inboxrclpucl.TabKelengkapanDokumen)
	records := readCSV(t, res.Body.String())

	header := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		header = append(header, column.Title)
	}
	require.Equal(t, header, records[0])
	require.Greater(t, len(records), 1, "tab Kelengkapan Dokumen contoh berisi baris")
}

func TestExportOfTheMSIGTabNamesItsFile(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKlaimMSIG)
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, `attachment; filename="rcl-pucl-klaim-msig.csv"`,
		res.Header().Get("Content-Disposition"))
}

// gridItem menyusun satu baris grid yang setiap isiannya terbaca.
func gridItem(id string) inboxrclpucl.WorkItem {
	return inboxrclpucl.WorkItem{
		Reference:       "REF-" + id,
		CaseID:          "PNC-" + id,
		PolicyNumber:    "POL-" + id,
		InsuredName:     "PT " + id,
		InboxEntryAt:    "2026-09-10 09:30:00",
		AnalystNote:     "catatan " + id,
		Track:           "PUCL",
		LetterPrintedAt: "2026-09-11 10:00:00",
		ClaimAge:        "2026-09-10 09:29:59",
		ExpiryStatus:    "0",
	}
}

// pagedGrid menghasilkan repo yang menyerahkan halaman-halaman tertentu berurutan.
func pagedGrid(total int, pages ...[]inboxrclpucl.WorkItem) (fakeRepo, *[]int) {
	asked := &[]int{}
	return fakeRepo{list: func(_ inboxrclpucl.Query, p inboxrclpucl.Pagination) (inboxrclpucl.Page, error) {
		*asked = append(*asked, p.Page)
		items := []inboxrclpucl.WorkItem{}
		if p.Page-1 < len(pages) {
			items = pages[p.Page-1]
		}
		return inboxrclpucl.Page{Items: items, Total: total, Pagination: p.Normalize()}, nil
	}}, asked
}

func TestExportOfAGridWritesEveryCellInColumnOrder(t *testing.T) {
	repo, _ := pagedGrid(1, []inboxrclpucl.WorkItem{gridItem("1")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)
	require.Equal(t, http.StatusOK, res.Code)

	records := readCSV(t, res.Body.String())
	require.Len(t, records, 2)

	tab, _ := inboxrclpucl.FindTab(inboxrclpucl.TabKelengkapanDokumen)
	byKey := map[string]string{}
	for i, column := range tab.Columns {
		byKey[column.Key] = records[1][i]
	}
	require.Equal(t, "PNC-1", byKey[inboxrclpucl.FieldCaseID])
	require.Equal(t, "POL-1", byKey[inboxrclpucl.FieldPolicyNumber])
	require.Equal(t, "PT 1", byKey[inboxrclpucl.FieldInsuredName])
	require.Equal(t, "2026-09-10 09:30:00", byKey[inboxrclpucl.FieldInboxEntryAt])
	require.Equal(t, "catatan 1", byKey[inboxrclpucl.FieldAnalystNote])
	require.Equal(t, "PUCL", byKey[inboxrclpucl.FieldTrack])
	require.Equal(t, "2026-09-11 10:00:00", byKey[inboxrclpucl.FieldLetterPrintedAt])
	require.Equal(t, "2026-09-10 09:29:59", byKey[inboxrclpucl.FieldClaimAge])
	require.Equal(t, "0", byKey[inboxrclpucl.FieldExpiryStatus])
}

func TestExportOfAGridFollowsEveryPage(t *testing.T) {
	first := make([]inboxrclpucl.WorkItem, 0, inboxrclpucl.MaxPageSize)
	for i := 0; i < inboxrclpucl.MaxPageSize; i++ {
		first = append(first, gridItem("A"))
	}
	repo, asked := pagedGrid(inboxrclpucl.MaxPageSize+2, first,
		[]inboxrclpucl.WorkItem{gridItem("B1"), gridItem("B2")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)
	require.Equal(t, http.StatusOK, res.Code)

	records := readCSV(t, res.Body.String())
	require.Len(t, records, 1+inboxrclpucl.MaxPageSize+2)
	require.Equal(t, []int{1, 2}, *asked)
}

// Halaman kosong menghentikan unduhan meski jumlah seluruhnya belum tercapai — tanpa itu
// ekspor berputar selamanya ketika isi antrean berubah di tengah unduhan.
func TestExportOfAGridStopsAtAnEmptyPage(t *testing.T) {
	repo, asked := pagedGrid(10, []inboxrclpucl.WorkItem{gridItem("1")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)
	require.Equal(t, http.StatusOK, res.Code)

	require.Len(t, readCSV(t, res.Body.String()), 2)
	require.Equal(t, []int{1, 2}, *asked)
}

func TestExportOfAGridStopsAndLogsWhenALaterPageFails(t *testing.T) {
	var buf bytes.Buffer
	calls := 0
	repo := fakeRepo{list: func(_ inboxrclpucl.Query, p inboxrclpucl.Pagination) (inboxrclpucl.Page, error) {
		calls++
		if p.Page > 1 {
			return inboxrclpucl.Page{}, errors.New("koneksi putus di halaman dua")
		}
		return inboxrclpucl.Page{
			Items: []inboxrclpucl.WorkItem{gridItem("1")}, Total: 5, Pagination: p.Normalize(),
		}, nil
	}}
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)

	// Header sudah terkirim, sehingga statusnya tetap 200 dan berkasnya separuh jadi.
	require.Equal(t, http.StatusOK, res.Code)
	require.Len(t, readCSV(t, res.Body.String()), 2)
	require.Equal(t, 2, calls)
	require.Contains(t, buf.String(), "ekspor inbox RCL/PUCL terputus")
	require.Contains(t, buf.String(), "koneksi putus di halaman dua")
}

// Berkas yang menyentuh batas diberi tanda di baris terakhir, bukan dipotong diam-diam.
func TestExportOfAGridMarksATruncatedFile(t *testing.T) {
	page := make([]inboxrclpucl.WorkItem, inboxrclpucl.MaxPageSize)
	for i := range page {
		page[i] = inboxrclpucl.WorkItem{CaseID: "X"}
	}
	repo := fakeRepo{list: func(_ inboxrclpucl.Query, p inboxrclpucl.Pagination) (inboxrclpucl.Page, error) {
		return inboxrclpucl.Page{Items: page, Total: 60_000, Pagination: p.Normalize()}, nil
	}}
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen)
	require.Equal(t, http.StatusOK, res.Code)

	records := readCSV(t, res.Body.String())
	require.Len(t, records, 1+50_000+1)
	require.Equal(t,
		"-- Terpotong pada 50000 baris dari 60000 yang cocok. Persempit rentangnya. --",
		records[len(records)-1][0])
}

func TestExportRejectsAnUnknownTabBeforeWritingAnything(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl/ekspor?tab=9")
	require.Equal(t, http.StatusUnprocessableEntity, res.Code)
	require.Equal(t, rclpuclhttp.CodeValidationFail, errorCode(t, res))
	require.Empty(t, res.Header().Get("Content-Disposition"))
}

func TestExportRequiresACaller(t *testing.T) {
	server := testServer(t, serverOptions{noCaller: true})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl/ekspor")
	require.Equal(t, http.StatusConflict, res.Code)
}

// failingWriter adalah ResponseWriter yang menolak setiap penulisan badan.
type failingWriter struct {
	header http.Header
	status int
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(status int)    { f.status = status }
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("sambungan ditutup") }
func newFailingWriter() *failingWriter             { return &failingWriter{header: http.Header{}} }
func portalRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)
	return request
}

// Sambungan yang putus saat potong pertama didorong keluar menghentikan unduhan dan
// meninggalkan jejak.
func TestExportOfAGridLogsAFailedFlush(t *testing.T) {
	var buf bytes.Buffer
	repo, asked := pagedGrid(5, []inboxrclpucl.WorkItem{gridItem("1")})
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	server.ServeHTTP(newFailingWriter(),
		portalRequest("/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen))

	require.Equal(t, []int{1}, *asked, "halaman berikutnya tidak diminta")
	require.Contains(t, buf.String(), "ekspor inbox RCL/PUCL terputus")
	require.Contains(t, buf.String(), "sambungan ditutup")
}

// Baris yang lebih besar daripada penyangga penulis memaksa penulisan langsung, dan
// kegagalannya terbaca saat baris itu ditulis.
func TestExportOfAGridLogsAFailedRowWrite(t *testing.T) {
	var buf bytes.Buffer
	huge := gridItem("1")
	huge.InsuredName = strings.Repeat("x", 8192)
	repo, _ := pagedGrid(1, []inboxrclpucl.WorkItem{huge})
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	server.ServeHTTP(newFailingWriter(),
		portalRequest("/api/inbox-rcl-pucl/ekspor?tab="+inboxrclpucl.TabKelengkapanDokumen))

	require.Contains(t, buf.String(), "ekspor inbox RCL/PUCL terputus")
}

// ---------------------------------------------------------------------------
// Ekspor laporan harian
// ---------------------------------------------------------------------------

func reportRow(id string) inboxrclpucl.DailyReportRow {
	return inboxrclpucl.DailyReportRow{
		Reference:       "REF-" + id,
		CaseID:          "PNC-" + id,
		PolicyNumber:    "POL-" + id,
		InsuredName:     "PT " + id,
		SentAt:          "2026-09-10 09:30:00",
		AnalystNote:     "catatan " + id,
		LetterPrintedAt: "2026-09-11 10:00:00",
		Track:           "RCL",
		ClaimStatus:     "Register",
	}
}

func pagedReport(total int, pages ...[]inboxrclpucl.DailyReportRow) (fakeRepo, *[]inboxrclpucl.DateRange) {
	asked := &[]inboxrclpucl.DateRange{}
	return fakeRepo{report: func(rng inboxrclpucl.DateRange, p inboxrclpucl.Pagination) ([]inboxrclpucl.DailyReportRow, int, error) {
		*asked = append(*asked, rng)
		rows := []inboxrclpucl.DailyReportRow{}
		if p.Page-1 < len(pages) {
			rows = pages[p.Page-1]
		}
		return rows, total, nil
	}}, asked
}

func TestExportOfCetakSuratWritesTheDailyReport(t *testing.T) {
	repo, asked := pagedReport(1, []inboxrclpucl.DailyReportRow{reportRow("1")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30")
	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t,
		`attachment; filename="laporan-harian-rcl-pucl-2026-09-01-sd-2026-09-30.csv"`,
		res.Header().Get("Content-Disposition"))
	require.Equal(t, []inboxrclpucl.DateRange{{From: "2026-09-01", To: "2026-09-30"}}, *asked)

	records := readCSV(t, res.Body.String())
	require.Len(t, records, 2)

	byKey := map[string]string{}
	for i, column := range inboxrclpucl.DailyReportColumns {
		require.Equal(t, column.Title, records[0][i])
		byKey[column.Key] = records[1][i]
	}
	require.Equal(t, "PNC-1", byKey[inboxrclpucl.FieldCaseID])
	require.Equal(t, "POL-1", byKey[inboxrclpucl.FieldPolicyNumber])
	require.Equal(t, "PT 1", byKey[inboxrclpucl.FieldInsuredName])
	require.Equal(t, "2026-09-10 09:30:00", byKey[inboxrclpucl.FieldReportSentAt])
	require.Equal(t, "catatan 1", byKey[inboxrclpucl.FieldAnalystNote])
	require.Equal(t, "2026-09-11 10:00:00", byKey[inboxrclpucl.FieldLetterPrintedAt])
	require.Equal(t, "RCL", byKey[inboxrclpucl.FieldTrack])
	require.Equal(t, "Register", byKey[inboxrclpucl.FieldReportClaimStatus])
}

func TestExportOfTheDailyReportRejectsMissingDatesBeforeWriting(t *testing.T) {
	server := testServer(t, serverOptions{})

	res := send(t, server, http.MethodGet, "/api/inbox-rcl-pucl/ekspor?tab=1")
	require.Equal(t, http.StatusUnprocessableEntity, res.Code)
	require.Empty(t, res.Header().Get("Content-Disposition"))

	var body struct {
		Detail []struct {
			Field string `json:"field"`
		} `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Len(t, body.Detail, 2, "kedua tanggal disampaikan sekaligus")
}

func TestExportOfTheDailyReportFollowsEveryPage(t *testing.T) {
	first := make([]inboxrclpucl.DailyReportRow, inboxrclpucl.MaxPageSize)
	for i := range first {
		first[i] = reportRow("A")
	}
	repo, asked := pagedReport(inboxrclpucl.MaxPageSize+1, first,
		[]inboxrclpucl.DailyReportRow{reportRow("B")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30")
	require.Equal(t, http.StatusOK, res.Code)
	require.Len(t, readCSV(t, res.Body.String()), 1+inboxrclpucl.MaxPageSize+1)
	require.Len(t, *asked, 2)
}

func TestExportOfTheDailyReportStopsAtAnEmptyPage(t *testing.T) {
	repo, asked := pagedReport(7, []inboxrclpucl.DailyReportRow{reportRow("1")})
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30")
	require.Equal(t, http.StatusOK, res.Code)
	require.Len(t, readCSV(t, res.Body.String()), 2)
	require.Len(t, *asked, 2)
}

func TestExportOfTheDailyReportStopsAndLogsWhenALaterPageFails(t *testing.T) {
	var buf bytes.Buffer
	repo := fakeRepo{report: func(_ inboxrclpucl.DateRange, p inboxrclpucl.Pagination) ([]inboxrclpucl.DailyReportRow, int, error) {
		if p.Page > 1 {
			return nil, 0, errors.New("laporan putus")
		}
		return []inboxrclpucl.DailyReportRow{reportRow("1")}, 3, nil
	}}
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30")
	require.Equal(t, http.StatusOK, res.Code)
	require.Len(t, readCSV(t, res.Body.String()), 2)
	require.Contains(t, buf.String(), "laporan putus")
}

func TestExportOfTheDailyReportMarksATruncatedFile(t *testing.T) {
	page := make([]inboxrclpucl.DailyReportRow, inboxrclpucl.MaxPageSize)
	for i := range page {
		page[i] = inboxrclpucl.DailyReportRow{CaseID: "X"}
	}
	repo := fakeRepo{report: func(inboxrclpucl.DateRange, inboxrclpucl.Pagination) ([]inboxrclpucl.DailyReportRow, int, error) {
		return page, 70_000, nil
	}}
	server := testServer(t, serverOptions{repo: repo})

	res := send(t, server, http.MethodGet,
		"/api/inbox-rcl-pucl/ekspor?dari=2026-01-01&sampai=2026-09-30")
	require.Equal(t, http.StatusOK, res.Code)

	records := readCSV(t, res.Body.String())
	require.Len(t, records, 1+50_000+1)
	require.Equal(t,
		"-- Terpotong pada 50000 baris dari 70000 yang cocok. Persempit rentangnya. --",
		records[len(records)-1][0])
}

func TestExportOfTheDailyReportLogsAFailedFlush(t *testing.T) {
	var buf bytes.Buffer
	repo, asked := pagedReport(5, []inboxrclpucl.DailyReportRow{reportRow("1")})
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	server.ServeHTTP(newFailingWriter(),
		portalRequest("/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30"))

	require.Len(t, *asked, 1, "halaman berikutnya tidak diminta")
	require.Contains(t, buf.String(), "ekspor inbox RCL/PUCL terputus")
}

func TestExportOfTheDailyReportLogsAFailedRowWrite(t *testing.T) {
	var buf bytes.Buffer
	huge := reportRow("1")
	huge.AnalystNote = strings.Repeat("y", 8192)
	repo, _ := pagedReport(1, []inboxrclpucl.DailyReportRow{huge})
	server := testServer(t, serverOptions{repo: repo, logBuffer: &buf})

	server.ServeHTTP(newFailingWriter(),
		portalRequest("/api/inbox-rcl-pucl/ekspor?dari=2026-09-01&sampai=2026-09-30"))

	require.Contains(t, buf.String(), "ekspor inbox RCL/PUCL terputus")
}
