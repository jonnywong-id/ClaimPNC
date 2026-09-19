package masterstatusprogreshttp_test

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
	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/repo/memory"
	masterstatusprogresusecase "claim-pnc/internal/masterstatusprogres/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterstatusprogreshttp "claim-pnc/internal/masterstatusprogres/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer2 merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja, alasannya sama dengan tingkat 1: yang diuji bukan
// handler-nya sendiri melainkan KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa
// permintaan tanpa portal yang sah ditolak. Menguji handler terpisah tidak membuktikan
// keduanya.
type testServer2 struct {
	server *httptest.Server
	token  string

	asmParent *memory.Repo
	asm       *memory.Repo2

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
	asi *memory.Repo2
}

func newTestServer2(t *testing.T) *testServer2 {
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

	asmParent := memory.NewRepo(memory.SampleList()...)
	asiParent := memory.NewRepo()
	asm := memory.NewRepo2(asmParent, memory.SampleList2()...)
	asi := memory.NewRepo2(asiParent)

	service, err := masterstatusprogresusecase.NewService2(masterstatusprogresusecase.Options2{
		RepoSelector: func(alias string) (masterstatusprogres.Repo2, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		ParentSelector: func(alias string) (masterstatusprogres.Repo, error) {
			switch alias {
			case "ASM":
				return asmParent, nil
			case "ASI":
				return asiParent, nil
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
	// Dirantai sama seperti cmd/claimpnc. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := masterstatusprogreshttp.NewHandler2(masterstatusprogreshttp.Options2{
		Service:       service,
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
				masterstatusprogreshttp.Mount2(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer2{server: server, asmParent: asmParent, asm: asm, asi: asi}
	p.token = p.login(t)
	return p
}

func (p *testServer2) login(t *testing.T) string {
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
func (p *testServer2) call(t *testing.T, method, path, portalAlias, body string) (*http.Response, map[string]any) {
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

// Seluruh rute berada di balik sesi. Tanpa token, tidak satu pun dapat dibaca.
func TestRoutes2RequireSession(t *testing.T) {
	p := newTestServer2(t)
	noToken := *p
	noToken.token = ""

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/master/status-progres-2", ""},
		{http.MethodGet, "/api/master/status-progres-2/induk", ""},
		{http.MethodPost, "/api/master/status-progres-2", `{"nama":"X","id_induk":"01"}`},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response, _ := noToken.call(t, route.method, route.path, "ASM", route.body)
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		})
	}
}

// SELURUH rute menuntut portal — termasuk daftar induk.
//
// Berbeda dari tingkat 1, yang menyisakan daftar posisi klaim di luar pemeriksaan portal:
// di sini tidak ada satu pun rute yang isinya milik aplikasi. Daftar induk pun dibaca dari
// basis data entitas.
func TestRoutes2RequirePortal(t *testing.T) {
	p := newTestServer2(t)

	for _, path := range []string{
		"/api/master/status-progres-2",
		"/api/master/status-progres-2/induk",
	} {
		t.Run(path, func(t *testing.T) {
			response, _ := p.call(t, http.MethodGet, path, "", "")
			require.Equal(t, http.StatusBadRequest, response.StatusCode,
				"permintaan tanpa portal WAJIB ditolak, tidak pernah jatuh ke portal utama")
		})
	}
}

// Portal yang belum siap ditolak, TIDAK dialihkan ke portal utama sebagai cadangan.
//
// Inilah jalur kegagalan R-20 yang paling berbahaya, karena ia tidak terlihat sebagai
// galat: layarnya tampil normal dan angkanya masuk akal; yang salah hanya milik siapa data
// itu.
func TestRoutes2RejectPortalThatIsNotReady(t *testing.T) {
	p := newTestServer2(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/status-progres-2", "SMAS", "")
	require.GreaterOrEqual(t, response.StatusCode, 400)
	require.NotEqual(t, http.StatusOK, response.StatusCode)

	response, _ = p.call(t, http.MethodGet, "/api/master/status-progres-2", "TIDAKADA", "")
	require.GreaterOrEqual(t, response.StatusCode, 400)
}

// Daftar menjawab isi portal yang diminta, dan MENYEBUT portal yang menjawabnya.
func TestList2IsPerPortal(t *testing.T) {
	p := newTestServer2(t)

	response, content := p.call(t, http.MethodGet, "/api/master/status-progres-2", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.NotEmpty(t, content["status_progres_2"])

	response, content = p.call(t, http.MethodGet, "/api/master/status-progres-2", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASI", content["portal"])
	require.Empty(t, content["status_progres_2"],
		"portal lain tidak boleh melihat baris milik ASM")
}

// Tabel kosong terkirim sebagai `[]`, bukan `null`.
//
// `null` memaksa setiap layar memeriksanya sendiri, dan satu layar yang lupa menampilkan
// galat alih-alih tabel kosong.
func TestEmptyList2IsArrayNotNull(t *testing.T) {
	p := newTestServer2(t)

	response, err := http.NewRequest(http.MethodGet, p.server.URL+"/api/master/status-progres-2", nil)
	require.NoError(t, err)
	response.Header.Set("Authorization", "Bearer "+p.token)
	response.Header.Set(portalhttp.HeaderPortal, "ASI")

	raw, callErr := http.DefaultClient.Do(response)
	require.NoError(t, callErr)
	defer func() { _ = raw.Body.Close() }()

	var body struct {
		ProgressStatus2 *[]any `json:"status_progres_2"`
		Parent          *[]any `json:"induk"`
	}
	require.NoError(t, json.NewDecoder(raw.Body).Decode(&body))
	require.NotNil(t, body.ProgressStatus2, "daftar kosong tetap berupa array")
	require.Empty(t, *body.ProgressStatus2)
}

// Dropdown induk dibaca dari tabel tingkat 1 milik portal yang sama.
func TestParentList2IsPerPortal(t *testing.T) {
	p := newTestServer2(t)

	response, content := p.call(t, http.MethodGet, "/api/master/status-progres-2/induk", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	parent, isList := content["induk"].([]any)
	require.True(t, isList)
	require.NotEmpty(t, parent)

	first, isObject := parent[0].(map[string]any)
	require.True(t, isObject)
	require.Contains(t, first, "id")
	require.Contains(t, first, "nama")
	// Kode posisi milik induk sengaja TIDAK ikut: layar tingkat 2 tidak memakainya, dan
	// mengirim lebih dari yang dipakai membuat kontrak lebih sulit diubah kelak.
	require.NotContains(t, first, "kode_posisi")

	response, content = p.call(t, http.MethodGet, "/api/master/status-progres-2/induk", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, content["induk"])
}

// Penambahan menjawab 201 dengan baris yang benar-benar tersimpan.
//
// ID dan nama induk keduanya diterbitkan server, sehingga layar tidak punya cara lain
// mengetahuinya selain dari badan respons ini.
func TestCreate2ReturnsStoredRow(t *testing.T) {
	p := newTestServer2(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-2", "ASM",
		`{"nama":"  DOKUMEN TAMBAHAN DITERIMA  ","id_induk":"01"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	saved, isObject := content["status_progres_2"].(map[string]any)
	require.True(t, isObject)
	require.NotEmpty(t, saved["id"])
	require.Equal(t, "DOKUMEN TAMBAHAN DITERIMA", saved["nama"], "isian dipangkas lebih dulu")
	require.Equal(t, "01", saved["id_induk"])
	require.NotEmpty(t, saved["nama_induk"], "nama induk disalin server dari barisnya")
	require.Equal(t, "", saved["tipe"], "TIPE tidak pernah ditulis")

	// Barisnya benar-benar masuk ke portal ASM, bukan ke portal lain.
	_, listContent := p.call(t, http.MethodGet, "/api/master/status-progres-2", "ASM", "")
	list, isList := listContent["status_progres_2"].([]any)
	require.True(t, isList)
	require.Len(t, list, len(memory.SampleList2())+1)
}

// Isian yang salah dijawab 422 dengan SELURUH pelanggarannya sekaligus, menempel pada nama
// isian yang sama persis dengan yang dikirim layar.
func TestCreate2ReportsEveryViolation(t *testing.T) {
	p := newTestServer2(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-2", "ASM",
		`{"nama":"","id_induk":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeValidationFailed, content["kode"])

	detail, isList := content["detail"].([]any)
	require.True(t, isList)
	require.Len(t, detail, 2, "pelanggaran dikumpulkan, bukan dihentikan pada yang pertama")

	field := make([]string, 0, len(detail))
	for _, item := range detail {
		violation, isObject := item.(map[string]any)
		require.True(t, isObject)
		field = append(field, violation["kolom"].(string))
		require.NotEmpty(t, violation["pesan"])
	}
	require.ElementsMatch(t, []string{"nama", "id_induk"}, field)
}

// Induk yang sudah tidak ada dijawab 422 pada isian `id_induk`, BUKAN 404 dan bukan 500.
//
// 404 membuat layar tampak kehilangan barisnya sendiri; 500 adalah yang terjadi bila
// galatnya tidak dipetakan sama sekali — dan itulah yang uji ini jaga agar tidak kembali.
func TestCreate2RejectsMissingParent(t *testing.T) {
	p := newTestServer2(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-2", "ASM",
		`{"nama":"APA SAJA","id_induk":"99"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeValidationFailed, content["kode"])

	detail, isList := content["detail"].([]any)
	require.True(t, isList)
	require.Len(t, detail, 1)

	violation := detail[0].(map[string]any)
	require.Equal(t, "id_induk", violation["kolom"],
		"keterangan wajib menempel pada isian induk, bukan menggantung di atas form")
	require.NotEmpty(t, violation["pesan"])
}

// Badan permintaan yang cacat dijawab 400, dan rinciannya tidak pernah ikut terkirim:
// isinya memuat cuplikan badan permintaan.
func TestCreate2RejectsMalformedBody(t *testing.T) {
	p := newTestServer2(t)

	for _, body := range []string{
		`{"nama":`,
		`{"nama":"X","id_induk":"01"}{"nama":"Y","id_induk":"01"}`,
		`{"nama":"X","id_induk":"01","id":"99"}`,
	} {
		t.Run(body, func(t *testing.T) {
			response, content := p.call(t, http.MethodPost, "/api/master/status-progres-2", "ASM", body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, masterstatusprogreshttp.CodeMalformedRequest, content["kode"])
			require.NotContains(t, content["pesan"], "json")
		})
	}
}

// Tidak ada PUT dan tidak ada DELETE, dan itu bukan pekerjaan yang belum selesai.
//
// Sistem lama tidak memiliki satu pun pernyataan yang mengubah isi tabel ini setelah
// barisnya tersimpan. Mendaftarkan rute yang tidak dapat berbuat apa-apa hanya memindahkan
// kejutannya dari layar ke API.
func TestNoEditRoute2Exists(t *testing.T) {
	p := newTestServer2(t)

	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			response, _ := p.call(t, method, "/api/master/status-progres-2/1", "ASM",
				`{"nama":"X","id_induk":"01"}`)
			require.Equal(t, http.StatusNotFound, response.StatusCode)
		})
	}
}
