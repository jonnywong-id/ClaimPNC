package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
	"claim-pnc/internal/inboxsalvage/usecase"
)

var errBoom = errors.New("penyimpanan mati")

// failingRepo adalah pengisi seam yang selalu gagal dengan galat yang ditentukan.
type failingRepo struct{ err error }

func (f failingRepo) List(context.Context, inboxsalvage.Query, inboxsalvage.Pagination) (inboxsalvage.Page, error) {
	return inboxsalvage.Page{}, f.err
}

func (f failingRepo) Counts(context.Context, inboxsalvage.Caller) ([]inboxsalvage.StatusCount, error) {
	return nil, f.err
}

func (f failingRepo) Detail(context.Context, string) (inboxsalvage.Detail, error) {
	return inboxsalvage.Detail{}, f.err
}

func (f failingRepo) DetailByClaim(context.Context, string) (inboxsalvage.Detail, error) {
	return inboxsalvage.Detail{}, f.err
}

func (f failingRepo) Currencies(context.Context) ([]inboxsalvage.CurrencyOption, error) {
	return nil, f.err
}

func (f failingRepo) AttachDocument(
	context.Context, inboxsalvage.DocumentUpload,
) (inboxsalvage.AttachedDocument, error) {
	return inboxsalvage.AttachedDocument{}, f.err
}

func (f failingRepo) Create(context.Context, inboxsalvage.Form) (string, error) {
	return "", f.err
}

func (f failingRepo) MarkSentToAuction(
	context.Context, string, inboxsalvage.AuctionReceipt,
) error {
	return f.err
}

var caller = inboxsalvage.Caller{Login: memory.SampleCallerPIC}

// newService membentuk layanan dengan satu portal "ASM" berisi data contoh, dan logger
// yang menulis ke penyangga supaya jejaknya dapat diperiksa.
func newService(t *testing.T, repo inboxsalvage.Repo) (*usecase.Service, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxsalvage.Repo, error) {
			if alias != "ASM" {
				return nil, errors.New("portal belum siap")
			}
			return repo, nil
		},
		Logger: slog.New(slog.NewTextHandler(&logs, nil)),
	})
	require.NoError(t, err)
	return service, &logs
}

func TestNewServiceRequiresARepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")
}

func TestMetadataListsTheOfferedTabsAndEveryUploadColumn(t *testing.T) {
	service, _ := newService(t, memory.NewSampleStore())

	meta := service.Metadata(context.Background(), "ASM")
	require.Len(t, meta.Tabs, len(inboxsalvage.Tabs()))
	require.Equal(t, inboxsalvage.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxsalvage.StatusOptions(), meta.StatusOptions)
	require.Equal(t, []string{"item", "quantity", "satuan", "remarks"}, meta.UploadColumns)
	require.Equal(t, inboxsalvage.PlannedDifferences, meta.PlannedDifferences)

	// Satu isian Metadata BUKAN konstanta melainkan dibaca dari database portal. Tanpa
	// pernyataan ini, seluruh isian yang diperiksa di atas tetap lolos meskipun pembacaan
	// mata uangnya tidak pernah dipanggil sama sekali.
	require.Equal(t, []inboxsalvage.CurrencyOption{
		{Code: "IDR", Label: "IDR"},
		{Code: "USD", Label: "USD"},
	}, meta.CurrencyOptions)
}

// Mata uang yang tidak terbaca TIDAK boleh menutup layar.
//
// Ini satu-satunya tempat di modul ini yang menelan galat basis data, dan penelanan yang
// tidak diuji adalah penelanan yang diam-diam berubah menjadi galat sungguhan — atau
// sebaliknya, berubah menjadi hening tanpa jejak. Keduanya diperiksa di sini: layarnya
// tetap dapat digambar, DAN sebabnya tetap tertulis di jejak log.
func TestMetadataSurvivesCurrenciesFailingAndSaysWhy(t *testing.T) {
	service, logs := newService(t, failingRepo{err: errBoom})

	meta := service.Metadata(context.Background(), "ASM")
	require.Empty(t, meta.CurrencyOptions, "daftarnya kosong")
	require.NotNil(t, meta.CurrencyOptions, "kosong, BUKAN nil — layar tetap menggambar kolomnya")

	// Sisa Metadata tetap utuh: kegagalannya terbatas pada satu isian.
	require.Len(t, meta.Tabs, len(inboxsalvage.Tabs()))
	require.Equal(t, inboxsalvage.DefaultTab, meta.DefaultTab)

	require.Contains(t, logs.String(), "pilihan mata uang tidak terbaca")
	require.Contains(t, logs.String(), errBoom.Error())
}

