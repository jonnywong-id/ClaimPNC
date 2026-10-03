package inboxoutstandinghttp_test

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
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/usecase"

	outstandinghttp "claim-pnc/internal/inboxoutstanding/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// fakeService menjawab menurut fungsi yang diisi tiap uji.
type fakeService struct {
	list    func(usecase.Query) (usecase.Result, error)
	export  func(usecase.ExportQuery) (usecase.ExportResult, error)
	summary func(usecase.Query) (usecase.SummaryResult, error)
}

func (f fakeService) List(_ context.Context, q usecase.Query) (usecase.Result, error) {
	return f.list(q)
}

func (f fakeService) Export(_ context.Context, q usecase.ExportQuery) (usecase.ExportResult, error) {
	return f.export(q)
}

func (f fakeService) Summary(_ context.Context, q usecase.Query) (usecase.SummaryResult, error) {
	return f.summary(q)
}

func jsonWriter(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// fakeHandler membentuk handler di atas layanan tiruan, TANPA penulis galat cadangan.
func fakeHandler(service outstandinghttp.Service, logs *bytes.Buffer, caller *outstandinghttp.Caller) *outstandinghttp.Handler {
	return outstandinghttp.NewHandler(outstandinghttp.Options{
		Service: service,
		GetCaller: func(context.Context) (outstandinghttp.Caller, bool) {
			if caller == nil {
				return outstandinghttp.Caller{}, false
			}
			return *caller, true
		},
		Logger:    slog.New(slog.NewTextHandler(logs, nil)),
		WriteJSON: jsonWriter,
		Location:  wib,
		Now:       func() time.Time { return testNow },
	})
}

// withPortal menyertakan portal aktif ke konteks, seperti yang dilakukan middleware.
func withPortal(r *http.Request) *http.Request {
	return r.WithContext(portalhttp.WithActivePortal(r.Context(), portalmemory.SampleList()[0]))
}

func decodeCode(t *testing.T, body []byte) string {
	t.Helper()
	var payload struct {
		Code string `json:"kode"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Code
}

var admin = &outstandinghttp.Caller{Login: "ADMINPNC"}

// ---------------------------------------------------------------------------
// Ringkasan
// ---------------------------------------------------------------------------

func TestSummaryReturnsEveryTabWithCountsAndOwner(t *testing.T) {
	lengkap := sampleClaim("0001", "006")
	lengkap.DocumentComplete = true
	server := testServer(t, "ADMINPNC", lengkap, sampleClaim("0002", "006"))

	res := get(t, server, "/api/inbox-outstanding/ringkasan")
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Status []struct {
			Kode         string `json:"kode"`
			Judul        string `json:"judul"`
			Jumlah       *int   `json:"jumlah"`
			DapatDipilih bool   `json:"dapat_dipilih"`
		} `json:"status"`
		Total   int    `json:"total"`
		Pemilik string `json:"pemilik"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))

	require.Equal(t, 2, body.Total)
	require.Equal(t, "ADMINPNC", body.Pemilik)
	require.Len(t, body.Status, 9)

	require.Equal(t, "lengkap", body.Status[0].Kode)
	require.Equal(t, "Complete documents", body.Status[0].Judul)
	require.True(t, body.Status[0].DapatDipilih)
	require.Equal(t, 1, *body.Status[0].Jumlah)
	require.Equal(t, 1, *body.Status[1].Jumlah)

	// Tab yang belum dapat dihitung dikirim tanpa jumlah dan tidak dapat dipilih.
	require.Equal(t, "loss-adjuster", body.Status[4].Kode)
	require.Nil(t, body.Status[4].Jumlah)
	require.False(t, body.Status[4].DapatDipilih)
}

func TestSummaryRejectsAMalformedQuery(t *testing.T) {
	server := testServer(t, "ADMINPNC")

	res := get(t, server, "/api/inbox-outstanding/ringkasan?batas=banyak")
	require.Equal(t, http.StatusBadRequest, res.Code)
	require.Equal(t, outstandinghttp.CodeBadRequest, decodeCode(t, res.Body.Bytes()))
}

