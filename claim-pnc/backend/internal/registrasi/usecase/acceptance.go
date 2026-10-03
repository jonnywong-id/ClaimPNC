package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// AcceptanceFile adalah satu berkas "Unggah Dokumen Persetujuan LOD". DocumentTypeID adalah
// jenis dokumen yang dipilih per berkas (`ASMAttachFileList`, `.pyFileType` dari
// `BrowseLstDocType_RD`) — diperiksa terhadap checklist lini bisnis klaim seperti Unggah
// Dokumen.
type AcceptanceFile struct {
	DocumentTypeID string
	FileName       string
	Content        []byte
	Note           string
}

// AcceptanceCommand adalah tombol Simpan form Persetujuan / Akseptasi satu baris adjustment.
// Object, Coverage, dan Adjustment berbasis 1.
type AcceptanceCommand struct {
	ClaimID    string
	TaskID     string
	Portal     string
	Object     int
	Coverage   int
	Adjustment int
	Form       registrasi.AcceptanceForm
	Files      []AcceptanceFile
}

// ErrPremiumUnavailable menandai status premi yang tidak dapat dibaca. Pega tidak menangani
// kegagalan layanannya sama sekali (status lama pada case tetap dipakai); di sini akseptasi
// DITAHAN, bukan diloloskan diam-diam.
var ErrPremiumUnavailable = errors.New("premium status could not be checked")

