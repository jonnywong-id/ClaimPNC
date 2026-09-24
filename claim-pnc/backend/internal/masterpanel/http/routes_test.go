package masterpanelhttp_test

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
	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/repo/memory"
	masterpanelusecase "claim-pnc/internal/masterpanel/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterpanelhttp "claim-pnc/internal/masterpanel/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Baris contoh pada memory.SampleList.
const (
	approvedID = "01000001"
	pendingID  = "01000002"
	rejectedID = "01000003"
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

	// asi tidak diberi satu baris master pun; ia yang membuktikan pemisahan antarentitas
	// (ADR-0030, R-20).
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

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{})

	panelService, err := masterpanelusecase.NewService(masterpanelusecase.Options{
		RepoSelector: func(alias string) (masterpanel.Store, error) {
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

	writeResponse := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal, sisanya
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal
	// akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterpanelhttp.NewHandler(masterpanelhttp.Options{
		Service: panelService,
		Caller: func(ctx context.Context) (masterpanelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterpanelhttp.Caller{}, false
			}
			return masterpanelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterpanelhttp.ErrorWriter(writeError),
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
				masterpanelhttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai panel dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["panel"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai panel: %v", content)
	return list
}

// validBody adalah badan permintaan yang lolos seluruh pemeriksaan.
func validBody() string {
	return `{
		"nama_panel":"Kap Mesin",
		"status_repair":"1",
		"status_edit_quantity":"1",
		"status_premium_repair":"0",
		"status_pecah":"0",
		"status_sticker":"1",
		"status_sisi":"-",
		"status_rusak_parah":"0",
		"status_aktif":"1",
		"exclusion_c":"0",
		"lokasi":[{"lokasi_panel":"DEPAN","sisi_panel":"-"}]
	}`
}

// Tanpa sesi, seluruh rute data tertutup.
func TestRoutesRequireSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/panel"},
		{http.MethodGet, "/api/master/panel/" + approvedID},
		{http.MethodPost, "/api/master/panel"},
		{http.MethodPut, "/api/master/panel/" + approvedID},
		{http.MethodPost, "/api/master/panel/keputusan"},
		{http.MethodGet, "/api/master/panel/pilihan"},
	} {
		response, _ := p.call(t, r.method, r.path, "ASM", "")
		require.Equalf(t, http.StatusUnauthorized, response.StatusCode,
			"%s %s harus menuntut sesi", r.method, r.path)
	}
}

// Permintaan tanpa portal DITOLAK, bukan jatuh ke portal utama (TKT-F6-002, R-20).
func TestDataRoutesRequirePortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/panel", "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", content["kode"])
}

// Daftar pilihan TIDAK menuntut portal: isinya konstanta yang ditanam di activity Pega,
// bukan bacaan basis data mana pun. Memasangi pemeriksaan portal padanya akan membuat form
// tidak dapat menggambar dropdown-nya sebelum portal dipilih.
func TestOptionsDoesNotRequirePortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/panel/pilihan", "", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	location, ok := content["lokasi_panel"].([]any)
	require.True(t, ok)
	require.Equal(t, []any{"KIRI", "KANAN", "DEPAN", "BELAKANG", "LAIN-LAIN"}, location)

	side, ok := content["sisi_panel"].([]any)
	require.True(t, ok)
	require.Len(t, side, 3)
}

func TestUnreadyPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/panel", "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "portal_belum_siap", content["kode"])
}

// Dua entitas melihat data yang berbeda. Inilah yang membuktikan pemisahan antarbadan
// hukum bekerja — ASI sengaja tidak diberi satu baris pun.
func TestPortalsSeeDifferentData(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, "/api/master/panel?status=1", "ASM", "")
	require.NotEmpty(t, rows(t, asm))
	require.Equal(t, "ASM", asm["portal"])

	_, asi := p.call(t, http.MethodGet, "/api/master/panel?status=1", "ASI", "")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

// Ketiga tab dilayani satu endpoint dengan penyaring status yang berbeda.
func TestEachTabIsServed(t *testing.T) {
	p := newTestServer(t)

	for status, wanted := range map[string]string{"0": pendingID, "1": approvedID, "2": rejectedID} {
		_, content := p.call(t, http.MethodGet, "/api/master/panel?status="+status, "ASM", "")
		list := rows(t, content)
		require.Lenf(t, list, 1, "tab %q harus memuat satu baris contoh", status)

		first, ok := list[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, wanted, first["id_panel"])
		require.Equal(t, status, content["status"])
	}
}

