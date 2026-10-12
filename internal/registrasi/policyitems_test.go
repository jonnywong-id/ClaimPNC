package registrasi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func percent(p int64) registrasi.Percent { return registrasi.Percent(p * 10_000) }

// Tabel sumber dipilih seperti GetListObjectFromDatabase: PA/Travel, Fire, Marine, dan
// Aneka sebagai sisanya.
func TestSourceOfFollowsGetListObjectFromDatabase(t *testing.T) {
	for _, tc := range []struct {
		line         registrasi.LineOfBusiness
		businessType string
		want         registrasi.PolicySource
	}{
		{registrasi.LinePersonalAccident, "PA", registrasi.SourcePerson},
		{registrasi.LineTravel, "Travel", registrasi.SourcePerson},
		{registrasi.LineFire, "Fire", registrasi.SourceProperty},
		{registrasi.LineMiscellaneous, "Fire", registrasi.SourceProperty},
		{registrasi.LineMarineCargo, "MarineCargo", registrasi.SourceCargo},
		{registrasi.LineMiscellaneous, "AllRisk", registrasi.SourceAneka},
	} {
		got := registrasi.SourceOf(registrasi.Policy{Line: tc.line, BusinessType: tc.businessType})
		require.Equal(t, tc.want, got, "lini %s / %s", tc.line, tc.businessType)
	}
}

func TestBuildInsuredItemsCarriesCoverageAndSpreading(t *testing.T) {
	policy := registrasi.Policy{
		Line:          registrasi.LineMiscellaneous,
		CoverageStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CoverageEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	source := []registrasi.SourceItem{
		{ID: "", Name: "tanpa id"}, // dibuang, seperti langkah 5 GetListObjectFromDatabase
		{ID: "1", Name: "Gedung", Location: "Jakarta", Coverage: []registrasi.SourceCoverage{
			{Code: "100014", Name: "All Risk", TSI: registrasi.Rupiah(100), TSISublimit: registrasi.Rupiah(40),
				Spreading: []registrasi.SourceSpreading{
					{TreatyType: "10001", Share: percent(30)},
					{TreatyType: "10007", Share: percent(50)},
					{TreatyType: "10001", Share: percent(20)}, // dijumlahkan dengan baris pertama
					{TreatyType: "10003", Share: 0},           // share nol dibuang
					{TreatyType: "10010", Share: percent(5), Deleted: true},
				}},
			{Code: "100015", Name: "Dihapus", Deleted: true},
			{Code: "100016", Name: "Tanpa TSI", SumTSI: registrasi.Rupiah(7)},
		}},
	}

	got := registrasi.BuildInsuredItems(policy, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), source)

	require.Len(t, got, 1)
	require.Equal(t, "1", got[0].ID)
	require.Equal(t, "Jakarta", got[0].Location)
	require.Len(t, got[0].Coverage, 2)

	first := got[0].Coverage[0]
	require.Equal(t, "All Risk", first.Name)
	require.Equal(t, registrasi.Rupiah(40), first.TSI, "TSISublimit didahulukan")
	require.Equal(t, []registrasi.Spreading{
		{TreatyKind: "10001", Share: percent(50)},
		{TreatyKind: "10007", Share: percent(50)},
	}, first.Spreading)

	require.Equal(t, registrasi.Rupiah(7), got[0].Coverage[1].TSI, "SumTSI bila TSI kosong")
}

// Coverage hanya disalin bila tanggal kejadian di dalam periode polis; objeknya tetap.
func TestBuildInsuredItemsSkipsCoverageOutsidePeriod(t *testing.T) {
	policy := registrasi.Policy{
		Line:          registrasi.LineMiscellaneous,
		CoverageStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CoverageEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	source := []registrasi.SourceItem{{ID: "1", Coverage: []registrasi.SourceCoverage{{Code: "1"}}}}

	outside := registrasi.BuildInsuredItems(policy, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), source)
	require.Len(t, outside, 1)
	require.Empty(t, outside[0].Coverage)

	unknown := registrasi.BuildInsuredItems(policy, time.Time{}, source)
	require.Len(t, unknown[0].Coverage, 1, "tanggal kejadian belum diketahui: coverage tetap diisi")
}

// Kargo Open Policy yang coverage-nya tanpa spreading memakai spreading coverage pertama
// objek pertama (SetspreadingtoCoverage langkah 5).
func TestOpenCargoBorrowsFirstSpreading(t *testing.T) {
	source := []registrasi.SourceItem{
		{ID: "1", Coverage: []registrasi.SourceCoverage{{Code: "A",
			Spreading: []registrasi.SourceSpreading{{TreatyType: "10007", Share: percent(100)}}}}},
		{ID: "2", Coverage: []registrasi.SourceCoverage{{Code: "B"}}},
	}

	open := registrasi.Policy{Line: registrasi.LineMarineCargo, Kind: "2"}
	got := registrasi.BuildInsuredItems(open, time.Time{}, source)
	require.Equal(t, "10007", got[1].Coverage[0].Spreading[0].TreatyKind)

	regular := registrasi.Policy{Line: registrasi.LineMarineCargo, Kind: "1"}
	got = registrasi.BuildInsuredItems(regular, time.Time{}, source)
	require.Empty(t, got[1].Coverage[0].Spreading)
}
