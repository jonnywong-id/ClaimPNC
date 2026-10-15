package mastertipesurveyorshttp_test

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
	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/mastertipesurveyors/repo/memory"
	mastertipesurveyorsusecase "claim-pnc/internal/mastertipesurveyors/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastertipesurveyorshttp "claim-pnc/internal/mastertipesurveyors/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/master/tipe-surveyor"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang
// sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()

	surveyorTypeService, err := mastertipesurveyorsusecase.NewService(mastertipesurveyorsusecase.Options{
		RepoSelector: func(alias string) (mastertipesurveyors.Repo, error) {
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

	handler, err := mastertipesurveyorshttp.NewHandler(mastertipesurveyorshttp.Options{
		Service:       surveyorTypeService,
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
				mastertipesurveyorshttp.Mount(protected, handler, portalDeps)
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
// pemeriksaan portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])
}

// PERMINTAAN TANPA PORTAL DITOLAK, bukan dilayani portal utama (TKT-F6-002, R-20).
//
// Ini uji terpenting di berkas ini. Bila permintaan tanpa portal jatuh ke koneksi default,
// golongan surveyor satu badan hukum akan terbaca atau tertulis di basis data badan hukum
// lain — dan layarnya tampak normal, karena isinya masuk akal. Yang salah hanya milik
// siapa data itu.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	t.Run("membaca daftar", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, route, "", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	t.Run("membaca satu baris", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, route+"/1001", "", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	t.Run("menambah", func(t *testing.T) {
		response, content := p.call(t, http.MethodPost, route, "", `{"deskripsi":"UJI"}`)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	t.Run("mengubah", func(t *testing.T) {
		response, content := p.call(t, http.MethodPut, route+"/1001", "", `{"deskripsi":"UJI"}`)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	// Dan tidak ada satu baris pun yang berubah di entitas mana pun.
	listASM, err := p.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, listASM, 4)
	listASI, err := p.asi.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, listASI)
}

// "Tidak ada" dibedakan dari "belum tersedia": keduanya menuntut tindak lanjut berbeda —
// yang pertama kesalahan pemanggil, yang kedua pekerjaan tim infrastruktur.
func TestUnknownAndNotReadyPortalAreDistinguished(t *testing.T) {
	p := newTestServer(t)

	t.Run("tidak ada di daftar", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, route, "TIDAKADA", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeUnknown, content["kode"])
	})

	t.Run("ada tetapi koneksinya belum hidup", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, route, "SMAS", "")
		require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotReady, content["kode"])
	})
}

// Entitas yang MENJAWAB disebut di setiap respons, bukan diandaikan sama dengan yang
// diminta. Tanpa itu, layar tidak punya cara membuktikan data yang ditampilkannya memang
// milik entitas yang sedang dipilih.
func TestResponseNamesAnsweringPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
}

// Entitas yang berbeda menjawab dengan isinya sendiri. ASI kosong, ASM berisi empat —
// dan keduanya dilayani rute yang sama.
func TestEachPortalAnswersWithItsOwnRows(t *testing.T) {
	p := newTestServer(t)

	_, isiASM := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(4), isiASM["total"])

	response, isiASI := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(0), isiASI["total"])
	require.Equal(t, "ASI", isiASI["portal"])

	// Kosong dikirim sebagai senarai kosong, bukan null: layar yang melakukan .map atas
	// null akan gagal, dan kegagalannya tampak seperti layar rusak.
	require.NotNil(t, isiASI["tipe_surveyor"])
	require.Empty(t, isiASI["tipe_surveyor"])
}

// ── Membaca ─────────────────────────────────────────────────────────────────────

// Jumlah datang dari server, bukan dihitung klien dari panjang senarai.
func TestListSendsTotalFromServer(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(4), content["total"])

	list, _ := content["tipe_surveyor"].([]any)
	require.Len(t, list, 4)

	first, _ := list[0].(map[string]any)
	require.Equal(t, "1001", first["kode"])
	require.Equal(t, "INTERNAL SURVEYOR", first["deskripsi"])
	require.Equal(t, "", first["kode_lama"], "kolomnya dikirim walau kosong")
}

func TestGetSingleRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/1002", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	tipe, _ := content["tipe_surveyor"].(map[string]any)
	require.Equal(t, "LOSS ADJUSTER", tipe["deskripsi"])
}

func TestGetMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/9999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeNotFound, content["kode"])
}

// ── Menambah dan mengubah ───────────────────────────────────────────────────────

// Penambahan menjawab 201 beserta KODE yang baru diterbitkan server. Kode itu tidak
// diketahui klien dengan cara lain — menebaknya di peramban akan menampilkan kode yang
// salah sampai muat ulang berikutnya.
func TestCreateReturnsIssuedCode(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{"deskripsi":"ADJUSTER INDEPENDEN"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	tipe, _ := content["tipe_surveyor"].(map[string]any)
	require.Equal(t, "1005", tipe["kode"])
	require.Equal(t, "ADJUSTER INDEPENDEN", tipe["deskripsi"])

	// Dan entitas lain tidak ikut bertambah.
	listASI, err := p.asi.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, listASI)
}

func TestUpdateExistingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/1003", "ASM", `{"deskripsi":"TENAGA AHLI"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	tipe, _ := content["tipe_surveyor"].(map[string]any)
	require.Equal(t, "1003", tipe["kode"], "kode tidak ikut berubah")
	require.Equal(t, "TENAGA AHLI", tipe["deskripsi"])
}

func TestUpdateMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/9999", "ASM", `{"deskripsi":"APA SAJA"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeNotFound, content["kode"])
}

// Nama ganda dijawab 409, BUKAN 422: isian penggunanya sah, tetapi bentrok dengan keadaan
// penyimpanan saat ini. Layar menanganinya berbeda.
func TestDuplicateDescriptionAnsweredAsConflict(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{"deskripsi":"  expert  "}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeDescriptionTaken, content["kode"])
}

// Validasi dijawab 422 beserta SELURUH pelanggarannya, lengkap dengan nama field supaya
// layar dapat menandai kolom yang salah — bukan sekadar satu pesan di atas form.
func TestValidationFailureSendsViolations(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{"deskripsi":"   "}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeValidationFailed, content["kode"])

	detail, _ := content["detail"].([]any)
	require.NotEmpty(t, detail)

	first, _ := detail[0].(map[string]any)
	require.Equal(t, mastertipesurveyors.FieldDescription, first["field"])
	require.NotEmpty(t, first["pesan"])
}

// ── Bentuk permintaan ───────────────────────────────────────────────────────────

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam. Ia juga yang menolak klien
// yang mencoba mengirim `kode` — kode tidak pernah boleh datang dari luar, karena ia
// dipatok tiga kueri Pega.
func TestUnknownFieldRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"deskripsi":"TIPE BARU","kode":"9999"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeMalformedRequest, content["kode"])

	// Dan tidak ada baris yang tersimpan.
	list, err := p.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 4)
}

func TestMalformedBodyRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeMalformedRequest, content["kode"])
}

// Badan yang memuat lebih dari satu dokumen JSON ditolak.
func TestTrailingJSONRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"deskripsi":"SATU"}{"deskripsi":"DUA"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, mastertipesurveyorshttp.CodeMalformedRequest, content["kode"])
}

// DELETE tidak pernah didaftarkan. Rute yang tidak ada tidak dapat dipanggil kode yang
// ditulis kemudian tanpa keputusan sadar.
func TestDeleteNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/1001", "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)

	list, err := p.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 4, "tidak ada baris yang hilang")
}
