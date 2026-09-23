package registrasi

// Peran yang benar-benar dipakai alur ini untuk memilih cabang.
//
// Tabel peran dan izin menu adalah `TKT-F3-004` yang masih terhalang artefak; yang
// dideklarasikan di sini hanya ketiga nama yang dibaca rule `IsAnalystDoctor`, apa
// adanya termasuk awalan ruleset-nya.
const (
	RoleAdministrator = "GCNMFW:Administrators"
	RoleAnalystDoctor = "GCNMFW:PncAnalystDoctor"
	RoleViewClaimPNC  = "GCNMFW:ViewClaimPNC"
)

// OperatorAnalystDoctor adalah operator yang menerima seluruh tugas tahap Analyst
// Doctor.
//
// NAMA ORANG DI DALAM ATURAN. Di `Flow/Register_Flow.xml`, shape `Assignment13`
// merutekan dengan `pyRouteTo=Operator` dan `pyOperator=DRRATNA` — satu pegawai
// bernama, tertanam di dalam rule. `D-15` dan `ADR-0025` menetapkan nilai semacam ini
// naik menjadi master.
//
// Work Owner memutuskan 2026-09-17 nilainya **dibawa apa adanya**, mengikuti `P-5`
// (perilaku dipertahankan lebih dulu, diperbaiki kemudian). Yang ikut terbawa adalah
// akibatnya, yang `ADR-0019` sudah sebutkan: bila orang ini tidak hadir, tugas tahap
// Analyst Doctor tidak punya penerima lain.
//
// Konstanta ini dipisahkan supaya perubahannya kelak berupa satu baris, bukan
// perburuan di seluruh berkas.
const OperatorAnalystDoctor = "DRRATNA"

// Nama workbasket, apa adanya dari `Flow/Register_Flow.xml`.
const (
	WorkbasketRCLPUCL      = "RCLPUCL"
	WorkbasketInvestigator = "InvestigatorPNC"
	WorkbasketCompliance   = "CompliancePNC"
)

// Nama aturan routing, apa adanya dari `Flow/Register_Flow.xml` (`ADR-0019`).
//
// Tiga di antaranya tidak ada di export (`R-04`): PNCAdminRouter, PNCTeknikRouter, dan
// RouterRCLDoctor. Namanya tetap dibawa supaya seam Penugasan punya sesuatu yang dapat
// diisi begitu logikanya diketahui, alih-alih menamai ulang dan kehilangan jejaknya.
const (
	RouterPNCAdmin        = "PNCAdminRouter"
	RouterPNCTechnical    = "PNCTeknikRouter"
	RouterRCLDoctor       = "RouterRCLDokter"
	RouterCurrentOperator = "ToCurrentOperator"
	RouterToWorklist      = "ToWorkList"
	RouterToWorkbasket    = "ToWorkbasket"
)

// Nama tindakan penutup tiap tahap — Flow Action di sistem lama.
const (
	ActionViewPolicy         = "ViewPolis"
	ActionInputRegister      = "InputRegister"
	ActionInputEstimate      = "InputEstimasi"
	ActionInputSurveyor      = "InputSurveyor"
	ActionInputInvestigation = "InputInvestigator"
	ActionSendToRCLPUCL      = "SendtoRCLPUCL"
	ActionSendToRCLDoctor    = "SendToRCLDokter"
	ActionSendAnalystDoctor  = "SendAnalystDoctor"
	ActionCompliance         = "ComplianceChecker"
)

