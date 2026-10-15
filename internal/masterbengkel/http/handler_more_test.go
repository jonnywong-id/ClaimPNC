package masterbengkelhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	"claim-pnc/internal/masterbengkel/usecase"
	"claim-pnc/internal/portal"

	masterbengkelhttp "claim-pnc/internal/masterbengkel/http"
	portalhttp "claim-pnc/internal/portal/http"
)

const sampleID = "010000000001"

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// directHandler memanggil handler langsung supaya jalur yang dijaga middleware ikut teruji.
type directHandler struct {
	handler *masterbengkelhttp.Handler
	store   *memory.Repo
	errors  []error
	known   bool
}

// uploaderPalsu berdiri di tempat layanan penyimpanan internal.
//
// Tanpa ini, setiap unggahan menjawab UploadMisconfigured — dan uji jalur berhasil akan
// menguji jalur gagal tanpa ada yang menyadarinya.
type uploaderPalsu struct{}

func (uploaderPalsu) Upload(context.Context, masterbengkel.DocumentFile) (string, error) {
	return "img-contoh", nil
}

func newDirectHandler(t *testing.T) *directHandler {
	t.Helper()

	d := &directHandler{store: memory.NewSampleRepo(), known: true}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return d.store, nil },
		Clock:        fixedClock{at: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)},
		Uploader:     uploaderPalsu{},
	})
	require.NoError(t, err)

	d.handler, err = masterbengkelhttp.NewHandler(masterbengkelhttp.Options{
		Service: service,
		Caller: func(context.Context) (masterbengkelhttp.Caller, bool) {
			return masterbengkelhttp.Caller{Login: "penguji"}, d.known
		},
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.errors = append(d.errors, err)
			w.WriteHeader(http.StatusTeapot)
		},
	})
	require.NoError(t, err)
	return d
}

func withPortalAndID(r *http.Request, id string) *http.Request {
	ctx := portalhttp.WithActivePortal(r.Context(), portal.Portal{Alias: "ASM"})
	if id != "" {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("id", id)
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeContext)
	}
	return r.WithContext(ctx)
}

func jsonRequest(method, body, id string) *http.Request {
	return withPortalAndID(httptest.NewRequest(method, "/", strings.NewReader(body)), id)
}

func multipartRequest(t *testing.T, field, name string, content []byte, id string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if field != "" {
		part, err := writer.CreateFormFile(field, name)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return withPortalAndID(request, id)
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestNewHandlerRejectsAnIncompleteAssembly(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return nil, nil },
	})
	require.NoError(t, err)
	caller := func(context.Context) (masterbengkelhttp.Caller, bool) {
		return masterbengkelhttp.Caller{}, true
	}

	_, err = masterbengkelhttp.NewHandler(masterbengkelhttp.Options{Caller: caller})
	require.EqualError(t, err, "masterbengkel/http: Service wajib diisi")
	_, err = masterbengkelhttp.NewHandler(masterbengkelhttp.Options{Service: service})
	require.EqualError(t, err, "masterbengkel/http: Caller wajib diisi")
	_, err = masterbengkelhttp.NewHandler(masterbengkelhttp.Options{Service: service, Caller: caller})
	require.EqualError(t, err, "masterbengkel/http: WriteResponse dan WriteError wajib diisi")
}

// Setiap handler di luar middleware portal menolak dengan galat portal.
func TestEveryHandlerWithoutAPortalIsRejected(t *testing.T) {
	d := newDirectHandler(t)
	handlers := []http.HandlerFunc{
		d.handler.List, d.handler.Get, d.handler.Create, d.handler.Save, d.handler.Decide,
		d.handler.Branches, d.handler.Cities, d.handler.Banks, d.handler.UploadDocument,
		d.handler.Document,
	}

	for _, call := range handlers {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}
	require.Len(t, d.errors, len(handlers))
	for _, err := range d.errors {
		require.ErrorIs(t, err, portal.ErrNotStated)
	}
}

