package inboxsalvagehttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
	"claim-pnc/internal/inboxsalvage/usecase"
	"claim-pnc/internal/portal"

	inboxsalvagehttp "claim-pnc/internal/inboxsalvage/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const portalASM = "ASM"

var errBoom = errors.New("penyimpanan mati")

// serverOptions mengatur cara server uji dirakit.
type serverOptions struct {
	repo       inboxsalvage.Repo
	noCaller   bool
	nilCaller  bool
	noFallback bool
	logs       *bytes.Buffer
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// newServer merakit rute modul di balik pemeriksaan portal, sama seperti cmd/claimpnc.
func newServer(t *testing.T, o serverOptions) http.Handler {
	t.Helper()

	repo := o.repo
	if repo == nil {
		repo = memory.NewSampleStore()
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxsalvage.Repo, error) {
			if alias == portalASM {
				return repo, nil
			}
			return nil, portal.ErrNotReady
		},
	})
	require.NoError(t, err)

	var sink io.Writer = io.Discard
	if o.logs != nil {
		sink = o.logs
	}
	logger := slog.New(slog.NewTextHandler(sink, nil))

	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_cadangan", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	var fallback inboxsalvagehttp.ErrorWriter = inboxsalvagehttp.ErrorWriter(writeError)
	if o.noFallback {
		fallback = nil
	}

	var getCaller inboxsalvagehttp.CallerReader = func(context.Context) (inboxsalvagehttp.Caller, bool) {
		return inboxsalvagehttp.Caller{Login: memory.SampleCallerPIC}, !o.noCaller
	}
	if o.nilCaller {
		getCaller = nil
	}

	handler := inboxsalvagehttp.NewHandler(inboxsalvagehttp.Options{
		Service:             service,
		GetCaller:           getCaller,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: fallback,
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{portalASM} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		inboxsalvagehttp.Mount(api, handler, portalDeps)
	})
	return router
}

func do(t *testing.T, server http.Handler, method, path string, body io.Reader,
	contentType, portalAlias string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, server, http.MethodGet, path, nil, "", portalASM)
}

func postJSON(t *testing.T, server http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	return do(t, server, http.MethodPost, path, bytes.NewReader(raw), "application/json", portalASM)
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &value))
	return value
}

// ── Keterangan layar ────────────────────────────────────────────────────────────

func TestMetadataDescribesTheOfferedTabsAndThePortal(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}), "/api/inbox-salvage/daftar")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxsalvagehttp.MetadataResponse](t, recorder)
	require.Len(t, body.Tabs, len(inboxsalvage.Tabs()))
	require.Equal(t, inboxsalvage.DefaultTab, body.DefaultTab)
	require.Equal(t, portalASM, body.Portal)
	require.NotEmpty(t, body.StatusOptions)
	require.Contains(t, body.UploadColumns, "item")

	// Kunci rincian diturunkan dari keluarga kueri tiap daftar.
	keys := map[string]string{}
	for _, tab := range body.Tabs {
		keys[tab.Code] = tab.DetailKey
		require.NotEmpty(t, tab.Columns)
	}
	require.Equal(t, string(inboxsalvage.DetailKeyClaim), keys[inboxsalvage.TabOutstanding])
	require.Equal(t, string(inboxsalvage.DetailKeySubmission), keys[inboxsalvage.TabHistori])
}

// Permintaan tanpa portal ditolak pada SETIAP rute, termasuk yang tidak memeriksa identitas.
func TestEveryRouteRejectsARequestWithoutPortal(t *testing.T) {
	server := newServer(t, serverOptions{})

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/inbox-salvage/daftar"},
		{http.MethodGet, "/api/inbox-salvage"},
		{http.MethodGet, "/api/inbox-salvage/ringkas"},
		{http.MethodGet, "/api/inbox-salvage/pengajuan/1"},
		{http.MethodGet, "/api/inbox-salvage/klaim/PNC-1"},
		{http.MethodGet, "/api/inbox-salvage/ekspor"},
		{http.MethodPost, "/api/inbox-salvage"},
		{http.MethodPost, "/api/inbox-salvage/unggah-detail"},
		{http.MethodPost, "/api/inbox-salvage/tindakan"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			recorder := do(t, server, route.method, route.path, nil, "", "")
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
			require.Equal(t, portalhttp.CodeNotStated, body.Code)
		})
	}
}

