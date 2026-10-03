package reportkpihttp_test

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

	"claim-pnc/internal/portal"
	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/repo/memory"
	"claim-pnc/internal/reportkpi/usecase"

	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
	reportkpihttp "claim-pnc/internal/reportkpi/http"
)

// Uji lapisan transport modul Report KPI: rute, status HTTP, isi JSON, dan berkas ekspor.
//
// Middleware autentikasi diganti satu middleware kecil yang membaca nama pengguna dari
// header — yang diuji di sini kontrak modul ini, bukan modul auth.

const headerLogin = "X-Test-Login"

type loginKey struct{}

var errRepo = errors.New("penyimpanan gagal")

// harness memegang router beserta jejak log-nya.
type harness struct {
	router  http.Handler
	handler *reportkpihttp.Handler
	logs    *bytes.Buffer
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// portalFallback menjawab galat portal, meniru penulis galat portal di cmd.
func portalFallback(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, portal.ErrNotStated) || errors.Is(err, portal.ErrNotReady) {
		writeJSON(w, r, http.StatusBadRequest, map[string]string{"kode": "portal_ditolak"})
		return
	}
	writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
}

func newHarness(t *testing.T, repo reportkpi.Repo) *harness {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (reportkpi.Repo, error) {
			if alias == "ASM" {
				return repo, nil
			}
			return nil, portal.ErrNotReady
		},
		Logger: logger,
	})
	require.NoError(t, err)

	handler := reportkpihttp.NewHandler(reportkpihttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (reportkpihttp.Caller, bool) {
			login, ok := ctx.Value(loginKey{}).(string)
			return reportkpihttp.Caller{Login: login}, ok
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: portalFallback,
	})

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if login := r.Header.Get(headerLogin); login != "" {
				r = r.WithContext(context.WithValue(r.Context(), loginKey{}, login))
			}
			next.ServeHTTP(w, r)
		})
	})
	reportkpihttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   portalFallback,
	})

	return &harness{router: router, handler: handler, logs: logs}
}

// do mengirim permintaan sebagai pengguna sah pada portal ASM.
func (h *harness) do(method, target string) *httptest.ResponseRecorder {
	return h.doAs(method, target, "ASM", "PENYELIACONTOH")
}

