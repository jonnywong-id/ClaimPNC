package archivedokumenklaimhttp_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/gateway"
	"claim-pnc/internal/archivedokumenklaim/repo/memory"
	"claim-pnc/internal/archivedokumenklaim/usecase"
	"claim-pnc/internal/portal"

	archivehttp "claim-pnc/internal/archivedokumenklaim/http"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// failingRepo membungkus repo memori dan menggagalkan operasi tertentu.
type failingRepo struct {
	archivedokumenklaim.Repo
	searchErr     error
	searchCalls   int
	failFromCall  int
	fillingErr    error
	claimsErr     error
	typesErr      error
	pendingErr    error
	pendingResult *archivedokumenklaim.ArchivePage
	searchPage    *archivedokumenklaim.ArchivePage
}

func (f *failingRepo) Search(
	ctx context.Context, c archivedokumenklaim.Criteria, p archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	f.searchCalls++
	if f.searchErr != nil && f.searchCalls >= f.failFromCall {
		return archivedokumenklaim.ArchivePage{}, f.searchErr
	}
	if f.searchPage != nil {
		return *f.searchPage, nil
	}
	return f.Repo.Search(ctx, c, p)
}

func (f *failingRepo) FillingCodes(
	ctx context.Context, keyword string,
) ([]archivedokumenklaim.FillingCodeOption, error) {
	if f.fillingErr != nil {
		return nil, f.fillingErr
	}
	return f.Repo.FillingCodes(ctx, keyword)
}

func (f *failingRepo) SearchClaims(
	ctx context.Context, c archivedokumenklaim.ClaimCriteria,
) ([]archivedokumenklaim.ClaimCandidate, error) {
	if f.claimsErr != nil {
		return nil, f.claimsErr
	}
	return f.Repo.SearchClaims(ctx, c)
}

func (f *failingRepo) DocumentTypes(
	ctx context.Context,
) ([]archivedokumenklaim.DocumentTypeOption, error) {
	if f.typesErr != nil {
		return nil, f.typesErr
	}
	return f.Repo.DocumentTypes(ctx)
}

func (f *failingRepo) PendingBranch(
	ctx context.Context, s archivedokumenklaim.BranchScope, p archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	if f.pendingErr != nil {
		return archivedokumenklaim.ArchivePage{}, f.pendingErr
	}
	if f.pendingResult != nil {
		return *f.pendingResult, nil
	}
	return f.Repo.PendingBranch(ctx, s, p)
}

type harness struct {
	router   http.Handler
	repo     *failingRepo
	recorder *gateway.Recorder
	caller   *archivehttp.Caller
	logs     *bytes.Buffer
	fallback bool
}

type options struct {
	noCaller bool
	fallback bool
}

func newHarness(t *testing.T, o options) *harness {
	t.Helper()

	h := &harness{
		repo:     &failingRepo{Repo: memory.NewSampleRepo()},
		recorder: gateway.NewRecorder(),
		caller:   &archivehttp.Caller{Login: "USERARSIP1", Position: "PICTEKNIK", BranchCode: "01"},
		logs:     &bytes.Buffer{},
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (archivedokumenklaim.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return h.repo, nil
		},
		Gateway: h.recorder,
		Clock:   fixedClock{at: time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewJSONHandler(h.logs, nil))
	writeJSON := func(w http.ResponseWriter, _ *http.Request, status int, body any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}

	var fallback archivehttp.ErrorWriter
	if o.fallback {
		fallback = func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusTeapot, map[string]string{"kode": "cadangan"})
		}
	}

	var reader archivehttp.CallerReader
	if !o.noCaller {
		reader = func(context.Context) (archivehttp.Caller, bool) {
			return *h.caller, h.caller.Login != "#absen"
		}
	}

	handler := archivehttp.NewHandler(archivehttp.Options{
		Service:             service,
		GetCaller:           reader,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: fallback,
	})

	portalError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "x"})
		}, writeJSON)

	router := chi.NewRouter()
	archivehttp.Mount(router, handler, portalhttp.ActivePortalDeps{
		Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
		ReadyAliases: func() []string { return []string{"ASM", "ASI"} },
		Logger:       logger,
		WriteError:   portalError,
	})
	h.router = router
	return h
}

