package inboxreceivetkahttp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/inboxreceivetka/notification"
	"claim-pnc/internal/inboxreceivetka/repo/memory"
	inboxreceivetkausecase "claim-pnc/internal/inboxreceivetka/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxreceivetkahttp "claim-pnc/internal/inboxreceivetka/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena uji
// lain kebetulan memakai jalur yang benar.
const (
	listRoute     = "/api/inbox/receive-tka"
	completeRoute = "/api/inbox/receive-tka/kelengkapan-dokumen"
)

// Jumlah baris pada memory.SampleTasks.
const sampleRows = 6

// sessionAt adalah waktu tetap untuk jam modul auth, supaya sesi ujinya tidak kedaluwarsa di
// tengah jalan. Modul ini sendiri TIDAK memakai jam apa pun.
var sessionAt = time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi dan
// middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah
// ditolak, dan bahwa satu entitas tidak pernah menyentuh data entitas lain. Menguji handler
// secara terpisah tidak dapat membuktikan ketiganya, dan pada modul yang MENULIS yang
// ketiga bukan lagi soal kebocoran baca melainkan soal mengubah klaim milik badan hukum
// lain.
type testServer struct {
	server   *httptest.Server
	token    string
	recorder *notification.Recorder

	// asi tidak diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (`ADR-0030`, `R-20`).
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
	recorder := notification.NewRecorder()

	service, err := inboxreceivetkausecase.NewService(inboxreceivetkausecase.Options{
		RepoSelector: func(alias string) (inboxreceivetka.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Notifier: recorder,
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

	handler, err := inboxreceivetkahttp.NewHandler(inboxreceivetkahttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    inboxreceivetkahttp.ErrorWriter(writeError),
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
				inboxreceivetkahttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi, recorder: recorder}
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

// call menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim, dan
// payload kosong berarti tanpa badan.
func (p *testServer) call(
	t *testing.T,
	method, path, portalAlias, payload string,
) (*http.Response, map[string]any) {
	t.Helper()

	var body io.Reader
	if payload != "" {
		body = strings.NewReader(payload)
	}

	request, err := http.NewRequest(method, p.server.URL+path, body)
	require.NoError(t, err)
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}
	if payload != "" {
		request.Header.Set("Content-Type", "application/json")
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

func TestListReturnsAllSampleRows(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, listRoute, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Len(t, tasks(t, content), sampleRows)
	require.Equal(t, false, content["terpotong"])
	require.Equal(t, float64(inboxreceivetka.MaxRows), content["batas_baris"])
	require.Equal(t, "ASM", content["portal"])
}

// Ketujuh field baris dikirim dengan nama yang disepakati kontrak.
//
// Diuji satu per satu, bukan sekadar jumlahnya: nama field yang berubah tidak menghasilkan
// galat apa pun — ia menghasilkan kolom kosong di layar.
func TestListRowCarriesTheAgreedFieldNames(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet,
		listRoute+"?"+search("PNC-1546"), "ASM", "")
	list := tasks(t, content)
	require.Len(t, list, 1)

	row, ok := list[0].(map[string]any)
	require.True(t, ok)

	require.Equal(t, "PNC-1546", row["nomor_klaim"])
	require.Equal(t, "12000000000002", row["nomor_polis"])
	require.Equal(t, "PT Contoh Sejahtera Abadi", row["nama_tertanggung"])
	require.Equal(t, "Peserta Contoh Satu", row["nama_peserta"])
	require.Equal(t, "2021-08-16", row["tanggal_kejadian"])
	require.Equal(t, "2023-05-10", row["tanggal_registrasi"])
	require.Equal(t, true, row["klaim_tersedia"])
	require.NotEmpty(t, row["referensi"])
}

// Tanggal pendaftaran yang tidak tercatat dikirim sebagai null, bukan sebagai tanggal
// karangan.
//
// Kolom sumbernya VARCHAR2, sehingga bentuk yang tidak dikenali tidak dapat diurai. Null
// berarti "tidak diketahui" — berbeda dari "baru masuk".
func TestListSendsNullRegistrationWhenUnparsable(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet,
		listRoute+"?"+search("PNC-1955"), "ASM", "")
	list := tasks(t, content)
	require.Len(t, list, 1)
	require.Nil(t, list[0].(map[string]any)["tanggal_registrasi"])
}

// Tanggal kejadian yang kosong dikirim sebagai null, bukan sebagai teks kosong.
func TestListSendsNullForMissingDate(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet,
		listRoute+"?"+search("PNC-1902"), "ASM", "")
	list := tasks(t, content)
	require.Len(t, list, 1)
	require.Nil(t, list[0].(map[string]any)["tanggal_kejadian"])
}

func TestListRejectsOverlongKeyword(t *testing.T) {
	server := newTestServer(t)

	long := strings.Repeat("a", inboxreceivetkahttp.MaxKeywordLength+1)
	response, content := server.call(t, http.MethodGet,
		listRoute+"?"+search(long), "ASM", "")

	require.Equal(t, http.StatusBadRequest, response.StatusCode,
		"permintaan yang tidak dapat dipenuhi ditolak dengan sebabnya, bukan dijawab "+
			"daftar kosong yang terbaca seperti antrean yang memang habis")
	require.Equal(t, inboxreceivetkahttp.CodeMalformedRequest, content["kode"])
}

// Daftar kosong dikirim sebagai `[]`, bukan `null`.
//
// Inbox yang bersih adalah keadaan yang DIHARAPKAN di sini, bukan keadaan luar biasa.
func TestListOfEmptyPortalIsAnEmptyArray(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, listRoute, "ASI", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotNil(t, content["tugas"])
	require.Empty(t, tasks(t, content))
}

// SELURUH rute menolak permintaan tanpa portal, dan tidak pernah jatuh ke portal utama
// sebagai cadangan (`R-20`, `TKT-F6-002`).
//
// Rute tulisnya diuji bersama rute bacanya: tanpa pemeriksaan portal, satu permintaan dapat
// mengubah tanggal pada klaim milik badan hukum lain.
func TestEveryRouteRequiresPortal(t *testing.T) {
	server := newTestServer(t)

	t.Run("baca", func(t *testing.T) {
		response, _ := server.call(t, http.MethodGet, listRoute, "", "")
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("tulis", func(t *testing.T) {
		response, _ := server.call(t, http.MethodPost, completeRoute, "",
			`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`)
		require.Equal(t, http.StatusBadRequest, response.StatusCode)

		// Yang terpenting: TIDAK ADA yang berubah.
		page, err := server.asm.List(t.Context(), inboxreceivetka.Filter{})
		require.NoError(t, err)
		require.Len(t, page.Tasks, sampleRows)
	})
}

func TestEveryRouteRequiresSession(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	for _, one := range []struct {
		name, method, path, payload string
	}{
		{name: "baca", method: http.MethodGet, path: listRoute},
		{
			name: "tulis", method: http.MethodPost, path: completeRoute,
			payload: `{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`,
		},
	} {
		t.Run(one.name, func(t *testing.T) {
			response, _ := server.call(t, one.method, one.path, "ASM", one.payload)
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		})
	}
}

// Jalur lengkap tombol Submit: tersimpan, barisnya hilang, surel terkirim.
func TestCompleteSavesAndRemovesRow(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
		`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	require.Equal(t, "PNC-1546", content["nomor_klaim"])
	require.Equal(t, "2026-09-24", content["tanggal_dokumen_lengkap"])
	require.Equal(t, true, content["pemberitahuan_dicoba"])
	require.Equal(t, true, content["pemberitahuan_terkirim"])
	require.Equal(t, "ASM", content["portal"])

	_, list := server.call(t, http.MethodGet, listRoute, "ASM", "")
	require.Len(t, tasks(t, list), sampleRows-1)

	require.Len(t, server.recorder.Notices(), 1)
}

// Submit kedua atas baris yang sama ditolak 409, dan surelnya TIDAK dikirim dua kali.
//
// Inilah yang menggantikan kunci idempotensi `10-API-STRATEGY.md` §7 pada modul ini.
func TestCompleteTwiceIsRejectedAndSendsOneEmail(t *testing.T) {
	server := newTestServer(t)
	payload := `{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`

	response, _ := server.call(t, http.MethodPost, completeRoute, "ASM", payload)
	require.Equal(t, http.StatusOK, response.StatusCode)

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM", payload)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, inboxreceivetkahttp.CodeTaskNotFound, content["kode"])

	require.Len(t, server.recorder.Notices(), 1, "surel tidak boleh terkirim dua kali")
}

// Tanggal kosong ditolak 422 dengan kalimat yang SAMA PERSIS seperti sistem lama.
func TestCompleteRejectsEmptyDateWithTheOldMessage(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
		`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":""}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode,
		"bentuk permintaannya benar; isinya yang melanggar aturan bisnis")
	require.Equal(t, inboxreceivetkahttp.CodeDateRequired, content["kode"])
	require.Equal(t, "Silahkan isi tanggal terlebih dahulu", content["pesan"],
		"kalimatnya ditiru apa adanya dari Local.ErrMessages, termasuk ejaan 'Silahkan'")
}

// Tanggal yang bentuknya rusak adalah cacat FRONTEND, dijawab 400 — dibedakan dari tanggal
// kosong yang merupakan kesalahan pengguna.
func TestCompleteRejectsMalformedDateWith400(t *testing.T) {
	server := newTestServer(t)

	for _, bad := range []string{"24-09-2026", "2026/09/24", "besok", "2026-13-45"} {
		t.Run(bad, func(t *testing.T) {
			response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
				`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"`+bad+`"}`)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, inboxreceivetkahttp.CodeMalformedRequest, content["kode"])
		})
	}
}

// Baris YATIM ditolak dengan kode TERSENDIRI, bukan disamakan dengan "tidak ditemukan".
//
// Tindakannya berbeda: menyegarkan daftar tidak akan menolong, karena barisnya akan muncul
// lagi dan gagal lagi. Layar harus mengatakannya begitu.
func TestCompleteRejectsOrphanClaimWithItsOwnCode(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
		`{"nomor_klaim":"PNC-1977","tanggal_dokumen_lengkap":"2026-09-24"}`)

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, inboxreceivetkahttp.CodeClaimMissing, content["kode"])
	require.Empty(t, server.recorder.Notices())
}

