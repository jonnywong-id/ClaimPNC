package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/repo/memory"
)

func TestDetailAppliesTheSameGuardsAsTheList(t *testing.T) {
	// Popup memuat nama tertanggung, kronologi, dan nilai uang. Pemeriksaannya tidak boleh
	// lebih longgar daripada layar yang membukanya.
	service := newService(t, memory.NewSampleStore())

	_, err := service.Detail(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{DetailBranchCode: "078"}, "PNC-9001")
	require.ErrorIs(t, err, inboxosclaimpercabang.ErrCallerUnknown)

	_, err = service.Detail(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{Login: "MITRA1"}, "PNC-9001")
	require.ErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)

	_, err = service.Detail(context.Background(), "SMI", callerAt("078"), "PNC-9001")
	require.Error(t, err, "portal yang tidak dikenal wajib gagal, bukan jatuh ke portal utama")
}

func TestDetailFillsAgingFromTheClock(t *testing.T) {
	// Popup menampilkan umur yang SAMA dengan barisnya di grid. Bila salah satunya dihitung
	// di tempat lain, keduanya akan berselisih dan pengguna tidak punya cara menjelaskannya.
	service := newService(t, memory.NewSampleStore())

	detailed, err := service.Detail(context.Background(), primaryPortal,
		callerAt("078"), "PNC-9001")
	require.NoError(t, err)

	// PNC-9001 terdaftar 15 Januari 2024; terhadap jam uji 28 September 2026 itu 987 hari.
	require.Equal(t, 987, detailed.Detail.AgingDays)
	require.Equal(t, "100099", detailed.Query.Branch.Code)
	require.Equal(t, "CILEGON", detailed.Query.Branch.Name)
}

func TestDetailRefusesAClaimOfAnotherBranch(t *testing.T) {
	// PNC-9004 ada, tetapi milik BANDUNG. Jawabannya harus ErrClaimNotFound — bukan isinya,
	// dan bukan pula galat yang menyatakan klaim itu ada di cabang lain (`R-20`).
	service := newService(t, memory.NewSampleStore())

	_, err := service.Detail(context.Background(), primaryPortal,
		callerAt("078"), "PNC-9004")

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrClaimNotFound)
}

func TestDetailCarriesItsOwnPlannedDifferences(t *testing.T) {
	// Daftar popup BERBEDA dari daftar grid: catatan tentang "Total Sum Insured" tidak
	// berlaku di grid, dan catatan tentang paginasi tidak berlaku di popup.
	service := newService(t, memory.NewSampleStore())

	detailed, err := service.Detail(context.Background(), primaryPortal,
		callerAt("078"), "PNC-9001")
	require.NoError(t, err)

	require.NotEmpty(t, detailed.PlannedDifferences)
	require.NotEqual(t, inboxosclaimpercabang.PlannedDifferences, detailed.PlannedDifferences)
}
