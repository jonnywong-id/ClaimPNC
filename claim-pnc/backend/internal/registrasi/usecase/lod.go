package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// LODCommand adalah permintaan tombol "Print LOD" satu baris adjustment. Object, Coverage,
// dan Adjustment berbasis 1; Type adalah `.PDFType` (`.IDAdjustClaim`) yang dipilih.
type LODCommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int
	Type       string
}

// LODResult adalah PDF yang diunduh.
type LODResult struct {
	FileName string
	Content  []byte
}

// LODDialog adalah isi awal dialog Print LOD: pilihan Tipe PDF (`SetTypePDFAdjustment`)
// dan isian section `PrintLODdanEmail` yang disiapkan `SetDataEmailTertanggung`.
type LODDialog struct {
	Types       []registrasi.LODType
	EmailLOD    string // ClaimData.Email + "," + ClaimData.UserTeknisEmail
	InsuredName string // Policy.QQName (isian Nama Tertanggung)
}

// LODDialog membaca isi awal dialog Print LOD sebuah klaim.
func (l *Service) LODDialog(ctx context.Context, claimID string, taskID string, by Caller) (LODDialog, error) {
	claim, _, err := l.lodTask(ctx, claimID, taskID, by)
	if err != nil {
		return LODDialog{}, err
	}
	insured, pic, err := l.pla.LODEmails(ctx, claim.Number, claim.TechnicalPIC)
	if err != nil {
		return LODDialog{}, err
	}
	return LODDialog{
		Types:       registrasi.LODTypesFor(claim.Policy),
		EmailLOD:    registrasi.LODEmailDraft(insured, pic),
		InsuredName: firstText(claim.Policy.QQName, claim.Policy.InsuredName),
	}, nil
}

// PrintLOD mencetak Letter of Discharge (flow action `PrintLODUP`).
//
// Penjaganya sama dengan tombol Transfer Komite di grid yang sama, ditambah aturan tombol
// Print LOD `ShowAdjustment_sect`. Yang ditulis hanya tanggal cetak (bila masih kosong) dan
// jenis LOD pada baris adjustment — keduanya tampil baca saja di form akseptasi — serta jejak
// audit LOD_CETAK. Activity di balik tombol Print LOD (`DownloadProposeAdjustment`) tidak ada
// di export; tanggal cetak mengikuti `AutoPrintPDFDraftLOD` langkah 1.
func (l *Service) PrintLOD(ctx context.Context, p LODCommand, by Caller) (LODResult, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return LODResult{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return LODResult{}, err
	}
	if err := registrasi.CanPrintLOD(*line, claim.Policy); err != nil {
		return LODResult{}, err
	}
	known := false
	for _, t := range registrasi.LODTypesFor(claim.Policy) {
		known = known || t.ID == strings.TrimSpace(p.Type)
	}
	if !known {
		return LODResult{}, registrasi.ErrLODTypeUnknown()
	}
	if !registrasi.LODTemplateReady(p.Type) {
		return LODResult{}, registrasi.ErrLODTemplateMissing()
	}

	members, err := l.pla.CoinsMembers(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return LODResult{}, err
	}
	names, _, err := l.currencyNames(ctx)
	if err != nil {
		return LODResult{}, err
	}
	// Alamat tertanggung hanya dicetak LOD Ex Gratia, dan tidak disimpan di T_CLAIM_PNC —
	// dibaca dari dokumen polis seperti Print PLA.
	address := ""
	if registrasi.LODPrintsInsuredAddress(p.Type) {
		policy, err := l.policy.Get(ctx, claim.Policy.Number)
		if err != nil {
			return LODResult{}, err
		}
		address = policy.DeliveryAddress
	}
	now := l.clock.Now().UTC()
	doc := registrasi.BuildLOD(*line, p.Type, members, registrasi.LODFacts{
		Currency:       firstText(names[line.Currency], line.Currency),
		PolicyNumber:   claim.Policy.Number,
		DateOfLoss:     claim.DateOfLoss,
		LossLocation:   claim.Location,
		InsuredName:    firstText(claim.Policy.QQName, claim.Policy.InsuredName),
		InsuredAddress: address,
	}, now)
	content, err := l.lodRenderer.Render(doc)
	if err != nil {
		return LODResult{}, fmt.Errorf("registrasi/usecase: membentuk LOD: %w", err)
	}
	object := claim.InsuredItem[p.Object-1]
	if err := l.acceptance.RecordLODPrint(ctx, claim.ID, object.ID, p.Coverage, p.Adjustment, now, strings.TrimSpace(p.Type)); err != nil {
		return LODResult{}, err
	}
	if err := l.audit.Record(ctx, registrasi.AuditTrail{
		ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "LOD_CETAK", Actor: by.Identity, At: now,
		Note: fmt.Sprintf("Print LOD — objek %d jaminan %d adjustment %d, Tipe PDF %s", p.Object, p.Coverage, p.Adjustment, p.Type),
	}); err != nil {
		return LODResult{}, err
	}
	return LODResult{
		FileName: fmt.Sprintf("LOD_%s_%d_%d_%d.pdf", strings.ReplaceAll(claim.Number, ".", ""), p.Object, p.Coverage, p.Adjustment),
		Content:  content,
	}, nil
}

