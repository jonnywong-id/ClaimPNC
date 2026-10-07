package sqlfile

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

const sample = `-- kepala berkas, diabaikan
-- name: pertama
-- komentar di atas kueri
SELECT 1
  FROM DUAL -- komentar sebaris tetap ada
-- komentar di tengah
 WHERE 1 = 1
-- komentar di ujung

-- name: kosong
-- hanya komentar

-- name: kedua
SELECT 2 FROM DUAL
`

func TestSplitDefaultDropsCommentsAndEmpty(t *testing.T) {
	got := Split(sample)
	require.Equal(t, map[string]string{
		"pertama": "SELECT 1\n  FROM DUAL -- komentar sebaris tetap ada\n WHERE 1 = 1",
		"kedua":   "SELECT 2 FROM DUAL",
	}, got)
}

func TestSplitKeepComments(t *testing.T) {
	got := Split(sample, KeepComments())
	require.Equal(t, "-- komentar di atas kueri\nSELECT 1\n  FROM DUAL -- komentar sebaris tetap ada\n-- komentar di tengah\n WHERE 1 = 1\n-- komentar di ujung", got["pertama"])
	require.Equal(t, "-- hanya komentar", got["kosong"])
}

func TestSplitKeepCommentsStripTrailing(t *testing.T) {
	got := Split(sample, KeepComments(), StripTrailingComments())
	require.Equal(t, "-- komentar di atas kueri\nSELECT 1\n  FROM DUAL -- komentar sebaris tetap ada\n-- komentar di tengah\n WHERE 1 = 1", got["pertama"])
	_, ada := got["kosong"]
	require.False(t, ada, "badan yang hanya berisi komentar menjadi kosong dan dilewati")
}

func TestSplitKeepEmpty(t *testing.T) {
	got := Split(sample, KeepEmpty())
	text, ada := got["kosong"]
	require.True(t, ada)
	require.Empty(t, text)
}

func TestSplitHandlesCRLF(t *testing.T) {
	got := Split("-- name: a\r\nSELECT 1\r\n-- c\r\n")
	require.Equal(t, "SELECT 1", got["a"])
}

func TestMustLoadAndGet(t *testing.T) {
	files := fstest.MapFS{
		"a.sql": {Data: []byte("-- name: satu\nSELECT 1\n")},
		"b.sql": {Data: []byte("-- name: dua\nSELECT 2\n")},
	}
	q := MustLoad(files, "uji/sqlstore")
	require.Equal(t, "SELECT 1", MustGet(q, "uji/sqlstore", "satu"))
	require.Equal(t, "SELECT 2", MustGet(q, "uji/sqlstore", "dua"))
	require.PanicsWithValue(t, `uji/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`, func() {
		MustGet(q, "uji/sqlstore", "tidak_ada")
	})
}

func TestMustLoadRejectsDuplicateName(t *testing.T) {
	files := fstest.MapFS{
		"a.sql": {Data: []byte("-- name: sama\nSELECT 1\n")},
		"b.sql": {Data: []byte("-- name: sama\nSELECT 2\n")},
	}
	require.PanicsWithValue(t, "uji/sqlstore: nama kueri ganda: sama", func() { MustLoad(files, "uji/sqlstore") })
}
