package daftardetailtipedokumenhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/repo/memory"
	daftardetailtipedokumenusecase "claim-pnc/internal/daftardetailtipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	daftardetailtipedokumenhttp "claim-pnc/internal/daftardetailtipedokumen/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi,
// middleware portal, dan jembatan identitas pemanggil.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, bahwa data satu entitas tidak pernah
// terbaca lewat entitas lain, dan bahwa daftar pilihan tetap menjawab ketika salah satu
// masternya bermasalah. Menguji handler secara terpisah tidak dapat membuktikan satu pun
// di antaranya.
type testServer struct {
	server *httptest.Server
	token  string

	// asi sengaja dibiarkan kosong; ia yang membuktikan pemisahan antarentitas.
	asm       *memory.Repo
	asi       *memory.Repo
	reference *memory.ReferenceRepo
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

	reference := memory.NewSampleReferenceRepo()

	asm := memory.NewRepo(memory.SampleList()...)
	asm.UseReferences(reference)
	asi := memory.NewRepo()
	asi.UseReferences(reference)

	detailService, err := daftardetailtipedokumenusecase.NewService(daftardetailtipedokumenusecase.Options{
		RepoSelector: func(alias string) (daftardetailtipedokumen.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		ReferenceSelector: func(alias string) (daftardetailtipedokumen.ReferenceRepo, error) {
			switch alias {
			case "ASM", "ASI":
				return reference, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)),
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

	handler, err := daftardetailtipedokumenhttp.NewHandler(daftardetailtipedokumenhttp.Options{
		Service: detailService,
		Logger:  logger,
		Caller: func(ctx context.Context) (daftardetailtipedokumenhttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return daftardetailtipedokumenhttp.Caller{}, false
			}
			return daftardetailtipedokumenhttp.Caller{Identity: baseCtx.User.Identity}, true
		},
		WriteResponse: writeResponse,
		WriteError:    daftardetailtipedokumenhttp.ErrorWriter(writeError),
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
				daftardetailtipedokumenhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm, asi: asi, reference: reference}
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

const route = "/api/master/detail-tipe-dokumen"

func TestListRepliesWithTheEntityThatAnsweredIt(t *testing.T) {
	// Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
	// diandaikan — server menyebutkannya (`R-20`).
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route, "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, float64(5), content["total"])
}

// Berpindah entitas berarti berpindah data — bukan menyaring baris yang sama.
//
// Inilah uji yang paling langsung menjawab `R-20`: kebocoran antarbadan hukum tidak
// terlihat sebagai galat, hanya sebagai daftar yang "masuk akal" tetapi milik orang lain.
func TestListOfOneEntityNeverLeaksIntoAnother(t *testing.T) {
	server := newTestServer(t)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(5), asm["total"])

	_, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(0), asi["total"])
	require.Equal(t, "ASI", asi["portal"])
}

// Permintaan tanpa portal DITOLAK, bukan dijatuhkan ke portal utama.
//
// Jatuh ke koneksi baku adalah salah satu dari dua jalur kegagalan `R-20` yang sudah
// terbaca di rancangan.
func TestRequestWithoutPortalIsRejected(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "", "")

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Rutenya berada di balik sesi.
func TestRoutesRequireASession(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	response, _ := server.call(t, http.MethodGet, route, "ASM", "")

	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Daftar TIDAK membawa aturan bisnis; satu baris membawanya.
//
// Uji ini mengunci perbedaan yang paling mudah dilanggar tanpa disadari: layar yang
// memakai baris dari daftar untuk mengisi form akan tampak seolah seluruh lini bisnisnya
// sudah dihapus — dan menyimpannya benar-benar menghapusnya.
func TestListOmitsBusinessRulesButGetCarriesThem(t *testing.T) {
	server := newTestServer(t)

	_, list := server.call(t, http.MethodGet, route, "ASM", "")
	rows := list["detail_tipe_dokumen"].([]any)
	first := rows[0].(map[string]any)
	require.Equal(t, "100001", first["id"])
	require.Empty(t, first["bisnis"])

	_, single := server.call(t, http.MethodGet, route+"/100001", "ASM", "")
	detail := single["detail_tipe_dokumen"].(map[string]any)
	businesses := detail["bisnis"].([]any)
	require.Len(t, businesses, 2)
}

// SATU keterangan hasil join, DUA keterangan tersimpan — dan pembedaannya yang diuji.
//
// `nama_tipe_dokumen` dan `nama_bisnis` tidak pernah ada di halaman `TempDTDoc`, sehingga
// keduanya mustahil tersimpan dan pasti hasil join. Kedua keterangan lain ADA di halaman
// itu, sehingga keduanya tersimpan di kolomnya sendiri.
func TestOnlyDocumentTypeNameAndBusinessNameAreJoined(t *testing.T) {
	server := newTestServer(t)

	_, single := server.call(t, http.MethodGet, route+"/100001", "ASM", "")
	detail := single["detail_tipe_dokumen"].(map[string]any)

	// Hasil join: terisi meski data contoh tidak menyimpannya.
	require.Equal(t, "10001", detail["id_tipe_dokumen"])
	require.Equal(t, "Dokumen Registrasi", detail["nama_tipe_dokumen"])

	// Tersimpan: terbaca apa adanya dari barisnya.
	require.Equal(t, "Contoh Golongan A", detail["keterangan_penyebab_kerugian"])
	require.Equal(t, "Polis Asli", detail["keterangan_objek_dokumen"])

	businesses := detail["bisnis"].([]any)
	first := businesses[0].(map[string]any)
	require.NotEmpty(t, first["nama_bisnis"])
}

// Keterangan yang diketik bebas — tanpa kode — TERSIMPAN dan terbaca kembali utuh.
//
// Inilah uji yang membuktikan kedua keterangan itu memang kolom, bukan hasil join. Bila
// keduanya dijoin dari master, isian ini akan kembali KOSONG karena kodenya tidak ada —
// dan petugas kehilangan apa yang baru saja diketiknya tanpa satu pun pesan.
func TestFreeTypedDescriptionSurvivesWithoutACode(t *testing.T) {
	server := newTestServer(t)

	body := `{
		"id_tipe_dokumen": "10001",
		"detail_dokumen": "Dokumen dengan keterangan bebas",
		"status_tertanggung": "",
		"id_penyebab_kerugian": "",
		"keterangan_penyebab_kerugian": "Keterangan yang tidak ada di master",
		"id_objek_dokumen": "",
		"keterangan_objek_dokumen": "Objek yang tidak ada di master",
		"resiko": "",
		"bisnis": []
	}`
	response, content := server.call(t, http.MethodPost, route, "ASM", body)

	require.Equal(t, http.StatusCreated, response.StatusCode)
	detail := content["detail_tipe_dokumen"].(map[string]any)
	require.Equal(t, "", detail["id_penyebab_kerugian"])
	require.Equal(t, "Keterangan yang tidak ada di master", detail["keterangan_penyebab_kerugian"])
	require.Equal(t, "", detail["id_objek_dokumen"])
	require.Equal(t, "Objek yang tidak ada di master", detail["keterangan_objek_dokumen"])
}

// Baris yang rujukannya sudah tidak ada di master TETAP TAMPIL.
//
// Ini penyimpangan yang disengaja dari kueri lama, yang memakai INNER JOIN dan
// menyembunyikannya. Baris yang hilang dari layar master tidak dapat diperbaiki petugas,
// sementara aturan yang tak terlihat itu TETAP BERLAKU pada klaim.
//
// Uji ini sekaligus memperlihatkan pembedaan join versus tersimpan pada satu baris yang
// sama: yang DIJOIN kosong karena kodenya tidak ada di master, sementara yang TERSIMPAN
// tetap terbaca utuh.
func TestRowsWithMissingMasterReferencesAreStillReturned(t *testing.T) {
	server := newTestServer(t)

	_, single := server.call(t, http.MethodGet, route+"/100005", "ASM", "")
	detail := single["detail_tipe_dokumen"].(map[string]any)

	require.Equal(t, "19999", detail["id_tipe_dokumen"])

	// Dijoin -> kosong, karena kodenya tidak ada di master.
	require.Equal(t, "", detail["nama_tipe_dokumen"])

	// Tersimpan -> tetap terbaca, meski kodenya pun tidak ada di master.
	require.Equal(t, "Objek Warisan Tanpa Master", detail["keterangan_objek_dokumen"])
	require.Equal(t, "Keterangan Warisan Tanpa Master", detail["keterangan_penyebab_kerugian"])

	businesses := detail["bisnis"].([]any)
	require.Len(t, businesses, 1)
	require.Equal(t, "099", businesses[0].(map[string]any)["id_bisnis"])
	// Nama bisnis dijoin -> kosong.
	require.Equal(t, "", businesses[0].(map[string]any)["nama_bisnis"])
}

// Penambahan menerbitkan ID di server dan mengembalikannya.
func TestCreateIssuesTheIDOnTheServer(t *testing.T) {
	server := newTestServer(t)

	body := `{
		"id_tipe_dokumen": "10003",
		"detail_dokumen": "Berita Acara Komite",
		"status_tertanggung": "Tertanggung",
		"id_penyebab_kerugian": "1005",
		"keterangan_penyebab_kerugian": "Contoh Golongan E",
		"id_objek_dokumen": "10001",
		"keterangan_objek_dokumen": "KTP Tertanggung",
		"resiko": "0",
		"bisnis": [{"id_bisnis": "003", "status_wajib": true, "minimum_dokumen": 1}]
	}`
	response, content := server.call(t, http.MethodPost, route, "ASM", body)

	require.Equal(t, http.StatusCreated, response.StatusCode)
	detail := content["detail_tipe_dokumen"].(map[string]any)
	require.NotEmpty(t, detail["id"])
	require.Equal(t, "Berita Acara Komite", detail["detail_dokumen"])
	require.Equal(t, "Dokumen Komite", detail["nama_tipe_dokumen"])
}

// Penyuntingan MENGGANTI seluruh daftar bisnis, bukan menambahinya.
//
// Grid di form memang mengirim susunan akhir yang dikehendaki petugas, dan tidak ada satu
// pun penanda di sana yang menyatakan baris mana yang baru, mana yang berubah, dan mana
// yang dibuang.
func TestUpdateReplacesTheWholeBusinessList(t *testing.T) {
	server := newTestServer(t)

	body := `{
		"id_tipe_dokumen": "10001",
		"detail_dokumen": "Formulir Laporan Kerugian",
		"status_tertanggung": "Tertanggung",
		"id_penyebab_kerugian": "1001",
		"keterangan_penyebab_kerugian": "Contoh Golongan A",
		"id_objek_dokumen": "10002",
		"keterangan_objek_dokumen": "Polis Asli",
		"resiko": "0",
		"bisnis": [{"id_bisnis": "005", "status_wajib": false, "minimum_dokumen": 3}]
	}`
	response, content := server.call(t, http.MethodPut, route+"/100001", "ASM", body)

	require.Equal(t, http.StatusOK, response.StatusCode)
	detail := content["detail_tipe_dokumen"].(map[string]any)
	businesses := detail["bisnis"].([]any)
	require.Len(t, businesses, 1)

	only := businesses[0].(map[string]any)
	require.Equal(t, "005", only["id_bisnis"])
	require.Equal(t, false, only["status_wajib"])
	require.Equal(t, float64(3), only["minimum_dokumen"])
}

// Menyunting baris yang tidak ada dijawab 404 dengan kode yang dikenali frontend.
func TestUpdateOfAMissingRowIsNotFound(t *testing.T) {
	server := newTestServer(t)

	body := `{"id_tipe_dokumen":"","detail_dokumen":"x","status_tertanggung":"","id_penyebab_kerugian":"","id_objek_dokumen":"","resiko":"","bisnis":[]}`
	response, content := server.call(t, http.MethodPut, route+"/999999", "ASM", body)

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

// Isian yang terlalu panjang dijawab 422 beserta SELURUH pelanggarannya.
func TestOverlongFieldsAreRejectedWithEveryViolation(t *testing.T) {
	server := newTestServer(t)

	tooLong := strings.Repeat("x", daftardetailtipedokumen.MaxDetailLength+1)
	tooLongStatus := strings.Repeat("y", daftardetailtipedokumen.MaxInsuredStatusLength+1)
	body := `{
		"id_tipe_dokumen": "10001",
		"detail_dokumen": "` + tooLong + `",
		"status_tertanggung": "` + tooLongStatus + `",
		"id_penyebab_kerugian": "",
		"id_objek_dokumen": "",
		"resiko": "",
		"bisnis": []
	}`
	response, content := server.call(t, http.MethodPost, route, "ASM", body)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.Len(t, content["detail"].([]any), 2)
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam.
//
// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah.
func TestUnknownFieldsAreRejected(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{"detail_dokumn":"salah ketik"}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Daftar pilihan membawa keempat master sekaligus.
func TestReferenceListCarriesAllFourMastersAtOnce(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/pilihan", "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Len(t, content["tipe_dokumen"].([]any), 6)
	require.Len(t, content["penyebab_kerugian"].([]any), 10)
	require.Len(t, content["objek_dokumen"].([]any), 4)
	require.Len(t, content["bisnis"].([]any), 5)
	require.Empty(t, content["tidak_tersedia"])
	require.Equal(t, "ASM", content["portal"])
}

// Rute `/pilihan` TIDAK tertangkap `/{id}`.
//
// Uji ini menjaga hal yang mudah rusak tanpa disadari: bila urutannya kelak berubah,
// `/pilihan` akan dibaca sebagai ID dan dijawab 404 — dan form akan kehilangan seluruh
// daftar pilihannya tanpa satu pun pesan yang menjelaskan sebabnya.
func TestReferenceRouteIsNotSwallowedByTheIDRoute(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/pilihan", "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Nil(t, content["detail_tipe_dokumen"])
}

// Satu master yang gagal dibaca TIDAK menggagalkan seluruh daftar pilihan.
//
// Keempat kode boleh diketik sendiri, sehingga daftar yang gagal dimuat hanya
// menghilangkan kenyamanan memilih — bukan kemampuan menyimpan. Yang hilang DISEBUTKAN,
// bukan disembunyikan sebagai daftar kosong yang terbaca "masternya memang kosong".
func TestOneBrokenMasterDoesNotBreakTheWholeChoiceList(t *testing.T) {
	server := newTestServer(t)
	server.reference.SetError(errors.New("master tidak dapat dibaca"))

	response, content := server.call(t, http.MethodGet, route+"/pilihan", "ASM", "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Empty(t, content["tipe_dokumen"])
	require.Len(t, content["tidak_tersedia"].([]any), 4)
}

// Penyimpanan tetap berjalan meski daftar pilihannya sedang tidak dapat dibaca.
func TestSavingStillWorksWhileTheChoiceListIsBroken(t *testing.T) {
	server := newTestServer(t)
	server.reference.SetError(errors.New("master tidak dapat dibaca"))

	body := `{
		"id_tipe_dokumen": "10001",
		"detail_dokumen": "Disimpan saat master bermasalah",
		"status_tertanggung": "",
		"id_penyebab_kerugian": "",
		"id_objek_dokumen": "",
		"resiko": "",
		"bisnis": []
	}`
	response, _ := server.call(t, http.MethodPost, route, "ASM", body)

	require.Equal(t, http.StatusCreated, response.StatusCode)
}
