package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
)

func idsOf(list []masterbengkel.Workshop) []string {
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.ID)
	}
	return out
}

// Daftar disaring menurut status, dan kata kunci mencocokkan nama, kota, atau cabang.
func TestListFiltersByStatusAndKeyword(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Rows: []masterbengkel.Workshop{
		{ID: "A", Name: "Bengkel Satu", CityName: "Bandung", Status: masterbengkel.StatusPending},
		{ID: "C", Name: "Bengkel Dua", BranchName: "Cabang Medan", Status: masterbengkel.StatusPending},
		{ID: "B", Name: "Bengkel Tiga", Status: masterbengkel.StatusApproved},
	}})
	ctx := context.Background()

	pending, err := repo.List(ctx, masterbengkel.Filter{Status: masterbengkel.StatusPending})
	require.NoError(t, err)
	require.Equal(t, []string{"C", "A"}, idsOf(pending))

	byCity, err := repo.List(ctx, masterbengkel.Filter{
		Status: masterbengkel.StatusPending, Keyword: " bandung "})
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, idsOf(byCity))

	byBranch, err := repo.List(ctx, masterbengkel.Filter{
		Status: masterbengkel.StatusPending, Keyword: "medan"})
	require.NoError(t, err)
	require.Equal(t, []string{"C"}, idsOf(byBranch))

	none, err := repo.List(ctx, masterbengkel.Filter{
		Status: masterbengkel.StatusPending, Keyword: "tidak-ada"})
	require.NoError(t, err)
	require.Empty(t, none)
}

// Get, FindByName, dan FindByLogin mengabaikan spasi tepi dan besar-kecil huruf.
func TestLookupsIgnoreCaseAndSpaces(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	got, err := repo.Get(ctx, " 010000000001 ")
	require.NoError(t, err)
	require.Equal(t, "Bengkel Contoh Utama", got.Name)
	_, err = repo.Get(ctx, "x")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)

	byName, err := repo.FindByName(ctx, "  bengkel contoh UTAMA ")
	require.NoError(t, err)
	require.Equal(t, "010000000001", byName.ID)
	_, err = repo.FindByName(ctx, "")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	_, err = repo.FindByName(ctx, "tidak ada")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)

	byLogin, err := repo.FindByLogin(ctx, " BENGKELCONTOH1 ")
	require.NoError(t, err)
	require.Equal(t, "010000000001", byLogin.ID)
	_, err = repo.FindByLogin(ctx, " ")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	_, err = repo.FindByLogin(ctx, "tidak-ada")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
}

// Penambahan menolak nama dan login yang sudah dipakai; login kosong boleh berulang.
func TestInsertRejectsTakenNameAndLogin(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Rows: []masterbengkel.Workshop{
		{ID: "1", Name: "Satu", Login: "loginsatu"},
		{ID: "2", Name: "Dua"},
	}})
	ctx := context.Background()

	require.ErrorIs(t, repo.Insert(ctx, masterbengkel.Workshop{ID: "3", Name: " satu "}),
		masterbengkel.ErrNameTaken)
	require.ErrorIs(t, repo.Insert(ctx, masterbengkel.Workshop{
		ID: "3", Name: "Tiga", Login: " LOGINSATU "}), masterbengkel.ErrLoginTaken)
	require.NoError(t, repo.Insert(ctx, masterbengkel.Workshop{ID: "3", Name: "Tiga"}))

	stored, err := repo.Get(ctx, "3")
	require.NoError(t, err)
	require.Equal(t, "Tiga", stored.Name)
}

// Penyuntingan tidak mengganti ID; baris yang hilang dijawab tidak ditemukan.
func TestUpdateKeepsTheID(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, masterbengkel.Workshop{
		ID: " 010000000001 ", Name: "Nama Baru"}))
	got, err := repo.Get(ctx, "010000000001")
	require.NoError(t, err)
	require.Equal(t, "Nama Baru", got.Name)
	require.Equal(t, "010000000001", got.ID)

	require.ErrorIs(t, repo.Update(ctx, masterbengkel.Workshop{ID: "x"}),
		masterbengkel.ErrNotFound)
}

