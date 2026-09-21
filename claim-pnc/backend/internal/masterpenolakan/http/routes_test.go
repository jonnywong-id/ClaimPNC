package masterpenolakanhttp_test

import (
	"bytes"
	"context"
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
	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
	masterpenolakanusecase "claim-pnc/internal/masterpenolakan/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterpenolakanhttp "claim-pnc/internal/masterpenolakan/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji bukan handler-nya sendiri melainkan KONTRAKNYA —
// bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah ditolak, dan
// bahwa kolom pelaku terisi dari sesi. Menguji handler terpisah tidak membuktikan satu pun
// dari ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	asm       *memory.Repo
	asmKomite *memory.RepoKomite

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
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

	asm := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	asi := memory.NewRepo(nil)
	asmKomite := memory.NewRepoKomite(memory.SampleListKomite()...)
	asiKomite := memory.NewRepoKomite()

	service, err := masterpenolakanusecase.NewService(masterpenolakanusecase.Options{
		RepoSelector: func(alias string) (masterpenolakan.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)

	komiteService, err := masterpenolakanusecase.NewServiceKomite(masterpenolakanusecase.OptionsKomite{
		RepoSelector: func(alias string) (masterpenolakan.RepoKomite, error) {
			switch alias {
			case "ASM":
				return asmKomite, nil
			case "ASI":
				return asiKomite, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	writeResponse := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterpenolakanhttp.NewHandler(masterpenolakanhttp.Options{
		Service: service,
		Komite:  komiteService,
		Caller: func(ctx context.Context) (masterpenolakanhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpenolakanhttp.Caller{}, false
			}
			return masterpenolakanhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterpenolakanhttp.ErrorWriter(writeError),
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
				protected.Use(authhttp.Authenticate(authService, authhttp.ErrorWriter(writeError)))
				masterpenolakanhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asmKomite: asmKomite, asi: asi}
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

// route menyebut seluruh rute modul ini beserta badan yang sah untuknya.
//
// Dipakai uji sesi dan uji portal, supaya rute yang kelak ditambahkan tidak luput dari
// keduanya hanya karena lupa disalin ke salah satunya.
func routes() []struct {
	method string
	path   string
	body   string
} {
	return []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/master/penolakan-klaim", ""},
		{http.MethodGet, "/api/master/penolakan-klaim/status-1", ""},
		{http.MethodPost, "/api/master/penolakan-klaim", `{"nama":"X","id_status_1":"1","nama_status_1":""}`},
		{http.MethodPut, "/api/master/penolakan-klaim/1", `{"nama":"X","id_status_1":"1","nama_status_1":""}`},
		{http.MethodGet, "/api/master/penolakan-komite", ""},
		{http.MethodPost, "/api/master/penolakan-komite", `{"catatan":"X"}`},
		{http.MethodPut, "/api/master/penolakan-komite/111", `{"catatan":"X"}`},
	}
}

// Seluruh rute berada di balik sesi. Tanpa token, tidak satu pun dapat dibaca.
func TestRoutesRequireSession(t *testing.T) {
	p := newTestServer(t)
	noToken := *p
	noToken.token = ""

	for _, route := range routes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response, _ := noToken.call(t, route.method, route.path, "ASM", route.body)
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		})
	}
}

// SELURUH rute menuntut portal — termasuk daftar Status Penolakan 1.
//
// Tidak ada satu pun rute di modul ini yang isinya milik aplikasi; bahkan daftar pilihan
// pun dibaca dari basis data entitas.
func TestRoutesRequirePortal(t *testing.T) {
	p := newTestServer(t)

	for _, route := range routes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			response, content := p.call(t, route.method, route.path, "", route.body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode,
				"permintaan tanpa portal WAJIB ditolak, tidak pernah jatuh ke portal utama")
			require.Equal(t, "portal_tidak_disebut", content["kode"])
		})
	}
}

// Portal yang tidak dikenal ditolak, TIDAK dialihkan ke portal utama sebagai cadangan.
//
// Inilah jalur kegagalan R-20 yang paling berbahaya, karena ia tidak terlihat sebagai
// galat: layarnya tampil normal dan angkanya masuk akal; yang salah hanya milik siapa data
// itu.
func TestUnknownPortalIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/penolakan-klaim", "TIDAKADA", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_dikenal", content["kode"])
}

// Data satu entitas tidak pernah terlihat dari entitas lain.
func TestEachPortalAnswersWithItsOwnRows(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, "/api/master/penolakan-klaim", "ASM", "")
	require.NotEmpty(t, asm["penolakan_klaim"])
	require.Equal(t, "ASM", asm["portal"], "respons menyebut entitas yang menjawabnya")

	_, asi := p.call(t, http.MethodGet, "/api/master/penolakan-klaim", "ASI", "")
	require.Empty(t, asi["penolakan_klaim"])
	require.Equal(t, "ASI", asi["portal"])
}

// Daftar kosong terkirim sebagai `[]`, bukan `null`.
//
// Layar yang menerima `null` harus menjaganya sendiri, dan satu layar yang lupa akan gagal
// saat tabelnya masih kosong.
func TestEmptyListIsSentAsArray(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/penolakan-klaim", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, isArray := content["penolakan_klaim"].([]any)
	require.True(t, isArray, "daftar kosong harus berupa senarai, bukan null")
	require.Empty(t, list)
}

