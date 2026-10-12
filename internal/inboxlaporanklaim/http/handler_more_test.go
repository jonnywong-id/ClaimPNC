package inboxlaporanklaimhttp

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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/repo/memory"
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Uji handler secara langsung, tanpa modul auth.
//
// routes_test.go membuktikan kontrak dari luar; berkas ini menelusuri setiap cabang
// handler — termasuk yang tidak dapat dicapai lewat server utuh, seperti pemanggil tanpa
// identitas dan penulis jawaban yang gagal di tengah ekspor.

const testLogin = "adminpnc"

var testNow = time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC)

// harness merakit handler di atas repo memori dan penulis galat perekam.
type harness struct {
	router      chi.Router
	repo        *memory.Repo
	resolver    *memory.BranchResolver
	logs        *bytes.Buffer
	fallback    []error
	callerKnown bool
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	fixed := clock.FixedAt(testNow)
	h := &harness{
		repo:        memory.NewRepo(memory.SampleOptions(fixed)),
		resolver:    memory.NewBranchResolver(map[string]string{testLogin: "1001"}),
		logs:        &bytes.Buffer{},
		callerKnown: true,
	}
	return h.mount(t, func(alias string) (inboxlaporanklaim.Repo, error) {
		if alias != "ASM" {
			return nil, portal.ErrNotReady
		}
		return h.repo, nil
	})
}

// mount memasang handler dengan pemilih repo tertentu.
func (h *harness) mount(t *testing.T, selector inboxlaporanklaim.RepoSelector) *harness {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector:   selector,
		BranchResolver: h.resolver,
		Clock:          clock.FixedAt(testNow),
	})
	require.NoError(t, err)

	handler, err := NewHandler(Options{
		Service: service,
		Caller: func(context.Context) (inboxlaporanklaim.Caller, bool) {
			if !h.callerKnown {
				return inboxlaporanklaim.Caller{}, false
			}
			return inboxlaporanklaim.Caller{Login: testLogin, Name: "Admin PNC"}, true
		},
		Logger: slog.New(slog.NewJSONHandler(h.logs, nil)),
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			h.fallback = append(h.fallback, err)
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	require.NoError(t, err)

	router := chi.NewRouter()
	// Portal hanya ditaruh bila header-nya dikirim, meniru middleware ActivePortal.
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if alias := r.Header.Get("X-Test-Portal"); alias != "" {
				r = r.WithContext(portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: alias}))
			}
			next.ServeHTTP(w, r)
		})
	})
	router.Get("/inbox/laporan-klaim", handler.List)
	router.Post("/inbox/laporan-klaim", handler.Create)
	router.Get("/inbox/laporan-klaim/pilihan", handler.Options)
	router.Get("/inbox/laporan-klaim/ekspor", handler.Export)
	router.Get("/inbox/laporan-klaim/polis", handler.Policy)
	router.Get("/inbox/laporan-klaim/{id}", handler.Get)
	router.Put("/inbox/laporan-klaim/{id}", handler.Save)
	h.router = router
	return h
}

func (h *harness) do(method, path, alias, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if alias != "" {
		request.Header.Set("X-Test-Portal", alias)
	}
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	content := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &content), "badan: %s", recorder.Body.String())
	return content
}

