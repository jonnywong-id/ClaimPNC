package inboxacceptopenprotectionhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/repo/memory"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
	"claim-pnc/internal/portal"

	accepthttp "claim-pnc/internal/inboxacceptopenprotection/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal harus ada di daftar portal contoh dan dinyatakan siap.
const testPortal = "ASM"

const basePath = "/api/inbox-accept-open-protection"

var wib = time.FixedZone("WIB", 7*60*60)

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// realService membentuk service sungguhan berepo memory contoh.
func realService(t *testing.T, at time.Time) (*usecase.Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewRepoWithSamples()
	s, err := usecase.NewService(usecase.Options{
		Protections: func(alias string) (inboxacceptopenprotection.Repo, error) {
			if alias != testPortal {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		Groups: memory.NewSampleGroupRepo(),
		Now:    func() time.Time { return at },
	})
	require.NoError(t, err)
	return s, repo
}

// server memasang handler di belakang middleware portal, seperti di cmd/claimpnc.
func server(t *testing.T, service accepthttp.Service, login string) http.Handler {
	t.Helper()
	logger := discardLogger()

	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_internal", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := accepthttp.NewHandler(accepthttp.Options{
		Service: service,
		GetCaller: func(context.Context) (accepthttp.Caller, bool) {
			return accepthttp.Caller{Login: login}, login != ""
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: accepthttp.ErrorWriter(writeError),
		Location:            wib,
	})

	deps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		accepthttp.Mount(api, handler, deps)
	})
	return router
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-Portal", testPortal)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v), rec.Body.String())
	return v
}

type errorBody struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

type listBody struct {
	Proteksi []struct {
		NomorProteksi   string `json:"nomor_proteksi"`
		TipeProteksi    string `json:"tipe_proteksi"`
		TanggalProteksi string `json:"tanggal_proteksi"`
		Antrean         string `json:"antrean"`
	} `json:"proteksi"`
	Total   int    `json:"total"`
	Antrean string `json:"antrean"`
}

type detailBody struct {
	NomorProteksi     string `json:"nomor_proteksi"`
	NamaTertanggung   string `json:"nama_tertanggung"`
	PolisMulai        string `json:"polis_mulai"`
	PolisAkhir        string `json:"polis_akhir"`
	StatusAkseptasi   string `json:"status_akseptasi"`
	TanggalAkseptasi  string `json:"tanggal_akseptasi"`
	DiaksepOleh       string `json:"diaksep_oleh"`
	MenungguKeputusan bool   `json:"menunggu_keputusan"`
	DetailPerubahan   *struct {
		Judul           string `json:"judul"`
		DolSebelum      string `json:"dol_sebelum"`
		DolSesudah      string `json:"dol_sesudah"`
		PenyebabSebelum string `json:"penyebab_sebelum"`
		PenyebabSesudah string `json:"penyebab_sesudah"`
		NamaObjek       string `json:"nama_objek"`
		Kosong          bool   `json:"kosong"`
	} `json:"detail_perubahan"`
}

// ── Daftar ─────────────────────────────────────────────────────────────────────

func TestListDefaultsToNonPremiumQueue(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "KEDUACONTOH"), http.MethodGet, basePath, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := decode[listBody](t, rec)
	require.Equal(t, "non-premi", body.Antrean)
	require.Equal(t, 4, body.Total)
	require.Equal(t, "OPCN.26.0005", body.Proteksi[0].NomorProteksi)
	require.Equal(t, "2026-09-19", body.Proteksi[0].TanggalProteksi)
	require.Equal(t, "non-premi", body.Proteksi[0].Antrean)
}

func TestListPremiumQueueWithPaginationAndSearch(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "KOLEKSICONTOH"), http.MethodGet,
		basePath+"?antrean=premi&batas=5&lewati=0&cari=0003", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := decode[listBody](t, rec)
	require.Equal(t, "premi", body.Antrean)
	require.Equal(t, 1, body.Total)
	require.Equal(t, "2", body.Proteksi[0].TipeProteksi)
	require.Equal(t, "premi", body.Proteksi[0].Antrean)
}

