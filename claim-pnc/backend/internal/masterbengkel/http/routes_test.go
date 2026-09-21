package masterbengkelhttp_test

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
	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	masterbengkelusecase "claim-pnc/internal/masterbengkel/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterbengkelhttp "claim-pnc/internal/masterbengkel/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// approvedID dan pendingID adalah baris contoh pada memory.SampleList.
const (
	approvedID = "010000000001"
	pendingID  = "010000000002"
	rejectedID = "010000000003"
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

	// asi tidak diberi satu baris master pun; ia yang membuktikan pemisahan
	// antarentitas (ADR-0030, R-20).
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

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{Banks: memory.SampleBanks()})

	workshopService, err := masterbengkelusecase.NewService(masterbengkelusecase.Options{
		RepoSelector: func(alias string) (masterbengkel.Store, error) {
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

	handler, err := masterbengkelhttp.NewHandler(masterbengkelhttp.Options{
		Service: workshopService,
		Caller: func(ctx context.Context) (masterbengkelhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterbengkelhttp.Caller{}, false
			}
			return masterbengkelhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterbengkelhttp.ErrorWriter(writeError),
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
				masterbengkelhttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai bengkel dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["bengkel"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai bengkel: %v", content)
	return list
}

// validBody adalah badan permintaan yang lolos seluruh pemeriksaan.
func validBody() string {
	return `{
		"nama_bengkel":"Bengkel Contoh Baru",
		"alamat_bengkel":"Jalan Contoh Nomor 9",
		"telp_bengkel":"021-0000009",
		"nohp_bengkel":"0800-0000-009",
		"email":"baru@contoh.example",
		"email_wo":"wo-baru@contoh.example",
		"id_cabang":"001",
		"nama_cabang":"Cabang Contoh Pusat",
		"id_kota":"3171",
		"nama_kota":"Jakarta Pusat",
		"status_rekanan":"1",
		"status_bengkel":"1",
		"alasan_status_bengkel":"Pengajuan baru.",
		"tanggal_status":"01/02/2026",
		"login_aplikasi":"bengkelbaru",
		"id_bank":"002",
		"nama_bank":"Bank Contoh Satu",
		"no_rekening":"9000000009",
		"nama_rekening":"Bengkel Contoh Baru",
		"id_rekening":"ACC-0009",
		"nama_npwp":"Bengkel Contoh Baru",
		"no_npwp":"00.000.000.0-000.009",
		"alamat_npwp":"Jalan Contoh Nomor 9",
		"jenis_pph":"PPh 23",
		"ppn":"11",
		"diskon_jasa":"10",
		"diskon_sparepart":"5",
		"persen_material":"100",
		"pct_selisih_pl":"0",
		"sla":"3",
		"status_disupply_asm":"1",
		"supplier":"",
		"status_eklaim":"1",
		"status_auto_aksep":"0",
		"status_payment":"1",
		"status_autopayment":"0",
		"status_tekno":"1",
		"status_order":"1"
	}`
}

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, content := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])
}

// PERMINTAAN TANPA PORTAL DITOLAK, bukan dilayani portal utama (TKT-F6-002, R-20).
//
// Jatuh ke koneksi default berarti membaca atau menulis data satu badan hukum di basis
// data badan hukum lain tanpa satu pun pesan galat.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/master/bengkel?status=1",
		"/api/master/bengkel/cabang",
		"/api/master/bengkel/kota?cari=jakarta",
		"/api/master/bengkel/bank",
		"/api/master/bengkel/" + approvedID,
	} {
		t.Run(path, func(t *testing.T) {
			response, _ := p.call(t, http.MethodGet, path, "", "")
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
		})
	}

	response, _ := p.call(t, http.MethodPost, "/api/master/bengkel", "", validBody())
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Portal yang ada di daftar tetapi koneksinya belum hidup dijawab berbeda dari portal
// yang tidak dikenal. Keduanya menolak; yang berbeda adalah apa yang dapat dilakukan
// pengguna sesudahnya.
func TestPortalNotReadyRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "SMAS", "")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
}

// Ketiga tab layar adalah tiga nilai status atas satu endpoint.
func TestListPerTab(t *testing.T) {
	p := newTestServer(t)

	for _, c := range []struct {
		status string
		count  int
	}{
		{"1", 1}, // APPROVE
		{"0", 1}, // WAITING APPROVAL
		{"2", 1}, // REJECT
	} {
		t.Run("status="+c.status, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, "/api/master/bengkel?status="+c.status, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Len(t, rows(t, content), c.count)
			require.Equal(t, c.status, content["status"])
			require.Equal(t, "ASM", content["portal"])
		})
	}
}

// Status yang tidak disebut dianggap "1" — tab Approve, tab pertama layar.
func TestListDefaultsToApproved(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, "/api/master/bengkel", "ASM", "")
	require.Equal(t, "1", content["status"])
}

// Status di luar ketiganya ditolak 422, bukan dijawab daftar kosong.
func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/bengkel?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeUnknownStatus, content["kode"])
}

// SATU ENTITAS TIDAK PERNAH MELIHAT DATA ENTITAS LAIN.
//
// ASI tidak diberi satu baris master pun. Bila pemisahan koneksi bocor, daftar ASI akan
// memuat bengkel milik ASM — dan tidak ada apa pun di layar yang menandakannya (R-20).
func TestEntitiesAreSeparated(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "ASM", "")
	require.Len(t, rows(t, asm), 1)

	_, asi := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "ASI", "")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

