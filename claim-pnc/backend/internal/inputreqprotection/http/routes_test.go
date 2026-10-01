package inputreqprotectionhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
	inputreqprotectionhttp "claim-pnc/internal/inputreqprotection/http"
	"claim-pnc/internal/inputreqprotection/repo/memory"
	"claim-pnc/internal/inputreqprotection/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	portalmemory "claim-pnc/internal/portal/repo/memory"
)

// Uji transport: rute dipasang lewat Mount di balik middleware portal, dengan service
// sungguhan ber-repo memori, supaya yang teruji adalah kontrak HTTP-nya.

const base = "/api/input-req-protection"

var wib = time.FixedZone("WIB", 7*60*60)

var fixedAt = time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)

func writeJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

type env struct {
	router  http.Handler
	repo    *memory.Repo
	logs    *bytes.Buffer
	caller  *inputreqprotectionhttp.Caller
	service inputreqprotectionhttp.Service
}

type options struct {
	service  inputreqprotectionhttp.Service
	noCaller bool
	fallback inputreqprotectionhttp.ErrorWriter
	location *time.Location
}

func newEnv(t *testing.T, o options) *env {
	t.Helper()

	repo := memory.NewRepo()
	service := o.service
	if service == nil {
		s, err := usecase.NewService(usecase.Options{
			Protections: func(alias string) (inputreqprotection.Stores, error) {
				return inputreqprotection.Stores{
					Protections: repo,
					Types:       memory.NewTypeRepoWithSamples(),
					Claims:      memory.NewClaimRepoWithSamples(),
				}, nil
			},
			Now:      func() time.Time { return fixedAt },
			Location: wib,
		})
		require.NoError(t, err)
		service = s
	}

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	caller := &inputreqprotectionhttp.Caller{Login: "ADMINCONTOH"}

	var getCaller inputreqprotectionhttp.GetCaller
	if !o.noCaller {
		getCaller = func(context.Context) (inputreqprotectionhttp.Caller, bool) {
			return *caller, caller.Login != ""
		}
	}

	location := o.location
	if location == nil {
		location = wib
	}
	h := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service:             service,
		GetCaller:           getCaller,
		Logger:              logger,
		WriteJSON:           writeJSON,
		FallbackErrorWriter: o.fallback,
		Location:            location,
	})

	writeError := portalhttp.WithPortalError(
		func(w http.ResponseWriter, r *http.Request, err error) {
			writeJSON(w, r, http.StatusInternalServerError, map[string]string{"kode": "lain"})
		}, writeJSON)

	r := chi.NewRouter()
	r.Route("/api", func(api chi.Router) {
		inputreqprotectionhttp.Mount(api, h, portalhttp.ActivePortalDeps{
			Repo:         portalmemory.NewRepo(portalmemory.SampleList()...),
			ReadyAliases: func() []string { return []string{"ASM"} },
			Logger:       logger,
			WriteError:   writeError,
		})
	})

	return &env{router: r, repo: repo, logs: logs, caller: caller, service: service}
}

func (e *env) do(t *testing.T, method, path, body string, withPortal bool) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withPortal {
		req.Header.Set(portalhttp.HeaderPortal, "ASM")
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	var decoded map[string]any
	if rec.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &decoded), rec.Body.String())
	}
	return rec, decoded
}

func TestListReturnsRowsWithDerivedFlags(t *testing.T) {
	e := newEnv(t, options{})
	e.repo.Add(
		inputreqprotection.Protection{Number: "OPCN.26.0001", PolicyNumber: "POL-1", Type: "2",
			TypeName: "Premi Belum Lunas", InputDate: time.Date(2026, 9, 22, 18, 0, 0, 0, time.UTC),
			Note: "n", CreatedBy: "U", CreatedAt: fixedAt},
		inputreqprotection.Protection{Number: "OPCN.26.0002", ClaimNumber: "PNCN.26.0007", Type: "1",
			CreatedAt: fixedAt.Add(-time.Hour)},
	)

	rec, body := e.do(t, http.MethodGet, base+"/?cari=+&batas=10&lewati=0", "", true)
	require.Equal(t, http.StatusOK, rec.Code)
	require.EqualValues(t, 2, body["total"])

	rows := body["proteksi"].([]any)
	first := rows[0].(map[string]any)
	require.Equal(t, "OPCN.26.0001", first["nomor_proteksi"])
	require.Equal(t, "Premi Belum Lunas", first["nama_tipe_proteksi"])
	// 18.00 UTC adalah 01.00 WIB hari berikutnya.
	require.Equal(t, "2026-09-23", first["tanggal_proteksi"])
	require.Equal(t, true, first["dapat_disunting"])
	require.Equal(t, true, first["premi"])

	second := rows[1].(map[string]any)
	require.Equal(t, false, second["dapat_disunting"])
	require.Equal(t, "", second["tanggal_proteksi"], "tanggal nol dikirim sebagai teks kosong")
}

