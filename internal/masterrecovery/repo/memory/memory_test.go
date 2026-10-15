package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
)

func TestSampleRepoContents(t *testing.T) {
	r := NewSampleRepo()
	ctx := context.Background()

	principal, err := r.ListPrincipal(ctx)
	require.NoError(t, err)
	require.Len(t, principal, 2)
	// Terurut menurut nama: "MITRA" mendahului "PENJAMINAN".
	require.Equal(t, "PT CONTOH MITRA SEJAHTERA", principal[0].Name)
	require.Equal(t, "PT CONTOH PENJAMINAN NUSANTARA", principal[1].Name)

	// Pencarian polis tidak peduli spasi tepi dan besar-kecil huruf.
	ref, err := r.LookupPolicy(ctx, " contoh-polis-0001 ")
	require.NoError(t, err)
	require.Equal(t, masterrecovery.PolicyReference{BusinessID: "01", BranchID: "001", AgentID: "AG0001", MarketingID: "MO0001"}, ref)

	_, err = r.LookupPolicy(ctx, "TIDAK-ADA")
	require.ErrorIs(t, err, masterrecovery.ErrPolicyNotFound)
}

func TestNextBatchAndInsert(t *testing.T) {
	r := NewRepo(nil, nil)
	ctx := context.Background()

	next, err := r.NextBatch(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), next)

	line := []masterrecovery.ClaimLine{{PolicyNo: "P1", ClaimAmount: 10}}
	saved, err := r.Insert(ctx, masterrecovery.Recovery{PrincipalName: "A", ClaimLine: line})
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.Batch)
	require.False(t, saved.InputDate.IsZero())

	// Senarai yang disunting pemanggil tidak mengubah yang tersimpan.
	line[0].PolicyNo = "DIUBAH"
	require.Equal(t, "P1", r.Saved()[0].ClaimLine[0].PolicyNo)

	next, err = r.NextBatch(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), next)
}

func TestListFiltersSortsAndPagesByPrincipal(t *testing.T) {
	r := NewRepo(nil, nil)
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Disusun langsung supaya waktu input deterministik.
	r.recovery = []masterrecovery.Recovery{
		{Batch: 3, PrincipalName: "PT B", Year: "2026", InputDate: base},
		{Batch: 1, PrincipalName: "PT A", Year: "2026", InputDate: base.Add(time.Hour)},
		{Batch: 2, PrincipalName: "PT A", Year: "2026", InputDate: base},
		{Batch: 5, PrincipalName: "PT A", Year: "2026", InputDate: base},
		{Batch: 4, PrincipalName: "PT C", Year: "2025", InputDate: base, DocumentID: "D1"},
		{Batch: 6, PrincipalName: "PT C", Year: "2025", InputDate: base, DocumentID: "TIDAK-ADA"},
	}
	r.document["D1"] = masterrecovery.Document{Name: "bukti.pdf", UploadedBy: "U1"}
	r.documentAt["D1"] = base

	rows, total, err := r.List(ctx, masterrecovery.ListFilter{})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	var batch []int64
	for _, row := range rows {
		batch = append(batch, row.Batch)
	}
	// PT A (waktu menaik, batch sebagai pemutus), lalu PT B, lalu PT C.
	require.Equal(t, []int64{2, 5, 1, 3, 4, 6}, batch)
	require.NotNil(t, rows[4].Attachment)
	require.Equal(t, masterrecovery.AttachmentInfo{ID: "D1", Name: "bukti.pdf", UploadedBy: "U1", UploadedAt: base}, *rows[4].Attachment)
	require.Nil(t, rows[5].Attachment)

	// Halaman memotong principal, bukan baris.
	rows, total, err = r.List(ctx, masterrecovery.ListFilter{Limit: 1, Offset: 1})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, rows, 1)
	require.Equal(t, "PT B", rows[0].PrincipalName)

	// Offset di luar jumlah principal → kosong, total tetap dilaporkan.
	rows, total, err = r.List(ctx, masterrecovery.ListFilter{Offset: 3})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Nil(t, rows)

	// Penyaring nama (sebagian, tanpa peduli huruf) dan tahun.
	rows, total, err = r.List(ctx, masterrecovery.ListFilter{PrincipalName: " pt a ", Year: "2026"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, rows, 3)

	rows, total, err = r.List(ctx, masterrecovery.ListFilter{Year: "2025"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, rows, 2)
}

func TestFindAndSavePrincipal(t *testing.T) {
	r := NewRepo(SamplePrincipal(), nil)
	ctx := context.Background()

	p, err := r.FindPrincipal(ctx, "contoh-principal-001", "pt contoh penjaminan nusantara")
	require.NoError(t, err)
	require.Equal(t, "0000000000000001", p.VirtualAccountNumber)

	_, err = r.FindPrincipal(ctx, "BARU", "PT BARU")
	require.ErrorIs(t, err, masterrecovery.ErrPrincipalNotFound)

	require.NoError(t, r.SavePrincipal(ctx, masterrecovery.Principal{ClientID: "BARU", Name: "PT BARU"}))
	p, err = r.FindPrincipal(ctx, "BARU", "PT BARU")
	require.NoError(t, err)
	require.Equal(t, "PT BARU", p.Name)
}

func TestSaveAndFindDocument(t *testing.T) {
	r := NewRepo(nil, nil)
	ctx := context.Background()

	id, err := r.SaveDocument(ctx, masterrecovery.Document{Name: "a.pdf", Content: []byte("isi")})
	require.NoError(t, err)
	require.Equal(t, SampleYear+"0000000001", id)

	doc, err := r.FindDocument(ctx, " "+id+" ")
	require.NoError(t, err)
	require.Equal(t, "a.pdf", doc.Name)

	stored, ok := r.Document(id)
	require.True(t, ok)
	require.Equal(t, []byte("isi"), stored.Content)
	_, ok = r.Document("lain")
	require.False(t, ok)

	_, err = r.FindDocument(ctx, "tidak-ada")
	require.ErrorIs(t, err, masterrecovery.ErrDocumentNotFound)

	// Baris ada tetapi isinya kosong → isinya disimpan di tempat lain.
	emptyID, err := r.SaveDocument(ctx, masterrecovery.Document{Name: "kosong"})
	require.NoError(t, err)
	_, err = r.FindDocument(ctx, emptyID)
	require.ErrorIs(t, err, masterrecovery.ErrDocumentElsewhere)
}

func TestTenDigits(t *testing.T) {
	require.Equal(t, "0000000042", tenDigits(42))
	require.Equal(t, "12345678901", tenDigits(12345678901))
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	r := NewSampleRepo()
	want := errors.New("rusak")
	r.SetError(want)
	ctx := context.Background()

	_, err := r.NextBatch(ctx)
	require.ErrorIs(t, err, want)
	_, err = r.Insert(ctx, masterrecovery.Recovery{})
	require.ErrorIs(t, err, want)
	_, _, err = r.List(ctx, masterrecovery.ListFilter{})
	require.ErrorIs(t, err, want)
	_, err = r.FindDocument(ctx, "x")
	require.ErrorIs(t, err, want)
	_, err = r.ListPrincipal(ctx)
	require.ErrorIs(t, err, want)
	_, err = r.FindPrincipal(ctx, "a", "b")
	require.ErrorIs(t, err, want)
	require.ErrorIs(t, r.SavePrincipal(ctx, masterrecovery.Principal{}), want)
	_, err = r.LookupPolicy(ctx, "x")
	require.ErrorIs(t, err, want)
	_, err = r.SaveDocument(ctx, masterrecovery.Document{})
	require.ErrorIs(t, err, want)
}
