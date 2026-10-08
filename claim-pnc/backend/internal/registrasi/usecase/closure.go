package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"claim-pnc/internal/registrasi"
)

// CloseClaimCommand adalah tombol Ya pada dialog "Prevent Close Claim".
type CloseClaimCommand struct {
	TaskID  string
	Closure registrasi.Closure
}

// CloseClaim menjalankan `Activity/CloseClaim-act.xml` — lihat registrasi/closure.go untuk cakupannya.
//
// Kotak tutup sementara di Pega menulis penanda dan log begitu diubah (`SetStatusCloseSementara`,
// event change), bahkan sebelum Ya ditekan. Di sini keduanya ditulis saat Ya, supaya tidak ada
// yang berubah sebelum dikonfirmasi.
func (l *Service) CloseClaim(ctx context.Context, p CloseClaimCommand, by Caller) (CompleteResult, error) {
	c := registrasi.Closure{
		Note:      strings.TrimSpace(p.Closure.Note),
		Proposal:  strings.TrimSpace(p.Closure.Proposal),
		Effort:    strings.TrimSpace(p.Closure.Effort),
		Obstacle:  strings.TrimSpace(p.Closure.Obstacle),
		Temporary: p.Closure.Temporary,
	}
	if err := closureLengths(c); err != nil {
		return CompleteResult{}, err
	}

	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID})
	if err != nil {
		return CompleteResult{}, err
	}
	if !sendToInputorStages[claim.CurrentStage] {
		return CompleteResult{}, fmt.Errorf("%w: Tutup Klaim tidak tersedia pada tahap %q",
			registrasi.ErrInvalidAction, claim.CurrentStage)
	}
	if claim.PendingClose, err = l.closures.PendingClose(ctx, claim.ID); err != nil {
		return CompleteResult{}, err
	}
	if claim.PendingClose {
		return CompleteResult{}, fmt.Errorf("%w: klaim sudah ditutup sementara", registrasi.ErrInvalidAction)
	}

	legacy, err := l.closures.LegacyOperator(ctx, by.Identity)
	if err != nil {
		return CompleteResult{}, err
	}
	if v := registrasi.ValidateClosure(claim, c, by.Identity, legacy); v != nil {
		return CompleteResult{}, &registrasi.ValidationError{Violation: []registrasi.Violation{*v}}
	}
	// Langkah 13 — `ValidationDLA_Act`, kecuali klaim Ex Gratia.
	if !claim.ExGratia {
		if v, err := l.closureDLA(ctx, claim); err != nil {
			return CompleteResult{}, err
		} else if v != nil {
			return CompleteResult{}, &registrasi.ValidationError{Violation: []registrasi.Violation{*v}}
		}
	}

	now := l.clock.Now().UTC()
	log := registrasi.ClosureLog{
		CaseKey: claim.Keys().Prefixed, ClaimNumber: claim.Number, User: by.Identity, Note: c.Note, At: now,
	}
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	if c.Temporary {
		// Tutup sementara: CloseClaim melompat ke label TEMP (langkah 15) — status klaim dan tugas
		// tidak disentuh.
		claim.PendingClose = true
		log.Action = registrasi.ClosureActionTemporary
		err = l.unit.Run(ctx, func(ctx context.Context) error {
			if err := l.claim.Save(ctx, claim); err != nil {
				return err
			}
			if err := l.closures.SaveClosure(ctx, claim.ID, c, time.Time{}); err != nil {
				return err
			}
			if err := l.closures.LogClosure(ctx, log); err != nil {
				return err
			}
			return l.audit.Record(ctx, registrasi.AuditTrail{
				ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "TUTUP_SEMENTARA",
				Actor: by.Identity, At: now, Note: c.Note,
			})
		})
		if err != nil {
			return CompleteResult{}, err
		}
		return CompleteResult{Claim: claim}, nil
	}

	if err := task.Complete(by.Identity, registrasi.ActionCloseClaim, now); err != nil {
		return CompleteResult{}, err
	}
	from := claim.CurrentStage
	claim.ClaimStatus = registrasi.StatusClosed
	claim.ProcessStatus = registrasi.ProcessDone
	claim.CurrentStage = ""
	log.Action = registrasi.ClosureActionClose

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if err := l.closures.SaveClosure(ctx, claim.ID, c, now); err != nil {
			return err
		}
		if pic := strings.TrimSpace(claim.TechnicalPIC); pic != "" {
			if err := l.closures.ReleaseTechnicalPIC(ctx, pic); err != nil {
				return err
			}
		}
		if err := l.closures.MarkDashboardClosed(ctx, claim.Number); err != nil {
			return err
		}
		if err := l.closures.LogClosure(ctx, log); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "TUTUP_KLAIM",
			Actor: by.Identity, At: now, Note: from + " → tutup: " + c.Note,
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Claim: claim}, nil
}

// closureDLA — `ValidationDLA_Act` langkah 9–12 untuk setiap adjustment yang sudah disetujui LOD
// (`.AcceptationStatusLOD == "1"`): DLA yang belum diprint atau belum dikirim menahan penutupan.
//
// Langkah 13–15 (koasuransi tanpa DLA, dokumen DLA belum diunggah pada entitas SIMASNET) belum
// dibawa: penanda DLA per adjustment (`.IsDLA`) dan dokumen bertipe 14940 belum dibaca modul ini.
func (l *Service) closureDLA(ctx context.Context, claim registrasi.Claim) (*registrasi.Violation, error) {
	for _, o := range claim.InsuredItem {
		for ci, cov := range o.Coverage {
			for ai, line := range cov.Settlement {
				if strings.TrimSpace(line.AcceptanceLODStatus) != registrasi.LODAgreed {
					continue
				}
				list, err := l.dla.Issued(ctx, claim.ID, o.ID, ci+1, ai+1)
				if err != nil {
					return nil, err
				}
				for _, d := range list {
					msg := ""
					switch {
					case !d.Printed:
						msg = "No DLA " + d.Number + " belum diprint"
					case !d.Sent:
						msg = "No DLA " + d.Number + " belum dikirim"
					}
					if msg != "" {
						return &registrasi.Violation{Code: registrasi.ViolationClosure, Field: "catatan_tutup", Message: msg}, nil
					}
				}
			}
		}
	}
	return nil, nil
}

// closureLengths menolak isian yang melebihi panjang kolomnya.
func closureLengths(c registrasi.Closure) error {
	var v []registrasi.Violation
	check := func(value, field string, max int) {
		if utf8.RuneCountInString(value) > max {
			v = append(v, registrasi.Violation{
				Code: registrasi.ViolationNoteTooLong, Field: field, Message: fmt.Sprintf("Max %d characters.", max),
			})
		}
	}
	check(c.Note, "catatan_tutup", registrasi.ClosureNoteMax)
	check(c.Proposal, "usulan", registrasi.ClosureShortMax)
	check(c.Effort, "effort_tutup", registrasi.ClosureShortMax)
	check(c.Obstacle, "kendala_tutup", registrasi.ClosureShortMax)
	if len(v) > 0 {
		return &registrasi.ValidationError{Violation: v}
	}
	return nil
}
