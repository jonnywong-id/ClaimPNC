package sqlstore

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	ki "claim-pnc/internal/konversiobjectitemfire"
)

func TestObjectFilterDipasangLangsungDiKueri(t *testing.T) {
	where, args := objectFilter("P1", ki.Filter{})
	require.Equal(t, "NOPOLIS = :1", where)
	require.Equal(t, []any{"P1"}, args)

	where, args = objectFilter("P1", ki.Filter{ProdKe: "39749851", IndexObject: "77"})
	require.Equal(t, "NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2 AND TO_CHAR(INDEXOBJECT) = :3", where)
	require.Equal(t, []any{"P1", "39749851", "77"}, args)
}

func TestDeleteScopes(t *testing.T) {
	objects := []ki.ObjectRow{{ProdKe: "0", IndexObject: "1"}, {ProdKe: "0", IndexObject: "2"}}
	require.Equal(t, []scope{{ProdKe: "0"}}, deleteScopes(objects, false), "tanpa filter INDEXOBJECT: per versi")
	require.Equal(t, []scope{{ProdKe: "0", IndexObject: "1"}, {ProdKe: "0", IndexObject: "2"}}, deleteScopes(objects, true))
}

func TestDeleteOldPerObjek(t *testing.T) {
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer conn.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM POOLDATA.T_PROPERTYITEMLIST WHERE NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2 AND TO_CHAR(INDEXOBJECT) = :3")).
		WithArgs("P1", "1", "77").WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectRollback()

	tx, err := conn.Begin()
	require.NoError(t, err)
	n, err := deleteOld(context.Background(), tx, map[string]ki.Column{"INDEXOBJECT": {Name: "INDEXOBJECT"}}, "P1",
		[]scope{{ProdKe: "1", IndexObject: "77"}})
	require.NoError(t, err)
	require.Equal(t, 3, n)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOldMenolakBilaKolomIndexObjectTidakAda(t *testing.T) {
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer conn.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()
	tx, err := conn.Begin()
	require.NoError(t, err)

	_, err = deleteOld(context.Background(), tx, map[string]ki.Column{}, "P1", []scope{{ProdKe: "1", IndexObject: "77"}})
	require.ErrorContains(t, err, "tidak punya kolom INDEXOBJECT")
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