func TestListRejectsMalformedParameters(t *testing.T) {
	s, _ := realService(t, time.Now())
	h := server(t, s, "KEDUACONTOH")

	cases := map[string]string{
		"?antrean=semua": `Parameter antrean harus "premi" atau "non-premi".`,
		"?batas=abc":     "Parameter batas harus berupa angka lebih besar dari nol.",
		"?batas=0":       "Parameter batas harus berupa angka lebih besar dari nol.",
		"?batas=101":     "Parameter batas melebihi 100.",
		"?lewati=-1":     "Parameter lewati harus berupa angka nol atau lebih.",
		"?lewati=x":      "Parameter lewati harus berupa angka nol atau lebih.",
	}
	for query, pesan := range cases {
		rec := do(t, h, http.MethodGet, basePath+query, "")
		require.Equal(t, http.StatusBadRequest, rec.Code, query)
		body := decode[errorBody](t, rec)
		require.Equal(t, accepthttp.CodeBadRequest, body.Code)
		require.Equal(t, pesan, body.Message, query)
	}
}

func TestListForbiddenQueue(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "TEKNIKCONTOH"), http.MethodGet, basePath+"?antrean=premi", "")
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, accepthttp.CodeForbidden, decode[errorBody](t, rec).Code)
}

func TestListWithoutCallerIsInternalError(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, ""), http.MethodGet, basePath, "")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, decode[errorBody](t, rec).Message, "identitas pemanggil tidak tersedia")
}

func TestRequestWithoutPortalHeaderIsRejectedByMiddleware(t *testing.T) {
	s, _ := realService(t, time.Now())
	req := httptest.NewRequest(http.MethodGet, basePath, nil)
	rec := httptest.NewRecorder()
	server(t, s, "KEDUACONTOH").ServeHTTP(rec, req)
	require.GreaterOrEqual(t, rec.Code, 400)
	require.NotEqual(t, http.StatusOK, rec.Code)
}

// ── Antrean ────────────────────────────────────────────────────────────────────

