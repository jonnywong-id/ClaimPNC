package inboxkomunikasicabanghttp_test

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
	authusecase "claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/repo/memory"
	inboxkomunikasicabangusecase "claim-pnc/internal/inboxkomunikasicabang/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxkomunikasicabanghttp "claim-pnc/internal/inboxkomunikasicabang/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji rute modul Inbox Komunikasi Cabang.
//
// # KENAPA BERKAS INI ADA — dan kenapa ketiadaannya sempat mahal
//
// Modul ini sempat punya 40 uji yang seluruhnya lulus sementara DUA TOMBOL yang ada di layar
// lama tidak terpasang sama sekali. Tidak satu pun uji menangkapnya, dan sebabnya jelas:
// seluruhnya menguji apa yang DIBANGUN, tidak satu pun menguji apa yang SAMPAI KE LAYAR.
//
// Bentuk grid layar ini — termasuk kedua kolom tombolnya — ditetapkan PELADEN dan dikirim
// lewat `/tab`. Layar menggambar tombol hanya bila kolomnya ada di jawaban itu. Jadi uji
// domain yang membuktikan kolomnya terdaftar TIDAK membuktikan tombolnya muncul; yang
// membuktikannya adalah jawaban HTTP-nya sendiri.
//
// Itu pula yang membuat binary lama berakibat fatal di layar ini: bundel SPA yang baru tetap
// tidak menggambar tombol apa pun bila peladen yang melayaninya belum mengenal kolomnya.
const (
	branchLogin = "pictekniks"
	otherLogin  = "adminpnc"
)

type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T, login string) *testServer {
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

	store := memory.NewSampleStore()

	service, err := inboxkomunikasicabangusecase.NewService(
		inboxkomunikasicabangusecase.Options{
			RepoSelector: func(alias string) (inboxkomunikasicabang.Repo, error) {
				if alias != "ASM" {
					return nil, portal.ErrNotReady
				}
				return store, nil
			},
			BranchResolver: memory.NewSampleBranchResolver(),
		})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(
		&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	// Jembatan pemanggil ditiru SAMA PERSIS dengan cmd/claimpnc. Bila ia tidak ditiru, uji
	// akan lulus di sini dan gagal di aplikasi sungguhan — dan di layar ini akibatnya bukan
	// galat melainkan batas cabang yang tidak dapat diturunkan.
	handler := inboxkomunikasicabanghttp.NewHandler(inboxkomunikasicabanghttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (inboxkomunikasicabanghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxkomunikasicabanghttp.Caller{}, false
			}
			return inboxkomunikasicabanghttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           writeResponse,
		FallbackErrorWriter: inboxkomunikasicabanghttp.ErrorWriter(writeError),
	})

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
				inboxkomunikasicabanghttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server}
	p.token = p.login(t, login)
	return p
}

