package inboxclaimtreatynonprophttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
	"claim-pnc/internal/inboxclaimtreatynonprop/usecase"
	"claim-pnc/internal/portal"

	nonprophttp "claim-pnc/internal/inboxclaimtreatynonprop/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const portalMain = "ASM"

// pagedRepo menghasilkan halaman buatan dengan total tetap, untuk menguji ekspor bertahap
// tanpa menyimpan puluhan ribu baris.
type pagedRepo struct {
	total   int
	failOn  int
	longRow bool
	calls   int
}

func (p *pagedRepo) List(
	_ context.Context, _ inboxclaimtreatynonprop.Query, page inboxclaimtreatynonprop.Pagination,
) (inboxclaimtreatynonprop.Page, error) {
	p.calls++
	clean := page.Normalize()
	if p.failOn != 0 && clean.Page == p.failOn {
		return inboxclaimtreatynonprop.Page{}, errors.New("koneksi putus di tengah ekspor")
	}

	items := []inboxclaimtreatynonprop.WorkItem{}
	for i := clean.Offset(); i < p.total && len(items) < clean.Size; i++ {
		item := inboxclaimtreatynonprop.WorkItem{ClaimID: fmt.Sprintf("CLMNP-%05d", i)}
		if p.longRow {
			item.InsuredName = strings.Repeat("x", 200)
		}
		items = append(items, item)
	}
	return inboxclaimtreatynonprop.Page{Items: items, Total: p.total, Pagination: clean}, nil
}

type harness struct {
	server  http.Handler
	handler *nonprophttp.Handler
	logs    *bytes.Buffer
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// build merakit rantai yang meniru cmd/claimpnc: middleware portal di depan, penulis galat
// portal sebagai cadangan.
func build(
	t *testing.T,
	selector inboxclaimtreatynonprop.RepoSelector,
	getCaller nonprophttp.CallerReader,
	withFallback bool,
) harness {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(logs, nil))

	service, err := usecase.NewService(usecase.Options{RepoSelector: selector, Logger: logger})
	require.NoError(t, err)

	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_cadangan", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	options := nonprophttp.Options{
		Service:   service,
		GetCaller: getCaller,
		Logger:    logger,
		WriteJSON: writeJSON,
	}
	if withFallback {
		options.FallbackErrorWriter = nonprophttp.ErrorWriter(portalError)
	}
	handler := nonprophttp.NewHandler(options)

	deps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{portalMain} },
		Logger:       logger,
		WriteError:   portalError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		nonprophttp.Mount(api, handler, deps)
	})
	return harness{server: router, handler: handler, logs: logs}
}

func adminCaller(context.Context) (nonprophttp.Caller, bool) {
	return nonprophttp.Caller{Login: "ADMINNONPROP1"}, true
}

func sample(string) (inboxclaimtreatynonprop.Repo, error) { return memory.NewSampleStore(), nil }

func send(h harness, method, path, portalAlias string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if portalAlias != "" {
		request.Header.Set("X-Portal", portalAlias)
	}
	recorder := httptest.NewRecorder()
	h.server.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestMetadataReturnsTabsAndPortal(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/tab", portalMain)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, portalMain, body["portal"])
	require.Equal(t, inboxclaimtreatynonprop.DefaultTab, body["tab_bawaan"])
	require.Len(t, body["selisih_terencana"], len(inboxclaimtreatynonprop.PlannedDifferences))

	tabs := body["tab"].([]any)
	require.Len(t, tabs, 3)
	first := tabs[0].(map[string]any)
	require.Equal(t, "Treaty-In Admin", first["nama"])
	require.Equal(t, "Work Treatyin Non Propotional Admin", first["judul_grid"])
	require.Equal(t, true, first["pakai_lihat_semua"])
	require.Len(t, first["kolom"], 11)
	committee := tabs[2].(map[string]any)
	require.Equal(t, true, committee["terhalang"])
	require.NotEmpty(t, committee["alasan_terhalang"])
	require.Empty(t, committee["kolom"])
}

func TestListReturnsRowsPaginationAndFilter(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet,
		"/api/inbox-claim-treaty-non-prop?lihat_semua=YA&halaman=2&ukuran=3", portalMain)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, portalMain, body["portal"])
	require.Equal(t, map[string]any{"lihat_semua": true, "lihat_tba": false}, body["penyaring"])
	require.Equal(t, map[string]any{
		"halaman": float64(2), "ukuran": float64(3), "total": float64(4), "total_halaman": float64(2),
	}, body["paginasi"])

	rows := body["baris"].([]any)
	require.Len(t, rows, 1)
	row := rows[0].(map[string]any)
	require.Equal(t, "CLMNP-1001", row["no_klaim"])
	require.Equal(t, "99.002.2026.00001", row["no_polis"])
	require.Equal(t, "PT Contoh Sejahtera", row["nama_tertanggung"])
	require.Contains(t, row, "referensi")
	require.Contains(t, row, "aging")
}

