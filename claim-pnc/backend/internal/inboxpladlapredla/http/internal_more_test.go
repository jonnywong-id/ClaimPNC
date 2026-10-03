package inboxpladlapredlahttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

func recordJSON(w http.ResponseWriter, _ *http.Request, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Galat yang tidak dikenali tanpa cadangan dijawab 500 umum, dan rinciannya hanya ke log.
func TestUnknownErrorsWithoutAFallbackBecomeAGenericInternalError(t *testing.T) {
	log := &bytes.Buffer{}
	write := WriteError(slog.New(slog.NewTextHandler(log, nil)), recordJSON, nil)

	recorder := httptest.NewRecorder()
	write(recorder, httptest.NewRequest(http.MethodGet, "/x", nil), errors.New("rahasia DB"))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, CodeInternalError, body.Code)
	require.NotContains(t, recorder.Body.String(), "rahasia DB")
	require.Contains(t, log.String(), "rahasia DB")
}

// Pemetaan galat yang tidak dapat dicapai lewat rute dengan penyimpanan contoh.
func TestMapErrorCoversEveryDomainError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("%w: catatan gagal", inboxpladlapredla.ErrSentButNotMarked),
			http.StatusInternalServerError, CodeSentButNotMarked},
		{inboxpladlapredla.ErrWriteNotAvailable, http.StatusNotImplemented,
			CodeWriteNotAvailable},
		{inboxpladlapredla.NewNotAvailable("kirim"), http.StatusNotImplemented,
			CodeWriteNotAvailable},
		{inboxpladlapredla.ErrPreDLAAlreadySent, http.StatusConflict, CodeAlreadySent},
	}

	for _, c := range cases {
		status, body, known := mapError(c.err)
		require.True(t, known, c.err.Error())
		require.Equal(t, c.status, status, c.err.Error())
		require.Equal(t, c.code, body.Code, c.err.Error())
	}

	// Galat "terkirim tanpa catatan" membawa pesan aslinya supaya pengguna tidak mengulang.
	_, body, _ := mapError(fmt.Errorf("%w: catatan gagal", inboxpladlapredla.ErrSentButNotMarked))
	require.Contains(t, body.Message, "catatan gagal")

	_, _, known := mapError(errors.New("asing"))
	require.False(t, known)
}

// Handler yang dipanggil tanpa portal di konteks menolak, bukan memakai portal utama.
func TestHandlersWithoutAnActivePortalAreRejected(t *testing.T) {
	h := NewHandler(Options{
		WriteJSON:           recordJSON,
		FallbackErrorWriter: ErrorWriter(portalhttp.WithPortalError(nil, recordJSON)),
	})

	for _, call := range []http.HandlerFunc{h.Metadata, h.List, h.RejectWrite} {
		recorder := httptest.NewRecorder()
		call(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Contains(t, recorder.Body.String(), portalhttp.CodeNotStated)
	}
}

// Tanpa jembatan identitas, pemanggil tidak dikenal.
func TestWithoutACallerBridgeTheCallerIsUnknown(t *testing.T) {
	h := NewHandler(Options{WriteJSON: recordJSON})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(portalhttp.WithActivePortal(
		request.Context(), portal.Portal{Alias: "ASM"}))

	recorder := httptest.NewRecorder()
	h.List(recorder, request)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), CodeCallerUnknown)
}

// Angka halaman yang tidak terbaca atau negatif menjadi nol, lalu dibetulkan Normalize.
func TestPositiveNumberTurnsGarbageIntoZero(t *testing.T) {
	require.Equal(t, 0, positiveNumber("abc"))
	require.Equal(t, 0, positiveNumber("-3"))
	require.Equal(t, 7, positiveNumber(" 7 "))
}

// Batas rentang yang tidak dipasang menjadi teks kosong di kedua arah.
func TestDateTextOfAMissingBoundIsEmpty(t *testing.T) {
	require.Empty(t, dateText(nil))
	require.Empty(t, inclusiveDateText(nil))

	exclusive := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-01-31", inclusiveDateText(&exclusive))
}

// Kolom yang tidak dikenal menghasilkan sel kosong, bukan panik.
func TestGridCellOfAnUnknownColumnIsEmpty(t *testing.T) {
	row := inboxpladlapredla.Row{
		ClaimNo: "N", PolicyNo: "P", Insured: "I", RegisterDate: "R", LossDate: "L",
		PICTeknik: "T", AdviceDate: "A",
	}
	require.Equal(t, "", gridCell(row, "entah"))
	require.Equal(t, "R", gridCell(row, inboxpladlapredla.FieldRegisterDate))
	require.Equal(t, "L", gridCell(row, inboxpladlapredla.FieldLossDate))
	require.Equal(t, "T", gridCell(row, inboxpladlapredla.FieldPICTeknik))
}

// Nama berkas tanpa kode daftar jatuh ke nama modul.
func TestExportFilenameWithoutATabCode(t *testing.T) {
	require.Equal(t, "inbox-pla-dla-pre-dla.csv", exportFilename(inboxpladlapredla.Tab{}))
}

// Penanda potongan menyebut batas dan jumlah yang cocok di kolom pertama.
func TestTruncationNoticeNamesTheLimit(t *testing.T) {
	require.Empty(t, truncationNotice(0, 10))

	notice := truncationNotice(3, 60000)
	require.Len(t, notice, 3)
	require.Equal(t,
		"-- Terpotong pada 50000 baris dari 60000 yang cocok. Persempit rentang tanggalnya. --",
		notice[0])
	require.Empty(t, notice[1])
}
