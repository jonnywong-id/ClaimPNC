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

	"claim-pnc/internal/inboxadmin"
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func findTab(t *testing.T, code string) inboxadmin.Tab {
	t.Helper()
	tab, ok := inboxadmin.FindTab(code)
	require.True(t, ok)
	return tab
}

// fullRow mengisi ke-29 kolom dengan nilai yang dapat dikenali kembali.
func fullRow(at time.Time) []driver.Value {
	return []driver.Value{
		"PNC-1", "ASM-FW-GCNMFW-WORK PNC-1", "16.001", "Tertanggung",
		"Aneka", "Broker", "Jakarta", "Jakarta Klaim", "ADMINKLAIM",
		at, at.AddDate(0, 0, 1), at.AddDate(0, 0, 2), at.AddDate(0, 0, 3),
		"catatan", "On Progress", "Register", "Sudah Upload",
		at.AddDate(0, 0, 4), "Cabang Polis", "Cabang Survei", "PICTEKNIS",
		"Surveyor A", "SRV-1",
		at.AddDate(0, 0, 5), "deskripsi analis", "RCL", at.AddDate(0, 0, 6),
		"10 hari", "Kadaluarsa",
	}
}

func nullRow() []driver.Value {
	values := make([]driver.Value, len(resultColumns))
	values[0] = "PNC-2"
	return values
}

