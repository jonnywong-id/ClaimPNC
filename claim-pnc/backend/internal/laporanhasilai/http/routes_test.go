package laporanhasilaihttp_test

import (
	"bytes"
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
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/laporanhasilai/repo/memory"
	laporanhasilaiusecase "claim-pnc/internal/laporanhasilai/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	laporanhasilaihttp "claim-pnc/internal/laporanhasilai/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena uji
// lain kebetulan memakai jalur yang benar.
const (
	route       = "/api/laporan-hasil-ai"
	exportRoute = "/api/laporan-hasil-ai/ekspor"
)

// wholeSeptember adalah parameter rentang yang mencakup seluruh baris contoh 2026.
const wholeSeptember = "?dari=2026-09-01&sampai=2026-09-30"

// Banyaknya baris contoh yang LOLOS penyaring — tiga dari sembilan sengaja tersaring.
const matchingRows = 6

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa satu entitas tidak pernah melihat data entitas lain.
type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	authService, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           clock.FixedAt(time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	// ASI sengaja TIDAK diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (`ADR-0030`, `R-20`).
	asm := memory.NewRepo(memory.SampleRows()...)
	asi := memory.NewRepo()

	service, err := laporanhasilaiusecase.NewService(laporanhasilaiusecase.Options{
		RepoSelector: func(alias string) (laporanhasilai.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{},
		&slog.HandlerOptions{Level: slog.LevelDebug}))

	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeJSON)

	handler, err := laporanhasilaihttp.NewHandler(laporanhasilaihttp.Options{
		Service:             service,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: laporanhasilaihttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	server := httptest.NewServer(httpserver.Router(httpserver.Deps{
		Logger: logger,
		MountAPI: func(api chi.Router) {
			authhttp.Mount(api, authhttp.NewHandler(authService, logger), authService, logger)
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(authService,
					authhttp.ErrorWriter(writeError)))
				laporanhasilaihttp.Mount(protected, handler, portalDeps)
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

	body := strings.NewReader(`{"nama_pengguna":"` + loginName + `","kata_sandi":"rahasia123"}`)
	response, err := http.Post(p.server.URL+"/api/masuk", "application/json", body)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var content struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&content))
	require.NotEmpty(t, content.Token)
	return content.Token
}

// raw menjalankan satu permintaan dan mengembalikan responsnya apa adanya.
//
// portalAlias kosong berarti header portal tidak dikirim.
func (p *testServer) raw(t *testing.T, path, portalAlias string) *http.Response {
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

// call menjalankan satu permintaan dan membaca badannya sebagai JSON.
func (p *testServer) call(t *testing.T, path, portalAlias string) (*http.Response, map[string]any) {
	t.Helper()

	response := p.raw(t, path, portalAlias)
	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// rows mengambil senarai baris dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["baris"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai baris: %v", content)
	return list
}

// summary mengambil senarai ringkasan dari badan respons.
func summary(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["ringkasan"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat ringkasan: %v", content)
	return list
}

// TestSearchRequiresSession membuktikan rutenya benar-benar berada di balik middleware sesi.
func TestSearchRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, route+wholeSeptember, "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestExportRequiresSession membuktikan jalur unduhan pun dijaga.
//
// Ia diuji terpisah dari Search karena rute unduhan mudah terlewat saat memasang
// middleware — keluarannya bukan JSON, sehingga kegagalannya tidak terlihat sebagai galat
// API melainkan sebagai berkas yang aneh.
func TestExportRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, exportRoute+wholeSeptember, "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestSearchRequiresPortal membuktikan permintaan tanpa portal DITOLAK.
//
// Bukan dilayani portal bawaan: itulah jalur kegagalan pertama `R-20`, dan kegagalannya
// tidak terlihat sebagai galat — layarnya tampil normal dengan data badan hukum lain.
func TestSearchRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, route+wholeSeptember, "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// TestUnknownPortalIsRejected membuktikan portal yang tidak dikenal tidak jatuh ke mana pun.
func TestUnknownPortalIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, route+wholeSeptember, "TIDAKADA")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
}

// TestOtherEntitySeesItsOwnEmptiness membuktikan data satu entitas tidak bocor ke entitas
// lain (`R-20`).
func TestOtherEntitySeesItsOwnEmptiness(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, route+wholeSeptember, "ASI")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, rows(t, content))
	require.Equal(t, "ASI", content["portal"])
}

// TestBothDatesAreRequired membuktikan kedua isian kosong dijawab 422 beserta keduanya.
//
// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan. Layar
// menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan pengguna yang harus
// ditandai di isiannya.
func TestBothDatesAreRequired(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, route, "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.Truef(t, ok, "galat tidak memuat rincian isian: %v", content)
	require.Len(t, detail, 2, "kedua isian yang kosong harus disebut sekaligus")
}

// TestMalformedDateFallsBackToTheMissingMessage membuktikan tanggal yang tidak terbaca
// tidak melahirkan pesan kedua.
//
// Isian di layar adalah pemilih tanggal; bentuk yang salah hanya mungkin datang dari alamat
// yang disunting tangan. Memberinya pesan tersendiri berarti menjelaskan bentuk tanggal
// kepada pengguna yang tidak pernah mengetiknya.
func TestMalformedDateFallsBackToTheMissingMessage(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, route+"?dari=03-09-2026&sampai=2026-09-30", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail, _ := content["detail"].([]any)
	require.Len(t, detail, 1)
	first, _ := detail[0].(map[string]any)
	require.Equal(t, "dari", first["field"])
}

// TestSearchReturnsSummaryAndRows membuktikan kedua grid terisi dalam satu permintaan.
//
// Keduanya bersama, bukan dua permintaan: layar lama pun mengisi `DatasearchLaporan` dan
// `TempTotal` dalam satu kali jalan, dan memisahkannya membuka kemungkinan ringkasan dan
// rinciannya dibaca dari keadaan yang berbeda.
func TestSearchReturnsSummaryAndRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, route+wholeSeptember, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Len(t, rows(t, content), matchingRows)

	tallies := summary(t, content)
	require.Len(t, tallies, 2, "ringkasan selalu dua baris")

	first, _ := tallies[0].(map[string]any)
	second, _ := tallies[1].(map[string]any)
	require.Equal(t, "Komite", first["keputusan"], "Komite digambar lebih dulu")
	require.Equal(t, "AI", second["keputusan"])
}

// TestSummaryTotalExcludesPending mengunci arti kolom Total pada kontrak API.
func TestSummaryTotalExcludesPending(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, route+wholeSeptember, "ASM")
	ai, _ := summary(t, content)[1].(map[string]any)

	require.Equal(t, float64(3), ai["diterima"])
	require.Equal(t, float64(2), ai["ditolak"])
	require.Equal(t, float64(1), ai["menunggu"])
	require.Equal(t, float64(5), ai["total"], "Total seharusnya Diterima + Ditolak")
}

