package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// SupportingFilesCommand adalah tombol "Unggah File Penunjang" pada satu baris adjustment di
// tab Adjustment & Akseptasi. Object, Coverage, dan Adjustment berbasis 1.
type SupportingFilesCommand struct {
	ClaimID    string
	TaskID     string
	Portal     string
	Object     int
	Coverage   int
	Adjustment int
	Files      []AcceptanceFile
}

// UploadSupportingFiles menyimpan berkas dari modal "Unggah File Penunjang".
//
// # Padanannya di Pega
//
// Tombolnya (`Section/InputAdjustment_sect.xml:82677`) membuka local action
// `UploadDokumen_Adj` (kelas `ASM-FW-GCNMFW-Data-Adjustment`) dalam modal. Flow action itu
// TIDAK ada di export, begitu pula `RemoveDragdrop` dan `Call_OCR_LOD` yang dijalankan tombol
// sesudahnya. Yang ada adalah activity penyimpannya, `SetUploadDokumenAdjustment_Act`: untuk
// setiap berkas di `dragDropFileUpload.pxResults` ia memanggil `PNCSaveAttachmentToDB`
// (jenis dokumen `.pyFileType`, catatan `.pyNote`) lalu mencatat berkasnya pada
// `AdjustmentList(n).DocumentList`, kemudian Obj-Save dan Commit — berdiri sendiri, tidak
// menunggu Submit akseptasi. Itu sebabnya tombol ini tetap berguna setelah klaim dibayar.
//
// # Yang sama dengan unggahan lain di modul ini
//
// Berkas dikirim ke layanan penyimpanan lebih dulu (`D-16`, IMAGEID diterbitkan layanan),
// lalu baris DATA_ATTACHFILE disisipkan — jalur yang sama dengan Unggah Dokumen dan unggahan
// akseptasi. Jenis dokumen diperiksa terhadap checklist lini bisnis klaim.
//
// # Yang berbeda dari Pega, disengaja
//
// Pega meng-commit per berkas, sehingga berkas yang gagal di tengah meninggalkan berkas
// sebelumnya tersimpan. Di sini seluruh baris lampiran disisipkan dalam SATU transaksi,
// sama seperti unggahan akseptasi: semua tercatat atau tidak satu pun. Kaitan ke baris
// adjustment (`DocumentList` di clipboard Pega) tidak punya kolom di DATA_ATTACHFILE, jadi
// ia dicatat di jejak audit.
func (l *Service) UploadSupportingFiles(ctx context.Context, p SupportingFilesCommand, by Caller) ([]registrasi.Attachment, error) {
	if len(p.Files) == 0 {
		return nil, registrasi.ErrDocumentFileEmpty
	}
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return nil, err
	}
	if _, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment); err != nil {
		return nil, err
	}

	now := l.clock.Now().UTC()
	rows, err := l.uploadAcceptanceFiles(ctx, portalOf(p.Portal, claim.Portal), claim, p.Files, by, now)
	if err != nil {
		return nil, err
	}

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		for _, row := range rows {
			if err := l.attachments.AddAttachment(ctx, row); err != nil {
				return err
			}
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "FILE_PENUNJANG_ADJUSTMENT", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Unggah File Penunjang — objek %d jaminan %d adjustment %d, %d berkas",
				p.Object, p.Coverage, p.Adjustment, len(rows)),
		})
	})
	if err != nil {
		// Berkasnya SUDAH di penyimpanan; mengulang meninggalkan salinan yatim.
		return nil, &registrasi.DocumentUploadError{
			Kind: registrasi.UploadHalfDone,
			Message: "Berkas sudah terkirim ke penyimpanan tetapi catatan lampiran klaim gagal disimpan. " +
				"JANGAN unggah ulang — laporkan ke administrator.",
			Err: err,
		}
	}

	saved := make([]registrasi.Attachment, 0, len(rows))
	for _, row := range rows {
		saved = append(saved, registrasi.Attachment{
			Name: row.Name, MimeType: row.Extension, Note: row.Note, Category: row.Category,
			SubCategory: row.SubCategory, ImageID: row.ImageID, UploadedBy: row.By, UploadedAt: row.At,
		})
	}
	return saved, nil
}
