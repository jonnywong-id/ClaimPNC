package pelaporanklaimhttp_test

import (
	"bytes"
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

	"claim-pnc/internal/pelaporanklaim/repo/memory"
	"claim-pnc/internal/pelaporanklaim/usecase"
	"claim-pnc/internal/platform/waktu"

	reporthttp "claim-pnc/internal/pelaporanklaim/http"
)

var testNow = time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)

func testServer(t *testing.T, withSamples bool) http.Handler {
	t.Helper()

	var repo *memory.Repo
	if withSamples {
		repo = memory.NewRepo(memory.SampleReports()...)
	} else {
		repo = memory.NewRepo()
	}

	service, err := usecase.NewService(usecase.Options{
		Repo:  repo,
		Clock: waktu.JamTetapPada(testNow),
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

	handler := reporthttp.NewHandler(reporthttp.Options{
		Service: service,
		GetCaller: func(context.Context) (reporthttp.Caller, bool) {
			return reporthttp.Caller{Login: "petugas.penerimaan", BranchCode: "100081"}, true
		},
		Logger:    logger,
		WriteJSON: writeJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		reporthttp.Mount(api, handler)
	})
	return router
}

func call(t *testing.T, server http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var content io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		content = bytes.NewReader(raw)
	}
	request := httptest.NewRequest(method, path, content)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func decode[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var content T
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &content))
	return content
}

func validForm() map[string]any {
	return map[string]any{
		"nama_pelapor":     "Bagas Prasetya",
		"email_pengirim":   "bagas@contoh.example",
		"nomor_polis":      "CONTOH-PL-000117",
		"nama_tertanggung": "PT Harapan Sentosa",
		"tanggal_kejadian": "2026-09-09",
		"kronologi":        "Api muncul dari panel listrik lantai satu.",
	}
}

// Daftar mengembalikan halaman, jumlah, dan ringkasan kelima tahap sekaligus.
//
// Ketiganya datang dari satu permintaan supaya lencana tab konsisten dengan isi tabel.
// Dua permintaan terpisah dapat tiba di antara dua perubahan, dan pengguna melihat lencana
// "3" di atas tabel berisi 4 baris.
func TestListReturnsPageTotalAndSummary(t *testing.T) {
	recorder := call(t, testServer(t, true), http.MethodGet, "/api/pelaporan-klaim", nil)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[reporthttp.ListResponse](t, recorder)
	require.Len(t, body.Reports, 6)
	require.Equal(t, 6, body.Total)
	require.Equal(t, 50, body.Limit, "batas baku mengikuti pyPageSize Report Definition lama")
	require.Len(t, body.Summary, 5, "kelima tahap selalu dikirim, termasuk yang kosong")

	for _, s := range body.Summary {
		require.NotEmpty(t, s.Stage)
		require.NotEmpty(t, s.Label, "layar menampilkan label ini apa adanya")
	}
}

// Urutan tab tetap, mengikuti perjalanan laporan.
//
// Map Go tidak punya urutan; membiarkannya membuat tab berpindah tempat setiap kali
// halaman dimuat.
func TestTabOrderIsFixed(t *testing.T) {
	body := decode[reporthttp.ListResponse](t,
		call(t, testServer(t, true), http.MethodGet, "/api/pelaporan-klaim", nil))

	want := []string{
		"BELUM_TRANSFER", "BELUM_REGISTRASI", "SUDAH_REGISTRASI",
		"SUDAH_AKSEPTASI", "DITOLAK",
	}
	for i, stage := range want {
		require.Equal(t, stage, body.Summary[i].Stage)
	}
}

// Penyaring tahap benar-benar menyaring.
func TestStageFilterFilters(t *testing.T) {
	server := testServer(t, true)

	body := decode[reporthttp.ListResponse](t,
		call(t, server, http.MethodGet, "/api/pelaporan-klaim?tahap=DITOLAK", nil))

	require.Equal(t, 1, body.Total)
	require.Len(t, body.Reports, 1)
	require.Equal(t, "DITOLAK", body.Reports[0].Stage)
	require.Equal(t, "Ditolak", body.Reports[0].StageLabel)
}

// Tahap karangan dijawab 422, bukan diabaikan diam-diam.
//
// Penyaring yang diabaikan mengembalikan SELURUH baris sementara pemanggilnya mengira
// sudah tersaring.
func TestUnknownStageAnswered422(t *testing.T) {
	recorder := call(t, testServer(t, true), http.MethodGet,
		"/api/pelaporan-klaim?tahap=KARANGAN", nil)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	body := decode[reporthttp.ErrorResponse](t, recorder)
	require.Equal(t, reporthttp.CodeValidationFail, body.Code)
}