// TestEveryRowCarriesAllTenColumns membuktikan kesepuluh field selalu dikirim.
//
// Termasuk lima yang SELALU kosong. Mengirimkannya tetap penting: bila ia hilang dari
// badan respons, kolomnya akan menghilang dari layar begitu saja — dan keputusan Work
// Owner 2026-09-26 adalah menampilkannya kosong, bukan menghapusnya.
func TestEveryRowCarriesAllTenColumns(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, route+wholeSeptember, "ASM")
	first, ok := rows(t, content)[0].(map[string]any)
	require.True(t, ok)

	for _, field := range []string{
		"no_klaim", "nama_object", "komite_status", "tanggal_komite",
		"ai_status", "tanggal_ai", "note_ai_terima", "note_ai_tolak",
		"coverage_final", "kategori_kronologi",
	} {
		require.Containsf(t, first, field, "kolom %q hilang dari badan respons", field)
	}

	require.Equal(t, "", first["nama_object"])
	require.Equal(t, "", first["note_ai_terima"])
	require.Equal(t, "", first["coverage_final"])
}

// TestPaginationWindowsTheRows membuktikan paginasinya benar-benar menggeser jendela.
func TestPaginationWindowsTheRows(t *testing.T) {
	p := newTestServer(t)

	_, first := p.call(t, route+wholeSeptember+"&halaman=1&ukuran=2", "ASM")
	_, second := p.call(t, route+wholeSeptember+"&halaman=2&ukuran=2", "ASM")

	require.Len(t, rows(t, first), 2)
	require.Len(t, rows(t, second), 2)

	firstRow, _ := rows(t, first)[0].(map[string]any)
	secondRow, _ := rows(t, second)[0].(map[string]any)
	require.NotEqual(t, firstRow["id"], secondRow["id"], "kedua halaman memuat baris yang sama")

	page, _ := first["paginasi"].(map[string]any)
	require.Equal(t, float64(matchingRows), page["total"])
	require.Equal(t, float64(3), page["total_halaman"])
}

