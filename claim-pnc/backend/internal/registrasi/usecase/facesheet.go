package usecase

import (
	"context"
	"fmt"
	"strconv"

	"claim-pnc/internal/registrasi"
)

// FaceSheetCommand adalah permintaan tombol "Download Claim Face Sheet" pada satu baris
// jaminan. Object dan Coverage berbasis 1, sama dengan nomor baris di layar.
type FaceSheetCommand struct {
	ClaimID  string
	TaskID   string
	Object   int
	Coverage int
}

// FaceSheetResult adalah dokumen yang diunduh.
type FaceSheetResult struct {
	FileName string
	Content  []byte
}

// DownloadFaceSheet membuat Claim Face Sheet satu jaminan, mencatat revisinya, dan
// mengunci estimasinya.
//
// Dokumen dibentuk SEBELUM transaksi dibuka: bila pembentukannya gagal, tidak ada estimasi
// yang terkunci tanpa dokumennya pernah sampai ke petugas.
func (l *Service) DownloadFaceSheet(ctx context.Context, p FaceSheetCommand, by Caller) (FaceSheetResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputEstimate})
	if err != nil {
		return FaceSheetResult{}, err
	}
	if claim.ID != p.ClaimID {
		return FaceSheetResult{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !l.canWork(task, by) {
		return FaceSheetResult{}, registrasi.ErrNotTaskOwner
	}
	if p.Object < 1 || p.Object > len(claim.InsuredItem) ||
		p.Coverage < 1 || p.Coverage > len(claim.InsuredItem[p.Object-1].Coverage) {
		return FaceSheetResult{}, fmt.Errorf("%w: objek %d jaminan %d tidak ada", registrasi.ErrInvalidAction, p.Object, p.Coverage)
	}
	object := &claim.InsuredItem[p.Object-1]
	coverage := &object.Coverage[p.Coverage-1]
	if !registrasi.HasUnprintedEstimate(*coverage) {
		return FaceSheetResult{}, registrasi.ErrFaceSheetNothingNew()
	}

	in, err := l.faceSheetInput(ctx, claim, p)
	if err != nil {
		return FaceSheetResult{}, err
	}
	sheet := registrasi.BuildFaceSheet(in)
	content, err := l.renderer.Render(sheet)
	if err != nil {
		return FaceSheetResult{}, fmt.Errorf("registrasi/usecase: membentuk Claim Face Sheet: %w", err)
	}

	last, found, err := l.faceSheet.LastRevision(ctx, claim.ID, object.ID, p.Coverage)
	if err != nil {
		return FaceSheetResult{}, err
	}
	revision := 0
	if found {
		revision = last + 1
	}
	now := in.Now
	fileName := registrasi.FaceSheetFileName(p.Object, p.Coverage, revision)

	registrasi.LockEstimates(coverage, now)
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		if err := l.faceSheet.SaveRevision(ctx, registrasi.FaceSheetRevision{
			ClaimID: claim.ID, ObjectID: object.ID, CoverageSeq: p.Coverage, Revision: revision,
			Date: now, FileName: fileName, Reserve: sheet.Reserve,
		}); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "CLAIM_FACE_SHEET",
			Actor: by.Identity, At: now,
			Note: "Claim Face Sheet objek " + strconv.Itoa(p.Object) + " jaminan " + strconv.Itoa(p.Coverage) +
				" revisi " + strconv.Itoa(revision) + "; estimasinya dikunci",
		})
	})
	if err != nil {
		return FaceSheetResult{}, err
	}
	return FaceSheetResult{FileName: "Revisi" + strconv.Itoa(revision) + ".pdf", Content: content}, nil
}

// faceSheetInput mengumpulkan data pendamping: polis (periode, CoinsList, FacOfferList),
// penyebab kerugian, nama PIC Admin, dan nama mata uang.
func (l *Service) faceSheetInput(ctx context.Context, claim registrasi.Claim, p FaceSheetCommand) (registrasi.FaceSheetInput, error) {
	in := registrasi.FaceSheetInput{
		Claim: claim, ObjectIndex: p.Object - 1, CoverageIndex: p.Coverage - 1,
		Now: l.clock.Now().UTC(), CurrencyName: map[string]string{},
	}
	var err error
	if in.Policy, err = l.policy.Get(ctx, claim.Policy.Number); err != nil {
		return in, err
	}
	if in.CoMember, err = l.faceSheet.Coinsurance(ctx, claim.Policy.Number); err != nil {
		return in, err
	}
	if in.Reinsurer, err = l.faceSheet.FacReinsurers(ctx, claim.Policy.Number); err != nil {
		return in, err
	}
	cause := claim.InsuredItem[p.Object-1].Coverage[p.Coverage-1].CauseOfLoss
	if cause != "" {
		if in.CauseOfLoss, err = l.faceSheet.CauseOfLossName(ctx, cause); err != nil {
			return in, err
		}
	}
	if in.AdminName, err = l.faceSheet.OperatorName(ctx, claim.CreatedBy); err != nil {
		return in, err
	}
	currencies, err := l.currency.Currencies(ctx)
	if err != nil {
		return in, err
	}
	for _, c := range currencies {
		in.CurrencyName[c.ID] = c.Name
	}
	return in, nil
}
