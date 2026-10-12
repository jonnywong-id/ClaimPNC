package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
)

func ids(list []masterpenyebabkerugian.CauseOfLoss) []string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}

// Daftar contoh terurut menurut ID dan memuat sepuluh baris.
func TestListReturnsTheSampleInIDOrder(t *testing.T) {
	list, err := NewRepo(SampleList()...).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"1001", "1002", "1003", "1004", "1005", "1006", "1007",
		"1008", "1009", "1010"}, ids(list))
	require.Equal(t, "01", list[0].LegacyID)
}

// Get memangkas ID yang dicari; ID yang tidak ada dijawab tidak ditemukan.
func TestGetTrimsTheID(t *testing.T) {
	repo := NewRepo(SampleList()...)

	cause, err := repo.Get(context.Background(), " 1002 ")
	require.NoError(t, err)
	require.Equal(t, masterpenyebabkerugian.CauseOfLoss{
		ID: "1002", LegacyID: "02", Description: "Contoh Golongan B"}, cause)

	_, err = repo.Get(context.Background(), "9999")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)
}

// Penambahan menerbitkan nomor berikutnya setelah yang tertinggi, tanpa ID lama.
func TestInsertIssuesTheNextNumber(t *testing.T) {
	repo := NewRepo(SampleList()...)

	cause, err := repo.Insert(context.Background(), "  Golongan Baru  ")
	require.NoError(t, err)
	require.Equal(t, masterpenyebabkerugian.CauseOfLoss{
		ID: "1011", Description: "Golongan Baru"}, cause)

	stored, err := repo.Get(context.Background(), "1011")
	require.NoError(t, err)
	require.Equal(t, cause, stored)

	next, err := repo.Insert(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, "1012", next.ID)
}

// ID warisan yang tidak berbentuk situs+angka tidak menaikkan nomor urut.
func TestLegacyIDsDoNotRaiseTheSequence(t *testing.T) {
	repo := NewRepo(
		masterpenyebabkerugian.CauseOfLoss{ID: "2050"},
		masterpenyebabkerugian.CauseOfLoss{ID: "1"},
		masterpenyebabkerugian.CauseOfLoss{ID: "1A9"},
	)

	cause, err := repo.Insert(context.Background(), "x")
	require.NoError(t, err)
	require.Equal(t, "1001", cause.ID)
}

// Penyuntingan hanya mengubah deskripsi; ID lama dipertahankan.
func TestUpdateChangesTheDescriptionOnly(t *testing.T) {
	repo := NewRepo(SampleList()...)

	cause, err := repo.Update(context.Background(), " 1001 ", " Baru ")
	require.NoError(t, err)
	require.Equal(t, masterpenyebabkerugian.CauseOfLoss{
		ID: "1001", LegacyID: "01", Description: "Baru"}, cause)

	_, err = repo.Update(context.Background(), "9999", "x")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)
}

// Galat yang dipasang dijawab oleh setiap operasi.
func TestSetErrorFailsEveryOperation(t *testing.T) {
	failure := errors.New("oracle mati")
	repo := NewRepo(SampleList()...)
	repo.SetError(failure)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "1001")
	require.ErrorIs(t, err, failure)
	_, err = repo.Insert(ctx, "x")
	require.ErrorIs(t, err, failure)
	_, err = repo.Update(ctx, "1001", "x")
	require.ErrorIs(t, err, failure)
}

// ThreeDigits menambah nol di depan, tidak pernah memotong.
func TestThreeDigits(t *testing.T) {
	require.Equal(t, "000", ThreeDigits(0))
	require.Equal(t, "042", ThreeDigits(42))
	require.Equal(t, "1000", ThreeDigits(1000))
}

// sequenceFromID hanya membaca ID berawalan situs yang sisanya angka.
func TestSequenceFromID(t *testing.T) {
	require.Equal(t, 12, sequenceFromID("1012", "1"))
	require.Equal(t, 0, sequenceFromID("2012", "1"))
	require.Equal(t, 0, sequenceFromID("1", "1"))
	require.Equal(t, 0, sequenceFromID("10A2", "1"))
}
