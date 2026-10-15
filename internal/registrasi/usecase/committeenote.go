package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// CommitteeNoteCommand adalah isian modal "Transfer Claim ke Komite" satu jaminan.
// Object dan Coverage berbasis 1.
type CommitteeNoteCommand struct {
	TaskID   string
	ClaimID  string
	Object   int
	Coverage int
	Note     registrasi.CommitteeNote
}

// SaveCommitteeNote menyimpan isian modal "Transfer Claim ke Komite"
// (`Section/ClaimComitee_OC`).
//
// Di Pega isian ini dikirim bersama tombol modalnya (Simpan → PNCSaveButton2, Kirim Komite →
// ValidationTypePayment, Kirim Analyst → setTicketToAnalyst); layar di sini memanggilnya lebih
// dulu, lalu menjalankan tombolnya. Modal dibuka dari grid Adjustment tahap teknis dan dari
// tahap Estimation PA — keduanya tahap adjustment (`settlementStages`).
func (l *Service) SaveCommitteeNote(ctx context.Context, p CommitteeNoteCommand, by Caller) (registrasi.Claim, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID})
	if err != nil {
		return registrasi.Claim{}, err
	}
	if p.ClaimID != "" && claim.ID != p.ClaimID {
		return registrasi.Claim{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !settlementStages[task.Stage] {
		return registrasi.Claim{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, task.Stage)
	}
	if !l.canWork(task, by) {
		return registrasi.Claim{}, registrasi.ErrNotTaskOwner
	}
	if p.Object < 1 || p.Object > len(claim.InsuredItem) {
		return registrasi.Claim{}, fmt.Errorf("%w: objek %d tidak ada", registrasi.ErrInvalidAction, p.Object)
	}
	item := &claim.InsuredItem[p.Object-1]
	if p.Coverage < 1 || p.Coverage > len(item.Coverage) {
		return registrasi.Claim{}, fmt.Errorf("%w: jaminan %d tidak ada", registrasi.ErrInvalidAction, p.Coverage)
	}
	if err := registrasi.ValidateCommitteeNote(p.Note); err != nil {
		return registrasi.Claim{}, err
	}

	cov := &item.Coverage[p.Coverage-1]
	note := p.Note
	note.InitialName, note.CommitteeDate = cov.Committee.InitialName, cov.Committee.CommitteeDate
	now := l.clock.Now().UTC()
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.SaveCommitteeNote(ctx, claim.ID, p.Object, p.Coverage, note); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "ISIAN_KOMITE_JAMINAN",
			Actor:       by.Identity,
			At:          now,
			Note:        fmt.Sprintf("jaminan %d/%d (%s)", p.Object, p.Coverage, cov.ID),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	cov.Committee = note
	return claim, nil
}

// SearchDiagnosis menjalankan tombol Cari modal "Transfer Claim ke Komite" (PA) —
// `CariKodeDiagnosKlaimPA` (flag kosong) → `GetKodeDiagnosaKlaimPa`.
func (l *Service) SearchDiagnosis(ctx context.Context, text string) ([]registrasi.DiagnosisOption, error) {
	if l.diagnosis == nil {
		return nil, nil
	}
	return l.diagnosis.SearchDiagnosis(ctx, text)
}
