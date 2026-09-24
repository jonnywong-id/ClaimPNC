package mastermaskinghttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	authmemory "claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/mastermasking/repo/memory"
	mastermaskingusecase "claim-pnc/internal/mastermasking/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastermaskinghttp "claim-pnc/internal/mastermasking/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/master/masking"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang
// sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya, dan pada
// modul ini keduanya menentukan apakah daftar kewenangan data pribadi satu badan hukum
// dapat terbaca dari badan hukum lain (`R-20`).
type testServer struct {
	server *httptest.Server
	token  string

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...).WithBranches(memory.SampleBranches()...)
	asi := memory.NewRepo().WithBranches(memory.SampleBranches()...)

	maskingService, err := mastermaskingusecase.NewService(mastermaskingusecase.Options{
		RepoSelector: func(alias string) (mastermasking.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Now: func() time.Time { return time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC) },
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal
	// akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := mastermaskinghttp.NewHandler(mastermaskinghttp.Options{
		Service: maskingService,
		Caller: func(ctx context.Context) (mastermaskinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastermaskinghttp.Caller{}, false
			}
			return mastermaskinghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    writeError,
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
				protected.Use(authhttp.Authenticate(authService, writeError))
				mastermaskinghttp.Mount(protected, handler, portalDeps)
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

	body := strings.NewReader(`{"nama_pengguna":"adminpnc","kata_sandi":"rahasia123"}`)
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
func (p *testServer) call(t *testing.T, method, path, portalAlias, body string) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, p.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// ── Sesi dan portal ─────────────────────────────────────────────────────────────

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun penyimpanan.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Permintaan tanpa portal DITOLAK, tidak pernah dialihkan ke portal utama sebagai cadangan.
//
// Inilah `TKT-F6-002`. Mengalihkannya diam-diam akan menampilkan daftar kewenangan data
// pribadi entitas lain pada layar yang tampak normal — kegagalan yang tidak memberi satu
// pun tanda.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", body["kode"])
}

// Portal yang belum siap dibedakan dari portal yang tidak dikenal.
func TestPortalNotReady(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "portal_belum_siap", body["kode"])
}

// Dua entitas menjawab dari penyimpanan yang berbeda.
//
// Uji ini adalah pembuktian `ADR-0030`: pemisahan ada di tingkat koneksi, bukan penyaringan
// baris. ASI sengaja dibiarkan kosong supaya kebocoran dari ASM langsung terlihat.
func TestPortalsAreSeparated(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(5), asm["total"])
	require.Equal(t, "ASM", asm["portal"])

	_, asi := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(0), asi["total"])
	require.Equal(t, "ASI", asi["portal"])
}

// ── Daftar dan pencarian ────────────────────────────────────────────────────────

// Daftar memuat baris aktif MAUPUN nonaktif secara baku — `SearchData.Type == "1"`.
func TestListIncludesInactive(t *testing.T) {
	p := newTestServer(t)

	_, all := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(5), all["total"])
}

// Pencarian menurut status — tipe keempat layar lama — menyaring kedua arahnya.
func TestSearchByStatus(t *testing.T) {
	p := newTestServer(t)

	_, active := p.call(t, http.MethodGet, route+"?cari_di=status&kata_kunci=AKTIF", "ASM", "")
	require.Equal(t, float64(4), active["total"])

	_, inactive := p.call(t, http.MethodGet,
		route+"?cari_di=status&kata_kunci="+url.QueryEscape("TIDAK AKTIF"), "ASM", "")
	require.Equal(t, float64(1), inactive["total"])
}

// Pencarian status tanpa menyebut statusnya ditolak dengan pesan layar lama.
func TestSearchByStatusWithoutValue(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"?cari_di=status", "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "status_belum_dipilih", body["kode"])
}

// Pencarian berdasarkan cabang menelusuri NAMA cabang.
func TestSearchByBranch(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodGet, route+"?cari_di=cabang&kata_kunci=malang", "ASM", "")
	require.Equal(t, float64(1), body["total"])
}

// Tipe pencarian yang tidak dikenal ditolak sebagai permintaan cacat.
//
// Mengabaikannya akan menampilkan SELURUH baris kepada pengguna yang mengira ia sedang
// menyaring — pada layar ini, itu berarti membuka seluruh peta kewenangan tanpa diminta.
func TestUnknownSearchTypeRejected(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"?cari_di=modul&kata_kunci=x", "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", body["kode"])
}

