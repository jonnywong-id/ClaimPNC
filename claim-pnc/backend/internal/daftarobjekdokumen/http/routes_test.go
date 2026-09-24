package daftarobjekdokumenhttp_test

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
	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/repo/memory"
	daftarobjekdokumenusecase "claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	daftarobjekdokumenhttp "claim-pnc/internal/daftarobjekdokumen/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const routeList = "/api/master/objek-dokumen"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi dan
// middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang sah
// ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()

	service, err := daftarobjekdokumenusecase.NewService(daftarobjekdokumenusecase.Options{
		RepoSelector: func(alias string) (daftarobjekdokumen.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		BusinessSelector: func(alias string) (daftarobjekdokumen.BusinessRepo, error) {
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
	// mendeklarasikan tipe penulisnya SENDIRI supaya modul tidak saling mengimpor. Yang
	// menjembatani ketiganya adalah berkas perakitan — di aplikasi itu cmd/claimpnc, di sini
	// uji ini.
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

	handler, err := daftarobjekdokumenhttp.NewHandler(daftarobjekdokumenhttp.Options{
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
				daftarobjekdokumenhttp.Mount(protected, handler, portalDeps)
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

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh pemeriksaan
// portal maupun basis data.
func TestWithoutSessionRejected(t *testing.T) {
	s := newTestServer(t)
	s.token = ""

	response, content := s.call(t, http.MethodGet, routeList, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Equal(t, authhttp.CodeInvalidSession, content["kode"])
}

// Permintaan tanpa portal DITOLAK, bukan jatuh ke portal utama sebagai cadangan. Itu jalur
// kegagalan `R-20` yang paling mudah terjadi tanpa disadari.
func TestWithoutPortalRejected(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", content["kode"])
}

// Portal yang ADA di daftar tetapi koneksinya belum hidup dibedakan dari portal yang tidak
// dikenal sama sekali — keduanya perlu tindakan yang berbeda.
func TestPortalNotReady(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "portal_belum_siap", content["kode"])
}

// Daftar menjawab dengan amplop lengkap: isi, total, dan portal yang BENAR-BENAR menjawab.
func TestListMengembalikanAmplopLengkap(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	rows, ok := content["objek_dokumen"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 4)
	require.Equal(t, float64(4), content["total"])

	first, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "10001", first["id"])
	require.Equal(t, "KTP Tertanggung", first["objek_dokumen"])
}

// Dua entitas menjawab dengan isi yang berbeda, dan portalnya disebut di setiap jawaban.
func TestEntitasTerpisah(t *testing.T) {
	s := newTestServer(t)

	_, asm := s.call(t, http.MethodGet, routeList, "ASM", "")
	require.NotEmpty(t, asm["objek_dokumen"])

	_, asi := s.call(t, http.MethodGet, routeList, "ASI", "")
	require.Equal(t, "ASI", asi["portal"])
	require.Empty(t, asi["objek_dokumen"])
	require.Equal(t, float64(0), asi["total"])
}

// Tabel kosong terkirim sebagai `[]`, bukan `null`.
//
// Layar yang menerima `null` harus menjaganya sendiri, dan satu layar yang lupa akan gagal
// tepat saat tabelnya masih kosong — yaitu pada entitas yang baru disiapkan.
func TestDaftarKosongBukanNull(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, routeList, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	// Badan mentahnya diperiksa, bukan hasil decode — `null` dan `[]` sama-sama menjadi nil
	// setelah di-decode ke map, sehingga perbedaannya hanya terlihat sebelum itu.
	raw := s.rawBody(t, http.MethodGet, routeList, "ASI")
	require.Contains(t, raw, `"objek_dokumen":[]`)
}

// rawBody mengambil badan respons apa adanya, tanpa decode.
func (s *testServer) rawBody(t *testing.T, method, path, portalAlias string) string {
	t.Helper()

	request, err := http.NewRequest(method, s.server.URL+path, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+s.token)
	request.Header.Set(portalhttp.HeaderPortal, portalAlias)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(response.Body)
	require.NoError(t, err)
	return body.String()
}

// Daftar TIDAK membawa pemetaan bisnis; pengambilan satu baris membawanya.
func TestDaftarTanpaBisnisSatuBarisDenganBisnis(t *testing.T) {
	s := newTestServer(t)

	_, list := s.call(t, http.MethodGet, routeList, "ASM", "")
	rows, _ := list["objek_dokumen"].([]any)
	first, _ := rows[0].(map[string]any)
	require.Empty(t, first["bisnis"])

	_, one := s.call(t, http.MethodGet, routeList+"/10002", "ASM", "")
	row, _ := one["objek_dokumen"].(map[string]any)
	businesses, ok := row["bisnis"].([]any)
	require.True(t, ok)
	require.Len(t, businesses, 2)
}

// ID lama DIKIRIM server meski tidak ditampilkan layar.
//
// Uji ini menjaga keputusan itu tetap terlihat: lapisan data mengikuti Report Definition,
// yang memuat OLD_ID, sedangkan lapisan layar mengikuti section-nya, yang tidak.
func TestIDLamaTetapDikirim(t *testing.T) {
	s := newTestServer(t)

	_, content := s.call(t, http.MethodGet, routeList+"/10003", "ASM", "")
	row, _ := content["objek_dokumen"].(map[string]any)
	require.Equal(t, "07", row["id_lama"])
}

// Baris yang tidak ada menjawab 404 dengan kode yang dapat dibaca mesin.
func TestBarisTidakAda(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodGet, routeList+"/99999", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Penambahan menjawab 201 beserta baris yang benar-benar tersimpan — termasuk ID yang
// diterbitkan server, yang tidak punya cara lain diketahui layar.
func TestCreate(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"objek_dokumen":"Surat Kuasa","bisnis":["ANEKA","TRAVEL"]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	row, _ := content["objek_dokumen"].(map[string]any)
	require.Equal(t, "10005", row["id"])
	require.Equal(t, "Surat Kuasa", row["objek_dokumen"])

	businesses, _ := row["bisnis"].([]any)
	require.Len(t, businesses, 2)
	first, _ := businesses[0].(map[string]any)
	require.Equal(t, "003", first["id"], "nama yang cocok master diselesaikan menjadi ID")
}

// Penyuntingan MENGGANTI seluruh pemetaan bisnis, bukan menggabungkannya.
func TestUpdateMenggantiPemetaanBisnis(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPut, routeList+"/10002", "ASM",
		`{"objek_dokumen":"Polis Asli","bisnis":["TRAVEL"]}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	row, _ := content["objek_dokumen"].(map[string]any)
	businesses, _ := row["bisnis"].([]any)
	require.Len(t, businesses, 1)
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam.
//
// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah — termasuk `id`, yang memang tidak boleh
// datang dari klien.
func TestFieldTidakDikenalDitolak(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"objek_dokumen":"Surat","id":"99999"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Badan yang bukan JSON ditolak tanpa membocorkan rincian penguraiannya — isinya memuat
// cuplikan badan permintaan.
func TestBadanCacatDitolak(t *testing.T) {
	s := newTestServer(t)

	response, content := s.call(t, http.MethodPost, routeList, "ASM", `{bukan json`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
	require.Equal(t, "Permintaan tidak dapat dibaca.", content["pesan"])
}

// Isian yang melanggar aturan menjawab 422 — bukan 400 — beserta SELURUH pelanggarannya,
// masing-masing menyebut nama isian yang dikenali layar.
func TestValidasiGagal(t *testing.T) {
	s := newTestServer(t)

	panjang := strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1)
	response, content := s.call(t, http.MethodPost, routeList, "ASM",
		`{"objek_dokumen":"`+panjang+`","bisnis":[]}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.Len(t, detail, 1)
	first, _ := detail[0].(map[string]any)
	require.Equal(t, "objek_dokumen", first["kolom"])
}

// Keterangan KOSONG tetap diterima, mengikuti layar Pega yang tidak memvalidasi apa pun.
func TestKeteranganKosongDiterima(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodPost, routeList, "ASM", `{"objek_dokumen":"","bisnis":[]}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
}

// Modul ini TIDAK mendaftarkan /master/bisnis — rute itu milik modul Master COL Simas Online,
// dan chi akan PANIK saat start bila dua modul mendaftarkannya bersamaan.
//
// Uji ini yang menjaga batas itu: menambahkannya di sini akan membuat aplikasi sungguhan
// gagal start, dan kegagalan seperti itu jauh lebih mahal ditemukan di lingkungan lain.
func TestRuteBisnisTidakDidaftarkanModulIni(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodGet, "/api/master/bisnis", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

// Tidak ada rute hapus. Seluruh grid di layar lama ber-`pyGridDeleteActivityExists=false`,
// `D-66` melarang penghapusan fisik, dan barisnya dirujuk
// LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan.
func TestTidakAdaRuteHapus(t *testing.T) {
	s := newTestServer(t)

	response, _ := s.call(t, http.MethodDelete, routeList+"/10001", "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}
