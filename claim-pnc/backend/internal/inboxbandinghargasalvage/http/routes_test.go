package inboxbandinghargasalvagehttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
	"claim-pnc/internal/inboxbandinghargasalvage/usecase"
	"claim-pnc/internal/portal"

	salvagehttp "claim-pnc/internal/inboxbandinghargasalvage/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji rute modul Inbox Banding Harga Salvage lewat router chi: portal dibaca middleware modul
// portal, identitas dibaca dari header uji lewat jembatan yang sama bentuknya dengan cmd.

const headerLogin = "X-Uji-Login"

type loginKey struct{}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type harness struct {
	router http.Handler
	log    *bytes.Buffer
}

type options struct {
	repo       inboxbandinghargasalvage.Repo
	noWriter   bool
	noDocument bool
	noFallback bool
}

func newHarness(t *testing.T, o options) *harness {
	t.Helper()

	store := memory.NewSampleStore()
	var repo inboxbandinghargasalvage.Repo = store
	if o.repo != nil {
		repo = o.repo
	}

	log := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(log, nil))

	serviceOptions := usecase.Options{
		RepoSelector: func(alias string) (inboxbandinghargasalvage.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		Logger: logger,
	}
	if !o.noWriter {
		writer := memory.NewWriter(store)
		serviceOptions.WriterSelector = func(string) (inboxbandinghargasalvage.Writer, error) {
			return writer, nil
		}
	}
	if !o.noDocument {
		documents := memory.NewSampleDocumentStore(store)
		serviceOptions.DocumentSelector = func(string) (inboxbandinghargasalvage.DocumentReader, error) {
			return documents, nil
		}
	}
	service, err := usecase.NewService(serviceOptions)
	require.NoError(t, err)

	writeError := portalhttp.WithPortalError(func(w http.ResponseWriter, r *http.Request, _ error) {
		writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
	}, writeJSON)

	handlerOptions := salvagehttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (salvagehttp.Caller, bool) {
			login, ok := ctx.Value(loginKey{}).(string)
			if !ok {
				return salvagehttp.Caller{}, false
			}
			return salvagehttp.Caller{Login: login}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
	}
	if !o.noFallback {
		handlerOptions.FallbackErrorWriter = salvagehttp.ErrorWriter(writeError)
	}
	handler := salvagehttp.NewHandler(handlerOptions)

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), loginKey{}, r.Header.Get(headerLogin))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	salvagehttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	})
	return &harness{router: router, log: log}
}

func (h *harness) do(
	t *testing.T, method, path, login, body string,
) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var payload io.Reader
	if body != "" {
		payload = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	if login != "" {
		request.Header.Set(headerLogin, login)
	}

	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)

	content := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &content)
	return recorder, content
}

func (h *harness) get(t *testing.T, path string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	return h.do(t, http.MethodGet, path, memory.SampleOwner, "")
}

const base = "/inbox-banding-harga-salvage"

func TestMetadataDescribesBothTabs(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"/tab")
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, body["tab"].([]any), 2)
	require.Equal(t, inboxbandinghargasalvage.DefaultTab, body["tab_bawaan"])
	require.Equal(t, "Cari No Klaim", body["label_cari"])
	require.NotEmpty(t, body["kolom_rincian"])
	require.NotContains(t, body, "selisih_terencana")
	require.NotEmpty(t, body["keterbatasan"])
	require.Equal(t, "ASM", body["portal"])
}

// Isi tab Request milik komite pemanggil, dengan umur sebagai teks "<n> days".
func TestListReturnsTheCommitteeQueue(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"?tab="+inboxbandinghargasalvage.TabRequest+
		"&halaman=abc&ukuran=-1")
	require.Equal(t, http.StatusOK, response.Code)
	rows := body["baris"].([]any)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		item := row.(map[string]any)
		require.Equal(t, memory.SampleOwner, item["nama_komite"])
		if item["tanggal_request"] != nil {
			require.Contains(t, item["aging"], " days")
		} else {
			require.Equal(t, "", item["aging"])
		}
	}

	paging := body["paginasi"].(map[string]any)
	require.Equal(t, 1.0, paging["halaman"])
	require.Equal(t, float64(inboxbandinghargasalvage.DefaultPageSize), paging["ukuran"])
	require.Equal(t, memory.SampleOwner, body["antrean"].(map[string]any)["milik"])
}

// Tab History membawa kolom pengajuan salvage.
func TestListHistoryTab(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"?tab="+inboxbandinghargasalvage.TabHistory)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, inboxbandinghargasalvage.TabHistory,
		body["tab"].(map[string]any)["kode"])
}

func TestListErrors(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.do(t, http.MethodGet, base, "", "")
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, salvagehttp.CodeCallerUnknown, body["kode"])

	response, body = h.get(t, base+"?tab=entah")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, salvagehttp.CodeValidationFail, body["kode"])
	require.NotEmpty(t, body["detail"])

	// Galat yang bukan milik modul diserahkan ke penulis cadangan.
	broken := newHarness(t, options{repo: repoRusak{}})
	response, body = broken.get(t, base)
	require.Equal(t, http.StatusTeapot, response.Code)
	require.Equal(t, "cadangan", body["kode"])
}

