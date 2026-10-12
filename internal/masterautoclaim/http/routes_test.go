package masterautoclaimhttp_test

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
	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterautoclaim/repo/memory"
	masterautoclaimusecase "claim-pnc/internal/masterautoclaim/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterautoclaimhttp "claim-pnc/internal/masterautoclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// loginName adalah pengguna contoh pada provider tiruan.
//
// Ia sengaja SAMA dengan memory.SampleCommittee — dalam huruf yang berbeda — supaya tab
// Komite Approval benar-benar terisi saat diuji. Perbedaan hurufnya bukan kebetulan:
// ia yang membuktikan pencocokan komite mengabaikan besar-kecil huruf, seperti kolom
// OPERATOR_ID yang tersimpan huruf besar sementara pengguna mengetik huruf kecil.
const loginName = "adminpnc"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware
// sesi dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang
// sah ditolak, dan bahwa identitas pemanggil datang dari sesi dan bukan dari badan
// permintaan. Menguji handler secara terpisah tidak dapat membuktikan ketiganya.
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

	autoClaimService, err := masterautoclaimusecase.NewService(masterautoclaimusecase.Options{
		RepoSelector: func(alias string) (masterautoclaim.Store, error) {
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
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterautoclaimhttp.NewHandler(masterautoclaimhttp.Options{
		Service: autoClaimService,
		// Jembatan yang SAMA PERSIS dengan cmd/claimpnc. Menyalin login dari sesi, bukan
		// dari permintaan — itu yang membuat antrean komite orang lain tidak dapat
		// dilihat dengan mengganti satu nilai.
		Caller: func(ctx context.Context) (masterautoclaimhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterautoclaimhttp.Caller{}, false
			}
			return masterautoclaimhttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterautoclaimhttp.ErrorWriter(writeError),
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
				masterautoclaimhttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai auto_claim dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["auto_claim"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai auto_claim: %v", content)
	return list
}

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim?status=1", "ASM", "")
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
		"/api/master/auto-claim?status=1",
		"/api/master/auto-claim/bank",
		"/api/master/auto-claim/sumber-bisnis?cari=contoh",
		"/api/master/auto-claim/client?cari=contoh",
		"/api/master/auto-claim/AGN001",
	} {
		t.Run(path, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, path, "", "")
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "portal_tidak_disebut", content["kode"])
		})
	}
}

func TestUnknownPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim?status=1", "TIDAKADA", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_dikenal", content["kode"])
}

// Keempat tab layar lama adalah empat kombinasi penyaring pada satu endpoint.
func TestListServesTheFourTabs(t *testing.T) {
	p := newTestServer(t)

	for _, c := range []struct {
		tab, query string
		count      int
	}{
		{"Master Auto Klaim", "?status=1", 2},
		{"Waiting Approval", "?status=0", 3},
		{"Komite Approval", "?status=0&komite_saya=true", 2},
		{"Reject", "?status=2", 1},
	} {
		t.Run(c.tab, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, "/api/master/auto-claim"+c.query, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Len(t, rows(t, content), c.count)
			require.Equal(t, "ASM", content["portal"])
		})
	}
}

// Tanpa `status`, daftarnya adalah tab pertama — Master Auto Klaim, APPROVAL="1".
func TestListDefaultsToApproved(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "1", content["status"])
	require.Len(t, rows(t, content), 2)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// Data entitas lain TIDAK terlihat, meski tokennya sah dan portalnya siap.
func TestEntitiesAreSeparate(t *testing.T) {
	p := newTestServer(t)

	_, fromASM := p.call(t, http.MethodGet, "/api/master/auto-claim?status=1", "ASM", "")
	require.NotEmpty(t, rows(t, fromASM))

	_, fromASI := p.call(t, http.MethodGet, "/api/master/auto-claim?status=1", "ASI", "")
	require.Empty(t, rows(t, fromASI))
}

// Baris baru lahir MENUNGGU, ber-CLAIM_ALLOWED "1", dan penyetujunya diisi server.
// Ketiganya tidak pernah datang dari badan permintaan.
func TestCreateDerivesServerSideValues(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/auto-claim", "ASM", `{
		"inisial":"AGN008",
		"nama_penerima":"PT CONTOH ARTHA FINANSIAL",
		"nama_bank":"BANK CONTOH NIAGA",
		"no_rekening":"1000000008",
		"pct_max":"70",
		"pic_lapor":"PIC Contoh Delapan",
		"email_lapor":"pic.delapan@contoh.example",
		"alamat_penerima":"Jalan Contoh Nomor 8",
		"id_client":"CLI007",
		"nama_client":"PT. TERTANGGUNG CONTOH KETUJUH"
	}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, ok := content["auto_claim"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat auto_claim: %v", content)
	require.Equal(t, "0", saved["status"])
	require.Equal(t, "Waiting Approval", saved["status_label"])
	require.Equal(t, "1", saved["claim_allowed"])
	require.Equal(t, memory.SampleCommittee, saved["komite"])
	require.Equal(t, false, saved["dapat_dipakai"], "baris yang belum disetujui tidak boleh dapat dipakai")
}

// Sumber bisnis yang sudah punya baris ditolak 409 — padanan "Data sudah pernah
// diinput." pada Activity/ValidasiAutoClaim.
func TestCreateRejectsDuplicate(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/auto-claim", "ASM", `{
		"inisial":"AGN001",
		"nama_penerima":"MITRA CONTOH SEJAHTERA",
		"nama_bank":"BANK CONTOH NIAGA",
		"no_rekening":"1000000001",
		"pct_max":"100",
		"pic_lapor":"PIC",
		"email_lapor":"a@contoh.example",
		"alamat_penerima":"Jalan Contoh",
		"id_client":"",
		"nama_client":""
	}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "sumber_bisnis_sudah_ada", content["kode"])
}

