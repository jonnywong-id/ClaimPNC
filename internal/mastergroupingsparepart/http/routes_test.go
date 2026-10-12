package mastergroupingspareparthttp_test

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
	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/mastergroupingsparepart/repo/memory"
	mastergroupingsparepartusecase "claim-pnc/internal/mastergroupingsparepart/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastergroupingspareparthttp "claim-pnc/internal/mastergroupingsparepart/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Baris contoh pada memory.SampleList.
const (
	approvedID = "1"
	pendingID  = "3"
	rejectedID = "4"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi dan
// middleware portal.
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

	groupingService, err := mastergroupingsparepartusecase.NewService(
		mastergroupingsparepartusecase.Options{
			RepoSelector: func(alias string) (mastergroupingsparepart.Store, error) {
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
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan portal akan
	// lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := mastergroupingspareparthttp.NewHandler(
		mastergroupingspareparthttp.Options{
			Service: groupingService,
			Caller: func(ctx context.Context) (mastergroupingspareparthttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return mastergroupingspareparthttp.Caller{}, false
				}
				return mastergroupingspareparthttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    mastergroupingspareparthttp.ErrorWriter(writeError),
		})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap. Itulah
	// yang membedakan "tidak ada" dari "belum tersedia".
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
				mastergroupingspareparthttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai grouping dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["grouping"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai grouping: %v", content)
	return list
}

// one mengambil satu objek grouping dari badan respons tunggal.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	object, ok := content["grouping"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat objek grouping: %v", content)
	return object
}

// violationFields mengambil nama kolom dari `detail` sebuah galat.
func violationFields(t *testing.T, content map[string]any) []string {
	t.Helper()

	detail, ok := content["detail"].([]any)
	require.Truef(t, ok, "badan galat tidak memuat detail: %v", content)

	field := make([]string, 0, len(detail))
	for _, one := range detail {
		entry, ok := one.(map[string]any)
		require.True(t, ok)
		field = append(field, entry["kolom"].(string))
	}
	return field
}

// validBody adalah badan permintaan yang lolos seluruh pemeriksaan.
const validBody = `{
  "nomor_sparepart": "SP-1002",
  "id_panel": "PNL02",
  "nama_panel": "KABIN",
  "sisi": "2",
  "no_rangka": "MHFNEW0001K0000001",
  "tipe_kendaraan": "EXCAVATOR",
  "grouping_dengan_no_rangka": "",
  "catatan": "Baris baru."
}`

// SELURUH rute berada di balik sesi. Tanpa uji ini, satu rute yang lupa dipasang di balik
// middleware baru ketahuan setelah datanya terbaca orang luar.
func TestEveryRouteRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, one := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/grouping-sparepart"},
		{http.MethodGet, "/api/master/grouping-sparepart/pilihan"},
		{http.MethodGet, "/api/master/grouping-sparepart/sisi?id_panel=PNL01"},
		{http.MethodGet, "/api/master/grouping-sparepart/sparepart?nomor=SP-1001"},
		{http.MethodGet, "/api/master/grouping-sparepart/" + approvedID},
		{http.MethodPost, "/api/master/grouping-sparepart"},
		{http.MethodPut, "/api/master/grouping-sparepart/" + approvedID},
		{http.MethodPost, "/api/master/grouping-sparepart/keputusan"},
	} {
		response, _ := p.call(t, one.method, one.path, "ASM", "")
		require.Equalf(t, http.StatusUnauthorized, response.StatusCode,
			"%s %s harus menolak permintaan tanpa sesi", one.method, one.path)
	}
}

// SELURUH rute menuntut portal, termasuk keempat rute acuan.
//
// Isi keempatnya dibaca dari basis data entitas; menyajikan daftar panel satu entitas kepada
// entitas lain adalah kebocoran yang justru dicegah `R-20`.
func TestEveryRouteRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	for _, one := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/grouping-sparepart"},
		{http.MethodGet, "/api/master/grouping-sparepart/pilihan"},
		{http.MethodGet, "/api/master/grouping-sparepart/sisi?id_panel=PNL01"},
		{http.MethodGet, "/api/master/grouping-sparepart/sparepart?nomor=SP-1001"},
		{http.MethodGet, "/api/master/grouping-sparepart/" + approvedID},
		{http.MethodPost, "/api/master/grouping-sparepart/keputusan"},
	} {
		response, _ := p.call(t, one.method, one.path, "", "")
		require.NotEqualf(t, http.StatusOK, response.StatusCode,
			"%s %s harus menolak permintaan tanpa portal", one.method, one.path)
	}
}

// Portal yang belum siap DITOLAK, bukan dijawab dengan data portal lain.
func TestUnreadyPortalIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/grouping-sparepart", "SMAS", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// Data satu entitas tidak pernah bocor ke entitas lain.
func TestPortalsAreIsolated(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, "/api/master/grouping-sparepart?status=1", "ASM", "")
	require.NotEmpty(t, rows(t, asm))
	require.Equal(t, "ASM", asm["portal"])

	_, asi := p.call(t, http.MethodGet, "/api/master/grouping-sparepart?status=1", "ASI", "")
	require.Empty(t, rows(t, asi))
	require.Equal(t, "ASI", asi["portal"])
}

