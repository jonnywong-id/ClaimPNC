package account_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/account"
)

// Perekam mencatat permintaan tanpa membuat akun, dan menuliskannya ke log.
func TestPerekamMencatatPermintaan(t *testing.T) {
	logs := &bytes.Buffer{}
	recorder := account.NewRecorder(slog.New(slog.NewJSONHandler(logs, nil)))

	req := mastersurveyors.AccountRequest{
		UserID: "SURVBARU", UserName: "Surveyor Baru", Unit: "Internal",
		AccessGroup: "GCNMFW:PNCSurveyor", MustChangePassword: true, Password: "rahasia",
	}
	require.NoError(t, recorder.Register(context.Background(), "ASM", req))

	recorded := recorder.Recorded()
	require.Equal(t, []account.Recorded{{PortalAlias: "ASM", Request: req}}, recorded)

	recorded[0].PortalAlias = "DIUBAH"
	require.Equal(t, "ASM", recorder.Recorded()[0].PortalAlias, "hasilnya salinan")

	require.Contains(t, logs.String(), "akun BELUM dibuat")
	require.Contains(t, logs.String(), `"login":"SURVBARU"`)
	require.NotContains(t, logs.String(), "rahasia", "sandi sementara tidak masuk log")
}

func TestPerekamMenjawabLoginTerpakai(t *testing.T) {
	recorder := account.NewRecorder(nil)
	require.NoError(t, recorder.Register(context.Background(), "ASM",
		mastersurveyors.AccountRequest{UserID: " SurvBaru "}))

	require.True(t, recorder.Taken("survbaru"))
	require.False(t, recorder.Taken("lain"))
	require.False(t, recorder.Taken("  "))
}
