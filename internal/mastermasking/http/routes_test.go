package mastermaskinghttp_test

import (
	"bytes"
	"context"
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
	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/mastermasking/repo/memory"
	mastermaskingusecase "claim-pnc/internal/mastermasking/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	mastermaskinghttp "claim-pnc/internal/mastermasking/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

const route = "/api/master/masking"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa permintaan tanpa portal yang
// sah ditolak. Menguji handler secara terpisah tidak dapat membuktikan keduanya, dan pada
// modul ini keduanya menentukan apakah daftar kewenangan data pribadi satu badan hukum
// dapat terbaca dari badan hukum lain (`R-20`).
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewRepo(memory.SampleList()...).WithBranches(memory.SampleBranches()...)
	// Petugas per cabang ikut dimuat: tanpa itu, form Tambah tidak punya satu baris pun
	// untuk diisi, dan alurnya tidak dapat diuji sampai tersimpan.
	for branchID, operator := range memory.SampleOperators() {
		asm = asm.WithOperators(branchID, operator...)
	}
	asi := memory.NewRepo().WithBranches(memory.SampleBranches()...)

	maskingService, err := mastermaskingusecase.NewService(mastermaskingusecase.Options{
		RepoSelector: func(alias string) (mastermasking.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Now: func() time.Time { return time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC) },
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

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

	handler, err := mastermaskinghttp.NewHandler(mastermaskinghttp.Options{
		Service: maskingService,
		Caller: func(ctx context.Context) (mastermaskinghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return mastermaskinghttp.Caller{}, false
			}
			return mastermaskinghttp.Caller{
				Identity: baseCtx.User.Identity,
				Name:     baseCtx.User.Name,
			}, true
		},
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
				mastermaskinghttp.Mount(protected, handler, portalDeps)
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

// ── Sesi dan portal ─────────────────────────────────────────────────────────────

// Rutenya berada di balik sesi. Tanpa token, permintaan ditolak sebelum menyentuh
// pemeriksaan portal maupun penyimpanan.
func TestWithoutSessionRejected(t *testing.T) {
	p := newTestServer(t)
	p.token = ""

	response, _ := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Permintaan tanpa portal DITOLAK, tidak pernah dialihkan ke portal utama sebagai cadangan.
//
// Inilah `TKT-F6-002`. Mengalihkannya diam-diam akan menampilkan daftar kewenangan data
// pribadi entitas lain pada layar yang tampak normal — kegagalan yang tidak memberi satu
// pun tanda.
func TestWithoutPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route, "", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_disebut", body["kode"])
}

// Portal yang belum siap dibedakan dari portal yang tidak dikenal.
func TestPortalNotReady(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route, "SMAS", "")
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "portal_belum_siap", body["kode"])
}

// Dua entitas menjawab dari penyimpanan yang berbeda.
//
// Uji ini adalah pembuktian `ADR-0030`: pemisahan ada di tingkat koneksi, bukan penyaringan
// baris. ASI sengaja dibiarkan kosong supaya kebocoran dari ASM langsung terlihat.
func TestPortalsAreSeparated(t *testing.T) {
	p := newTestServer(t)

	_, asm := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(5), asm["total"])
	require.Equal(t, "ASM", asm["portal"])

	_, asi := p.call(t, http.MethodGet, route, "ASI", "")
	require.Equal(t, float64(0), asi["total"])
	require.Equal(t, "ASI", asi["portal"])
}

// ── Daftar dan pencarian ────────────────────────────────────────────────────────

// Daftar memuat baris aktif MAUPUN nonaktif secara baku — `SearchData.Type == "1"`.
func TestListIncludesInactive(t *testing.T) {
	p := newTestServer(t)

	_, all := p.call(t, http.MethodGet, route, "ASM", "")
	require.Equal(t, float64(5), all["total"])
}

// Pencarian menurut status — tipe keempat layar lama — menyaring kedua arahnya.
func TestSearchByStatus(t *testing.T) {
	p := newTestServer(t)

	_, active := p.call(t, http.MethodGet, route+"?cari_di=status&kata_kunci=AKTIF", "ASM", "")
	require.Equal(t, float64(4), active["total"])

	_, inactive := p.call(t, http.MethodGet,
		route+"?cari_di=status&kata_kunci="+url.QueryEscape("TIDAK AKTIF"), "ASM", "")
	require.Equal(t, float64(1), inactive["total"])
}

// Pencarian status tanpa menyebut statusnya ditolak dengan pesan layar lama.
func TestSearchByStatusWithoutValue(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"?cari_di=status", "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "status_belum_dipilih", body["kode"])
}

// Pencarian berdasarkan cabang menelusuri NAMA cabang.
func TestSearchByBranch(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodGet, route+"?cari_di=cabang&kata_kunci=malang", "ASM", "")
	require.Equal(t, float64(1), body["total"])
}

