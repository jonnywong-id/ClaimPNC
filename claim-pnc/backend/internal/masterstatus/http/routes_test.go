package masterstatushttp_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus/repo/memory"
	"claim-pnc/internal/masterstatus/usecase"

	masterstatushttp "claim-pnc/internal/masterstatus/http"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()

	repo := memory.NewRepo(memory.SampleList()...)
	service, err := usecase.NewService(usecase.Options{Repo: repo})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}

	handler := masterstatushttp.NewHandler(masterstatushttp.Options{
		Service:       service,
		Logger:        logger,
		WriteResponse: writeJSON,
	})

	router := chi.NewRouter()
	router.Route("/api", func(api chi.Router) {
		masterstatushttp.Mount(api, handler)
	})
	return router
}

func call(t *testing.T, server http.Handler, metode, filePath string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var content io.Reader
	if body != nil {
		mentah, err := json.Marshal(body)
		require.NoError(t, err)
		content = bytes.NewReader(mentah)
	}
	request := httptest.NewRequest(metode, filePath, content)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, request)
	return rekaman
}

func TestListReturns33StatusesAndTheirTotal(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodGet, "/api/master/status-klaim", nil)
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.ListResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))

	require.Len(t, respons.ClaimStatus, 33)
	require.Equal(t, 33, respons.Total, "total datang dari server, bukan dihitung klien")
	require.Equal(t, "1134", respons.ClaimStatus[0].Code)
	require.Equal(t, "01", respons.ClaimStatus[0].LegacyCode)
	require.Empty(t, respons.ClaimStatus[32].LegacyCode)
}

func TestGetSingleStatus(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodGet, "/api/master/status-klaim/1149", nil)
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.SingleResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "Claim Committee", respons.ClaimStatus.Label)
}

func TestGetMissingStatusAnswered404(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodGet, "/api/master/status-klaim/9999", nil)
	require.Equal(t, http.StatusNotFound, rekaman.Code)

	var issues masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
	require.Equal(t, masterstatushttp.ErrCodeNotFound, issues.Code)
}

func TestCreateAnswered201WithNewCode(t *testing.T) {
	server := testServer(t)

	rekaman := call(t, server, http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.SaveRequest{Label: "Status Percobaan"})
	require.Equal(t, http.StatusCreated, rekaman.Code, "sumber daya baru terbentuk")

	var respons masterstatushttp.SingleResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1167", respons.ClaimStatus.Code)

	list := call(t, server, http.MethodGet, "/api/master/status-klaim", nil)
	var content masterstatushttp.ListResponse
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &content))
	require.Equal(t, 34, content.Total)
}

// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan bisnis.
// Frontend menanganinya berbeda.
func TestEmptyLabelAnswered422WithPerFieldDetail(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.SaveRequest{Label: "  "})
	require.Equal(t, http.StatusUnprocessableEntity, rekaman.Code)

	var issues masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
	require.Equal(t, masterstatushttp.ErrCodeValidationFailed, issues.Code)
	require.Len(t, issues.Detail, 1, "layar perlu tahu kolom mana yang salah")
	require.Equal(t, "label", issues.Detail[0].Field)
	require.NotEmpty(t, issues.Detail[0].Message)
}

// 409, bukan 422: isian penggunanya sah tetapi bentrok dengan keadaan penyimpanan.
func TestDuplicateLabelAnswered409(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodPost, "/api/master/status-klaim",
		masterstatushttp.SaveRequest{Label: "Paid"})
	require.Equal(t, http.StatusConflict, rekaman.Code)

	var issues masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
	require.Equal(t, masterstatushttp.ErrCodeLabelTaken, issues.Code)
	require.Empty(t, issues.Detail, "konflik bukan galat per-field")
}

func TestUpdateAnswered200(t *testing.T) {
	server := testServer(t)

	rekaman := call(t, server, http.MethodPut, "/api/master/status-klaim/1163",
		masterstatushttp.SaveRequest{Label: "Sudah Dibayar"})
	require.Equal(t, http.StatusOK, rekaman.Code)

	var respons masterstatushttp.SingleResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1163", respons.ClaimStatus.Code)
	require.Equal(t, "Sudah Dibayar", respons.ClaimStatus.Label)
}

// PUT mengirim seluruh isi yang boleh diubah, sehingga mengirimnya dua kali menghasilkan
// keadaan akhir yang sama.
func TestUpdateIsIdempotent(t *testing.T) {
	server := testServer(t)
	request := masterstatushttp.SaveRequest{Label: "Sudah Dibayar"}

	pertama := call(t, server, http.MethodPut, "/api/master/status-klaim/1163", request)
	require.Equal(t, http.StatusOK, pertama.Code)

	kedua := call(t, server, http.MethodPut, "/api/master/status-klaim/1163", request)
	require.Equal(t, http.StatusOK, kedua.Code)
	require.JSONEq(t, pertama.Body.String(), kedua.Body.String())
}

func TestUpdateMissingStatusAnswered404(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodPut, "/api/master/status-klaim/9999",
		masterstatushttp.SaveRequest{Label: "Apa Saja"})
	require.Equal(t, http.StatusNotFound, rekaman.Code)
}

func TestMalformedRequestBodyAnswered400(t *testing.T) {
	server := testServer(t)

	request := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim",
		bytes.NewReader([]byte("{bukan json")))
	request.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, request)

	require.Equal(t, http.StatusBadRequest, rekaman.Code, "400 berarti klien salah membentuk permintaan")

	var issues masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
	require.Equal(t, masterstatushttp.ErrCodeBadRequest, issues.Code)
}

// Kode tidak pernah datang dari klien. Mengirimnya di badan permintaan tidak boleh
// memindahkan status ke kode lain.
func TestCodeInRequestBodyIgnored(t *testing.T) {
	server := testServer(t)

	mentah := []byte(`{"kode":"7777","label":"Status Percobaan"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim", bytes.NewReader(mentah))
	request.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, request)
	require.Equal(t, http.StatusCreated, rekaman.Code)

	var respons masterstatushttp.SingleResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &respons))
	require.Equal(t, "1167", respons.ClaimStatus.Code, "kode tetap dibuat sistem, bukan diambil dari klien")
}

// Tidak ada rute hapus. Sama seperti layar Pega yang tidak punya tombol hapus.
func TestDeleteRouteDoesNotExist(t *testing.T) {
	rekaman := call(t, testServer(t), http.MethodDelete, "/api/master/status-klaim/1163", nil)
	require.Equal(t, http.StatusMethodNotAllowed, rekaman.Code)
}

// Badan permintaan yang sangat besar ditolak, bukan dibaca seluruhnya ke memori.
func TestOversizedRequestBodyRejected(t *testing.T) {
	server := testServer(t)

	large := make([]byte, 64<<10)
	for i := range large {
		large[i] = 'a'
	}
	mentah := append(append([]byte(`{"label":"`), large...), []byte(`"}`)...)

	request := httptest.NewRequest(http.MethodPost, "/api/master/status-klaim", bytes.NewReader(mentah))
	request.Header.Set("Content-Type", "application/json")

	rekaman := httptest.NewRecorder()
	server.ServeHTTP(rekaman, request)
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
}
