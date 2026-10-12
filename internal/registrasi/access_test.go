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

	// PncPICTeknik — access group PIC Teknik Pega — diterima berdampingan dengan
	// PNCKomiteTeknik (Work Owner, 2026-10-08).
	picGroup := d.GroupStages(registrasi.RolesOfGroups([]string{"PncPICTeknik"}))
	require.Equal(t, pic, picGroup)

	admin := d.GroupStages([]string{registrasi.RoleAdmin})
	require.Contains(t, admin, registrasi.StageEstimateAdmin)
	require.NotContains(t, admin, registrasi.StageCompliance)
}

// Tugas PIC Teknik milik orang lain boleh dikerjakan pemegang PncPICTeknik maupun
// PNCKomiteTeknik; grup Admin tidak.
func TestTechnicalStageAcceptsBothPICGroups(t *testing.T) {
	d := registrasi.RegisterFlow()
	stage, ok := d.Stage(registrasi.StageChooseSurveyor)
	require.True(t, ok)
	task := registrasi.Task{Owner: "OPERATORPEGA"}
	for _, group := range []string{"PncPICTeknik", "PNCKomiteTeknik", "pncpicteknik"} {
		roles := registrasi.RolesOfGroups([]string{group})
		require.True(t, registrasi.CanWork(task, stage, "LOGIN@CONTOH", roles), group)
	}
	require.False(t, registrasi.CanWork(task, stage, "LOGIN@CONTOH", registrasi.RolesOfGroups([]string{"PncAdmin"})))
}
