package memory

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

func caseNumbers(tasks []inboxinvestigator.Task) []string {
	out := make([]string, 0, len(tasks))
	for _, one := range tasks {
		out = append(out, one.CaseNumber)
	}
	return out
}

// Contoh lengkap diurutkan menurut nomor case, sama seperti ORDER BY PYID.
func TestSampleRepoListsEveryTaskOrderedByCaseNumber(t *testing.T) {
	page, err := NewSampleRepo().List(context.Background(), inboxinvestigator.Filter{})
	require.NoError(t, err)
	require.False(t, page.Truncated)
	// MENURUN — Report Definition menyatakan `pySortType = DESC` pada `.pyID`.
	require.Equal(t, []string{
		"PNCN.26.0007", "PNC-100241", "PNC-100238", "PNC-100236", "PNC-100230",
		"PNC-100222", "PNC-099876", "PNC-099801",
	}, caseNumbers(page.Tasks))
}

// Kata kunci mencocokkan ketujuh kolom tanpa memandang huruf besar-kecil.
func TestKeywordMatchesEverySearchableColumn(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	for keyword, expected := range map[string][]string{
		"pnc-100241":             {"PNC-100241"},
		"00.000.2026.00004":      {"PNC-100230"},
		"bahari":                 {"PNC-100236"},
		"peserta contoh delapan": {"PNC-099801"},
		"contractors":            {"PNC-099801"},
		"cabang selatan":         {"PNC-100222"},
		"admincontoh3":           {"PNC-100230", "PNC-099876"},
		"tidak-ada-di-mana-pun":  {},
	} {
		page, err := repo.List(ctx, inboxinvestigator.Filter{Keyword: "  " + keyword + " "})
		require.NoError(t, err)
		require.Equal(t, expected, caseNumbers(page.Tasks), keyword)
	}
}

// Nomor case yang sama diurutkan menurut tanggal pendaftaran MENURUN, kosong paling akhir,
// lalu menurut kunci kerja menurun.
func TestTiesAreBrokenByRegistrationThenReference(t *testing.T) {
	early := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

	repo := NewRepo(Options{Tasks: []inboxinvestigator.Task{
		{Reference: "D", CaseNumber: "PNC-1", RegisteredAt: &late},
		{Reference: "C", CaseNumber: "PNC-1", RegisteredAt: &early},
		{Reference: "B", CaseNumber: "PNC-1"},
		{Reference: "A", CaseNumber: "PNC-1"},
		{Reference: "E", CaseNumber: "PNC-1", RegisteredAt: &early},
	}})

	page, err := repo.List(context.Background(), inboxinvestigator.Filter{})
	require.NoError(t, err)

	references := make([]string, 0, len(page.Tasks))
	for _, one := range page.Tasks {
		references = append(references, one.Reference)
	}
	require.Equal(t, []string{"D", "E", "C", "B", "A"}, references)
}

// Lebih dari MaxRows baris dipotong dan dinyatakan terpotong.
func TestMoreThanMaxRowsIsTruncatedAndSaysSo(t *testing.T) {
	tasks := make([]inboxinvestigator.Task, 0, inboxinvestigator.MaxRows+1)
	for i := 0; i <= inboxinvestigator.MaxRows; i++ {
		tasks = append(tasks, inboxinvestigator.Task{
			Reference: strconv.Itoa(i), CaseNumber: "PNC-" + strconv.Itoa(100000+i)})
	}

	page, err := NewRepo(Options{Tasks: tasks}).
		List(context.Background(), inboxinvestigator.Filter{})
	require.NoError(t, err)
	require.True(t, page.Truncated)
	require.Len(t, page.Tasks, inboxinvestigator.MaxRows)
	// Yang BERTAHAN adalah nomor tertinggi, karena urutannya menurun. Arah urutan
	// menentukan baris mana yang hilang saat antreannya panjang — itulah sebabnya ia diuji
	// di sini, bukan hanya di uji urutan.
	require.Equal(t, "PNC-100500", page.Tasks[0].CaseNumber)
}

// Galat yang dipasang dikembalikan apa adanya.
func TestSetErrorMakesListFail(t *testing.T) {
	repo := NewSampleRepo()
	failure := errors.New("oracle mati")
	repo.SetError(failure)

	_, err := repo.List(context.Background(), inboxinvestigator.Filter{})
	require.ErrorIs(t, err, failure)
}

// Teks waktu contoh yang salah adalah cacat pemrograman dan panik.
func TestAnUnreadableSampleTimePanics(t *testing.T) {
	require.Panics(t, func() { at("bukan waktu") })
}
