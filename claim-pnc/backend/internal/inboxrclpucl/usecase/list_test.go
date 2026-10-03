package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// failingRepo adalah pengisi seam yang selalu gagal dengan galat yang ditentukan.
//
// inboxrclpucl.Repo disematkan dan SENGAJA dibiarkan nil: uji di berkas ini hanya menyentuh
// jalur baca. Metode lain akan panic bila kelak dipanggil — lebih jujur daripada mengembalikan
// galat yang sama, karena fake ini memang tidak dirancang untuk jalur itu.
type failingRepo struct {
	inboxrclpucl.Repo

	err error
}

func (f failingRepo) List(
	context.Context, inboxrclpucl.Query, inboxrclpucl.Pagination,
) (inboxrclpucl.Page, error) {
	return inboxrclpucl.Page{}, f.err
}

func (f failingRepo) DailyReport(
	context.Context, inboxrclpucl.DateRange, inboxrclpucl.Pagination,
) ([]inboxrclpucl.DailyReportRow, int, error) {
	return nil, 0, f.err
}

func (f failingRepo) Detail(context.Context, string) (inboxrclpucl.ClaimDetail, error) {
	return inboxrclpucl.ClaimDetail{}, f.err
}

// newService membentuk layanan di atas satu repo dan pencatat yang menulis ke buf.
func newService(t *testing.T, repo inboxrclpucl.Repo, buf *bytes.Buffer) *usecase.Service {
	t.Helper()

	var logger *slog.Logger
	if buf != nil {
		logger = slog.New(slog.NewTextHandler(buf, nil))
	}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return repo, nil },
		Logger:       logger,
	})
	require.NoError(t, err)
	return service
}

func caller() inboxrclpucl.Caller {
	return inboxrclpucl.Caller{Login: "PETUGASCONTOH"}
}

func TestNewServiceRequiresARepoSelector(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
	require.Nil(t, service)
	require.Contains(t, err.Error(), "RepoSelector wajib diisi")
}

func TestMetadataReturnsCopiesOfTheScreenShape(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	meta := service.Metadata()
	require.Len(t, meta.Tabs, 3)
	require.Equal(t, inboxrclpucl.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxrclpucl.DailyReportColumns, meta.ReportColumns)
	require.Equal(t, inboxrclpucl.PlannedDifferences, meta.PlannedDifferences)

	// Menulisi hasilnya tidak boleh mengubah daftar aslinya.
	meta.ReportColumns[0].Title = "DIUBAH"
	meta.PlannedDifferences[0].Summary = "DIUBAH"
	require.NotEqual(t, "DIUBAH", inboxrclpucl.DailyReportColumns[0].Title)
	require.NotEqual(t, "DIUBAH", inboxrclpucl.PlannedDifferences[0].Summary)
}

func TestListReturnsTheTabAndLogsTheOpening(t *testing.T) {
	var buf bytes.Buffer
	service := newService(t, memory.NewSampleStore(), &buf)

	listed, err := service.List(context.Background(), "ASM", caller(),
		inboxrclpucl.QueryInput{}, inboxrclpucl.Pagination{})
	require.NoError(t, err)

	// Tab kosong menjadi tab bawaan, dan jawabannya menyebut tab yang sebenarnya dipakai.
	require.Equal(t, inboxrclpucl.DefaultTab, listed.Query.Tab.Code)
	require.Equal(t, 2, listed.Page.Total)
	require.Len(t, listed.Page.Items, 2)

	require.Contains(t, buf.String(), "antrean RCL/PUCL dibuka")
	require.Contains(t, buf.String(), "pemanggil=PETUGASCONTOH")
	require.Contains(t, buf.String(), "portal=ASM")
}

func TestListWorksWithoutALogger(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	listed, err := service.List(context.Background(), "ASM", caller(),
		inboxrclpucl.QueryInput{Tab: inboxrclpucl.TabKelengkapanDokumen},
		inboxrclpucl.Pagination{})
	require.NoError(t, err)
	require.Equal(t, inboxrclpucl.TabKelengkapanDokumen, listed.Query.Tab.Code)
}

func TestListRejectsAnInvalidQuery(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.List(context.Background(), "ASM", inboxrclpucl.Caller{},
		inboxrclpucl.QueryInput{}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, inboxrclpucl.ErrCallerUnknown)

	_, err = service.List(context.Background(), "ASM", caller(),
		inboxrclpucl.QueryInput{Tab: "9"}, inboxrclpucl.Pagination{})
	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestListPropagatesARepoSelectorFailure(t *testing.T) {
	selectorErr := errors.New("portal belum siap")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return nil, selectorErr },
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "ASM", caller(),
		inboxrclpucl.QueryInput{}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, selectorErr)

	_, err = service.Detail(context.Background(), "ASM", caller(), "kunci")
	require.ErrorIs(t, err, selectorErr)

	_, err = service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{From: "2026-09-01", To: "2026-09-30"},
		inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, selectorErr)
}

