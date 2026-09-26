package mastertipespareparthttp_test

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
	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesparepart/repo/memory"
	mastertipesparepartusecase "claim-pnc/internal/mastertipesparepart/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastertipespareparthttp "claim-pnc/internal/mastertipesparepart/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur dasar modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena
// uji lain kebetulan memakai jalur yang benar.
const route = "/api/master/tipe-sparepart"

// Baris contoh pada memory.SampleList.
const (
	approvedID = "1" // FUEL FILTER, kategori ENGINE
	pendingID  = "4" // TRACK ROLLER
	rejectedID = "6" // CONTROL VALVE
	orphanID   = "7" // SPROCKET, kategori "99" yang tidak ada
)

// approvedCategory adalah kunci kategori yang ADA dan disetujui pada contoh — HYDRAULIC.
const approvedCategory = "2"

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

	service, err := mastertipesparepartusecase.NewService(
		mastertipesparepartusecase.Options{
			RepoSelector: func(alias string) (mastertipesparepart.Store, error) {
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

	handler, err := mastertipespareparthttp.NewHandler(
		mastertipespareparthttp.Options{
			Service: service,
			Caller: func(ctx context.Context) (mastertipespareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastertipespareparthttp.Caller{}, false
				}
				return mastertipespareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    mastertipespareparthttp.ErrorWriter(writeError),
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
				mastertipespareparthttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai tipe dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["tipe_sparepart"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai tipe: %v", content)
	return list
}

// one mengambil satu objek tipe dari badan respons tunggal.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	object, ok := content["tipe_sparepart"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat objek tipe: %v", content)
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

func validBody() string {
	return `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"` +
		approvedCategory + `"}`
}

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

func TestListRejectsPortalWithoutConnection(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "SMAS", "")
	require.GreaterOrEqual(t, response.StatusCode, http.StatusBadRequest)
}

// Bawaan penyaringnya "1" — tab Approve, tab pertama pada layar lama.
func TestListDefaultsToApprovedTab(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "1", content["status"])
	require.Equal(t, "ASM", content["portal"])
	require.Len(t, rows(t, content), 4)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
	require.Equal(t, []string{"status"}, violationFields(t, content))
}

// Setiap baris membawa nama kategori induknya — inilah yang di Pega datang dari JOIN.
func TestListCarriesCategoryName(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route+"?status=0", "ASM", "")
	for _, item := range rows(t, content) {
		row := item.(map[string]any)
		require.Equal(t, "UNDERCARRIAGE", row["nama_kategori_sparepart"])
	}
}

// Baris yatim TETAP terkirim, dengan nama kategori kosong.
//
// Di Pega baris ini HILANG dari daftar karena inner join-nya. Selisih perilaku ini
// disengaja; lihat banner pada berkas .sql.
func TestListKeepsOrphanRowWithEmptyCategoryName(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM", "")

	var orphan map[string]any
	for _, item := range rows(t, content) {
		row := item.(map[string]any)
		if row["id_tipe_sparepart"] == orphanID {
			orphan = row
		}
	}
	require.NotNil(t, orphan, "baris yatim wajib tetap terkirim")
	require.Equal(t, "", orphan["nama_kategori_sparepart"])
}

// Satu entitas tidak pernah melihat data entitas lain (ADR-0030, R-20).
func TestListIsolatesEachPortal(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, route, "ASM", "")
	require.NotEmpty(t, rows(t, asm))

	_, asi := p.call(t, http.MethodGet, route, "ASI", "")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

// Daftar kosong dikirim sebagai `[]`, bukan `null`.
func TestListSendsEmptyArrayNotNull(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+route, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASI")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	raw := make([]byte, 4096)
	size, _ := response.Body.Read(raw)
	require.Contains(t, string(raw[:size]), `"tipe_sparepart":[]`)
}

func TestGetReturnsSingleRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := one(t, content)
	require.Equal(t, approvedID, row["id_tipe_sparepart"])
	require.Equal(t, "FUEL FILTER", row["nama_tipe_sparepart"])
	require.Equal(t, "ENGINE", row["nama_kategori_sparepart"])
	require.Equal(t, "Approve", row["status_label"])
}

func TestGetReturnsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/404", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// `/pilihan` tidak pernah terbaca sebagai sebuah ID tipe.
func TestChoicesIsNotReadAsAnID(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route+"/pilihan", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	category, ok := content["kategori"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai kategori: %v", content)
	require.Len(t, category, 3)
	require.Equal(t, false, content["terpotong"])
	require.Equal(t, "ASM", content["portal"])

	first := category[0].(map[string]any)
	require.Equal(t, "ENGINE", first["nama"], "daftar diurutkan menurut nama")
	require.NotEmpty(t, first["kode"])
}

// `/pilihan` ikut dipasangi pemeriksaan portal: isinya dibaca dari basis data entitas.
func TestChoicesRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route+"/pilihan", "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestCreateStoresPendingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := one(t, content)
	require.NotEmpty(t, row["id_tipe_sparepart"])
	require.Equal(t, "SWING MOTOR", row["nama_tipe_sparepart"])
	require.Equal(t, approvedCategory, row["id_kategori_sparepart"])
	require.Equal(t, "HYDRAULIC", row["nama_kategori_sparepart"])
	require.Equal(t, "0", row["status"])
	require.Equal(t, "Waiting Approval", row["status_label"])
}

// Kedua isian wajib dilaporkan SEKALIGUS, bukan satu per satu (P-5).
func TestCreateReportsEveryViolationAtOnce(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route, "ASM", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.ElementsMatch(t,
		[]string{"nama_tipe_sparepart", "id_kategori_sparepart"},
		violationFields(t, content))
}

// Nama ganda dijawab 409 dengan kode tersendiri — ia konflik keadaan, bukan isian cacat.
func TestCreateRejectsDuplicateName(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"fuel filter","id_kategori_sparepart":"` +
		approvedCategory + `"}`
	response, content := p.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "kunci_tipe_sparepart_sudah_ada", content["kode"])

	// Pesannya menyebut KEDUA kemungkinan yang tidak terlihat dari tab yang sedang dibuka.
	detail := content["detail"].([]any)[0].(map[string]any)
	require.Contains(t, detail["pesan"], "kategori yang")
	require.Contains(t, detail["pesan"], "Reject")
}

