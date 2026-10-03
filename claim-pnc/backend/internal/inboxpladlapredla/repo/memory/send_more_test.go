package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/repo/memory"
)

const (
	sampleKey  = "ASM-FW-GCNMFW-WORK PNC-1001"
	unknownKey = "ASM-FW-GCNMFW-WORK PNC-9999"
)

func tabOf(t *testing.T, code string) inboxpladlapredla.Tab {
	t.Helper()
	tab, ok := inboxpladlapredla.FindTab(code)
	require.True(t, ok)
	return tab
}

func fixedNow() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }

// Pembaca dokumen kirim menolak kunci kosong, tab tanpa rincian, dan dokumen yang tidak ada.
func TestAdviceForSendingRejectsWhatCannotBeSent(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	_, _, err := store.AdviceForSending(ctx, tabOf(t, "pla"), " ", "PLA/2026/0001")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	_, _, err = store.AdviceForSending(ctx, tabOf(t, "pla"), sampleKey, "")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	_, _, err = store.AdviceForSending(ctx, tabOf(t, "pre-dla"), sampleKey, "PRE/2026/0001")
	require.ErrorIs(t, err, inboxpladlapredla.ErrDocumentsNotOnTab)

	_, _, err = store.AdviceForSending(ctx, tabOf(t, "pla"), sampleKey, "PLA/2026/9999")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Dokumen dan klaimnya dibaca bersama.
func TestAdviceForSendingReturnsTheAdviceAndItsClaim(t *testing.T) {
	store := memory.NewSampleStore()

	advice, claim, err := store.AdviceForSending(
		context.Background(), tabOf(t, "pla"), " "+sampleKey+" ", " PLA/2026/0001 ")
	require.NoError(t, err)

	require.Equal(t, inboxpladlapredla.SendableAdvice{
		AdviceNo: "PLA/2026/0001", AdviceType: "OR", Reinsurer: "Reasuransi Contoh A",
		ReinsurerID: "R001", Email: "reas-a@contoh.example", Sent: "",
		Login: "REASA", Country: "INDONESIA",
	}, advice)
	require.Equal(t, inboxpladlapredla.ClaimSummary{
		ClaimNo: "PNC-1001", PolicyNo: "POL-2026-0001", Insured: "PT Contoh Satu",
		Business: "Marine Cargo", LossDate: "2026-01-02",
	}, claim)
}

// Dokumen yang klaimnya tidak ada dinyatakan tidak ditemukan.
func TestAdviceForSendingWithoutItsClaimIsNotFound(t *testing.T) {
	store := memory.NewStore()
	store.Seed([]memory.Claim{{Key: sampleKey, No: "PNC-1001"}}, []memory.Advice{{
		ClaimKey: unknownKey, Kind: inboxpladlapredla.KindDLA, No: "DLA/1",
		Email: "x@contoh.example",
	}})

	_, _, err := store.AdviceForSending(
		context.Background(), tabOf(t, "dla"), unknownKey, "DLA/1")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Lampiran disaring per klaim dan kategori, yang kosong dilewati, dan urut nama.
func TestAttachmentsForClaimFiltersAndSorts(t *testing.T) {
	store := memory.NewSampleStore()
	store.SeedAttachments([]memory.Attachment{
		{ClaimKey: sampleKey, Category: "PLA", Name: "b.pdf", Content: []byte("b")},
		{ClaimKey: sampleKey, Category: "PLA", Name: "a.pdf", MIMEType: "application/pdf",
			Content: []byte("a")},
		{ClaimKey: sampleKey, Category: "PLA", Name: "kosong.pdf"},
		{ClaimKey: sampleKey, Category: "DLA", Name: "dla.pdf", Content: []byte("d")},
		{ClaimKey: unknownKey, Category: "PLA", Name: "lain.pdf", Content: []byte("l")},
	})

	items, err := store.AttachmentsForClaim(context.Background(), sampleKey, " PLA ")
	require.NoError(t, err)
	require.Equal(t, []inboxpladlapredla.Attachment{
		{Name: "a.pdf", MIMEType: "application/pdf", Content: []byte("a")},
		{Name: "b.pdf", Content: []byte("b")},
	}, items)

	none, err := store.AttachmentsForClaim(context.Background(), sampleKey, "XYZ")
	require.NoError(t, err)
	require.Empty(t, none)
	require.NotNil(t, none)

	_, err = store.AttachmentsForClaim(context.Background(), "  ", "PLA")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Penandaan terkirim mengisi tanggal kirim, tanggal terima, dan alamat; penandaan kedua
// tidak mengubah apa pun.
func TestMarkAdviceSentRecordsOnceOnly(t *testing.T) {
	store := memory.NewSampleStore()
	store.Now = fixedNow
	ctx := context.Background()
	tab := tabOf(t, "dla")

	changed, err := store.MarkAdviceSent(ctx, tab, sampleKey, "DLA/2026/0001",
		inboxpladlapredla.SendableAdvice{Email: "baru@contoh.example"})
	require.NoError(t, err)
	require.True(t, changed)

	docs, err := store.Documents(ctx, tab, sampleKey)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	require.Equal(t, "1", docs[0].Sent)
	require.Equal(t, "2026-03-10", docs[0].SentDate)
	require.Equal(t, "2026-03-10", docs[0].ReceivedDate)
	require.Equal(t, "baru@contoh.example", docs[0].Email)
	require.Equal(t, "AKS-2026-0001", docs[0].AcceptanceNo)

	again, err := store.MarkAdviceSent(ctx, tab, sampleKey, "DLA/2026/0001",
		inboxpladlapredla.SendableAdvice{Email: "lain@contoh.example"})
	require.NoError(t, err)
	require.False(t, again)

	missing, err := store.MarkAdviceSent(ctx, tab, sampleKey, "DLA/2026/9999",
		inboxpladlapredla.SendableAdvice{})
	require.NoError(t, err)
	require.False(t, missing)

	_, err = store.MarkAdviceSent(ctx, tab, "", "DLA/2026/0001",
		inboxpladlapredla.SendableAdvice{})
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Tanpa jam yang digeser, penanda memakai jam proses dan tanggal kirimnya terisi.
func TestMarkPreDLASentWithoutAFixedClockStillRecordsADate(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	changed, err := store.MarkPreDLASent(ctx, sampleKey, "PRE/2026/0001")
	require.NoError(t, err)
	require.True(t, changed)

	items, err := store.PrintPreDLA(ctx, sampleKey)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "1", items[0].Sent)
	require.NotEmpty(t, items[0].SentDate)

	_, err = store.MarkPreDLASent(ctx, " ", "PRE/2026/0001")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Panel Print Pre DLA mengganti ISKIRIM kosong dengan "0", mengurutkan nomor, dan menolak
// kunci yang kosong maupun tidak dikenal.
func TestPrintPreDLAPanelRules(t *testing.T) {
	store := memory.NewStore()
	store.Seed([]memory.Claim{{Key: sampleKey, No: "PNC-1001"}}, []memory.Advice{
		{ClaimKey: sampleKey, Kind: inboxpladlapredla.KindPreDLA, No: "PRE/2",
			AttachmentKey: "ATT-2", Sent: "1"},
		{ClaimKey: sampleKey, Kind: inboxpladlapredla.KindPreDLA, No: "PRE/1",
			AttachmentKey: "ATT-1"},
		{ClaimKey: sampleKey, Kind: inboxpladlapredla.KindPreDLA, No: "PRE/3"},
	})
	ctx := context.Background()

	items, err := store.PrintPreDLA(ctx, sampleKey)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "PRE/1", items[0].AdviceNo)
	require.Equal(t, "0", items[0].Sent)
	require.Equal(t, "PRE/2", items[1].AdviceNo)
	require.Equal(t, "1", items[1].Sent)

	_, err = store.PrintPreDLA(ctx, "")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
	_, err = store.PrintPreDLA(ctx, unknownKey)
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}

// Rincian bertanggal sama diurutkan menurut nomornya; kunci kosong tidak ditemukan.
func TestDocumentsWithTheSameDateAreOrderedByNumber(t *testing.T) {
	sameDay := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	store.Seed([]memory.Claim{{Key: sampleKey, No: "PNC-1001"}}, []memory.Advice{
		{ClaimKey: sampleKey, Kind: inboxpladlapredla.KindPLA, No: "PLA/B", Date: sameDay},
		{ClaimKey: sampleKey, Kind: inboxpladlapredla.KindPLA, No: "PLA/A", Date: sameDay},
	})

	docs, err := store.Documents(context.Background(), tabOf(t, "pla"), sampleKey)
	require.NoError(t, err)
	require.Equal(t, "PLA/A", docs[0].AdviceNo)
	require.Equal(t, "PLA/B", docs[1].AdviceNo)

	_, err = store.Documents(context.Background(), tabOf(t, "pla"), " ")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)
}
