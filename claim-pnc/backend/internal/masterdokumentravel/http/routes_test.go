package masterdokumentravelhttp_test

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
	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/repo/memory"
	masterdokumentravelusecase "claim-pnc/internal/masterdokumentravel/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterdokumentravelhttp "claim-pnc/internal/masterdokumentravel/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware
// sesi dan middleware portal.
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

	travelDocumentService, err := masterdokumentravelusecase.NewService(masterdokumentravelusecase.Options{
		RepoSelector: func(alias string) (masterdokumentravel.Repo, error) {
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
	// diteruskan ke pemeta modul auth. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler, err := masterdokumentravelhttp.NewHandler(masterdokumentravelhttp.Options{
		Service:       travelDocumentService,
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterdokumentravelhttp.ErrorWriter(writeError),
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
				masterdokumentravelhttp.Mount(protected, handler, portalDeps)
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

const route = "/api/master/dokumen-travel"

func TestListRepliesWithTheEntityThatAnsweredIt(t *testing.T) {
	// Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
	// diandaikan — server menyebutkannya.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, float64(6), content["total"])
}

func TestDataOfOneEntityIsNeverVisibleThroughAnother(t *testing.T) {
	// Inilah cacat yang `R-20` sebut: layar tampil normal, angkanya masuk akal, dan yang
	// salah hanya MILIK SIAPA data itu.
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(0), content["total"])
	require.Empty(t, content["dokumen_travel"])
}

func TestRequestWithoutPortalIsRejectedInsteadOfFallingBackToTheDefault(t *testing.T) {
	// Jatuh ke koneksi baku adalah salah satu dari dua jalur kegagalan `R-20`.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestRequestWithoutSessionIsRejected(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	response, _ := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestNewDocumentGetsItsDOCIDFromTheServer(t *testing.T) {
	// Sentinel "UnknownID" milik sistem lama tidak dibawa: ID tidak pernah ada di badan
	// permintaan sama sekali.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{"judul":"Visa"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, _ := content["dokumen_travel"].(map[string]any)
	require.Equal(t, "100007", saved["id"])
	require.Equal(t, "Visa", saved["judul"])
}

// TestEmptyTitleIsAcceptedLikeInPega mengunci keputusan Work Owner 2026-09-21 pada
// tingkat kontrak API, bukan hanya di domain.
func TestEmptyTitleIsAcceptedLikeInPega(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM", `{"judul":""}`)
	require.Equal(t, http.StatusCreated, response.StatusCode,
		"layar ini meniru Pega apa adanya: tanpa validasi")
}

func TestUnknownFieldIsRejectedRatherThanIgnoredSilently(t *testing.T) {
	// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim", dan pada modul
	// tanpa validasi judul kosong itu justru akan TERSIMPAN tanpa satu pun tanda.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM", `{"judul_dokumen":"Visa"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestSavingChangesTheTitleAndKeepsTheDOCID(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/100001", "ASM", `{"judul":"Paspor / KITAS"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved, _ := content["dokumen_travel"].(map[string]any)
	require.Equal(t, "100001", saved["id"])
	require.Equal(t, "Paspor / KITAS", saved["judul"])
}

func TestEditingARowThatIsNoLongerThereIsReportedAsNotFound(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/999999", "ASM", `{"judul":"Apa saja"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "dokumen_travel_tidak_ditemukan", content["kode"])
}

func TestOpeningOneRowForEditingReturnsItsCurrentContents(t *testing.T) {
	// Menggantikan SetMstDocTravelValue_act, yang menyalin baris ke TempMstDocTravel.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/100002", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	doc, _ := content["dokumen_travel"].(map[string]any)
	require.Equal(t, "Tiket Perjalanan", doc["judul"])
}

func TestWritingThroughOneEntityNeverTouchesAnother(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASI", `{"judul":"Hanya milik ASI"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(6), asm["total"], "isi ASM tidak boleh bertambah")

	_, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(1), asi["total"])
}
