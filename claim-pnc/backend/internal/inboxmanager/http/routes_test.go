package inboxmanagerhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/repo/memory"
	"claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	inboxmanagerhttp "claim-pnc/internal/inboxmanager/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/inbox-manager"

var refreshed = time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC)

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func write500(w http.ResponseWriter, r *http.Request, _ error) {
	writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "cadangan"})
}

// scriptedRepo meneruskan ke penyimpanan memori dengan dua tambahan: waktu penyegaran
// dashboard, dan galat pada pemanggilan Queue ke-n (untuk ekspor yang gagal di tengah).
type scriptedRepo struct {
	*memory.Store
	refreshedAt  *time.Time
	failQueueAt  int
	queueCalls   *int
	failCounters error
}

func (s scriptedRepo) Dashboard(ctx context.Context, q inboxmanager.Query) (inboxmanager.DashboardView, error) {
	view, err := s.Store.Dashboard(ctx, q)
	view.RefreshedAt = s.refreshedAt
	return view, err
}

func (s scriptedRepo) Queue(ctx context.Context, q inboxmanager.Query) ([]inboxmanager.QueueRow, error) {
	*s.queueCalls++
	if s.failQueueAt > 0 && *s.queueCalls >= s.failQueueAt {
		return nil, errors.New("ORA-03113 di tengah")
	}
	return s.Store.Queue(ctx, q)
}

func (s scriptedRepo) Counters(ctx context.Context, c inboxmanager.Caller) ([]inboxmanager.Counter, error) {
	if s.failCounters != nil {
		return nil, s.failCounters
	}
	return s.Store.Counters(ctx, c)
}

type serverOptions struct {
	login       string
	orgUnit     string
	noCaller    bool
	withoutFall bool
	repo        scriptedRepo
	logs        *bytes.Buffer
}

func newServer(t *testing.T, o serverOptions) (http.Handler, *inboxmanagerhttp.Handler) {
	t.Helper()

	if o.repo.Store == nil {
		o.repo.Store = memory.NewSampleStore()
	}
	if o.repo.queueCalls == nil {
		o.repo.queueCalls = new(int)
	}
	repo := o.repo

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanager.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		LineBusinessSelector: func(string) (inboxmanager.LineBusinessRepo, error) { return repo, nil },
		Clock:                clock.FixedAt(time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	out := io.Writer(io.Discard)
	if o.logs != nil {
		out = o.logs
	}
	logger := slog.New(slog.NewTextHandler(out, nil))

	opts := inboxmanagerhttp.Options{Service: service, Logger: logger, WriteJSON: writeJSON}
	if !o.withoutFall {
		opts.FallbackErrorWriter = inboxmanagerhttp.ErrorWriter(portalhttp.WithPortalError(write500, writeJSON))
	}
	if !o.noCaller {
		login, org := o.login, o.orgUnit
		opts.GetCaller = func(context.Context) (inboxmanagerhttp.Caller, bool) {
			return inboxmanagerhttp.Caller{Login: login, OrgUnit: org}, true
		}
	}
	handler := inboxmanagerhttp.NewHandler(opts)

	deps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   portalhttp.WithPortalError(write500, writeJSON),
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		inboxmanagerhttp.Mount(api, handler, deps)
	})
	return router, handler
}

func call(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	content := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &content)
	return rec, content
}

func TestMetadataPerCaller(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"/tab", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, inboxmanager.LineNonMBU, body["lini_bisnis_anda"])
	require.Equal(t, inboxmanager.DefaultTab, body["tab_bawaan"])
	require.Len(t, body["tab"], 13)
	require.NotEmpty(t, body["selisih_terencana"])

	first := body["tab"].([]any)[0].(map[string]any)
	require.Equal(t, "dashboard", first["jenis"])
	require.Len(t, first["panel"], 2)
	require.Nil(t, first["kolom"])

	payment := body["tab"].([]any)[11].(map[string]any)
	require.Equal(t, inboxmanager.TabPaymentAkseptasi, payment["kode"])
	decision := payment["keputusan"].(map[string]any)
	require.Equal(t, true, decision["dapat_diputuskan"])
	require.NotEmpty(t, decision["alasan_setuju_ditahan"])
	require.Equal(t, "Catatan Atasan", decision["label_alasan"])
}

