package registrasihttp_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	registrasihttp "claim-pnc/internal/registrasi/http"
)

// POST /tugas/{id}/kirim-inputor memindahkan klaim ke Input Register dan mengembalikan tugas barunya.
func TestSendToInputorRoute(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.upToChooseSurveyor(t)

	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+task.ID+"/kirim-inputor",
		registrasihttp.SendToInputorRequest{Note: "Lengkapi KTP."})
	require.Equal(t, http.StatusOK, w.Code, "badan = %s", w.Body.String())
	out := decode[registrasihttp.ClaimResponse](t, w)
	require.Equal(t, registrasi.StageInputRegister, out.Claim.CurrentStage)
	require.NotNil(t, out.Task)
	require.Equal(t, registrasi.StageInputRegister, out.Task.Stage)

	// Tugas lama sudah selesai.
	w = e.do(t, http.MethodPost, "/registrasi/tugas/"+task.ID+"/kirim-inputor", registrasihttp.SendToInputorRequest{})
	require.Equal(t, http.StatusConflict, w.Code, "badan = %s", w.Body.String())
}

// Catatan melebihi 4000 karakter ditolak sebagai pelanggaran validasi.
func TestSendToInputorRouteRejectsLongNote(t *testing.T) {
	e := newHTTPEnv(t)
	task := e.upToChooseSurveyor(t)

	w := e.do(t, http.MethodPost, "/registrasi/tugas/"+task.ID+"/kirim-inputor",
		registrasihttp.SendToInputorRequest{Note: strings.Repeat("a", 4001)})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, "badan = %s", w.Body.String())
	require.Contains(t, w.Body.String(), "catatan_terlalu_panjang")
}
