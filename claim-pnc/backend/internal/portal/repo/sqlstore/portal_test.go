package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
)

const portalListSQL = "SELECT PORTAL_ID,\n       PORTAL_NAME,\n       PORTAL_ALIAS\n  FROM POOLDATA.M_PORTAL_PNC\n ORDER BY PORTAL_ID"

var errDB = errors.New("ORA-03113: koneksi terputus")

func newRepoWithMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func TestNewRepoLoadsQueryWithoutHeaderComments(t *testing.T) {
	repo, _ := newRepoWithMock(t)
	// Komentar kepala berkas dan penanda nama tidak ikut dikirim ke basis data.
	require.Equal(t, portalListSQL, repo.query)
}

func TestListTrimsAndUppercasesAlias(t *testing.T) {
	repo, mock := newRepoWithMock(t)
	mock.ExpectQuery("^" + regexp.QuoteMeta(portalListSQL) + "$").
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"PORTAL_ID", "PORTAL_NAME", "PORTAL_ALIAS"}).
			AddRow(" 202600101 ", " ASURANSI SINAR MAS ", " asm ").
			AddRow("202600102", "ASURANSI SIMAS INSURTECH", "Asi"))

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []portal.Portal{
		{ID: "202600101", Name: "ASURANSI SINAR MAS", Alias: "ASM"},
		{ID: "202600102", Name: "ASURANSI SIMAS INSURTECH", Alias: "ASI"},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEmptyTableReturnsNil(t *testing.T) {
	repo, mock := newRepoWithMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(portalListSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"PORTAL_ID", "PORTAL_NAME", "PORTAL_ALIAS"}))

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Nil(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrorBranches(t *testing.T) {
	ctx := context.Background()

	repo, mock := newRepoWithMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(portalListSQL)).WillReturnError(errDB)
	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "portal/sqlstore: membaca daftar portal")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newRepoWithMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(portalListSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"PORTAL_ID"}).AddRow("1"))
	_, err = repo.List(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "portal/sqlstore: membaca baris portal")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newRepoWithMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(portalListSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"PORTAL_ID", "PORTAL_NAME", "PORTAL_ALIAS"}).
			AddRow("1", "A", "B").RowError(0, errDB))
	_, err = repo.List(ctx)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "portal/sqlstore: menelusuri daftar portal")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentAfterMarkerWithoutMarkerIsEmpty(t *testing.T) {
	require.Equal(t, "", contentAfterMarker("-- hanya komentar\nSELECT 1"))
	require.Equal(t, "SELECT 1\n-- ikut", contentAfterMarker("-- name: a\nSELECT 1\n-- ikut\n"))
}
