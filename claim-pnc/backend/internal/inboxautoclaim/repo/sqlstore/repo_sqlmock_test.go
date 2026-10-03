package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah kueri bernama milik satu tab menjadi pola regexp literal.
func q(source inboxautoclaim.Source, name string) string {
	return "^" + regexp.QuoteMeta(getQueryFor(source, name)) + "$"
}

// qd memakai tab bawaan.
func qd(name string) string { return q(inboxautoclaim.DefaultSource, name) }

var (
	batchColumns = []string{"KODE", "NAMA", "NOBATCH", "TGL", "UPLOAD", "PROSES", "SUKSES", "GAGAL", "OLEH"}
	lineColumns  = []string{"KODE", "BATCH", "POLIS", "PRODKE", "CLAIMID", "AKSEP", "CUR", "CURCODE", "NILAI",
		"COL", "DOL", "LAPOR", "TGLPROSES", "NOTE", "KEYWORD", "OBJEK", "FLAG", "PESAN", "OLEH"}
	processedAt = time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
)

func TestListBatchAllCompanies(t *testing.T) {
	repo, mock := newMock(t)
	source := inboxautoclaim.SourceAneka
	mock.ExpectQuery(q(source, "auto_claim_batch_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(41))
	mock.ExpectQuery(q(source, "auto_claim_batch_list")).
		WithArgs(inboxautoclaim.MessageSuccess, inboxautoclaim.MessageSuccess, 20, 20).
		WillReturnRows(sqlmock.NewRows(batchColumns).
			AddRow(" BRI ", " Bank ", " 0001 ", processedAt, 10, 9, 8, 1, " admin ").
			AddRow("BNI", nil, "0002", nil, nil, nil, nil, nil, nil))

	page, err := repo.ListBatch(context.Background(), inboxautoclaim.BatchFilter{Source: source,
		Page: inboxautoclaim.PageRequest{Number: 2, Size: 20}})
	require.NoError(t, err)
	require.Equal(t, 41, page.Total)
	require.Equal(t, inboxautoclaim.Batch{CompanyCode: "BRI", CompanyName: "Bank", BatchNumber: "0001",
		ProcessedDate: "03/09/2026", Uploaded: 10, Processed: 9, Succeeded: 8, Failed: 1, UploadedBy: "admin"}, page.Item[0])
	require.Empty(t, page.Item[1].ProcessedDate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBatchByCompanyAndErrors(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceKredit
	filter := inboxautoclaim.BatchFilter{Source: source, CompanyCode: " BRI "}

	repo, mock := newMock(t)
	mock.ExpectQuery(q(source, "auto_claim_batch_count_by_company")).WithArgs("BRI").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(nil))
	mock.ExpectQuery(q(source, "auto_claim_batch_list_by_company")).
		WithArgs(inboxautoclaim.MessageSuccess, inboxautoclaim.MessageSuccess, "BRI", 0, inboxautoclaim.DefaultPageSize).
		WillReturnRows(sqlmock.NewRows(batchColumns))
	page, err := repo.ListBatch(ctx, filter)
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Item)

	mock.ExpectQuery(q(source, "auto_claim_batch_count")).WillReturnError(errDB)
	_, err = repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: source})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "auto_claim_batch_count")

	mock.ExpectQuery(q(source, "auto_claim_batch_count_by_company")).WillReturnError(errDB)
	_, err = repo.ListBatch(ctx, filter)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q(source, "auto_claim_batch_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(q(source, "auto_claim_batch_list")).WillReturnError(errDB)
	_, err = repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: source})
	require.ErrorContains(t, err, "membaca daftar batch")

	mock.ExpectQuery(q(source, "auto_claim_batch_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(q(source, "auto_claim_batch_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: source})
	require.ErrorContains(t, err, "membaca baris batch")

	mock.ExpectQuery(q(source, "auto_claim_batch_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(q(source, "auto_claim_batch_list")).WillReturnRows(sqlmock.NewRows(batchColumns).
		AddRow("a", "b", "c", nil, 1, 1, 1, 1, "d").RowError(0, errDB))
	_, err = repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: source})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListCompanyAndSummary(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(qd("auto_claim_company_list")).WillReturnRows(
		sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow(" BRI ", " Bank ").AddRow(nil, nil))
	companies, err := repo.ListCompany(ctx)
	require.NoError(t, err)
	require.Equal(t, []inboxautoclaim.Company{{Code: "BRI", Name: "Bank"}, {}}, companies)

	mock.ExpectQuery(qd("auto_claim_company_list")).WillReturnError(errDB)
	_, err = repo.ListCompany(ctx)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(qd("auto_claim_company_list")).WillReturnRows(sqlmock.NewRows([]string{"K"}).AddRow("x"))
	_, err = repo.ListCompany(ctx)
	require.ErrorContains(t, err, "membaca perusahaan")
	mock.ExpectQuery(qd("auto_claim_company_list")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "N"}).AddRow("a", "b").RowError(0, errDB))
	_, err = repo.ListCompany(ctx)
	require.ErrorIs(t, err, errDB)

	source := inboxautoclaim.SourceTravel
	mock.ExpectQuery(q(source, "auto_claim_company_summary")).WillReturnRows(
		sqlmock.NewRows([]string{"KODE", "NAMA", "N"}).AddRow(" BRI ", "Bank", 3).AddRow("BNI", "Nasional", 4))
	summary, err := repo.SummarizeCompany(ctx, source)
	require.NoError(t, err)
	require.Equal(t, 7, summary.Total)
	require.Equal(t, inboxautoclaim.CompanySummary{Code: "BRI", Name: "Bank", BatchCount: 3}, summary.Company[0])

	mock.ExpectQuery(q(source, "auto_claim_company_summary")).WillReturnRows(sqlmock.NewRows([]string{"K", "N", "C"}))
	summary, err = repo.SummarizeCompany(ctx, source)
	require.NoError(t, err)
	require.NotNil(t, summary.Company, "ringkasan kosong tetap senarai, bukan nil")

	mock.ExpectQuery(q(source, "auto_claim_company_summary")).WillReturnError(errDB)
	_, err = repo.SummarizeCompany(ctx, source)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q(source, "auto_claim_company_summary")).WillReturnRows(sqlmock.NewRows([]string{"K"}).AddRow("x"))
	_, err = repo.SummarizeCompany(ctx, source)
	require.ErrorContains(t, err, "membaca ringkasan")
	mock.ExpectQuery(q(source, "auto_claim_company_summary")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "N", "C"}).AddRow("a", "b", 1).RowError(0, errDB))
	_, err = repo.SummarizeCompany(ctx, source)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func lineRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(" BRI ", " 0001 ", " P1 ", " 1 ", " C1 ", " A1 ", " 10001 ", " IDR ", " 100 ",
		" Banjir ", " 01/09/2026 ", " 02/09/2026 ", processedAt, " n ", " k ", " o ", " Y ", " Sukses Klaim ", " admin ")
}