func (p *testServer) login(t *testing.T, login string) string {
	t.Helper()

	body := strings.NewReader(
		`{"nama_pengguna":"` + login + `","kata_sandi":"rahasia123"}`)
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
	t *testing.T, method, path, portalAlias string,
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

// columnKeys membaca kunci kolom sebuah tab dari jawaban JSON.
func columnKeys(tab map[string]any) []string {
	raw, _ := tab["kolom"].([]any)
	keys := make([]string, 0, len(raw))
	for _, item := range raw {
		column, _ := item.(map[string]any)
		key, _ := column["kunci"].(string)
		keys = append(keys, key)
	}
	return keys
}

func TestMetadataSendsBothActionColumnsForEveryTab(t *testing.T) {
	// INILAH uji yang seharusnya ada sejak awal.
	//
	// Layar menggambar tombol hanya bila kolomnya ada di jawaban ini. Uji domain yang
	// membuktikan kolomnya terdaftar di `tab.go` TIDAK membuktikan ia sampai ke layar —
	// dan justru di celah itulah kedua tombol sempat hilang tanpa satu pun uji gagal.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang/tab", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	tabs, _ := body["tab"].([]any)
	require.Len(t, tabs, 2)

	for _, item := range tabs {
		tab, _ := item.(map[string]any)
		keys := columnKeys(tab)

		require.Containsf(t, keys, inboxkomunikasicabang.FieldActionDetail,
			"tab %v tidak mengirim kolom tombol Detail Komunikasi", tab["nama"])
		require.Containsf(t, keys, inboxkomunikasicabang.FieldActionFinish,
			"tab %v tidak mengirim kolom tombol Selesai Komunikasi", tab["nama"])

		// Urutannya pun diuji: keduanya paling belakang, Detail lebih dulu — persis urutan
		// di section. Kolom tombol yang berpindah ke tengah akan menggeser seluruh kolom
		// isian tanpa satu pun galat.
		require.Equal(t,
			[]string{
				inboxkomunikasicabang.FieldActionDetail,
				inboxkomunikasicabang.FieldActionFinish,
			},
			keys[len(keys)-2:])
	}
}

func TestMetadataKeepsTheLiteralButtonHeading(t *testing.T) {
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang/tab", "ASM")
	tabs, _ := body["tab"].([]any)

	for _, item := range tabs {
		tab, _ := item.(map[string]any)
		raw, _ := tab["kolom"].([]any)

		for _, entry := range raw {
			column, _ := entry.(map[string]any)
			key, _ := column["kunci"].(string)
			if !inboxkomunikasicabang.IsAction(key) {
				continue
			}
			require.Equal(t, "Button", column["judul"])
		}
	}
}

func TestListAnswersWithTheTabShapeSoTheScreenDrawsTheSameColumns(t *testing.T) {
	// Layar menggambar kolomnya dari `tab` pada jawaban DAFTAR, bukan dari metadata yang
	// diambil sekali. Bila keduanya berbeda, tombolnya muncul lalu hilang saat berpindah
	// halaman — dan itu tidak menghasilkan satu pun galat.
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	tab, _ := body["tab"].(map[string]any)
	keys := columnKeys(tab)

	require.Contains(t, keys, inboxkomunikasicabang.FieldActionDetail)
	require.Contains(t, keys, inboxkomunikasicabang.FieldActionFinish)
}

func TestListCarriesTheConversationNumberEachButtonNeeds(t *testing.T) {
	// Kedua tombol mengirim nomor percakapan — `KOMID = .ClaimNo` di Pega. Baris yang tidak
	// membawanya menghasilkan tombol yang tampak normal lalu menembak alamat kosong.
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	rows, _ := body["baris"].([]any)
	require.NotEmpty(t, rows, "penyimpanan contoh harus memberi baris bagi petugas cabang 1001")

	for _, item := range rows {
		row, _ := item.(map[string]any)
		require.NotEmpty(t, row["komunikasi"], "setiap baris wajib membawa nomor percakapan")
	}
}

func TestFinishActionIsAnsweredWithAReasonNotSilence(t *testing.T) {
	// Tombol "Selesai Komunikasi" menembak rute ini. Jawaban 404 akan membuat tombolnya
	// terbaca sebagai kerusakan; 501 menyatakan yang sebenarnya.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodPost,
		"/api/inbox-komunikasi-cabang/tindakan?tindakan=selesai-komunikasi&komunikasi=KOM-0001",
		"ASM")

	require.Equal(t, http.StatusNotImplemented, response.StatusCode)
	require.Equal(t, "belum_tersedia", body["kode"])
	require.Contains(t, body["pesan"], "Pega")
}

func TestDetailIsReachableWithTheNumberTheButtonSends(t *testing.T) {
	// Tombol "Detail Komunikasi" mengirim nomor percakapan apa adanya. Rute ini yang
	// menerimanya — dan bila jalurnya tidak cocok, tombolnya akan tampak tidak bekerja.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0001", "ASM")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "KOM-0001", body["komunikasi"])

	messages, _ := body["pesan"].([]any)
	require.NotEmpty(t, messages)
}

func TestDetailOfAnotherBranchIsRefused(t *testing.T) {
	// KOM-0010 milik cabang 1003. Petugas cabang 1001 tidak boleh membukanya lewat nomornya,
	// meski tombolnya tidak pernah muncul untuk baris itu — tombol bukan penjagaan.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0010", "ASM")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "komunikasi_tidak_ditemukan", body["kode"])
}

func TestEveryRouteRefusesARequestWithoutAPortal(t *testing.T) {
	// Jatuh ke portal utama berarti menampilkan percakapan satu badan hukum kepada petugas
	// badan hukum lain tanpa satu pun pesan galat (`R-20`).
	server := newTestServer(t, branchLogin)

	for _, path := range []string{
		"/api/inbox-komunikasi-cabang/tab",
		"/api/inbox-komunikasi-cabang",
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0001",
		"/api/inbox-komunikasi-cabang/ekspor",
	} {
		response, _ := server.call(t, http.MethodGet, path, "")
		require.NotEqualf(t, http.StatusOK, response.StatusCode,
			"%s dilayani tanpa header portal", path)
	}
}

func TestHeadOfficeStaffSeesHeadOfficeConversations(t *testing.T) {
	// `adminpnc` dipetakan ke cabang kantor pusat di penyimpanan contoh, sehingga jalur
	// kantor pusat punya saksi yang cabangnya BENAR-BENAR terbaca.
	server := newTestServer(t, otherLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	branch, _ := body["batas_cabang"].(map[string]any)
	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, branch["kode"])
	require.Equal(t, true, branch["kantor_pusat"])
	require.Equal(t, true, branch["terbaca"])
}

func TestExportGoesThroughTheSameBranchBoundaryAsTheList(t *testing.T) {
	// Batas yang berlaku pada daftar tetapi tidak pada unduhan bukan batas sama sekali — ia
	// hanya menyulitkan orang yang patuh.
	server := newTestServer(t, branchLogin)

	request, err := http.NewRequest(http.MethodGet,
		server.server.URL+"/api/inbox-komunikasi-cabang/ekspor?tab=1", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+server.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Disposition"), "cabang-1001")
}
