package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// pattern mengubah teks kueri bernama menjadi pola regexp literal.
func pattern(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

var errDatabase = errors.New("db rusak")

func TestListMapsRowsAndTotal(t *testing.T) {
	repo, mock := newMockRepo(t)
	q := sampleQuery(t, inboxclaimtreatyprop.TabWorkList, false)
	page := inboxclaimtreatyprop.Pagination{Page: 2, Size: 10}

	mock.ExpectQuery(pattern("list_worklist")).
		WithArgs(inboxclaimtreatyprop.WorkClass, 10, 10).
		WillReturnRows(sqlmock.NewRows(resultColumns).
			AddRow("WK1", "REF1", "CLMP-1", "OP1", "M1", "POL1", "2026-01-01",
				"BIS", "SRC", "CED", "INS", nil, "OP2", "Pending", 23).
			AddRow(nil, nil, "CLMP-2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 23))

	result, err := repo.List(context.Background(), q, page)
	require.NoError(t, err)
	require.Equal(t, 23, result.Total)
	require.Equal(t, page.Normalize(), result.Pagination)
	require.Len(t, result.Items, 2)
	require.Equal(t, inboxclaimtreatyprop.WorkItem{
		WorkKey: "WK1", Reference: "REF1", ClaimID: "CLMP-1", AssignedOperator: "OP1",
		MasterID: "M1", PolicyNumber: "POL1", LossDate: "2026-01-01", BusinessName: "BIS",
		BusinessSource: "SRC", CedingCompany: "CED", InsuredName: "INS",
		LastUpdateOperator: "OP2", ClaimStatus: "Pending",
	}, result.Items[0])
	require.Equal(t, inboxclaimtreatyprop.WorkItem{ClaimID: "CLMP-2"}, result.Items[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEmptyResultKeepsEmptySlice(t *testing.T) {
	repo, mock := newMockRepo(t)
	q := sampleQuery(t, inboxclaimtreatyprop.TabTechnical, false)

	mock.ExpectQuery(pattern("list_workbasket")).WillReturnRows(sqlmock.NewRows(resultColumns))
	result, err := repo.List(context.Background(), q, samplePage())
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	require.Empty(t, result.Items)
	require.Equal(t, 0, result.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailures(t *testing.T) {
	ctx := context.Background()
	q := sampleQuery(t, inboxclaimtreatyprop.TabWorkList, false)

	t.Run("tab without query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		_, err := repo.List(ctx, inboxclaimtreatyprop.Query{Tab: inboxclaimtreatyprop.Tab{Code: "9"}}, samplePage())
		require.ErrorContains(t, err, `tab "9" belum punya kueri`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(pattern("list_worklist")).WillReturnError(errDatabase)
		_, err := repo.List(ctx, q, samplePage())
		require.ErrorIs(t, err, errDatabase)
		require.ErrorContains(t, err, "menjalankan kueri list_worklist")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(pattern("list_worklist")).WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("x"))
		_, err := repo.List(ctx, q, samplePage())
		require.ErrorContains(t, err, "membaca baris kueri list_worklist")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(pattern("list_worklist")).WillReturnRows(sqlmock.NewRows(resultColumns).
			AddRow("WK", "R", "C", "O", "M", "P", "L", "B", "S", "C", "I", "S", "O", "S", 1).
			RowError(0, errDatabase))
		_, err := repo.List(ctx, q, samplePage())
		require.ErrorIs(t, err, errDatabase)
		require.ErrorContains(t, err, "menelusuri hasil kueri list_worklist")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(pattern("check_worklist")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))
	mock.ExpectQuery(pattern("check_workbasket")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))
	require.NoError(t, repo.CheckTable(ctx))

	mock.ExpectQuery(pattern("check_worklist")).WillReturnError(errDatabase)
	err := repo.CheckTable(ctx)
	require.ErrorIs(t, err, errDatabase)
	require.ErrorContains(t, err, "PC_ASSIGN_WORKLIST")

	mock.ExpectQuery(pattern("check_worklist")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(1))
	mock.ExpectQuery(pattern("check_workbasket")).WillReturnError(errDatabase)
	err = repo.CheckTable(ctx)
	require.ErrorContains(t, err, "PC_ASSIGN_WORKBASKET")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSplitByNameIgnoresHeaderAndEmptyStatements(t *testing.T) {
	got := splitByName("-- kepala\n-- name: a\n-- penjelas\nselect 1\n-- name: kosong\n\n-- name: b\nselect 2")
	require.Equal(t, map[string]string{"a": "select 1", "b": "select 2"}, got)
}
