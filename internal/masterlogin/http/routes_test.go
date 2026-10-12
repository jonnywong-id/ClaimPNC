package masterloginhttp_test

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
	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/repo/memory"
	masterloginusecase "claim-pnc/internal/masterlogin/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterloginhttp "claim-pnc/internal/masterlogin/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur dasar modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena
// uji lain kebetulan memakai jalur yang benar.
const route = "/api/master/login"

// Baris contoh pada memory.SampleList.
const (
	leaderKey = "BudiHartono"

	// Perhatikan bentuknya: "Rina Ayu Lestari" tanpa spasi, bukan "RinaAyu". Kuncinya
	// DITURUNKAN dari nama lengkapnya, dan menuliskannya dengan tangan di sini adalah cara
	// termudah salah — persis seperti yang terjadi saat berkas ini pertama ditulis.
	memberKey = "RinaAyuLestari"

	noTeamKey  = "DewiKartika"
	sampleRows = 6
)

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
	// (ADR-0030, R-20). Pada modul ini pemisahan itu lebih berarti daripada pada master
	// penggolongan: yang bocor bukan sekadar data, melainkan orang yang muncul di daftar
	// entitas yang bukan haknya.
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

	service, err := masterloginusecase.NewService(
		masterloginusecase.Options{
			RepoSelector: func(alias string) (masterlogin.Repo, error) {
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

	handler, err := masterloginhttp.NewHandler(
		masterloginhttp.Options{
			Service: service,
			Caller: func(ctx context.Context) (masterloginhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterloginhttp.Caller{}, false
				}
				return masterloginhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    masterloginhttp.ErrorWriter(writeError),
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
				masterloginhttp.Mount(protected, handler, portalDeps)
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
	method, path, portalAlias, body string,
) (*http.Response, map[string]any) {
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

// rows mengambil senarai login dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["login_surveyor"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai login: %v", content)
	return list
}

// one mengambil satu objek login dari badan respons tunggal.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	object, ok := content["login_surveyor"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat satu login: %v", content)
	return object
}

// saveBody menyusun badan permintaan simpan.
func saveBody(nama, email, telp, alamat string) string {
	body, _ := json.Marshal(map[string]string{
		"nama":   nama,
		"email":  email,
		"telp":   telp,
		"alamat": alamat,
	})
	return string(body)
}

// TestListRequiresSession membuktikan rutenya benar-benar berada di balik middleware sesi.
func TestListRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestListRequiresPortal membuktikan permintaan tanpa portal DITOLAK, bukan dilayani portal
// utama (TKT-F6-002, R-20).
func TestListRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.NotEmpty(t, content["kode"])
}

// TestListReturnsEveryRow membuktikan daftar memuat seluruh baris entitas itu.
func TestListReturnsEveryRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), sampleRows)
	require.Equal(t, "ASM", content["portal"])
}

// TestListIsSeparatePerPortal membuktikan satu entitas tidak pernah melihat data entitas
// lain — dan bahwa daftar kosong tetap berupa senarai, bukan null.
func TestListIsSeparatePerPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, rows(t, content))
	require.Equal(t, "ASI", content["portal"])
}

// TestListSearchNarrowsResult membuktikan penyaring `cari` bekerja lewat HTTP.
func TestListSearchNarrowsResult(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?cari=rina", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), 1)
}

// TestGetReturnsEverySevenColumn membuktikan ketujuh kolom dikirim ke klien — termasuk dua
// yang TIDAK pernah digambar di layar Pega.
//
// Keduanya dikirim karena tanpa itu tidak ada cara apa pun mengetahui sebuah baris bertaut
// ke tim siapa dan berperan apa. Lihat SurveyorLoginDTO.
func TestGetReturnsEverySevenColumn(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/"+memberKey, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	found := one(t, content)
	require.Equal(t, "Rina Ayu Lestari", found["nama"])
	require.Equal(t, memberKey, found["login"])
	require.NotEmpty(t, found["email"])
	require.NotEmpty(t, found["telp"])
	require.NotEmpty(t, found["alamat"])
	require.Equal(t, masterlogin.LoginStatusMember, found["status_login"])
	require.Equal(t, leaderKey, found["login_leader"])
}

// TestGetMissingRowIsNotFound membuktikan baris yang tidak ada menjawab 404.
func TestGetMissingRowIsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/TidakAda", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// TestCreateDerivesLoginAndDerivedColumns membuktikan ketiga kolom turunan diterbitkan
// server dan dikembalikan ke layar.
//
// Pemanggilnya `adminpnc`, yang TIDAK punya baris di tabel ini — sehingga LOGINLEADER-nya
// kosong. Itu perilaku Pega apa adanya; lihat usecase.Service.Create.
func TestCreateDerivesLoginAndDerivedColumns(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		saveBody("Sari Dewi-Anggraini", "sari@contoh.invalid", "021-777", "Jl. Baru"))
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved := one(t, content)
	require.Equal(t, "SariDewiAnggraini", saved["login"])
	require.Equal(t, masterlogin.LoginStatusMember, saved["status_login"])
	require.Equal(t, "", saved["login_leader"])
}

