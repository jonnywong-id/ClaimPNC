package detailpenyebabhttp_test

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
	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/detailpenyebab/repo/memory"
	detailpenyebabusecase "claim-pnc/internal/detailpenyebab/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	detailpenyebabhttp "claim-pnc/internal/detailpenyebab/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const loginName = "adminpnc"

// Jalur dasar modul ini. Disebut sekali supaya salah ketik pada satu uji tidak lolos karena
// uji lain kebetulan memakai jalur yang benar.
const route = "/api/master/detail-penyebab"

// Baris contoh pada memory.NewSampleRepo.
const (
	sampleRows = 8

	// Baris ber-TIGA lini bisnis.
	rowWithThreeBusiness = "990004"

	// Baris YATIM — induknya tidak ada di daftar master contoh.
	orphanRow = "990008"
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

	// asi tidak diberi satu baris pun; ia yang membuktikan pemisahan antarentitas
	// (ADR-0030, R-20). Pada modul ini yang bocor bukan sekadar daftar acuan: sebab
	// kerugian yang dapat dipilih menentukan bagaimana klaim dinilai pada badan hukum itu.
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

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo()

	service, err := detailpenyebabusecase.NewService(
		detailpenyebabusecase.Options{
			RepoSelector: func(alias string) (detailpenyebab.Store, error) {
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

	handler, err := detailpenyebabhttp.NewHandler(
		detailpenyebabhttp.Options{
			Service: service,
			Caller: func(ctx context.Context) (detailpenyebabhttp.Caller, bool) {
				baseCtx, existing := authhttp.CallerFromContext(ctx)
				if !existing {
					return detailpenyebabhttp.Caller{}, false
				}
				return detailpenyebabhttp.Caller{Login: baseCtx.User.Login}, true
			},
			Logger:        logger,
			WriteResponse: writeResponse,
			WriteError:    detailpenyebabhttp.ErrorWriter(writeError),
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
				detailpenyebabhttp.Mount(protected, handler, portalDeps)
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

// detailList membaca daftar dari badan jawaban.
func detailList(t *testing.T, content map[string]any) []any {
	t.Helper()
	list, existing := content["detail"].([]any)
	require.True(t, existing, "badan jawaban tidak memuat daftar detail: %v", content)
	return list
}

func TestPermintaanTanpaSesiDitolak(t *testing.T) {
	server := newTestServer(t)
	server.token = ""

	response, _ := server.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestPermintaanTanpaPortalDitolakAlihAlihJatuhKePortalUtama(t *testing.T) {
	// Inilah yang mencegah R-20. Permintaan tanpa portal TIDAK boleh dilayani koneksi
	// bawaan — layarnya akan tampil normal, angkanya masuk akal, dan yang salah hanya
	// milik siapa datanya.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestPortalYangBelumSiapDitolakDenganPesanYangBerbeda(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.NotEmpty(t, content["kode"])
}

func TestDaftarHanyaMemuatBarisPortalYangDiminta(t *testing.T) {
	server := newTestServer(t)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.Len(t, detailList(t, asm), sampleRows)

	// ASI tidak diberi satu baris pun.
	_, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Empty(t, detailList(t, asi))
}

func TestDaftarTidakMemuatLiniBisnisKarenaGridTidakMenampilkannya(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, route, "ASM", "")
	for _, row := range detailList(t, content) {
		one := row.(map[string]any)
		require.Empty(t, one["bisnis"],
			"daftar seharusnya tidak memuat lini bisnis: %v", one["id"])
	}
}

func TestSatuBarisDikirimLengkapDenganLiniBisnisnya(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/"+rowWithThreeBusiness, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	one := content["detail"].(map[string]any)
	require.Len(t, one["bisnis"], 3)
}

func TestLabelStatusAktifDiturunkanServer(t *testing.T) {
	// Menurunkannya di frontend berarti aturan itu hidup di dua tempat.
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, route, "ASM", "")
	for _, row := range detailList(t, content) {
		one := row.(map[string]any)
		require.NotEmpty(t, one["label_status_aktif"],
			"label status aktif kosong pada baris %v", one["id"])
	}
}

func TestBarisYatimTetapMunculDenganSebutanIndukKosong(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/"+orphanRow, "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	one := content["detail"].(map[string]any)
	require.Empty(t, one["nama_master"])
	require.NotEmpty(t, one["id_master"], "baris yatim tetap menyimpan kode induknya")
}

func TestBarisYangTidakAdaDijawabTidakDitemukan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/tidak-ada", "ASM", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])
}

func TestMenambahBarisMenjawab201BesertaIDYangDiterbitkan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{
		"id_lama": "",
		"id_master": "9001",
		"deskripsi_kerugian": "Kebakaran akibat ledakan tabung gas",
		"kode_kehilangan": "FIRE-09",
		"status_aktif": "1",
		"bisnis": [{"id": "006", "nama": "Fire / Property"}]
	}`)

	require.Equal(t, http.StatusCreated, response.StatusCode)
	one := content["detail"].(map[string]any)
	require.NotEmpty(t, one["id"], "ID diterbitkan server, layar tidak punya cara lain mengetahuinya")
	require.Equal(t, "Kebakaran", one["nama_master"], "sebutan induk ikut diturunkan")
	require.Equal(t, "Aktif", one["label_status_aktif"])
}

func TestBarisBaruMasukKePortalYangDimintaSaja(t *testing.T) {
	server := newTestServer(t)

	_, _ = server.call(t, http.MethodPost, route, "ASI", `{
		"id_lama": "", "id_master": "", "deskripsi_kerugian": "Hanya milik ASI",
		"kode_kehilangan": "", "status_aktif": "1", "bisnis": []
	}`)

	_, asi := server.call(t, http.MethodGet, route, "ASI", "")
	require.Len(t, detailList(t, asi), 1)

	_, asm := server.call(t, http.MethodGet, route, "ASM", "")
	require.Len(t, detailList(t, asm), sampleRows, "baris ASI tidak boleh muncul di ASM")
}

func TestSeluruhIsianKosongDiterimaKarenaPegaTidakMemeriksaApaPun(t *testing.T) {
	// Kesetaraan perilaku (P-5); lihat detailpenyebab.Input.Check.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPost, route, "ASM", `{
		"id_lama": "", "id_master": "", "deskripsi_kerugian": "",
		"kode_kehilangan": "", "status_aktif": "", "bisnis": []
	}`)
	require.Equal(t, http.StatusCreated, response.StatusCode)
}

func TestStatusAktifDiLuarPilihannyaDijawab422(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{
		"id_lama": "", "id_master": "", "deskripsi_kerugian": "apa saja",
		"kode_kehilangan": "", "status_aktif": "9", "bisnis": []
	}`)

	// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.NotEmpty(t, content["detail"])
}

func TestFieldYangTidakDikenalDitolakAlihAlihDiabaikanDiamDiam(t *testing.T) {
	// `id`, `nama_master`, dan `label_status_aktif` DIKIRIM pada setiap jawaban tetapi
	// tidak dapat dikirim balik. Klien yang mengembalikan seluruh objek apa adanya harus
	// mengetahuinya saat pertama dicoba.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPost, route, "ASM", `{
		"id": "990001",
		"deskripsi_kerugian": "apa saja"
	}`)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestMenyimpanPerubahanMenjawab200DenganBarisTerbaru(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodPut, route+"/990001", "ASM", `{
		"id_lama": "COL-0001",
		"id_master": "9002",
		"deskripsi_kerugian": "Deskripsi yang sudah diperbaiki",
		"kode_kehilangan": "FIRE-01",
		"status_aktif": "0",
		"bisnis": [{"id": "003", "nama": "Aneka"}]
	}`)

	require.Equal(t, http.StatusOK, response.StatusCode)
	one := content["detail"].(map[string]any)
	require.Equal(t, "Deskripsi yang sudah diperbaiki", one["deskripsi_kerugian"])
	require.Equal(t, "Tidak Aktif", one["label_status_aktif"])
	require.Equal(t, "Kecelakaan Diri", one["nama_master"], "induk baru ikut diturunkan")
}

func TestIDWarisanTidakHilangSaatLayarTidakMengirimkannya(t *testing.T) {
	// OLD_D_COL_ID tidak digambar di layar mana pun; klien yang tidak mengirimnya tidak
	// boleh menghapusnya — lihat usecase.Service.Save.
	server := newTestServer(t)

	_, content := server.call(t, http.MethodPut, route+"/990001", "ASM", `{
		"id_lama": "",
		"id_master": "9001",
		"deskripsi_kerugian": "Kebakaran akibat hubungan arus pendek",
		"kode_kehilangan": "FIRE-01",
		"status_aktif": "1",
		"bisnis": []
	}`)

	one := content["detail"].(map[string]any)
	require.Equal(t, "COL-0001", one["id_lama"])
}

func TestMenyimpanBarisYangSudahTidakAdaDijawabTidakDitemukan(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodPut, route+"/tidak-ada", "ASM", `{
		"id_lama": "", "id_master": "", "deskripsi_kerugian": "apa saja",
		"kode_kehilangan": "", "status_aktif": "", "bisnis": []
	}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestTidakAdaRuteHapus(t *testing.T) {
	// Layar lamanya tidak punya tombolnya, dan D-66 melarang penghapusan fisik.
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodDelete, route+"/990001", "ASM", "")
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

func TestPenyaringIndukMempersempitDaftar(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, route+"?id_master=9001", "ASM", "")
	require.Len(t, detailList(t, content), 2)
}

func TestPenyaringLiniBisnisMempersempitDaftar(t *testing.T) {
	server := newTestServer(t)

	_, content := server.call(t, http.MethodGet, route+"?bisnis=005", "ASM", "")
	list := detailList(t, content)
	require.Len(t, list, 1)
	require.Equal(t, orphanRow, list[0].(map[string]any)["id"])
}

func TestKataPencarianTerlaluPanjangDitolak(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet,
		route+"?cari="+strings.Repeat("a", 101), "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", content["kode"])
}

