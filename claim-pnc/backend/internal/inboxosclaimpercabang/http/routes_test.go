package inboxosclaimpercabanghttp_test

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

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/repo/memory"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/platform/clock"

	oscabanghttp "claim-pnc/internal/inboxosclaimpercabang/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
//
// Ia harus ada di daftar portal contoh DAN dinyatakan siap, karena middleware membedakan
// "portal tidak dikenal" dari "portal belum siap".
const testPortal = "ASM"

// testNow tetap, supaya kolom Aging dapat diperiksa dengan angka pasti.
var testNow = time.Date(2026, time.September, 28, 5, 0, 0, 0, time.UTC)

// testServer membentuk server dengan pemanggil yang punya cabang.
func testServer(t *testing.T) http.Handler {
	t.Helper()
	// "078" adalah kode cabang RINCI CILEGON — yang dikirim HCQ. Barisnya sendiri berkode
	// "100099", dan perbedaan itu sengaja: bila kode diteruskan tanpa diterjemahkan, uji di
	// berkas ini gagal seketika alih-alih lulus dengan daftar kosong.
	return buildServer(t, oscabanghttp.Caller{
		Login:            "PETUGAS1",
		DetailBranchCode: "078",
	}, true)
}

func buildServer(t *testing.T, caller oscabanghttp.Caller, known bool) http.Handler {
	t.Helper()

	store := memory.NewSampleStore()

	service, err := usecase.NewService(usecase.Options{
		// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport,
		// bukan pemilihan basis data per entitas. Pemilihan itu diuji di usecase.
		RepoSelector: func(string) (inboxosclaimpercabang.Repo, error) { return store, nil },
		Clock:        clock.FixedAt(testNow),
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

	// Galat portal dipetakan modul portal, persis seperti rantai di cmd/claimpnc. Bila rantai
	// ini tidak ditiru, uji penolakan portal akan lulus di sini tetapi gagal di aplikasi
	// sungguhan.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode":  "galat_internal",
				"pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := oscabanghttp.NewHandler(oscabanghttp.Options{
		Service: service,
		GetCaller: func(context.Context) (oscabanghttp.Caller, bool) {
			return caller, known
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: oscabanghttp.ErrorWriter(writeError),
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		oscabanghttp.Mount(api, handler, portalDeps)
	})
	return router
}

// get mengirim permintaan BESERTA header portal.
//
// Portalnya disertakan di sini, bukan di tiap uji, supaya uji yang sengaja menghilangkannya
// terlihat jelas sebagai pengecualian.
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

func TestListReturnsTheBranchRowsAndItsTitle(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)

	branch := body["cabang"].(map[string]any)
	require.Equal(t, "100099", branch["kode"])
	require.Equal(t, "CILEGON", branch["nama"],
		"judul layar mengambil nama dari master cabang, bukan dari profil sesi")

	rows := body["data"].([]any)
	require.Len(t, rows, 3)

	require.Equal(t, testPortal, body["portal"],
		"layar memakai ini untuk memastikan jawabannya milik portal yang sedang dipilih")
	require.Equal(t, float64(inboxosclaimpercabang.AgingThreshold), body["ambang_aging"])
	require.NotEmpty(t, body["selisih_terencana"])
}

func TestListMarksTheRowsThatNeedAttention(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang")
	require.Equal(t, http.StatusOK, recorder.Code)

	rows := decode(t, recorder)["data"].([]any)

	marked := map[string]map[string]any{}
	for _, raw := range rows {
		row := raw.(map[string]any)
		marked[row["no_klaim"].(string)] = row
	}

	// Yang satu merah karena UMURNYA, yang lain karena PROGRESNYA MANDEK. Keduanya diperiksa
	// terpisah supaya satu syarat yang hilang tidak tertutup oleh syarat yang lain.
	require.True(t, marked["PNC-9001"]["perlu_perhatian"].(bool))
	require.False(t, marked["PNC-9001"]["progres_mandek"].(bool))

	require.True(t, marked["PNC-9002"]["perlu_perhatian"].(bool))
	require.True(t, marked["PNC-9002"]["progres_mandek"].(bool))
	require.Less(t, marked["PNC-9002"]["aging_hari"].(float64),
		float64(inboxosclaimpercabang.AgingThreshold))

	require.False(t, marked["PNC-9003"]["perlu_perhatian"].(bool))
}