func TestQueuesForCaller(t *testing.T) {
	s, _ := realService(t, time.Now())

	rec := do(t, server(t, s, "KEDUACONTOH"), http.MethodGet, basePath+"/antrean", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"antrean":["non-premi","premi"]}`, rec.Body.String())

	rec = do(t, server(t, s, "ADMINCONTOH"), http.MethodGet, basePath+"/antrean", "")
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = do(t, server(t, s, ""), http.MethodGet, basePath+"/antrean", "")
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── Rincian ────────────────────────────────────────────────────────────────────

func TestGetLossDateChangeDetail(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "TEKNIKCONTOH"), http.MethodGet, basePath+"/OPCN.26.0004", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := decode[detailBody](t, rec)
	require.Equal(t, "OPCN.26.0004", body.NomorProteksi)
	require.Equal(t, "Tertanggung Contoh", body.NamaTertanggung)
	require.Equal(t, "2026-01-01", body.PolisMulai)
	require.Equal(t, "2026-12-31", body.PolisAkhir)
	require.True(t, body.MenungguKeputusan)
	require.Empty(t, body.TanggalAkseptasi)
	require.NotNil(t, body.DetailPerubahan)
	require.Equal(t, "Detail Perubahan DOL", body.DetailPerubahan.Judul)
	require.Equal(t, "2026-08-03", body.DetailPerubahan.DolSebelum)
	require.Equal(t, "2026-08-05", body.DetailPerubahan.DolSesudah)
	require.False(t, body.DetailPerubahan.Kosong)
}

func TestGetCauseOfLossDetailAndLegacyEmptyPanel(t *testing.T) {
	s, _ := realService(t, time.Now())
	h := server(t, s, "TEKNIKCONTOH")

	body := decode[detailBody](t, do(t, h, http.MethodGet, basePath+"/OPCN.26.0005", ""))
	require.Equal(t, "Detail Perubahan Cause Of Loss", body.DetailPerubahan.Judul)
	require.Equal(t, "12001", body.DetailPerubahan.PenyebabSebelum)
	require.Equal(t, "12002", body.DetailPerubahan.PenyebabSesudah)
	require.Empty(t, body.DetailPerubahan.DolSebelum)

	// Baris warisan: panel muncul tetapi kosong.
	body = decode[detailBody](t, do(t, h, http.MethodGet, basePath+"/OPC-216", ""))
	require.NotNil(t, body.DetailPerubahan)
	require.True(t, body.DetailPerubahan.Kosong)
	require.Empty(t, body.PolisMulai)
}

func TestGetDecidedProtectionHasNoPanel(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "TEKNIKCONTOH"), http.MethodGet, basePath+"/OPCN.26.0007", "")
	require.Equal(t, http.StatusOK, rec.Code)

	body := decode[detailBody](t, rec)
	require.Nil(t, body.DetailPerubahan)
	require.False(t, body.MenungguKeputusan)
	require.Equal(t, "1", body.StatusAkseptasi)
	require.Equal(t, "KOLEKSICONTOH", body.DiaksepOleh)
	require.Equal(t, "2026-09-20", body.TanggalAkseptasi)
}

func TestGetUnknownIsNotFound(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, "TEKNIKCONTOH"), http.MethodGet, basePath+"/TIDAK-ADA", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, accepthttp.CodeNotFound, decode[errorBody](t, rec).Code)
}

// ── Keputusan ──────────────────────────────────────────────────────────────────

func TestDecideApprove(t *testing.T) {
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, wib)
	s, repo := realService(t, at)
	rec := do(t, server(t, s, "TEKNIKCONTOH"), http.MethodPut,
		basePath+"/OPCN.26.0004/akseptasi", `{"keputusan":" setuju "}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	body := decode[detailBody](t, rec)
	require.Equal(t, "1", body.StatusAkseptasi)
	require.Equal(t, "TEKNIKCONTOH", body.DiaksepOleh)
	require.Equal(t, "2026-09-25", body.TanggalAkseptasi)
	require.False(t, body.MenungguKeputusan)

	tanggal, ada := repo.LossDateOf("PNCN.26.0009")
	require.True(t, ada)
	require.Equal(t, time.Date(2026, 8, 5, 0, 0, 0, 0, wib), tanggal)
}

func TestDecideErrorMapping(t *testing.T) {
	s, _ := realService(t, time.Now())
	h := server(t, s, "KEDUACONTOH")

	cases := []struct {
		path, body string
		status     int
		code       string
		contains   string
	}{
		{"/OPCN.26.0002/akseptasi", `{"keputusan":"mungkin"}`, http.StatusBadRequest, accepthttp.CodeBadRequest, `"setuju" atau "tolak"`},
		{"/OPCN.26.0002/akseptasi", `{"lain":1}`, http.StatusBadRequest, accepthttp.CodeBadRequest, "Badan permintaan tidak dapat dibaca."},
		{"/OPCN.26.0002/akseptasi", `bukan json`, http.StatusBadRequest, accepthttp.CodeBadRequest, "Badan permintaan tidak dapat dibaca."},
		{"/OPCN.26.0007/akseptasi", `{"keputusan":"setuju"}`, http.StatusConflict, accepthttp.CodeConflict, "sudah diakseptasi"},
		{"/OPCN.26.0006/akseptasi", `{"keputusan":"setuju"}`, http.StatusConflict, accepthttp.CodeConflict, "belum tertaut ke klaim"},
		{"/OPC-216/akseptasi", `{"keputusan":"tolak"}`, http.StatusOK, "", ""},
		{"/TIDAK-ADA/akseptasi", `{"keputusan":"setuju"}`, http.StatusNotFound, accepthttp.CodeNotFound, "tidak ditemukan"},
	}
	for _, c := range cases {
		rec := do(t, h, http.MethodPut, basePath+c.path, c.body)
		require.Equal(t, c.status, rec.Code, c.path+" "+c.body+" -> "+rec.Body.String())
		if c.code != "" {
			body := decode[errorBody](t, rec)
			require.Equal(t, c.code, body.Code)
			require.Contains(t, body.Message, c.contains)
		}
	}
}

