package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
)

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var (
	columns1 = []string{"ID", "NAME", "POSITION"}
	columns2 = []string{"ID", "NAME", "PARENT", "PARENTNAME", "KIND"}
	idColumn = []string{"ID"}
)

func TestRepoListAndGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(exact("progress_status_list")).WillReturnRows(sqlmock.NewRows(columns1).
		AddRow(" 01 ", " DITERIMA ", " REGISTER ").AddRow(nil, nil, nil))
	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterstatusprogres.ProgressStatus{
		{ID: "01", Name: "DITERIMA", PositionCode: "REGISTER"}, {},
	}, rows)

	mock.ExpectQuery(exact("progress_status_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(exact("progress_status_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background())
	require.Error(t, err)

	mock.ExpectQuery(exact("progress_status_list")).
		WillReturnRows(sqlmock.NewRows(columns1).AddRow("1", "a", "b").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar")

	mock.ExpectQuery(exact("progress_status_get")).WithArgs("01").
		WillReturnRows(sqlmock.NewRows(columns1).AddRow("01", "A", "SURVEY"))
	got, err := repo.Get(context.Background(), "01")
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.ProgressStatus{ID: "01", Name: "A", PositionCode: "SURVEY"}, got)

	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(sqlmock.NewRows(columns1))
	_, err = repo.Get(context.Background(), "01")
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)

	mock.ExpectQuery(exact("progress_status_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "01")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepoInsertNewPicksTheNextIDAndCommits(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	input := masterstatusprogres.Input{Name: "BARU", PositionCode: "SURVEY"}

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).
		WillReturnRows(sqlmock.NewRows(idColumn).AddRow(" 01 ").AddRow("2").AddRow("03").AddRow(nil))
	mock.ExpectExec(exact("progress_status_insert")).WithArgs("04", "BARU", "SURVEY").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	fresh, err := repo.InsertNew(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.ProgressStatus{ID: "04", Name: "BARU", PositionCode: "SURVEY"}, fresh)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepoInsertNewFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	input := masterstatusprogres.Input{Name: "BARU"}
	ids := func() *sqlmock.Rows { return sqlmock.NewRows(idColumn).AddRow("01") }

	mock.ExpectBegin().WillReturnError(errDB)
	_, err := repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "memulai transaksi")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "mengunci daftar ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "membaca ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).WillReturnRows(ids().RowError(0, errDB))
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "menelusuri daftar ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("progress_status_insert")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, `menyisipkan "02"`)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("progress_status_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "menutup transaksi sisip")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepoUpdateAndCheckTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	sp := masterstatusprogres.ProgressStatus{ID: "01", Name: "A", PositionCode: "SURVEY"}

	mock.ExpectExec(exact("progress_status_update")).WithArgs("A", "SURVEY", "01").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), sp))

	mock.ExpectExec(exact("progress_status_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), sp), masterstatusprogres.ErrNotFound)

	// Driver yang tidak melaporkan jumlah baris dianggap berhasil.
	mock.ExpectExec(exact("progress_status_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.NoError(t, repo.Update(context.Background(), sp))

	mock.ExpectExec(exact("progress_status_update")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Update(context.Background(), sp), `memperbarui "01"`)

	mock.ExpectQuery(exact("progress_status_check_table")).WillReturnRows(sqlmock.NewRows(idColumn))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("progress_status_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo2ListAndGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo2(db)

	mock.ExpectQuery(exact("progress_status2_list")).WillReturnRows(sqlmock.NewRows(columns2).
		AddRow(" 1 ", " ANAK ", " 01 ", " INDUK ", " T "))
	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterstatusprogres.ProgressStatus2{
		{ID: "1", Name: "ANAK", ParentID: "01", ParentName: "INDUK", Kind: "T"},
	}, rows)

	mock.ExpectQuery(exact("progress_status2_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(exact("progress_status2_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background())
	require.Error(t, err)

	mock.ExpectQuery(exact("progress_status2_list")).
		WillReturnRows(sqlmock.NewRows(columns2).AddRow("1", "a", "b", "c", "d").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar")

	mock.ExpectQuery(exact("progress_status2_get")).WithArgs("1").
		WillReturnRows(sqlmock.NewRows(columns2).AddRow("1", "ANAK", "01", "INDUK", nil))
	got, err := repo.Get(context.Background(), "1")
	require.NoError(t, err)
	require.Equal(t, "INDUK", got.ParentName)

	mock.ExpectQuery(exact("progress_status2_get")).WillReturnRows(sqlmock.NewRows(columns2))
	_, err = repo.Get(context.Background(), "1")
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)

	mock.ExpectQuery(exact("progress_status2_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "1")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func parentRows() *sqlmock.Rows {
	return sqlmock.NewRows(columns1).AddRow("01", "INDUK", "REGISTER")
}

func TestRepo2InsertNewReadsTheParentInTheTransaction(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo2(db)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WithArgs("01").WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).
		WillReturnRows(sqlmock.NewRows(idColumn).AddRow("1").AddRow(" 5 ").AddRow("x"))
	mock.ExpectExec(exact("progress_status2_insert")).WithArgs("6", "INDUK", "ANAK", "01").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	fresh, err := repo.InsertNew(context.Background(),
		masterstatusprogres.Input2{Name: "ANAK", ParentID: "01"})
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.ProgressStatus2{
		ID: "6", Name: "ANAK", ParentID: "01", ParentName: "INDUK",
	}, fresh)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo2InsertNewFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo2(db)
	input := masterstatusprogres.Input2{Name: "ANAK", ParentID: "01"}
	ids := func() *sqlmock.Rows { return sqlmock.NewRows(idColumn).AddRow("1") }

	mock.ExpectBegin().WillReturnError(errDB)
	_, err := repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "memulai transaksi")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(sqlmock.NewRows(columns1))
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, `membaca induk "01"`)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "mengunci daftar ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "membaca ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).WillReturnRows(ids().RowError(0, errDB))
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "menelusuri daftar ID")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("progress_status2_insert")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, `menyisipkan "2"`)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectQuery(exact("progress_status2_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("progress_status2_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.InsertNew(context.Background(), input)
	require.ErrorContains(t, err, "menutup transaksi sisip")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo2Update(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo2(db)
	input := masterstatusprogres.Input2{Name: "UBAH", ParentID: "01"}

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectExec(exact("progress_status2_update")).WithArgs("UBAH", "01", "INDUK", "3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	updated, err := repo.Update(context.Background(), "3", input)
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.ProgressStatus2{
		ID: "3", Name: "UBAH", ParentID: "01", ParentName: "INDUK",
	}, updated)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectExec(exact("progress_status2_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	_, err = repo.Update(context.Background(), "3", input)
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)

	// Jumlah baris yang tidak terbaca tidak dianggap galat; transaksi tetap ditutup.
	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectExec(exact("progress_status2_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.Update(context.Background(), "3", input)
	require.ErrorContains(t, err, "menutup transaksi ubah")

	mock.ExpectBegin().WillReturnError(errDB)
	_, err = repo.Update(context.Background(), "3", input)
	require.ErrorContains(t, err, "memulai transaksi ubah")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(sqlmock.NewRows(columns1))
	mock.ExpectRollback()
	_, err = repo.Update(context.Background(), "3", input)
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("progress_status_get")).WillReturnRows(parentRows())
	mock.ExpectExec(exact("progress_status2_update")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.Update(context.Background(), "3", input)
	require.ErrorContains(t, err, `memperbarui "3"`)

	mock.ExpectQuery(exact("progress_status2_check_table")).WillReturnRows(sqlmock.NewRows(idColumn))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("progress_status2_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryPanicsOnAnUnknownName(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
