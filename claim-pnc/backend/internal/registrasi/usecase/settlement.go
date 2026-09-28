package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// SettlementCommand adalah isian baris Adjustment satu jaminan (tab Adjustment & Akseptasi
// layar InputSurveyor). Object dan Coverage berbasis 1.
type SettlementCommand struct {
	ClaimID  string
	TaskID   string
	Object   int
	Coverage int
	Input    registrasi.SettlementInput
}

// SettlementPreview adalah hasil hitungan baris yang belum disimpan, beserta bahan
// tampilan layar InputAdjustment: Spreading jaminan.
type SettlementPreview struct {
	Line      registrasi.SettlementLine
	Spreading []registrasi.Spreading
}

// settlementStages adalah tahap yang layarnya memuat grid Adjustment — layar InputSurveyor
// untuk Choose Surveyor (Non-MBU) dan Send To PIC Teknik (Travel).
var settlementStages = map[string]bool{
	registrasi.StageChooseSurveyor:     true,
	registrasi.StageSendToTechnicalPIC: true,
}

// settlementScope adalah klaim, jaminan, dan bahan hitungan sebuah permintaan adjustment.
type settlementScope struct {
	claim    registrasi.Claim
	coverage *registrasi.Coverage
	context  registrasi.SettlementContext
	input    registrasi.SettlementInput
}

func (l *Service) settlementScopeOf(ctx context.Context, p SettlementCommand, by Caller) (settlementScope, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputSurveyor})
	if err != nil {
		return settlementScope{}, err
	}
	if claim.ID != p.ClaimID {
		return settlementScope{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !settlementStages[task.Stage] {
		return settlementScope{}, registrasi.ErrStageMismatch
	}
	if !l.canWork(task, by) {
		return settlementScope{}, registrasi.ErrNotTaskOwner
	}
	if p.Object < 1 || p.Object > len(claim.InsuredItem) ||
		p.Coverage < 1 || p.Coverage > len(claim.InsuredItem[p.Object-1].Coverage) {
		return settlementScope{}, fmt.Errorf("%w: objek %d jaminan %d tidak ada", registrasi.ErrInvalidAction, p.Object, p.Coverage)
	}

	input := p.Input
	if input.Currency == "" {
		input.Currency = claim.Policy.Currency
	}
	// Kurs tanggal kejadian untuk mata uang baris dan mata uang polis (`D-48`): kurs yang
	// tidak ditemukan menolak permintaan, tidak pernah dianggap 1.
	rate, err := l.rate.Find(ctx, input.Currency, claim.DateOfLoss)
	if err != nil {
		return settlementScope{}, err
	}
	tsiRate, err := l.rate.Find(ctx, claim.Policy.Currency, claim.DateOfLoss)
	if err != nil {
		return settlementScope{}, err
	}

	s := settlementScope{claim: claim, input: input}
	s.coverage = &s.claim.InsuredItem[p.Object-1].Coverage[p.Coverage-1]
	s.context = registrasi.SettlementContext{
		Claim: s.claim, Coverage: *s.coverage, Rate: rate, TSIRate: tsiRate, Now: l.clock.Now().UTC(),
	}
	return s, nil
}

// PreviewSettlement menghitung baris tanpa memeriksa dan tanpa menyimpan — layar
// memanggilnya setiap isian berubah, seperti Pega yang menjalankan `SetNilaiResikoSendiri`
// pada perubahan field InputAdjustment.
func (l *Service) PreviewSettlement(ctx context.Context, p SettlementCommand, by Caller) (SettlementPreview, error) {
	s, err := l.settlementScopeOf(ctx, p, by)
	if err != nil {
		return SettlementPreview{}, err
	}
	return SettlementPreview{
		Line:      registrasi.ComputeSettlementLine(s.input, s.context),
		Spreading: s.coverage.Spreading,
	}, nil
}

// AddSettlement menambahkan satu baris Adjustment pada sebuah jaminan.
//
// Baris dihitung dan diperiksa registrasi.NewSettlementLine. Ia belum ditransfer ke komite:
// status akseptasi dan nomor akseptasinya kosong sampai Transfer Komite dibangun.
func (l *Service) AddSettlement(ctx context.Context, p SettlementCommand, by Caller) (registrasi.Claim, error) {
	s, err := l.settlementScopeOf(ctx, p, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	line, err := registrasi.NewSettlementLine(s.input, s.context)
	if err != nil {
		return registrasi.Claim{}, err
	}
	now := s.context.Now
	s.coverage.Settlement = append(s.coverage.Settlement, line)
	claim := s.claim
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "ADJUSTMENT_DITAMBAH",
			Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Objek %d jaminan %d adjustment %d: %s, nilai ASM %d sen",
				p.Object, p.Coverage, len(s.coverage.Settlement),
				registrasi.PaymentTypeName(line.PaymentType), int64(line.Value)),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return claim, nil
}
