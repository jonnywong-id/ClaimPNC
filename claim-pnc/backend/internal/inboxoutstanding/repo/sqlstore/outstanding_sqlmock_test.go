package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

// newMockRepo membentuk Repo di atas sqlmock yang mencocokkan kueri dengan regex.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exactly mengubah teks kueri bernama menjadi regex yang mencocokkannya persis.
func exactly(name string) string {
	return "^" + regexp.QuoteMeta(query(name)) + "$"
}

// claimColumns mengikuti urutan pemindaian scanClaim.
var claimColumns = []string{
	"CLAIM_ID", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "BUSINESS_NAME",
	"BUSINESS_SOURCE", "BRANCH_NAME", "GROUP_PANEL", "RCV_ID",
	"REGISTER_DATE", "CREATED_AT", "LOSS_DATE", "REPORT_DATE", "AGING",
	"PROCESS_STATUS", "CLAIM_STATUS", "PROGRESS_STATUS",
	"TECHNICAL_PIC", "RECORDED_BY", "CURRENT_STAGE", "CURRENT_HOLDER", "DOCUMENT_DONE",
}

var (
	registered = time.Date(2026, time.September, 15, 20, 0, 0, 0, time.UTC)
	created    = time.Date(2026, time.September, 14, 1, 0, 0, 0, time.UTC)
	lossAt     = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	reportAt   = time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
)

// fullRow adalah baris yang setiap kolomnya terisi.
func fullRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(
		"KLM-1", " PNCN.26.0001 ", " POL-1 ", "PT Satu", "Fire", "Direct", "JKT", " 006 ", " RCV-1 ",
		registered, created, lossAt, reportAt, int64(4),
		" New ", " Register ", " On Progress ",
		" PICSATU ", " ADMIN ", " Komite ", " BUDI ", " 1 ",
	)
}

// sparseRow adalah baris yang hampir seluruh kolomnya NULL; register date kosong sehingga
// tanggal pembuatan baris dipakai sebagai cadangan.
func sparseRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(
		"KLM-2", nil, nil, nil, nil, nil, nil, nil, nil,
		nil, created, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, "0",
	)
}

