package committee_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/committee"
)

func TestKomiteTetapMengembalikanIdentitasTerpangkas(t *testing.T) {
	got, err := committee.Fixed{Identity: " KOMITEUJI "}.Resolve(
		context.Background(), "ASM", mastersurveyors.Surveyor{})
	require.NoError(t, err)
	require.Equal(t, "KOMITEUJI", got)

	empty, err := committee.Fixed{}.Resolve(context.Background(), "ASM", mastersurveyors.Surveyor{})
	require.NoError(t, err)
	require.Empty(t, empty)
}