func TestListMapsEveryColumnAndSendsTabArguments(t *testing.T) {
	repo, mock := newMock(t)
	at := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows(resultColumns).AddRow(fullRow(at)...).AddRow(nullRow()...)
	mock.ExpectQuery(regexp.QuoteMeta(query("list_all"))).
		WithArgs("PNC", "NONMBU").
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), inboxadmin.Query{
		Tab:      findTab(t, inboxadmin.TabAll),
		Business: inboxadmin.BusinessNonMBU,
		Keyword:  "PNC",
	})
	require.NoError(t, err)
	require.Len(t, items, 2)

	first := items[0]
	require.Equal(t, "PNC-1", first.CaseID)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1", first.Reference)
	require.Equal(t, "16.001", first.PolicyNumber)
	require.Equal(t, "Tertanggung", first.InsuredName)
	require.Equal(t, "Aneka", first.BusinessName)
	require.Equal(t, "Broker", first.BusinessSource)
	require.Equal(t, "Jakarta", first.BranchName)
	require.Equal(t, "Jakarta Klaim", first.ClaimBranch)
	require.Equal(t, "ADMINKLAIM", first.Creator)
	require.Equal(t, at, *first.LossDate)
	require.Equal(t, at.AddDate(0, 0, 1), *first.ReportDate)
	require.Equal(t, at.AddDate(0, 0, 2), *first.InputDate)
	require.Equal(t, at.AddDate(0, 0, 3), *first.LODDate)
	require.Equal(t, "catatan", first.Note)
	require.Equal(t, "On Progress", first.ClaimPosition)
	require.Equal(t, "Register", first.ClaimStatus)
	require.Equal(t, "Sudah Upload", first.LODStatus)
	require.Equal(t, at.AddDate(0, 0, 4), *first.RequestDate)
	require.Equal(t, "Cabang Polis", first.PolicyBranch)
	require.Equal(t, "Cabang Survei", first.SurveyBranch)
	require.Equal(t, "PICTEKNIS", first.TechnicalPIC)
	require.Equal(t, "Surveyor A", first.Surveyor)
	require.Equal(t, "SRV-1", first.SurveyNumber)
	require.Equal(t, at.AddDate(0, 0, 5), *first.InboxDate)
	require.Equal(t, "deskripsi analis", first.AnalystNote)
	require.Equal(t, "RCL", first.RCLPUCLStatus)
	require.Equal(t, at.AddDate(0, 0, 6), *first.LetterPrintDate)
	require.Equal(t, "10 hari", first.ClaimAge)
	require.Equal(t, "Kadaluarsa", first.ExpiryStatus)

	// Kolom NULL menjadi teks kosong dan tanggal nil — bukan tanggal tahun 1.
	second := items[1]
	require.Equal(t, "PNC-2", second.CaseID)
	require.Empty(t, second.PolicyNumber)
	require.Nil(t, second.LossDate)
	require.Nil(t, second.InputDate)
	require.Nil(t, second.LetterPrintDate)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSendsPerTabArguments(t *testing.T) {
	caller := inboxadmin.Caller{Login: "ADMINKLAIM"}

	cases := []struct {
		tab  string
		name string
		args []driver.Value
	}{
		{inboxadmin.TabUnregisteredRCV, "list_unregistered", []driver.Value{nil, "ALL", "NORMAL"}},
		{inboxadmin.TabRCVOnline, "list_unregistered", []driver.Value{nil, "ALL", "ONLINE"}},
		{inboxadmin.TabRequestSurvey, "list_request_survey", []driver.Value{nil, "ADMINKLAIM"}},
		{inboxadmin.TabRequestDocument, "list_request_document", []driver.Value{"ADMINKLAIM"}},
		{inboxadmin.TabAllCaseAdmin, "list_all_case_admin", []driver.Value{nil, "ADMINKLAIM"}},
		{inboxadmin.TabBranchClaim, "list_branch_claim", []driver.Value{nil, "ALL"}},
	}

	for _, c := range cases {
		repo, mock := newMock(t)
		mock.ExpectQuery(regexp.QuoteMeta(query(c.name))).
			WithArgs(c.args...).
			WillReturnRows(sqlmock.NewRows(resultColumns))

		items, err := repo.List(context.Background(), inboxadmin.Query{
			Tab: findTab(t, c.tab), Business: inboxadmin.BusinessAll, Caller: caller,
		})
		require.NoErrorf(t, err, "tab %s", c.tab)
		require.NotNil(t, items)
		require.Empty(t, items)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestListRCLPUCLSendsNoArguments(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(query("list_rcl_pucl"))).
		WithoutArgs().
		WillReturnRows(sqlmock.NewRows(resultColumns))

	_, err := repo.List(context.Background(), inboxadmin.Query{Tab: findTab(t, inboxadmin.TabRCLPUCL)})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUnknownTabFailsWithoutTouchingTheDatabase(t *testing.T) {
	repo, mock := newMock(t)

	_, err := repo.List(context.Background(), inboxadmin.Query{Tab: inboxadmin.Tab{Code: "99"}})
	require.ErrorContains(t, err, `"99"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsQueryError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("ORA-00942")
	mock.ExpectQuery(regexp.QuoteMeta(query("list_rcl_pucl"))).WillReturnError(boom)

	_, err := repo.List(context.Background(), inboxadmin.Query{Tab: findTab(t, inboxadmin.TabRCLPUCL)})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "list_rcl_pucl")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsScanError(t *testing.T) {
	repo, mock := newMock(t)
	values := nullRow()
	values[9] = "bukan-tanggal"
	mock.ExpectQuery(regexp.QuoteMeta(query("list_rcl_pucl"))).
		WillReturnRows(sqlmock.NewRows(resultColumns).AddRow(values...))

	_, err := repo.List(context.Background(), inboxadmin.Query{Tab: findTab(t, inboxadmin.TabRCLPUCL)})
	require.ErrorContains(t, err, "membaca baris kueri list_rcl_pucl")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsRowsError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("koneksi putus")
	mock.ExpectQuery(regexp.QuoteMeta(query("list_rcl_pucl"))).
		WillReturnRows(sqlmock.NewRows(resultColumns).AddRow(nullRow()...).RowError(0, boom))

	_, err := repo.List(context.Background(), inboxadmin.Query{Tab: findTab(t, inboxadmin.TabRCLPUCL)})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "menelusuri hasil kueri")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableReadsTheCoreTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(query("check_table"))).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))

	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableWrapsError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("ORA-01031")
	mock.ExpectQuery(regexp.QuoteMeta(query("check_table"))).WillReturnError(boom)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
	require.NoError(t, mock.ExpectationsWereMet())
}
