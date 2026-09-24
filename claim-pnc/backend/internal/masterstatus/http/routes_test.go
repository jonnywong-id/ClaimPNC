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

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memory"
	"claim-pnc/internal/masterstatus/usecase"
	"claim-pnc/internal/portal"

	masterstatushttp "claim-pnc/internal/masterstatus/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// portalASM adalah entitas yang dipakai hampir seluruh uji di berkas ini. Ia disebut
// eksplisit pada setiap permintaan — persis seperti yang dituntut aplikasi sungguhan.
const portalASM = "ASM"

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
		RepoSelector: func(alias string) (masterstatus.Repo, error) {
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

	handler := masterstatushttp.NewHandler(masterstatushttp.Options{
		Service:             service,
		Logger:              logger,
		WriteResponse:       writeJSON,
		FallbackErrorWriter: masterstatushttp.ErrorWriter(writeError),
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
		masterstatushttp.Mount(api, handler, portalDeps)
	})
	return &entities{handler: router, asm: asm, asi: asi}
}

// call menembak dengan portal ASM. Uji yang menguji portal lain memakai callPortal.
func call(t *testing.T, server *entities, metode, filePath string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return callPortal(t, server, metode, filePath, portalASM, body)
}

// callPortal menembak dengan portal tertentu. Alias kosong berarti header tidak dikirim.
func callPortal(t *testing.T, server *entities, metode, filePath, portalAlias string, body any) *httptest.ResponseRecorder {
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
	if portalAlias != "" {
		request.Header.Set(portalhttp.HeaderPortal, portalAlias)
	}

	rekaman := httptest.NewRecorder()
	server.handler.ServeHTTP(rekaman, request)
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
	request.Header.Set(portalhttp.HeaderPortal, portalASM)

	rekaman := httptest.NewRecorder()
	server.handler.ServeHTTP(rekaman, request)

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
	request.Header.Set(portalhttp.HeaderPortal, portalASM)

	rekaman := httptest.NewRecorder()
	server.handler.ServeHTTP(rekaman, request)
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
	// Portal WAJIB disebut, kalau tidak permintaannya ditolak pemeriksaan portal dan uji
	// ini lulus karena alasan yang salah — 400 yang sama, sebab yang berbeda.
	request.Header.Set(portalhttp.HeaderPortal, portalASM)

	rekaman := httptest.NewRecorder()
	server.handler.ServeHTTP(rekaman, request)
	require.Equal(t, http.StatusBadRequest, rekaman.Code)

	// Kodenya diperiksa, bukan hanya statusnya: itulah yang membedakan "badan permintaan
	// ditolak" dari "portal tidak disebut".
	var issues masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
	require.Equal(t, masterstatushttp.ErrCodeBadRequest, issues.Code)
}

// ── Portal entitas ──────────────────────────────────────────────────────────────

// PERMINTAAN TANPA PORTAL DITOLAK, bukan dilayani portal utama (TKT-F6-002, R-20).
//
// Ini uji terpenting yang lahir dari penyelarasan 2026-09-19. Sebelumnya modul ini selalu
// membaca basis data portal utama; bila permintaan tanpa portal jatuh ke sana lagi, status
// klaim satu badan hukum akan terbaca di badan hukum lain — dan layarnya tampak normal,
// karena isinya masuk akal. Yang salah hanya milik siapa data itu.
func TestWithoutPortalRejected(t *testing.T) {
	server := testServer(t)

	for _, perkara := range []struct {
		nama, metode, jalur string
		badan               any
	}{
		{"membaca daftar", http.MethodGet, "/api/master/status-klaim", nil},
		{"membaca satu baris", http.MethodGet, "/api/master/status-klaim/1163", nil},
		{"menambah", http.MethodPost, "/api/master/status-klaim", masterstatushttp.SaveRequest{Label: "Uji"}},
		{"mengubah", http.MethodPut, "/api/master/status-klaim/1163", masterstatushttp.SaveRequest{Label: "Uji"}},
	} {
		t.Run(perkara.nama, func(t *testing.T) {
			rekaman := callPortal(t, server, perkara.metode, perkara.jalur, "", perkara.badan)
			require.Equal(t, http.StatusBadRequest, rekaman.Code)

			var issues masterstatushttp.ErrorResponse
			require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &issues))
			require.Equal(t, portalhttp.CodeNotStated, issues.Code)
		})
	}

	// Dan tidak ada satu baris pun yang berubah di entitas mana pun.
	isiASM, err := server.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, isiASM, 33)
	isiASI, err := server.asi.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, isiASI)
}

// "Tidak ada" dibedakan dari "belum tersedia": keduanya menuntut tindak lanjut berbeda.
func TestUnknownAndNotReadyPortalAreDistinguished(t *testing.T) {
	server := testServer(t)

	rekaman := callPortal(t, server, http.MethodGet, "/api/master/status-klaim", "TIDAKADA", nil)
	require.Equal(t, http.StatusBadRequest, rekaman.Code)
	var takDikenal masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &takDikenal))
	require.Equal(t, portalhttp.CodeUnknown, takDikenal.Code)

	rekaman = callPortal(t, server, http.MethodGet, "/api/master/status-klaim", "SMAS", nil)
	require.Equal(t, http.StatusServiceUnavailable, rekaman.Code)
	var belumSiap masterstatushttp.ErrorResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &belumSiap))
	require.Equal(t, portalhttp.CodeNotReady, belumSiap.Code)
}

// Setiap entitas menjawab dengan isinya sendiri, dan menyebut namanya.
func TestEachPortalAnswersWithItsOwnRows(t *testing.T) {
	server := testServer(t)

	rekaman := callPortal(t, server, http.MethodGet, "/api/master/status-klaim", "ASM", nil)
	var isiASM masterstatushttp.ListResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &isiASM))
	require.Equal(t, 33, isiASM.Total)
	require.Equal(t, "ASM", isiASM.Portal)

	rekaman = callPortal(t, server, http.MethodGet, "/api/master/status-klaim", "ASI", nil)
	require.Equal(t, http.StatusOK, rekaman.Code)
	var isiASI masterstatushttp.ListResponse
	require.NoError(t, json.Unmarshal(rekaman.Body.Bytes(), &isiASI))
	require.Equal(t, 0, isiASI.Total, "ASI kosong; isinya TIDAK diwarisi dari ASM")
	require.Equal(t, "ASI", isiASI.Portal)
	require.NotNil(t, isiASI.ClaimStatus, "kosong dikirim sebagai senarai kosong, bukan null")
}

// Menambah di satu entitas tidak menyentuh entitas lain.
func TestCreateTouchesOnlyRequestedPortal(t *testing.T) {
	server := testServer(t)

	rekaman := callPortal(t, server, http.MethodPost, "/api/master/status-klaim", "ASI",
		masterstatushttp.SaveRequest{Label: "Status Khusus ASI"})
	require.Equal(t, http.StatusCreated, rekaman.Code)

	isiASI, err := server.asi.List(t.Context())
	require.NoError(t, err)
	require.Len(t, isiASI, 1)

	isiASM, err := server.asm.List(t.Context())
	require.NoError(t, err)
	require.Len(t, isiASM, 33, "entitas lain tidak ikut bertambah")
}