// Kolom PASSWORD tidak pernah dikirim ke peramban.
//
// Uji ini menjaga keputusan yang paling mudah batal tanpa disadari saat DTO disunting
// kelak. Isinya literal coba-coba yang tertinggal di sistem lama, dan tidak ada satu pun
// alasan ia sampai ke layar.
func TestPasswordNeverLeavesServer(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+route, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	raw := new(bytes.Buffer)
	_, err = raw.ReadFrom(response.Body)
	require.NoError(t, err)

	require.NotContains(t, strings.ToLower(raw.String()), "password")
	require.NotContains(t, strings.ToLower(raw.String()), "sandi")
}

// ── Menambah ────────────────────────────────────────────────────────────────────

const newRow = `{"cabang":"100001","login":"BARU.SEKALI","modul":"PNCSearchKlaim",
	"sub_modul":"Registrasi,","maks_cari":5,"maks_lihat":5,
	"lihat_ktp":true,"lihat_email":false,"lihat_notelp":false}`

// Penambahan menjawab 201 dan mengembalikan ID yang dibuat server.
func TestCreate(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM", newRow)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved := body["masking"].(map[string]any)
	require.NotEmpty(t, saved["id"])
	require.Equal(t, "BARU.SEKALI", saved["login"])
	require.Equal(t, true, saved["aktif"], "baris baru selalu aktif")
	require.Equal(t, true, saved["lihat_ktp"])
	require.Equal(t, false, saved["lihat_email"])
	// Nama cabang dihitung server, bukan dikirim klien.
	require.Equal(t, "AGENCY MANADO", saved["nama_cabang"])
}

// Pelaku diisi dari SESI, bukan dari badan permintaan.
func TestCreateStampsCallerFromSession(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodPost, route, "ASM", newRow)
	saved := body["masking"].(map[string]any)
	require.NotEmpty(t, saved["dicatat_oleh"])
	require.NotEmpty(t, saved["dicatat_pada"])
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam.
//
// Ketiganya sengaja tidak pernah diterima dari klien — `id`, `nama_cabang`, dan
// `dicatat_oleh`. Menolaknya terang-terangan lebih baik daripada mengabaikan diam-diam:
// salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah.
//
// `aktif` TIDAK ada di daftar ini — ia field yang sah, karena form layar lama pun memuat
// isian "STATUS".
func TestUnknownFieldsRejected(t *testing.T) {
	for name, body := range map[string]string{
		"id":           `{"cabang":"100001","login":"X","modul":"M","sub_modul":"","maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,"aktif":true,"id":"9"}`,
		"nama_cabang":  `{"cabang":"100001","login":"X","modul":"M","sub_modul":"","maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,"aktif":true,"nama_cabang":"PALSU"}`,
		"dicatat_oleh": `{"cabang":"100001","login":"X","modul":"M","sub_modul":"","maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,"aktif":true,"dicatat_oleh":"ORANG.LAIN"}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := newTestServer(t)
			response, content := p.call(t, http.MethodPost, route, "ASM", body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "permintaan_cacat", content["kode"])
		})
	}
}

// Isian cacat dijawab 422 beserta SELURUH pelanggarannya dan kolom yang melanggar.
func TestValidationFailure(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		`{"cabang":"","login":"","modul":"","sub_modul":"","maks_cari":-1,"maks_lihat":0,
		  "lihat_ktp":false,"lihat_email":false,"lihat_notelp":false}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])

	detail := body["detail"].([]any)
	require.GreaterOrEqual(t, len(detail), 4, "seluruh pelanggaran dikirim sekaligus")

	field := map[string]bool{}
	for _, row := range detail {
		field[row.(map[string]any)["field"].(string)] = true
	}
	require.True(t, field["cabang"])
	require.True(t, field["login"])
	require.True(t, field["modul"])
	require.True(t, field["maks_cari"])
}

