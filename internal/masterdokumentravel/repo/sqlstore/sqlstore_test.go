package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdokumentravel"
)

var errDB = errors.New("db mati")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah teks kueri bernama menjadi pola regexp yang cocok persis.
func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

var docColumns = []string{"DOCID", "NAMADOKUMEN"}

func TestSplitByNameIgnoresPreambleAndComments(t *testing.T) {
	got := splitByName("SELECT 0\n-- name: a\n-- komentar\nSELECT 1\n-- name: kosong\n-- name: b\nSELECT 2")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}

func TestGetQueryPanicsOnUnknownName(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
	require.Contains(t, getQuery("travel_document_list"), "FROM POOLDATA.M_DOCTRAVEL")
}

func TestListMapsAndTrimsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("travel_document_list")).
		WillReturnRows(sqlmock.NewRows(docColumns).AddRow(" 100001 ", " Paspor ").AddRow("100002", nil))

	rows, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterdokumentravel.TravelDocument{{ID: "100001", Name: "Paspor"}, {ID: "100002"}}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("travel_document_list")).WillReturnError(errDB)
	_, err := repo.List(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca daftar dokumen")

	mock.ExpectQuery(q("travel_document_list")).WillReturnRows(sqlmock.NewRows([]string{"DOCID"}).AddRow("1"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca baris dokumen")

	mock.ExpectQuery(q("travel_document_list")).
		WillReturnRows(sqlmock.NewRows(docColumns).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menelusuri daftar dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("travel_document_get")).WithArgs("100001").
		WillReturnRows(sqlmock.NewRows(docColumns).AddRow("100001", " Paspor "))
	doc, err := repo.Get(context.Background(), " 100001 ")
	require.NoError(t, err)
	require.Equal(t, masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor"}, doc)

	mock.ExpectQuery(q("travel_document_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(docColumns))
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)

	mock.ExpectQuery(q("travel_document_get")).WithArgs("9").WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, `membaca dokumen "9"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewIssuesIDFromSiteAndSequence(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("travel_document_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	mock.ExpectQuery(q("travel_document_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	mock.ExpectExec(q("travel_document_insert")).WithArgs("100007", "Visa").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	doc, err := NewRepo(db).InsertNew(context.Background(), masterdokumentravel.Input{Name: "Visa"})
	require.NoError(t, err)
	require.Equal(t, masterdokumentravel.TravelDocument{ID: "100007", Name: "Visa"}, doc)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewErrors(t *testing.T) {
	siteRows := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"ID"}).AddRow("1") }
	seqRows := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"N"}).AddRow(1) }
	cases := []struct {
		name   string
		setup  func(m sqlmock.Sqlmock)
		want   string
		wrapDB bool
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, "memulai transaksi", true},
		{"site missing", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("travel_document_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectRollback()
		}, "tidak memuat baris CURRENT_SITE aktif", false},
		{"site error", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("travel_document_site")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "membaca kode situs", true},
		{"sequence", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("travel_document_site")).WillReturnRows(siteRows())
			m.ExpectQuery(q("travel_document_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengambil nomor urut", true},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("travel_document_site")).WillReturnRows(siteRows())
			m.ExpectQuery(q("travel_document_next_sequence")).WillReturnRows(seqRows())
			m.ExpectExec(q("travel_document_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `menyisipkan dokumen "100001"`, true},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("travel_document_site")).WillReturnRows(siteRows())
			m.ExpectQuery(q("travel_document_next_sequence")).WillReturnRows(seqRows())
			m.ExpectExec(q("travel_document_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, "menutup transaksi sisip", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.setup(mock)
			doc, err := NewRepo(db).InsertNew(context.Background(), masterdokumentravel.Input{Name: "x"})
			require.ErrorContains(t, err, tc.want)
			if tc.wrapDB {
				require.ErrorIs(t, err, errDB)
			}
			require.Equal(t, masterdokumentravel.TravelDocument{}, doc)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	doc := masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor Baru"}

	mock.ExpectExec(q("travel_document_update")).WithArgs("Paspor Baru", "100001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), doc))

	mock.ExpectExec(q("travel_document_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), doc), masterdokumentravel.ErrNotFound)

	// Driver yang tidak dapat melaporkan jumlah baris dianggap berhasil.
	mock.ExpectExec(q("travel_document_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.NoError(t, repo.Update(context.Background(), doc))

	mock.ExpectExec(q("travel_document_update")).WillReturnError(errDB)
	err := repo.Update(context.Background(), doc)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, `memperbarui dokumen "100001"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("travel_document_check_table")).WillReturnRows(sqlmock.NewRows(docColumns))
	require.NoError(t, NewRepo(db).CheckTable(context.Background()))

	mock.ExpectQuery(q("travel_document_check_table")).WillReturnError(errDB)
	err := NewRepo(db).CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "POOLDATA.M_DOCTRAVEL tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}