// Tanpa cadangan, galat asing menjadi 500 umum dan rinciannya hanya masuk log.
func TestUnknownErrorsWithoutAFallbackAreGeneric(t *testing.T) {
	h := newHarness(t, options{repo: repoRusak{}, noFallback: true})

	response, body := h.get(t, base+"/ringkas")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, salvagehttp.CodeInternalError, body["kode"])
	require.NotContains(t, response.Body.String(), "oracle rusak")
	require.Contains(t, h.log.String(), "oracle rusak")
}

// Tabel ringkas menghitung kedua tab.
func TestSummaryCountsBothTabs(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"/ringkas")
	require.Equal(t, http.StatusOK, response.Code)
	rows := body["baris"].([]any)
	require.Len(t, rows, 2)
	require.Equal(t, inboxbandinghargasalvage.TabRequest, rows[0].(map[string]any)["tab"])

	response, _ = h.do(t, http.MethodGet, base+"/ringkas", "", "")
	require.Equal(t, http.StatusConflict, response.Code)
}

// Keputusan disimpan lalu terbaca di panel rincian klaimnya; penekanan kedua ditolak.
func TestDecideThenReadTheDecision(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner,
		`{"detail_object":"PNC-0451/1","id_salvage":"451","harga_request":"9750000.00",`+
			`"catatan":"cek","setujui":true}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, true, body["tersimpan"])
	require.Contains(t, body["pesan"], "Persetujuan tersimpan")
	require.Contains(t, body["pesan"], "Balai lelang TIDAK diberi tahu")

	response, body = h.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner,
		`{"detail_object":"PNC-0451/1","id_salvage":"451","harga_request":"1","setujui":true}`)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, salvagehttp.CodeAlreadyDecided, body["kode"])

	response, body = h.get(t, base+"/riwayat/PNCN.26.0451")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "PNCN.26.0451", body["no_klaim"])
	items := body["baris"].([]any)
	require.Len(t, items, 1)
	decision := items[0].(map[string]any)
	require.Equal(t, inboxbandinghargasalvage.DecisionApproved, decision["jawaban_checker_kode"])
	require.NotEmpty(t, decision["jawaban_checker"])
	require.NotNil(t, decision["tanggal_approve"])
}

// Persetujuan jenjang terakhir menerapkan harga, dan pesannya menyatakan itu.
func TestFinalApprovalAppliesThePrice(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.do(t, http.MethodPost, base+"/keputusan", "DANIELLISWANDI",
		`{"detail_object":"PNC-0461/1","id_salvage":"461","harga_request":"8200000.00",`+
			`"setujui":true}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, true, body["harga_diterapkan"])
	require.Contains(t, body["pesan"], "Harga barang diperbarui")
}

// Nomor halaman yang sah dipakai apa adanya.
func TestAValidPageNumberIsUsed(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"?halaman=2&ukuran=1")
	require.Equal(t, http.StatusOK, response.Code)
	paging := body["paginasi"].(map[string]any)
	require.Equal(t, 2.0, paging["halaman"])
	require.Equal(t, 1.0, paging["ukuran"])
}

// Penolakan menyebut penolakan, tanpa kalimat tentang harga.
func TestRejectingSaysSo(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner,
		`{"detail_object":"PNC-0451/1","id_salvage":"451","setujui":false}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, body["pesan"], "Penolakan tersimpan")
	require.NotContains(t, body["pesan"], "Harga barang")
}

func TestDecideErrors(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner, "{")
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, salvagehttp.CodeBadRequest, body["kode"])

	response, body = h.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner,
		`{"setujui":true}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, salvagehttp.CodeValidationFail, body["kode"])

	response, _ = h.do(t, http.MethodPost, base+"/keputusan", "", `{}`)
	require.Equal(t, http.StatusConflict, response.Code)

	readOnly := newHarness(t, options{noWriter: true})
	response, body = readOnly.do(t, http.MethodPost, base+"/keputusan", memory.SampleOwner,
		`{"detail_object":"PNC-0451/1","id_salvage":"451","setujui":false}`)
	require.Equal(t, http.StatusNotImplemented, response.Code)
	require.Equal(t, salvagehttp.CodeWriteNotBuilt, body["kode"])
}

func TestDecisionsRejectAnUnknownCaller(t *testing.T) {
	h := newHarness(t, options{})

	response, _ := h.do(t, http.MethodGet, base+"/riwayat/PNCN.26.0451", "", "")
	require.Equal(t, http.StatusConflict, response.Code)

	broken := newHarness(t, options{repo: repoRusak{}})
	response, _ = broken.get(t, base+"/riwayat/PNCN.26.0451")
	require.Equal(t, http.StatusTeapot, response.Code)
}

func documentQuery(detail, salvage string) string {
	values := url.Values{}
	values.Set("detail_object", detail)
	values.Set("id_salvage", salvage)
	return values.Encode()
}

