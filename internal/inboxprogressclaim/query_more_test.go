package inboxprogressclaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
)

func TestBusinessLineLabels(t *testing.T) {
	require.Equal(t, "Non-MBU", inboxprogressclaim.BusinessNonMBU.Label())
	require.Equal(t, "Travel", inboxprogressclaim.BusinessTravel.Label())
	require.Equal(t, "Bonding", inboxprogressclaim.BusinessBonding.Label())
	require.Equal(t, "Personal Accident", inboxprogressclaim.BusinessPA.Label())
	// Kode yang tidak dikenal ditampilkan apa adanya.
	require.Equal(t, "LAIN", inboxprogressclaim.BusinessLine("LAIN").Label())
}

func TestValidationErrorMessage(t *testing.T) {
	require.EqualError(t, inboxprogressclaim.NewValidationError(nil), "inboxprogressclaim: isian tidak sah")
	require.EqualError(t, inboxprogressclaim.NewValidationError([]inboxprogressclaim.Violation{
		{Field: "bisnis", Message: "Wajib."}, {Field: "dari", Message: "Salah."},
	}), "inboxprogressclaim: bisnis: Wajib.; dari: Salah.")
}

func TestPerPICRejectsUnknownBusinessAndUnreadableDates(t *testing.T) {
	_, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: "motor",
		From:     "21-09-2026",
		To:       "kemarin",
	}, inboxprogressclaim.Caller{Login: "ADMIN"})

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	fields := []string{}
	for _, violation := range validation.Violations {
		fields = append(fields, violation.Field)
	}
	require.Equal(t, []string{inboxprogressclaim.FieldBusiness, inboxprogressclaim.FieldFrom, inboxprogressclaim.FieldTo}, fields)
	require.Equal(t, "Pilihan lini bisnis tidak dikenal.", validation.Violations[0].Message)
}

func TestPerPICAcceptsValidFilters(t *testing.T) {
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: " pa ",
		From:     "2026-09-01",
		To:       "2026-09-30",
	}, inboxprogressclaim.Caller{Login: " ADMIN "})
	require.NoError(t, err)
	require.Equal(t, inboxprogressclaim.BusinessPA, query.Business)
	require.Equal(t, "ADMIN", query.Caller.Login)
	require.Equal(t, 1, query.From.Day())
	require.Equal(t, 30, query.To.Day())
}
