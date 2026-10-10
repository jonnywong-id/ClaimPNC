package inboxpladlahttp_test

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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
	"claim-pnc/internal/inboxpladla/usecase"
	"claim-pnc/internal/portal"

	inboxpladlahttp "claim-pnc/internal/inboxpladla/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// headerLogin adalah header buatan uji yang membawa login pemanggil.
const headerLogin = "X-Uji-Login"

type callerKey struct{}

// errBoom adalah galat penyimpanan buatan.
var errBoom = errors.New("penyimpanan rusak")

// flakyRepo menggagalkan pemanggilan List ke-N; N = 0 berarti tidak pernah gagal.
type flakyRepo struct {
	inboxpladla.Repo
	failListAt int
	calls      *int
}

func (f flakyRepo) List(
	ctx context.Context, q inboxpladla.Query, p inboxpladla.Pagination,
) (inboxpladla.Page, error) {
	*f.calls++
	if f.failListAt != 0 && *f.calls == f.failListAt {
		return inboxpladla.Page{}, errBoom
	}
	return f.Repo.List(ctx, q, p)
}

type fixture struct {
	router  http.Handler
	handler *inboxpladlahttp.Handler
	logs    *bytes.Buffer
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// fallbackWriter menjawab galat yang bukan milik modul dengan kode tersendiri.
func fallbackWriter(w http.ResponseWriter, r *http.Request, err error) {
	portalhttp.WithPortalError(func(w http.ResponseWriter, r *http.Request, err error) {
		writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
	}, writeJSON)(w, r, err)
}

func newFixture(t *testing.T, repo inboxpladla.Repo, withFallback bool) fixture {
	t.Helper()

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxpladla.Repo, error) {
			if alias == "ASM" {
				return repo, nil
			}
			return nil, portal.ErrNotReady
		},
		Logger: logger,
	})
	require.NoError(t, err)

	options := inboxpladlahttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (inboxpladlahttp.Caller, bool) {
			login, ok := ctx.Value(callerKey{}).(string)
			if !ok {
				return inboxpladlahttp.Caller{}, false
			}
			return inboxpladlahttp.Caller{Login: login, Name: "Mitra Contoh"}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
	}
	if withFallback {
		options.FallbackErrorWriter = fallbackWriter
	}
	handler := inboxpladlahttp.NewHandler(options)

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if login := r.Header.Get(headerLogin); login != "" {
				r = r.WithContext(context.WithValue(r.Context(), callerKey{}, login))
			}
			next.ServeHTTP(w, r)
		})
	})
	router.Route("/api", func(api chi.Router) {
		inboxpladlahttp.Mount(api, handler, portalhttp.ActivePortalDeps{
			Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
			ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
			Logger:       logger,
			WriteError:   fallbackWriter,
		})
	})

	return fixture{router: router, handler: handler, logs: logs}
}

func sampleFixture(t *testing.T) fixture {
	return newFixture(t, memory.NewSampleStore(), true)
}

func (f fixture) do(t *testing.T, method, path, login, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	if login != "" {
		request.Header.Set(headerLogin, login)
	}
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &out), recorder.Body.String())
	return out
}

const partner = memory.SampleReinsurerLogin

var claimPath = "/api/inbox-pla-dla/klaim/" + url.PathEscape("ASM-FW-GCNMFW-WORK PNC-2001")

func TestMetadataDescribesTheScreen(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet, "/api/inbox-pla-dla/daftar", "", "")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxpladlahttp.MetadataResponse](t, recorder)
	require.Equal(t, inboxpladla.DefaultTab, body.DefaultTab)
	require.Equal(t, "ASM", body.Portal)
	require.Len(t, body.Tabs, len(inboxpladla.Tabs()))
	require.Equal(t, inboxpladla.TabPLA, body.Tabs[0].Code)
	require.NotEmpty(t, body.Tabs[0].Columns)
}

