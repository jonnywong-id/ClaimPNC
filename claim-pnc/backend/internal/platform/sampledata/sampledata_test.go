package sampledata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type item struct {
	Code  string
	Count int
}

const raw = `{"items": [{"Code": "A", "Count": 2}], "names": {"x": "y"}}`

func TestMustDecodesTheNamedPart(t *testing.T) {
	require.Equal(t, []item{{Code: "A", Count: 2}}, Must[[]item]([]byte(raw), "items"))
	require.Equal(t, map[string]string{"x": "y"}, Must[map[string]string]([]byte(raw), "names"))
}

func TestMustPanicsOnBrokenData(t *testing.T) {
	require.PanicsWithValue(t, `sampledata: data contoh "lain" tidak ada`, func() { Must[[]item]([]byte(raw), "lain") })
	require.Panics(t, func() { Must[[]item]([]byte(`{`), "items") })
	require.Panics(t, func() { Must[[]item]([]byte(`{"items": [{"Kode": "A"}]}`), "items") })
}
