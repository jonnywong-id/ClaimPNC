package menuhttp_test

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
	"claim-pnc/internal/menu/repo/memory"
	menuusecase "claim-pnc/internal/menu/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"

	authhttp "claim-pnc/internal/auth/http"
	menuhttp "claim-pnc/internal/menu/http"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware
// sesi dan jembatan konteks pemanggil.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa menu yang dikirim memang
// milik login yang bersangkutan.
type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T, repo *memory.Repo) *testServer {
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

	menuService, err := menuusecase.NewService(menuusecase.Options{Repo: repo})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	writeJSON := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	writeError := authhttp.WriteError(logger)

	handler, err := menuhttp.NewHandler(menuhttp.Options{
		Service: menuService,
		Caller: func(ctx context.Context) (menuhttp.Caller, bool) {
			base, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return menuhttp.Caller{}, false
			}
			return menuhttp.Caller{Login: base.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeJSON,
		WriteError:    menuhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	server := httptest.NewServer(httpserver.Router(httpserver.Deps{
		Logger: logger,
		MountAPI: func(api chi.Router) {
			authhttp.Mount(api, authhttp.NewHandler(authService, logger), authService, logger)
			api.Group(func(protected chi.Router) {
				protected.Use(authhttp.Authenticate(authService, writeError))
				menuhttp.Mount(protected, handler)
			})
		},
	}))
	t.Cleanup(server.Close)

	s := &testServer{server: server}
	s.token = s.login(t)
	return s
}

func (s *testServer) login(t *testing.T) string {
	t.Helper()

	body := strings.NewReader(`{"nama_pengguna":"adminpnc","kata_sandi":"rahasia123"}`)
	response, err := http.Post(s.server.URL+"/api/masuk", "application/json", body)
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

func (s *testServer) getMenu(t *testing.T) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/menu", nil)
	require.NoError(t, err)
	if s.token != "" {
		request.Header.Set("Authorization", "Bearer "+s.token)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// devRepo memberi login contoh `adminpnc` keanggotaan group IT, sehingga menunya
// terisi seperti pengguna sungguhan.
func devRepo() *memory.Repo { return memory.NewDevRepo() }

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh satu
// baris pun.
func TestWithoutSessionRejected(t *testing.T) {
	s := newTestServer(t, devRepo())
	s.token = ""

	response, content := s.getMenu(t)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])
}

func TestMenuIsGroupedAndOrderedBySequence(t *testing.T) {
	s := newTestServer(t, devRepo())

	response, content := s.getMenu(t)
	require.Equal(t, http.StatusOK, response.StatusCode)

	groups, ok := content["menu"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 3, "group IT memegang MASTER, INBOX, dan VIEW")

	var names []string
	for _, g := range groups {
		names = append(names, g.(map[string]any)["nama"].(string))
	}
	require.Equal(t, []string{"MASTER", "INBOX", "VIEW"}, names)
}

// Submenu tersusun di bawah induknya lewat MENU_ID_LEADER, dan urutannya mengikuti
// MENU_SEQUENCE — bukan urutan MENU_ID.
func TestSubmenuIsNestedUnderItsLeader(t *testing.T) {
	s := newTestServer(t, devRepo())

	_, content := s.getMenu(t)
	master := content["menu"].([]any)[0].(map[string]any)
	require.Equal(t, "MASTER", master["nama"])

	children := master["submenu"].([]any)
	require.NotEmpty(t, children)

	first := children[0].(map[string]any)
	require.Equal(t, "Master Status Klaim", first["nama"])
	require.Equal(t, "StatusClaimInbox", first["program"])

	second := children[1].(map[string]any)
	require.Equal(t, "Master Rekening", second["nama"])
}

// Nama harness dikirim apa adanya. Frontend yang memutuskan layarnya sudah ada atau
// belum, dan ia hanya dapat melakukannya bila namanya ikut dikirim.
func TestEveryItemCarriesItsProgram(t *testing.T) {
	s := newTestServer(t, devRepo())

	_, content := s.getMenu(t)
	program := map[string]string{}
	for _, g := range content["menu"].([]any) {
		for _, c := range g.(map[string]any)["submenu"].([]any) {
			item := c.(map[string]any)
			program[item["nama"].(string)] = item["program"].(string)
		}
	}

	require.Equal(t, "StatusProgress", program["Master Status Progress 1"])
	require.Equal(t, "InboxRegister_Harness", program["My Inbox"])
}

// Pengguna tanpa satu pun izin menerima senarai KOSONG, bukan null dan bukan galat.
// Layar yang menerima null harus menjaganya sendiri, dan satu layar yang lupa akan
// gagal justru pada pengguna yang paling sedikit haknya.
func TestLoginWithoutGrantGetsEmptyArray(t *testing.T) {
	s := newTestServer(t, memory.NewSampleRepo())

	response, content := s.getMenu(t)
	require.Equal(t, http.StatusOK, response.StatusCode)

	menu, ok := content["menu"].([]any)
	require.True(t, ok, "menu harus berupa senarai, bukan null")
	require.Empty(t, menu)
}

// Kelompok yang seluruh anaknya tersaring tidak boleh tersisa sebagai judul kosong.
func TestNoGroupIsReturnedWithoutChildren(t *testing.T) {
	s := newTestServer(t, devRepo())

	_, content := s.getMenu(t)
	for _, g := range content["menu"].([]any) {
		group := g.(map[string]any)
		require.NotEmpty(t, group["submenu"], "kelompok %q tampil tanpa satu pun anak", group["nama"])
	}
}
