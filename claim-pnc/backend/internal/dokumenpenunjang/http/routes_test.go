package dokumenpenunjanghttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/portal"

	dokumenpenunjanghttp "claim-pnc/internal/dokumenpenunjang/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// stubService merekam perintah yang diterima dan menjawab dengan nilai yang ditentukan.
type stubService struct {
	uploaded []usecase.UploadCommand
	listed   []string
	document dokumenpenunjang.Document
	list     []dokumenpenunjang.Document
	err      error
}

func (s *stubService) Upload(_ context.Context, cmd usecase.UploadCommand) (dokumenpenunjang.Document, error) {
	s.uploaded = append(s.uploaded, cmd)
	return s.document, s.err
}

func (s *stubService) List(_ context.Context, alias, nomor string) ([]dokumenpenunjang.Document, error) {
	s.listed = append(s.listed, alias+"/"+nomor)
	return s.list, s.err
}

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type fixture struct {
	router  http.Handler
	handler *dokumenpenunjanghttp.Handler
	logs    *bytes.Buffer
}

func newFixture(t *testing.T, service *stubService, login string, fallback bool) fixture {
	t.Helper()
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))

	options := dokumenpenunjanghttp.Options{
		Service: service,
		GetCaller: func(context.Context) (dokumenpenunjanghttp.Caller, bool) {
			return dokumenpenunjanghttp.Caller{Login: login}, login != ""
		},
		Logger:    logger,
		WriteJSON: writeJSON,
		Location:  time.UTC,
	}
	if fallback {
		options.FallbackErrorWriter = func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
		}
	}
	handler := dokumenpenunjanghttp.NewHandler(options)

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		dokumenpenunjanghttp.Mount(api, handler, portalhttp.ActivePortalDeps{
			Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
			ReadyAliases: func() []string { return []string{"ASM"} },
			Logger:       logger,
			WriteError: func(w http.ResponseWriter, r *http.Request, err error) {
				writeJSON(w, r, http.StatusBadRequest, map[string]string{"kode": "portal"})
			},
		})
	})
	return fixture{router: router, handler: handler, logs: logs}
}

func multipartBody(t *testing.T, field, name string, content []byte, docType string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if field != "" {
		part, err := writer.CreateFormFile(field, name)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	if docType != "" {
		require.NoError(t, writer.WriteField("jenis_dokumen", docType))
	}
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func (f fixture) do(t *testing.T, request *http.Request) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	content := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &content)
	return recorder, content
}

const route = "/api/klaim/PNC-1/dokumen-penunjang"

func TestListFormatsTimesAndExpiry(t *testing.T) {
	uploaded := time.Date(2026, 9, 1, 3, 4, 0, 0, time.UTC)
	expired := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	service := &stubService{list: []dokumenpenunjang.Document{
		{ImageID: "A", FileName: "a.pdf", DocumentType: "KTP", URL: "http://u",
			UploadedAt: &uploaded, ExpiresAt: &expired},
		{ImageID: "B"},
	}}
	f := newFixture(t, service, "BUDI", true)

	recorder, body := f.do(t, httptest.NewRequest(http.MethodGet, route+"/", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []string{"ASM/PNC-1"}, service.listed)

	data := body["data"].([]any)
	require.Len(t, data, 2)
	first := data[0].(map[string]any)
	require.Equal(t, "A", first["id"])
	require.Equal(t, "a.pdf", first["nama_berkas"])
	require.Equal(t, "KTP", first["jenis_dokumen"])
	require.Equal(t, "01/09/2026 03:04", first["tanggal_unggah"])
	require.Equal(t, "01/01/2020 00:00", first["berlaku_sampai"])
	require.Equal(t, true, first["kedaluwarsa"])
	second := data[1].(map[string]any)
	require.Equal(t, "", second["tanggal_unggah"])
	require.Equal(t, false, second["kedaluwarsa"])
}

func TestUploadPassesTheFileAndCaller(t *testing.T) {
	service := &stubService{document: dokumenpenunjang.Document{ImageID: "X", FileName: "foto.png"}}
	f := newFixture(t, service, " BUDI ", true)

	body, contentType := multipartBody(t, "berkas", `C:\Users\budi\foto.png`, []byte("isi"), " KTP ")
	request := httptest.NewRequest(http.MethodPost, route+"/", body)
	request.Header.Set("Content-Type", contentType)

	recorder, content := f.do(t, request)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "X", content["data"].(map[string]any)["id"])

	require.Len(t, service.uploaded, 1)
	cmd := service.uploaded[0]
	require.Equal(t, "ASM", cmd.PortalAlias)
	require.Equal(t, dokumenpenunjang.UploadRequest{
		ClaimNumber: "PNC-1", FileName: "foto.png", DocumentType: "KTP",
		Content: []byte("isi"), By: "BUDI",
	}, cmd.Request)
}

