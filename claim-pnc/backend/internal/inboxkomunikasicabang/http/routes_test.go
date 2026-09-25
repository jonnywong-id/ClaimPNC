package inboxkomunikasicabanghttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
	authusecase "claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/inboxkomunikasicabang"
	"claim-pnc/internal/inboxkomunikasicabang/repo/memory"
	inboxkomunikasicabangusecase "claim-pnc/internal/inboxkomunikasicabang/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxkomunikasicabanghttp "claim-pnc/internal/inboxkomunikasicabang/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji rute modul Inbox Komunikasi Cabang.
//
// # KENAPA BERKAS INI ADA — dan kenapa ketiadaannya sempat mahal
//
// Modul ini sempat punya 40 uji yang seluruhnya lulus sementara DUA TOMBOL yang ada di layar
// lama tidak terpasang sama sekali. Tidak satu pun uji menangkapnya, dan sebabnya jelas:
// seluruhnya menguji apa yang DIBANGUN, tidak satu pun menguji apa yang SAMPAI KE LAYAR.
//
// Bentuk grid layar ini — termasuk kedua kolom tombolnya — ditetapkan PELADEN dan dikirim
// lewat `/tab`. Layar menggambar tombol hanya bila kolomnya ada di jawaban itu. Jadi uji
// domain yang membuktikan kolomnya terdaftar TIDAK membuktikan tombolnya muncul; yang
// membuktikannya adalah jawaban HTTP-nya sendiri.
//
// Itu pula yang membuat binary lama berakibat fatal di layar ini: bundel SPA yang baru tetap
// tidak menggambar tombol apa pun bila peladen yang melayaninya belum mengenal kolomnya.
const (
	branchLogin = "pictekniks"
	otherLogin  = "adminpnc"
)

type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T, login string) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	fixed := clock.FixedAt(time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC))

	authService, err := authusecase.NewService(authusecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           fixed,
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	store := memory.NewSampleStore()

	service, err := inboxkomunikasicabangusecase.NewService(
		inboxkomunikasicabangusecase.Options{
			RepoSelector: func(alias string) (inboxkomunikasicabang.Repo, error) {
				if alias != "ASM" {
					return nil, portal.ErrNotReady
				}
				return store, nil
			},
			BranchResolver: memory.NewSampleBranchResolver(),

			// Jam TETAP, sama dengan jam modul auth di atas. Tanggal balasan adalah dasar
			// pengurutan tab "Sudah Dijawab", sehingga jam berjalan akan membuat uji urutan
			// bergantung pada kapan ia dijalankan.
			Clock: fixed,
		})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(
		&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	// Jembatan pemanggil ditiru SAMA PERSIS dengan cmd/claimpnc. Bila ia tidak ditiru, uji
	// akan lulus di sini dan gagal di aplikasi sungguhan — dan di layar ini akibatnya bukan
	// galat melainkan batas cabang yang tidak dapat diturunkan.
	handler := inboxkomunikasicabanghttp.NewHandler(inboxkomunikasicabanghttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (inboxkomunikasicabanghttp.Caller, bool) {
			baseCtx, existing := authhttp.CallerFromContext(ctx)
			if !existing {
				return inboxkomunikasicabanghttp.Caller{}, false
			}
			return inboxkomunikasicabanghttp.Caller{
				Login: baseCtx.User.Login,
				Name:  baseCtx.User.Name,
			}, true
		},
		Logger:              logger,
		WriteJSON:           writeResponse,
		FallbackErrorWriter: inboxkomunikasicabanghttp.ErrorWriter(writeError),
	})

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
				inboxkomunikasicabanghttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server}
	p.token = p.login(t, login)
	return p
}