// Tanpa identitas pemanggil, setiap jalur tulis menolak keras.
func TestWritesWithoutACallerFailLoudly(t *testing.T) {
	d := newDirectHandler(t)
	d.known = false

	for _, call := range []http.HandlerFunc{
		d.handler.Create, d.handler.Save, d.handler.Decide, d.handler.UploadDocument,
	} {
		recorder := httptest.NewRecorder()
		call(recorder, jsonRequest(http.MethodPost, "{}", sampleID))
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}
	require.Len(t, d.errors, 4)
	for _, err := range d.errors {
		require.EqualError(t, err, "masterbengkel/http: identitas pemanggil tidak ada di konteks")
	}
}

// Tanpa parameter id, setiap jalur berkunci menjawab tidak ditemukan.
func TestKeyedHandlersWithoutAnIDAreNotFound(t *testing.T) {
	d := newDirectHandler(t)

	for _, call := range []http.HandlerFunc{
		d.handler.Get, d.handler.Save, d.handler.UploadDocument, d.handler.Document,
	} {
		recorder := httptest.NewRecorder()
		call(recorder, jsonRequest(http.MethodPut, "{}", ""))
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, masterbengkelhttp.CodeNotFound, decode(t, recorder)["kode"])
	}
}

// Get mengembalikan satu bengkel; yang tidak ada dijawab 404.
func TestGetReturnsOneWorkshop(t *testing.T) {
	d := newDirectHandler(t)

	recorder := httptest.NewRecorder()
	d.handler.Get(recorder, jsonRequest(http.MethodGet, "", sampleID))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Bengkel Contoh Utama")

	recorder = httptest.NewRecorder()
	d.handler.Get(recorder, jsonRequest(http.MethodGet, "", "999999999999"))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// Badan yang cacat — termasuk dua dokumen JSON — ditolak 400.
func TestMalformedBodiesAreRejected(t *testing.T) {
	d := newDirectHandler(t)

	for _, c := range []struct {
		call http.HandlerFunc
		body string
	}{
		{d.handler.Save, "{"},
		{d.handler.Decide, `{"id_bengkel":["1"],"status":"1"} {}`},
	} {
		recorder := httptest.NewRecorder()
		c.call(recorder, jsonRequest(http.MethodPost, c.body, sampleID))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, masterbengkelhttp.CodeMalformedRequest, decode(t, recorder)["kode"])
	}
}

// Keputusan atas terlalu banyak baris sekaligus ditolak sebagai pelanggaran isian.
func TestDecidingTooManyRowsIsRejected(t *testing.T) {
	d := newDirectHandler(t)

	ids := make([]string, 201)
	for i := range ids {
		ids[i] = `"x"`
	}
	recorder := httptest.NewRecorder()
	d.handler.Decide(recorder, jsonRequest(http.MethodPost,
		`{"id_bengkel":[`+strings.Join(ids, ",")+`],"status":"1"}`, ""))
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, masterbengkelhttp.CodeValidationFailed, decode(t, recorder)["kode"])
}

// Kegagalan penyimpanan diserahkan ke penulis bersama pada setiap jalur baca.
func TestStorageFailuresGoToTheSharedWriter(t *testing.T) {
	d := newDirectHandler(t)
	failure := errors.New("oracle mati")
	d.store.SetError(failure)

	handlers := []http.HandlerFunc{d.handler.Branches, d.handler.Cities, d.handler.Banks}
	for _, call := range handlers {
		recorder := httptest.NewRecorder()
		request := jsonRequest(http.MethodGet, "", "")
		request.URL.RawQuery = "cari=jakarta"
		call(recorder, request)
		require.Equal(t, http.StatusTeapot, recorder.Code)
	}
	require.Len(t, d.errors, len(handlers))
	for _, err := range d.errors {
		require.ErrorIs(t, err, failure)
	}
}