func assignedFilter() inboxoutstanding.Filter {
	return inboxoutstanding.Filter{AssignedTo: " budi ", Limit: 10, Offset: 20}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestListCountsThenReadsThePageWithTheSameFilters(t *testing.T) {
	repo, mock := newMockRepo(t)
	f := assignedFilter().Normalize()
	filters := append(filterArgs(f), statusArgs(f)...)

	mock.ExpectQuery(exactly("my_inbox_count")).
		WithArgs(toDriver(filters)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(42))

	listArgs := append(append([]any(nil), filters...), 20, 10)
	mock.ExpectQuery(exactly("my_inbox_list")).
		WithArgs(toDriver(listArgs)...).
		WillReturnRows(sparseRow(fullRow(sqlmock.NewRows(claimColumns))))

	page, err := repo.List(context.Background(), assignedFilter())
	require.NoError(t, err)
	require.Equal(t, 42, page.Total)
	require.Len(t, page.Claims, 2)

	full := page.Claims[0]
	require.Equal(t, "KLM-1", full.ClaimID)
	require.Equal(t, "PNCN.26.0001", full.ClaimNumber)
	require.Equal(t, "POL-1", full.PolicyNumber)
	require.Equal(t, "PT Satu", full.InsuredName)
	require.Equal(t, "Fire", full.BusinessName)
	require.Equal(t, "Direct", full.BusinessSource)
	require.Equal(t, "JKT", full.BranchName)
	require.Equal(t, "006", full.GroupPanel)
	require.Equal(t, "RCV-1", full.RCVID)
	require.Equal(t, registered, full.RegisteredAt, "REGISTERDATE_1 didahulukan")
	require.Equal(t, lossAt, *full.LossDate)
	require.Equal(t, reportAt, *full.ReportDate)
	require.Equal(t, 4, *full.AgingDays)
	require.Equal(t, "New", full.ProcessStatus)
	require.Equal(t, "Register", full.ClaimStatus)
	require.Equal(t, "On Progress", full.ProgressStatus)
	require.Equal(t, "PICSATU", full.TechnicalPIC)
	require.Equal(t, "ADMIN", full.RecordedBy)
	require.Equal(t, "Komite", full.CurrentStage)
	require.Equal(t, "BUDI", full.CurrentHolder)
	require.True(t, full.DocumentComplete)

	sparse := page.Claims[1]
	require.Empty(t, sparse.ClaimNumber)
	require.Equal(t, created, sparse.RegisteredAt, "PXCREATEDATETIME menjadi cadangan")
	require.Nil(t, sparse.LossDate)
	require.Nil(t, sparse.ReportDate)
	require.Nil(t, sparse.AgingDays)
	require.False(t, sparse.DocumentComplete, "'0' berarti belum lengkap")

	require.NoError(t, mock.ExpectationsWereMet())
}

// toDriver mengubah argumen menjadi bentuk yang diterima WithArgs; nil dicocokkan
// sebagai nil.
func toDriver(args []any) []driver.Value {
	out := make([]driver.Value, len(args))
	for i, arg := range args {
		out[i] = arg
	}
	return out
}

func TestListWithoutAnAssigneeDoesNotTouchTheDatabase(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, err := repo.List(context.Background(), inboxoutstanding.Filter{AssignedTo: "  "})
	require.ErrorIs(t, err, inboxoutstanding.ErrAssigneeRequired)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsACountFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00942")
	mock.ExpectQuery(exactly("my_inbox_count")).WillReturnError(cause)

	_, err := repo.List(context.Background(), assignedFilter())
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menghitung klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsAListQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-01008")
	mock.ExpectQuery(exactly("my_inbox_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exactly("my_inbox_list")).WillReturnError(cause)

	_, err := repo.List(context.Background(), assignedFilter())
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "membaca daftar klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsAScanFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactly("my_inbox_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	// Kolom tanggal berisi teks yang bukan waktu.
	mock.ExpectQuery(exactly("my_inbox_list")).
		WillReturnRows(sqlmock.NewRows(claimColumns).AddRow(
			"KLM-1", nil, nil, nil, nil, nil, nil, nil, nil,
			"bukan-tanggal", nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil, nil,
		))

	_, err := repo.List(context.Background(), assignedFilter())
	require.ErrorContains(t, err, "membaca baris klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsARowIterationFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("sambungan putus")
	mock.ExpectQuery(exactly("my_inbox_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exactly("my_inbox_list")).
		WillReturnRows(fullRow(sqlmock.NewRows(claimColumns)).RowError(0, cause))

	_, err := repo.List(context.Background(), assignedFilter())
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menelusuri daftar klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

func TestExportBindsTheLineFiveTimesAndEachDateTwice(t *testing.T) {
	repo, mock := newMockRepo(t)
	wib := time.FixedZone("WIB", 7*60*60)
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, wib)
	to := time.Date(2026, time.September, 30, 0, 0, 0, 0, wib)

	args := []any{"PA", "PA", "PA", "PA", "PA", from.UTC(), from.UTC(), to.UTC(), to.UTC()}
	mock.ExpectQuery(exactly("my_inbox_export_count")).
		WithArgs(toDriver(args)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(3))
	mock.ExpectQuery(exactly("my_inbox_export")).
		WithArgs(toDriver(append(append([]any(nil), args...), 0, 500))...).
		WillReturnRows(fullRow(sqlmock.NewRows(claimColumns)))

	page, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{
		LineBusiness: "pa", From: &from, To: &to, Limit: 500,
	})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Claims, 1)
	require.Equal(t, "KLM-1", page.Claims[0].ClaimID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExportWithoutScopeOrDatesBindsEmptyLineAndNulls(t *testing.T) {
	repo, mock := newMockRepo(t)
	args := []any{"", "", "", "", "", nil, nil, nil, nil}

	mock.ExpectQuery(exactly("my_inbox_export_count")).
		WithArgs(toDriver(args)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	mock.ExpectQuery(exactly("my_inbox_export")).
		WithArgs(toDriver(append(append([]any(nil), args...), 0, inboxoutstanding.DefaultLimit))...).
		WillReturnRows(sqlmock.NewRows(claimColumns))

	page, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.NotNil(t, page.Claims)
	require.Empty(t, page.Claims)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExportWrapsEachFailure(t *testing.T) {
	cause := errors.New("ORA-03113")

	t.Run("hitung", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactly("my_inbox_export_count")).WillReturnError(cause)

		_, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{})
		require.ErrorIs(t, err, cause)
		require.ErrorContains(t, err, "menghitung baris export")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kueri", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactly("my_inbox_export_count")).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
		mock.ExpectQuery(exactly("my_inbox_export")).WillReturnError(cause)

		_, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{})
		require.ErrorIs(t, err, cause)
		require.ErrorContains(t, err, "membaca baris export")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactly("my_inbox_export_count")).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
		mock.ExpectQuery(exactly("my_inbox_export")).
			WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("KLM-1"))

		_, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{})
		require.ErrorContains(t, err, "membaca satu baris export")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("telusur", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactly("my_inbox_export_count")).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
		mock.ExpectQuery(exactly("my_inbox_export")).
			WillReturnRows(fullRow(sqlmock.NewRows(claimColumns)).RowError(0, cause))

		_, err := repo.Export(context.Background(), inboxoutstanding.ExportFilter{})
		require.ErrorIs(t, err, cause)
		require.ErrorContains(t, err, "menelusuri baris export")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ---------------------------------------------------------------------------
// Lini bisnis dan identitas lama
// ---------------------------------------------------------------------------

func TestLineBusinessForReadsAndNormalizesTheColumn(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactly("line_business_for")).
		WithArgs("SITI").
		WillReturnRows(sqlmock.NewRows([]string{"LINE_BUSINESS"}).AddRow(" travel "))

	line, err := repo.LineBusinessFor(context.Background(), " siti ")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineTravel, line)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLineBusinessForTreatsMissingAndEmptyAsUnknown(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Login kosong tidak menyentuh basis data.
	line, err := repo.LineBusinessFor(context.Background(), "  ")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineUnknown, line)

	mock.ExpectQuery(exactly("line_business_for")).
		WithArgs("TANPABARIS").
		WillReturnRows(sqlmock.NewRows([]string{"LINE_BUSINESS"}))
	line, err = repo.LineBusinessFor(context.Background(), "tanpabaris")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineUnknown, line)

	mock.ExpectQuery(exactly("line_business_for")).
		WithArgs("KOSONG").
		WillReturnRows(sqlmock.NewRows([]string{"LINE_BUSINESS"}).AddRow(nil))
	line, err = repo.LineBusinessFor(context.Background(), "kosong")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineUnknown, line)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLineBusinessForWrapsAQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00942")
	mock.ExpectQuery(exactly("line_business_for")).WillReturnError(cause)

	line, err := repo.LineBusinessFor(context.Background(), "siti")
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "membaca lini bisnis petugas")
	require.Equal(t, inboxoutstanding.LineUnknown, line)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyOperatorForReturnsADifferentOldIdentity(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactly("legacy_operator_for")).
		WithArgs("BUDI@CONTOH").
		WillReturnRows(sqlmock.NewRows([]string{"OLD"}).AddRow(" budilama "))

	legacy, err := repo.LegacyOperatorFor(context.Background(), "budi@contoh")
	require.NoError(t, err)
	require.Equal(t, "BUDILAMA", legacy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyOperatorForReturnsNothingWhenTheIdentityNeverChanged(t *testing.T) {
	repo, mock := newMockRepo(t)

	legacy, err := repo.LegacyOperatorFor(context.Background(), " ")
	require.NoError(t, err)
	require.Empty(t, legacy)

	mock.ExpectQuery(exactly("legacy_operator_for")).
		WithArgs("SAMA").
		WillReturnRows(sqlmock.NewRows([]string{"OLD"}).AddRow("sama"))
	legacy, err = repo.LegacyOperatorFor(context.Background(), "sama")
	require.NoError(t, err)
	require.Empty(t, legacy)

	mock.ExpectQuery(exactly("legacy_operator_for")).
		WithArgs("TANPABARIS").
		WillReturnRows(sqlmock.NewRows([]string{"OLD"}))
	legacy, err = repo.LegacyOperatorFor(context.Background(), "tanpabaris")
	require.NoError(t, err)
	require.Empty(t, legacy)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyOperatorForWrapsAQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactly("legacy_operator_for")).WillReturnError(sql.ErrConnDone)

	_, err := repo.LegacyOperatorFor(context.Background(), "siti")
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.ErrorContains(t, err, "membaca identitas lama petugas")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Ringkasan
// ---------------------------------------------------------------------------

func TestSummarizeDocumentStatusBindsTheFiltersWithoutTheStatus(t *testing.T) {
	repo, mock := newMockRepo(t)
	f := inboxoutstanding.Filter{
		AssignedTo: "budi", DocumentStatus: inboxoutstanding.StatusComplete,
	}.Normalize()

	mock.ExpectQuery(exactly("my_inbox_document_status")).
		WithArgs(toDriver(filterArgs(f))...).
		WillReturnRows(sqlmock.NewRows([]string{"LENGKAP", "BELUM", "TOTAL"}).AddRow(2, 5, 7))

	summary, err := repo.SummarizeDocumentStatus(context.Background(), f)
	require.NoError(t, err)
	require.Equal(t, 7, summary.Total)

	counts := map[inboxoutstanding.DocumentStatus]*int{}
	for _, item := range summary.Status {
		counts[item.Status] = item.Count
	}
	require.Equal(t, 2, *counts[inboxoutstanding.StatusComplete])
	require.Equal(t, 5, *counts[inboxoutstanding.StatusIncomplete])
	require.Equal(t, 7, *counts[inboxoutstanding.StatusAll])
	require.NoError(t, mock.ExpectationsWereMet())
}

// SUM atas himpunan kosong mengembalikan NULL; inbox kosong harus menghasilkan nol.
func TestSummarizeDocumentStatusReadsNullSumsAsZero(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactly("my_inbox_document_status")).
		WillReturnRows(sqlmock.NewRows([]string{"LENGKAP", "BELUM", "TOTAL"}).AddRow(nil, nil, nil))

	summary, err := repo.SummarizeDocumentStatus(context.Background(),
		inboxoutstanding.Filter{AssignedTo: "budi"})
	require.NoError(t, err)
	require.Zero(t, summary.Total)
	require.Equal(t, 0, *summary.Status[0].Count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSummarizeDocumentStatusRequiresAnAssignee(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, err := repo.SummarizeDocumentStatus(context.Background(), inboxoutstanding.Filter{})
	require.ErrorIs(t, err, inboxoutstanding.ErrAssigneeRequired)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSummarizeDocumentStatusWrapsAQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00904")
	mock.ExpectQuery(exactly("my_inbox_document_status")).WillReturnError(cause)

	_, err := repo.SummarizeDocumentStatus(context.Background(),
		inboxoutstanding.Filter{AssignedTo: "budi"})
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "meringkas status dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nama kueri yang tidak ada adalah cacat pemrograman dan harus panik.
func TestUnknownQueryNamePanics(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxoutstanding/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { query("tidak_ada") })
}
