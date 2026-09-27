package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterkategorisparepart/repo/memory"
	"claim-pnc/internal/masterkategorisparepart/usecase"
)

const portalAlias = "asm"

func newService(t *testing.T, repo *memory.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterkategorisparepart.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func newSampleService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewSampleRepo()
	return newService(t, repo), repo
}

func actor() usecase.Actor { return usecase.Actor{Login: "PNCADMIN"} }

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.List(context.Background(), portalAlias, "9", "")
	require.ErrorIs(t, err, masterkategorisparepart.ErrUnknownStatus)
}

// Ketiga tab terisi dari contoh yang sama, dan tidak ada baris yang bocor antartab.
func TestListSeparatesEachTab(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	for _, one := range []struct {
		status masterkategorisparepart.ApprovalStatus
		want   int
	}{
		{masterkategorisparepart.StatusApproved, 3},
		{masterkategorisparepart.StatusPending, 2},
		{masterkategorisparepart.StatusRejected, 1},
	} {
		list, err := service.List(ctx, portalAlias, one.status, "")
		require.NoError(t, err)
		require.Len(t, list, one.want, "tab %q", one.status)
		for _, row := range list {
			require.Equal(t, one.status, row.Status)
		}
	}
}

// Daftar diurutkan menurut ANGKA kuncinya, bukan menurut teksnya.
//
// Kolomnya bertipe angka di basis data (lihat banner masterkategorisparepart.sql), sehingga
// 9 mendahului 10. Pengurutan teks akan menaruh "10" sebelum "9" — dan itu yang benar di
// Master Sparepart, yang ID-nya memang teks. Uji ini menjaga keduanya tidak tertukar.
func TestListOrdersByNumericKey(t *testing.T) {
	repo := memory.NewRepo(memory.Options{
		Rows: []masterkategorisparepart.PartCategory{
			{ID: "10", Name: "SEPULUH", Status: masterkategorisparepart.StatusApproved},
			{ID: "9", Name: "SEMBILAN", Status: masterkategorisparepart.StatusApproved},
			{ID: "100", Name: "SERATUS", Status: masterkategorisparepart.StatusApproved},
		},
	})
	service := newService(t, repo)

	list, err := service.List(context.Background(), portalAlias,
		masterkategorisparepart.StatusApproved, "")
	require.NoError(t, err)
	require.Equal(t, []string{"9", "10", "100"},
		[]string{list[0].ID, list[1].ID, list[2].ID})
}

func TestListSearchesByName(t *testing.T) {
	service, _ := newSampleService(t)

	list, err := service.List(context.Background(), portalAlias,
		masterkategorisparepart.StatusApproved, "hydra")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "HYDRAULIC", list[0].Name)
}

func TestListRejectsUnknownPortal(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.List(context.Background(), "entitas-lain",
		masterkategorisparepart.StatusApproved, "")
	require.Error(t, err)
}

// Penambahan menerbitkan ID berikutnya dan SELALU berstatus menunggu.
//
// Contoh berisi enam baris berkunci "1".."6", sehingga yang berikutnya "7".
func TestCreateIssuesNextKeyAndStartsPending(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Create(context.Background(), portalAlias,
		masterkategorisparepart.Input{Name: "  Final Drive  "}, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, "7", saved.ID)
	require.Equal(t, "Final Drive", saved.Name, "nama dipangkas, bukan di-uppercase")
	require.Equal(t, masterkategorisparepart.StatusPending, saved.Status)
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		masterkategorisparepart.Input{Name: "   "}, actor(), nil)

	var failure *masterkategorisparepart.ValidationError
	require.ErrorAs(t, err, &failure)
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		masterkategorisparepart.Input{Name: "hydraulic"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken,
		"perbandingan nama tidak memandang huruf besar-kecil")
}

// Nama baris yang sudah DITOLAK tetap memblokir — perilaku sistem lama yang ditiru apa
// adanya (P-5). `ValidationSparepartCat` tidak menyaring APPROVAL sama sekali.
//
// Ini perilaku yang paling mengejutkan pada modul ini, dan justru karena itu ia diuji: bila
// kelak seseorang "memperbaikinya", uji ini yang memberi tahu bahwa perbaikan itu adalah
// selisih yang menuntut persetujuan Work Owner.
func TestCreateRejectsNameOfRejectedRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		masterkategorisparepart.Input{Name: "ATTACHMENT"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken)
}

