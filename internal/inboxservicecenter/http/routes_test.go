package inboxservicecenterhttp_test

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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/repo/memory"
	"claim-pnc/internal/inboxservicecenter/usecase"
	"claim-pnc/internal/portal"

	inboxservicecenterhttp "claim-pnc/internal/inboxservicecenter/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const routeInbox = "/api/inbox-service-center"

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// write500 adalah ujung rantai galat: apa pun yang tidak dikenali menjadi 500 tanpa badan.
func write500(w http.ResponseWriter, r *http.Request, _ error) {
	writeJSON(w, r, http.StatusInternalServerError, nil)
}

// failingRepo selalu gagal, untuk membuktikan galat yang tidak dikenali menjadi 500.
type failingRepo struct{}

func (failingRepo) List(context.Context, inboxservicecenter.Query, inboxservicecenter.Pagination) (inboxservicecenter.Page, error) {
	return inboxservicecenter.Page{}, errors.New("ORA-03113 rahasia")
}

func (failingRepo) FindDetail(context.Context, inboxservicecenter.DetailQuery) (inboxservicecenter.ClaimDetail, error) {
	return inboxservicecenter.ClaimDetail{}, errors.New("ORA-03113 rahasia")
}

func (failingRepo) ListProgress(context.Context, string) ([]inboxservicecenter.ProgressNote, error) {
	return nil, errors.New("ORA-03113 rahasia")
}

type serverOptions struct {
	login    string
	noCaller bool
	fallback bool
	logs     *bytes.Buffer
}

// newServer merakit rute modul di balik middleware portal, sama seperti cmd/claimpnc.
// Portal ASM berisi data contoh; ASI berisi repo yang selalu gagal.
func newServer(t *testing.T, o serverOptions) (http.Handler, *inboxservicecenterhttp.Handler) {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxservicecenter.Repo, error) {
			switch alias {
			case "ASM":
				return memory.NewSampleStore(), nil
			case "ASI":
				return failingRepo{}, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	out := io.Writer(io.Discard)
	if o.logs != nil {
		out = o.logs
	}
	logger := slog.New(slog.NewTextHandler(out, nil))

	var fallback inboxservicecenterhttp.ErrorWriter
	if o.fallback {
		fallback = inboxservicecenterhttp.ErrorWriter(portalhttp.WithPortalError(write500, writeJSON))
	}

	opts := inboxservicecenterhttp.Options{
		Service:             service,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: fallback,
	}
	if !o.noCaller {
		login := o.login
		opts.GetCaller = func(context.Context) (inboxservicecenterhttp.Caller, bool) {
			return inboxservicecenterhttp.Caller{Login: login}, true
		}
	}
	handler := inboxservicecenterhttp.NewHandler(opts)

	portalError := portalhttp.WithPortalError(write500, writeJSON)
	deps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   portalError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		inboxservicecenterhttp.Mount(api, handler, deps)
	})
	return router, handler
}

func call(t *testing.T, h http.Handler, path, portalAlias string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if portalAlias != "" {
		req.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

func TestMetadataListsFourTabs(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"/tab", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ASM", body["portal"])
	require.Equal(t, inboxservicecenter.DefaultTab, body["tab_bawaan"])
	require.Len(t, body["tab"], 4)
	require.NotEmpty(t, body["keterbatasan"])

	first := body["tab"].([]any)[0].(map[string]any)
	require.Equal(t, inboxservicecenter.TabRegistration, first["kode"])
	require.Len(t, first["kolom"], 6)
}

func TestWithoutPortalRejected(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})
	for _, path := range []string{routeInbox, routeInbox + "/tab", routeInbox + "/SC-000101"} {
		rec, body := call(t, server, path, "")
		require.Equal(t, http.StatusBadRequest, rec.Code, path)
		require.Equal(t, portalhttp.CodeNotStated, body["kode"], path)
	}
}

func TestListReturnsRowsAndPagination(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"?tab="+inboxservicecenter.TabRegistration+"&halaman=2&ukuran=1", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ASM", body["portal"])

	rows := body["baris"].([]any)
	require.Len(t, rows, 1)
	row := rows[0].(map[string]any)
	require.Equal(t, "SC-000102", row["id"])
	require.Nil(t, row["tanggal_input"], "tanggal kosong dikirim null")
	require.Equal(t, "Repair Assesment", row["status_perbaikan_label"])
	require.Equal(t, "Belum diajukan", row["status_persetujuan_label"])

	paging := body["paginasi"].(map[string]any)
	require.Equal(t, map[string]any{
		"halaman": float64(2), "ukuran": float64(1), "total": float64(2),
		"total_halaman": float64(2), "aktif": true,
	}, paging)

	// Halaman pertama membawa tanggal berformat YYYY-MM-DD.
	_, body = call(t, server, routeInbox+"?halaman=abc&ukuran=-3", "ASM")
	first := body["baris"].([]any)[0].(map[string]any)
	require.Equal(t, "SC-000101", first["id"])
	require.Equal(t, "2026-09-22", first["tanggal_input"])
	require.Equal(t, float64(1), body["paginasi"].(map[string]any)["halaman"])
	require.Equal(t, float64(inboxservicecenter.DefaultPageSize), body["paginasi"].(map[string]any)["ukuran"])
}

