package masterpasalaihttp_test

import (
	"bytes"
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
	"claim-pnc/internal/masterpasalai"
	"claim-pnc/internal/masterpasalai/repo/memory"
	masterpasalaiusecase "claim-pnc/internal/masterpasalai/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterpasalaihttp "claim-pnc/internal/masterpasalai/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena uji
// lain kebetulan memakai jalur yang benar.
const route = "/api/master/pasal-ai"

// Jumlah baris pada memory.SampleClause.
const sampleRows = 28

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi dan
// middleware portal.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	// ASI sengaja TIDAK diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (`ADR-0030`, `R-20`).
	asm := memory.NewRepo(memory.SampleClause())
	asi := memory.NewRepo(nil)

	service, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{
		RepoSelector: func(alias string) (masterpasalai.Repo, error) {
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

	writeResponse := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterpasalaihttp.NewHandler(masterpasalaihttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterpasalaihttp.ErrorWriter(writeError),
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
				masterpasalaihttp.Mount(protected, handler, portalDeps)
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

// call menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim.
func (p *testServer) call(
	t *testing.T,
	method, path, portalAlias string,
) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, p.server.URL+path, nil)
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

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// rows mengambil senarai pasal dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["pasal_ai"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai pasal: %v", content)
	return list
}

// pagination mengambil keterangan paginasi dari badan respons.
func pagination(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	page, ok := content["paginasi"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat keterangan paginasi: %v", content)
	return page
}

// TestListRequiresSession membuktikan rutenya benar-benar berada di balik middleware sesi.
func TestListRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestListRequiresPortal membuktikan permintaan tanpa header portal DITOLAK, bukan dilayani
// portal utama sebagai cadangan (`TKT-F6-002`, `R-20`).
func TestListRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// TestUnknownPortalRejected membuktikan portal yang belum siap menghasilkan galat, bukan
// daftar kosong yang tampak sah.
func TestUnknownPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "SMAS")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
}

// TestListIsolatedPerEntity membuktikan daftar satu entitas tidak pernah bocor ke entitas
// lain — kelas cacat yang `R-20` sebut paling berbahaya karena tidak terlihat sebagai galat.
func TestListIsolatedPerEntity(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotEmpty(t, rows(t, content))
	require.Equal(t, "ASM", content["portal"])

	response, content = p.call(t, http.MethodGet, route, "ASI")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, rows(t, content))
	require.Equal(t, "ASI", content["portal"])
	require.Equal(t, float64(0), pagination(t, content)["jumlah_baris"])
}

// TestFirstPageIsFullAndReportsTotal membuktikan halaman pertama dipotong pada PageSize
// sementara cacahnya melaporkan SELURUH baris.
//
// Inilah cacat paginasi yang paling sering lolos: cacah yang dihitung atas halaman membuat
// paginator selalu melaporkan satu halaman.
func TestFirstPageIsFullAndReportsTotal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Len(t, rows(t, content), masterpasalai.PageSize)

	page := pagination(t, content)
	require.Equal(t, float64(1), page["halaman"])
	require.Equal(t, float64(masterpasalai.PageSize), page["ukuran_halaman"])
	require.Equal(t, float64(sampleRows), page["jumlah_baris"])
	require.Equal(t, float64(2), page["jumlah_halaman"])
}

// TestSecondPageHoldsTheRemainder membuktikan jendela halaman kedua benar.
func TestSecondPageHoldsTheRemainder(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?halaman=2", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Len(t, rows(t, content), sampleRows-masterpasalai.PageSize)
	require.Equal(t, float64(2), pagination(t, content)["halaman"])
}

// TestPageBeyondRangeIsEmptyNotAnError membuktikan halaman di luar jangkauan dijawab halaman
// kosong — tautan paginasi yang basi bukan kesalahan pengguna.
func TestPageBeyondRangeIsEmptyNotAnError(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?halaman=99", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, rows(t, content))

	// Cacahnya tetap dikembalikan supaya layar dapat menunjukkan halaman mana yang ada.
	require.Equal(t, float64(sampleRows), pagination(t, content)["jumlah_baris"])
}

// TestPageNumberIsForgiving membuktikan nomor halaman cacat menjadi halaman pertama, bukan
// galat.
func TestPageNumberIsForgiving(t *testing.T) {
	p := newTestServer(t)

	for _, raw := range []string{"0", "-3", "abc", ""} {
		response, content := p.call(t, http.MethodGet, route+"?halaman="+raw, "ASM")
		require.Equalf(t, http.StatusOK, response.StatusCode, "halaman=%q", raw)
		require.Equalf(t, float64(1), pagination(t, content)["halaman"], "halaman=%q", raw)
	}
}

