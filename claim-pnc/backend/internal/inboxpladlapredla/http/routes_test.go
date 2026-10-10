package inboxpladlapredlahttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/repo/memory"
	"claim-pnc/internal/inboxpladlapredla/usecase"
	"claim-pnc/internal/portal"

	inboxhttp "claim-pnc/internal/inboxpladlapredla/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji rute modul Inbox PLA, DLA, Pre DLA lewat router chi sungguhan: portal dibaca dari
// header oleh middleware modul portal, dan identitas pemanggil dibaca dari header uji
// lewat jembatan yang sama bentuknya dengan cmd/claimpnc.

const (
	headerLogin = "X-Uji-Login"
	claimKey    = "ASM-FW-GCNMFW-WORK PNC-1001"
)

type loginKey struct{}

// notifierPalsu merekam surat tanpa mengirimnya.
type notifierPalsu struct {
	letters []inboxpladlapredla.Letter
	err     error
}

func (n *notifierPalsu) SendAdvice(_ context.Context, l inboxpladlapredla.Letter) error {
	if n.err != nil {
		return n.err
	}
	n.letters = append(n.letters, l)
	return nil
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fallbackWriter(w http.ResponseWriter, r *http.Request, _ error) {
	writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
}

type harness struct {
	router http.Handler
	log    *bytes.Buffer
}

// newHarness merakit handler di atas repo pilihan. Repo nil berarti penyimpanan contoh.
func newHarness(
	t *testing.T,
	repo inboxpladlapredla.Repo,
	notifier inboxpladlapredla.Notifier,
) *harness {
	t.Helper()

	if repo == nil {
		store := memory.NewSampleStore()
		store.Now = func() time.Time { return time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC) }
		repo = store
	}

	log := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(log, nil))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxpladlapredla.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		Notifier: notifier,
	})
	require.NoError(t, err)

	writeError := portalhttp.WithPortalError(fallbackWriter, writeJSON)

	handler := inboxhttp.NewHandler(inboxhttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (inboxhttp.Caller, bool) {
			login, ok := ctx.Value(loginKey{}).(string)
			if !ok || login == "" {
				return inboxhttp.Caller{}, false
			}
			return inboxhttp.Caller{Login: login}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: inboxhttp.ErrorWriter(writeError),
	})

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), loginKey{}, r.Header.Get(headerLogin))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	inboxhttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	})

	return &harness{router: router, log: log}
}

func (h *harness) do(
	t *testing.T, method, path, portalAlias, login, body string,
) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var payload io.Reader
	if body != "" {
		payload = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, payload)
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
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
	return h.do(t, http.MethodGet, path, "ASM", "penguji", "")
}

func keyPath(prefix, key, suffix string) string {
	return "/inbox-pla-dla-pre-dla/" + prefix + "/" + url.PathEscape(key) + suffix
}

// Keterangan layar menyerahkan ketiga daftar beserta portal yang dijawab.
func TestMetadataDescribesEveryList(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, "/inbox-pla-dla-pre-dla/daftar")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "ASM", body["portal"])
	require.Equal(t, inboxpladlapredla.DefaultTab, body["daftar_bawaan"])

	tabs := body["daftar"].([]any)
	require.Len(t, tabs, 3)
	first := tabs[0].(map[string]any)
	require.NotEmpty(t, first["kolom"])
	require.NotContains(t, body, "selisih_terencana")
}

// Tanpa header portal, middleware menolak sebelum handler disentuh.
func TestRequestsWithoutAPortalAreRejected(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.do(t, http.MethodGet, "/inbox-pla-dla-pre-dla/daftar", "",
		"penguji", "")
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, portalhttp.CodeNotStated, body["kode"])
}

// Daftar memakai penyaring yang dikirim, dan mengembalikan batas atas dalam bentuk inklusif.
func TestListAppliesTheFilterAndEchoesIt(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, "/inbox-pla-dla-pre-dla?daftar=pla&cari=PNC-1001"+
		"&dari=2026-01-01&sampai=2026-01-31&halaman=1&ukuran=5")
	require.Equal(t, http.StatusOK, response.Code)

	rows := body["baris"].([]any)
	require.Len(t, rows, 1)
	row := rows[0].(map[string]any)
	require.Equal(t, claimKey, row["kunci_klaim"])
	require.Equal(t, "PNC-1001", row["no_klaim"])
	require.Equal(t, "2026-01-10", row["tanggal_advice"])

	require.Equal(t, map[string]any{
		"cari": "PNC-1001", "dari": "2026-01-01", "sampai": "2026-01-31"}, body["penyaring"])
	require.Equal(t, map[string]any{
		"halaman": 1.0, "ukuran": 5.0, "total": 1.0, "total_halaman": 1.0}, body["paginasi"])
	require.Equal(t, "pla", body["daftar"].(map[string]any)["kode"])
}