// Kategori yang tidak ada dijawab 409 dengan kode tersendiri.
func TestCreateRejectsUnknownCategory(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"404"}`
	response, content := p.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "kategori_sparepart_tidak_ditemukan", content["kode"])
	require.Equal(t, []string{"id_kategori_sparepart"}, violationFields(t, content))
}

// Field yang tidak dikenal DITOLAK, bukan diabaikan diam-diam.
func TestCreateRejectsUnknownField(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"2","warna":"merah"}`
	response, content := p.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Status tidak dapat diselundupkan lewat jalur simpan — ia bukan field yang dikenal.
func TestCreateRejectsStatusInBody(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"2","status":"1"}`
	response, content := p.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Nama kategori tidak dapat dikirim klien — ia milik tabel kategori dan tidak pernah
// ditulis modul ini.
func TestCreateRejectsCategoryNameInBody(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"SWING MOTOR","id_kategori_sparepart":"2",` +
		`"nama_kategori_sparepart":"KARANGAN"}`
	response, _ := p.call(t, http.MethodPost, route, "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Menyimpan MENGEMBALIKAN baris ke antrean persetujuan.
func TestSaveReturnsRowToPending(t *testing.T) {
	p := newTestServer(t)

	body := `{"nama_tipe_sparepart":"FUEL FILTER","id_kategori_sparepart":"` +
		approvedCategory + `"}`
	response, content := p.call(t, http.MethodPut, route+"/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := one(t, content)
	require.Equal(t, "0", row["status"])
	require.Equal(t, approvedCategory, row["id_kategori_sparepart"],
		"tipe dapat dipindahkan ke kategori lain")
}

func TestSaveRejectsMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut, route+"/404", "ASM", validBody())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// `/keputusan` tidak pernah terbaca sebagai sebuah ID tipe.
func TestDecisionPathIsNotReadAsAnID(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_tipe_sparepart":["` + pendingID + `"],"status":"1"}`
	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 1, content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])
}

func TestDecisionRejectsEmptySelection(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM",
		`{"id_tipe_sparepart":[],"status":"1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, []string{"id_tipe_sparepart"}, violationFields(t, content))
}

func TestDecisionRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_tipe_sparepart":["` + pendingID + `"],"status":"9"}`
	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// Batas borongan dijaga transport, bukan basis data.
func TestDecisionRejectsTooManyRows(t *testing.T) {
	p := newTestServer(t)

	id := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		id = append(id, `"`+pendingID+`"`)
	}
	body := `{"id_tipe_sparepart":[` + strings.Join(id, ",") + `],"status":"1"}`

	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, []string{"id_tipe_sparepart"}, violationFields(t, content))
}

// Baris yang sudah ditolak dapat dikembalikan ke antrean lewat keputusan.
func TestDecisionCanMoveRejectedRowBack(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_tipe_sparepart":["` + rejectedID + `"],"status":"0"}`
	response, content := p.call(t, http.MethodPost, route+"/keputusan", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 1, content["jumlah_berubah"])
	require.Equal(t, "Waiting Approval", content["status_label"])
}

// TIDAK ADA DELETE. `D-66` melarang penghapusan fisik data bernilai bisnis, dan sistem lama
// pun tidak punya satu pun terhadap tabel ini.
func TestDeleteIsNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

// Seluruh jalur tulis menuntut sesi.
func TestWriteRoutesRequireSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, one := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, route, validBody()},
		{http.MethodPut, route + "/" + approvedID, validBody()},
		{http.MethodPost, route + "/keputusan", `{"id_tipe_sparepart":["4"],"status":"1"}`},
		{http.MethodGet, route + "/pilihan", ""},
	} {
		response, _ := p.call(t, one.method, one.path, "ASM", one.body)
		require.Equalf(t, http.StatusUnauthorized, response.StatusCode,
			"%s %s seharusnya menuntut sesi", one.method, one.path)
	}
}
