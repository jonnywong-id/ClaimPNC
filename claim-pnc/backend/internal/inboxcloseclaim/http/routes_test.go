package inboxcloseclaimhttp_test

import (
	"context"
	"encoding/csv"
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

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/inboxcloseclaim/repo/memory"
	"claim-pnc/internal/inboxcloseclaim/usecase"

	closeclaimhttp "claim-pnc/internal/inboxcloseclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testNow tetap, supaya kolom "Lama Waktu Klaim" dapat diperiksa dengan angka pasti.
var testNow = time.Date(2026, time.September, 23, 3, 0, 0, 0, time.UTC)

// wib dipakai agar uji tidak bergantung pada basis data zona waktu mesin penjalan.
var wib = time.FixedZone("WIB", 7*60*60)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
//
// Ia harus ada di daftar portal contoh DAN dinyatakan siap, karena middleware membedakan
// "portal tidak dikenal" dari "portal belum siap".
const testPortal = "ASM"

// testServer membentuk server dengan gerbang pengajuan TERBUKA.
//
// Gerbang yang sesungguhnya — When rule `IsGCNMUser` — selalu salah, sehingga tanpa ini
// seluruh uji pengajuan di bawah hanya akan membuktikan bahwa gerbangnya menutup. Yang
// hendak dijaga justru pemetaan galat, bentuk respons, dan penolakan permintaan ganda — kode
// yang menyala pada hari gerbang itu dibuka.
//
// Bahwa gerbangnya benar-benar menutup dijaga `testServerGerbangTertutup` di bawahnya.
func testServer(t *testing.T, login string) (http.Handler, *memory.Store) {
	t.Helper()
	return buildServer(t, login, func() bool { return true })
}

// testServerGerbangTertutup memakai gerbang BAWAAN — yaitu aturan yang sesungguhnya.
func testServerGerbangTertutup(t *testing.T, login string) (http.Handler, *memory.Store) {
	t.Helper()
	return buildServer(t, login, nil)
}

func buildServer(
	t *testing.T,
	login string,
	canRequest func() bool,
) (http.Handler, *memory.Store) {
	t.Helper()

	store := memory.NewStoreWithSamples()

	// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport, bukan
	// pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
	options := usecase.Options{
		Claims:   func(string) (inboxcloseclaim.Repo, error) { return store, nil },
		Requests: func(string) (inboxcloseclaim.RequestRepo, error) { return store, nil },
		IDs:      memory.IDGenerator{},
		Clock:    memory.FixedClock{At: testNow},
	}
	if canRequest != nil {
		options.CanRequest = canRequest
	}

	service, err := usecase.NewService(options)
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}

	// Galat portal dipetakan modul portal, persis seperti rantai di cmd/claimpnc. Bila
	// rantai ini tidak ditiru, uji penolakan portal akan lulus di sini tetapi gagal di
	// aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode":  "galat_internal",
				"pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := closeclaimhttp.NewHandler(closeclaimhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (closeclaimhttp.Caller, bool) {
			return closeclaimhttp.Caller{Login: login, Name: "Budi Santoso"}, true
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: closeclaimhttp.ErrorWriter(writeError),
		Location:            wib,
		Now:                 func() time.Time { return testNow },
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		closeclaimhttp.Mount(api, handler, portalDeps)
	})
	return router, store
}