// Tanggal yang tidak terbaca ditolak bersamaan, bukan diabaikan.
func TestListRejectsUnreadableDates(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, "/inbox-pla-dla-pre-dla?dari=kemarin&sampai=31-01-2026")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeValidationFail, body["kode"])

	details := body["detail"].([]any)
	require.Len(t, details, 2)
	require.Equal(t, inboxpladlapredla.FieldFrom, details[0].(map[string]any)["field"])
	require.Equal(t, inboxpladlapredla.FieldTo, details[1].(map[string]any)["field"])
}

func TestListErrors(t *testing.T) {
	h := newHarness(t, nil, nil)

	// Tanpa identitas.
	response, body := h.do(t, http.MethodGet, "/inbox-pla-dla-pre-dla", "ASM", "", "")
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, inboxhttp.CodeCallerUnknown, body["kode"])

	// Daftar yang tidak dikenal.
	response, body = h.get(t, "/inbox-pla-dla-pre-dla?daftar=entah")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeValidationFail, body["kode"])

	// Portal yang koneksinya belum siap dijawab penulis galat portal.
	response, body = h.do(t, http.MethodGet, "/inbox-pla-dla-pre-dla", "ASI", "penguji", "")
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Equal(t, portalhttp.CodeNotReady, body["kode"])
}

// Grid rincian klaim dibuka lewat kunci yang terkodekan.
func TestDocumentsOpenTheGridOfOneClaim(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, keyPath("klaim", claimKey, "?daftar=pla"))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, claimKey, body["kunci_klaim"])
	require.Len(t, body["baris"].([]any), 2)
	first := body["baris"].([]any)[0].(map[string]any)
	require.Equal(t, "PLA/2026/0001", first["no_advice"])
	require.Equal(t, "Reasuransi Contoh A", first["reasuradur"])
}

func TestDocumentsErrors(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, keyPath("klaim", claimKey, "?daftar=pre-dla"))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, inboxhttp.CodeNoDocumentGrid, body["kode"])

	response, body = h.get(t, keyPath("klaim", "ASM-FW-GCNMFW-WORK PNC-9999", "?daftar=dla"))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, inboxhttp.CodeRowNotFound, body["kode"])

	response, body = h.get(t, "/inbox-pla-dla-pre-dla/klaim/%20?daftar=pla")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, inboxhttp.CodeRowNotFound, body["kode"])

	response, body = h.do(t, http.MethodGet, keyPath("klaim", claimKey, ""), "ASM", "", "")
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, inboxhttp.CodeCallerUnknown, body["kode"])
}

// Panel Print Pre DLA hanya memuat Pre-DLA berlampiran.
func TestPrintOpensThePanel(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, keyPath("cetak", claimKey, ""))
	require.Equal(t, http.StatusOK, response.Code)
	rows := body["baris"].([]any)
	require.Len(t, rows, 1)
	row := rows[0].(map[string]any)
	require.Equal(t, "PRE/2026/0001", row["no_advice"])
	require.Equal(t, "0", row["terkirim"])
	require.Equal(t, "ATT-PNC-1001-0001", row["kunci_lampiran"])
	require.Equal(t, "pre-dla", body["daftar"].(map[string]any)["kode"])

	response, body = h.get(t, "/inbox-pla-dla-pre-dla/cetak/%20")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, inboxhttp.CodeRowNotFound, body["kode"])

	response, body = h.do(t, http.MethodGet, keyPath("cetak", claimKey, ""), "ASM", "", "")
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, inboxhttp.CodeCallerUnknown, body["kode"])

	response, body = h.get(t, keyPath("cetak", "ASM-FW-GCNMFW-WORK PNC-9999", ""))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, inboxhttp.CodeRowNotFound, body["kode"])
}

