package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterpicteknik/repo/memory"
)

var errStore = errors.New("penyimpanan rusak")

func TestListShowsOnlyActiveTechniciansSorted(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	rows, err := repo.List(context.Background())
	require.NoError(t, err)

	ids := []string{}
	for _, r := range rows {
		ids = append(ids, r.OperatorID)
	}
	require.Equal(t, []string{"PICTEKNIK01", "PICTEKNIK02", "PICTEKNIK03"}, ids,
		"yang nonaktif disembunyikan")
}

func TestGetIgnoresCase(t *testing.T) {
	repo := memory.NewRepo(masterpicteknik.Technician{OperatorID: " budi ", Name: " Budi "})

	got, err := repo.Get(context.Background(), "BUDI")
	require.NoError(t, err)
	require.Equal(t, "budi", got.OperatorID)
	require.Equal(t, "Budi", got.Name, "dibersihkan saat disimpan")

	_, err = repo.Get(context.Background(), "ANI")
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)
}

func TestInsertRefusesADuplicateIdentity(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	_, err := repo.Insert(context.Background(), masterpicteknik.Technician{OperatorID: "picteknik01"})
	require.ErrorIs(t, err, masterpicteknik.ErrAlreadyExists)

	created, err := repo.Insert(context.Background(), masterpicteknik.Technician{
		OperatorID: " BARU ", Name: " Petugas ", Active: true,
	})
	require.NoError(t, err)
	require.Equal(t, "BARU", created.OperatorID)
	require.Equal(t, "Petugas", created.Name)
}

func TestUpdateKeepsTheFieldsTheScreenDoesNotOwn(t *testing.T) {
	repo := memory.NewRepo(masterpicteknik.Technician{
		OperatorID: "BUDI", PanelGroup: "006", Workload: 7, Quota: 5, Active: true,
	})

	updated, err := repo.Update(context.Background(), masterpicteknik.Technician{
		OperatorID: "budi", PanelGroup: "999", Workload: 0, Quota: 9, Name: "Budi",
	})
	require.NoError(t, err)
	require.Equal(t, "BUDI", updated.OperatorID, "ejaan identitas lama dipertahankan")
	require.Equal(t, "006", updated.PanelGroup)
	require.Equal(t, 7, updated.Workload)
	require.Equal(t, 9, updated.Quota)

	stored, _ := repo.Get(context.Background(), "BUDI")
	require.Equal(t, "Budi", stored.Name)

	_, err = repo.Update(context.Background(), masterpicteknik.Technician{OperatorID: "ANI"})
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(errStore)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errStore)
	_, err = repo.Get(ctx, "PICTEKNIK01")
	require.ErrorIs(t, err, errStore)
	_, err = repo.Insert(ctx, masterpicteknik.Technician{OperatorID: "X"})
	require.ErrorIs(t, err, errStore)
	_, err = repo.Update(ctx, masterpicteknik.Technician{OperatorID: "PICTEKNIK01"})
	require.ErrorIs(t, err, errStore)
}
