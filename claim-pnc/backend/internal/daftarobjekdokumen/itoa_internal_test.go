package daftarobjekdokumen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// itoa menghasilkan "0" untuk nol, bukan teks kosong.
func TestItoaHandlesZeroAndMultipleDigits(t *testing.T) {
	require.Equal(t, "0", itoa(0))
	require.Equal(t, "7", itoa(7))
	require.Equal(t, "100", itoa(100))
}
