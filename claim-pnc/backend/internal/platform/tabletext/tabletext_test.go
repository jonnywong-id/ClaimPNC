package tabletext

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type source string

type column struct {
	Key     string
	Title   string
	Source  source
	Blocked bool
	Width   int
	hidden  string
	Note    []string
}

func TestRowsDecodesTable(t *testing.T) {
	got := Rows[column](`
		Key | Title     | Source   | Blocked | Width
		// catatan baris pertama
		id  | Treaty ID | tersedia |         | 12

		ri  | R/I Type  |          | true    |
	`)
	require.Equal(t, []column{
		{Key: "id", Title: "Treaty ID", Source: "tersedia", Width: 12},
		{Key: "ri", Title: "R/I Type", Blocked: true},
	}, got)
	require.Equal(t, []column{}, Rows[column]("Key"))
}

func TestRowsPanicsOnBadTable(t *testing.T) {
	require.Panics(t, func() { Rows[int]("Key") })
	require.Panics(t, func() { Rows[column]("Nope\nx") })
	require.Panics(t, func() { Rows[column]("hidden\nx") })
	require.Panics(t, func() { Rows[column]("Key | Title\nx") })
	require.Panics(t, func() { Rows[column]("Blocked\nya") })
	require.Panics(t, func() { Rows[column]("Width\nabc") })
	require.Panics(t, func() { Rows[column]("Note\nx") })
}
