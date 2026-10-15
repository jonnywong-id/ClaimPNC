package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// AttachmentStore menyisipkan lampiran klaim ke POOLDATA.DATA_ATTACHFILE.
type AttachmentStore struct{ db *sql.DB }

// NewAttachmentStore membentuk penyimpan lampiran di atas sebuah koneksi.
func NewAttachmentStore(db *sql.DB) *AttachmentStore { return &AttachmentStore{db: db} }

// AddAttachment menyisipkan satu baris lampiran. DATAID diterbitkan ATTACHFILE_SEQ di dalam
// pernyataan yang sama — lihat lampiran_sisip.
func (s *AttachmentStore) AddAttachment(ctx context.Context, a registrasi.NewAttachment) error {
	year := fmt.Sprintf("%02d", clock.DateWIB(a.At).Year()%100)
	// INPUTDATE ditulis sebagai jam dinding WIB, sama seperti Pega (`SET_ATTACHMENT_64BIT` mengisi
	// SYSDATE, dan zona server basis data +07:00 — terverifikasi 2026-10-04). Sebelumnya ditulis
	// jam UTC, sehingga baris dari aplikasi ini tampil 7 jam lebih awal daripada baris Pega.
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("lampiran_sisip"),
		year, a.ClaimKey, emptyTextAsNil(a.By), emptyTextAsNil(a.Name), emptyTextAsNil(a.Note),
		emptyTextAsNil(a.Extension), a.ImageID, emptyTextAsNil(a.Category), emptyTextAsNil(a.SubCategory),
		wallOrNil(a.At))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan lampiran klaim %s: %w", a.ClaimKey, err)
	}
	return nil
}

// DeleteAttachment memenuhi registrasi.AttachmentStore.
func (s *AttachmentStore) DeleteAttachment(ctx context.Context, imageID string) error {
	exec := executorFrom(ctx, s.db)
	if _, err := exec.ExecContext(ctx, loadQuery("lampiran_hapus"), imageID); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menghapus lampiran %s: %w", imageID, err)
	}
	if _, err := exec.ExecContext(ctx, loadQuery("form_klaim_hapus"), imageID); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menghapus JSON_FORM_KLAIM %s: %w", imageID, err)
	}
	return nil
}

var _ registrasi.AttachmentStore = (*AttachmentStore)(nil)
