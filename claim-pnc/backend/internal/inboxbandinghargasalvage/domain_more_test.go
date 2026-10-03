package inboxbandinghargasalvage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Keputusan tanpa identitas pemanggil ditolak sebelum isiannya diperiksa.
func TestDecisionCommandWithoutACallerIsRejected(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{DetailObject: "D", SalvageID: "1"},
		inboxbandinghargasalvage.Caller{Login: "  "})
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)
}

// Approved membedakan persetujuan dari penolakan.
func TestApprovedFollowsTheStatus(t *testing.T) {
	approve, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{
			DetailObject: "D", SalvageID: "1", RequestPrice: "10", Approve: true},
		inboxbandinghargasalvage.Caller{Login: "KOMITE"})
	require.NoError(t, err)
	require.True(t, approve.Approved())

	reject, err := inboxbandinghargasalvage.NewDecisionCommand(
		inboxbandinghargasalvage.DecisionInput{DetailObject: "D", SalvageID: "1"},
		inboxbandinghargasalvage.Caller{Login: "KOMITE"})
	require.NoError(t, err)
	require.False(t, reject.Approved())
}

// Kategori dokumen selalu konstanta yang sama.
func TestDocumentRowCategoryIsTheConstant(t *testing.T) {
	require.Equal(t, inboxbandinghargasalvage.DocumentCategory,
		inboxbandinghargasalvage.DocumentRow{ID: "1"}.Category())
}

// Permintaan dokumen menuntut identitas dan kedua id, dan membawa komite pemanggil.
func TestNewDocumentQuery(t *testing.T) {
	_, err := inboxbandinghargasalvage.NewDocumentQuery("D", "1",
		inboxbandinghargasalvage.Caller{})
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrCallerUnknown)

	_, err = inboxbandinghargasalvage.NewDocumentQuery(" ", " ",
		inboxbandinghargasalvage.Caller{Login: "KOMITE"})
	var validation *inboxbandinghargasalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
	require.Equal(t, inboxbandinghargasalvage.FieldDetailObject, validation.Violations[0].Field)
	require.Equal(t, inboxbandinghargasalvage.FieldSalvageID, validation.Violations[1].Field)

	q, err := inboxbandinghargasalvage.NewDocumentQuery(" D ", " 1 ",
		inboxbandinghargasalvage.Caller{Login: " KOMITE "})
	require.NoError(t, err)
	require.Equal(t, "D", q.DetailObject)
	require.Equal(t, "1", q.SalvageID)
	require.Equal(t, inboxbandinghargasalvage.ReviewerFor(
		inboxbandinghargasalvage.Caller{Login: "KOMITE"}), q.Reviewer)
}

// Pesan log galat validasi.
func TestValidationErrorMessage(t *testing.T) {
	require.Equal(t, "inboxbandinghargasalvage: isian tidak sah",
		inboxbandinghargasalvage.NewValidationError(nil).Error())
	require.Equal(t, "inboxbandinghargasalvage: a: satu; b: dua",
		inboxbandinghargasalvage.NewValidationError([]inboxbandinghargasalvage.Violation{
			{Field: "a", Message: "satu"}, {Field: "b", Message: "dua"},
		}).Error())
}

// Offset dihitung dari paginasi yang sudah dibetulkan.
func TestPaginationOffset(t *testing.T) {
	require.Equal(t, 0, inboxbandinghargasalvage.Pagination{}.Offset())
	require.Equal(t, 40, inboxbandinghargasalvage.Pagination{Page: 3, Size: 20}.Offset())
}
