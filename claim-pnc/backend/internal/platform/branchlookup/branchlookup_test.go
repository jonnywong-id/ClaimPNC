package branchlookup

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := New(db, "BRANCH", "m/sqlstore")
	ctx := context.Background()

	code, ok, err := r.Resolve(ctx, "  ")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, code)

	mock.ExpectQuery("BRANCH").WithArgs("budi").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(" 001 "))
	code, ok, err = r.Resolve(ctx, " budi ")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "001", code)

	mock.ExpectQuery("BRANCH").WillReturnRows(sqlmock.NewRows([]string{"c"}))
	_, ok, err = r.Resolve(ctx, "x")
	require.NoError(t, err)
	require.False(t, ok)

	mock.ExpectQuery("BRANCH").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(" "))
	_, ok, err = r.Resolve(ctx, "x")
	require.NoError(t, err)
	require.False(t, ok)

	mock.ExpectQuery("BRANCH").WillReturnError(errors.New("mati"))
	_, _, err = r.Resolve(ctx, "x")
	require.EqualError(t, err, `m/sqlstore: menerjemahkan cabang "x": mati`)

	mock.ExpectQuery("BRANCH").WillReturnError(context.DeadlineExceeded)
	_, _, err = r.Resolve(ctx, "x")
	require.ErrorContains(t, err, "tidak dijawab dalam 5s")
}

func TestCheckTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := New(db, "BRANCH", "m/sqlstore")

	mock.ExpectQuery("BRANCH").WithArgs("__periksa__").WillReturnRows(sqlmock.NewRows([]string{"c"}))
	require.NoError(t, r.CheckTable(context.Background()))
	mock.ExpectQuery("BRANCH").WillReturnError(errors.New("hilang"))
	require.ErrorContains(t, r.CheckTable(context.Background()), "tidak dapat dibaca: hilang")
}
