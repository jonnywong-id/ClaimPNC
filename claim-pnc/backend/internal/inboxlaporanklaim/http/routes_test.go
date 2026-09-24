package inboxlaporanklaimhttp_test

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
	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/repo/memory"
	inboxlaporanklaimusecase "claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxlaporanklaimhttp "claim-pnc/internal/inboxlaporanklaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji rute modul Inbox Laporan Klaim.
//
// # Kenapa dirakit utuh, bukan handler saja
//
// Yang diuji di sini bukan handler-nya melainkan KONTRAKNYA dari luar: rutenya benar,
// metodenya benar, sesi dan portal ditegakkan, dan badan jawabannya berbentuk seperti
// yang dibaca layar. Menguji handler secara terpisah tidak dapat membuktikan satu pun
// dari itu — dan justru di situlah "tombol yang tampak tidak bekerja" bersembunyi:
// tombolnya benar, permintaannya terkirim, lalu sesuatu di antaranya menolak diam-diam.

const (
	adminLogin  = "adminpnc"
	adminBranch = "1001"
)

type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T, resolver inboxlaporanklaim.BranchResolver) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	fixed := clock.FixedAt(time.Date(2026, 9, 22, 3, 0, 0, 0, time.UTC))

	authService, err := authusecase.NewService(authusecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           fixed,
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	repo := memory.NewRepo(memory.SampleOptions(fixed))

	service, err := inboxlaporanklaimusecase.NewService(inboxlaporanklaimusecase.Options{
		RepoSelector: func(alias string) (inboxlaporanklaim.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		BranchResolver: resolver,
		Clock:          fixed,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Tipe fungsinya ditulis lugas, bukan tipe bernama milik salah satu modul: keempat
	// pemakainya menuntut tipe bernama yang berbeda-beda meski bentuknya sama persis.
	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := inboxlaporanklaimhttp.NewHandler(inboxlaporanklaimhttp.Options{
		Service: service,
		// Jembatan yang sama persis dengan cmd/claimpnc. Bila jembatan ini tidak ditiru,
		// uji akan lulus di sini dan gagal di aplikasi sungguhan.
		Caller: func(ctx context.Context) (inboxlaporanklaim.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxlaporanklaim.Caller{}, false
			}
			return inboxlaporanklaim.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    inboxlaporanklaimhttp.ErrorWriter(writeError),
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
				protected.Use(authhttp.Authenticate(authService, writeError))
				inboxlaporanklaimhttp.Mount(protected, handler, portalDeps)
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

// knownBranch menerjemahkan login petugas uji menjadi cabangnya.
func knownBranch() inboxlaporanklaim.BranchResolver {
	return memory.NewBranchResolver(map[string]string{adminLogin: adminBranch})
}

func TestCreateAnswersWithTheNewReportSoTheScreenCanOpenIt(t *testing.T) {
	// Inilah yang membuat tombol "Buat Baru" berarti: jawabannya harus memuat NOMOR
	// berkas, karena layar memakainya untuk berpindah ke form isian. Jawaban 201 tanpa
	// nomor akan terbaca sebagai tombol yang tidak melakukan apa-apa.
	server := newTestServer(t, knownBranch())

	response, content := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusCreated, response.StatusCode, "badan: %v", content)

	report, ok := content["laporan"].(map[string]any)
	require.True(t, ok, "jawaban tidak memuat laporan: %v", content)

	id, _ := report["id"].(string)
	require.NotEmpty(t, id, "berkas baru tanpa nomor: %v", report)
	require.True(t, strings.HasPrefix(id, "RCVN."), "nomor %q bukan terbitan aplikasi ini", id)

	// Cabangnya terisi dari penerjemahan login — bukan dari profil identitas.
	require.Equal(t, adminBranch, report["kode_cabang"])

	// Dan berkas itu benar-benar dapat dibuka di alamat yang dituju layar.
	follow, detail := server.call(t, http.MethodGet, "/api/inbox/laporan-klaim/"+id, "ASM", "")
	require.Equal(t, http.StatusOK, follow.StatusCode, "badan: %v", detail)
	require.Equal(t, true, detail["dapat_disunting"])
}

func TestCreateWithoutBodyIsAcceptedBecausePegaAsksForNothing(t *testing.T) {
	// `CreateNewCaseRCV` tidak meminta satu pun isian. Menuntut badan permintaan akan
	// membuat tombolnya menjawab 400 pada klien yang benar.
	server := newTestServer(t, knownBranch())

	response, content := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusCreated, response.StatusCode, "badan: %v", content)
}

func TestCreateWithoutPortalIsRefusedNotServedByThePrimary(t *testing.T) {
	// Empat badan hukum, satu basis data masing-masing. Permintaan tanpa portal yang
	// jatuh ke koneksi bawaan berarti menulis berkas satu entitas ke basis data entitas
	// lain tanpa satu pun pesan galat (`R-20`).
	server := newTestServer(t, knownBranch())

	response, _ := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "", "")
	require.NotEqual(t, http.StatusCreated, response.StatusCode)
}

func TestCreateWithoutSessionIsRefused(t *testing.T) {
	server := newTestServer(t, knownBranch())
	server.token = ""

	response, _ := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestCreateIsRefusedWithAReasonWhenTheBranchIsUnknown(t *testing.T) {
	// Petugas yang cabangnya tidak terbaca tidak boleh melihat seluruh cabang (Work
	// Owner, 2026-09-22) — dan berkas yang lahir tanpa cabang tidak akan pernah terlihat
	// siapa pun.
	//
	// Yang diuji di sini BUKAN penolakannya, melainkan bahwa penolakan itu MEMBAWA
	// SEBAB. Penolakan tanpa kode dan pesan adalah persis yang membuat sebuah tombol
	// terbaca sebagai "tidak melakukan apa-apa".
	server := newTestServer(t, memory.NewBranchResolver(map[string]string{}))

	response, content := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.Equal(t, "cabang_tidak_dikenali", content["kode"])
	require.NotEmpty(t, content["pesan"])
}

func TestUnreadableBranchSourceIsReportedAsTemporary(t *testing.T) {
	resolver := memory.NewBranchResolver(map[string]string{adminLogin: adminBranch})
	resolver.SetError(errStub{})

	server := newTestServer(t, resolver)

	response, content := server.call(t, http.MethodPost, "/api/inbox/laporan-klaim", "ASM", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "sumber_cabang_tidak_terbaca", content["kode"])
}

type errStub struct{}

func (errStub) Error() string { return "DB link tidak dapat dihubungi" }

func TestListAnswersWithTheShapeTheScreenReads(t *testing.T) {
	// Layar membaca empat field pada tingkat teratas. Bila salah satu hilang atau
	// berganti nama, tabelnya kosong tanpa satu pun pesan galat — kelas kegagalan yang
	// sudah sekali terjadi di modul ini.
	server := newTestServer(t, knownBranch())

	response, content := server.call(t, http.MethodGet,
		"/api/inbox/laporan-klaim?kategori=semua", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode, "badan: %v", content)

	for _, field := range []string{"laporan", "kategori", "halaman", "batas_cabang"} {
		require.Contains(t, content, field)
	}
	require.Equal(t, adminBranch, content["batas_cabang"])
}