// RegisterFlow menyusun definisi alur Register.
//
// Setiap simpul membawa `PegaID`-nya. Perbandingan dengan `Flow/Register_Flow.xml`
// karena itu dapat dilakukan langsung, tanpa kamus penerjemah di antaranya.
func RegisterFlow() Definition {
	node := map[string]Node{}
	add := func(s Node) { node[s.ID()] = s }

	stage := func(t Stage) { add(Node{Kind: NodeStage, Stage: t}) }
	decision := func(k Decision) { add(Node{Kind: NodeDecision, Decision: k}) }
	end := func(a End) { add(Node{Kind: NodeEnd, End: a}) }

	// ── Jalur pembuka ────────────────────────────────────────────────────────────
	//
	// Start → View Polis → Input Register. Keduanya Worklist; View Polis dirutekan ke
	// operator yang sedang membuka klaim, bukan dibagi ulang.
	stage(Stage{
		ID: StageViewPolicy, Name: "View Polis", PegaID: "Assignment2",
		Queue: QueueWorklist, Router: RouterCurrentOperator,
		ExitAction: ActionViewPolicy,
		Next:       StageInputRegister,
	})

	stage(Stage{
		ID: StageInputRegister, Name: "Input Register", PegaID: "Assignment1",
		Queue: QueueWorklist, Router: RouterPNCAdmin,
		ExitAction: ActionInputRegister,
		// `setToRegister_ticket` — Ticket rule yang TIDAK ADA di export (`ADR-0021`).
		// Namanya dibawa sebagai tujuan lompatan; pemicunya tidak dibuat-buat.
		LateralJump: "setToRegister_ticket",
		Next:        DecisionReturnFromRegister,
	})

	// ── Empat gerbang Back ───────────────────────────────────────────────────────
	//
	// Keempatnya memakai rule yang sama, `IsBackStage`, dan hanya berbeda pada tujuan
	// mundurnya. Di sistem lama rule itu membandingkan `.pyNote == "Back"`.
	back := func(id, pegaID, back, advance, advanceLabel string) {
		decision(Decision{
			ID: id, Name: "IsBack", PegaID: pegaID,
			Branch: []Branch{{
				Name:   "IsBackStage",
				Test:   func(k FlowContext) bool { return k.Claim.RequestReturn },
				Target: back,
			}},
			Otherwise:     advance,
			OtherwiseName: advanceLabel,
		})
	}

	back(DecisionReturnFromRegister, "Decision3", StageViewPolicy, DecisionLinePA, "Continue")
	back(DecisionReturnFromEstimatePA, "Decision4", StageInputRegister, StageInvestigator, "Continue")
	back(DecisionReturnFromEstimateAdmin, "Decision5", StageInputRegister, StageChooseSurveyor, "[Else]")
	back(DecisionReturnFromEstimateTravel, "Decision6", StageInputRegister, StageSendToTechnicalPIC, "[Else]")

	// ── Percabangan lini bisnis ──────────────────────────────────────────────────
	decision(Decision{
		ID: DecisionLinePA, Name: "IsPA", PegaID: "Decision1",
		Branch: []Branch{{
			Name:   "isPA_PNC",
			Test:   func(k FlowContext) bool { return k.Claim.Policy.Line == LinePersonalAccident },
			Target: StageEstimatePA,
		}},
		Otherwise:     DecisionLineTravel,
		OtherwiseName: "NotPA",
	})

	decision(Decision{
		ID: DecisionLineTravel, Name: "IsTravel", PegaID: "Decision2",
		Branch: []Branch{{
			Name:   "IsTravel",
			Test:   func(k FlowContext) bool { return k.Claim.Policy.Line == LineTravel },
			Target: StageEstimateTravel,
		}},
		Otherwise: StageEstimateAdmin,
		// Label diagramnya "Non MBU", dan itu MENYESATKAN: cabang ini ELSE, sehingga
		// seluruh lini selain `002` dan `005` masuk ke sini — termasuk `007` dan `008`
		// yang tidak disebut rule `IsNonMBU` sama sekali. Labelnya dipertahankan apa
		// adanya; artinya yang sebenarnya dicatat di sini.
		OtherwiseName: "NonMBU",
	})

	// ── Tiga tahap estimasi, satu per jalur lini ─────────────────────────────────
	stage(Stage{
		ID: StageEstimatePA, Name: "Estimation", PegaID: "Assignment4",
		Queue: QueueWorklist, Router: RouterPNCAdmin,
		ExitAction:  ActionInputSurveyor,
		LateralJump: "SendToEstimatorPA",
		Next:        DecisionReturnFromEstimatePA,
	})

	stage(Stage{
		ID: StageEstimateTravel, Name: "Input Estimasi", PegaID: "Assignment10",
		Queue: QueueWorklist, Router: RouterPNCAdmin,
		ExitAction:  ActionInputEstimate,
		LateralJump: "SendToEstTravel",
		Next:        DecisionReturnFromEstimateTravel,
	})

	stage(Stage{
		ID: StageEstimateAdmin, Name: "Input Estimasi", PegaID: "Assignment7",
		Queue: QueueWorklist, Router: RouterPNCAdmin,
		ExitAction:  ActionInputEstimate,
		LateralJump: "SendToEstAdmin",
		Next:        DecisionReturnFromEstimateAdmin,
	})

	// ── Jalur PA: Investigator → Send To Analis ──────────────────────────────────
	stage(Stage{
		ID: StageInvestigator, Name: "Investigator", PegaID: "Assignment11",
		Queue: QueueWorkbasket, Router: RouterToWorkbasket, Workbasket: WorkbasketInvestigator,
		ExitAction:  ActionInputInvestigation,
		LateralJump: "SendToInvestigator",
		Next:        StageSendToAnalyst,
	})

	stage(Stage{
		ID: StageSendToAnalyst, Name: "Send To Analis", PegaID: "Assignment5",
		Queue: QueueWorklist, Router: RouterPNCTechnical,
		ExitAction:  ActionInputSurveyor,
		LateralJump: "SendtoAnalysator",
		Next:        DecisionAfterAnalyst,
	})

	// ── Jalur Travel: Send To PIC Teknik ─────────────────────────────────────────
	stage(Stage{
		ID: StageSendToTechnicalPIC, Name: "Send To PIC Teknik", PegaID: "Assignment8",
		Queue: QueueWorklist, Router: RouterPNCTechnical,
		ExitAction:  ActionInputSurveyor,
		LateralJump: "SendToPICTravel",
		Next:        DecisionAfterAnalyst,
	})

	// ── Jalur Non-MBU: Choose Surveyor, lalu selesai ─────────────────────────────
	stage(Stage{
		ID: StageChooseSurveyor, Name: "Choose Surveyor", PegaID: "Assignment3",
		Queue: QueueWorklist, Router: RouterPNCTechnical,
		ExitAction:  ActionInputSurveyor,
		LateralJump: "AcceptanceKomite",
		Next:        EndAfterSurveyor,
	})

	// ── Keputusan terakhir: ke mana klaim pergi setelah analis ───────────────────
	//
	// Urutan cabang di sini MENGIKAT dan sama persis dengan urutan `pyFromTasks` pada
	// `Decision7`: IsAnalystDoctor, lalu IsPUCL, lalu IsCompliance, lalu ELSE.
	//
	// CATAT SATU HAL. Cabang pertama tidak menguji klaim; ia menguji PERAN ORANG YANG
	// SEDANG MENEKAN TOMBOL. Rule `IsAnalystDoctor` berkelas `@baseclass` dan
	// membandingkan `AccessGroup.pyAccessGroup`. Akibatnya klaim yang sama berakhir di
	// tahap yang berbeda tergantung siapa yang menyelesaikan tahap sebelumnya —
	// seorang Administrator selalu mengirimnya ke Analyst Doctor. Dua cabang lain di
	// keputusan yang sama menguji data klaim. Asimetri ini dibawa apa adanya (`P-5`)
	// dan dicatat sebagai calon perbaikan.
	decision(Decision{
		ID: DecisionAfterAnalyst, Name: "RCLDokter, PUCL, AnalystDokter, Compliance", PegaID: "Decision7",
		Branch: []Branch{
			{
				Name: "IsAnalystDoctor",
				Test: func(k FlowContext) bool {
					return k.HasRole(RoleAdministrator, RoleAnalystDoctor) &&
						!k.HasRole(RoleViewClaimPNC)
				},
				Target: StageAnalystDoctor,
			},
			{
				Name:   "IsPUCL",
				Test:   func(k FlowContext) bool { return k.Claim.PUCLStatus == PUCLStatusInProgress },
				Target: StageRCLPUCL,
			},
			{
				Name:   "IsCompliance",
				Test:   func(k FlowContext) bool { return k.Claim.ComplianceTransfer },
				Target: StageCompliance,
			},
		},
		Otherwise:     StageRCLDoctor,
		OtherwiseName: "ElseRCLMSIG",
	})

	// ── Empat tahap penutup ──────────────────────────────────────────────────────
	stage(Stage{
		ID: StageAnalystDoctor, Name: "Analyst Doctor", PegaID: "Assignment13",
		Queue: QueueWorklist, Router: RouterToWorklist,
		Operator:   OperatorAnalystDoctor,
		ExitAction: ActionSendAnalystDoctor,
		Next:       EndAfterAnalyst,
	})

	stage(Stage{
		ID: StageRCLPUCL, Name: "RCL/PUCL", PegaID: "Assignment6",
		Queue: QueueWorkbasket, Router: RouterToWorkbasket, Workbasket: WorkbasketRCLPUCL,
		ExitAction:  ActionSendToRCLPUCL,
		LateralJump: "SendtoPUCL",
		Next:        EndAfterAnalyst,
	})

	stage(Stage{
		ID: StageCompliance, Name: "Compliance", PegaID: "Assignment9",
		Queue: QueueWorkbasket, Router: RouterToWorkbasket, Workbasket: WorkbasketCompliance,
		ExitAction:  ActionCompliance,
		LateralJump: "CompliancePNC",
		Next:        EndAfterAnalyst,
	})

	stage(Stage{
		ID: StageRCLDoctor, Name: "RCLDokter", PegaID: "Assignment12",
		Queue: QueueWorklist, Router: RouterRCLDoctor,
		ExitAction:  ActionSendToRCLDoctor,
		LateralJump: "RCLDokter",
		Next:        EndAfterAnalyst,
	})

	// ── Dua simpul akhir ─────────────────────────────────────────────────────────
	//
	// Keduanya menutup klaim dengan status proses yang sama. Yang satu membawa Ticket
	// rule `Status-Resolved` — bawaan Pega, bukan rule custom yang hilang.
	end(End{
		ID: EndAfterSurveyor, Name: "Selesai — setelah Choose Surveyor", PegaID: "END52",
		ProcessStatus: ProcessDone, LateralJump: "Status-Resolved",
	})
	end(End{
		ID: EndAfterAnalyst, Name: "Selesai — setelah jalur analis", PegaID: "End1",
		ProcessStatus: ProcessDone,
	})

	return Definition{
		Name:  "Register",
		Start: StageViewPolicy,
		node:  node,
	}
}

// PUCLStatusInProgress adalah nilai `ClaimData.PUCLStatus.RCL_PUCL` yang mengarahkan
// klaim ke tahap RCL/PUCL (rule `IsPUCL`).
const PUCLStatusInProgress = 2
