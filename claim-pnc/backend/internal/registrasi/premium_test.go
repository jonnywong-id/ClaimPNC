package registrasi_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Tabel keputusan GetStatusPremi langkah 7–9: hanya cicilan pertama yang dinilai.
func TestPremiumStatusFollowsGetStatusPremi(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 29, 3, 0, 0, 0, time.UTC)
	past, future := now.AddDate(0, -1, 0), now.AddDate(0, 1, 0)
	rat := func(n int64) *big.Rat { return big.NewRat(n, 1) }

	cases := []struct {
		name string
		s    registrasi.PremiumStatement
		want string
	}{
		{"aging di bawah 1", registrasi.PremiumStatement{AgingAmount: big.NewRat(1, 2)}, registrasi.PremiumPaid},
		{"cicilan pertama belum jatuh tempo", registrasi.PremiumStatement{AgingAmount: rat(5), Installments: []registrasi.PremiumInstallment{{DueDate: future}}}, registrasi.PremiumPaid},
		{"cicilan pertama sudah dibayar", registrasi.PremiumStatement{AgingAmount: rat(5), Installments: []registrasi.PremiumInstallment{{DueDate: past, PaymentDate: "20260801", PaymentAmount: "100"}}}, registrasi.PremiumPaid},
		{"jatuh tempo belum dibayar, aging > 1", registrasi.PremiumStatement{AgingAmount: rat(5), Installments: []registrasi.PremiumInstallment{{DueDate: past}}}, registrasi.PremiumUnpaid},
		{"jatuh tempo belum dibayar, aging tepat 1", registrasi.PremiumStatement{AgingAmount: rat(1), Installments: []registrasi.PremiumInstallment{{DueDate: past}}}, registrasi.PremiumPaid},
		{"jatuh tempo belum dibayar, aging kosong", registrasi.PremiumStatement{Installments: []registrasi.PremiumInstallment{{DueDate: past}}}, registrasi.PremiumUnpaid},
		{"tanpa cicilan, aging > 1", registrasi.PremiumStatement{AgingAmount: rat(3)}, registrasi.PremiumUnpaid},
		{"tanpa cicilan, aging kosong", registrasi.PremiumStatement{}, ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, registrasi.PremiumStatusOf(c.s, now), c.name)
	}
}

func TestPremiumBlocksHonoursExemptions(t *testing.T) {
	t.Parallel()
	final := registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, Value: registrasi.Rupiah(1_000_000)}
	require.False(t, registrasi.PremiumBlocks(final, registrasi.PremiumPaid, registrasi.PremiumStatement{}, registrasi.PremiumExemption{}))
	require.True(t, registrasi.PremiumBlocks(final, registrasi.PremiumUnpaid, registrasi.PremiumStatement{}, registrasi.PremiumExemption{}))
	require.False(t, registrasi.PremiumBlocks(final, registrasi.PremiumUnpaid, registrasi.PremiumStatement{}, registrasi.PremiumExemption{OpenProtection: true}))
	require.False(t, registrasi.PremiumBlocks(final, registrasi.PremiumUnpaid, registrasi.PremiumStatement{}, registrasi.PremiumExemption{KBRU: true}))

	fee := final
	fee.PaymentType = registrasi.PaymentAdjusterFee
	require.False(t, registrasi.PremiumBlocks(fee, registrasi.PremiumUnpaid, registrasi.PremiumStatement{}, registrasi.PremiumExemption{}), "tipe 4 lolos")

	paid := registrasi.PremiumStatement{Installments: []registrasi.PremiumInstallment{{PaymentAmount: "1000000"}}}
	require.False(t, registrasi.PremiumBlocks(final, registrasi.PremiumUnpaid, paid, registrasi.PremiumExemption{TravelClientName: "DIRECT"}))
	short := registrasi.PremiumStatement{Installments: []registrasi.PremiumInstallment{{PaymentAmount: "999999"}}}
	require.True(t, registrasi.PremiumBlocks(final, registrasi.PremiumUnpaid, short, registrasi.PremiumExemption{TravelClientName: "DIRECT"}))
}
