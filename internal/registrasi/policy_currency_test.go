package registrasi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFillPolicyCurrencyOnlyWhenPolicyHasNone(t *testing.T) {
	k := Claim{Currency: " 10026 "}
	k.FillPolicyCurrency()
	require.Equal(t, "10026", k.Policy.Currency, "polis tanpa mata uang memakai mata uang klaim")

	k = Claim{Currency: "10026", Policy: Policy{Currency: "10001"}}
	k.FillPolicyCurrency()
	require.Equal(t, "10001", k.Policy.Currency, "mata uang polis yang terisi tidak ditimpa")

	k = Claim{}
	k.FillPolicyCurrency()
	require.Empty(t, k.Policy.Currency)
}
