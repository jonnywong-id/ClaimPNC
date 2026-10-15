package masterpenyebabkerugian

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// itoa menuliskan nol sebagai "0", bukan teks kosong.
func TestItoaWritesZero(t *testing.T) {
	require.Equal(t, "0", itoa(0))
	require.Equal(t, "100", itoa(100))
}
