package registrasihttp

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// mapError memetakan setiap galat modul ke status dan kode yang tetap; galat yang dibungkus
// tetap dikenali.
func TestMapErrorStatusAndCode(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadInvalid, Message: "salah"}, http.StatusBadRequest, CodeDocumentUpload},
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadTooLarge, Message: "m"}, http.StatusRequestEntityTooLarge, CodeDocumentUpload},
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadUnavailable, Message: "m"}, http.StatusBadGateway, CodeDocumentUpload},
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadHalfDone, Message: "m"}, http.StatusInternalServerError, CodeDocumentUpload},
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadMisconfigured, Message: "m"}, http.StatusInternalServerError, CodeDocumentUpload},
		{&registrasi.DocumentUploadError{Kind: registrasi.UploadFailure(99), Message: "m"}, http.StatusInternalServerError, CodeDocumentUpload},
		{registrasi.ErrDocumentTypeUnknown, http.StatusUnprocessableEntity, CodeDocumentUpload},
		{registrasi.ErrDocumentFileEmpty, http.StatusBadRequest, CodeDocumentUpload},
		{&registrasi.ReportAlreadyRegisteredError{ClaimNumber: "PNCN.26.1"}, http.StatusConflict, CodeReportRegistered},
		{&registrasi.ValidationError{}, http.StatusUnprocessableEntity, CodeValidationFailed},
		{registrasi.ErrPolicyNotFound, http.StatusUnprocessableEntity, CodePolicyNotFound},
		{registrasi.ErrUnknownAreaLevel, http.StatusBadRequest, CodeMalformedRequest},
		{registrasi.ErrAccountNotFound, http.StatusNotFound, CodeAccountNotFound},
		{registrasi.ErrCommitteeNotFound, http.StatusNotFound, CodeCommitteeNotFound},
		{registrasi.ErrNotCommitteeTurn, http.StatusForbidden, CodeNotCommitteeTurn},
		{registrasi.ErrClaimNotFound, http.StatusNotFound, CodeClaimNotFound},
		{registrasi.ErrTaskNotFound, http.StatusNotFound, CodeTaskNotFound},
		{registrasi.ErrTaskAlreadyClaimed, http.StatusConflict, CodeTaskAlreadyClaimed},
		{registrasi.ErrNotTaskOwner, http.StatusForbidden, CodeNotTaskOwner},
		{registrasi.ErrTaskAlreadyDone, http.StatusConflict, CodeTaskAlreadyDone},
		{registrasi.ErrStageMismatch, http.StatusConflict, CodeStageMismatch},
		{registrasi.ErrInvalidAction, http.StatusBadRequest, CodeInvalidAction},
		{usecase.ErrCashierUnavailable, http.StatusBadGateway, CodeCashierUnavailable},
		{&usecase.CashierRejectedError{Message: "ditolak"}, http.StatusUnprocessableEntity, CodeCashierRejected},
		{usecase.ErrPremiumUnavailable, http.StatusBadGateway, CodePremiumUnavailable},
		{registrasi.ErrExchangeRateNotFound, http.StatusUnprocessableEntity, CodeExchangeRateNotFound},
		{errors.New("lain"), http.StatusInternalServerError, CodeInternalError},
	}
	for _, c := range cases {
		wrapped := fmt.Errorf("lapisan: %w", c.err)
		status, body := mapError(wrapped)
		require.Equal(t, c.status, status, "galat = %v", c.err)
		require.Equal(t, c.code, body.Code, "galat = %v", c.err)
		require.NotEmpty(t, body.Message, "galat = %v", c.err)
	}
}

// Pesan unggah dan nomor klaim terdaftar diteruskan ke pengguna.
func TestMapErrorMessages(t *testing.T) {
	_, body := mapError(&registrasi.DocumentUploadError{Kind: registrasi.UploadInvalid, Message: "Berkas rusak"})
	require.Equal(t, "Berkas rusak", body.Message)

	_, body = mapError(&registrasi.ReportAlreadyRegisteredError{ClaimNumber: "PNCN.26.7"})
	require.Contains(t, body.Message, "PNCN.26.7")

	_, body = mapError(&usecase.CashierRejectedError{Message: "Rekening diblokir"})
	require.Equal(t, "Rekening diblokir", body.Message)
}

// Galat validasi tanpa pelanggaran memakai pesan umum; dengan pelanggaran memakai yang pertama.
func TestShortMessageAndViolations(t *testing.T) {
	require.Equal(t, "Data klaim belum memenuhi ketentuan.", shortMessage(&registrasi.ValidationError{}))
	require.Empty(t, violationsDTO(&registrasi.ValidationError{}))

	g := &registrasi.ValidationError{Violation: []registrasi.Violation{
		{Code: "satu", Field: "f1", Message: "pesan satu"},
		{Code: "dua", Field: "f2", Message: "pesan dua"},
	}}
	require.Equal(t, "pesan satu", shortMessage(g))
	require.Equal(t, []ViolationDTO{
		{Code: "satu", Field: "f1", Message: "pesan satu"},
		{Code: "dua", Field: "f2", Message: "pesan dua"},
	}, violationsDTO(g))
}

// Tanggal kosong ditulis kosong; waktu UTC ditulis sebagai tanggal WIB.
func TestFormatDateAndMoment(t *testing.T) {
	require.Empty(t, formatDate(time.Time{}))
	require.Empty(t, formatMoment(time.Time{}))
	at := time.Date(2026, time.June, 9, 18, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-06-10", formatDate(at))
	require.Equal(t, "2026-06-10T01:00:00+07:00", formatMoment(at))
}

// Tanggal opsional yang kosong diterima sebagai waktu nol.
func TestOptionalDate(t *testing.T) {
	d, err := optionalDate("  ", "x")
	require.NoError(t, err)
	require.True(t, d.IsZero())
	d, err = optionalDate(" 2026-06-09 ", "x")
	require.NoError(t, err)
	require.Equal(t, 9, d.Day())
	_, err = optionalDate("09/06/2026", "tanggal")
	require.EqualError(t, err, "tanggal harus berformat YYYY-MM-DD")
}