func TestListFiltersByStatus(t *testing.T) {
	p := newTestServer(t)

	for _, one := range []struct {
		status string
		count  int
	}{
		{"1", 2},
		{"0", 1},
		{"2", 1},
	} {
		_, content := p.call(t, http.MethodGet,
			"/api/master/grouping-sparepart?status="+one.status, "ASM", "")
		require.Lenf(t, rows(t, content), one.count, "status %q", one.status)
		require.Equal(t, one.status, content["status"])
	}
}

// Tanpa penyaring status, yang dijawab adalah tab Approve — sama seperti tab yang terbuka
// lebih dulu di layar.
func TestListDefaultsToApproved(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, "/api/master/grouping-sparepart", "ASM", "")
	require.Equal(t, "1", content["status"])
}

func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// Pencarian menelusuri keempat kolom.
func TestListSearchesFourColumns(t *testing.T) {
	p := newTestServer(t)

	for _, keyword := range []string{"SP-1001", "FILTER", "KABIN", "MHFXW1234"} {
		_, content := p.call(t, http.MethodGet,
			"/api/master/grouping-sparepart?status=1&cari="+keyword, "ASM", "")
		require.NotEmptyf(t, rows(t, content), "kata kunci %q harus menemukan baris", keyword)
	}
}

// Sebutan status dan sisi dihitung SERVER, supaya layar tidak menyimpan salinan sandinya.
func TestResponseCarriesComputedLabels(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/"+approvedID, "ASM", "")
	object := one(t, content)

	require.Equal(t, "Approve", object["status_label"])
	require.Equal(t, "KIRI", object["sisi_label"], "sandi 1 adalah KIRI")
}

func TestGetUnknownRowIsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Penambahan menjawab 201 dan memuat baris yang benar-benar tersimpan.
func TestCreateReturnsCreatedRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart", "ASM", validBody)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	object := one(t, content)
	require.Equal(t, "5", object["id_grouping"])
	require.Equal(t, "0", object["status"], "baris baru lahir menunggu persetujuan")
	require.Equal(t, "SEAL KIT BOOM", object["nama_sparepart"],
		"kelima isian turunan diisi server dari Master Sparepart")
	require.NotEmpty(t, object["nomor_grup"])
}

// Kelima isian turunan TIDAK dapat dikirim klien.
//
// `DisallowUnknownFields` yang menolaknya, supaya cacat pada klien terlihat saat pertama
// dicoba alih-alih menjadi kebiasaan yang tidak berakibat.
func TestCreateRejectsClientSuppliedDerivedFields(t *testing.T) {
	p := newTestServer(t)

	body := `{
	  "nomor_sparepart": "SP-1002",
	  "nama_sparepart": "NAMA KARANGAN",
	  "id_panel": "PNL02",
	  "nama_panel": "KABIN",
	  "sisi": "2",
	  "no_rangka": "MHFNEW0002K0000002"
	}`

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Nomor sparepart yang tidak ada ditolak 422 dengan pesan yang menempel pada isiannya.
func TestCreateRejectsUnknownPartNumber(t *testing.T) {
	p := newTestServer(t)

	body := `{
	  "nomor_sparepart": "SP-TIDAK-ADA",
	  "id_panel": "PNL02",
	  "nama_panel": "KABIN",
	  "sisi": "2",
	  "no_rangka": "MHFNEW0003K0000003"
	}`

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Contains(t, violationFields(t, content), "nomor_sparepart")
}

// Kunci alami yang sudah dipakai ditolak 409 — konflik keadaan, bukan isian yang cacat.
func TestCreateRejectsDuplicateKey(t *testing.T) {
	p := newTestServer(t)

	body := `{
	  "nomor_sparepart": "SP-1001",
	  "id_panel": "PNL02",
	  "nama_panel": "KABIN",
	  "sisi": "1",
	  "no_rangka": "MHFXW1234K5678901"
	}`

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart", "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "grouping_sudah_ada", content["kode"])
}

// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5).
func TestCreateReportsAllViolationsTogether(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart", "ASM", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Len(t, violationFields(t, content), 4)
}

