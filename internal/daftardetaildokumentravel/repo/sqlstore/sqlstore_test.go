package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetaildokumentravel"
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

var detailColumns = []string{"ID", "DOCID", "DOCUMENTNAME", "STSWAJIB", "MINUNGGAH"}

func TestSplitByNameIgnoresPreambleAndComments(t *testing.T) {
	got := splitByName("-- kepala\nSELECT 0\n-- name: a\n-- komentar\nSELECT 1\n-- name: kosong\n-- hanya komentar\n-- name: b\nSELECT 2\n")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}

func TestGetQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`daftardetaildokumentravel/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { getQuery("tidak_ada") })
	require.Contains(t, getQuery("detail_list"), "FROM POOLDATA.V_LST_DOC_TRAVEL")
}

// TestNoCoverageQueriesRemain menjaga sebuah KEPUTUSAN, bukan sebuah perhitungan.
//
// Grid Plan dan Jaminan tidak ada di layar Pega yang berjalan (Work Owner, 2026-10-03),
// sehingga modul ini tidak menyentuh V_LST_DOC_TRAVEL_COVERAGE maupun M_PLANTRAVEL.
// Kueri yang menyebut keduanya pernah ada di berkas .sql ini dan dicabut; uji ini yang
// membuat penambahannya kembali gagal di CI alih-alih lolos diam-diam.
func TestNoCoverageQueriesRemain(t *testing.T) {
	for _, name := range []string{
		"detail_coverage_list",
		"detail_coverage_insert",
		"detail_coverage_delete_all",
		"detail_coverage_check_table",
		"plan_list",
		"coverage_list",
		"plan_check_table",
	} {
		_, exists := query[name]
		require.False(t, exists, "kueri %q seharusnya sudah tidak ada", name)
	}

	for name, text := range query {
		require.NotContains(t, text, "LST_DOC_TRAVEL_COVERAGE", "kueri %q menyentuh tabel coverage", name)
		require.NotContains(t, text, "M_PLANTRAVEL", "kueri %q menyentuh master plan", name)
	}
}

func TestListMapsAndTrimsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("detail_list")).WillReturnRows(sqlmock.NewRows(detailColumns).
		AddRow(" 00001 ", " 100001 ", " Paspor ", 1, 2).
		AddRow("00002", nil, nil, nil, nil).
		AddRow("00003", "X", "Y", 0, 0))

	rows, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftardetaildokumentravel.Detail{
		{ID: "00001", DocumentID: "100001", DocumentName: "Paspor", Mandatory: true, MinUpload: 2},
		{ID: "00002"},
		{ID: "00003", DocumentID: "X", DocumentName: "Y"},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_list")).WillReturnError(errDB)
		_, err := NewRepo(db).List(context.Background())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "membaca daftar detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_list")).WillReturnRows(sqlmock.NewRows(detailColumns).
			AddRow("1", "a", "b", "bukan angka", 0))
		_, err := NewRepo(db).List(context.Background())
		require.ErrorContains(t, err, "membaca baris detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_list")).WillReturnRows(sqlmock.NewRows(detailColumns).
			AddRow("1", "a", "b", 1, 0).RowError(0, errDB))
		_, err := NewRepo(db).List(context.Background())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "menelusuri daftar detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetReadsOneRowAndTrimsKey(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("detail_get")).WithArgs("00003").
		WillReturnRows(sqlmock.NewRows(detailColumns).AddRow("00003", "100004", "Bagasi", 0, 2))

	row, err := NewRepo(db).Get(context.Background(), " 00003 ")
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID: "00003", DocumentID: "100004", DocumentName: "Bagasi", MinUpload: 2,
	}, row)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").WillReturnRows(sqlmock.NewRows(detailColumns))
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").WillReturnError(errDB)
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, `membaca detail "1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func sequenceRows(n int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(n)
}

