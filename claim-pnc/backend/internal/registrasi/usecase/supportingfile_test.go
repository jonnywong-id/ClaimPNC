package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

func supportingFiles(task registrasi.Task, files ...usecase.AcceptanceFile) usecase.SupportingFilesCommand {
	return usecase.SupportingFilesCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1, Files: files,
	}
}

// Unggah File Penunjang: setiap berkas terkirim ke penyimpanan dan tercatat sebagai lampiran
// klaim dengan jenis dokumennya (`SetUploadDokumenAdjustment_Act`).
func TestUploadSupportingFilesStoresEachFile(t *testing.T) {
	l := setup(t)
	task, claim := l.approvedClaim(t)

	saved, err := l.service.UploadSupportingFiles(context.Background(), supportingFiles(task,
		usecase.AcceptanceFile{DocumentTypeID: "14904", FileName: "survei.pdf", Content: []byte("%PDF"), Note: "laporan"},
		usecase.AcceptanceFile{DocumentTypeID: "14904", FileName: "foto.jpg", Content: []byte("jpg")},
	), l.caller)
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Len(t, l.uploader.Files, 2)

	stored := l.records.Attachment[claim.Number]
	require.Len(t, stored, 2)
	require.Equal(t, "laporan", stored[0].Note)
	require.Equal(t, "14904", stored[0].SubCategory)
	require.NotEmpty(t, stored[0].ImageID)
}

// Jenis dokumen di luar checklist lini bisnis ditolak sebelum apa pun terkirim.
func TestUploadSupportingFilesRejectsUnknownType(t *testing.T) {
	l := setup(t)
	task, claim := l.approvedClaim(t)

	_, err := l.service.UploadSupportingFiles(context.Background(), supportingFiles(task,
		usecase.AcceptanceFile{DocumentTypeID: "99999", FileName: "x.pdf", Content: []byte("%PDF")},
	), l.caller)
	require.ErrorIs(t, err, registrasi.ErrDocumentTypeUnknown)
	require.Empty(t, l.uploader.Files)
	require.Empty(t, l.records.Attachment[claim.Number])
}

// Tanpa berkas, dan baris adjustment yang tidak ada, keduanya ditolak.
func TestUploadSupportingFilesValidatesInput(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	ctx := context.Background()

	_, err := l.service.UploadSupportingFiles(ctx, supportingFiles(task), l.caller)
	require.ErrorIs(t, err, registrasi.ErrDocumentFileEmpty)

	cmd := supportingFiles(task, usecase.AcceptanceFile{DocumentTypeID: "14904", FileName: "x.pdf", Content: []byte("%PDF")})
	cmd.Adjustment = 9
	_, err = l.service.UploadSupportingFiles(ctx, cmd, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
	require.Empty(t, l.uploader.Files)
}