func TestDecideWithoutCallerIsInternalError(t *testing.T) {
	s, _ := realService(t, time.Now())
	rec := do(t, server(t, s, ""), http.MethodPut, basePath+"/OPCN.26.0002/akseptasi", `{"keputusan":"setuju"}`)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── Pemetaan galat tanpa fallback dan handler tanpa middleware ─────────────────

// stubService mengembalikan galat yang ditentukan uji pada setiap pemanggilan.
type stubService struct{ err error }

func (s stubService) List(context.Context, usecase.ListQuery) (inboxacceptopenprotection.Page, error) {
	return inboxacceptopenprotection.Page{}, s.err
}
func (s stubService) Queues(context.Context, string) ([]inboxacceptopenprotection.Queue, error) {
	return nil, s.err
}
func (s stubService) Get(context.Context, string, string, string) (inboxacceptopenprotection.Protection, error) {
	return inboxacceptopenprotection.Protection{}, s.err
}
func (s stubService) Decide(context.Context, usecase.DecideCommand) (inboxacceptopenprotection.Protection, error) {
	return inboxacceptopenprotection.Protection{}, s.err
}

func TestWriteErrorMapsClaimNotSyncedAndInternal(t *testing.T) {
	write := accepthttp.WriteError(discardLogger(), writeJSON, nil)

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		inboxacceptopenprotection.ErrClaimNotSynced)
	require.Equal(t, http.StatusConflict, rec.Code)
	body := decode[errorBody](t, rec)
	require.Equal(t, accepthttp.CodeConflict, body.Code)
	require.Contains(t, body.Message, "Keputusan tidak disimpan")

	// Galat lain tanpa fallback: 500 dengan pesan umum, rinciannya tidak bocor.
	rec = httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/x", nil), errors.New("rahasia internal"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	body = decode[errorBody](t, rec)
	require.Equal(t, accepthttp.CodeInternalError, body.Code)
	require.Equal(t, "Terjadi kesalahan pada sistem.", body.Message)
	require.NotContains(t, rec.Body.String(), "rahasia")
}

func newDirectHandler(service accepthttp.Service, getCaller accepthttp.GetCaller) *accepthttp.Handler {
	// Location sengaja kosong: handler jatuh ke Asia/Jakarta.
	return accepthttp.NewHandler(accepthttp.Options{
		Service: service, GetCaller: getCaller, Logger: discardLogger(), WriteJSON: writeJSON,
	})
}

func withPortal(r *http.Request) *http.Request {
	return r.WithContext(portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: testPortal}))
}