func TestCountersIncludeOverview(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"/ringkasan", "")
	require.Equal(t, http.StatusOK, rec.Code)
	counters := body["pencacah"].([]any)
	require.Len(t, counters, 11)

	var sparepart, overview map[string]any
	for _, c := range counters {
		entry := c.(map[string]any)
		switch entry["tab"] {
		case inboxmanager.TabMasterSparepart:
			sparepart = entry
		case inboxmanager.TabApprovalMaster:
			overview = entry
		}
	}
	require.Equal(t, inboxmanager.TabApprovalMaster, sparepart["induk"])
	require.NotEmpty(t, sparepart["tidak_tersedia"])
	require.Contains(t, overview["tidak_tersedia"], "BELUM lengkap")
	// 2 bengkel + 1 panel + 1 rangka + 1 kategori + 1 tipe + 1 grouping + 1 payment +
	// 1 penolakan; Master Sparepart tidak ikut dijumlahkan karena sumbernya rusak.
	require.Equal(t, float64(9), overview["jumlah"])
}

func TestListDashboardWithRefreshAndPeriod(t *testing.T) {
	at := refreshed
	server, _ := newServer(t, serverOptions{login: "MGR", repo: scriptedRepo{refreshedAt: &at}})

	rec, body := call(t, server, http.MethodGet, route+"?tab=2&bulan=2026-02", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "2026-09-28T02:00:00Z", body["disegarkan_pada"])
	require.Equal(t, map[string]any{"dari": "2026-02-01", "sampai": "2026-02-28"}, body["periode"])

	panels := body["panel"].([]any)
	require.Len(t, panels, 2)
	first := panels[0].(map[string]any)
	require.Equal(t, "grup_bisnis", first["kunci"])
	row := first["baris"].([]any)[0].(map[string]any)
	require.Equal(t, map[string]any{"teks": "ANEKA"}, row[inboxmanager.FieldDimensi])
	require.Equal(t, map[string]any{"jumlah": float64(21)}, row[inboxmanager.FieldTotalPeriodeIni])
	require.Nil(t, body["baris"])
	require.Nil(t, body["paginasi"])
}

func TestListDashboardKlaimAmountsAreText(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"?tab=3", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Nil(t, body["periode"], "tab Klaim tanpa periode tidak mengirim periode")
	require.Nil(t, body["disegarkan_pada"])

	row := body["panel"].([]any)[0].(map[string]any)["baris"].([]any)[0].(map[string]any)
	require.Equal(t, map[string]any{"nilai": "1250000000"}, row[inboxmanager.FieldNilaiAksep])
}

func TestListQueuePaginated(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"?tab=5&halaman=2&ukuran=1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, map[string]any{
		"halaman": float64(2), "ukuran": float64(1), "total": float64(2), "total_halaman": float64(2),
	}, body["paginasi"])
	rows := body["baris"].([]any)
	require.Len(t, rows, 1)
	require.Equal(t, "BGK-002", rows[0].(map[string]any)["kunci"])
	require.Nil(t, body["panel"])
}