// Tombol yang belum dibangun menjawab alasannya sendiri, dan penekanannya dicatat.
func TestRejectWriteAnswersTheReasonOfEachButton(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.do(t, http.MethodPost,
		"/inbox-pla-dla-pre-dla/tindakan?tindakan=unggah-penunjang&daftar=pla",
		"ASM", "penguji", "")
	require.Equal(t, http.StatusNotImplemented, response.Code)
	require.Equal(t, inboxhttp.CodeWriteNotAvailable, body["kode"])
	require.Equal(t, inboxpladlapredla.NewNotAvailable("unggah-penunjang").Reason(),
		body["pesan"])
	require.Contains(t, h.log.String(), "tindakan=unggah-penunjang")

	response, body = h.do(t, http.MethodPost, "/inbox-pla-dla-pre-dla/tindakan?tindakan=x",
		"ASM", "penguji", "")
	require.Equal(t, http.StatusNotImplemented, response.Code)
	require.Equal(t, "Tindakan ini belum tersedia di sistem baru. Kerjakan lewat Pega.",
		body["pesan"])
}

// "Kirim Pre DLA" menandai sekali; penekanan kedua dijawab konflik.
func TestSendPreDLAMarksOnce(t *testing.T) {
	h := newHarness(t, nil, nil)
	path := keyPath("cetak", claimKey, "/kirim")

	response, body := h.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"no_advice":"PRE/2026/0001"}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "Pre-DLA ditandai terkirim.", body["pesan"])

	response, body = h.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"no_advice":"PRE/2026/0001"}`)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, inboxhttp.CodeAlreadySent, body["kode"])

	// Badan yang tidak terbaca berujung pada nomor kosong.
	response, body = h.do(t, http.MethodPost, path, "ASM", "penguji", `bukan json`)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeValidationFail, body["kode"])

	response, _ = h.do(t, http.MethodPost, "/inbox-pla-dla-pre-dla/cetak/%20/kirim",
		"ASM", "penguji", `{"no_advice":"PRE/2026/0001"}`)
	require.Equal(t, http.StatusNotFound, response.Code)

	response, _ = h.do(t, http.MethodPost, path, "ASM", "", `{"no_advice":"X"}`)
	require.Equal(t, http.StatusConflict, response.Code)
}

// "SEND" mengirim surat lalu menandai; jumlah penerima dan lampiran dikembalikan.
func TestSendAdviceSendsAndMarks(t *testing.T) {
	notifier := &notifierPalsu{}
	h := newHarness(t, nil, notifier)
	path := keyPath("klaim", claimKey, "/kirim")

	response, body := h.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"daftar":"pla","no_advice":"PLA/2026/0001"}`)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, map[string]any{
		"pesan":    "Surat terkirim dan dokumen ditandai terkirim.",
		"penerima": 1.0, "lampiran": 0.0,
	}, body)
	require.Len(t, notifier.letters, 1)

	response, body = h.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"daftar":"pla","no_advice":"PLA/2026/0001"}`)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, inboxhttp.CodeAdviceSent, body["kode"])

	// Dokumen tanpa alamat reasuradur.
	response, body = h.do(t, http.MethodPost,
		keyPath("klaim", "ASM-FW-GCNMFW-WORK PNC-1005", "/kirim"), "ASM", "penguji",
		`{"daftar":"pla","no_advice":"PLA/2026/0005"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeReinsurerNoEmail, body["kode"])

	response, _ = h.do(t, http.MethodPost, "/inbox-pla-dla-pre-dla/klaim/%20/kirim",
		"ASM", "penguji", `{"daftar":"pla","no_advice":"X"}`)
	require.Equal(t, http.StatusNotFound, response.Code)

	response, _ = h.do(t, http.MethodPost, path, "ASM", "", `{}`)
	require.Equal(t, http.StatusConflict, response.Code)
}