func TestListRejectsBadPagingParameters(t *testing.T) {
	e := newEnv(t, options{})
	for _, c := range []struct {
		query string
		pesan string
	}{
		{"?batas=abc", "Parameter batas harus berupa angka lebih besar dari nol."},
		{"?batas=0", "Parameter batas harus berupa angka lebih besar dari nol."},
		{"?batas=101", "Parameter batas melebihi 100."},
		{"?lewati=-1", "Parameter lewati harus berupa angka nol atau lebih."},
		{"?lewati=x", "Parameter lewati harus berupa angka nol atau lebih."},
	} {
		rec, body := e.do(t, http.MethodGet, base+"/"+c.query, "", true)
		require.Equal(t, http.StatusBadRequest, rec.Code, c.query)
		require.Equal(t, inputreqprotectionhttp.CodeBadRequest, body["kode"])
		require.Equal(t, c.pesan, body["pesan"])
	}
}

func TestRequestWithoutPortalIsRejectedByMiddleware(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodGet, base+"/", "", false)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, portalhttp.CodeNotStated, body["kode"])
}

func TestListTypesReturnsMaster(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodGet, base+"/tipe", "", true)
	require.Equal(t, http.StatusOK, rec.Code)
	tipe := body["tipe"].([]any)
	require.Len(t, tipe, 9)
	require.Equal(t, map[string]any{"kode": "7", "nama": "Perubahan DOL"}, tipe[6])
}

func TestFindClaimReturnsDerivedFields(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodGet, base+"/klaim/PNCN.26.0007", "", true)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, map[string]any{
		"nomor_klaim": "PNCN.26.0007", "nomor_polis": "99.001.2026.00000001",
		"nama_tertanggung": "TERTANGGUNG CONTOH SATU", "dol": "2026-08-17",
		"penyebab_kerugian": "Kebakaran", "nama_objek": "OBJEK CONTOH SATU",
		"nama_cabang": "CABANG CONTOH",
	}, body)
}

func TestFindClaimMissingNumberIsBadRequest(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodGet, base+"/klaim/%20", "", true)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "Nomor klaim wajib disebutkan.", body["pesan"])
}

func TestCreateThenGetRoundTrip(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodPost, base+"/",
		`{"nomor_klaim":"PNCN.26.0007","tipe_proteksi":"7","keterangan":"ubah DOL",
		  "detail_perubahan":{"dol_baru":"2026-08-20"}}`, true)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Equal(t, "OPCN.26.0001", body["nomor_proteksi"])
	require.Equal(t, "ADMINCONTOH", body["user_create"])
	detail := body["detail_perubahan"].(map[string]any)
	require.Equal(t, "2026-08-17", detail["dol_sebelum"])
	require.Equal(t, "2026-08-20", detail["dol_baru"])
	require.Equal(t, "OBJEK CONTOH SATU", detail["nama_objek"])

	rec, body = e.do(t, http.MethodGet, base+"/OPCN.26.0001", "", true)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "PNCN.26.0007", body["referensi_klaim"])
	require.Equal(t, "ubah DOL", body["keterangan"])
}

func TestCreateValidationFailureReturns422WithDetails(t *testing.T) {
	e := newEnv(t, options{})
	// Tanggal yang tidak dapat dibaca menjadi nil dan dilaporkan validasi domain.
	rec, body := e.do(t, http.MethodPost, base+"/",
		`{"tipe_proteksi":"7","detail_perubahan":{"dol_baru":"20/08/2026"}}`, true)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Equal(t, inputreqprotectionhttp.CodeValidation, body["kode"])
	require.Equal(t, "Isian belum lengkap atau belum benar.", body["pesan"])

	fields := map[string]bool{}
	for _, d := range body["detail"].([]any) {
		fields[d.(map[string]any)["field"].(string)] = true
	}
	require.True(t, fields[inputreqprotection.FieldClaimNumber])
	require.True(t, fields[inputreqprotection.FieldNote])
	require.True(t, fields[inputreqprotection.FieldLossDateAfter])
}

