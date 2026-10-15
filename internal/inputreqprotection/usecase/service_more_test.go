package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
	"claim-pnc/internal/inputreqprotection/repo/memory"
	"claim-pnc/internal/inputreqprotection/usecase"
)

// failingRepo membungkus repo memori dan dapat dipaksa gagal per method.
type failingRepo struct {
	*memory.Repo
	listErr, dupErr, createErr, updateErr error
	dupCalls                              []inputreqprotection.DuplicateKey
	dupExcept                             []string
}

func (f *failingRepo) List(ctx context.Context, fl inputreqprotection.Filter) (inputreqprotection.Page, error) {
	if f.listErr != nil {
		return inputreqprotection.Page{}, f.listErr
	}
	return f.Repo.List(ctx, fl)
}

func (f *failingRepo) HasDuplicate(ctx context.Context, k inputreqprotection.DuplicateKey, except string) (bool, error) {
	f.dupCalls = append(f.dupCalls, k)
	f.dupExcept = append(f.dupExcept, except)
	if f.dupErr != nil {
		return false, f.dupErr
	}
	return f.Repo.HasDuplicate(ctx, k, except)
}

func (f *failingRepo) Create(ctx context.Context, d inputreqprotection.Draft, c inputreqprotection.Claim, sel inputreqprotection.CoverageRow, by string, at time.Time) (inputreqprotection.Protection, error) {
	if f.createErr != nil {
		return inputreqprotection.Protection{}, f.createErr
	}
	return f.Repo.Create(ctx, d, c, sel, by, at)
}

func (f *failingRepo) Update(ctx context.Context, n string, d inputreqprotection.Draft, c inputreqprotection.Claim, sel inputreqprotection.CoverageRow, by string, at time.Time) (inputreqprotection.Protection, error) {
	if f.updateErr != nil {
		return inputreqprotection.Protection{}, f.updateErr
	}
	return f.Repo.Update(ctx, n, d, c, sel, by, at)
}

// failingTypes adalah master tipe yang selalu gagal.
type failingTypes struct{ err error }

func (f failingTypes) ListTypes(context.Context) ([]inputreqprotection.ProtectionType, error) {
	return nil, f.err
}

var errBoom = errors.New("basis data mati")

// buildWith membentuk service atas penyimpanan yang ditentukan uji.
func buildWith(t *testing.T, at time.Time, stores inputreqprotection.Stores) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		Protections: func(alias string) (inputreqprotection.Stores, error) {
			if alias != portal {
				return inputreqprotection.Stores{}, errors.New("portal tidak dikenal")
			}
			return stores, nil
		},
		Now:      func() time.Time { return at },
		Location: wib,
	})
	require.NoError(t, err)
	return service
}

func defaultStores(repo inputreqprotection.Repo) inputreqprotection.Stores {
	return inputreqprotection.Stores{
		Protections: repo,
		Types:       memory.NewTypeRepoWithSamples(),
		Claims:      memory.NewClaimRepoWithSamples(),
		Causes:      memory.NewCauseRepoWithSamples(),
	}
}

var fixedAt = time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)

func TestNewServiceRequiresRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.EqualError(t, err, "inputreqprotection/usecase: pemilih repo proteksi wajib diisi")
}

func TestNewServiceDefaultsClockAndLocation(t *testing.T) {
	// Tanpa Now dan Location, service tetap dapat menyimpan memakai jam sistem dan WIB.
	repo := memory.NewRepo()
	service, err := usecase.NewService(usecase.Options{
		Protections: func(string) (inputreqprotection.Stores, error) { return defaultStores(repo), nil },
	})
	require.NoError(t, err)

	before := time.Now()
	saved, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)
	require.False(t, saved.CreatedAt.Before(before))
	require.True(t, inputreqprotection.IssuedHere(saved.Number))
}

func TestListWrapsRepoError(t *testing.T) {
	repo := &failingRepo{Repo: memory.NewRepo(), listErr: errBoom}
	service := buildWith(t, fixedAt, defaultStores(repo))

	_, err := service.List(context.Background(), usecase.ListQuery{PortalAlias: portal})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca daftar proteksi")
}

