package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/repo/memory"
	"claim-pnc/internal/inboxrcl/usecase"
)

var decidedAt = time.Date(2026, 10, 5, 4, 0, 0, 0, time.UTC)

func decideService(t *testing.T) (*usecase.Service, *memory.Store) {
	t.Helper()
	store := memory.NewSampleStore()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrcl.Repo, error) { return store, nil },
	})
	require.NoError(t, err)
	return svc, store
}

func TestSetujuMengeluarkanKlaimDariAntreanDokter(t *testing.T) {
	svc, store := decideService(t)
	caller := inboxrcl.Caller{Login: memory.SampleLogin}

	out, err := svc.Decide(context.Background(), "ASM", caller, "PNCN.26.0412", "SETUJU", "", decidedAt)
	require.NoError(t, err)
	require.Equal(t, inboxrcl.StageRCLPUCL, out.NextStage)

	listed, err := svc.List(context.Background(), "ASM", caller, inboxrcl.Filter{})
	require.NoError(t, err)
	for _, task := range listed.Page.Tasks {
		require.NotEqual(t, "PNCN.26.0412", task.ClaimNumber)
	}

	decisions := store.Decisions()
	require.Len(t, decisions, 1)
	require.Equal(t, inboxrcl.WorkbasketRCLPUCL, decisions[0].Assignee)
	require.Equal(t, memory.SampleOperator, decisions[0].Operator)

	// Tombol yang sama ditekan lagi (tab lain): klaim sudah bukan di antrean.
	_, err = svc.Decide(context.Background(), "ASM", caller, "PNCN.26.0412", "SETUJU", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrClaimNotFound)
}

func TestTidakSetujuKePICTeknik(t *testing.T) {
	svc, store := decideService(t)

	_, err := svc.Decide(context.Background(), "ASM", inboxrcl.Caller{Login: memory.SampleLogin},
		"PNCN.26.0412", "TidakSetuju", "Diagnosa dijamin.", decidedAt)
	require.NoError(t, err)

	decisions := store.Decisions()
	require.Len(t, decisions, 1)
	require.Equal(t, memory.SampleTechnicalPIC, decisions[0].Assignee)
	require.Equal(t, "Diagnosa dijamin.", decisions[0].Outcome.DoctorReason)
}

func TestKeputusanDitolakBilaTidakSesuaiModeAtauTidakDikenal(t *testing.T) {
	svc, _ := decideService(t)
	caller := inboxrcl.Caller{Login: memory.SampleLogin}

	_, err := svc.Decide(context.Background(), "ASM", caller, "PNCN.26.0412", "MSIG", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrDecisionNotAllowed, "Submit tidak ada pada mode RCL")

	_, err = svc.Decide(context.Background(), "ASM", caller, "PNCN.26.0405", "SETUJU", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrDecisionNotAllowed, "Setuju tidak ada pada mode MSIG")

	_, err = svc.Decide(context.Background(), "ASM", caller, "PNCN.26.0412", "Setuju", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrUnknownDecision)
}

func TestKeputusanKlaimDokterLainDanLoginTidakAktifDitolak(t *testing.T) {
	svc, store := decideService(t)

	_, err := svc.Decide(context.Background(), "ASM", inboxrcl.Caller{Login: memory.SampleLogin},
		"PNCN.26.0350", "SETUJU", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrClaimNotFound)

	_, err = svc.Decide(context.Background(), "ASM", inboxrcl.Caller{Login: memory.SampleLoginInactive},
		"PNCN.26.0412", "SETUJU", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrClaimNotFound)

	_, err = svc.Decide(context.Background(), "ASM", inboxrcl.Caller{}, "PNCN.26.0412", "SETUJU", "", decidedAt)
	require.ErrorIs(t, err, inboxrcl.ErrCallerUnknown)

	require.Empty(t, store.Decisions())
}

func TestAlasanDokterDipotongSesuaiKolom(t *testing.T) {
	svc, store := decideService(t)

	long := strings.Repeat("é", usecase.MaxDoctorReasonLength) // 2 byte per karakter
	_, err := svc.Decide(context.Background(), "ASM", inboxrcl.Caller{Login: memory.SampleLogin},
		"PNCN.26.0405", "BackMSIG", long, decidedAt)
	require.NoError(t, err)

	reason := store.Decisions()[0].Outcome.DoctorReason
	require.LessOrEqual(t, len(reason), usecase.MaxDoctorReasonLength)
	require.Equal(t, usecase.MaxDoctorReasonLength/2, len([]rune(reason)), "tidak membelah karakter")
}
