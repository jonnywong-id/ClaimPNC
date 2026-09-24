package inboxanalystdoctorhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/repo/memory"
	"claim-pnc/internal/inboxanalystdoctor/usecase"

	analystdoctorhttp "claim-pnc/internal/inboxanalystdoctor/http"
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

// fixedClock memenuhi seam waktu dengan satu waktu tetap.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// testServer membentuk server lengkap dengan pemanggil yang dikenali.
func testServer(t *testing.T, login string) http.Handler {
	t.Helper()
	return buildServer(t, login, true)
}

// testServerTanpaIdentitas membentuk server yang pemanggilnya TIDAK terbaca.
func testServerTanpaIdentitas(t *testing.T) http.Handler {
	t.Helper()
	return buildServer(t, "", false)
}

func buildServer(t *testing.T, login string, callerKnown bool) http.Handler {
	t.Helper()

	store := memory.NewSampleStore()

	// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport, bukan
	// pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxanalystdoctor.Repo, error) { return store, nil },
	})
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

	handler := analystdoctorhttp.NewHandler(analystdoctorhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (analystdoctorhttp.Caller, bool) {
			if !callerKnown {
				return analystdoctorhttp.Caller{}, false
			}
			return analystdoctorhttp.Caller{Login: login}, true
		},
		Clock:               fixedClock{at: testNow},
		Location:            wib,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: analystdoctorhttp.ErrorWriter(writeError),
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		analystdoctorhttp.Mount(api, handler, portalDeps)
	})
	return router
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

func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

// rows mengambil senarai baris dari badan jawaban.
func rows(t *testing.T, recorder *httptest.ResponseRecorder) []any {
	t.Helper()
	body := decode(t, recorder)
	data, ok := body["data"].([]any)
	require.True(t, ok, "badan jawaban tidak memuat senarai `data`")
	return data
}

func TestAntreanDikembalikanBesertaTotal(t *testing.T) {
	recorder := get(t, testServer(t, memory.SampleOperator), "/api/inbox-analyst-doctor")

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, testPortal, body["portal"])
	require.Equal(t, float64(5), body["total"])
	require.Len(t, body["data"], 5)
}

// TestBarisMembawaKedelapanIsianLayar menjaga kontrak tidak menyusut diam-diam.
//
// Kolom yang hilang dari jawaban tidak menghasilkan galat apa pun — layar hanya berhenti
// menggambarnya, dan tidak ada yang menyadarinya sampai seseorang mencari isian itu.
func TestBarisMembawaKedelapanIsianLayar(t *testing.T) {
	data := rows(t, get(t, testServer(t, memory.SampleOperator), "/api/inbox-analyst-doctor"))

	first, ok := data[0].(map[string]any)
	require.True(t, ok)

	for _, field := range []string{
		"klaim_id", "nomor_case", "nomor_polis", "nama_tertanggung", "nama_cabang",
		"nama_admin", "komentar_pic_teknis", "pic_teknis", "tanggal_pendaftaran",
		"lama_hari", "status_proses", "operator_penerima",
	} {
		require.Containsf(t, first, field, "isian %s hilang dari jawaban", field)
	}
}

// TestTanggalDanUmurDihitungDalamWIB mengunci konversi zona waktu di satu tempat.
//
// Tugas pertama terdaftar 18 September pukul 09.00 UTC = 16.00 WIB, dan `testNow` adalah
// 23 September pukul 03.00 UTC = 10.00 WIB. Umurnya 5 hari.
func TestTanggalDanUmurDihitungDalamWIB(t *testing.T) {
	data := rows(t, get(t, testServer(t, memory.SampleOperator), "/api/inbox-analyst-doctor"))

	first := data[0].(map[string]any)
	require.Equal(t, "2026-09-18", first["tanggal_pendaftaran"])
	require.Equal(t, float64(5), first["lama_hari"])
}

// TestAntreanTerbatasPadaPemanggil adalah uji kewenangan, bukan uji penyaring.
//
// Petugas lain punya satu tugas, dan hanya itu yang boleh ia lihat.
func TestAntreanTerbatasPadaPemanggil(t *testing.T) {
	data := rows(t, get(t, testServer(t, memory.SampleOtherOperator), "/api/inbox-analyst-doctor"))

	require.Len(t, data, 1)
	require.Equal(t, "PNCN.26.0222", data[0].(map[string]any)["nomor_case"])
}

