package dashboardclaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
)

// Keempat tile berurutan dan hasilnya salinan, bukan senarai aslinya.
func TestTilesOrderAndCopy(t *testing.T) {
	got := dashboardclaim.Tiles()
	require.Equal(t, []dashboardclaim.Tile{
		dashboardclaim.TileOutstanding, dashboardclaim.TileCloseClaim,
		dashboardclaim.TileLossAdjuster, dashboardclaim.TileInternalSurveyor,
	}, got)

	got[0] = "diubah"
	require.Equal(t, dashboardclaim.TileOutstanding, dashboardclaim.Tiles()[0])
}

func TestTileLabelAndShape(t *testing.T) {
	cases := map[dashboardclaim.Tile]struct {
		label string
		shape dashboardclaim.RowShape
	}{
		dashboardclaim.TileOutstanding:      {"Outstanding", dashboardclaim.ShapeClaim},
		dashboardclaim.TileCloseClaim:       {"Close Claim", dashboardclaim.ShapeClaim},
		dashboardclaim.TileLossAdjuster:     {"Loss Adjuster", dashboardclaim.ShapeSurvey},
		dashboardclaim.TileInternalSurveyor: {"Internal Surveyor", dashboardclaim.ShapeSurvey},
		"lain":                              {"lain", dashboardclaim.ShapeClaim},
	}
	for tile, want := range cases {
		require.Equal(t, want.label, tile.Label(), tile)
		require.Equal(t, want.shape, tile.Shape(), tile)
	}
}

func TestParseTile(t *testing.T) {
	tile, ok := dashboardclaim.ParseTile("  Loss-Adjuster ")
	require.True(t, ok)
	require.Equal(t, dashboardclaim.TileLossAdjuster, tile)

	tile, ok = dashboardclaim.ParseTile("tidak-ada")
	require.False(t, ok)
	require.Equal(t, dashboardclaim.Tile(""), tile)
}

func TestSurveyorTypeFor(t *testing.T) {
	kind, ok := dashboardclaim.SurveyorTypeFor(dashboardclaim.TileInternalSurveyor)
	require.True(t, ok)
	require.Equal(t, dashboardclaim.SurveyorInternal, kind)

	kind, ok = dashboardclaim.SurveyorTypeFor(dashboardclaim.TileLossAdjuster)
	require.True(t, ok)
	require.Equal(t, dashboardclaim.SurveyorAdjuster, kind)

	_, ok = dashboardclaim.SurveyorTypeFor(dashboardclaim.TileOutstanding)
	require.False(t, ok)
}

func TestValidationError(t *testing.T) {
	require.NoError(t, dashboardclaim.NewValidationError(nil))

	err := dashboardclaim.NewValidationError([]dashboardclaim.Violation{
		{Field: dashboardclaim.FieldPage, Message: "bukan angka"},
		{Field: dashboardclaim.FieldSize, Message: "negatif"},
	})
	require.EqualError(t, err, "dashboardclaim: halaman: bukan angka; ukuran: negatif")

	var empty *dashboardclaim.ValidationError
	require.Equal(t, "dashboardclaim: permintaan tidak valid", empty.Error())
	require.Equal(t, "dashboardclaim: permintaan tidak valid",
		(&dashboardclaim.ValidationError{}).Error())
}

func TestBusinessLinesAndLabels(t *testing.T) {
	lines := dashboardclaim.BusinessLines()
	require.Len(t, lines, 5)
	lines[0] = "diubah"
	require.Equal(t, dashboardclaim.BusinessAll, dashboardclaim.BusinessLines()[0])

	labels := map[dashboardclaim.BusinessLine]string{
		dashboardclaim.BusinessAll:     "Semua Lini Bisnis",
		dashboardclaim.BusinessNonMBU:  "Non-MBU",
		dashboardclaim.BusinessBonding: "Bonding",
		dashboardclaim.BusinessPA:      "Personal Accident",
		dashboardclaim.BusinessTravel:  "Travel",
		"LAIN":                         "LAIN",
	}
	for line, label := range labels {
		require.Equal(t, label, line.Label())
	}
}

func TestParseBusinessLine(t *testing.T) {
	line, ok := dashboardclaim.ParseBusinessLine("  ")
	require.True(t, ok)
	require.Equal(t, dashboardclaim.BusinessAll, line)

	line, ok = dashboardclaim.ParseBusinessLine("travel")
	require.True(t, ok)
	require.Equal(t, dashboardclaim.BusinessTravel, line)

	_, ok = dashboardclaim.ParseBusinessLine("MBU")
	require.False(t, ok)
}

func TestFilterNormalizeAndCount(t *testing.T) {
	require.Equal(t,
		dashboardclaim.Filter{Business: dashboardclaim.BusinessAll, Search: "abc", Limit: 25},
		dashboardclaim.Filter{Search: "  abc ", Limit: 0, Offset: -4}.Normalize())
	require.Equal(t, dashboardclaim.MaxLimit,
		dashboardclaim.Filter{Limit: 999}.Normalize().Limit)
	require.Equal(t, 7, dashboardclaim.Filter{Business: dashboardclaim.BusinessPA, Limit: 10, Offset: 7}.
		Normalize().Offset)

	counted := dashboardclaim.Filter{Business: dashboardclaim.BusinessPA, Search: "x", Limit: 10, Offset: 20}.
		CountFilter()
	require.Equal(t, dashboardclaim.Filter{Business: dashboardclaim.BusinessPA, Search: "x"}, counted)
}
