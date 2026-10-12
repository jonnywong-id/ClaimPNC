package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterautoclaim"
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

var rowColumns = []string{"INISIALID", "PENERIMA", "BANK", "REKENING", "MAKS", "PIC", "EMAIL",
	"BOLEH", "ALAMAT", "KOMITE", "STATUS", "CLIENTID", "CLIENT"}

func autoRows() *sqlmock.Rows { return sqlmock.NewRows(rowColumns) }

func sampleAutoClaim() masterautoclaim.AutoClaim {
	return masterautoclaim.AutoClaim{Initial: "BRI", ReceiverName: "PT BRI", BankName: "BRI",
		AccountNumber: "123", MaxPercent: "10", ReporterPIC: "PIC", ReporterEmail: "p@x",
		ClaimAllowed: "1", ReceiverAddress: "Jl", SubmittedBy: "S", Committee: "K",
		Status: masterautoclaim.StatusPending, ClientID: "C1", ClientName: "Client"}
}

func TestListPicksQueryByFilter(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("auto_claim_list")).WithArgs("1").WillReturnRows(autoRows().
		AddRow(" BRI ", " PT ", "BRI", "123", "10", "P", "e", "1", "A", "K", " 1 ", "C", "N"))
	got, err := repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, "BRI", got[0].Initial)
	require.Equal(t, masterautoclaim.StatusApproved, got[0].Status)

	mock.ExpectQuery(q("auto_claim_list_by_committee")).WithArgs("0", "KOMITE1").WillReturnRows(autoRows())
	got, err = repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusPending,
		CommitteeOnly: true, CommitteeID: " KOMITE1 "})
	require.NoError(t, err)
	require.Empty(t, got)

	mock.ExpectQuery(q("auto_claim_list")).WillReturnError(errDB)
	_, err = repo.List(ctx, masterautoclaim.Filter{})
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("auto_claim_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(ctx, masterautoclaim.Filter{})
	require.ErrorContains(t, err, "membaca baris daftar")

	mock.ExpectQuery(q("auto_claim_list")).WillReturnRows(autoRows().
		AddRow("a", "", "", "", "", "", "", "", "", "", "", "", "").RowError(0, errDB))
	_, err = repo.List(ctx, masterautoclaim.Filter{})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("auto_claim_get")).WithArgs("BRI").WillReturnRows(autoRows().
		AddRow("BRI", "PT", nil, nil, nil, nil, nil, nil, nil, nil, "0", nil, nil))
	got, err := repo.Get(ctx, " BRI ")
	require.NoError(t, err)
	require.Equal(t, masterautoclaim.AutoClaim{Initial: "BRI", ReceiverName: "PT", Status: masterautoclaim.StatusPending}, got)

	mock.ExpectQuery(q("auto_claim_get")).WillReturnRows(autoRows())
	_, err = repo.Get(ctx, "X")
	require.ErrorIs(t, err, masterautoclaim.ErrNotFound)

	mock.ExpectQuery(q("auto_claim_get")).WillReturnError(errDB)
	_, err = repo.Get(ctx, "X")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsert(t *testing.T) {
	ctx := context.Background()
	ac := sampleAutoClaim()
	args := []driver.Value{ac.Initial, ac.ReceiverName, ac.BankName, ac.AccountNumber, ac.MaxPercent, ac.ReporterPIC,
		ac.ReporterEmail, ac.ClaimAllowed, ac.ReceiverAddress, ac.SubmittedBy, ac.Committee, "0", ac.ClientID, ac.ClientName}

	t.Run("ok", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("auto_claim_list_initial_locked")).WithArgs("BRI").WillReturnRows(sqlmock.NewRows([]string{"X"}))
		mock.ExpectExec(q("auto_claim_insert")).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.Insert(ctx, ac))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  error
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "memulai transaksi"},
		{"check", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("auto_claim_list_initial_locked")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "memeriksa"},
		{"check rows", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("auto_claim_list_initial_locked")).WillReturnRows(
				sqlmock.NewRows([]string{"X"}).AddRow("1").RowError(0, errDB))
			m.ExpectRollback()
		}, errDB, "menelusuri pemeriksaan"},
		{"taken", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("auto_claim_list_initial_locked")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow("BRI"))
			m.ExpectRollback()
		}, masterautoclaim.ErrInitialTaken, `"BRI"`},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("auto_claim_list_initial_locked")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
			m.ExpectExec(q("auto_claim_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan"},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("auto_claim_list_initial_locked")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
			m.ExpectExec(q("auto_claim_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menutup transaksi sisip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			err := repo.Insert(ctx, ac)
			require.ErrorIs(t, err, tc.want)
			require.ErrorContains(t, err, tc.text)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	ac := sampleAutoClaim()
	ac.Initial = " BRI "
	repo, mock := newMock(t)

	mock.ExpectExec(q("auto_claim_update")).WithArgs(ac.BankName, ac.AccountNumber, ac.MaxPercent, ac.ReporterPIC,
		ac.ReporterEmail, ac.ClaimAllowed, ac.ReceiverAddress, "0", ac.SubmittedBy, ac.Committee,
		ac.ClientID, ac.ClientName, "BRI").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(ctx, ac))

	mock.ExpectExec(q("auto_claim_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(ctx, ac), masterautoclaim.ErrNotFound)

	// Jumlah baris yang tidak terbaca tidak dianggap "tidak ditemukan".
	mock.ExpectExec(q("auto_claim_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.NoError(t, repo.Update(ctx, ac))

	mock.ExpectExec(q("auto_claim_update")).WillReturnError(errDB)
	require.ErrorIs(t, repo.Update(ctx, ac), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("auto_claim_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(q("auto_claim_check_table")).WillReturnError(errDB)
	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "M_AUTO_CLAIM_PNC")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchBusinessSourcesAndClientsDropBlankIDs(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("auto_claim_business_source_search")).WithArgs("%BR\\_I%", "BR_I").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" BRI ", " Bank ").AddRow(" ", "kosong"))
	sources, err := repo.SearchBusinessSources(ctx, " b.r_i ")
	require.NoError(t, err)
	require.Equal(t, []masterautoclaim.BusinessSource{{ID: "BRI", Name: "Bank"}}, sources)

	mock.ExpectQuery(q("auto_claim_client_search")).WithArgs("%%", "").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("C1", " Client ").AddRow(nil, "x"))
	clients, err := repo.SearchClients(ctx, "")
	require.NoError(t, err)
	require.Equal(t, []masterautoclaim.Client{{ID: "C1", Name: "Client"}}, clients)

	for _, name := range []string{"auto_claim_business_source_search", "auto_claim_client_search"} {
		call := func() error {
			if name == "auto_claim_client_search" {
				_, err := repo.SearchClients(ctx, "x")
				return err
			}
			_, err := repo.SearchBusinessSources(ctx, "x")
			return err
		}
		mock.ExpectQuery(q(name)).WillReturnError(errDB)
		require.ErrorIs(t, call(), errDB)
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		require.Error(t, call())
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"ID", "N"}).AddRow("1", "n").RowError(0, errDB))
		require.ErrorIs(t, call(), errDB)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBanksSkipsIncompleteRows(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("auto_claim_bank_list")).WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}).
		AddRow(" 002 ", " BRI ").AddRow("", "TANPA KODE").AddRow("009", ""))
	banks, err := repo.ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, []masterautoclaim.Bank{{Code: "002", Name: "BRI"}}, banks)

	mock.ExpectQuery(q("auto_claim_bank_list")).WillReturnError(errDB)
	_, err = repo.ListBanks(ctx)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q("auto_claim_bank_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListBanks(ctx)
	require.ErrorContains(t, err, "membaca baris bank")
	mock.ExpectQuery(q("auto_claim_bank_list")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "N"}).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.ListBanks(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindBankByName(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	_, err := repo.FindBankByName(ctx, "  ")
	require.ErrorIs(t, err, masterautoclaim.ErrBankNotFound)

	mock.ExpectQuery(q("auto_claim_bank_by_name")).WithArgs("BRI").
		WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow("002", "BRI"))
	bank, err := repo.FindBankByName(ctx, " bri ")
	require.NoError(t, err)
	require.Equal(t, masterautoclaim.Bank{Code: "002", Name: "BRI"}, bank)

	mock.ExpectQuery(q("auto_claim_bank_by_name")).WillReturnRows(sqlmock.NewRows([]string{"KODE", "NAMA"}))
	_, err = repo.FindBankByName(ctx, "x")
	require.ErrorIs(t, err, masterautoclaim.ErrBankNotFound)

	mock.ExpectQuery(q("auto_claim_bank_by_name")).WillReturnError(errDB)
	_, err = repo.FindBankByName(ctx, "x")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCommittee(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("auto_claim_committee")).WillReturnRows(sqlmock.NewRows([]string{"OP"}).AddRow(" KOMITE1 "))
	got, err := repo.Committee(ctx)
	require.NoError(t, err)
	require.Equal(t, "KOMITE1", got)

	mock.ExpectQuery(q("auto_claim_committee")).WillReturnRows(sqlmock.NewRows([]string{"OP"}))
	got, err = repo.Committee(ctx)
	require.NoError(t, err)
	require.Empty(t, got)

	mock.ExpectQuery(q("auto_claim_committee")).WillReturnError(errDB)
	_, err = repo.Committee(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}
