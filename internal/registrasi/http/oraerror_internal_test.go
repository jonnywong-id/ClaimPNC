package registrasihttp

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Galat Oracle ditampilkan apa adanya beserta tabelnya, bukan "Terjadi kesalahan pada sistem"
// (Work Owner 2026-10-10).
func TestOracleErrorIsShownWithItsTable(t *testing.T) {
	err := fmt.Errorf("registrasi/sqlstore: menyisipkan klaim ke POOLDATA.T_CLAIM_PNC: %w",
		errors.New("ORA-01427: single-row subquery returns more than one row"))
	status, body := mapError(err)
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, CodeDatabaseError, body.Code)
	require.Equal(t, "Gagal penyimpanan ke tabel POOLDATA.T_CLAIM_PNC: ORA-01427: single-row subquery returns more than one row", body.Message)

	status, body = mapError(errors.New("jaringan putus"))
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, CodeInternalError, body.Code)
}
