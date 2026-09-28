package casestudyclaimhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
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
	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/casestudyclaim/repo/memory"
	casestudyusecase "claim-pnc/internal/casestudyclaim/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpserver"
	"claim-pnc/internal/portal"

	authhttp "claim-pnc/internal/auth/http"
	casestudyclaimhttp "claim-pnc/internal/casestudyclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Jumat 2026-09-25 pukul 10.00 WIB, dipakai sebagai jam tetap.
var now = time.Date(2026, time.September, 25, 10, 0, 0, 0, clock.ZoneWIB)

const route = "/api/case-study-claim"

// testServer merakit aplikasi sama seperti cmd/claimpnc — lengkap dengan middleware sesi
// dan middleware portal.
//
// Dirakit utuh dengan sengaja: yang diuji di sini bukan handler-nya sendiri melainkan
// KONTRAKNYA — bahwa rutenya berada di balik sesi, dan bahwa klaim satu badan hukum tidak
// pernah terbaca lewat badan hukum lain. Menguji handler secara terpisah tidak dapat
// membuktikan keduanya.
type testServer struct {
	server *httptest.Server
	token  string
	asm    *memory.Store
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

	// ASM punya klaim; ASI sengaja dibiarkan KOSONG — ia yang membuktikan pemisahan
	// antarentitas.
	asm := memory.NewSampleStore()
	asi := memory.NewStore()

	service, err := casestudyusecase.NewService(casestudyusecase.Options{
		RepoSelector: func(alias string) (casestudyclaim.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
		Logger: slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
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

	handler := casestudyclaimhttp.NewHandler(casestudyclaimhttp.Options{
		Service: service,
		GetCaller: func(ctx context.Context) (casestudyclaimhttp.Caller, bool) {
			base, ok := authhttp.CallerFromContext(ctx)
			if !ok {
				return casestudyclaimhttp.Caller{}, false
			}
			return casestudyclaimhttp.Caller{Login: base.User.Login}, true
		},
		Logger:              logger,
		WriteJSON:           casestudyclaimhttp.JSONWriter(writeResponse),
		FallbackErrorWriter: casestudyclaimhttp.ErrorWriter(writeError),
		Location:            clock.ZoneWIB,
		Now:                 func() time.Time { return now },
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
				casestudyclaimhttp.Mount(protected, handler, portalDeps)
			})
		},
	}))
	t.Cleanup(server.Close)

	p := &testServer{server: server, asm: asm}
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

// do menjalankan satu permintaan. portalAlias kosong berarti header tidak dikirim, dan
// token kosong berarti permintaan dikirim tanpa sesi.
func (p *testServer) do(
	t *testing.T, method, path, portalAlias, token, body string,
) *http.Response {
	t.Helper()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	request, err := http.NewRequest(method, p.server.URL+path, reader)
	require.NoError(t, err)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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
	return response
}

func (p *testServer) get(t *testing.T, path, portalAlias, token string) (*http.Response, map[string]any) {
	t.Helper()

	response := p.do(t, http.MethodGet, path, portalAlias, token, "")
	content := map[string]any{}
	_ = json.NewDecoder(response.Body).Decode(&content)
	return response, content
}

// ── Sesi dan portal ────────────────────────────────────────────────────────────────

// Seluruh rute berada di balik sesi.
func TestSeluruhRuteMenuntutSesi(t *testing.T) {
	server := newTestServer(t)

	for _, path := range []string{
		route + "/penyaring",
		route + "?dari=2024-01-01&sampai=2024-12-31",
		route + "/unduh?dari=2024-01-01&sampai=2024-12-31",
	} {
		response, _ := server.get(t, path, "ASM", "")
		require.Equal(t, http.StatusUnauthorized, response.StatusCode, path)
	}
}

// Permintaan tanpa portal DITOLAK, tidak dilayani portal utama sebagai cadangan.
//
// Barisnya adalah klaim di atas Rp 5 miliar beserta nama tertanggungnya; jatuh ke koneksi
// default berarti menampilkannya di layar badan hukum lain tanpa satu pun galat (`R-20`).
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.get(t, route+"?dari=2024-01-01&sampai=2024-12-31", "", server.token)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Entitas yang datanya kosong menjawab daftar KOSONG, bukan data entitas lain.
func TestEntitasLainTidakMelihatKlaimEntitasIni(t *testing.T) {
	server := newTestServer(t)

	_, asm := server.get(t, route+"?dari=2000-01-01&sampai=2099-12-31", "ASM", server.token)
	require.Equal(t, float64(8), asm["total"])

	_, asi := server.get(t, route+"?dari=2000-01-01&sampai=2099-12-31", "ASI", server.token)
	require.Equal(t, float64(0), asi["total"])
}

// ── Keterangan layar ───────────────────────────────────────────────────────────────

