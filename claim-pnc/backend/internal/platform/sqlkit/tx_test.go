package sqlkit

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestInTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectCommit()
	require.NoError(t, InTx(ctx, db, "m: mulai", "m: tutup", func(*sql.Tx) error { return nil }))

	mock.ExpectBegin().WillReturnError(errors.New("x"))
	require.EqualError(t, InTx(ctx, db, "m: mulai", "m: tutup", nil), "m: mulai: x")

	mock.ExpectBegin()
	mock.ExpectRollback()
	require.EqualError(t, InTx(ctx, db, "m: mulai", "m: tutup", func(*sql.Tx) error { return errors.New("isi") }), "isi")

	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("z"))
	require.EqualError(t, InTx(ctx, db, "m: mulai", "m: tutup", func(*sql.Tx) error { return nil }), "m: tutup: z")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSerialAndCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	mock.ExpectQuery("NEXT").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(8))
	id, err := NextSerial(ctx, db, "NEXT", "m", "ID kategori")
	require.NoError(t, err)
	require.Equal(t, "8", id)

	mock.ExpectQuery("NEXT").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	_, err = NextSerial(ctx, db, "NEXT", "m", "ID kategori")
	require.EqualError(t, err, "m: ID kategori berikutnya tidak masuk akal: 0")

	mock.ExpectQuery("NEXT").WillReturnError(errors.New("mati"))
	_, err = NextSerial(ctx, db, "NEXT", "m", "ID kategori")
	require.EqualError(t, err, "m: menerbitkan ID kategori: mati")

	mock.ExpectQuery("COUNT").WithArgs("1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(3))
	n, err := Count(ctx, db, "COUNT", "m: hitung", "1")
	require.NoError(t, err)
	require.Equal(t, 3, n)

	mock.ExpectQuery("COUNT").WillReturnError(errors.New("y"))
	_, err = Count(ctx, db, "COUNT", "m: hitung")
	require.EqualError(t, err, "m: hitung: y")
}

func TestCheckReadableAndRequireAffected(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	mock.ExpectQuery("CHECK").WillReturnRows(sqlmock.NewRows([]string{"a"}))
	require.NoError(t, CheckReadable(ctx, db, "CHECK", "m: periksa"))
	mock.ExpectQuery("CHECK").WillReturnError(errors.New("hilang"))
	require.EqualError(t, CheckReadable(ctx, db, "CHECK", "m: periksa"), "m: periksa: hilang")

	notFound := errors.New("tidak ada")
	require.ErrorIs(t, RequireAffected(sqlmock.NewResult(0, 0), notFound), notFound)
	require.NoError(t, RequireAffected(sqlmock.NewResult(0, 1), notFound))
	require.NoError(t, RequireAffected(sqlmock.NewErrorResult(errors.New("tak didukung")), notFound))
}