// Portal yang tidak dapat dipilih menempuh cabang yang BERBEDA dari Currencies yang gagal,
// dan hasilnya harus sama: layar tetap tergambar, sebabnya tetap tercatat.
//
// Keduanya diuji terpisah karena keduanya memang dua cabang terpisah di currencies() —
// satu pernyataan saja akan membiarkan salah satunya berubah tanpa ketahuan.
func TestMetadataSurvivesAnUnselectablePortal(t *testing.T) {
	service, logs := newService(t, memory.NewSampleStore())

	meta := service.Metadata(context.Background(), "PORTAL-TIDAK-ADA")
	require.Empty(t, meta.CurrencyOptions)
	require.NotNil(t, meta.CurrencyOptions)
	require.Len(t, meta.Tabs, len(inboxsalvage.Tabs()))

	require.Contains(t, logs.String(), "pilihan mata uang dilewati")
	require.Contains(t, logs.String(), "portal belum siap")
}

func TestListReturnsThePageAndLogsWithoutTheSearchTerm(t *testing.T) {
	service, logs := newService(t, memory.NewSampleStore())

	// Premis fixture, diperiksa SEBELUM dipakai.
	//
	// Tab bawaan menyaring `STSSALVAGE` ∈ {3,5} DAN pekerjaan klaim yang belum ditutup,
	// sehingga klaim contoh mana pun dapat terlempar keluar darinya tanpa satu pun galat
	// — ujinya hanya akan menemukan nol baris dan menyalahkan pencariannya. Pernyataan
	// ini membuat sebabnya terbaca langsung. Ia pernah benar-benar terjadi: uji ini
	// mencari klaim yang tidak pernah ada di tab bawaan, dan kegagalannya tersembunyi
	// di balik berkas uji yang tidak dapat dikompilasi.
	const sought = "PNC-2044"
	all, err := service.List(context.Background(), "ASM", caller,
		inboxsalvage.QueryInput{}, inboxsalvage.Pagination{})
	require.NoError(t, err)
	require.Greater(t, len(all.Page.Items), 1, "tab bawaan harus berisi lebih dari satu baris")
	require.True(t, slices.ContainsFunc(all.Page.Items, func(row inboxsalvage.Row) bool {
		return row.ClaimNo == sought
	}), "klaim yang dicari harus ada di tab bawaan sebelum pencariannya diuji")

	listed, err := service.List(context.Background(), "ASM", caller,
		inboxsalvage.QueryInput{Search: sought}, inboxsalvage.Pagination{})
	require.NoError(t, err)
	require.Equal(t, inboxsalvage.DefaultTab, listed.Query.Tab.Code)
	require.Equal(t, sought, listed.Query.Search)
	require.Len(t, listed.Page.Items, 1, "pencarian menyempitkan daftarnya")
	require.Equal(t, sought, listed.Page.Items[0].ClaimNo)

	require.Contains(t, logs.String(), "daftar salvage dibuka")
	require.Contains(t, logs.String(), "mencari=true")
	require.NotContains(t, logs.String(), sought, "kata kunci tidak boleh dicatat")
}

func TestListRejectsInvalidQueriesUnknownPortalsAndRepoFailures(t *testing.T) {
	service, _ := newService(t, failingRepo{err: errBoom})

	_, err := service.List(context.Background(), "ASM", inboxsalvage.Caller{},
		inboxsalvage.QueryInput{}, inboxsalvage.Pagination{})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)

	_, err = service.List(context.Background(), "LAIN", caller,
		inboxsalvage.QueryInput{}, inboxsalvage.Pagination{})
	require.ErrorContains(t, err, "portal belum siap")

	_, err = service.List(context.Background(), "ASM", caller,
		inboxsalvage.QueryInput{}, inboxsalvage.Pagination{})
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "mengambil isi daftar outstanding")
}

// Tanpa logger, layanan tetap bekerja dan tidak panik.
func TestServiceWorksWithoutALogger(t *testing.T) {
	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxsalvage.Repo, error) { return store, nil },
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "ASM", caller,
		inboxsalvage.QueryInput{}, inboxsalvage.Pagination{})
	require.NoError(t, err)

	_, err = service.Detail(context.Background(), "ASM", caller,
		inboxsalvage.DetailKeyClaim, memory.SampleClaimWithoutSalvage)
	require.NoError(t, err)

	_, err = service.Create(context.Background(), "ASM", caller, validInput())
	require.NoError(t, err)
}

