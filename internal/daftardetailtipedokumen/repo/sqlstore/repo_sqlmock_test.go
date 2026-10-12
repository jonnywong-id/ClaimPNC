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

	"claim-pnc/internal/daftardetailtipedokumen"
)

var errDB = errors.New("basis data mati")

var detailColumns = []string{
	"ID", "ID_TYPE", "NAMA_TYPE", "DETAIL", "STATUS_TERTANGGUNG",
	"ID_COL", "KET_COL", "ID_OBJ", "DESC_OBJ", "RESIKO",
}

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

func detailRow(id string) *sqlmock.Rows {
	return sqlmock.NewRows(detailColumns).AddRow(
		" "+id+" ", "10001", " Dokumen Registrasi ", "Kwitansi", "Tertanggung",
		"1001", "Gol A", "10002", "Polis", "0")
}

var editAt = time.Date(2026, 9, 20, 4, 5, 6, 0, time.UTC)

const editStamp = "20260920T040506.000 GMT"

func TestListMapsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("detail_list")).WillReturnRows(detailRow("100001").AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	got, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, daftardetailtipedokumen.DetailType{
		ID: "100001", DocumentTypeID: "10001", DocumentTypeName: "Dokumen Registrasi",
		Detail: "Kwitansi", InsuredStatus: "Tertanggung", CauseOfLossID: "1001",
		CauseOfLossDescription: "Gol A", ObjectDocumentID: "10002",
		ObjectDocumentDescription: "Polis", Risk: "0",
	}, got[0])
	require.Equal(t, daftardetailtipedokumen.DetailType{}, got[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("detail_list")).WillReturnError(errDB)
	_, err := repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar detail")

	mock.ExpectQuery(q("detail_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca baris detail")

	mock.ExpectQuery(q("detail_list")).WillReturnRows(detailRow("1").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar detail")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("detail_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(detailColumns))
	_, err := repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, daftardetailtipedokumen.ErrNotFound)

	mock.ExpectQuery(q("detail_get")).WithArgs("9").WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "9")
	require.ErrorContains(t, err, `membaca detail "9"`)

	require.NoError(t, mock.ExpectationsWereMet())
}

func sampleInput() daftardetailtipedokumen.Input {
	return daftardetailtipedokumen.Input{
		DocumentTypeID: "10001", Detail: "Kwitansi", InsuredStatus: "Tertanggung",
		CauseOfLossID: "1001", CauseOfLossDescription: "Gol A",
		ObjectDocumentID: "10002", ObjectDocumentDescription: "Polis", Risk: "0",
	}
}

func TestInsertNewWritesDetailThenRereads(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("10"))
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(7)))
	mock.ExpectExec(q("detail_insert")).
		WithArgs("100007", "10001", "Kwitansi", "Tertanggung", "1001", "Gol A", "10002", "Polis", "0", editStamp, "adminpnc").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("detail_get")).WithArgs("100007").WillReturnRows(detailRow("100007"))

	got, err := NewRepo(db).InsertNew(context.Background(), sampleInput(), daftardetailtipedokumen.Editor{Identity: "adminpnc", At: editAt})
	require.NoError(t, err)
	require.Equal(t, "100007", got.ID)
	require.Equal(t, "Dokumen Registrasi", got.DocumentTypeName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewFailures(t *testing.T) {
	by := daftardetailtipedokumen.Editor{Identity: "u", At: editAt}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, "memulai transaksi"},
		{"site", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_site")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "membaca kode situs"},
		{"sequence", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("10"))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengambil nomor urut"},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("10"))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1)))
			m.ExpectExec(q("detail_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `menyisipkan detail "100001"`},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("10"))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1)))
			m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, "menutup transaksi"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.setup(mock)
			_, err := NewRepo(db).InsertNew(context.Background(), sampleInput(), by)
			require.ErrorIs(t, err, errDB)
			require.ErrorContains(t, err, c.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateWritesDetailThenRereads(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("detail_update")).
		WithArgs("10001", "Kwitansi", "Tertanggung", "1001", "Gol A", "10002", "Polis", "0", editStamp, "pic", "100001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("detail_get")).WithArgs("100001").WillReturnRows(detailRow("100001"))

	got, err := NewRepo(db).Update(context.Background(), " 100001 ", sampleInput(), daftardetailtipedokumen.Editor{Identity: "pic", At: editAt})
	require.NoError(t, err)
	require.Equal(t, "100001", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	by := daftardetailtipedokumen.Editor{Identity: "u", At: editAt}

	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	_, err := NewRepo(db).Update(context.Background(), "9", sampleInput(), by)
	require.ErrorIs(t, err, daftardetailtipedokumen.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("detail_update")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = NewRepo(db).Update(context.Background(), "9", sampleInput(), by)
	require.ErrorContains(t, err, `memperbarui detail "9"`)
	require.NoError(t, mock.ExpectationsWereMet())

}

func TestCheckTables(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("detail_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(q("detail_write_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckWriteTable(context.Background()))

	mock.ExpectQuery(q("detail_write_check_table")).WillReturnError(errDB)
	err := repo.CheckWriteTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "(detail_write_check_table)")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceListMapsPairs(t *testing.T) {
	db, mock := newMock(t)
	repo := NewReferenceRepo(db)
	ctx := context.Background()
	pairRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"A", "B"}).AddRow(" 1 ", " Satu ").AddRow(nil, nil)
	}

	mock.ExpectQuery(q("document_type_choice_list")).WillReturnRows(pairRows())
	documentTypes, err := repo.ListDocumentTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, []daftardetailtipedokumen.DocumentTypeOption{{ID: "1", Name: "Satu"}, {}}, documentTypes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceListPropagatesErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewReferenceRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(q("document_type_choice_list")).WillReturnError(errDB)
	_, err := repo.ListDocumentTypes(ctx)
	require.ErrorContains(t, err, "membaca POOLDATA.V_LST_DOC_TYPE")

	mock.ExpectQuery(q("document_type_choice_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1"))
	_, err = repo.ListDocumentTypes(ctx)
	require.ErrorContains(t, err, "membaca baris POOLDATA.V_LST_DOC_TYPE")

	mock.ExpectQuery(q("document_type_choice_list")).WillReturnRows(
		sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "x").RowError(0, errDB))
	_, err = repo.ListDocumentTypes(ctx)
	require.ErrorContains(t, err, "menelusuri POOLDATA.V_LST_DOC_TYPE")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceCheckTable(t *testing.T) {
	// Urutan kueri dari peta tidak tetap, jadi pencocokan tanpa urutan.
	db, mock := newMock(t)
	mock.MatchExpectationsInOrder(false)
	for _, name := range []string{"document_type_check_table"} {
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	}
	require.NoError(t, NewReferenceRepo(db).CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())

	// Seluruh kueri gagal: apa pun yang dijalankan lebih dulu, galatnya menyebut objeknya.
	db, mock = newMock(t)
	mock.MatchExpectationsInOrder(false)
	for _, name := range []string{"document_type_check_table"} {
		mock.ExpectQuery(q(name)).WillReturnError(errDB)
	}
	err := NewReferenceRepo(db).CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "tidak dapat dibaca")
}

func TestGetQueryPanicsOnUnknownName(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("tidak_ada") })
}
