package masterpenolakanhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
	"claim-pnc/internal/masterpenolakan/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

var errStorage = errors.New("penyimpanan mati")

// unitHandler adalah Handler yang dirakit tanpa server, supaya cabang yang tidak dapat
// dicapai lewat rantai middleware (portal tidak ada di context, pemanggil tanpa login)
// dapat diuji langsung.
type unitHandler struct {
	handler   *Handler
	repo      *memory.Repo
	komite    *memory.RepoKomite
	lastError error
	login     string
}

func newUnitHandler(t *testing.T) *unitHandler {
	t.Helper()
	u := &unitHandler{
		repo:   memory.NewRepo(memory.SampleParents(), memory.SampleList()...),
		komite: memory.NewRepoKomite(memory.SampleListKomite()...),
		login:  "adminpnc",
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpenolakan.Repo, error) { return u.repo, nil },
		Clock:        clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	komite, err := usecase.NewServiceKomite(usecase.OptionsKomite{
		RepoSelector: func(string) (masterpenolakan.RepoKomite, error) { return u.komite, nil },
	})
	require.NoError(t, err)

	u.handler, err = NewHandler(Options{
		Service: service,
		Komite:  komite,
		Caller: func(context.Context) (Caller, bool) {
			if u.login == "" {
				return Caller{}, false
			}
			return Caller{Login: u.login}, true
		},
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			u.lastError = err
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	require.NoError(t, err)
	return u
}

// serve menjalankan handler dengan portal ASM (bila withPortal) dan parameter id opsional.
func (u *unitHandler) serve(h http.HandlerFunc, method, body string, withPortal bool, id string) (*httptest.ResponseRecorder, map[string]any) {
	request := httptest.NewRequest(method, "/x", strings.NewReader(body))
	ctx := request.Context()
	if withPortal {
		ctx = portalhttp.WithActivePortal(ctx, portal.Portal{ID: "1", Name: "Asuransi Sinar Mas", Alias: "ASM"})
	}
	route := chi.NewRouteContext()
	route.URLParams.Add("id", id)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, route)

	recorder := httptest.NewRecorder()
	h(recorder, request.WithContext(ctx))

	content := map[string]any{}
	_ = json.NewDecoder(recorder.Body).Decode(&content)
	return recorder, content
}

func TestNewHandlerRejectsMissingParts(t *testing.T) {
	service := &usecase.Service{}
	komite := &usecase.ServiceKomite{}
	caller := func(context.Context) (Caller, bool) { return Caller{}, false }
	writeResponse := func(http.ResponseWriter, *http.Request, int, any) {}
	writeError := func(http.ResponseWriter, *http.Request, error) {}

	_, err := NewHandler(Options{Komite: komite, Caller: caller, WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Service dan Komite wajib diisi")
	_, err = NewHandler(Options{Service: service, Caller: caller, WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Service dan Komite wajib diisi")
	_, err = NewHandler(Options{Service: service, Komite: komite, WriteResponse: writeResponse, WriteError: writeError})
	require.ErrorContains(t, err, "Caller wajib diisi")
	_, err = NewHandler(Options{Service: service, Komite: komite, Caller: caller, WriteError: writeError})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
	_, err = NewHandler(Options{Service: service, Komite: komite, Caller: caller, WriteResponse: writeResponse})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

func TestHandlersWithoutActivePortalAreRejected(t *testing.T) {
	u := newUnitHandler(t)
	for name, h := range map[string]http.HandlerFunc{
		"list":          u.handler.List,
		"parent":        u.handler.Parent,
		"create":        u.handler.Create,
		"update":        u.handler.Update,
		"list komite":   u.handler.ListKomite,
		"create komite": u.handler.CreateKomite,
		"update komite": u.handler.UpdateKomite,
	} {
		t.Run(name, func(t *testing.T) {
			u.lastError = nil
			recorder, _ := u.serve(h, http.MethodGet, `{}`, false, "1")
			// portal.ErrNotStated tidak dikenal mapError modul ini, jadi diteruskan ke WriteError.
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.ErrorIs(t, u.lastError, portal.ErrNotStated)
		})
	}
}

func TestParentListsStatusOneOfPortal(t *testing.T) {
	u := newUnitHandler(t)
	recorder, content := u.serve(u.handler.Parent, http.MethodGet, "", true, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "ASM", content["portal"])
	parents := content["status_1"].([]any)
	require.Len(t, parents, 4)
	require.Equal(t, map[string]any{"id": "4", "nama": "DOKUMEN TIDAK DILENGKAPI"}, parents[0])
}

func TestRepositoryFailuresGoToGenericErrorWriter(t *testing.T) {
	u := newUnitHandler(t)
	u.repo.SetError(errStorage)
	u.komite.SetError(errStorage)

	cases := map[string]struct {
		h    http.HandlerFunc
		body string
	}{
		"list":          {u.handler.List, ""},
		"parent":        {u.handler.Parent, ""},
		"create":        {u.handler.Create, `{"nama":"A","nama_status_1":"B"}`},
		"update":        {u.handler.Update, `{"nama":"A","nama_status_1":"B"}`},
		"list komite":   {u.handler.ListKomite, ""},
		"create komite": {u.handler.CreateKomite, `{"catatan":"A"}`},
		"update komite": {u.handler.UpdateKomite, `{"catatan":"A"}`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			u.lastError = nil
			recorder, _ := u.serve(c.h, http.MethodPost, c.body, true, "1")
			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.ErrorIs(t, u.lastError, errStorage)
		})
	}
}

func TestSaveWithoutCallerLoginIsGenericError(t *testing.T) {
	u := newUnitHandler(t)
	u.login = ""
	recorder, _ := u.serve(u.handler.Create, http.MethodPost, `{"nama":"A","nama_status_1":"B"}`, true, "")
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.ErrorContains(t, u.lastError, "identitas pemanggil tidak tersedia")
}

func TestUpdateWithoutIDIsNotFound(t *testing.T) {
	u := newUnitHandler(t)

	recorder, content := u.serve(u.handler.Update, http.MethodPut, `{}`, true, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, CodeNotFound, content["kode"])
	require.Contains(t, content["pesan"], "Status penolakan")

	recorder, content = u.serve(u.handler.UpdateKomite, http.MethodPut, `{}`, true, "")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, content["pesan"], "Penolakan komite")
}

func TestUpdateSuccessThroughHandler(t *testing.T) {
	u := newUnitHandler(t)
	recorder, content := u.serve(u.handler.Update, http.MethodPut, `{"nama":"UBAH","id_status_1":"2"}`, true, "1")
	require.Equal(t, http.StatusOK, recorder.Code)
	row := content["penolakan_klaim"].(map[string]any)
	require.Equal(t, "UBAH", row["nama"])
	require.Equal(t, "0", row["status"])
	require.Equal(t, "MENUNGGU", row["status_label"])
	require.Equal(t, "2026-09-19T03:00:00Z", row["diajukan_pada"])
	// Jejak persetujuan lama ikut dikirim apa adanya.
	require.Equal(t, "2026-09-12T03:00:00Z", row["disetujui_pada"])
}

func TestMalformedBodiesAreBadRequest(t *testing.T) {
	u := newUnitHandler(t)
	for name, body := range map[string]string{
		"not json":       `{`,
		"trailing value": `{"catatan":"A"} {}`,
		"unknown field":  `{"lain":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			recorder, content := u.serve(u.handler.CreateKomite, http.MethodPost, body, true, "")
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, CodeMalformedRequest, content["kode"])
		})
	}

	// Badan rusak pada ubah komite juga ditolak sebelum menyentuh penyimpanan.
	recorder, _ := u.serve(u.handler.UpdateKomite, http.MethodPut, `{`, true, "111")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	recorder, _ = u.serve(u.handler.Update, http.MethodPut, `{"nama":"A"} 1`, true, "1")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestMapErrorIDTakenIsConflict(t *testing.T) {
	status, body, known := mapError(masterpenolakan.ErrIDTaken)
	require.True(t, known)
	require.Equal(t, http.StatusConflict, status)
	require.Equal(t, CodeValidationFailed, body.Code)

	_, _, known = mapError(errStorage)
	require.False(t, known)
}

func TestFormatTimeEmptyForZero(t *testing.T) {
	require.Equal(t, "", formatTime(time.Time{}))
	require.Equal(t, "", formatTimePointer(nil))
	wib := time.Date(2026, 9, 19, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	require.Equal(t, "2026-09-19T03:00:00Z", formatTime(wib))
}
