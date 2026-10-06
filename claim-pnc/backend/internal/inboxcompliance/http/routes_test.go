package inboxcompliancehttp_test

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
	authusecase "claim-pnc/internal/auth/usecase"
	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	inboxcomplianceusecase "claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	inboxcompliancehttp "claim-pnc/internal/inboxcompliance/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Rabu 2026-09-23 pukul 09.00 WIB, dipakai sebagai jam tetap.
var now = time.Date(2026, time.September, 23, 9, 0, 0, 0, clock.ZoneWIB)

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa antrean satu badan hukum tidak
// pernah terbaca lewat badan hukum lain. Menguji handler secara terpisah tidak dapat
// membuktikan keduanya.
type testServer struct {
	server *httptest.Server
	token  string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	identitySystem, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	authService, err := authusecase.NewService(authusecase.Options{
		Identity:        identitySystem,
		UserRepo:        authmemory.NewUserRepo(),
		SessionRepo:     authmemory.NewSessionRepo(),
		Clock:           clock.FixedAt(now),
		SessionLifetime: 30 * time.Minute,
	})
	require.NoError(t, err)

	sent := now.Add(-26 * time.Hour)
	postAuditSent := now.Add(-72 * time.Hour)

	// ASM punya antrean; ASI sengaja dibiarkan KOSONG — ia yang membuktikan pemisahan
	// antarentitas.
	asm := memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now.Add(-3 * time.Hour),
		Item: inboxcompliance.WorkItem{
			CaseID:             "PNC-900101",
			Reference:          "ASM-FW-GCNMFW-WORK PNC-900101",
			PolicyNumber:       "00.000.0000.00001",
			InsuredName:        "Tertanggung Contoh Satu",
			ComplianceSentDate: &sent,
		},
	},
		// Baris tab Post Audit. Ia TIDAK punya workbasket dan TIDAK punya status —
		// tabelnya memang tidak menyimpan keduanya.
		memory.Row{
			Tab: inboxcompliance.TabPostAudit,
			// Nomornya berbentuk `CPL-nn`, bukan karangan seperti `PNC-PA-1`.
			//
			// Bentuk itu menentukan hasil uji, bukan sekadar enak dibaca: tab ini
			// diurutkan sebagai TEKS, sehingga nomor contoh yang berawalan huruf lain
			// akan menghasilkan urutan yang tidak mungkin terjadi pada data sebenarnya —
			// dan ujinya akan membuktikan sesuatu yang tidak ada.
			Item: inboxcompliance.WorkItem{
				CaseID:            "CPL-19",
				Reference:         "ASM-FW-GCNMFW-WORK PNC-PA-1",
				PolicyNumber:      "00.000.0000.00009",
				InsuredName:       "Tertanggung Contoh Sembilan",
				PostAuditSentDate: &postAuditSent,
				ComplianceRemarks: "Contoh catatan compliance",
			},
		},
	)
	asi := memory.NewStore()

	service, err := inboxcomplianceusecase.NewService(inboxcomplianceusecase.Options{
		RepoSelector: func(alias string) (inboxcompliance.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Clock: clock.FixedAt(now),
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Keduanya dideklarasikan sebagai tipe fungsi TANPA NAMA, bukan dibiarkan mengambil
	// tipe bernama milik salah satu modul. Setiap modul mendeklarasikan tipe penulisnya
	// sendiri — persis supaya modul tidak saling mengimpor.
	var writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any) = func(
		w http.ResponseWriter, r *http.Request, status int, body any,
	) {
		authhttp.WriteJSON(w, r, status, body, logger)
	}
	var writeError func(w http.ResponseWriter, r *http.Request, err error) = portalhttp.WithPortalError(
		authhttp.WriteError(logger), writeResponse,
	)

	handler := inboxcompliancehttp.NewHandler(inboxcompliancehttp.Options{
		Service: service,
		Logger:  logger,

		// Jembatan ke modul auth dirakit sama seperti cmd/claimpnc, bukan dibiarkan nil.
		//
		// Dibiarkan nil, seluruh jalur tulis akan dijawab 409 "pemanggil tidak dikenal" —
		// dan ujinya akan lulus atas penolakan itu, bukan atas pengirimannya. Ini satu-
		// satunya bagian rakitan yang BEDA dari jalur baca, sehingga ia yang paling mudah
		// tertinggal saat modul ini dipasang di tempat lain.
		GetCaller: func(ctx context.Context) (inboxcompliancehttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return inboxcompliancehttp.Caller{}, false
			}
			return inboxcompliancehttp.Caller{Login: base.User.Login}, true
		},

		WriteJSON:           inboxcompliancehttp.JSONWriter(writeResponse),
		FallbackErrorWriter: inboxcompliancehttp.ErrorWriter(writeError),
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
				inboxcompliancehttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server}
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

// call menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim, dan
// token kosong berarti permintaan dikirim tanpa sesi.
func (p *testServer) call(
	t *testing.T, path, portalAlias, token string,
) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, p.server.URL+path, nil)
	require.NoError(t, err)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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

const route = "/api/inbox-compliance"

func TestDaftarMenyebutEntitasYangMenjawabnya(t *testing.T) {
	// Pada aplikasi yang melayani empat badan hukum, "antrean siapa ini" tidak boleh hanya
	// diandaikan — server menyebutkannya (`R-20`).
	server := newTestServer(t)

	response, content := server.call(t, route, "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])

	rows, ok := content["baris"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)

	first, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "PNC-900101", first["nomor_case"])

	// 26 jam pada hari kerja, tanpa akhir pekan yang terpotong.
	require.Equal(t, "1 days 2 hours ago", first["aging"])
	require.InDelta(t, 26.0, first["aging_jam"], 0.001)
}

func TestEntitasLainTidakMelihatAntreanEntitasIni(t *testing.T) {
	// Inilah uji yang membenarkan seluruh mekanisme pemilihan portal.
	server := newTestServer(t)

	_, asm := server.call(t, route, "ASM", server.token)
	require.Len(t, asm["baris"], 1)

	response, asi := server.call(t, route, "ASI", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASI", asi["portal"])
	require.Empty(t, asi["baris"])
}

// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
func TestTanpaPortalDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, route, "", server.token)
	require.NotEqual(t, http.StatusOK, response.StatusCode)
}

func TestTanpaSesiDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.call(t, route, "ASM", "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Keterangan layar menyebutkan kedua tab, termasuk yang belum dapat dilayani.
func TestKeteranganLayarMenyebutKeduaTab(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, route+"/tab", "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "ASM", content["portal"])
	require.Equal(t, inboxcompliance.TabCompliance, content["tab_bawaan"])

	tabs, ok := content["tab"].([]any)
	require.True(t, ok)
	require.Len(t, tabs, 2)

	postAudit, ok := tabs[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, inboxcompliance.TabPostAudit, postAudit["kode"])
	require.Equal(t, true, postAudit["tersedia"])
	require.Empty(t, postAudit["penghalang"])

	require.NotEmpty(t, content["keterbatasan"],
		"keterbatasan dikirim sebagai data supaya hilang sendiri saat penghalangnya hilang")
}

// Tab Post Audit dilayani, dan isinya TERPISAH dari tab Compliance.
func TestTabPostAuditDilayani(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(
		t, route+"?tab="+inboxcompliance.TabPostAudit, "ASM", server.token)

	require.Equal(t, http.StatusOK, response.StatusCode)

	rows, ok := content["baris"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)

	first, ok := rows[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "CPL-19", first["nomor_case"])
	require.Equal(t, "Contoh catatan compliance", first["catatan_compliance"])
	require.NotNil(t, first["tanggal_kirim_post_audit"])

	// Tab ini tidak punya kolom Aging, dan tabelnya tidak menyimpan Tanggal Kirim
	// Compliance — sehingga Aging-nya kosong, bukan "0 hours ago".
	require.Empty(t, first["aging"])
	require.Nil(t, first["aging_jam"])
}

// post mengirim satu permintaan tulis, dengan badan JSON apa adanya.
//
// Terpisah dari call karena jalur tulis punya tiga hal yang tidak dimiliki jalur baca:
// kata kerja POST, badan permintaan, dan pemanggil yang harus dikenali. Memakai satu
// pembantu untuk keduanya akan menyembunyikan ketiganya.
func (p *testServer) post(
	t *testing.T, path, portalAlias, token, body string,
) (*http.Response, map[string]any) {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodPost, p.server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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

const postAuditRoute = route + "/post-audit"

// Pengiriman ke Post Audit berhasil lewat HTTP, dan hasilnya BENAR-BENAR tampil di tabnya.
//
// Uji ini sengaja menempuh dua permintaan, bukan satu. Permintaan pertama saja hanya
// membuktikan handler-nya menjawab 201; yang dipertaruhkan sebenarnya adalah apakah baris
// itu tampil ketika petugas membuka tab Post Audit — dan di antara keduanya ada pemetaan
// DTO, pemilihan tab, dan pengurutan yang masing-masing bisa menjatuhkannya tanpa satu pun
// galat muncul.
func TestKirimKePostAuditTampilDiTabnya(t *testing.T) {
	server := newTestServer(t)

	response, content := server.post(t, postAuditRoute, "ASM", server.token,
		`{"referensi":"ASM-FW-GCNMFW-WORK PNC-900101","catatan":"Dokumen lengkap"}`)

	require.Equal(t, http.StatusCreated, response.StatusCode)

	// Nomornya berasal dari rentang aplikasi baru, bukan dari rentang terbitan Pega.
	require.Equal(t, "CPL-100001", content["nomor_case"])

	// Lalu tabnya dibuka seperti petugas membukanya.
	listing, listed := server.call(
		t, route+"?tab="+inboxcompliance.TabPostAudit, "ASM", server.token)
	require.Equal(t, http.StatusOK, listing.StatusCode)

	rows, ok := listed["baris"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 2, "baris contoh ditambah satu baris yang baru dikirim")

	// Baris yang baru dikirim TIDAK berada di paling atas.
	//
	// Pengurutannya teks menurun, meniru Pega, dan `CPL-100001` lebih kecil daripada
	// `CPL-19` sebagai teks — perbandingannya berhenti pada `0` lawan `9`. Urutan inilah
	// yang akan dilihat petugas, dan ia ditulis di sini supaya tidak "dirapikan" menjadi
	// urutan angka, yang akan menyimpang dari Pega.
	urutan := []string{}
	for _, row := range rows {
		urutan = append(urutan, row.(map[string]any)["nomor_case"].(string))
	}
	require.Equal(t, []string{"CPL-19", "CPL-100001"}, urutan)

	baru, ok := rows[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-900101", baru["no_klaim"])
	require.Equal(t, "Tertanggung Contoh Satu", baru["nama_tertanggung"])
	require.Equal(t, "00.000.0000.00001", baru["no_polis"])
	require.Equal(t, "Dokumen lengkap", baru["catatan_compliance"])
	require.NotEmpty(t, baru["tanggal_kirim_post_audit"])
}

// Klaim yang TIDAK sedang menunggu di antrean Compliance ditolak, bukan dikirim diam-diam.
func TestKirimKePostAuditMenolakKlaimDiLuarAntrean(t *testing.T) {
	server := newTestServer(t)

	response, content := server.post(t, postAuditRoute, "ASM", server.token,
		`{"referensi":"ASM-FW-GCNMFW-WORK PNC-TIDAK-ADA"}`)

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.NotEmpty(t, content["kode"])
}

// Jalur tulis berada di balik sesi dan di balik portal, sama seperti jalur baca.
//
// Keduanya diuji di sini pula, tidak dianggap sudah tercakup uji jalur baca: rute tulis
// dipasang lewat pemanggilan Mount yang sama, tetapi menyimpang satu baris saja sudah
// cukup membuatnya terbuka — dan pada jalur yang MENULIS ke basis data entitas, akibatnya
// bukan kebocoran bacaan melainkan penulisan lintas badan hukum (`R-20`).
func TestKirimKePostAuditMenuntutSesiDanPortal(t *testing.T) {
	server := newTestServer(t)
	body := `{"referensi":"ASM-FW-GCNMFW-WORK PNC-900101"}`

	tanpaSesi, _ := server.post(t, postAuditRoute, "ASM", "", body)
	require.Equal(t, http.StatusUnauthorized, tanpaSesi.StatusCode)

	tanpaPortal, _ := server.post(t, postAuditRoute, "", server.token, body)
	require.NotEqual(t, http.StatusCreated, tanpaPortal.StatusCode,
		"permintaan tanpa portal TIDAK boleh jatuh ke portal utama (`R-20`)")
}

func TestTabTidakDikenalDijawab422(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(t, route+"?tab=tidak-ada", "ASM", server.token)

	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
}

// Parameter halaman yang tidak dapat dibaca DIBETULKAN, bukan menggagalkan permintaan.
func TestParameterHalamanRusakDibetulkan(t *testing.T) {
	server := newTestServer(t)

	response, content := server.call(
		t, route+"?halaman=abc&ukuran=-5", "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)

	pagination, ok := content["paginasi"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, pagination["halaman"])
	require.EqualValues(t, inboxcompliance.DefaultPageSize, pagination["ukuran"])
}
