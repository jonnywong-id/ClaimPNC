package masterrecoveryhttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// stubService adalah layanan tiruan yang jawabannya dipatok per uji.
type stubService struct {
	err        error
	groups     []masterrecovery.PrincipalGroup
	principals []masterrecovery.Principal
	document   masterrecovery.Document
	saved      masterrecovery.Recovery
	gotFilter  masterrecovery.ListFilter
	gotSave    masterrecovery.Recovery
}

func (s *stubService) NextBatch(context.Context, string) (int64, error) { return 1, s.err }
func (s *stubService) Years(context.Context, string) []string           { return []string{"2025"} }
func (s *stubService) List(_ context.Context, _ string, f masterrecovery.ListFilter) ([]masterrecovery.PrincipalGroup, int, error) {
	s.gotFilter = f
	return s.groups, len(s.groups), s.err
}
func (s *stubService) Document(context.Context, string, string) (masterrecovery.Document, error) {
	return s.document, s.err
}
func (s *stubService) Principals(context.Context, string) ([]masterrecovery.Principal, error) {
	return s.principals, s.err
}
func (s *stubService) LookupPolicy(context.Context, string, string) (masterrecovery.PolicyReference, error) {
	return masterrecovery.PolicyReference{}, s.err
}
func (s *stubService) IssueVirtualAccount(context.Context, string, masterrecovery.VirtualAccountRequest) (masterrecovery.VirtualAccount, error) {
	return masterrecovery.VirtualAccount{}, s.err
}
func (s *stubService) SaveDocument(context.Context, string, masterrecovery.Document) (string, error) {
	return "", s.err
}
func (s *stubService) ReadClaimLine(string, io.Reader) ([]masterrecovery.ClaimLine, []masterrecovery.Violation, error) {
	return nil, nil, s.err
}
func (s *stubService) Save(_ context.Context, _ string, r masterrecovery.Recovery) (masterrecovery.Recovery, error) {
	s.gotSave = r
	return s.saved, s.err
}

// recorder mencatat apa yang ditulis handler lewat kedua penulisnya.
type recorder struct {
	status int
	body   any
	err    error
}

type harness struct {
	handler *Handler
	rec     *recorder
	router  chi.Router
}

func newHarness(t *testing.T, service Service, callerKnown bool, withPortal bool) *harness {
	t.Helper()
	rec := &recorder{}
	h, err := NewHandler(Options{
		Service: service,
		Caller: func(context.Context) (Caller, bool) {
			return Caller{Identity: "PETUGAS1", Name: "Petugas"}, callerKnown
		},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			rec.status, rec.body = status, body
			w.WriteHeader(status)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			rec.err = err
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)

	router := chi.NewRouter()
	if withPortal {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: "ASM"})
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
	}
	router.Get("/form", h.Form)
	router.Get("/", h.List)
	router.Get("/principal", h.Principals)
	router.Get("/polis/{nomor}", h.Policy)
	router.Get("/format", h.Template)
	router.Post("/va", h.IssueVirtualAccount)
	router.Post("/bukti", h.UploadDocument)
	router.Get("/bukti/{id}", h.Document)
	router.Get("/bukti/", h.Document)
	router.Post("/baris", h.ReadClaimLine)
	router.Post("/", h.Save)
	return &harness{handler: h, rec: rec, router: router}
}

func (hs *harness) do(method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, body)
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	hs.router.ServeHTTP(w, r)
	return w
}

func multipartBody(t *testing.T, field, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = io.WriteString(part, content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return &buf, mw.FormDataContentType()
}

var errInternal = errors.New("galat dalam")

func TestNewHandlerRejectsMissingDependencies(t *testing.T) {
	ok := func(http.ResponseWriter, *http.Request, int, any) {}
	okErr := func(http.ResponseWriter, *http.Request, error) {}
	caller := func(context.Context) (Caller, bool) { return Caller{}, true }

	_, err := NewHandler(Options{Caller: caller, WriteResponse: ok, WriteError: okErr})
	require.ErrorContains(t, err, "Service wajib diisi")
	_, err = NewHandler(Options{Service: &stubService{}, WriteResponse: ok, WriteError: okErr})
	require.ErrorContains(t, err, "Caller wajib diisi")
	_, err = NewHandler(Options{Service: &stubService{}, Caller: caller, WriteResponse: ok})
	require.ErrorContains(t, err, "WriteResponse dan WriteError")
}

func TestEveryRouteRejectsMissingPortal(t *testing.T) {
	hs := newHarness(t, &stubService{}, true, false)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/form"}, {http.MethodGet, "/"}, {http.MethodGet, "/principal"},
		{http.MethodGet, "/polis/X"}, {http.MethodGet, "/format"}, {http.MethodPost, "/va"},
		{http.MethodPost, "/bukti"}, {http.MethodGet, "/bukti/1"}, {http.MethodPost, "/baris"},
		{http.MethodPost, "/"},
	} {
		hs.rec.err = nil
		w := hs.do(c.method, c.path, strings.NewReader(""), "")
		require.Equal(t, http.StatusTeapot, w.Code, c.path)
		require.ErrorIs(t, hs.rec.err, portal.ErrNotStated, c.path)
	}
}

