package masterspareparthttp_test

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
	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/repo/memory"
	mastersparepartusecase "claim-pnc/internal/mastersparepart/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterspareparthttp "claim-pnc/internal/mastersparepart/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Baris contoh pada memory.SampleList.
const (
	approvedID = "SP0000000001"
	pendingID  = "SP0000000002"
	rejectedID = "SP0000000003"
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

	sparepartService, err := mastersparepartusecase.NewService(mastersparepartusecase.Options{
		RepoSelector: func(alias string) (mastersparepart.Store, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
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

	handler, err := masterspareparthttp.NewHandler(masterspareparthttp.Options{
		Service: sparepartService,
		Caller: func(ctx context.Context) (masterspareparthttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return masterspareparthttp.Caller{}, false
			}
			return masterspareparthttp.Caller{Login: baseCtx.User.Login}, true
		},
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterspareparthttp.ErrorWriter(writeError),
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
				masterspareparthttp.Mount(protected, handler, portalDeps)
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

// rows mengambil senarai sparepart dari badan respons.
func rows(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["sparepart"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai sparepart: %v", content)
	return list
}

// one mengambil satu objek sparepart dari badan respons tunggal.
func one(t *testing.T, content map[string]any) map[string]any {
	t.Helper()

	object, ok := content["sparepart"].(map[string]any)
	require.Truef(t, ok, "badan respons tidak memuat objek sparepart: %v", content)
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

// validBody adalah badan permintaan yang lolos seluruh pemeriksaan.
func validBody() string {
	return `{
		"nomor_sparepart":"20Y-70-21120",
		"nama_sparepart":"BUCKET PIN",
		"kode_sparepart":"BKT-UND-020",
		"harga_jual":"2500000",
		"kategori_sparepart":"KAT03",
		"tipe_sparepart":"TIP05",
		"berat":"4500",
		"panjang":"40",
		"lebar":"8",
		"tinggi":"8",
		"stock_minimal":"2",
		"stock_maximal":"10",
		"kuantitas_pesanan":"4",
		"tanggal_produksi":"2026-01-15",
		"part_substitusi":"",
		"jenis_sparepart":"ORIGINAL",
		"satuan":"PCS",
		"status_aktif":"1",
		"status_sparepart":"READY"
	}`
}

// Tanpa sesi, seluruh rute tertutup — TERMASUK daftar pilihan.
func TestRoutesRequireSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/sparepart"},
		{http.MethodGet, "/api/master/sparepart/pilihan"},
		{http.MethodGet, "/api/master/sparepart/" + approvedID},
		{http.MethodPost, "/api/master/sparepart"},
		{http.MethodPut, "/api/master/sparepart/" + approvedID},
		{http.MethodPost, "/api/master/sparepart/keputusan"},
	} {
		response, _ := p.call(t, r.method, r.path, "ASM", "")
		require.Equalf(t, http.StatusUnauthorized, response.StatusCode,
			"%s %s harus menuntut sesi", r.method, r.path)
	}
}

// SELURUH rute menuntut portal — termasuk `/pilihan`.
//
// Itu yang membedakannya dari Master Panel: daftar pilihan modul ini DIBACA DARI BASIS
// DATA entitas, bukan konstanta yang ditanam di activity Pega.
func TestEveryRouteRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/master/sparepart"},
		{http.MethodGet, "/api/master/sparepart/pilihan"},
		{http.MethodGet, "/api/master/sparepart/" + approvedID},
		{http.MethodPost, "/api/master/sparepart"},
		{http.MethodPut, "/api/master/sparepart/" + approvedID},
		{http.MethodPost, "/api/master/sparepart/keputusan"},
	} {
		response, _ := p.call(t, r.method, r.path, "", "")
		require.Equalf(t, http.StatusBadRequest, response.StatusCode,
			"%s %s harus menolak permintaan tanpa portal", r.method, r.path)
	}
}

// Portal yang ada di daftar tetapi koneksinya belum hidup ditolak — bukan dilayani portal
// utama sebagai jalan pintas (R-20).
func TestPortalNotReadyIsRejected(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/sparepart", "SMAS", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// Entitas yang berbeda melihat data yang berbeda. ASI tidak diberi satu baris pun.
func TestPortalSeparatesData(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart?status=1", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), 1)
	require.Equal(t, "ASM", content["portal"])

	response, content = p.call(t, http.MethodGet, "/api/master/sparepart?status=1", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, rows(t, content))
	require.Equal(t, "ASI", content["portal"])
}

// Daftar kosong terkirim sebagai `[]`, bukan `null`.
func TestEmptyListIsArray(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, "/api/master/sparepart?status=1", "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	// Dibaca mentah supaya `null` tidak tersamar menjadi slice kosong oleh decoder.
	raw, err := http.NewRequest(http.MethodGet,
		p.server.URL+"/api/master/sparepart?status=1", nil)
	require.NoError(t, err)
	raw.Header.Set("Authorization", "Bearer "+p.token)
	raw.Header.Set(portalhttp.HeaderPortal, "ASI")

	got, err := http.DefaultClient.Do(raw)
	require.NoError(t, err)
	defer func() { _ = got.Body.Close() }()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(got.Body)
	require.NoError(t, err)
	require.Contains(t, body.String(), `"sparepart":[]`)
}

// Tanpa `?status`, yang dijawab adalah tab Approve.
func TestListDefaultsToApproved(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "1", content["status"])
	require.Len(t, rows(t, content), 1)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart?status=9", "ASM", "")
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "status_tidak_dikenal", content["kode"])
}

// Pencarian menelusuri ketiga kunci alami.
func TestListSearchCoversThreeKeys(t *testing.T) {
	p := newTestServer(t)

	for name, keyword := range map[string]string{
		"nama":  "filter",
		"nomor": "1r-07",
		"kode":  "flt-eng",
	} {
		t.Run(name, func(t *testing.T) {
			response, content := p.call(t, http.MethodGet,
				"/api/master/sparepart?status=1&cari="+keyword, "ASM", "")
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Len(t, rows(t, content), 1)
		})
	}
}

// Nama kategori dan tipe DIHITUNG server; tabelnya hanya menyimpan ID-nya.
func TestListCarriesLookupNames(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart?status=1", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := rows(t, content)[0].(map[string]any)
	require.Equal(t, "KAT01", row["kategori_sparepart"])
	require.Equal(t, "ENGINE", row["nama_kategori_sparepart"])
	require.Equal(t, "TIP01", row["tipe_sparepart"])
	require.Equal(t, "FILTER", row["nama_tipe_sparepart"])
}

func TestOptionsReturnsBothLookups(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart/pilihan", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	category, ok := content["kategori"].([]any)
	require.True(t, ok)
	require.Len(t, category, 3)

	partType, ok := content["tipe"].([]any)
	require.True(t, ok)
	require.Len(t, partType, 5)

	// Setiap tipe menyebut kategori induknya; layar memakainya untuk mempersempit daftar.
	require.Equal(t, "KAT01", partType[0].(map[string]any)["kode_kategori"])
}

// `/pilihan` tidak pernah terbaca sebagai sebuah ID sparepart.
func TestStaticSegmentsAreNotTreatedAsID(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart/pilihan", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotContains(t, content, "sparepart")
}

func TestGetReturnsNotFound(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart/SP9999999999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Penambahan menjawab 201, dan badannya memuat keempat nilai yang diterbitkan server.
func TestCreateReturns201WithServerIssuedValues(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart", "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := one(t, content)
	require.Equal(t, "SP0000000004", row["id_sparepart"])
	require.Equal(t, "0", row["status"])
	require.Equal(t, "Waiting Approval", row["status_label"])
	require.Equal(t, loginName, row["user_update"])
	require.Equal(t, "2026-09-20T03:00:00Z", row["tanggal_update_harga"])
}

// Harga jual dikirim sebagai TEKS, bukan angka JSON.
//
// JSON number diurai peramban sebagai IEEE-754 double, sehingga nilai uang dengan banyak
// digit dapat berubah hanya karena melewati jaringan (`I-12`, `D-51`).
func TestPriceIsSentAsText(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart", "ASM", validBody())
	require.Equal(t, http.StatusCreated, response.StatusCode)

	price, isText := one(t, content)["harga_jual"].(string)
	require.True(t, isText, "harga jual harus berupa teks pada kontrak API")
	require.Equal(t, "2500000", price)
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart", "ASM", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.ElementsMatch(t,
		[]string{"nomor_sparepart", "nama_sparepart", "kode_sparepart", "harga_jual"},
		violationFields(t, content))
}

// Kunci ganda menjawab 409, bukan 422 — ia konflik keadaan, bukan isian yang cacat.
func TestCreateRejectsDuplicateKeyWith409(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `"nomor_sparepart":"20Y-70-21120"`,
		`"nomor_sparepart":"1R-0716"`, 1)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart", "ASM", body)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, "kunci_sparepart_sudah_ada", content["kode"])
	require.ElementsMatch(t, []string{"nomor_sparepart"}, violationFields(t, content))
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam.
func TestUnknownFieldIsRejected(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `{`, `{"nama_sparepar":"salah ketik",`, 1)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Menyimpan SELALU mengembalikan baris ke antrean persetujuan.
func TestSaveReturnsRowToPending(t *testing.T) {
	p := newTestServer(t)

	body := strings.Replace(validBody(), `"nama_sparepart":"BUCKET PIN"`,
		`"nama_sparepart":"BUCKET PIN REVISI"`, 1)

	response, content := p.call(t, http.MethodPut,
		"/api/master/sparepart/"+approvedID, "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := one(t, content)
	require.Equal(t, approvedID, row["id_sparepart"])
	require.Equal(t, "0", row["status"])
	require.Equal(t, "BUCKET PIN REVISI", row["nama_sparepart"])
	// DOKUMENID dipertahankan; sistem lama menimpanya dengan kosong setiap penyimpanan.
	require.Equal(t, "DOC-0001", row["id_dokumen"])
}

func TestSaveRejectsMissingRow(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPut,
		"/api/master/sparepart/SP9999999999", "ASM", validBody())
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

func TestDecideMovesRows(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_sparepart":["` + pendingID + `"],"status":"1"}`
	response, content := p.call(t, http.MethodPost,
		"/api/master/sparepart/keputusan", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(1), content["jumlah_berubah"])
	require.Equal(t, "Approve", content["status_label"])

	// Barisnya benar-benar berpindah tab.
	_, listed := p.call(t, http.MethodGet, "/api/master/sparepart?status=1", "ASM", "")
	require.Len(t, rows(t, listed), 2)
}

// Badan keputusan TIDAK menerima catatan: tabelnya tidak punya kolom penampungnya.
func TestDecisionRejectsNoteField(t *testing.T) {
	p := newTestServer(t)

	body := `{"id_sparepart":["` + pendingID + `"],"status":"2","catatan":"tidak lengkap"}`
	response, content := p.call(t, http.MethodPost,
		"/api/master/sparepart/keputusan", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodPost, "/api/master/sparepart/keputusan",
		"ASM", `{"id_sparepart":[],"status":"1"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.ElementsMatch(t, []string{"id_sparepart"}, violationFields(t, content))
}

func TestDecideRejectsTooManyRows(t *testing.T) {
	p := newTestServer(t)

	id := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		id = append(id, `"`+pendingID+`"`)
	}
	body := `{"id_sparepart":[` + strings.Join(id, ",") + `],"status":"1"}`

	response, content := p.call(t, http.MethodPost,
		"/api/master/sparepart/keputusan", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.ElementsMatch(t, []string{"id_sparepart"}, violationFields(t, content))
}

// Baris yang ditolak tetap terbaca di tabnya sendiri.
func TestRejectedRowIsListed(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/sparepart?status=2", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, rows(t, content), 1)
	require.Equal(t, rejectedID, rows(t, content)[0].(map[string]any)["id_sparepart"])
}

// Baris yang belum pernah distempel harga tidak mengirim tanggal sama sekali — bukan
// tanggal tahun 1 yang tampak sah.
func TestRowWithoutPriceStampOmitsTheDate(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, "/api/master/sparepart/"+rejectedID, "ASM", "")
	require.NotContains(t, one(t, content), "tanggal_update_harga")
}

// Tidak ada DELETE terhadap sparepart; `D-66` melarang penghapusan fisik.
func TestDeleteIsNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete,
		"/api/master/sparepart/"+approvedID, "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

// Handler menolak rakitan yang tidak lengkap saat dibentuk, bukan saat permintaan datang.
func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := masterspareparthttp.NewHandler(masterspareparthttp.Options{})
	require.Error(t, err)

	service, err := mastersparepartusecase.NewService(mastersparepartusecase.Options{
		RepoSelector: func(string) (mastersparepart.Store, error) { return nil, nil },
		Clock:        clock.FixedAt(time.Now()),
	})
	require.NoError(t, err)

	_, err = masterspareparthttp.NewHandler(masterspareparthttp.Options{Service: service})
	require.Error(t, err)

	_, err = masterspareparthttp.NewHandler(masterspareparthttp.Options{
		Service: service,
		Caller: func(context.Context) (masterspareparthttp.Caller, bool) {
			return masterspareparthttp.Caller{}, false
		},
	})
	require.Error(t, err)
}
