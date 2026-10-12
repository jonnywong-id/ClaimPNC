package masterpasalhttp_test

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
	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasal/repo/memory"
	masterpasalusecase "claim-pnc/internal/masterpasal/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	masterpasalhttp "claim-pnc/internal/masterpasal/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji bukan handler-nya sendiri melainkan KONTRAKNYA —
// bahwa rutenya berada di balik sesi, bahwa permintaan tanpa portal yang sah ditolak, dan
// bahwa entitas yang berbeda benar-benar membaca penyimpanan yang berbeda. Menguji handler
// terpisah tidak membuktikan satu pun dari ketiganya.
type testServer struct {
	server *httptest.Server
	token  string

	asm *memory.Repo

	// asi tidak diberi isi apa pun; ia yang membuktikan pemisahan antarentitas.
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
		Clock:           clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	asm := memory.NewSampleRepo()
	// Master lini bisnisnya tetap diisi meski daftar pasalnya kosong: POOLDATA.BUSINESS
	// milik sistem lain, dan ia ada di setiap entitas terlepas dari ada-tidaknya pasal.
	asi := memory.NewRepo(nil, memory.SampleBusiness())

	service, err := masterpasalusecase.NewService(masterpasalusecase.Options{
		RepoSelector: func(alias string) (masterpasal.Store, error) {
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

	writeResponse := func(w http.ResponseWriter, r *http.Request, status int, body any) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	// Dirantai sama seperti cmd/claimpnc. Bila rantai ini tidak ditiru, uji penolakan
	// portal akan lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(authhttp.WriteError(logger), writeResponse)

	handler, err := masterpasalhttp.NewHandler(masterpasalhttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: writeResponse,
		WriteError:    masterpasalhttp.ErrorWriter(writeError),
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
				protected.Use(authhttp.Authenticate(authService, authhttp.ErrorWriter(writeError)))
				masterpasalhttp.Mount(protected, handler, portalDeps)
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

type route struct {
	method string
	path   string
	body   string
}

const sampleBody = `{"no_pasal":"PSL-900","isi_pasal":"isi","deskripsi":"ket","kategori":"1","bisnis":[]}`

// portalRoutes menyebut rute yang MENYENTUH basis data entitas.
//
// Dipakai uji sesi dan uji portal, supaya rute yang kelak ditambahkan tidak luput dari
// keduanya hanya karena lupa disalin ke salah satunya.
func portalRoutes() []route {
	return []route{
		{http.MethodGet, "/api/master/pasal-kerugian", ""},
		{http.MethodGet, "/api/master/pasal-kerugian/bisnis?cari=fire", ""},
		{http.MethodGet, "/api/master/pasal-kerugian/1", ""},
		{http.MethodPost, "/api/master/pasal-kerugian", sampleBody},
		{http.MethodPut, "/api/master/pasal-kerugian/1", sampleBody},
		{http.MethodDelete, "/api/master/pasal-kerugian/1", ""},
	}
}

// Seluruh rute berada di balik sesi — TERMASUK daftar kategori yang isinya milik aplikasi.
//
// Kategori memang tidak memuat data entitas, tetapi ia tetap bagian dari layar yang hanya
// boleh dibuka pengguna yang sudah masuk. Membiarkannya terbuka berarti satu rute modul
// ini punya aturan akses yang berbeda dari rute lainnya, tanpa alasan.
func TestRoutesRequireSession(t *testing.T) {
	p := newTestServer(t)
	noToken := *p
	noToken.token = ""

	all := append(portalRoutes(), route{http.MethodGet, "/api/master/pasal-kerugian/kategori", ""})

	for _, r := range all {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			response, _ := noToken.call(t, r.method, r.path, "ASM", r.body)
			require.Equal(t, http.StatusUnauthorized, response.StatusCode)
		})
	}
}

// Rute yang menyentuh basis data entitas menuntut portal — tanpa satu pun kecuali.
func TestPortalRoutesRequirePortal(t *testing.T) {
	p := newTestServer(t)

	for _, r := range portalRoutes() {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			response, content := p.call(t, r.method, r.path, "", r.body)
			require.Equal(t, http.StatusBadRequest, response.StatusCode,
				"permintaan tanpa portal WAJIB ditolak, tidak pernah jatuh ke portal utama")
			require.Equal(t, "portal_tidak_disebut", content["kode"])
		})
	}
}

// Daftar Kategori TIDAK menuntut portal, dan itu satu-satunya rute modul ini yang begitu.
//
// Isinya diturunkan dari ekspresi di `Activity/CNMInsertPasalDataMaster-Act.xml`, bukan
// dibaca dari basis data mana pun. Menuntut portal di sini akan membuat form gagal dimuat
// justru saat pengguna belum memilih entitas.
func TestCategoryRouteNeedsNoPortal(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/pasal-kerugian/kategori", "", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	list, ok := content["kategori"].([]any)
	require.True(t, ok)
	require.Len(t, list, 3)

	first, _ := list[0].(map[string]any)
	require.Equal(t, "1", first["kode"])
	require.Equal(t, "Jaminan Polis", first["nama"])

	third, _ := list[2].(map[string]any)
	require.Equal(t, "", third["kode"], "Notifikasi adalah cabang else; kodenya memang kosong")
	require.Equal(t, "Notifikasi", third["nama"])
}

// Portal yang tidak dikenal ditolak, TIDAK dialihkan ke portal utama sebagai cadangan.
func TestUnknownPortalRejected(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodGet, "/api/master/pasal-kerugian", "TIDAKADA", "")
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "portal_tidak_dikenal", content["kode"])
}

