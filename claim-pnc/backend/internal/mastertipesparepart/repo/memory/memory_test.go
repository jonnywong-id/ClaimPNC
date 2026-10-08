package memory

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesparepart"
)

var errBoom = errors.New("boom")

// Daftar menyaring status dan mencari di NAMA TIPE saja.
//
// Tanpa JOIN: grid Pega tidak menampilkan nama kategori, dan kueri daftarnya tidak
// menggabungkan tabel apa pun (koreksi Work Owner 2026-10-04).
func TestListFiltersStatusAndSearchesTypeName(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	approved, err := repo.List(ctx, mastertipesparepart.Filter{Status: mastertipesparepart.StatusApproved})
	require.NoError(t, err)
	require.Len(t, approved, 4)
	require.Equal(t, "1", approved[0].CategoryID)
	// Baris yatim tetap muncul — tidak ada JOIN yang dapat membuangnya.
	require.Equal(t, "7", approved[3].ID)
	require.Equal(t, "99", approved[3].CategoryID)

	// " hydraulic " cocok dengan NAMA TIPE "HYDRAULIC PUMP", bukan dengan nama kategori.
	byTypeName, err := repo.List(ctx, mastertipesparepart.Filter{Status: mastertipesparepart.StatusApproved, Keyword: " hydraulic "})
	require.NoError(t, err)
	require.Len(t, byTypeName, 1)
	require.Equal(t, "HYDRAULIC PUMP", byTypeName[0].Name)

	// "ENGINE" hanya nama KATEGORI, bukan nama tipe mana pun — tidak boleh terjaring.
	byCategoryName, err := repo.List(ctx, mastertipesparepart.Filter{Status: mastertipesparepart.StatusApproved, Keyword: "engine"})
	require.NoError(t, err)
	require.Empty(t, byCategoryName)

	byName, err := repo.List(ctx, mastertipesparepart.Filter{Status: mastertipesparepart.StatusPending, Keyword: "idler"})
	require.NoError(t, err)
	require.Len(t, byName, 1)
	require.Equal(t, "5", byName[0].ID)
}

// Kunci angka diurutkan sebagai angka; kunci bukan angka ditaruh di belakang, terurut teks.
func TestListSortsNumericThenText(t *testing.T) {
	repo := NewRepo(Options{Rows: []mastertipesparepart.PartType{
		{ID: "b", Status: "1"}, {ID: "10", Status: "1"}, {ID: "a", Status: "1"}, {ID: "9", Status: "1"},
	}})
	got, err := repo.List(context.Background(), mastertipesparepart.Filter{Status: "1"})
	require.NoError(t, err)
	ids := make([]string, 0, len(got))
	for _, one := range got {
		ids = append(ids, one.ID)
	}
	require.Equal(t, []string{"9", "10", "a", "b"}, ids)
}

func TestGetTrimsKeyAndReportsMissing(t *testing.T) {
	repo := NewSampleRepo()
	got, err := repo.Get(context.Background(), " 3 ")
	require.NoError(t, err)
	require.Equal(t, "HYDRAULIC PUMP", got.Name)
	require.Equal(t, "2", got.CategoryID)

	_, err = repo.Get(context.Background(), "404")
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)
}

// Pencarian nama tidak menyaring status; bila kembar, yang ID-nya terkecil yang dikembalikan.
func TestFindByNamePicksSmallestID(t *testing.T) {
	repo := NewRepo(Options{
		Rows: []mastertipesparepart.PartType{
			{ID: "5", Name: "Valve", CategoryID: "1", Status: "2"},
			{ID: "2", Name: "VALVE", CategoryID: "1", Status: "0"},
		},
		Category: []mastertipesparepart.Category{{ID: "1", Name: "ENGINE"}},
	})
	got, err := repo.FindByName(context.Background(), " valve ")
	require.NoError(t, err)
	require.Equal(t, "2", got.ID)

	_, err = repo.FindByName(context.Background(), "lain")
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)
}

// Kategori diurutkan menurut nama dan dipotong pada MaxLookupRows.
func TestListCategoriesSortsAndCaps(t *testing.T) {
	got, err := NewSampleRepo().ListCategories(context.Background())
	require.NoError(t, err)
	require.Equal(t, SampleCategoryList(), got)

	many := make([]mastertipesparepart.Category, 0, mastertipesparepart.MaxLookupRows+2)
	for i := 0; i < mastertipesparepart.MaxLookupRows+2; i++ {
		many = append(many, mastertipesparepart.Category{ID: fmt.Sprint(i), Name: fmt.Sprintf("K%04d", i)})
	}
	capped, err := NewRepo(Options{Category: many}).ListCategories(context.Background())
	require.NoError(t, err)
	require.Len(t, capped, mastertipesparepart.MaxLookupRows)
	require.Equal(t, "K0000", capped[0].Name)
}

// Penambahan menolak nama kembar (tanpa memandang status), mengabaikan ID dan nama kategori
// dari pemanggil, dan menerbitkan max+1 dengan melewati kunci bukan angka.
func TestInsertIssuesMaxPlusOne(t *testing.T) {
	repo := NewRepo(Options{Rows: []mastertipesparepart.PartType{
		{ID: "4", Name: "Ditolak", Status: "2"}, {ID: "x", Name: "Aneh"},
	}})
	ctx := context.Background()

	_, err := repo.Insert(ctx, mastertipesparepart.PartType{Name: " DITOLAK "})
	require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken)

	fresh, err := repo.Insert(ctx, mastertipesparepart.PartType{
		ID: "99", Name: "Baru", CategoryID: "1", Status: "0",
	})
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.PartType{ID: "5", Name: "Baru", CategoryID: "1", Status: "0"}, fresh)

	next, err := repo.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, "6", next)
}

func TestUpdateWritesOnlyEditableColumns(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, mastertipesparepart.PartType{
		ID: " 1 ", Name: "Ubah", CategoryID: "2", Status: "0",
	}))
	got, err := repo.Get(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.PartType{ID: "1", Name: "Ubah", CategoryID: "2", Status: "0"}, got)

	require.ErrorIs(t, repo.Update(ctx, mastertipesparepart.PartType{ID: "404"}), mastertipesparepart.ErrNotFound)
}

// Baris tidak ada dilewati, dan baris yang sudah berstatus itu tidak dihitung berubah.
func TestSetStatusCountsOnlyChangedRows(t *testing.T) {
	repo := NewSampleRepo()
	changed, err := repo.SetStatus(context.Background(), []string{"4", "1", "404"}, mastertipesparepart.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	got, err := repo.Get(context.Background(), "4")
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.StatusApproved, got.Status)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := NewSampleRepo()
	repo.SetError(errBoom)
	ctx := context.Background()

	_, err := repo.List(ctx, mastertipesparepart.Filter{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListCategories(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Insert(ctx, mastertipesparepart.PartType{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, errBoom)
	require.ErrorIs(t, repo.Update(ctx, mastertipesparepart.PartType{}), errBoom)
	_, err = repo.SetStatus(ctx, []string{"1"}, "1")
	require.ErrorIs(t, err, errBoom)
}
