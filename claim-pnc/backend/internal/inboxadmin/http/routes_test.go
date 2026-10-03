package inboxadminhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
	"claim-pnc/internal/inboxadmin/usecase"

	inboxadminhttp "claim-pnc/internal/inboxadmin/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal harus ada di daftar portal contoh DAN dinyatakan siap.
const testPortal = "ASM"

// testNow tetap, supaya kolom Aging dapat diperiksa dengan angka pasti.
var testNow = time.Date(2026, time.September, 20, 8, 0, 0, 0, time.UTC)

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// serverOptions mengatur bagian server uji yang berbeda antar-uji.
type serverOptions struct {
	// getCaller nil berarti handler dibentuk TANPA jembatan identitas sama sekali.
	getCaller inboxadminhttp.CallerReader

	// selector menggantikan pemilih repo bawaan (memori contoh).
	selector inboxadmin.RepoSelector

	// noFallback membentuk handler tanpa penulis galat cadangan.
	noFallback bool

	// logs menampung log bila diisi.
	logs *bytes.Buffer
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// fallbackStatus adalah status yang dipakai cadangan untuk galat yang tidak dikenal
// siapa pun — dibedakan dari 500 bawaan modul supaya jalurnya terbukti.
const fallbackStatus = http.StatusTeapot

func buildHandler(t *testing.T, o serverOptions) (*inboxadminhttp.Handler, portalhttp.ErrorWriter, *slog.Logger) {
	t.Helper()

	selector := o.selector
	if selector == nil {
		store := memory.NewSampleStore()
		selector = func(string) (inboxadmin.Repo, error) { return store, nil }
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: selector,
		Clock:        fixedClock{at: testNow},
	})
	require.NoError(t, err)

	var sink io.Writer = io.Discard
	if o.logs != nil {
		sink = o.logs
	}
	logger := slog.New(slog.NewTextHandler(sink, nil))

	// Rantai galat ditiru dari cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// ke penulis cadangan.
	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, fallbackStatus, map[string]string{"kode": "cadangan"})
		},
		writeJSON,
	)

	options := inboxadminhttp.Options{
		Service:   service,
		GetCaller: o.getCaller,
		Logger:    logger,
		WriteJSON: writeJSON,
	}
	if !o.noFallback {
		options.FallbackErrorWriter = inboxadminhttp.ErrorWriter(portalError)
	}

	return inboxadminhttp.NewHandler(options), portalError, logger
}

func buildServer(t *testing.T, o serverOptions) http.Handler {
	t.Helper()

	handler, portalError, logger := buildHandler(t, o)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   portalError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		inboxadminhttp.Mount(api, handler, portalDeps)
	})
	return router
}

// knownCaller adalah jembatan identitas yang selalu mengenali login tertentu.
func knownCaller(login string) inboxadminhttp.CallerReader {
	return func(context.Context) (inboxadminhttp.Caller, bool) {
		return inboxadminhttp.Caller{Login: login}, true
	}
}

func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestMetadataReturnsTabsLinesAndPortal(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin/tab")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, testPortal, body["portal"])
	require.Equal(t, inboxadmin.TabAllCaseAdmin, body["tab_bawaan"])
	require.Len(t, body["tab"], 8)
	require.Len(t, body["tab_dinonaktifkan"], 3)
	require.Len(t, body["keterbatasan"], 3)

	lines := body["lini_bisnis"].([]any)
	require.Len(t, lines, 5)
	first := lines[0].(map[string]any)
	require.Equal(t, "ALL", first["kode"])
	require.Equal(t, "Semua Lini Bisnis", first["label"])

	disabled := body["tab_dinonaktifkan"].([]any)[0].(map[string]any)
	require.Equal(t, "4", disabled["kode"])
	require.Equal(t, "Not Answered", disabled["nama"])
	require.NotEmpty(t, disabled["alasan"])

	tab := body["tab"].([]any)[0].(map[string]any)
	require.Equal(t, inboxadmin.TabAllCaseAdmin, tab["kode"])
	require.Equal(t, true, tab["hanya_milik_saya"])
	require.Equal(t, true, tab["pakai_pencarian"])
	require.Equal(t, false, tab["pakai_lini_bisnis"])
	column := tab["kolom"].([]any)[0].(map[string]any)
	require.NotEmpty(t, column["kunci"])
	require.NotEmpty(t, column["judul"])
}