func TestHandlerWithoutActivePortalRejects(t *testing.T) {
	h := newDirectHandler(stubService{}, func(context.Context) (accepthttp.Caller, bool) {
		return accepthttp.Caller{Login: "X"}, true
	})

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, accepthttp.CodeInternalError, decode[errorBody](t, rec).Code)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nomor", "A")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	h.Get(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	req = httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"keputusan":"setuju"}`))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	h.Decide(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerWithoutCallerReaderRejects(t *testing.T) {
	h := newDirectHandler(stubService{}, nil)

	rec := httptest.NewRecorder()
	h.Queues(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandlerGetWithoutCallerRejects(t *testing.T) {
	h := newDirectHandler(stubService{}, nil)

	req := withPortal(httptest.NewRequest(http.MethodGet, "/", nil))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nomor", "A")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, accepthttp.CodeInternalError, decode[errorBody](t, rec).Code)
}

func TestHandlerEmptyNumberIsBadRequest(t *testing.T) {
	h := newDirectHandler(stubService{}, nil)

	rec := httptest.NewRecorder()
	h.Get(rec, withPortal(httptest.NewRequest(http.MethodGet, "/", nil)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "Nomor proteksi wajib disebutkan.", decode[errorBody](t, rec).Message)

	rec = httptest.NewRecorder()
	h.Decide(rec, withPortal(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{}`))))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "Nomor proteksi wajib disebutkan.", decode[errorBody](t, rec).Message)
}

func TestHandlerGetForwardsServiceError(t *testing.T) {
	h := newDirectHandler(stubService{err: inboxacceptopenprotection.ErrForbidden},
		func(context.Context) (accepthttp.Caller, bool) { return accepthttp.Caller{Login: "X"}, true })

	req := withPortal(httptest.NewRequest(http.MethodGet, "/", nil))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("nomor", "A")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandlerListFormatsDateInJakarta(t *testing.T) {
	// Tanggal 17:30 UTC adalah esok hari di WIB.
	at := time.Date(2026, 9, 20, 17, 30, 0, 0, time.UTC)
	svc := listStub{page: inboxacceptopenprotection.Page{
		Protections: []inboxacceptopenprotection.Protection{{Number: "A", Type: "2", InputDate: at}, {Number: "B"}},
		Total:       2,
	}}
	h := newDirectHandler(svc, func(context.Context) (accepthttp.Caller, bool) {
		return accepthttp.Caller{Login: "X"}, true
	})

	rec := httptest.NewRecorder()
	h.List(rec, withPortal(httptest.NewRequest(http.MethodGet, "/", nil)))
	require.Equal(t, http.StatusOK, rec.Code)

	body := decode[listBody](t, rec)
	require.Equal(t, "2026-09-21", body.Proteksi[0].TanggalProteksi)
	require.Equal(t, "premi", body.Proteksi[0].Antrean)
	require.Empty(t, body.Proteksi[1].TanggalProteksi)
}

type listStub struct {
	stubService
	page inboxacceptopenprotection.Page
}

func (s listStub) List(context.Context, usecase.ListQuery) (inboxacceptopenprotection.Page, error) {
	return s.page, nil
}

// TestWriteErrorMapsUnknownCauseOfLoss menjaga galat master DIBEDAKAN dari galat data klaim.
//
// Keduanya 409 dan keduanya membatalkan keputusan, tetapi pekerjaan perbaikannya berbeda:
// yang satu di master penyebab kerugian, yang lain di data klaim. Pesan yang sama untuk
// keduanya membuat yang pertama dicoba hampir pasti yang salah.
func TestWriteErrorMapsUnknownCauseOfLoss(t *testing.T) {
	write := accepthttp.WriteError(discardLogger(), writeJSON, nil)

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		inboxacceptopenprotection.ErrUnknownCauseOfLoss)

	require.Equal(t, http.StatusConflict, rec.Code)
	body := decode[errorBody](t, rec)
	require.Equal(t, accepthttp.CodeConflict, body.Code)
	require.Contains(t, body.Message, "Penyebab Kerugian")
	require.Contains(t, body.Message, "master")

	// Pesannya TIDAK menyebut klaim: yang harus dibereskan bukan di sana.
	require.NotContains(t, body.Message, "daftar klaim")
}

// TestPesanClaimNotSyncedTidakMenyebutDOLSaja menjaga pesan tetap benar bagi KEDUA tipe.
//
// Sejak perubahan Penyebab Kerugian ikut diterapkan, galat yang sama muncul ketika baris
// coverage yang ditunjuk sudah dibuang dari klaim. Pesan yang menyebut Tanggal Kejadian akan
// menyuruh petugas memeriksa hal yang sama sekali tidak berhubungan.
func TestPesanClaimNotSyncedTidakMenyebutDOLSaja(t *testing.T) {
	write := accepthttp.WriteError(discardLogger(), writeJSON, nil)

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequest(http.MethodGet, "/x", nil),
		inboxacceptopenprotection.ErrClaimNotSynced)

	body := decode[errorBody](t, rec)
	require.NotContains(t, body.Message, "Tanggal Kejadian")
	require.Contains(t, body.Message, "coverage")
}
