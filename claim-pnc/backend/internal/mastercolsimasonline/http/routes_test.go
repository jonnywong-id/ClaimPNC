package mastercolsimasonlinehttp_test

import (
	"bytes"
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
	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/mastercolsimasonline/repo/memory"
	mastercolusecase "claim-pnc/internal/mastercolsimasonline/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastercolhttp "claim-pnc/internal/mastercolsimasonline/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const routeList = "/api/master/col-simas-online"
const routeBusiness = "/api/master/bisnis"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang
// sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
type testServer struct {
	server *httptest.Server
	token  string

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
		Clock:           clock.FixedAt(time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	asm := memory.NewRepo(memory.SampleList()...).WithBusiness(business)
	asi := memory.NewRepo().WithBusiness(business)

	service, err := mastercolusecase.NewService(mastercolusecase.Options{
		RepoSelector: func(alias string) (mastercolsimasonline.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		BusinessSelector: func(alias string) (mastercolsimasonline.BusinessRepo, error) {
			switch alias {
			case "ASM", "ASI":
				return business, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Tipe fungsinya ditulis eksplisit, bukan diserahkan ke inferensi: setiap modul
	// mendeklarasikan tipe penulisnya SENDIRI (mastercolhttp.ErrorWriter,
	// portalhttp.ErrorWriter, authhttp.ErrorWriter) supaya modul tidak saling mengimpor.
	// Yang menjembatani ketiganya adalah berkas perakitan — di aplikasi itu
	// cmd/claimpnc, di sini uji ini.
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

	handler, err := mastercolhttp.NewHandler(mastercolhttp.Options{
		Service:       service,
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
				mastercolhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	s := &testServer{server: server, asm: asm, asi: asi}
	s.token = s.login(t)
	return s
}

func (s *testServer) login(t *testing.T) string {
	t.Helper()

	body := strings.NewReader(`{"nama_pengguna":"adminpnc","kata_sandi":"rahasia123"}`)
	response, err := http.Post(s.server.URL+"/api/masuk", "application/json", body)
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
func (s *testServer) call(t *testing.T, method, path, portalAlias, body string) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, s.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	if s.token != "" {
		request.Header.Set("Authorization", "Bearer "+s.token)
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

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	s := newTestServer(t)
	s.token = ""

	response, content := s.call(t, http.MethodGet, routeList, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])
}

// Permintaan tanpa portal DITOLAK, bukan jatuh ke portal utama sebagai cadangan.
// Itu jalur kegagalan `R-20` yang paling mudah terjadi tanpa disadari.
func TestWithoutPortalRejected(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, routeList, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestUnknownPortalRejected(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, routeList, "TIDAK-ADA", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// Portal yang terdaftar tetapi koneksinya belum hidup dibedakan dari portal yang tidak
// ada sama sekali — pengguna perlu tahu mana yang "salah pilih" dan mana yang "belum
// siap".
func TestKnownButNotReadyPortalRejected(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, routeList, "SMAS", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

func TestListReturnsRowsOfTheChosenPortal(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"], "entitas yang menjawab disebut terang-terangan")

	rows, matched := content["cause_of_loss"].([]any)
	require.True(t, matched)
	require.NotEmpty(t, rows)
}

// Berpindah portal berarti berpindah data. Ini inti `ADR-0030`, dan gejala kegagalannya
// sangat halus: layarnya tampil normal, angkanya masuk akal, hanya miliknya salah.
func TestAnotherPortalSeesItsOwnDataOnly(t *testing.T) {
	s := newTestServer(t)

	_, content := s.call(t, http.MethodGet, routeList, "ASI", "")
	require.Equal(t, "ASI", content["portal"])

	rows, matched := content["cause_of_loss"].([]any)
	require.True(t, matched)
	require.Empty(t, rows, "ASI belum punya satu baris pun")
}

// Daftar kosong terkirim sebagai `[]`, bukan `null`. Layar yang menerima `null` harus
// menjaganya sendiri, dan satu layar yang lupa akan gagal saat tabelnya masih kosong.
func TestEmptyListIsAnArrayNotNull(t *testing.T) {
	s := newTestServer(t)

	response, err := http.NewRequest(http.MethodGet, s.server.URL+routeList, nil)
	require.NoError(t, err)
	response.Header.Set("Authorization", "Bearer "+s.token)
	response.Header.Set(portalhttp.HeaderPortal, "ASI")

	raw, err := http.DefaultClient.Do(response)
	require.NoError(t, err)
	defer func() { _ = raw.Body.Close() }()

	body := new(bytes.Buffer)
	_, err = body.ReadFrom(raw.Body)
	require.NoError(t, err)
	require.Contains(t, body.String(), `"cause_of_loss":[]`)
}

// Daftar SENGAJA tidak membawa pemetaan bisnis; layar memuatnya saat baris dibuka.
func TestListDoesNotCarryBusinessMapping(t *testing.T) {
	s := newTestServer(t)

	_, content := s.call(t, http.MethodGet, routeList, "ASM", "")
	rows := content["cause_of_loss"].([]any)
	first := rows[0].(map[string]any)
	require.Empty(t, first["bisnis"])
}

func TestGetCarriesTheBusinessMapping(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList+"/1001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	require.Equal(t, "1001", row["id"])
	require.Len(t, row["bisnis"], 2)
}

func TestGetMissingRowIs404(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList+"/9999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, mastercolhttp.CodeNotFound, content["kode"])
}

// 201, dan badannya memuat baris yang benar-benar tersimpan beserta ID-nya. ID
// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
func TestCreateAnswers201WithTheStoredRow(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"1001","bisnis":["FIRE / PROPERTY"]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	require.NotEmpty(t, row["id"])
	require.Equal(t, "BANJIR", row["nama"])
	require.Equal(t, "1001", row["id_master_kerugian"])
	require.Len(t, row["bisnis"], 1)
}

// Klien mengirim NAMA bisnis; server yang menyelesaikannya menjadi ID dengan mencocokkan
// ke master — cara yang sama dengan autocomplete Pega yang mengisi `.ID` saat sebuah
// pilihan diambil dari daftar.
func TestCreateResolvesBusinessNameToItsID(t *testing.T) {
	s := newTestServer(t)

	_, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"","bisnis":["FIRE / PROPERTY"]}`)

	row := content["cause_of_loss"].(map[string]any)
	business := row["bisnis"].([]any)[0].(map[string]any)
	require.Equal(t, "006", business["id"])
	require.Equal(t, "FIRE / PROPERTY", business["nama"])
}

// PERILAKU PEGA YANG DIPERTAHANKAN (`pyAllowFreeFormInput=true`): nama bisnis di luar
// master TETAP diterima, dan tersimpan dengan `id` kosong. Layar harus menyiapkan
// keadaan itu — ia bukan tanda data rusak.
func TestBusinessNameOutsideTheMasterIsAcceptedWithoutAnID(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"","bisnis":["BENGKEL BARU"]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	business := row["bisnis"].([]any)[0].(map[string]any)
	require.Equal(t, "", business["id"])
	require.Equal(t, "BENGKEL BARU", business["nama"])
}

// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
func TestValidationFailureIs422(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"9999","bisnis":[]}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, mastercolhttp.CodeValidationFailed, content["kode"])
}

// Nama kosong DITERIMA — kesetaraan dengan Pega, yang tidak memvalidasi apa pun pada
// layar ini (`pyRequired=false` pada seluruh isian, nol Validate rule).
func TestEmptyNameIsAcceptedJustLikePega(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"","id_master_kerugian":"","bisnis":[]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
}

// Bisnis kembar DITERIMA — grid Pega tidak punya penanda keunikan sama sekali.
func TestDuplicateBusinessIsAcceptedJustLikePega(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"","bisnis":["ANEKA","ANEKA"]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	require.Len(t, row["bisnis"], 2, "kedua baris kembar tersimpan apa adanya")
}

// ID Master Kerugian menunjuk cause of loss LAIN. Kode yang tidak ada ditolak sebagai
// pelanggaran isian, bukan galat 500.
func TestUnknownMasterCodeIsRejectedWithAFieldViolation(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"9999","bisnis":[]}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	detail := content["detail"].([]any)
	first := detail[0].(map[string]any)
	require.Equal(t, mastercolsimasonline.FieldMasterCode, first["kolom"])
}

// Rujukan-diri DITERIMA — dropdown Pega menawarkan baris itu sendiri, dan tidak ada
// apa pun yang menelusuri jenjangnya.
func TestSelfReferenceIsAcceptedJustLikePega(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPut, routeList+"/1001", "ASM",
		`{"nama":"KEBAKARAN","id_master_kerugian":"1001","bisnis":[]}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	require.Equal(t, "1001", row["id_master_kerugian"])
}

func TestUpdateChangesTheRowButNotItsID(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPut, routeList+"/1001", "ASM",
		`{"nama":"KEBAKARAN DAN PETIR","id_master_kerugian":"","bisnis":["ANEKA"]}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row := content["cause_of_loss"].(map[string]any)
	require.Equal(t, "1001", row["id"])
	require.Equal(t, "KEBAKARAN DAN PETIR", row["nama"])
	require.Len(t, row["bisnis"], 1)
}

func TestUpdateMissingRowIs404(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodPut, routeList+"/9999", "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"","bisnis":[]}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
// tanda bahwa ada yang salah.
func TestUnknownFieldIsRejected(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","namaa":"salah ketik"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, mastercolhttp.CodeMalformedRequest, content["kode"])
}

func TestMalformedJSONIsRejected(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM", `{"nama":`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, mastercolhttp.CodeMalformedRequest, content["kode"])
}

// Badan yang memuat lebih dari satu dokumen JSON ditolak — dokumen kedua tidak boleh
// diam-diam diabaikan.
func TestTrailingJSONDocumentIsRejected(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodPost, routeList, "ASM",
		`{"nama":"BANJIR","id_master_kerugian":"","bisnis":[]}{"nama":"LAIN"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Daftar bisnis ikut menuntut portal: POOLDATA.BUSINESS hidup di basis data setiap
// entitas, sehingga "bisnis milik siapa" ditentukan portal yang aktif.
func TestBusinessListRequiresPortal(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, routeBusiness, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestBusinessListReturnsTheMaster(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeBusiness, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	rows, matched := content["bisnis"].([]any)
	require.True(t, matched)
	require.Len(t, rows, len(memory.SampleBusinessList()))
}

// Tidak ada rute hapus, dan itu bukan kelalaian — `D-66` melarang penghapusan fisik data
// bernilai bisnis. Uji ini menjaganya tidak ditambahkan diam-diam kelak.
func TestThereIsNoDeleteRoute(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodDelete, routeList+"/1001", "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}