// Surat yang gagal dan pengirim yang belum dipasang dijawab berbeda.
func TestSendAdviceFailuresAreDistinguished(t *testing.T) {
	failing := newHarness(t, nil, &notifierPalsu{err: errors.New("relay menolak")})
	path := keyPath("klaim", claimKey, "/kirim")

	response, body := failing.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"daftar":"pla","no_advice":"PLA/2026/0001"}`)
	require.Equal(t, http.StatusBadGateway, response.Code)
	require.Equal(t, inboxhttp.CodeLetterNotSent, body["kode"])
	require.Contains(t, body["pesan"], "relay menolak")

	unwired := newHarness(t, nil, nil)
	response, body = unwired.do(t, http.MethodPost, path, "ASM", "penguji",
		`{"daftar":"pla","no_advice":"PLA/2026/0001"}`)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Equal(t, inboxhttp.CodeLetterNotSent, body["kode"])
	require.Contains(t, body["pesan"], "pengirim surat belum dipasang")
}

// Ekspor menulis berkas CSV berjudul kolom grid dan bernama menurut daftarnya.
func TestExportWritesTheVisibleList(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, _ := h.get(t, "/inbox-pla-dla-pre-dla/ekspor?daftar=dla")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "text/csv; charset=utf-8", response.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="inbox-dla.csv"`,
		response.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)

	tab, _ := inboxpladlapredla.FindTab("dla")
	titles := make([]string, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		titles = append(titles, column.Title)
	}
	require.Equal(t, titles, records[0])

	// Dua klaim lolos penyaring tab DLA, dan keduanya lolos karena alasan yang berbeda:
	// PNC-1001 ber-`ISKIRIM` KOSONG, PNC-1008 ber-`ISKIRIM = '0'`. Penyaringnya menerima
	// keduanya (`ISKIRIM IS NULL OR ISKIRIM = '0'`).
	require.Len(t, records, 3, "PNC-1001 dan PNC-1008 lolos penyaring tab DLA")
	require.Contains(t, records[1], "PNC-1001")
	require.Contains(t, records[1], "POL-2026-0001")

	// Kolom "Tanggal DLA" PNC-1008 KOSONG, dan itu bukan data yang hilang.
	//
	// Sub-kueri tanggalnya menyaring `ISKIRIM IS NULL` saja — lebih sempit daripada
	// penyaring keanggotaan barisnya. Klaim yang seluruh dokumennya ber-`'0'` karena itu
	// masuk daftar tanpa tanggal. Itu perilaku Pega (`P-5`).
	require.Contains(t, records[2], "PNC-1008")
	require.Equal(t, "", records[2][len(records[2])-1],
		"tanggal advice PNC-1008 kosong karena seluruh dokumennya ber-ISKIRIM='0'")
}

func TestExportRejectsBeforeWritingAnything(t *testing.T) {
	h := newHarness(t, nil, nil)

	response, body := h.get(t, "/inbox-pla-dla-pre-dla/ekspor?dari=salah")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeValidationFail, body["kode"])

	response, body = h.get(t, "/inbox-pla-dla-pre-dla/ekspor?daftar=entah")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Equal(t, inboxhttp.CodeValidationFail, body["kode"])

	response, _ = h.do(t, http.MethodGet, "/inbox-pla-dla-pre-dla/ekspor", "ASM", "", "")
	require.Equal(t, http.StatusConflict, response.Code)
}

// repoBerhalaman mengembalikan halaman penuh berisi baris karangan, lalu gagal setelah
// halaman tertentu.
type repoBerhalaman struct {
	inboxpladlapredla.Repo
	total    int
	failFrom int
	pages    []int
}

func (r *repoBerhalaman) List(
	_ context.Context, _ inboxpladlapredla.Query, page inboxpladlapredla.Pagination,
) (inboxpladlapredla.Page, error) {
	clean := page.Normalize()
	r.pages = append(r.pages, clean.Page)
	if r.failFrom > 0 && clean.Page >= r.failFrom {
		return inboxpladlapredla.Page{}, errors.New("halaman berikutnya gagal")
	}

	items := []inboxpladlapredla.Row{}
	for i := clean.Offset(); i < clean.Offset()+clean.Size && i < r.total; i++ {
		items = append(items, inboxpladlapredla.Row{
			ClaimNo: "PNC-" + strings.Repeat("9", 3), PolicyNo: "POL",
			Insured: strings.Repeat("Nama Panjang ", 3),
		})
	}
	return inboxpladlapredla.Page{Items: items, Total: r.total, Pagination: clean}, nil
}

// Ekspor mengambil halaman demi halaman sampai seluruh baris tertulis.
func TestExportWalksEveryPage(t *testing.T) {
	repo := &repoBerhalaman{total: 150}
	h := newHarness(t, repo, nil)

	response, _ := h.get(t, "/inbox-pla-dla-pre-dla/ekspor")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []int{1, 2}, repo.pages)

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 151)
	require.Equal(t, `attachment; filename="inbox-pla.csv"`,
		response.Header().Get("Content-Disposition"))
}

// Galat setelah header terkirim tidak lagi dapat dijawab JSON: ia dicatat dan berkasnya
// berhenti.
func TestExportFailureAfterTheHeaderIsLogged(t *testing.T) {
	repo := &repoBerhalaman{total: 150, failFrom: 2}
	h := newHarness(t, repo, nil)

	response, _ := h.get(t, "/inbox-pla-dla-pre-dla/ekspor")
	require.Equal(t, http.StatusOK, response.Code)

	records, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 101)
	require.Contains(t, h.log.String(), "ekspor inbox PLA/DLA/Pre DLA terputus")
	require.Contains(t, h.log.String(), "halaman berikutnya gagal")
}

