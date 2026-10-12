package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tombol Unggah Dokumen: berkas ke layanan penyimpanan, lalu satu baris DATA_ATTACHFILE
// yang membuat "Total Sudah Diunggah" bertambah.
func TestUploadDocumentCountsInTheChecklist(t *testing.T) {
	ctx := context.Background()
	l := setup(t)

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	claim := start.Claim
	// Checklist dibaca menurut kode bisnis polis; polis contoh tidak membawanya.
	claim.Policy.BusinessCode = "10027"
	require.NoError(t, l.store.Save(ctx, claim))

	attachment, err := l.service.UploadDocument(ctx, usecase.UploadDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", DocumentTypeID: "14901",
		FileName: "Pelaporan Klaim.PDF", Content: []byte("%PDF-1.4"), Note: "dari tertanggung",
	}, l.caller)
	require.NoError(t, err)

	// Kategori dan sub-kategori dari master, bukan dari permintaan; ekstensi huruf kecil.
	require.Equal(t, "10064", attachment.Category)
	require.Equal(t, "14901", attachment.SubCategory)
	require.Equal(t, "pdf", attachment.MimeType)
	require.Equal(t, "IMG-1", attachment.ImageID)

	// Berkasnya dikirim atas nomor klaim dan portal aktif.
	require.Len(t, l.uploader.Files, 1)
	require.Equal(t, claim.Number, l.uploader.Files[0].ClaimNumber)
	require.Equal(t, "ASM", l.uploader.Files[0].Portal)

	view, err := l.service.Documents(ctx, claim.ID)
	require.NoError(t, err)
	var uploaded int
	for _, c := range view.Category {
		for _, row := range c.Row {
			if row.Type.ID == "14901" {
				uploaded = row.Uploaded
			}
		}
	}
	require.Equal(t, 1, uploaded, "Total Sudah Diunggah harus bertambah")
}

func TestUploadDocumentRejectsTypeOutsideTheChecklist(t *testing.T) {
	ctx := context.Background()
	l := setup(t)

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)

	_, err = l.service.UploadDocument(ctx, usecase.UploadDocumentCommand{
		ClaimID: start.Claim.ID, DocumentTypeID: "99999", FileName: "a.pdf", Content: []byte("x"),
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrDocumentTypeUnknown)
	require.Empty(t, l.uploader.Files, "berkas tidak boleh terkirim untuk jenis yang tidak sah")
}