func (p *testServer) login(t *testing.T, login string) string {
	t.Helper()

	body := strings.NewReader(
		`{"nama_pengguna":"` + login + `","kata_sandi":"rahasia123"}`)
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
	t *testing.T, method, path, portalAlias string,
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

// post menjalankan satu permintaan tulis dengan badan JSON.
//
// Badan kosong dikirim sebagai TANPA badan sama sekali, bukan sebagai `{}`, supaya rute yang
// memang tidak menerima badan permintaan diuji sebagaimana layar memanggilnya.
func (p *testServer) post(
	t *testing.T, path, portalAlias, body string,
) (*http.Response, map[string]any) {
	t.Helper()

	var payload io.Reader
	if body != "" {
		payload = strings.NewReader(body)
	}

	request, err := http.NewRequest(http.MethodPost, p.server.URL+path, payload)
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

// idsIn membaca nomor percakapan dari jawaban daftar.
func idsIn(body map[string]any) []string {
	rows, _ := body["baris"].([]any)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		item, _ := row.(map[string]any)
		id, _ := item["komunikasi"].(string)
		ids = append(ids, id)
	}
	return ids
}

// columnKeys membaca kunci kolom sebuah tab dari jawaban JSON.
func columnKeys(tab map[string]any) []string {
	raw, _ := tab["kolom"].([]any)
	keys := make([]string, 0, len(raw))
	for _, item := range raw {
		column, _ := item.(map[string]any)
		key, _ := column["kunci"].(string)
		keys = append(keys, key)
	}
	return keys
}

func TestMetadataSendsBothActionColumnsForEveryTab(t *testing.T) {
	// INILAH uji yang seharusnya ada sejak awal.
	//
	// Layar menggambar tombol hanya bila kolomnya ada di jawaban ini. Uji domain yang
	// membuktikan kolomnya terdaftar di `tab.go` TIDAK membuktikan ia sampai ke layar —
	// dan justru di celah itulah kedua tombol sempat hilang tanpa satu pun uji gagal.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang/tab", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	tabs, _ := body["tab"].([]any)
	require.Len(t, tabs, 2)

	for _, item := range tabs {
		tab, _ := item.(map[string]any)
		keys := columnKeys(tab)

		require.Containsf(t, keys, inboxkomunikasicabang.FieldActionDetail,
			"tab %v tidak mengirim kolom tombol Detail Komunikasi", tab["nama"])
		require.Containsf(t, keys, inboxkomunikasicabang.FieldActionFinish,
			"tab %v tidak mengirim kolom tombol Selesai Komunikasi", tab["nama"])

		// Urutannya pun diuji: keduanya paling belakang, Detail lebih dulu — persis urutan
		// di section. Kolom tombol yang berpindah ke tengah akan menggeser seluruh kolom
		// isian tanpa satu pun galat.
		require.Equal(t,
			[]string{
				inboxkomunikasicabang.FieldActionDetail,
				inboxkomunikasicabang.FieldActionFinish,
			},
			keys[len(keys)-2:])
	}
}

func TestMetadataKeepsTheLiteralButtonHeading(t *testing.T) {
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang/tab", "ASM")
	tabs, _ := body["tab"].([]any)

	for _, item := range tabs {
		tab, _ := item.(map[string]any)
		raw, _ := tab["kolom"].([]any)

		for _, entry := range raw {
			column, _ := entry.(map[string]any)
			key, _ := column["kunci"].(string)
			if !inboxkomunikasicabang.IsAction(key) {
				continue
			}
			require.Equal(t, "Button", column["judul"])
		}
	}
}

func TestListAnswersWithTheTabShapeSoTheScreenDrawsTheSameColumns(t *testing.T) {
	// Layar menggambar kolomnya dari `tab` pada jawaban DAFTAR, bukan dari metadata yang
	// diambil sekali. Bila keduanya berbeda, tombolnya muncul lalu hilang saat berpindah
	// halaman — dan itu tidak menghasilkan satu pun galat.
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	tab, _ := body["tab"].(map[string]any)
	keys := columnKeys(tab)

	require.Contains(t, keys, inboxkomunikasicabang.FieldActionDetail)
	require.Contains(t, keys, inboxkomunikasicabang.FieldActionFinish)
}

func TestListCarriesTheConversationNumberEachButtonNeeds(t *testing.T) {
	// Kedua tombol mengirim nomor percakapan — `KOMID = .ClaimNo` di Pega. Baris yang tidak
	// membawanya menghasilkan tombol yang tampak normal lalu menembak alamat kosong.
	server := newTestServer(t, branchLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	rows, _ := body["baris"].([]any)
	require.NotEmpty(t, rows, "penyimpanan contoh harus memberi baris bagi petugas cabang 1001")

	for _, item := range rows {
		row, _ := item.(map[string]any)
		require.NotEmpty(t, row["komunikasi"], "setiap baris wajib membawa nomor percakapan")
	}
}

// CATATAN. Uji `TestFinishActionIsAnsweredWithAReasonNotSilence` DIHAPUS pada 2026-09-24.
//
// Ia membuktikan tombol "Selesai Komunikasi" dijawab 501, dan pernyataan itu TIDAK LAGI
// BENAR — tombolnya kini benar-benar menutup percakapan lewat rute tersendiri. Uji yang
// menjaga perilaku lama pada modul yang sudah berubah lebih buruk daripada tidak ada uji: ia
// lulus, dan kelulusannya menyatakan hal yang keliru.
//
// Penggantinya ada dua: TestFinishingAConversationRemovesItFromBothTabsOverHTTP untuk yang
// sekarang berjalan, dan TestCreatingANewConversationStillAnswersWithAReason untuk satu-
// satunya tindakan yang masih ditolak.

func TestDetailIsReachableWithTheNumberTheButtonSends(t *testing.T) {
	// Tombol "Detail Komunikasi" mengirim nomor percakapan apa adanya. Rute ini yang
	// menerimanya — dan bila jalurnya tidak cocok, tombolnya akan tampak tidak bekerja.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0001", "ASM")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "KOM-0001", body["komunikasi"])

	messages, _ := body["pesan"].([]any)
	require.NotEmpty(t, messages)
}

func TestDetailOfAnotherBranchIsRefused(t *testing.T) {
	// KOM-0010 milik cabang 1003. Petugas cabang 1001 tidak boleh membukanya lewat nomornya,
	// meski tombolnya tidak pernah muncul untuk baris itu — tombol bukan penjagaan.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t, http.MethodGet,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0010", "ASM")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "komunikasi_tidak_ditemukan", body["kode"])
}

func TestEveryRouteRefusesARequestWithoutAPortal(t *testing.T) {
	// Jatuh ke portal utama berarti menampilkan percakapan satu badan hukum kepada petugas
	// badan hukum lain tanpa satu pun pesan galat (`R-20`).
	server := newTestServer(t, branchLogin)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/inbox-komunikasi-cabang/tab"},
		{http.MethodGet, "/api/inbox-komunikasi-cabang"},
		{http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0001"},
		{http.MethodGet, "/api/inbox-komunikasi-cabang/ekspor"},

		// Kedua rute TULIS ikut, dan merekalah yang paling menuntutnya: yang dipertaruhkan
		// bukan lagi percakapan badan hukum lain yang TERLIHAT, melainkan yang BERUBAH.
		{http.MethodPost, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/balas"},
		{http.MethodPost, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/selesai"},
		{http.MethodPost, "/api/inbox-komunikasi-cabang/tindakan"},
	} {
		response, _ := server.call(t, route.method, route.path, "")
		require.NotEqualf(t, http.StatusOK, response.StatusCode,
			"%s dilayani tanpa header portal", route.path)
	}
}

func TestHeadOfficeStaffSeesHeadOfficeConversations(t *testing.T) {
	// `adminpnc` dipetakan ke cabang kantor pusat di penyimpanan contoh, sehingga jalur
	// kantor pusat punya saksi yang cabangnya BENAR-BENAR terbaca.
	server := newTestServer(t, otherLogin)

	_, body := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang", "ASM")

	branch, _ := body["batas_cabang"].(map[string]any)
	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, branch["kode"])
	require.Equal(t, true, branch["kantor_pusat"])
	require.Equal(t, true, branch["terbaca"])
}

func TestExportGoesThroughTheSameBranchBoundaryAsTheList(t *testing.T) {
	// Batas yang berlaku pada daftar tetapi tidak pada unduhan bukan batas sama sekali — ia
	// hanya menyulitkan orang yang patuh.
	server := newTestServer(t, branchLogin)

	request, err := http.NewRequest(http.MethodGet,
		server.server.URL+"/api/inbox-komunikasi-cabang/ekspor?tab=1", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+server.token)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Disposition"), "cabang-1001")
}

// ── Aksi tulis ────────────────────────────────────────────────────────────────

func TestReplyingMovesTheConversationBetweenTabsAsSeenOverHTTP(t *testing.T) {
	// Uji ujung-ke-ujung terpendek yang membuktikan tombol "Balas" benar-benar bekerja:
	// barisnya ada di tab "Belum Dijawab", dibalas, lalu muncul di tab "Sudah Dijawab".
	//
	// Uji penyimpanan sudah membuktikan aturannya. Yang INI buktikan adalah bahwa aturan itu
	// dapat dicapai lewat alamat yang benar-benar dipanggil layar — celah yang sama yang
	// sempat menyembunyikan kedua tombol.
	server := newTestServer(t, branchLogin)

	_, before := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang?tab=1", "ASM")
	require.Contains(t, idsIn(before), "KOM-0005")

	response, body := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/balas", "ASM",
		`{"pesan":"Sudah kami tindak lanjuti."}`)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "KOM-0005", body["komunikasi"])
	require.Equal(t, "ASM", body["portal"])
	require.NotEmpty(t, body["pesan"])

	_, notAnswered := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang?tab=1", "ASM")
	require.NotContains(t, idsIn(notAnswered), "KOM-0005")

	_, answered := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang?tab=2", "ASM")
	require.Contains(t, idsIn(answered), "KOM-0005")
}

func TestAnEmptyReplyIsRefusedWithAFieldLevelMessage(t *testing.T) {
	// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar. Layar menandai
	// isiannya — dan untuk itu ia butuh NAMA isiannya, bukan sekadar kalimat.
	server := newTestServer(t, branchLogin)

	response, body := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/balas", "ASM", `{"pesan":"   "}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])

	details, _ := body["detail"].([]any)
	require.Len(t, details, 1)
	first, _ := details[0].(map[string]any)
	require.Equal(t, "pesan", first["isian"])
}

func TestAReplyWithAnUnknownFieldIsRefusedRatherThanSilentlyIgnored(t *testing.T) {
	// Layar yang salah menamai isiannya akan mengirim balasan KOSONG tanpa satu pun tanda,
	// dan balasan kosong yang tersimpan memindahkan percakapan ke tab "Sudah Dijawab" —
	// terbaca sudah dijawab padahal tidak ada jawabannya.
	server := newTestServer(t, branchLogin)

	response, _ := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/balas", "ASM",
		`{"message":"salah nama isian"}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	_, after := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang?tab=1", "ASM")
	require.Contains(t, idsIn(after), "KOM-0005",
		"percakapan tidak boleh berpindah tab karena permintaan yang ditolak")
}

func TestReplyingToAnotherBranchConversationIsRefusedOverHTTP(t *testing.T) {
	// Batas cabang WAJIB berlaku pada rute tulis, bukan hanya pada daftar. Balasan yang
	// telanjur tersimpan di percakapan cabang lain tidak dapat ditarik kembali (`R-20`).
	//
	// KOM-0010 milik cabang 1003→1004; petugas contoh berada di cabang 1001.
	server := newTestServer(t, branchLogin)

	response, body := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0010/balas", "ASM",
		`{"pesan":"seharusnya tidak tersimpan"}`)

	require.Equal(t, http.StatusNotFound, response.StatusCode)
	require.Equal(t, "komunikasi_tidak_ditemukan", body["kode"])
}

func TestFinishingAConversationRemovesItFromBothTabsOverHTTP(t *testing.T) {
	server := newTestServer(t, branchLogin)

	response, body := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/selesai", "ASM", "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "KOM-0005", body["komunikasi"])

	for _, tab := range []string{"1", "2"} {
		_, list := server.call(t,
			http.MethodGet, "/api/inbox-komunikasi-cabang?tab="+tab, "ASM")
		require.NotContainsf(t, idsIn(list), "KOM-0005",
			"percakapan yang ditutup masih tampil di tab %s", tab)
	}
}

func TestFinishingTwiceIsRefusedOverHTTP(t *testing.T) {
	// Dua orang dapat menekan tombol yang sama pada layar yang sama-sama usang. Yang kedua
	// harus tahu bahwa BUKAN dia yang menutupnya — jawabannya 404 dengan pesan yang menyuruh
	// menyegarkan, bukan 200 yang menyatakan sesuatu yang tidak terjadi.
	server := newTestServer(t, branchLogin)

	first, _ := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/selesai", "ASM", "")
	require.Equal(t, http.StatusOK, first.StatusCode)

	second, body := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/selesai", "ASM", "")
	require.Equal(t, http.StatusNotFound, second.StatusCode)
	require.Contains(t, body["pesan"], "segarkan")
}

// CATATAN. TestCreatingANewConversationStillAnswersWithAReason DIHAPUS pada 2026-09-24.
//
// Ia membuktikan "Kirim Pesan" dijawab 501, dan pernyataan itu TIDAK LAGI BENAR: formnya
// ternyata tidak hilang dari export melainkan tersembunyi sebagai blok bersyarat di dalam
// section daftar, dan kini ia benar-benar membuat percakapan.
//
// Rute `/tindakan` yang dijawabnya ikut dicabut, sehingga uji ini bukan hanya usang — ia
// menembak alamat yang sudah tidak ada. Penggantinya ada di bawah, pada bagian "Kirim Pesan".

func TestTheDetailScreenAnnouncesThatReplyingIsAvailable(t *testing.T) {
	// Layar menggambar kotak balasannya AKTIF atau tidak berdasarkan penanda ini, bukan
	// berdasarkan tulisan tetap di kodenya. Penandanya berubah nilai pada 2026-09-24, dan
	// justru itulah alasan ia berupa data.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0005", "ASM")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, true, body["balas_tersedia"])
}

// ── Kirim Pesan ───────────────────────────────────────────────────────────────

func TestTheBranchPickerIsServedWithBothDestinations(t *testing.T) {
	// Layar menggambar dropdown tujuan dari jawaban ini, bukan dari daftar yang ditulis tetap
	// di kodenya — alasannya sama dengan kolom grid: keduanya hasil pembacaan export, dan
	// tempat pembacaan itu tercatat adalah peladen.
	server := newTestServer(t, branchLogin)

	response, body := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/cabang", "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	destinations, _ := body["tujuan"].([]any)
	require.Equal(t, []any{"PUSAT", "CABANG"}, destinations)

	branches, _ := body["cabang"].([]any)
	require.NotEmpty(t, branches)

	first, _ := branches[0].(map[string]any)
	require.NotEmpty(t, first["kode"])
	require.NotEmpty(t, first["nama"])
}

func TestTheBranchPickerNeverLeaksBranchEmailAddresses(t *testing.T) {
	// Alamat surel cabang dibaca peladen untuk keperluan notifikasi dan TIDAK pernah
	// meninggalkan peladen: yang dikirim ke peramban ikut tercatat di cache, log proxy, dan
	// alat pengembang.
	server := newTestServer(t, branchLogin)

	response, err := http.NewRequest(http.MethodGet,
		server.server.URL+"/api/inbox-komunikasi-cabang/cabang", nil)
	require.NoError(t, err)
	response.Header.Set("Authorization", "Bearer "+server.token)
	response.Header.Set(portalhttp.HeaderPortal, "ASM")

	raw, err := http.DefaultClient.Do(response)
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Body.Close() })

	payload, err := io.ReadAll(raw.Body)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "@",
		"jawaban tidak boleh memuat satu pun alamat surel")
}

func TestSendingAMessageCreatesAConversationVisibleInTheNotAnsweredTab(t *testing.T) {
	// Uji ujung-ke-ujung terpendek yang membuktikan "Kirim Pesan" bekerja. Login contoh
	// berada di cabang 1001, sehingga pesannya ke PUSAT harus tampil di daftarnya sendiri —
	// penyaringnya `OR`, bukan `AND`.
	server := newTestServer(t, branchLogin)

	response, body := server.post(t, "/api/inbox-komunikasi-cabang/pesan", "ASM",
		`{"tujuan":"PUSAT","cabang":"","pesan":"Mohon konfirmasi kelengkapan dokumen."}`)

	// 201, bukan 200: sebuah sumber daya BARU terbit, dan nomornya belum ada sebelum
	// permintaan ini.
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.NotEmpty(t, body["komunikasi"])
	require.Equal(t, "ASM", body["portal"])

	_, list := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang?tab=1", "ASM")
	require.Contains(t, idsIn(list), body["komunikasi"])
}

func TestAMessageToTheBranchWithoutChoosingOneIsRefused(t *testing.T) {
	// Kalimatnya dibawa APA ADANYA dari `Local.msgErr` pada activity lama — pengguna layar
	// ini sudah mengenalnya (`D-13`).
	server := newTestServer(t, branchLogin)

	response, body := server.post(t, "/api/inbox-komunikasi-cabang/pesan", "ASM",
		`{"tujuan":"CABANG","cabang":"","pesan":"halo"}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", body["kode"])

	details, _ := body["detail"].([]any)
	require.Len(t, details, 1)
	first, _ := details[0].(map[string]any)
	require.Equal(t, "cabang", first["isian"])
	require.Equal(t, "Silakan pilih cabang terlebih dahulu", first["pesan"])
}

func TestAMessageWithAnUnknownFieldIsRefusedRatherThanSilentlyIgnored(t *testing.T) {
	// Isian `cabang` yang salah nama akan mengirim pesan ke kantor pusat padahal penggunanya
	// memilih sebuah cabang — dan tidak ada satu pun tanda bahwa itu terjadi.
	server := newTestServer(t, branchLogin)

	response, _ := server.post(t, "/api/inbox-komunikasi-cabang/pesan", "ASM",
		`{"tujuan":"CABANG","branch":"1002","pesan":"halo"}`)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
}

func TestANewMessageCanBeRepliedToThroughTheSameAPI(t *testing.T) {
	// Uji rantai lewat HTTP: pesan baru harus benar-benar menjadi percakapan yang utuh,
	// bukan baris yang bentuknya berbeda dari baris warisan.
	server := newTestServer(t, branchLogin)

	_, created := server.post(t, "/api/inbox-komunikasi-cabang/pesan", "ASM",
		`{"tujuan":"PUSAT","cabang":"","pesan":"Mohon konfirmasi."}`)
	id, _ := created["komunikasi"].(string)
	require.NotEmpty(t, id)

	response, _ := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/"+id+"/balas", "ASM",
		`{"pesan":"Sudah kami tindak lanjuti."}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	_, answered := server.call(t, http.MethodGet, "/api/inbox-komunikasi-cabang?tab=2", "ASM")
	require.Contains(t, idsIn(answered), id)
}

func TestTheRemovedActionRouteIsGone(t *testing.T) {
	// Rute `/tindakan` dicabut bersama keempat penolakannya. Uji ini mengunci pencabutannya:
	// rute yang hidup kembali diam-diam akan menjawab 501 untuk tindakan yang sebenarnya
	// sudah bekerja.
	server := newTestServer(t, branchLogin)

	response, _ := server.post(t,
		"/api/inbox-komunikasi-cabang/tindakan?tindakan=tambah", "ASM", "")

	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

// ── Utas layar detail ─────────────────────────────────────────────────────────

func TestTheThreadGrowsWithEveryUtteranceOverHTTP(t *testing.T) {
	// Koreksi terbesar pada modul ini, diuji lewat alamat yang benar-benar dipanggil layar.
	// Sampai 2026-09-24 layar detail selalu menampilkan tepat satu ucapan, karena utasnya
	// dibaca dari tabel yang salah.
	server := newTestServer(t, branchLogin)

	_, before := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0005", "ASM")
	first, _ := before["pesan"].([]any)
	require.Len(t, first, 1)

	response, _ := server.post(t,
		"/api/inbox-komunikasi-cabang/komunikasi/KOM-0005/balas", "ASM",
		`{"pesan":"Sudah kami tindak lanjuti."}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	_, after := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0005", "ASM")
	second, _ := after["pesan"].([]any)
	require.Len(t, second, 2)

	last, _ := second[1].(map[string]any)
	require.Equal(t, "Sudah kami tindak lanjuti.", last["pesan"])
	require.Equal(t, "pictekniks", last["pengirim"])
}

func TestEachUtteranceCarriesExactlyTheThreeFieldsTheSectionDraws(t *testing.T) {
	// Tanggal, Pengirim, Pesan. Tidak lebih.
	//
	// Isian `jawaban`, `penjawab`, dan `tanggal_jawaban` yang sempat dikirim lahir dari
	// tabel yang keliru: balasan BUKAN isian pada sebuah ucapan, ia ucapan tersendiri.
	server := newTestServer(t, branchLogin)

	_, body := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0002", "ASM")

	messages, _ := body["pesan"].([]any)
	require.NotEmpty(t, messages)

	first, _ := messages[0].(map[string]any)
	require.ElementsMatch(t, []string{"tanggal", "pengirim", "pesan"}, keysOf(first))
}

func TestAConversationWithoutHistoryOpensEmptyInsteadOfAnsweringNotFound(t *testing.T) {
	// KOM-0003 punya kepala tanpa utas — percakapan nyata yang riwayatnya belum pernah
	// ditulis. Menjawab 404 untuknya akan dilaporkan sebagai kerusakan.
	server := newTestServer(t, otherLogin)

	response, body := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/KOM-0003", "ASM")

	require.Equal(t, http.StatusOK, response.StatusCode)
	messages, _ := body["pesan"].([]any)
	require.Empty(t, messages)
}

func TestANewMessageOpensAsAOneUtteranceThreadOverHTTP(t *testing.T) {
	// Uji rantai lewat HTTP: "Kirim Pesan" menulis ke dua tabel, dan layar detail membaca
	// yang kedua. Bila salah satunya terlewat, percakapan baru terbuka dengan utas kosong.
	server := newTestServer(t, branchLogin)

	_, created := server.post(t, "/api/inbox-komunikasi-cabang/pesan", "ASM",
		`{"tujuan":"PUSAT","cabang":"","pesan":"Mohon konfirmasi."}`)
	id, _ := created["komunikasi"].(string)
	require.NotEmpty(t, id)

	response, body := server.call(t,
		http.MethodGet, "/api/inbox-komunikasi-cabang/komunikasi/"+id, "ASM")
	require.Equal(t, http.StatusOK, response.StatusCode)

	messages, _ := body["pesan"].([]any)
	require.Len(t, messages, 1)
}

// keysOf mengumpulkan nama field sebuah objek JSON.
func keysOf(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	return keys
}
