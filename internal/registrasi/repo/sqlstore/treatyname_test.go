package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Nama treaty diisi dari master per jenis treaty; nama yang sudah ada tidak ditimpa.
func TestNameTreatiesFillsFromMaster(t *testing.T) {
	items := []registrasi.SourceItem{{Coverage: []registrasi.SourceCoverage{{Spreading: []registrasi.SourceSpreading{
		{TreatyType: "10007"}, {TreatyType: " 10015 "}, {TreatyType: "10001", TreatyName: "SUDAH"}, {TreatyType: "99999"},
	}}}}}
	nameTreaties(items, map[string]string{"10007": "ORS", "10015": "FAC-OUT", "10001": "OR"})

	got := items[0].Coverage[0].Spreading
	require.Equal(t, "ORS", got[0].TreatyName)
	require.Equal(t, "FAC-OUT", got[1].TreatyName)
	require.Equal(t, "SUDAH", got[2].TreatyName)
	require.Equal(t, "", got[3].TreatyName)
}