func TestSummaryAnswersAServiceFailureAsInternalError(t *testing.T) {
	var logs bytes.Buffer
	handler := fakeHandler(fakeService{summary: func(usecase.Query) (usecase.SummaryResult, error) {
		return usecase.SummaryResult{}, errors.New("basis data mati")
	}}, &logs, admin)

	recorder := httptest.NewRecorder()
	handler.Summary(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/ringkasan", nil)))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, outstandinghttp.CodeInternalError, decodeCode(t, recorder.Body.Bytes()))
	// Rincian galat hanya masuk log, tidak pernah ke peramban.
	require.NotContains(t, recorder.Body.String(), "basis data mati")
	require.Contains(t, logs.String(), "permintaan gagal")
	require.Contains(t, logs.String(), "basis data mati")
}

// ---------------------------------------------------------------------------
// Penyaring daftar
// ---------------------------------------------------------------------------

func TestListPassesTheScreenFiltersToTheService(t *testing.T) {
	var got usecase.Query
	handler := fakeHandler(fakeService{list: func(q usecase.Query) (usecase.Result, error) {
		got = q
		return usecase.Result{AssignedTo: q.LoginID}, nil
	}}, &bytes.Buffer{}, admin)

	recorder := httptest.NewRecorder()
	handler.List(recorder, withPortal(httptest.NewRequest(http.MethodGet,
		"/api/inbox-outstanding?cari=%20POL-1%20&tahap=Komite&cabang=JKT&status_dokumen=LENGKAP&batas=10&lewati=20", nil)))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, usecase.Query{
		LoginID:        "ADMINPNC",
		PortalAlias:    "ASM",
		Search:         "POL-1",
		Stage:          "Komite",
		BranchCode:     "JKT",
		DocumentStatus: inboxoutstanding.StatusComplete,
		Limit:          10,
		Offset:         20,
	}, got)
}

func TestListRejectsUnknownAndUncountableDocumentStatuses(t *testing.T) {
	server := testServer(t, "ADMINPNC")

	for _, path := range []string{
		"/api/inbox-outstanding?status_dokumen=entah",
		"/api/inbox-outstanding?status_dokumen=tka",
	} {
		res := get(t, server, path)
		require.Equal(t, http.StatusBadRequest, res.Code, path)
		require.Equal(t, outstandinghttp.CodeBadRequest, decodeCode(t, res.Body.Bytes()), path)
	}
}

func TestListRejectsAnUnknownCaller(t *testing.T) {
	handler := fakeHandler(fakeService{}, &bytes.Buffer{}, nil)

	recorder := httptest.NewRecorder()
	handler.List(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil)))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "identitas pemanggil tidak dikenali")

	blank := &outstandinghttp.Caller{Login: "  "}
	handler = fakeHandler(fakeService{}, &bytes.Buffer{}, blank)
	recorder = httptest.NewRecorder()
	handler.List(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil)))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// Handler yang dipasang di luar middleware portal tetap menolak, tidak dilayani portal utama.
func TestHandlersRejectAMissingActivePortal(t *testing.T) {
	handler := fakeHandler(fakeService{}, &bytes.Buffer{}, admin)

	for name, serve := range map[string]http.HandlerFunc{
		"daftar":    handler.List,
		"ringkasan": handler.Summary,
		"unduh":     handler.Export,
	} {
		recorder := httptest.NewRecorder()
		serve(recorder, httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code, name)
		require.Contains(t, recorder.Body.String(), "portal aktif tidak dikenali", name)
	}
}

func TestListAnswersAServiceFailureThroughTheFallbackWriter(t *testing.T) {
	var handed error
	cause := errors.New("koneksi putus")
	handler := outstandinghttp.NewHandler(outstandinghttp.Options{
		Service: fakeService{list: func(usecase.Query) (usecase.Result, error) {
			return usecase.Result{}, cause
		}},
		GetCaller: func(context.Context) (outstandinghttp.Caller, bool) { return *admin, true },
		WriteJSON: jsonWriter,
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			handed = err
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	})

	recorder := httptest.NewRecorder()
	handler.List(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil)))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, cause, handed)
}

