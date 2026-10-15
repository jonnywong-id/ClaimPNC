package masterreashttp_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/masterreas/repo/memory"
	masterreasusecase "claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/portal"

	masterreashttp "claim-pnc/internal/masterreas/http"
	portalhttp "claim-pnc/internal/portal/http"
)

// directHandler merakit Handler tanpa router, supaya jalur yang tidak terjangkau lewat
// middleware portal dapat diuji langsung.
type directHandler struct {
	handler *masterreashttp.Handler
	logs    *bytes.Buffer
	gotErr  error
	status  int
	body    any
}

func newDirectHandler(t *testing.T, repo masterreas.Repo) *directHandler {
	t.Helper()

	service, err := masterreasusecase.NewService(masterreasusecase.Options{
		RepoSelector: func(string) (masterreas.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)

	d := &directHandler{logs: &bytes.Buffer{}}
	logger := slog.New(slog.NewJSONHandler(d.logs, nil))
	d.handler, err = masterreashttp.NewHandler(masterreashttp.Options{
		Service: service,
		// Caller menjadi WAJIB sejak jalur ubah ditambahkan (2026-10-05). Di uji ini ia
		// tidak dipakai — seluruh ujinya menembak jalur baca — tetapi NewHandler menolak
		// rakitan yang tidak lengkap, dan itu memang yang diinginkan.
		Caller: func(context.Context) (masterreashttp.Caller, bool) {
			return masterreashttp.Caller{Login: "uji"}, true
		},
		Logger: logger,
		WriteResponse: func(w http.ResponseWriter, _ *http.Request, status int, body any) {
			d.status, d.body = status, body
			w.WriteHeader(status)
		},
		WriteError: func(w http.ResponseWriter, _ *http.Request, err error) {
			d.gotErr = err
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	require.NoError(t, err)
	return d
}

func TestNewHandlerRejectsIncompleteOptions(t *testing.T) {
	_, err := masterreashttp.NewHandler(masterreashttp.Options{})
	require.ErrorContains(t, err, "Service wajib diisi")

	service, err := masterreasusecase.NewService(masterreasusecase.Options{
		RepoSelector: func(string) (masterreas.Repo, error) { return memory.NewRepo(memory.Options{}), nil },
	})
	require.NoError(t, err)

	// Caller menjadi WAJIB sejak jalur ubah ditambahkan (2026-10-05): identitas pemanggil
	// mengisi log, dan pada tabel ini log adalah SATU-SATUNYA tempat "siapa yang mengubah
	// surel ini" terekam — kolom pencatat pelaku memang tidak ada.
	_, err = masterreashttp.NewHandler(masterreashttp.Options{Service: service})
	require.ErrorContains(t, err, "Caller wajib diisi")

	caller := func(context.Context) (masterreashttp.Caller, bool) {
		return masterreashttp.Caller{Login: "uji"}, true
	}

	_, err = masterreashttp.NewHandler(masterreashttp.Options{Service: service, Caller: caller})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")

	_, err = masterreashttp.NewHandler(masterreashttp.Options{
		Service:       service,
		Caller:        caller,
		WriteResponse: func(http.ResponseWriter, *http.Request, int, any) {},
	})
	require.ErrorContains(t, err, "WriteResponse dan WriteError wajib diisi")
}

// Tanpa portal aktif di konteks, galatnya ErrNotStated dan TIDAK dicatat sebagai kegagalan.
func TestListWithoutActivePortalInContext(t *testing.T) {
	d := newDirectHandler(t, memory.NewSampleRepo())

	recorder := httptest.NewRecorder()
	d.handler.List(recorder, httptest.NewRequest(http.MethodGet, "/master/reas", nil))

	require.ErrorIs(t, d.gotErr, portal.ErrNotStated)
	require.Empty(t, d.logs.String(), "galat klien tidak boleh dicatat sebagai kegagalan")
}

// Galat teknis dari penyimpanan diteruskan ke penulis galat DAN dicatat ke log.
func TestListRepositoryFailureIsLogged(t *testing.T) {
	repo := memory.NewSampleRepo()
	boom := errors.New("oracle mati")
	repo.SetError(boom)
	d := newDirectHandler(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/master/reas", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(),
		portal.Portal{Alias: "ASM"}))
	recorder := httptest.NewRecorder()
	d.handler.List(recorder, request)

	require.ErrorIs(t, d.gotErr, boom)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, d.logs.String(), "permintaan gagal")
	require.Contains(t, d.logs.String(), "oracle mati")
	require.Contains(t, d.logs.String(), "/master/reas")
}

func TestListWithActivePortalWritesResponse(t *testing.T) {
	d := newDirectHandler(t, memory.NewSampleRepo())

	request := httptest.NewRequest(http.MethodGet, "/master/reas?cari=andalas", nil)
	request = request.WithContext(portalhttp.WithActivePortal(request.Context(),
		portal.Portal{Alias: "ASM"}))
	d.handler.List(httptest.NewRecorder(), request)

	require.NoError(t, d.gotErr)
	require.Equal(t, http.StatusOK, d.status)
	response, ok := d.body.(masterreashttp.ListResponse)
	require.True(t, ok)
	require.Equal(t, "ASM", response.Portal)
	require.Len(t, response.Member, 2)
}
