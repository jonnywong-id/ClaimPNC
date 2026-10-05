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

// DocumentLinkCommand meminta alamat baca satu lampiran klaim — tombol "Lihat dokumen".
type DocumentLinkCommand struct {
	ClaimID      string
	Portal       string
	AttachmentID string
}

// DocumentLink mengembalikan alamat baca satu lampiran klaim.
//
// Lampirannya dicari di antara lampiran KLAIM ITU (DATA_ATTACHFILE), bukan langsung menurut
// IMAGEID dari permintaan: tanpa itu, siapa pun yang mengetahui sebuah IMAGEID dapat membuka
// berkas klaim lain lewat klaim yang kebetulan ia pegang.
//
// Alamat yang kedaluwarsa diperpanjang adapter (NewLinkDokumenPNC). Bila yang kembali tetap
// sudah lewat masa berlakunya, ia ditolak, bukan dikirim: membukanya hanya menghasilkan galat
// dari layanan penyimpanan.
func (l *Service) DocumentLink(ctx context.Context, p DocumentLinkCommand, by Caller) (registrasi.DocumentLink, error) {
	claim, err := l.claim.Get(ctx, p.ClaimID)
	if err != nil {
		return registrasi.DocumentLink{}, err
	}
	files, err := l.records.Attachments(ctx, claim.Keys())
	if err != nil {
		return registrasi.DocumentLink{}, err
	}
	var imageID string
	found := false
	for _, f := range files {
		if strings.TrimSpace(f.ID) == strings.TrimSpace(p.AttachmentID) {
			imageID, found = strings.TrimSpace(f.ImageID), true
			break
		}
	}
	if !found {
		return registrasi.DocumentLink{}, registrasi.ErrAttachmentNotFound
	}
	if imageID == "" {
		return registrasi.DocumentLink{}, registrasi.ErrDocumentLinkEmpty
	}
	linker, ok := l.documents.(registrasi.DocumentLinker)
	if !ok {
		return registrasi.DocumentLink{}, registrasi.ErrDocumentLinkUnavailable
	}
	link, err := linker.Link(ctx, portalOf(p.Portal, claim.Portal), imageID, by.Identity)
	if err != nil {
		return registrasi.DocumentLink{}, err
	}
	if strings.TrimSpace(link.URL) == "" {
		return registrasi.DocumentLink{}, registrasi.ErrDocumentLinkEmpty
	}
	if !link.ExpiresAt.IsZero() && !link.ExpiresAt.After(l.clock.Now()) {
		return registrasi.DocumentLink{}, &registrasi.DocumentLinkExpiredError{ExpiresAt: link.ExpiresAt}
	}
	return link, nil
}

// CanDeleteAttachment menyatakan tombol Delete tampil untuk lampiran ini bagi pemanggil.
func (l *Service) CanDeleteAttachment(a registrasi.Attachment, by Caller) bool {
	return registrasi.CanDeleteAttachment(a, by.Identity, l.clock.Now())
}

// DeleteDocumentCommand adalah tombol Delete pada daftar berkas Lihat dokumen.
type DeleteDocumentCommand struct {
	ClaimID      string
	Portal       string
	AttachmentID string
}

// DeleteDocument menghapus satu lampiran klaim secara PERMANEN — `Activity/DeleteAttachDoc-act.xml`.
//
// Urutannya mengikuti Pega: berkas dihapus dari layanan penyimpanan lebih dulu, dan catatan
// lampirannya baru dihapus bila layanan menyatakan berkasnya terhapus. Kebalikannya akan
// meninggalkan berkas yatim di penyimpanan tanpa catatan yang menunjuknya.
//
// Syarat tombolnya diperiksa ulang di sini, bukan dipercayakan ke layar: pemanggilan langsung
// ke API tidak melewati layar.
func (l *Service) DeleteDocument(ctx context.Context, p DeleteDocumentCommand, by Caller) error {
	claim, err := l.claim.Get(ctx, p.ClaimID)
	if err != nil {
		return err
	}
	files, err := l.records.Attachments(ctx, claim.Keys())
	if err != nil {
		return err
	}
	var target *registrasi.Attachment
	for i := range files {
		if strings.TrimSpace(files[i].ID) == strings.TrimSpace(p.AttachmentID) {
			target = &files[i]
			break
		}
	}
	if target == nil {
		return registrasi.ErrAttachmentNotFound
	}
	if !l.CanDeleteAttachment(*target, by) {
		return registrasi.ErrAttachmentDeleteNotAllowed
	}
	remover, ok := l.documents.(registrasi.DocumentRemover)
	if !ok {
		return registrasi.ErrDocumentDeleteFailed
	}
	if err := remover.Remove(ctx, portalOf(p.Portal, claim.Portal), target.ImageID, by.Identity); err != nil {
		return err
	}

	now := l.clock.Now().UTC()
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.attachments.DeleteAttachment(ctx, target.ImageID); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID,
			Event:   "DOKUMEN_DIHAPUS",
			Actor:   by.Identity,
			At:      now,
			Note:    target.Name,
		})
	})
	if err != nil {
		return fmt.Errorf("%w: %v", registrasi.ErrDocumentDeleteHalfDone, err)
	}
	return nil
}

// portalOf mendahulukan portal aktif permintaan, lalu portal tempat klaim dibuat.
func portalOf(active, claim string) string {
	if a := strings.TrimSpace(active); a != "" {
		return a
	}
	return strings.TrimSpace(claim)
}