func (h *harness) doAs(method, target, portalAlias, login string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	if portalAlias != "" {
		req.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	if login != "" {
		req.Header.Set(headerLogin, login)
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body
}

func readCSV(t *testing.T, rec *httptest.ResponseRecorder) [][]string {
	t.Helper()
	records, err := csv.NewReader(bytes.NewReader(rec.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	return records
}

const period = "dari=2026-03-01&sampai=2026-03-31"

// --- Metadata dan penjaga umum ---

func TestMetadataReturnsScreenDescription(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/tab")
	require.Equal(t, http.StatusOK, rec.Code)

	body := decode(t, rec)
	require.Equal(t, reportkpi.DefaultTabKPI, body["tab_bawaan"])
	require.Equal(t, reportkpi.SourceTable, body["tabel_sumber"])
	require.Equal(t, reportkpi.BandTable, body["tabel_tangga_nilai"])
	require.Equal(t, reportkpi.CoordinatorInQuery, body["koordinator_di_kueri"])
	require.Len(t, body["komponen"], 9)
	require.Len(t, body["tipe_report"], 3)
	require.Len(t, body["kelompok_admin"], 2)
	require.Len(t, body["tab"], len(reportkpi.Tabs()))
	require.Len(t, body["komponen_pic"], len(reportkpi.PICComponents()))
	require.Len(t, body["lini_bisnis"], len(reportkpi.BusinessLines()))
	require.NotEmpty(t, body["pic_dikecualikan_sla"])

	// Kolom TIPE pada grid Summary ditandai hanya untuk tipe gabungan.
	tabs := body["tab"].([]any)
	found := false
	for _, tab := range tabs {
		for _, grid := range tab.(map[string]any)["grid"].([]any) {
			for _, column := range grid.(map[string]any)["kolom"].([]any) {
				c := column.(map[string]any)
				if c["kunci"] == reportkpi.FieldType && c["hanya_tipe_gabungan"] == true {
					found = true
				}
			}
		}
	}
	require.True(t, found)
}

// Tanpa header portal, middleware menolak lebih dulu.
func TestRequestsWithoutPortalAreRejected(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.doAs(http.MethodGet, "/report-kpi/tab", "", "PENYELIACONTOH")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "portal_ditolak", decode(t, rec)["kode"])
}

// Handler yang dipanggil tanpa middleware portal tetap menolak, bukan jatuh ke bawaan.
func TestHandlersGuardMissingPortalThemselves(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	handlers := map[string]http.HandlerFunc{
		"Metadata":    h.handler.Metadata,
		"Summary":     h.handler.Summary,
		"RejectWrite": h.handler.RejectWrite,
		"AdminDetail": h.handler.AdminDetail,
		"PICTeknik":   h.handler.PICTeknik,
	}
	for name, handle := range handlers {
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equalf(t, http.StatusBadRequest, rec.Code, name)
		require.Equalf(t, "portal_ditolak", decode(t, rec)["kode"], name)
	}
}

// Sesi tanpa nama pengguna dijawab 409, bukan 401.
func TestCallerUnknownIs409(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	targets := []string{
		"/report-kpi/adjuster/ringkasan", "/report-kpi/adjuster", "/report-kpi/adjuster/pilihan",
		"/report-kpi/adjuster/ekspor", "/report-kpi/admin/kartu-skor", "/report-kpi/admin",
		"/report-kpi/admin/ekspor", "/report-kpi/pic-teknik", "/report-kpi/pic-teknik/ekspor",
	}
	for _, target := range targets {
		rec := h.doAs(http.MethodGet, target, "ASM", "")
		require.Equalf(t, http.StatusConflict, rec.Code, target)
		require.Equalf(t, reportkpihttp.CodeCallerUnknown, decode(t, rec)["kode"], target)
	}
}

// Handler tanpa pembaca identitas memperlakukan setiap pemanggil sebagai tidak dikenal.
func TestHandlerWithoutCallerReaderRejects(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (reportkpi.Repo, error) { return memory.NewSampleStore(), nil },
	})
	require.NoError(t, err)
	handler := reportkpihttp.NewHandler(reportkpihttp.Options{Service: service, WriteJSON: writeJSON})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req = req.WithContext(portalhttp.WithActivePortal(req.Context(), portal.Portal{Alias: "ASM"}))
	rec := httptest.NewRecorder()
	handler.Summary(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)
}

// Galat dari pemilihan penyimpanan diteruskan ke penulis cadangan.
func TestRepoSelectorErrorGoesToFallback(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	// ASI lolos middleware portal, tetapi RepoSelector menolaknya.
	rec := h.doAs(http.MethodGet, "/report-kpi/adjuster/ringkasan?tipe_report=FINAL&"+period,
		"ASI", "PENYELIACONTOH")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRejectWriteIs501AndLogged(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodPost, "/report-kpi/tindakan?tindakan=%20hitung%20")
	require.Equal(t, http.StatusNotImplemented, rec.Code)
	body := decode(t, rec)
	require.Equal(t, reportkpihttp.CodeWriteNotAvailable, body["kode"])
	require.Contains(t, body["pesan"], reportkpi.SourceTable)
	require.Contains(t, h.logs.String(), `"tindakan":"hitung"`)
}

func TestWriteErrorWithoutFallbackAnswers500(t *testing.T) {
	logs := &bytes.Buffer{}
	write := reportkpihttp.WriteError(slog.New(slog.NewJSONHandler(logs, nil)), writeJSON, nil)

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/jalur", nil), errors.New("rahasia internal"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body := decode(t, rec)
	require.Equal(t, reportkpihttp.CodeInternalError, body["kode"])
	require.NotContains(t, rec.Body.String(), "rahasia internal")
	require.Contains(t, logs.String(), "rahasia internal")

	// Tanpa logger pun tidak panik.
	rec = httptest.NewRecorder()
	reportkpihttp.WriteError(nil, writeJSON, nil)(rec, httptest.NewRequest(http.MethodGet, "/", nil), errRepo)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// RejectWrite tanpa logger tetap menjawab 501.
func TestRejectWriteWithoutLogger(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (reportkpi.Repo, error) { return memory.NewStore(), nil },
	})
	require.NoError(t, err)
	handler := reportkpihttp.NewHandler(reportkpihttp.Options{Service: service, WriteJSON: writeJSON})

	req := httptest.NewRequest(http.MethodPost, "/report-kpi/tindakan", nil)
	req = req.WithContext(portalhttp.WithActivePortal(req.Context(), portal.Portal{Alias: "ASM"}))
	rec := httptest.NewRecorder()
	handler.RejectWrite(rec, req)
	require.Equal(t, http.StatusNotImplemented, rec.Code)
}