// TestSearchSpansAllThreeColumns membuktikan satu kata kunci dicocokkan ke KETIGA kolom,
// digabung OR — persis `Activity/GetListPasalAI_act.xml:548`.
func TestSearchSpansAllThreeColumns(t *testing.T) {
	p := newTestServer(t)

	// "18" hanya ada di kolom No Pasal.
	_, content := p.call(t, http.MethodGet, route+"?cari=18", "ASM")
	require.NotEmpty(t, rows(t, content), "kata kunci pada kolom No Pasal harus cocok")

	// "kebakaran" hanya ada di kolom Kejadian.
	_, content = p.call(t, http.MethodGet, route+"?cari=kebakaran", "ASM")
	require.NotEmpty(t, rows(t, content), "kata kunci pada kolom Kejadian harus cocok")
}

// TestSearchIgnoresLetterCase membuktikan pencarian tidak bergantung besar-kecil huruf.
//
// Oracle peka huruf pada LIKE, sehingga adapter SQL nanti wajib memasang UPPER di kedua sisi.
// Uji ini yang menjaga keduanya menjawab sama.
func TestSearchIgnoresLetterCase(t *testing.T) {
	p := newTestServer(t)

	_, lower := p.call(t, http.MethodGet, route+"?cari=kebakaran", "ASM")
	_, upper := p.call(t, http.MethodGet, route+"?cari=KEBAKARAN", "ASM")

	require.NotEmpty(t, rows(t, lower))
	require.Equal(t, len(rows(t, lower)), len(rows(t, upper)))
}

// TestSearchNarrowsTheTotal membuktikan penyaring memengaruhi CACAH, bukan hanya isi halaman.
func TestSearchNarrowsTheTotal(t *testing.T) {
	p := newTestServer(t)

	_, all := p.call(t, http.MethodGet, route, "ASM")
	_, filtered := p.call(t, http.MethodGet, route+"?cari=petir", "ASM")

	total := pagination(t, filtered)["jumlah_baris"].(float64)
	require.Greater(t, total, float64(0))
	require.Less(t, total, pagination(t, all)["jumlah_baris"].(float64))
}

// TestEmptyKeywordIsNotAnError membuktikan kata kunci kosong adalah jalur normal.
//
// Layar lama pun membuka daftar tanpa penyaring pada pemuatan pertama.
func TestEmptyKeywordIsNotAnError(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?cari=", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(sampleRows), pagination(t, content)["jumlah_baris"])
}

// TestOverlongKeywordRejected membuktikan kata kunci raksasa ditolak, bukan dipotong diam-diam.
func TestOverlongKeywordRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		route+"?cari="+strings.Repeat("a", 500), "ASM")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// TestRowsCarryTheKeyColumn membuktikan WP_ID ikut dikirim sebagai kunci baris.
//
// Ia tidak digambar sebagai kolom — layar lamanya hanya punya tiga — tetapi kuerinya memilih
// dan mengurutkan dengannya (`WP_ID AS "CaseID"` … `ORDER BY WP_ID`), dan layar
// membutuhkannya sebagai kunci baris tabel.
func TestRowsCarryTheKeyColumn(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM")
	first, ok := rows(t, content)[0].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, first["id"], "baris tidak membawa kunci WP_ID")
}

// TestRowsAreOrderedByKey membuktikan urutannya `ORDER BY WP_ID`, sama dengan kuerinya.
func TestRowsAreOrderedByKey(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM")

	previous := ""
	for _, raw := range rows(t, content) {
		one, ok := raw.(map[string]any)
		require.True(t, ok)
		id, _ := one["id"].(string)
		require.Greaterf(t, id, previous, "urutan tidak menaik pada id %q", id)
		previous = id
	}
}

// TestNewHandlerRejectsIncompleteOptions membuktikan rakitan setengah jadi gagal saat start,
// bukan saat pengguna sedang bekerja.
func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := masterpasalaihttp.NewHandler(masterpasalaihttp.Options{})
	require.Error(t, err)

	service, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{
		RepoSelector: func(string) (masterpasalai.Repo, error) { return nil, nil },
	})
	require.NoError(t, err)

	_, err = masterpasalaihttp.NewHandler(masterpasalaihttp.Options{Service: service})
	require.Error(t, err)
}

// TestNewServiceRejectsMissingSelector melengkapi uji di atas pada lapisan aplikasi.
func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := masterpasalaiusecase.NewService(masterpasalaiusecase.Options{})
	require.Error(t, err)
}