func TestListErrorsMapped(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"?tab=99", "")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeValidationFail, body["kode"])
	require.Equal(t, []any{map[string]any{"isian": "tab", "pesan": "Tab tidak dikenal."}}, body["rincian"])

	rec, body = call(t, server, http.MethodGet, route+"?tab=8", "")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeSourceDown, body["kode"])

	// Tab Nomor Rangka dibatasi NONMBU: petugas PA tanpa unit Development ditolak.
	store := memory.NewStore()
	store.SetLineBusiness("PAUSER", inboxmanager.LinePA)
	limited, _ := newServer(t, serverOptions{login: "PAUSER", repo: scriptedRepo{Store: store}})
	rec, body = call(t, limited, http.MethodGet, route+"?tab=7", "")
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeTabNotAllowed, body["kode"])

	// Unit Development membuka tab itu.
	dev, _ := newServer(t, serverOptions{login: "PAUSER", orgUnit: " development ", repo: scriptedRepo{Store: store}})
	rec, _ = call(t, dev, http.MethodGet, route+"?tab=7", "")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCallerUnknownOnEveryRoute(t *testing.T) {
	for name, o := range map[string]serverOptions{
		"tanpa pembaca": {noCaller: true},
		"login kosong":  {login: "   "},
	} {
		t.Run(name, func(t *testing.T) {
			server, _ := newServer(t, o)
			for _, path := range []string{route, route + "/tab", route + "/ringkasan", route + "/ekspor"} {
				rec, body := call(t, server, http.MethodGet, path, "")
				require.Equal(t, http.StatusConflict, rec.Code, path)
				require.Equal(t, inboxmanagerhttp.CodeCallerUnknown, body["kode"], path)
			}
			rec, body := call(t, server, http.MethodPost, route+"/keputusan", `{}`)
			require.Equal(t, http.StatusConflict, rec.Code)
			require.Equal(t, inboxmanagerhttp.CodeCallerUnknown, body["kode"])
		})
	}

	// Pembaca yang menyatakan tidak ada sesi juga ditolak.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector:         func(string) (inboxmanager.Repo, error) { return memory.NewStore(), nil },
		LineBusinessSelector: func(string) (inboxmanager.LineBusinessRepo, error) { return memory.NewStore(), nil },
		Clock:                clock.FixedAt(refreshed),
	})
	require.NoError(t, err)
	handler := inboxmanagerhttp.NewHandler(inboxmanagerhttp.Options{
		Service: service, WriteJSON: writeJSON,
		GetCaller: func(context.Context) (inboxmanagerhttp.Caller, bool) {
			return inboxmanagerhttp.Caller{Login: "X"}, false
		},
	})
	req := httptest.NewRequest(http.MethodGet, route, nil)
	req = req.WithContext(portalhttp.WithActivePortal(req.Context(), portal.Portal{Alias: "ASM"}))
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandlersWithoutActivePortal(t *testing.T) {
	_, handler := newServer(t, serverOptions{login: "MGR"})
	for name, serve := range map[string]http.HandlerFunc{
		"metadata": handler.Metadata, "counters": handler.Counters, "list": handler.List,
		"export": handler.Export, "decide": handler.Decide,
	} {
		rec := httptest.NewRecorder()
		serve(rec, httptest.NewRequest(http.MethodGet, route, strings.NewReader("{}")))
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
		body := map[string]any{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, portalhttp.CodeNotStated, body["kode"], name)
	}
}

func TestUnrecognizedErrorBecomes500OrFallback(t *testing.T) {
	logs := &bytes.Buffer{}
	server, _ := newServer(t, serverOptions{
		login: "MGR", withoutFall: true, logs: logs,
		repo: scriptedRepo{failCounters: errors.New("ORA-03113 rahasia")},
	})
	rec, body := call(t, server, http.MethodGet, route+"/ringkasan", "")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeInternalError, body["kode"])
	require.NotContains(t, rec.Body.String(), "ORA-03113")
	require.Contains(t, logs.String(), "ORA-03113 rahasia")

	withFallback, _ := newServer(t, serverOptions{
		login: "MGR", repo: scriptedRepo{failCounters: errors.New("x")},
	})
	rec, body = call(t, withFallback, http.MethodGet, route+"/ringkasan", "")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "cadangan", body["kode"])

	// Tanpa logger pun 500 tetap ditulis.
	rec = httptest.NewRecorder()
	inboxmanagerhttp.WriteError(nil, writeJSON, nil)(rec, httptest.NewRequest(http.MethodGet, "/x", nil), errors.New("y"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDecideFlow(t *testing.T) {
	t.Run("berhasil penuh", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, route+"/keputusan",
			`{"tab":"5","keputusan":" setujui ","kunci":["BGK-001"]}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, map[string]any{
			"diminta": float64(1), "berubah": float64(1), "tidak_berubah": float64(0),
			"pesan": "1 baris diputuskan.",
		}, body)
	})

	t.Run("sebagian sudah diputuskan", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, route+"/keputusan",
			`{"tab":"5","keputusan":"tolak","kunci":["BGK-001","LAMA"],"alasan":"tidak lengkap"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, float64(1), body["tidak_berubah"])
		require.Contains(t, body["pesan"], "1 dari 2 baris berubah")
	})

	t.Run("badan rusak", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, route+"/keputusan", `{bukan json`)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		require.Equal(t, []any{map[string]any{
			"isian": inboxmanager.FieldKeys, "pesan": "Badan permintaan tidak dapat dibaca.",
		}}, body["rincian"])
	})

	t.Run("bukan antrean", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, route+"/keputusan",
			`{"tab":"1","keputusan":"setujui","kunci":["X"]}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, inboxmanagerhttp.CodeNotDecidable, body["kode"])
	})

	t.Run("persetujuan ditahan", func(t *testing.T) {
		server, _ := newServer(t, serverOptions{login: "MGR"})
		rec, body := call(t, server, http.MethodPost, route+"/keputusan",
			`{"tab":"12","keputusan":"setujui","kunci":["AKS-2026-0001"]}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, inboxmanagerhttp.CodeApproveBlocked, body["kode"])
	})
}

