package inboxrclhttp_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/repo/memory"

	rclhttp "claim-pnc/internal/inboxrcl/http"
)

func post(t *testing.T, server http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("X-Portal", testPortal)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestSetujuMemindahkanKlaimKeRCLPUCL(t *testing.T) {
	server := buildServer(t, memory.SampleLogin, true)

	recorder := post(t, server, "/api/inbox-rcl/klaim/PNCN.26.0412/keputusan", `{"keputusan":"SETUJU"}`)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	body := decode(t, recorder)
	require.Equal(t, inboxrcl.StatusClaimRCL, body["status_klaim"])
	require.Equal(t, inboxrcl.StageNameRCLPUCL, body["tahap_berikutnya"])

	// Klaimnya tidak lagi terbuka di layar kerja dokter.
	require.Equal(t, http.StatusNotFound, get(t, server, "/api/inbox-rcl/klaim/PNCN.26.0412").Code)
}

func TestBackMSIGDenganAlasanDokter(t *testing.T) {
	recorder := post(t, buildServer(t, memory.SampleLogin, true),
		"/api/inbox-rcl/klaim/PNCN.26.0405/keputusan",
		`{"keputusan":"BackMSIG","alasan_dokter":"Dokumen medis belum lengkap."}`)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, inboxrcl.StageNameSendToAnalyst, decode(t, recorder)["tahap_berikutnya"])
}

func TestKeputusanSalahDitolak(t *testing.T) {
	server := buildServer(t, memory.SampleLogin, true)

	recorder := post(t, server, "/api/inbox-rcl/klaim/PNCN.26.0412/keputusan", `{"keputusan":"tolak"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, rclhttp.CodeUnknownDecision, decode(t, recorder)["kode"])

	recorder = post(t, server, "/api/inbox-rcl/klaim/PNCN.26.0412/keputusan", `{"keputusan":"MSIG"}`)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, rclhttp.CodeDecisionNotAllowed, decode(t, recorder)["kode"])

	recorder = post(t, server, "/api/inbox-rcl/klaim/PNCN.26.0412/keputusan", `bukan json`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, rclhttp.CodeUnreadableBody, decode(t, recorder)["kode"])

	recorder = post(t, server, "/api/inbox-rcl/klaim/PNCN.26.0350/keputusan", `{"keputusan":"SETUJU"}`)
	require.Equal(t, http.StatusNotFound, recorder.Code, "milik dokter lain")
}

func TestKeputusanTanpaPortalDitolak(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/inbox-rcl/klaim/PNCN.26.0412/keputusan",
		strings.NewReader(`{"keputusan":"SETUJU"}`))
	recorder := httptest.NewRecorder()
	buildServer(t, memory.SampleLogin, true).ServeHTTP(recorder, request)
	require.NotEqual(t, http.StatusOK, recorder.Code)
}