// Tanpa zona waktu yang ditetapkan, tampilan memakai WIB — bukan zona mesin penjalan.
func TestHandlerDefaultsToJakartaTime(t *testing.T) {
	handler := outstandinghttp.NewHandler(outstandinghttp.Options{
		Service: fakeService{list: func(q usecase.Query) (usecase.Result, error) {
			return usecase.Result{
				Page:       inboxoutstanding.Page{Claims: []inboxoutstanding.OutstandingClaim{sampleClaim("0001", "006")}, Total: 1},
				AssignedTo: q.LoginID,
			}, nil
		}},
		GetCaller: func(context.Context) (outstandinghttp.Caller, bool) { return *admin, true },
		WriteJSON: jsonWriter,
	})

	recorder := httptest.NewRecorder()
	handler.List(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding", nil)))
	require.Equal(t, http.StatusOK, recorder.Code)

	var body struct {
		Klaim []struct {
			TanggalPendaftaran string `json:"tanggal_pendaftaran"`
		} `json:"klaim"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	// 15 September 20.00 UTC adalah 16 September WIB.
	require.Equal(t, "2026-09-16", body.Klaim[0].TanggalPendaftaran)
}

// ---------------------------------------------------------------------------
// Unduhan
// ---------------------------------------------------------------------------

func TestExportPassesTheDateRangeWithAnExclusiveUpperBound(t *testing.T) {
	var got usecase.ExportQuery
	handler := fakeHandler(fakeService{export: func(q usecase.ExportQuery) (usecase.ExportResult, error) {
		got = q
		return usecase.ExportResult{}, nil
	}}, &bytes.Buffer{}, admin)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet,
		"/api/inbox-outstanding/unduh?dari=2026-09-10&sampai=2026-09-10&batas=3&lewati=9", nil)))
	require.Equal(t, http.StatusOK, recorder.Code)

	require.Equal(t, "ADMINPNC", got.LoginID)
	require.Equal(t, "ASM", got.PortalAlias)
	require.Equal(t, time.Date(2026, time.September, 10, 0, 0, 0, 0, wib), *got.From)
	require.Equal(t, time.Date(2026, time.September, 11, 0, 0, 0, 0, wib), *got.To)
	require.Zero(t, got.Offset, "halaman diabaikan")
	require.Equal(t, 500, got.Limit)
	require.Equal(t, `attachment; filename="inbox-outstanding-20260920-100000.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
}

func TestExportRejectsAnUnknownCaller(t *testing.T) {
	handler := fakeHandler(fakeService{}, &bytes.Buffer{}, nil)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "identitas pemanggil tidak dikenali")
}

func TestExportAnswersAFirstBatchFailureBeforeWritingTheFile(t *testing.T) {
	var logs bytes.Buffer
	handler := fakeHandler(fakeService{export: func(usecase.ExportQuery) (usecase.ExportResult, error) {
		return usecase.ExportResult{}, errors.New("basis data mati")
	}}, &logs, admin)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Empty(t, recorder.Header().Get("Content-Disposition"))
}

// batch menyusun n klaim untuk satu kumpulan unduhan.
func batch(n int, prefix string) []inboxoutstanding.OutstandingClaim {
	claims := make([]inboxoutstanding.OutstandingClaim, n)
	for i := range claims {
		claims[i] = sampleClaim(prefix, "006")
	}
	return claims
}

func TestExportFollowsEveryBatch(t *testing.T) {
	var offsets []int
	handler := fakeHandler(fakeService{export: func(q usecase.ExportQuery) (usecase.ExportResult, error) {
		offsets = append(offsets, q.Offset)
		if q.Offset == 0 {
			return usecase.ExportResult{Page: inboxoutstanding.Page{Claims: batch(500, "A"), Total: 503}}, nil
		}
		return usecase.ExportResult{Page: inboxoutstanding.Page{Claims: batch(3, "B"), Total: 503}}, nil
	}}, &bytes.Buffer{}, admin)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))
	require.Equal(t, http.StatusOK, recorder.Code)

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+503)
	require.Equal(t, []int{0, 500}, offsets)
}

// Unduhan berhenti pada batas pengaman, tanpa meminta kumpulan berikutnya.
func TestExportStopsAtTheMaximumRowCount(t *testing.T) {
	full := batch(500, "X")
	calls := 0
	handler := fakeHandler(fakeService{export: func(usecase.ExportQuery) (usecase.ExportResult, error) {
		calls++
		return usecase.ExportResult{Page: inboxoutstanding.Page{Claims: full, Total: 20_000}}, nil
	}}, &bytes.Buffer{}, admin)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))
	require.Equal(t, http.StatusOK, recorder.Code)

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+10_000)
	require.Equal(t, 20, calls)
}

func TestExportStopsAndLogsWhenALaterBatchFails(t *testing.T) {
	var logs bytes.Buffer
	handler := fakeHandler(fakeService{export: func(q usecase.ExportQuery) (usecase.ExportResult, error) {
		if q.Offset > 0 {
			return usecase.ExportResult{}, errors.New("putus di kumpulan kedua")
		}
		return usecase.ExportResult{Page: inboxoutstanding.Page{Claims: batch(500, "A"), Total: 900}}, nil
	}}, &logs, admin)

	recorder := httptest.NewRecorder()
	handler.Export(recorder, withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))

	// Badan sudah mengalir, sehingga statusnya tetap 200.
	require.Equal(t, http.StatusOK, recorder.Code)
	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 1+500)
	require.Contains(t, logs.String(), "export berhenti sebelum selesai")
	require.Contains(t, logs.String(), "putus di kumpulan kedua")
}

// failingWriter menolak setiap penulisan badan, seperti sambungan yang sudah ditutup.
type failingWriter struct {
	header http.Header
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(int)           {}
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("sambungan ditutup") }

// Baris yang lebih besar daripada penyangga memaksa penulisan langsung, sehingga sambungan
// yang putus terbaca saat baris itu ditulis.
func TestExportLogsAFailedRowWrite(t *testing.T) {
	var logs bytes.Buffer
	huge := sampleClaim("0001", "006")
	huge.InsuredName = strings.Repeat("x", 8192)
	handler := fakeHandler(fakeService{export: func(usecase.ExportQuery) (usecase.ExportResult, error) {
		return usecase.ExportResult{Page: inboxoutstanding.Page{
			Claims: []inboxoutstanding.OutstandingClaim{huge}, Total: 1,
		}}, nil
	}}, &logs, admin)

	handler.Export(&failingWriter{header: http.Header{}},
		withPortal(httptest.NewRequest(http.MethodGet, "/api/inbox-outstanding/unduh", nil)))

	require.Contains(t, logs.String(), "export berhenti sebelum selesai")
	require.Contains(t, logs.String(), "sambungan ditutup")
}
