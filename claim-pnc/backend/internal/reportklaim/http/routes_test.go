package reportklaimhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	authmemory "claim-pnc/internal/auth/repo/memory"
	authusecase "claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/repo/memory"
	reportklaimusecase "claim-pnc/internal/reportklaim/usecase"

	authhttp "claim-pnc/internal/auth/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
	reportklaimhttp "claim-pnc/internal/reportklaim/http"
)

// Uji rute modul Report Klaim.
//
// # Kenapa dirakit utuh, bukan handler saja
//
// Yang diuji di sini bukan handler-nya melainkan KONTRAKNYA dari luar: rutenya benar,
// sesi dan portal ditegakkan, berkasnya berbentuk CSV dengan judul kolom yang benar, dan
// penolakannya menyebutkan sebab yang tepat. Menguji handler secara terpisah tidak dapat
// membuktikan satu pun dari itu.

const (
	adminLogin = "adminpnc"
	portalUji  = "ASM"
)

type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	fixed := clock.FixedAt(time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC))

	authService, err := authusecase.NewService(authusecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           fixed,
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	service, err := reportklaimusecase.NewService(reportklaimusecase.Options{
		RepoSelector: memory.Selector(portalUji),
		Clock:        fixed,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := reportklaimhttp.NewHandler(reportklaimhttp.Options{
		Service: service,
		// Jembatan yang sama persis dengan cmd/claimpnc. Bila jembatan ini tidak ditiru,
		// uji akan lulus di sini dan gagal di aplikasi sungguhan.
		Caller: func(ctx context.Context) (reportklaim.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return reportklaim.Caller{}, false
			}
			return reportklaim.Caller{Login: baseCtx.User.Login, Name: baseCtx.User.Name}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    reportklaimhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{portalUji, "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	server := httptest.NewServer(httpserver.Router(httpserver.Deps{
		Logger: logger,
		MountAPI: func(api chi.Router) {
			authhttp.Mount(api, authhttp.NewHandler(authService, logger), authService, logger)
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(authService, writeError))
				reportklaimhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server}
	p.token = p.login(t)
	return p
}

func (p *testServer) login(t *testing.T) string {
	t.Helper()

	body := strings.NewReader(`{"nama_pengguna":"` + adminLogin + `","kata_sandi":"rahasia123"}`)
	response, err := http.Post(p.server.URL+"/api/masuk", "application/json", body)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var content struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&content))
	return content.Token
}

// get menjalankan satu permintaan GET. portalAlias kosong berarti header tidak dikirim.
func (p *testServer) get(t *testing.T, path, portalAlias string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, p.server.URL+path, nil)
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func decodeJSON(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	return body
}

const rentangSah = "dari=2026-09-01&sampai=2026-09-30"

// ---------------------------------------------------------------------------

func TestKatalogMengembalikanKeduapuluhDelapanPanelBerkelompok(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim", portalUji)
	require.Equal(t, http.StatusOK, response.StatusCode)

	body := decodeJSON(t, response)
	require.Equal(t, "Report Claim", body["judul"], "judul layar disalin dari harness")

	groups, _ := body["kelompok"].([]any)
	require.Len(t, groups, 5)

	total := 0
	for _, g := range groups {
		item, _ := g.(map[string]any)
		laporan, _ := item["laporan"].([]any)
		total += len(laporan)
	}
	require.Equal(t, 28, total)

	lines, _ := body["lini_bisnis"].([]any)
	require.Len(t, lines, 5)
}

// Panel yang terhalang tetap TAMPIL, bertanda sebab dan penghalangnya.
func TestPanelTerhalangTampilBesertaSebabnya(t *testing.T) {
	p := newTestServer(t)
	body := decodeJSON(t, p.get(t, "/api/report-klaim", portalUji))

	terhalang := map[string]map[string]any{}
	groups, _ := body["kelompok"].([]any)
	for _, g := range groups {
		item, _ := g.(map[string]any)
		laporan, _ := item["laporan"].([]any)
		for _, l := range laporan {
			r, _ := l.(map[string]any)
			if r["tersedia"] == false {
				terhalang[r["kode"].(string)] = r
			}
		}
	}

	require.Len(t, terhalang, 3)
	for kode, r := range terhalang {
		require.NotEmptyf(t, r["alasan"], "%s tanpa alasan", kode)
		require.NotEmptyf(t, r["penghalang"], "%s tanpa penghalang", kode)
	}
}

// Kartu menyebutkan isian mana yang AKTIF. Tanpa itu, pengguna mengisi rentang tanggal
// untuk laporan yang kuerinya tidak menerima tanggal sama sekali.
func TestKartuMenyebutkanPenyaringYangBerlaku(t *testing.T) {
	p := newTestServer(t)
	body := decodeJSON(t, p.get(t, "/api/report-klaim", portalUji))

	byCode := map[string]map[string]any{}
	groups, _ := body["kelompok"].([]any)
	for _, g := range groups {
		item, _ := g.(map[string]any)
		laporan, _ := item["laporan"].([]any)
		for _, l := range laporan {
			r, _ := l.(map[string]any)
			byCode[r["kode"].(string)] = r
		}
	}

	tat, _ := byCode["tat"]["penyaring"].(map[string]any)
	require.True(t, tat["rentang_tanggal"].(bool))
	require.True(t, tat["lini_bisnis"].(bool))
	require.False(t, tat["bisnis"].(bool))

	// Laporan Komunikasi Klaim tidak menerima satu pun penyaring.
	komunikasi, _ := byCode["komunikasi-klaim"]["penyaring"].(map[string]any)
	require.False(t, komunikasi["rentang_tanggal"].(bool))
	require.False(t, komunikasi["lini_bisnis"].(bool))

	// Panel Klaim Per Bisnis satu-satunya yang memakai autocomplete Bisnis.
	bisnis, _ := byCode["klaim-per-bisnis"]["penyaring"].(map[string]any)
	require.True(t, bisnis["bisnis"].(bool))
}

func TestPanelDataKomiteMengirimDuaTombol(t *testing.T) {
	p := newTestServer(t)
	body := decodeJSON(t, p.get(t, "/api/report-klaim", portalUji))

	var tombol []any
	groups, _ := body["kelompok"].([]any)
	for _, g := range groups {
		item, _ := g.(map[string]any)
		laporan, _ := item["laporan"].([]any)
		for _, l := range laporan {
			r, _ := l.(map[string]any)
			if r["kode"] == "komite" {
				tombol, _ = r["tombol"].([]any)
			}
		}
	}
	require.Len(t, tombol, 2)
}

// ---------------------------------------------------------------------------

func TestEksporMengembalikanBerkasCSVBerjudulKolom(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/pla/ekspor?"+rentangSah, portalUji)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "text/csv")
	require.Equal(t, "no-store", response.Header.Get("Cache-Control"),
		"berkas berisi data nasabah tidak boleh mengendap di cache")

	disposition := response.Header.Get("Content-Disposition")
	require.Contains(t, disposition, "attachment;")
	require.Contains(t, disposition, "Laporan Data PLA 20260901-20260930.csv")

	record, err := csv.NewReader(response.Body).ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, record)

	report, ok := reportklaim.Find(reportklaim.CodePLA)
	require.True(t, ok)
	require.Equal(t, reportklaim.Headers(report.Columns(reportklaim.Filter{})), record[0])
	require.Len(t, record, 1+3, "satu baris judul + tiga baris contoh")
}

// Susunan kolom TAT berbeda menurut lini bisnis, dan berkasnya harus mengikutinya.
func TestSusunanKolomBerkasMengikutiLiniBisnis(t *testing.T) {
	p := newTestServer(t)

	for _, tc := range []struct {
		lini  string
		kolom int
	}{
		{"002", 52},
		{"346", 39},
		{"003", 34},
	} {
		response := p.get(t, "/api/report-klaim/tat/ekspor?"+rentangSah+"&lini="+tc.lini, portalUji)
		require.Equal(t, http.StatusOK, response.StatusCode)

		record, err := csv.NewReader(response.Body).ReadAll()
		require.NoError(t, err)
		require.Lenf(t, record[0], tc.kolom, "lini %s", tc.lini)
	}
}

// Rentang tanggal wajib pada laporan yang memakainya, dan penolakannya menyebut ISIANNYA
// — bukan sekadar "permintaan tidak sah".
func TestTanggalWajibDitolakDenganMenyebutIsiannya(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/tat/ekspor", portalUji)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	body := decodeJSON(t, response)
	require.Equal(t, "validasi_gagal", body["kode"])

	detail, _ := body["detail"].([]any)
	require.Len(t, detail, 1)
	first, _ := detail[0].(map[string]any)
	require.Equal(t, "dari", first["isian"])
}

// Laporan yang TIDAK memakai rentang tanggal tidak boleh ikut mewajibkannya.
func TestLaporanTanpaRentangTanggalTidakMewajibkannya(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/komunikasi-klaim/ekspor", portalUji)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

// Laporan yang terhalang dijawab 409 dengan kode tersendiri — bukan 404, karena
// laporannya ADA.
func TestLaporanTerhalangDijawabDenganKodeTersendiri(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/adjuster/ekspor?"+rentangSah, portalUji)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "laporan_belum_tersedia", decodeJSON(t, response)["kode"])
}

func TestLaporanTidakDikenalDijawab404(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/tidak-ada/ekspor?"+rentangSah, portalUji)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", decodeJSON(t, response)["kode"])
}

// Panel bertombol dua menolak permintaan tanpa kode tombol: memilihkan salah satunya
// berarti mengunduh laporan yang berbeda isi dari yang diminta.
func TestPanelBertombolDuaMenolakPermintaanTanpaTombol(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/komite/ekspor?"+rentangSah, portalUji)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "tombol_tidak_dikenal", decodeJSON(t, response)["kode"])

	ok := p.get(t, "/api/report-klaim/komite/ekspor?"+rentangSah+"&aksi=approve", portalUji)
	require.Equal(t, http.StatusOK, ok.StatusCode)
}

// Nilai lini bisnis asing DITOLAK. Menerimanya berarti kendali yang di layar diberikan
// dropdown hilang begitu permintaan datang dari luar layar.
func TestLiniBisnisAsingDitolak(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/tat/ekspor?"+rentangSah+"&lini=004", portalUji)
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// Panel Klaim Per Bisnis TIDAK dapat dijalankan tanpa memilih bisnis — kuerinya tidak
// punya cabang "bila kosong", dan hasilnya akan kosong tanpa satu pun tanda.
func TestKlaimPerBisnisMewajibkanPilihanBisnis(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/klaim-per-bisnis/ekspor", portalUji)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail, _ := decodeJSON(t, response)["detail"].([]any)
	first, _ := detail[0].(map[string]any)
	require.Equal(t, "bisnis", first["isian"])

	ok := p.get(t, "/api/report-klaim/klaim-per-bisnis/ekspor?bisnis=10076", portalUji)
	require.Equal(t, http.StatusOK, ok.StatusCode)
}

func TestPilihanBisnisDapatDibaca(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/pilihan-bisnis", portalUji)
	require.Equal(t, http.StatusOK, response.StatusCode)

	bisnis, _ := decodeJSON(t, response)["bisnis"].([]any)
	require.NotEmpty(t, bisnis)
	first, _ := bisnis[0].(map[string]any)
	require.NotEmpty(t, first["kode"])
	require.NotEmpty(t, first["nama"])
}

// ---------------------------------------------------------------------------

// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti mengunduh data satu badan hukum dari basis
// data badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/report-klaim",
		"/api/report-klaim/pilihan-bisnis",
		"/api/report-klaim/pla/ekspor?" + rentangSah,
	} {
		response := p.get(t, path, "")
		require.NotEqualf(t, http.StatusOK, response.StatusCode, "jalur %s", path)
	}
}

func TestPermintaanTanpaSesiDitolak(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response := p.get(t, "/api/report-klaim", portalUji)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Alamat API yang tidak dikenal harus dijawab JSON, bukan halaman SPA — cacat yang
// pernah membuat layar kosong tanpa satu pun petunjuk (catatan-pengembangan §47).
func TestAlamatAPIYangSalahDijawabJSON(t *testing.T) {
	p := newTestServer(t)

	response := p.get(t, "/api/report-klaim/salah/ketik/lagi", portalUji)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "application/json")
}
