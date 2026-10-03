package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
)

func newDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

var leaderIdentity = inboxsurvey.SurveyorIdentity{
	Login: "ADJLEADER", Name: "BUDI", Scope: []string{" budi ", "", "Siti"},
}

func TestListSkipsUnavailableTabWithoutQuery(t *testing.T) {
	db, mock := newDB(t)

	page, err := NewRepo(db).List(context.Background(), leaderIdentity,
		inboxsurvey.Filter{Tab: inboxsurvey.TabOutstanding})
	require.NoError(t, err)
	require.Empty(t, page.Tasks)
	require.NotNil(t, page.Tasks)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBindsScopeTabAndScansTasks(t *testing.T) {
	db, mock := newDB(t)
	loss := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(query("list_tasks")).
		WithArgs("|BUDI|SITI|", "belum-dijawab", "ADJLEADER",
			inboxsurvey.CommunicationOpen, inboxsurvey.CommunicationAnswered,
			"PNC", 10, 5).
		WillReturnRows(sqlmock.NewRows(taskColumns).
			AddRow("S1", "C1", "1", "PNCN.26.1", "POL", "Nama", "Fire", "Api", "Jakarta",
				"PIC", "BUDI", loss, created, "Final Report", "2", 4).
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				nil, 4))

	page, err := NewRepo(db).List(context.Background(), leaderIdentity, inboxsurvey.Filter{
		Tab: inboxsurvey.TabNotAnswered, Search: " PNC ", Offset: 10, Limit: 5,
	})
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.Len(t, page.Tasks, 2)
	require.Equal(t, inboxsurvey.SurveyTask{
		SurveyID: "S1", ClaimID: "C1", SurveyIndex: "1", ClaimNumber: "PNCN.26.1",
		PolicyNumber: "POL", InsuredName: "Nama", ClassOfBusiness: "Fire", CauseOfLoss: "Api",
		Location: "Jakarta", TechnicalPIC: "PIC", AdjusterPIC: "BUDI", DateOfLoss: loss,
		CreatedAt: created, ASMStatus: "Final Report", SurveyorType: "2",
	}, page.Tasks[0])
	require.True(t, page.Tasks[1].DateOfLoss.IsZero())
	require.True(t, page.Tasks[1].CreatedAt.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBindsNilForEmptySearch(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("list_tasks")).
		WithArgs("|", "sudah-dibalas-asm", "", inboxsurvey.CommunicationOpen,
			inboxsurvey.CommunicationAnswered, nil, 0, inboxsurvey.DefaultLimit).
		WillReturnRows(sqlmock.NewRows(taskColumns))

	page, err := NewRepo(db).List(context.Background(), inboxsurvey.SurveyorIdentity{},
		inboxsurvey.Filter{Tab: inboxsurvey.TabReplied})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsErrors(t *testing.T) {
	filter := inboxsurvey.Filter{Tab: inboxsurvey.TabReplied}

	db, mock := newDB(t)
	mock.ExpectQuery(query("list_tasks")).WillReturnError(errors.New("ora"))
	_, err := NewRepo(db).List(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "menjalankan kueri list_tasks: ora")

	db, mock = newDB(t)
	mock.ExpectQuery(query("list_tasks")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = NewRepo(db).List(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "membaca baris kueri list_tasks")

	db, mock = newDB(t)
	mock.ExpectQuery(query("list_tasks")).
		WillReturnRows(sqlmock.NewRows(taskColumns).
			AddRow(make([]driver.Value, len(taskColumns))...).
			RowError(0, errors.New("putus")))
	_, err = NewRepo(db).List(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "menelusuri hasil kueri list_tasks: putus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountsMapsColumnsToAvailableTabsInOrder(t *testing.T) {
	db, mock := newDB(t)

	values := make([]driver.Value, len(countColumns))
	for i := range values {
		values[i] = int64(i + 1)
	}
	values[len(values)-1] = nil

	mock.ExpectQuery(query("count_tabs")).
		WithArgs("|BUDI|SITI|", "ADJLEADER", inboxsurvey.CommunicationOpen,
			inboxsurvey.CommunicationAnswered).
		WillReturnRows(sqlmock.NewRows(countColumns).AddRow(values...))

	counts, err := NewRepo(db).Counts(context.Background(), leaderIdentity)
	require.NoError(t, err)

	var available []inboxsurvey.Tab
	for _, tab := range inboxsurvey.Tabs() {
		if tab.Available() {
			available = append(available, tab)
		}
	}
	require.Len(t, counts, len(available))
	for i, count := range counts {
		require.Equal(t, available[i], count.Tab)
	}
	require.Equal(t, 1, counts[0].Total)
	require.Equal(t, 0, counts[len(counts)-1].Total, "NULL dibaca nol")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountsWrapsScanError(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("count_tabs")).WillReturnError(errors.New("ora"))
	_, err := NewRepo(db).Counts(context.Background(), leaderIdentity)
	require.ErrorContains(t, err, "membaca hasil kueri count_tabs: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}

func kpiValues(group string) []driver.Value {
	values := []driver.Value{group}
	for i := 1; i < len(kpiColumns); i++ {
		values = append(values, float64(i))
	}
	return values
}

func TestKPIOutstandingUsesCategoryAndAdjusterQuery(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("kpi_by_adjuster")).
		WithArgs("|BUDI|SITI|", "INTERIM", "2026").
		WillReturnRows(sqlmock.NewRows(kpiColumns).AddRow(kpiValues("BUDI")...))

	rows, err := NewRepo(db).KPI(context.Background(), leaderIdentity, inboxsurvey.KPIFilter{
		Kind: inboxsurvey.KPIOutstanding, Category: "INTERIM", Year: "2026",
	})
	require.NoError(t, err)
	require.Equal(t, []inboxsurvey.KPIRow{{
		Group: "BUDI", SurveyScheduling: 1, ImmediateAdvice: 2, PreliminaryAdvice: 3,
		InterimReport: 4, ProgressUpdate: 5, CommunicationResponse: 6, ProposeAdjustment: 7,
		FinalReport: 8, Value: 9,
	}}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKPIFinalAndQuarterlyForceFinalCategory(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("kpi_by_adjuster")).
		WithArgs("|BUDI|SITI|", inboxsurvey.KPITypeFinal, nil).
		WillReturnRows(sqlmock.NewRows(kpiColumns))
	mock.ExpectQuery(query("kpi_by_year")).
		WithArgs("|BUDI|SITI|", inboxsurvey.KPITypeFinal, nil).
		WillReturnRows(sqlmock.NewRows(kpiColumns).AddRow(kpiValues("2026")...))

	repo := NewRepo(db)
	rows, err := repo.KPI(context.Background(), leaderIdentity,
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIFinal, Category: "abaikan"})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NotNil(t, rows)

	rows, err = repo.KPI(context.Background(), leaderIdentity,
		inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIQuarterly})
	require.NoError(t, err)
	require.Equal(t, "2026", rows[0].Group)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKPIWrapsErrors(t *testing.T) {
	filter := inboxsurvey.KPIFilter{Kind: inboxsurvey.KPIQuarterly}

	db, mock := newDB(t)
	mock.ExpectQuery(query("kpi_by_year")).WillReturnError(errors.New("ora"))
	_, err := NewRepo(db).KPI(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "menjalankan kueri kpi_by_year: ora")

	db, mock = newDB(t)
	mock.ExpectQuery(query("kpi_by_year")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = NewRepo(db).KPI(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "membaca baris kueri kpi_by_year")

	db, mock = newDB(t)
	mock.ExpectQuery(query("kpi_by_year")).
		WillReturnRows(sqlmock.NewRows(kpiColumns).AddRow(kpiValues("x")...).
			RowError(0, errors.New("putus")))
	_, err = NewRepo(db).KPI(context.Background(), leaderIdentity, filter)
	require.ErrorContains(t, err, "menelusuri hasil kueri kpi_by_year: putus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScopeValueAndKeyword(t *testing.T) {
	require.Equal(t, "|", scopeValue(nil))
	require.Equal(t, "|A|B C|", scopeValue([]string{" a ", " ", "b c"}))
	require.Nil(t, keyword("  "))
	require.Equal(t, "x", keyword("x"))
}

func TestResolveSurveyorRejectsBlankLogin(t *testing.T) {
	db, mock := newDB(t)
	_, err := NewDirectory(db).ResolveSurveyor(context.Background(), "  ")
	require.ErrorIs(t, err, inboxsurvey.ErrCallerUnknown)
	require.NoError(t, mock.ExpectationsWereMet())
}

var surveyorColumns = []string{"LOGIN", "SURVEYOR_NAME", "LOGINLEADER", "STATUS"}

func TestResolveSurveyorMemberDoesNotLoadTeam(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("resolve_surveyor")).WithArgs("adjmember").
		WillReturnRows(sqlmock.NewRows(surveyorColumns).
			AddRow(" ADJMEMBER ", " SITI ", "ADJLEADER", "1"))

	identity, err := NewDirectory(db).ResolveSurveyor(context.Background(), " adjmember ")
	require.NoError(t, err)
	require.Equal(t, inboxsurvey.SurveyorIdentity{
		Login: "ADJMEMBER", Name: "SITI", Scope: []string{"SITI"},
	}, identity)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveSurveyorLeaderCollectsDistinctMembers(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("resolve_surveyor")).WithArgs("adjleader").
		WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(nil, "BUDI", nil, "1"))
	mock.ExpectQuery(query("resolve_members")).WithArgs("adjleader").
		WillReturnRows(sqlmock.NewRows([]string{"SURVEYOR_NAME"}).
			AddRow("Siti").AddRow("budi").AddRow(" ").AddRow("SITI").AddRow(nil).AddRow("Agus"))

	identity, err := NewDirectory(db).ResolveSurveyor(context.Background(), "adjleader")
	require.NoError(t, err)
	require.Equal(t, "adjleader", identity.Login, "login kosong jatuh ke yang diketik")
	require.True(t, identity.IsLeader)
	require.Equal(t, []string{"BUDI", "Siti", "Agus"}, identity.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveSurveyorWithoutMembersIsNotLeader(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("resolve_surveyor")).
		WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow("SOLO", "SOLO", "", "1"))
	mock.ExpectQuery(query("resolve_members")).
		WillReturnRows(sqlmock.NewRows([]string{"SURVEYOR_NAME"}))

	identity, err := NewDirectory(db).ResolveSurveyor(context.Background(), "solo")
	require.NoError(t, err)
	require.False(t, identity.IsLeader)
	require.Equal(t, []string{"SOLO"}, identity.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveSurveyorNotRegisteredOrNameless(t *testing.T) {
	db, mock := newDB(t)
	mock.ExpectQuery(query("resolve_surveyor")).WillReturnError(sql.ErrNoRows)
	_, err := NewDirectory(db).ResolveSurveyor(context.Background(), "x")
	require.ErrorIs(t, err, inboxsurvey.ErrNotSurveyor)

	mock.ExpectQuery(query("resolve_surveyor")).
		WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow("X", "  ", nil, "1"))
	_, err = NewDirectory(db).ResolveSurveyor(context.Background(), "x")
	require.ErrorIs(t, err, inboxsurvey.ErrNotSurveyor)

	mock.ExpectQuery(query("resolve_surveyor")).WillReturnError(errors.New("ora"))
	_, err = NewDirectory(db).ResolveSurveyor(context.Background(), "x")
	require.ErrorContains(t, err, "menjalankan kueri resolve_surveyor: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveMembersErrors(t *testing.T) {
	leaderRow := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(query("resolve_surveyor")).
			WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow("L", "L", nil, "1"))
	}

	db, mock := newDB(t)
	leaderRow(mock)
	mock.ExpectQuery(query("resolve_members")).WillReturnError(errors.New("ora"))
	_, err := NewDirectory(db).ResolveSurveyor(context.Background(), "l")
	require.ErrorContains(t, err, "menjalankan kueri resolve_members: ora")

	db, mock = newDB(t)
	leaderRow(mock)
	mock.ExpectQuery(query("resolve_members")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("x", "y"))
	_, err = NewDirectory(db).ResolveSurveyor(context.Background(), "l")
	require.ErrorContains(t, err, "membaca baris kueri resolve_members")

	db, mock = newDB(t)
	leaderRow(mock)
	mock.ExpectQuery(query("resolve_members")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x").RowError(0, errors.New("putus")))
	_, err = NewDirectory(db).ResolveSurveyor(context.Background(), "l")
	require.ErrorContains(t, err, "menelusuri hasil kueri resolve_members: putus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func oneRow(n int) *sqlmock.Rows {
	cols := make([]string, n)
	vals := make([]driver.Value, n)
	for i := range cols {
		cols[i] = "C" + string(rune('A'+i))
		vals[i] = int64(1)
	}
	return sqlmock.NewRows(cols).AddRow(vals...)
}

func TestCheckQueries(t *testing.T) {
	db, mock := newDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(query("check_tables")).WillReturnRows(oneRow(1))
	require.NoError(t, repo.CheckTables(context.Background()))
	mock.ExpectQuery(query("check_tables")).WillReturnError(errors.New("ora"))
	require.ErrorContains(t, repo.CheckTables(context.Background()), "membaca tabel antrean survei: ora")

	mock.ExpectQuery(query("check_columns")).WillReturnRows(oneRow(13))
	require.NoError(t, repo.CheckColumns(context.Background()))
	mock.ExpectQuery(query("check_columns")).WillReturnError(errors.New("ora"))
	require.ErrorContains(t, repo.CheckColumns(context.Background()), "LOSSTYPE")

	mock.ExpectQuery(query("check_kpi")).WillReturnRows(oneRow(12))
	require.NoError(t, repo.CheckKPI(context.Background()))
	mock.ExpectQuery(query("check_kpi")).WillReturnError(errors.New("ora"))
	require.ErrorContains(t, repo.CheckKPI(context.Background()), "DETAIL_KPI_ADJUSTER")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckNewColumnsReportsPresence(t *testing.T) {
	db, mock := newDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(query("check_new_columns")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D", "E"}).AddRow(1, 0, 2, 1, 0))
	cols, err := repo.CheckNewColumns(context.Background())
	require.NoError(t, err)
	require.Equal(t, NewColumns{
		Accept: true, Reference: false, WorkStatus: true,
		AdjusterPIC: true, SurveyLocation: false,
	}, cols)
	require.False(t, cols.All())
	require.True(t, NewColumns{
		Accept: true, Reference: true, WorkStatus: true,
		AdjusterPIC: true, SurveyLocation: true,
	}.All())

	mock.ExpectQuery(query("check_new_columns")).WillReturnError(errors.New("ora"))
	_, err = repo.CheckNewColumns(context.Background())
	require.ErrorContains(t, err, "membaca katalog kolom POOLDATA.T_SURVEYORLIST: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckFilledColumns(t *testing.T) {
	db, mock := newDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(query("check_filled_columns")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D", "E", "F"}).
			AddRow(100, 0, 3, 4, 5, 6))
	filled, err := repo.CheckFilledColumns(context.Background())
	require.NoError(t, err)
	require.Equal(t, FilledColumns{
		TotalRows: 100, Accept: 0, Reference: 3, WorkStatus: 4,
		AdjusterPIC: 5, SurveyLocation: 6,
	}, filled)

	mock.ExpectQuery(query("check_filled_columns")).WillReturnError(errors.New("ora"))
	_, err = repo.CheckFilledColumns(context.Background())
	require.ErrorContains(t, err, "menghitung keterisian kolom POOLDATA.T_SURVEYORLIST: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}
