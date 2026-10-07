package idformat

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestNext(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("SITE").WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow(" B "))
	mock.ExpectQuery("SEQ").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(7))
	mock.ExpectCommit()
	id, err := Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.NoError(t, err)
	require.Equal(t, "B0007", id)

	mock.ExpectBegin().WillReturnError(errors.New("x"))
	_, err = Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.EqualError(t, err, "m: memulai transaksi penomoran: x")

	mock.ExpectBegin()
	mock.ExpectQuery("SITE").WillReturnRows(sqlmock.NewRows([]string{"s"}))
	mock.ExpectRollback()
	_, err = Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.EqualError(t, err, "m: POOLDATA.M_SITE_DATABASE tidak punya baris CURRENT_SITE='1'")

	mock.ExpectBegin()
	mock.ExpectQuery("SITE").WillReturnError(errors.New("y"))
	mock.ExpectRollback()
	_, err = Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.EqualError(t, err, "m: membaca kode situs: y")

	mock.ExpectBegin()
	mock.ExpectQuery("SITE").WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("B"))
	mock.ExpectQuery("SEQ").WillReturnError(errors.New("z"))
	mock.ExpectRollback()
	_, err = Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.EqualError(t, err, "m: mengambil nomor urut: z")

	mock.ExpectBegin()
	mock.ExpectQuery("SITE").WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("B"))
	mock.ExpectQuery("SEQ").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	mock.ExpectCommit().WillReturnError(errors.New("c"))
	_, err = Next(ctx, db, "SITE", "SEQ", "m", 4)
	require.EqualError(t, err, "m: menutup transaksi penomoran: c")
}
