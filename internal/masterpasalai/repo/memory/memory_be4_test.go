package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
)

// TestNewRepoDropsEmptyRows membuktikan baris yang ketiga isiannya kosong dibuang.
func TestNewRepoDropsEmptyRows(t *testing.T) {
	repo := NewRepo([]masterpasalai.Clause{
		{ID: "001", Number: "1"},
		{ID: "002", Number: " ", Paragraph: "", Event: "  "},
		{ID: "003", Event: "Banjir"},
	})

	total, err := repo.CountAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, total)
}

// TestListWithoutKeywordPaginatesSample memeriksa halaman pertama, kedua, dan di luar jangkauan.
func TestListWithoutKeywordPaginatesSample(t *testing.T) {
	repo := NewRepo(SampleClause())
	ctx := context.Background()

	first, err := repo.List(ctx, masterpasalai.Filter{Page: 1}.Clean())
	require.NoError(t, err)
	require.Equal(t, 28, first.Total)
	require.Equal(t, 1, first.Number)
	require.Len(t, first.Clause, masterpasalai.PageSize)
	require.Equal(t, "001", first.Clause[0].ID)

	second, err := repo.List(ctx, masterpasalai.Filter{Page: 2})
	require.NoError(t, err)
	require.Len(t, second.Clause, 3)
	require.Equal(t, "026", second.Clause[0].ID)

	beyond, err := repo.List(ctx, masterpasalai.Filter{Page: 9})
	require.NoError(t, err)
	require.Equal(t, 28, beyond.Total)
	require.Equal(t, 9, beyond.Number)
	require.Empty(t, beyond.Clause)
}

// TestListSearchesAllColumnsIgnoringCase memeriksa penyaring OR atas tiga kolom.
func TestListSearchesAllColumnsIgnoringCase(t *testing.T) {
	repo := NewRepo([]masterpasalai.Clause{
		{ID: "3", Number: "A", Paragraph: "B", Event: "kebakaran gudang"},
		{ID: "1", Number: "Pasal-X", Paragraph: "1", Event: "lain"},
		{ID: "2", Number: "9", Paragraph: "ayat-x", Event: "lain"},
		{ID: "4", Number: "9", Paragraph: "1", Event: "banjir"},
	})

	page, err := repo.List(context.Background(), masterpasalai.Filter{Keyword: "  X ", Page: 1})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Equal(t, []string{"1", "2"}, []string{page.Clause[0].ID, page.Clause[1].ID})

	page, err = repo.List(context.Background(), masterpasalai.Filter{Keyword: "KEBAKARAN"})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "3", page.Clause[0].ID)

	page, err = repo.List(context.Background(), masterpasalai.Filter{Keyword: "tidakada"})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Clause)
}

// TestListOrdersByTextID membuktikan urutannya pengurutan teks, sama dengan ORDER BY WP_ID.
func TestListOrdersByTextID(t *testing.T) {
	repo := NewRepo([]masterpasalai.Clause{
		{ID: "9", Number: "a"},
		{ID: "10", Number: "b"},
	})
	page, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.NoError(t, err)
	require.Equal(t, "10", page.Clause[0].ID)
	require.Equal(t, "9", page.Clause[1].ID)
}

// TestListReturnsCopy membuktikan halaman adalah salinan, bukan potongan isi repo.
func TestListReturnsCopy(t *testing.T) {
	repo := NewRepo([]masterpasalai.Clause{{ID: "1", Number: "asli"}})
	page, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.NoError(t, err)
	page.Clause[0].Number = "diubah"

	again, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.NoError(t, err)
	require.Equal(t, "asli", again.Clause[0].Number)
}
