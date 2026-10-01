package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterlogin"
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

var loginColumns = []string{
	"SURVEYOR_NAME", "LOGIN", "EMAIL", "PHONE", "ADDRESS", "STATUS", "LOGINLEADER",
}

func sampleRows() *sqlmock.Rows {
	return sqlmock.NewRows(loginColumns).
		AddRow(" BUDI ", " ADJLEADER ", " budi@contoh ", " 0812 ", " Jl ", " 1 ", nil).
		AddRow("SITI", "ADJMEMBER", nil, nil, nil, "0", "ADJLEADER")
}

func TestEveryQueryNameExists(t *testing.T) {
	for _, name := range []string{
		"login_list", "login_list_search", "login_get", "login_find_leader",
		"login_lock_table", "login_find_by_key", "login_insert", "login_update",
		"login_count_all", "login_count_duplicate_key", "login_count_empty_key",
		"login_count_orphan_leader", "login_count_missing_contact", "login_check_table",
	} {
		require.NotEmpty(t, getQuery(name), name)
	}
	require.Panics(t, func() { getQuery("tidak_ada") })
}

func TestSplitByNameDropsCommentsAndEmptyStatements(t *testing.T) {
	got := splitByName("pembuka\n-- name: a\n-- komentar\nSELECT 1\n-- name: kosong\n-- hanya komentar\n-- name: b\nSELECT 2\n")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}

