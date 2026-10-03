package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
)

var errInjected = errors.New("galat sisipan")

func TestRepoSetErrorFailsEveryMethod(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	repo.SetError(errInjected)
	ctx := context.Background()

	_, err := repo.ListParent(ctx)
	require.ErrorIs(t, err, errInjected)
	_, err = repo.List(ctx)
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, errInjected)
	_, err = repo.InsertNew(ctx, submission("A", "", "B"))
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Update(ctx, "1", submission("A", "", "B"))
	require.ErrorIs(t, err, errInjected)

	// Galat dapat dibersihkan lagi.
	repo.SetError(nil)
	_, err = repo.List(ctx)
	require.NoError(t, err)
}

func TestRepoListParentSortsByNameThenID(t *testing.T) {
	repo := memory.NewRepo([]masterpenolakan.RejectionStatus{
		{ID: "2", Name: "SAMA"}, {ID: "1", Name: "SAMA"}, {ID: "3", Name: "AWAL"},
	})
	got, err := repo.ListParent(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"3", "1", "2"}, []string{got[0].ID, got[1].ID, got[2].ID})
}

func TestRepoListSortsByParentThenID(t *testing.T) {
	repo := memory.NewRepo(nil,
		masterpenolakan.RejectionStatus2{ID: "2", ParentID: "1"},
		masterpenolakan.RejectionStatus2{ID: "1", ParentID: "1"},
		masterpenolakan.RejectionStatus2{ID: "3", ParentID: "0"},
	)
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"3", "1", "2"}, []string{got[0].ID, got[1].ID, got[2].ID})
}

func TestRepoGetTrimsAndReportsMissing(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	got, err := repo.Get(context.Background(), " 2 ")
	require.NoError(t, err)
	require.Equal(t, "PERIODE PERTANGGUNGAN SUDAH BERAKHIR", got.Name)

	_, err = repo.Get(context.Background(), "99")
	require.ErrorIs(t, err, masterpenolakan.ErrNotFound)
}

func TestRepoUpdateWithUnknownParentFails(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	_, err := repo.Update(context.Background(), "1", submission("A", "77", ""))
	require.ErrorIs(t, err, masterpenolakan.ErrParentNotFound)
	_, err = repo.InsertNew(context.Background(), submission("A", "77", ""))
	require.ErrorIs(t, err, masterpenolakan.ErrParentNotFound)
}

func TestRepoKomiteSetErrorFailsEveryMethod(t *testing.T) {
	repo := memory.NewRepoKomite(memory.SampleListKomite()...)
	repo.SetError(errInjected)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Get(ctx, "111")
	require.ErrorIs(t, err, errInjected)
	_, err = repo.InsertNew(ctx, masterpenolakan.InputKomite{Note: "A"})
	require.ErrorIs(t, err, errInjected)
	_, err = repo.Update(ctx, "111", masterpenolakan.InputKomite{Note: "A"})
	require.ErrorIs(t, err, errInjected)
}

func TestRepoKomiteGet(t *testing.T) {
	repo := memory.NewRepoKomite(memory.SampleListKomite()...)
	got, err := repo.Get(context.Background(), " 113 ")
	require.NoError(t, err)
	require.Equal(t, "OBJEK TIDAK TERCANTUM DALAM POLIS", got.Note)

	_, err = repo.Get(context.Background(), "1")
	require.ErrorIs(t, err, masterpenolakan.ErrKomiteNotFound)
}

func TestRepoKomiteListOrdersNumericThenText(t *testing.T) {
	// "9" mendahului "10" karena keduanya angka; "abc" dibandingkan sebagai teks.
	repo := memory.NewRepoKomite(
		masterpenolakan.CommitteeRejection{ID: "10"},
		masterpenolakan.CommitteeRejection{ID: "abc"},
		masterpenolakan.CommitteeRejection{ID: "9"},
	)
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "9", got[0].ID)
	require.Equal(t, "10", got[1].ID)
	require.Equal(t, "abc", got[2].ID)
}

func TestRepoKomiteInsertIntoEmptyStartsAtFirstID(t *testing.T) {
	repo := memory.NewRepoKomite()
	got, err := repo.InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "A"})
	require.NoError(t, err)
	require.Equal(t, "111", got.ID)

	got, err = repo.InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "B"})
	require.NoError(t, err)
	require.Equal(t, "112", got.ID)
}