// SetLODType menyimpan pilihan dropdown Tipe LOD kolom Adjustment, lalu memuat ulang klaim.
func (l *Service) SetLODType(ctx context.Context, p LODCommand, by Caller) (registrasi.Claim, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return registrasi.Claim{}, err
	}
	if err := registrasi.CanChooseLODType(*line, claim.Policy); err != nil {
		return registrasi.Claim{}, err
	}
	kind := strings.TrimSpace(p.Type)
	if kind != "" && registrasi.LODTypeName(kind) == "" {
		return registrasi.Claim{}, registrasi.ErrLODTypeUnknown()
	}
	object := claim.InsuredItem[p.Object-1]

	// SetShareAsmWhenPilihAdjustment: jenis Ex Gratia mengubah ExGratia dan Share ASM baris.
	policy, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return registrasi.Claim{}, err
	}
	before := line.ShareASM
	changed := registrasi.ApplyLODType(line, kind, claim.Portal, claim.Policy.TypeOfCoins, policy.Coins)

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if changed {
			if err := l.claim.Save(ctx, claim); err != nil {
				return err
			}
		}
		if err := l.acceptance.SetLODType(ctx, claim.ID, object.ID, p.Coverage, p.Adjustment, kind); err != nil {
			return err
		}
		note := fmt.Sprintf("Tipe LOD objek %d jaminan %d adjustment %d: %s", p.Object, p.Coverage, p.Adjustment, firstText(kind, "(kosong)"))
		if line.ShareASM != before {
			note += fmt.Sprintf("; Share ASM %s%% → %s%%", percentText(before), percentText(line.ShareASM))
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "LOD_TIPE", Actor: by.Identity, At: l.clock.Now().UTC(), Note: note,
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return l.claim.Get(ctx, claim.ID)
}

// percentText menulis persen e4 tanpa nol di belakang: 575000 → "57.5".
func percentText(p registrasi.Percent) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", float64(p)/10_000), "0"), ".")
}

// lodTask memeriksa tugas InputSurveyor pemanggil pada klaim itu.
func (l *Service) lodTask(ctx context.Context, claimID, taskID string, by Caller) (registrasi.Claim, registrasi.Task, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: taskID, action: registrasi.ActionInputSurveyor})
	if err != nil {
		return registrasi.Claim{}, registrasi.Task{}, err
	}
	if claim.ID != claimID {
		return registrasi.Claim{}, registrasi.Task{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, taskID, claimID)
	}
	if !settlementStages[task.Stage] {
		return registrasi.Claim{}, registrasi.Task{}, registrasi.ErrStageMismatch
	}
	if !l.canWork(task, by) {
		return registrasi.Claim{}, registrasi.Task{}, registrasi.ErrNotTaskOwner
	}
	return claim, task, nil
}
