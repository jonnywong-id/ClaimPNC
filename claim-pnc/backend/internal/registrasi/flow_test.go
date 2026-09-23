package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// claimForLine menyusun klaim minimal yang cukup untuk menguji percabangan alur.
func claimForLine(line registrasi.LineOfBusiness, stage string) registrasi.Claim {
	return registrasi.Claim{
		ID:           "klaim-uji",
		Policy:       registrasi.Policy{Number: "POL-1", Line: line},
		CurrentStage: stage,
	}
}

func next(t *testing.T, d registrasi.Definition, from string, fctx registrasi.FlowContext) registrasi.Node {
	t.Helper()
	node, _, err := d.Next(from, fctx)
	require.NoError(t, err)
	return node
}

// TestFlowMatchesRegisterFlow menelusuri ketiga jalur lini bisnis dari ujung ke ujung.
//
// Ketiganya dibaca langsung dari `Flow/Register_Flow.xml`, dan urutan tahapnya adalah
// bukti yang dibandingkan pada gerbang 1.
func TestFlowMatchesRegisterFlow(t *testing.T) {
	d := registrasi.RegisterFlow()

	t.Run("mulai dari View Polis", func(t *testing.T) {
		require.Equal(t, registrasi.StageViewPolicy, d.Start)

		node := next(t, d, registrasi.StageViewPolicy, registrasi.FlowContext{})
		require.Equal(t, registrasi.NodeStage, node.Kind)
		require.Equal(t, registrasi.StageInputRegister, node.Stage.ID)
	})

	t.Run("lini PA menuju Estimation", func(t *testing.T) {
		fctx := registrasi.FlowContext{Claim: claimForLine(registrasi.LinePersonalAccident, registrasi.StageInputRegister)}
		node := next(t, d, registrasi.StageInputRegister, fctx)
		require.Equal(t, registrasi.StageEstimatePA, node.Stage.ID)
	})

	t.Run("lini Travel menuju Input Estimasi Travel", func(t *testing.T) {
		fctx := registrasi.FlowContext{Claim: claimForLine(registrasi.LineTravel, registrasi.StageInputRegister)}
		node := next(t, d, registrasi.StageInputRegister, fctx)
		require.Equal(t, registrasi.StageEstimateTravel, node.Stage.ID)
	})

	// Cabang ELSE bernama "NonMBU" menampung SELURUH lini selain 002 dan 005 — termasuk
	// kode yang tidak disebut rule IsNonMBU sama sekali. Ini temuan yang dibawa apa
	// adanya, dan uji ini adalah yang menjaganya tetap terlihat.
	t.Run("lini di luar PA dan Travel menuju Input Estimasi Admin", func(t *testing.T) {
		for _, line := range []registrasi.LineOfBusiness{
			registrasi.LineMiscellaneous,
			registrasi.LineMarineCargo,
			registrasi.LineFire,
			registrasi.LineOfBusiness("007"),
			registrasi.LineOfBusiness("008"),
			registrasi.LineOfBusiness("009"),
		} {
			fctx := registrasi.FlowContext{Claim: claimForLine(line, registrasi.StageInputRegister)}
			node := next(t, d, registrasi.StageInputRegister, fctx)
			require.Equal(t, registrasi.StageEstimateAdmin, node.Stage.ID, "lini %s", line)
		}
	})

	t.Run("jalur PA lengkap sampai akhir", func(t *testing.T) {
		fctx := registrasi.FlowContext{Claim: claimForLine(registrasi.LinePersonalAccident, registrasi.StageEstimatePA)}

		node := next(t, d, registrasi.StageEstimatePA, fctx)
		require.Equal(t, registrasi.StageInvestigator, node.Stage.ID)

		node = next(t, d, registrasi.StageInvestigator, fctx)
		require.Equal(t, registrasi.StageSendToAnalyst, node.Stage.ID)

		// Tanpa peran Analyst Doctor dan tanpa penanda PUCL/Compliance, cabang ELSE
		// mengirim klaim ke RCLDokter.
		node = next(t, d, registrasi.StageSendToAnalyst, fctx)
		require.Equal(t, registrasi.StageRCLDoctor, node.Stage.ID)

		node = next(t, d, registrasi.StageRCLDoctor, fctx)
		require.Equal(t, registrasi.NodeEnd, node.Kind)
		require.Equal(t, registrasi.ProcessDone, node.End.ProcessStatus)
	})

	t.Run("jalur Non-MBU berakhir setelah Choose Surveyor", func(t *testing.T) {
		fctx := registrasi.FlowContext{Claim: claimForLine(registrasi.LineFire, registrasi.StageEstimateAdmin)}

		node := next(t, d, registrasi.StageEstimateAdmin, fctx)
		require.Equal(t, registrasi.StageChooseSurveyor, node.Stage.ID)

		node = next(t, d, registrasi.StageChooseSurveyor, fctx)
		require.Equal(t, registrasi.NodeEnd, node.Kind)
	})
}

