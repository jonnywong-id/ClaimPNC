package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

var (
	taskColumns = []string{
		"REFERENCE", "CASE_NUMBER", "POLICY_NUMBER", "INSURED", "PARTICIPANT",
		"BUSINESS", "BRANCH", "ADMIN", "REGISTERED_AT", "SURVEY_DATE", "BUSINESS_LINE",
	}
	errOracle = errors.New("oracle menolak")
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func sqlNamed(name string) string { return regexp.QuoteMeta(getQuery(name)) }

func basket() string { return strings.ToUpper(inboxinvestigator.Workbasket) }

// Tanpa kata kunci, kueri daftar dipakai dengan workbasket dan batas MaxRows+1.
func TestListWithoutKeywordMapsEveryColumn(t *testing.T) {
	repo, mock := newMock(t)
	registered := time.Date(2026, 9, 21, 2, 15, 0, 0, time.UTC)
	survey := time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)

	// SURVEY_DATE datang sebagai TEKS, bukan tanggal: ia hasil `JSON_VALUE` atas dokumen
	// klaim, dan bentuknya persis seperti yang disimpan Pega di dalam JSON.
	mock.ExpectQuery(sqlNamed("investigator_inbox_list")).
		WithArgs(basket(), inboxinvestigator.MaxRows+1).
		WillReturnRows(sqlmock.NewRows(taskColumns).
			AddRow(" REF-1 ", "PNC-1 ", "POL-1", "PT Satu", "Peserta", "PA", "Pusat",
				"ADMIN1  ", registered, "20260922T010000.000 GMT", " 002 ").
			AddRow("REF-2", "PNC-2", nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := repo.List(context.Background(), inboxinvestigator.Filter{Keyword: "  "})
	require.NoError(t, err)
	require.False(t, page.Truncated)
	require.Equal(t, []inboxinvestigator.Task{
		{
			Reference: "REF-1", CaseNumber: "PNC-1", PolicyNumber: "POL-1",
			InsuredName: "PT Satu", ParticipantName: "Peserta", BusinessName: "PA",
			BranchName: "Pusat", AdminName: "ADMIN1", RegisteredAt: &registered,
			SurveyDate: &survey, BusinessLine: "002",
		},
		{Reference: "REF-2", CaseNumber: "PNC-2"},
	}, page.Tasks)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci dikirim tujuh kali sebagai pola LIKE yang sudah diloloskan.
func TestListWithKeywordBindsTheEscapedPatternSevenTimes(t *testing.T) {
	repo, mock := newMock(t)
	pattern := `%A\%B\_C\\D%`

	mock.ExpectQuery(sqlNamed("investigator_inbox_search")).
		WithArgs(basket(), pattern, pattern, pattern, pattern, pattern, pattern, pattern,
			inboxinvestigator.MaxRows+1).
		WillReturnRows(sqlmock.NewRows(taskColumns))

	page, err := repo.List(context.Background(),
		inboxinvestigator.Filter{Keyword: ` a%b_c\d `})
	require.NoError(t, err)
	require.Empty(t, page.Tasks)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Baris ke-(MaxRows+1) dibuang dan menandai daftar terpotong.
func TestListTruncatesTheExtraRow(t *testing.T) {
	repo, mock := newMock(t)

	rows := sqlmock.NewRows(taskColumns)
	for i := 0; i <= inboxinvestigator.MaxRows; i++ {
		rows.AddRow("R"+strconv.Itoa(i), "PNC-"+strconv.Itoa(i), nil, nil, nil, nil, nil,
			nil, nil, nil, nil)
	}
	mock.ExpectQuery(sqlNamed("investigator_inbox_list")).WillReturnRows(rows)

	page, err := repo.List(context.Background(), inboxinvestigator.Filter{})
	require.NoError(t, err)
	require.True(t, page.Truncated)
	require.Len(t, page.Tasks, inboxinvestigator.MaxRows)
	require.Equal(t, "PNC-0", page.Tasks[0].CaseNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMock(t)
	mock.ExpectQuery(sqlNamed("investigator_inbox_list")).WillReturnError(errOracle)
	_, err := repo.List(ctx, inboxinvestigator.Filter{})
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "membaca antrean")

	mock.ExpectQuery(sqlNamed("investigator_inbox_list")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.List(ctx, inboxinvestigator.Filter{})
	require.ErrorContains(t, err, "membaca baris antrean")

	mock.ExpectQuery(sqlNamed("investigator_inbox_list")).
		WillReturnRows(sqlmock.NewRows(taskColumns).
			AddRow("R", "C", nil, nil, nil, nil, nil, nil, nil, nil, nil).RowError(0, errOracle))
	_, err = repo.List(ctx, inboxinvestigator.Filter{})
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "menelusuri antrean")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kedua pencacah memakai workbasket Investigator dan mengembalikan angkanya.
func TestCountersReadTheirTotals(t *testing.T) {
	repo, mock := newMock(t)
	ctx := context.Background()

	mock.ExpectQuery(sqlNamed("investigator_inbox_count_waiting")).WithArgs(basket()).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(42))
	waiting, err := repo.CountWaiting(ctx)
	require.NoError(t, err)
	require.Equal(t, 42, waiting)

	mock.ExpectQuery(sqlNamed("investigator_inbox_count_without_survey")).
		WithArgs(basket()).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(7))
	without, err := repo.CountWithoutSurvey(ctx)
	require.NoError(t, err)
	require.Equal(t, 7, without)

	mock.ExpectQuery(sqlNamed("investigator_inbox_count_waiting")).
		WillReturnError(sql.ErrConnDone)
	_, err = repo.CountWaiting(ctx)
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.ErrorContains(t, err, "investigator_inbox_count_waiting")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan tabel menjalankan kuerinya dan meneruskan galat pembukaan maupun penelusuran.
func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	ctx := context.Background()

	mock.ExpectQuery(sqlNamed("investigator_inbox_check_table")).
		WillReturnRows(sqlmock.NewRows(taskColumns))
	require.NoError(t, repo.CheckTable(ctx))

	mock.ExpectQuery(sqlNamed("investigator_inbox_check_table")).WillReturnError(errOracle)
	err := repo.CheckTable(ctx)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "memeriksa tabel inbox investigator")
	require.NoError(t, mock.ExpectationsWereMet())
}
