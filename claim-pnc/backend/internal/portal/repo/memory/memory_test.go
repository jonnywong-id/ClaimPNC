package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/portal/repo/memory"
)

func TestRepoReturnsStoredListInOrder(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 6)
	require.Equal(t, portal.Portal{ID: "202600102", Name: "ASURANSI SIMAS INSURTECH", Alias: "ASI"}, list[1])
	require.Equal(t, "SPK", list[4].Alias)
}

func TestEmptyRepoReturnsNoPortals(t *testing.T) {
	list, err := memory.NewRepo().List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestSetErrorMakesListFail(t *testing.T) {
	// Jalur gagal dibutuhkan untuk menguji middleware yang membaca daftar portal.
	boom := errors.New("tabel portal tidak terbaca")
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(boom)

	list, err := repo.List(context.Background())
	require.ErrorIs(t, err, boom)
	require.Nil(t, list)
}
