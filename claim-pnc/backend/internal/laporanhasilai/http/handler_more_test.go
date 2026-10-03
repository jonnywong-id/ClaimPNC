package laporanhasilaihttp_test

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
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
	laporanhasilaiusecase "claim-pnc/internal/laporanhasilai/usecase"
	"claim-pnc/internal/portal"

	laporanhasilaihttp "claim-pnc/internal/laporanhasilai/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// pagedRepo menghasilkan baris buatan sebanyak total, per halaman yang diminta.
type pagedRepo struct {
	total int

	// emptyPages membuat setiap halaman kosong meski total > 0.
	emptyPages bool

	// failFromPage membuat List gagal mulai halaman ini (0 = tidak pernah gagal).
	failFromPage int

	// claimNumber mengisi kolom No Klaim setiap baris.
	claimNumber string

	listErr      error
	summarizeErr error
}

var errPageBroken = errors.New("halaman lanjutan gagal dibaca")

func (r *pagedRepo) List(
	_ context.Context, _ laporanhasilai.Filter, page laporanhasilai.Pagination,
) (laporanhasilai.Page, error) {
	if r.listErr != nil {
		return laporanhasilai.Page{}, r.listErr
	}
	if r.failFromPage > 0 && page.Page >= r.failFromPage {
		return laporanhasilai.Page{}, errPageBroken
	}

	result := laporanhasilai.Page{Total: r.total}
	if r.emptyPages {
		return result, nil
	}
	for i := page.Offset(); i < r.total && i < page.Offset()+page.Size; i++ {
		number := r.claimNumber
		if number == "" {
			number = fmt.Sprintf("PNC-%06d", i)
		}
		result.Rows = append(result.Rows, laporanhasilai.Row{
			ID:              fmt.Sprint(i),
			ClaimNumber:     number,
			CommitteeStatus: laporanhasilai.LabelAccepted,
			CommitteeDate:   time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC),
		})
	}
	return result, nil
}

func (r *pagedRepo) Summarize(context.Context, laporanhasilai.Filter) (laporanhasilai.Summary, error) {
	return laporanhasilai.Summary{}, r.summarizeErr
}

