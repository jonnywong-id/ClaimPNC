package daftardetaildokumentravelhttp_test

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
	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/repo/memory"
	daftardetaildokumentravelusecase "claim-pnc/internal/daftardetaildokumentravel/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	daftardetaildokumentravelhttp "claim-pnc/internal/daftardetaildokumentravel/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa data satu entitas tidak
// pernah terbaca lewat entitas lain. Menguji handler secara terpisah tidak dapat
// membuktikan keduanya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi sengaja dibiarkan kosong; ia yang membuktikan pemisahan antarentitas.
	asm *memory.Repo
	asi *memory.Repo

	// plan dipegang supaya jalur GAGALNYA dapat diuji. Kegagalan membaca master plan
	// TIDAK BOLEH menghalangi penyimpanan — nama plan dan jaminan memang boleh diketik
	// sendiri, sehingga yang hilang hanya kenyamanan memilih.
	plan *memory.PlanRepo
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	authService, err := usecase.NewService(usecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           clock.FixedAt(time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()
	document := memory.NewDocumentRepo(memory.SampleDocumentList()...)
	plan := memory.NewPlanRepo(memory.SamplePlanList(), memory.SampleCoverageList())

	perPortal := func(alias string) error {
		if alias != "ASM" && alias != "ASI" {
			return portal.ErrNotReady
		}
		return nil
	}

	service, err := daftardetaildokumentravelusecase.NewService(daftardetaildokumentravelusecase.Options{
		RepoSelector: func(alias string) (daftardetaildokumentravel.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		DocumentSelector: func(alias string) (daftardetaildokumentravel.DocumentRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return document, nil
		},
		PlanSelector: func(alias string) (daftardetaildokumentravel.PlanRepo, error) {
			if err := perPortal(alias); err != nil {
				return nil, err
			}
			return plan, nil
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Keduanya dideklarasikan sebagai tipe fungsi TANPA NAMA, bukan dibiarkan mengambil
	// tipe bernama milik salah satu modul. Setiap modul mendeklarasikan tipe penulisnya
	// sendiri — persis supaya modul tidak saling mengimpor — dan hanya bentuk tanpa nama
	// yang dapat diserahkan ke semuanya.
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

	handler, err := daftardetaildokumentravelhttp.NewHandler(daftardetaildokumentravelhttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    daftardetaildokumentravelhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; sisanya ada di daftar tetapi belum
	// siap. Itulah yang membedakan "tidak ada" dari "belum tersedia".
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
				daftardetaildokumentravelhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi, plan: plan}
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

const route = "/api/master/daftar-detail-dokumen-travel"

func TestDaftarMenyebutEntitasYangMenjawabnya(t *testing.T) {
	// Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
	// diandaikan — server menyebutkannya (`R-20`).
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.NotZero(t, content["total"])
}

func TestEntitasLainTidakMelihatDataEntitasIni(t *testing.T) {
	// Inilah uji yang membenarkan seluruh mekanisme pemilihan portal. Bila ia lulus
	// karena kebetulan, `R-20` tidak terjaga oleh apa pun.
	server := newTestServer(t)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	_, asi := server.call(t, http.MethodGet, route, "ASI", "")

	require.NotZero(t, asm["total"])
	require.Equal(t, float64(0), asi["total"])
}

func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	// Menjatuhkannya ke koneksi bawaan berarti membaca data satu badan hukum lewat
	// permintaan yang tidak menyebutkan badan hukum mana pun (TKT-F6-002).
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestPermintaanTanpaSesiDitolak(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	response, _ := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestDaftarTidakMembawaJaminanTetapiPengambilanSatuBarisMembawanya(t *testing.T) {
	// Perbedaan ini disengaja dan harus tetap begitu: grid tidak menampilkan jaminan,
	// dan menariknya untuk seluruh baris berarti satu kueri yang hasilnya tidak pernah
	// dilihat siapa pun. Layar karena itu WAJIB memuat ulang baris saat dibuka untuk
	// disunting — uji ini yang membuat kewajiban itu terlihat.
	server := newTestServer(t)

	_, list := server.call(t, http.MethodGet, route, "ASM", "")
	rows := list["detail_dokumen_travel"].([]any)
	for _, row := range rows {
		require.Empty(t, row.(map[string]any)["jaminan"])
	}

	_, one := server.call(t, http.MethodGet, route+"/00003", "ASM", "")
	detail := one["detail_dokumen_travel"].(map[string]any)
	require.Len(t, detail["jaminan"], 2)
}

func TestPenambahanMengembalikanBarisTersimpanBesertaIDnya(t *testing.T) {
	// ID diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	server := newTestServer(t)

	body := `{"id_dokumen":"100006","nama_dokumen":"Surat Keterangan Maskapai",` +
		`"status_wajib":true,"minimal_unggah":2,` +
		`"jaminan":[{"id_plan":"TP01","nama_plan":"Travel Plan Silver",` +
		`"id_jaminan":"TC02","nama_jaminan":"Kehilangan Bagasi"}]}`

	response, content := server.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved := content["detail_dokumen_travel"].(map[string]any)
	require.NotEmpty(t, saved["id"])
	require.Equal(t, "100006", saved["id_dokumen"])
	require.Equal(t, true, saved["status_wajib"])
	require.Equal(t, float64(2), saved["minimal_unggah"])
	require.Len(t, saved["jaminan"], 1)
}

func TestIsianKosongTetapDapatDisimpan(t *testing.T) {
	// Work Owner menetapkan 2026-09-21 layar ini tanpa validasi. Uji ini ada supaya
	// ketiadaan validasi menjadi keputusan yang terlihat di tingkat kontrak API juga,
	// bukan hanya di lapisan domain.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route,
		"ASM", `{"id_dokumen":"","nama_dokumen":"","status_wajib":false,"minimal_unggah":0,"jaminan":[]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
}

func TestPenyimpananMenggantiSeluruhDaftarJaminan(t *testing.T) {
	server := newTestServer(t)

	body := `{"id_dokumen":"100004","nama_dokumen":"Laporan Kehilangan Bagasi",` +
		`"status_wajib":false,"minimal_unggah":2,` +
		`"jaminan":[{"id_plan":"TP03","nama_plan":"Travel Plan Platinum",` +
		`"id_jaminan":"TC04","nama_jaminan":"Pembatalan Perjalanan"}]}`

	response, content := server.call(t, http.MethodPut, route+"/00003", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved := content["detail_dokumen_travel"].(map[string]any)
	// Semula dua baris; setelah disimpan tinggal satu — penggantian menyeluruh, bukan
	// penambahan.
	require.Len(t, saved["jaminan"], 1)
}

func TestMengubahBarisYangTidakAdaMenjawab404(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/99999",
		"ASM", `{"id_dokumen":"100001","nama_dokumen":"Paspor","status_wajib":true,"minimal_unggah":1,"jaminan":[]}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "detail_dokumen_travel_tidak_ditemukan", content["kode"])
}

func TestFieldYangTidakDikenalDitolak(t *testing.T) {
	// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan
	// nilai kosong tanpa satu pun tanda bahwa ada yang salah — dan pada modul tanpa
	// validasi, nilai kosong memang akan diterima.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{"nama_dokumn":"Paspor"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestDaftarPilihanDokumenDanPlanDibacaPerEntitas(t *testing.T) {
	server := newTestServer(t)

	response, documents := server.call(t, http.MethodGet, "/api/master/dokumen-travel-pilihan", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotEmpty(t, documents["dokumen"])
	require.Equal(t, "ASM", documents["portal"])

	response, plans := server.call(t, http.MethodGet, "/api/master/plan-travel", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotEmpty(t, plans["plan"])
	require.NotEmpty(t, plans["jaminan"])
}

func TestDaftarPilihanYangGagalTidakMenghalangiPenyimpanan(t *testing.T) {
	// Nama plan dan jaminan memang boleh diketik sendiri, sehingga yang hilang saat
	// daftarnya gagal dimuat hanyalah kenyamanan memilih. Perlakuan yang sama dipakai
	// daftar bisnis pada modul Master COL Simas Online.
	server := newTestServer(t)
	server.plan.SetError(context.DeadlineExceeded)

	response, _ := server.call(t, http.MethodGet, "/api/master/plan-travel", "ASM", "")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusInternalServerError)

	body := `{"id_dokumen":"100001","nama_dokumen":"Paspor","status_wajib":true,"minimal_unggah":1,` +
		`"jaminan":[{"id_plan":"","nama_plan":"Plan yang diketik sendiri","id_jaminan":"","nama_jaminan":"Jaminan yang diketik sendiri"}]}`
	response, content := server.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Len(t, content["detail_dokumen_travel"].(map[string]any)["jaminan"], 1)
}