func (h *harness) do(t *testing.T, method, target, body, alias string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	if alias != "" {
		request.Header.Set(portalhttp.HeaderPortal, alias)
	}
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, request)
	return response
}

func decode(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body), response.Body.String())
	return body
}

func TestOpenMenyerahkanDropdown(t *testing.T) {
	h := newHarness(t, options{})
	h.caller.Position = "nonmbu"

	response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)

	body := decode(t, response)
	require.Equal(t, "ASM", body["portal"])
	require.Len(t, body["tipe_input"], 3)
	require.Len(t, body["tipe_dokumen"], 3)
	kinds := body["jenis_dokumen"].([]any)
	require.Len(t, kinds, 5)
	require.Equal(t, "0001", kinds[0].(map[string]any)["kode_tipe_dokumen"])
	scope := body["cakupan_cabang"].(map[string]any)
	require.ElementsMatch(t, []any{"002", "005"}, scope["lini_disembunyikan"])
}

func TestOpenGalatRepoTidakDikenaliMenjadi500(t *testing.T) {
	h := newHarness(t, options{})
	h.repo.typesErr = errors.New("rusak")

	response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "ASM")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, archivehttp.CodeInternalError, decode(t, response)["kode"])
	require.Contains(t, h.logs.String(), "rusak", "rincian galat hanya masuk log")
}

func TestGalatTidakDikenaliDiteruskanKeCadangan(t *testing.T) {
	h := newHarness(t, options{fallback: true})
	h.repo.typesErr = errors.New("rusak")

	response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "ASM")
	require.Equal(t, http.StatusTeapot, response.Code)
	require.Equal(t, "cadangan", decode(t, response)["kode"])
}

// Tanpa header portal, middleware menolak sebelum handler dipanggil.
func TestTanpaPortalDitolak(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "")
	require.Equal(t, http.StatusBadRequest, response.Code)
}