// Menyimpan mengembalikan baris ke antrean persetujuan.
func TestSaveReturnsRowToPending(t *testing.T) {
	p := newTestServer(t)

	body := `{
	  "nomor_sparepart": "SP-1001",
	  "id_panel": "PNL02",
	  "nama_panel": "KABIN",
	  "sisi": "1",
	  "no_rangka": "MHFXW1234K5678901",
	  "tipe_kendaraan": "EXCAVATOR",
	  "catatan": "Diubah."
	}`

	response, content := p.call(t, http.MethodPut,
		"/api/master/grouping-sparepart/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	object := one(t, content)
	require.Equal(t, "0", object["status"])
	require.Equal(t, "Diubah.", object["catatan"])
}

// Jalur simpan TIDAK menerima status; mengirimnya ditolak sebagai permintaan cacat.
//
// Pada bentuk lama, satu permintaan simpan dapat sekaligus menyetujui dirinya sendiri hanya
// dengan mengirim `Approval=1`; lihat usecase.Service.Save.
func TestSaveRejectsStatusInBody(t *testing.T) {
	p := newTestServer(t)

	body := `{
	  "nomor_sparepart": "SP-1001",
	  "id_panel": "PNL02",
	  "nama_panel": "KABIN",
	  "sisi": "1",
	  "no_rangka": "MHFXW1234K5678901",
	  "status": "1"
	}`

	response, content := p.call(t, http.MethodPut,
		"/api/master/grouping-sparepart/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestDecideMovesSelectedRows(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart/keputusan", "ASM",
		`{"id_grouping":["`+pendingID+`"],"status":"1"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 1, content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])
}

// Baris yang sudah berstatus itu TIDAK terhitung sebagai berubah.
func TestDecideDoesNotCountUnchangedRows(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart/keputusan", "ASM",
		`{"id_grouping":["`+rejectedID+`"],"status":"2"}`)
	require.EqualValues(t, 0, content["jumlah_berubah"])
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart/keputusan", "ASM", `{"id_grouping":[],"status":"1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Contains(t, violationFields(t, content), "id_grouping")
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart/keputusan", "ASM",
		`{"id_grouping":["`+pendingID+`"],"status":"9"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

func TestOptionsReturnsBothLookups(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/pilihan", "ASM", "")

	panel, ok := content["panel"].([]any)
	require.True(t, ok)
	require.Len(t, panel, 3)

	vehicle, ok := content["tipe_kendaraan"].([]any)
	require.True(t, ok)
	require.Len(t, vehicle, 3)
}

// Daftar Sisi memuat sandi DAN sebutannya.
func TestSidesCarryCodeAndLabel(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/sisi?id_panel=PNL01&nama_panel=BUMPER%20DEPAN",
		"ASM", "")

	side, ok := content["sisi"].([]any)
	require.True(t, ok)
	require.Len(t, side, 2)

	first := side[0].(map[string]any)
	require.Equal(t, "1", first["kode"])
	require.Equal(t, "KIRI", first["nama"])
}

// Permintaan sisi tanpa panel sama sekali DITOLAK — kueri aslinya akan mencocoki seluruh
// baris lokasi di basis data.
func TestSidesWithoutPanelIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/sisi", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Contains(t, violationFields(t, content), "id_panel")
}

func TestPartLookupReturnsDerivedFields(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/sparepart?nomor=SP-1001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "FILTER OLI", content["nama_sparepart"])
	require.Equal(t, "KAT01", content["kategori_sparepart"])
}

func TestPartLookupRejectsUnknownNumber(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet,
		"/api/master/grouping-sparepart/sparepart?nomor=SP-TIDAK-ADA", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "sparepart_tidak_ditemukan", content["kode"])
}

// Segmen statis tidak pernah terbaca sebagai sebuah ID grouping.
func TestStaticSegmentsAreNotTreatedAsID(t *testing.T) {
	p := newTestServer(t)

	for _, path := range []string{
		"/api/master/grouping-sparepart/pilihan",
		"/api/master/grouping-sparepart/sparepart?nomor=SP-1001",
	} {
		response, content := p.call(t, http.MethodGet, path, "ASM", "")
		require.Equalf(t, http.StatusOK, response.StatusCode, "%s", path)
		require.NotContainsf(t, content, "grouping", "%s tidak boleh dilayani handler Get", path)
	}
}

// TIDAK ADA rute DELETE.
//
// Sistem lama tidak punya satu pun terhadap kedua tabel ini, dan `D-66` melarang penghapusan
// fisik data bernilai bisnis.
func TestNoDeleteRoute(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete,
		"/api/master/grouping-sparepart/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

// Badan permintaan raksasa ditolak sebelum memakan memori.
func TestOversizedBodyIsRejected(t *testing.T) {
	p := newTestServer(t)

	huge := `{"catatan":"` + strings.Repeat("x", 64<<10) + `"}`
	response, _ := p.call(t, http.MethodPost, "/api/master/grouping-sparepart", "ASM", huge)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Keputusan atas terlalu banyak baris sekaligus ditolak.
//
// Bukan aturan bisnis melainkan penjaga sumber daya: transaksi yang menahan ribuan kunci baris
// menghalangi Pega yang sedang melayani produksi pada tabel yang sama (D-21).
func TestTooManyDecisionRowsRejected(t *testing.T) {
	p := newTestServer(t)

	id := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		id = append(id, `"`+pendingID+`"`)
	}
	body := `{"id_grouping":[` + strings.Join(id, ",") + `],"status":"1"}`

	response, content := p.call(t, http.MethodPost,
		"/api/master/grouping-sparepart/keputusan", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Contains(t, violationFields(t, content), "id_grouping")
}