func TestListSearchEchoesKeyword(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"?tab="+inboxservicecenter.TabApproved+"&cari=%20imei%20", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, map[string]any{"cari": "imei"}, body["penyaring"])
	require.Equal(t, false, body["paginasi"].(map[string]any)["aktif"])
	require.Equal(t, inboxservicecenter.TabApproved, body["tab"].(map[string]any)["kode"])
}

func TestListUnknownTabIsValidationError(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"?tab=tidakada", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, inboxservicecenterhttp.CodeValidationFail, body["kode"])
	require.Equal(t, []any{map[string]any{"field": "tab", "pesan": "Tab tidak dikenal."}}, body["detail"])
}

func TestCallerUnknownIsConflict(t *testing.T) {
	for name, o := range map[string]serverOptions{
		"tanpa pembaca": {noCaller: true, fallback: true},
		"login kosong":  {login: "", fallback: true},
	} {
		t.Run(name, func(t *testing.T) {
			server, _ := newServer(t, o)
			for _, path := range []string{routeInbox, routeInbox + "/SC-000101"} {
				rec, body := call(t, server, path, "ASM")
				require.Equal(t, http.StatusConflict, rec.Code, path)
				require.Equal(t, inboxservicecenterhttp.CodeCallerUnknown, body["kode"], path)
			}
		})
	}
}

func TestDetailReturnsClaimProgressAndGroups(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"/SC-000101", "ASM")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ASM", body["portal"])

	claim := body["klaim"].(map[string]any)
	require.Equal(t, "SC-000101", claim["id"])
	require.Equal(t, "SIMAS INSURTECH", claim["asuransi"])
	require.Equal(t, "2026-01-15", claim["active_date_warranty"])
	require.Equal(t, "Repair Submitted", claim["repair_status_label"])
	require.Equal(t, "Belum diajukan", claim["status_approval_label"])
	require.Equal(t, "2214500", claim["total_biaya"])
	require.Equal(t, "Retak", claim["lcd_text"])

	progress := body["riwayat_progres"].([]any)
	require.Len(t, progress, 2)
	require.Equal(t, map[string]any{
		"tanggal": "2026-09-22", "catatan": "Unit diterima di gerai, kelengkapan dicek.",
		"oleh": memory.SampleOwner,
	}, progress[0])

	groups := body["kelompok"].([]any)
	require.Len(t, groups, 7)
	general := groups[0].(map[string]any)
	require.Equal(t, inboxservicecenter.GroupGeneral, general["kode"])
	require.Equal(t, map[string]any{"kunci": "id", "judul": "ID"}, general["isian"].([]any)[0])
}

func TestDetailNotFound(t *testing.T) {
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	rec, body := call(t, server, routeInbox+"/SC-000103", "ASM")
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inboxservicecenterhttp.CodeNotFound, body["kode"])
}

// Galat yang tidak dikenali dan tanpa fallback menjadi 500 berpesan umum; rinciannya di log.
func TestUnrecognizedErrorWithoutFallbackIs500(t *testing.T) {
	logs := &bytes.Buffer{}
	server, _ := newServer(t, serverOptions{login: memory.SampleOwner, logs: logs})

	for _, path := range []string{routeInbox, routeInbox + "/SC-1"} {
		rec, body := call(t, server, path, "ASI")
		require.Equal(t, http.StatusInternalServerError, rec.Code, path)
		require.Equal(t, inboxservicecenterhttp.CodeInternalError, body["kode"])
		require.Equal(t, "Terjadi kesalahan pada sistem.", body["pesan"])
		require.NotContains(t, rec.Body.String(), "ORA-03113")
	}
	require.Contains(t, logs.String(), "ORA-03113")
	require.Contains(t, logs.String(), "permintaan gagal")
}

// Galat yang tidak dikenali diserahkan ke fallback bila ada.
func TestUnrecognizedErrorGoesToFallback(t *testing.T) {
	var got error
	writer := inboxservicecenterhttp.WriteError(nil, writeJSON,
		func(w http.ResponseWriter, _ *http.Request, err error) {
			got = err
			w.WriteHeader(http.StatusTeapot)
		})

	rec := httptest.NewRecorder()
	boom := errors.New("lain")
	writer(rec, httptest.NewRequest(http.MethodGet, "/x", nil), boom)
	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Equal(t, boom, got)

	// Tanpa logger pun 500 tetap ditulis.
	rec = httptest.NewRecorder()
	inboxservicecenterhttp.WriteError(nil, writeJSON, nil)(rec, httptest.NewRequest(http.MethodGet, "/x", nil), boom)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Handler yang dipanggil tanpa middleware portal menolak dengan ErrNotStated.
func TestHandlersWithoutActivePortal(t *testing.T) {
	_, handler := newServer(t, serverOptions{login: memory.SampleOwner, fallback: true})

	for name, serve := range map[string]http.HandlerFunc{
		"metadata": handler.Metadata,
		"list":     handler.List,
		"detail":   handler.Detail,
	} {
		rec := httptest.NewRecorder()
		serve(rec, httptest.NewRequest(http.MethodGet, routeInbox, nil))
		require.Equal(t, http.StatusBadRequest, rec.Code, name)

		body := map[string]any{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, portalhttp.CodeNotStated, body["kode"], name)
	}
}