// Seluruh pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
func TestValidationReportsEveryViolationAtOnce(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-klaim", "ASM",
		`{"nama":"","id_status_1":"","nama_status_1":""}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode,
		"422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis")
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, isArray := content["detail"].([]any)
	require.True(t, isArray)
	require.Len(t, detail, 2)
}

// Status Penolakan 1 yang sudah tidak ada adalah ISIAN yang salah, bukan sumber daya yang
// hilang — 422 dengan keterangan menempel di isiannya, bukan 404.
func TestMissingParentIsReportedOnTheField(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-klaim", "ASM",
		`{"nama":"X","id_status_1":"999","nama_status_1":""}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail, isArray := content["detail"].([]any)
	require.True(t, isArray)
	require.Len(t, detail, 1)

	first, isObject := detail[0].(map[string]any)
	require.True(t, isObject)
	require.Equal(t, "id_status_1", first["kolom"],
		"nama isiannya sama persis dengan field JSON yang dikirim layar")
}

// Penambahan mengembalikan baris yang benar-benar tersimpan, beserta nilai yang hanya
// diketahui server.
func TestCreateReturnsTheStoredRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-klaim", "ASM",
		`{"nama":"KLAIM DI LUAR WILAYAH","id_status_1":"1","nama_status_1":""}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, isObject := content["penolakan_klaim"].(map[string]any)
	require.True(t, isObject)
	require.NotEmpty(t, saved["id"], "ID diterbitkan server")
	require.Equal(t, "0", saved["status"], "baris baru selalu menunggu persetujuan")
	require.Equal(t, "MENUNGGU", saved["status_label"])
	require.Equal(t, "POLIS TIDAK BERLAKU", saved["nama_status_1"],
		"nama induk disalin server dari baris induknya, bukan dikirim layar")
}

// Kolom pelaku diisi dari SESI, tidak pernah dari badan permintaan.
//
// Ia satu-satunya jejak pertanggungjawaban yang dimiliki tabel ini; menerimanya dari
// peramban berarti siapa pun dapat mengaku sebagai orang lain.
func TestSubmitterComesFromSession(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodPost, "/api/master/penolakan-klaim", "ASM",
		`{"nama":"X","id_status_1":"1","nama_status_1":""}`)

	saved := content["penolakan_klaim"].(map[string]any)
	require.Equal(t, "adminpnc", saved["diajukan_oleh"])
	require.NotEmpty(t, saved["diajukan_pada"])
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam.
//
// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah.
func TestUnknownFieldIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-klaim", "ASM",
		`{"nama":"X","id_status_1":"1","namaa_status_1":"salah ketik"}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Pengubahan MENGEMBALIKAN baris ke antrean persetujuan, dan jejak persetujuan lamanya
// tetap terbaca.
func TestUpdateSendsRowBackToApprovalQueue(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/penolakan-klaim/1", "ASM",
		`{"nama":"TEKS BARU","id_status_1":"1","nama_status_1":""}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved := content["penolakan_klaim"].(map[string]any)
	require.Equal(t, "0", saved["status"])
	require.Equal(t, "TEKS BARU", saved["nama"])
	require.NotEmpty(t, saved["disetujui_oleh"],
		"jejak keputusan yang PERNAH ada dibiarkan, persis seperti procedure lama")
}

func TestUpdateOnMissingRowIsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/penolakan-klaim/999", "ASM",
		`{"nama":"X","id_status_1":"1","nama_status_1":""}`)

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Tab kedua — Penolakan Komite — dilayani rute yang sama sekali terpisah.
func TestCommitteeTabListsItsOwnTable(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/penolakan-komite", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, isArray := content["penolakan_komite"].([]any)
	require.True(t, isArray)
	require.Len(t, list, 3)
}

func TestCommitteeCreateIssuesTheID(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-komite", "ASM",
		`{"catatan":"CATATAN BARU"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved := content["penolakan_komite"].(map[string]any)
	require.Equal(t, "114", saved["id"])
	require.Equal(t, "CATATAN BARU", saved["catatan"])
}

func TestCommitteeEmptyNoteIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/penolakan-komite", "ASM", `{"catatan":"   "}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
}

func TestCommitteeUpdateOnMissingRowSaysWhichMaster(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/penolakan-komite/999", "ASM", `{"catatan":"X"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
	require.Contains(t, content["pesan"], "Penolakan komite",
		"pesannya menyebut master yang benar; kedua tab hidup di satu layar")
}

// Tidak ada DELETE pada tab mana pun.
//
// Mendaftarkan rute yang tidak dapat berbuat apa-apa hanya memindahkan kejutannya dari
// layar ke API.
func TestDeleteIsNotRouted(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/master/penolakan-klaim/1",
		"/api/master/penolakan-komite/111",
	} {
		t.Run(path, func(t *testing.T) {
			response, _ := p.call(t, http.MethodDelete, path, "ASM", "")
			require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
		})
	}
}
