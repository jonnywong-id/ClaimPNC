package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ActionSendToInputor adalah nama local action tombol "Kirim ke Inputor"
// (`Flow Action/AnalystRemarks-FA.xml`). Ia dicatat sebagai tindakan penutup tugas.
const ActionSendToInputor = "AnalystRemarks"

// TicketSendToInputor adalah Ticket rule yang dipasang tombol Kirim di
// `Section/AnalystRemarks_sect`: `SetTicket(Ticket = setToRegister_ticket)`. Tujuannya tahap
// Input Register.
const TicketSendToInputor = "setToRegister_ticket"

// maxCommunicationMessage adalah panjang kolom M_KOMUNIKASI_PNC.MESSAGE (VARCHAR2 4000).
const maxCommunicationMessage = 4000

// sendToInputorStages adalah tahap yang layarnya `ClaimSurvey_sect` — tempat tombol
// "Kirim ke Inputor" berada.
var sendToInputorStages = map[string]bool{
	registrasi.StageChooseSurveyor:     true,
	registrasi.StageSendToTechnicalPIC: true,
	registrasi.StageEstimatePA:         true,
	registrasi.StageInvestigator:       true,
	registrasi.StageSendToAnalyst:      true,
}

// SendToInputorCommand adalah isi modal `AnalystRemarks`.
type SendToInputorCommand struct {
	TaskID string

	// Note adalah `.ClaimData.AnaylstRemarks` — Text Area modal. Section tidak mewajibkannya
	// (`pyRequired=false`); bila kosong, tidak ada baris komunikasi yang ditulis.
	Note string
}

// SendToInputor menjalankan tombol Kirim pada modal "Kirim ke Inputor".
//
// Urutan Pega (`Section/AnalystRemarks_sect`): `sendToInputor_act(sendToInvest="SENDTOINPUTOR")`,
// lalu `SetTicket(setToRegister_ticket)`, lalu Save. Tugas berjalan ditutup dan klaim melompat
// ke Input Register, yang dirutekan PNCAdminRouter ke Inputor (pembuat klaim).
//
// `sendToInputor_act` TIDAK ADA di export: yang diketahui darinya hanya parameter
// SENDTOINPUTOR dan bahwa layar Input Register menampilkan catatannya ("Catatan dari Analyst",
// `ViewInputRegisterDetail`). Catatannya ditulis sebagai baris komunikasi berkanal
// ChannelSendToInputor (Work Owner, 2026-10-03), bukan kolom baru — status klaim tidak diubah.
func (l *Service) SendToInputor(ctx context.Context, p SendToInputorCommand, by Caller) (CompleteResult, error) {
	note := strings.TrimSpace(p.Note)
	if len([]rune(note)) > maxCommunicationMessage {
		return CompleteResult{}, &registrasi.ValidationError{Violation: []registrasi.Violation{{
			Code: registrasi.ViolationNoteTooLong, Field: "catatan", Message: fmt.Sprintf("Catatan paling banyak %d karakter.", maxCommunicationMessage),
		}}}
	}

	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID})
	if err != nil {
		return CompleteResult{}, err
	}
	if !sendToInputorStages[claim.CurrentStage] {
		return CompleteResult{}, fmt.Errorf("%w: Kirim ke Inputor tidak tersedia pada tahap %q",
			registrasi.ErrInvalidAction, claim.CurrentStage)
	}

	target, ok := l.flow.StageByTicket(TicketSendToInputor)
	if !ok {
		return CompleteResult{}, fmt.Errorf("%w: tujuan %q", registrasi.ErrUnknownStage, TicketSendToInputor)
	}

	now := l.clock.Now().UTC()
	if err := task.Complete(by.Identity, ActionSendToInputor, now); err != nil {
		return CompleteResult{}, err
	}
	recipients, err := l.assigner.Assign(ctx, target, claim, by.Identity)
	if err != nil {
		return CompleteResult{}, fmt.Errorf("registrasi/usecase: menentukan penerima tahap %q: %w", target.ID, err)
	}
	fresh := registrasi.NewTask(l.id.New(), claim, target, recipients, now)

	from := task.Stage
	claim.CurrentStage = target.ID
	claim.RequestReturn = false
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if err := l.task.Save(ctx, fresh); err != nil {
			return err
		}
		if note != "" {
			if err := l.records.AddCommunication(ctx, registrasi.NewCommunication{
				ClaimID:     claim.ID,
				ClaimNumber: claim.Number,
				Sender:      by.Identity,
				SenderName:  by.Name,
				Message:     note,
				Recipient:   recipients.Operator,
				Channel:     registrasi.ChannelSendToInputor,
				Status:      registrasi.CommunicationStatusOpen,
				At:          now,
			}); err != nil {
				return err
			}
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "KIRIM_KE_INPUTOR",
			Actor:       by.Identity,
			At:          now,
			Note:        from + " → " + target.ID,
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Claim: claim, NextTask: &fresh}, nil
}