func TestNewHandlerRequiresEveryDependency(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxlaporanklaim.Repo, error) { return nil, nil },
		Clock:        clock.FixedAt(testNow),
	})
	require.NoError(t, err)
	caller := func(context.Context) (inboxlaporanklaim.Caller, bool) { return inboxlaporanklaim.Caller{}, false }
	writeResponse := func(http.ResponseWriter, *http.Request, int, any) {}
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err = NewHandler(Options{Caller: caller, WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Service")
	_, err = NewHandler(Options{Service: service, WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Caller")
	_, err = NewHandler(Options{Service: service, Caller: caller, WriteError: writeError})
	require.ErrorContains(t, err, "WriteResponse")
	_, err = NewHandler(Options{Service: service, Caller: caller, WriteResponse: writeResponse})
	require.ErrorContains(t, err, "WriteError")
}

// Permintaan tanpa portal tidak pernah dilayani; galatnya diserahkan ke penulis bersama.
func TestEveryRouteRefusesRequestWithoutPortal(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/inbox/laporan-klaim"},
		{http.MethodPost, "/inbox/laporan-klaim"},
		{http.MethodGet, "/inbox/laporan-klaim/pilihan"},
		{http.MethodGet, "/inbox/laporan-klaim/ekspor"},
		{http.MethodGet, "/inbox/laporan-klaim/polis?nomor=1"},
		{http.MethodGet, "/inbox/laporan-klaim/RCV-0001"},
		{http.MethodPut, "/inbox/laporan-klaim/RCV-0001"},
	}
	for _, route := range routes {
		h := newHarness(t)
		recorder := h.do(route.method, route.path, "", "{}")
		require.Equalf(t, http.StatusInternalServerError, recorder.Code, "%s %s", route.method, route.path)
		require.Lenf(t, h.fallback, 1, "%s %s", route.method, route.path)
		require.ErrorIs(t, h.fallback[0], portal.ErrNotStated)
	}
}

// Pemanggil tanpa identitas ditolak dengan kode tersendiri pada rute yang membutuhkannya.
func TestRoutesNeedingCallerRefuseUnknownCaller(t *testing.T) {
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/inbox/laporan-klaim"},
		{http.MethodPost, "/inbox/laporan-klaim"},
		{http.MethodGet, "/inbox/laporan-klaim/ekspor"},
		{http.MethodPut, "/inbox/laporan-klaim/RCV-0001"},
	} {
		h := newHarness(t)
		h.callerKnown = false
		recorder := h.do(route.method, route.path, "ASM", "{}")
		require.Equal(t, http.StatusConflict, recorder.Code)
		require.Equal(t, CodeCallerIncomplete, decode(t, recorder)["kode"])
	}
}

func TestListAnswersPageCategoriesAndScope(t *testing.T) {
	h := newHarness(t)
	// Angka halaman yang tidak dapat dibaca diabaikan, bukan ditolak.
	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim?halaman=x&ukuran=2", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)

	var body ListResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "ASM", body.Portal)
	require.Equal(t, "1001", body.BatasCabang)
	require.Equal(t, PaginationDTO{Halaman: 1, Ukuran: 2, Total: 5, TotalHalaman: 3}, body.Halaman)
	require.Len(t, body.Laporan, 2)
	require.Equal(t, "RCV-0012", body.Laporan[0].ID)
	require.Equal(t, "2026-09-20", body.Laporan[0].TanggalAging)
	require.Equal(t, 2, body.Laporan[0].UmurHari)

	require.Len(t, body.Kategori, 9)
	counted := map[string]*int{}
	for _, c := range body.Kategori {
		counted[c.Kode] = c.Jumlah
	}
	require.NotNil(t, counted["semua"])
	require.Equal(t, 5, *counted["semua"])
	require.Nil(t, counted["ditolak"], "tab penolakan tidak pernah dicacah sistem lama")
}

func TestListRejectsUnknownFilters(t *testing.T) {
	h := newHarness(t)

	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim?kategori=entah", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	content := decode(t, recorder)
	require.Equal(t, CodeValidationFailed, content["kode"])
	require.Equal(t, "kategori", content["detail"].([]any)[0].(map[string]any)["kolom"])

	recorder = h.do(http.MethodGet, "/inbox/laporan-klaim?bisnis=entah", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, "bisnis", decode(t, recorder)["detail"].([]any)[0].(map[string]any)["kolom"])
}

func TestOptionsListsTabsBusinessLinesAndRegions(t *testing.T) {
	h := newHarness(t)
	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/pilihan", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)

	var body OptionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Kategori, 9)
	for _, c := range body.Kategori {
		require.Nil(t, c.Jumlah, "lencana pada pilihan sengaja kosong")
	}
	require.Len(t, body.Bisnis, 5)
	require.Equal(t, OptionDTO{Kode: "", Nama: "Semua bisnis"}, body.Bisnis[0])
	require.Equal(t, []OptionDTO{
		{Kode: "01", Nama: "Kanwil Jakarta"}, {Kode: "02", Nama: "Kanwil Jawa Barat"},
		{Kode: "03", Nama: "Kanwil Jawa Timur"},
	}, body.Kanwil)
	require.Equal(t, "ASM", body.Portal)
}

