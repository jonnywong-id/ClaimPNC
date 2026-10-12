package mastersurveyors_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
)

func TestAwaitingDecisionHanyaUntukMenunggu(t *testing.T) {
	require.True(t, mastersurveyors.Surveyor{Status: mastersurveyors.StatusPending}.AwaitingDecision())
	require.False(t, mastersurveyors.Surveyor{Status: mastersurveyors.StatusApproved}.AwaitingDecision())
}

// Pesan galat menyebut field dalam urutan abjad supaya stabil antarpemanggilan.
func TestValidationErrorMengurutkanField(t *testing.T) {
	err := mastersurveyors.Surveyor{TypeCode: mastersurveyors.InternalTypeCode}.Check()
	require.EqualError(t, err,
		"mastersurveyors: isian tidak lengkap: email, login_aplikasi, nama")
}

func TestEmailLooksValid(t *testing.T) {
	require.True(t, mastersurveyors.EmailLooksValid(" a@b.c "))
	require.False(t, mastersurveyors.EmailLooksValid("@b.c"))
	require.False(t, mastersurveyors.EmailLooksValid("a@"))
	require.False(t, mastersurveyors.EmailLooksValid("a@b@c.d"))
	require.False(t, mastersurveyors.EmailLooksValid("a@.c"))
	require.False(t, mastersurveyors.EmailLooksValid("a@bc."))
}
