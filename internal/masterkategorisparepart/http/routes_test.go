package masterkategorispareparthttp_test

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
	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterkategorisparepart/repo/memory"
	masterkategorisparepartusecase "claim-pnc/internal/masterkategorisparepart/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterkategorispareparthttp "claim-pnc/internal/masterkategorisparepart/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur dasar modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena
// uji lain kebetulan memakai jalur yang benar.
const route = "/api/master/kategori-sparepart"

// Baris contoh pada memory.SampleList.
const (
	approvedID = "1"
	pendingID  = "4"
	rejectedID = "6"
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{})

	service, err := masterkategorisparepartusecase.NewService(
		masterkategorisparepartusecase.Options{
			RepoSelector: func(alias string) (masterkategorisparepart.Store, error) {
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

	handler, err := masterkategorispareparthttp.NewHandler(
		masterkategorispareparthttp.Options{
			Service: service,
			Caller: func(ctx context.Context) (masterkategorispareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return masterkategorispareparthttp.Caller{}, false
				}
				return masterkategorispareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    masterkategorispareparthttp.ErrorWriter(writeError),
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
				masterkategorispareparthttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai kategori dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["kategori_sparepart"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai kategori: %v", content)
	return list
}

// one mengambil satu objek kategori dari badan respons tunggal.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	object, ok := content["kategori_sparepart"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat objek kategori: %v", content)
	return object
}

// violationFields mengambil nama kolom dari `detail` sebuah galat.
func violationFields(t *testing.T, content map[string]any) []string {
	t.Helper()

	detail, ok := content["detail"].([]any)
	require.Truef(t, ok, "badan galat tidak memuat detail: %v", content)

	field := make([]string, 0, len(detail))
	for _, item := range detail {
		row, ok := item.(map[string]any)
		require.True(t, ok)
		field = append(field, row["kolom"].(string))
	}
	return field
}

func validBody() string { return `{"nama_kategori_sparepart":"FINAL DRIVE"}` }

func TestListRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Permintaan tanpa portal DITOLAK, tidak jatuh ke portal utama (TKT-F6-002, R-20).
func TestListRejectsRequestWithoutPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestListRejectsPortalThatIsNotReady(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "SMAS", "")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
}

// Bawaannya tab Approve — tab pertama pada layar lama.
func TestListDefaultsToApproved(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "1", content["status"])
	require.Equal(t, "ASM", content["portal"])
	require.Len(t, rows(t, content), 3)
}

func TestListFiltersByStatus(t *testing.T) {
	p := newTestServer(t)

	for _, one := range []struct {
		status string
		want   int
	}{{"1", 3}, {"0", 2}, {"2", 1}} {
		response, content := p.call(t, http.MethodGet, route+"?status="+one.status, "ASM", "")
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Len(t, rows(t, content), one.want, "tab %q", one.status)
	}
}

func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

func TestListSearchesByName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?cari=hydra", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), 1)
}

// Daftar kosong dikirim sebagai `[]`, bukan `null` — lihat toListDTO.
func TestListReturnsEmptyArrayNotNull(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotNil(t, content["kategori_sparepart"])
	require.Empty(t, rows(t, content))
}

// Entitas lain tidak melihat satu baris pun milik ASM (ADR-0030, R-20).
func TestEntitiesDoNotSeeEachOther(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASI", "")
	require.Empty(t, rows(t, content))
	require.Equal(t, "ASI", content["portal"])

	response, _ := p.call(t, http.MethodGet, route+"/"+approvedID, "ASI", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetReturnsSingleRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := one(t, content)
	require.Equal(t, approvedID, row["id_kategori_sparepart"])
	require.Equal(t, "ENGINE", row["nama_kategori_sparepart"])
	require.Equal(t, "1", row["status"])
	require.Equal(t, "Approve", row["status_label"])
}

func TestGetReportsMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

func TestCreateReturns201WithIssuedKeyAndPendingStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := one(t, content)
	require.Equal(t, "7", row["id_kategori_sparepart"])
	require.Equal(t, "FINAL DRIVE", row["nama_kategori_sparepart"])
	require.Equal(t, "0", row["status"], "baris baru selalu masuk antrean persetujuan")
	require.Equal(t, "Waiting Approval", row["status_label"])
}

func TestCreateRejectsEmptyName(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"nama_kategori_sparepart":"   "}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.Equal(t, []string{"nama_kategori_sparepart"}, violationFields(t, content))
}

// Nama ganda dijawab 409, bukan 422: ia konflik keadaan, bukan isian yang cacat.
func TestCreateRejectsDuplicateNameWith409(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"nama_kategori_sparepart":"hydraulic"}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "kunci_kategori_sparepart_sudah_ada", content["kode"])
	require.Equal(t, []string{"nama_kategori_sparepart"}, violationFields(t, content))
}

// Nama baris yang sudah DITOLAK tetap memblokir, dan pesannya MENJELASKAN sebabnya.
//
// Perilakunya ditiru dari sistem lama (P-5); keterangan pada `detail` yang ditambahkan,
// karena tanpa itu penolakannya tidak dapat dijelaskan dari layar — barisnya tidak terlihat
// di tab Approve maupun Waiting Approval.
func TestCreateRejectsNameOfRejectedRowAndSaysWhy(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"nama_kategori_sparepart":"ATTACHMENT"}`)
	require.Equal(t, http.StatusConflict, response.StatusCode)

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	first := detail[0].(map[string]any)
	require.Contains(t, first["pesan"], "Reject")
}

func TestCreateRejectsUnknownField(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM",
		`{"nama_kategori":"SALAH KETIK"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestCreateRejectsMalformedBody(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Menyimpan mengembalikan baris ke antrean persetujuan, tanpa syarat apa pun.
func TestSaveReturnsRowToPending(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/"+approvedID, "ASM",
		`{"nama_kategori_sparepart":"ENGINE ASSEMBLY"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := one(t, content)
	require.Equal(t, approvedID, row["id_kategori_sparepart"])
	require.Equal(t, "ENGINE ASSEMBLY", row["nama_kategori_sparepart"])
	require.Equal(t, "0", row["status"])
}

func TestSaveAllowsUnchangedName(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodPut, route+"/"+rejectedID, "ASM",
		`{"nama_kategori_sparepart":"ATTACHMENT"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
}

func TestSaveReportsMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/999", "ASM", validBody())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

func TestDecideApprovesSelectedRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM",
		`{"id_kategori_sparepart":["4","5"],"status":"1"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 2, content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])

	_, after := p.call(t, http.MethodGet, route+"?status=0", "ASM", "")
	require.Empty(t, rows(t, after))
}

func TestDecideRejectsSelectedRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM",
		`{"id_kategori_sparepart":["`+pendingID+`"],"status":"2"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 1, content["jumlah_berubah"])
	require.Equal(t, "Reject", content["status_label"])
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM",
		`{"id_kategori_sparepart":[],"status":"1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, []string{"id_kategori_sparepart"}, violationFields(t, content))
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM",
		`{"id_kategori_sparepart":["`+pendingID+`"],"status":"9"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// `/keputusan` tidak boleh terbaca sebagai sebuah ID kategori.
func TestDecisionPathIsNotReadAsKey(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route+"/keputusan", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode,
		"GET /keputusan tidak terdaftar; ia hanya menerima POST")
}

// Tidak ada DELETE terhadap kategori — sistem lama tidak punya, dan D-66 melarangnya.
func TestDeleteIsNotRegistered(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/"+approvedID, "ASM", "")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := masterkategorispareparthttp.NewHandler(masterkategorispareparthttp.Options{})
	require.Error(t, err)
}