// handlerOver membentuk handler di atas repo buatan, tanpa middleware apa pun.
func handlerOver(
	t *testing.T, repo laporanhasilai.Repo, fallback laporanhasilaihttp.ErrorWriter, logs *bytes.Buffer,
) *laporanhasilaihttp.Handler {
	t.Helper()

	service, err := laporanhasilaiusecase.NewService(laporanhasilaiusecase.Options{
		RepoSelector: func(string) (laporanhasilai.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	handler, err := laporanhasilaihttp.NewHandler(laporanhasilaihttp.Options{
		Service:             service,
		Logger:              slog.New(slog.NewTextHandler(logs, nil)),
		WriteJSON:           writeJSONPlain,
		FallbackErrorWriter: fallback,
	})
	require.NoError(t, err)
	return handler
}

func writeJSONPlain(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// withPortal membentuk permintaan yang sudah melewati middleware portal.
func withPortal(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	return request.WithContext(portalhttp.WithActivePortal(
		request.Context(), portal.Portal{Alias: "ASM"}))
}

const septemberQuery = "?dari=2026-09-01&sampai=2026-09-30"

func readCSV(t *testing.T, body string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}

func TestNewHandlerRejectsMissingParts(t *testing.T) {
	service, err := laporanhasilaiusecase.NewService(laporanhasilaiusecase.Options{
		RepoSelector: func(string) (laporanhasilai.Repo, error) { return &pagedRepo{}, nil },
	})
	require.NoError(t, err)

	_, err = laporanhasilaihttp.NewHandler(laporanhasilaihttp.Options{WriteJSON: writeJSONPlain})
	require.ErrorContains(t, err, "Service")

	_, err = laporanhasilaihttp.NewHandler(laporanhasilaihttp.Options{Service: service})
	require.ErrorContains(t, err, "WriteJSON")
}

func TestHandlersWithoutActivePortalAreRejected(t *testing.T) {
	// Handler yang dipanggil tanpa middleware portal menolak dengan galat portal, yang
	// diteruskan ke penulis cadangan — tidak pernah melayani portal bawaan.
	var seen error
	fallback := func(w http.ResponseWriter, _ *http.Request, err error) {
		seen = err
		w.WriteHeader(http.StatusBadRequest)
	}
	handler := handlerOver(t, &pagedRepo{total: 1}, fallback, &bytes.Buffer{})

	for name, serve := range map[string]http.HandlerFunc{
		"search": handler.Search,
		"export": handler.Export,
	} {
		seen = nil
		recorder := httptest.NewRecorder()
		serve(recorder, httptest.NewRequest(http.MethodGet, "/x"+septemberQuery, nil))

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "handler %s", name)
		require.ErrorIs(t, seen, portal.ErrNotStated)
		require.Empty(t, recorder.Header().Get("Content-Disposition"))
	}
}

func TestUnrecognizedErrorWithoutFallbackIs500AndLogged(t *testing.T) {
	logs := &bytes.Buffer{}
	handler := handlerOver(t, &pagedRepo{summarizeErr: errors.New("ORA-03113 rahasia")}, nil, logs)

	recorder := httptest.NewRecorder()
	handler.Search(recorder, withPortal("/api/laporan-hasil-ai"+septemberQuery))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, laporanhasilaihttp.CodeInternalError, body["kode"])
	require.NotContains(t, recorder.Body.String(), "rahasia")
	require.Contains(t, logs.String(), "ORA-03113 rahasia")
}

func TestUnrecognizedErrorWithoutLoggerIsStill500(t *testing.T) {
	writeError := laporanhasilaihttp.WriteError(nil, writeJSONPlain, nil)

	recorder := httptest.NewRecorder()
	writeError(recorder, httptest.NewRequest(http.MethodGet, "/x", nil), errors.New("boom"))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestExportFirstPageFailureAnswersBeforeDownloadStarts(t *testing.T) {
	logs := &bytes.Buffer{}
	handler := handlerOver(t, &pagedRepo{listErr: errors.New("koneksi putus")}, nil, logs)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Disposition"))
}

func TestExportWalksEveryChunk(t *testing.T) {
	// 250 baris = tiga potongan (100, 100, 50) — seluruhnya tertulis tanpa terpotong.
	handler := handlerOver(t, &pagedRepo{total: 250}, nil, &bytes.Buffer{})

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="laporan-hasil-ai-2026-09-01-sd-2026-09-30.csv"`,
		recorder.Header().Get("Content-Disposition"))

	records := readCSV(t, recorder.Body.String())
	require.Len(t, records, 251)
	require.Equal(t, "No Klaim", records[0][0])
	require.Equal(t, "PNC-000000", records[1][0])
	require.Equal(t, "PNC-000249", records[250][0])
	require.Equal(t, "02/09/2026", records[1][3])
	// Tanggal AI kosong ditulis sebagai sel kosong.
	require.Equal(t, "", records[1][5])
}

func TestExportStopsWhenTheRepoReturnsAnEmptyPage(t *testing.T) {
	handler := handlerOver(t, &pagedRepo{total: 7, emptyPages: true}, nil, &bytes.Buffer{})

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder.Body.String()), 1, "hanya kepala kolom")
}

func TestExportLaterPageFailureIsLoggedAndCutsTheFile(t *testing.T) {
	logs := &bytes.Buffer{}
	handler := handlerOver(t, &pagedRepo{total: 250, failFromPage: 2}, nil, logs)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	// Unduhan sudah berjalan, sehingga status tidak lagi dapat diubah — berkasnya terpotong
	// pada potongan pertama dan kegagalannya hanya tercatat di log.
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder.Body.String()), 101)
	require.Contains(t, logs.String(), "ekspor Laporan Hasil AI terputus")
	require.Contains(t, logs.String(), errPageBroken.Error())
}

func TestExportIsTruncatedAtTheLimitWithANotice(t *testing.T) {
	handler := handlerOver(t, &pagedRepo{total: 50_150, claimNumber: "X"}, nil, &bytes.Buffer{})

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	records := readCSV(t, recorder.Body.String())
	// Kepala + 50.000 baris + satu baris pemberitahuan.
	require.Len(t, records, 1+50_000+1)
	notice := records[len(records)-1]
	require.Len(t, notice, 10)
	require.Equal(t,
		"-- Terpotong pada 50000 baris dari 50150 yang cocok. Persempit rentang tanggalnya. --",
		notice[0])
	require.Equal(t, "", notice[1])
}

// brokenWriter menerima header tetapi gagal setiap kali badan ditulis.
type brokenWriter struct {
	header http.Header
	status int
}

func (w *brokenWriter) Header() http.Header       { return w.header }
func (w *brokenWriter) WriteHeader(status int)    { w.status = status }
func (w *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("peramban menutup sambungan") }

func TestExportLogsAFailedFlush(t *testing.T) {
	logs := &bytes.Buffer{}
	handler := handlerOver(t, &pagedRepo{total: 3}, nil, logs)

	writer := &brokenWriter{header: http.Header{}}
	handler.Export(writer, withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	require.Equal(t, `attachment; filename="laporan-hasil-ai-2026-09-01-sd-2026-09-30.csv"`,
		writer.header.Get("Content-Disposition"))
	require.Contains(t, logs.String(), "ekspor Laporan Hasil AI terputus")
	require.Contains(t, logs.String(), "peramban menutup sambungan")
}

func TestExportLogsAFailedRowWrite(t *testing.T) {
	// Baris yang lebih besar daripada penyangga csv memaksa penulisan langsung ke
	// sambungan, sehingga kegagalannya muncul pada Write baris — bukan pada Flush.
	logs := &bytes.Buffer{}
	handler := handlerOver(t, &pagedRepo{total: 3, claimNumber: strings.Repeat("A", 8192)}, nil, logs)

	handler.Export(&brokenWriter{header: http.Header{}},
		withPortal("/api/laporan-hasil-ai/ekspor"+septemberQuery))

	require.Equal(t, 1, strings.Count(logs.String(), "ekspor Laporan Hasil AI terputus"))
	require.Contains(t, logs.String(), "peramban menutup sambungan")
}