func TestCreateRejectsUnreadableAndUnknownFields(t *testing.T) {
	e := newEnv(t, options{})
	for _, raw := range []string{`{bukan json`, `{"nomor_polis":"POL-1"}`} {
		rec, body := e.do(t, http.MethodPost, base+"/", raw, true)
		require.Equal(t, http.StatusBadRequest, rec.Code, raw)
		require.Equal(t, "Badan permintaan tidak dapat dibaca.", body["pesan"])
	}
}

func TestCreateWithoutCallerReaderIsInternalError(t *testing.T) {
	e := newEnv(t, options{noCaller: true})
	rec, body := e.do(t, http.MethodPost, base+"/", `{"nomor_klaim":"PNCN.26.0007"}`, true)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, inputreqprotectionhttp.CodeInternalError, body["kode"])
	require.Equal(t, "Terjadi kesalahan pada sistem.", body["pesan"])
	require.Contains(t, e.logs.String(), "pembaca identitas tidak dipasang")
}

func TestCreateWithEmptyCallerIsInternalError(t *testing.T) {
	e := newEnv(t, options{})
	e.caller.Login = ""
	rec, _ := e.do(t, http.MethodPost, base+"/", `{"nomor_klaim":"PNCN.26.0007"}`, true)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, e.logs.String(), "identitas pemanggil tidak tersedia di konteks")
}

func TestUpdateFlowAndConflicts(t *testing.T) {
	e := newEnv(t, options{})
	e.repo.Add(
		inputreqprotection.Protection{Number: "OPCN.26.0050", PolicyNumber: "X", Type: "1",
			CreatedBy: "PEMBUAT", CreatedAt: fixedAt},
		inputreqprotection.Protection{Number: "OPCN.26.0051", ClaimNumber: "PNCN.26.0001"},
		inputreqprotection.Protection{Number: "OPCN.26.0052", AcceptStatus: inputreqprotection.AcceptApproved},
	)
	payload := `{"nomor_klaim":"PNCN.26.0008","tipe_proteksi":"8","keterangan":"ubah COL",
	  "detail_perubahan":{"penyebab_kerugian_baru":"COL-2"}}`

	rec, body := e.do(t, http.MethodPut, base+"/OPCN.26.0050", payload, true)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "PEMBUAT", body["user_create"])
	detail := body["detail_perubahan"].(map[string]any)
	require.Equal(t, "Banjir", detail["penyebab_kerugian"])
	require.Equal(t, "COL-2", detail["penyebab_kerugian_master"])

	rec, body = e.do(t, http.MethodPut, base+"/OPCN.26.0051", payload, true)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, inputreqprotectionhttp.CodeConflict, body["kode"])
	require.Equal(t, "Permintaan proteksi sudah tertaut ke klaim dan tidak dapat diubah lagi.", body["pesan"])

	rec, body = e.do(t, http.MethodPut, base+"/OPCN.26.0052", payload, true)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, "Permintaan proteksi sudah diakseptasi dan tidak dapat diubah lagi.", body["pesan"])

	rec, body = e.do(t, http.MethodPut, base+"/OPCN.26.0099", payload, true)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inputreqprotectionhttp.CodeNotFound, body["kode"])
	require.Equal(t, "Permintaan proteksi tidak ditemukan.", body["pesan"])
}

func TestUpdateRejectsBadBodyAndMissingCaller(t *testing.T) {
	e := newEnv(t, options{})
	rec, _ := e.do(t, http.MethodPut, base+"/OPCN.26.0050", `[`, true)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	e2 := newEnv(t, options{noCaller: true})
	rec, _ = e2.do(t, http.MethodPut, base+"/OPCN.26.0050", `{}`, true)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetUnknownNumberIs404(t *testing.T) {
	e := newEnv(t, options{})
	rec, body := e.do(t, http.MethodGet, base+"/OPC-999", "", true)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, inputreqprotectionhttp.CodeNotFound, body["kode"])
}