func TestExportQueueAsCSV(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, _ := call(t, server, http.MethodGet, route+"/ekspor?tab=5", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="inbox-manager-master-bengkel.csv"`,
		rec.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.Len(t, lines, 3)
	require.Contains(t, lines[1], "Bengkel Maju Jaya")
	require.Contains(t, lines[2], "Bengkel Sentosa")
}

// Antrean di atas satu halaman ekspor ditulis seluruhnya, halaman demi halaman.
func TestExportWalksEveryPage(t *testing.T) {
	store := memory.NewStore()
	rows := make([]inboxmanager.QueueRow, 0, 150)
	for i := 0; i < 150; i++ {
		rows = append(rows, inboxmanager.QueueRow{
			Key:   fmt.Sprintf("K%03d", i),
			Cells: map[string]string{inboxmanager.FieldID: fmt.Sprintf("K%03d", i)},
		})
	}
	store.SetQueue(inboxmanager.TabMasterPanel, rows)
	server, _ := newServer(t, serverOptions{login: "MGR", repo: scriptedRepo{Store: store}})

	rec, _ := call(t, server, http.MethodGet, route+"/ekspor?tab=6", "")
	require.Equal(t, http.StatusOK, rec.Code)
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.Len(t, lines, 151, "judul + 150 baris")
	require.True(t, strings.HasPrefix(lines[150], "K149"))
}

// Galat pada halaman kedua tidak dapat dijawab sebagai JSON: berkas berhenti dan galat dicatat.
func TestExportStopsWhenSecondPageFails(t *testing.T) {
	store := memory.NewStore()
	rows := make([]inboxmanager.QueueRow, 0, 120)
	for i := 0; i < 120; i++ {
		rows = append(rows, inboxmanager.QueueRow{Key: fmt.Sprintf("K%03d", i)})
	}
	store.SetQueue(inboxmanager.TabMasterPanel, rows)
	logs := &bytes.Buffer{}
	server, _ := newServer(t, serverOptions{
		login: "MGR", logs: logs, repo: scriptedRepo{Store: store, failQueueAt: 2},
	})

	rec, _ := call(t, server, http.MethodGet, route+"/ekspor?tab=6", "")
	require.Equal(t, http.StatusOK, rec.Code)
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.Len(t, lines, 101, "judul + halaman pertama saja")
	require.Contains(t, logs.String(), "ekspor inbox manager gagal di tengah berkas")
	require.Contains(t, logs.String(), "ORA-03113 di tengah")
}

func TestExportRejections(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: "MGR"})

	rec, body := call(t, server, http.MethodGet, route+"/ekspor?tab=1", "")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, []any{map[string]any{
		"isian": "tab", "pesan": "Hanya antrean persetujuan yang dapat diekspor.",
	}}, body["rincian"])

	rec, body = call(t, server, http.MethodGet, route+"/ekspor?tab=99", "")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, inboxmanagerhttp.CodeValidationFail, body["kode"])
}

// brokenWriter menolak setiap tulisan badan, sehingga csv.Writer melaporkan galat saat Flush.
type brokenWriter struct {
	header http.Header
	status int
}

func (b *brokenWriter) Header() http.Header       { return b.header }
func (b *brokenWriter) WriteHeader(status int)    { b.status = status }
func (b *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("koneksi terputus") }

func TestExportWriteFailureIsLogged(t *testing.T) {
	logs := &bytes.Buffer{}
	server, _ := newServer(t, serverOptions{login: "MGR", logs: logs})

	req := httptest.NewRequest(http.MethodGet, route+"/ekspor?tab=5", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	writer := &brokenWriter{header: http.Header{}}
	server.ServeHTTP(writer, req)

	require.Contains(t, logs.String(), "ekspor inbox manager gagal di tengah berkas")
	require.Contains(t, logs.String(), "koneksi terputus")
}
