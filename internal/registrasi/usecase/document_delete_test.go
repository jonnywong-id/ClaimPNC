package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tombol Delete: berkas dihapus dari penyimpanan, lalu barisnya; checklist berkurang.
func TestDeleteDocumentRemovesFileThenRow(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)

	err := l.service.DeleteDocument(context.Background(), usecase.DeleteDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, []string{"IMG-1"}, l.uploader.Removed)

	view, err := l.service.Documents(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Empty(t, view.Attachment)
}

// Hanya pengunggahnya sendiri.
func TestDeleteDocumentOnlyByUploader(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)
	other := l.caller
	other.Identity = "PENGGUNA-LAIN"

	err := l.service.DeleteDocument(context.Background(), usecase.DeleteDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, other)
	require.ErrorIs(t, err, registrasi.ErrAttachmentDeleteNotAllowed)
	require.Empty(t, l.uploader.Removed)
}

// Tanpa batas waktu (Work Owner 2026-10-04): berkas lama tetap dapat dihapus pengunggahnya.
func TestDeleteDocumentLongAfterUpload(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)

	l.clock.Advance(30 * 24 * time.Hour)
	err := l.service.DeleteDocument(context.Background(), usecase.DeleteDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, []string{"IMG-1"}, l.uploader.Removed)
}

// Penyimpanan menolak: catatan lampiran TIDAK disentuh — mengikuti Pega langkah 17–19.
func TestDeleteDocumentStorageFailureKeepsRow(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)
	l.uploader.RemoveErr = errors.New("layanan mati")

	err := l.service.DeleteDocument(context.Background(), usecase.DeleteDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	require.Error(t, err)

	view, err := l.service.Documents(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Len(t, view.Attachment, 1)
}

func TestCanDeleteAttachment(t *testing.T) {
	now := time.Date(2026, 10, 4, 10, 0, 0, 0, time.UTC)
	a := registrasi.Attachment{ImageID: "IMG", UploadedBy: "jonny", UploadedAt: now.Add(-59 * time.Minute)}
	require.True(t, registrasi.CanDeleteAttachment(a, "JONNY", now))
	require.False(t, registrasi.CanDeleteAttachment(a, "LAIN", now))
	require.False(t, registrasi.CanDeleteAttachment(a, "", now))

	a.UploadedAt = now.Add(-30 * 24 * time.Hour)
	require.True(t, registrasi.CanDeleteAttachment(a, "JONNY", now), "tanpa batas waktu")

	a.UploadedAt = now
	a.ImageID = ""
	require.False(t, registrasi.CanDeleteAttachment(a, "JONNY", now), "tanpa berkas di penyimpanan")
}
