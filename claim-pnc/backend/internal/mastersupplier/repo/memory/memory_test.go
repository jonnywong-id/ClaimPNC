package memory

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
)

var errBoom = errors.New("boom")

// Daftar diurutkan menurut nama, dan kata kunci mencocokkan nama, kota, atau narahubung.
func TestListSortsByNameAndFiltersThreeColumns(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	all, err := repo.List(ctx, mastersupplier.Filter{})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "Supplier Contoh Aneka", all[0].Name)
	require.Equal(t, "Supplier Contoh Utama", all[2].Name)

	byCity, err := repo.List(ctx, mastersupplier.Filter{Keyword: " bandung "})
	require.NoError(t, err)
	require.Len(t, byCity, 1)
	require.Equal(t, "0100000000002", byCity[0].ID)

	byContact, err := repo.List(ctx, mastersupplier.Filter{Keyword: "narahubung tiga"})
	require.NoError(t, err)
	require.Len(t, byContact, 1)
	require.Equal(t, "0100000000003", byContact[0].ID)

	none, err := repo.List(ctx, mastersupplier.Filter{Keyword: "tidak ada"})
	require.NoError(t, err)
	require.Empty(t, none)
}

// Pembacaan menurunkan JENIS_STATUS dari SUPPLIER_HE, bukan memakai isian yang tersimpan.
func TestReadDerivesSupplyTypeFromHeavyEquipment(t *testing.T) {
	repo := NewRepo(Options{Rows: []mastersupplier.Supplier{{
		ID: "X1", Name: "A", HeavyEquipment: mastersupplier.SupplyTypeHeavyEquipment, SupplyType: "9",
	}}})
	got, err := repo.Get(context.Background(), " X1 ")
	require.NoError(t, err)
	require.Equal(t, mastersupplier.DeriveSupplyType(mastersupplier.SupplyTypeHeavyEquipment), got.SupplyType)

	_, err = repo.Get(context.Background(), "tidak-ada")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)
}

func TestFindByNameIgnoresCaseAndRejectsBlank(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	got, err := repo.FindByName(ctx, "  supplier contoh utama ")
	require.NoError(t, err)
	require.Equal(t, "0100000000001", got.ID)

	_, err = repo.FindByName(ctx, "   ")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)
	_, err = repo.FindByName(ctx, "Lain")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)
}

// Nama yang sudah dipakai ditolak tanpa memandang besar-kecil huruf.
func TestInsertRejectsTakenName(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	err := repo.Insert(ctx, mastersupplier.Supplier{ID: "N", Name: "SUPPLIER CONTOH ANEKA"})
	require.ErrorIs(t, err, mastersupplier.ErrNameTaken)

	require.NoError(t, repo.Insert(ctx, mastersupplier.Supplier{ID: "N", Name: "Baru"}))
	got, err := repo.Get(ctx, "N")
	require.NoError(t, err)
	require.Equal(t, "Baru", got.Name)
}

// Penyimpanan mempertahankan OldID dari baris tersimpan.
func TestUpdateKeepsOldID(t *testing.T) {
	repo := NewRepo(Options{Rows: []mastersupplier.Supplier{{ID: "A1", Name: "Lama", OldID: "L-9"}}})
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, mastersupplier.Supplier{ID: "A1", Name: "Baru", OldID: "DARI-LUAR"}))
	got, err := repo.Get(ctx, "A1")
	require.NoError(t, err)
	require.Equal(t, "Baru", got.Name)
	require.Equal(t, "L-9", got.OldID)

	require.ErrorIs(t, repo.Update(ctx, mastersupplier.Supplier{ID: "B"}), mastersupplier.ErrNotFound)
}

func TestRequestApprovalIsRecordedAndCopied(t *testing.T) {
	repo := NewSampleRepo()
	require.Empty(t, repo.Approval())

	require.NoError(t, repo.RequestApproval(context.Background(), mastersupplier.ApprovalRequest{}))
	list := repo.Approval()
	require.Len(t, list, 1)

	// Salinan: menambah ke hasil tidak mengubah isi repo.
	_ = append(list, mastersupplier.ApprovalRequest{})
	require.Len(t, repo.Approval(), 1)
}

