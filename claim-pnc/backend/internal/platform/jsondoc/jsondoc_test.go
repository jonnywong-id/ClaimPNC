package jsondoc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndLookup(t *testing.T) {
	doc, err := Parse("  ")
	require.NoError(t, err)
	require.Equal(t, Document{}, doc)
	_, err = Parse("{")
	require.ErrorContains(t, err, "mengurai dokumen klaim")

	doc, err = Parse(`{"A": {"B": 1234567890123.45}, "C": "x"}`)
	require.NoError(t, err)
	v, ok := doc.Lookup("A.B")
	require.True(t, ok)
	require.Equal(t, "1234567890123.45", Text(v))
	_, ok = doc.Lookup("")
	require.False(t, ok)
	_, ok = doc.Lookup("C.D")
	require.False(t, ok)
	_, ok = doc.Lookup("A.Z")
	require.False(t, ok)
}

func TestText(t *testing.T) {
	require.Equal(t, "", Text(nil))
	require.Equal(t, "a", Text("a"))
	require.Equal(t, "5", Text(json.Number("5")))
	require.Equal(t, "true", Text(true))
	require.Equal(t, "", Text(map[string]any{}))
}

func TestRows(t *testing.T) {
	cols := []Column{{Key: "k", Path: "P"}, {Key: "none"}}
	require.Equal(t, []map[string]string{{"k": "1"}}, Rows([]any{map[string]any{"P": "1"}, "skip"}, cols))
	require.Equal(t, []map[string]string{{"k": ""}}, Rows(map[string]any{}, cols))
	require.Nil(t, Rows("x", cols))
}
