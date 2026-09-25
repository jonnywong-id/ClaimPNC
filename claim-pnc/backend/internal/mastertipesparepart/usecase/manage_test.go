package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesparepart/repo/memory"
	"claim-pnc/internal/mastertipesparepart/usecase"
)

const portalAlias = "asm"

// approvedCategory adalah kunci kategori yang ADA dan disetujui pada contoh — HYDRAULIC.
const approvedCategory = "2"

func newService(t *testing.T, repo *memory.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastertipesparepart.Store, error) {
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

func newInput(name string) mastertipesparepart.Input {
	return mastertipesparepart.Input{Name: name, CategoryID: approvedCategory}
}

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.List(context.Background(), portalAlias, "9", "")
	require.ErrorIs(t, err, mastertipesparepart.ErrUnknownStatus)
}

// Ketiga tab terisi dari contoh yang sama, dan tidak ada baris yang bocor antartab.
func TestListSeparatesEachTab(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	for _, one := range []struct {
		status mastertipesparepart.ApprovalStatus
		want   int
	}{
		// Empat disetujui: tiga bertaut kategori sah, satu YATIM (SPROCKET, kategori "99").
		{mastertipesparepart.StatusApproved, 4},
		{mastertipesparepart.StatusPending, 2},
		{mastertipesparepart.StatusRejected, 1},
	} {
		list, err := service.List(ctx, portalAlias, one.status, "")
		require.NoError(t, err)
		require.Len(t, list, one.want, "tab %q", one.status)
		for _, row := range list {
			require.Equal(t, one.status, row.Status)
		}
	}
}

// Nama kategori ikut terbaca pada setiap baris — inilah yang di Pega datang dari JOIN.
func TestListJoinsCategoryName(t *testing.T) {
	service, _ := newSampleService(t)

	list, err := service.List(context.Background(), portalAlias,
		mastertipesparepart.StatusPending, "")
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, row := range list {
		require.Equal(t, "UNDERCARRIAGE", row.CategoryName,
			"baris %q seharusnya membawa nama kategori induknya", row.Name)
	}
}

// Baris YATIM tetap terlihat, dengan nama kategori KOSONG.
//
// Inilah satu-satunya selisih perilaku yang disengaja pada jalur baca modul ini: di Pega
// baris seperti ini HILANG dari daftar karena inner join-nya
// (`BrowseMasterSparepartTypeClaimHE_sql`), sehingga ia tidak dapat dilihat maupun
// diperbaiki siapa pun. Lihat banner pada berkas .sql.
func TestListKeepsRowWhoseCategoryIsMissing(t *testing.T) {
	service, _ := newSampleService(t)

	list, err := service.List(context.Background(), portalAlias,
		mastertipesparepart.StatusApproved, "")
	require.NoError(t, err)

	var orphan *mastertipesparepart.PartType
	for i := range list {
		if list[i].Name == "SPROCKET" {
			orphan = &list[i]
		}
	}
	require.NotNil(t, orphan, "baris yatim wajib TETAP terlihat, tidak boleh dibuang seperti di Pega")
	require.Equal(t, "99", orphan.CategoryID)
	require.Empty(t, orphan.CategoryName, "nama kategorinya kosong, bukan dikarang")
}

// Pencarian menyentuh nama tipe DAN nama kategorinya.
//
// Yang kedua DITAMBAHKAN terhadap sistem lama, dan ia yang membuat "apa saja tipe di
// HYDRAULIC" dapat dijawab tanpa memindai seluruh daftar.
func TestListSearchesTypeNameAndCategoryName(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	byType, err := service.List(ctx, portalAlias, mastertipesparepart.StatusApproved, "turbo")
	require.NoError(t, err)
	require.Len(t, byType, 1)
	require.Equal(t, "TURBOCHARGER", byType[0].Name)

	byCategory, err := service.List(ctx, portalAlias,
		mastertipesparepart.StatusApproved, "engine")
	require.NoError(t, err)
	require.Len(t, byCategory, 2, "kedua tipe di bawah ENGINE harus terjaring lewat nama kategori")
}

func TestChoicesReturnsApprovedCategoriesOnly(t *testing.T) {
	service, _ := newSampleService(t)

	set, err := service.Choices(context.Background(), portalAlias)
	require.NoError(t, err)
	require.Len(t, set.Category, 3)
	require.False(t, set.Truncated)

	// Urutannya menurut NAMA — itulah urutan yang dipindai mata pengguna pada dropdown.
	require.Equal(t, "ENGINE", set.Category[0].Name)
	require.Equal(t, "HYDRAULIC", set.Category[1].Name)
	require.Equal(t, "UNDERCARRIAGE", set.Category[2].Name)
}