func TestDeleteRouteIsNotRegistered(t *testing.T) {
	e := newEnv(t, options{})
	req := httptest.NewRequest(http.MethodDelete, base+"/OPCN.26.0001", nil)
	req.Header.Set(portalhttp.HeaderPortal, "ASM")
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

// stubService mengembalikan galat yang ditentukan uji untuk setiap method.
type stubService struct{ err error }

func (s stubService) List(context.Context, usecase.ListQuery) (inputreqprotection.Page, error) {
	return inputreqprotection.Page{}, s.err
}
func (s stubService) ListTypes(context.Context, string) ([]inputreqprotection.ProtectionType, error) {
	return nil, s.err
}
func (s stubService) FindClaim(context.Context, string, string) (inputreqprotection.Claim, error) {
	return inputreqprotection.Claim{}, s.err
}
func (s stubService) Get(context.Context, string, string) (inputreqprotection.Protection, error) {
	return inputreqprotection.Protection{}, s.err
}
func (s stubService) Create(context.Context, usecase.SaveCommand) (inputreqprotection.Protection, error) {
	return inputreqprotection.Protection{}, s.err
}
func (s stubService) Update(context.Context, usecase.SaveCommand) (inputreqprotection.Protection, error) {
	return inputreqprotection.Protection{}, s.err
}

func TestUnknownServiceErrorsAreInternalAndLogged(t *testing.T) {
	e := newEnv(t, options{service: stubService{err: errors.New("rahasia basis data")}})
	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, base + "/", ""},
		{http.MethodGet, base + "/tipe", ""},
		{http.MethodGet, base + "/klaim/PNC-1", ""},
		{http.MethodGet, base + "/OPC-1", ""},
		{http.MethodPost, base + "/", `{}`},
		{http.MethodPut, base + "/OPC-1", `{}`},
	} {
		rec, body := e.do(t, c.method, c.path, c.body, true)
		require.Equal(t, http.StatusInternalServerError, rec.Code, c.path)
		require.Equal(t, "Terjadi kesalahan pada sistem.", body["pesan"])
		require.NotContains(t, rec.Body.String(), "rahasia", "rincian internal tidak boleh bocor")
	}
	require.Contains(t, e.logs.String(), "rahasia basis data")
}

func TestFallbackErrorWriterReceivesUnknownErrors(t *testing.T) {
	var got error
	fallback := func(w http.ResponseWriter, r *http.Request, err error) {
		got = err
		writeJSON(w, r, http.StatusUnauthorized, map[string]string{"kode": "sesi_habis"})
	}
	boom := errors.New("sesi kedaluwarsa")
	e := newEnv(t, options{service: stubService{err: boom}, fallback: fallback})

	rec, body := e.do(t, http.MethodGet, base+"/", "", true)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "sesi_habis", body["kode"])
	require.Same(t, boom, got)
}

func TestHandlerWithoutActivePortalInContextIsRejected(t *testing.T) {
	// Handler yang dipasang di luar middleware portal menolak, bukan jatuh ke portal utama.
	h := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service:   stubService{},
		Logger:    slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
		WriteJSON: writeJSON,
	})
	for _, fn := range []http.HandlerFunc{h.List, h.ListTypes} {
		rec := httptest.NewRecorder()
		fn(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	}

	// Dengan portal beralias kosong pun tetap ditolak.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(portalhttp.WithActivePortal(req.Context(), portal.Portal{Alias: " "}))
	rec := httptest.NewRecorder()
	h.ListTypes(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMissingNumberRouteParamIsBadRequest(t *testing.T) {
	// Get dan Update menolak nomor kosong sebelum menyentuh service.
	h := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service:   stubService{},
		WriteJSON: writeJSON,
	})
	for _, fn := range []http.HandlerFunc{h.Get, h.Update} {
		rec := httptest.NewRecorder()
		fn(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "Nomor proteksi wajib disebutkan.")
	}
}

func TestNilLocationDefaultsToJakarta(t *testing.T) {
	// Tanpa zona, handler memakai Asia/Jakarta: 18.00 UTC menjadi tanggal berikutnya.
	e := newEnv(t, options{location: nil})
	h := inputreqprotectionhttp.NewHandler(inputreqprotectionhttp.Options{
		Service: e.service, WriteJSON: writeJSON,
		GetCaller: func(context.Context) (inputreqprotectionhttp.Caller, bool) {
			return inputreqprotectionhttp.Caller{Login: "U"}, true
		},
	})
	e.repo.Add(inputreqprotection.Protection{Number: "OPCN.26.0001",
		InputDate: time.Date(2026, 9, 22, 18, 0, 0, 0, time.UTC)})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(portalhttp.WithActivePortal(req.Context(), portal.Portal{Alias: "ASM"}))
	rec := httptest.NewRecorder()
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"tanggal_proteksi":"2026-09-23"`)
}