// Unggah lalu baca metadatanya.
func TestUploadThenReadTheDocument(t *testing.T) {
	d := newDirectHandler(t)

	recorder := httptest.NewRecorder()
	d.handler.UploadDocument(recorder, multipartRequest(t, "berkas", `C:\folder\bukti.pdf`,
		[]byte("%PDF isi"), sampleID))
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	body := decode(t, recorder)
	document := body["dokumen"].(map[string]any)
	require.Equal(t, "application/pdf", document["tipe_media"])
	require.Equal(t, "img-contoh", document["image_id"])
	require.Equal(t, true, document["berisi"])
	require.Equal(t, "penguji", document["diunggah_oleh"])
	require.Equal(t, "2026-09-30T10:00:00Z", document["diunggah_pada"])

	recorder = httptest.NewRecorder()
	d.handler.Document(recorder, jsonRequest(http.MethodGet, "", sampleID))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, document["id_dokumen"],
		decode(t, recorder)["dokumen"].(map[string]any)["id_dokumen"])

}

// Unggahan tanpa bagian berkas, bukan multipart, atau berjenis terlarang ditolak.
func TestUploadRejections(t *testing.T) {
	d := newDirectHandler(t)

	recorder := httptest.NewRecorder()
	d.handler.UploadDocument(recorder, jsonRequest(http.MethodPost, "bukan multipart", sampleID))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, masterbengkelhttp.CodeMalformedRequest, decode(t, recorder)["kode"])

	recorder = httptest.NewRecorder()
	d.handler.UploadDocument(recorder, multipartRequest(t, "", "", nil, sampleID))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, decode(t, recorder)["pesan"], `"berkas"`)

	recorder = httptest.NewRecorder()
	d.handler.UploadDocument(recorder, multipartRequest(t, "berkas", "skrip.exe",
		[]byte("x"), sampleID))
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	require.Equal(t, masterbengkelhttp.CodeValidationFailed, decode(t, recorder)["kode"])
}

// Bengkel tanpa lampiran dijawab "dokumen tidak ditemukan", dan dokumen warisan tanpa
// IMAGEID dikenali sebagai dokumen yang berkasnya tidak pernah tersimpan.
func TestDocumentEdgeCases(t *testing.T) {
	d := newDirectHandler(t)
	ctx := context.Background()

	recorder := httptest.NewRecorder()
	d.handler.Document(recorder, jsonRequest(http.MethodGet, "", sampleID))
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, masterbengkelhttp.CodeDocumentNotFound, decode(t, recorder)["kode"])

	// Dokumen warisan: barisnya ada, IMAGEID-nya kosong. Ia tetap dijawab 200 beserta
	// keterangannya — yang membedakannya hanyalah `berisi: false`, dan itulah yang membuat
	// layar dapat menjelaskan sebabnya alih-alih menyatakan dokumennya tidak ada.
	require.NoError(t, d.store.SaveDocument(ctx, sampleID,
		masterbengkel.Document{ID: "260000000099", Name: "warisan.pdf"}))
	recorder = httptest.NewRecorder()
	d.handler.Document(recorder, jsonRequest(http.MethodGet, "", sampleID))
	require.Equal(t, http.StatusOK, recorder.Code)
	warisan := decode(t, recorder)["dokumen"].(map[string]any)
	require.Equal(t, false, warisan["berisi"])
	require.Equal(t, "", warisan["image_id"])
	require.Equal(t, "", warisan["diunggah_pada"])

	require.NoError(t, d.store.SaveDocument(ctx, sampleID,
		masterbengkel.Document{ID: "260000000100", Name: "tanpa-tipe", ImageID: "img-1"}))
	recorder = httptest.NewRecorder()
	d.handler.Document(recorder, jsonRequest(http.MethodGet, "", sampleID))
	require.Equal(t, http.StatusOK, recorder.Code)
	berisi := decode(t, recorder)["dokumen"].(map[string]any)
	require.Equal(t, true, berisi["berisi"])
	require.Equal(t, "img-1", berisi["image_id"])
}