func TestListLinePicksQueryByResult(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceAneka
	cases := []struct {
		result      inboxautoclaim.Result
		list, count string
		filtered    bool
	}{
		{inboxautoclaim.ResultAll, "auto_claim_line_list", "auto_claim_line_count", false},
		{inboxautoclaim.ResultSucceeded, "auto_claim_line_list_succeeded", "auto_claim_line_count_succeeded", true},
		{inboxautoclaim.ResultFailed, "auto_claim_line_list_failed", "auto_claim_line_count_failed", true},
	}
	for _, tc := range cases {
		t.Run(string(tc.result), func(t *testing.T) {
			repo, mock := newMock(t)
			countArgs := []driver.Value{"BRI", "0001"}
			listArgs := []driver.Value{"BRI", "0001"}
			if tc.filtered {
				countArgs = append(countArgs, inboxautoclaim.MessageSuccess)
				listArgs = append(listArgs, inboxautoclaim.MessageSuccess)
			}
			listArgs = append(listArgs, 0, 5)
			mock.ExpectQuery(q(source, tc.count)).WithArgs(countArgs...).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectQuery(q(source, tc.list)).WithArgs(listArgs...).WillReturnRows(lineRow(sqlmock.NewRows(lineColumns)))

			page, err := repo.ListLine(ctx, inboxautoclaim.LineQuery{Source: source, CompanyCode: " BRI ",
				BatchNumber: " 0001 ", Result: tc.result, Page: inboxautoclaim.PageRequest{Size: 5}})
			require.NoError(t, err)
			require.Equal(t, 1, page.Total)
			require.Equal(t, inboxautoclaim.Line{CompanyCode: "BRI", BatchNumber: "0001", PolicyNo: "P1",
				ProductSeq: "1", ClaimID: "C1", AcceptanceNo: "A1", Currency: "10001", CurrencyCode: "IDR",
				ClaimAmount: "100", CauseOfLoss: "Banjir", DateOfLoss: "01/09/2026", ReportDate: "02/09/2026",
				ProcessedDate: "03/09/2026", Note: "n", Keyword: "k", ObjectName: "o", FlagNoPayout: "Y",
				Message: "Sukses Klaim", UploadedBy: "admin"}, page.Item[0])
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestListLineErrors(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceKredit
	query := inboxautoclaim.LineQuery{Source: source}
	count := func(m sqlmock.Sqlmock) {
		m.ExpectQuery(q(source, "auto_claim_line_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	}
	repo, mock := newMock(t)

	mock.ExpectQuery(q(source, "auto_claim_line_count")).WillReturnError(errDB)
	_, err := repo.ListLine(ctx, query)
	require.ErrorIs(t, err, errDB)

	count(mock)
	mock.ExpectQuery(q(source, "auto_claim_line_list")).WillReturnError(errDB)
	_, err = repo.ListLine(ctx, query)
	require.ErrorContains(t, err, "membaca rincian batch")

	count(mock)
	mock.ExpectQuery(q(source, "auto_claim_line_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListLine(ctx, query)
	require.ErrorContains(t, err, "membaca baris rincian")

	count(mock)
	mock.ExpectQuery(q(source, "auto_claim_line_list")).WillReturnRows(lineRow(sqlmock.NewRows(lineColumns)).RowError(0, errDB))
	_, err = repo.ListLine(ctx, query)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExportLine(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceTravel
	columns := []string{"KODE", "POLIS", "CLAIMID", "AKSEP", "NILAI", "COL", "CUR", "PESAN"}
	cases := []struct {
		result   inboxautoclaim.Result
		name     string
		filtered bool
	}{
		{inboxautoclaim.ResultAll, "auto_claim_export", false},
		{inboxautoclaim.ResultSucceeded, "auto_claim_export_succeeded", true},
		{inboxautoclaim.ResultFailed, "auto_claim_export_failed", true},
	}
	for _, tc := range cases {
		t.Run(string(tc.result), func(t *testing.T) {
			repo, mock := newMock(t)
			args := []driver.Value{"BRI", "0002"}
			if tc.filtered {
				args = append(args, inboxautoclaim.MessageSuccess)
			}
			mock.ExpectQuery(q(source, tc.name)).WithArgs(args...).WillReturnRows(sqlmock.NewRows(columns).
				AddRow(" BRI ", " P1 ", " C1 ", " A1 ", " 100 ", " Banjir ", " IDR ", " pesan "))
			got, err := repo.ExportLine(ctx, inboxautoclaim.LineQuery{Source: source, CompanyCode: "BRI",
				BatchNumber: " 0002 ", Result: tc.result})
			require.NoError(t, err)
			require.Equal(t, []inboxautoclaim.Line{{CompanyCode: "BRI", BatchNumber: "0002", PolicyNo: "P1",
				ClaimID: "C1", AcceptanceNo: "A1", ClaimAmount: "100", CauseOfLoss: "Banjir",
				CurrencyCode: "IDR", Message: "pesan"}}, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	repo, mock := newMock(t)
	query := inboxautoclaim.LineQuery{Source: source}
	mock.ExpectQuery(q(source, "auto_claim_export")).WillReturnError(errDB)
	_, err := repo.ExportLine(ctx, query)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q(source, "auto_claim_export")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ExportLine(ctx, query)
	require.ErrorContains(t, err, "membaca baris ekspor")
	mock.ExpectQuery(q(source, "auto_claim_export")).WillReturnRows(sqlmock.NewRows(columns).
		AddRow("a", "b", "c", "d", "e", "f", "g", "h").RowError(0, errDB))
	_, err = repo.ExportLine(ctx, query)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchExists(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceAneka
	repo, mock := newMock(t)

	mock.ExpectQuery(q(source, "auto_claim_batch_exists")).WithArgs("BRI", "0001").
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	found, err := repo.BatchExists(ctx, source, " BRI ", " 0001 ")
	require.NoError(t, err)
	require.True(t, found)

	mock.ExpectQuery(q(source, "auto_claim_batch_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	found, err = repo.BatchExists(ctx, source, "BRI", "9")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(q(source, "auto_claim_batch_exists")).WillReturnError(errDB)
	_, err = repo.BatchExists(ctx, source, "BRI", "9")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveReceiver(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(qd("auto_claim_receiver")).WithArgs("P1").
		WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow(" BRI ", " Bank "))
	company, found, err := repo.ResolveReceiver(ctx, " P1 ")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, inboxautoclaim.Company{Code: "BRI", Name: "Bank"}, company)

	mock.ExpectQuery(qd("auto_claim_receiver")).WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow(" ", "x"))
	_, found, err = repo.ResolveReceiver(ctx, "P1")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_receiver")).WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}))
	_, found, err = repo.ResolveReceiver(ctx, "P1")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_receiver")).WillReturnError(errDB)
	_, _, err = repo.ResolveReceiver(ctx, "P1")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindPolicy(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(qd("auto_claim_policy")).WithArgs("P1").WillReturnRows(sqlmock.NewRows([]string{"PRODKE"}).AddRow(" 2 "))
	seq, found, err := repo.FindPolicyProductSeq(ctx, " P1 ")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "2", seq)

	mock.ExpectQuery(qd("auto_claim_policy")).WillReturnRows(sqlmock.NewRows([]string{"PRODKE"}).AddRow(nil))
	_, found, err = repo.FindPolicyProductSeq(ctx, "P1")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_policy")).WillReturnRows(sqlmock.NewRows([]string{"PRODKE"}))
	_, found, err = repo.FindPolicyProductSeq(ctx, "P1")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_policy")).WillReturnError(errDB)
	_, _, err = repo.FindPolicyProductSeq(ctx, "P1")
	require.ErrorIs(t, err, errDB)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	columns := []string{"MULAI", "AKHIR", "BISNIS", "STATUS", "BATAL", "CUR", "SOB", "PANEL"}
	mock.ExpectQuery(qd("auto_claim_policy_detail")).WithArgs("P1", "2").WillReturnRows(sqlmock.NewRows(columns).
		AddRow(start, end, " 01 ", " A ", " 0 ", " IDR ", " BRI ", " 006 "))
	detail, found, err := repo.FindPolicyDetail(ctx, " P1 ", " 2 ")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, inboxautoclaim.PolicyDetail{StartDate: "01/01/2026", EndDate: "31/12/2026", BusinessCode: "01",
		StatusBusiness: "A", FlagEdmBatal: "0", Currency: "IDR", SourceOfBusiness: "BRI", GroupPanel: "006"}, detail)

	mock.ExpectQuery(qd("auto_claim_policy_detail")).WillReturnRows(sqlmock.NewRows(columns))
	_, found, err = repo.FindPolicyDetail(ctx, "P1", "2")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_policy_detail")).WillReturnError(errDB)
	_, _, err = repo.FindPolicyDetail(ctx, "P1", "2")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurrencyID(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	id, found, err := repo.CurrencyID(ctx, "  ")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, id)

	mock.ExpectQuery(qd("auto_claim_currency_id")).WithArgs("IDR").WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 10001 "))
	id, found, err = repo.CurrencyID(ctx, " IDR ")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "10001", id)

	mock.ExpectQuery(qd("auto_claim_currency_id")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	_, found, err = repo.CurrencyID(ctx, "XXX")
	require.NoError(t, err)
	require.False(t, found)

	mock.ExpectQuery(qd("auto_claim_currency_id")).WillReturnError(errDB)
	_, _, err = repo.CurrencyID(ctx, "XXX")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContractClaimedAndOpenProtection(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	claimed, err := repo.ContractClaimed(ctx, "BRI", " ")
	require.NoError(t, err)
	require.False(t, claimed)

	mock.ExpectQuery(qd("auto_claim_contract_claimed")).WithArgs("BRI", "K-01", inboxautoclaim.MessageSuccess).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	claimed, err = repo.ContractClaimed(ctx, " BRI ", " k-01 ")
	require.NoError(t, err)
	require.True(t, claimed)

	mock.ExpectQuery(qd("auto_claim_open_protection")).WithArgs("P1", "1").WillReturnRows(sqlmock.NewRows([]string{"X"}))
	open, err := repo.HasOpenProtection(ctx, " P1 ", " 1 ")
	require.NoError(t, err)
	require.False(t, open)

	mock.ExpectQuery(qd("auto_claim_open_protection")).WillReturnError(errDB)
	_, err = repo.HasOpenProtection(ctx, "P1", "1")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "memeriksa open protection")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPremiumCheck(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(qd("auto_claim_premium_business")).WillReturnRows(
		sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow(" 01 ", " Fire ").AddRow(" ", "kosong"))
	mock.ExpectQuery(qd("auto_claim_premium_source")).WillReturnRows(
		sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow("BRI", "Bank"))
	choices, err := repo.PremiumCheckChoices(ctx)
	require.NoError(t, err)
	require.Equal(t, []inboxautoclaim.Choice{{Code: "01", Name: "Fire"}}, choices.Business)
	require.Equal(t, []inboxautoclaim.Choice{{Code: "BRI", Name: "Bank"}}, choices.SourceOfBusiness)

	mock.ExpectQuery(qd("auto_claim_premium_business")).WillReturnError(errDB)
	_, err = repo.PremiumCheckChoices(ctx)
	require.ErrorContains(t, err, "daftar bisnis")

	mock.ExpectQuery(qd("auto_claim_premium_business")).WillReturnRows(sqlmock.NewRows([]string{"K", "N"}))
	mock.ExpectQuery(qd("auto_claim_premium_source")).WillReturnRows(sqlmock.NewRows([]string{"K"}).AddRow("x"))
	_, err = repo.PremiumCheckChoices(ctx)
	require.ErrorContains(t, err, "daftar sumber bisnis")

	mock.ExpectQuery(qd("auto_claim_premium_business")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "N"}).AddRow("a", "b").RowError(0, errDB))
	_, err = repo.PremiumCheckChoices(ctx)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(qd("auto_claim_premium_claim_total")).WithArgs(inboxautoclaim.MessageSuccess, "01", "BRI").
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(" 1500 "))
	total, err := repo.SucceededClaimTotal(ctx, inboxautoclaim.PremiumCheckQuery{BusinessCode: "01", SourceOfBusiness: "BRI"})
	require.NoError(t, err)
	require.Equal(t, "1500", total)

	mock.ExpectQuery(qd("auto_claim_premium_claim_total")).WillReturnError(errDB)
	_, err = repo.SucceededClaimTotal(ctx, inboxautoclaim.PremiumCheckQuery{})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func uploadLines() []inboxautoclaim.UploadLine {
	return []inboxautoclaim.UploadLine{
		{CompanyCode: "BRI", ProductSeq: "1", CurrencyID: "10001", Row: inboxautoclaim.UploadRow{LineNumber: 1,
			PolicyNo: "P1", ClaimAmount: "100", ContractNo: "k1", ReportType: "A", PaymentDate: "05/09/2026"}},
		{CompanyCode: "BRI", Message: "Polis tidak ditemukan", Row: inboxautoclaim.UploadRow{LineNumber: 2,
			PolicyNo: "P2", ClaimAmount: "50", PaymentDate: "salah"}},
		{CompanyCode: "BNI", ProductSeq: "1", Row: inboxautoclaim.UploadRow{LineNumber: 3, PolicyNo: "P3"}},
	}
}

func TestInsertUploadKreditNumbersBatchPerCompany(t *testing.T) {
	repo, mock := newMock(t)
	source := inboxautoclaim.SourceKredit
	lines := uploadLines()
	briBatch := inboxautoclaim.NextBatchNumber([]string{"0001", "0002"})
	bniBatch := inboxautoclaim.NextBatchNumber(nil)
	paid := time.Date(2026, 9, 5, 0, 0, 0, 0, time.Local)

	mock.ExpectBegin()
	mock.ExpectQuery(q(source, "auto_claim_batch_number_used")).WithArgs("BRI").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(" 0001 ").AddRow("0002"))
	mock.ExpectExec(q(source, "auto_claim_line_insert_kredit")).WithArgs(briBatch, "BRI", "P1", "1", "admin",
		nil, nil, nil, "10001", "100", "K1", "A", paid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q(source, "auto_claim_line_insert_kredit")).WithArgs(briBatch, "BRI", "P2", nil, "admin",
		"Polis tidak ditemukan", "Polis tidak ditemukan", "Polis tidak ditemukan", nil, "50", nil, nil, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q(source, "auto_claim_batch_number_used")).WithArgs("BNI").
		WillReturnRows(sqlmock.NewRows([]string{"N"}))
	mock.ExpectExec(q(source, "auto_claim_line_insert_kredit")).WithArgs(bniBatch, "BNI", "P3", "1", "admin",
		nil, nil, nil, nil, "", nil, nil, nil).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.InsertUpload(context.Background(), source, lines, "admin")
	require.NoError(t, err)
	require.Equal(t, 3, result.Rows)
	require.Len(t, result.Batch, 2)
	require.Equal(t, briBatch, result.Batch[0].BatchNumber)
	require.Equal(t, 2, result.Batch[0].Rows)
	require.Equal(t, 1, result.Batch[0].Succeeded)
	require.Equal(t, 1, result.Batch[0].Failed)
	require.Equal(t, bniBatch, result.Batch[1].BatchNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertUploadTravelAndAnekaStatements(t *testing.T) {
	line := inboxautoclaim.UploadLine{CompanyCode: "BRI", ProductSeq: "2", CurrencyID: "10001",
		Row: inboxautoclaim.UploadRow{LineNumber: 1, PolicyNo: "P1", ClaimAmount: "10", DateOfLoss: "01/09/2026",
			ReportDate: "02/09/2026", CauseOfLoss: "Banjir", Reason: "r", Keyword: "k", ObjectName: "o",
			FlagNoPayout: "Y", ReportDescription: "d"}}

	travelName, travelArgs := insertStatement(inboxautoclaim.SourceTravel, "0001", "BRI", "admin", line)
	require.Equal(t, "auto_claim_line_insert_travel", travelName)
	require.Equal(t, []any{"0001", "BRI", "P1", "2", "admin", nil, nil, nil, "01/09/2026", "10001", "10", "Y", "d"}, travelArgs)

	anekaName, anekaArgs := insertStatement(inboxautoclaim.SourceAneka, "0001", "BRI", "admin", line)
	require.Equal(t, "auto_claim_line_insert", anekaName)
	require.Equal(t, []any{"0001", "BRI", "P1", "2", "admin", nil, "01/09/2026", "02/09/2026", "10001", "Banjir",
		"10", "r", "k", nil, nil, "o", "Y"}, anekaArgs)

	// Satu baris lewat jalur transaksi supaya kueri tab Travel ikut dibuktikan.
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q(inboxautoclaim.SourceTravel, "auto_claim_batch_number_used")).WillReturnRows(sqlmock.NewRows([]string{"N"}))
	mock.ExpectExec(q(inboxautoclaim.SourceTravel, "auto_claim_line_insert_travel")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := repo.InsertUpload(context.Background(), inboxautoclaim.SourceTravel, []inboxautoclaim.UploadLine{line}, "admin")
	require.NoError(t, err)
	require.Equal(t, 1, result.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertUploadFailures(t *testing.T) {
	ctx := context.Background()
	source := inboxautoclaim.SourceKredit
	lines := uploadLines()[:1]

	repo, _ := newMock(t)
	_, err := repo.InsertUpload(ctx, source, nil, "admin")
	require.ErrorIs(t, err, inboxautoclaim.ErrEmptyUpload)

	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, "memulai transaksi"},
		{"used query", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q(source, "auto_claim_batch_number_used")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "nomor batch terpakai"},
		{"used scan", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q(source, "auto_claim_batch_number_used")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
			m.ExpectRollback()
		}, "membaca nomor batch"},
		{"used rows", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q(source, "auto_claim_batch_number_used")).WillReturnRows(
				sqlmock.NewRows([]string{"N"}).AddRow("1").RowError(0, errDB))
			m.ExpectRollback()
		}, "menelusuri nomor batch"},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q(source, "auto_claim_batch_number_used")).WillReturnRows(sqlmock.NewRows([]string{"N"}))
			m.ExpectExec(q(source, "auto_claim_line_insert_kredit")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "menyisipkan baris 1"},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q(source, "auto_claim_batch_number_used")).WillReturnRows(sqlmock.NewRows([]string{"N"}))
			m.ExpectExec(q(source, "auto_claim_line_insert_kredit")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, "menutup transaksi unggah"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.InsertUpload(ctx, source, lines, "admin")
			if tc.name != "used scan" {
				require.ErrorIs(t, err, errDB)
			}
			require.ErrorContains(t, err, tc.text)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckTableReadsEveryTable(t *testing.T) {
	ctx := context.Background()
	expectAll := func(m sqlmock.Sqlmock) {
		for _, source := range inboxautoclaim.AllSource() {
			m.ExpectQuery(q(source, "auto_claim_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
		}
		m.ExpectQuery(qd("auto_claim_check_master")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	}

	repo, mock := newMock(t)
	expectAll(mock)
	mock.ExpectQuery(qd("auto_claim_check_currency")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	require.NoError(t, repo.CheckTable(ctx))
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMock(t)
	expectAll(mock)
	mock.ExpectQuery(qd("auto_claim_check_currency")).WillReturnError(errDB)
	err := repo.CheckTable(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "POOLDATA.CURRENCY")

	repo, mock = newMock(t)
	first := inboxautoclaim.AllSource()[0]
	mock.ExpectQuery(q(first, "auto_claim_check_table")).WillReturnError(errDB)
	err = repo.CheckTable(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "tidak dapat dibaca")
	// Berhenti pada tabel pertama yang gagal; tabel sesudahnya tidak dibaca.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPaymentDateAndNullable(t *testing.T) {
	require.Nil(t, paymentDate(" "))
	require.Nil(t, paymentDate("2026-09-05"))
	require.Equal(t, time.Date(2026, 9, 5, 0, 0, 0, 0, time.Local), paymentDate(" 05/09/2026 "))
	require.Nil(t, nullable("  "))
	require.Equal(t, "x", nullable("x"))
}

func TestGetQueryReturnsRawText(t *testing.T) {
	require.Contains(t, getQuery("auto_claim_batch_exists"), "{{TABEL}}")
}