func TestRequestWithoutPortalIsRejectedOnBothRoutes(t *testing.T) {
	// Tanpa header portal, middleware menolak dengan galat portal — tidak pernah jatuh ke
	// portal utama (`R-20`).
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	for _, path := range []string{"/api/inbox-admin/tab", "/api/inbox-admin"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "jalur %s", path)
	}
}

func TestHandlersCalledWithoutActivePortalAnswerNotStated(t *testing.T) {
	// Handler yang terpasang tanpa middleware portal tetap menolak, bukan melayani tanpa
	// portal. Galat portal diteruskan ke penulis cadangan dan dipetakan menjadi 400.
	handler, _, _ := buildHandler(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	for name, serve := range map[string]http.HandlerFunc{
		"metadata": handler.Metadata,
		"list":     handler.List,
	} {
		recorder := httptest.NewRecorder()
		serve(recorder, httptest.NewRequest(http.MethodGet, "/api/inbox-admin", nil))

		require.Equalf(t, http.StatusBadRequest, recorder.Code, "handler %s", name)
		require.Equal(t, portalhttp.CodeNotStated, decode(t, recorder)["kode"])
	}
}

func TestListDefaultTabReturnsCallerRowsWithFormattedDates(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, testPortal, body["portal"])
	require.Equal(t, inboxadmin.TabAllCaseAdmin, body["tab"].(map[string]any)["kode"])

	items := body["baris"].([]any)
	require.NotEmpty(t, items)
	for _, raw := range items {
		row := raw.(map[string]any)
		require.Containsf(t, row, "referensi", "baris %v", row["case_id"])
		require.Contains(t, row, "tanggal_kejadian")
		require.Contains(t, row, "aging_total")
	}

	paging := body["paginasi"].(map[string]any)
	require.Equal(t, float64(1), paging["halaman"])
	require.Equal(t, float64(inboxadmin.DefaultPageSize), paging["ukuran"])
	require.Equal(t, float64(len(items)), paging["total"])
	require.Equal(t, float64(1), paging["total_halaman"])

	filter := body["penyaring"].(map[string]any)
	require.Equal(t, "ALL", filter["bisnis"])
	require.Equal(t, "", filter["cari"])
}

func TestListAllTabFormatsDatesAsCalendarDays(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin?tab="+inboxadmin.TabAll+"&cari=PNC-8801")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	items := body["baris"].([]any)
	require.Len(t, items, 1)

	row := items[0].(map[string]any)
	require.Equal(t, "PNC-8801", row["case_id"])
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-8801", row["referensi"])
	require.Equal(t, "2026-08-02", row["tanggal_kejadian"])
	require.Equal(t, "2026-08-05", row["tanggal_lapor"])
	require.Equal(t, "2026-08-06", row["tanggal_input"])
	// Tanggal yang tidak ada dikirim sebagai null, bukan teks kosong.
	require.Nil(t, row["tanggal_request"])
	require.Nil(t, row["aging_lod"])
	// 6 Agustus → 20 September = 45 hari kalender.
	require.Equal(t, float64(45), row["aging_total"])

	require.Equal(t, "PNC-8801", body["penyaring"].(map[string]any)["cari"])
}

func TestUnreadablePaginationFallsBackToDefaults(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin?tab="+inboxadmin.TabAll+"&halaman=abc&ukuran=-4")
	require.Equal(t, http.StatusOK, recorder.Code)

	paging := decode(t, recorder)["paginasi"].(map[string]any)
	require.Equal(t, float64(1), paging["halaman"])
	require.Equal(t, float64(inboxadmin.DefaultPageSize), paging["ukuran"])
}

