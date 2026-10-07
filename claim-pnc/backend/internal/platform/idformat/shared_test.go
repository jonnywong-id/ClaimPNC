package idformat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompose(t *testing.T) {
	require.Equal(t, "JKT00042", Compose(" JKT ", 42, 5))
	require.Equal(t, "JKT123456", Compose("JKT", 123456, 5))
}