// Portal yang belum siap tidak dikenali modul ini dan diserahkan ke penulis bersama.
func TestPortalNotReadyIsHandedToSharedWriter(t *testing.T) {
	for _, path := range []string{
		"/inbox/laporan-klaim/pilihan",
		"/inbox/laporan-klaim/RCV-0001",
		"/inbox/laporan-klaim/polis?nomor=1",
		"/inbox/laporan-klaim",
	} {
		h := newHarness(t)
		recorder := h.do(http.MethodGet, path, "ASI", "")
		require.Equal(t, http.StatusInternalServerError, recorder.Code, path)
		require.Len(t, h.fallback, 1, path)
		require.ErrorIs(t, h.fallback[0], portal.ErrNotReady, path)
	}
}

func TestGetReadsLegacyReportAsReadOnly(t *testing.T) {
	h := newHarness(t)
	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/RCV-0001", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)

	var body SingleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "RCV-0001", body.Laporan.ID)
	require.Equal(t, "ASSIGN-WORKLIST ASM-FW-GCNMFW-WORK RCV-0001!ReceiveDocument_Flow", body.Laporan.RujukanPega)
	require.False(t, body.DapatDisunting)
	require.True(t, body.SudahDiregistrasi)
	require.NotNil(t, body.Isian)
	require.Equal(t, "POL-PA-0001", body.Isian.NomorPolis)
}

func TestGetUnknownOrBlankIDIsNotFound(t *testing.T) {
	for _, path := range []string{"/inbox/laporan-klaim/RCV-9999", "/inbox/laporan-klaim/%20"} {
		h := newHarness(t)
		recorder := h.do(http.MethodGet, path, "ASM", "")
		require.Equal(t, http.StatusNotFound, recorder.Code, path)
		require.Equal(t, CodeNotFound, decode(t, recorder)["kode"], path)
	}
}

func TestCreateAnswersCreatedWithEditableReport(t *testing.T) {
	h := newHarness(t)
	recorder := h.do(http.MethodPost, "/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusCreated, recorder.Code)

	var body SingleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "RCVN.26.1", body.Laporan.ID)
	require.True(t, body.DapatDisunting)
	require.False(t, body.SudahDiregistrasi)
	require.Equal(t, "Admin PNC", body.Isian.NamaPelapor)
}