// ID diterbitkan dari kode situs dan nomor urut berikutnya; situs kosong memakai SampleSite.
func TestNextIDUsesSiteAndSequence(t *testing.T) {
	repo := NewRepo(Options{Sequence: 41})
	first, err := repo.NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastersupplier.ComposeID(SampleSite, 42, mastersupplier.SequenceWidth), first)

	second, err := NewRepo(Options{Site: " 07 "}).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastersupplier.ComposeID("07", 1, mastersupplier.SequenceWidth), second)
}

func TestReferenceListsReturnCopies(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	branches, err := repo.ListBranches(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleBranches(), branches)

	countries, err := repo.ListCountries(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleCountries(), countries)

	banks, err := repo.ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleBanks(), banks)

	branches[0].Name = "DIUBAH"
	again, err := repo.ListBranches(ctx)
	require.NoError(t, err)
	require.Equal(t, "Cabang Contoh Pusat", again[0].Name)
}

// Kota dicari menurut nama atau ID persis; kata kunci terlalu pendek tidak mencari.
func TestSearchCities(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	short, err := repo.SearchCities(ctx, "j")
	require.NoError(t, err)
	require.Nil(t, short)

	byName, err := repo.SearchCities(ctx, "band")
	require.NoError(t, err)
	require.Equal(t, []mastersupplier.City{{ID: "3273", Name: "Bandung"}}, byName)

	byID, err := repo.SearchCities(ctx, "1271")
	require.NoError(t, err)
	require.Equal(t, []mastersupplier.City{{ID: "1271", Name: "Medan"}}, byID)
}

// Hasil kota dibatasi MaxLookupRows.
func TestSearchCitiesStopsAtMaxLookupRows(t *testing.T) {
	cities := make([]mastersupplier.City, 0, mastersupplier.MaxLookupRows+5)
	for i := 0; i < mastersupplier.MaxLookupRows+5; i++ {
		cities = append(cities, mastersupplier.City{ID: fmt.Sprintf("C%d", i), Name: "Kota Contoh"})
	}
	got, err := NewRepo(Options{Cities: cities}).SearchCities(context.Background(), "kota")
	require.NoError(t, err)
	require.Len(t, got, mastersupplier.MaxLookupRows)
}

// Sandi dikumpulkan dari baris tanpa ganda dan tanpa nilai kosong; labelnya sandinya sendiri.
func TestListCodesCollectsDistinctNonEmpty(t *testing.T) {
	repo := NewRepo(Options{Rows: []mastersupplier.Supplier{
		{ID: "1", PartnerStatus: "1", HeavyEquipment: "1", SupplierType: "2", ActiveRequested: "1", AutoPayment: " "},
		{ID: "2", PartnerStatus: "1", HeavyEquipment: "0", SupplierType: "2", ActiveRequested: "0", AutoPayment: "0"},
	}})
	got, err := repo.ListCodes(context.Background())
	require.NoError(t, err)
	require.Equal(t, []mastersupplier.CodeOption{{Value: "1", Label: "1"}}, got.PartnerStatus)
	require.Equal(t, []mastersupplier.CodeOption{{Value: "2", Label: "2"}}, got.SupplierType)
	require.Len(t, got.Active, 2)
	require.Equal(t, []mastersupplier.CodeOption{{Value: "0", Label: "0"}}, got.AutoPayment)
	// ListCodes membaca SupplyType yang TERSIMPAN, bukan turunan — di sini kosong.
	require.Empty(t, got.SupplyType)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := NewSampleRepo()
	repo.SetError(errBoom)
	ctx := context.Background()

	_, err := repo.List(ctx, mastersupplier.Filter{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Get(ctx, "x")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, errBoom)
	require.ErrorIs(t, repo.Insert(ctx, mastersupplier.Supplier{}), errBoom)
	require.ErrorIs(t, repo.Update(ctx, mastersupplier.Supplier{}), errBoom)
	require.ErrorIs(t, repo.RequestApproval(ctx, mastersupplier.ApprovalRequest{}), errBoom)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListBranches(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.SearchCities(ctx, "bandung")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListCountries(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListBanks(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListCodes(ctx)
	require.ErrorIs(t, err, errBoom)
}
