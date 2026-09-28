package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi/usecase"
)

// Klaim yang baru dibuka sudah membawa objek, coverage, dan spreading polisnya
// (CallActivityInputRegister), dan ketiganya tersimpan bersama klaim.
func TestStartFillsItemsFromPolicy(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)

	require.Len(t, start.Claim.InsuredItem, 1)
	item := start.Claim.InsuredItem[0]
	require.Len(t, item.Coverage, 1)
	require.Equal(t, "FLEXAS", item.Coverage[0].Name)
	require.Len(t, item.Coverage[0].Spreading, 1)
	require.Equal(t, 1, start.Claim.SpreadingCount())

	saved, err := l.service.ViewClaim(ctx, start.Claim.ID, l.caller)
	require.NoError(t, err)
	require.Equal(t, start.Claim.InsuredItem, saved.Claim.InsuredItem)
}
