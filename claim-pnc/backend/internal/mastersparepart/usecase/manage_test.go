package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/repo/memory"
	"claim-pnc/internal/mastersparepart/usecase"
	"claim-pnc/internal/platform/clock"
)

const portalAlias = "asm"

// fixedNow adalah waktu yang dipegang jam uji.
//
// Tetap, supaya stempel TGL_UPDATE_HARGA dapat dibandingkan persis. Jam sungguhan membuat
// uji ini hanya dapat memeriksa "kira-kira sekarang", dan itu tidak membuktikan bahwa yang
// dipakai memang seam Clock.
var fixedNow = time.Date(2026, time.September, 20, 4, 30, 0, 0, time.UTC)

func newService(t *testing.T, repo *memory.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastersparepart.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		Clock: clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)
	return service
}

func newSampleService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewSampleRepo()
	return newService(t, repo), repo
}

func validInput() mastersparepart.Input {
	return mastersparepart.Input{
		Number:       "20Y-70-21120",
		Name:         "BUCKET PIN",
		Code:         "BKT-UND-020",
		SellingPrice: "2500000",
		CategoryID:   "KAT03",
		TypeID:       "TIP05",
	}
}

func TestNewServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(fixedNow)})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersparepart.Store, error) { return nil, nil },
	})
	require.Error(t, err)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.List(context.Background(), portalAlias, "9", "")
	require.ErrorIs(t, err, mastersparepart.ErrUnknownStatus)
}

func TestListFiltersByStatus(t *testing.T) {
	service, _ := newSampleService(t)

	approved, err := service.List(context.Background(), portalAlias,
		mastersparepart.StatusApproved, "")
	require.NoError(t, err)
	require.Len(t, approved, 1)
	require.Equal(t, "SP0000000001", approved[0].ID)
}

// Pencarian menelusuri KETIGA kunci alami, bukan nama saja.
//
// Itu yang membedakannya dari Master Panel, dan alasannya nyata: petugas gudang mencari suku
// cadang lewat nomornya jauh lebih sering daripada lewat namanya.
func TestListSearchesNameNumberAndCode(t *testing.T) {
	service, _ := newSampleService(t)

	for name, keyword := range map[string]string{
		"lewat nama":  "filter",
		"lewat nomor": "1r-07",
		"lewat kode":  "flt-eng",
	} {
		t.Run(name, func(t *testing.T) {
			found, err := service.List(context.Background(), portalAlias,
				mastersparepart.StatusApproved, keyword)
			require.NoError(t, err)
			require.Len(t, found, 1)
			require.Equal(t, "SP0000000001", found[0].ID)
		})
	}
}

func TestListRejectsUnknownPortal(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.List(context.Background(), "entitas-lain",
		mastersparepart.StatusApproved, "")
	require.Error(t, err)
}

func TestOptionsReadsBothLookups(t *testing.T) {
	service, _ := newSampleService(t)

	set, err := service.Options(context.Background(), portalAlias)
	require.NoError(t, err)
	require.Len(t, set.Category, 3)
	require.Len(t, set.Type, 5)

	// Setiap tipe menyebut kategori induknya; layar memakainya untuk mempersempit daftar.
	for _, one := range set.Type {
		require.NotEmpty(t, one.CategoryID, one.ID)
	}
}

func TestCreateIssuesIDAndStampsActor(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(),
		usecase.Actor{Login: "INTANHENNY"}, nil)
	require.NoError(t, err)

	// ID melanjutkan deret contoh, dengan lebar sepuluh digit.
	require.Equal(t, "SP0000000004", saved.ID)
	// Baris baru SELALU lahir menunggu persetujuan.
	require.Equal(t, mastersparepart.StatusPending, saved.Status)
	// USER_UPDATE benar-benar tersimpan — berbeda dari Master Panel.
	require.Equal(t, "INTANHENNY", saved.UpdatedBy)
}