func TestCountsReturnsTheSummaryRows(t *testing.T) {
	service, _ := newService(t, memory.NewSampleStore())

	counts, err := service.Counts(context.Background(), "ASM", caller)
	require.NoError(t, err)
	require.Len(t, counts, len(inboxsalvage.CountRows()))
}

func TestCountsRejectsUnknownCallerPortalAndRepoFailures(t *testing.T) {
	service, _ := newService(t, failingRepo{err: errBoom})

	_, err := service.Counts(context.Background(), "ASM", inboxsalvage.Caller{Login: " "})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)

	_, err = service.Counts(context.Background(), "LAIN", caller)
	require.ErrorContains(t, err, "portal belum siap")

	_, err = service.Counts(context.Background(), "ASM", caller)
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "mengambil tabel ringkas salvage")
}

func TestDetailPicksTheLookupByKeyAndLogsTheSalvageID(t *testing.T) {
	store := memory.NewSampleStore()
	service, logs := newService(t, store)

	// Ambil satu ID pengajuan nyata dari daftar Histori.
	listed, err := service.List(context.Background(), "ASM", caller,
		inboxsalvage.QueryInput{Tab: inboxsalvage.TabHistori}, inboxsalvage.Pagination{})
	require.NoError(t, err)
	require.NotEmpty(t, listed.Page.Items)
	salvageID := listed.Page.Items[0].SalvageID

	detail, err := service.Detail(context.Background(), "ASM", caller,
		inboxsalvage.DetailKeySubmission, salvageID)
	require.NoError(t, err)
	require.Equal(t, salvageID, detail.SalvageID)
	require.True(t, detail.HasSubmission)
	require.Contains(t, logs.String(), "detail salvage dibuka")
	require.Contains(t, logs.String(), "id_salvage="+salvageID)

	byClaim, err := service.Detail(context.Background(), "ASM", caller,
		inboxsalvage.DetailKeyClaim, memory.SampleClaimWithoutSalvage)
	require.NoError(t, err)
	require.False(t, byClaim.HasSubmission)
	require.Equal(t, memory.SampleClaimWithoutSalvage, byClaim.ClaimNo)
}

func TestDetailPassesNotFoundThroughAndWrapsOtherFailures(t *testing.T) {
	notFound, _ := newService(t, failingRepo{err: inboxsalvage.ErrRowNotFound})
	_, err := notFound.Detail(context.Background(), "ASM", caller,
		inboxsalvage.DetailKeySubmission, "1")
	require.Equal(t, inboxsalvage.ErrRowNotFound, err, "tidak dibungkus")

	broken, _ := newService(t, failingRepo{err: errBoom})
	_, err = broken.Detail(context.Background(), "ASM", caller, inboxsalvage.DetailKeyClaim, "X")
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "mengambil detail salvage")

	_, err = broken.Detail(context.Background(), "ASM", inboxsalvage.Caller{},
		inboxsalvage.DetailKeyClaim, "X")
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)

	_, err = broken.Detail(context.Background(), "LAIN", caller, inboxsalvage.DetailKeyClaim, "X")
	require.ErrorContains(t, err, "portal belum siap")
}

func validInput() inboxsalvage.FormInput {
	return inboxsalvage.FormInput{
		ClaimNo:      "PNC-2044",
		ObjectName:   "Panel Listrik",
		CoverageName: "Property All Risk",
		SalvageType:  "Besi Tua",
		InputDate:    "2026-09-25",
		Items: []inboxsalvage.DetailItem{
			{Name: "Besi", Quantity: "2"},
			{},
		},
	}
}

func TestCreateStoresTheFormAndCountsOnlyNonEmptyItems(t *testing.T) {
	service, logs := newService(t, memory.NewSampleStore())

	created, err := service.Create(context.Background(), "ASM", caller, validInput())
	require.NoError(t, err)
	require.NotEmpty(t, created.SalvageID)
	require.Equal(t, 1, created.ItemCount, "baris kosong dibuang")
	require.Contains(t, logs.String(), "pengajuan salvage disimpan")
	require.Contains(t, logs.String(), "id_salvage="+created.SalvageID)
	require.NotContains(t, logs.String(), "PNC-2044", "nomor klaim tidak dicatat")
}

func TestCreateRejectsInvalidFormsUnknownPortalsAndRepoFailures(t *testing.T) {
	service, _ := newService(t, failingRepo{err: errBoom})

	_, err := service.Create(context.Background(), "ASM", caller, inboxsalvage.FormInput{})
	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.Create(context.Background(), "LAIN", caller, validInput())
	require.ErrorContains(t, err, "portal belum siap")

	_, err = service.Create(context.Background(), "ASM", caller, validInput())
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "menyimpan pengajuan salvage")
}