// get mengirim permintaan BESERTA header portal.
//
// Portalnya disertakan di sini, bukan di tiap uji, supaya uji yang sengaja
// menghilangkannya — TestPermintaanTanpaPortalDitolak — terlihat jelas sebagai pengecualian.
func get(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Portal", testPortal)

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func post(t *testing.T, server http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("X-Portal", testPortal)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestDaftarDikembalikanBesertaTotal(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := get(t, server, "/api/inbox-close-claim")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.NotEmpty(t, body["klaim"])
	require.Greater(t, body["total"], float64(0))
	require.Equal(t, true, body["permintaan_terbaca"])
}

// TestSelisihTerencanaDinyatakanDiLayar menjaga `D-54`.
//
// Selisih terhadap Pega dinyatakan kepada pengguna, bukan disembunyikan sebagai detail
// teknis. Ketiganya sudah diputuskan Work Owner 2026-09-23, dan menyatakannya di layar
// itulah yang membuat keputusan itu terlihat oleh orang yang memakai layarnya.
func TestSelisihTerencanaDinyatakanDiLayar(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	body := decode(t, get(t, server, "/api/inbox-close-claim"))
	selisih, ok := body["selisih_terencana"].([]any)
	require.True(t, ok)
	require.Len(t, selisih, 3)
}

// TestKolomLamaWaktuKlaimBerisiAngkaHari menjaga selisih terencana yang paling terlihat.
//
// Di Pega kolom itu menampilkan tanggal pendaftaran untuk kedua kalinya.
func TestKolomLamaWaktuKlaimBerisiAngkaHari(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	body := decode(t, get(t, server, "/api/inbox-close-claim?no_klaim=PNCN.26.0001"))
	claims := body["klaim"].([]any)
	require.Len(t, claims, 1)

	claim := claims[0].(map[string]any)
	// Contohnya didaftarkan 12 Januari dan ditutup 41 hari kemudian.
	require.Equal(t, float64(41), claim["lama_hari"])
	require.NotEqual(t, claim["tanggal_pendaftaran"], claim["lama_hari"])
}

func TestPenyaringDikembalikanSebagaiBentukLayar(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := get(t, server, "/api/inbox-close-claim/penyaring")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Len(t, body["lini_bisnis"], 5, "kelima cabang activity lama, tidak lebih")
	require.Len(t, body["status_transfer"], 3)
	require.Len(t, body["status_bayar"], 3)
}

func TestPilihanPenyaringYangTidakDikenalDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	require.Equal(t, http.StatusBadRequest,
		get(t, server, "/api/inbox-close-claim?lini=MBU").Code)
	require.Equal(t, http.StatusBadRequest,
		get(t, server, "/api/inbox-close-claim?status_bayar=ENTAH").Code)
}

// TestBatasMelebihiMaksimumDitolak menjaga `10-API-STRATEGY.md` §4.
//
// Ditolak, bukan dipangkas diam-diam: klien yang meminta seribu baris lalu menerima seratus
// tanpa diberi tahu akan menampilkan daftar yang ia kira lengkap.
func TestBatasMelebihiMaksimumDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	require.Equal(t, http.StatusBadRequest,
		get(t, server, "/api/inbox-close-claim?batas=1000").Code)
}

// TestPermintaanTanpaPortalDitolak menjaga `R-20`.
//
// Pada modul ini akibatnya melampaui tampilan: portal yang salah berarti permintaan ReOpen
// tercatat di basis data badan hukum yang keliru.
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	request := httptest.NewRequest(http.MethodGet, "/api/inbox-close-claim", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code,
		"permintaan tanpa portal tidak boleh dilayani portal utama sebagai cadangan")
}

func TestReopenMencatatPermintaan(t *testing.T) {
	server, store := testServer(t, "BUDI")

	recorder := post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"reopen","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001","alasan":"dokumen susulan"}`)

	// 201, bukan 200: sebuah PERMINTAAN dibuat. 200 akan menyiratkan tindakannya sendiri
	// sudah dijalankan, dan itu justru yang belum terjadi.
	require.Equal(t, http.StatusCreated, recorder.Code)

	body := decode(t, recorder)
	permintaan := body["permintaan"].(map[string]any)
	require.Equal(t, "reopen", permintaan["jenis"])
	require.Equal(t, "menunggu", permintaan["status"])
	require.Equal(t, "New", permintaan["efek_status_kerja"])
	require.Equal(t, "1164", permintaan["efek_status_klaim"])
	require.Equal(t, "BUDI", permintaan["pemohon"])

	// Pesannya menyebut apa yang SUDAH dan BELUM terjadi. Tombol yang berhasil ditekan tanpa
	// perubahan apa pun di layar adalah keadaan yang paling mudah disalahpahami.
	require.Contains(t, body["pesan"], "belum berubah")

	require.Len(t, store.Requests(), 1)
}

func TestSalinKlaimMencatatLingkupYangDisepakati(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"salin","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001"}`)
	require.Equal(t, http.StatusCreated, recorder.Code)

	permintaan := decode(t, recorder)["permintaan"].(map[string]any)
	require.Equal(t, "polis_objek_coverage", permintaan["lingkup_salin"])
}