// TestSummaryDoesNotFollowThePage membuktikan ringkasan tidak ikut dipaginasi.
func TestSummaryDoesNotFollowThePage(t *testing.T) {
	p := newTestServer(t)

	_, small := p.call(t, route+wholeSeptember+"&ukuran=2", "ASM")
	_, whole := p.call(t, route+wholeSeptember+"&ukuran=100", "ASM")

	require.Equal(t, summary(t, whole), summary(t, small),
		"ringkasan seharusnya tidak berubah saat pengguna berpindah halaman")
}

// TestFilterIsEchoedBack membuktikan jawabannya menyebutkan penyaring yang dipakainya.
//
// Tanpa itu, layar tidak dapat membedakan jawaban atas isian yang sedang terlihat dari sisa
// jawaban permintaan sebelumnya yang datang terlambat.
func TestFilterIsEchoedBack(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, route+wholeSeptember, "ASM")
	filter, ok := content["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "2026-09-01", filter["dari"])
	require.Equal(t, "2026-09-30", filter["sampai"])
}

// TestExportWritesCSVWithTheLegacyHeader mengunci judul kolom berkas ekspor.
//
// Keempat judul yang BERBEDA dari judul kolom di layar disalin persis dari `CSVPropHeaders`
// pada activity lama. Menyeragamkannya akan terasa lebih rapi dan sekaligus mengubah berkas
// yang sudah dipakai orang — nama kolom di CSV ikut terbawa ke formula Excel yang
// menunjuknya.
func TestExportWritesCSVWithTheLegacyHeader(t *testing.T) {
	p := newTestServer(t)

	response := p.raw(t, exportRoute+wholeSeptember, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, response.Header.Get("Content-Disposition"), "laporan-hasil-ai")

	records, err := csv.NewReader(response.Body).ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records)

	require.Equal(t, []string{
		"No Klaim", "Nama Object", "Komite Status", "Tanggal Komite",
		"AI Status", "Tanggal AI", "Note Terima", "Note Tolak",
		"Coverage AI Final", "Kategori Kronologi",
	}, records[0])

	require.Len(t, records, matchingRows+1, "seluruh baris yang cocok harus ikut, bukan satu halaman")
}

// TestExportFormatsDatesAsInPega membuktikan tanggal di berkas berbentuk dd/mm/yyyy.
//
// Bentuk itu berasal dari `.DISC` dan `.DISC2` — kedua tanggal yang disusun ulang activity
// lama khusus untuk berkas CSV-nya.
func TestExportFormatsDatesAsInPega(t *testing.T) {
	p := newTestServer(t)

	response := p.raw(t, exportRoute+wholeSeptember, "ASM")
	records, err := csv.NewReader(response.Body).ReadAll()
	require.NoError(t, err)

	first := records[1]
	require.Equal(t, "03/09/2026", first[3], "Tanggal Komite")
	require.Equal(t, "01/09/2026", first[5], "Tanggal AI")
}

// TestExportBlanksClaimNumberOnLaterSteps membuktikan aturan pengosongan ikut ke berkas.
//
// Akibatnya diterima secara sadar: berkasnya memuat sel kosong pada baris lanjutan,
// sehingga ia tidak dapat disaring menurut nomor klaim tanpa diisi lebih dulu. Itu
// keputusan Work Owner 2026-09-26.
func TestExportBlanksClaimNumberOnLaterSteps(t *testing.T) {
	p := newTestServer(t)

	response := p.raw(t, exportRoute+wholeSeptember, "ASM")
	records, err := csv.NewReader(response.Body).ReadAll()
	require.NoError(t, err)

	// Baris ketiga data adalah jenjang kedua KMT-000101.
	require.Equal(t, "", records[3][0], "nomor klaim seharusnya kosong pada jenjang kedua")
	require.NotEmpty(t, records[1][0], "nomor klaim seharusnya ada pada jenjang pertama")
}

// TestExportRejectsEmptyDatesBeforeWritingAnything membuktikan galat masih dapat dijawab
// sebagai JSON.
//
// Setelah satu byte berkas terkirim, galat tidak dapat lagi dijawab — yang sampai ke
// pengguna akan berupa berkas separuh jadi tanpa satu pun keterangan.
func TestExportRejectsEmptyDatesBeforeWritingAnything(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, exportRoute, "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.NotContains(t, response.Header.Get("Content-Type"), "text/csv")
}