func TestHandlersWithoutAPortalAnswerNotStated(t *testing.T) {
	f := sampleFixture(t)
	handlers := map[string]http.HandlerFunc{
		"metadata": f.handler.Metadata,
		"list":     f.handler.List,
		"reject":   f.handler.RejectWrite,
	}
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, portalhttp.CodeNotStated,
				decode[map[string]any](t, recorder)["kode"])
		})
	}
}

func TestListReturnsTheRowsOfTheTab(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet,
		"/api/inbox-pla-dla?daftar=pla&halaman=1&ukuran=5", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxpladlahttp.ListResponse](t, recorder)
	require.Equal(t, "pla", body.Tab.Code)
	require.Equal(t, "ASM", body.Portal)
	require.Equal(t, 1, body.Paging.Page)
	require.Equal(t, 5, body.Paging.Size)
	require.NotEmpty(t, body.Rows)
	require.Equal(t, "PNC-2001", body.Rows[0].ClaimNo)
	require.Equal(t, "PLA/2026/2001-R1", body.Rows[0].AdviceNo)
}

func TestListTreatsAnUnreadablePageAsTheDefault(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet,
		"/api/inbox-pla-dla?daftar=pla&halaman=abc&ukuran=-3", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxpladlahttp.ListResponse](t, recorder)
	require.Equal(t, 1, body.Paging.Page)
	require.Equal(t, inboxpladla.DefaultPageSize, body.Paging.Size)
}

func TestListErrorsAreMappedToTheirCodes(t *testing.T) {
	f := sampleFixture(t)

	cases := []struct {
		name   string
		path   string
		login  string
		status int
		code   string
	}{
		{"tanpa identitas", "/api/inbox-pla-dla?daftar=pla", "", http.StatusConflict,
			inboxpladlahttp.CodeCallerUnknown},
		{"bukan reasuradur", "/api/inbox-pla-dla?daftar=pla", "PEGAWAI", http.StatusForbidden,
			inboxpladlahttp.CodeNotAReinsurer},
		{"tab tidak dikenal", "/api/inbox-pla-dla?daftar=entah", partner,
			http.StatusUnprocessableEntity, inboxpladlahttp.CodeValidationFail},
		// "xol" BUKAN lagi tab. Ia ditolak sebagai tab tak dikenal, sama seperti
		// kode karangan mana pun 2014 tampilan itu kode mati di Pega dan tidak dibawa.
		{"tampilan xol tidak ada lagi", "/api/inbox-pla-dla?daftar=xol", partner,
			http.StatusUnprocessableEntity, inboxpladlahttp.CodeValidationFail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := f.do(t, http.MethodGet, tc.path, tc.login, "")
			require.Equal(t, tc.status, recorder.Code)
			body := decode[inboxpladlahttp.ErrorResponse](t, recorder)
			require.Equal(t, tc.code, body.Code)
			require.NotEmpty(t, body.Message)
		})
	}

	// Rincian pelanggaran ikut dikirim pada galat validasi.
	recorder := f.do(t, http.MethodGet, "/api/inbox-pla-dla?daftar=entah", partner, "")
	require.NotEmpty(t, decode[inboxpladlahttp.ErrorResponse](t, recorder).Details)
}