// Pasangan cabang+login yang sudah ada dijawab 409, bukan 422.
//
// Isian penggunanya sah; yang bentrok adalah keadaan penyimpanan.
func TestDuplicatePairConflict(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		`{"cabang":"100081","login":"CONTOH.ADMIN","modul":"PNCSearchKlaim","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false}`)

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "masking_pengguna_sudah_ada", body["kode"])
}

// Cabang yang tidak dikenal ditolak dan ditandai pada kolom cabang.
func TestUnknownBranchRejected(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		`{"cabang":"999999","login":"SIAPA","modul":"PNCSearchKlaim","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "cabang_tidak_dikenal", body["kode"])
}

// ── Mengubah dan status ─────────────────────────────────────────────────────────

// Mengubah baris yang tidak ada dijawab 404.
func TestUpdateMissing(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPut, route+"/9999", "ASM", newRow)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "masking_tidak_ditemukan", body["kode"])
}

// Menyimpan form TIDAK dapat menghidupkan kembali baris yang sudah dinonaktifkan.
//
// Ini uji terpenting di berkas ini. Tanpa pemisahan jalur status, menyunting baris nonaktif
// akan diam-diam mengembalikan kewenangan membuka data pribadi — tanpa pesan apa pun, dan
// tanpa ada yang bermaksud demikian.
func TestUpdateCarriesStatusFromForm(t *testing.T) {
	p := newTestServer(t)

	_, before := p.call(t, http.MethodGet, route+"/1", "ASM", "")
	require.Equal(t, true, before["masking"].(map[string]any)["aktif"])

	response, body := p.call(t, http.MethodPut, route+"/1", "ASM",
		`{"cabang":"100081","login":"CONTOH.ADMIN","modul":"PNCSearchKlaim","sub_modul":"Registrasi,",
		  "maks_cari":9,"maks_lihat":9,"lihat_ktp":true,"lihat_email":true,"lihat_notelp":true,
		  "aktif":false}`)

	require.Equal(t, http.StatusOK, response.StatusCode)
	saved := body["masking"].(map[string]any)
	require.Equal(t, false, saved["aktif"], "status pada form ikut tersimpan")
	require.Equal(t, float64(9), saved["maks_cari"], "isian lain tetap tersimpan")
}

// Baris baru SELALU aktif, apa pun yang dikirim klien.
//
// Layar lama tidak punya cara membuat baris nonaktif — tombol penonaktifan hanya ada pada
// baris yang sudah tersimpan.
func TestCreateAlwaysActive(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodPost, route, "ASM",
		`{"cabang":"100001","login":"COBA.NONAKTIF","modul":"PNCSearchKlaim","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,
		  "aktif":false}`)

	require.Equal(t, true, body["masking"].(map[string]any)["aktif"])
}

// Menonaktifkan tidak membuang baris; ia tetap terbaca dan dapat dihidupkan kembali.
func TestSetStatusIsSoftDelete(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPut, route+"/1/status", "ASM", `{"aktif":false}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, false, body["masking"].(map[string]any)["aktif"])

	// Barisnya masih ada — "hapus" tidak pernah membuang apa pun (`D-66`).
	_, still := p.call(t, http.MethodGet, route+"/1", "ASM", "")
	require.Equal(t, false, still["masking"].(map[string]any)["aktif"])

	_, back := p.call(t, http.MethodPut, route+"/1/status", "ASM", `{"aktif":true}`)
	require.Equal(t, true, back["masking"].(map[string]any)["aktif"])
}

// DELETE tidak pernah didaftarkan.
//
// Layar lama punya tombol bernama DELETE, tetapi ia hanya mengubah STS_AKTF. Rute yang
// tidak ada tidak dapat dipanggil kode yang ditulis kemudian tanpa keputusan sadar.
func TestDeleteNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/1", "ASM", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
	require.NotEqual(t, http.StatusNoContent, response.StatusCode)
}

// ── Daftar cabang ───────────────────────────────────────────────────────────────

// Jalur "cabang" tidak tertangkap sebagai sebuah ID.
func TestBranchRouteNotMistakenForID(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/cabang", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotNil(t, body["cabang"], "yang dikembalikan daftar cabang, bukan satu baris masking")
}

// Daftar cabang dapat disaring dengan kata kunci.
func TestBranchSearch(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodGet, route+"/cabang?kata_kunci=manado", "ASM", "")
	require.Equal(t, float64(1), body["total"])
}
