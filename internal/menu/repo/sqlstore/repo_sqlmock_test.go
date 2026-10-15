package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/menu"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

func appExists(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(q("menu_app_exists")).WithArgs("CLAIM PNC").WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
}

var menuColumns = []string{"MENU_ID", "MENU_DESC", "PROGRAM", "LEADER", "URUT"}

func TestListMapsParentAndTrims(t *testing.T) {
	repo, mock := newMock(t)
	appExists(mock)
	mock.ExpectQuery(q("menu_list")).WithArgs("CLAIM PNC").WillReturnRows(sqlmock.NewRows(menuColumns).
		AddRow(2, " INBOX ", nil, nil, 1).
		AddRow(53, " Inbox XOL ", " Inbox_XOL_Harness ", 2, 4))
	got, err := repo.List(context.Background(), "CLAIM PNC")
	require.NoError(t, err)
	require.Equal(t, menu.Item{ID: 2, Description: "INBOX", Sequence: 1}, got[0])
	require.Equal(t, 53, got[1].ID)
	require.Equal(t, "Inbox_XOL_Harness", got[1].Program)
	require.NotNil(t, got[1].ParentID)
	require.Equal(t, 2, *got[1].ParentID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("menu_app_exists")).WillReturnError(errDB)
	_, err := repo.List(ctx, "CLAIM PNC")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("menu_app_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	_, err = repo.List(ctx, "TIDAK ADA")
	require.ErrorIs(t, err, menu.ErrAppNotFound)
	require.ErrorContains(t, err, `"TIDAK ADA"`)

	mock.ExpectQuery(q("menu_app_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1).RowError(0, errDB))
	_, err = repo.List(ctx, "CLAIM PNC")
	require.ErrorIs(t, err, errDB)

	appExists(mock)
	mock.ExpectQuery(q("menu_list")).WillReturnError(errDB)
	_, err = repo.List(ctx, "CLAIM PNC")
	require.ErrorContains(t, err, "membaca daftar menu")

	appExists(mock)
	mock.ExpectQuery(q("menu_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow(1))
	_, err = repo.List(ctx, "CLAIM PNC")
	require.ErrorContains(t, err, "membaca baris menu")

	appExists(mock)
	mock.ExpectQuery(q("menu_list")).WillReturnRows(sqlmock.NewRows(menuColumns).AddRow(1, "a", "b", nil, 1).RowError(0, errDB))
	_, err = repo.List(ctx, "CLAIM PNC")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupsOfDropsBlank(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("menu_groups_of_login")).WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"G"}).AddRow(" IT ").AddRow(" ").AddRow(nil))
	groups, err := repo.GroupsOf(ctx, "admin")
	require.NoError(t, err)
	require.Equal(t, []string{"IT"}, groups)

	mock.ExpectQuery(q("menu_groups_of_login")).WillReturnError(errDB)
	_, err = repo.GroupsOf(ctx, "admin")
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q("menu_groups_of_login")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow(1, 2))
	_, err = repo.GroupsOf(ctx, "admin")
	require.ErrorContains(t, err, "membaca group")
	mock.ExpectQuery(q("menu_groups_of_login")).WillReturnRows(sqlmock.NewRows([]string{"G"}).AddRow("IT").RowError(0, errDB))
	_, err = repo.GroupsOf(ctx, "admin")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizedIDs(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	ids, err := repo.AuthorizedIDs(ctx, "CLAIM PNC", nil)
	require.NoError(t, err)
	require.Nil(t, ids)

	statement, _ := expandSubjects(getQuery("menu_authorized_ids"), "CLAIM PNC", []string{"it", " admin "})
	mock.ExpectQuery("^"+regexp.QuoteMeta(statement)+"$").WithArgs("CLAIM PNC", "IT", "ADMIN").
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(53).AddRow(nil).AddRow(2))
	ids, err = repo.AuthorizedIDs(ctx, "CLAIM PNC", []string{"it", " admin "})
	require.NoError(t, err)
	require.Equal(t, []int{53, 2}, ids)

	mock.ExpectQuery("^" + regexp.QuoteMeta(statement) + "$").WillReturnError(errDB)
	_, err = repo.AuthorizedIDs(ctx, "CLAIM PNC", []string{"it", "admin"})
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery("^" + regexp.QuoteMeta(statement) + "$").WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow(1, 2))
	_, err = repo.AuthorizedIDs(ctx, "CLAIM PNC", []string{"it", "admin"})
	require.ErrorContains(t, err, "MENU_ID")
	mock.ExpectQuery("^" + regexp.QuoteMeta(statement) + "$").WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(1).RowError(0, errDB))
	_, err = repo.AuthorizedIDs(ctx, "CLAIM PNC", []string{"it", "admin"})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("menu_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(q("menu_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckTable(context.Background()), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}
