package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetaildokumentravel"
)

func TestRepoListSortsAndDropsCoverages(t *testing.T) {
	repo := NewRepo(SampleList()...)

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 4)
	require.Equal(t, []string{"00001", "00002", "00003", "00004"},
		[]string{rows[0].ID, rows[1].ID, rows[2].ID, rows[3].ID})
	// Daftar tidak membawa coverage meski barisnya punya.
	for _, row := range rows {
		require.Nil(t, row.Coverages)
	}
}

func TestRepoTrimsSeededRows(t *testing.T) {
	// Baris semaian dengan spasi di kedua ujung dipangkas sebelum disimpan.
	repo := NewRepo(daftardetaildokumentravel.Detail{ID: " 00009 ", DocumentID: " D1 "})

	row, err := repo.Get(context.Background(), "00009")
	require.NoError(t, err)
	require.Equal(t, "00009", row.ID)
	require.Equal(t, "D1", row.DocumentID)
}

func TestRepoGetReturnsCopyWithCoverages(t *testing.T) {
	repo := NewRepo(SampleList()...)

	row, err := repo.Get(context.Background(), " 00003 ")
	require.NoError(t, err)
	require.Len(t, row.Coverages, 2)

	// Mengubah salinan tidak boleh mengubah isi repo.
	row.Coverages[0].PlanName = "diubah"
	again, err := repo.Get(context.Background(), "00003")
	require.NoError(t, err)
	require.Equal(t, "Travel Plan Silver", again.Coverages[0].PlanName)

	_, err = repo.Get(context.Background(), "99999")
	require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
}

func TestRepoInsertNewContinuesSequenceAfterSeededCoverages(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.InsertNew(context.Background(), daftardetaildokumentravel.Input{
		DocumentID:   "100006",
		DocumentName: "Surat Keterangan Maskapai",
		Mandatory:    true,
		MinUpload:    3,
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: "TP03", PlanName: "Platinum", CoverageID: "TC04", CoverageName: "Pembatalan"},
		},
	})
	require.NoError(t, err)
	// Nomor urut terbesar pada sampel adalah 00007 (milik coverage), jadi baris baru 00008.
	require.Equal(t, "00008", saved.ID)
	require.Len(t, saved.Coverages, 1)
	require.Equal(t, "00009", saved.Coverages[0].ID)
	require.Equal(t, "TC04", saved.Coverages[0].CoverageID)

	got, err := repo.Get(context.Background(), "00008")
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
	require.Nil(t, saved.Coverages)
}

func TestRepoUpdateReplacesCoverages(t *testing.T) {
	repo := NewRepo(SampleList()...)

	saved, err := repo.Update(context.Background(), " 00003 ", daftardetaildokumentravel.Input{
		DocumentID:   "100004",
		DocumentName: "Laporan Bagasi",
		MinUpload:    1,
	})
	require.NoError(t, err)
	require.Equal(t, "00003", saved.ID)
	require.Equal(t, "Laporan Bagasi", saved.DocumentName)
	require.Nil(t, saved.Coverages)

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

func TestPlanRepoListsSortedAndFails(t *testing.T) {
	repo := NewPlanRepo(SamplePlanList(), SampleCoverageList())

	plans, err := repo.ListPlans(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"Travel Plan Gold", "Travel Plan Platinum", "Travel Plan Silver"},
		[]string{plans[0].Name, plans[1].Name, plans[2].Name})

	coverages, err := repo.ListCoverages(context.Background())
	require.NoError(t, err)
	require.Len(t, coverages, 7)
	require.Equal(t, "TP01", coverages[0].PlanID)
	require.Equal(t, "Biaya Pengobatan Darurat", coverages[0].Name)
	require.Equal(t, "Kehilangan Bagasi", coverages[1].Name)
	require.Equal(t, "TP03", coverages[6].PlanID)
	require.Equal(t, "Pembatalan Perjalanan", coverages[6].Name)

	boom := errors.New("boom")
	repo.SetError(boom)
	_, err = repo.ListPlans(context.Background())
	require.ErrorIs(t, err, boom)
	_, err = repo.ListCoverages(context.Background())
	require.ErrorIs(t, err, boom)
}
