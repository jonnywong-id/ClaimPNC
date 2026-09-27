package inboxinvestigatorhttp_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	authmemory "claim-pnc/internal/auth/repo/memory"
	"claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/inboxinvestigator/repo/memory"
	inboxinvestigatorusecase "claim-pnc/internal/inboxinvestigator/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxinvestigatorhttp "claim-pnc/internal/inboxinvestigator/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena uji
// lain kebetulan memakai jalur yang benar.
const route = "/api/inbox/investigator"

// Jumlah baris pada memory.SampleTasks.
const sampleRows = 8

// sessionAt adalah waktu tetap untuk jam modul auth, supaya sesi ujinya tidak kedaluwarsa
// di tengah jalan. Modul ini sendiri TIDAK memakai jam apa pun.
var sessionAt = time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi dan
// middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa satu entitas tidak pernah melihat antrean entitas lain. Menguji handler
// secara terpisah tidak dapat membuktikan ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi tidak diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (`ADR-0030`, `R-20`). Pada modul ini yang bocor adalah antrean pekerjaan satu badan
	// hukum, lengkap dengan nama tertanggung dan nama pesertanya.
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
		Clock:           clock.FixedAt(sessionAt),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{})

	service, err := inboxinvestigatorusecase.NewService(
		inboxinvestigatorusecase.Options{
			RepoSelector: func(alias string) (inboxinvestigator.Repo, error) {
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

	handler, err := inboxinvestigatorhttp.NewHandler(
		inboxinvestigatorhttp.Options{
			Service:       service,
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    inboxinvestigatorhttp.ErrorWriter(writeError),
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
				inboxinvestigatorhttp.Mount(protected, handler, portalDeps)
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
	method, path, portalAlias string,
) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(method, p.server.URL+path, nil)
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })

	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// tasks mengambil senarai tugas dari badan respons.
func tasks(t *testing.T, content map[string]any) []any {
	t.Helper()

	list, ok := content["tugas"].([]any)
	require.Truef(t, ok, "badan respons tidak memuat senarai tugas: %v", content)
	return list
}

// search menyusun query string penyaring dengan pengodean yang benar.
func search(keyword string) string {
	return url.Values{"cari": {keyword}}.Encode()
}

// rowByCase mencari satu baris menurut nomor case-nya.
func rowByCase(t *testing.T, content map[string]any, caseNumber string) map[string]any {
	t.Helper()

	for _, item := range tasks(t, content) {
		row, ok := item.(map[string]any)
		require.True(t, ok)
		if row["nomor_case"] == caseNumber {
			return row
		}
	}
	t.Fatalf("baris %q tidak ada di dalam respons", caseNumber)
	return nil
}

// TestListRequiresSession membuktikan rutenya benar-benar berada di balik middleware sesi.
func TestListRequiresSession(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// TestListRequiresPortal membuktikan permintaan tanpa header portal DITOLAK, bukan dilayani
// portal utama sebagai cadangan.
//
// Ini uji `R-20`: jatuh ke koneksi bawaan berarti menampilkan antrean pekerjaan satu badan
// hukum kepada petugas badan hukum lain — tanpa satu pun pesan galat, dan layarnya tampak
// normal.
func TestListRequiresPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestListRejectsUnknownPortal(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodGet, route, "TIDAKADA")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// TestListReturnsQueueOfActivePortal membuktikan antrean yang terkirim adalah milik portal
// yang diminta — dan bahwa entitas yang antreannya kosong benar-benar menerima daftar
// kosong, bukan antrean entitas lain.
func TestListReturnsQueueOfActivePortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, tasks(t, content), sampleRows)
	require.Equal(t, "ASM", content["portal"])

	response, content = p.call(t, http.MethodGet, route, "ASI")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, tasks(t, content))
	require.Equal(t, "ASI", content["portal"])
}

// TestEmptyQueueIsNotAnError membuktikan inbox yang bersih dijawab 200 dengan senarai
// KOSONG — bukan galat, dan bukan `null`.
//
// Senarai `null` akan membuat layar yang memetakannya gagal; inbox yang habis dikerjakan
// adalah keadaan yang justru diharapkan.
func TestEmptyQueueIsNotAnError(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, route, "ASI")

	require.Equal(t, http.StatusOK, response.StatusCode)
	list, ok := content["tugas"].([]any)
	require.True(t, ok, "senarai tugas wajib [] dan tidak pernah null")
	require.Empty(t, list)
}

