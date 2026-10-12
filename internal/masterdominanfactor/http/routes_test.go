package masterdominanfactorhttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/masterdominanfactor/repo/memory"
	"claim-pnc/internal/masterdominanfactor/usecase"
	"claim-pnc/internal/portal"

	masterdominanfactorhttp "claim-pnc/internal/masterdominanfactor/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// portalASM adalah entitas yang dipakai hampir seluruh uji di berkas ini. Ia disebut
// eksplisit pada setiap permintaan — persis seperti yang dituntut aplikasi sungguhan.
const portalASM = "ASM"

const basePath = "/api/master/dominan-factor"

// entities memegang penyimpanan tiap entitas supaya uji dapat memeriksa bahwa yang
// tersentuh memang entitas yang diminta, bukan entitas lain.
type entities struct {
	handler http.Handler
	asm     *memory.Repo
	asi     *memory.Repo
}

func testServer(t *testing.T) *entities {
	t.Helper()

	asm := memory.NewRepo(memory.SampleList()...)
	// ASI sengaja KOSONG; ia yang membuktikan pemisahan antarentitas.
	asi := memory.NewRepo()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterdominanfactor.Repo, error) {
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}
	// Dirantai sama seperti cmd/claimpnc: galat portal dipetakan modul portal. Tanpa
	// rantai ini, uji penolakan portal lulus di sini tetapi gagal di aplikasi sungguhan.
	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{
				"kode": "galat_internal", "pesan": err.Error(),
			})
		},
		writeJSON,
	)

	handler := masterdominanfactorhttp.NewHandler(masterdominanfactorhttp.Options{
		Service:             service,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterdominanfactorhttp.ErrorWriter(writeError),
	})

	// Hanya ASM dan ASI yang koneksinya "hidup"; SMAS ada di daftar tetapi belum siap.
	portalDeps := portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   writeError,
	}

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		masterdominanfactorhttp.Mount(api, handler, portalDeps)
	})
	return &entities{handler: router, asm: asm, asi: asi}
}

// call menembak dengan portal ASM. Uji yang menguji portal lain memakai callPortal.
func call(t *testing.T, server *entities, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return callPortal(t, server, method, path, portalASM, body)
}

// callPortal menembak dengan portal tertentu. Alias kosong berarti header tidak dikirim.
func callPortal(t *testing.T, server *entities, method, path, portalAlias string, body any) *httptest.ResponseRecorder {
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
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	record := httptest.NewRecorder()
	server.handler.ServeHTTP(record, request)
	return record
}

func decode[T any](t *testing.T, record *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	require.NoError(t, json.Unmarshal(record.Body.Bytes(), &body))
	return body
}

func TestListReturnsFactorsAndTheirTotal(t *testing.T) {
	record := call(t, testServer(t), http.MethodGet, basePath, nil)
	require.Equal(t, http.StatusOK, record.Code)

	body := decode[masterdominanfactorhttp.ListResponse](t, record)
	require.Len(t, body.DominantFactor, 10)
	require.Equal(t, 10, body.Total, "total datang dari server, bukan dihitung klien")
	require.Equal(t, portalASM, body.Portal, "entitas yang menjawab disebut eksplisit")

	require.Equal(t, "1", body.DominantFactor[0].ID)
	require.Equal(t, "10", body.DominantFactor[9].ID, "urutan numerik, bukan teks")
}

// Permintaan tanpa portal DITOLAK, tidak pernah dialihkan ke portal utama sebagai
// cadangan. Jalan pintas itu adalah `R-20` yang sesungguhnya.
func TestRequestWithoutPortalRejected(t *testing.T) {
	record := callPortal(t, testServer(t), http.MethodGet, basePath, "", nil)
	require.Equal(t, http.StatusBadRequest, record.Code)
}

func TestRequestWithNotReadyPortalRejected(t *testing.T) {
	record := callPortal(t, testServer(t), http.MethodGet, basePath, "SMAS", nil)
	require.NotEqual(t, http.StatusOK, record.Code,
		"portal yang koneksinya belum hidup tidak boleh dilayani portal lain")
}

// Entitas yang satu tidak melihat isi entitas lain.
func TestEntitiesAreSeparated(t *testing.T) {
	server := testServer(t)

	record := callPortal(t, server, http.MethodGet, basePath, "ASI", nil)
	require.Equal(t, http.StatusOK, record.Code)

	body := decode[masterdominanfactorhttp.ListResponse](t, record)
	require.Equal(t, 0, body.Total, "ASI kosong; isinya tidak boleh datang dari ASM")
	require.Equal(t, "ASI", body.Portal)
}

func TestGetReturnsOneFactor(t *testing.T) {
	record := call(t, testServer(t), http.MethodGet, basePath+"/3", nil)
	require.Equal(t, http.StatusOK, record.Code)

	body := decode[masterdominanfactorhttp.SingleResponse](t, record)
	require.Equal(t, "3", body.DominantFactor.ID)
	require.NotEmpty(t, body.DominantFactor.Nama)
}