func TestUploadRefusals(t *testing.T) {
	// Tanpa pemanggil.
	f := newFixture(t, &stubService{}, "", true)
	body, contentType := multipartBody(t, "berkas", "a.pdf", []byte("x"), "")
	request := httptest.NewRequest(http.MethodPost, route+"/", body)
	request.Header.Set("Content-Type", contentType)
	recorder, _ := f.do(t, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	// Tanpa bagian "berkas".
	f = newFixture(t, &stubService{}, "BUDI", true)
	body, contentType = multipartBody(t, "", "", nil, "KTP")
	request = httptest.NewRequest(http.MethodPost, route+"/", body)
	request.Header.Set("Content-Type", contentType)
	recorder, content := f.do(t, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, dokumenpenunjanghttp.CodeBadRequest, content["kode"])

	// Badan melampaui batas.
	big := bytes.Repeat([]byte("a"), dokumenpenunjang.BatasUkuranBerkas+(2<<20))
	body, contentType = multipartBody(t, "berkas", "besar.pdf", big, "")
	request = httptest.NewRequest(http.MethodPost, route+"/", body)
	request.Header.Set("Content-Type", contentType)
	recorder, content = f.do(t, request)
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Equal(t, dokumenpenunjanghttp.CodeTooLarge, content["kode"])
}

func TestHandlersRequireAClaimNumberAndAPortal(t *testing.T) {
	f := newFixture(t, &stubService{}, "BUDI", true)

	for _, handler := range []http.HandlerFunc{f.handler.List, f.handler.Upload} {
		// Tanpa nomor klaim.
		recorder := httptest.NewRecorder()
		handler(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)

		// Dengan nomor klaim tetapi tanpa portal aktif — galatnya bukan milik modul,
		// sehingga diteruskan ke penulis cadangan.
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("nomor", "PNC-1")
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
		recorder = httptest.NewRecorder()
		handler(recorder, request)
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}
}

func TestEveryDomainErrorHasItsOwnStatus(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{dokumenpenunjang.ErrBerkasTerlaluBesar, http.StatusRequestEntityTooLarge, dokumenpenunjanghttp.CodeTooLarge},
		{dokumenpenunjang.ErrBerkasKosong, http.StatusBadRequest, dokumenpenunjanghttp.CodeBadRequest},
		{dokumenpenunjang.ErrNamaBerkasKosong, http.StatusBadRequest, dokumenpenunjanghttp.CodeBadRequest},
		{dokumenpenunjang.ErrPengunggahKosong, http.StatusUnauthorized, dokumenpenunjanghttp.CodeBadRequest},
		{dokumenpenunjang.ErrFolderAplikasiTidakAda, http.StatusInternalServerError, dokumenpenunjanghttp.CodeInternalError},
		{dokumenpenunjang.ErrLinkTakTerjangkau, http.StatusBadGateway, dokumenpenunjanghttp.CodeUpstream},
		{dokumenpenunjang.ErrKonversiGagal, http.StatusBadGateway, dokumenpenunjanghttp.CodeUpstream},
		{dokumenpenunjang.ErrUnggahGagal, http.StatusBadGateway, dokumenpenunjanghttp.CodeUpstream},
		{fmt.Errorf("bungkus: %w", dokumenpenunjang.ErrMetadataGagal), http.StatusInternalServerError, dokumenpenunjanghttp.CodeHalfDone},
		{dokumenpenunjang.ErrTidakDitemukan, http.StatusNotFound, dokumenpenunjanghttp.CodeNotFound},
		{errors.New("tak dikenal"), http.StatusTeapot, "cadangan"},
	}
	for _, tc := range cases {
		f := newFixture(t, &stubService{err: tc.err}, "BUDI", true)
		recorder, content := f.do(t, httptest.NewRequest(http.MethodGet, route+"/", nil))
		require.Equal(t, tc.status, recorder.Code, tc.err.Error())
		require.Equal(t, tc.code, content["kode"], tc.err.Error())
		require.NotEmpty(t, content["kode"])
	}
}

func TestAnUnknownErrorWithoutFallbackIsLoggedAs500(t *testing.T) {
	f := newFixture(t, &stubService{err: errors.New("tak dikenal")}, "BUDI", false)
	recorder, content := f.do(t, httptest.NewRequest(http.MethodGet, route+"/", nil))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, dokumenpenunjanghttp.CodeInternalError, content["kode"])
	require.Contains(t, f.logs.String(), "permintaan gagal")
}

func TestUploadServiceFailureIsWritten(t *testing.T) {
	f := newFixture(t, &stubService{err: dokumenpenunjang.ErrUnggahGagal}, "BUDI", true)
	body, contentType := multipartBody(t, "berkas", "a.pdf", []byte("x"), "")
	request := httptest.NewRequest(http.MethodPost, route+"/", body)
	request.Header.Set("Content-Type", contentType)
	recorder, _ := f.do(t, request)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
}

func TestTheDefaultLocationIsJakarta(t *testing.T) {
	uploaded := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	service := &stubService{list: []dokumenpenunjang.Document{{ImageID: "A", UploadedAt: &uploaded}}}
	handler := dokumenpenunjanghttp.NewHandler(dokumenpenunjanghttp.Options{
		Service: service, WriteJSON: writeJSON,
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("nomor", "PNC-1")
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routeContext)
	ctx = portalhttp.WithActivePortal(ctx, portal.Portal{Alias: "ASM"})
	recorder := httptest.NewRecorder()
	handler.List(recorder, request.WithContext(ctx))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, strings.Contains(recorder.Body.String(), "01/09/2026 10:00"), recorder.Body.String())
}
