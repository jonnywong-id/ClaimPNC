package komitelink_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	komitememory "claim-pnc/internal/komite/repo/memory"
	komiteusecase "claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/komitelink"
)

// Penjenjangan registrasi meneruskan hitungan modul komite dan memangkas pengenal penyetuju.
func TestTieringRoutesThroughCommitteeModule(t *testing.T) {
	service, err := komiteusecase.NewService(komiteusecase.Options{Repo: komitememory.NewSampleRepo()})
	require.NoError(t, err)
	tiers := komitelink.New(service)

	route, err := tiers.Route(context.Background(), "NONMBU", registrasi.Rupiah(2_000_000_000), "")
	require.NoError(t, err)
	require.NotEmpty(t, route.Approvers)
	for _, a := range route.Approvers {
		require.Equal(t, strings.TrimSpace(a.OperatorID), a.OperatorID)
		require.NotEmpty(t, a.OperatorID)
	}

	_, err = tiers.Route(context.Background(), "LINI-TIDAK-ADA", registrasi.Rupiah(1), "")
	require.Error(t, err)
}
