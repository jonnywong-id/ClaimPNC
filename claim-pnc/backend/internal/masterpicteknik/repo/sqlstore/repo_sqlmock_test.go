package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
)

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var columns = []string{
	"ID", "NAME", "EMAIL", "LINE", "GROUP", "SUPERVISOR", "QUOTA", "EXTQUOTA", "X", "ACTIVE",
}

func TestListMapsViewRows(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("technician_list")).WithArgs(masterpicteknik.ActiveCode).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(" BUDI ", " Budi ", " b@contoh.co.id ", "NONMBU", "TEKNIK", "ANI", 10, 2, 4, " 1 ").
			AddRow("ANI", nil, nil, nil, nil, nil, nil, nil, nil, nil))

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpicteknik.Technician{
		{OperatorID: "BUDI", Name: "Budi", Email: "b@contoh.co.id", BusinessLine: "NONMBU",
			Group: "TEKNIK", Supervisor: "ANI", Quota: 10, ExternalQuota: 2, Workload: 4, Active: true},
		{OperatorID: "ANI"},
	}, rows)

	mock.ExpectQuery(exact("technician_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar PIC teknik")

	mock.ExpectQuery(exact("technician_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca baris PIC teknik")

	mock.ExpectQuery(exact("technician_list")).WillReturnRows(sqlmock.NewRows(columns).
		AddRow("A", nil, nil, nil, nil, nil, nil, nil, nil, "1").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar PIC teknik")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMapsTableRows(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("technician_get")).WithArgs("BUDI").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("BUDI", "Budi", nil, nil, nil, nil, 5, 0, " 006 ", "0"))
	got, err := repo.Get(context.Background(), "BUDI")
	require.NoError(t, err)
	require.Equal(t, masterpicteknik.Technician{
		OperatorID: "BUDI", Name: "Budi", Quota: 5, PanelGroup: "006", Active: false,
	}, got)

	mock.ExpectQuery(exact("technician_get")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.Get(context.Background(), "BUDI")
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)

	mock.ExpectQuery(exact("technician_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "BUDI")
	require.ErrorContains(t, err, "membaca PIC teknik")
	require.NoError(t, mock.ExpectationsWereMet())
}

func sample() masterpicteknik.Technician {
	return masterpicteknik.Technician{
		OperatorID: "BUDI", Name: "Budi", Email: "", BusinessLine: "NONMBU", Group: "TEKNIK",
		Supervisor: "ANI", Quota: 10, ExternalQuota: 2, Active: true,
	}
}

func TestInsertChecksForAnExistingOperatorInsideATransaction(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WithArgs("BUDI").WillReturnRows(sqlmock.NewRows(columns))
	mock.ExpectExec(exact("technician_insert")).
		WithArgs("BUDI", 10, "NONMBU", nil, masterpicteknik.ActiveCode, "TEKNIK", "ANI", 2, "Budi").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	created, err := repo.Insert(context.Background(), sample())
	require.NoError(t, err)
	require.Equal(t, sample(), created)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	repo, mock := newMock(t)
	none := func() *sqlmock.Rows { return sqlmock.NewRows(columns) }

	mock.ExpectBegin().WillReturnError(errDB)
	_, err := repo.Insert(context.Background(), sample())
	require.ErrorContains(t, err, "memulai transaksi")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WillReturnRows(sqlmock.NewRows(columns).
		AddRow("BUDI", nil, nil, nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), sample())
	require.ErrorIs(t, err, masterpicteknik.ErrAlreadyExists)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), sample())
	require.ErrorContains(t, err, "memeriksa ID operator")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WillReturnRows(none())
	mock.ExpectExec(exact("technician_insert")).
		WillReturnError(errors.New("ORA-00001: unique constraint violated"))
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), sample())
	require.ErrorIs(t, err, masterpicteknik.ErrAlreadyExists)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WillReturnRows(none())
	mock.ExpectExec(exact("technician_insert")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), sample())
	require.ErrorContains(t, err, "menambah PIC teknik")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("technician_get")).WillReturnRows(none())
	mock.ExpectExec(exact("technician_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.Insert(context.Background(), sample())
	require.ErrorContains(t, err, "menyimpan PIC teknik")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateChecksTheTouchedRow(t *testing.T) {
	repo, mock := newMock(t)
	inactive := sample()
	inactive.Active = false

	mock.ExpectExec(exact("technician_update")).
		WithArgs("Budi", nil, "NONMBU", "TEKNIK", "ANI", 10, 2, masterpicteknik.InactiveCode, "BUDI").
		WillReturnResult(sqlmock.NewResult(0, 1))
	updated, err := repo.Update(context.Background(), inactive)
	require.NoError(t, err)
	require.Equal(t, inactive, updated)

	mock.ExpectExec(exact("technician_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(context.Background(), sample())
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)

	// Driver yang tidak melaporkan jumlah baris dianggap berhasil.
	mock.ExpectExec(exact("technician_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = repo.Update(context.Background(), sample())
	require.NoError(t, err)

	mock.ExpectExec(exact("technician_update")).WillReturnError(errDB)
	_, err = repo.Update(context.Background(), sample())
	require.ErrorContains(t, err, "mengubah PIC teknik")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableAndView(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("technician_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("technician_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "MST_USER_TEKNIK")

	mock.ExpectQuery(exact("technician_check_view")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckView(context.Background()))
	mock.ExpectQuery(exact("technician_check_view")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckView(context.Background()), "V_MST_USER_TEKNIS")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryPanicsOnAnUnknownName(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
