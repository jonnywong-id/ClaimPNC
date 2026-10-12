package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

func filterFor(investigated string) inboxinvestigator.ExportFilter {
	return inboxinvestigator.ExportFilter{
		From:         time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		To:           time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Investigated: investigated,
	}
}

// Dropdown "Pilih Investigation" benar-benar menyaring.
//
// Ia satu-satunya yang membedakan kedua isi berkas, dan repo memori yang mengabaikannya
// akan membuat uji penyaring selalu lulus tanpa membuktikan apa pun.
func TestExportFiltersByInvestigationChoice(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	sudah, err := repo.Export(ctx, filterFor(inboxinvestigator.InvestigatedYes))
	require.NoError(t, err)
	require.NotEmpty(t, sudah)
	for _, one := range sudah {
		require.Equal(t, inboxinvestigator.InvestigatedYes, one.Investigated)
	}

	belum, err := repo.Export(ctx, filterFor(inboxinvestigator.InvestigatedNo))
	require.NoError(t, err)
	require.Len(t, belum, 1)
	require.Equal(t, inboxinvestigator.InvestigatedNo, belum[0].Investigated)
}

// Hari "Sampai" IKUT terbawa seluruhnya, dan hari sesudahnya tidak.
func TestExportIncludesTheWholeUpperBoundDay(t *testing.T) {
	onTheDay := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)
	nextDay := time.Date(2026, 10, 1, 0, 1, 0, 0, time.UTC)

	repo := NewRepo(Options{Exports: []inboxinvestigator.ExportRow{
		{InvestigatedAt: &onTheDay, Investigated: inboxinvestigator.InvestigatedYes,
			Remarks: "ikut"},
		{InvestigatedAt: &nextDay, Investigated: inboxinvestigator.InvestigatedYes,
			Remarks: "di luar"},
	}})

	rows, err := repo.Export(context.Background(),
		filterFor(inboxinvestigator.InvestigatedYes))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "ikut", rows[0].Remarks)
}

// Baris di luar rentang bawah dibuang.
func TestExportDropsRowsBeforeTheLowerBound(t *testing.T) {
	tooEarly := time.Date(2026, 8, 31, 23, 59, 0, 0, time.UTC)

	repo := NewRepo(Options{Exports: []inboxinvestigator.ExportRow{
		{InvestigatedAt: &tooEarly, Investigated: inboxinvestigator.InvestigatedYes},
	}})

	rows, err := repo.Export(context.Background(),
		filterFor(inboxinvestigator.InvestigatedYes))
	require.NoError(t, err)
	require.Empty(t, rows)
}

// Baris tanpa tanggal investigasi tidak dapat masuk rentang mana pun, jadi dibuang.
func TestExportDropsRowsWithoutAnInvestigationDate(t *testing.T) {
	repo := NewRepo(Options{Exports: []inboxinvestigator.ExportRow{
		{Investigated: inboxinvestigator.InvestigatedYes},
	}})

	rows, err := repo.Export(context.Background(),
		filterFor(inboxinvestigator.InvestigatedYes))
	require.NoError(t, err)
	require.Empty(t, rows)
}

// Urutannya menurun menurut tanggal investigasi, sama seperti kueri SQL.
func TestExportIsOrderedNewestFirst(t *testing.T) {
	rows, err := NewSampleRepo().Export(context.Background(),
		filterFor(inboxinvestigator.InvestigatedYes))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2)

	for i := 1; i < len(rows); i++ {
		require.False(t, rows[i-1].InvestigatedAt.Before(*rows[i].InvestigatedAt),
			"baris %d lebih tua daripada baris sesudahnya", i-1)
	}
}

// Penyaring yang tidak sah ditolak, dan galatnya dapat dikenali lapisan transport.
func TestExportRejectsAnInvalidFilter(t *testing.T) {
	_, err := NewSampleRepo().Export(context.Background(),
		inboxinvestigator.ExportFilter{})
	require.ErrorIs(t, err, inboxinvestigator.ErrExportFilterInvalid)
}