func TestServiceErrorsAreMappedOrForwarded(t *testing.T) {
	// Galat yang tidak dikenal modul diteruskan ke penulis galat umum.
	hs := newHarness(t, &stubService{err: errInternal}, true, true)
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/form", ""}, {http.MethodGet, "/", ""}, {http.MethodGet, "/principal", ""},
		{http.MethodGet, "/polis/X", ""}, {http.MethodGet, "/bukti/1", ""},
		{http.MethodPost, "/va", `{"client_id":"a"}`},
	} {
		hs.rec.err = nil
		w := hs.do(c.method, c.path, strings.NewReader(c.body), "application/json")
		require.Equal(t, http.StatusTeapot, w.Code, c.path)
		require.ErrorIs(t, hs.rec.err, errInternal, c.path)
	}

	// Galat modul berstatus 5xx dicatat lalu dijawab dengan kodenya sendiri.
	hs = newHarness(t, &stubService{err: masterrecovery.ErrDocumentNotSaved}, true, true)
	body, ct := multipartBody(t, "berkas", "a.pdf", "isi")
	w := hs.do(http.MethodPost, "/bukti", body, ct)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, CodeDocumentNotSaved, hs.rec.body.(ErrorResponse).Code)
}

func TestListMapsGroupsAndParsesPaging(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	row := masterrecovery.Recovery{
		Batch: 1, PrincipalName: "PT A", InputDate: at, DocumentID: " D1 ", ClaimAmount: 9,
		Attachment: &masterrecovery.AttachmentInfo{ID: "D1", Name: "a.pdf", UploadedBy: "U", UploadedAt: at},
	}
	noDate := masterrecovery.Recovery{Batch: 2, PrincipalName: "PT A", Attachment: &masterrecovery.AttachmentInfo{ID: "D2"}}
	service := &stubService{groups: []masterrecovery.PrincipalGroup{{Name: "PT A", Latest: row, Batch: []masterrecovery.Recovery{row, noDate}}}}
	hs := newHarness(t, service, true, true)

	w := hs.do(http.MethodGet, "/?cari=+PT+&tahun=2026&limit=-3&lewati=abc", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, masterrecovery.ListFilter{PrincipalName: "PT", Year: "2026"}, service.gotFilter)

	list := hs.rec.body.(RecoveryListResponse)
	require.Equal(t, 1, list.Total)
	require.Equal(t, int64(9), list.Principal[0].ClaimAmount)
	first := list.Principal[0].Batch[0]
	require.Equal(t, "2026-01-02T03:04:05Z", first.InputDate)
	require.Equal(t, "D1", first.DocumentID)
	require.Equal(t, &AttachmentDTO{ID: "D1", Name: "a.pdf", UploadedBy: "U", UploadedAt: "2026-01-02T03:04:05Z"}, first.Attachment)
	second := list.Principal[0].Batch[1]
	require.Empty(t, second.InputDate)
	require.Empty(t, second.Attachment.UploadedAt)
}

func TestPrincipalsMapsList(t *testing.T) {
	service := &stubService{principals: []masterrecovery.Principal{{ClientID: "C", Name: "N", VirtualAccountNumber: "V", Email: "e"}}}
	hs := newHarness(t, service, true, true)
	w := hs.do(http.MethodGet, "/principal", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, PrincipalListResponse{
		Principal: []PrincipalDTO{{ClientID: "C", Name: "N", VirtualAccountNumber: "V", Email: "e"}},
		Total:     1, Portal: "ASM",
	}, hs.rec.body)
}

