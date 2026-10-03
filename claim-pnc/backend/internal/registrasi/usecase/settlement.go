package usecase

import (
	"context"
	"fmt"
	"strings"

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

	// Adjustment adalah nomor baris (berbasis 1) yang diubah — hanya UpdateSettlement.
	Adjustment int
}

// SettlementPreview adalah hasil hitungan baris yang belum disimpan, beserta bahan
// tampilan layar InputAdjustment: Spreading jaminan.
type SettlementPreview struct {
	Line      registrasi.SettlementLine
	Spreading []registrasi.Spreading
}

// settlementStages adalah tahap yang layarnya memuat grid Adjustment — tahap yang ditutup flow
// action InputSurveyor (layar ClaimSurvey_sect): Choose Surveyor (Non-MBU), Send To PIC Teknik
// (Travel), serta Estimation (Assignment4) dan Send To Analis (Assignment5) untuk PA.
var settlementStages = map[string]bool{
	registrasi.StageChooseSurveyor:     true,
	registrasi.StageSendToTechnicalPIC: true,
	registrasi.StageEstimatePA:         true,
	registrasi.StageSendToAnalyst:      true,
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
		return settlementScope{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, task.Stage)
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
		Analyst: registrasi.IsAnalyst(by.Roles),
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

// PrepareSettlement menjalankan tombol Tambah grid Adjustment — `ValidationAdjustment` sebelum
// baris baru dibuat. Untuk lini PA pada jaminan yang belum punya adjustment, ia menambahkan
// estimasi `NewEstimationPA` (TSI jaminan) supaya Claim Face Sheet dapat dibuat. Lini lain tidak
// mengubah apa pun.
func (l *Service) PrepareSettlement(ctx context.Context, p SettlementCommand, by Caller) (registrasi.Claim, error) {
	s, err := l.settlementScopeOf(ctx, p, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	now := s.context.Now
	if !registrasi.NewEstimationPA(s.claim, s.coverage, now) {
		return s.claim, nil
	}
	claim := s.claim
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "ESTIMASI_PA",
			Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Objek %d jaminan %d: estimasi NewEstimationPA sebesar TSI %d sen", p.Object, p.Coverage, int64(s.coverage.TSI)),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return claim, nil
}

// UpdateSettlement menghitung, memeriksa, lalu menyimpan ulang satu baris Adjustment yang sudah
// ada — padanan `SetNilaiResikoSendiri` yang Pega jalankan pada setiap perubahan isian
// `InputAdjustment` lalu diakhiri `Obj-Save` (step 55). Pelanggaran dikembalikan dan baris tidak
// disimpan, seperti Obj-Save Pega yang gagal selama halaman memuat pesan.
//
// Isian section nonaktif begitu `.AcceptanceStatus != ”`; di sini baris yang sudah diakseptasi,
// ditransfer ke komite, atau ditransfer ke kasir ditolak.
func (l *Service) UpdateSettlement(ctx context.Context, p SettlementCommand, by Caller) (registrasi.Claim, error) {
	s, err := l.settlementScopeOf(ctx, p, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	if p.Adjustment < 1 || p.Adjustment > len(s.coverage.Settlement) {
		return registrasi.Claim{}, fmt.Errorf("%w: adjustment %d tidak ada", registrasi.ErrInvalidAction, p.Adjustment)
	}
	existing := s.coverage.Settlement[p.Adjustment-1]
	if strings.TrimSpace(existing.AcceptanceStatus) != "" || existing.Transferred() || !existing.CashierTransferredAt.IsZero() {
		return registrasi.Claim{}, fmt.Errorf("%w: adjustment %d sudah diproses dan tidak dapat diubah", registrasi.ErrInvalidAction, p.Adjustment)
	}

	// Pemeriksaan total memakai baris lain jaminan ini, tanpa baris yang sedang diubah.
	others := make([]registrasi.SettlementLine, 0, len(s.coverage.Settlement)-1)
	others = append(others, s.coverage.Settlement[:p.Adjustment-1]...)
	others = append(others, s.coverage.Settlement[p.Adjustment:]...)
	sc := s.context
	sc.Coverage.Settlement = others

	line, err := registrasi.NewSettlementLine(s.input, sc)
	if err != nil {
		return registrasi.Claim{}, err
	}
	line.CreatedAt = existing.CreatedAt
	line.AcceptanceLODStatus = existing.AcceptanceLODStatus
	line.Acceptance = existing.Acceptance
	if line.Chronology == "" {
		line.Chronology = existing.Chronology
	}
	if line.Notes == "" {
		line.Notes = existing.Notes
	}
	s.coverage.Settlement[p.Adjustment-1] = line

	now := s.context.Now
	claim := s.claim
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "ADJUSTMENT_DIUBAH",
			Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Objek %d jaminan %d adjustment %d: %s, nilai ASM %d sen",
				p.Object, p.Coverage, p.Adjustment, registrasi.PaymentTypeName(line.PaymentType), int64(line.Value)),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return claim, nil
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
		if err := l.mirrorInbox(ctx, claim); err != nil {
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
