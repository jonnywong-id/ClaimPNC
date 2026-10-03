package masterpanel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

// Sisi yang tidak dikenal tidak punya label.
func TestAnUnknownSideHasNoLabel(t *testing.T) {
	require.Equal(t, "", masterpanel.Side("9").Label())
	require.Equal(t, "KIRI", masterpanel.SideLeft.Label())
}

// Pesan log galat validasi menyebut setiap isian.
func TestValidationErrorMessageListsEveryField(t *testing.T) {
	err := &masterpanel.ValidationError{Violation: []masterpanel.Violation{
		{Field: "nama_panel", Message: "wajib"},
		{Field: "lokasi", Message: "terlalu banyak"},
	}}
	require.Equal(t,
		"masterpanel: isian tidak sah (nama_panel: wajib; lokasi: terlalu banyak)", err.Error())

	single := masterpanel.OneViolation("catatan", "panjang")
	require.EqualError(t, single, "masterpanel: isian tidak sah (catatan: panjang)")
}

// Nama lokasi yang melebihi batas ditolak pada barisnya sendiri.
func TestALocationNameLongerThanTheLimitIsRejected(t *testing.T) {
	input := validInput()
	input.Location = []masterpanel.PanelLocation{{
		Name: strings.Repeat("L", masterpanel.MaxLocationLength+1),
		Side: masterpanel.SideNone,
	}}

	require.Contains(t, violationFields(t, input.Clean().Check()), "lokasi.0.lokasi_panel")
}
