package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasal"
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

var clauseColumns = []string{"ID", "NUMBER", "PAYLOAD"}

const payload = `{"DESCRIPTION":" teks ","OLD_D_COL_ID":" ket ","pyCountry":"2",` +
	`"BISNISID":[{"ID":" 2001 ","Note":" lama "},{"ID":"","Note":"tanpa kode"},` +
	`{"ID":"2002","Note":"Travel"},{"ID":"2001","Note":"ulang"}]}`

func TestListDropsBusinessAndMapsRows(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("clause_list")).WillReturnRows(sqlmock.NewRows(clauseColumns).
		AddRow(" 1 ", " PSL-1 ", payload).AddRow("2", "PSL-2", nil))

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Clause{
		{ID: "1", Number: "PSL-1", Text: "teks", Description: "ket", Category: "2",
			CategoryLabel: "Pengecualian"},
		{ID: "2", Number: "PSL-2", CategoryLabel: "Notifikasi"},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailures(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("clause_list")).WillReturnError(errDB)
	_, err := repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(exact("clause_list")).
		WillReturnRows(sqlmock.NewRows(clauseColumns).AddRow("1", "x", "{bukan json"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "tidak dapat diurai")

	mock.ExpectQuery(exact("clause_list")).
		WillReturnRows(sqlmock.NewRows(clauseColumns).AddRow("1", "x", nil).RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetResolvesEachBusinessCodeOnce(t *testing.T) {
	repo, mock := newMock(t)
	business := []string{"ID", "NOTE"}

	mock.ExpectQuery(exact("clause_get")).WithArgs("1").
		WillReturnRows(sqlmock.NewRows(clauseColumns).AddRow("1", "PSL-1", payload))
	mock.ExpectQuery(exact("business_get")).WithArgs("2001").
		WillReturnRows(sqlmock.NewRows(business).AddRow("2001", " Personal Accident "))
	mock.ExpectQuery(exact("business_get")).WithArgs("2002").
		WillReturnRows(sqlmock.NewRows(business))

	clause, err := repo.Get(context.Background(), " 1 ")
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Business{
		{ID: "2001", Name: "Personal Accident"},
		{ID: "", Name: "tanpa kode"},
		{ID: "2002", Name: "Travel"},
		{ID: "2001", Name: "Personal Accident"},
	}, clause.Business, "kode tak dikenal mempertahankan nama tersimpan; kode berulang dibaca sekali")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFailures(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("clause_get")).WillReturnRows(sqlmock.NewRows(clauseColumns))
	_, err := repo.Get(context.Background(), "1")
	require.ErrorIs(t, err, masterpasal.ErrNotFound)

	mock.ExpectQuery(exact("clause_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "1")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("clause_get")).
		WillReturnRows(sqlmock.NewRows(clauseColumns).AddRow("1", "PSL-1", payload))
	mock.ExpectQuery(exact("business_get")).WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "1")
	require.ErrorContains(t, err, `membaca lini bisnis "2001"`)

	// Tanpa lini bisnis tidak ada kueri tambahan.
	mock.ExpectQuery(exact("clause_get")).
		WillReturnRows(sqlmock.NewRows(clauseColumns).AddRow("3", "PSL-3", `{}`))
	clause, err := repo.Get(context.Background(), "3")
	require.NoError(t, err)
	require.Nil(t, clause.Business)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertTakesTheNextFreeIDInsideATransaction(t *testing.T) {
	repo, mock := newMock(t)
	in := masterpasal.Input{
		Number: "PSL-9", Text: "teks", Category: "1",
		Business: []masterpasal.Business{{ID: "2001", Name: "PA"}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1").AddRow("7").AddRow(nil))
	mock.ExpectExec(exact("clause_insert")).
		WithArgs("8", "PSL-9", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	created, err := repo.Insert(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "8", created.ID)
	require.Equal(t, "Jaminan Polis", created.CategoryLabel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailuresRollBack(t *testing.T) {
	repo, mock := newMock(t)
	in := masterpasal.Input{Number: "PSL-9"}
	ids := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"ID"}).AddRow("1") }

	mock.ExpectBegin().WillReturnError(errDB)
	_, err := repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, "memulai transaksi")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, "mengunci daftar")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, "membaca ID terpakai")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).WillReturnRows(ids().RowError(0, errDB))
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, "menelusuri ID terpakai")

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("clause_insert")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, `menyisipkan "2"`)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("clause_list_id_locked")).WillReturnRows(ids())
	mock.ExpectExec(exact("clause_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.Insert(context.Background(), in)
	require.ErrorContains(t, err, "menutup transaksi sisip")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAndDeleteCheckTheTouchedRow(t *testing.T) {
	repo, mock := newMock(t)
	in := masterpasal.Input{Number: "PSL-1"}

	mock.ExpectExec(exact("clause_update")).WithArgs("PSL-1", sqlmock.AnyArg(), "1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	saved, err := repo.Update(context.Background(), " 1 ", in)
	require.NoError(t, err)
	require.Equal(t, "1", saved.ID)

	mock.ExpectExec(exact("clause_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(context.Background(), "1", in)
	require.ErrorIs(t, err, masterpasal.ErrNotFound)

	mock.ExpectExec(exact("clause_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = repo.Update(context.Background(), "1", in)
	require.ErrorContains(t, err, "membaca hasil pembaruan")

	mock.ExpectExec(exact("clause_update")).WillReturnError(errDB)
	_, err = repo.Update(context.Background(), "1", in)
	require.ErrorContains(t, err, "memperbarui")

	mock.ExpectExec(exact("clause_delete")).WithArgs("1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Delete(context.Background(), " 1 "))

	mock.ExpectExec(exact("clause_delete")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Delete(context.Background(), "1"), masterpasal.ErrNotFound)

	mock.ExpectExec(exact("clause_delete")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.ErrorContains(t, repo.Delete(context.Background(), "1"), "membaca hasil penghapusan")

	mock.ExpectExec(exact("clause_delete")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Delete(context.Background(), "1"), "menghapus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchBusinessBindsAnEscapedPattern(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"ID", "NOTE"}

	mock.ExpectQuery(exact("business_search")).WithArgs(`%TRA\_V%`, "TRA_V").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(" 2002 ", " Travel ").AddRow(nil, "x"))
	rows, err := repo.SearchBusiness(context.Background(), " tra_v ")
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Business{{ID: "2002", Name: "Travel"}}, rows)

	mock.ExpectQuery(exact("business_search")).WillReturnError(errDB)
	_, err = repo.SearchBusiness(context.Background(), "x")
	require.ErrorContains(t, err, "mencari lini bisnis")

	mock.ExpectQuery(exact("business_search")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.SearchBusiness(context.Background(), "x")
	require.ErrorContains(t, err, "membaca lini bisnis")

	mock.ExpectQuery(exact("business_search")).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.SearchBusiness(context.Background(), "x")
	require.ErrorContains(t, err, "menelusuri lini bisnis")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTables(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("clause_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("clause_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckTable(context.Background()), errDB)

	mock.ExpectQuery(exact("business_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckBusinessTable(context.Background()))
	mock.ExpectQuery(exact("business_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckBusinessTable(context.Background()), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryPanicsOnAnUnknownName(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
