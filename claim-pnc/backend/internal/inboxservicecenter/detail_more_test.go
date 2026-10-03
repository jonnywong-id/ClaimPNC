package inboxservicecenter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
)

// Label status pada rincian memakai penerjemah yang sama dengan baris daftar.
func TestClaimDetailStatusNames(t *testing.T) {
	d := inboxservicecenter.ClaimDetail{RepairStatus: "7", ApprovalStatus: inboxservicecenter.ApprovalApproved}
	require.Equal(t, "Repair Completed", d.RepairStatusName())
	require.Equal(t, "APPROVED", d.ApprovalStatusName())

	kosong := inboxservicecenter.ClaimDetail{}
	require.Equal(t, "", kosong.RepairStatusName())
	require.Equal(t, "Belum diajukan", kosong.ApprovalStatusName())
}

// Urutan dan kode ketujuh kelompok rincian dikunci apa adanya.
func TestDetailGroupsOrder(t *testing.T) {
	groups := inboxservicecenter.DetailGroups()
	codes := make([]string, 0, len(groups))
	for _, g := range groups {
		codes = append(codes, g.Code)
	}
	require.Equal(t, []string{
		inboxservicecenter.GroupGeneral,
		inboxservicecenter.GroupUnit,
		inboxservicecenter.GroupRepair,
		inboxservicecenter.GroupDates,
		inboxservicecenter.GroupCost,
		inboxservicecenter.GroupCostApproved,
		inboxservicecenter.GroupAccessories,
	}, codes)
	require.Equal(t, "General Information", groups[0].Title)
	require.Equal(t, "id", groups[0].Fields[0].Key)
}

func TestNewDetailQuery(t *testing.T) {
	t.Run("ID dan login dipangkas", func(t *testing.T) {
		q, err := inboxservicecenter.NewDetailQuery("  SC-1  ", inboxservicecenter.Caller{Login: " pic "})
		require.NoError(t, err)
		require.Equal(t, "SC-1", q.ID)
		require.Equal(t, "pic", q.Caller.Login)
	})

	t.Run("tanpa login ditolak lebih dulu daripada ID kosong", func(t *testing.T) {
		_, err := inboxservicecenter.NewDetailQuery("", inboxservicecenter.Caller{Login: "  "})
		require.ErrorIs(t, err, inboxservicecenter.ErrCallerUnknown)
	})

	t.Run("ID kosong menjadi galat validasi", func(t *testing.T) {
		_, err := inboxservicecenter.NewDetailQuery(" ", inboxservicecenter.Caller{Login: "pic"})
		var validation *inboxservicecenter.ValidationError
		require.ErrorAs(t, err, &validation)
		require.Equal(t, []inboxservicecenter.Violation{{
			Field:   inboxservicecenter.FieldID,
			Message: "ID klaim wajib diisi.",
		}}, validation.Violations)
	})
}

func TestValidationErrorMessage(t *testing.T) {
	require.Equal(t, "inboxservicecenter: isian tidak sah",
		inboxservicecenter.NewValidationError(nil).Error())

	err := inboxservicecenter.NewValidationError([]inboxservicecenter.Violation{
		{Field: "tab", Message: "salah"},
		{Field: "cari", Message: "terlalu panjang"},
	})
	require.Equal(t, "inboxservicecenter: tab: salah; cari: terlalu panjang", err.Error())
}
