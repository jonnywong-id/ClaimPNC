package outstandingclaimhttp_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/outstandingclaim"
	"claim-pnc/internal/outstandingclaim/repo/memory"
	"claim-pnc/internal/outstandingclaim/usecase"

	outstandingclaimhttp "claim-pnc/internal/outstandingclaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// testPortal adalah entitas yang dipakai seluruh uji di berkas ini.
//
// Ia harus ada di daftar portal contoh DAN dinyatakan siap, karena middleware membedakan
// "portal tidak dikenal" dari "portal belum siap".
const testPortal = "ASM"

// Nomor klaim contoh — lihat repo/memory/sample.go.
const (
	claimWithDocument    = "CLMP-1001"
	claimWithoutDocument = "CLMP-1002"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	return buildServer(t, outstandingclaimhttp.Caller{Login: "ADMINTREATY1"}, true)
}

func buildServer(
	t *testing.T,
	caller outstandingclaimhttp.Caller,
	known bool,
) http.Handler {
	t.Helper()

	store := memory.NewSampleStore()

	service, err := usecase.NewService(usecase.Options{
		// Pemilih mengabaikan alias: yang diuji di berkas ini adalah lapisan transport,
		// bukan pemilihan basis data per entitas.
		RepoSelector: func(string) (outstandingclaim.Repo, error) { return store, nil },
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

	handler := outstandingclaimhttp.NewHandler(outstandingclaimhttp.Options{
		Service: service,
		GetCaller: func(context.Context) (outstandingclaimhttp.Caller, bool) {
			return caller, known
		},
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: outstandingclaimhttp.ErrorWriter(writeError),
	})

	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{testPortal} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		outstandingclaimhttp.Mount(api, handler, portalDeps)
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

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestLayoutRouteIsMounted(t *testing.T) {
	// Ia didaftarkan SEBELUM `/{no_klaim}`. Bila urutannya terbalik, "tata-letak"
	// tertangkap sebagai sebuah nomor klaim dan jawabannya 404 — kegagalan yang terbaca
	// seperti data hilang, bukan seperti rute yang bertabrakan.
	recorder := get(t, testServer(t), "/api/outstanding-claim/tata-letak")
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[struct {
		Groups []struct {
			Code   string `json:"kode"`
			Title  string `json:"judul"`
			Fields []struct {
				Key     string `json:"kunci"`
				Blocked bool   `json:"terhalang"`
			} `json:"isian"`
			Grids []string `json:"grid"`
		} `json:"kelompok"`
		Grids []struct {
			Code    string `json:"kode"`
			Blocked bool   `json:"terhalang"`
		} `json:"grid"`
		Portal string `json:"portal"`
	}](t, recorder)

	require.Len(t, body.Groups, 10)
	require.Len(t, body.Grids, 10)
	require.Equal(t, testPortal, body.Portal)
	require.Equal(t, "Treaty Information", body.Groups[0].Title)
}

func TestLayoutNeverLeaksDocumentPaths(t *testing.T) {
	// `Path` adalah jalur di dalam dokumen klaim Pega — pengetahuan penyimpanan, bukan
	// pengetahuan layar. Mengirimkannya berarti bentuk penyimpanan lama ikut menjadi
	// kontrak yang harus dipertahankan.
	recorder := get(t, testServer(t), "/api/outstanding-claim/tata-letak")

	raw := recorder.Body.String()

	// Dua NILAI jalur yang paling khas — bila DTO diam-diam membawa `Path`, keduanya pasti
	// ikut muncul.
	require.NotContains(t, raw, "QuotationData")
	require.NotContains(t, raw, "InterestList")

	// Dan KUNCI-nya, dicari sebagai kunci JSON lengkap dengan tanda kutipnya. Tanpa tanda
	// kutip, kata "jalur" pada prosa alasan terhalang ikut tertangkap — ia kalimat untuk
	// pengguna, bukan jalur yang bocor.
	require.NotContains(t, raw, `"jalur"`)
	require.NotContains(t, raw, `"path"`)
}

func TestDetailRouteIsMounted(t *testing.T) {
	recorder := get(t, testServer(t), "/api/outstanding-claim/"+claimWithDocument)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[struct {
		ClaimID    string                         `json:"no_klaim"`
		StatusWork string                         `json:"status_kerja"`
		Values     map[string]string              `json:"isian"`
		Rows       map[string][]map[string]string `json:"baris"`
		Portal     string                         `json:"portal"`
	}](t, recorder)

	require.Equal(t, claimWithDocument, body.ClaimID)
	require.Equal(t, "Pending-Teknik", body.StatusWork)
	require.Equal(t, "PT Contoh Sejahtera", body.Values["insured_name"])
	require.Len(t, body.Rows["interest_list"], 2)
	require.Equal(t, testPortal, body.Portal)
}

func TestDetailNeverCarriesBlockedFields(t *testing.T) {
	// Kedelapan isian `.TreatyInMaster.*` tidak punya sumber. Mengirimkannya kosong akan
	// membuat layar menggambarnya sebagai tanda pisah biasa — tidak dapat dibedakan dari
	// isian yang datanya memang belum diisi.
	recorder := get(t, testServer(t), "/api/outstanding-claim/"+claimWithDocument)
	body := decode[struct {
		Values map[string]string              `json:"isian"`
		Rows   map[string][]map[string]string `json:"baris"`
	}](t, recorder)

	for _, key := range []string{
		"ri_type", "ceding_name", "sob_name", "bordeaux",
		"bordereaux_note", "accounting_mode", "teritorial_scope", "asm_share",
	} {
		require.NotContainsf(t, body.Values, key,
			"isian %q terhalang dan tidak boleh ikut dikirim", key)
	}

	require.NotContains(t, body.Rows, "attachment")
}

func TestClaimWithoutDocumentStillOpens(t *testing.T) {
	// Inilah yang dijamin LEFT JOIN. Klaim yang belum punya baris di JSON_KLAIM TETAP dapat
	// dibuka — nomor dan statusnya terbaca, isinya kosong. Menjawabnya 404 akan menyatakan
	// klaimnya tidak ada, padahal ia ada dan hanya isinya yang belum tersalin.
	recorder := get(t, testServer(t), "/api/outstanding-claim/"+claimWithoutDocument)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[struct {
		ClaimID    string            `json:"no_klaim"`
		StatusWork string            `json:"status_kerja"`
		Values     map[string]string `json:"isian"`
	}](t, recorder)

	require.Equal(t, claimWithoutDocument, body.ClaimID)
	require.Equal(t, "New", body.StatusWork)
	require.Empty(t, body.Values)
}

func TestUnknownClaimIsNotFound(t *testing.T) {
	recorder := get(t, testServer(t), "/api/outstanding-claim/CLMP-9999")
	require.Equal(t, http.StatusNotFound, recorder.Code)

	body := decode[struct {
		Code    string `json:"kode"`
		Message string `json:"pesan"`
	}](t, recorder)

	require.Equal(t, "tidak_ditemukan", body.Code)

	// Pesannya MENYEBUT entitas. Pada aplikasi yang melayani empat badan hukum dengan basis
	// data terpisah, sebab paling sering dari "tidak ditemukan" bukan salah ketik melainkan
	// salah portal — dan pengguna tidak punya cara menduganya sendiri.
	require.Contains(t, body.Message, "entitas")
}

func TestRequestWithoutPortalIsRejected(t *testing.T) {
	// Jatuh ke koneksi bawaan berarti menampilkan rincian klaim satu badan hukum kepada
	// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
	request := httptest.NewRequest(
		http.MethodGet, "/api/outstanding-claim/"+claimWithDocument, nil)
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestLayoutAlsoRequiresPortal(t *testing.T) {
	// Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias portal,
	// dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal akan
	// membuat layar mengira ia sudah berada di portal yang benar.
	request := httptest.NewRequest(http.MethodGet, "/api/outstanding-claim/tata-letak", nil)
	recorder := httptest.NewRecorder()
	testServer(t).ServeHTTP(recorder, request)

	require.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestUnknownCallerIsRejected(t *testing.T) {
	// Identitas diwajibkan meski tidak menyaring apa pun: setiap pembukaan dicatat, dan
	// catatan tanpa pelaku tidak menjelaskan apa pun saat ditelusuri kemudian.
	server := buildServer(t, outstandingclaimhttp.Caller{}, false)
	recorder := get(t, server, "/api/outstanding-claim/"+claimWithDocument)

	require.Equal(t, http.StatusConflict, recorder.Code)

	body := decode[struct {
		Code string `json:"kode"`
	}](t, recorder)
	require.Equal(t, "profil_pemanggil_tidak_lengkap", body.Code)
}

func TestWriteMethodsAreNotMounted(t *testing.T) {
	// Flow Action `OutstandingClaim` di Pega MENYIMPAN kembali objek kerjanya. Modul ini
	// hanya membaca (`P-1`), dan satu rute tulis yang masuk diam-diam akan membuat dua
	// sistem menulis isi satu klaim yang sama.
	for _, method := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
	} {
		request := httptest.NewRequest(
			method, "/api/outstanding-claim/"+claimWithDocument, nil)
		request.Header.Set("X-Portal", testPortal)

		recorder := httptest.NewRecorder()
		testServer(t).ServeHTTP(recorder, request)

		require.NotEqualf(t, http.StatusOK, recorder.Code,
			"%s dilayani; modul ini hanya membaca", method)
	}
}