// Penambahan menjawab 201 dan memuat kunci yang diterbitkan server.
func TestCreate(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel", "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, ok := content["bengkel"].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, saved["id_bengkel"])
	require.Equal(t, "0", saved["status"], "baris baru selalu lahir menunggu")
	require.Equal(t, "Waiting Approval", saved["status_label"])
	require.Equal(t, true, saved["rekanan"])
}

// Nama bengkel yang sudah dipakai dijawab 409 dengan kode tersendiri.
//
// Ia konflik KEADAAN, bukan isian yang cacat — dan layar menanganinya berbeda.
func TestCreateRejectsDuplicateName(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), "Bengkel Contoh Baru", "Bengkel Contoh Utama", 1)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel", "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeNameTaken, content["kode"])
}

// Login aplikasi yang sudah dipakai dijawab 409 dengan kodenya sendiri.
func TestCreateRejectsDuplicateLogin(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `"login_aplikasi":"bengkelbaru"`, `"login_aplikasi":"bengkelcontoh1"`, 1)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel", "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeLoginTaken, content["kode"])
}

// Isian yang tidak sah dijawab 422 beserta SELURUH pelanggarannya sekaligus (P-5).
func TestCreateValidation(t *testing.T) {
	p := newTestServer(t)

	body := strings.NewReplacer(
		`"nama_bengkel":"Bengkel Contoh Baru"`, `"nama_bengkel":""`,
		`"ppn":"11"`, `"ppn":"seratus"`,
	).Replace(validBody())

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeValidationFailed, content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(detail), 2, "seluruh pelanggaran dikirim sekaligus")
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam.
//
// Pada form berisi tiga puluh tiga isian, salah ketik nama field adalah kelas cacat yang
// paling mudah lolos: isiannya akan tersimpan kosong tanpa satu pun tanda.
func TestUnknownFieldRejected(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `"nama_bengkel"`, `"namaBengkel"`, 1)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeMalformedRequest, content["kode"])
}

// Menyimpan MENGEMBALIKAN baris yang sudah disetujui ke antrean persetujuan.
//
// Padanan `Activity/UpdateBengkelHE_act` step 7, yang menetapkan APPROVAL := "0" tanpa
// syarat apa pun.
func TestSaveReturnsRowToPending(t *testing.T) {
	p := newTestServer(t)

	body := strings.NewReplacer(
		`"nama_bengkel":"Bengkel Contoh Baru"`, `"nama_bengkel":"Bengkel Contoh Utama"`,
		`"login_aplikasi":"bengkelbaru"`, `"login_aplikasi":"bengkelcontoh1"`,
		`"alamat_bengkel":"Jalan Contoh Nomor 9"`, `"alamat_bengkel":"Alamat sudah diubah"`,
	).Replace(validBody())

	response, content := p.call(t, http.MethodPut, "/api/master/bengkel/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved := content["bengkel"].(map[string]any)
	require.Equal(t, "0", saved["status"])
	require.Equal(t, approvedID, saved["id_bengkel"], "kunci baris tidak boleh berpindah")
	require.Equal(t, "Alamat sudah diubah", saved["alamat_bengkel"])
}

// Baris yang tidak ada dijawab 404.
func TestSaveMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/bengkel/tidak-ada", "ASM", validBody())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeNotFound, content["kode"])
}

// Keputusan borongan menetapkan status seluruh baris yang dipilih.
func TestDecide(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_bengkel":["` + pendingID + `","` + rejectedID + `"],"status":"1"}`

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel/keputusan", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 2, content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])

	_, list := p.call(t, http.MethodGet, "/api/master/bengkel?status=1", "ASM", "")
	require.Len(t, rows(t, list), 3)
}

// Keputusan tanpa satu pun baris ditolak 422, bukan dijawab "0 berubah".
func TestDecideWithoutRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel/keputusan", "ASM",
		`{"id_bengkel":[],"status":"1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeValidationFailed, content["kode"])
}

// Status yang tidak dikenal pada keputusan ditolak 422.
func TestDecideRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/bengkel/keputusan", "ASM",
		`{"id_bengkel":["`+pendingID+`"],"status":"9"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, masterbengkelhttp.CodeUnknownStatus, content["kode"])
}

// Ruas `/kota`, `/bank`, `/cabang`, dan `/keputusan` TIDAK PERNAH terbaca sebagai ID
// bengkel.
//
// chi mencocokkan segmen statis lebih dulu, tetapi uji ini yang membuktikannya — bukan
// pengetahuan tentang chi.
func TestStaticSegmentsAreNotReadAsKey(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/master/bengkel/cabang",
		"/api/master/bengkel/bank",
		"/api/master/bengkel/kota?cari=jakarta",
	} {
		t.Run(path, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, path, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.NotContains(t, content, "bengkel", "ruas statis terbaca sebagai ID bengkel")
		})
	}
}

// Lookup Kota menolak kata kunci yang terlalu pendek dengan daftar kosong, BUKAN galat.
func TestCityLookupShortKeyword(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/bengkel/kota?cari=j", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, content["kota"])

	_, content = p.call(t, http.MethodGet, "/api/master/bengkel/kota?cari=jakarta", "ASM", "")
	require.NotEmpty(t, content["kota"])
}

// Ketiga lookup menjawab dengan menyebut portal yang melayaninya.
func TestLookupsNameTheirPortal(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/master/bengkel/cabang",
		"/api/master/bengkel/bank",
		"/api/master/bengkel/kota?cari=jakarta",
	} {
		_, content := p.call(t, http.MethodGet, path, "ASM", "")
		require.Equal(t, "ASM", content["portal"], path)
	}
}

// Tidak ada rute DELETE. D-66 melarang penghapusan fisik data bernilai bisnis.
func TestNoDeleteRoute(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, "/api/master/bengkel/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}
