package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputEstimate, alsoAction: registrasi.ActionInputSurveyor})
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

	// `IsCFS_PNC == ""`: dibaca SEBELUM estimasinya dikunci.
	firstFaceSheet := !claim.HasFaceSheet()
	registrasi.LockEstimates(coverage, now)
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.adoptTechnicalPICOnFaceSheet(ctx, &claim, by); err != nil {
			return err
		}
		// `DownloadClaimFaceSheet_act` step 14 — `AddTJobCQuota_SQL` untuk PIC akhir klaim,
		// hanya pada Claim Face Sheet pertama.
		if firstFaceSheet && registrasi.HasTechnicalPIC(claim) {
			if err := l.faceSheet.AddTechnicalPICJob(ctx, strings.TrimSpace(claim.TechnicalPIC)); err != nil {
				return err
			}
		}
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
	if in.CoMember, err = l.faceSheet.Coinsurance(ctx, claim.Policy.Number, claim.Policy.ProdKe); err != nil {
		return in, err
	}
	if in.Reinsurer, err = l.faceSheet.FacReinsurers(ctx, claim.Policy.Number, claim.Policy.ProdKe); err != nil {
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

// adoptTechnicalPICOnFaceSheet memilih PIC Teknis klaim saat Claim Face Sheet diunduh, bila
// klaim belum punya PIC — keputusan Work Owner 2026-10-08. Pega memilihnya lebih awal
// (`getRandomTeam_act` saat klaim dibuka); di sini titiknya CFS, supaya PIC tampil di layar
// Input Estimasi sebelum tombol Kirim PIC Teknik ditekan.
//
// Pemilihannya sama dengan router tahap Send To PIC Teknik (beban paling sedikit per lini),
// dan berjalan di dalam transaksi CFS: bila CFS gagal tersimpan, beban petugas tidak ikut
// naik. Saat Kirim PIC Teknik ditekan, PIC yang sudah tercatat ini yang menerima tugasnya
// (AssignedTechnicalPIC), sehingga bebannya tidak dinaikkan dua kali.
//
// Lini tanpa petugas aktif dibiarkan kosong — antrean ServicePNC bukan nama orang.
//
// Sesudah pemilihan, `DownloadClaimFaceSheet_act` menimpa PIC untuk admin JONI_1 (step 8)
// dan jaminan PA PHK (step 12) — registrasi.FaceSheetTechnicalPIC. Penimpaan itu berlaku
// juga bila klaim sudah punya PIC, sama seperti Pega.
func (l *Service) adoptTechnicalPICOnFaceSheet(ctx context.Context, claim *registrasi.Claim, by Caller) error {
	if !registrasi.HasTechnicalPIC(*claim) {
		stage, ok := l.flow.Stage(registrasi.StageSendToTechnicalPIC)
		if !ok {
			return fmt.Errorf("registrasi/usecase: tahap %q tidak ada di flow", registrasi.StageSendToTechnicalPIC)
		}
		to, err := l.assigner.Assign(ctx, stage, *claim, by.Identity)
		if err != nil {
			return fmt.Errorf("registrasi/usecase: memilih PIC Teknis: %w", err)
		}
		if strings.TrimSpace(to.Operator) != registrasi.OperatorUnassigned {
			registrasi.AdoptTechnicalPIC(claim, stage, to)
		}
	}
	if pic := registrasi.FaceSheetTechnicalPIC(*claim); pic != "" {
		claim.TechnicalPIC = pic
	}
	return nil
}
