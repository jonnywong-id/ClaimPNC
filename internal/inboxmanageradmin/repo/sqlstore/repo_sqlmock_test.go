package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanageradmin"
)

// newMock membentuk basis data tiruan; kueri dicocokkan persis dengan isi berkas .sql.
func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regex yang cocok persis.
func q(name string) string { return regexp.QuoteMeta(query(name)) }

var errBoom = errors.New("boom")

func tabQuery(orgUnit string) inboxmanageradmin.Query {
	return inboxmanageradmin.Query{Tab: inboxmanageradmin.Tab{OrgUnit: orgUnit}}
}

func TestListMapsRowsAndDerivesStatus(t *testing.T) {
	db, mock := newMock(t)
	registered := time.Date(2026, 9, 25, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery(q("list_by_org_unit")).WithArgs(inboxmanageradmin.OrgUnitPA).WillReturnRows(
		sqlmock.NewRows(resultColumns).
			AddRow("ASM-FW-GCNMFW-WORK PNC-1", "PNC-1", "00.1", "Tertanggung", "PA", "Agen",
				registered, "Admin", "New").
			AddRow(nil, "PNC-2", nil, nil, nil, nil, nil, nil, nil))

	got, err := NewRepo(db).List(context.Background(), tabQuery(inboxmanageradmin.OrgUnitPA))
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1", got[0].Reference)
	require.Equal(t, "Agen", got[0].BusinessSource)
	require.NotNil(t, got[0].RegisteredAt)
	require.True(t, registered.Equal(*got[0].RegisteredAt))
	require.Equal(t, inboxmanageradmin.DisplayStatusFor("New"), got[0].ClaimStatus)
	require.Equal(t, "PNC-2", got[1].CaseID)
	require.Nil(t, got[1].RegisteredAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEmptyIsSliceNotNil(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("list_by_org_unit")).WithArgs("X").WillReturnRows(sqlmock.NewRows(resultColumns))

	got, err := NewRepo(db).List(context.Background(), tabQuery("X"))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("missing column", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("list_by_org_unit")).WillReturnError(
			errors.New(`ORA-00904: "PXASSIGNEDORGUNIT": invalid identifier`))
		_, err := NewRepo(db).List(context.Background(), tabQuery("AdminPA"))
		require.ErrorIs(t, err, inboxmanageradmin.ErrSourceColumnMissing)
		require.Contains(t, err.Error(), "PXASSIGNEDORGUNIT")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("list_by_org_unit")).WillReturnError(errBoom)
		_, err := NewRepo(db).List(context.Background(), tabQuery("AdminPA"))
		require.ErrorIs(t, err, errBoom)
		require.NotErrorIs(t, err, inboxmanageradmin.ErrSourceColumnMissing)
		require.Contains(t, err.Error(), "menjalankan kueri antrean unit organisasi AdminPA")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("list_by_org_unit")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1"))
		_, err := NewRepo(db).List(context.Background(), tabQuery("AdminPA"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris antrean")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("list_by_org_unit")).WillReturnRows(
			sqlmock.NewRows(resultColumns).
				AddRow("r", "c", "p", "i", "b", "s", nil, "a", "New").RowError(0, errBoom))
		_, err := NewRepo(db).List(context.Background(), tabQuery("AdminPA"))
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri hasil antrean")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	t.Run("both readable, empty tables", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("check_table")).WillReturnRows(sqlmock.NewRows([]string{"N"}))
		mock.ExpectQuery(q("check_line_business")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("queue table fails", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("check_table")).WillReturnError(errBoom)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "POOLDATA.T_CLAIMLIST_ADMIN")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("line business column fails", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("check_table")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(q("check_line_business")).WillReturnError(errBoom)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "LINE_BUSINESS")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestLineBusinessFor(t *testing.T) {
	t.Run("blank login does not query", func(t *testing.T) {
		db, mock := newMock(t)
		line, err := NewRepo(db).LineBusinessFor(context.Background(), "   ")
		require.NoError(t, err)
		require.Empty(t, line)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("found is trimmed and login uppercased", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("line_business_for")).WithArgs("PETUGAS.A").WillReturnRows(
			sqlmock.NewRows([]string{"LINE_BUSINESS"}).AddRow(" PA "))
		line, err := NewRepo(db).LineBusinessFor(context.Background(), " petugas.a ")
		require.NoError(t, err)
		require.Equal(t, "PA", line)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("null column is empty", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("line_business_for")).WithArgs("A").WillReturnRows(
			sqlmock.NewRows([]string{"LINE_BUSINESS"}).AddRow(nil))
		line, err := NewRepo(db).LineBusinessFor(context.Background(), "a")
		require.NoError(t, err)
		require.Empty(t, line)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no row is empty without error", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("line_business_for")).WithArgs("A").WillReturnRows(
			sqlmock.NewRows([]string{"LINE_BUSINESS"}))
		line, err := NewRepo(db).LineBusinessFor(context.Background(), "a")
		require.NoError(t, err)
		require.Empty(t, line)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("line_business_for")).WithArgs("A").WillReturnError(errBoom)
		_, err := NewRepo(db).LineBusinessFor(context.Background(), "a")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca lini bisnis petugas A")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.Panics(t, func() { _ = query("tidak_ada") })
}

func TestSplitByNameDropsHeaderCommentsAndEmptyBodies(t *testing.T) {
	got := splitByName("-- kepala\nSELECT 0\n-- name: a\n-- catatan\nSELECT 1\n-- name: kosong\n-- name: b\nSELECT 2")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}