// TestBackGateReturnsToPreviousStage menguji keempat gerbang `IsBackStage`.
func TestBackGateReturnsToPreviousStage(t *testing.T) {
	d := registrasi.RegisterFlow()

	cases := []struct {
		name string
		from string
		line registrasi.LineOfBusiness
		back string
	}{
		{"dari Input Register", registrasi.StageInputRegister, registrasi.LineFire, registrasi.StageViewPolicy},
		{"dari Estimation PA", registrasi.StageEstimatePA, registrasi.LinePersonalAccident, registrasi.StageInputRegister},
		{"dari Input Estimasi Admin", registrasi.StageEstimateAdmin, registrasi.LineFire, registrasi.StageInputRegister},
		{"dari Input Estimasi Travel", registrasi.StageEstimateTravel, registrasi.LineTravel, registrasi.StageInputRegister},
	}

	for _, k := range cases {
		t.Run(k.name, func(t *testing.T) {
			claim := claimForLine(k.line, k.from)
			claim.RequestReturn = true

			node := next(t, d, k.from, registrasi.FlowContext{Claim: claim})
			require.Equal(t, registrasi.NodeStage, node.Kind)
			require.Equal(t, k.back, node.Stage.ID)
		})
	}
}

// TestDecisionAfterAnalyst menguji urutan keempat cabang `Decision7`, termasuk cabang
// yang bergantung pada PERAN PEMANGGIL dan bukan pada data klaim.
func TestDecisionAfterAnalyst(t *testing.T) {
	d := registrasi.RegisterFlow()
	base := claimForLine(registrasi.LinePersonalAccident, registrasi.StageSendToAnalyst)

	t.Run("peran Analyst Doctor menang atas seluruh cabang lain", func(t *testing.T) {
		claim := base
		// Kedua penanda data klaim menyala; cabang pertama tetap menang karena ia
		// diuji lebih dulu. Inilah yang membuat rute klaim bergantung pada siapa yang
		// menekan tombol.
		claim.PUCLStatus = registrasi.PUCLStatusInProgress
		claim.ComplianceTransfer = true

		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{
			Claim:       claim,
			CallerRoles: []string{registrasi.RoleAnalystDoctor},
		})
		require.Equal(t, registrasi.StageAnalystDoctor, node.Stage.ID)
	})

	t.Run("administrator ikut memicu cabang Analyst Doctor", func(t *testing.T) {
		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{
			Claim:       base,
			CallerRoles: []string{registrasi.RoleAdministrator},
		})
		require.Equal(t, registrasi.StageAnalystDoctor, node.Stage.ID)
	})

	t.Run("peran ViewClaimPNC membatalkan cabang Analyst Doctor", func(t *testing.T) {
		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{
			Claim:       base,
			CallerRoles: []string{registrasi.RoleAdministrator, registrasi.RoleViewClaimPNC},
		})
		require.Equal(t, registrasi.StageRCLDoctor, node.Stage.ID)
	})

	t.Run("penanda PUCL mengarahkan ke RCL/PUCL", func(t *testing.T) {
		claim := base
		claim.PUCLStatus = registrasi.PUCLStatusInProgress

		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{Claim: claim})
		require.Equal(t, registrasi.StageRCLPUCL, node.Stage.ID)
		require.Equal(t, registrasi.QueueWorkbasket, node.Stage.Queue)
		require.Equal(t, registrasi.WorkbasketRCLPUCL, node.Stage.Workbasket)
	})

	t.Run("penanda Compliance mengarahkan ke Compliance", func(t *testing.T) {
		claim := base
		claim.ComplianceTransfer = true

		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{Claim: claim})
		require.Equal(t, registrasi.StageCompliance, node.Stage.ID)
	})

	t.Run("PUCL didahulukan atas Compliance", func(t *testing.T) {
		claim := base
		claim.PUCLStatus = registrasi.PUCLStatusInProgress
		claim.ComplianceTransfer = true

		node := next(t, d, registrasi.StageSendToAnalyst, registrasi.FlowContext{Claim: claim})
		require.Equal(t, registrasi.StageRCLPUCL, node.Stage.ID)
	})
}

