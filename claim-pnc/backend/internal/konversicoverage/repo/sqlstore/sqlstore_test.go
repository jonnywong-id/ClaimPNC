package sqlstore

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	kc "claim-pnc/internal/konversicoverage"
	"claim-pnc/internal/konversicoverage/usecase"
)

func TestObjectFilterHanyaMenambahSyaratYangTerisi(t *testing.T) {
	where, args := objectFilter("P1", kc.Filter{})
	require.Equal(t, "NOPOLIS = :1", where)
	require.Equal(t, []any{"P1"}, args)

	where, args = objectFilter("P1", kc.Filter{IndexObject: "3"})
	require.Equal(t, "NOPOLIS = :1 AND TO_CHAR(INDEXOBJECT) = :2", where)
	require.Equal(t, []any{"P1", "3"}, args)

	where, args = objectFilter("P1", kc.Filter{ProdKe: "1", IndexObject: "3"})
	require.Equal(t, "NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2 AND TO_CHAR(INDEXOBJECT) = :3", where)
	require.Equal(t, []any{"P1", "1", "3"}, args)
}

func TestDeleteScopes(t *testing.T) {
	lb := usecase.LineBatch{Line: kc.Lines[0], Objects: []kc.ObjectRow{
		{ProdKe: "0", IndexObject: "1", ListIndex: "11"},
		{ProdKe: "0", IndexObject: "2", ListIndex: "12"},
	}}
	key := func(o kc.ObjectRow) (string, string) { return kc.CoverageObjectKey(lb.Line, o) }

	require.Equal(t, []scope{{ProdKe: "0"}}, deleteScopes(lb, false, key), "tanpa filter INDEXOBJECT: per versi")
	require.Equal(t, []scope{
		{ProdKe: "0", Column: "INDEXTANEKALIST", Value: "11"},
		{ProdKe: "0", Column: "INDEXTANEKALIST", Value: "12"},
	}, deleteScopes(lb, true, key))
}

func TestDeleteOldPerObjekMembatasiDenganKolomPenghubung(t *testing.T) {
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer conn.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM POOLDATA.T_COVERAGELIST_CARGO WHERE NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2 AND TO_CHAR(INDEXOBJECT) = :3")).
		WithArgs("P1", "1", "2").WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectRollback()

	tx, err := conn.Begin()
	require.NoError(t, err)
	columns := map[string]kc.Column{"INDEXOBJECT": {Name: "INDEXOBJECT"}}
	n, err := deleteOld(context.Background(), tx, "T_COVERAGELIST_CARGO", columns, "P1",
		[]scope{{ProdKe: "1", Column: "INDEXOBJECT", Value: "2"}})
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOldMenolakBilaKolomPenghubungTidakAda(t *testing.T) {
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer conn.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()
	tx, err := conn.Begin()
	require.NoError(t, err)

	// Tidak ada DELETE yang diharapkan: menghapus per versi akan membuang objek lain.
	_, err = deleteOld(context.Background(), tx, "T_SPREADINGLIST", map[string]kc.Column{}, "P1",
		[]scope{{ProdKe: "1", Column: "INDEXOBJECT", Value: "2"}})
	require.ErrorContains(t, err, "tidak punya kolom INDEXOBJECT")
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
