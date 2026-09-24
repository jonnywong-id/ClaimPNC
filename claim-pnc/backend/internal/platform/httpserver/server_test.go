package httpserver_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/httpserver"
)

// Uji penyajian berkas antarmuka.
//
// # Kenapa uji ini ada
//
// Kerangka halaman yang tertahan di cache peramban membuat binary baru menyajikan layar
// LAMA — tanpa satu pun galat, tanpa satu pun jejak. Fitur yang sudah diperbaiki karena
// itu tampak masih rusak, dan tidak ada tempat mana pun yang mengatakan sebabnya.
//
// Itu benar-benar terjadi pada 2026-09-22, dan sebabnya bukan header yang lupa ditulis
// melainkan header yang dipasang di CABANG YANG SALAH: ia hanya ada pada jalur cadangan,
// sedangkan permintaan ke "/" menemukan index.html sebagai berkas nyata dan melewatinya.

func newSPA() http.Handler {
	files := fstest.MapFS{
		"index.html":          {Data: []byte("<!doctype html><title>Claim PNC</title>")},
		"assets/index-abc.js": {Data: []byte("console.log('hai')")},
	}
	return httpserver.Router(httpserver.Deps{
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
		MountAPI: func(chi.Router) {},
		SPAFiles: files,
	})
}

func TestPageSkeletonIsNeverCachedOnAnyPath(t *testing.T) {
	// Ketiga jalur menyajikan kerangka halaman yang sama, dan ketiganya harus menolak
	// cache. Jalur "/" yang paling sering dipakai justru yang dulu melewatinya.
	suite := []struct {
		name string
		path string
	}{
		// "/index.html" sengaja TIDAK diuji: http.FileServer menjawabnya 301 ke "./",
		// dan peramban memang tidak pernah memintanya langsung.
		{"akar", "/"},
		{"rute dalam milik router peramban", "/inbox/laporan-klaim"},
		{"rute dalam berparameter", "/inbox/laporan-klaim/RCVN.26.0001"},
	}

	server := newSPA()

	for _, c := range suite {
		t.Run(c.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, c.path, nil))

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"),
				"kerangka halaman pada %q boleh ditahan cache peramban", c.path)
		})
	}
}

func TestHashedAssetsAreNotForcedOutOfCache(t *testing.T) {
	// Kebalikannya juga penting. Nama berkas aset sudah memuat sidik isi, sehingga rilis
	// baru menghasilkan nama baru dan cache-nya tidak pernah basi. Memaksa no-store di
	// sini hanya membuat setiap muat ulang mengunduh ratusan kilobita tanpa manfaat.
	recorder := httptest.NewRecorder()
	newSPA().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/index-abc.js", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEqual(t, "no-store", recorder.Header().Get("Cache-Control"))
}

func TestUnknownAPIPathAnswersJSONNotFoundInsteadOfThePage(t *testing.T) {
	// Kegagalan yang ditahan uji ini benar-benar terjadi, dan bentuknya paling buruk:
	// LAYAR KOSONG tanpa satu pun petunjuk.
	//
	// Sebelum perbaikan, `r.NotFound(spa(...))` juga berlaku di dalam /api, sehingga alamat
	// API yang belum terdaftar dijawab `index.html` dengan status 200. Klien API membaca
	// 200, gagal mengurai HTML, lalu mengembalikan `null` — bukan melempar. Layar yang
	// menunggu datanya tidak menampilkan apa pun: bukan memuat, bukan galat, bukan data.
	r := httpserver.Router(httpserver.Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		MountAPI: func(api chi.Router) { api.Get("/ada", func(http.ResponseWriter, *http.Request) {}) },
		SPAFiles: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<!doctype html><html></html>")},
		},
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tidak-ada", nil))

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	require.NotContains(t, rec.Body.String(), "<!doctype html")

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "rute_tidak_ditemukan", body["kode"])
	require.Contains(t, body["pesan"], "/api/tidak-ada")
}

func TestUnknownAPIMethodIsToldApart(t *testing.T) {
	// Alamat yang benar dengan metode yang salah menuntut perbaikan yang BERBEDA dari
	// alamat yang salah. Menjawab keduanya "tidak ditemukan" membuat pemanggil mencari
	// alamat yang sebenarnya sudah benar.
	r := httpserver.Router(httpserver.Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		MountAPI: func(api chi.Router) { api.Get("/ada", func(http.ResponseWriter, *http.Request) {}) },
		SPAFiles: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<!doctype html><html></html>")},
		},
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/ada", nil))

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "metode_tidak_dilayani", body["kode"])
}

func TestSPARoutesOutsideAPIStillFallBackToThePage(t *testing.T) {
	// Yang TIDAK boleh ikut berubah: rute dalam milik router peramban. Memuat ulang halaman
	// pada `/inbox-rcl-pucl/klaim/...` harus tetap menghasilkan kerangka halaman, bukan 404.
	r := httpserver.Router(httpserver.Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		SPAFiles: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<!doctype html><html></html>")},
		},
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/inbox-rcl-pucl/klaim/PNC-1", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<!doctype html")
}
