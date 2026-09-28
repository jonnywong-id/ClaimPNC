package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tugas Choose Surveyor milik PIC Teknik yang dipilih router; pengguna bergrup
// PNCKomiteTeknik (dianggap PIC Teknik) boleh mengerjakannya dan melihatnya di Inbox,
// pengguna tanpa grup itu tidak.
func TestTechnicPICGroupMayWorkOthersTask(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	require.NotEqual(t, "PICGRUP", task.Owner)

	in := registrasi.SettlementInput{
		PaymentType: registrasi.PaymentInterim, Propose: registrasi.Rupiah(1_000_000),
		Submitted: registrasi.Rupiah(1_000_000), RiskType: registrasi.RiskOther,
	}
	outsider := usecase.Caller{Identity: "PICGRUP"}
	_, err := l.service.AddSettlement(ctx, addSettlement(task, in), outsider)
	require.ErrorIs(t, err, registrasi.ErrNotTaskOwner)

	l.groups["picgrup"] = []string{"PNCKomiteTeknik"}
	member, err := l.service.ResolveCaller(ctx, outsider)
	require.NoError(t, err)
	require.Equal(t, []string{registrasi.RoleTechnicPIC}, member.Roles)

	inbox, err := l.service.Inbox(ctx, member)
	require.NoError(t, err)
	require.Contains(t, taskIDs(inbox), task.ID)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, in), member)
	require.NoError(t, err)

	// Grup admin tidak memberi hak atas tahap PIC Teknik.
	l.groups["adminsaja"] = []string{"PncAdmin"}
	admin, err := l.service.ResolveCaller(ctx, usecase.Caller{Identity: "ADMINSAJA"})
	require.NoError(t, err)
	_, err = l.service.AddSettlement(ctx, addSettlement(task, in), admin)
	require.ErrorIs(t, err, registrasi.ErrNotTaskOwner)
}

func taskIDs(tasks []registrasi.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	return ids
}