func TestInsertNewIssuesIDAndWritesInOneTransaction(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(12))
	mock.ExpectExec(q("detail_insert")).WithArgs("00012", "100006", "Surat", 1, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertNew(context.Background(), daftardetaildokumentravel.Input{
		DocumentID: "100006", DocumentName: "Surat", Mandatory: true, MinUpload: 3,
	})
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID: "00012", DocumentID: "100006", DocumentName: "Surat", Mandatory: true, MinUpload: 3,
	}, saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewKeepsLongSequenceAsIs(t *testing.T) {
	// Nomor di atas 99999 dikembalikan APA ADANYA, tanpa dipotong — memotongnya akan
	// menghasilkan ID GANDA, dan ID ganda di sini berarti dua aturan dokumen berbagi satu
	// kunci yang dirujuk jalur registrasi klaim.
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1234567))
	mock.ExpectExec(q("detail_insert")).WithArgs("1234567", "", "", 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertNew(context.Background(), daftardetaildokumentravel.Input{})
	require.NoError(t, err)
	require.Equal(t, "1234567", saved.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewErrorsRollBack(t *testing.T) {
	input := daftardetaildokumentravel.Input{DocumentID: "D"}
	cases := []struct {
		name  string
		setup func(mock sqlmock.Sqlmock)
		want  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, "memulai transaksi"},
		{"sequence", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengambil nomor urut"},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1))
			m.ExpectExec(q("detail_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `menyisipkan detail "00001"`},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1))
			m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, "menutup transaksi"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.setup(mock)
			saved, err := NewRepo(db).InsertNew(context.Background(), input)
			require.ErrorIs(t, err, errDB)
			require.ErrorContains(t, err, tc.want)
			require.Equal(t, daftardetaildokumentravel.Detail{}, saved)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateWritesRowAndTrimsKey(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(q("detail_update")).WithArgs("100004", "Bagasi", 0, 2, "00003").
		WillReturnResult(sqlmock.NewResult(0, 1))

	saved, err := NewRepo(db).Update(context.Background(), " 00003 ", daftardetaildokumentravel.Input{
		DocumentID: "100004", DocumentName: "Bagasi", MinUpload: 2,
	})
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID: "00003", DocumentID: "100004", DocumentName: "Bagasi", MinUpload: 2,
	}, saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		// UPDATE terhadap ID yang tidak ada berhasil tanpa galat di SQL. Membiarkannya
		// akan melaporkan "tersimpan" atas perubahan yang tidak pernah terjadi.
		db, mock := newMock(t)
		mock.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		_, err := NewRepo(db).Update(context.Background(), "9", daftardetaildokumentravel.Input{})
		require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectExec(q("detail_update")).WillReturnError(errDB)
		_, err := NewRepo(db).Update(context.Background(), "9", daftardetaildokumentravel.Input{})
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, `memperbarui detail "9"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepoCheckTable(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_check_table")).WillReturnRows(sqlmock.NewRows(detailColumns))
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_check_table")).WillReturnError(errDB)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "POOLDATA.V_LST_DOC_TRAVEL tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentRepo(t *testing.T) {
	cols := []string{"DOCID", "NAMADOKUMEN"}
	t.Run("list", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_list")).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(" 1 ", " Paspor ").AddRow("2", nil))
		rows, err := NewDocumentRepo(db).List(context.Background())
		require.NoError(t, err)
		require.Equal(t, []daftardetaildokumentravel.Document{{ID: "1", Name: "Paspor"}, {ID: "2"}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_list")).WillReturnError(errDB)
		_, err := NewDocumentRepo(db).List(context.Background())
		require.ErrorContains(t, err, "membaca master dokumen travel")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_list")).WillReturnRows(sqlmock.NewRows([]string{"DOCID"}).AddRow("1"))
		_, err := NewDocumentRepo(db).List(context.Background())
		require.ErrorContains(t, err, "membaca baris master dokumen")
	})
	t.Run("rows", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_list")).WillReturnRows(sqlmock.NewRows(cols).AddRow("1", "a").RowError(0, errDB))
		_, err := NewDocumentRepo(db).List(context.Background())
		require.ErrorContains(t, err, "menelusuri master dokumen travel")
	})
	t.Run("check", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_check_table")).WillReturnRows(sqlmock.NewRows(cols))
		require.NoError(t, NewDocumentRepo(db).CheckTable(context.Background()))
		mock.ExpectQuery(q("document_check_table")).WillReturnError(errDB)
		err := NewDocumentRepo(db).CheckTable(context.Background())
		require.ErrorContains(t, err, "POOLDATA.M_DOCTRAVEL tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