func TestDocumentDefaultsAndEmptyID(t *testing.T) {
	service := &stubService{document: masterrecovery.Document{Content: []byte("isi")}}
	hs := newHarness(t, service, true, true)

	// Tanpa jenis dan nama → octet-stream, unduhan, nama baku.
	w := hs.do(http.MethodGet, "/bukti/1", nil, "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename=bukti-bayar`, w.Header().Get("Content-Disposition"))
	require.Equal(t, "3", w.Header().Get("Content-Length"))
	require.Equal(t, "isi", w.Body.String())

	// Penanda kosong → tidak ditemukan.
	w = hs.do(http.MethodGet, "/bukti/", nil, "")
	require.Equal(t, http.StatusTeapot, w.Code)
	require.ErrorIs(t, hs.rec.err, masterrecovery.ErrDocumentNotFound)
}

// failingWriter menolak setiap penulisan badan.
type failingWriter struct{ *httptest.ResponseRecorder }

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("putus") }

func TestDocumentWriteFailureIsOnlyLogged(t *testing.T) {
	service := &stubService{document: masterrecovery.Document{Name: "a.pdf", MimeType: "application/pdf", Content: []byte("isi")}}
	var logs bytes.Buffer
	h, err := NewHandler(Options{
		Service: service,
		Caller:  func(context.Context) (Caller, bool) { return Caller{}, true },
		Logger:  slog.New(slog.NewTextHandler(&logs, nil)),
		WriteResponse: func(http.ResponseWriter, *http.Request, int, any) {
		},
		WriteError: func(http.ResponseWriter, *http.Request, error) {},
	})
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/bukti/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	w := failingWriter{httptest.NewRecorder()}
	h.Document(w, r.WithContext(ctx))

	require.Equal(t, "inline; filename=a.pdf", w.Header().Get("Content-Disposition"))
	require.Contains(t, logs.String(), "bukti bayar gagal dialirkan")
}

func TestIssueVirtualAccountRejectsMalformedJSON(t *testing.T) {
	hs := newHarness(t, &stubService{}, true, true)
	w := hs.do(http.MethodPost, "/va", strings.NewReader(`{"tidak_dikenal":1}`), "application/json")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, CodeMalformedRequest, hs.rec.body.(ErrorResponse).Code)
}

func TestUploadDocumentFailurePaths(t *testing.T) {
	// Pemanggil tidak dikenal.
	hs := newHarness(t, &stubService{}, false, true)
	body, ct := multipartBody(t, "berkas", "a.pdf", "isi")
	w := hs.do(http.MethodPost, "/bukti", body, ct)
	require.Equal(t, http.StatusTeapot, w.Code)
	require.ErrorContains(t, hs.rec.err, "identitas pemanggil tidak tersedia")

	// Badan bukan multipart.
	hs = newHarness(t, &stubService{}, true, true)
	w = hs.do(http.MethodPost, "/bukti", strings.NewReader("x"), "text/plain")
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	// Bagian "berkas" tidak ada.
	body, ct = multipartBody(t, "lain", "a.pdf", "isi")
	w = hs.do(http.MethodPost, "/bukti", body, ct)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, hs.rec.body.(ErrorResponse).Message, `"berkas"`)
}

func TestReadClaimLineFailurePaths(t *testing.T) {
	hs := newHarness(t, &stubService{}, true, true)
	w := hs.do(http.MethodPost, "/baris", strings.NewReader("x"), "text/plain")
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	body, ct := multipartBody(t, "lain", "a.csv", "x")
	w = hs.do(http.MethodPost, "/baris", body, ct)
	require.Equal(t, http.StatusBadRequest, w.Code)

	hs = newHarness(t, &stubService{err: masterrecovery.ErrClaimLineTooMany}, true, true)
	body, ct = multipartBody(t, "berkas", "a.csv", "x")
	w = hs.do(http.MethodPost, "/baris", body, ct)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	require.Equal(t, CodeClaimLineTooBig, hs.rec.body.(ErrorResponse).Code)
}

