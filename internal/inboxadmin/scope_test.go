package inboxadmin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
)

func TestManagerGroupsIgnoreCaseAndPrefix(t *testing.T) {
	require.True(t, inboxadmin.IsManager([]string{"PncAdmin", "GCNMFW:CaseManager"}))
	require.True(t, inboxadmin.IsManager([]string{" pncmanageradmin "}))
	require.False(t, inboxadmin.IsManager([]string{"PncAdmin", "PNCReportClaimInternal"}))
	require.False(t, inboxadmin.IsManager(nil))
}

func TestScopeFollowsLegacySteps(t *testing.T) {
	// Langkah 2: manajer tidak dibatasi cabang, dan kanwil pilihannya berlaku.
	scope, viewer := inboxadmin.ResolveScope(true, "100351", "KANWIL I")
	require.Equal(t, inboxadmin.Scope{RegionCode: "KANWIL I"}, scope)
	require.True(t, viewer.Manager)

	// Langkah 8: petugas cabang dibatasi cabangnya; kanwil pilihannya diabaikan.
	scope, viewer = inboxadmin.ResolveScope(false, "100351", "KANWIL I")
	require.Equal(t, inboxadmin.Scope{BranchCode: "100351"}, scope)
	require.Equal(t, "100351", viewer.BranchCode)

	// Kantor pusat dan cabang kosong melewati penyaring cabang.
	for _, branch := range []string{inboxadmin.HeadOfficeBranch, "", "  "} {
		scope, _ = inboxadmin.ResolveScope(false, branch, "")
		require.Equal(t, inboxadmin.Scope{}, scope, "cabang %q", branch)
	}
}