// ── Isi daftar ──────────────────────────────────────────────────────────────────

func TestListAnswersTheDefaultTabWithRowsAndPaging(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}),
		"/api/inbox-salvage?halaman=abc&ukuran=-3")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxsalvagehttp.ListResponse](t, recorder)
	require.Equal(t, inboxsalvage.TabOutstanding, body.Tab.Code)
	require.Equal(t, portalASM, body.Portal)
	require.Equal(t, 1, body.Paging.Page, "halaman tak terbaca dibetulkan")
	require.Equal(t, inboxsalvage.DefaultPageSize, body.Paging.Size)
	require.Equal(t, len(body.Rows), body.Paging.Total)
	require.GreaterOrEqual(t, body.Paging.TotalPages, 1)
	require.NotEmpty(t, body.Rows)
	require.NotEmpty(t, body.Rows[0].ClaimNo)
	require.Equal(t, body.Rows[0].ClaimNo, body.Rows[0].Reference)
}

func TestListOfASalvageTabCarriesTheSearchItUsed(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}),
		"/api/inbox-salvage?daftar=+histori+&cari=+PNC+&halaman=1&ukuran=5")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxsalvagehttp.ListResponse](t, recorder)
	require.Equal(t, inboxsalvage.TabHistori, body.Tab.Code)
	require.Equal(t, "PNC", body.Search)
	require.Equal(t, 5, body.Paging.Size)
	require.NotEmpty(t, body.Rows)
	require.NotEmpty(t, body.Rows[0].SalvageID)
}

func TestListUnknownTabAnswered422WithTheField(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}), "/api/inbox-salvage?daftar=checker")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxsalvagehttp.CodeValidationFail, body.Code)
	require.Len(t, body.Details, 1)
	require.Equal(t, inboxsalvage.FieldTab, body.Details[0].Field)
}

// Sesi tanpa identitas dijawab 409, baik karena jembatannya tidak ada maupun karena
// identitasnya tidak terbaca.
func TestUnknownCallerAnswered409(t *testing.T) {
	for name, options := range map[string]serverOptions{
		"identitas tidak terbaca": {noCaller: true},
		"jembatan tidak dipasang": {nilCaller: true},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := get(t, newServer(t, options), "/api/inbox-salvage")
			require.Equal(t, http.StatusConflict, recorder.Code)
			body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
			require.Equal(t, inboxsalvagehttp.CodeCallerUnknown, body.Code)
		})
	}
}

// Galat yang bukan milik modul diserahkan ke penulis cadangan.
func TestRepoFailureIsHandedToTheFallbackWriter(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{repo: brokenRepo{}}), "/api/inbox-salvage")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "galat_cadangan")
}

// Tanpa penulis cadangan, galat asing menjadi 500 berpesan umum dan rinciannya hanya di log.
func TestUnrecognizedErrorWithoutFallbackAnswered500AndLogged(t *testing.T) {
	var logs bytes.Buffer
	recorder := get(t, newServer(t, serverOptions{
		repo: brokenRepo{}, noFallback: true, logs: &logs,
	}), "/api/inbox-salvage")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxsalvagehttp.CodeInternalError, body.Code)
	require.Equal(t, "Terjadi kesalahan pada sistem.", body.Message)
	require.NotContains(t, recorder.Body.String(), errBoom.Error(), "rincian tidak bocor")
	require.Contains(t, logs.String(), "permintaan gagal")
	require.Contains(t, logs.String(), errBoom.Error())
}

// ── Tabel ringkas ───────────────────────────────────────────────────────────────

func TestCountsAnswersOneRowPerVisibleCounter(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}), "/api/inbox-salvage/ringkas")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxsalvagehttp.CountsResponse](t, recorder)
	require.Equal(t, portalASM, body.Portal)
	require.Len(t, body.Rows, len(inboxsalvage.CountRows()))
	require.Equal(t, "Outstanding", body.Rows[0].Label)
	require.Equal(t, inboxsalvage.TabOutstanding, body.Rows[0].Tab)
}