func TestExplicitPaginationIsHonoured(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin?tab="+inboxadmin.TabAll+"&halaman=2&ukuran=3")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	paging := body["paginasi"].(map[string]any)
	require.Equal(t, float64(2), paging["halaman"])
	require.Equal(t, float64(3), paging["ukuran"])
	require.Equal(t, float64(4), paging["total"])
	require.Equal(t, float64(2), paging["total_halaman"])
	require.Len(t, body["baris"], 1)
}

func TestUnknownCallerIsAnswered409(t *testing.T) {
	cases := map[string]inboxadminhttp.CallerReader{
		"tanpa jembatan": nil,
		"tidak terbaca": func(context.Context) (inboxadminhttp.Caller, bool) {
			return inboxadminhttp.Caller{}, false
		},
		"login kosong": func(context.Context) (inboxadminhttp.Caller, bool) {
			return inboxadminhttp.Caller{Login: ""}, true
		},
	}

	for name, reader := range cases {
		recorder := get(t, buildServer(t, serverOptions{getCaller: reader}), "/api/inbox-admin")

		require.Equalf(t, http.StatusConflict, recorder.Code, "kasus %s", name)
		body := decode(t, recorder)
		require.Equal(t, inboxadminhttp.CodeCallerUnknown, body["kode"])
		require.NotContains(t, body, "baris")
	}
}

func TestInvalidFiltersAreAnswered422WithDetails(t *testing.T) {
	server := buildServer(t, serverOptions{getCaller: knownCaller(memory.SampleOwner)})

	recorder := get(t, server, "/api/inbox-admin?tab=99&bisnis=ENTAH")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, inboxadminhttp.CodeValidationFail, body["kode"])

	details := body["detail"].([]any)
	require.NotEmpty(t, details)
	fields := []string{}
	for _, raw := range details {
		detail := raw.(map[string]any)
		require.NotEmpty(t, detail["pesan"])
		fields = append(fields, detail["field"].(string))
	}
	require.Contains(t, fields, inboxadmin.FieldTab)
}

func TestUnrecognizedErrorGoesToFallback(t *testing.T) {
	server := buildServer(t, serverOptions{
		getCaller: knownCaller(memory.SampleOwner),
		selector: func(string) (inboxadmin.Repo, error) {
			return nil, errors.New("koneksi entitas mati")
		},
	})

	recorder := get(t, server, "/api/inbox-admin")
	require.Equal(t, fallbackStatus, recorder.Code)
	require.Equal(t, "cadangan", decode(t, recorder)["kode"])
}

func TestUnrecognizedErrorWithoutFallbackIs500AndLogged(t *testing.T) {
	logs := &bytes.Buffer{}
	server := buildServer(t, serverOptions{
		getCaller:  knownCaller(memory.SampleOwner),
		noFallback: true,
		logs:       logs,
		selector: func(string) (inboxadmin.Repo, error) {
			return nil, errors.New("koneksi entitas mati")
		},
	})

	recorder := get(t, server, "/api/inbox-admin")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, inboxadminhttp.CodeInternalError, body["kode"])
	// Rincian galat internal tidak pernah dikirim ke peramban — hanya ke log.
	require.NotContains(t, recorder.Body.String(), "koneksi entitas mati")
	require.Contains(t, logs.String(), "koneksi entitas mati")
	require.Contains(t, logs.String(), "/api/inbox-admin")
}

func TestValidationErrorWithoutFallbackIsNotLogged(t *testing.T) {
	// Galat 4xx milik modul ini dijawab tanpa menulis log galat.
	logs := &bytes.Buffer{}
	server := buildServer(t, serverOptions{
		getCaller:  knownCaller(memory.SampleOwner),
		noFallback: true,
		logs:       logs,
	})

	recorder := get(t, server, "/api/inbox-admin?tab=99")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Empty(t, logs.String())
}
