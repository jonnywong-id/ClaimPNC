package businessline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLines(t *testing.T) {
	got := Lines()
	require.Equal(t, []Line{All, NonMBU, Bonding, PA, Travel}, got)
	got[0] = "X"
	require.Equal(t, All, Lines()[0])

	require.Equal(t, "Semua Lini Bisnis", All.Label())
	require.Equal(t, "Non-MBU", NonMBU.Label())
	require.Equal(t, "Bonding", Bonding.Label())
	require.Equal(t, "Personal Accident", PA.Label())
	require.Equal(t, "Travel", Travel.Label())
	require.Equal(t, "LAIN", Line("LAIN").Label())

	v, ok := Parse(" pa ")
	require.True(t, ok)
	require.Equal(t, PA, v)
	v, ok = Parse("")
	require.True(t, ok)
	require.Equal(t, All, v)
	_, ok = Parse("motor")
	require.False(t, ok)
}