// failingWriter menolak setiap penulisan badan.
type failingWriter struct {
	header http.Header
	status int
}

func (f *failingWriter) Header() http.Header        { return f.header }
func (f *failingWriter) WriteHeader(status int)     { f.status = status }
func (f *failingWriter) Write([]byte) (int, error)  { return 0, errors.New("koneksi putus") }
func (f *failingWriter) result() (int, http.Header) { return f.status, f.header }

// Penulisan yang gagal di tengah unduhan menghentikan ekspor dan meninggalkan jejak.
func TestExportStopsWhenTheConnectionBreaks(t *testing.T) {
	for _, total := range []int{1, 150} {
		repo := &repoBerhalaman{total: total}
		h := newHarness(t, repo, nil)

		request := httptest.NewRequest(http.MethodGet, "/inbox-pla-dla-pre-dla/ekspor", nil)
		request.Header.Set(portalhttp.HeaderPortal, "ASM")
		request.Header.Set(headerLogin, "penguji")

		writer := &failingWriter{header: http.Header{}}
		h.router.ServeHTTP(writer, request)

		_, header := writer.result()
		require.Equal(t, "text/csv; charset=utf-8", header.Get("Content-Type"))
		require.Contains(t, h.log.String(), "koneksi putus")
		require.Equal(t, []int{1}, repo.pages, "ekspor berlanjut setelah koneksi putus")
	}
}

// Jawaban grid rincian MENYATAKAN baris mana yang bertombol "SEND".
//
// Layar tidak menyimpulkannya sendiri dari `terkirim`: syaratnya berbeda antara kedua tab,
// dan perbedaan itu hasil pembacaan section — bukan pilihan tampilan.
//
//	tab PLA  ISKIRIM bukan "1"   -> bertombol
//	tab DLA  ISKIRIM KOSONG      -> bertombol
//
// Syaratnya ditulis sebagai blok di atas, bukan sebagai prosa. Ditulis mengalir, bentuk
// string kosong Pega dua tanda petik berurutan diubah gofmt menjadi tanda kutip tipografis
// — dan syarat yang berubah bentuk tidak lagi menyatakan apa yang dibaca dari section.
func TestDocumentsStateWhichRowsCarryTheSendButton(t *testing.T) {
	h := newHarness(t, nil, nil)

	// PNC-1001 — satu PLA belum terkirim, satu sudah.
	_, body := h.get(t, keyPath("klaim", claimKey, "?daftar=pla"))
	rows := body["baris"].([]any)
	require.Len(t, rows, 2)

	belum := rows[0].(map[string]any)
	require.Equal(t, "", belum["terkirim"])
	require.Equal(t, true, belum["dapat_dikirim"])

	sudah := rows[1].(map[string]any)
	require.Equal(t, "1", sudah["terkirim"])
	require.Equal(t, false, sudah["dapat_dikirim"],
		"dokumen yang sudah terkirim tidak boleh bertombol SEND")
}

// `ISKIRIM = '0'` bertombol di tab PLA tetapi TIDAK di tab DLA.
//
// Itu kejanggalan Pega yang dibawa apa adanya, dan ia hanya terlihat pada nilai yang satu
// ini. Uji ini menembak keduanya lewat HTTP supaya penyeragaman di lapisan mana pun —
// domain, DTO, atau layar — akan tertangkap di sini.
func TestSentFlagZeroIsSendableOnPLAButNotOnDLA(t *testing.T) {
	h := newHarness(t, nil, nil)
	const key = "ASM-FW-GCNMFW-WORK PNC-1008"

	_, body := h.get(t, keyPath("klaim", key, "?daftar=pla"))
	for _, baris := range body["baris"].([]any) {
		row := baris.(map[string]any)
		require.Equal(t, "0", row["terkirim"])
		require.Equal(t, true, row["dapat_dikirim"],
			"PLA memakai `!= '1'`, sehingga '0' tetap bertombol")
	}

	_, body = h.get(t, keyPath("klaim", key, "?daftar=dla"))
	rows := body["baris"].([]any)
	require.NotEmpty(t, rows)
	for _, baris := range rows {
		row := baris.(map[string]any)
		require.Equal(t, "0", row["terkirim"])
		require.Equal(t, false, row["dapat_dikirim"],
			"DLA memakai `== ''`, sehingga '0' TIDAK bertombol")
	}
}