// TestPermintaanKeduaDijawab409 menjaga tombol yang ditekan dua kali.
func TestPermintaanKeduaDijawab409(t *testing.T) {
	server, _ := testServer(t, "BUDI")
	badan := `{"jenis":"reopen","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001"}`

	require.Equal(t, http.StatusCreated,
		post(t, server, "/api/inbox-close-claim/permintaan", badan).Code)

	recorder := post(t, server, "/api/inbox-close-claim/permintaan", badan)
	// 409, bukan 422: bukan isiannya yang salah melainkan KEADAAN yang berkonflik.
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, "permintaan_masih_menunggu", decode(t, recorder)["kode"])
}

// TestPermintaanAtasKlaimBerjalanDijawab404 menjaga gerbang kedua aksi.
func TestPermintaanAtasKlaimBerjalanDijawab404(t *testing.T) {
	server, store := testServer(t, "BUDI")
	store.AddClaim(inboxcloseclaim.ClosedClaim{
		ClaimID:       "ASM-FW-GCNMFW-WORK PNCN.26.7777",
		ClaimNumber:   "PNCN.26.7777",
		ProcessStatus: "New",
	})

	recorder := post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"reopen","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.7777"}`)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "klaim_tidak_ditemukan", decode(t, recorder)["kode"])
}

func TestJenisPermintaanYangTidakDikenalDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"hapus","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestBadanPermintaanBerfieldAsingDitolak menjaga kontrak tetap tegas.
//
// Field yang tidak dikenali biasanya berarti klien mengirim sesuatu yang ia kira berpengaruh
// — dan diterima diam-diam, ia tidak berpengaruh apa pun tanpa seorang pun tahu.
func TestBadanPermintaanBerfieldAsingDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"reopen","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001","status":"dijalankan"}`)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestPermintaanTertundaDitandaiDiDaftar adalah uji yang menutup lingkaran kedua aksi.
func TestPermintaanTertundaDitandaiDiDaftar(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	require.Equal(t, http.StatusCreated, post(t, server, "/api/inbox-close-claim/permintaan",
		`{"jenis":"reopen","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001"}`).Code)

	body := decode(t, get(t, server, "/api/inbox-close-claim?no_klaim=PNCN.26.0001"))
	claim := body["klaim"].([]any)[0].(map[string]any)

	tertunda := claim["permintaan_tertunda"].([]any)
	require.Len(t, tertunda, 1)
	require.Equal(t, "reopen", tertunda[0].(map[string]any)["jenis"])
}

// TestUnduhMengalirkanCSVBerjudulSamaDenganLayar menjaga berkas yang diunduh terbaca sebagai
// salinan apa yang dilihat pengguna.
func TestUnduhMengalirkanCSVBerjudulSamaDenganLayar(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	recorder := get(t, server, "/api/inbox-close-claim/unduh")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "inbox-close-claim-")

	rows, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Greater(t, len(rows), 1)

	require.Equal(t, []string{
		"No Klaim", "No Polis", "Nama Tertanggung", "Nama Bisnis", "Sumber Bisnis",
		"Nama Cabang", "Tanggal Pendaftaran", "Lama Waktu Klaim", "PIC Teknik", "Admin PNC",
		"Status", "Status Klaim", "Sudah Transfer", "Permintaan Tertunda",
	}, rows[0])
}