func TestListWithoutKeywordTrimsColumns(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(getQuery("login_list")).WillReturnRows(sampleRows())

	list, err := repo.List(context.Background(), masterlogin.Filter{Keyword: "  "})
	require.NoError(t, err)
	require.Equal(t, []masterlogin.SurveyorLogin{
		{Name: "BUDI", Login: "ADJLEADER", Email: "budi@contoh", Phone: "0812", Address: "Jl",
			LoginStatus: "1"},
		{Name: "SITI", Login: "ADJMEMBER", LoginStatus: "0", LeaderLogin: "ADJLEADER"},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWithKeywordEscapesLikePattern(t *testing.T) {
	repo, mock := newMock(t)
	pattern := `%A\%B\_C\\D%`
	mock.ExpectQuery(getQuery("login_list_search")).
		WithArgs(pattern, pattern, pattern).
		WillReturnRows(sqlmock.NewRows(loginColumns))

	list, err := repo.List(context.Background(), masterlogin.Filter{Keyword: ` a%b_c\d `})
	require.NoError(t, err)
	require.Empty(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(getQuery("login_list")).WillReturnError(errors.New("ora"))
	_, err := repo.List(context.Background(), masterlogin.Filter{})
	require.EqualError(t, err, "masterlogin/sqlstore: membaca daftar: ora")

	mock.ExpectQuery(getQuery("login_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background(), masterlogin.Filter{})
	require.ErrorContains(t, err, "masterlogin/sqlstore: membaca baris")

	mock.ExpectQuery(getQuery("login_list")).
		WillReturnRows(sampleRows().RowError(1, errors.New("putus")))
	_, err = repo.List(context.Background(), masterlogin.Filter{})
	require.EqualError(t, err, "masterlogin/sqlstore: menelusuri daftar: putus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(getQuery("login_get")).WithArgs("ADJMEMBER").
		WillReturnRows(sqlmock.NewRows(loginColumns).
			AddRow("SITI", "ADJMEMBER", nil, nil, nil, "1", "ADJLEADER"))
	found, err := repo.Get(context.Background(), " ADJMEMBER ")
	require.NoError(t, err)
	require.Equal(t, "ADJLEADER", found.LeaderLogin)

	mock.ExpectQuery(getQuery("login_get")).WillReturnError(sql.ErrNoRows)
	_, err = repo.Get(context.Background(), "x")
	require.ErrorIs(t, err, masterlogin.ErrNotFound)

	mock.ExpectQuery(getQuery("login_get")).WillReturnError(errors.New("ora"))
	_, err = repo.Get(context.Background(), "x")
	require.EqualError(t, err, "masterlogin/sqlstore: membaca baris: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindLeaderOf(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(getQuery("login_find_leader")).WithArgs("ADJMEMBER").
		WillReturnRows(sqlmock.NewRows([]string{"L"}).AddRow(" ADJLEADER "))
	leader, err := repo.FindLeaderOf(context.Background(), " ADJMEMBER ")
	require.NoError(t, err)
	require.Equal(t, "ADJLEADER", leader)

	mock.ExpectQuery(getQuery("login_find_leader")).WillReturnError(sql.ErrNoRows)
	_, err = repo.FindLeaderOf(context.Background(), "x")
	require.ErrorIs(t, err, masterlogin.ErrNotFound)

	mock.ExpectQuery(getQuery("login_find_leader")).WillReturnError(errors.New("ora"))
	_, err = repo.FindLeaderOf(context.Background(), "x")
	require.EqualError(t, err, `masterlogin/sqlstore: membaca login leader "x": ora`)
	require.NoError(t, mock.ExpectationsWereMet())
}

var newLogin = masterlogin.SurveyorLogin{
	Name: "AGUS", Login: "agus", Email: "e", Phone: "p", Address: "a", LoginStatus: "1",
	LeaderLogin: "ADJLEADER",
}

func TestInsertLocksChecksAndCommits(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(getQuery("login_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(getQuery("login_find_by_key")).WithArgs("AGUS").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(getQuery("login_insert")).
		WithArgs("AGUS", "agus", "e", "p", "a", "1", "ADJLEADER").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	saved, err := repo.Insert(context.Background(), newLogin)
	require.NoError(t, err)
	require.Equal(t, newLogin, saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertRejectsTakenLogin(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(getQuery("login_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(getQuery("login_find_by_key")).
		WillReturnRows(sqlmock.NewRows(loginColumns).AddRow("A", "AGUS", nil, nil, nil, "1", nil))
	mock.ExpectRollback()

	_, err := repo.Insert(context.Background(), newLogin)
	require.ErrorIs(t, err, masterlogin.ErrLoginTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errors.New("ora"))
		_, err := repo.Insert(context.Background(), newLogin)
		require.EqualError(t, err, "masterlogin/sqlstore: memulai transaksi: ora")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("lock", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(getQuery("login_lock_table")).WillReturnError(errors.New("ora"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newLogin)
		require.EqualError(t, err, "masterlogin/sqlstore: mengunci tabel login surveyor: ora")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("find", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(getQuery("login_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(getQuery("login_find_by_key")).WillReturnError(errors.New("ora"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newLogin)
		require.EqualError(t, err, "masterlogin/sqlstore: membaca baris: ora")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(getQuery("login_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(getQuery("login_find_by_key")).WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(getQuery("login_insert")).WillReturnError(errors.New("ora"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newLogin)
		require.EqualError(t, err, `masterlogin/sqlstore: menyisipkan "agus": ora`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(getQuery("login_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(getQuery("login_find_by_key")).WillReturnError(sql.ErrNoRows)
		mock.ExpectExec(getQuery("login_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("ora"))
		_, err := repo.Insert(context.Background(), newLogin)
		require.EqualError(t, err, "masterlogin/sqlstore: menutup transaksi sisip: ora")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdate(t *testing.T) {
	repo, mock := newMock(t)
	updated := newLogin
	updated.Login = " agus "

	mock.ExpectExec(getQuery("login_update")).
		WithArgs("AGUS", " agus ", "e", "p", "a", "1", "ADJLEADER", "agus").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), updated))

	mock.ExpectExec(getQuery("login_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), newLogin), masterlogin.ErrNotFound)

	mock.ExpectExec(getQuery("login_update")).WillReturnError(errors.New("ora"))
	require.EqualError(t, repo.Update(context.Background(), newLogin),
		`masterlogin/sqlstore: memperbarui "agus": ora`)

	// Jumlah baris yang tidak dapat dibaca tidak dianggap galat.
	mock.ExpectExec(getQuery("login_update")).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("tidak didukung")))
	require.NoError(t, repo.Update(context.Background(), newLogin))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCounts(t *testing.T) {
	repo, mock := newMock(t)
	calls := map[string]func(context.Context) (int, error){
		"login_count_all":             repo.CountAll,
		"login_count_duplicate_key":   repo.CountDuplicateKey,
		"login_count_empty_key":       repo.CountEmptyKey,
		"login_count_orphan_leader":   repo.CountOrphanLeader,
		"login_count_missing_contact": repo.CountMissingContact,
	}
	for name, call := range calls {
		mock.ExpectQuery(getQuery(name)).WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(7))
		total, err := call(context.Background())
		require.NoError(t, err, name)
		require.Equal(t, 7, total, name)

		mock.ExpectQuery(getQuery(name)).WillReturnError(errors.New("ora"))
		_, err = call(context.Background())
		require.EqualError(t, err, "masterlogin/sqlstore: "+name+": ora")
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(getQuery("login_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(getQuery("login_check_table")).WillReturnError(errors.New("ora"))
	require.EqualError(t, repo.CheckTable(context.Background()),
		"masterlogin/sqlstore: memeriksa tabel login surveyor: ora")

	require.NoError(t, mock.ExpectationsWereMet())
}