// Mencatat laporan menjawab 201 beserta nomor yang dibuat sistem.
func TestRecordAnswers201WithSystemNumber(t *testing.T) {
	recorder := call(t, testServer(t, false), http.MethodPost, "/api/pelaporan-klaim", validForm())
	require.Equal(t, http.StatusCreated, recorder.Code)

	body := decode[reporthttp.SingleResponse](t, recorder)
	require.NotEmpty(t, body.Report.Number)
	require.Equal(t, "BELUM_TRANSFER", body.Report.Stage)
	require.Equal(t, "petugas.penerimaan", body.Report.CreatedBy, "pencatat datang dari sesi")
	require.Equal(t, "100081", body.Report.BranchCode, "cabang datang dari profil")
	require.True(t, body.Report.CanTransfer)
	require.True(t, body.Report.CanEdit)
}

// Tanggal kalender dikirim sebagai YYYY-MM-DD, tanpa jam dan tanpa zona.
//
// Membawa jam pada tanggal kalender hanya mengundang pergeseran satu hari saat zonanya
// ditafsirkan berbeda di ujung yang lain — persis kelas cacat yang `R-12` catat.
func TestCalendarDateSentWithoutTime(t *testing.T) {
	body := decode[reporthttp.SingleResponse](t,
		call(t, testServer(t, false), http.MethodPost, "/api/pelaporan-klaim", validForm()))

	require.Equal(t, "2026-09-09", body.Report.LossDate)
}

// Bentuk tanggal yang salah ditandai pada KOLOMNYA, bukan disimpan diam-diam sebagai
// kosong.
//
// Tanpa pemeriksaan ini, tanggal yang salah ketik hilang tanpa jejak dan pengguna baru
// menyadarinya saat laporan tidak muncul pada penyaringan periode.
func TestMalformedDateFlaggedOnItsOwnField(t *testing.T) {
	form := validForm()
	form["tanggal_kejadian"] = "09/09/2026"
	form["tanggal_terima_dokumen"] = "kemarin"

	recorder := call(t, testServer(t, false), http.MethodPost, "/api/pelaporan-klaim", form)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode[reporthttp.ErrorResponse](t, recorder)
	require.Equal(t, reporthttp.CodeValidationFail, body.Code)
	require.Len(t, body.Details, 2, "kedua tanggal dilaporkan sekaligus")

	fields := map[string]bool{}
	for _, d := range body.Details {
		fields[d.Field] = true
	}
	require.True(t, fields["tanggal_kejadian"])
	require.True(t, fields["tanggal_terima_dokumen"])
}

// Galat validasi membawa SELURUH pelanggaran beserta nama fieldnya.
//
// Mengembalikan satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk
// menemukan kesalahan berikutnya — pada form berisi 20 kolom, itu menyiksa.
func TestValidationCarriesEveryViolation(t *testing.T) {
	form := map[string]any{
		"nama_pelapor":   "",
		"email_pengirim": "bukan-surel",
		"nilai_estimasi": "seratus juta",
	}

	recorder := call(t, testServer(t, false), http.MethodPost, "/api/pelaporan-klaim", form)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	body := decode[reporthttp.ErrorResponse](t, recorder)
	require.Len(t, body.Details, 3)
	for _, d := range body.Details {
		require.NotEmpty(t, d.Field, "setiap pelanggaran harus menunjuk kolomnya")
		require.NotEmpty(t, d.Message)
	}
}

// Badan permintaan yang tidak dapat dibaca dijawab 400, dan isinya TIDAK dipantulkan.
//
// Form ini memuat nama tertanggung, nomor polis, dan alamat surel; memantulkan masukan
// mentah ke peramban akan membuat data nasabah ikut tercatat di log proxy.
func TestMalformedBodyAnswered400WithoutEchoingIt(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/pelaporan-klaim",
		bytes.NewReader([]byte(`{"nama_pelapor": "PT Rahasia`)))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	testServer(t, false).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "Rahasia",
		"isi permintaan tidak boleh dipantulkan ke peramban")
}

// Transfer memindahkan tahap, dan transfer kedua dijawab 409.
func TestTransferMovesStageAndRejectsSecondTransfer(t *testing.T) {
	server := testServer(t, false)

	created := decode[reporthttp.SingleResponse](t,
		call(t, server, http.MethodPost, "/api/pelaporan-klaim", validForm()))
	path := "/api/pelaporan-klaim/" + created.Report.Number + "/transfer"

	recorder := call(t, server, http.MethodPost, path, nil)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := decode[reporthttp.SingleResponse](t, recorder)
	require.True(t, body.Report.Transferred)
	require.Equal(t, "BELUM_REGISTRASI", body.Report.Stage)
	require.False(t, body.Report.CanTransfer, "tombolnya harus mati sesudah ini")

	again := call(t, server, http.MethodPost, path, nil)
	require.Equal(t, http.StatusConflict, again.Code)
	require.Equal(t, reporthttp.CodeAlreadyMoved,
		decode[reporthttp.ErrorResponse](t, again).Code)
}