// Penambahan lahir berstatus MENUNGGU, dengan ID diterbitkan server dan nama kategori
// terisi dari daftar yang sudah dibaca untuk memeriksanya.
func TestCreateStartsPendingWithIssuedID(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Create(context.Background(), portalAlias,
		newInput("SWING MOTOR"), actor(), nil)
	require.NoError(t, err)

	require.Equal(t, mastertipesparepart.StatusPending, saved.Status)
	require.Equal(t, "8", saved.ID, "ID berikutnya setelah contoh berkunci 1..7")
	require.Equal(t, approvedCategory, saved.CategoryID)
	require.Equal(t, "HYDRAULIC", saved.CategoryName,
		"nama kategori diisi tanpa pembacaan kedua")
}

// Kategori yang tidak ada ditolak — pemeriksaan yang DITAMBAHKAN terhadap sistem lama.
func TestCreateRejectsUnknownCategory(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		mastertipesparepart.Input{Name: "SWING MOTOR", CategoryID: "404"}, actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrCategoryNotFound)
}

// Kategori yang ADA tetapi BELUM disetujui juga ditolak.
//
// Dari sudut pandang daftar acuan keduanya sama saja — ListCategories memang hanya
// mengembalikan yang disetujui. Uji ini memakai kunci "99" yang tidak ada di daftar acuan
// contoh, mewakili kategori yang persetujuannya dicabut sementara form terbuka.
func TestCreateRejectsCategoryOutsideApprovedList(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		mastertipesparepart.Input{Name: "SWING MOTOR", CategoryID: "99"}, actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrCategoryNotFound)
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		mastertipesparepart.Input{}, actor(), nil)

	var failure *mastertipesparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Len(t, failure.Violation, 2, "kedua isian wajib dilaporkan sekaligus")
}

// Nama ganda ditolak tanpa memandang huruf besar-kecil.
func TestCreateRejectsDuplicateNameIgnoringCase(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		newInput("fuel filter"), actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken)
}

// Nama ganda ditolak meski KATEGORINYA BERBEDA.
//
// Inilah perilaku yang paling mudah mengejutkan di modul ini, dan ia ditiru apa adanya:
// `ValidationSparepartType` tidak menyaring PART_CATEGORY_ID sama sekali (`P-5`, keputusan
// Work Owner 2026-09-21). "FUEL FILTER" sudah dipakai di ENGINE, dan mencoba memakainya di
// HYDRAULIC pun ditolak.
func TestCreateRejectsDuplicateNameEvenInDifferentCategory(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		mastertipesparepart.Input{Name: "FUEL FILTER", CategoryID: approvedCategory},
		actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken,
		"keunikan nama berlaku di SELURUH tabel, bukan per kategori")
}

// Nama yang dipakai baris DITOLAK pun tetap memblokir.
//
// Pemeriksaannya tidak menyaring APPROVAL sama sekali. "CONTROL VALVE" ada di tab Reject,
// dan ia tidak terlihat dari tab mana pun yang sedang dibuka pengguna — itulah sebabnya
// pesan galatnya menyebut tab Reject secara eksplisit.
func TestCreateRejectsNameHeldByRejectedRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Create(context.Background(), portalAlias,
		newInput("CONTROL VALVE"), actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken)
}

// Menyimpan MENGEMBALIKAN baris ke antrean persetujuan, tanpa syarat apa pun.
//
// Padanan `Activity/UpdateTypeSparepart_act2` yang menetapkan
// `InputKategori.NO_ACCOUNT := "0"`.
func TestSaveAlwaysReturnsRowToPending(t *testing.T) {
	service, repo := newSampleService(t)
	ctx := context.Background()

	updated, err := service.Save(ctx, portalAlias, "1", newInput("FUEL FILTER"), actor(), nil)
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.StatusPending, updated.Status)

	stored, err := repo.Get(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.StatusPending, stored.Status)
}

// Menyimpan tanpa mengubah nama TIDAK ditolak oleh dirinya sendiri.
//
// Ia jalur yang benar-benar dilewati pengguna: memindahkan tipe ke kategori lain, atau
// mengembalikan baris yang ditolak ke antrean, keduanya menyimpan nama yang sama.
func TestSaveAllowsKeepingItsOwnName(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "1",
		newInput("FUEL FILTER"), actor(), nil)
	require.NoError(t, err)
}