// Daftar dibaca dari entitas yang diminta, dan entitas lain TIDAK ikut terbaca.
//
// Ini uji R-20 pada modul ini: satu identitas menjangkau empat basis data, dan yang
// memisahkannya adalah portal pada setiap permintaan.
func TestListIsolatedPerEntity(t *testing.T) {
	p := newTestServer(t)

	t.Run("ASM berisi contoh", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, "/api/master/pasal-kerugian", "ASM", "")
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "ASM", content["portal"])

		list, ok := content["pasal_kerugian"].([]any)
		require.True(t, ok)
		require.Len(t, list, 3)
	})

	t.Run("ASI kosong", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet, "/api/master/pasal-kerugian", "ASI", "")
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Equal(t, "ASI", content["portal"])
		require.Empty(t, content["pasal_kerugian"])
	})
}

// Daftar TIDAK memuat lini bisnis; satu baris memuatnya.
//
// Perbedaan itu disengaja, dan uji ini yang menjaganya: grid layar lama pun tidak
// menampilkan lini bisnis, dan memuatnya untuk setiap baris berarti satu pembacaan master
// per pasal demi kolom yang tidak ada.
func TestBusinessOnlyLoadedOnSingleRead(t *testing.T) {
	p := newTestServer(t)

	_, list := p.call(t, http.MethodGet, "/api/master/pasal-kerugian", "ASM", "")
	rows, _ := list["pasal_kerugian"].([]any)
	require.NotEmpty(t, rows)
	for _, row := range rows {
		clause, _ := row.(map[string]any)
		require.Empty(t, clause["bisnis"], "daftar tidak memuat lini bisnis")
	}

	response, single := p.call(t, http.MethodGet, "/api/master/pasal-kerugian/2", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	clause, _ := single["pasal_kerugian"].(map[string]any)
	require.Equal(t, "PSL-002", clause["no_pasal"])
	require.Equal(t, "Pengecualian", clause["kategori_label"])

	business, ok := clause["bisnis"].([]any)
	require.True(t, ok)
	require.Len(t, business, 2)
}

// Penambahan menerbitkan ID dan MENURUNKAN sebutan kategori; keduanya dari server.
func TestCreate(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"PSL-010","isi_pasal":"Isi pasal baru","deskripsi":"Keterangan",` +
		`"kategori":"2","bisnis":[{"id":"2004","nama":"Fire / Property"}]}`

	response, content := p.call(t, http.MethodPost, "/api/master/pasal-kerugian", "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	clause, _ := content["pasal_kerugian"].(map[string]any)
	require.Equal(t, "4", clause["id"], "nomor melanjutkan yang terbesar di contoh")
	require.Equal(t, "PSL-010", clause["no_pasal"])
	require.Equal(t, "2", clause["kategori"])
	require.Equal(t, "Pengecualian", clause["kategori_label"])

	// Baris baru benar-benar masuk ke entitas yang diminta, bukan ke entitas lain.
	saved, err := p.asm.Get(t.Context(), "4")
	require.NoError(t, err)
	require.Equal(t, "PSL-010", saved.Number)

	_, err = p.asi.Get(t.Context(), "4")
	require.ErrorIs(t, err, masterpasal.ErrNotFound)
}

// No Pasal kosong ditolak 422, dengan keterangan menempel di isiannya.
//
// Ia satu-satunya pemeriksaan isian modul ini, dan pesannya mengikuti layar lama apa
// adanya (`D-13`).
func TestCreateRejectsEmptyNumber(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"   ","isi_pasal":"isi","deskripsi":"","kategori":"1","bisnis":[]}`

	response, content := p.call(t, http.MethodPost, "/api/master/pasal-kerugian", "ASM", body)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])

	detail, ok := content["detail"].([]any)
	require.True(t, ok)
	require.Len(t, detail, 1)

	violation, _ := detail[0].(map[string]any)
	require.Equal(t, "no_pasal", violation["field"])
	require.Contains(t, violation["pesan"], "No Pasal")
}

