package registrasi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// CallActivityInputRegister langkah 44: kalimat baku + tanggal kejadian DD/MM/YYYY + Kronologis,
// tanpa spasi sebelum Kronologis.
func TestPAReportDescription(t *testing.T) {
	loss := time.Date(2026, 8, 13, 0, 0, 0, 0, clock.ZoneWIB).UTC()
	require.Equal(t,
		"Berdasarkan surat keterangan kematian dari ... menyatakan bahwa benar telah meninggal dunia "+
			"tertanggung atas nama ... pada tanggal 13/08/2026 yang diakibatkan oleh karena ...Rawat inap",
		registrasi.PAReportDescription(loss, "Rawat inap"))
	require.Contains(t, registrasi.PAReportDescription(time.Time{}, ""), "pada tanggal // yang diakibatkan oleh karena ...")
}
