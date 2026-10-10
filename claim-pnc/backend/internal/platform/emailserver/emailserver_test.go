package emailserver

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var columns = []string{"EMAIL_ACCOUNT", "HOST", "PORT", "EMAIL_ADDRESS", "PASS", "DISPLAY_NAME"}

func TestAccountReadsRowByName(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("M_EMAIL_SERVER_PNC").WithArgs("ClaimPNC").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("ClaimPNC ", " smtp.contoh.example ", " 587 ", "pengirim@contoh.example", "rahasia", "Claim PNC"))

	store := NewStore(db, "")
	require.Equal(t, DefaultAccount, store.AccountName())
	a, err := store.Account(context.Background())
	require.NoError(t, err)
	require.Equal(t, Account{
		Name: "ClaimPNC", Host: "smtp.contoh.example", Port: 587,
		Address: "pengirim@contoh.example", Password: "rahasia", DisplayName: "Claim PNC",
	}, a)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountMissingOrInvalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("M_EMAIL_SERVER_PNC").WithArgs("Lain").WillReturnRows(sqlmock.NewRows(columns))
	_, err = NewStore(db, "Lain").Account(context.Background())
	require.True(t, errors.Is(err, ErrNotFound))

	mock.ExpectQuery("M_EMAIL_SERVER_PNC").WithArgs("ClaimPNC").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("ClaimPNC", "smtp.contoh.example", "x", "a@contoh.example", "p", ""))
	_, err = NewStore(db, "ClaimPNC").Account(context.Background())
	require.ErrorContains(t, err, "PORT")

	mock.ExpectQuery("M_EMAIL_SERVER_PNC").WithArgs("ClaimPNC").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("ClaimPNC", "", "587", "a@contoh.example", "p", ""))
	_, err = NewStore(db, "ClaimPNC").Account(context.Background())
	require.ErrorContains(t, err, "HOST")
	require.NoError(t, mock.ExpectationsWereMet())
}