func TestGetUnknownIDReturns404(t *testing.T) {
	record := call(t, testServer(t), http.MethodGet, basePath+"/999", nil)
	require.Equal(t, http.StatusNotFound, record.Code)

	body := decode[masterdominanfactorhttp.ErrorResponse](t, record)
	require.Equal(t, masterdominanfactorhttp.ErrCodeNotFound, body.Code)
}

// 201, bukan 200: sumber daya baru terbentuk dan nomornya baru diketahui di sini.
func TestCreateReturns201AndIssuedID(t *testing.T) {
	server := testServer(t)
	record := call(t, server, http.MethodPost, basePath,
		masterdominanfactorhttp.SaveRequest{Nama: "Faktor baru"})

	require.Equal(t, http.StatusCreated, record.Code)
	body := decode[masterdominanfactorhttp.SingleResponse](t, record)
	require.Equal(t, "11", body.DominantFactor.ID, "melanjutkan dari 10")
	require.Equal(t, "Faktor baru", body.DominantFactor.Nama)
}

// Nama kosong dan nama ganda DITERIMA — keputusan Work Owner 2026-09-20, meniru layar
// Pega apa adanya (`P-5`).
func TestCreateAcceptsEmptyAndDuplicateNames(t *testing.T) {
	server := testServer(t)

	empty := call(t, server, http.MethodPost, basePath,
		masterdominanfactorhttp.SaveRequest{Nama: ""})
	require.Equal(t, http.StatusCreated, empty.Code)

	first := call(t, server, http.MethodPost, basePath,
		masterdominanfactorhttp.SaveRequest{Nama: "Kembar"})
	require.Equal(t, http.StatusCreated, first.Code)

	second := call(t, server, http.MethodPost, basePath,
		masterdominanfactorhttp.SaveRequest{Nama: "Kembar"})
	require.Equal(t, http.StatusCreated, second.Code,
		"tidak ada constraint keunikan di sistem lama, dan tidak ditambahkan")
}

// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
func TestCreateRejectsTooLongNameWith422(t *testing.T) {
	record := call(t, testServer(t), http.MethodPost, basePath,
		masterdominanfactorhttp.SaveRequest{
			Nama: strings.Repeat("a", masterdominanfactor.MaxNameLength+1),
		})

	require.Equal(t, http.StatusUnprocessableEntity, record.Code)
	body := decode[masterdominanfactorhttp.ErrorResponse](t, record)
	require.Equal(t, masterdominanfactorhttp.ErrCodeValidationFailed, body.Code)
	require.Len(t, body.Detail, 1)
	require.Equal(t, masterdominanfactor.FieldName, body.Detail[0].Field,
		"field dikirim supaya layar dapat menandai kolom yang salah")
}

func TestUpdateChangesNameOnly(t *testing.T) {
	server := testServer(t)
	record := call(t, server, http.MethodPut, basePath+"/3",
		masterdominanfactorhttp.SaveRequest{Nama: "Nama diperbarui"})

	require.Equal(t, http.StatusOK, record.Code)
	body := decode[masterdominanfactorhttp.SingleResponse](t, record)
	require.Equal(t, "3", body.DominantFactor.ID, "ID diambil dari jalur URL dan tidak berubah")
	require.Equal(t, "Nama diperbarui", body.DominantFactor.Nama)
}

func TestUpdateUnknownIDReturns404(t *testing.T) {
	record := call(t, testServer(t), http.MethodPut, basePath+"/999",
		masterdominanfactorhttp.SaveRequest{Nama: "apa pun"})
	require.Equal(t, http.StatusNotFound, record.Code)
}

// PUT idempoten: mengirim permintaan yang sama dua kali menghasilkan keadaan akhir yang
// sama, dan TIDAK menambah baris baru.
func TestUpdateIsIdempotent(t *testing.T) {
	server := testServer(t)
	for i := 0; i < 2; i++ {
		record := call(t, server, http.MethodPut, basePath+"/5",
			masterdominanfactorhttp.SaveRequest{Nama: "Tetap sama"})
		require.Equal(t, http.StatusOK, record.Code)
	}

	list := decode[masterdominanfactorhttp.ListResponse](t,
		call(t, server, http.MethodGet, basePath, nil))
	require.Equal(t, 10, list.Total, "PUT tidak boleh menambah baris")
}

func TestUnreadableBodyReturns400(t *testing.T) {
	server := testServer(t)
	request := httptest.NewRequest(http.MethodPost, basePath, strings.NewReader("{bukan json"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(portalhttp.HeaderPortal, portalASM)

	record := httptest.NewRecorder()
	server.handler.ServeHTTP(record, request)

	require.Equal(t, http.StatusBadRequest, record.Code)
	body := decode[masterdominanfactorhttp.ErrorResponse](t, record)
	require.Equal(t, masterdominanfactorhttp.ErrCodeBadRequest, body.Code)
	require.NotContains(t, record.Body.String(), "bukan json",
		"masukan mentah tidak dipantulkan kembali ke peramban")
}

// Tidak ada rute hapus. Layar Pega tidak punya tombolnya, procedure lama tidak punya
// cabangnya, dan `ADR-0012` melarangnya.
func TestNoDeleteRoute(t *testing.T) {
	record := call(t, testServer(t), http.MethodDelete, basePath+"/3", nil)
	require.Equal(t, http.StatusMethodNotAllowed, record.Code)
}