// Galat bersifat sementara (503) dicatat di log supaya terlihat operator.
func TestCreateFailuresMapToTheirCodes(t *testing.T) {
	h := newHarness(t)
	h.resolver.SetError(errors.New("db link mati"))
	recorder := h.do(http.MethodPost, "/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, CodeBranchUnreadable, decode(t, recorder)["kode"])
	require.Contains(t, h.logs.String(), "permintaan gagal")
	require.Contains(t, h.logs.String(), "db link mati")

	h = newHarness(t)
	h.repo.SetError(fmt.Errorf("sisip: %w", inboxlaporanklaim.ErrStorageNotReady))
	recorder = h.do(http.MethodPost, "/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, CodePenyimpananBelumSiap, decode(t, recorder)["kode"])

	// Galat 4xx tidak dicatat sebagai galat.
	h = newHarness(t)
	h.resolver = memory.NewBranchResolver(nil)
	h.mount(t, func(string) (inboxlaporanklaim.Repo, error) { return h.repo, nil })
	recorder = h.do(http.MethodPost, "/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Empty(t, h.logs.String())
}

func createReport(t *testing.T, h *harness) string {
	t.Helper()
	recorder := h.do(http.MethodPost, "/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusCreated, recorder.Code)
	var body SingleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Laporan.ID
}

func TestSaveStoresFormFieldsAndAnswersThem(t *testing.T) {
	h := newHarness(t)
	id := createReport(t, h)

	recorder := h.do(http.MethodPut, "/inbox/laporan-klaim/"+id, "ASM", `{
		"tanggal_terima_dokumen": "2026-09-21",
		"tanggal_kejadian": "bukan-tanggal",
		"nama_pelapor": " Budi ",
		"nomor_polis": "126.00000000001",
		"estimasi_kerugian": 150000,
		"kronologis": "terbakar",
		"jumlah_dokumen": 3
	}`)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var body SingleResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "2026-09-21", body.Isian.TanggalTerimaDokumen)
	require.Equal(t, "", body.Isian.TanggalKejadian, "tanggal yang tidak terbaca menjadi kosong")
	require.Equal(t, "Budi", body.Isian.NamaPelapor)
	require.Equal(t, int64(150000), body.Isian.EstimasiKerugian)
	require.Equal(t, 3, body.Isian.JumlahDokumen)

	stored, err := h.repo.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "terbakar", stored.Chronology)
	require.Equal(t, testLogin, stored.UpdatedBy)
}

func TestSaveRejectsMalformedBodies(t *testing.T) {
	for name, body := range map[string]string{
		"bukan json":        `{`,
		"field tak dikenal": `{"nama_lain":"x"}`,
		"dua dokumen":       `{} {}`,
	} {
		h := newHarness(t)
		id := createReport(t, h)
		recorder := h.do(http.MethodPut, "/inbox/laporan-klaim/"+id, "ASM", body)
		require.Equalf(t, http.StatusBadRequest, recorder.Code, name)
		content := decode(t, recorder)
		require.Equalf(t, CodeMalformedRequest, content["kode"], name)
		require.Equal(t, "Permintaan tidak dapat dibaca.", content["pesan"])
	}
}

func TestSaveRefusals(t *testing.T) {
	h := newHarness(t)
	id := createReport(t, h)

	// Isian yang melanggar aturan dijawab 422 beserta isian yang salah.
	recorder := h.do(http.MethodPut, "/inbox/laporan-klaim/"+id, "ASM", `{"estimasi_kerugian": -1}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	content := decode(t, recorder)
	require.Equal(t, CodeValidationFailed, content["kode"])
	require.Equal(t, "estimasi_kerugian", content["detail"].([]any)[0].(map[string]any)["kolom"])

	// Polis Syariah memblokir penyimpanan.
	recorder = h.do(http.MethodPut, "/inbox/laporan-klaim/"+id, "ASM", `{"nomor_polis": "12600000000002"}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, "nomor_polis", decode(t, recorder)["detail"].([]any)[0].(map[string]any)["kolom"])

	// Berkas milik Pega hanya dapat dibaca.
	recorder = h.do(http.MethodPut, "/inbox/laporan-klaim/RCV-0001", "ASM", `{}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, CodeReadOnly, decode(t, recorder)["kode"])

	// Nomor kosong tidak ditemukan.
	recorder = h.do(http.MethodPut, "/inbox/laporan-klaim/%20", "ASM", `{}`)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestPolicyLookupAnswers(t *testing.T) {
	h := newHarness(t)

	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/polis?nomor=126.000.000.000.02", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	var syariah PolicyResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &syariah))
	require.Equal(t, "12600000000002", syariah.NomorPolis)
	require.True(t, syariah.Ditemukan)
	require.True(t, syariah.Syariah)
	require.True(t, syariah.Memblokir)
	require.Equal(t, []PolicyNoticeDTO{{
		Kode: inboxlaporanklaim.PolicySyariah, Pesan: "Polis Syariah harus registrasi klaim melalui Pega SMAS",
		Memblokir: true,
	}}, syariah.Pesan)

	recorder = h.do(http.MethodGet, "/inbox/laporan-klaim/polis?nomor=999", "ASM", "")
	var missing PolicyResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &missing))
	require.False(t, missing.Ditemukan)
	require.False(t, missing.Memblokir, "polis tidak ditemukan tidak memblokir")
	require.Equal(t, inboxlaporanklaim.PolicyNotFound, missing.Pesan[0].Kode)

	recorder = h.do(http.MethodGet, "/inbox/laporan-klaim/polis", "ASM", "")
	var empty PolicyResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &empty))
	require.Equal(t, PolicyResponse{Pesan: []PolicyNoticeDTO{}}, empty)
}

func readCSV(t *testing.T, recorder *httptest.ResponseRecorder) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportWritesCSVWithTheSameFilterAsTheList(t *testing.T) {
	h := newHarness(t)
	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/ekspor", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="laporan-klaim-semua-20260922.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

	records := readCSV(t, recorder)
	require.Equal(t, exportHeader, records[0])
	require.Len(t, records, 6, "lima berkas cabang 1001 ditambah judul")
	require.Equal(t, []string{
		"RCV-0012", "", "POL-PA-0012", "Tertanggung Contoh L", "Personal Accident", "REF-0012",
		"2026-09-20", "2026-09-20", "adminpnc", "Cabang Contoh Jakarta 1", "2026-09-20", "2",
		"Not Transferred", "", "", "pega",
	}, records[1])
}

// Galat sebelum header terkirim masih dijawab sebagai JSON.
func TestExportFirstPageFailureIsAnsweredAsJSON(t *testing.T) {
	h := newHarness(t)
	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/ekspor?kategori=entah", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, CodeValidationFailed, decode(t, recorder)["kode"])
}

// pagedRepo adalah repo tiruan yang menjawab halaman penuh tanpa batas, untuk menguji
// batas ekspor dan kegagalan di tengah aliran tanpa menyimpan puluhan ribu baris.
type pagedRepo struct {
	total     int
	perPage   int
	failAfter int
	calls     int
}

func (p *pagedRepo) List(_ context.Context, _ inboxlaporanklaim.Filter, page inboxlaporanklaim.Pagination) (inboxlaporanklaim.Page, error) {
	p.calls++
	if p.failAfter > 0 && p.calls > p.failAfter {
		return inboxlaporanklaim.Page{}, errors.New("koneksi putus")
	}
	rows := make([]inboxlaporanklaim.ClaimReport, p.perPage)
	for i := range rows {
		rows[i] = inboxlaporanklaim.ClaimReport{ID: fmt.Sprintf("R-%d-%d", page.Page, i)}
	}
	return inboxlaporanklaim.Page{Report: rows, Total: p.total, Pagination: page}, nil
}

func (p *pagedRepo) Summarize(context.Context, inboxlaporanklaim.Filter) (inboxlaporanklaim.Summary, error) {
	return inboxlaporanklaim.Summary{}, nil
}

func (p *pagedRepo) ListRegions(context.Context) ([]inboxlaporanklaim.Region, error) { return nil, nil }

func (p *pagedRepo) Get(context.Context, string) (inboxlaporanklaim.ClaimReport, error) {
	return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrNotFound
}

func (p *pagedRepo) Insert(context.Context, inboxlaporanklaim.ClaimReport) (inboxlaporanklaim.ClaimReport, error) {
	return inboxlaporanklaim.ClaimReport{}, nil
}

func (p *pagedRepo) Update(context.Context, inboxlaporanklaim.ClaimReport) error { return nil }

func (p *pagedRepo) FindPolicy(context.Context, string) (inboxlaporanklaim.Policy, bool, error) {
	return inboxlaporanklaim.Policy{}, false, nil
}

func pagedHarness(t *testing.T, repo *pagedRepo) *harness {
	t.Helper()
	h := newHarness(t)
	return h.mount(t, func(string) (inboxlaporanklaim.Repo, error) { return repo, nil })
}

// Ekspor yang menyentuh batas diberi tanda di baris terakhir, bukan dipotong diam-diam.
func TestExportStopsAtLimitWithTruncationNotice(t *testing.T) {
	repo := &pagedRepo{total: 60000, perPage: exportChunk}
	h := pagedHarness(t, repo)

	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/ekspor?kategori=outstanding", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder)
	require.Len(t, records, 1+exportLimit+1)
	last := records[len(records)-1]
	require.Equal(t, "-- Terpotong pada 50000 baris dari 60000 yang cocok. Persempit penyaringnya. --", last[0])
	require.Len(t, last, len(exportHeader))
	require.Equal(t, exportLimit/exportChunk+1, repo.calls, "potongan sesudah batas tidak boleh dibaca")
}

func TestExportStopsWhenAPageComesBackEmpty(t *testing.T) {
	repo := &pagedRepo{total: 500, perPage: 0}
	h := pagedHarness(t, repo)

	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/ekspor", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder), 1, "hanya judul")
	require.Equal(t, 1, repo.calls)
}

// Kegagalan setelah header terkirim hanya dapat dicatat; berkasnya berhenti di situ.
func TestExportFailureMidStreamIsLogged(t *testing.T) {
	repo := &pagedRepo{total: 250, perPage: exportChunk, failAfter: 1}
	h := pagedHarness(t, repo)

	recorder := h.do(http.MethodGet, "/inbox/laporan-klaim/ekspor", "ASM", "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder), 1+exportChunk)
	require.Contains(t, h.logs.String(), "ekspor laporan klaim terputus")
	require.Contains(t, h.logs.String(), "koneksi putus")
}

// failingWriter menolak setiap penulisan badan, meniru sambungan yang diputus pengguna.
type failingWriter struct {
	header http.Header
	status int
}

func (f *failingWriter) Header() http.Header       { return f.header }
func (f *failingWriter) WriteHeader(status int)    { f.status = status }
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("sambungan ditutup") }

func exportInto(h *harness, w http.ResponseWriter) {
	request := httptest.NewRequest(http.MethodGet, "/inbox/laporan-klaim/ekspor", nil)
	request.Header.Set("X-Test-Portal", "ASM")
	h.router.ServeHTTP(w, request)
}

func TestExportWriteFailuresAreLogged(t *testing.T) {
	// Satu potongan penuh melampaui penyangga CSV, sehingga penulisan baris yang gagal.
	full := pagedHarness(t, &pagedRepo{total: exportChunk, perPage: exportChunk})
	exportInto(full, &failingWriter{header: http.Header{}})
	require.Contains(t, full.logs.String(), "ekspor laporan klaim terputus")
	require.Contains(t, full.logs.String(), "sambungan ditutup")

	// Satu baris muat di penyangga; kegagalannya baru terlihat saat didorong keluar.
	small := pagedHarness(t, &pagedRepo{total: 1, perPage: 1})
	exportInto(small, &failingWriter{header: http.Header{}})
	require.Contains(t, small.logs.String(), "sambungan ditutup")
}

func TestMapErrorCoversEveryKnownError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&inboxlaporanklaim.ValidationError{Violation: []inboxlaporanklaim.Violation{{Field: "a", Message: "b"}}},
			http.StatusUnprocessableEntity, CodeValidationFailed},
		{inboxlaporanklaim.ErrUnknownCategory, http.StatusUnprocessableEntity, CodeValidationFailed},
		{inboxlaporanklaim.ErrUnknownBusinessLine, http.StatusUnprocessableEntity, CodeValidationFailed},
		{inboxlaporanklaim.ErrNotFound, http.StatusNotFound, CodeNotFound},
		{inboxlaporanklaim.ErrCallerUnknown, http.StatusConflict, CodeCallerIncomplete},
		{inboxlaporanklaim.ErrBranchUnknown, http.StatusForbidden, CodeBranchUnknown},
		{inboxlaporanklaim.ErrBranchUnreadable, http.StatusServiceUnavailable, CodeBranchUnreadable},
		{inboxlaporanklaim.ErrStorageNotReady, http.StatusServiceUnavailable, CodePenyimpananBelumSiap},
		{inboxlaporanklaim.ErrReadOnlyOrigin, http.StatusConflict, CodeReadOnly},
		{inboxlaporanklaim.ErrAlreadyRegistered, http.StatusConflict, CodeReadOnly},
	}
	for _, c := range cases {
		status, body, known := mapError(fmt.Errorf("dibungkus: %w", c.err))
		require.Truef(t, known, "%v", c.err)
		require.Equalf(t, c.status, status, "%v", c.err)
		require.Equalf(t, c.code, body.Code, "%v", c.err)
		require.NotEmpty(t, body.Message)
	}

	_, body, _ := mapError(&inboxlaporanklaim.ValidationError{
		Violation: []inboxlaporanklaim.Violation{{Field: "a", Message: "b"}},
	})
	require.Equal(t, []ViolationDTO{{Field: "a", Message: "b"}}, body.Detail)

	status, _, known := mapError(errors.New("lain"))
	require.False(t, known)
	require.Zero(t, status)
}
