package masterrekening_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
)

func TestStatusKnownKeyAndAwaitingDecision(t *testing.T) {
	require.True(t, masterrekening.StatusPending.Known())
	require.True(t, masterrekening.StatusApproved.Known())
	require.True(t, masterrekening.StatusRejected.Known())
	require.False(t, masterrekening.ApprovalStatus("9").Known())

	acct := masterrekening.Account{Number: "1", BankCode: "014", Status: masterrekening.StatusPending}
	require.Equal(t, masterrekening.Key{Number: "1", BankCode: "014"}, acct.KeyOf())
	require.True(t, acct.AwaitingDecision())
	acct.Status = masterrekening.StatusApproved
	require.False(t, acct.AwaitingDecision())
}

func TestValidationErrorListsFieldsAlphabetically(t *testing.T) {
	err := masterrekening.Account{}.Check()
	require.Error(t, err)
	require.Equal(t,
		"masterrekening: isian tidak lengkap: alamat_bank, cabang_bank, email, kode_bank, "+
			"nama_bank, nama_pemilik, nik, nomor_rekening, tipe_rekening",
		err.Error())
}

func TestSubmitterEmailIsCheckedWhenGiven(t *testing.T) {
	acct := masterrekening.Account{
		Number: "1", OwnerName: "A", BankName: "B", BankBranch: "C", BankAddress: "D",
		BankCode: "E", AccountType: "F", Email: "a@contoh.co.id", NIK: "G",
		SubmitterEmail: "salah",
	}
	var validation *masterrekening.ValidationError
	require.ErrorAs(t, acct.Check(), &validation)
	require.Equal(t, map[string]string{
		"email_penginput": "Format email penginput tidak benar.",
	}, validation.Field)
}

func TestEmailShapeRules(t *testing.T) {
	require.True(t, masterrekening.EmailLooksValid(" a@b.co "))
	for _, bad := range []string{"@b.co", "a@", "a@b@c.co", "a@bco", "a@.co", "a@b."} {
		require.False(t, masterrekening.EmailLooksValid(bad), bad)
	}
}