func TestListPassesSearchAndPagingToRepo(t *testing.T) {
	service, repo := bangun(t, fixedAt)
	repo.Add(
		inputreqprotection.Protection{Number: "OPCN.26.0001", PolicyNumber: "POL-A", CreatedAt: fixedAt},
		inputreqprotection.Protection{Number: "OPCN.26.0002", PolicyNumber: "POL-B", CreatedAt: fixedAt},
		inputreqprotection.Protection{Number: "OPCN.26.0003", PolicyNumber: "POL-B", CreatedAt: fixedAt},
	)

	page, err := service.List(context.Background(), usecase.ListQuery{
		PortalAlias: portal, Search: "pol-b", Limit: 1, Offset: 1,
	})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Protections, 1)
	require.Equal(t, "OPCN.26.0002", page.Protections[0].Number)
}

func TestGetReturnsProtectionAndNotFound(t *testing.T) {
	service, repo := bangun(t, fixedAt)
	repo.Add(inputreqprotection.Protection{Number: "OPC-201", Note: "contoh"})

	p, err := service.Get(context.Background(), portal, "opc-201")
	require.NoError(t, err)
	require.Equal(t, "contoh", p.Note)

	_, err = service.Get(context.Background(), portal, "OPC-999")
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)

	_, err = service.Get(context.Background(), "entitas-lain", "OPC-201")
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestListTypesReadsMasterOfActivePortal(t *testing.T) {
	service, _ := bangun(t, fixedAt)

	types, err := service.ListTypes(context.Background(), portal)
	require.NoError(t, err)
	require.Len(t, types, 9)
	require.Equal(t, inputreqprotection.ProtectionType{ID: "2", Name: "Premi Belum Lunas"}, types[1])

	_, err = service.ListTypes(context.Background(), "entitas-lain")
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestListTypesWrapsMasterError(t *testing.T) {
	stores := defaultStores(memory.NewRepo())
	stores.Types = failingTypes{err: errBoom}
	service := buildWith(t, fixedAt, stores)

	_, err := service.ListTypes(context.Background(), portal)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca master tipe proteksi")
}

func TestFindClaimDelegatesToClaimRepo(t *testing.T) {
	service, _ := bangun(t, fixedAt)

	c, err := service.FindClaim(context.Background(), portal, "PNC-1865")
	require.NoError(t, err)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1865", c.PegaID)

	// Galat klaim tidak ditemukan diteruskan apa adanya, tidak dibungkus.
	_, err = service.FindClaim(context.Background(), portal, "PNCN.26.9999")
	require.Equal(t, inputreqprotection.ErrClaimNotFound, err)

	_, err = service.FindClaim(context.Background(), "entitas-lain", "PNC-1865")
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestCreateRejectsUnknownPortalAfterValidation(t *testing.T) {
	service, _ := bangun(t, fixedAt)
	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: "entitas-lain", Draft: draft(), By: "ADMINCONTOH",
	})
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestCreateRejectsUnknownClaim(t *testing.T) {
	service, _ := bangun(t, fixedAt)
	d := draft()
	d.ClaimNumber = "PNCN.26.9999"

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: d, By: "ADMINCONTOH",
	})
	require.ErrorIs(t, err, inputreqprotection.ErrClaimNotFound)
}

func TestCreateUsesClaimPolicyForDuplicateKey(t *testing.T) {
	// Kunci ganda memakai polis dari KLAIM dan hari WIB dari jam service.
	repo := &failingRepo{Repo: memory.NewRepo()}
	service := buildWith(t, fixedAt, defaultStores(repo))

	saved, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.NoError(t, err)
	require.Equal(t, "99.001.2026.00000001", saved.PolicyNumber)

	require.Len(t, repo.dupCalls, 1)
	require.Equal(t, "99.001.2026.00000001", repo.dupCalls[0].PolicyNumber)
	require.Equal(t, "1", repo.dupCalls[0].Type)
	require.Equal(t, time.Date(2026, time.September, 23, 0, 0, 0, 0, wib), repo.dupCalls[0].Day)
	require.Equal(t, "", repo.dupExcept[0])
}