// Penautan klaim memindahkan tahap, dan laporan yang sudah diregistrasi tidak dapat
// diubah lagi.
func TestLinkClaimClosesEditing(t *testing.T) {
	server := testServer(t, false)

	created := decode[reporthttp.SingleResponse](t,
		call(t, server, http.MethodPost, "/api/pelaporan-klaim", validForm()))
	base := "/api/pelaporan-klaim/" + created.Report.Number

	linked := decode[reporthttp.SingleResponse](t, call(t, server, http.MethodPost,
		base+"/klaim", map[string]any{"nomor_klaim": "PNCN.26.0148"}))
	require.Equal(t, "PNCN.26.0148", linked.Report.ClaimNumber)
	require.Equal(t, "SUDAH_REGISTRASI", linked.Report.Stage)
	require.False(t, linked.Report.CanEdit)

	update := call(t, server, http.MethodPut, base, validForm())
	require.Equal(t, http.StatusConflict, update.Code)
	require.Equal(t, reporthttp.CodeAlreadyRegd,
		decode[reporthttp.ErrorResponse](t, update).Code)
}

// Mengubah dua kali dengan isi yang sama menghasilkan jawaban yang sama — PUT idempoten.
func TestUpdateIsIdempotent(t *testing.T) {
	server := testServer(t, false)

	created := decode[reporthttp.SingleResponse](t,
		call(t, server, http.MethodPost, "/api/pelaporan-klaim", validForm()))
	path := "/api/pelaporan-klaim/" + created.Report.Number

	form := validForm()
	form["nama_pelapor"] = "Nama Baru"

	first := decode[reporthttp.SingleResponse](t,
		call(t, server, http.MethodPut, path, form))
	second := decode[reporthttp.SingleResponse](t,
		call(t, server, http.MethodPut, path, form))

	require.Equal(t, first.Report, second.Report)
	require.Equal(t, "Nama Baru", second.Report.ReporterName)
}

// Laporan yang tidak ada dijawab 404 pada keempat jalurnya.
func TestMissingReportAnswered404(t *testing.T) {
	server := testServer(t, false)
	base := "/api/pelaporan-klaim/LPK.00.9999"

	cases := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPut, base, validForm()},
		{http.MethodPost, base + "/transfer", nil},
		{http.MethodPost, base + "/klaim", map[string]any{"nomor_klaim": "PNCN.26.0148"}},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			recorder := call(t, server, c.method, c.path, c.body)
			require.Equal(t, http.StatusNotFound, recorder.Code)
			require.Equal(t, reporthttp.CodeNotFound,
				decode[reporthttp.ErrorResponse](t, recorder).Code)
		})
	}
}

// DELETE tidak ada, dan ketiadaannya dikunci uji.
//
// `ADR-0012` menetapkan penghapusan lunak menyeluruh, dan laporan yang sudah tertaut klaim
// dirujuk klaimnya lewat `ClaimData.RCV_ID`. Uji ini membuat penambahan DELETE kelak
// menjadi keputusan sadar, bukan kelalaian yang lolos review.
func TestNoDeleteRoute(t *testing.T) {
	server := testServer(t, true)

	list := call(t, server, http.MethodDelete, "/api/pelaporan-klaim", nil)
	require.Equal(t, http.StatusMethodNotAllowed, list.Code)

	one := call(t, server, http.MethodDelete, "/api/pelaporan-klaim/LPK.00.0001", nil)
	require.Equal(t, http.StatusMethodNotAllowed, one.Code)
}

// Paginasi yang salah ketik jatuh ke halaman pertama, bukan menolak permintaan.
func TestMistypedPaginationFallsBackToDefaults(t *testing.T) {
	body := decode[reporthttp.ListResponse](t, call(t, testServer(t, true),
		http.MethodGet, "/api/pelaporan-klaim?batas=banyak&lewati=-9", nil))

	require.Equal(t, 50, body.Limit)
	require.Zero(t, body.Offset)
}

// Batas raksasa dipotong ke batas maksimum.
//
// Batas yang tidak dibatasi adalah cara termurah membuat satu permintaan menarik seluruh
// tabel — dan pada tabel yang bertambah ribuan baris per bulan, itu berarti mengirim
// seluruh riwayat laporan ke peramban.
func TestHugeLimitIsCapped(t *testing.T) {
	body := decode[reporthttp.ListResponse](t, call(t, testServer(t, true),
		http.MethodGet, "/api/pelaporan-klaim?batas=100000", nil))

	require.Equal(t, 200, body.Limit)
}