// TestMetodeYangTidakDidaftarkanDitolak menjaga permukaan tulis tetap sempit.
//
// Rute yang tidak didaftarkan dijawab chi dengan 405, sehingga penambahan rute tulis kelak
// menjadi keputusan sadar — bukan sesuatu yang lolos review.
func TestMetodeYangTidakDidaftarkanDitolak(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	request := httptest.NewRequest(http.MethodDelete, "/api/inbox-close-claim/permintaan", nil)
	request.Header.Set("X-Portal", testPortal)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
}

// TestGerbangTertutupMenolakPengajuanDengan403 menjaga keputusan Work Owner 2026-09-23.
//
// Penjaganya When rule `IsGCNMUser`, yang isinya `1 = 2` — selalu salah. Akibatnya kedua
// aksi tidak dapat dipakai siapa pun, dan itu memang yang diputuskan.
//
// 403, bukan 401: pemanggilnya SUDAH masuk. Menjawab 401 akan membuat layar mengira sesinya
// habis lalu mengeluarkan pengguna dari aplikasi.
func TestGerbangTertutupMenolakPengajuanDengan403(t *testing.T) {
	server, store := testServerGerbangTertutup(t, "BUDI")

	for _, jenis := range []string{"reopen", "salin"} {
		recorder := post(t, server, "/api/inbox-close-claim/permintaan",
			`{"jenis":"`+jenis+`","klaim_id":"ASM-FW-GCNMFW-WORK PNCN.26.0001"}`)

		require.Equalf(t, http.StatusForbidden, recorder.Code, "jenis %s", jenis)
		require.Equal(t, "tidak_berwenang", decode(t, recorder)["kode"])
	}

	require.Empty(t, store.Requests(),
		"tidak satu pun baris boleh tercatat saat gerbangnya menutup")
}

// TestDaftarMenyatakanPengajuanTertutup menjaga layar TAHU lebih dulu, bukan tahu lewat galat.
//
// Tombol yang tampak dapat ditekan lalu selalu dijawab 403 terbaca sebagai gangguan sistem,
// bukan sebagai kewenangan yang memang tidak ada — dan yang dilaporkan pengguna adalah
// "aplikasinya rusak".
func TestDaftarMenyatakanPengajuanTertutup(t *testing.T) {
	server, _ := testServerGerbangTertutup(t, "BUDI")

	body := decode(t, get(t, server, "/api/inbox-close-claim"))

	require.Equal(t, false, body["boleh_mengajukan"])
	require.NotEmpty(t, body["alasan_tidak_boleh"], "alasannya wajib disebutkan, bukan hanya ditolak")

	// Keterangan pelaksana TIDAK ikut dinyatakan saat pengajuannya sendiri tertutup —
	// menambah kalimat yang tidak berlaku hanya mengaburkan sebab yang sebenarnya.
	require.Equal(t, false, body["pelaksana_belum_ada"])

	// Daftarnya TETAP tampil. Gerbang ini menutup PENGAJUAN, bukan pembacaan.
	require.NotEmpty(t, body["klaim"])
}

// TestDaftarMenyatakanPelaksanaBelumAda menjaga keterangan kedua, yang berlaku saat
// pengajuannya terbuka.
//
// Ditetapkan Work Owner 2026-09-23: siapa yang menjalankan permintaan di sisi Pega belum
// ditentukan. Pengguna yang mengajukan lalu menunggu perubahan yang tidak akan datang akan
// melaporkannya sebagai kegagalan.
func TestDaftarMenyatakanPelaksanaBelumAda(t *testing.T) {
	server, _ := testServer(t, "BUDI")

	body := decode(t, get(t, server, "/api/inbox-close-claim"))

	require.Equal(t, true, body["boleh_mengajukan"])
	require.Equal(t, true, body["pelaksana_belum_ada"])
	require.Empty(t, body["alasan_tidak_boleh"])
}
