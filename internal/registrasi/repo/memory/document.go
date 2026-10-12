package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"claim-pnc/internal/registrasi"
)

// AddAttachment menambahkan lampiran ke catatan klaim, sehingga checklist Unggah Dokumen
// ikut menghitungnya — seperti baris DATA_ATTACHFILE pada pengisi SQL.
func (r *ClaimRecords) AddAttachment(_ context.Context, a registrasi.NewAttachment) error {
	number := strings.TrimSpace(strings.TrimPrefix(a.ClaimKey, "ASM-FW-GCNMFW-WORK "))
	if r.Attachment == nil {
		r.Attachment = map[string][]registrasi.Attachment{}
	}
	r.Attachment[number] = append(r.Attachment[number], registrasi.Attachment{
		ID:   fmt.Sprintf("%d", len(r.Attachment[number])+1),
		Name: a.Name, MimeType: a.Extension, Note: a.Note, Category: a.Category,
		SubCategory: a.SubCategory, ImageID: a.ImageID, UploadedBy: a.By, UploadedAt: a.At,
	})
	return nil
}

// DeleteAttachment menghapus lampiran ber-IMAGEID itu dari seluruh klaim.
func (r *ClaimRecords) DeleteAttachment(_ context.Context, imageID string) error {
	for number, list := range r.Attachment {
		kept := list[:0]
		for _, a := range list {
			if a.ImageID != imageID {
				kept = append(kept, a)
			}
		}
		r.Attachment[number] = kept
	}
	return nil
}

var _ registrasi.AttachmentStore = (*ClaimRecords)(nil)

// DocumentUploader adalah layanan penyimpanan palsu: ia merekam berkas dan menerbitkan
// IMAGEID berurutan.
type DocumentUploader struct {
	mu    sync.Mutex
	Files []registrasi.DocumentFile

	// Links adalah alamat berkas menurut IMAGEID, dipasang uji.
	Links map[string]registrasi.DocumentLink

	// Removed merekam IMAGEID yang dihapus; RemoveErr membuat penghapusan gagal.
	Removed   []string
	RemoveErr error
}

// Remove merekam penghapusan, atau gagal dengan RemoveErr.
func (u *DocumentUploader) Remove(_ context.Context, _, imageID, _ string) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.RemoveErr != nil {
		return u.RemoveErr
	}
	u.Removed = append(u.Removed, imageID)
	return nil
}

// Upload merekam berkas dan mengembalikan IMAGEID.
func (u *DocumentUploader) Upload(_ context.Context, f registrasi.DocumentFile) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Files = append(u.Files, f)
	return fmt.Sprintf("IMG-%d", len(u.Files)), nil
}

// Link mengembalikan alamat yang dipasang uji pada Links menurut IMAGEID; kosong bila tidak ada.
func (u *DocumentUploader) Link(_ context.Context, _, imageID, _ string) (registrasi.DocumentLink, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.Links[imageID], nil
}

var (
	_ registrasi.DocumentUploader = (*DocumentUploader)(nil)
	_ registrasi.DocumentLinker   = (*DocumentUploader)(nil)
	_ registrasi.DocumentRemover  = (*DocumentUploader)(nil)
)
