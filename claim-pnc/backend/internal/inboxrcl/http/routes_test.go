package inboxrclhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/repo/memory"
	"claim-pnc/internal/inboxrcl/usecase"

	rclhttp "claim-pnc/internal/inboxrcl/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

var wib = time.FixedZone("WIB", 7*60*60)

const testPortal = "ASM"

func buildServer(t *testing.T, login string, callerKnown bool) http.Handler {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrcl.Repo, error) { return store, nil },
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}

	// Galat portal dipetakan modul portal, persis seperti rantai di cmd/claimpnc.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_internal", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := rclhttp.NewHandler(rclhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (rclhttp.Caller, bool) {
			if !callerKnown {
				return rclhttp.Caller{}, false
			}
			return rclhttp.Caller{Login: login}, true
		},
		Location:            wib,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: rclhttp.ErrorWriter(writeError),
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		rclhttp.Mount(api, handler, portalDeps)
	})
	return router
}

func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestAntreanDikembalikanBesertaTotal(t *testing.T) {
	recorder := get(t, buildServer(t, memory.SampleLogin, true), "/api/inbox-rcl")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, testPortal, body["portal"])
	require.Equal(t, float64(4), body["total"])
	require.Equal(t, true, body["identitas_lama_ditemukan"])
}

// TestBarisMembawaIsianLayar menjaga kontrak tidak menyusut diam-diam, dan memeriksa
// konversi WIB: 2026-09-22 03:00 UTC harus tampil 10:00.
func TestBarisMembawaIsianLayar(t *testing.T) {
	data := decode(t, get(t, buildServer(t, memory.SampleLogin, true), "/api/inbox-rcl"))["data"].([]any)
	first := data[0].(map[string]any)

	for _, field := range []string{
		"klaim_id", "nomor_case", "nomor_polis", "nama_tertanggung",
		"tanggal_masuk_inbox", "deskripsi_analyst", "dokter_rcl", "status_proses",
		"operator_penerima",
	} {
		require.Containsf(t, first, field, "isian %s hilang dari jawaban", field)
	}
	require.Equal(t, "PNCN.26.0412", first["nomor_case"])
	require.Equal(t, "2026-09-22 10:00", first["tanggal_masuk_inbox"])
}

func TestTanpaIdentitasLamaDijawab200DenganPenanda(t *testing.T) {
	recorder := get(t, buildServer(t, memory.SampleLoginNoLegacy, true), "/api/inbox-rcl")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, false, body["identitas_lama_ditemukan"])
	require.Equal(t, float64(0), body["total"])
	require.Empty(t, body["data"])
}

func TestPemanggilTakTerbacaDitolak409(t *testing.T) {
	recorder := get(t, buildServer(t, "", false), "/api/inbox-rcl")
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, rclhttp.CodeCallerUnknown, decode(t, recorder)["kode"])
}

func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/inbox-rcl", nil)
	recorder := httptest.NewRecorder()
	buildServer(t, memory.SampleLogin, true).ServeHTTP(recorder, request)

	require.GreaterOrEqual(t, recorder.Code, http.StatusBadRequest)
	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestKeteranganMemuatKelimaKolom(t *testing.T) {
	recorder := get(t, buildServer(t, memory.SampleLogin, true), "/api/inbox-rcl/keterangan")
	require.Equal(t, http.StatusOK, recorder.Code)

	kolom := decode(t, recorder)["kolom"].([]any)
	require.Len(t, kolom, 5)
	require.Equal(t, "Tanggal Masuk Inbox", kolom[3].(map[string]any)["judul"])
}

func TestPencarianDanPaginasiDibawaKembali(t *testing.T) {
	body := decode(t, get(t, buildServer(t, memory.SampleLogin, true),
		"/api/inbox-rcl?cari=26.005&batas=5000&lewati=abc"))

	require.Equal(t, float64(1), body["total"])
	require.Equal(t, float64(inboxrcl.MaxLimit), body["batas"])
	require.Equal(t, float64(0), body["lewati"])
	require.Equal(t, "26.005", body["cari"])
}
