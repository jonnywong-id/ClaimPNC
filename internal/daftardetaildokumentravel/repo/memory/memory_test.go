package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetaildokumentravel"
)

func TestRepoListSortsByIDThenDocumentID(t *testing.T) {
	repo := NewRepo(SampleList()...)

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, []string{"00001", "00002", "00003", "00004"},
		[]string{rows[0].ID, rows[1].ID, rows[2].ID, rows[3].ID})
}

func TestRepoTrimsSeededRows(t *testing.T) {
	// Baris semaian dengan spasi di kedua ujung dipangkas sebelum disimpan.
	repo := NewRepo(daftardetaildokumentravel.Detail{ID: " 00009 ", DocumentID: " D1 "})

	row, err := repo.Get(context.Background(), "00009")
	require.NoError(t, err)
	require.Equal(t, "00009", row.ID)
	require.Equal(t, "D1", row.DocumentID)
}

func TestRepoGetTrimsKeyAndReportsMissingRow(t *testing.T) {
	repo := NewRepo(SampleList()...)

	row, err := repo.Get(context.Background(), " 00003 ")
	require.NoError(t, err)
	require.Equal(t, "Laporan Kehilangan Bagasi", row.DocumentName)

	_, err = repo.Get(context.Background(), "99999")
	require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
}

func TestRepoInsertNewContinuesSequenceAfterSeededRows(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.InsertNew(context.Background(), daftardetaildokumentravel.Input{
		DocumentID:   "100006",
		DocumentName: "Surat Keterangan Maskapai",
		Mandatory:    true,
		MinUpload:    3,
	})
	require.NoError(t, err)
	// Nomor urut terbesar pada sampel adalah 00004, jadi baris baru 00005.
	require.Equal(t, "00005", saved.ID)
	require.Equal(t, "Surat Keterangan Maskapai", saved.DocumentName)

	got, err := repo.Get(context.Background(), "00005")
	require.NoError(t, err)
	require.Equal(t, saved, got)
}

func TestRepoInsertNewSkipsTakenIDs(t *testing.T) {
	// ID tak-angka tidak menaikkan urutan; baris "00001" yang sudah ada harus dilompati.
	repo := NewRepo(
		daftardetaildokumentravel.Detail{ID: "ABC"},
		daftardetaildokumentravel.Detail{ID: "00001"},
	)
	repo.sequence = 0

	saved, err := repo.InsertNew(context.Background(), daftardetaildokumentravel.Input{})
	require.NoError(t, err)
	require.Equal(t, "00002", saved.ID)
}

func TestRepoUpdateReplacesRowAndReportsMissing(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.Update(context.Background(), " 00003 ", daftardetaildokumentravel.Input{
		DocumentID:   "100004",
		DocumentName: "Laporan Bagasi",
		MinUpload:    1,
	})
	require.NoError(t, err)
	require.Equal(t, "00003", saved.ID)
	require.Equal(t, "Laporan Bagasi", saved.DocumentName)

	_, err = repo.Update(context.Background(), "77777", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
}

func TestRepoSetErrorFailsEveryOperation(t *testing.T) {
	boom := errors.New("boom")
	repo := NewRepo(SampleList()...)
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "00001")
	require.ErrorIs(t, err, boom)
	_, err = repo.InsertNew(ctx, daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, boom)
	_, err = repo.Update(ctx, "00001", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, boom)
}

func TestDocumentRepoListSortsAndFails(t *testing.T) {
	repo := NewDocumentRepo(
		daftardetaildokumentravel.Document{ID: "2", Name: "B"},
		daftardetaildokumentravel.Document{ID: "1", Name: "A"},
	)

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftardetaildokumentravel.Document{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}}, rows)

	boom := errors.New("boom")
	repo.SetError(boom)
	_, err = repo.List(context.Background())
	require.ErrorIs(t, err, boom)

	require.Len(t, SampleDocumentList(), 6)
}
