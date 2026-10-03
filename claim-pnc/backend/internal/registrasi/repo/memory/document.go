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

var _ registrasi.AttachmentStore = (*ClaimRecords)(nil)

// DocumentUploader adalah layanan penyimpanan palsu: ia merekam berkas dan menerbitkan
// IMAGEID berurutan.
type DocumentUploader struct {
	mu    sync.Mutex
	Files []registrasi.DocumentFile
}

// Upload merekam berkas dan mengembalikan IMAGEID.
func (u *DocumentUploader) Upload(_ context.Context, f registrasi.DocumentFile) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Files = append(u.Files, f)
	return fmt.Sprintf("IMG-%d", len(u.Files)), nil
}

var _ registrasi.DocumentUploader = (*DocumentUploader)(nil)