// --- Tab Adjuster ---

func TestSummaryReturnsRowsWithNullScores(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster/ringkasan?tipe_report=outstanding&"+period)
	require.Equal(t, http.StatusOK, rec.Code)

	body := decode(t, rec)
	require.Equal(t, "ASM", body["portal"])
	filter := body["penyaring"].(map[string]any)
	require.Equal(t, "OUTSTANDING", filter["tipe_report"])
	require.Equal(t, "2026-03-01", filter["dari"])

	rows := body["baris"].([]any)
	require.Len(t, rows, 2)
	cv := rows[0].(map[string]any)
	require.Equal(t, "CV SURVEI CONTOH SEJAHTERA", cv["adjuster"])
	scores := cv["nilai"].(map[string]any)
	require.Len(t, scores, 9)
	require.Nil(t, scores[reportkpi.ComponentInterimReport], "komponen tanpa nilai menjadi null")
	require.Equal(t, 4.0, scores[reportkpi.ComponentSurvey])
}

// Seluruh pelanggaran dikirim sekaligus sebagai 422.
func TestSummaryValidationIs422WithAllViolations(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster/ringkasan")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := decode(t, rec)
	require.Equal(t, reportkpihttp.CodeValidationFail, body["kode"])
	require.Len(t, body["detail"], 3)
}

func TestDetailPaginates(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster?tipe_report=ALL&halaman=2&ukuran=3&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, map[string]any{
		"halaman": 2.0, "ukuran": 3.0, "total": 7.0, "total_halaman": 3.0,
	}, body["paginasi"])
	rows := body["baris"].([]any)
	require.Len(t, rows, 3)
	first := rows[0].(map[string]any)
	require.NotEmpty(t, first["no_case"])
	require.NotEmpty(t, first["tanggal"])
}

// Angka halaman yang tidak terbaca atau negatif dibetulkan, bukan ditolak.
func TestDetailNormalizesBadPagination(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster?tipe_report=FINAL&halaman=abc&ukuran=-5&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	pagination := decode(t, rec)["paginasi"].(map[string]any)
	require.Equal(t, 1.0, pagination["halaman"])
	require.Equal(t, float64(reportkpi.DefaultPageSize), pagination["ukuran"])
}

func TestDetailValidationIs422(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())
	rec := h.do(http.MethodGet, "/report-kpi/adjuster?tipe_report=FINAL")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// Penyaring yang dikirim balik daftar pilihan tidak memuat adjuster.
func TestAdjustersListIgnoresChosenAdjuster(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet,
		"/report-kpi/adjuster/pilihan?tipe_report=FINAL&adjuster=PT%20TEPI%20CONTOH%20MANDIRI&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, []any{
		"PT ADJUSTER NUSA CONTOH", "PT PENILAI CONTOH PRATAMA", "PT TEPI CONTOH MANDIRI",
	}, body["adjuster"])
	require.Equal(t, "", body["penyaring"].(map[string]any)["adjuster"])
}