// Tipe pencarian yang tidak dikenal ditolak sebagai permintaan cacat.
//
// Mengabaikannya akan menampilkan SELURUH baris kepada pengguna yang mengira ia sedang
// menyaring — pada layar ini, itu berarti membuka seluruh peta kewenangan tanpa diminta.
func TestUnknownSearchTypeRejected(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"?cari_di=modul&kata_kunci=x", "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "permintaan_cacat", body["kode"])
}

// Kolom PASSWORD tidak pernah dikirim ke peramban.
//
// Uji ini menjaga keputusan yang paling mudah batal tanpa disadari saat DTO disunting
// kelak. Isinya literal coba-coba yang tertinggal di sistem lama, dan tidak ada satu pun
// alasan ia sampai ke layar.
func TestPasswordNeverLeavesServer(t *testing.T) {
	p := newTestServer(t)

	request, err := http.NewRequest(http.MethodGet, p.server.URL+route, nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	raw := new(bytes.Buffer)
	_, err = raw.ReadFrom(response.Body)
	require.NoError(t, err)

	require.NotContains(t, strings.ToLower(raw.String()), "password")
	require.NotContains(t, strings.ToLower(raw.String()), "sandi")
}

// ── Menambah — form MASSAL ──────────────────────────────────────────────────────
//
// Form Tambah layar lama berbentuk daftar: satu cabang di atas, lalu tabel petugas cabang
// itu dengan Template Akses per baris, lalu satu tombol SIMPAN. Uji di bagian ini menjaga
// bentuk itu, karena ia yang paling mudah "disederhanakan" kembali menjadi satu baris oleh
// siapa pun yang membacanya tanpa melihat layar aslinya.

// bulk menyusun badan permintaan Tambah untuk satu cabang.
func bulk(branchID string, row ...string) string {
	return `{"cabang":"` + branchID + `","baris":[` + strings.Join(row, ",") + `]}`
}

// rowFor menyusun satu baris Template Akses.
func rowFor(login string) string {
	return `{"login":"` + login + `","modul":"PNCSearchKlaim","sub_modul":"Registrasi,",
	  "maks_cari":5,"maks_lihat":7,"lihat_ktp":true,"lihat_email":false,"lihat_notelp":false}`
}

// Daftar petugas mengisi tabel pada form Tambah.
func TestOperatorList(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/pengguna?cabang=100001", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(3), body["total"], "tiga petugas contoh pada cabang itu")

	first := body["pengguna"].([]any)[0].(map[string]any)
	require.NotEmpty(t, first["login"])
	require.NotEmpty(t, first["nama"])
}

// Daftar petugas tanpa cabang ditolak — di layar lama isian CABANG bertanda wajib.
func TestOperatorListWithoutBranch(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/pengguna", "ASM", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "cabang_belum_dipilih", body["kode"])
}

// Jalur "pengguna" tidak tertangkap sebagai sebuah ID.
func TestOperatorRouteNotMistakenForID(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodGet, route+"/pengguna?cabang=100001", "ASM", "")
	require.NotNil(t, body["pengguna"], "yang dikembalikan daftar petugas, bukan satu baris masking")
}

// Menyimpan beberapa baris sekaligus — bentuk pokok form Tambah.
func TestCreateMany(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		bulk("100001", rowFor("CONTOH.BARU.SATU"), rowFor("CONTOH.BARU.DUA")))

	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(2), body["tersimpan"])
	require.Equal(t, float64(0), body["ditolak"])
	require.Len(t, body["hasil"].([]any), 2)
}