// TestCreateRejectsDuplicateLogin membuktikan login ganda menjawab 409 dengan kode yang
// dapat dibedakan klien.
//
// "Budi.Hartono" menghasilkan LOGIN yang sama persis dengan "Budi Hartono" yang sudah ada —
// bentuk bentrok yang paling mengejutkan di modul ini.
func TestCreateRejectsDuplicateLogin(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		saveBody("Budi.Hartono", "x@contoh.invalid", "021", ""))
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "kunci_login_surveyor_sudah_ada", content["kode"])
}

// TestCreateReportsEveryViolationAtOnce membuktikan SELURUH pelanggaran isian dikirim
// bersamaan sebagai 422 (P-5).
func TestCreateReportsEveryViolationAtOnce(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", saveBody("", "", "", ""))
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.Truef(t, ok, "respons tidak memuat detail pelanggaran: %v", content)
	require.Len(t, detail, 3)
}

// TestCreateRejectsDerivedColumnsInBody membuktikan ketiga kolom turunan TIDAK dapat
// dikirim klien.
//
// Menerimanya berarti membuka jalan menetapkan kunci baris, peran, dan induk tim lewat
// permintaan HTTP biasa. `DisallowUnknownFields` menolaknya sebagai permintaan cacat — bukan
// mengabaikannya diam-diam, supaya cacat pada klien terlihat saat pertama dicoba.
func TestCreateRejectsDerivedColumnsInBody(t *testing.T) {
	p := newTestServer(t)

	for _, field := range []string{"login", "status_login", "login_leader"} {
		t.Run(field, func(t *testing.T) {
			body := `{"nama":"Zaki","email":"z@contoh.invalid","telp":"021","alamat":"",` +
				`"` + field + `":"apa saja"}`
			response, content := p.call(t, http.MethodPost, route, "ASM", body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "permintaan_cacat", content["kode"])
		})
	}
}

// TestSaveChangesOnlyContactColumns membuktikan penyimpanan mengubah Email, Telp, dan
// Alamat — dan TIDAK menyentuh ketiga kolom turunan.
func TestSaveChangesOnlyContactColumns(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/"+memberKey, "ASM",
		saveBody("Rina Ayu Lestari", "rina.baru@contoh.invalid", "0899-1", "Jl. Pindah"))
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved := one(t, content)
	require.Equal(t, "rina.baru@contoh.invalid", saved["email"])
	require.Equal(t, "0899-1", saved["telp"])
	require.Equal(t, "Jl. Pindah", saved["alamat"])

	require.Equal(t, memberKey, saved["login"])
	require.Equal(t, masterlogin.LoginStatusMember, saved["status_login"])
	require.Equal(t, leaderKey, saved["login_leader"])
}

// TestSaveRejectsChangedName membuktikan penguncian Nama ditegakkan DI SERVER, bukan hanya
// di layar.
//
// Penguncian di antarmuka adalah kenyamanan tampilan; permintaan yang tidak datang dari
// layar itu tidak tersentuh olehnya. Lihat masterlogin.ErrNameLocked.
func TestSaveRejectsChangedName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/"+memberKey, "ASM",
		saveBody("Nama Yang Berbeda", "rina@contoh.invalid", "021", ""))
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "nama_login_surveyor_terkunci", content["kode"])
}

// TestSaveMissingRowIsNotFound membuktikan penyimpanan atas baris yang sudah tidak ada
// menjawab 404, bukan diam-diam berhasil.
func TestSaveMissingRowIsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/TidakAda", "ASM",
		saveBody("Siapa Saja", "x@contoh.invalid", "021", ""))
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// TestCreatedRowIsVisibleOnlyInItsOwnPortal membuktikan penambahan di satu entitas tidak
// pernah muncul di entitas lain (R-20).
func TestCreatedRowIsVisibleOnlyInItsOwnPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPost, route, "ASI",
		saveBody("Hanya Di ASI", "asi@contoh.invalid", "021", ""))
	require.Equal(t, http.StatusCreated, response.StatusCode)

	_, asi := p.call(t, http.MethodGet, route, "ASI", "")
	require.Len(t, rows(t, asi), 1)

	_, asm := p.call(t, http.MethodGet, route, "ASM", "")
	require.Len(t, rows(t, asm), sampleRows)
}

// TestUnreadyPortalIsRejected membuktikan portal yang ada di daftar tetapi koneksinya belum
// hidup DITOLAK dengan kode tersendiri — bukan dijawab daftar kosong.
//
// Keduanya sangat berbeda artinya, dan mudah tertukar: daftar kosong berarti "entitas ini
// belum punya login surveyor", sedangkan penolakan berarti "kami belum dapat menjawab".
func TestUnreadyPortalIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "SMAS", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
	require.NotEmpty(t, content["kode"])
}

// TestNoDeleteRoute membuktikan modul ini tidak punya jalur hapus.
//
// Tidak satu pun rule di export menghapus baris tabel ini, dan D-66 melarang penghapusan
// fisik data bernilai bisnis. Uji ini menjaga ketiadaan itu tetap DISENGAJA — bila kelak
// seseorang menambahkannya, ia harus memutuskannya sebagai perubahan.
func TestNoDeleteRoute(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/"+noTeamKey, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}
