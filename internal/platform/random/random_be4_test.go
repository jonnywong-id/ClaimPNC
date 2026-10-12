package random_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/random"
)

// TestPickStaysInRange membuktikan indeks selalu dalam [0, count) dan nilai janggal dijawab nol.
func TestPickStaysInRange(t *testing.T) {
	picker := random.System{}
	require.Equal(t, 0, picker.Pick(0))
	require.Equal(t, 0, picker.Pick(-4))
	require.Equal(t, 0, picker.Pick(1))
	for i := 0; i < 200; i++ {
		got := picker.Pick(3)
		require.GreaterOrEqual(t, got, 0)
		require.Less(t, got, 3)
	}
}
