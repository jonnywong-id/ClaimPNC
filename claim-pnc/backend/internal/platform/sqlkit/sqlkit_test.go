package sqlkit

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type fakeRow struct{ err error }

func (f fakeRow) Scan(target ...any) error {
	if f.err != nil {
		return f.err
	}
	*(target[0].(*string)) = "isi"
	return nil
}

func scanText(s Scanner) (string, error) {
	var v string
	err := s.Scan(&v)
	return v, err
}

var errNotFound = errors.New("tidak ada")

func TestOneAndOneOr(t *testing.T) {
	got, err := One(fakeRow{}, scanText, errNotFound)
	require.NoError(t, err)
	require.Equal(t, "isi", got)
	_, err = One(fakeRow{err: sql.ErrNoRows}, scanText, errNotFound)
	require.ErrorIs(t, err, errNotFound)
	broken := errors.New("rusak")
	_, err = One(fakeRow{err: broken}, scanText, errNotFound)
	require.ErrorIs(t, err, broken)

	_, err = OneOr(fakeRow{err: broken}, scanText, errNotFound, func(err error) error { return errors.Join(errNotFound, err) })
	require.ErrorIs(t, err, broken)
	_, err = OneOr(fakeRow{err: sql.ErrNoRows}, scanText, errNotFound, func(err error) error { return err })
	require.ErrorIs(t, err, errNotFound)
	got, err = OneOr(fakeRow{}, scanText, errNotFound, func(err error) error { return err })
	require.NoError(t, err)
	require.Equal(t, "isi", got)
}

func TestTaken(t *testing.T) {
	taken := errors.New("dipakai")
	require.NoError(t, Taken(fakeRow{err: sql.ErrNoRows}, scanText, taken))
	require.ErrorIs(t, Taken(fakeRow{}, scanText, taken), taken)
	broken := errors.New("rusak")
	require.ErrorIs(t, Taken(fakeRow{err: broken}, scanText, taken), broken)
}

func TestCollect(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow("a").AddRow("b"))
	rows, qerr := db.Query("SELECT v")
	got, err := Collect(rows, qerr, scanText, "q", "", "i")
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, got)

	_, err = Collect[string, Scanner](nil, errors.New("mati"), scanText, "m: membaca", "", "i")
	require.EqualError(t, err, "m: membaca: mati")

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow("a").RowError(0, errors.New("putus")))
	rows, qerr = db.Query("SELECT v")
	_, err = Collect(rows, qerr, scanText, "q", "", "m: menelusuri")
	require.EqualError(t, err, "m: menelusuri: putus")

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"v", "w"}).AddRow("a", "b"))
	rows, qerr = db.Query("SELECT v")
	_, err = Collect(rows, qerr, scanText, "q", "m: membaca baris", "i")
	require.ErrorContains(t, err, "m: membaca baris: ")
}

func TestSetEach(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	n, err := SetEach(context.Background(), db, "UPDATE", "1", nil, "m")
	require.NoError(t, err)
	require.Zero(t, n)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WithArgs("1", "a").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE").WithArgs("1", "b").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	n, err = SetEach(context.Background(), db, "UPDATE", "1", []string{" a ", "b"}, "m")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	mock.ExpectBegin().WillReturnError(errors.New("x"))
	_, err = SetEach(context.Background(), db, "UPDATE", "1", []string{"a"}, "m")
	require.EqualError(t, err, "m: memulai transaksi keputusan: x")

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnError(errors.New("y"))
	mock.ExpectRollback()
	_, err = SetEach(context.Background(), db, "UPDATE", "1", []string{"a"}, "m")
	require.EqualError(t, err, `m: menetapkan status "a": y`)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("z"))
	_, err = SetEach(context.Background(), db, "UPDATE", "1", []string{"a"}, "m")
	require.EqualError(t, err, "m: menutup transaksi keputusan: z")
}
