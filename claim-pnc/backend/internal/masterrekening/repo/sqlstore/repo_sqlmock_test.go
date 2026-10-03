package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
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

var accountColumns = []string{
	"NO", "NAME", "BANK", "BRANCH", "ADDRESS", "BANKCODE", "TYPE", "ACTIVE", "STATUS",
	"COMMITTEE", "DECIDED", "EMAIL", "EMAILINPUT", "PHONE", "NIK", "DOC", "NOTE",
	"CREATEDBY", "CREATED", "UPDATEDBY", "SERVICE", "CASHIERID", "RESPONSE", "FLAG",
	"PREVBANK", "PREVNO", "PREVOWNER",
}

var at = time.Date(2026, time.March, 1, 3, 0, 0, 0, time.UTC)

func accountRow(rows *sqlmock.Rows) *sqlmock.Rows {
	return rows.AddRow(
		" 123 ", " PT A ", " BANK BCA ", " Thamrin ", " Jakarta ", " 014 ", " Giro ", " Ya ",
		" 1 ", " Budi ", at, " a@contoh.example ", " b@contoh.example ", " 021 ", " 317 ",
		" DOK ", " catatan ", " SITI ", at, " ANI ", " 1 ", " ID-1 ", " [0] diterima ",
		" U ", " 002 ", " 999 ", " PT Lama ",
	)
}

func wantAccount() masterrekening.Account {
	decided := at
	return masterrekening.Account{
		Number: "123", OwnerName: "PT A", BankName: "BANK BCA", BankBranch: "Thamrin",
		BankAddress: "Jakarta", BankCode: "014", AccountType: "Giro", Active: true,
		Status: masterrekening.StatusApproved, CommitteeApproval: "Budi", DecidedAt: &decided,
		Email: "a@contoh.example", SubmitterEmail: "b@contoh.example", Phone: "021",
		NIK: "317", DocumentID: "DOK", Note: "catatan", CreatedBy: "SITI", CreatedAt: at,
		UpdatedBy: "ANI", ServiceStatus: "1", CashierAccountID: "ID-1",
		CashierResponse: "diterima", ChangeFlag: "U", PreviousBankCode: "002",
		PreviousNumber: "999", PreviousOwnerName: "PT Lama",
	}
}

