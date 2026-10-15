package money_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/money"
)

// be4Stringer meniru tipe angka milik driver yang berperilaku seperti string.
type be4Stringer string

func (s be4Stringer) String() string { return string(s) }

// TestFromRupiahPanicsBeyondLimit membuktikan konstanta berlebih adalah cacat yang terlihat.
func TestFromRupiahPanicsBeyondLimit(t *testing.T) {
	require.Equal(t, int64(100), money.FromRupiah(1).MinorUnits())
	require.Panics(t, func() { money.FromRupiah(math.MaxInt64 / 10) })
	require.Panics(t, func() { money.FromRupiah(math.MinInt64 / 10) })
}

// TestStringHandlesMinInt64 membuktikan nilai terkecil tidak membalik tanda.
func TestStringHandlesMinInt64(t *testing.T) {
	require.Equal(t, "-92233720368547758.08", money.FromMinorUnits(math.MinInt64).String())
	require.Equal(t, "-0.05", money.FromMinorUnits(-5).String())
}

// TestParseRejectsTooLargeWholePart membuktikan bagian utuh yang melampaui batas ditolak.
func TestParseRejectsTooLargeWholePart(t *testing.T) {
	_, err := money.Parse("99999999999999999999")
	require.ErrorIs(t, err, money.ErrFormat)
	require.Contains(t, err.Error(), "terlalu besar")

	_, err = money.Parse("92233720368547759")
	require.ErrorIs(t, err, money.ErrFormat)
	require.Contains(t, err.Error(), "terlalu besar")
}

// TestFromSQLValueOtherShapes memeriksa int, Stringer driver, tipe asing, dan NaN.
func TestFromSQLValueOtherShapes(t *testing.T) {
	got, err := money.FromSQLValue(int(42))
	require.NoError(t, err)
	require.Equal(t, money.FromRupiah(42), got)

	got, err = money.FromSQLValue(be4Stringer("12.50"))
	require.NoError(t, err)
	require.Equal(t, money.FromMinorUnits(1250), got)

	_, err = money.FromSQLValue(true)
	require.ErrorIs(t, err, money.ErrFormat)
	require.Contains(t, err.Error(), "tipe bool tidak dikenali")

	_, err = money.FromSQLValue(math.NaN())
	require.ErrorIs(t, err, money.ErrFormat)
	_, err = money.FromSQLValue(math.Inf(-1))
	require.ErrorIs(t, err, money.ErrFormat)
	_, err = money.FromSQLValue(-float64(1 << 53))
	require.ErrorIs(t, err, money.ErrFormat)

	_, err = money.FromSQLValue(int64(math.MinInt64 / 10))
	require.ErrorIs(t, err, money.ErrFormat)
}