func TestSaveFailurePathsAndMapping(t *testing.T) {
	hs := newHarness(t, &stubService{}, false, true)
	w := hs.do(http.MethodPost, "/", strings.NewReader(`{}`), "application/json")
	require.Equal(t, http.StatusTeapot, w.Code)
	require.ErrorContains(t, hs.rec.err, "identitas pemanggil")

	// Badan berisi dua objek JSON ditolak.
	hs = newHarness(t, &stubService{}, true, true)
	w = hs.do(http.MethodPost, "/", strings.NewReader(`{} {}`), "application/json")
	require.Equal(t, http.StatusBadRequest, w.Code)

	hs = newHarness(t, &stubService{err: masterrecovery.ErrBatchTaken}, true, true)
	w = hs.do(http.MethodPost, "/", strings.NewReader(`{}`), "application/json")
	require.Equal(t, http.StatusConflict, w.Code)

	// Baris klaim diteruskan ke layanan dan ikut tampil di jawaban.
	saved := masterrecovery.Recovery{
		Batch: 4, PolicyNo: "P1", BusinessID: "B",
		ClaimLine: []masterrecovery.ClaimLine{{PolicyNo: "P1", ClaimAmount: 7}},
	}
	service := &stubService{saved: saved}
	hs = newHarness(t, service, true, true)
	w = hs.do(http.MethodPost, "/", strings.NewReader(`{"baris_klaim":[{"nomor_polis":"P1","nilai_klaim":7}]}`), "application/json")
	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, []masterrecovery.ClaimLine{{PolicyNo: "P1", ClaimAmount: 7}}, service.gotSave.ClaimLine)
	require.Equal(t, "PETUGAS1", service.gotSave.InputBy)
	response := hs.rec.body.(SaveResponse)
	require.True(t, response.PolicyResolved)
	require.Equal(t, []ClaimLineDTO{{PolicyNo: "P1", ClaimAmount: 7}}, response.Recovery.ClaimLine)
}

func TestMapErrorCoversEveryDomainError(t *testing.T) {
	cases := map[error]struct {
		status int
		code   string
	}{
		masterrecovery.ErrPolicyNotFound:      {http.StatusNotFound, CodePolicyNotFound},
		masterrecovery.ErrIssuerUnconfigured:  {http.StatusInternalServerError, CodeIssuerUnconfigured},
		masterrecovery.ErrIssuerUnreachable:   {http.StatusBadGateway, CodeIssuerUnreachable},
		masterrecovery.ErrIssuerRejected:      {http.StatusBadGateway, CodeIssuerRejected},
		masterrecovery.ErrDocumentNotSaved:    {http.StatusInternalServerError, CodeDocumentNotSaved},
		masterrecovery.ErrDocumentNotFound:    {http.StatusNotFound, CodeDocumentNotFound},
		masterrecovery.ErrDocumentElsewhere:   {http.StatusNotFound, CodeDocumentElsewhere},
		masterrecovery.ErrClaimLineEmpty:      {http.StatusUnprocessableEntity, CodeClaimLineEmpty},
		masterrecovery.ErrClaimLineTooMany:    {http.StatusRequestEntityTooLarge, CodeClaimLineTooBig},
		masterrecovery.ErrClaimLineUnreadable: {http.StatusUnprocessableEntity, CodeClaimLineBad},
		masterrecovery.ErrBatchTaken:          {http.StatusConflict, CodeBatchTaken},
	}
	for err, want := range cases {
		status, body, known := mapError(err)
		require.True(t, known, err.Error())
		require.Equal(t, want.status, status, err.Error())
		require.Equal(t, want.code, body.Code, err.Error())
	}

	status, body, known := mapError(masterrecovery.NewValidationError([]masterrecovery.Violation{{Field: "f", Message: "m"}}))
	require.True(t, known)
	require.Equal(t, http.StatusUnprocessableEntity, status)
	require.Equal(t, []ViolationDTO{{Field: "f", Message: "m"}}, body.Detail)

	_, _, known = mapError(errInternal)
	require.False(t, known)
}

func TestHelpers(t *testing.T) {
	require.Equal(t, 0, atoiOrZero("-1"))
	require.Equal(t, 0, atoiOrZero("x"))
	require.Equal(t, 12, atoiOrZero(" 12 "))

	require.Equal(t, "image/png", mimeTypeOf("image/png; charset=x", "a.bin"))
	require.Equal(t, "application/pdf", mimeTypeOf("application/octet-stream", "A.PDF"))
	require.Equal(t, "application/octet-stream", mimeTypeOf("", "tanpa-ekstensi"))
}
