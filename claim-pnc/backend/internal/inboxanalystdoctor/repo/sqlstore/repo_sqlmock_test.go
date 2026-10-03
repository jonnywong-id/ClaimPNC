package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxanalystdoctor"
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
func q(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

var rowColumns = []string{"REF", "CASEID", "POLIS", "TERTANGGUNG", "CABANG", "ADMIN", "PIC", "CATATAN",
	"TGL", "STATUS", "OPERATOR", "TOTAL"}

func TestListMapsRowsAndTotal(t *testing.T) {
	repo, mock := newMock(t)
	wib := time.FixedZone("WIB", 7*3600)
	registered := time.Date(2026, 9, 20, 10, 0, 0, 0, wib)
	mock.ExpectQuery(q("list_tasks")).
		WithArgs(inboxanalystdoctor.TransferAnalystDoctor, "DOKTER1", inboxanalystdoctor.StatusKerjaSelesai,
			"banjir", 50, inboxanalystdoctor.MaxLimit).
		WillReturnRows(sqlmock.NewRows(rowColumns).
			AddRow("REF-1", "PNC-1", "POL", "PT", "Jakarta", "Admin", "PIC", "catatan", registered, "Pending", "DOKTER1", 42).
			AddRow("REF-2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 42))
	page, err := repo.List(context.Background(), "DOKTER1",
		inboxanalystdoctor.Filter{Search: " banjir ", Limit: 999, Offset: 50})
	require.NoError(t, err)
	require.Equal(t, 42, page.Total)
	require.Len(t, page.Tasks, 2)
	first := page.Tasks[0]
	require.Equal(t, "REF-1", first.ClaimID)
	require.Equal(t, "PNC-1", first.ClaimNumber)
	require.Equal(t, "catatan", first.TechnicalPICNote)
	require.Equal(t, "DOKTER1", first.AssignedOperator)
	require.Equal(t, time.UTC, first.RegisteredAt.Location())
	require.True(t, registered.Equal(first.RegisteredAt))
	require.True(t, page.Tasks[1].RegisteredAt.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEmptySearchSendsNullAndEmptySlice(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("list_tasks")).
		WithArgs(inboxanalystdoctor.TransferAnalystDoctor, "D", inboxanalystdoctor.StatusKerjaSelesai,
			nil, 0, inboxanalystdoctor.DefaultLimit).
		WillReturnRows(sqlmock.NewRows(rowColumns))
	page, err := repo.List(context.Background(), "D", inboxanalystdoctor.Filter{})
	require.NoError(t, err)
	require.NotNil(t, page.Tasks)
	require.Empty(t, page.Tasks)
	require.Zero(t, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("list_tasks")).WillReturnError(errDB)
	_, err := repo.List(ctx, "D", inboxanalystdoctor.Filter{})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menjalankan kueri list_tasks")

	mock.ExpectQuery(q("list_tasks")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(ctx, "D", inboxanalystdoctor.Filter{})
	require.ErrorContains(t, err, "membaca baris kueri list_tasks")

	mock.ExpectQuery(q("list_tasks")).WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow("R", "C", "P", "I", "B", "A", "T", "N", nil, "S", "O", 1).RowError(0, errDB))
	_, err = repo.List(ctx, "D", inboxanalystdoctor.Filter{})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTablesAndColumns(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("check_tables")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	require.NoError(t, repo.CheckTables(ctx))
	mock.ExpectQuery(q("check_tables")).WillReturnError(errDB)
	err := repo.CheckTables(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "tabel antrean Analyst Doctor")

	mock.ExpectQuery(q("check_columns")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow(1, 1))
	require.NoError(t, repo.CheckColumns(ctx))
	mock.ExpectQuery(q("check_columns")).WillReturnError(errDB)
	err = repo.CheckColumns(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "ISCOMPLIANCETRANSFER_1")
	require.NoError(t, mock.ExpectationsWereMet())
}