// Status yang tidak dikenal ditolak 422, bukan dijawab daftar kosong.
func TestUnknownStatusRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/panel?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// Baris daftar membawa lokasinya, dan panel TANPA lokasi terkirim sebagai `[]` — bukan
// null.
func TestListCarriesLocationAndNeverSendsNull(t *testing.T) {
	p := newTestServer(t)

	_, approved := p.call(t, http.MethodGet, "/api/master/panel?status=1", "ASM", "")
	first, ok := rows(t, approved)[0].(map[string]any)
	require.True(t, ok)

	location, ok := first["lokasi"].([]any)
	require.True(t, ok)
	require.Len(t, location, 2)

	one, ok := location[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "KIRI", one["lokasi_panel"])
	require.Equal(t, "1", one["sisi_panel"])
	require.Equal(t, "KIRI", one["sisi_label"], "label dihitung server")

	// 01000003 sengaja tidak punya lokasi sama sekali.
	_, rejected := p.call(t, http.MethodGet, "/api/master/panel?status=2", "ASM", "")
	last, ok := rows(t, rejected)[0].(map[string]any)
	require.True(t, ok)
	empty, ok := last["lokasi"].([]any)
	require.True(t, ok, "panel tanpa lokasi harus terkirim sebagai [] bukan null")
	require.Empty(t, empty)
}

func TestGetUnknownPanelIsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/panel/tidak-ada", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Penambahan menjawab 201 dan membawa kunci yang diterbitkan server.
func TestCreateReturnsIssuedKey(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/panel", "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, ok := content["panel"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "01000004", saved["id_panel"])
	require.Equal(t, "0", saved["status"], "baris baru lahir menunggu persetujuan")
	require.Equal(t, "Waiting Approval", saved["status_label"])
}

// Nama yang sudah dipakai dijawab 409 — konflik KEADAAN, bukan isian yang cacat.
func TestDuplicateNameIsConflict(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), "Kap Mesin", "Pintu Depan", 1)
	response, content := p.call(t, http.MethodPost, "/api/master/panel", "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "nama_panel_sudah_ada", content["kode"])
}

// Seluruh pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
func TestValidationReportsEveryViolationAtOnce(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/panel", "ASM", `{
		"nama_panel":"",
		"status_repair":"",
		"status_edit_quantity":"",
		"status_premium_repair":"",
		"status_pecah":"",
		"status_sticker":"",
		"status_sisi":"",
		"status_rusak_parah":"",
		"status_aktif":"",
		"exclusion_c":"",
		"lokasi":[]
	}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.Len(t, detail, 10, "kesepuluh isian induk wajib")
}

// Pelanggaran pada baris lokasi menyebut INDEKS barisnya, supaya layar dapat menyorot
// baris yang tepat.
func TestLocationViolationNamesItsRow(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(),
		`"lokasi":[{"lokasi_panel":"DEPAN","sisi_panel":"-"}]`,
		`"lokasi":[{"lokasi_panel":"DEPAN","sisi_panel":"-"},{"lokasi_panel":"KIRI","sisi_panel":"9"}]`, 1)

	response, content := p.call(t, http.MethodPost, "/api/master/panel", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	first, ok := detail[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "lokasi.1.sisi_panel", first["kolom"])
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam.
func TestUnknownFieldRejected(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `"nama_panel"`, `"nama_panell"`, 1)
	response, content := p.call(t, http.MethodPost, "/api/master/panel", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Menyimpan SELALU mengembalikan baris ke antrean persetujuan.
func TestSaveReturnsRowToTheQueue(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), "Kap Mesin", "Pintu Depan Diperbarui", 1)
	response, content := p.call(t, http.MethodPut, "/api/master/panel/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved, ok := content["panel"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "0", saved["status"])
	require.Equal(t, approvedID, saved["id_panel"], "kunci tidak berpindah")
}

// Keputusan borongan: satu permintaan untuk seluruh baris yang dicentang.
func TestDecisionMovesChosenRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/panel/keputusan", "ASM",
		`{"id_panel":["`+pendingID+`"],"status":"1","catatan":""}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(1), content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])
}

// Catatan penolakan tersimpan dan terbaca kembali.
func TestRejectionReasonIsStoredAndReadBack(t *testing.T) {
	p := newTestServer(t)

	_, _ = p.call(t, http.MethodPost, "/api/master/panel/keputusan", "ASM",
		`{"id_panel":["`+pendingID+`"],"status":"2","catatan":"Nama tidak baku."}`)

	_, content := p.call(t, http.MethodGet, "/api/master/panel/"+pendingID, "ASM", "")
	saved, ok := content["panel"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Nama tidak baku.", saved["alasan_tolak"])
}

func TestDecisionWithoutSelectionRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/panel/keputusan", "ASM",
		`{"id_panel":[],"status":"1","catatan":""}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
}

// Batas jumlah baris pada satu keputusan dijaga transport, sebelum menyentuh basis data.
func TestDecisionRowLimit(t *testing.T) {
	p := newTestServer(t)

	id := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		id = append(id, `"01000001"`)
	}
	body := `{"id_panel":[` + strings.Join(id, ",") + `],"status":"1","catatan":""}`

	response, content := p.call(t, http.MethodPost, "/api/master/panel/keputusan", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
}

// Tidak ada DELETE terhadap panel: panel yang tidak dipakai ditolak atau ditandai lewat
// STS_AKTIF, bukan dibuang (D-66).
func TestNoDeleteEndpoint(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, "/api/master/panel/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}