// Dialog Lihat File: daftar dokumen dan unduhan isinya.
func TestDocumentsAndDownload(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"/dokumen?"+documentQuery("PNC-0451/1", "451"))
	require.Equal(t, http.StatusOK, response.Code)
	rows := body["baris"].([]any)
	require.Len(t, rows, 2)
	first := rows[0].(map[string]any)
	require.Equal(t, "9001", first["id"])
	require.Equal(t, inboxbandinghargasalvage.DocumentCategory, first["kategori"])

	download, _ := h.get(t, base+"/dokumen/9001?"+documentQuery("PNC-0451/1", "451"))
	require.Equal(t, http.StatusOK, download.Code)
	require.Equal(t, "application/pdf", download.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="penawaran-balai-lelang.pdf"`,
		download.Header().Get("Content-Disposition"))
	require.Equal(t, "nosniff", download.Header().Get("X-Content-Type-Options"))
	require.True(t, strings.HasPrefix(download.Body.String(), "%PDF"))
}

func TestDocumentErrors(t *testing.T) {
	h := newHarness(t, options{})

	response, body := h.get(t, base+"/dokumen?"+documentQuery("", ""))
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Len(t, body["detail"].([]any), 2)

	response, body = h.get(t, base+"/dokumen/9004?"+documentQuery("PNC-0456/1", "456"))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, salvagehttp.CodeDocumentGone, body["kode"])

	response, _ = h.do(t, http.MethodGet, base+"/dokumen?"+documentQuery("D", "1"), "", "")
	require.Equal(t, http.StatusConflict, response.Code)

	response, _ = h.do(t, http.MethodGet, base+"/dokumen/1?"+documentQuery("D", "1"), "", "")
	require.Equal(t, http.StatusConflict, response.Code)
}

// Tipe isi kosong menjadi octet-stream; nama berkas dibersihkan dari karakter berbahaya.
func TestDownloadHeadersAreSafe(t *testing.T) {
	store := memory.NewSampleStore()
	documents := memory.NewDocumentStore(store,
		memory.DocumentRecord{ID: "1", DetailObject: "PNC-0451/1", SalvageID: "451",
			Name: `..\"jahat"/berkas.html`, Content: []byte("<script>")},
		memory.DocumentRecord{ID: "2", DetailObject: "PNC-0451/1", SalvageID: "451",
			Name: `"/"`, MIMEType: "image/png", Content: []byte("png")},
	)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) { return store, nil },
		DocumentSelector: func(string) (inboxbandinghargasalvage.DocumentReader, error) {
			return documents, nil
		},
	})
	require.NoError(t, err)

	handler := salvagehttp.NewHandler(salvagehttp.Options{
		Service: service,
		GetCaller: func(context.Context) (salvagehttp.Caller, bool) {
			return salvagehttp.Caller{Login: memory.SampleOwner}, true
		},
		WriteJSON: writeJSON,
	})

	download := func(id string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet,
			"/?"+documentQuery("PNC-0451/1", "451"), nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("dokumen", id)
		ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routeContext)
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
		recorder := httptest.NewRecorder()
		handler.DocumentContent(recorder, request.WithContext(ctx))
		return recorder
	}

	first := download("1")
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "application/octet-stream", first.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="..jahatberkas.html"`,
		first.Header().Get("Content-Disposition"))

	second := download("2")
	require.Equal(t, "image/png", second.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="dokumen-banding-salvage"`,
		second.Header().Get("Content-Disposition"))
}

// Handler di luar middleware portal, dan tanpa jembatan identitas, menolak.
func TestHandlersOutsideThePortalMiddleware(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxbandinghargasalvage.Repo, error) {
			return memory.NewSampleStore(), nil
		},
	})
	require.NoError(t, err)

	var received []error
	handler := salvagehttp.NewHandler(salvagehttp.Options{
		Service:   service,
		WriteJSON: writeJSON,
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			received = append(received, err)
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	for _, call := range []http.HandlerFunc{handler.Metadata, handler.List} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
	require.Len(t, received, 2)
	for _, err := range received {
		require.ErrorIs(t, err, portal.ErrNotStated)
	}

	// Tanpa jembatan identitas, pemanggil tidak dikenal.
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(portalhttp.WithActivePortal(
		request.Context(), portal.Portal{Alias: "ASM"}))
	recorder := httptest.NewRecorder()
	handler.Summary(recorder, request)
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// repoRusak menjawab setiap pembacaan dengan galat.
type repoRusak struct{}

var errRusak = errors.New("oracle rusak")

func (repoRusak) List(
	context.Context, inboxbandinghargasalvage.Query, inboxbandinghargasalvage.Pagination,
) (inboxbandinghargasalvage.Page, error) {
	return inboxbandinghargasalvage.Page{}, errRusak
}

func (repoRusak) Count(context.Context, inboxbandinghargasalvage.Query) (int, error) {
	return 0, errRusak
}

func (repoRusak) ListDecisions(
	context.Context, inboxbandinghargasalvage.DecisionQuery,
) ([]inboxbandinghargasalvage.Decision, error) {
	return nil, errRusak
}