func TestEmptyDatesAreEmptyStringsNotZeroTime(t *testing.T) {
	// Kosong berbeda artinya dari tanggal mana pun. Mengirim waktu nol akan menampilkan
	// 1 Januari tahun 1 di kolom "Tgl Update Progress Terakhir".
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang")
	rows := decode(t, recorder)["data"].([]any)

	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["no_klaim"] != "PNC-9003" {
			continue
		}
		require.Equal(t, "", row["tanggal_update_progres"])
		require.Equal(t, "", row["status_progres_1"])
		require.NotEqual(t, "", row["tanggal_registrasi"])
	}
}

func TestRequestWithoutPortalIsRejected(t *testing.T) {
	// Jatuh ke koneksi bawaan berarti menampilkan klaim satu badan hukum kepada petugas badan
	// hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
	server := testServer(t)

	request := httptest.NewRequest(http.MethodGet, "/api/inbox-os-claim-per-cabang", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestExportWithoutPortalIsRejectedToo(t *testing.T) {
	// Berkas ekspor terunduh ke perangkat pengguna, sehingga pemeriksaan portalnya tidak
	// boleh lebih longgar daripada layarnya.
	server := testServer(t)

	request := httptest.NewRequest(http.MethodGet,
		"/api/inbox-os-claim-per-cabang/ekspor", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestCallerWithoutIdentityGetsAReadableError(t *testing.T) {
	server := buildServer(t, oscabanghttp.Caller{}, false)

	recorder := get(t, server, "/api/inbox-os-claim-per-cabang")
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, "profil_pemanggil_tidak_lengkap", decode(t, recorder)["kode"])
}

func TestCallerWithoutBranchGetsTheLegacyNoticeNotAnEmptyList(t *testing.T) {
	// Inilah perilaku yang paling mudah salah dibawa. Daftar kosong dan "Anda belum punya
	// cabang" terlihat sama di layar, dan sistem lama pun membedakannya —
	// `OutstandingperCabang_PreAct` langkah 2 menampilkan pesan.
	server := buildServer(t, oscabanghttp.Caller{Login: "MITRA1"}, true)

	recorder := get(t, server, "/api/inbox-os-claim-per-cabang")
	require.Equal(t, http.StatusConflict, recorder.Code)

	body := decode(t, recorder)
	require.Equal(t, "cabang_tidak_diketahui", body["kode"])
	require.Equal(t, inboxosclaimpercabang.BranchUnknownNotice, body["pesan"],
		"pesannya harus sama persis dengan yang dibaca pengguna di Pega")
}

func TestClaimBranchCodeSentAsDetailCodeGetsTheSameNoticeNotOtherBranchRows(t *testing.T) {
	// Jembatan dari modul auth punya DUA field bernama "cabang" di profil yang sama, dan hanya
	// satu yang benar di sini. Bila yang keliru diteruskan, layar tidak menghasilkan galat —
	// ia hanya berhenti menemukan baris, dan itu terbaca sebagai "tidak ada pekerjaan".
	server := buildServer(t, oscabanghttp.Caller{
		Login:            "PETUGAS1",
		DetailBranchCode: "100099", // kode klaim, bukan kode rinci
	}, true)

	recorder := get(t, server, "/api/inbox-os-claim-per-cabang")
	require.Equal(t, http.StatusConflict, recorder.Code,
		"kode yang salah ruang harus ditolak, bukan menghasilkan daftar kosong")

	body := decode(t, recorder)
	require.Equal(t, "cabang_tidak_diketahui", body["kode"])
}

func TestPaginationIsReportedBack(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang?halaman=2&ukuran=2")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode(t, recorder)
	paging := body["paginasi"].(map[string]any)

	require.Equal(t, float64(2), paging["halaman"])
	require.Equal(t, float64(2), paging["ukuran"])
	require.Equal(t, float64(3), paging["total"])
	require.Equal(t, float64(2), paging["total_halaman"])
	require.Len(t, body["data"].([]any), 1)
}

func TestUnreadableParametersFallBackInsteadOfFailing(t *testing.T) {
	// `halaman=abc` datang dari URL yang mudah salah ketik. Menolak seluruh permintaan
	// karenanya membuat layar gagal tanpa alasan yang terbaca pengguna, sementara halaman
	// pertama adalah jawaban yang selalu masuk akal.
	recorder := get(t, testServer(t),
		"/api/inbox-os-claim-per-cabang?halaman=abc&ukuran=-5")
	require.Equal(t, http.StatusOK, recorder.Code)

	paging := decode(t, recorder)["paginasi"].(map[string]any)
	require.Equal(t, float64(1), paging["halaman"])
	require.Equal(t, float64(inboxosclaimpercabang.DefaultPageSize), paging["ukuran"])
}

func TestExportStreamsCSVWithTheBranchInItsName(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/ekspor")
	require.Equal(t, http.StatusOK, recorder.Code)

	require.Contains(t, recorder.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "SummaryOS-100099.csv")
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"),
		"berkas memuat nama tertanggung dan nilai uang; ia tidak boleh mengendap di cache")

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 4, "satu baris judul ditambah tiga baris data")
}

func TestExportHeaderAndRowsHaveTheSameWidth(t *testing.T) {
	// Satu pergeseran kolom pada berkas berisi angka uang tidak menghasilkan satu pun galat.
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/ekspor")

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)

	header := records[0]
	for index, row := range records[1:] {
		require.Lenf(t, row, len(header),
			"baris data ke-%d berbeda lebar dari judulnya", index+1)
	}
}