// TestQueueSplit menjaga pembagian Worklist/Workbasket tetap sama dengan
// `Flow/Register_Flow.xml` (`ADR-0019`).
func TestQueueSplit(t *testing.T) {
	d := registrasi.RegisterFlow()

	workbasket := map[string]string{
		registrasi.StageRCLPUCL:      registrasi.WorkbasketRCLPUCL,
		registrasi.StageInvestigator: registrasi.WorkbasketInvestigator,
		registrasi.StageCompliance:   registrasi.WorkbasketCompliance,
	}

	for _, stage := range d.Stages() {
		name, isWorkbasket := workbasket[stage.ID]
		if isWorkbasket {
			require.Equal(t, registrasi.QueueWorkbasket, stage.Queue, "tahap %s", stage.ID)
			require.Equal(t, name, stage.Workbasket, "tahap %s", stage.ID)
			continue
		}
		require.Equal(t, registrasi.QueueWorklist, stage.Queue, "tahap %s", stage.ID)
		require.Empty(t, stage.Workbasket, "tahap %s", stage.ID)
	}

	// Tiga belas Assignment pada diagram; tiga di antaranya Workbasket.
	require.Len(t, d.Stages(), 13)
}

// TestOnlyAnalystDoctorUsesNamedOperator menjaga agar penugasan ke ORANG BERNAMA tidak
// menyebar ke tahap lain.
//
// Satu-satunya tahap yang begitu di sistem lama adalah Analyst Doctor, dan Work Owner
// memutuskan nilainya dibawa apa adanya (`P-5`). Uji ini memastikan keputusan itu tetap
// berupa satu pengecualian yang disadari, bukan pola yang diam-diam ditiru.
func TestOnlyAnalystDoctorUsesNamedOperator(t *testing.T) {
	d := registrasi.RegisterFlow()

	for _, stage := range d.Stages() {
		if stage.ID == registrasi.StageAnalystDoctor {
			require.Equal(t, registrasi.OperatorAnalystDoctor, stage.Operator)
			continue
		}
		require.Empty(t, stage.Operator, "tahap %s tidak boleh merutekan ke orang bernama", stage.ID)
	}
}

// TestPathDoesNotLoop memastikan gambaran jalur yang dikirim ke layar selalu
// berhenti, termasuk saat klaim meminta kembali.
func TestPathDoesNotLoop(t *testing.T) {
	d := registrasi.RegisterFlow()

	claim := claimForLine(registrasi.LineFire, registrasi.StageInputRegister)
	claim.RequestReturn = true

	path, err := d.Path(registrasi.StageInputRegister, registrasi.FlowContext{Claim: claim})
	require.NoError(t, err)
	require.NotEmpty(t, path)
	require.Equal(t, registrasi.StageInputRegister, path[0])
}
