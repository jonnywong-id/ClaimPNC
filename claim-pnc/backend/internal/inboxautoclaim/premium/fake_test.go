package premium_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/premium"
)

func TestFakeCheckPremium(t *testing.T) {
	fake := premium.NewFake()
	fake.SetAging(" p1 ", "1500")
	fake.SetUnreachable("p2")
	ctx := context.Background()

	answer, err := fake.CheckPremium(ctx, "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P1"})
	require.NoError(t, err)
	require.Equal(t, "1500", answer.AgingAmount)

	answer, err = fake.CheckPremium(ctx, "ASM", inboxautoclaim.PremiumQuery{PolicyNo: "P9"})
	require.NoError(t, err)
	require.Equal(t, "0", answer.AgingAmount, "polis tanpa jawaban dianggap lunas")

	_, err = fake.CheckPremium(ctx, "ASM", inboxautoclaim.PremiumQuery{PolicyNo: " P2 "})
	require.EqualError(t, err, "inboxautoclaim/premium: layanan tiruan dibuat tidak dapat dihubungi")
	require.Equal(t, 3, fake.Calls())
}

func TestFakePremiumPaidBySource(t *testing.T) {
	fake := premium.NewFake()
	fake.SetPremiumPaid(" BRI ", " 01 ", "2500")
	ctx := context.Background()

	total, err := fake.PremiumPaidBySource(ctx, "ASM", inboxautoclaim.PremiumCheckQuery{BusinessCode: "01", SourceOfBusiness: "BRI"})
	require.NoError(t, err)
	require.Equal(t, "2500", total)

	total, err = fake.PremiumPaidBySource(ctx, "ASM", inboxautoclaim.PremiumCheckQuery{BusinessCode: "02", SourceOfBusiness: "BRI"})
	require.NoError(t, err)
	require.Equal(t, "0", total)

	fake.SetTotalUnreachable()
	_, err = fake.PremiumPaidBySource(ctx, "ASM", inboxautoclaim.PremiumCheckQuery{})
	require.Error(t, err)
	require.Equal(t, 3, fake.Calls())
}