// Keterangan layar memuat kedua dropdown, seluruh kolom, dan ambangnya.
//
// Ia TIDAK menyentuh basis data, sehingga layar tetap tergambar lengkap meski Oracle
// sedang tidak dapat dihubungi.
func TestKeteranganLayarMemuatPenyaringDanKolom(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t, route+"/penyaring", "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)

	business, ok := content["bisnis"].([]any)
	require.True(t, ok)
	require.Len(t, business, 4)
	require.Equal(t, "PA", business[0].(map[string]any)["label"])

	status, ok := content["status"].([]any)
	require.True(t, ok)
	require.Len(t, status, 2)
	require.Equal(t, "CLAIM ON PROGRESS/ ACCEPT", status[0].(map[string]any)["label"])

	columns, ok := content["kolom"].([]any)
	require.True(t, ok)
	require.Len(t, columns, 24)
	require.Equal(t, "PNC Case ID", columns[0].(map[string]any)["judul"])

	// Ambang dikirim dalam SEN, dan layar menyebutkannya kepada pengguna — sehingga
	// "kenapa klaim saya tidak muncul" terjawab tanpa membuka kode.
	require.Equal(t, float64(500_000_000_000), content["ambang_nilai_klaim"])

	require.NotEmpty(t, content["catatan_periode"])
	require.NotEmpty(t, content["catatan_kolom_kembar"])
}

// ── Penyaring ──────────────────────────────────────────────────────────────────────

// Periode yang kosong dijawab 422 beserta alasannya, bukan daftar kosong tanpa penjelasan.
//
// Di Pega, isian yang kosong menghasilkan nol baris secara diam. Kedua sistem menampilkan
// hasil yang sama; yang berbeda hanyalah pengguna diberi tahu sebabnya.
func TestPeriodeKosongDijawab422(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t, route, "ASM", server.token)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "validasi_gagal", content["kode"])
	require.Equal(t, "dari", content["isian"])
}

func TestPeriodeTerbalikDijawab422(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t,
		route+"?dari=2026-01-01&sampai=2024-12-31", "ASM", server.token)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	require.Equal(t, "sampai", content["isian"])
}

// Inilah perilaku yang paling mudah disangka cacat: HARI dan BULAN dibuang.
//
// Memilih 1 sampai 31 Januari 2024 mengembalikan SELURUH klaim tahun 2024, termasuk yang
// terjadi bulan Juli. Perilakunya sama dengan layar lama (`P-5`), dan respons menyebutkan
// tahun yang benar-benar dipakai supaya layar dapat menyatakannya.
func TestHanyaTahunDariTanggalYangDipakaiMenyaring(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t,
		route+"?dari=2024-01-01&sampai=2024-01-31", "ASM", server.token)
	require.Equal(t, http.StatusOK, response.StatusCode)

	period, ok := content["periode"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "2024", period["tahun_awal"])
	require.Equal(t, "2024", period["tahun_akhir"])

	// STD-0002 terjadi pada Juli 2024 dan tetap muncul.
	require.Equal(t, float64(3), content["total"])
}

func TestPenyaringBisnisTidakDikenalDitolak(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t,
		route+"?dari=2024-01-01&sampai=2024-12-31&bisnis=999", "ASM", server.token)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "bisnis", content["isian"])
}

func TestPenyaringStatusTidakDikenalDitolak(t *testing.T) {
	server := newTestServer(t)

	response, content := server.get(t,
		route+"?dari=2024-01-01&sampai=2024-12-31&status=selesai", "ASM", server.token)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Equal(t, "status", content["isian"])
}

// Batas halaman yang melebihi maksimum DITOLAK, bukan dipangkas diam-diam.
//
// Klien yang meminta seribu baris lalu menerima seratus tanpa diberi tahu akan menampilkan
// daftar yang ia kira lengkap.
func TestBatasMelebihiMaksimumDitolak(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.get(t,
		route+"?dari=2024-01-01&sampai=2024-12-31&batas=1000", "ASM", server.token)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// ── Bentuk baris ───────────────────────────────────────────────────────────────────

// Baris memuat kedua kolom kembar berisi nilai yang SAMA, dan nilai kosong tetap `null`.
func TestBentukBarisSesuaiKontrak(t *testing.T) {
	server := newTestServer(t)

	_, content := server.get(t,
		route+"?dari=2025-01-01&sampai=2025-12-31&status=reject", "ASM", server.token)

	rows, ok := content["baris"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)

	row := rows[0].(map[string]any)
	require.Equal(t, "STD-0003", row["nomor_klaim"])

	// Dua kolom, satu nilai — persis seperti grid Pega.
	require.Equal(t, row["nature_of_loss"], row["cause_of_loss"])
	require.NotEmpty(t, row["cause_of_loss"])

	// Nilai yang kosong dikirim `null`, BUKAN 0. Perbedaannya menentukan: pada berkas yang
	// dibuka di pengolah angka, nol ikut terhitung dalam rata-rata.
	require.Nil(t, row["deductible"])
	require.Nil(t, row["share_asm"])
	require.Nil(t, row["lack_of_doc"])

	// Nilai uang dikirim dalam SEN. Rp 5.500.000.000 = 550.000.000.000 sen.
	require.Equal(t, float64(550_000_000_000), row["nilai_klaim_100"])

	// Tanggal dikirim sebagai tanggal saja, bukan timestamp.
	require.Equal(t, "2025-01-18", row["tanggal_kejadian"])

	// Bulan klaim dua digit, sama persis dengan `to_char(…,'mm')` di Pega.
	require.Equal(t, "01", row["bulan_klaim"])
}

// ── Unduhan ────────────────────────────────────────────────────────────────────────

// Berkas CSV berjudul kolom yang SAMA PERSIS dengan grid, dalam urutan yang sama.
func TestUnduhanBerisiKolomYangSamaDenganGrid(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodGet,
		route+"/unduh?dari=2025-01-01&sampai=2025-12-31&status=reject", "ASM", server.token, "")
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, response.Header.Get("Content-Disposition"), "case-study-claim-20260925-100000.csv")

	records, err := csv.NewReader(response.Body).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2, "satu baris judul, satu baris data")

	require.Equal(t, casestudyclaim.ColumnTitles(), records[0])

	row := records[1]
	require.Equal(t, "STD-0003", row[0])

	// Nilai uang ditulis sebagai ANGKA berdesimal titik, bukan "Rp 5.500.000.000,00":
	// berkas ini dibuka di pengolah angka, dan teks bersatuan tidak dapat dijumlahkan.
	require.Equal(t, "5500000000", row[15], "NILAI KLAIM 100%")

	// Yang kosong menjadi sel KOSONG, bukan "0".
	require.Equal(t, "", row[12], "ASM SHARE")
	require.Equal(t, "", row[13], "Deductible")
}

