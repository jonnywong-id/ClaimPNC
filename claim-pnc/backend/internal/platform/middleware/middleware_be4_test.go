package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/middleware"
)

// TestRequestIDKeepsIncomingHeader membuktikan ID dari klien dipakai apa adanya.
func TestRequestIDKeepsIncomingHeader(t *testing.T) {
	var seen string
	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = logging.RequestID(r.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-Id", "dari-klien")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, "dari-klien", seen)
	require.Equal(t, "dari-klien", recorder.Header().Get("X-Request-Id"))
}

// TestRequestIDGeneratesHexID membuktikan ID baru dibuat (16 heksadesimal) bila tidak dikirim.
func TestRequestIDGeneratesHexID(t *testing.T) {
	var seen string
	handler := middleware.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = logging.RequestID(r.Context())
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Regexp(t, `^[0-9a-f]{16}$`, seen)
	require.Equal(t, seen, recorder.Header().Get("X-Request-Id"))
}

// TestLogRecordsStatusWithoutSecrets membuktikan satu baris log berisi status, bukan header rahasia.
func TestLogRecordsStatusWithoutSecrets(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := middleware.Log(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/klaim", nil)
	request.Header.Set("Authorization", "Bearer rahasia")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	var line map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &line))
	require.Equal(t, "permintaan selesai", line["msg"])
	require.Equal(t, "POST", line["metode"])
	require.Equal(t, "/api/klaim", line["jalur"])
	require.EqualValues(t, http.StatusAccepted, line["status"])
	require.NotContains(t, buffer.String(), "rahasia")
}

// TestLogDefaultsToOKWhenHandlerWritesNoHeader membuktikan status baku 200.
func TestLogDefaultsToOKWhenHandlerWritesNoHeader(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := middleware.Log(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	require.Contains(t, buffer.String(), `"status":200`)
}

// TestRecoverTurnsPanicIntoJSON500 membuktikan panik dijawab 500 JSON dan dicatat.
func TestRecoverTurnsPanicIntoJSON500(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	handler := middleware.Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("rusak")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/x", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.JSONEq(t, `{"kode":"galat_internal","pesan":"Terjadi kesalahan pada sistem."}`,
		recorder.Body.String())
	require.True(t, strings.Contains(buffer.String(), "permintaan panik"))
}

// TestRecoverPassesThroughNormalRequests membuktikan permintaan normal tidak tersentuh.
func TestRecoverPassesThroughNormalRequests(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	handler := middleware.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusNoContent, recorder.Code)
}
