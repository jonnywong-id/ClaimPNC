package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdokumentravel"
)

func TestRepoListSortsByID(t *testing.T) {
	repo := NewRepo(
		masterdokumentravel.TravelDocument{ID: " 100002 ", Name: " Tiket "},
		masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor"},
	)

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterdokumentravel.TravelDocument{
		{ID: "100001", Name: "Paspor"},
		{ID: "100002", Name: "Tiket"},
	}, rows)
}

func TestRepoGet(t *testing.T) {
	repo := NewRepo(SampleList()...)

	doc, err := repo.Get(context.Background(), " 100003 ")
	require.NoError(t, err)
	require.Equal(t, "Boarding Pass", doc.Name)

	_, err = repo.Get(context.Background(), "999999")
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)
}

func TestRepoInsertNewContinuesSequence(t *testing.T) {
	repo := NewRepo(SampleList()...)

	doc, err := repo.InsertNew(context.Background(), masterdokumentravel.Input{Name: "Visa"})
	require.NoError(t, err)
	// Sampel berakhir di 100006, jadi nomor berikutnya 100007.
	require.Equal(t, masterdokumentravel.TravelDocument{ID: "100007", Name: "Visa"}, doc)

	stored, err := repo.Get(context.Background(), "100007")
	require.NoError(t, err)
	require.Equal(t, doc, stored)
}

func TestRepoInsertNewIgnoresForeignIDsAndSkipsTaken(t *testing.T) {
	// ID tanpa awalan situs, ID hanya awalan, dan ID tak-angka tidak menaikkan urutan.
	repo := NewRepo(
		masterdokumentravel.TravelDocument{ID: "2xyz"},
		masterdokumentravel.TravelDocument{ID: "1"},
		masterdokumentravel.TravelDocument{ID: "1abc"},
	)
	// Baris "100001" disisipkan langsung supaya urutan harus melompatinya.
	repo.rows["100001"] = masterdokumentravel.TravelDocument{ID: "100001"}

	doc, err := repo.InsertNew(context.Background(), masterdokumentravel.Input{Name: "Baru"})
	require.NoError(t, err)
	require.Equal(t, "100002", doc.ID)
}

func TestRepoUpdate(t *testing.T) {
	repo := NewRepo(SampleList()...)

	require.NoError(t, repo.Update(context.Background(),
		masterdokumentravel.TravelDocument{ID: " 100001 ", Name: "Paspor Baru"}))
	doc, err := repo.Get(context.Background(), "100001")
	require.NoError(t, err)
	require.Equal(t, "Paspor Baru", doc.Name)

	err = repo.Update(context.Background(), masterdokumentravel.TravelDocument{ID: "9"})
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)
}

func TestRepoSetErrorFailsEveryOperation(t *testing.T) {
	boom := errors.New("boom")
	repo := NewRepo(SampleList()...)
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "100001")
	require.ErrorIs(t, err, boom)
	_, err = repo.InsertNew(ctx, masterdokumentravel.Input{})
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, repo.Update(ctx, masterdokumentravel.TravelDocument{ID: "100001"}), boom)
}
