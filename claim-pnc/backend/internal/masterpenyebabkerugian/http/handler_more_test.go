package masterpenyebabkerugianhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// layananPalsu menjawab setiap operasi dengan galat yang sudah ditentukan.
type layananPalsu struct{ err error }

func (s layananPalsu) List(context.Context, string) ([]masterpenyebabkerugian.CauseOfLoss, error) {
	return nil, s.err
}

func (s layananPalsu) Get(context.Context, string, string) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, s.err
}

func (s layananPalsu) Create(context.Context, string, string) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, s.err
}

func (s layananPalsu) Update(
	context.Context, string, string, string,
) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, s.err
}

func writeJSONForTest(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func codeOf(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body.Code
}

func requestWithPortal(method, body string) *http.Request {
	request := httptest.NewRequest(method, "/", strings.NewReader(body))
	return request.WithContext(portalhttp.WithActivePortal(
		request.Context(), portal.Portal{Alias: "ASM"}))
}

// Setiap handler di luar middleware portal menolak dengan galat portal.
func TestHandlersWithoutAnActivePortalAreRejected(t *testing.T) {
	var received []error
	h := NewHandler(Options{
		Service:       layananPalsu{},
		WriteResponse: writeJSONForTest,
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			received = append(received, err)
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	for _, call := range []http.HandlerFunc{h.List, h.Get, h.Create, h.Update} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
	require.Len(t, received, 4)
	for _, err := range received {
		require.ErrorIs(t, err, portal.ErrNotStated)
	}
}

// Galat layanan dipetakan menjadi kode modul pada setiap handler.
func TestServiceErrorsAreMappedOnEveryHandler(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{masterpenyebabkerugian.ErrNotFound, http.StatusNotFound, ErrCodeNotFound},
		{masterpenyebabkerugian.ErrIDTaken, http.StatusConflict, ErrCodeIDTaken},
		{masterpenyebabkerugian.ErrNoSite, http.StatusInternalServerError, ErrCodeSiteMissing},
	}

	for _, c := range cases {
		// Logger wajib dipasang: galat 5xx dicatat lewat logging.From, dan logger nil
		// membuat penulis galat panik (lihat laporan temuan).
		h := NewHandler(Options{
			Service:       layananPalsu{err: c.err},
			Logger:        slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			WriteResponse: writeJSONForTest,
		})

		for _, call := range []http.HandlerFunc{h.List, h.Get} {
			recorder := httptest.NewRecorder()
			call(recorder, requestWithPortal(http.MethodGet, ""))
			require.Equal(t, c.status, recorder.Code, c.code)
			require.Equal(t, c.code, codeOf(t, recorder))
		}
		for _, call := range []http.HandlerFunc{h.Create, h.Update} {
			recorder := httptest.NewRecorder()
			call(recorder, requestWithPortal(http.MethodPost, `{"deskripsi":"X"}`))
			require.Equal(t, c.status, recorder.Code, c.code)
			require.Equal(t, c.code, codeOf(t, recorder))
		}
	}
}

// Badan penyuntingan yang tidak terbaca ditolak sebelum layanan dipanggil.
func TestUpdateWithAnUnreadableBodyIsRejected(t *testing.T) {
	h := NewHandler(Options{
		Service:       layananPalsu{err: errors.New("tidak boleh dipanggil")},
		WriteResponse: writeJSONForTest,
	})

	recorder := httptest.NewRecorder()
	h.Update(recorder, requestWithPortal(http.MethodPut, "{"))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, ErrCodeBadRequest, codeOf(t, recorder))
}

// Galat asing tanpa cadangan menjadi 500 umum, dan rinciannya hanya masuk log.
func TestUnknownErrorsWithoutAFallbackBecomeAGenericInternalError(t *testing.T) {
	log := &bytes.Buffer{}
	h := NewHandler(Options{
		Service:       layananPalsu{err: errors.New("rahasia oracle")},
		Logger:        slog.New(slog.NewTextHandler(log, nil)),
		WriteResponse: writeJSONForTest,
	})

	recorder := httptest.NewRecorder()
	h.List(recorder, requestWithPortal(http.MethodGet, ""))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, ErrCodeInternal, codeOf(t, recorder))
	require.NotContains(t, recorder.Body.String(), "rahasia oracle")
	require.Contains(t, log.String(), "rahasia oracle")
}

// Galat asing diteruskan ke cadangan bila ada.
func TestUnknownErrorsGoToTheFallbackWriter(t *testing.T) {
	failure := errors.New("asing")
	var received error
	h := NewHandler(Options{
		Service:       layananPalsu{err: failure},
		WriteResponse: writeJSONForTest,
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			received = err
			w.WriteHeader(http.StatusTeapot)
		},
	})

	recorder := httptest.NewRecorder()
	h.Get(recorder, requestWithPortal(http.MethodGet, ""))
	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.ErrorIs(t, received, failure)
}