func TestCreateWrapsDuplicateCheckError(t *testing.T) {
	repo := &failingRepo{Repo: memory.NewRepo(), dupErr: errBoom}
	service := buildWith(t, fixedAt, defaultStores(repo))

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "memeriksa proteksi ganda")
}

func TestCreateWrapsRepoCreateError(t *testing.T) {
	repo := &failingRepo{Repo: memory.NewRepo(), createErr: errBoom}
	service := buildWith(t, fixedAt, defaultStores(repo))

	_, err := service.Create(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Draft: draft(), By: "ADMINCONTOH",
	})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menyimpan proteksi")
}

// editableRow adalah proteksi yang belum tertaut klaim dan belum diakseptasi.
func editableRow() inputreqprotection.Protection {
	return inputreqprotection.Protection{
		Number: "OPCN.26.0050", PolicyNumber: "99.001.2026.00000001", Type: "1",
		InputDate: fixedAt, CreatedAt: fixedAt, CreatedBy: "PEMBUAT",
	}
}

func TestUpdateRequiresCaller(t *testing.T) {
	service, _ := bangun(t, fixedAt)
	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(),
	})
	require.EqualError(t, err, "inputreqprotection/usecase: identitas pemanggil kosong")
}

func TestUpdateRejectsUnknownPortalAndMissingRow(t *testing.T) {
	service, _ := bangun(t, fixedAt)

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: "entitas-lain", Number: "OPCN.26.0050", Draft: draft(), By: "U",
	})
	require.EqualError(t, err, "portal tidak dikenal")

	_, err = service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(), By: "U",
	})
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)
}

func TestUpdateValidatesDraftOfEditableRow(t *testing.T) {
	service, repo := bangun(t, fixedAt)
	repo.Add(editableRow())

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: inputreqprotection.Draft{}, By: "U",
	})
	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Len(t, v.Errors, 3)
}

func TestUpdateRejectsUnknownClaim(t *testing.T) {
	service, repo := bangun(t, fixedAt)
	repo.Add(editableRow())
	d := draft()
	d.ClaimNumber = "PNCN.26.9999"

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: d, By: "U",
	})
	require.ErrorIs(t, err, inputreqprotection.ErrClaimNotFound)
}

func TestUpdateExcludesItselfFromDuplicateCheck(t *testing.T) {
	// Polis dan tipe yang sama dengan dirinya sendiri tidak dianggap ganda.
	repo := &failingRepo{Repo: memory.NewRepo()}
	repo.Add(editableRow())
	service := buildWith(t, fixedAt, defaultStores(repo))

	saved, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(), By: "PENYUNTING",
	})
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0007", saved.ClaimNumber)
	require.Equal(t, "PEMBUAT", saved.CreatedBy, "pembuat tidak berubah saat disunting")
	require.Equal(t, []string{"OPCN.26.0050"}, repo.dupExcept)
}

func TestUpdateRejectsDuplicateOfAnotherRow(t *testing.T) {
	service, repo := bangun(t, fixedAt)
	other := editableRow()
	other.Number = "OPCN.26.0051"
	repo.Add(editableRow(), other)

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(), By: "U",
	})
	var v *inputreqprotection.ValidationError
	require.True(t, errors.As(err, &v))
	require.Equal(t, inputreqprotection.DuplicateMessage, v.Errors[0].Message)
}

func TestUpdateWrapsDuplicateAndRepoErrors(t *testing.T) {
	repo := &failingRepo{Repo: memory.NewRepo(), dupErr: errBoom}
	repo.Add(editableRow())
	service := buildWith(t, fixedAt, defaultStores(repo))

	_, err := service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(), By: "U",
	})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "memeriksa proteksi ganda")

	repo.dupErr = nil
	repo.updateErr = errBoom
	_, err = service.Update(context.Background(), usecase.SaveCommand{
		PortalAlias: portal, Number: "OPCN.26.0050", Draft: draft(), By: "U",
	})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menyunting proteksi")
}