// TGL_UPDATE_HARGA distempel pada SETIAP penyimpanan yang harganya terisi, meniru
// precondition `@PropertyHasValue(TempSparepart.HARGA_JUAL)` pada
// `Activity/UpdateSparepartHE_act`.
func TestCreateStampsPriceDateWhenPriceFilled(t *testing.T) {
	service, _ := newSampleService(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(),
		usecase.Actor{Login: "INTANHENNY"}, nil)
	require.NoError(t, err)
	require.NotNil(t, saved.PriceUpdatedAt)
	require.Equal(t, fixedNow, saved.PriceUpdatedAt.UTC())
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	service, _ := newSampleService(t)

	input := validInput()
	input.SellingPrice = ""

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "INTANHENNY"}, nil)

	var failure *mastersparepart.ValidationError
	require.ErrorAs(t, err, &failure)
}

func TestCreateRejectsDuplicateKeys(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*mastersparepart.Input)
		want   error
	}{
		"nomor": {func(i *mastersparepart.Input) { i.Number = "1R-0716" },
			mastersparepart.ErrNumberTaken},
		"nama": {func(i *mastersparepart.Input) { i.Name = "filter oli mesin" },
			mastersparepart.ErrNameTaken},
		"kode": {func(i *mastersparepart.Input) { i.Code = "FLT-ENG-001" },
			mastersparepart.ErrCodeTaken},
	} {
		t.Run(name, func(t *testing.T) {
			service, _ := newSampleService(t)

			input := validInput()
			tc.mutate(&input)

			_, err := service.Create(context.Background(), portalAlias, input,
				usecase.Actor{Login: "INTANHENNY"}, nil)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

// Menyimpan SELALU mengembalikan baris ke antrean, persis seperti
// `Activity/UpdateSparepartHE_act` yang menetapkan `APPROVAL := "0"` tanpa syarat.
func TestSaveReturnsRowToPending(t *testing.T) {
	service, _ := newSampleService(t)

	input := validInput()
	input.Number = "1R-0716"
	input.Name = "FILTER OLI MESIN"
	input.Code = "FLT-ENG-001"
	input.Name = "FILTER OLI MESIN BARU"

	saved, err := service.Save(context.Background(), portalAlias, "SP0000000001", input,
		usecase.Actor{Login: "BUDI"}, nil)
	require.NoError(t, err)
	require.Equal(t, mastersparepart.StatusPending, saved.Status)
	require.Equal(t, "FILTER OLI MESIN BARU", saved.Name)
	require.Equal(t, "BUDI", saved.UpdatedBy)
}

// Kolom yang TIDAK ikut berubah saat disimpan: ID dan DOKUMENID.
//
// DOKUMENID patut diperhatikan — sistem lama menimpanya dengan kosong setiap kali barisnya
// disimpan tanpa lampiran, sehingga lampiran yang sudah ada lenyap.
func TestSaveKeepsIDAndDocument(t *testing.T) {
	service, _ := newSampleService(t)

	input := validInput()
	input.Number = "1R-0716"
	input.Name = "FILTER OLI MESIN"
	input.Code = "FLT-ENG-001"

	saved, err := service.Save(context.Background(), portalAlias, "SP0000000001", input,
		usecase.Actor{Login: "BUDI"}, nil)
	require.NoError(t, err)
	require.Equal(t, "SP0000000001", saved.ID)
	require.Equal(t, "DOC-0001", saved.DocumentID)
}

// Baris yang sedang disunting dikecualikan dari pemeriksaan keunikan: menyimpan tanpa
// mengubah nomornya tidak boleh ditolak karena nomornya sendiri sudah dipakai dirinya.
func TestSaveAllowsUnchangedKeys(t *testing.T) {
	service, _ := newSampleService(t)

	input := validInput()
	input.Number = "1R-0716"
	input.Name = "FILTER OLI MESIN"
	input.Code = "FLT-ENG-001"

	_, err := service.Save(context.Background(), portalAlias, "SP0000000001", input,
		usecase.Actor{Login: "BUDI"}, nil)
	require.NoError(t, err)
}

// Keunikan diperiksa juga pada jalur simpan, dan SELURUH kunci yang bentrok dilaporkan
// sekaligus.
func TestSaveReportsEveryClashingKey(t *testing.T) {
	service, _ := newSampleService(t)

	// Baris kedua disunting memakai ketiga kunci milik baris PERTAMA.
	input := validInput()
	input.Number = "1R-0716"
	input.Name = "FILTER OLI MESIN"
	input.Code = "FLT-ENG-001"

	_, err := service.Save(context.Background(), portalAlias, "SP0000000002", input,
		usecase.Actor{Login: "BUDI"}, nil)

	var failure *mastersparepart.ValidationError
	require.ErrorAs(t, err, &failure)

	field := make([]string, 0, len(failure.Violation))
	for _, one := range failure.Violation {
		field = append(field, one.Field)
	}
	require.ElementsMatch(t,
		[]string{"nomor_sparepart", "nama_sparepart", "kode_sparepart"}, field)
}

func TestSaveRejectsMissingRow(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Save(context.Background(), portalAlias, "SP9999999999", validInput(),
		usecase.Actor{Login: "BUDI"}, nil)
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)

	_, err = service.Save(context.Background(), portalAlias, "  ", validInput(),
		usecase.Actor{Login: "BUDI"}, nil)
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)
}

func TestDecideChangesOnlyRowsThatMove(t *testing.T) {
	service, _ := newSampleService(t)

	// SP0000000002 menunggu, SP0000000001 sudah disetujui. Hanya yang pertama yang berubah.
	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"SP0000000002", "SP0000000001"},
		mastersparepart.StatusApproved, usecase.Actor{Login: "BUDI"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestDecideDropsDuplicateAndEmptyKeys(t *testing.T) {
	service, _ := newSampleService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"SP0000000002", " SP0000000002 ", "", "   "},
		mastersparepart.StatusApproved, usecase.Actor{Login: "BUDI"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"", "  "},
		mastersparepart.StatusApproved, usecase.Actor{Login: "BUDI"}, nil)

	var failure *mastersparepart.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "id_sparepart", failure.Violation[0].Field)
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"SP0000000002"},
		"9", usecase.Actor{Login: "BUDI"}, nil)
	require.ErrorIs(t, err, mastersparepart.ErrUnknownStatus)
}

