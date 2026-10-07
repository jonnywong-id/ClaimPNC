package httpquery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNonNegative(t *testing.T) {
	require.Equal(t, 5, NonNegative("5"))
	require.Equal(t, 0, NonNegative("-1"))
	require.Equal(t, 0, NonNegative("x"))
	require.Equal(t, 0, NonNegative(" 5"))
	require.Equal(t, 5, NonNegativeTrimmed(" 5 "))
	require.Equal(t, 0, NonNegativeTrimmed("-5"))
}

func TestNonNegativeOr(t *testing.T) {
	v, err := NonNegativeOr("", 7)
	require.NoError(t, err)
	require.Equal(t, 7, v)
	v, err = NonNegativeOr(" 3 ", 7)
	require.NoError(t, err)
	require.Equal(t, 3, v)
	_, err = NonNegativeOr("-3", 7)
	require.Error(t, err)
}

func TestDate(t *testing.T) {
	got, err := Date("", nil)
	require.NoError(t, err)
	require.Nil(t, got)
	got, err = Date("2026-10-05", nil)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), *got)
	_, err = Date("05-10-2026", nil)
	require.Error(t, err)
}