// Penambahan yang ditolak karena nama ganda TIDAK memakai nomor urut.
//
// Nomor yang terlewat tidak merusak apa pun, tetapi deret yang berlubang membuat orang
// mencari baris yang tidak pernah ada. Urutan langkah di Repo.Insert yang menjaganya.
func TestCreateDoesNotConsumeKeyWhenRejected(t *testing.T) {
	service, repo := newSampleService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, portalAlias,
		masterkategorisparepart.Input{Name: "ENGINE"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken)

	next, err := repo.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, "7", next)
}

// Menyimpan SELALU mengembalikan baris ke antrean persetujuan, tanpa syarat apa pun —
// `Activity/UpdateKategoriSparepart_act2` menetapkan APPROVAL := "0".
func TestSaveAlwaysReturnsRowToPending(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Save(context.Background(), portalAlias, "1",
		masterkategorisparepart.Input{Name: "ENGINE ASSEMBLY"}, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, "1", saved.ID, "kunci tidak pernah berubah")
	require.Equal(t, "ENGINE ASSEMBLY", saved.Name)
	require.Equal(t, masterkategorisparepart.StatusPending, saved.Status)
}

// Menyimpan tanpa mengubah nama tidak boleh ditolak oleh dirinya sendiri.
//
// Ia hal yang wajar dilakukan pengguna — misalnya mengembalikan baris yang ditolak ke
// antrean — dan tanpa pengecualian ini, penolakannya tidak dapat dijelaskan kepada siapa pun.
func TestSaveAllowsUnchangedName(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Save(context.Background(), portalAlias, "6",
		masterkategorisparepart.Input{Name: "ATTACHMENT"}, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, masterkategorisparepart.StatusPending, saved.Status)
}

func TestSaveRejectsNameOwnedByAnotherRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "1",
		masterkategorisparepart.Input{Name: "HYDRAULIC"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken)
}

func TestSaveRejectsMissingRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "999",
		masterkategorisparepart.Input{Name: "BARU"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)
}

func TestSaveRejectsEmptyKey(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "  ",
		masterkategorisparepart.Input{Name: "BARU"}, actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)
}

func TestDecideApprovesSeveralRowsAtOnce(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	changed, err := service.Decide(ctx, portalAlias, []string{"4", "5"},
		masterkategorisparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	list, err := service.List(ctx, portalAlias, masterkategorisparepart.StatusPending, "")
	require.NoError(t, err)
	require.Empty(t, list)
}

// Kunci ganda dibuang, sehingga jumlah yang dilaporkan mencerminkan BARIS dan bukan centang.
func TestDecideIgnoresRepeatedKeys(t *testing.T) {
	service, _ := newSampleService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"4", "4", " 4 ", ""},
		masterkategorisparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

// Baris yang sudah berstatus itu tidak dihitung berubah — meniru RowsAffected pada adapter
// SQL, yang melaporkan nol untuk UPDATE yang tidak mengubah apa pun.
func TestDecideDoesNotCountUnchangedRow(t *testing.T) {
	service, _ := newSampleService(t)

	changed, err := service.Decide(context.Background(), portalAlias, []string{"1"},
		masterkategorisparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Zero(t, changed)
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"  ", ""},
		masterkategorisparepart.StatusApproved, actor(), nil)

	var failure *masterkategorisparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "id_kategori_sparepart", failure.Violation[0].Field)
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"4"}, "9",
		actor(), nil)
	require.ErrorIs(t, err, masterkategorisparepart.ErrUnknownStatus)
}

func TestGetReturnsSingleRow(t *testing.T) {
	service, _ := newSampleService(t)

	found, err := service.Get(context.Background(), portalAlias, " 2 ")
	require.NoError(t, err)
	require.Equal(t, "HYDRAULIC", found.Name)
}

func TestEnsurePortalReady(t *testing.T) {
	service, _ := newSampleService(t)

	require.NoError(t, service.EnsurePortalReady(portalAlias))
	require.Error(t, service.EnsurePortalReady("entitas-lain"))
}

// Kegagalan penyimpanan diteruskan apa adanya, tidak ditelan.
func TestRepoFailureReachesCaller(t *testing.T) {
	service, repo := newSampleService(t)
	repo.SetError(errors.New("basis data mati"))

	_, err := service.List(context.Background(), portalAlias,
		masterkategorisparepart.StatusApproved, "")
	require.Error(t, err)
}