func TestExportKeepsTheTwentyFourTreatyColumnsInOrder(t *testing.T) {
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/ekspor")

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)
	header := records[0]

	start := -1
	for index, title := range header {
		if title == inboxosclaimpercabang.ExportTreatyColumns[0] {
			start = index
			break
		}
	}
	require.NotEqual(t, -1, start, "kolom treaty pertama tidak ada di berkas")

	for offset, title := range inboxosclaimpercabang.ExportTreatyColumns {
		require.Equalf(t, title, header[start+offset],
			"kolom treaty ke-%d bergeser", offset+1)
	}
}

func TestExportWritesMoneyWithoutThousandSeparators(t *testing.T) {
	// Berkas ini dibuka di lembar kerja. Angka berpemisah akan terbaca sebagai TEKS di sana,
	// dan penjumlahan kolomnya berhenti bekerja.
	recorder := get(t, testServer(t), "/api/inbox-os-claim-per-cabang/ekspor")

	records, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
	require.NoError(t, err)

	header := records[0]
	column := -1
	for index, title := range header {
		if title == "Reserve Claim ASM" {
			column = index
			break
		}
	}
	require.NotEqual(t, -1, column)

	for _, row := range records[1:] {
		require.NotContains(t, row[column], ",", "nilai uang memakai pemisah ribuan")
		require.NotContains(t, row[column], "Rp", "nilai uang memakai simbol mata uang")
	}
}

func TestExportRefusesBeforeWritingAnythingWhenBranchIsUnknown(t *testing.T) {
	// Setelah header terkirim, galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke
	// pengguna akan berupa berkas separuh jadi tanpa satu pun keterangan.
	server := buildServer(t, oscabanghttp.Caller{Login: "MITRA1"}, true)

	recorder := get(t, server, "/api/inbox-os-claim-per-cabang/ekspor")
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.Empty(t, recorder.Header().Get("Content-Disposition"))
}