// Isian yang salah dijawab 422 beserta SELURUH pelanggarannya, masing-masing menyebut
// isiannya sendiri (P-5).
func TestCreateReportsEveryViolation(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/auto-claim", "ASM", `{
		"inisial":"",
		"nama_penerima":"",
		"nama_bank":"",
		"no_rekening":"",
		"pct_max":"",
		"pic_lapor":"",
		"email_lapor":"",
		"alamat_penerima":"",
		"id_client":"",
		"nama_client":""
	}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.Truef(t, ok, "galat validasi tanpa detail: %v", content)
	require.Len(t, detail, 8)

	for _, item := range detail {
		violation, ok := item.(map[string]any)
		require.True(t, ok)
		require.NotEmpty(t, violation["kolom"])
		require.NotEmpty(t, violation["pesan"])
	}
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam: salah ketik nama field
// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong.
func TestCreateRejectsUnknownField(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/auto-claim", "ASM",
		`{"inisial":"AGN008","salah_ketik":"x"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Approve dan Reject memakai endpoint yang SAMA dengan simpan, dibedakan `status` —
// persis UpdateMstAutoClaim_act(stsapprove).
func TestSaveCarriesCommitteeDecision(t *testing.T) {
	for _, c := range []struct {
		name, status, label string
		usable              bool
	}{
		{"approve", "1", "Approve", true},
		{"reject", "2", "Reject", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := newTestServer(t)

			response, content := p.call(t, http.MethodPut, "/api/master/auto-claim/AGN003", "ASM", `{
				"nama_bank":"BANK CONTOH NIAGA",
				"no_rekening":"1000000003",
				"pct_max":"75,5",
				"pic_lapor":"PIC Contoh Tiga",
				"email_lapor":"pic.tiga@contoh.example",
				"alamat_penerima":"Jalan Contoh Nomor 3, Surabaya",
				"id_client":"CLI003",
				"nama_client":"TERTANGGUNG CONTOH KETIGA",
				"status":"`+c.status+`"
			}`)
			require.Equal(t, http.StatusOK, response.StatusCode)

			saved := content["auto_claim"].(map[string]any)
			require.Equal(t, c.status, saved["status"])
			require.Equal(t, c.label, saved["status_label"])
			require.Equal(t, c.usable, saved["dapat_dipakai"])
			// Penyetujunya tidak hilang meski tidak ikut dikirim — lihat
			// usecase.Service.Save.
			require.Equal(t, memory.SampleCommittee, saved["komite"])
		})
	}
}

// Nama penerima tidak dapat diubah lewat penyimpanan. Layar pun tidak mengirimkannya —
// dan bila ia dikirim, permintaannya ditolak sebagai field yang tidak dikenal.
func TestSaveCannotChangeReceiverName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/auto-claim/AGN001", "ASM", `{
		"nama_penerima":"NAMA YANG DICOBA DIGANTI",
		"nama_bank":"BANK CONTOH NIAGA",
		"no_rekening":"1000000001",
		"pct_max":"100",
		"pic_lapor":"PIC Contoh Satu",
		"email_lapor":"pic.satu@contoh.example",
		"alamat_penerima":"Jalan Contoh Nomor 1, Jakarta",
		"id_client":"CLI001",
		"nama_client":"TERTANGGUNG CONTOH PERTAMA",
		"status":"1"
	}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestSaveRejectsMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, "/api/master/auto-claim/TIDAKADA", "ASM", `{
		"nama_bank":"BANK CONTOH NIAGA",
		"no_rekening":"1",
		"pct_max":"10",
		"pic_lapor":"PIC",
		"email_lapor":"a@contoh.example",
		"alamat_penerima":"Jalan Contoh",
		"id_client":"",
		"nama_client":"",
		"status":"1"
	}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Ketiga rute lookup tidak pernah terbaca sebagai sebuah inisial.
func TestLookupRoutesAreNotMistakenForAnInitial(t *testing.T) {
	p := newTestServer(t)

	for _, c := range []struct{ path, key string }{
		{"/api/master/auto-claim/bank", "bank"},
		{"/api/master/auto-claim/sumber-bisnis?cari=contoh", "sumber_bisnis"},
		{"/api/master/auto-claim/client?cari=contoh", "client"},
	} {
		t.Run(c.path, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet, c.path, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Contains(t, content, c.key)
			require.NotContains(t, content, "auto_claim")
		})
	}
}

// Kata kunci terlalu pendek dijawab daftar KOSONG, bukan galat.
func TestLookupShortKeywordIsNotAnError(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim/sumber-bisnis?cari=a", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, content["sumber_bisnis"])
}

func TestGetLoadsSingleRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/auto-claim/AGN001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	found := content["auto_claim"].(map[string]any)
	require.Equal(t, "AGN001", found["inisial"])
	require.Equal(t, "MITRA CONTOH SEJAHTERA", found["nama_penerima"])
	// Client ikut terbaca daftar maupun satu baris — itu yang membuat pengiriman ulang
	// pada Approve tidak dapat menghapusnya.
	require.Equal(t, "CLI001", found["id_client"])
}

// Daftar memuat CLIENTID dan CLIENTNAME meski kueri Pega tidak.
//
// Tanpa keduanya, tombol Approve — yang mengirim ulang seluruh isian — akan mengirim
// client kosong dan MENGHAPUS data client. Uji ini yang menjaga kedua kolom itu tetap
// ada di daftar.
func TestListCarriesClientColumns(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, "/api/master/auto-claim?status=1", "ASM", "")
	for _, item := range rows(t, content) {
		row := item.(map[string]any)
		require.Contains(t, row, "id_client")
		require.Contains(t, row, "nama_client")
	}
}
