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
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("lampiran_sisip"),
		year, a.ClaimKey, emptyTextAsNil(a.By), emptyTextAsNil(a.Name), emptyTextAsNil(a.Note),
		emptyTextAsNil(a.Extension), a.ImageID, emptyTextAsNil(a.Category), emptyTextAsNil(a.SubCategory),
		a.At.UTC())
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan lampiran klaim %s: %w", a.ClaimKey, err)
	}
	return nil
}

var _ registrasi.AttachmentStore = (*AttachmentStore)(nil)