func TestCountsFailureIsHandedOn(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{repo: brokenRepo{}}), "/api/inbox-salvage/ringkas")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	recorder = get(t, newServer(t, serverOptions{noCaller: true}), "/api/inbox-salvage/ringkas")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// ── Penyimpanan ─────────────────────────────────────────────────────────────────

func validRequest() inboxsalvagehttp.CreateRequest {
	return inboxsalvagehttp.CreateRequest{
		ClaimNo:       "pnc-2044",
		ObjectName:    "Panel Listrik",
		CoverageName:  "Property All Risk",
		SalvageType:   "Besi Tua",
		InputDate:     "2026-09-25",
		InJabodetabek: true,
		Items: []inboxsalvagehttp.DetailItemRequest{
			{Name: "Besi", Quantity: "2", Unit: "kg", Remarks: "karat"},
			{},
		},
	}
}

func TestCreateAnswered201WithTheIssuedIDAndWhatDidNotHappen(t *testing.T) {
	store := memory.NewSampleStore()
	server := newServer(t, serverOptions{repo: store})

	recorder := postJSON(t, server, "/api/inbox-salvage", validRequest())
	require.Equal(t, http.StatusCreated, recorder.Code)

	body := decode[inboxsalvagehttp.CreateResponse](t, recorder)
	require.NotEmpty(t, body.SalvageID)
	require.Equal(t, 1, body.ItemCount, "baris kosong dibuang")
	require.Equal(t, portalASM, body.Portal)
	require.Contains(t, body.Message, "tidak dikirim ke balai lelang")

	// Pengajuannya benar-benar tersimpan dan dapat dibuka.
	detail := get(t, server, "/api/inbox-salvage/pengajuan/"+body.SalvageID)
	require.Equal(t, http.StatusOK, detail.Code)
	opened := decode[inboxsalvagehttp.DetailResponse](t, detail)
	require.Equal(t, "PNC-2044", opened.ClaimNo, "nomor klaim dibesarkan hurufnya")
}

