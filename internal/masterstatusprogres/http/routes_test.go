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

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal
// yang sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()

	progressStatusService, err := masterstatusprogresusecase.NewService(masterstatusprogresusecase.Options{
		RepoSelector: func(alias string) (masterstatusprogres.Repo, error) {
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
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := masterstatusprogreshttp.NewHandler(masterstatusprogreshttp.Options{
		Service:       progressStatusService,
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
				masterstatusprogreshttp.Mount(protected, handler, portalDeps)
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

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request, err := http.NewRequest(method, p.server.URL+path, reader)
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

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	realToken := p.token
	p.token = ""

	response, content := p.call(t, http.MethodGet, "/api/master/status-progres-1", "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])

	p.token = realToken
}

// PERMINTAAN TANPA PORTAL DITOLAK, bukan dilayani portal utama (TKT-F6-002, R-20).
//
// Ini uji terpenting di berkas ini. Bila permintaan tanpa portal jatuh ke koneksi
// default, data satu badan hukum akan terbaca atau tertulis di basis data badan hukum
// lain — dan layarnya tampak normal, karena angkanya masuk akal. Yang salah hanya
// milik siapa data itu.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	t.Run("membaca daftar", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, "/api/master/status-progres-1", "", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	t.Run("menambah", func(t *testing.T) {
		response, content := p.call(t, http.MethodPost, "/api/master/status-progres-1", "",
			`{"nama":"UJI","kode_posisi":"REGISTER"}`)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	t.Run("mengubah", func(t *testing.T) {
		response, content := p.call(t, http.MethodPut, "/api/master/status-progres-1/01", "",
			`{"nama":"UJI","kode_posisi":"REGISTER"}`)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotStated, content["kode"])
	})

	// Dan tidak ada satu baris pun yang berubah di entitas mana pun.
	listASM, err := p.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, listASM, 6)
	listASI, err := p.asi.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, listASI)
}

// "Tidak ada" dibedakan dari "belum tersedia": keduanya menuntut tindak lanjut berbeda.
func TestUnknownAndNotReadyPortalAreDistinguished(t *testing.T) {
	p := newTestServer(t)

	t.Run("tidak ada di daftar", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, "/api/master/status-progres-1", "TIDAKADA", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, portalhttp.CodeUnknown, content["kode"])
	})

	t.Run("ada tetapi koneksinya belum hidup", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, "/api/master/status-progres-1", "SMAS", "")
		require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
		require.Equal(t, portalhttp.CodeNotReady, content["kode"])
	})
}

func TestListNamesAnsweringPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/status-progres-1", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"], "layar harus dapat memastikan data ini milik entitas yang dipilih")

	rows, ok := content["status_progres"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 6)

	first, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "01", first["id"])
	require.Equal(t, "DOKUMEN DITERIMA", first["nama"])
	require.Equal(t, "REGISTER", first["kode_posisi"])
	require.Equal(t, "REGISTER", first["nama_posisi"], "label dikirim bersama kodenya")
}

// Tabel kosong terkirim sebagai [] dan bukan null: layar yang menerima null harus
// menjaganya sendiri, dan satu layar yang lupa akan gagal justru saat tabelnya kosong.
func TestEmptyListSentAsEmptyArray(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/status-progres-1", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+"/api/master/status-progres-1", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASI")
	raw, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = raw.Body.Close() }()

	body := make([]byte, 512)
	n, _ := raw.Body.Read(body)
	require.Contains(t, string(body[:n]), `"status_progres":[]`)
}

func TestCreateReturnsSavedRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-1", "ASI",
		`{"nama":"MENUNGGU BERKAS","kode_posisi":"SURVEY"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	rows, ok := content["status_progres"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "01", rows["id"], "ID diterbitkan server; layar tidak punya cara lain mengetahuinya")
	require.Equal(t, "MENUNGGU BERKAS", rows["nama"])
	require.Equal(t, "SURVEY", rows["nama_posisi"])

	// Tersimpan di ASI, dan HANYA di ASI.
	inASI, err := p.asi.List(t.Context())
	require.NoError(t, err)
	require.Len(t, inASI, 1)
	inASM, err := p.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, inASM, 6)
}

// Seluruh pelanggaran isian dikirim sekaligus, dan status 422 — bukan 400.
func TestValidationFailureSendsAllViolations(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-1", "ASM",
		`{"nama":"","kode_posisi":"999"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeValidationFailed, content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok, "detail per isian wajib ada supaya layar dapat menyorot isian yang salah")
	require.Len(t, detail, 2)

	fields := map[string]bool{}
	for _, d := range detail {
		rows, ok := d.(map[string]any)
		require.True(t, ok)
		fields[rows["kolom"].(string)] = true
	}
	require.True(t, fields["nama"])
	require.True(t, fields["kode_posisi"])
}

func TestUpdateExistingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/status-progres-1/03", "ASM",
		`{"nama":"SURVEI DIJADWALKAN","kode_posisi":"KOMITE"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	rows, ok := content["status_progres"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "03", rows["id"])
	require.Equal(t, "SURVEI DIJADWALKAN", rows["nama"])
	require.Equal(t, "KOMITE", rows["nama_posisi"])
}

func TestUpdateMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/status-progres-1/99", "ASM",
		`{"nama":"APA SAJA","kode_posisi":"REGISTER"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeNotFound, content["kode"])
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa tanda.
func TestUnknownFieldRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-1", "ASM",
		`{"nama":"UJI","kode_posisi":"REGISTER","namaa":"salah ketik"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeMalformedRequest, content["kode"])
}

func TestMalformedBodyRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/status-progres-1", "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, masterstatusprogreshttp.CodeMalformedRequest, content["kode"])
}

// Daftar posisi TIDAK menuntut portal: ia daftar milik aplikasi, bukan isi basis data
// entitas mana pun. Menuntut portal di sini akan membuat dropdown gagal justru saat
// pengguna belum memilih portal.
func TestPositionListDoesNotRequirePortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/posisi-klaim", "", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	position, ok := content["posisi"].([]any)
	require.True(t, ok)
	// Sembilan BARIS, delapan nilai berbeda: "All" memang terulang di layar lama.
	require.Len(t, position, 9)

	first, ok := position[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "All", first["kode"])
	require.Equal(t, "All", first["nama"])
}

// Tetapi ia tetap berada di balik sesi.
func TestPositionListStillRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, "/api/master/posisi-klaim", "", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}