// Unduhan menolak periode kosong dengan cara yang SAMA dengan daftar.
//
// Keduanya memakai pembaca penyaring yang sama; uji ini menjaga keduanya tidak menyimpang.
func TestUnduhanMenolakPeriodeKosong(t *testing.T) {
	server := newTestServer(t)

	response, _ := server.get(t, route+"/unduh", "ASM", server.token)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
}

// ── Simpan catatan ─────────────────────────────────────────────────────────────────

func TestSimpanCatatanMengubahBarisnya(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodPut, route+"/STD-0001/catatan", "ASM", server.token,
		`{"catatan":"  sudah ditelaah  "}`)
	require.Equal(t, http.StatusOK, response.StatusCode)

	body := map[string]any{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, "STD-0001", body["nomor_klaim"])
	require.Equal(t, "sudah ditelaah", body["catatan"], "spasi di ujung dipangkas")

	_, content := server.get(t, route+"?dari=2024-01-01&sampai=2024-12-31", "ASM", server.token)
	rows := content["baris"].([]any)
	require.Equal(t, "sudah ditelaah", rows[0].(map[string]any)["remark"])
}

// Klaim yang tidak ada dijawab 404 beserta pesan yang menyuruh MEMUAT ULANG.
//
// Bukan 500 beserta pesan yang menyuruh menghubungi tim teknis: baris memang dapat hilang
// di antara saat daftar dibaca dan saat Save ditekan.
func TestSimpanCatatanPadaKlaimYangTidakAdaDijawab404(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodPut, route+"/STD-9999/catatan", "ASM", server.token,
		`{"catatan":"apa pun"}`)
	require.Equal(t, http.StatusNotFound, response.StatusCode)

	body := map[string]any{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, "tidak_ditemukan", body["kode"])
	require.Contains(t, body["pesan"], "Muat ulang")
}

// Field asing DITOLAK, tidak diabaikan.
//
// Klien yang mengirim `{"remark": …}` alih-alih `{"catatan": …}` akan mengira catatannya
// tersimpan padahal yang tersimpan adalah teks kosong — dan kosong di sini berarti
// MENGHAPUS catatan yang sudah ada.
func TestFieldAsingPadaBadanPermintaanDitolak(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodPut, route+"/STD-0001/catatan", "ASM", server.token,
		`{"remark":"salah nama field"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestSimpanCatatanTerlaluPanjangDijawab422(t *testing.T) {
	server := newTestServer(t)

	long := strings.Repeat("a", casestudyclaim.MaxRemarkLength+1)
	response := server.do(t, http.MethodPut, route+"/STD-0001/catatan", "ASM", server.token,
		`{"catatan":"`+long+`"}`)
	require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)

	body := map[string]any{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
	require.Equal(t, "catatan", body["isian"])
}

// Menyimpan tanpa portal DITOLAK — taruhannya di jalur tulis lebih besar daripada baca.
func TestSimpanCatatanTanpaPortalDitolak(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodPut, route+"/STD-0001/catatan", "", server.token,
		`{"catatan":"apa pun"}`)
	require.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestSimpanCatatanMenuntutSesi(t *testing.T) {
	server := newTestServer(t)

	response := server.do(t, http.MethodPut, route+"/STD-0001/catatan", "ASM", "",
		`{"catatan":"apa pun"}`)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

// Modul ini tidak punya rute tulis lain. Rute yang tidak didaftarkan dijawab chi dengan 405,
// sehingga penambahan rute tulis kelak menjadi keputusan sadar — bukan sesuatu yang lolos
// review.
func TestTidakAdaRuteTulisLain(t *testing.T) {
	server := newTestServer(t)

	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		response := server.do(t, method, route, "ASM", server.token, `{}`)
		require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode, method)
	}
}