// Handler yang dipanggil tanpa portal di context menolak, bukan memakai portal utama.
func TestHandlerTanpaPortalDiContextDitolak(t *testing.T) {
	handler := archivehttp.NewHandler(archivehttp.Options{
		Logger: slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
		WriteJSON: func(w http.ResponseWriter, _ *http.Request, status int, _ any) {
			w.WriteHeader(status)
		},
		FallbackErrorWriter: func(w http.ResponseWriter, _ *http.Request, err error) {
			require.ErrorIs(t, err, portal.ErrNotStated)
			w.WriteHeader(http.StatusBadRequest)
		},
	})

	response := httptest.NewRecorder()
	handler.Open(response, httptest.NewRequest(http.MethodGet, "/arsip-dokumen/buka", nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestPemanggilTidakTerbacaDitolak(t *testing.T) {
	t.Run("tanpa pembaca", func(t *testing.T) {
		h := newHarness(t, options{noCaller: true})
		response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "ASM")
		require.Equal(t, http.StatusConflict, response.Code)
		require.Equal(t, archivehttp.CodeCallerUnknown, decode(t, response)["kode"])
	})

	t.Run("tidak ada", func(t *testing.T) {
		h := newHarness(t, options{})
		h.caller.Login = "#absen"
		response := h.do(t, http.MethodGet, "/arsip-dokumen", "", "ASM")
		require.Equal(t, http.StatusConflict, response.Code)
	})

	t.Run("login kosong", func(t *testing.T) {
		h := newHarness(t, options{})
		h.caller.Login = "  "
		response := h.do(t, http.MethodGet, "/arsip-dokumen/kirim-cabang", "", "ASM")
		require.Equal(t, http.StatusConflict, response.Code)
	})
}

func TestPortalBelumSiapDiteruskanKePemetaCadangan(t *testing.T) {
	h := newHarness(t, options{fallback: true})

	response := h.do(t, http.MethodGet, "/arsip-dokumen/buka", "", "ASI")
	require.Equal(t, http.StatusTeapot, response.Code,
		"galat portal dari selector bukan milik modul ini")
}

func TestSearchKataKunciMengembalikanHalaman(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen?mode=kata_kunci&kata_kunci=box-a-01&halaman=1&ukuran=1", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)

	body := decode(t, response)
	files := body["berkas"].([]any)
	require.Len(t, files, 1)
	first := files[0].(map[string]any)
	require.Equal(t, float64(2), first["id"])
	require.Equal(t, true, first["sudah_dikirim"])
	require.Equal(t, "2024-04-12", first["tanggal_kirim_dokumen"])

	page := body["halaman"].(map[string]any)
	require.Equal(t, float64(2), page["total"])
	require.Equal(t, float64(2), page["total_halaman"])
	require.Equal(t, float64(1), page["ukuran"])
}

func TestSearchRentangTanggal(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen?mode=tanggal_input&tanggal_dari=2024-03-01&tanggal_sampai=2024-04-30&halaman=abc",
		"", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	body := decode(t, response)
	require.Len(t, body["berkas"], 2)
	require.Equal(t, float64(1), body["halaman"].(map[string]any)["halaman"])
}

func TestSearchTanggalTidakTerbaca(t *testing.T) {
	h := newHarness(t, options{})

	for _, query := range []string{"tanggal_dari=kemarin", "tanggal_sampai=2024-13-40"} {
		response := h.do(t, http.MethodGet, "/arsip-dokumen?mode=tanggal_input&"+query, "", "ASM")
		require.Equal(t, http.StatusUnprocessableEntity, response.Code)
		body := decode(t, response)
		require.Equal(t, archivehttp.CodeValidationFail, body["kode"])
		require.Len(t, body["detail"], 1)
	}
}

func TestSearchValidasiGagal(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet, "/arsip-dokumen?mode=tanggal_input", "", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Len(t, decode(t, response)["detail"], 2)
}

func TestSearchClaims(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen/klaim?tipe=no_polis&nilai=POL-2024-000006", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	claims := decode(t, response)["klaim"].([]any)
	require.Len(t, claims, 2)
	first := claims[0].(map[string]any)
	require.Equal(t, "PNC-100006", first["nomor_klaim"])
	require.Equal(t, "2024-08-30", first["tanggal_close"])
	require.Equal(t, "006", first["group_panel"])

	invalid := h.do(t, http.MethodGet, "/arsip-dokumen/klaim?tipe=lain&nilai=x", "", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, invalid.Code)

	h.repo.claimsErr = errors.New("rusak")
	broken := h.do(t, http.MethodGet, "/arsip-dokumen/klaim?tipe=no_klaim&nilai=x", "", "ASM")
	require.Equal(t, http.StatusInternalServerError, broken.Code)
}

func TestFillingCodes(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet, "/arsip-dokumen/kode-filling?kata_kunci=2026", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	body := decode(t, response)
	require.Equal(t, true, body["dari_pemakaian"])
	codes := body["kode"].([]any)
	require.Len(t, codes, 1)
	require.Equal(t, float64(1), codes[0].(map[string]any)["jumlah_pemakaian"])

	h.repo.fillingErr = errors.New("rusak")
	broken := h.do(t, http.MethodGet, "/arsip-dokumen/kode-filling", "", "ASM")
	require.Equal(t, http.StatusInternalServerError, broken.Code)
}

const validSave = `{
	"nomor_klaim": "PNC-100009",
	"nomor_polis": "POL-9",
	"nama_tertanggung": "NAMA",
	"tanggal_kejadian": "2024-05-01",
	"pic_teknis": "PIC",
	"group_panel": "006",
	"tanggal_terima_dokumen": "2024-05-20",
	"jumlah_lembar": 12,
	"kode_tipe_dokumen": "0001",
	"kode_jenis_dokumen": "000101",
	"nama_box": "BOX-A-01",
	"kode_filling": "FIL-2024-001"
}`

func TestSaveBaruMenjawab201DanMengirim(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodPost, "/arsip-dokumen", validSave, "ASM")
	require.Equal(t, http.StatusCreated, response.Code)
	body := decode(t, response)
	require.Equal(t, float64(6), body["id"])
	require.Equal(t, true, body["baru"])
	require.Equal(t, true, body["terkirim"])
	require.Equal(t, "Berkas arsip tersimpan. Berkas juga dikirim ke sistem Arsip.", body["pesan"])
	require.Equal(t, "200", body["kode_layanan"])

	file, _, err := h.repo.FindByID(context.Background(), 6)
	require.NoError(t, err)
	require.Equal(t, "USERARSIP1", file.InputUser)
}

func TestSaveUbahGagalKirimTetap200(t *testing.T) {
	h := newHarness(t, options{})
	h.recorder.Err = archivedokumenklaim.ErrServiceFailed

	body := strings.Replace(validSave, `"nomor_klaim"`, `"id": 3, "nomor_klaim"`, 1)
	response := h.do(t, http.MethodPost, "/arsip-dokumen", body, "ASM")
	require.Equal(t, http.StatusOK, response.Code)

	decoded := decode(t, response)
	require.Equal(t, false, decoded["baru"])
	require.Equal(t, false, decoded["terkirim"])
	require.NotEmpty(t, decoded["galat_kirim"])
	require.True(t, strings.HasPrefix(decoded["pesan"].(string), "Berkas arsip diperbarui."))
	require.Contains(t, decoded["pesan"], "GAGAL")
}

func TestSaveBadanCacat(t *testing.T) {
	h := newHarness(t, options{})

	for _, body := range []string{`{bukan json`, `{"kolom_asing": 1}`} {
		response := h.do(t, http.MethodPost, "/arsip-dokumen", body, "ASM")
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Equal(t, archivehttp.CodeBadRequest, decode(t, response)["kode"])
	}
}

func TestSaveTanggalTidakTerbaca(t *testing.T) {
	h := newHarness(t, options{})

	loss := strings.Replace(validSave, `"2024-05-01"`, `"01/05/2024"`, 1)
	response := h.do(t, http.MethodPost, "/arsip-dokumen", loss, "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	detail := decode(t, response)["detail"].([]any)
	require.Equal(t, archivedokumenklaim.FieldClaimNumber, detail[0].(map[string]any)["field"])

	received := strings.Replace(validSave, `"2024-05-20"`, `"xx"`, 1)
	response = h.do(t, http.MethodPost, "/arsip-dokumen", received, "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	detail = decode(t, response)["detail"].([]any)
	require.Equal(t, archivedokumenklaim.FieldReceivedDate, detail[0].(map[string]any)["field"])
}

func TestSaveBerkasTidakAda404(t *testing.T) {
	h := newHarness(t, options{})

	body := strings.Replace(validSave, `"nomor_klaim"`, `"id": 999, "nomor_klaim"`, 1)
	response := h.do(t, http.MethodPost, "/arsip-dokumen", body, "ASM")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, archivehttp.CodeNotFound, decode(t, response)["kode"])
}

func TestPendingMengirimCakupan(t *testing.T) {
	h := newHarness(t, options{})
	h.caller.Position = "PA"

	response := h.do(t, http.MethodGet, "/arsip-dokumen/kirim-cabang?ukuran=2", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	body := decode(t, response)
	require.Equal(t, []any{"002"}, body["cakupan_cabang"].(map[string]any)["lini_disembunyikan"])
	require.Len(t, body["berkas"], 2)

	h.repo.pendingErr = errors.New("rusak")
	broken := h.do(t, http.MethodGet, "/arsip-dokumen/kirim-cabang", "", "ASM")
	require.Equal(t, http.StatusInternalServerError, broken.Code)
}

func TestSendMenjawabTerkirimDanDuaKaliDitolak(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodPost, "/arsip-dokumen/1/kirim-cabang", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	body := decode(t, response)
	require.Equal(t, float64(1), body["id"])
	require.Equal(t, "200", body["kode_layanan"])

	again := h.do(t, http.MethodPost, "/arsip-dokumen/1/kirim-cabang", "", "ASM")
	require.Equal(t, http.StatusConflict, again.Code)
	require.Equal(t, archivehttp.CodeAlreadySent, decode(t, again)["kode"])
}

func TestSendNomorTidakTerbaca(t *testing.T) {
	h := newHarness(t, options{})

	for _, id := range []string{"abc", "0", "-3"} {
		response := h.do(t, http.MethodPost, "/arsip-dokumen/"+id+"/kirim-cabang", "", "ASM")
		require.Equal(t, http.StatusBadRequest, response.Code, id)
	}
}

func TestSendGalatLayanan(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{archivedokumenklaim.ErrServiceAddress, http.StatusServiceUnavailable,
			archivehttp.CodeServiceAddress},
		{fmt.Errorf("%w: mati", archivedokumenklaim.ErrServiceFailed), http.StatusBadGateway,
			archivehttp.CodeServiceFailed},
	}

	for _, c := range cases {
		h := newHarness(t, options{})
		h.recorder.Err = c.err

		response := h.do(t, http.MethodPost, "/arsip-dokumen/1/kirim-cabang", "", "ASM")
		require.Equal(t, c.status, response.Code)
		require.Equal(t, c.code, decode(t, response)["kode"])
	}
}

func readCSV(t *testing.T, response *httptest.ResponseRecorder) [][]string {
	t.Helper()
	records, err := csv.NewReader(bytes.NewReader(response.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportMenulisCSV(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen/ekspor?mode=kata_kunci&kata_kunci=PNCN.26.0001", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "text/csv; charset=utf-8", response.Header().Get("Content-Type"))
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Header().Get("Content-Disposition"), "archive-dokumen-klaim-")

	records := readCSV(t, response)
	require.Len(t, records, 2)
	require.Equal(t, "NO KLAIM", records[0][0])
	row := records[1]
	require.Equal(t, "PNCN.26.0001", row[0])
	require.Equal(t, "2026-01-09", row[3])
	require.Equal(t, "40", row[7])
	require.Equal(t, "9999", row[8], "kode dipakai bila nama dokumen kosong")
	require.Equal(t, "999901", row[9])
	require.Equal(t, "", row[13], "tanggal kirim kosong menjadi sel kosong")
}

func TestExportMembacaBerpotongan(t *testing.T) {
	h := newHarness(t, options{})

	// 105 berkas dengan TGLINPUT sama, sehingga ekspor menembak dua potong.
	files := make([]archivedokumenklaim.ArchiveFile, 0, 105)
	input := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 105; i++ {
		files = append(files, archivedokumenklaim.ArchiveFile{
			ID: int64(i), ClaimNumber: fmt.Sprintf("K-%d", i), InputDate: &input,
			DocumentTypeName: "Nama",
		})
	}
	h.repo.Repo = memory.NewRepo(memory.Options{Files: files})

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen/ekspor?mode=tanggal_input&tanggal_dari=2025-01-01&tanggal_sampai=2025-01-03",
		"", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	records := readCSV(t, response)
	require.Len(t, records, 106)
	require.Equal(t, 2, h.repo.searchCalls)
}

// Galat pada potongan kedua hanya dicatat; isinya yang sudah tertulis tetap terkirim.
func TestExportGalatPotonganKeduaDicatat(t *testing.T) {
	h := newHarness(t, options{})

	files := make([]archivedokumenklaim.ArchiveFile, 0, 101)
	input := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 101; i++ {
		files = append(files, archivedokumenklaim.ArchiveFile{ID: int64(i), InputDate: &input})
	}
	h.repo.Repo = memory.NewRepo(memory.Options{Files: files})
	h.repo.searchErr = errors.New("putus")
	h.repo.failFromCall = 2

	response := h.do(t, http.MethodGet,
		"/arsip-dokumen/ekspor?mode=tanggal_input&tanggal_dari=2025-01-01&tanggal_sampai=2025-01-03",
		"", "ASM")
	require.Equal(t, http.StatusOK, response.Code)
	require.Len(t, readCSV(t, response), 101)
	require.Contains(t, h.logs.String(), "ekspor berkas arsip terputus")
	require.Contains(t, h.logs.String(), `"portal":"ASM"`)
}

func TestExportGalatSebelumHeaderDijawabJSON(t *testing.T) {
	h := newHarness(t, options{})

	response := h.do(t, http.MethodGet, "/arsip-dokumen/ekspor?mode=lain", "", "ASM")
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)

	for _, query := range []string{"tanggal_dari=x", "tanggal_sampai=x"} {
		response = h.do(t, http.MethodGet, "/arsip-dokumen/ekspor?"+query, "", "ASM")
		require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	}

	unknown := newHarness(t, options{noCaller: true})
	response = unknown.do(t, http.MethodGet, "/arsip-dokumen/ekspor", "", "ASM")
	require.Equal(t, http.StatusConflict, response.Code)
}

func TestMapErrorMembedakanPesan(t *testing.T) {
	h := newHarness(t, options{})
	h.recorder.Err = errors.New("tanpa sentinel")

	// Galat gateway yang tidak dikenali menjadi 500 tanpa membocorkan rinciannya.
	response := h.do(t, http.MethodPost, "/arsip-dokumen/1/kirim-cabang", "", "ASM")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.NotContains(t, response.Body.String(), "tanpa sentinel")
}

// failingWriter gagal menulis setelah sejumlah byte, meniru koneksi yang terputus.
type failingWriter struct {
	header http.Header
	status int
	limit  int
	buf    bytes.Buffer
}

func (f *failingWriter) Header() http.Header { return f.header }
func (f *failingWriter) WriteHeader(status int) {
	f.status = status
}
func (f *failingWriter) Write(p []byte) (int, error) {
	if f.buf.Len()+len(p) > f.limit {
		return 0, errors.New("koneksi putus")
	}
	return f.buf.Write(p)
}

// Kegagalan menulis badan CSV hanya dapat dicatat ke log.
func TestExportGagalMenulisDicatat(t *testing.T) {
	h := newHarness(t, options{})

	request := httptest.NewRequest(http.MethodGet,
		"/arsip-dokumen/ekspor?mode=kata_kunci&kata_kunci=BOX-A-01", nil)
	request.Header.Set(portalhttp.HeaderPortal, "ASM")
	writer := &failingWriter{header: http.Header{}, limit: 10}

	h.router.ServeHTTP(writer, request)
	require.Contains(t, h.logs.String(), "ekspor berkas arsip terputus")
}

// Ekspor yang menyentuh batas diberi baris tanda terpotong, bukan dipotong diam-diam.
func TestExportTerpotongDiBatas(t *testing.T) {
	h := newHarness(t, options{})

	chunk := make([]archivedokumenklaim.ArchiveFile, 100)
	for i := range chunk {
		chunk[i] = archivedokumenklaim.ArchiveFile{ID: int64(i + 1), ClaimNumber: "K"}
	}
	h.repo.searchPage = &archivedokumenklaim.ArchivePage{Files: chunk, Total: 60000}

	response := h.do(t, http.MethodGet, "/arsip-dokumen/ekspor?kata_kunci=K", "", "ASM")
	require.Equal(t, http.StatusOK, response.Code)

	records := readCSV(t, response)
	require.Len(t, records, 1+50000+1)
	require.Equal(t,
		"-- terpotong pada 50000 baris dari 60000; persempit pencariannya lalu ekspor lagi --",
		records[len(records)-1][0])
}
