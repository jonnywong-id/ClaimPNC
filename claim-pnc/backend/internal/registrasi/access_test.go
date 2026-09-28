package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// GROUP_ID tabel disamakan dengan literal access group Pega, tanpa duplikat.
func TestRolesOfGroupsAddsRulesetPrefix(t *testing.T) {
	require.Equal(t,
		[]string{"GCNMFW:PncAdmin", "GCNMFW:PNCKomiteTeknik", "GCNMFW:IT"},
		registrasi.RolesOfGroups([]string{" PncAdmin ", "PNCKomiteTeknik", "GCNMFW:PncAdmin", "", "IT"}))
}

// Grup menentukan tahap berdasarkan routernya: PncAdmin → PNCAdminRouter,
// PNCKomiteTeknik → PNCTeknikRouter. Workbasket tidak diatur grup.
func TestGroupStagesFollowRouter(t *testing.T) {
	d := registrasi.RegisterFlow()
	pic := d.GroupStages([]string{registrasi.RoleTechnicPIC})
	require.Contains(t, pic, registrasi.StageChooseSurveyor)
	require.Contains(t, pic, registrasi.StageSendToTechnicalPIC)
	require.NotContains(t, pic, registrasi.StageEstimateAdmin)

	admin := d.GroupStages([]string{registrasi.RoleAdmin})
	require.Contains(t, admin, registrasi.StageEstimateAdmin)
	require.NotContains(t, admin, registrasi.StageCompliance)
}
