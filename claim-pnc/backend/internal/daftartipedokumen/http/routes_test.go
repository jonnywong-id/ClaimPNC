package daftartipedokumenhttp_test

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
	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/daftartipedokumen/repo/memory"
	daftartipedokumenusecase "claim-pnc/internal/daftartipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	daftartipedokumenhttp "claim-pnc/internal/daftartipedokumen/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi,
// middleware portal, dan jembatan identitas pemanggil.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa data satu entitas tidak pernah
// terbaca lewat entitas lain, dan bahwa jejak simpan benar-benar terisi dari sesi. Menguji
// handler secara terpisah tidak dapat membuktikan satu pun di antaranya.
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

	documentTypeService, err := daftartipedokumenusecase.NewService(daftartipedokumenusecase.Options{
		RepoSelector: func(alias string) (daftartipedokumen.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)),
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

	handler, err := daftartipedokumenhttp.NewHandler(daftartipedokumenhttp.Options{
		Service: documentTypeService,
		Logger:  logger,
		Caller: func(ctx context.Context) (daftartipedokumenhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return daftartipedokumenhttp.Caller{}, false
			}
			return daftartipedokumenhttp.Caller{Identity: baseCtx.User.Identity}, true
		},
		WriteResponse: writeResponse,
		WriteError:    daftartipedokumenhttp.ErrorWriter(writeError),
	})
	require.NoError(t, err)

	// Hanya ASM dan ASI yang koneksinya "hidup"; sisanya ada di daftar tetapi belum siap.
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
				daftartipedokumenhttp.Mount(protected, handler, portalDeps)
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

const route = "/api/master/tipe-dokumen"

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
	require.Empty(t, content["tipe_dokumen"])
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

func TestNewDocumentTypeGetsItsIDFromTheServer(t *testing.T) {
	// Sentinel "UnknownID" milik sistem lama tidak dibawa: ID tidak pernah ada di badan
	// permintaan sama sekali.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM",
		`{"tipe_dokumen":"Dokumen Investigasi","status_proses":"Investigasi"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, _ := content["tipe_dokumen"].(map[string]any)
	require.Equal(t, "10007", saved["id"])
	require.Equal(t, "Dokumen Investigasi", saved["tipe_dokumen"])
	require.Equal(t, "Investigasi", saved["status_proses"])
}

// TestEmptyInputIsAcceptedLikeInPega mengunci keputusan Work Owner 2026-09-21 pada tingkat
// kontrak API, bukan hanya di domain.
func TestEmptyInputIsAcceptedLikeInPega(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM",
		`{"tipe_dokumen":"","status_proses":""}`)
	require.Equal(t, http.StatusCreated, response.StatusCode,
		"layar ini meniru Pega apa adanya: tanpa validasi")
}

// STS_PROSES adalah TEKS BEBAS, bukan enum. Nilai apa pun diterima — termasuk yang tidak
// ada di daftar mana pun, karena memang tidak ada daftarnya.
func TestProcessStatusAcceptsAnyFreeText(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM",
		`{"tipe_dokumen":"Dokumen Khusus","status_proses":"catatan apa pun boleh di sini"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	saved, _ := content["tipe_dokumen"].(map[string]any)
	require.Equal(t, "catatan apa pun boleh di sini", saved["status_proses"])
}

func TestUnknownFieldIsRejectedRatherThanIgnoredSilently(t *testing.T) {
	// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim", dan pada modul
	// tanpa validasi nilai kosong itu justru akan TERSIMPAN tanpa satu pun tanda.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM", `{"jenis_dokumen":"Dokumen X"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// ID tidak dapat dikirim klien. Bila ia diterima, klien dapat memindahkan satu tipe dokumen
// ke kunci lain — dan kunci itu dirujuk dua master turunan.
func TestIDCannotBeSuppliedByTheClient(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM",
		`{"id":"19999","tipe_dokumen":"Dokumen X"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Jejak simpan diisi server dari sesi, bukan diterima dari klien.
func TestSaveTrailCannotBeSuppliedByTheClient(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM",
		`{"tipe_dokumen":"Dokumen X","user_edit":"ORANGLAIN"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode,
		"USER_EDIT dipakai menelusuri siapa mengubah apa; ia tidak boleh dapat diaku-aku")
}

func TestSavingChangesTheContentsAndKeepsTheID(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/10002", "ASM",
		`{"tipe_dokumen":"Dokumen Survey Lapangan","status_proses":"Survey"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	saved, _ := content["tipe_dokumen"].(map[string]any)
	require.Equal(t, "10002", saved["id"])
	require.Equal(t, "Dokumen Survey Lapangan", saved["tipe_dokumen"])
}

func TestEditingARowThatIsNoLongerThereIsReportedAsNotFound(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/19999", "ASM",
		`{"tipe_dokumen":"Apa saja","status_proses":""}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tipe_dokumen_tidak_ditemukan", content["kode"])
}

func TestOpeningOneRowForEditingReturnsItsCurrentContents(t *testing.T) {
	// Menggantikan CNMSetListDocumentType_act, yang menyalin baris ke TempDcol.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/10002", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	doc, _ := content["tipe_dokumen"].(map[string]any)
	require.Equal(t, "Dokumen Survey", doc["tipe_dokumen"])
	require.Equal(t, "Survey", doc["status_proses"])
}

func TestWritingThroughOneEntityNeverTouchesAnother(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASI", `{"tipe_dokumen":"Hanya milik ASI"}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(6), asm["total"], "isi ASM tidak boleh bertambah")

	_, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(1), asi["total"])
}
