package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ActionTransferToAnalyst adalah local action yang dibuka tombol "Transfer ke Analyst"
// (`Section/TrfKomiteButton`): `Flow Action/ClaimComitee_OC`, berjudul "Transfer Claim ke
// Komite". Ia dicatat sebagai tindakan penutup tugas.
const ActionTransferToAnalyst = "ClaimComitee_OC"

// TicketTransferToAnalyst adalah Ticket rule yang dipasang `setTicketToAnalyst` step 7:
// `SetTicket(Ticket = "SendtoAnalysator")`. Tujuannya tahap Send To Analis.
const TicketTransferToAnalyst = "SendtoAnalysator"

// StatusClaimAnalyst adalah StatusClaim `1151` Analyst — `setTicketToAnalyst` step 6.
const StatusClaimAnalyst registrasi.ClaimStatus = "1151"

// TransferToAnalystCommand adalah jaminan yang tombolnya ditekan —
// `setTicketToAnalyst(CoverageID, ObjectID)`.
type TransferToAnalystCommand struct {
	TaskID     string
	ObjectID   string
	CoverageID string
}

// TransferToAnalyst menjalankan tombol Kirim Analyst pada modal "Transfer Claim ke Komite".
//
// Tombolnya ada pada SETIAP baris jaminan grid Adjustment & Akseptasi (`ObjectCoverageAdj` →
// include `TrfKomiteButton`), di dalam kontainer `IsPA`, dengan kondisi
// `.IsAnalisTransfer != '1' && !IsPHK && pyWorkPage.ClaimData.PNCStatus != '5'` — `.IsAnalisTransfer`
// milik jaminan itu (ISANALISTRANSFER).
//
// `setTicketToAnalyst` (Pega), diikuti apa adanya:
//   - step 5/8: jaminan yang ditekan ditandai (`IsAnalisTransfer`, `IsKomiteTransfer`,
//     `UserBusinessPA` := 1) — ISANALISTRANSFER, ISKOMITETRANSFER, USERBUSINESSPA.
//   - step 9: `AnalystTransferDate` diisi bila masih kosong — ANALYST_TRANSFERDATE.
//   - step 6–7: HANYA bila jaminan itu adalah jaminan TERAKHIR objeknya
//     (`ObjectCoverageList(<last>).CoverageID == param.CoverageID`): StatusClaim 1151 dan
//     `SetTicket(SendtoAnalysator)` → tahap Send To Analis. Jaminan lain hanya ditandai; klaim
//     dan tugasnya tetap di Estimation. Baris lain step 6 hanya untuk satu klaim bernomor tetap
//     (perbaikan produksi) dan tidak dibawa (`D-15`).
//
// Belum dibawa: `InsertHistoryClaimPNC`, `InsertJsonClaimNonMBU_act`, dan
// `PNCInsertMitraLog_Act` (step 10–13).
func (l *Service) TransferToAnalyst(ctx context.Context, p TransferToAnalystCommand, by Caller) (CompleteResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID})
	if err != nil {
		return CompleteResult{}, err
	}
	if claim.CurrentStage != registrasi.StageEstimatePA {
		return CompleteResult{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, claim.CurrentStage)
	}
	objectID, coverageID := strings.TrimSpace(p.ObjectID), strings.TrimSpace(p.CoverageID)
	i, j, err := findAnalystCoverage(claim, objectID, coverageID)
	if err != nil {
		return CompleteResult{}, err
	}
	last := j == len(claim.InsuredItem[i].Coverage)-1

	// PreClaimComitee_OC langkah 21 (modal ClaimComitee_OC dibuka): UserTeknis kosong diisi PIC
	// Teknik bawaan — di Pega satu Operator ID di dalam rule, di sini pengaturan
	// PIC_TEKNIK_PA_BAWAAN (D-15). PNCTeknikRouter tahap Send To Analis lalu menugaskan ke sana.
	if strings.TrimSpace(claim.TechnicalPIC) == "" && l.defaultPATechnicalPIC != "" {
		claim.TechnicalPIC = l.defaultPATechnicalPIC
	}

	now := l.clock.Now().UTC()
	claim.InsuredItem[i].Coverage[j].AnalystTransferred = true
	if claim.AnalystTransferredAt.IsZero() {
		claim.AnalystTransferredAt = now
	}
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	from := task.Stage
	var fresh *registrasi.Task
	if last {
		target, ok := l.flow.StageByTicket(TicketTransferToAnalyst)
		if !ok {
			return CompleteResult{}, fmt.Errorf("%w: tujuan %q", registrasi.ErrUnknownStage, TicketTransferToAnalyst)
		}
		if err := task.Complete(by.Identity, ActionTransferToAnalyst, now); err != nil {
			return CompleteResult{}, err
		}
		recipients, err := l.assigner.Assign(ctx, target, claim, by.Identity)
		if err != nil {
			return CompleteResult{}, fmt.Errorf("registrasi/usecase: menentukan penerima tahap %q: %w", target.ID, err)
		}
		// PIC yang dipilih router dicatat ke klaim (PICTEKNIK), seperti tahap teknis lain.
		registrasi.AdoptTechnicalPIC(&claim, target, recipients)
		next := registrasi.NewTask(l.id.New(), claim, target, recipients, now)
		fresh = &next
		claim.CurrentStage = target.ID
		claim.ClaimStatus = StatusClaimAnalyst
		claim.RequestReturn = false
	}

	note := "jaminan " + objectID + "/" + coverageID + " ditandai"
	if fresh != nil {
		note = from + " → " + fresh.Stage + " (jaminan " + objectID + "/" + coverageID + ")"
	}
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if fresh != nil {
			if err := l.task.Save(ctx, task); err != nil {
				return err
			}
			if err := l.task.Save(ctx, *fresh); err != nil {
				return err
			}
			if err := l.mirrorInbox(ctx, claim); err != nil {
				return err
			}
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "TRANSFER_KE_ANALYST",
			Actor:       by.Identity,
			At:          now,
			Note:        note,
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Claim: claim, NextTask: fresh}, nil
}

// findAnalystCoverage memeriksa kondisi tampil tombol pada satu jaminan: lini PA, jaminan ada,
// bukan PHK, dan belum ditandai. Ia mengembalikan indeks objek dan jaminannya.
func findAnalystCoverage(claim registrasi.Claim, objectID, coverageID string) (int, int, error) {
	reject := func(code registrasi.ViolationCode, msg string) (int, int, error) {
		return 0, 0, &registrasi.ValidationError{Violation: []registrasi.Violation{{Code: code, Field: "coverage_id", Message: msg}}}
	}
	if claim.Policy.Line != registrasi.LinePersonalAccident {
		return reject(registrasi.ViolationAnalystNotAllowed, "Transfer to Analyst is only available for Personal Accident claims.")
	}
	for i, o := range claim.InsuredItem {
		if o.ID != objectID {
			continue
		}
		for j, c := range o.Coverage {
			if c.ID != coverageID {
				continue
			}
			if registrasi.PHKCoverages[c.ID] {
				return reject(registrasi.ViolationAnalystNotAllowed, "Transfer to Analyst is not available for this coverage.")
			}
			if c.AnalystTransferred {
				return reject(registrasi.ViolationAnalystTransferred, "This coverage has already been transferred to Analyst.")
			}
			return i, j, nil
		}
	}
	return reject(registrasi.ViolationAnalystNotAllowed, "Coverage not found on this claim.")
}