// Keputusan TIDAK menyentuh USER_UPDATE, meniru `Activity/SetApprovalAllMaster` yang
// menetapkan APPROVAL dan tidak satu pun kolom lain.
//
// Akibatnya kolom itu tetap berisi siapa yang MENGAJUKAN, bukan siapa yang memutuskan — dan
// tidak ada tempat di tabel ini untuk mencatat yang kedua.
func TestDecideDoesNotTouchUpdatedBy(t *testing.T) {
	service, repo := newSampleService(t)

	before, err := repo.Get(context.Background(), "SP0000000002")
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, []string{"SP0000000002"},
		mastersparepart.StatusApproved, usecase.Actor{Login: "MANAGER"}, nil)
	require.NoError(t, err)

	after, err := repo.Get(context.Background(), "SP0000000002")
	require.NoError(t, err)
	require.Equal(t, before.UpdatedBy, after.UpdatedBy)
	require.NotEqual(t, "MANAGER", after.UpdatedBy)
}

func TestEnsurePortalReady(t *testing.T) {
	service, _ := newSampleService(t)

	require.NoError(t, service.EnsurePortalReady(portalAlias))
	require.Error(t, service.EnsurePortalReady("entitas-lain"))
}

func TestGetReturnsNotFound(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.Get(context.Background(), portalAlias, "SP9999999999")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)
}

func TestRepoFailureIsPropagated(t *testing.T) {
	repo := memory.NewSampleRepo()
	repo.SetError(errors.New("basis data mati"))
	service := newService(t, repo)

	_, err := service.List(context.Background(), portalAlias,
		mastersparepart.StatusApproved, "")
	require.Error(t, err)

	_, err = service.Options(context.Background(), portalAlias)
	require.Error(t, err)
}
