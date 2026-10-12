package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterreas"
)

// errBoom adalah galat tiruan dari basis data.
var errBoom = errors.New("boom")

var memberColumns = []string{"REINSURERID", "REINSURERNAME", "LOGIN", "EMAIL", "COUNTRY", "TYPE"}

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah teks kueri bernama menjadi pola regexp yang mencocokinya persis.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

func TestListWithoutKeywordTrimsAndMapsNulls(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("reas_list")).WithoutArgs().WillReturnRows(
		sqlmock.NewRows(memberColumns).
			AddRow(" RE-1 ", " Andalas Re ", " AndalasRe ", " a@x.invalid ", nil, nil))

	got, err := repo.List(context.Background(), masterreas.Filter{})
	require.NoError(t, err)
	require.Equal(t, []masterreas.Member{{
		ReinsurerID: "RE-1", ReinsurerName: "Andalas Re", Login: "AndalasRe",
		Email: "a@x.invalid",
	}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWithKeywordBindsEscapedPatternFourTimes(t *testing.T) {
	repo, mock := newMock(t)
	pattern := `%100\%%`
	mock.ExpectQuery(q("reas_list_search")).WithArgs(pattern, pattern, pattern, pattern).
		WillReturnRows(sqlmock.NewRows(memberColumns))

	got, err := repo.List(context.Background(), masterreas.Filter{Keyword: " 100% "})
	require.NoError(t, err)
	require.Nil(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("reas_list")).WillReturnError(errBoom)
		_, err := repo.List(context.Background(), masterreas.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("reas_list")).
			WillReturnRows(sqlmock.NewRows([]string{"REINSURERID"}).AddRow("RE-1"))
		_, err := repo.List(context.Background(), masterreas.Filter{})
		require.ErrorContains(t, err, "masterreas/sqlstore: membaca baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("reas_list")).WillReturnRows(
			sqlmock.NewRows(memberColumns).AddRow("A", "B", "C", "D", "E", "1").
				RowError(0, errBoom))
		_, err := repo.List(context.Background(), masterreas.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "menelusuri daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// scanNoRows meniru *sql.Row yang tidak menemukan baris.
type scanNoRows struct{}

func (scanNoRows) Scan(...any) error { return sql.ErrNoRows }

// sql.ErrNoRows diteruskan apa adanya, tidak dibungkus.
func TestScanRowPassesNoRowsThrough(t *testing.T) {
	_, err := scanRow(scanNoRows{})
	require.Same(t, sql.ErrNoRows, err)
}

func TestCountFunctions(t *testing.T) {
	counts := []struct {
		query string
		call  func(r *Repo) (int, error)
	}{
		{"reas_count_all", func(r *Repo) (int, error) { return r.CountAll(context.Background()) }},
		{"reas_count_duplicate_key",
			func(r *Repo) (int, error) { return r.CountDuplicateKey(context.Background()) }},
		{"reas_count_shared_login",
			func(r *Repo) (int, error) { return r.CountSharedLogin(context.Background()) }},
		{"reas_count_empty_login",
			func(r *Repo) (int, error) { return r.CountEmptyLogin(context.Background()) }},
		{"reas_count_missing_email",
			func(r *Repo) (int, error) { return r.CountMissingEmail(context.Background()) }},
		{"reas_count_without_fallback",
			func(r *Repo) (int, error) { return r.CountWithoutFallback(context.Background()) }},
	}
	for _, c := range counts {
		t.Run(c.query, func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(q(c.query)).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(3))
			n, err := c.call(repo)
			require.NoError(t, err)
			require.Equal(t, 3, n)

			mock.ExpectQuery(q(c.query)).WillReturnError(errBoom)
			_, err = c.call(repo)
			require.ErrorIs(t, err, errBoom)
			require.ErrorContains(t, err, c.query)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("reas_check_table")).WillReturnRows(sqlmock.NewRows(memberColumns))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(q("reas_check_table")).WillReturnError(errBoom)
	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "memeriksa tabel member reas")
	require.NoError(t, mock.ExpectationsWereMet())
}

// splitByName mengabaikan teks sebelum penanda pertama, membuang komentar, dan melewati
// kueri yang badannya kosong.
func TestSplitByName(t *testing.T) {
	got := splitByName("SELECT 0\n-- name: one\n-- komentar\nSELECT 1\n-- name: empty\n" +
		"-- name: two\nSELECT 2\n")
	require.Equal(t, map[string]string{"one": "SELECT 1", "two": "SELECT 2"}, got)
}