// TestNinthColumnCarriesSurveyDate mengunci pembacaan kolom kesembilan.
//
// Captionnya di layar lama berbunyi "Lama Masuk Inbox", tetapi selnya terikat
// `.ClaimData.SurveyResults(1).SurveyDate` — terbukti dari pemasangan caption-ke-sel satu
// lawan satu pada `Section/InputInvestigator_Section-Section.xml`.
//
// Kontraknya karena itu mengirim TANGGAL, bukan angka durasi. Uji ini gagal lebih dulu bila
// seseorang mengubahnya kembali menjadi hitungan.
func TestNinthColumnCarriesSurveyDate(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM")
	row := rowByCase(t, content, "PNC-100241")

	require.Equal(t, "2026-09-22T01:00:00Z", row["tanggal_survey"])
	require.NotContains(t, row, "lama_menunggu_jam",
		"kolom ini menampilkan tanggal survei, bukan durasi yang dihitung")
}

// TestSurveyDateIsNullWhenClaimHasNoSurvey membuktikan klaim yang belum disurvei tetap
// tampil — hanya kolom kesembilannya yang kosong.
func TestSurveyDateIsNullWhenClaimHasNoSurvey(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM")
	row := rowByCase(t, content, "PNC-100236")

	require.Nil(t, row["tanggal_survey"])
}

// TestResponseStatesTheRowLimit membuktikan batas 500 baris DINYATAKAN, bukan senyap.
//
// Inilah satu-satunya perbedaan yang disengaja terhadap `pyMaxRecords = 500` sistem lama,
// yang memotong tanpa memberi tahu siapa pun.
func TestResponseStatesTheRowLimit(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route, "ASM")

	require.Equal(t, float64(inboxinvestigator.MaxRows), content["batas_baris"])
	require.Equal(t, false, content["terpotong"],
		"antrean contoh jauh di bawah batas, sehingga tidak boleh ditandai terpotong")
}

// TestSearchNarrowsTheQueue membuktikan penyaring `cari` bekerja pada ketujuh kolom teks.
func TestSearchNarrowsTheQueue(t *testing.T) {
	p := newTestServer(t)

	_, content := p.call(t, http.MethodGet, route+"?"+search("100241"), "ASM")

	require.Len(t, tasks(t, content), 1)
	require.Equal(t, "PNC-100241", rowByCase(t, content, "PNC-100241")["nomor_case"])
}

// TestSearchIsCaseInsensitive membuktikan pencarian tidak memandang huruf besar-kecil, sama
// seperti `UPPER(...) LIKE ...` pada kuerinya.
//
// Kata kuncinya dikodekan dengan url.Values, bukan ditempel apa adanya: kata kunci yang
// dipakai memuat spasi, dan spasi yang tidak dikodekan membuat baris permintaannya cacat —
// server menolaknya sebelum handler mana pun dipanggil.
func TestSearchIsCaseInsensitive(t *testing.T) {
	p := newTestServer(t)

	_, lower := p.call(t, http.MethodGet, route+"?"+search("personal accident"), "ASM")
	_, upper := p.call(t, http.MethodGet, route+"?"+search("PERSONAL ACCIDENT"), "ASM")

	require.NotEmpty(t, tasks(t, lower))
	require.Len(t, tasks(t, upper), len(tasks(t, lower)))
}

// TestSearchRejectsOverlongKeyword membuktikan permintaan yang tidak dapat dipenuhi DITOLAK
// dengan sebabnya.
//
// Daftar kosong akan terbaca sebagai "antreannya memang kosong", dan pengguna tidak punya
// cara membedakan keduanya.
func TestSearchRejectsOverlongKeyword(t *testing.T) {
	p := newTestServer(t)

	long := strings.Repeat("a", inboxinvestigatorhttp.MaxKeywordLength+1)
	response, content := p.call(t, http.MethodGet, route+"?"+search(long), "ASM")

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, inboxinvestigatorhttp.CodeMalformedRequest, content["kode"])
}

// TestStorageFailureIsNotAnEmptyQueue membuktikan kegagalan membaca antrean dijawab galat.
//
// Menjawabnya dengan daftar kosong akan membuat petugas menyimpulkan tidak ada pekerjaan,
// lalu pulang.
func TestStorageFailureIsNotAnEmptyQueue(t *testing.T) {
	p := newTestServer(t)
	p.asm.SetError(portal.ErrNotReady)

	response, _ := p.call(t, http.MethodGet, route, "ASM")

	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

// TestNoWriteRoutesAreRegistered membuktikan modul ini benar-benar tidak menyediakan jalur
// tulis.
//
// Mengambil pekerjaan dari antrean dan mencatat hasil investigasi terjadi di layar kerja
// yang belum dibangun. Uji ini akan gagal pada hari seseorang mendaftarkan rute tulis di
// sini tanpa memindahkan kepemilikan tabelnya lebih dulu (`P-1`).
func TestNoWriteRoutesAreRegistered(t *testing.T) {
	p := newTestServer(t)

	for _, method := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
	} {
		t.Run(method, func(t *testing.T) {
			response, _ := p.call(t, method, route, "ASM")
			require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
		})
	}
}