// TestIdentitasTidakTerbacaDijawab409BukanDaftarKosong adalah uji terpenting di berkas ini.
//
// Menjawab 200 dengan daftar kosong akan membuat petugas mengira ia tidak punya pekerjaan.
// Itu jawaban yang tidak pernah dilaporkan siapa pun sebagai kerusakan — dan karena itu
// justru paling berbahaya.
func TestIdentitasTidakTerbacaDijawab409BukanDaftarKosong(t *testing.T) {
	recorder := get(t, testServerTanpaIdentitas(t), "/api/inbox-analyst-doctor")

	require.Equal(t, http.StatusConflict, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "profil_pemanggil_tidak_lengkap", body["kode"])
	require.NotContains(t, body, "data", "jawaban galat tidak boleh membawa daftar")
}

// TestPermintaanTanpaPortalDitolak menjaga `R-20`.
//
// Ia TIDAK boleh dilayani portal utama sebagai cadangan: jatuh ke koneksi bawaan berarti
// menampilkan klaim satu badan hukum kepada petugas badan hukum lain — dan di layar ini
// barisnya menyangkut data medis.
func TestPermintaanTanpaPortalDitolak(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/inbox-analyst-doctor", nil)
	recorder := httptest.NewRecorder()
	testServer(t, memory.SampleOperator).ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

// TestKeteranganLayarJugaMenuntutPortal menjaga jawaban tetap jujur.
//
// Isinya memang sama di seluruh entitas, tetapi jawabannya memuat alias portal. Mengembalikan
// alias portal utama untuk permintaan yang tidak menyebut portal akan membuat layar mengira
// ia sudah berada di portal yang benar.
func TestKeteranganLayarJugaMenuntutPortal(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet, "/api/inbox-analyst-doctor/keterangan", nil)
	recorder := httptest.NewRecorder()
	testServer(t, memory.SampleOperator).ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestKeteranganLayarMembawaKolomDanCatatan(t *testing.T) {
	recorder := get(t, testServer(t, memory.SampleOperator),
		"/api/inbox-analyst-doctor/keterangan")

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Len(t, body["kolom"], 8)
	require.NotEmpty(t, body["selisih_terencana"])
	require.NotEmpty(t, body["keterbatasan"])
	require.Equal(t, float64(inboxanalystdoctor.DefaultLimit), body["ukuran_halaman"])
}

func TestPencarianDiteruskanKePenyimpanan(t *testing.T) {
	recorder := get(t, testServer(t, memory.SampleOperator),
		"/api/inbox-analyst-doctor?cari=0298")

	data := rows(t, recorder)
	require.Len(t, data, 1)
	require.Equal(t, "PNCN.26.0298", data[0].(map[string]any)["nomor_case"])
	require.Equal(t, "0298", decode(t, recorder)["cari"])
}

// TestPaginasiYangDIPAKAIDikembalikan menjaga bilah halaman menghitung dari angka yang benar.
func TestPaginasiYangDIPAKAIDikembalikan(t *testing.T) {
	recorder := get(t, testServer(t, memory.SampleOperator),
		"/api/inbox-analyst-doctor?batas=5000&lewati=2")

	body := decode(t, recorder)
	require.Equal(t, float64(inboxanalystdoctor.MaxLimit), body["batas"],
		"batas berlebihan dipangkas, dan angka yang dipakai dikembalikan")
	require.Equal(t, float64(2), body["lewati"])
}

// TestParameterTidakTerbacaTidakMenjatuhkanLayar.
//
// `batas=abc` menghasilkan halaman pertama, bukan galat. Menolak seluruh permintaan akan
// membuat layar gagal tanpa alasan yang terbaca pengguna.
func TestParameterTidakTerbacaTidakMenjatuhkanLayar(t *testing.T) {
	recorder := get(t, testServer(t, memory.SampleOperator),
		"/api/inbox-analyst-doctor?batas=abc&lewati=-9")

	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, float64(inboxanalystdoctor.DefaultLimit), body["batas"])
	require.Equal(t, float64(0), body["lewati"])
}

// TestAntreanKosongMenjawabSenaraiKosongBukanNull.
//
// `null` memaksa setiap tempat di layar memeriksanya sebelum menelusuri. Satu tempat yang
// lupa akan menjatuhkan layar, dan justru pada keadaan yang paling sering terjadi.
func TestAntreanKosongMenjawabSenaraiKosongBukanNull(t *testing.T) {
	recorder := get(t, testServer(t, "PETUGASTANPATUGAS"), "/api/inbox-analyst-doctor")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"data":[]`)
}