// Pelaku diisi dari SESI, bukan dari badan permintaan.
func TestCreateManyStampsCallerFromSession(t *testing.T) {
	p := newTestServer(t)

	_, created := p.call(t, http.MethodPost, route, "ASM", bulk("100001", rowFor("CONTOH.BARU.SATU")))
	require.Equal(t, float64(1), created["tersimpan"])

	_, list := p.call(t, http.MethodGet, route+"?cari_di=login&kata_kunci=CONTOH.BARU.SATU", "ASM", "")
	saved := list["masking"].([]any)[0].(map[string]any)
	require.NotEmpty(t, saved["dicatat_oleh"])
	require.NotEmpty(t, saved["dicatat_pada"])
	require.Equal(t, true, saved["aktif"], "baris baru selalu aktif")
	require.Equal(t, "100001", saved["cabang"], "cabang datang dari atas form, bukan dari baris")
}

// Sebagian berhasil, sebagian ditolak — dan keduanya dilaporkan per baris.
//
// Ini perilaku layar lama: procedure dipanggil sekali per baris dan COMMIT sendiri,
// sehingga baris yang sah tetap tersimpan meski baris lain bentrok. Membungkusnya menjadi
// satu transaksi akan menggagalkan seluruhnya — perubahan yang terlihat pengguna.
func TestCreateManyPartialSuccess(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		// CONTOH.ADMIN sudah punya baris di cabang 100081.
		bulk("100081", rowFor("CONTOH.BARU.TIGA"), rowFor("CONTOH.ADMIN")))

	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(1), body["tersimpan"])
	require.Equal(t, float64(1), body["ditolak"])

	ditolak := map[string]string{}
	for _, item := range body["hasil"].([]any) {
		row := item.(map[string]any)
		if row["tersimpan"] == false {
			ditolak[row["login"].(string)], _ = row["pesan"].(string)
		}
	}
	require.Contains(t, ditolak, "CONTOH.ADMIN")
	require.Contains(t, ditolak["CONTOH.ADMIN"], "sudah punya data masking")
}

// Bila TIDAK ADA satu pun baris tersimpan, jawabannya 200 — bukan 201, dan bukan galat.
//
// Bukan 201 karena tidak ada yang dibuat. Bukan galat karena permintaannya berhasil
// dijalankan; yang ditolak adalah barisnya, dan alasannya ada di `hasil`.
//
// Status 2xx di sini WAJIB, dan bukan soal kerapian: klien memperlakukan setiap non-2xx
// sebagai galat dan mencari `pesan` di badannya. Badan ini membawa `hasil`, bukan `pesan`,
// sehingga status galat membuang laporan per baris tepat sebelum sampai ke layar dan
// menyisakan kalimat umum. Pernah terjadi — 409 dikembalikan di sini, dan petugas hanya
// melihat "Terjadi kesalahan pada sistem".
func TestCreateManyNoneSavedStillReportsPerRow(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM", bulk("100081", rowFor("CONTOH.ADMIN")))

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, float64(0), body["tersimpan"])
	require.Equal(t, float64(1), body["ditolak"])

	// Alasannya harus benar-benar sampai, bukan sekadar jumlahnya.
	row := body["hasil"].([]any)[0].(map[string]any)
	require.Equal(t, "CONTOH.ADMIN", row["login"])
	require.Equal(t, false, row["tersimpan"])
	require.Contains(t, row["pesan"], "sudah punya data masking")
}

// Permintaan tanpa satu baris pun ditolak, bukan dijawab "0 tersimpan".
func TestCreateManyWithoutRows(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM", `{"cabang":"100001","baris":[]}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "belum_ada_baris", body["kode"])
}

// Cabang yang tidak dikenal ditolak sebelum satu baris pun disentuh.
func TestCreateManyUnknownBranch(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		bulk("999999", rowFor("SIAPA.SAJA")))

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "cabang_tidak_dikenal", body["kode"])
}

// Isian cacat pada satu baris ditolak pada baris itu saja.
func TestCreateManyRowValidation(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPost, route, "ASM",
		bulk("100001",
			rowFor("CONTOH.BARU.SATU"),
			`{"login":"","modul":"","sub_modul":"","maks_cari":-1,"maks_lihat":0,
			  "lihat_ktp":false,"lihat_email":false,"lihat_notelp":false}`))

	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, float64(1), body["tersimpan"])
	require.Equal(t, float64(1), body["ditolak"])
}

