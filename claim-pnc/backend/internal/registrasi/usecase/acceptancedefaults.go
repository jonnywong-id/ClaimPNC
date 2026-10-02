package usecase

import (
	"context"
	"errors"
	"strings"

	"claim-pnc/internal/registrasi"
)

// AcceptanceDefaultsCommand menunjuk baris adjustment yang form AcceptationLOD-nya dibuka.
// Object, Coverage, dan Adjustment berbasis 1.
type AcceptanceDefaultsCommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int
}

// AcceptanceDefaults menghitung isian yang sudah terisi saat form AcceptationLOD dibuka
// (`AcceptationLOD_PreAct`). Ia hanya membaca; aturannya di registrasi.DefaultAcceptance.
func (l *Service) AcceptanceDefaults(ctx context.Context, p AcceptanceDefaultsCommand, by Caller) (registrasi.AcceptanceDefaults, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return registrasi.AcceptanceDefaults{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return registrasi.AcceptanceDefaults{}, err
	}
	var members []registrasi.CommitteeMember
	if caseID := strings.TrimSpace(line.CommitteeCaseID); caseID != "" {
		c, err := l.committees.Get(ctx, caseID)
		switch {
		case errors.Is(err, registrasi.ErrCommitteeNotFound):
		case err != nil:
			return registrasi.AcceptanceDefaults{}, err
		default:
			members = c.Members
		}
	}
	return registrasi.DefaultAcceptance(*line, claim.Policy, claim.Location, members, l.acceptanceMultiLevel), nil
}