// Nilai paginasi yang tidak terbaca atau negatif jatuh ke bawaan; checkbox yang salah ketik
// berarti TIDAK.
func TestListIgnoresUnreadableParameters(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	body := decode(t, send(h, http.MethodGet,
		"/api/inbox-claim-treaty-non-prop?halaman=abc&ukuran=-4&lihat_semua=mungkin&lihat_tba=",
		portalMain))
	require.Equal(t, map[string]any{"lihat_semua": false, "lihat_tba": false}, body["penyaring"])
	require.Equal(t, float64(1), body["paginasi"].(map[string]any)["halaman"])
	require.Equal(t, float64(inboxclaimtreatynonprop.DefaultPageSize),
		body["paginasi"].(map[string]any)["ukuran"])
	require.Equal(t, float64(3), body["paginasi"].(map[string]any)["total"])
}

func TestListBlockedTabIsValidationError(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop?tab=3", portalMain)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, nonprophttp.CodeValidationFail, body["kode"])
	detail := body["detail"].([]any)
	require.Len(t, detail, 1)
	require.Equal(t, inboxclaimtreatynonprop.FieldTab, detail[0].(map[string]any)["field"])
}

// Identitas yang tidak terbaca dijawab 409, baik karena pembacanya tidak dipasang, sesinya
// tidak ada, maupun loginnya kosong.
func TestListUnknownCaller(t *testing.T) {
	readers := map[string]nonprophttp.CallerReader{
		"tanpa_pembaca": nil,
		"tanpa_sesi":    func(context.Context) (nonprophttp.Caller, bool) { return nonprophttp.Caller{}, false },
		"login_kosong": func(context.Context) (nonprophttp.Caller, bool) {
			return nonprophttp.Caller{Login: "  "}, true
		},
	}

	for label, reader := range readers {
		h := build(t, sample, reader, true)
		recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop", portalMain)
		require.Equalf(t, http.StatusConflict, recorder.Code, label)
		require.Equalf(t, nonprophttp.CodeCallerUnknown, decode(t, recorder)["kode"], label)
	}
}

// Galat yang bukan milik modul diserahkan ke cadangan bila ada.
func TestListUnrecognizedErrorGoesToFallback(t *testing.T) {
	h := build(t,
		func(string) (inboxclaimtreatynonprop.Repo, error) { return nil, errors.New("koneksi mati") },
		adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop", portalMain)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "galat_cadangan", decode(t, recorder)["kode"])
}

// Tanpa cadangan, galat asing dijawab 500 umum dan rinciannya hanya masuk log.
func TestListUnrecognizedErrorWithoutFallback(t *testing.T) {
	h := build(t,
		func(string) (inboxclaimtreatynonprop.Repo, error) { return nil, errors.New("rahasia-internal") },
		adminCaller, false)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop", portalMain)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, nonprophttp.CodeInternalError, body["kode"])
	require.NotContains(t, recorder.Body.String(), "rahasia-internal")
	require.Contains(t, h.logs.String(), "rahasia-internal")
}

func TestRequestWithoutPortalRejected(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop", "")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "portal_tidak_disebut", decode(t, recorder)["kode"])
}

// Handler sendiri pun menolak bila konteks tidak membawa portal aktif.
func TestHandlersRejectMissingPortalInContext(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	cases := map[string]http.HandlerFunc{
		"metadata": h.handler.Metadata,
		"list":     h.handler.List,
		"export":   h.handler.Export,
		"tulis":    h.handler.RejectWrite,
	}
	for label, handle := range cases {
		recorder := httptest.NewRecorder()
		handle(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equalf(t, http.StatusBadRequest, recorder.Code, label)
		require.Equalf(t, "portal_tidak_disebut", decode(t, recorder)["kode"], label)
	}
}

func TestCreateClaimAnswers501AndIsLogged(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodPost, "/api/inbox-claim-treaty-non-prop/klaim", portalMain)
	require.Equal(t, http.StatusNotImplemented, recorder.Code)
	require.Equal(t, nonprophttp.CodeWriteNotAvailable, decode(t, recorder)["kode"])
	require.Contains(t, h.logs.String(), "aksi tulis diminta pada modul yang belum menulis")
}

// Penolakan tulis tetap bekerja tanpa logger.
func TestRejectWriteWithoutLogger(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{RepoSelector: sample})
	require.NoError(t, err)
	handler := nonprophttp.NewHandler(nonprophttp.Options{Service: service, WriteJSON: writeJSON})

	request := httptest.NewRequest(http.MethodPost, "/x", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(),
		portal.Portal{Alias: portalMain}))
	recorder := httptest.NewRecorder()
	handler.RejectWrite(recorder, request)
	require.Equal(t, http.StatusNotImplemented, recorder.Code)
}

