package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// UploadDocumentCommand adalah satu unggahan dari tombol "Unggah Dokumen" pada tab
// checklist klaim.
type UploadDocumentCommand struct {
	ClaimID string

	// Portal adalah portal aktif permintaan; dokumen disimpan di basis data portal itu.
	Portal string

	// DocumentTypeID adalah DOC_TYPE_DT_ID baris checklist yang tombolnya ditekan.
	DocumentTypeID string

	FileName string
	Content  []byte
	Note     string
}

// UploadDocument mengunggah satu berkas untuk sebuah jenis dokumen klaim.
//
// Urutannya mengikuti Pega: berkas dikirim ke layanan penyimpanan lebih dulu
// (`InsertDokumenPNC`) karena IMAGEID diterbitkan layanan, baru kemudian baris
// DATA_ATTACHFILE disisipkan (`PNCSaveAttachmentToDB`).
//
// Bila penyisipan gagal setelah berkasnya terkirim, berkasnya tetap ada di penyimpanan
// tanpa baris lampiran — sama seperti Pega, yang juga tidak menarik berkasnya kembali.
func (l *Service) UploadDocument(
	ctx context.Context,
	p UploadDocumentCommand,
	by Caller,
) (registrasi.Attachment, error) {
	if len(p.Content) == 0 {
		return registrasi.Attachment{}, registrasi.ErrDocumentFileEmpty
	}
	claim, err := l.claim.Get(ctx, p.ClaimID)
	if err != nil {
		return registrasi.Attachment{}, err
	}

	// Jenis dokumen harus milik checklist lini bisnis klaim ini — nilai kategori dan
	// sub-kategorinya diambil dari master, bukan dari permintaan.
	types, err := l.records.DocumentTypes(ctx, claim.Policy.BusinessCode)
	if err != nil {
		return registrasi.Attachment{}, err
	}
	var chosen *registrasi.DocumentType
	for i := range types {
		if strings.TrimSpace(types[i].ID) == strings.TrimSpace(p.DocumentTypeID) {
			chosen = &types[i]
			break
		}
	}
	if chosen == nil {
		return registrasi.Attachment{}, fmt.Errorf("%w: %q", registrasi.ErrDocumentTypeUnknown, p.DocumentTypeID)
	}

	// Nama unik seperti UploadDocumentToGoogleStorage — lihat registrasi.UploadFileName.
	name := registrasi.UploadFileName(l.clock.Now(), chosen.ID, p.FileName)
	imageID, err := l.documents.Upload(ctx, registrasi.DocumentFile{
		Portal:      portalOf(p.Portal, claim.Portal),
		ClaimNumber: claim.Number,
		FileName:    name,
		Content:     p.Content,
		By:          by.Identity,
	})
	if err != nil {
		return registrasi.Attachment{}, err
	}

	now := l.clock.Now().UTC()
	row := registrasi.NewAttachment{
		ClaimKey:    claim.Keys().Prefixed,
		Name:        registrasi.Truncate(name, registrasi.AttachmentNameMaxLength),
		Note:        registrasi.Truncate(p.Note, registrasi.AttachmentNoteMaxLength),
		Extension:   registrasi.AttachmentExtension(p.FileName),
		ImageID:     imageID,
		Category:    strings.TrimSpace(chosen.CategoryID),
		SubCategory: strings.TrimSpace(chosen.ID),
		By:          by.Identity,
		At:          now,
	}
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.attachments.AddAttachment(ctx, row); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID,
			Event:   "DOKUMEN_DIUNGGAH",
			Actor:   by.Identity,
			At:      now,
			Note:    chosen.Name + ": " + row.Name,
		})
	})
	if err != nil {
		// Berkasnya SUDAH di penyimpanan; mengulang meninggalkan salinan yatim.
		return registrasi.Attachment{}, &registrasi.DocumentUploadError{
			Kind: registrasi.UploadHalfDone,
			Message: "Berkas sudah terkirim ke penyimpanan tetapi catatan lampiran klaim gagal disimpan, " +
				"sehingga belum terhitung di checklist. JANGAN unggah ulang — laporkan ke administrator.",
			Err: err,
		}
	}

	return registrasi.Attachment{
		Name: row.Name, MimeType: row.Extension, Note: row.Note, Category: row.Category,
		SubCategory: row.SubCategory, ImageID: imageID, UploadedBy: row.By, UploadedAt: now,
	}, nil
}

// portalOf mendahulukan portal aktif permintaan, lalu portal tempat klaim dibuat.
func portalOf(active, claim string) string {
	if a := strings.TrimSpace(active); a != "" {
		return a
	}
	return strings.TrimSpace(claim)
}
