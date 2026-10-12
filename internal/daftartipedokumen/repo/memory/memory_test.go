package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumen"
)

func TestNewRepoTrimsRowsAndListsSortedByID(t *testing.T) {
	repo := NewRepo(
		daftartipedokumen.DocumentType{ID: " 10002 ", Type: " B ", ProcessStatus: " s "},
		daftartipedokumen.DocumentType{ID: "10001", Type: "A"},
	)

	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumen.DocumentType{
		{ID: "10001", Type: "A"},
		{ID: "10002", Type: "B", ProcessStatus: "s"},
	}, got)
}

func TestSampleListSeedsSixRows(t *testing.T) {
	repo := NewRepo(SampleList()...)
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 6)
	require.Equal(t, "10001", got[0].ID)
	require.Equal(t, "10006", got[5].ID)
}

func TestGetTrimsIDAndReportsNotFound(t *testing.T) {
	repo := NewRepo(SampleList()...)

	doc, err := repo.Get(context.Background(), " 10003 ")
	require.NoError(t, err)
	require.Equal(t, "Dokumen Komite", doc.Type)

	_, err = repo.Get(context.Background(), "99999")
	require.ErrorIs(t, err, daftartipedokumen.ErrNotFound)
}

// Nomor urut melanjutkan ID tertinggi milik situs "1"; ID situs lain atau yang bukan
// angka tidak memengaruhi deretnya.
func TestInsertNewContinuesSequenceFromHighestID(t *testing.T) {
	repo := NewRepo(
		daftartipedokumen.DocumentType{ID: "10004"},
		daftartipedokumen.DocumentType{ID: "10002"},
		daftartipedokumen.DocumentType{ID: "29999"},
		daftartipedokumen.DocumentType{ID: "1abc"},
		daftartipedokumen.DocumentType{ID: "1"},
	)

	doc, err := repo.InsertNew(context.Background(),
		daftartipedokumen.Input{Type: "Baru", ProcessStatus: "Catatan"}, daftartipedokumen.Editor{})
	require.NoError(t, err)
	require.Equal(t, daftartipedokumen.DocumentType{ID: "10005", Type: "Baru", ProcessStatus: "Catatan"}, doc)

	stored, err := repo.Get(context.Background(), "10005")
	require.NoError(t, err)
	require.Equal(t, doc, stored)
}

func TestInsertNewOnEmptyRepoStartsAtOne(t *testing.T) {
	repo := NewRepo()
	doc, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{Type: "X"}, daftartipedokumen.Editor{})
	require.NoError(t, err)
	require.Equal(t, "10001", doc.ID)
}

func TestUpdateReplacesRowAndRejectsUnknownID(t *testing.T) {
	repo := NewRepo(SampleList()...)

	err := repo.Update(context.Background(),
		daftartipedokumen.DocumentType{ID: " 10001 ", Type: "Ubah", ProcessStatus: "Catatan"},
		daftartipedokumen.Editor{})
	require.NoError(t, err)
	doc, err := repo.Get(context.Background(), "10001")
	require.NoError(t, err)
	require.Equal(t, daftartipedokumen.DocumentType{ID: "10001", Type: "Ubah", ProcessStatus: "Catatan"}, doc)

	err = repo.Update(context.Background(), daftartipedokumen.DocumentType{ID: "99999"}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, daftartipedokumen.ErrNotFound)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	boom := errors.New("boom")
	repo := NewRepo(SampleList()...)
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "10001")
	require.ErrorIs(t, err, boom)
	_, err = repo.InsertNew(ctx, daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, repo.Update(ctx, daftartipedokumen.DocumentType{ID: "10001"}, daftartipedokumen.Editor{}), boom)

	// Mengosongkan galat memulihkan perilaku normal.
	repo.SetError(nil)
	_, err = repo.List(ctx)
	require.NoError(t, err)
}
