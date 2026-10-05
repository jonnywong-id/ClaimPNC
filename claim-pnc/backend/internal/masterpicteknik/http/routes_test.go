package masterpicteknikhttp_test

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
	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterpicteknik/directory"
	"claim-pnc/internal/masterpicteknik/repo/memory"
	masterpicteknikusecase "claim-pnc/internal/masterpicteknik/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterpicteknikhttp "claim-pnc/internal/masterpicteknik/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/master/pic-teknik"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang
// sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
type testServer struct {
	server    *httptest.Server
	token     string
	directory *directory.Fake

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
	employeeDirectory := directory.NewFake(directory.SampleEmployees()...)

	picTeknikService, err := masterpicteknikusecase.NewService(masterpicteknikusecase.Options{
		RepoSelector: func(alias string) (masterpicteknik.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Directory: employeeDirectory,
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

	handler, err := masterpicteknikhttp.NewHandler(masterpicteknikhttp.Options{
		Service:       picTeknikService,
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
				masterpicteknikhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi, directory: employeeDirectory}
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
// daftar petugas satu badan hukum akan terbaca atau tertulis di basis data badan hukum
// lain — dan layarnya tampak normal, karena isinya masuk akal. Yang salah hanya milik
// siapa data itu.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"membaca daftar", http.MethodGet, route, ""},
		{"membaca satu baris", http.MethodGet, route + "/PICTEKNIK01", ""},
		{"mencari di direktori", http.MethodGet, route + "/direktori/PICTEKNIK01", ""},
		{"menambah", http.MethodPost, route, `{"id_operator":"PICTEKNIK05","email":"a@b.co","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`},
		{"mengubah", http.MethodPut, route + "/PICTEKNIK01", `{"id_operator":"","email":"a@b.co","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			response, content := p.call(t, c.method, c.path, "", c.body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, portalhttp.CodeNotStated, content["kode"])
		})
	}
}

func TestUnreadyPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "SMAS", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// Entitas yang menjawab disebut pada SETIAP jawaban, bukan diandaikan sama dengan yang
// diminta.
func TestResponseNamesAnsweringPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
}

// Entitas yang kosong menjawab kosong — bukan meminjam isi entitas lain.
func TestPortalsAreIsolated(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(0), content["total"])
	require.Equal(t, "ASI", content["portal"])
}

// ── Daftar dan ambil ────────────────────────────────────────────────────────────

// Daftar hanya memuat petugas AKTIF, meniru `STS_AKTIF = '1'` pada Report Definition lama.
// Daftar memuat petugas aktif MAUPUN nonaktif, sama dengan grid Pega yang menampilkan
// Status Aktif `0` dan `1` berdampingan.
func TestListIncludesInactiveTechnicians(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	rows, _ := content["pic_teknik"].([]any)
	require.NotEmpty(t, rows)

	status := map[string]bool{}
	for _, item := range rows {
		row, _ := item.(map[string]any)
		id, _ := row["id_operator"].(string)
		aktif, _ := row["aktif"].(bool)
		status[id] = aktif
	}
	require.True(t, status["PICTEKNIK01"])
	require.Contains(t, status, "PICTEKNIK04", "petugas nonaktif harus ikut terkirim")
	require.False(t, status["PICTEKNIK04"])
}

// Beban kerja ikut dikirim pada daftar: ia kolom view yang ditampilkan grid Pega.
func TestListCarriesWorkload(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM", "")
	rows, _ := content["pic_teknik"].([]any)
	require.NotEmpty(t, rows)

	first, _ := rows[0].(map[string]any)
	require.Contains(t, first, "beban_kerja")
}

// Petugas nonaktif tidak muncul di daftar, tetapi TETAP dapat dibuka lewat ID — itulah
// yang membuatnya dapat diaktifkan kembali.
func TestGetReachesInactiveTechnician(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/PICTEKNIK04", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	row, _ := content["pic_teknik"].(map[string]any)
	require.Equal(t, false, row["aktif"])
}

func TestGetUnknownReturns404(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/TIDAKADA", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeNotFound, content["kode"])
}

// Segmen "direktori" tidak boleh tertangkap sebagai sebuah ID operator.
func TestDirectoryRouteIsNotShadowedByIDRoute(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/direktori/PICTEKNIK02", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	employee, _ := content["pegawai"].(map[string]any)
	require.Equal(t, "Contoh Adjuster Madya", employee["nama"])
	// Atasan ikut terjawab dalam satu pencarian — blok EmpLeader pada respons direktori.
	require.Equal(t, "PICTEKNIK01", employee["atasan"])
	require.Equal(t, "Contoh Kepala Teknik", employee["nama_atasan"])
}

// ── Menambah dan mengubah ───────────────────────────────────────────────────────

func TestCreateReturns201AndDerivesName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"id_operator":"PICTEKNIK05","email":"baru@example.invalid","bisnis":"NONMBU","kelompok":"A","atasan":"","counter_klaim_kurang_1m":8,"counter_klaim_lebih_1m":0,"aktif":true}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row, _ := content["pic_teknik"].(map[string]any)
	require.Equal(t, "Contoh Petugas Baru", row["nama"])
	require.Equal(t, "PICTEKNIK01", row["atasan"])
}

func TestCreateDuplicateReturns409(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"id_operator":"PICTEKNIK01","email":"lagi@example.invalid","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeAlreadyExists, content["kode"])
}

// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan bisnis.
func TestCreateInvalidReturns422WithFieldDetail(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"id_operator":"PICTEKNIK05","email":"bukan-email","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":-1,"counter_klaim_lebih_1m":0,"aktif":true}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeValidationFailed, content["kode"])

	detail, _ := content["detail"].([]any)
	require.NotEmpty(t, detail)

	found := map[string]bool{}
	for _, item := range detail {
		row, _ := item.(map[string]any)
		field, _ := row["field"].(string)
		found[field] = true
	}
	require.True(t, found[masterpicteknik.FieldEmail])
	require.True(t, found[masterpicteknik.FieldClaimCounterBelow1M])
}

// Petugas yang tidak dikenal direktori TETAP tersimpan — Pega pun begitu.
func TestCreateUnknownEmployeeStillStored(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPost, route, "ASM",
		`{"id_operator":"TIDAKTERDAFTAR","email":"x@example.invalid","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`)

	require.Equal(t, http.StatusCreated, response.StatusCode)
}

// Direktori yang mati pun tidak menahan penyimpanan.
func TestCreateStoresDespiteDirectoryFailure(t *testing.T) {
	p := newTestServer(t)
	p.directory.SetError(masterpicteknik.ErrDirectoryNotConfigured)

	response, _ := p.call(t, http.MethodPost, route, "ASM",
		`{"id_operator":"PICTEKNIK05","email":"baru@example.invalid","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`)

	require.Equal(t, http.StatusCreated, response.StatusCode)
}

// Jalur PENCARIAN tetap melaporkan gangguan apa adanya — hasilnya diperlihatkan ke
// pengguna, jadi diamnya akan menyesatkan.
//
// ID yang dipakai sengaja TIDAK ada di master: bila ada, namanya terjawab dari master dan
// direktori tidak pernah dipanggil.
func TestDirectoryOutageReturns503(t *testing.T) {
	p := newTestServer(t)
	p.directory.SetError(masterpicteknik.ErrDirectoryUnreachable)

	response, content := p.call(t, http.MethodGet, route+"/direktori/PICTEKNIK05", "ASM", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeDirectoryUnreachable, content["kode"])
}

// Pencarian dijawab dari master lebih dulu, tanpa menyentuh direktori sama sekali.
func TestLookupAnswersFromMaster(t *testing.T) {
	p := newTestServer(t)
	p.directory.SetError(masterpicteknik.ErrDirectoryUnreachable)

	response, content := p.call(t, http.MethodGet, route+"/direktori/PICTEKNIK01", "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	employee, _ := content["pegawai"].(map[string]any)
	require.NotEmpty(t, employee["nama"])
}

func TestUpdateReturns200(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/PICTEKNIK02", "ASM",
		`{"id_operator":"","email":"adjuster.baru@example.invalid","bisnis":"NONMBU","kelompok":"C","atasan":"PICTEKNIK01","counter_klaim_kurang_1m":25,"counter_klaim_lebih_1m":4,"aktif":true}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row, _ := content["pic_teknik"].(map[string]any)
	require.Equal(t, "C", row["kelompok"])
	require.Equal(t, float64(25), row["counter_klaim_kurang_1m"])
	// ID diambil dari jalur URL, bukan dari badan permintaan yang sengaja dikosongkan.
	require.Equal(t, "PICTEKNIK02", row["id_operator"])
}

func TestUpdateUnknownReturns404(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/TIDAKADA", "ASM",
		`{"id_operator":"","email":"a@example.invalid","bisnis":"","kelompok":"","atasan":"","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"aktif":true}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeNotFound, content["kode"])
}

// ── Badan permintaan ────────────────────────────────────────────────────────────

// Tiga nilai yang tidak dikelola layar ini DITOLAK bila dikirim klien, bukan diabaikan
// diam-diam: nama dimiliki direktori, grup panel tidak pernah ditulis, dan beban kerja
// dihitung.
func TestUnknownFieldsRejected(t *testing.T) {
	p := newTestServer(t)

	for _, body := range []string{
		`{"id_operator":"PICTEKNIK05","email":"a@b.co","nama":"Dikarang","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"bisnis":"","kelompok":"","atasan":"","aktif":true}`,
		`{"id_operator":"PICTEKNIK05","email":"a@b.co","grup_panel":"X","counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"bisnis":"","kelompok":"","atasan":"","aktif":true}`,
		`{"id_operator":"PICTEKNIK05","email":"a@b.co","beban_kerja":99,"counter_klaim_kurang_1m":1,"counter_klaim_lebih_1m":0,"bisnis":"","kelompok":"","atasan":"","aktif":true}`,
	} {
		response, content := p.call(t, http.MethodPost, route, "ASM", body)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
		require.Equal(t, masterpicteknikhttp.CodeMalformedRequest, content["kode"])
	}
}

func TestMalformedBodyRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, masterpicteknikhttp.CodeMalformedRequest, content["kode"])
}

// Menghapus tidak pernah didaftarkan — layar Pega pun tidak punya tombolnya, dan
// menghapus petugas memutus rujukan penugasan pada klaim lama.
func TestDeleteNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/PICTEKNIK01", "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

// ID operator yang berupa ALAMAT SUREL dapat dibuka dan diubah.
//
// Sebagian OPERATOR_ID di master ini memang alamat surel, sehingga `@` menjadi `%40` di
// jalur URL. `chi.URLParam` mengembalikannya tanpa diurai — ia merutekan memakai
// `r.URL.RawPath`, yang justru terisi ketika jalurnya bersandi persen.
//
// Gejalanya menyesatkan: baris yang tampil di daftar dijawab "PIC teknik tidak ditemukan".
func TestOperatorIDWithAtSignIsDecoded(t *testing.T) {
	p := newTestServer(t)

	const operatorID = "PETUGAS89@CONTOH.INVALID"
	_, err := p.asm.Insert(context.Background(), masterpicteknik.Technician{
		OperatorID: operatorID,
		Email:      "petugas89@contoh.invalid",
		Active:     true,
	})
	require.NoError(t, err)

	// Disandikan seperti encodeURIComponent di peramban, yang MENYANDIKAN @ menjadi %40.
	// url.PathEscape tidak melakukannya karena @ sah di potongan jalur — tetapi yang
	// menentukan di sini adalah apa yang benar-benar dikirim layar.
	escaped := strings.ReplaceAll(operatorID, "@", "%40")

	response, _ := p.call(t, http.MethodGet, route+"/"+escaped, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode, "membuka baris ber-@ tidak boleh 404")

	response, _ = p.call(t, http.MethodPut, route+"/"+escaped, "ASM",
		`{"id_operator":"`+operatorID+`","email":"petugas89@contoh.invalid","bisnis":"NONMBU","kelompok":"B","atasan":"","counter_klaim_kurang_1m":3,"counter_klaim_lebih_1m":2,"aktif":true}`)
	require.Equal(t, http.StatusOK, response.StatusCode, "mengubah baris ber-@ tidak boleh 404")
}