func TestCountsAndXOLAnswerWithTheirRows(t *testing.T) {
	f := sampleFixture(t)

	recorder := f.do(t, http.MethodGet, "/api/inbox-pla-dla/ringkas?daftar=pla", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	counts := decode[inboxpladlahttp.CountsResponse](t, recorder)
	require.NotEmpty(t, counts.Rows)
	require.Positive(t, counts.Rows[0].Total)

	// Rute /xol dicabut bersama tampilannya.
	recorder = f.do(t, http.MethodGet, "/api/inbox-pla-dla/xol", partner, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAnUnrecognizedErrorGoesToTheFallbackOrBecomes500(t *testing.T) {
	calls := 0
	repo := flakyRepo{Repo: memory.NewSampleStore(), calls: &calls, failListAt: 1}

	withFallback := newFixture(t, repo, true)
	recorder := withFallback.do(t, http.MethodGet, "/api/inbox-pla-dla?daftar=pla", partner, "")
	require.Equal(t, http.StatusTeapot, recorder.Code)

	calls = 0
	without := newFixture(t, repo, false)
	recorder = without.do(t, http.MethodGet, "/api/inbox-pla-dla?daftar=pla", partner, "")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeInternalError,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)
	require.Contains(t, without.logs.String(), "permintaan gagal")
}

func TestAPortalThatIsNotReadyReachesThePortalErrorWriter(t *testing.T) {
	f := sampleFixture(t)
	request := httptest.NewRequest(http.MethodGet, "/api/inbox-pla-dla?daftar=pla", nil)
	request.Header.Set(portalhttp.HeaderPortal, "ASI")
	request.Header.Set(headerLogin, partner)
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestRejectWriteNamesThePressedButton(t *testing.T) {
	f := sampleFixture(t)
	recorder := f.do(t, http.MethodPost,
		"/api/inbox-pla-dla/tindakan?tindakan=unduh-semua-dla", partner, "")
	require.Equal(t, http.StatusNotImplemented, recorder.Code)

	body := decode[inboxpladlahttp.ErrorResponse](t, recorder)
	require.Equal(t, inboxpladlahttp.CodeWriteNotAvailable, body.Code)
	require.NotEmpty(t, body.Message)
	require.Contains(t, f.logs.String(), "tombol yang belum dibangun")
}

func TestDetailServesTheWholeScreen(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet, claimPath, partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxpladlahttp.DetailResponse](t, recorder)
	require.Equal(t, "PNC-2001", body.Claim.ClaimNo)
	require.Equal(t, "ASM", body.Portal)
	require.NotEmpty(t, body.PLA)
	require.Equal(t, "pla", strings.ToLower(body.PLA[0].Kind))
	require.NotEmpty(t, body.Conversations)
	require.NotEmpty(t, body.ColumnsPLA)
	require.NotEmpty(t, body.ColumnsDLA)
	require.NotEmpty(t, body.ColumnsDocument)
	require.NotEmpty(t, body.ColumnsConversation)
}

func TestDetailOfAnotherPartnersClaimIsNotFound(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet,
		"/api/inbox-pla-dla/klaim/"+url.PathEscape("ASM-FW-GCNMFW-WORK PNC-2008"), partner, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeClaimNotFound,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = sampleFixture(t).do(t, http.MethodGet, claimPath, "", "")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

func TestDocumentsAreListedForOneAdvice(t *testing.T) {
	f := sampleFixture(t)

	recorder := f.do(t, http.MethodGet,
		claimPath+"/dokumen?jenis=pla&nomor="+url.QueryEscape("PLA/2026/2001-R1"), partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	body := decode[inboxpladlahttp.DocumentsResponse](t, recorder)
	require.Len(t, body.Rows, 1)
	require.Equal(t, "DOK-01", body.Rows[0].ID)
	require.Equal(t, "laporan-kerugian.pdf", body.Rows[0].Name)
	require.NotEmpty(t, body.Columns)

	recorder = f.do(t, http.MethodGet, claimPath+"/dokumen?jenis=xol", partner, "")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeAdviceKindUnknown,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = f.do(t, http.MethodGet, claimPath+"/dokumen?jenis=pla", "", "")
	require.Equal(t, http.StatusConflict, recorder.Code)

	recorder = f.do(t, http.MethodGet,
		"/api/inbox-pla-dla/klaim/"+url.PathEscape("ASM-FW-GCNMFW-WORK PNC-2008")+
			"/dokumen?jenis=dla", partner, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDocumentContentIsWrittenAsAFile(t *testing.T) {
	f := sampleFixture(t)

	recorder := f.do(t, http.MethodGet, claimPath+"/dokumen/DOK-01", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename=laporan-kerugian.pdf`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "%PDF-1.4 contoh laporan kerugian", recorder.Body.String())

	recorder = f.do(t, http.MethodGet, claimPath+"/dokumen/DOK-03", partner, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeDocumentNotFound,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = f.do(t, http.MethodGet, claimPath+"/dokumen/DOK-01", "", "")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// documentStore membentuk penyimpanan contoh dengan satu dokumen tambahan.
func documentStore(name, mimeType string) *memory.Store {
	store := memory.NewSampleStore()
	documents := []memory.Document{{
		ID: "DOK-01", ClaimKey: "ASM-FW-GCNMFW-WORK PNC-2001",
		AdviceNo: "PLA/2026/2001-R1", Kind: "PLA", Login: partner,
		Category: "Dokumen Klaim", SubCategory: "Lain",
		Name: name, MimeType: mimeType, Content: []byte("isi"),
	}}
	store.SeedDocuments(documents)
	return store
}

func TestTheContentTypeAndFilenameFallBackSensibly(t *testing.T) {
	cases := []struct {
		name, mime, wantType, wantDisposition string
	}{
		{"foto.png", "", "image/png", "attachment; filename=foto.png"},
		{"tanpa-akhiran", "", "application/octet-stream", "attachment; filename=tanpa-akhiran"},
		{"  ", "text/plain", "text/plain", "attachment; filename=dokumen"},
		{"laporan \"akhir\".pdf", "application/pdf", "application/pdf",
			`attachment; filename="laporan \"akhir\".pdf"`},
		{"berkas\x01.pdf", "application/pdf", "application/pdf",
			"attachment; filename*=utf-8''berkas%01.pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, documentStore(tc.name, tc.mime), true)
			recorder := f.do(t, http.MethodGet, claimPath+"/dokumen/DOK-01", partner, "")
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, tc.wantType, recorder.Header().Get("Content-Type"))
			require.Equal(t, tc.wantDisposition, recorder.Header().Get("Content-Disposition"))
		})
	}
}

func TestReplyIsStoredAndValidated(t *testing.T) {
	f := sampleFixture(t)
	replyPath := claimPath + "/komunikasi/balas"

	recorder := f.do(t, http.MethodPost, replyPath, partner, "{bukan json")
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeValidationFail,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = f.do(t, http.MethodPost, replyPath, partner,
		`{"percakapan":"KOM-01","balasan":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	recorder = f.do(t, http.MethodPost, replyPath, partner,
		`{"percakapan":"KOM-04","balasan":"ok"}`)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeConversationNotFound,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = f.do(t, http.MethodPost, replyPath, partner,
		`{"percakapan":"KOM-01","balasan":"Disetujui."}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	body := decode[inboxpladlahttp.ReplyResponse](t, recorder)
	require.Equal(t, "ASM", body.Portal)
	require.NotEmpty(t, body.Message)

	recorder = f.do(t, http.MethodPost, replyPath, partner,
		`{"percakapan":"KOM-01","balasan":"Lagi."}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeConversationAnswered,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = f.do(t, http.MethodPost, replyPath, "", `{}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
}

func readCSV(t *testing.T, recorder *httptest.ResponseRecorder) [][]string {
	t.Helper()
	records, err := csv.NewReader(bytes.NewReader(recorder.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportWritesTheListAsCSV(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet,
		"/api/inbox-pla-dla/ekspor?daftar=dla", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="inbox-pla-dla-dla.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))

	records := readCSV(t, recorder)
	tab, _ := inboxpladla.FindTab(inboxpladla.TabPLADLA)
	require.Len(t, records[0], len(tab.Columns))
	require.Equal(t, tab.Columns[0].Title, records[0][0])
	require.Greater(t, len(records), 1)

	// Kolom status memuat LABEL — PNC-2005 diganti 1139 "Pending Close Claim".
	statusIndex := -1
	for i, column := range tab.Columns {
		if column.Key == inboxpladla.FieldStatus {
			statusIndex = i
		}
	}
	require.GreaterOrEqual(t, statusIndex, 0)
	found := false
	for _, record := range records[1:] {
		if record[statusIndex] == "Pending Close Claim" {
			found = true
		}
	}
	require.True(t, found, "label status ditulis ke berkas")
}

func TestExportRefusesBeforeWritingTheHeader(t *testing.T) {
	recorder := sampleFixture(t).do(t, http.MethodGet,
		"/api/inbox-pla-dla/ekspor?daftar=pla", "PEGAWAI", "")
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, inboxpladlahttp.CodeNotAReinsurer,
		decode[inboxpladlahttp.ErrorResponse](t, recorder).Code)

	recorder = sampleFixture(t).do(t, http.MethodGet, "/api/inbox-pla-dla/ekspor", "", "")
	require.Equal(t, http.StatusConflict, recorder.Code)
}

// bigStore membentuk penyimpanan berisi banyak klaim pada daftar PLA.
func bigStore(total int) *memory.Store {
	claims := make([]memory.Claim, 0, total)
	advices := make([]memory.Advice, 0, total)
	sentOn := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	for i := 0; i < total; i++ {
		no := fmt.Sprintf("PNC-%05d", i)
		key := "ASM-FW-GCNMFW-WORK " + no
		claims = append(claims, memory.Claim{
			Key: key, No: no, PolicyNo: "POL-" + no, Insured: "PT Uji",
			BusinessName: "FIRE", GroupPanel: "006",
			RegisterDate: sentOn, LossDate: sentOn,
			PICTeknik: "BUDI", CloseNote: "catatan",
			WorkStatus: "Open", StatusCode: "9999", HasWorkRow: true,
		})
		advices = append(advices, memory.Advice{
			ClaimKey: key, Kind: "pla", No: "PLA/" + no, ReinsCode: "R100",
			Sent: "1", SentDate: sentOn, Email: "reas@contoh.example",
		})
	}
	store := memory.NewStore()
	store.Seed(claims, advices,
		[]memory.Reinsurer{{Code: "R100", Login: partner}}, nil)
	return store
}

func TestExportFetchesEveryPage(t *testing.T) {
	f := newFixture(t, bigStore(150), true)
	recorder := f.do(t, http.MethodGet, "/api/inbox-pla-dla/ekspor?daftar=pla", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	records := readCSV(t, recorder)
	require.Len(t, records, 151, "judul ditambah 150 baris dari dua potong")

	// Status tanpa label jatuh ke kodenya; kolom lain terisi apa adanya.
	tab, _ := inboxpladla.FindTab(inboxpladla.TabPLA)
	row := map[string]string{}
	for i, column := range tab.Columns {
		row[column.Key] = records[1][i]
	}
	require.Equal(t, "PNC-00000", row[inboxpladla.FieldClaimNo])
	require.Equal(t, "POL-PNC-00000", row[inboxpladla.FieldPolicyNo])
	require.Equal(t, "PT Uji", row[inboxpladla.FieldInsured])
	require.Equal(t, "FIRE", row[inboxpladla.FieldBusinessName])
	require.Equal(t, "2026-01-15", row[inboxpladla.FieldRegisterDate])
	require.Equal(t, "2026-01-15", row[inboxpladla.FieldLossDate])
	require.Equal(t, "BUDI", row[inboxpladla.FieldPICTeknik])
	require.Equal(t, "9999", row[inboxpladla.FieldStatus])
	require.Equal(t, "PLA/PNC-00000", row[inboxpladla.FieldAdviceNo])
}

func TestExportOfAnEmptyListWritesOnlyTheHeader(t *testing.T) {
	store := memory.NewStore()
	store.Seed(nil, nil, []memory.Reinsurer{{Code: "R100", Login: partner}}, nil)

	recorder := newFixture(t, store, true).do(t, http.MethodGet,
		"/api/inbox-pla-dla/ekspor?daftar=close", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, `attachment; filename="inbox-pla-dla-close.csv"`,
		recorder.Header().Get("Content-Disposition"))
	require.Len(t, readCSV(t, recorder), 1)
}

func TestAFailureOnALaterPageIsLoggedAndTheFileStops(t *testing.T) {
	calls := 0
	repo := flakyRepo{Repo: bigStore(150), failListAt: 2, calls: &calls}
	f := newFixture(t, repo, true)

	recorder := f.do(t, http.MethodGet, "/api/inbox-pla-dla/ekspor?daftar=pla", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, readCSV(t, recorder), 101, "hanya potongan pertama yang sempat tertulis")
	require.Contains(t, f.logs.String(), "ekspor inbox PLA/DLA reasuradur terputus")
}

// brokenWriter menolak setiap penulisan badan jawaban.
type brokenWriter struct {
	header http.Header
	status int
}

func (b *brokenWriter) Header() http.Header       { return b.header }
func (b *brokenWriter) WriteHeader(status int)    { b.status = status }
func (b *brokenWriter) Write([]byte) (int, error) { return 0, errors.New("sambungan putus") }

func TestAWriteFailureDuringExportIsLogged(t *testing.T) {
	for _, total := range []int{1, 150} {
		f := newFixture(t, bigStore(total), true)
		request := httptest.NewRequest(http.MethodGet, "/api/inbox-pla-dla/ekspor?daftar=pla", nil)
		request.Header.Set(portalhttp.HeaderPortal, "ASM")
		request.Header.Set(headerLogin, partner)

		writer := &brokenWriter{header: http.Header{}}
		f.router.ServeHTTP(writer, request)

		require.Equal(t, "text/csv; charset=utf-8", writer.header.Get("Content-Type"))
		require.Contains(t, f.logs.String(), "ekspor inbox PLA/DLA reasuradur terputus")
	}
}

// Tabel "Status / Jumlah" menjawab KEENAM daftar sekaligus, bukan daftar yang terbuka.
//
// Inilah tabel yang digambar Pega, dan ia satu-satunya navigasi layar lama — karena itu
// barisnya harus lengkap, bernama, dan membawa kode yang dapat dipakai berpindah daftar.
func TestListCountsAnswerEverySixListsWithTheirCodes(t *testing.T) {
	f := sampleFixture(t)

	recorder := f.do(t, http.MethodGet, "/api/inbox-pla-dla/ringkas-daftar", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[inboxpladlahttp.ListCountsResponse](t, recorder)
	require.Len(t, body.Rows, 6)

	for _, row := range body.Rows {
		require.NotEmpty(t, row.Code, "kodenya dipakai layar untuk berpindah daftar")
		require.NotEmpty(t, row.Name, "namanya digambar di kolom Status")
		require.GreaterOrEqual(t, row.Total, 0)
	}

	// Kata kunci pencarian ikut disaring, dan barisnya TETAP enam.
	recorder = f.do(t, http.MethodGet,
		"/api/inbox-pla-dla/ringkas-daftar?cari=TIDAK-ADA", partner, "")
	require.Equal(t, http.StatusOK, recorder.Code)

	kosong := decode[inboxpladlahttp.ListCountsResponse](t, recorder)
	require.Len(t, kosong.Rows, 6, "daftar yang kosong tetap disebut")
	for _, row := range kosong.Rows {
		require.Zero(t, row.Total)
	}

	// Pemanggil tanpa portal ditolak, bukan dijawab enam angka nol.
	recorder = f.do(t, http.MethodGet, "/api/inbox-pla-dla/ringkas-daftar", "", "")
	require.Equal(t, http.StatusConflict, recorder.Code)
}
