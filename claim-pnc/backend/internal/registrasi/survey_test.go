package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func messages(v []registrasi.Violation) []string {
	out := []string{}
	for _, x := range v {
		out = append(out, x.Message)
	}
	return out
}

// setError_act: tanpa objek terpilih hanya "Harus Pilih Object"; lokasi survey kosong diisi lokasi
// objek lebih dulu; surveyor kosong ditolak kecuali tipe 1 dan 2 (perilaku rule apa adanya).
func TestCheckSurveyObjects(t *testing.T) {
	objects := []registrasi.SurveyObject{{ObjectID: "1"}, {ObjectID: "2"}}
	require.Equal(t, []string{registrasi.MsgSurveyMustSelect}, messages(registrasi.CheckSurveyObjects(objects, nil)))

	objects = []registrasi.SurveyObject{
		{ObjectID: "1", Selected: true, Location: "Jakarta", SurveyorType: registrasi.SurveyorInternal},
		{ObjectID: "2", Selected: true, SurveyorType: registrasi.SurveyorExpert},
		{ObjectID: "3", Selected: true, Location: "Bandung", SurveyorType: registrasi.SurveyorLossAdjuster},
	}
	v := registrasi.CheckSurveyObjects(objects, nil)
	require.Equal(t, []string{registrasi.MsgSurveyLocationEmpty, registrasi.MsgSurveyLoginEmpty}, messages(v))
	require.Equal(t, "Jakarta", objects[0].SurveyLocation)

	all := []registrasi.SurveyObject{{ObjectID: "1", Selected: true, Location: "x", Status: registrasi.SurveyInProgress}}
	require.Equal(t, []string{registrasi.MsgSurveyAllInProgress}, messages(registrasi.CheckSurveyObjects(all, nil)))
}

// SetErrorSRV_act: surveyor yang sama sedang survey objek yang sama.
func TestCheckSurveyObjectsSurveyorBusy(t *testing.T) {
	objects := []registrasi.SurveyObject{
		{ObjectID: "1", Selected: true, Location: "x", SurveyorType: registrasi.SurveyorInternal, SurveyorName: "A", Status: registrasi.SurveyInProgress},
		{ObjectID: "2", Selected: true, Location: "y", SurveyorType: registrasi.SurveyorInternal},
	}
	rows := []registrasi.SurveyRow{{CaseID: "SRVN.26.1", ObjectID: "1", SurveyorName: "A", Status: registrasi.SurveyRecordOnProgress}}
	require.Equal(t, []string{registrasi.MsgSurveySurveyorBusy}, messages(registrasi.CheckSurveyObjects(objects, rows)))

	rows[0].WorkStatus = registrasi.SurveyWorkRejected
	require.Empty(t, registrasi.CheckSurveyObjects(objects, rows))
}

// CancelSurvey langkah 2, 3, 8.
func TestCancelSurveyObject(t *testing.T) {
	objects := []registrasi.SurveyObject{
		{ObjectID: "1", Selected: true, Status: registrasi.SurveyInProgress, SurveyorName: "A", SurveyorType: "2", BranchCode: "B1", BranchName: "C1", SurveyID: "SRVN.26.1"},
		{ObjectID: "2", SurveyorName: "B", SurveyorType: "1", BranchCode: "B2", BranchName: "C2"},
	}
	registrasi.CancelSurveyObject(objects, 0, false)
	require.Equal(t, registrasi.SurveyCancelled, objects[0].Status)
	require.False(t, objects[0].Selected)
	require.Equal(t, "", objects[0].BranchCode)
	require.Equal(t, "SRVN.26.1", objects[0].SurveyID)
	require.Equal(t, "", objects[1].SurveyorName)
	require.Equal(t, "", objects[1].SurveyorType)
	require.Equal(t, "B2", objects[1].BranchCode)

	pa := []registrasi.SurveyObject{{ObjectID: "1", Status: registrasi.SurveyInProgress}, {ObjectID: "2"}}
	registrasi.CancelSurveyObject(pa, 0, true)
	require.Equal(t, registrasi.PABranchCodeAfterCancel, pa[0].BranchCode)
	require.Equal(t, registrasi.PABranchCodeAfterCancel, pa[1].BranchCode)
	require.Equal(t, "", pa[1].SurveyorType)
}

func TestSurveyHelpers(t *testing.T) {
	require.Equal(t, "SRVN.26.12", registrasi.FormatSurveyID(2026, 12))
	require.True(t, registrasi.IsMarineHull(registrasi.Policy{BusinessCode: "10152"}))
	require.Equal(t, "PA", registrasi.SurveyCommitteeLine(registrasi.Policy{Line: registrasi.LinePersonalAccident}))
	require.Equal(t, "NONMBU", registrasi.SurveyCommitteeLine(registrasi.Policy{Line: registrasi.LineFire}))
	require.True(t, registrasi.CoInsuranceMember(registrasi.Policy{Coinsurance: registrasi.Coinsurance{Role: "MEMBER"}}))
}
