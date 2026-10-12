package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/repo/memory"
)

func TestStreamFillsEveryColumnOfThreeRows(t *testing.T) {
	repo := memory.NewRepo()
	report, err := reportklaim.Lookup(reportklaim.CodeKlaimHarian)
	require.NoError(t, err)
	filter := reportklaim.Filter{}
	columns := report.Columns(filter)
	require.NotEmpty(t, columns)

	var rows []reportklaim.Row
	require.NoError(t, repo.Stream(context.Background(), report, filter, func(row reportklaim.Row) error {
		rows = append(rows, row)
		return nil
	}))
	require.Len(t, rows, 3)
	for _, c := range columns {
		require.Contains(t, rows[0], c.Field)
	}
}

func TestStreamOverridesClaimAndPolicyNumbers(t *testing.T) {
	repo := memory.NewRepo()
	for _, report := range reportklaim.CatalogInPegaOrder() {
		columns := report.Columns(reportklaim.Filter{})
		hasCase, hasClaimNo := false, false
		for _, c := range columns {
			hasCase = hasCase || c.Field == "CaseID"
			hasClaimNo = hasClaimNo || c.Field == "ClaimNo"
		}
		if !hasCase || !hasClaimNo {
			continue
		}
		var first reportklaim.Row
		require.NoError(t, repo.Stream(context.Background(), report, reportklaim.Filter{}, func(row reportklaim.Row) error {
			if first == nil {
				first = row
			}
			return nil
		}))
		require.Equal(t, "PNCN.26.0001", first["CaseID"])
		require.Equal(t, "POLIS-CONTOH-0001", first["ClaimNo"])
		return
	}
	t.Fatal("katalog tidak memuat laporan berkolom CaseID dan ClaimNo")
}

func TestStreamStopsWhenEmitFails(t *testing.T) {
	report, err := reportklaim.Lookup(reportklaim.CodeKlaimHarian)
	require.NoError(t, err)
	stop := errors.New("berhenti")
	calls := 0
	err = memory.NewRepo().Stream(context.Background(), report, reportklaim.Filter{}, func(reportklaim.Row) error {
		calls++
		return stop
	})
	require.ErrorIs(t, err, stop)
	require.Equal(t, 1, calls)
}

func TestListBusinessOptions(t *testing.T) {
	options, err := memory.NewRepo().ListBusinessOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, options, 3)
	require.Equal(t, "10076", options[0].Code)
}

func TestSelectorServesOnlyItsPortal(t *testing.T) {
	selector := memory.Selector("ASM")
	repo, err := selector("ASM")
	require.NoError(t, err)
	require.NotNil(t, repo)
	again, _ := selector("ASM")
	require.Same(t, repo, again, "penyimpanan yang sama dipakai ulang")

	_, err = selector("SMAS")
	require.ErrorContains(t, err, `portal "SMAS" tidak tersedia`)
}