func TestCreateMalformedOrUnknownFieldAnswered422(t *testing.T) {
	server := newServer(t, serverOptions{})

	for name, raw := range map[string]string{
		"bukan json":          `{bukan json`,
		"isian tidak dikenal": `{"nomor_klaim":"PNC-1","nilai_salah":"1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			recorder := do(t, server, http.MethodPost, "/api/inbox-salvage",
				strings.NewReader(raw), "application/json", portalASM)
			require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
			require.Equal(t, inboxsalvagehttp.CodeValidationFail, body.Code)
			require.Equal(t, inboxsalvage.FieldFormClaimNo, body.Details[0].Field)
		})
	}
}

// Mode ubah tanpa ID salvage ditolak, bukan diteruskan menjadi pengajuan baru.
func TestCreateUpdateWithoutSalvageIDAnswered422(t *testing.T) {
	request := validRequest()
	request.Mode = string(inboxsalvage.FormModeUpdate)

	recorder := postJSON(t, newServer(t, serverOptions{}), "/api/inbox-salvage", request)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Pengajuan yang diubah tidak dikenali")
}

func TestCreateUpdateOfAMissingSubmissionAnswered404(t *testing.T) {
	request := validRequest()
	request.Mode = string(inboxsalvage.FormModeUpdate)
	request.SalvageID = "99999"

	recorder := postJSON(t, newServer(t, serverOptions{}), "/api/inbox-salvage", request)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxsalvagehttp.CodeRowNotFound, body.Code)
}

func TestCreateWithoutCallerAnswered409(t *testing.T) {
	recorder := postJSON(t, newServer(t, serverOptions{noCaller: true}),
		"/api/inbox-salvage", validRequest())
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// ── Unggah detail ───────────────────────────────────────────────────────────────

func upload(t *testing.T, server http.Handler, field, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buffer bytes.Buffer
	form := multipart.NewWriter(&buffer)
	part, err := form.CreateFormFile(field, "detail.csv")
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, form.Close())

	return do(t, server, http.MethodPost, "/api/inbox-salvage/unggah-detail",
		&buffer, form.FormDataContentType(), portalASM)
}

func TestUploadReturnsTheParsedRowsWithoutSavingAnything(t *testing.T) {
	recorder := upload(t, newServer(t, serverOptions{}), "berkas",
		"Item,Quantity,Satuan,Remarks\nBesi,2,kg,karat\n,,,\nKayu,,,\n")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxsalvagehttp.UploadResponse](t, recorder)
	require.Equal(t, []inboxsalvagehttp.DetailItemRequest{
		{Name: "Besi", Quantity: "2", Unit: "kg", Remarks: "karat"},
		{Name: "Kayu"},
	}, body.Items)
	require.Contains(t, body.Message, "Belum ada yang tersimpan")
}

func TestUploadWithoutFileAnswered422OnTheFileField(t *testing.T) {
	recorder := upload(t, newServer(t, serverOptions{}), "lain", "Item\nBesi\n")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxsalvagehttp.CodeValidationFail, body.Code)
	require.Equal(t, inboxsalvage.FieldFormFile, body.Details[0].Field)
}

// Ketiga galat berkas dijawab dengan kode yang sama tetapi kalimat yang berbeda.
func TestUploadFileProblemsAnsweredWithTheirOwnMessage(t *testing.T) {
	many := "Item\n" + strings.Repeat("Besi\n", 501)

	for name, c := range map[string]struct{ content, message string }{
		"kosong":            {"", "Berkas yang diunggah kosong"},
		"kolom item hilang": {"Nama;Jumlah\nBesi;1\n", "tidak memuat kolom \"Item\""},
		"terlalu banyak":    {many, "terlalu banyak baris"},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := upload(t, newServer(t, serverOptions{}), "berkas", c.content)
			require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
			require.Equal(t, inboxsalvagehttp.CodeUploadInvalid, body.Code)
			require.Contains(t, body.Message, c.message)
		})
	}
}

// Galat pembacaan CSV yang bukan galat domain diteruskan sebagai galat asing.
func TestUploadMalformedCSVIsHandedToTheFallback(t *testing.T) {
	recorder := upload(t, newServer(t, serverOptions{}), "berkas", "Item\n\"tidak ditutup\n")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "galat_cadangan")
}

func TestUploadWithoutCallerAnswered409(t *testing.T) {
	recorder := upload(t, newServer(t, serverOptions{noCaller: true}), "berkas", "Item\nBesi\n")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// ── Aksi yang belum dibangun ────────────────────────────────────────────────────

func TestUnbuiltActionAnswered501AndLogged(t *testing.T) {
	var logs bytes.Buffer
	recorder := do(t, newServer(t, serverOptions{logs: &logs}), http.MethodPost,
		"/api/inbox-salvage/tindakan?tindakan=+approve+", nil, "", portalASM)
	require.Equal(t, http.StatusNotImplemented, recorder.Code)

	body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxsalvagehttp.CodeWriteNotAvailable, body.Code)
	require.Contains(t, logs.String(), "tindakan=approve")
}

// ── Panel Detail Salvage ────────────────────────────────────────────────────────

func TestDetailBySubmissionAndByClaim(t *testing.T) {
	server := newServer(t, serverOptions{})

	list := decode[inboxsalvagehttp.ListResponse](t,
		get(t, server, "/api/inbox-salvage?daftar=histori"))
	require.NotEmpty(t, list.Rows)
	salvageID := list.Rows[0].SalvageID

	recorder := get(t, server, "/api/inbox-salvage/pengajuan/"+salvageID)
	require.Equal(t, http.StatusOK, recorder.Code)
	detail := decode[inboxsalvagehttp.DetailResponse](t, recorder)
	require.Equal(t, salvageID, detail.SalvageID)
	require.True(t, detail.HasSubmission)
	require.Equal(t, portalASM, detail.Portal)
	require.NotNil(t, detail.Items)
	require.NotNil(t, detail.History)

	recorder = get(t, server, "/api/inbox-salvage/klaim/"+memory.SampleClaimWithoutSalvage)
	require.Equal(t, http.StatusOK, recorder.Code)
	byClaim := decode[inboxsalvagehttp.DetailResponse](t, recorder)
	require.False(t, byClaim.HasSubmission)
	require.Equal(t, memory.SampleClaimWithoutSalvage, byClaim.ClaimNo)
}

// Klaim yang punya pengajuan membuka panel berisi barang dan riwayat.
func TestDetailByClaimWithSubmissionCarriesItemsAndHistory(t *testing.T) {
	server := newServer(t, serverOptions{})

	list := decode[inboxsalvagehttp.ListResponse](t,
		get(t, server, "/api/inbox-salvage?daftar=histori"))
	claimNo := list.Rows[0].ClaimNo

	recorder := get(t, server, "/api/inbox-salvage/klaim/"+claimNo)
	require.Equal(t, http.StatusOK, recorder.Code)
	detail := decode[inboxsalvagehttp.DetailResponse](t, recorder)
	require.True(t, detail.HasSubmission)
	require.NotEmpty(t, detail.History)
}

func TestDetailNotFoundAndBlankReferenceAnswered404(t *testing.T) {
	server := newServer(t, serverOptions{})

	for _, path := range []string{
		"/api/inbox-salvage/pengajuan/99999",
		"/api/inbox-salvage/pengajuan/%20",
		"/api/inbox-salvage/klaim/TIDAK-ADA",
	} {
		recorder := get(t, server, path)
		require.Equal(t, http.StatusNotFound, recorder.Code, path)
		body := decode[inboxsalvagehttp.ErrorResponse](t, recorder)
		require.Equal(t, inboxsalvagehttp.CodeRowNotFound, body.Code)
		require.Contains(t, body.Message, "portal")
	}
}

func TestDetailWithoutCallerAnswered409(t *testing.T) {
	server := newServer(t, serverOptions{noCaller: true})
	require.Equal(t, http.StatusConflict, get(t, server, "/api/inbox-salvage/pengajuan/1").Code)
	require.Equal(t, http.StatusConflict, get(t, server, "/api/inbox-salvage/klaim/PNC-1").Code)
}

// ── Ekspor ──────────────────────────────────────────────────────────────────────

func readCSV(t *testing.T, recorder *httptest.ResponseRecorder) [][]string {
	t.Helper()
	records, err := csv.NewReader(bytes.NewReader(recorder.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportWritesTheGridAsCSVWithItsColumnTitles(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}), "/api/inbox-salvage/ekspor?daftar=histori")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="inbox-salvage-histori.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

	records := readCSV(t, recorder)
	tab, _ := inboxsalvage.FindTab(inboxsalvage.TabHistori)
	titles := []string{}
	for _, column := range tab.Columns {
		titles = append(titles, column.Title)
	}
	require.Equal(t, titles, records[0])
	require.Greater(t, len(records), 1, "baris daftar ikut terekspor")
}

func TestExportOfAnUnknownTabIsAnsweredAsJSONBeforeAnyByte(t *testing.T) {
	recorder := get(t, newServer(t, serverOptions{}), "/api/inbox-salvage/ekspor?daftar=tidak-ada")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	recorder = get(t, newServer(t, serverOptions{noCaller: true}), "/api/inbox-salvage/ekspor")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// Ekspor membaca halaman demi halaman sampai seluruh baris habis.
func TestExportReadsEveryPage(t *testing.T) {
	repo := &pagedRepo{total: 250}
	recorder := get(t, newServer(t, serverOptions{repo: repo}), "/api/inbox-salvage/ekspor?daftar=ekonomis")
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder)
	require.Len(t, records, 1+250)
	require.Equal(t, []int{1, 2, 3}, repo.pages)
	require.Equal(t, "PNC-249", records[250][0], "kolom pertama Ekonomis adalah No Klaim")
}

// Berkas yang menyentuh batas diberi tanda di baris terakhir, bukan dipotong diam-diam.
func TestExportMarksTheFileWhenItHitsTheLimit(t *testing.T) {
	repo := &pagedRepo{total: 50_150}
	recorder := get(t, newServer(t, serverOptions{repo: repo}), "/api/inbox-salvage/ekspor?daftar=ekonomis")
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder)
	require.Len(t, records, 1+50_000+1)
	last := records[len(records)-1]
	require.Equal(t, "-- Terpotong pada 50000 baris dari 50150 yang cocok. Persempit pencariannya. --", last[0])
	for _, cell := range last[1:] {
		require.Empty(t, cell)
	}
}

// Kegagalan setelah header terkirim hanya dapat dicatat; berkasnya berhenti di tempat.
func TestExportStopsAndLogsWhenALaterPageFails(t *testing.T) {
	var logs bytes.Buffer
	repo := &pagedRepo{total: 250, failOnPage: 2}
	recorder := get(t, newServer(t, serverOptions{repo: repo, logs: &logs}),
		"/api/inbox-salvage/ekspor?daftar=ekonomis")
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder)
	require.Len(t, records, 1+100, "hanya halaman pertama yang sempat tertulis")
	require.Contains(t, logs.String(), "ekspor inbox salvage terputus")
}

// Penulis jawaban yang menolak ditulisi menghentikan ekspor dan meninggalkan jejak.
func TestExportStopsWhenTheResponseCannotBeWritten(t *testing.T) {
	for name, total := range map[string]int{
		"gagal saat didorong": 3,
		"gagal saat menulis":  250,
	} {
		t.Run(name, func(t *testing.T) {
			var logs bytes.Buffer
			repo := &pagedRepo{total: total}
			server := newServer(t, serverOptions{repo: repo, logs: &logs})

			request := httptest.NewRequest(http.MethodGet,
				"/api/inbox-salvage/ekspor?daftar=ekonomis", nil)
			request.Header.Set(portalhttp.HeaderPortal, portalASM)
			writer := &brokenWriter{header: http.Header{}}
			server.ServeHTTP(writer, request)

			require.Equal(t, http.StatusOK, writer.status)
			require.Contains(t, logs.String(), "ekspor inbox salvage terputus")
			require.Equal(t, []int{1}, repo.pages, "halaman berikutnya tidak diminta")
		})
	}
}

// ── Tiruan ──────────────────────────────────────────────────────────────────────

// brokenRepo selalu gagal dengan galat yang bukan milik modul.
type brokenRepo struct{}

func (brokenRepo) List(context.Context, inboxsalvage.Query, inboxsalvage.Pagination) (inboxsalvage.Page, error) {
	return inboxsalvage.Page{}, errBoom
}

func (brokenRepo) Counts(context.Context, inboxsalvage.Caller) ([]inboxsalvage.StatusCount, error) {
	return nil, errBoom
}

func (brokenRepo) Detail(context.Context, string) (inboxsalvage.Detail, error) {
	return inboxsalvage.Detail{}, errBoom
}

func (brokenRepo) DetailByClaim(context.Context, string) (inboxsalvage.Detail, error) {
	return inboxsalvage.Detail{}, errBoom
}

func (brokenRepo) Create(context.Context, inboxsalvage.Form) (string, error) {
	return "", errBoom
}

// pagedRepo menghasilkan baris klaim bernomor urut sebanyak total, halaman demi halaman.
type pagedRepo struct {
	brokenRepo
	total      int
	failOnPage int
	pages      []int
}

func (p *pagedRepo) List(_ context.Context, _ inboxsalvage.Query, page inboxsalvage.Pagination) (inboxsalvage.Page, error) {
	clean := page.Normalize()
	p.pages = append(p.pages, clean.Page)
	if clean.Page == p.failOnPage {
		return inboxsalvage.Page{}, errBoom
	}

	items := []inboxsalvage.Row{}
	for i := clean.Offset(); i < clean.Offset()+clean.Size && i < p.total; i++ {
		number := "PNC-" + strconv.Itoa(i)
		items = append(items, inboxsalvage.Row{
			Reference: number, ClaimNo: number,
			ObjectName: "Objek " + strings.Repeat("x", 40), BusinessName: "Fire", PIC: "PIC",
		})
	}
	return inboxsalvage.Page{Items: items, Total: p.total, Pagination: clean}, nil
}

// brokenWriter menerima header tetapi menolak setiap penulisan badan.
type brokenWriter struct {
	header http.Header
	status int
}

func (b *brokenWriter) Header() http.Header { return b.header }

func (b *brokenWriter) WriteHeader(status int) { b.status = status }

func (b *brokenWriter) Write([]byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return 0, errBoom
}