func TestAdjustersValidationIs422(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())
	rec := h.do(http.MethodGet, "/report-kpi/adjuster/pilihan")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// Ekspor ringkasan bawaan: kolom TIPE hanya muncul pada tipe ALL.
func TestExportSummaryCSV(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster/ekspor?tipe_report=FINAL&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="ringkasan-kpi-adjuster-final-2026-03-01-sd-2026-03-31.csv"`,
		rec.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	records := readCSV(t, rec)
	require.Equal(t, "ADJUSTER", records[0][0])
	require.Len(t, records[0], 1+9)
	require.Len(t, records, 1+3)
	// PT PENILAI punya satu komponen "N/A" yang menjadi sel kosong.
	require.Equal(t, "PT PENILAI CONTOH PRATAMA", records[2][0])
	require.Equal(t, "", records[2][3])

	rec = h.do(http.MethodGet, "/report-kpi/adjuster/ekspor?tipe_report=ALL&"+period)
	records = readCSV(t, rec)
	require.Equal(t, []string{"ADJUSTER", "TIPE"}, records[0][:2])
	require.Contains(t, []string{"OUTSTANDING", "FINAL"}, records[1][1])
}

func TestExportDetailCSV(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/adjuster/ekspor?grid=%20rincian%20&tipe_report=FINAL&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Disposition"), "rincian-kpi-adjuster-final-")

	records := readCSV(t, rec)
	require.Equal(t, []string{"ADJUSTER", "NO CASE", "TIPE", "TANGGAL"}, records[0][:4])
	require.Len(t, records, 1+4)
	require.Equal(t, []string{"PT ADJUSTER NUSA CONTOH", "CONTOH-KPI-0003", "FINAL", "2026-03-25"},
		records[1][:4])
}

// Galat sebelum unduhan dimulai tetap dijawab sebagai JSON.
func TestExportValidationIsJSONError(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	for _, target := range []string{
		"/report-kpi/adjuster/ekspor", "/report-kpi/adjuster/ekspor?grid=rincian",
		"/report-kpi/admin/ekspor", "/report-kpi/pic-teknik/ekspor",
	} {
		rec := h.do(http.MethodGet, target)
		require.Equalf(t, http.StatusUnprocessableEntity, rec.Code, target)
		require.Equal(t, reportkpihttp.CodeValidationFail, decode(t, rec)["kode"])
	}
}

// --- Tab KPI Admin ---

func TestScorecardNonMBU(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/admin/kartu-skor?kelompok=nonmbu&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, "01/03/2026 - 31/03/2026", body["tanggal_efektif"])
	require.Equal(t, "MORASOTARDODOTARIGAN", body["identitas"].(map[string]any)["nama_koordinator"])
	require.Equal(t, map[string]any{"kelompok": "NONMBU", "dari": "2026-03-01", "sampai": "2026-03-31"},
		body["penyaring"])
	metrics := body["metrik"].([]any)
	require.Len(t, metrics, 14)
	first := metrics[0].(map[string]any)
	require.Equal(t, reportkpi.MetricLeaderOverSLA, first["kode"])
	require.Equal(t, 1.0, first["nilai"])
	require.Equal(t, string(reportkpi.FormatCount), first["bentuk"])
	require.NotEmpty(t, body["achievement"])
}

// Periode tanpa klaim: persentase tidak dapat dihitung dan dikirim sebagai null.
func TestScorecardEmptyPeriodSendsNull(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/admin/kartu-skor?kelompok=NONMBU&dari=2025-01-01&sampai=2025-01-31")
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	metrics := body["metrik"].([]any)
	require.Nil(t, metrics[2].(map[string]any)["nilai"])
	_, hasAchievement := body["achievement"]
	require.False(t, hasAchievement)
}

func TestScorecardValidationIs422(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())
	rec := h.do(http.MethodGet, "/report-kpi/admin/kartu-skor?kelompok=MBU&"+period)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAdminDetailPA(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/admin?kelompok=PA&ukuran=2&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, 4.0, body["paginasi"].(map[string]any)["total"])
	rows := body["baris"].([]any)
	require.Len(t, rows, 2)
	row := rows[0].(map[string]any)
	require.Equal(t, "CONTOH-ADM-0106", row["no_klaim"])
	require.Equal(t, 0.25, row["aging_regist_klaim"])
	require.Equal(t, "SLA", row["status_sla_regist"])
}

func TestAdminDetailValidationIs422(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())
	rec := h.do(http.MethodGet, "/report-kpi/admin?kelompok=PA")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAdminExportCSVPerGroup(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=NONMBU&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, `attachment; filename="rincian-kpi-admin-NONMBU-2026-03-01-sd-2026-03-31.csv"`,
		rec.Header().Get("Content-Disposition"))
	records := readCSV(t, rec)
	require.Equal(t, []string{
		"No Klaim", "No Polis", "BUSINESS", "Tgl Regist Klaim", "Tgl Terima Dokumen", "Flag",
		"Aging Regist Klaim",
	}, records[0])
	require.Len(t, records, 1+7)
	require.Equal(t, []string{
		"CONTOH-ADM-0007", "CONTOH-POL-0007", "BISNIS CONTOH NON MBU", "2026-03-28",
		"2026-03-28", "member", "1",
	}, records[1])

	rec = h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=PA&"+period)
	records = readCSV(t, rec)
	require.Equal(t, []string{
		"No Klaim", "No Polis", "Tgl Regist Klaim", "Aging Regist Klaim", "Tgl Terima Dokumen",
		"Status SLA Regist Klaim", "Tgl Terima LOD", "Tgl Pembayaran", "Aging Pembayaran klaim",
		"Status SLA Pembayaran Klaim",
	}, records[0])
	require.Len(t, records, 1+4)
}

// --- Tab KPI PIC Teknik ---

func TestPICTeknikReturnsScorecards(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/pic-teknik?lini_bisnis=NONMBU&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	body := decode(t, rec)
	require.Equal(t, map[string]any{"lini_bisnis": "NONMBU", "dari": "2026-03-01", "sampai": "2026-03-31"},
		body["penyaring"])
	cards := body["kartu_skor"].([]any)
	require.Len(t, cards, 2)
	first := cards[0].(map[string]any)
	require.Equal(t, "CONTOHPICDUA", first["pic"])
	require.NotEmpty(t, first["baris"])
	require.NotNil(t, body["rekapitulasi"].(map[string]any)["baris"])
}

func TestPICTeknikValidationIs422(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())
	rec := h.do(http.MethodGet, "/report-kpi/pic-teknik?lini_bisnis=MBU&"+period)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// Berkas ekspor meratakan kartu skor: satu baris per PIC per komponen, ditambah rekapitulasi.
func TestPICTeknikExportCSV(t *testing.T) {
	h := newHarness(t, memory.NewSampleStore())

	rec := h.do(http.MethodGet, "/report-kpi/pic-teknik/ekspor?lini_bisnis=NONMBU&"+period)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, `attachment; filename="kpi-pic-teknik-NONMBU-2026-03-01-sd-2026-03-31.csv"`,
		rec.Header().Get("Content-Disposition"))
	records := readCSV(t, rec)
	require.Equal(t, []string{"PIC", "KPI", "Total", "Tercapai", "Persentase", "Nilai"}, records[0])
	require.Greater(t, len(records), 4)
	require.Equal(t, "CONTOHPICDUA", records[1][0])
}

// --- Ekspor bervolume besar dan kegagalan di tengah unduhan ---

// bulkRepo menghasilkan baris buatan tanpa menyimpannya, supaya batas ekspor dapat diuji.
type bulkRepo struct {
	*memory.Store
	summaryRows int
	detailTotal int
	adminTotal  int
	failPage    int
}

func (r bulkRepo) Summary(context.Context, reportkpi.Query) ([]reportkpi.AdjusterSummary, error) {
	rows := make([]reportkpi.AdjusterSummary, r.summaryRows)
	for i := range rows {
		rows[i] = reportkpi.AdjusterSummary{
			Adjuster:   fmt.Sprintf("ADJUSTER CONTOH NOMOR %06d", i),
			ReportType: reportkpi.TypeFinal,
		}
	}
	return rows, nil
}

func pageBounds(page reportkpi.Pagination, total int) (int, int) {
	clean := page.Normalize()
	from := clean.Offset()
	to := from + clean.Size
	if from > total {
		from = total
	}
	if to > total {
		to = total
	}
	return from, to
}

func (r bulkRepo) Detail(
	_ context.Context, _ reportkpi.Query, page reportkpi.Pagination,
) (reportkpi.DetailPage, error) {
	if page.Page == r.failPage {
		return reportkpi.DetailPage{}, errRepo
	}
	from, to := pageBounds(page, r.detailTotal)
	rows := make([]reportkpi.AdjusterDetail, 0, to-from)
	for i := from; i < to; i++ {
		rows = append(rows, reportkpi.AdjusterDetail{
			Adjuster: "A", CaseID: fmt.Sprintf("C%d", i), ReportType: reportkpi.TypeFinal,
		})
	}
	return reportkpi.DetailPage{Rows: rows, Total: r.detailTotal}, nil
}

func (r bulkRepo) AdminDetail(
	_ context.Context, _ reportkpi.AdminQuery, page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	if page.Page == r.failPage {
		return reportkpi.AdminDetailPage{}, errRepo
	}
	from, to := pageBounds(page, r.adminTotal)
	rows := make([]reportkpi.AdminDetailRow, 0, to-from)
	for i := from; i < to; i++ {
		rows = append(rows, reportkpi.AdminDetailRow{ClaimNumber: fmt.Sprintf("K%d", i)})
	}
	return reportkpi.AdminDetailPage{Rows: rows, Total: r.adminTotal}, nil
}

const exportLimit = 50_000

func TestExportsAreTruncatedWithNotice(t *testing.T) {
	repo := bulkRepo{
		Store: memory.NewStore(), summaryRows: exportLimit + 1,
		detailTotal: exportLimit + 5, adminTotal: exportLimit + 7,
	}
	h := newHarness(t, repo)
	notice := func(total int) string {
		return fmt.Sprintf("-- Terpotong pada %d baris dari %d yang cocok. Persempit periodenya. --",
			exportLimit, total)
	}

	records := readCSV(t, h.do(http.MethodGet, "/report-kpi/adjuster/ekspor?tipe_report=FINAL&"+period))
	require.Len(t, records, 1+exportLimit+1)
	require.Equal(t, notice(exportLimit+1), records[len(records)-1][0])

	records = readCSV(t, h.do(http.MethodGet,
		"/report-kpi/adjuster/ekspor?grid=rincian&tipe_report=FINAL&"+period))
	require.Len(t, records, 1+exportLimit+1)
	require.Equal(t, notice(exportLimit+5), records[len(records)-1][0])

	records = readCSV(t, h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=PA&"+period))
	require.Len(t, records, 1+exportLimit+1)
	require.Equal(t, notice(exportLimit+7), records[len(records)-1][0])
}

// Potongan berikutnya yang gagal menghentikan berkas dan meninggalkan jejak di log.
func TestExportStopsWhenALaterPageFails(t *testing.T) {
	repo := bulkRepo{Store: memory.NewStore(), detailTotal: 250, adminTotal: 250, failPage: 2}
	h := newHarness(t, repo)

	records := readCSV(t, h.do(http.MethodGet,
		"/report-kpi/adjuster/ekspor?grid=rincian&tipe_report=FINAL&"+period))
	require.Len(t, records, 1+reportkpi.MaxPageSize)

	records = readCSV(t, h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=PA&"+period))
	require.Len(t, records, 1+reportkpi.MaxPageSize)

	require.Equal(t, 2, strings.Count(h.logs.String(), "ekspor Report KPI terputus"))
}

// Seluruh halaman terbaca bila tidak ada yang gagal.
func TestExportReadsEveryPage(t *testing.T) {
	repo := bulkRepo{Store: memory.NewStore(), detailTotal: 250, adminTotal: 201}
	h := newHarness(t, repo)

	records := readCSV(t, h.do(http.MethodGet,
		"/report-kpi/adjuster/ekspor?grid=rincian&tipe_report=FINAL&"+period))
	require.Len(t, records, 1+250)
	require.Equal(t, "C249", records[250][1])

	records = readCSV(t, h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=NONMBU&"+period))
	require.Len(t, records, 1+201)
}

// Halaman kosong padahal jumlahnya belum tercapai: ekspor berhenti, tidak berputar.
func TestExportStopsOnEmptyPage(t *testing.T) {
	h := newHarness(t, emptyPageRepo{Store: memory.NewStore()})

	records := readCSV(t, h.do(http.MethodGet,
		"/report-kpi/adjuster/ekspor?grid=rincian&tipe_report=FINAL&"+period))
	require.Len(t, records, 1)

	records = readCSV(t, h.do(http.MethodGet, "/report-kpi/admin/ekspor?kelompok=PA&"+period))
	require.Len(t, records, 1)
}

// emptyPageRepo mengaku punya baris tetapi tidak mengembalikan satu pun.
type emptyPageRepo struct{ *memory.Store }

func (emptyPageRepo) Detail(context.Context, reportkpi.Query, reportkpi.Pagination) (reportkpi.DetailPage, error) {
	return reportkpi.DetailPage{Total: 10}, nil
}

func (emptyPageRepo) AdminDetail(
	context.Context, reportkpi.AdminQuery, reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	return reportkpi.AdminDetailPage{Total: 10}, nil
}

// failingWriter adalah ResponseWriter yang gagal setiap kali ditulisi.
type failingWriter struct{ header http.Header }

func (w *failingWriter) Header() http.Header       { return w.header }
func (w *failingWriter) Write([]byte) (int, error) { return 0, errors.New("koneksi putus") }
func (w *failingWriter) WriteHeader(int)           {}

func serveFailing(h *harness, target string) {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	req.Header.Set(headerLogin, "PENYELIACONTOH")
	h.router.ServeHTTP(&failingWriter{header: http.Header{}}, req)
}

// Koneksi yang putus di tengah unduhan dicatat di log, bukan dijawab sebagai galat.
func TestExportLogsBrokenConnection(t *testing.T) {
	repo := bulkRepo{Store: memory.NewSampleStore(), summaryRows: 500, detailTotal: 5, adminTotal: 5}
	h := newHarness(t, repo)

	// Ringkasan bervolume: penyangga CSV penuh dan baris berikutnya gagal ditulis.
	serveFailing(h, "/report-kpi/adjuster/ekspor?tipe_report=FINAL&"+period)
	// Rincian dan admin: kegagalan muncul saat potongan pertama didorong keluar.
	serveFailing(h, "/report-kpi/adjuster/ekspor?grid=rincian&tipe_report=FINAL&"+period)
	serveFailing(h, "/report-kpi/admin/ekspor?kelompok=PA&"+period)
	// PIC Teknik: kegagalan muncul saat berkas didorong keluar di akhir.
	serveFailing(h, "/report-kpi/pic-teknik/ekspor?lini_bisnis=NONMBU&"+period)

	require.Equal(t, 4, strings.Count(h.logs.String(), "ekspor Report KPI terputus"))
	require.Contains(t, h.logs.String(), "koneksi putus")
}
