package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Setiap uji di berkas ini menjalankan satu operasi layanan sambil memasang galat pada
// setiap seam yang dipanggilnya, satu panggilan demi satu panggilan. Lulusnya uji berarti
// tidak ada galat penyimpanan atau layanan hulu yang diam-diam ditelan.

var bg = context.Background()

func TestPropagationStartAndViewPolicy(t *testing.T) {
	runPropagation(t, propagation{
		name:    "Start",
		prepare: func(t *testing.T, l environment) any { return nil },
		op: func(l environment, _ any) error {
			_, err := l.service.Start(bg, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name: "CompleteStage View Polis",
		prepare: func(t *testing.T, l environment) any {
			start, err := l.service.Start(bg, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
			require.NoError(t, err)
			return start.Task
		},
		op: func(l environment, s any) error {
			_, err := l.service.CompleteStage(bg, usecase.CompleteCommand{TaskID: s.(registrasi.Task).ID, Action: registrasi.ActionViewPolicy}, l.caller)
			return err
		},
	})
}

func TestPropagationRegister(t *testing.T) {
	prepare := func(t *testing.T, l environment) any {
		_, task := l.upToInputRegister(t, firePolicy)
		return task
	}
	runPropagation(t, propagation{
		name:    "SaveRegister",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.SaveRegister(bg, validInput(s.(registrasi.Task).ID), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "SaveDraft",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.SaveDraft(bg, validInput(s.(registrasi.Task).ID), l.caller)
			return err
		},
	})
}

// Registrasi bernilai besar memanggil Notice of Large Losses: parameter dan pemberitahuan.
func TestPropagationRegisterLargeLoss(t *testing.T) {
	runPropagation(t, propagation{
		name: "SaveRegister large loss",
		prepare: func(t *testing.T, l environment) any {
			_, task := l.upToInputRegister(t, firePolicy)
			return task
		},
		op: func(l environment, s any) error {
			input := validInput(s.(registrasi.Task).ID)
			input.EstimateValue = registrasi.Rupiah(2_000_000_000)
			_, err := l.service.SaveRegister(bg, input, l.caller)
			return err
		},
	})
}

func TestPropagationEstimate(t *testing.T) {
	prepare := func(t *testing.T, l environment) any {
		_, task := l.upToInputEstimate(t)
		return task
	}
	runPropagation(t, propagation{
		name:    "SaveEstimate",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.SaveEstimate(bg, oneEstimate(s.(registrasi.Task).ID, registrasi.Rupiah(50_000_000), 1), l.caller)
			return err
		},
	})
	withSheet := func(t *testing.T, l environment) any {
		_, task := l.upToInputEstimate(t)
		_, err := l.service.SaveEstimate(bg, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
		require.NoError(t, err)
		return task
	}
	runPropagation(t, propagation{
		name:    "DownloadFaceSheet",
		prepare: withSheet,
		op: func(l environment, s any) error {
			_, err := l.service.DownloadFaceSheet(bg, faceSheet(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name: "CompleteEstimate",
		prepare: func(t *testing.T, l environment) any {
			task := withSheet(t, l).(registrasi.Task)
			_, err := l.service.DownloadFaceSheet(bg, faceSheet(task), l.caller)
			require.NoError(t, err)
			return task
		},
		op: func(l environment, s any) error {
			_, err := l.service.CompleteEstimate(bg, oneEstimate(s.(registrasi.Task).ID, registrasi.Rupiah(50_000_000), 1), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ItemOptions",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.ItemOptions(bg, s.(registrasi.Task).ClaimID, "OBJ-1", "")
			return err
		},
	})
}

func TestPropagationPLA(t *testing.T) {
	prepare := func(t *testing.T, l environment) any {
		coinsLeader(l)
		_, task := l.upToInputEstimate(t)
		_, err := l.service.SaveEstimate(bg, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
		require.NoError(t, err)
		_, err = l.service.DownloadFaceSheet(bg, faceSheet(task), l.caller)
		require.NoError(t, err)
		return task
	}
	runPropagation(t, propagation{
		name:    "PrintPLA",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.PrintPLA(bg, printPLA(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ListPLA",
		prepare: prepare,
		op: func(l environment, s any) error {
			_, err := l.service.ListPLA(bg, printPLA(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name: "SavePLANotes",
		prepare: func(t *testing.T, l environment) any {
			task := prepare(t, l).(registrasi.Task)
			list, err := l.service.ListPLA(bg, printPLA(task), l.caller)
			require.NoError(t, err)
			return [2]any{task, list.PLA[0].Number}
		},
		op: func(l environment, s any) error {
			st := s.([2]any)
			_, err := l.service.SavePLANotes(bg, printPLA(st[0].(registrasi.Task)), map[string]string{st[1].(string): "n"}, l.caller)
			return err
		},
	})
}

func TestPropagationSettlementAndCommittee(t *testing.T) {
	surveyor := func(t *testing.T, l environment) any {
		return l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	}
	final := registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}
	runPropagation(t, propagation{
		name:    "AddSettlement",
		prepare: surveyor,
		op: func(l environment, s any) error {
			_, err := l.service.AddSettlement(bg, addSettlement(s.(registrasi.Task), final), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "PreviewSettlement",
		prepare: surveyor,
		op: func(l environment, s any) error {
			_, err := l.service.PreviewSettlement(bg, addSettlement(s.(registrasi.Task), final), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name: "TransferCommittee",
		prepare: func(t *testing.T, l environment) any {
			task := surveyor(t, l).(registrasi.Task)
			_, err := l.service.AddSettlement(bg, addSettlement(task, final), l.caller)
			require.NoError(t, err)
			return task
		},
		op: func(l environment, s any) error {
			_, err := l.service.TransferCommittee(bg, transfer(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	transferred := func(t *testing.T, l environment) any {
		_, result := l.transferredClaim(t)
		return result.Committee.ID
	}
	runPropagation(t, propagation{
		name:    "DecideCommittee setuju",
		prepare: transferred,
		op: func(l environment, s any) error {
			_, err := l.service.DecideCommittee(bg, decide(s.(string), registrasi.DecisionApprove, ""), committee1)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "DecideCommittee tolak",
		prepare: transferred,
		op: func(l environment, s any) error {
			_, err := l.service.DecideCommittee(bg, decide(s.(string), registrasi.DecisionReject, "tidak"), committee1)
			return err
		},
	})
	runPropagation(t, propagation{
		name: "DecideCommittee jenjang terakhir",
		prepare: func(t *testing.T, l environment) any {
			id := transferred(t, l).(string)
			_, err := l.service.DecideCommittee(bg, decide(id, registrasi.DecisionApprove, ""), committee1)
			require.NoError(t, err)
			return id
		},
		op: func(l environment, s any) error {
			_, err := l.service.DecideCommittee(bg, decide(s.(string), registrasi.DecisionApprove, ""), committee2)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "PendingCommittees",
		prepare: transferred,
		op: func(l environment, _ any) error {
			_, err := l.service.PendingCommittees(bg, committee1)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "Committee",
		prepare: transferred,
		op: func(l environment, s any) error {
			_, err := l.service.Committee(bg, s.(string))
			return err
		},
	})
}

func TestPropagationReceiverAndAcceptance(t *testing.T) {
	runPropagation(t, propagation{
		name: "SaveReceiver",
		prepare: func(t *testing.T, l environment) any {
			return l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
		},
		op: func(l environment, s any) error {
			task := s.(registrasi.Task)
			_, err := l.service.SaveReceiver(bg, usecase.ReceiverCommand{
				ClaimID: task.ClaimID, TaskID: task.ID, ReceiverID: "1", AccountNo: "1234567890", Email: "a@contoh.internal",
			}, l.caller)
			return err
		},
	})
	approved := func(t *testing.T, l environment) any {
		task, _ := l.approvedClaim(t)
		return task
	}
	runPropagation(t, propagation{
		name:    "AcceptSettlement",
		prepare: approved,
		// Status premi yang tidak terbaca diterjemahkan menjadi ErrPremiumUnavailable.
		translated: map[string]error{"Premium.Statement": usecase.ErrPremiumUnavailable},
		op: func(l environment, s any) error {
			_, err := l.service.AcceptSettlement(bg, acceptance(s.(registrasi.Task), registrasi.LODAgreed), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "PrintLOD",
		prepare: approved,
		op: func(l environment, s any) error {
			task := s.(registrasi.Task)
			_, err := l.service.PrintLOD(bg, usecase.LODCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1, Type: "4"}, l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "LODDialog",
		prepare: approved,
		op: func(l environment, s any) error {
			task := s.(registrasi.Task)
			_, err := l.service.LODDialog(bg, task.ClaimID, task.ID, l.caller)
			return err
		},
	})
}

func TestPropagationDLAAndCashier(t *testing.T) {
	accepted := func(t *testing.T, l environment) any {
		task, _ := l.acceptedForDLA(t)
		return task
	}
	runPropagation(t, propagation{
		name:    "ListDLA",
		prepare: accepted,
		op: func(l environment, s any) error {
			_, err := l.service.ListDLA(bg, dlaCommand(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "PrintDLA",
		prepare: accepted,
		op: func(l environment, s any) error {
			_, err := l.service.PrintDLA(bg, dlaCommand(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	ready := func(t *testing.T, l environment) any {
		task := l.readyForCashier(t)
		list, err := l.service.ListDLA(bg, dlaCommand(task), l.caller)
		require.NoError(t, err)
		for _, d := range list.DLA {
			cmd := dlaCommand(task)
			cmd.Number = d.Number
			_, err := l.service.PrintDLA(bg, cmd, l.caller)
			require.NoError(t, err)
		}
		l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}
		return task
	}
	runPropagation(t, propagation{
		name:    "PreviewCashierTransfer",
		prepare: ready,
		op: func(l environment, s any) error {
			_, err := l.service.PreviewCashierTransfer(bg, cashierCommand(s.(registrasi.Task)), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "TransferCashier",
		prepare: ready,
		op: func(l environment, s any) error {
			_, err := l.service.TransferCashier(bg, cashierCommand(s.(registrasi.Task)), l.caller)
			return err
		},
	})
}

func TestPropagationRecordsAndTasks(t *testing.T) {
	started := func(t *testing.T, l environment) any {
		start, err := l.service.Start(bg, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
		require.NoError(t, err)
		claim := start.Claim
		claim.Policy.BusinessCode = "10027"
		require.NoError(t, l.store.Save(bg, claim))
		return start
	}
	claimID := func(s any) string { return s.(usecase.StartResult).Claim.ID }
	runPropagation(t, propagation{
		name:    "UploadDocument",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.UploadDocument(bg, usecase.UploadDocumentCommand{
				ClaimID: claimID(s), Portal: "ASM", DocumentTypeID: "14901", FileName: "a.pdf", Content: []byte("%PDF"),
			}, l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "Documents",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.Documents(bg, claimID(s))
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "Surveys",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.Surveys(bg, claimID(s))
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ProgressRecords",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.ProgressRecords(bg, claimID(s))
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ViewClaim",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.ViewClaim(bg, claimID(s), l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ClaimTask",
		prepare: started,
		op: func(l environment, s any) error {
			_, err := l.service.ClaimTask(bg, s.(usecase.StartResult).Task.ID, l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "Inbox",
		prepare: started,
		op: func(l environment, _ any) error {
			_, err := l.service.Inbox(bg, l.caller)
			return err
		},
	})
	runPropagation(t, propagation{
		name:    "ResolveCaller, AreaOptions, Currencies, FindAccount",
		prepare: func(t *testing.T, l environment) any { return nil },
		op: func(l environment, _ any) error {
			if _, err := l.service.ResolveCaller(bg, l.caller); err != nil {
				return err
			}
			if _, err := l.service.AreaOptions(bg, registrasi.AreaCountry, ""); err != nil {
				return err
			}
			if _, err := l.service.Currencies(bg); err != nil {
				return err
			}
			_, err := l.service.FindAccount(bg, "1234567890")
			return err
		},
	})
}