func TestPencarianIndukMenjawabDaftarPilihan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet,
		route+"/pilihan/master?cari=Keba", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, existing := content["master"].([]any)
	require.True(t, existing)
	require.Len(t, list, 1)
	require.Equal(t, "9001 — Kebakaran", list[0].(map[string]any)["label"])
}

func TestPencarianLiniBisnisMenjawabDaftarPilihan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet,
		route+"/pilihan/bisnis?cari=Marine", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, existing := content["bisnis"].([]any)
	require.True(t, existing)
	require.Len(t, list, 1)
}

func TestDaftarPilihanStatusAktifTidakMenuntutPortal(t *testing.T) {
	// Isinya konstanta domain, sama di keempat portal — lihat Handler.Options.
	server := newTestServer(t)

	response, content := server.call(t, http.MethodGet, route+"/pilihan", "", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, existing := content["status_aktif"].([]any)
	require.True(t, existing)
	require.Len(t, list, 2)
}

func TestJalurPilihanTidakTertukarDenganJalurSatuBaris(t *testing.T) {
	// `/pilihan` adalah segmen statis dan TIDAK boleh dibaca sebagai {id}. Bila urutan
	// pendaftaran rutenya keliru, permintaan ini akan dijawab "tidak ditemukan".
	server := newTestServer(t)

	response, _ := server.call(t, http.MethodGet, route+"/pilihan", "", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
}