// Surel yang gagal TIDAK membatalkan penyimpanan, dan layar diberi tahu persis begitu.
func TestCompleteStillSavesWhenEmailFails(t *testing.T) {
	server := newTestServer(t)
	server.recorder.SetError(errors.New("server surel tidak dapat dihubungi"))

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
		`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`)

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, true, content["pemberitahuan_dicoba"])
	require.Equal(t, false, content["pemberitahuan_terkirim"])

	_, list := server.call(t, http.MethodGet, listRoute, "ASM", "")
	require.Len(t, tasks(t, list), sampleRows-1, "tanggalnya tetap tersimpan")
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam.
//
// Jawaban daftar memuat tujuh field per baris sementara permintaan ini hanya menerima dua;
// klien yang mengirim balik seluruh objek baris harus mengetahuinya saat pertama dicoba.
func TestCompleteRejectsUnknownFields(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, completeRoute, "ASM",
		`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24","aging":"27"}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, inboxreceivetkahttp.CodeMalformedRequest, content["kode"])
}

// Satu entitas tidak dapat menyentuh pekerjaan entitas lain.
//
// Pada modul baca-saja ini berarti kebocoran; pada modul yang MENULIS ia berarti mengubah
// tanggal pada klaim milik badan hukum lain (`R-20`).
func TestOnePortalCannotCompleteAnotherPortalsWork(t *testing.T) {
	server := newTestServer(t)

	// PNC-1546 hanya ada di ASM.
	response, content := server.call(t, http.MethodPost, completeRoute, "ASI",
		`{"nomor_klaim":"PNC-1546","tanggal_dokumen_lengkap":"2026-09-24"}`)

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Equal(t, inboxreceivetkahttp.CodeTaskNotFound, content["kode"])

	page, err := server.asm.List(t.Context(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	require.Len(t, page.Tasks, sampleRows, "daftar ASM tidak boleh tersentuh")
}

// Portal yang koneksinya belum hidup dijawab 503, bukan daftar kosong maupun 500.
func TestNotReadyPortalIsRejected(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, listRoute, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
}
