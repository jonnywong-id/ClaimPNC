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

// Grup IT — padanan `Administrators` Pega — mengerjakan tugas tahap MANA PUN milik siapa pun.
//
// Ditetapkan Work Owner 2026-10-12. Uji ini menjaga DUA arah sekaligus: IT tembus di setiap
// tahap, dan grup biasa TETAP dibatasi tahapnya. Tanpa pasangan kedua, melonggarkan seluruh
// grup sekaligus akan lolos tanpa terlihat.
func TestITGroupWorksEveryStage(t *testing.T) {
	d := registrasi.RegisterFlow()
	it := registrasi.RolesOfGroups([]string{"IT"})
	task := registrasi.Task{Owner: "OPERATORPEGA"}

	for _, id := range []string{
		registrasi.StageChooseSurveyor, // Worklist, router PIC Teknik
		registrasi.StageInputRegister,  // Worklist, router Admin
		registrasi.StageRCLDoctor,      // Worklist, router RCL Dokter
		registrasi.StageCompliance,     // Workbasket
	} {
		stage, ok := d.Stage(id)
		require.True(t, ok, id)
		require.True(t, registrasi.CanWork(task, stage, "JONNY", it), id)
	}

	// Huruf besar-kecil tidak membedakan — `M_LOGIN_GROUP_PNC` tidak konsisten soal itu.
	stage, ok := d.Stage(registrasi.StageChooseSurveyor)
	require.True(t, ok)
	require.True(t, registrasi.CanWork(task, stage, "JONNY",
		registrasi.RolesOfGroups([]string{"it"})))

	// Grup biasa TIDAK ikut longgar: Admin tetap ditolak di tahap PIC Teknik.
	require.False(t, registrasi.CanWork(task, stage, "LOGIN@CONTOH",
		registrasi.RolesOfGroups([]string{"PncAdmin"})))
}

// Inbox akun IT memuat tugas antrean SELURUH tahap, seperti `Administrators` di Pega.
func TestITGroupSeesEveryStageInInbox(t *testing.T) {
	d := registrasi.RegisterFlow()
	it := d.GroupStages(registrasi.RolesOfGroups([]string{"IT"}))

	// Tahap dari router yang berbeda-beda — bukan satu router saja.
	for _, id := range []string{
		registrasi.StageInputRegister,
		registrasi.StageChooseSurveyor,
		registrasi.StageRCLDoctor,
	} {
		require.Contains(t, it, id)
	}

	// Pembanding: grup Admin TETAP hanya memegang tahapnya sendiri.
	admin := d.GroupStages([]string{registrasi.RoleAdmin})
	require.NotContains(t, admin, registrasi.StageChooseSurveyor)
}