func TestListWrapsARepoFailureWithTheTabCode(t *testing.T) {
	repoErr := errors.New("koneksi putus")
	service := newService(t, failingRepo{err: repoErr}, nil)

	_, err := service.List(context.Background(), "ASM", caller(),
		inboxrclpucl.QueryInput{Tab: inboxrclpucl.TabKlaimMSIG}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, repoErr)
	require.Contains(t, err.Error(), "mengambil isi tab 3")
}

func TestDetailReturnsTheClaimAndLogsItsNumber(t *testing.T) {
	var buf bytes.Buffer
	service := newService(t, memory.NewSampleStore(), &buf)

	detail, err := service.Detail(context.Background(), "ASM", caller(),
		"  ASM-FW-GCNMFW-WORK PNC-700001  ")
	require.NoError(t, err)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-700001", detail.Reference)
	require.NotEmpty(t, detail.ClaimNumber)

	require.Contains(t, buf.String(), "layar kerja RCL/PUCL dibuka")
	require.Contains(t, buf.String(), "klaim="+detail.ClaimNumber)
}

func TestDetailWorksWithoutALogger(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	detail, err := service.Detail(context.Background(), "ASM", caller(),
		"ASM-FW-GCNMFW-WORK PNC-700002")
	require.NoError(t, err)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-700002", detail.Reference)
}

func TestDetailRequiresACaller(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.Detail(context.Background(), "ASM",
		inboxrclpucl.Caller{Login: "   "}, "ASM-FW-GCNMFW-WORK PNC-700001")
	require.ErrorIs(t, err, inboxrclpucl.ErrCallerUnknown)
}

func TestDetailRejectsAnEmptyReference(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.Detail(context.Background(), "ASM", caller(), "   ")

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxrclpucl.FieldReference, validation.Violations[0].Field)
}

func TestDetailPassesNotFoundThroughUnwrapped(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.Detail(context.Background(), "ASM", caller(),
		"ASM-FW-GCNMFW-WORK PNC-999999")

	// Tidak dibungkus: galatnya PERSIS ErrClaimNotFound.
	require.Equal(t, inboxrclpucl.ErrClaimNotFound, err)
}

func TestDetailWrapsOtherRepoFailures(t *testing.T) {
	repoErr := errors.New("koneksi putus")
	service := newService(t, failingRepo{err: repoErr}, nil)

	_, err := service.Detail(context.Background(), "ASM", caller(), "KUNCI-1")
	require.ErrorIs(t, err, repoErr)
	require.Contains(t, err.Error(), "mengambil layar kerja klaim KUNCI-1")
}

func TestDailyReportReturnsRowsTotalAndNormalizedPage(t *testing.T) {
	var buf bytes.Buffer
	service := newService(t, memory.NewSampleStore(), &buf)

	reported, err := service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{From: "2026-09-01", To: "2026-09-30"},
		inboxrclpucl.Pagination{})
	require.NoError(t, err)

	require.NotEmpty(t, reported.Rows)
	require.Equal(t, len(reported.Rows), reported.Total)
	require.Equal(t, inboxrclpucl.Pagination{}.Normalize(), reported.Pagination)
	require.Equal(t, "2026-09-01", reported.Request.Range.From)
	require.Equal(t, "2026-09-30", reported.Request.Range.To)

	require.Contains(t, buf.String(), "laporan harian RCL/PUCL diminta")
	require.Contains(t, buf.String(), "dari=2026-09-01")
	require.Contains(t, buf.String(), "sampai=2026-09-30")
}

func TestDailyReportWorksWithoutALogger(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	reported, err := service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{From: "2026-09-10", To: "2026-09-10"},
		inboxrclpucl.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 10, reported.Pagination.Size)
}

func TestDailyReportRejectsAnInvalidRequest(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{}, inboxrclpucl.Pagination{})
	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)

	_, err = service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{Tab: inboxrclpucl.TabKlaimMSIG}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, inboxrclpucl.ErrReportNotAvailable)
}

func TestDailyReportWrapsARepoFailureWithTheRange(t *testing.T) {
	repoErr := errors.New("koneksi putus")
	service := newService(t, failingRepo{err: repoErr}, nil)

	_, err := service.DailyReport(context.Background(), "ASM", caller(),
		inboxrclpucl.ReportInput{From: "2026-09-01", To: "2026-09-02"},
		inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, repoErr)
	require.Contains(t, err.Error(), "mengambil laporan harian 2026-09-01..2026-09-02")
}