// AcceptSettlement menjalankan Simpan Persetujuan / Akseptasi (`SetAdjustmentAcceptation`).
//
// Urutan mengikuti activity-nya: pemeriksaan isian dan aturan (langkah 2–30), pemeriksaan
// premi (36–49), penyimpanan berkas (50–59), lalu nomor akseptasi dan penulisan (62–96).
// Berkas dikirim ke penyimpanan SEBELUM transaksi karena IMAGEID diterbitkan layanan; bila
// transaksi gagal sesudahnya, berkasnya tertinggal di penyimpanan — sama seperti Pega.
func (l *Service) AcceptSettlement(ctx context.Context, p AcceptanceCommand, by Caller) (registrasi.Claim, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return registrasi.Claim{}, err
	}
	object := claim.InsuredItem[p.Object-1]
	if err := registrasi.CanAccept(*line, claim.Policy); err != nil {
		return registrasi.Claim{}, err
	}
	now := l.clock.Now().UTC()
	form := p.Form
	// Tanggal Cetak LOD baca saja di form (`.PrintDateLOD`, diisi saat Print LOD).
	form.PrintDate = line.Acceptance.Form.PrintDate
	if claim.Policy.Line == registrasi.LineTravel {
		form.PayableDate = now // langkah 5
	}

	otherDLA, err := l.acceptance.OtherDLA(ctx, claim.ID, object.ID, p.Coverage, p.Adjustment)
	if err != nil {
		return registrasi.Claim{}, err
	}
	receiver, err := registrasi.ValidateAcceptance(*line, claim.Policy, form, registrasi.AcceptanceCheck{
		Receivers: claim.Receiver, Files: len(p.Files), OtherDLA: otherDLA, Location: claim.Location,
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	app := portalOf(p.Portal, claim.Portal)
	if err := l.checkPremium(ctx, app, claim, *line, now); err != nil {
		return registrasi.Claim{}, err
	}
	attachments, err := l.uploadAcceptanceFiles(ctx, app, claim, p.Files, by, now)
	if err != nil {
		return registrasi.Claim{}, err
	}

	var result registrasi.Claim
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		for _, a := range attachments {
			if err := l.attachments.AddAttachment(ctx, a); err != nil {
				return err
			}
		}
		number, err := l.acceptance.NextNumber(ctx, now.In(clock.ZoneWIB).Year())
		if err != nil {
			return err
		}
		accepted := registrasi.Acceptance{Form: form, ReceiverName: receiver.Name, LODType: line.Acceptance.LODType}
		agreed := strings.TrimSpace(form.LODStatus) == registrasi.LODAgreed
		if agreed {
			accepted.AcceptedAt = now
			claim.ClaimStatus = registrasi.StatusClaimAccepted
		}
		line.Accept(accepted, number)
		claim.UpdatedBy, claim.UpdatedAt = by.Identity, now

		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.acceptance.Save(ctx, claim.ID, object.ID, p.Coverage, p.Adjustment, *line); err != nil {
			return err
		}
		if err := l.acceptance.AddHistory(ctx, claim.Keys().Prefixed, registrasi.AcceptanceHistoryNote(number), by.Identity, now); err != nil {
			return err
		}
		// Kedua baris progres menutup `.PosisiProgressID` adjustment — posisi AKSEPTASI yang
		// dibuka saat komite menyetujui (KomitePost_Adjustment langkah 19–21). POSISIID wajib
		// terisi (NOT NULL). DecideCommittee membukanya saat komite menyetujui; klaim yang disetujui
		// sebelum itu tidak punya posisinya, dan progres akseptasinya dilewati (dicatat di jejak
		// audit) alih-alih menggagalkan Submit.
		position, err := l.acceptance.OpenPosition(ctx, claim.Number, registrasi.PositionAcceptance)
		if err != nil {
			return err
		}
		progress := "progres AKSEPTASI ditutup"
		if position == "" {
			progress = "progres AKSEPTASI tidak ditulis: posisi AKSEPTASI terbuka tidak ada"
		}
		for _, note := range []string{registrasi.AcceptanceProgressNote1, registrasi.AcceptanceProgressNote2} {
			if position == "" {
				break
			}
			if err := l.acceptance.AddProgress(ctx, registrasi.ProgressUpdate{
				ClaimNumber: claim.Number, PositionID: position, Note: note, Progress1: registrasi.AcceptanceProgress1,
				Progress2: registrasi.AcceptanceProgress2, Position: registrasi.AcceptanceProgressDone,
				User: by.Identity, At: now,
			}); err != nil {
				return err
			}
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		result = claim
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "AKSEPTASI_LOD", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Akseptasi %s — objek %d jaminan %d adjustment %d, Persetujuan Tertanggung %s, %d berkas; %s",
				number, p.Object, p.Coverage, p.Adjustment, form.LODStatus, len(attachments), progress),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return result, nil
}

// checkPremium adalah `SetAdjustmentAcceptation` langkah 36–49.
func (l *Service) checkPremium(ctx context.Context, app string, claim registrasi.Claim, line registrasi.SettlementLine, now time.Time) error {
	caseID, err := l.acceptance.PolicyCaseID(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return err
	}
	statement, err := l.premium.Statement(ctx, app, registrasi.PremiumQuery{
		PolicyNumber: claim.Policy.Number, ProdKe: claim.Policy.ProdKe,
		RequestedAt: claim.Policy.CoverageEnd, CaseID: registrasi.PremiumCaseID(caseID),
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPremiumUnavailable, err)
	}
	status := registrasi.PremiumStatusOf(statement, now)
	if status == registrasi.PremiumPaid || status == "" {
		return nil
	}
	exemption := registrasi.PremiumExemption{
		KBRU: registrasi.KBRUSourcesOfBusiness[strings.TrimSpace(claim.Policy.SourceOfBusiness)] &&
			strings.EqualFold(app, "ASM"),
	}
	if claim.Policy.Line == registrasi.LineTravel {
		if exemption.TravelClientName, err = l.acceptance.TravelClientName(ctx, claim.Policy.Number); err != nil {
			return err
		}
	}
	if exemption.OpenProtection, err = l.acceptance.OpenProtectionApproved(ctx, claim.Policy.Number, claim.ID, claim.Number); err != nil {
		return err
	}
	if registrasi.PremiumBlocks(line, status, statement, exemption) {
		return registrasi.ErrAcceptancePremiumUnpaid()
	}
	return nil
}

// uploadAcceptanceFiles mengirim berkas ke penyimpanan dan menyiapkan baris DATA_ATTACHFILE.
func (l *Service) uploadAcceptanceFiles(ctx context.Context, app string, claim registrasi.Claim, files []AcceptanceFile, by Caller, now time.Time) ([]registrasi.NewAttachment, error) {
	if len(files) == 0 {
		return nil, nil
	}
	types, err := l.records.DocumentTypes(ctx, claim.Policy.BusinessCode)
	if err != nil {
		return nil, err
	}
	var rows []registrasi.NewAttachment
	for _, f := range files {
		if len(f.Content) == 0 {
			return nil, registrasi.ErrDocumentFileEmpty
		}
		var chosen *registrasi.DocumentType
		for i := range types {
			if strings.TrimSpace(types[i].ID) == strings.TrimSpace(f.DocumentTypeID) {
				chosen = &types[i]
				break
			}
		}
		if chosen == nil {
			return nil, fmt.Errorf("%w: %q", registrasi.ErrDocumentTypeUnknown, f.DocumentTypeID)
		}
		// Nama unik seperti SetAdjustmentAcceptation langkah 57.4 — lihat UploadFileName.
		name := registrasi.UploadFileName(l.clock.Now(), chosen.ID, f.FileName)
		imageID, err := l.documents.Upload(ctx, registrasi.DocumentFile{
			Portal: app, ClaimNumber: claim.Number, FileName: name, Content: f.Content, By: by.Identity,
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, registrasi.NewAttachment{
			ClaimKey:    claim.Keys().Prefixed,
			Name:        registrasi.Truncate(name, registrasi.AttachmentNameMaxLength),
			Note:        registrasi.Truncate(f.Note, registrasi.AttachmentNoteMaxLength),
			Extension:   registrasi.AttachmentExtension(f.FileName),
			ImageID:     imageID,
			Category:    strings.TrimSpace(chosen.CategoryID),
			SubCategory: strings.TrimSpace(chosen.ID),
			By:          by.Identity,
			At:          now.UTC(),
		})
	}
	return rows, nil
}
