package inboxlaporanklaimhttp

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Galat Oracle saat menyimpan berkas tampil di layar beserta tabelnya; galat lain tetap ke
// penulis galat bersama (Work Owner 2026-10-10).
func TestOracleErrorIsShownWithItsTable(t *testing.T) {
	var gotStatus int
	var gotBody any
	var fallback []error
	h := &Handler{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		writeResponse: func(_ http.ResponseWriter, _ *http.Request, status int, body any) {
			gotStatus, gotBody = status, body
		},
		writeError: func(_ http.ResponseWriter, _ *http.Request, err error) { fallback = append(fallback, err) },
	}
	r := httptest.NewRequest(http.MethodPut, "/inbox/laporan-klaim/RCVN.26.1", nil)

	err := fmt.Errorf("inboxlaporanklaim/sqlstore: menyimpan %q ke POOLDATA.T_CLAIM_RECIVEDCLAIM: %w",
		"RCVN.26.1", errors.New("ORA-01427: single-row subquery returns more than one row"))
	h.writeModuleError(httptest.NewRecorder(), r, err)
	require.Equal(t, http.StatusInternalServerError, gotStatus)
	require.Equal(t, ErrorResponse{
		Code:    CodeDatabaseError,
		Message: "Gagal penyimpanan ke tabel POOLDATA.T_CLAIM_RECIVEDCLAIM: ORA-01427: single-row subquery returns more than one row",
	}, gotBody)
	require.Empty(t, fallback)

	h.writeModuleError(httptest.NewRecorder(), r, errors.New("jaringan putus"))
	require.Len(t, fallback, 1)
}
