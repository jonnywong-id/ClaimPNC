package sqlstore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
)

func TestGalatSambunganDBLinkDitandai(t *testing.T) {
	asli := errors.New("ORA-12541: TNS:no listener\n error occur at position: 54")
	err := linkError(asli)
	require.ErrorIs(t, err, dokumenpenunjang.ErrLinkTakTerjangkau)
	require.ErrorIs(t, err, asli)
}

func TestGalatKueriBiasaTidakDitandai(t *testing.T) {
	err := linkError(errors.New("ORA-00942: table or view does not exist"))
	require.NotErrorIs(t, err, dokumenpenunjang.ErrLinkTakTerjangkau)
}