func TestListCountsThenReadsOnePage(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	filter := []any{"0", "0", "12", "12", nil, nil, "BCA", "BCA", "budi", "budi"}
	mock.ExpectQuery(exact("account_count")).WithArgs(toDriver(filter)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(9))
	mock.ExpectQuery(exact("account_list")).WithArgs(toDriver(append(filter, 5, 2000))...).
		WillReturnRows(accountRow(sqlmock.NewRows(accountColumns)))

	rows, total, err := repo.List(context.Background(), masterrekening.Filter{
		Status: masterrekening.StatusPending, Number: " 12 ", OwnerName: "  ", BankName: "BCA",
		MyCommitteeOnly: true, CommitteeIdentity: "budi", Limit: 9999, Offset: 5,
	})
	require.NoError(t, err)
	require.Equal(t, 9, total)
	require.Equal(t, []masterrekening.Account{wantAccount()}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func toDriver(values []any) []driver.Value {
	out := make([]driver.Value, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func TestListDefaultsTheLimitAndIgnoresCommitteeWhenNotMine(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	nils := []any{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	mock.ExpectQuery(exact("account_count")).WithArgs(toDriver(nils)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	mock.ExpectQuery(exact("account_list")).WithArgs(toDriver(append(nils, 0, 500))...).
		WillReturnRows(sqlmock.NewRows(accountColumns))

	rows, total, err := repo.List(context.Background(), masterrekening.Filter{
		CommitteeIdentity: "budi", Offset: -3,
	})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(exact("account_count")).WillReturnError(errDB)
	_, _, err := repo.List(context.Background(), masterrekening.Filter{})
	require.ErrorContains(t, err, "menghitung rekening")

	mock.ExpectQuery(exact("account_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("account_list")).WillReturnError(errDB)
	_, _, err = repo.List(context.Background(), masterrekening.Filter{})
	require.ErrorContains(t, err, "membaca daftar rekening")

	mock.ExpectQuery(exact("account_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("account_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, _, err = repo.List(context.Background(), masterrekening.Filter{})
	require.ErrorContains(t, err, "membaca baris rekening")

	mock.ExpectQuery(exact("account_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("account_list")).
		WillReturnRows(accountRow(sqlmock.NewRows(accountColumns)).RowError(0, errDB))
	_, _, err = repo.List(context.Background(), masterrekening.Filter{})
	require.ErrorContains(t, err, "menelusuri daftar rekening")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMapsNoRowsToNotFound(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	key := masterrekening.Key{Number: "123", BankCode: "014"}

	mock.ExpectQuery(exact("account_get")).WithArgs("123", "014").
		WillReturnRows(accountRow(sqlmock.NewRows(accountColumns)))
	got, err := repo.Get(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, wantAccount(), got)

	mock.ExpectQuery(exact("account_get")).WillReturnRows(sqlmock.NewRows(accountColumns))
	_, err = repo.Get(context.Background(), key)
	require.ErrorIs(t, err, masterrekening.ErrNotFound)

	mock.ExpectQuery(exact("account_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), key)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScanTreatsNullsAsEmptyAndInactive(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	values := make([]any, len(accountColumns))
	values[0] = "123"
	mock.ExpectQuery(exact("account_get")).
		WillReturnRows(sqlmock.NewRows(accountColumns).AddRow(toDriver(values)...))

	got, err := repo.Get(context.Background(), masterrekening.Key{Number: "123"})
	require.NoError(t, err)
	require.Equal(t, masterrekening.Account{Number: "123"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByNumber(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(exact("account_find_by_number")).WithArgs("123").
		WillReturnRows(accountRow(sqlmock.NewRows(accountColumns)))
	rows, err := repo.FindByNumber(context.Background(), " 123 ")
	require.NoError(t, err)
	require.Equal(t, []masterrekening.Account{wantAccount()}, rows)

	mock.ExpectQuery(exact("account_find_by_number")).WillReturnError(errDB)
	_, err = repo.FindByNumber(context.Background(), "123")
	require.ErrorContains(t, err, "mencari nomor rekening")

	mock.ExpectQuery(exact("account_find_by_number")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.FindByNumber(context.Background(), "123")
	require.ErrorContains(t, err, "membaca baris rekening")

	mock.ExpectQuery(exact("account_find_by_number")).
		WillReturnRows(accountRow(sqlmock.NewRows(accountColumns)).RowError(0, errDB))
	_, err = repo.FindByNumber(context.Background(), "123")
	require.ErrorContains(t, err, "menelusuri hasil pencarian nomor")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveBindsEveryColumnInOrder(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	acct := wantAccount()
	acct.CreatedAt = at.In(time.FixedZone("WIB", 7*3600))
	acct.Active = false

	mock.ExpectExec(exact("account_insert")).WithArgs(
		"123", "PT A", "BANK BCA", "Thamrin", "Jakarta", "014", "Giro", "Tidak", "1",
		"Budi", "a@contoh.example", "b@contoh.example", "021", "317", "DOK", "catatan",
		"SITI", at, "ANI", "1", "002", "999", "PT Lama",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Save(context.Background(), acct))

	mock.ExpectExec(exact("account_insert")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Save(context.Background(), acct), "menyisipkan rekening")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBindsTheDecisionAndChecksTheTouchedRow(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	acct := wantAccount()

	mock.ExpectExec(exact("account_update")).WithArgs(
		"PT A", "BANK BCA", "Thamrin", "Jakarta", "Giro", "Ya", "1", "Budi", at,
		"a@contoh.example", "b@contoh.example", "021", "317", "DOK", "catatan", "ANI",
		"1", "ID-1", "diterima", "002", "999", "PT Lama", "123", "014",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), acct))

	// Tanpa tanggal keputusan, bindnya NULL; tidak ada baris tersentuh berarti tidak ada.
	acct.DecidedAt = nil
	mock.ExpectExec(exact("account_update")).
		WithArgs("PT A", "BANK BCA", "Thamrin", "Jakarta", "Giro", "Ya", "1", "Budi", nil,
			"a@contoh.example", "b@contoh.example", "021", "317", "DOK", "catatan", "ANI",
			"1", "ID-1", "diterima", "002", "999", "PT Lama", "123", "014").
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), acct), masterrekening.ErrNotFound)

	// Driver yang tidak melaporkan jumlah baris dianggap berhasil.
	mock.ExpectExec(exact("account_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.NoError(t, repo.Update(context.Background(), acct))

	mock.ExpectExec(exact("account_update")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Update(context.Background(), acct), "memperbarui rekening")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClearRejected(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	key := masterrekening.Key{Number: "123", BankCode: "014"}

	mock.ExpectExec(exact("account_clear_rejected")).WithArgs("123", "014").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.ClearRejected(context.Background(), key))

	mock.ExpectExec(exact("account_clear_rejected")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.ClearRejected(context.Background(), key),
		masterrekening.ErrAlreadyDecided)

	mock.ExpectExec(exact("account_clear_rejected")).WillReturnError(errDB)
	require.ErrorContains(t, repo.ClearRejected(context.Background(), key),
		"membuang pengajuan yang ditolak")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(exact("account_check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	exists, err := repo.CheckTable(context.Background())
	require.NoError(t, err)
	require.True(t, exists)

	mock.ExpectQuery(exact("account_check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	exists, err = repo.CheckTable(context.Background())
	require.NoError(t, err)
	require.False(t, exists)

	mock.ExpectQuery(exact("account_check_table")).WillReturnError(errDB)
	_, err = repo.CheckTable(context.Background())
	require.ErrorContains(t, err, "memeriksa tabel LST_ACCOUNT")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBankListSkipsEmptyCodesAndFillsNames(t *testing.T) {
	db, mock := newMock(t)
	repo := NewBankRepo(db)
	columns := []string{"CODE", "NAME"}

	mock.ExpectQuery(exact("bank_list")).WillReturnRows(sqlmock.NewRows(columns).
		AddRow(" 014 ", " BANK BCA ").AddRow(nil, "TANPA KODE").AddRow("999", nil))
	banks, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterrekening.Bank{
		{Code: "014", Name: "BANK BCA"}, {Code: "999", Name: "999"},
	}, banks)

	mock.ExpectQuery(exact("bank_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar bank")

	mock.ExpectQuery(exact("bank_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca baris bank")

	mock.ExpectQuery(exact("bank_list")).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar bank")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryPanicsOnAnUnknownName(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
