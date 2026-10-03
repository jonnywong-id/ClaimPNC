package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
)

func TestListSortedByCodeAndGet(t *testing.T) {
	repo := NewRepo(
		masterstatus.ClaimStatus{Code: " 1147 ", Label: " Register "},
		masterstatus.ClaimStatus{Code: "1134", Label: "Open", LegacyCode: "01"},
	)
	ctx := context.Background()
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"1134", "1147"}, []string{list[0].Code, list[1].Code})
	require.Equal(t, "Register", list[1].Label)

	got, err := repo.Get(ctx, " 1134 ")
	require.NoError(t, err)
	require.Equal(t, "01", got.LegacyCode)
	_, err = repo.Get(ctx, "9999")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
}

func TestInsertContinuesHighestSequenceAndRejectsDuplicateLabel(t *testing.T) {
	repo := NewRepo(masterstatus.ClaimStatus{Code: "1166", Label: "Paid"}, masterstatus.ClaimStatus{Code: "X1", Label: "Lain"})
	ctx := context.Background()

	_, err := repo.Insert(ctx, " paid ")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)

	created, err := repo.Insert(ctx, " Baru ")
	require.NoError(t, err)
	require.Equal(t, masterstatus.ClaimStatus{Code: "1167", Label: "Baru"}, created)
}

func TestInsertRejectsTakenCode(t *testing.T) {
	repo := NewRepo(masterstatus.ClaimStatus{Code: "1005", Label: "Lima"})
	// Nomor urut sengaja tertinggal dari isi supaya kode berikutnya bentrok.
	repo.order = 4
	_, err := repo.Insert(context.Background(), "Baru")
	require.ErrorIs(t, err, masterstatus.ErrCodeTaken)
}

func TestUpdate(t *testing.T) {
	repo := NewRepo(masterstatus.ClaimStatus{Code: "1001", Label: "Satu"}, masterstatus.ClaimStatus{Code: "1002", Label: "Dua"})
	ctx := context.Background()

	updated, err := repo.Update(ctx, " 1001 ", " satu ")
	require.NoError(t, err, "label milik baris itu sendiri boleh dipakai ulang")
	require.Equal(t, "satu", updated.Label)

	_, err = repo.Update(ctx, "1001", "DUA")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)
	_, err = repo.Update(ctx, "9999", "x")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := NewRepo(SampleList()...)
	failure := errors.New("rusak")
	repo.SetError(failure)
	ctx := context.Background()
	_, err := repo.List(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "1134")
	require.ErrorIs(t, err, failure)
	_, err = repo.Insert(ctx, "x")
	require.ErrorIs(t, err, failure)
	_, err = repo.Update(ctx, "1134", "x")
	require.ErrorIs(t, err, failure)
}

func TestThreeDigitsAndSequence(t *testing.T) {
	require.Equal(t, "000", threeDigits(0))
	require.Equal(t, "007", threeDigits(7))
	require.Equal(t, "1234", threeDigits(1234))
	require.Equal(t, 5, sequenceFromCode("1005", "1"))
	require.Equal(t, 0, sequenceFromCode("2005", "1"))
	require.Equal(t, 0, sequenceFromCode("1", "1"))
	require.Equal(t, 0, sequenceFromCode("10a5", "1"))
}

func TestSampleListIsLoadable(t *testing.T) {
	repo := NewRepo(SampleList()...)
	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, len(SampleList()))
}
