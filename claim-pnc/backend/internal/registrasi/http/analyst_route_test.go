package registrasihttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	registrasihttp "claim-pnc/internal/registrasi/http"
)

// POST /tugas/{id}/transfer-analis hanya berlaku pada Estimation PA; tahap lain ditolak.
func TestTransferToAnalystRouteRejectsOtherStage(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.upToChooseSurveyor(t)

	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+task.ID+"/transfer-analis",
		registrasihttp.TransferToAnalystRequest{ObjectID: "1", CoverageID: "10001"})
	require.Equal(t, http.StatusConflict, w.Code, "badan = %s", w.Body.String())
	require.Contains(t, w.Body.String(), "tidak_tersedia_di_tahap")
}