// Memindahkan tipe ke kategori lain didukung — `UpdateMasterSparepartType_sql2` menulis
// PART_CATEGORY_ID pada setiap penyimpanan.
func TestSaveMovesTypeToAnotherCategory(t *testing.T) {
	service, repo := newSampleService(t)
	ctx := context.Background()

	updated, err := service.Save(ctx, portalAlias, "1",
		mastertipesparepart.Input{Name: "FUEL FILTER", CategoryID: "3"}, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, "3", updated.CategoryID)
	require.Equal(t, "UNDERCARRIAGE", updated.CategoryName)

	stored, err := repo.Get(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, "3", stored.CategoryID)
}

// Menyimpan dengan nama milik baris LAIN tetap ditolak.
func TestSaveRejectsNameHeldByAnotherRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "1",
		newInput("TURBOCHARGER"), actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken)
}

func TestSaveRejectsMissingRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "404",
		newInput("SWING MOTOR"), actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)
}

func TestSaveRejectsEmptyKey(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "   ",
		newInput("SWING MOTOR"), actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)
}

// Baris YATIM hanya dapat disimpan ulang setelah kategorinya diperbaiki.
//
// Ia akibat langsung dari LEFT JOIN: barisnya kini terlihat, tetapi penyimpanannya tetap
// menuntut kategori yang sah. Tanpa LEFT JOIN, baris ini tidak dapat diperbaiki sama
// sekali karena tidak pernah muncul di layar.
func TestSaveOnOrphanRowRequiresValidCategory(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	_, err := service.Save(ctx, portalAlias, "7",
		mastertipesparepart.Input{Name: "SPROCKET", CategoryID: "99"}, actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrCategoryNotFound)

	fixed, err := service.Save(ctx, portalAlias, "7", newInput("SPROCKET"), actor(), nil)
	require.NoError(t, err)
	require.Equal(t, "HYDRAULIC", fixed.CategoryName)
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"4"}, "9",
		actor(), nil)
	require.ErrorIs(t, err, mastertipesparepart.ErrUnknownStatus)
}

// Tanpa satu pun baris dipilih, keputusan ditolak sebagai pelanggaran isian — bukan
// dilaporkan berhasil dengan nol baris berubah.
func TestDecideRejectsEmptySelection(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias,
		[]string{"", "   "}, mastertipesparepart.StatusApproved, actor(), nil)

	var failure *mastertipesparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "id_tipe_sparepart", failure.Violation[0].Field)
}

// Kunci ganda dibuang, sehingga jumlah yang dilaporkan mencerminkan BARIS, bukan centang.
func TestDecideCountsRowsNotTicks(t *testing.T) {
	service, _ := newSampleService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"4", "4", " 4 "}, mastertipesparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

// Baris yang sudah tidak ada tidak menggagalkan keputusan; ia hanya tidak terhitung.
func TestDecideSkipsMissingRow(t *testing.T) {
	service, _ := newSampleService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"4", "404"}, mastertipesparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestDecideMovesRowsToChosenStatus(t *testing.T) {
	service, repo := newSampleService(t)
	ctx := context.Background()

	changed, err := service.Decide(ctx, portalAlias, []string{"4", "5"},
		mastertipesparepart.StatusApproved, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	for _, id := range []string{"4", "5"} {
		stored, err := repo.Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, mastertipesparepart.StatusApproved, stored.Status)
	}
}

func TestPortalUnknownIsRejectedEverywhere(t *testing.T) {
	service, _ := newSampleService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "lain", mastertipesparepart.StatusApproved, "")
	require.Error(t, err)

	_, err = service.Get(ctx, "lain", "1")
	require.Error(t, err)

	_, err = service.Choices(ctx, "lain")
	require.Error(t, err)

	_, err = service.Create(ctx, "lain", newInput("SWING MOTOR"), actor(), nil)
	require.Error(t, err)

	_, err = service.Save(ctx, "lain", "1", newInput("FUEL FILTER"), actor(), nil)
	require.Error(t, err)

	_, err = service.Decide(ctx, "lain", []string{"4"},
		mastertipesparepart.StatusApproved, actor(), nil)
	require.Error(t, err)

	require.Error(t, service.EnsurePortalReady("lain"))
	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

// Kegagalan penyimpanan diteruskan apa adanya, tidak ditelan.
func TestRepositoryFailureIsPropagated(t *testing.T) {
	service, repo := newSampleService(t)
	repo.SetError(errors.New("koneksi putus"))

	_, err := service.List(context.Background(), portalAlias,
		mastertipesparepart.StatusApproved, "")
	require.Error(t, err)
}
