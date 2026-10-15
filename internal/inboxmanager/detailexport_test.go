package inboxmanager

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestKolomEksporDetailSamaDenganPega mengunci jumlah dan pasangan kolom berkas.
//
// Sembilan puluh pasang, dibaca dari `CSVPropHeaders` dan `CSVProperties` di
// `Activity/ExportDataDetailKlaim-Act.xml`. Properti boleh muncul hanya sekali — dua judul yang
// menunjuk properti yang sama berarti satu kolom tersalin dua kali dan satu kolom hilang.
func TestKolomEksporDetailSamaDenganPega(t *testing.T) {
	require.Len(t, DetailExportColumns, 90)

	seen := map[string]bool{}
	for _, column := range DetailExportColumns {
		require.NotEmpty(t, column.Title)
		require.Falsef(t, seen[column.Property], "properti %s muncul dua kali", column.Property)
		seen[column.Property] = true
	}

	require.Equal(t, DetailExportColumn{"No Klaim", "CaseID"}, DetailExportColumns[0])
	require.Equal(t, DetailExportColumn{" Kanwil", "ResponseMsg"}, DetailExportColumns[5],
		"spasi di depan judul dibawa apa adanya")
	require.Equal(t, DetailExportColumn{"Kanwil", "HidePelapor"}, DetailExportColumns[89])

	for property := range DetailExportEmptyColumns {
		require.Truef(t, seen[property], "kolom kosong %s bukan kolom berkas", property)
	}
}

func TestNewDetailExportQuery(t *testing.T) {
	caller := Caller{Login: "MGR", LineBusiness: "NONMBU"}

	t.Run("tanpa tanggal: seluruh periode", func(t *testing.T) {
		q, err := NewDetailExportQuery(DetailExportInput{}, caller, "ASM")
		require.NoError(t, err)
		require.False(t, q.HasPeriod())
		require.Equal(t, "NONMBU", q.LineBusiness)
		require.False(t, q.AllLines)
	})

	t.Run("rentang: batas atas eksklusif", func(t *testing.T) {
		q, err := NewDetailExportQuery(
			DetailExportInput{From: "2026-09-01", Until: " 2026-09-30 "}, caller, "ASM")
		require.NoError(t, err)
		require.True(t, q.HasPeriod())
		require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), q.From)
		require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), q.Until)
	})

	t.Run("SIMASNET melepas saringan lini", func(t *testing.T) {
		q, err := NewDetailExportQuery(DetailExportInput{}, caller, " asi ")
		require.NoError(t, err)
		require.True(t, q.AllLines)
	})

	rejected := map[string]DetailExportInput{
		"hanya Dari":           {From: "2026-09-01"},
		"hanya Sampai":         {Until: "2026-09-30"},
		"Dari tidak terbaca":   {From: "01/09/2026", Until: "2026-09-30"},
		"Sampai mendahului":    {From: "2026-09-30", Until: "2026-09-01"},
		"Sampai tidak terbaca": {From: "2026-09-01", Until: "kemarin"},
	}
	for name, input := range rejected {
		t.Run(name, func(t *testing.T) {
			_, err := NewDetailExportQuery(input, caller, "ASM")
			var validation *ValidationError
			require.Truef(t, errors.As(err, &validation), "galat %v", err)
			require.NotEmpty(t, validation.Violations)
			require.Equal(t, FieldPeriod, validation.Violations[0].Field)
		})
	}

	t.Run("tanpa login", func(t *testing.T) {
		_, err := NewDetailExportQuery(DetailExportInput{}, Caller{}, "ASM")
		require.ErrorIs(t, err, ErrCallerUnknown)
	})
}
