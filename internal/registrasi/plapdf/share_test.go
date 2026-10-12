package plapdf

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Persen share BPPDAN pada PLA ditulis dua desimal (Work Owner 2026-10-09): bagian 0,2%
// dulu tercetak "0 %" karena dibulatkan ke bilangan bulat.
func TestTwoDecimals(t *testing.T) {
	cases := map[registrasi.Money]string{
		0:       "0,00",
		20:      "0,20",
		5:       "0,05",
		100:     "1,00",
		1234567: "12.345,67",
		-250:    "-2,50",
	}
	for in, want := range cases {
		require.Equal(t, want, twoDecimals(in), int64(in))
	}
}