// Isian selain No Pasal boleh kosong seluruhnya — keputusan Work Owner "jalankan as is".
func TestCreateAcceptsEverythingElseEmpty(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"PSL-011","isi_pasal":"","deskripsi":"","kategori":"","bisnis":[]}`

	response, content := p.call(t, http.MethodPost, "/api/master/pasal-kerugian", "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode)

	clause, _ := content["pasal_kerugian"].(map[string]any)
	require.Equal(t, "", clause["kategori"])
	require.Equal(t, "Notifikasi", clause["kategori_label"])
}

// No Pasal kembar diterima. Kuncinya IDDATA, dan Pega pun tidak memeriksanya.
func TestCreateAllowsDuplicateNumber(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"PSL-001","isi_pasal":"kembar","deskripsi":"","kategori":"1","bisnis":[]}`

	response, _ := p.call(t, http.MethodPost, "/api/master/pasal-kerugian", "ASM", body)
	require.Equal(t, http.StatusCreated, response.StatusCode,
		"nomor pasal kembar TIDAK ditolak; keputusan Work Owner 2026-09-19")
}

// Penyuntingan menyimpan isian baru tanpa mengubah kunci barisnya.
func TestUpdate(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"PSL-001-R","isi_pasal":"Isi yang diperbarui","deskripsi":"Baru",` +
		`"kategori":"1","bisnis":[{"id":"2004","nama":"Fire / Property"}]}`

	response, content := p.call(t, http.MethodPut, "/api/master/pasal-kerugian/1", "ASM", body)
	require.Equal(t, http.StatusOK, response.StatusCode)

	clause, _ := content["pasal_kerugian"].(map[string]any)
	require.Equal(t, "1", clause["id"], "kunci baris tidak pernah ikut berubah")
	require.Equal(t, "PSL-001-R", clause["no_pasal"])
	require.Equal(t, "Isi yang diperbarui", clause["isi_pasal"])
}

// Menyunting baris yang sudah tidak ada dijawab 404, BUKAN diam-diam menyisipkan baru.
//
// Procedure lama justru menyisipkan pada cabang itu (`PEGA_D_PASAL_MASTER.prc:8`), dan
// perilaku itu sengaja tidak dibawa: `PUT` atas baris yang sudah dihapus petugas lain akan
// menerbitkan baris kedua tanpa satu pun tanda.
func TestUpdateUnknownIsNotAnInsert(t *testing.T) {
	p := newTestServer(t)

	before, _ := p.asm.List(t.Context())

	response, content := p.call(t, http.MethodPut, "/api/master/pasal-kerugian/999", "ASM", sampleBody)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "tidak_ditemukan", content["kode"])

	after, _ := p.asm.List(t.Context())
	require.Len(t, after, len(before), "tidak ada baris baru yang diterbitkan")
}

// Penghapusan PERMANEN, dan penghapusan kedua dijawab 404.
//
// Ia menyupersede `D-66` untuk tabel ini atas keputusan Work Owner 2026-09-19. Uji ini
// membuktikan akibatnya sebagaimana adanya: barisnya benar-benar hilang, bukan ditandai.
func TestDeleteIsPermanent(t *testing.T) {
	p := newTestServer(t)

	response, content := p.call(t, http.MethodDelete, "/api/master/pasal-kerugian/3", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "3", content["id"])
	require.Equal(t, "ASM", content["portal"])

	_, err := p.asm.Get(t.Context(), "3")
	require.ErrorIs(t, err, masterpasal.ErrNotFound,
		"barisnya HILANG, bukan ditandai terhapus")

	again, _ := p.call(t, http.MethodDelete, "/api/master/pasal-kerugian/3", "ASM", "")
	require.Equal(t, http.StatusNotFound, again.StatusCode)
}

// Menghapus di satu entitas tidak menyentuh entitas lain.
func TestDeleteIsolatedPerEntity(t *testing.T) {
	p := newTestServer(t)

	response, _ := p.call(t, http.MethodDelete, "/api/master/pasal-kerugian/1", "ASI", "")
	require.Equal(t, http.StatusNotFound, response.StatusCode,
		"ASI tidak punya baris itu, dan permintaannya TIDAK boleh jatuh ke ASM")

	_, err := p.asm.Get(t.Context(), "1")
	require.NoError(t, err, "baris ASM tetap utuh")
}

// Pencarian lini bisnis menolak kata kunci terlalu pendek dengan daftar KOSONG, bukan galat.
//
// Pengguna yang baru mengetik satu huruf belum melakukan kesalahan apa pun.
func TestBusinessLookup(t *testing.T) {
	p := newTestServer(t)

	t.Run("kata kunci pendek menghasilkan daftar kosong", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet,
			"/api/master/pasal-kerugian/bisnis?cari=f", "ASM", "")
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Empty(t, content["bisnis"])
	})

	t.Run("kata kunci cukup menemukan baris", func(t *testing.T) {
		response, content := p.call(t, http.MethodGet,
			"/api/master/pasal-kerugian/bisnis?cari=fire", "ASM", "")
		require.Equal(t, http.StatusOK, response.StatusCode)

		list, ok := content["bisnis"].([]any)
		require.True(t, ok)
		require.Len(t, list, 1)

		first, _ := list[0].(map[string]any)
		require.Equal(t, "2004", first["id"])
		require.Equal(t, "Fire / Property", first["nama"])
	})

	t.Run("pencarian tidak peka besar-kecil huruf", func(t *testing.T) {
		_, content := p.call(t, http.MethodGet,
			"/api/master/pasal-kerugian/bisnis?cari=TRAVEL", "ASM", "")
		list, _ := content["bisnis"].([]any)
		require.Len(t, list, 1)
	})
}

// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam.
//
// Salah ketik nama field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai
// kosong tanpa satu pun tanda bahwa ada yang salah.
func TestUnknownFieldRejected(t *testing.T) {
	p := newTestServer(t)

	body := `{"no_pasal":"PSL-012","isi_pasal":"isi","deskripsi":"","kategori":"1",` +
		`"bisnis":[],"kategori_label":"Jaminan Polis"}`

	response, content := p.call(t, http.MethodPost, "/api/master/pasal-kerugian", "ASM", body)
	require.Equal(t, http.StatusBadRequest, response.StatusCode,
		"kategori_label adalah turunan server; mengirimkannya ditolak, bukan diabaikan")
	require.Equal(t, "permintaan_cacat", content["kode"])
}

// Handler menolak dirakit tanpa bahan yang wajib.
//
// Penolakannya terjadi saat perakitan, bukan saat permintaan pertama datang: rakitan yang
// setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := masterpasalhttp.NewHandler(masterpasalhttp.Options{})
	require.Error(t, err)
}
