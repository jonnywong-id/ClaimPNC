package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

func mustMoney(t *testing.T, text string) money.Money {
	t.Helper()
	value, err := money.Parse(text)
	require.NoError(t, err)
	return value
}

var at = time.Date(2026, 9, 20, 2, 0, 0, 0, time.UTC)

// --- master ambang ---

var thresholdColumns = []string{
	"ID", "NAME", "OPERATOR", "LINE", "TYPE", "LOWER", "UPPER", "DEGREE",
	"ACTIVE", "ADJ", "REG", "REJ", "ABS",
}

func TestListThresholdsConvertsEveryDriverType(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(exact("ambang_komite_daftar")).WillReturnRows(sqlmock.NewRows(thresholdColumns).
		AddRow(int64(7), " Komite ", []byte(" OPR "), "nonmbu", float64(2),
			"50000001.00", int64(100000000), float64(1),
			"1", int64(1), float64(1), "Y", true).
		AddRow(nil, nil, nil, "PA", nil, nil, nil, "3", nil, int64(0), float64(0), "N", false))

	rows, err := repo.ListThresholds(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 2)

	first := rows[0]
	require.Equal(t, "7", first.ID)
	require.Equal(t, "Komite", first.Name)
	require.Equal(t, "OPR", first.OperatorID)
	require.Equal(t, "2", first.CommitteeType)
	require.Equal(t, mustMoney(t, "50000001.00"), first.LowerBound)
	require.Equal(t, mustMoney(t, "100000000"), first.UpperBound)
	require.Equal(t, 1, first.Tier)
	require.True(t, first.Active)
	require.True(t, first.ForAdjustment)
	require.True(t, first.ForRegistration)
	require.True(t, first.ForRejection)
	require.True(t, first.Absent)

	second := rows[1]
	require.Equal(t, 3, second.Tier)
	require.False(t, second.Active)
	require.False(t, second.ForAdjustment)
	require.False(t, second.ForRejection)
	require.False(t, second.Absent)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListThresholdsFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	row := func(lower, upper any) *sqlmock.Rows {
		return sqlmock.NewRows(thresholdColumns).
			AddRow("1", "n", "o", "PA", "1", lower, upper, "1", "1", "1", "1", "1", "1")
	}

	mock.ExpectQuery(exact("ambang_komite_daftar")).WillReturnError(errDB)
	_, err := repo.ListThresholds(context.Background())
	require.ErrorContains(t, err, "membaca master ambang")

	mock.ExpectQuery(exact("ambang_komite_daftar")).WillReturnRows(row("bukan angka", "1"))
	_, err = repo.ListThresholds(context.Background())
	require.ErrorContains(t, err, "LIMIT_BOTTOM")

	mock.ExpectQuery(exact("ambang_komite_daftar")).WillReturnRows(row("1", "bukan angka"))
	_, err = repo.ListThresholds(context.Background())
	require.ErrorContains(t, err, "LIMIT_TOP")

	mock.ExpectQuery(exact("ambang_komite_daftar")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListThresholds(context.Background())
	require.ErrorContains(t, err, "membaca baris ambang")

	mock.ExpectQuery(exact("ambang_komite_daftar")).WillReturnRows(row("1", "2").RowError(0, errDB))
	_, err = repo.ListThresholds(context.Background())
	require.ErrorContains(t, err, "menelusuri master ambang")

	mock.ExpectQuery(exact("ambang_komite_periksa_tabel")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("ambang_komite_periksa_tabel")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "memeriksa tabel ambang")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConversionHelpers(t *testing.T) {
	require.Equal(t, "1.5", toText(1.5))
	require.Equal(t, "0", toText(false))
	require.Equal(t, "2026", toText(int32(2026)))

	value, err := toInt(" 4 ")
	require.NoError(t, err)
	require.Equal(t, 4, value)
	value, err = toInt("")
	require.NoError(t, err)
	require.Zero(t, value)
	_, err = toInt("x")
	require.Error(t, err)

	require.True(t, toFlag("y"))
	require.False(t, toFlag("0"))

	zero, err := moneyOrZero(nil)
	require.NoError(t, err)
	require.Equal(t, money.Zero, zero)

	require.Nil(t, nilIfEmpty("  "))
	require.Equal(t, "a", nilIfEmpty(" a "))
	require.Equal(t, `a\%b\_c\\`, searchPattern(` a%b_c\ `))
	require.Equal(t, "", searchPattern(" "))

	require.Equal(t, komite.OutcomePending, legacyOutcome(" "))
	require.Equal(t, komite.OutcomeApproved, legacyOutcome("1"))
	require.Equal(t, komite.OutcomeRejected, legacyOutcome("2"))
}

// --- jejak keputusan ---

var decisionColumns = []string{"ID", "CASE", "CLAIM", "TIER", "KIND", "NOTE", "LOGIN", "NAME", "AT"}

func TestDecisionRepoReadsAndWritesWhenTheTableExists(t *testing.T) {
	db, mock := newMock(t)
	repo := NewDecisionRepo(db)
	ctx := context.Background()

	// Tanpa kasus tidak ada satu pun kueri.
	byCase, err := repo.ListForCases(ctx, []string{" ", ""})
	require.NoError(t, err)
	require.Empty(t, byCase)

	mock.ExpectQuery(exact("decision_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	statement, _ := expandCases(query("decision_list_for_cases"), []string{"K-1", "K-2"})
	mock.ExpectQuery("^"+regexp.QuoteMeta(statement)+"$").WithArgs("K-1", "K-2").
		WillReturnRows(sqlmock.NewRows(decisionColumns).
			AddRow("D1", " K-1 ", " PNC-1 ", int64(1), " setuju ", "catatan", " budi ", " Budi ", at).
			AddRow("D2", "K-1", nil, nil, "tolak", nil, nil, nil, nil))

	byCase, err = repo.ListForCases(ctx, []string{"K-1", " K-2 ", "K-1"})
	require.NoError(t, err)
	require.Equal(t, komite.Decision{
		ID: "D1", CaseID: "K-1", ClaimNumber: "PNC-1", Tier: 1, Kind: komite.DecisionApprove,
		Note: "catatan", ActorLogin: komite.OperatorKey("budi"), ActorName: "Budi", DecidedAt: at,
	}, byCase["K-1"][0])
	require.Equal(t, komite.DecisionReject, byCase["K-1"][1].Kind)

	// Tabel sudah terbukti ada, sehingga tidak diperiksa ulang.
	mock.ExpectExec(exact("decision_insert")).
		WithArgs("D3", "K-1", nil, 2, "setuju", nil, komite.OperatorKey(" budi "), nil, at).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Record(ctx, komite.Decision{
		ID: "D3", CaseID: "K-1", Tier: 2, Kind: komite.DecisionApprove,
		ActorLogin: " budi ", DecidedAt: at.In(time.FixedZone("WIB", 7*3600)),
	}))
	require.True(t, repo.Available(ctx))

	mock.ExpectExec(exact("decision_insert")).
		WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.CPNC_KOMITE_KEPUTUSAN_UK) violated"))
	require.ErrorIs(t, repo.Record(ctx, komite.Decision{}), komite.ErrDecisionClosed)

	mock.ExpectExec(exact("decision_insert")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Record(ctx, komite.Decision{}), "mencatat keputusan komite")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecisionRepoWithoutTheTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewDecisionRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(exact("decision_check_table")).WillReturnError(errDB)
	byCase, err := repo.ListForCases(ctx, []string{"K-1"})
	require.NoError(t, err, "tabel yang belum ada berarti belum ada keputusan")
	require.Empty(t, byCase)

	// Dalam masa jeda pemeriksaan tidak diulang, dan penulisan ditolak.
	require.ErrorIs(t, repo.Record(ctx, komite.Decision{}), komite.ErrDecisionStoreUnavailable)
	require.False(t, repo.Available(ctx))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecisionRepoFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewDecisionRepo(db)
	ctx := context.Background()

	// Pemeriksaan pertama gagal; masa jeda dilewati dengan memundurkan waktu
	// pemeriksaan terakhir, lalu pemeriksaan kedua berhasil.
	mock.ExpectQuery(exact("decision_check_table")).WillReturnError(errDB)
	require.False(t, repo.Available(ctx))
	repo.tersedia.checkedAt = time.Now().Add(-2 * probeCooldown)
	mock.ExpectQuery(exact("decision_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	many := make([]string, maxCasesPerQuery+1)
	for i := range many {
		many[i] = "K-" + time.Duration(i).String()
	}
	_, err := repo.ListForCases(ctx, many)
	require.ErrorContains(t, err, "melebihi batas")

	statement, _ := expandCases(query("decision_list_for_cases"), []string{"K-1"})
	matcher := "^" + regexp.QuoteMeta(statement) + "$"

	mock.ExpectQuery(matcher).WillReturnError(errDB)
	_, err = repo.ListForCases(ctx, []string{"K-1"})
	require.ErrorContains(t, err, "membaca keputusan komite")

	mock.ExpectQuery(matcher).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListForCases(ctx, []string{"K-1"})
	require.ErrorContains(t, err, "membaca baris keputusan")

	mock.ExpectQuery(matcher).WillReturnRows(sqlmock.NewRows(decisionColumns).
		AddRow("D", "K-1", nil, nil, "setuju", nil, nil, nil, nil).RowError(0, errDB))
	_, err = repo.ListForCases(ctx, []string{"K-1"})
	require.ErrorContains(t, err, "menelusuri keputusan komite")

	mock.ExpectQuery(exact("decision_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(ctx))
	mock.ExpectQuery(exact("decision_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(ctx), "memeriksa tabel keputusan komite")
	require.NoError(t, mock.ExpectationsWereMet())
}

// --- inbox ---

var caseColumns = []string{
	"CASE", "CLAIM", "POLICY", "INSURED", "BUSINESS", "SOB", "BRANCH", "OPERATOR",
	"COMMITTEE", "CREATED", "STATUS", "APPROVE",
}

func caseRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(" K-1 ", " PNC-1 ", " POL ", " PT A ", " FIRE ", " Direct ", " Jakarta ",
		" budi ", at, at, " Open ", "1")
}

func TestInboxListCountsThenReadsOnePage(t *testing.T) {
	db, mock := newMock(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()

	page, err := repo.ListCases(ctx, komite.InboxFilter{})
	require.NoError(t, err)
	require.Empty(t, page.Cases, "tanpa operator tidak ada yang dibaca")

	f := komite.InboxFilter{
		Operator: "budi", Kind: komite.InboxAccepted, Search: "a_b",
		DateFrom: at, DateTo: at, Limit: 10,
	}
	normalized := f.Normalize()
	args := filterArgs(normalized)
	require.Equal(t, `a\_b`, args[6])
	require.Equal(t, normalized.DateTo.AddDate(0, 0, 1), args[11])

	mock.ExpectQuery(exact("inbox_count")).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(4))
	mock.ExpectQuery(exact("inbox_list")).WillReturnRows(caseRow(sqlmock.NewRows(caseColumns)))
	page, err = repo.ListCases(ctx, f)
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.Equal(t, komite.CommitteeCase{
		CaseID: "K-1", ClaimNumber: "PNC-1", PolicyNumber: "POL", InsuredName: "PT A",
		BusinessName: "FIRE", SourceOfBusiness: "Direct", BranchName: "Jakarta",
		AssignedOperator: "budi", CommitteeDate: at, CreatedAt: at, WorkStatus: "Open",
		LegacyOutcome: komite.OutcomeApproved,
	}, page.Cases[0])

	// Semua operator: pemilik diikat NULL.
	all := komite.InboxFilter{AllOperators: true, Limit: 5}.Normalize()
	require.Nil(t, filterArgs(all)[0])
	require.Nil(t, filterArgs(all)[9], "tanpa tanggal awal")
	mock.ExpectQuery(exact("inbox_count")).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	mock.ExpectQuery(exact("inbox_list")).WillReturnRows(sqlmock.NewRows(caseColumns))
	page, err = repo.ListCases(ctx, komite.InboxFilter{AllOperators: true, Limit: 5})
	require.NoError(t, err)
	require.Empty(t, page.Cases)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInboxListFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()
	f := komite.InboxFilter{Operator: "budi", Limit: 10}
	count := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"TOTAL"}).AddRow(1) }

	mock.ExpectQuery(exact("inbox_count")).WillReturnError(errDB)
	_, err := repo.ListCases(ctx, f)
	require.ErrorContains(t, err, "menghitung inbox")

	mock.ExpectQuery(exact("inbox_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("inbox_list")).WillReturnError(errDB)
	_, err = repo.ListCases(ctx, f)
	require.ErrorContains(t, err, "membaca inbox")

	mock.ExpectQuery(exact("inbox_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("inbox_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListCases(ctx, f)
	require.ErrorContains(t, err, "membaca baris inbox")

	mock.ExpectQuery(exact("inbox_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("inbox_list")).
		WillReturnRows(caseRow(sqlmock.NewRows(caseColumns)).RowError(0, errDB))
	_, err = repo.ListCases(ctx, f)
	require.ErrorContains(t, err, "menelusuri inbox")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInboxSummaryFindAndCheck(t *testing.T) {
	db, mock := newMock(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()

	summary, err := repo.Summarize(ctx, komite.InboxFilter{})
	require.NoError(t, err)
	require.Equal(t, komite.InboxSummary{}, summary)

	mock.ExpectQuery(exact("inbox_summary")).
		WillReturnRows(sqlmock.NewRows([]string{"O", "A", "R"}).AddRow(3, 1, nil))
	summary, err = repo.Summarize(ctx, komite.InboxFilter{Operator: "budi"})
	require.NoError(t, err)
	require.Equal(t, komite.InboxSummary{Outstanding: 3, Accepted: 1}, summary)

	mock.ExpectQuery(exact("inbox_summary")).WillReturnError(errDB)
	_, err = repo.Summarize(ctx, komite.InboxFilter{AllOperators: true})
	require.ErrorContains(t, err, "menghitung isi kotak")

	_, err = repo.FindCase(ctx, " ", "budi")
	require.ErrorIs(t, err, komite.ErrCaseNotFound)

	mock.ExpectQuery(exact("inbox_get")).WithArgs(komite.OperatorKey("budi"), "K-1").
		WillReturnRows(caseRow(sqlmock.NewRows(caseColumns)))
	found, err := repo.FindCase(ctx, " K-1 ", "budi")
	require.NoError(t, err)
	require.Equal(t, "K-1", found.CaseID)

	mock.ExpectQuery(exact("inbox_get")).WillReturnRows(sqlmock.NewRows(caseColumns))
	_, err = repo.FindCase(ctx, "K-1", "budi")
	require.ErrorIs(t, err, komite.ErrCaseNotFound)

	mock.ExpectQuery(exact("inbox_get")).WillReturnError(errDB)
	_, err = repo.FindCase(ctx, "K-1", "budi")
	require.ErrorContains(t, err, "membaca kasus komite")

	mock.ExpectQuery(exact("inbox_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTables(ctx))
	mock.ExpectQuery(exact("inbox_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTables(ctx), "memeriksa tabel inbox komite")
	require.NoError(t, mock.ExpectationsWereMet())
}
