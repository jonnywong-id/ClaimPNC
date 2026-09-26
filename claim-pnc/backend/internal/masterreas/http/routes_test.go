package masterreashttp_test

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
	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterreas/repo/memory"
	masterreasusecase "claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterreashttp "claim-pnc/internal/masterreas/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena uji
// lain kebetulan memakai jalur yang benar.
const route = "/api/master/reas"

// Jumlah baris pada memory.SampleList.
const sampleRows = 7

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa satu entitas tidak pernah melihat data entitas lain. Menguji handler
// secara terpisah tidak dapat membuktikan ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi tidak diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (`ADR-0030`, `R-20`). Pada modul ini yang bocor adalah daftar mitra reasuransi
	// beserta surel tujuan pemberitahuan klaim satu badan hukum.
	asm *memory.Repo
	asi *memory.Repo
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	authService, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           clock.FixedAt(time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{})

	service, err := masterreasusecase.NewService(
		masterreasusecase.Options{
			RepoSelector: func(alias string) (masterreas.Repo, error) {
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
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal
	// akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterreashttp.NewHandler(
		masterreashttp.Options{
			Service:       service,
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    masterreashttp.ErrorWriter(writeError),
		})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	// Itulah yang membedakan "tidak ada" dari "belum tersedia".
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
				masterreashttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi}
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

// rows mengambil senarai member dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["member_reas"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai member: %v", content)
	return list
}

// TestListRequiresSession membuktikan rutenya benar-benar berada di balik middleware sesi.
func TestListRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestListRequiresPortal membuktikan permintaan tanpa header portal DITOLAK, bukan dilayani
// portal utama sebagai cadangan.
//
// Ini penegakan `TKT-F6-002`. Jatuh ke koneksi bawaan berarti menampilkan mitra reasuransi
// satu badan hukum kepada pengguna badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestListRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// TestListMenolakPortalYangBelumSiap membuktikan entitas yang kredensial basis datanya belum
// diisi dijawab 503, bukan 400.
//
// Pembedaannya berarti bagi pengguna: 400 mengatakan "permintaan Anda salah", sedangkan
// keadaan yang sebenarnya adalah "entitas ini memang belum dilayani" — dan itu pekerjaan
// administrator, bukan pekerjaan pengguna.
func TestListMenolakPortalYangBelumSiap(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "SMAS")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
}

// TestListMengembalikanSeluruhBaris membuktikan daftar dan alias portalnya terkirim.
func TestListMengembalikanSeluruhBaris(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), sampleRows)
	require.Equal(t, "ASM", content["portal"])
}

// TestListMengirimEnamKolomDanPenandaCadangan membuktikan bentuk kontraknya.
//
// `cadangan` adalah TURUNAN, bukan kolom — dihitung server supaya layar tidak perlu
// mengetahui bahwa TYPE `'1'` punya arti khusus.
func TestListMengirimEnamKolomDanPenandaCadangan(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route+"?cari=RE-004", "ASM")
	list := rows(t, content)
	require.Len(t, list, 1)

	one, ok := list[0].(map[string]any)
	require.True(t, ok)

	require.Equal(t, "RE-004", one["kode_reas"])
	require.Equal(t, "Cakrawala Re Asia", one["nama_reas"])
	require.Equal(t, "CakrawalaReAsia", one["login"])
	require.Equal(t, "dla.cakrawala@contoh.invalid", one["email"])
	require.Equal(t, "Singapura", one["negara"])
	require.Equal(t, "2", one["tipe"])
	require.Equal(t, false, one["cadangan"],
		"RE-004 sengaja tidak punya baris TYPE '1'; lihat memory.SampleList")

	require.NotContains(t, one, "countryid",
		"COUNTRYID ditulis UPDATEREAS tetapi tidak dibaca satu pun rule; ia tidak dikirim")
}

// TestListMenandaiBarisCadangan membuktikan TYPE '1' dikenali sebagai cadangan.
func TestListMenandaiBarisCadangan(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route+"?cari=Bahtera", "ASM")
	list := rows(t, content)
	require.Len(t, list, 1)

	one, ok := list[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, one["cadangan"])
}

// TestListMenyaringKataKunci membuktikan penyaring `cari` benar-benar dipakai.
func TestListMenyaringKataKunci(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route+"?cari=nusantara", "ASM")
	require.Len(t, rows(t, content), 3)
}

// TestListMenolakKataKunciTerlaluPanjang membuktikan permintaan yang tidak dapat dipenuhi
// ditolak dengan sebabnya, bukan dijawab daftar kosong.
//
// Daftar kosong akan terbaca sebagai "tidak ada datanya", dan pengguna tidak punya cara
// membedakan keduanya (`10-API-STRATEGY.md` §4).
func TestListMenolakKataKunciTerlaluPanjang(t *testing.T) {
	p := newTestServer(t)

	panjang := strings.Repeat("a", masterreashttp.MaxKeywordLength+1)
	response, content := p.call(t, http.MethodGet, route+"?cari="+panjang, "ASM")

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, masterreashttp.CodeMalformedRequest, content["kode"])
}

// TestListTerpisahAntarPortal membuktikan satu entitas tidak pernah melihat data entitas
// lain lewat jalur HTTP yang sungguhan.
func TestListTerpisahAntarPortal(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, route, "ASM")
	require.Len(t, rows(t, asm), sampleRows)

	_, asi := p.call(t, http.MethodGet, route, "ASI")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

// TestDaftarKosongTerkirimSebagaiSenarai membuktikan entitas tanpa baris menjawab `[]`,
// bukan `null`.
//
// Klien yang memetakan hasilnya tanpa memeriksa nil akan gagal pada `null` — dan entitas
// yang belum pernah mengirim PLA/DLA memang tidak punya satu pun baris.
func TestDaftarKosongTerkirimSebagaiSenarai(t *testing.T) {
	p := newTestServer(t)

	// Badan mentahnya dibaca langsung, bukan lewat p.call: yang diuji adalah BENTUK JSON-nya
	// — `[]` versus `null` — dan keduanya diuraikan menjadi hal yang sama oleh decoder.
	request, err := http.NewRequest(http.MethodGet, p.server.URL+route, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASI")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(response.Body)
	require.NoError(t, err)
	require.Contains(t, body.String(), `"member_reas":[]`)
}

// TestTidakAdaJalurTulis membuktikan modul ini benar-benar hanya membaca.
//
// Uji ini menjaga keputusan yang berdasar bukti: satu-satunya penulis `T_REINSURER` di
// sistem lama adalah alur PLA/DLA lewat `UPDATEREAS`, dipanggil `UpdateDetailPLA2` dan
// `UpdateDetailDLA2` — bukan layar master ini.
//
// Bila kelak terbukti layar lamanya punya tombol simpan (section gridnya memang tidak ada di
// export, `R-16`), uji ini yang akan gagal lebih dulu — dan itu memang yang diinginkan:
// penambahan jalur tulis harus menjadi keputusan yang disadari, bukan yang menyelinap.
func TestTidakAdaJalurTulis(t *testing.T) {
	p := newTestServer(t)

	for _, method := range []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	} {
		response, _ := p.call(t, method, route, "ASM")
		require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode,
			"%s pada %s seharusnya tidak terdaftar", method, route)
	}
}
