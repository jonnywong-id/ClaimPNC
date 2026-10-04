package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// uploadedClaim mengunggah satu berkas ke klaim baru dan mengembalikan klaim beserta ID lampirannya.
func uploadedClaim(t *testing.T, l environment) (registrasi.Claim, string) {
	t.Helper()
	ctx := context.Background()
	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	claim := start.Claim
	claim.Policy.BusinessCode = "10027"
	require.NoError(t, l.store.Save(ctx, claim))
	_, err = l.service.UploadDocument(ctx, usecase.UploadDocumentCommand{
		ClaimID: claim.ID, Portal: "ASM", DocumentTypeID: "14901",
		FileName: "Pelaporan.pdf", Content: []byte("%PDF-1.4"),
	}, l.caller)
	require.NoError(t, err)
	view, err := l.service.Documents(ctx, claim.ID)
	require.NoError(t, err)
	require.Len(t, view.Attachment, 1)
	return claim, view.Attachment[0].ID
}

// Tombol Lihat dokumen: alamat dari metadata penyimpanan, selama masih berlaku.
func TestDocumentLinkReturnsValidURL(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)
	l.uploader.Links = map[string]registrasi.DocumentLink{
		"IMG-1": {URL: "https://penyimpanan.contoh/berkas", ExpiresAt: l.clock.Now().Add(time.Hour)},
	}

	link, err := l.service.DocumentLink(context.Background(), usecase.DocumentLinkCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "https://penyimpanan.contoh/berkas", link.URL)
}

// Alamat yang tetap kedaluwarsa sesudah melewati adapter (yang memperpanjangnya lewat
// NewLinkDokumenPNC) ditolak, bukan dikirim.
func TestDocumentLinkRejectsExpiredURL(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)
	l.uploader.Links = map[string]registrasi.DocumentLink{
		"IMG-1": {URL: "https://penyimpanan.contoh/berkas", ExpiresAt: l.clock.Now().Add(-time.Minute)},
	}

	_, err := l.service.DocumentLink(context.Background(), usecase.DocumentLinkCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	var expired *registrasi.DocumentLinkExpiredError
	require.ErrorAs(t, err, &expired)
}

func TestDocumentLinkWithoutRecordedURL(t *testing.T) {
	l := setup(t)
	claim, id := uploadedClaim(t, l)

	_, err := l.service.DocumentLink(context.Background(), usecase.DocumentLinkCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: id,
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrDocumentLinkEmpty)
}

// Lampiran dicari di antara lampiran klaim itu sendiri — ID lampiran klaim lain ditolak.
func TestDocumentLinkRejectsAttachmentOfAnotherClaim(t *testing.T) {
	l := setup(t)
	claim, _ := uploadedClaim(t, l)

	_, err := l.service.DocumentLink(context.Background(), usecase.DocumentLinkCommand{
		ClaimID: claim.ID, Portal: "ASM", AttachmentID: "999",
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrAttachmentNotFound)
}