func readCSV(t *testing.T, body string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportWritesHeaderAndRows(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet,
		"/api/inbox-claim-treaty-non-prop/ekspor?lihat_semua=1&lihat_tba=true", portalMain)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="klaim-treaty-non-prop-tab1-semua-tba.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

	records := readCSV(t, recorder.Body.String())
	require.Equal(t, []string{
		"No Klaim", "Business Name", "Source of Business", "Ceding Co Name",
		"Insured Name", "Date of Loss", "Create Operator",
	}, records[0])
	require.Len(t, records, 2)
	require.Equal(t, "CLMNP-1004", records[1][0])
	require.Equal(t, "PT Contoh Alat Berat", records[1][4])
}

func TestExportFilenameForTechnicalTab(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor?tab=2", portalMain)
	require.Equal(t, `attachment; filename="klaim-treaty-non-prop-tab2.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Len(t, readCSV(t, recorder.Body.String()), 3)
}

// Galat sebelum header terkirim tetap dijawab sebagai JSON yang terbaca.
func TestExportErrorBeforeHeaderIsJSON(t *testing.T) {
	h := build(t, sample, adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor?tab=3", portalMain)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, nonprophttp.CodeValidationFail, decode(t, recorder)["kode"])
}

func TestExportUnknownCaller(t *testing.T) {
	h := build(t, sample, nil, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor", portalMain)
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// Ekspor mengambil halaman demi halaman sampai seluruh baris tertulis.
func TestExportWalksEveryPage(t *testing.T) {
	repo := &pagedRepo{total: 250}
	h := build(t, func(string) (inboxclaimtreatynonprop.Repo, error) { return repo, nil },
		adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor", portalMain)
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder.Body.String())
	require.Len(t, records, 251)
	require.Equal(t, "CLMNP-00249", records[250][0])
	require.Equal(t, 3, repo.calls)
}

// Berkas yang melewati batas diberi tanda di baris terakhir, bukan dipotong diam-diam.
func TestExportTruncatesAtLimit(t *testing.T) {
	repo := &pagedRepo{total: 50_150}
	h := build(t, func(string) (inboxclaimtreatynonprop.Repo, error) { return repo, nil },
		adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor", portalMain)
	records := readCSV(t, recorder.Body.String())

	require.Len(t, records, 1+50_000+1)
	last := records[len(records)-1]
	require.Equal(t,
		"-- Terpotong pada 50000 baris dari 50150 yang cocok. Persempit penyaringnya. --", last[0])
	require.Len(t, last, 7)
}

// Galat pada halaman lanjutan menghentikan berkas dan meninggalkan jejak di log.
func TestExportFailureMidwayIsLogged(t *testing.T) {
	repo := &pagedRepo{total: 250, failOn: 2}
	h := build(t, func(string) (inboxclaimtreatynonprop.Repo, error) { return repo, nil },
		adminCaller, true)

	recorder := send(h, http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor", portalMain)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder.Body.String()), 101)
	require.Contains(t, h.logs.String(), "ekspor klaim treaty non-prop terputus")
	require.Contains(t, h.logs.String(), "koneksi putus di tengah ekspor")
}

// failingWriter menolak setiap penulisan badan, meniru peramban yang memutus unduhan.
type failingWriter struct {
	header http.Header
	code   int
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(code int)      { f.code = code }
func (f *failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func exportTo(t *testing.T, repo *pagedRepo) (*failingWriter, *bytes.Buffer) {
	t.Helper()
	h := build(t, func(string) (inboxclaimtreatynonprop.Repo, error) { return repo, nil },
		adminCaller, true)

	request := httptest.NewRequest(http.MethodGet, "/api/inbox-claim-treaty-non-prop/ekspor", nil)
	request.Header.Set("X-Portal", portalMain)
	writer := &failingWriter{header: http.Header{}}
	h.server.ServeHTTP(writer, request)
	return writer, h.logs
}

// Galat saat mendorong potongan keluar dicatat.
func TestExportFlushFailureIsLogged(t *testing.T) {
	_, logs := exportTo(t, &pagedRepo{total: 2})
	require.Contains(t, logs.String(), "ekspor klaim treaty non-prop terputus")
	require.Contains(t, logs.String(), io.ErrClosedPipe.Error())
}

// Galat saat menulis baris yang memenuhi penyangga dicatat.
func TestExportRowWriteFailureIsLogged(t *testing.T) {
	repo := &pagedRepo{total: 100, longRow: true}
	_, logs := exportTo(t, repo)
	require.Contains(t, logs.String(), "ekspor klaim treaty non-prop terputus")
	require.Equal(t, 1, repo.calls, "penulisan berhenti sebelum halaman berikutnya diminta")
}