// SetStatus hanya menghitung baris yang berubah.
func TestSetStatusCountsChangedRows(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Rows: []masterbengkel.Workshop{
		{ID: "1", Status: masterbengkel.StatusPending},
		{ID: "2", Status: masterbengkel.StatusApproved},
	}})

	changed, err := repo.SetStatus(context.Background(), []string{" 1 ", "2", "9"},
		masterbengkel.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

// Penomoran bengkel dan lampiran melanjutkan nomor urut dengan bentuk yang sama dengan SQL.
func TestNextIDsContinueTheirSequences(t *testing.T) {
	ctx := context.Background()

	sample := memory.NewSampleRepo()
	id, err := sample.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterbengkel.ComposeID(memory.SampleSite, memory.SampleSequence+1, 10), id)

	empty := memory.NewRepo(memory.Options{DocumentSequence: 4})
	first, err := empty.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterbengkel.ComposeID(memory.SampleSite, 1, 10), first)

	document, err := empty.NextDocumentID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterbengkel.ComposeDocumentID(memory.SampleDocumentYear, 5), document)

	custom := memory.NewRepo(memory.Options{Site: "09", DocumentYear: "27"})
	id, err = custom.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterbengkel.ComposeID("09", 1, 10), id)
	document, err = custom.NextDocumentID(ctx)
	require.NoError(t, err)
	require.Equal(t, "270000000001", document)
}

// Daftar acuan terurut menurut nama; kota dicari menurut nama atau kode.
func TestReferenceLists(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	branches, err := repo.ListBranches(ctx)
	require.NoError(t, err)
	require.Equal(t, "Cabang Contoh Bandung", branches[0].Name)

	banks, err := repo.ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, "Bank Contoh Dua", banks[0].Name)

	cities, err := repo.SearchCities(ctx, " jakarta ")
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.City{
		{ID: "3171", Name: "Jakarta Pusat"}, {ID: "3174", Name: "Jakarta Selatan"}}, cities)

	byCode, err := repo.SearchCities(ctx, "3273")
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.City{{ID: "3273", Name: "Bandung"}}, byCode)

	blank, err := repo.SearchCities(ctx, "  ")
	require.NoError(t, err)
	require.Nil(t, blank)
}

// Pencarian kota berhenti pada batas baris lookup.
func TestSearchCitiesStopsAtTheLookupLimit(t *testing.T) {
	cities := make([]masterbengkel.City, 0, masterbengkel.MaxLookupRows+5)
	for i := 0; i < masterbengkel.MaxLookupRows+5; i++ {
		cities = append(cities, masterbengkel.City{ID: "K", Name: "Kota"})
	}
	repo := memory.NewRepo(memory.Options{Cities: cities})

	result, err := repo.SearchCities(context.Background(), "kota")
	require.NoError(t, err)
	require.Len(t, result, masterbengkel.MaxLookupRows)
}

// Lampiran disimpan sekaligus ditautkan ke bengkelnya, lalu dapat dibaca kembali.
func TestSaveAndFindDocument(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()
	document := masterbengkel.Document{ID: " 260000000001 ", Name: "bukti.pdf",
		Content: []byte("isi")}

	require.NoError(t, repo.SaveDocument(ctx, " 010000000001 ", document))

	workshop, err := repo.Get(ctx, "010000000001")
	require.NoError(t, err)
	require.Equal(t, "260000000001", workshop.DocumentID)

	found, err := repo.FindDocument(ctx, "260000000001")
	require.NoError(t, err)
	require.Equal(t, "bukti.pdf", found.Name)

	_, err = repo.FindDocument(ctx, "tidak-ada")
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)

	require.ErrorIs(t, repo.SaveDocument(ctx, "x", document), masterbengkel.ErrNotFound)
}

// Galat yang dipasang dijawab setiap operasi.
func TestSetErrorFailsEveryOperation(t *testing.T) {
	failure := errors.New("oracle mati")
	repo := memory.NewSampleRepo()
	repo.SetError(failure)
	ctx := context.Background()

	_, err := repo.List(ctx, masterbengkel.Filter{})
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, failure)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, failure)
	_, err = repo.FindByLogin(ctx, "x")
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, repo.Insert(ctx, masterbengkel.Workshop{}), failure)
	require.ErrorIs(t, repo.Update(ctx, masterbengkel.Workshop{}), failure)
	_, err = repo.SetStatus(ctx, []string{"1"}, masterbengkel.StatusApproved)
	require.ErrorIs(t, err, failure)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.ListBranches(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.SearchCities(ctx, "x")
	require.ErrorIs(t, err, failure)
	_, err = repo.ListBanks(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.NextDocumentID(ctx)
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, repo.SaveDocument(ctx, "1", masterbengkel.Document{}), failure)
	_, err = repo.FindDocument(ctx, "1")
	require.ErrorIs(t, err, failure)
}
