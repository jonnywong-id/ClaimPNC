package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Dokumen polis mencampur teks dan angka untuk kolom yang sama, dan PA/Travel menyimpan
// daftarnya di ASMCoverage, bukan CoverageList. Contohnya bentuk nyata dari Oracle ASM.
func TestParseCoveragesReadsBothShapes(t *testing.T) {
	cargo := `{"CoverageList":[{"Coverage":"11022","CoverageNote":"ICC C","TSI":"250000.000000",
		"SpreadingList":[{"TreatyType":"10007","SharePercentage":"100.0000","FlagDelete":"0"}]}]}`
	got, err := parseCoverages(cargo)
	require.NoError(t, err)
	require.Equal(t, []registrasi.SourceCoverage{{
		Code: "11022", Name: "ICC C", TSI: registrasi.Rupiah(250_000),
		Spreading: []registrasi.SourceSpreading{{TreatyType: "10007", Share: registrasi.PercentFull}},
	}}, got)

	pa := `{"ASMCoverage":[{"Coverage":10003,"TSI":10000000,"FlagDelete":1,
		"SpreadingList":[{"TreatyType":10005,"SharePercentage":0.00000000000000000000}]}]}`
	got, err = parseCoverages(pa)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "10003", got[0].Code)
	require.True(t, got[0].Deleted)
	require.Equal(t, registrasi.Percent(0), got[0].Spreading[0].Share)

	empty, err := parseCoverages("")
	require.NoError(t, err)
	require.Empty(t, empty)
}