// Field yang tidak dikenal DITOLAK, tidak diabaikan diam-diam.
//
// `aktif` termasuk di sini pada form Tambah: baris baru selalu aktif, dan menerimanya akan
// menyiratkan pilihan yang tidak ada.
func TestBulkUnknownFieldsRejected(t *testing.T) {
	for name, body := range map[string]string{
		"aktif di baris": `{"cabang":"100001","baris":[{"login":"X","modul":"M","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,
		  "aktif":true}]}`,
		"id di baris": `{"cabang":"100001","baris":[{"login":"X","modul":"M","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,
		  "id":"9"}]}`,
		"cabang di baris": `{"cabang":"100001","baris":[{"login":"X","modul":"M","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,
		  "cabang":"100081"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			p := newTestServer(t)
			response, content := p.call(t, http.MethodPost, route, "ASM", body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode)
			require.Equal(t, "permintaan_cacat", content["kode"])
		})
	}
}

// ── Mengubah dan status ─────────────────────────────────────────────────────────

// Mengubah baris yang tidak ada dijawab 404.
func TestUpdateMissing(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPut, route+"/9999", "ASM",
		`{"cabang":"100001","login":"SIAPA","modul":"PNCSearchKlaim","sub_modul":"",
		  "maks_cari":1,"maks_lihat":1,"lihat_ktp":false,"lihat_email":false,"lihat_notelp":false,
		  "aktif":true}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "masking_tidak_ditemukan", body["kode"])
}

// Menyimpan form Ubah IKUT mengubah status — form layar lama memang memuat isian STATUS.
//
// Yang menjaga status tidak berubah tanpa sengaja adalah layar daftar: tombol Edit hanya
// muncul pada baris aktif, meniru `ActionMaskingData_Sec`.
func TestUpdateCarriesStatusFromForm(t *testing.T) {
	p := newTestServer(t)

	_, before := p.call(t, http.MethodGet, route+"/1", "ASM", "")
	require.Equal(t, true, before["masking"].(map[string]any)["aktif"])

	response, body := p.call(t, http.MethodPut, route+"/1", "ASM",
		`{"cabang":"100081","login":"CONTOH.ADMIN","modul":"PNCSearchKlaim","sub_modul":"Registrasi,",
		  "maks_cari":9,"maks_lihat":9,"lihat_ktp":true,"lihat_email":true,"lihat_notelp":true,
		  "aktif":false}`)

	require.Equal(t, http.StatusOK, response.StatusCode)
	saved := body["masking"].(map[string]any)
	require.Equal(t, false, saved["aktif"], "status pada form ikut tersimpan")
	require.Equal(t, float64(9), saved["maks_cari"], "isian lain tetap tersimpan")
}

// Menonaktifkan tidak membuang baris; ia tetap terbaca dan dapat dihidupkan kembali.
func TestSetStatusIsSoftDelete(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodPut, route+"/1/status", "ASM", `{"aktif":false}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, false, body["masking"].(map[string]any)["aktif"])

	// Barisnya masih ada — "hapus" tidak pernah membuang apa pun (`D-66`).
	_, still := p.call(t, http.MethodGet, route+"/1", "ASM", "")
	require.Equal(t, false, still["masking"].(map[string]any)["aktif"])

	_, back := p.call(t, http.MethodPut, route+"/1/status", "ASM", `{"aktif":true}`)
	require.Equal(t, true, back["masking"].(map[string]any)["aktif"])
}

// DELETE tidak pernah didaftarkan.
//
// Layar lama punya tombol bernama DELETE, tetapi ia hanya mengubah STS_AKTF. Rute yang
// tidak ada tidak dapat dipanggil kode yang ditulis kemudian tanpa keputusan sadar.
func TestDeleteNotRouted(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, route+"/1", "ASM", "")
	require.NotEqual(t, http.StatusOK, response.StatusCode)
	require.NotEqual(t, http.StatusNoContent, response.StatusCode)
}

// ── Daftar cabang ───────────────────────────────────────────────────────────────

// Jalur "cabang" tidak tertangkap sebagai sebuah ID.
func TestBranchRouteNotMistakenForID(t *testing.T) {
	p := newTestServer(t)

	response, body := p.call(t, http.MethodGet, route+"/cabang", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NotNil(t, body["cabang"], "yang dikembalikan daftar cabang, bukan satu baris masking")
}

// Daftar cabang dapat disaring dengan kata kunci.
func TestBranchSearch(t *testing.T) {
	p := newTestServer(t)

	_, body := p.call(t, http.MethodGet, route+"/cabang?kata_kunci=manado", "ASM", "")
	require.Equal(t, float64(1), body["total"])
}
