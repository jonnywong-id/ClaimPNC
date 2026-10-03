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
var coverageColumns = []string{"ID", "PLANID", "PLANNAME", "COVERAGEID", "COVERAGENAME"}

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

func TestGetReadsDetailAndCoverages(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("detail_get")).WithArgs("00003").
		WillReturnRows(sqlmock.NewRows(detailColumns).AddRow("00003", "100004", "Bagasi", 0, 2))
	mock.ExpectQuery(q("detail_coverage_list")).WithArgs("00003").
		WillReturnRows(sqlmock.NewRows(coverageColumns).AddRow(" 00003 ", " TP01 ", " Silver ", " TC02 ", " Bagasi "))

	row, err := NewRepo(db).Get(context.Background(), " 00003 ")
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID: "00003", DocumentID: "100004", DocumentName: "Bagasi", MinUpload: 2,
		Coverages: []daftardetaildokumentravel.Coverage{
			{ID: "00003", PlanID: "TP01", PlanName: "Silver", CoverageID: "TC02", CoverageName: "Bagasi"},
		},
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
	t.Run("detail query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").WillReturnError(errDB)
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, `membaca detail "1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("coverage query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows(detailColumns).AddRow("1", "a", "b", 1, 1))
		mock.ExpectQuery(q("detail_coverage_list")).WithArgs("1").WillReturnError(errDB)
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "membaca coverage detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("coverage scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows(detailColumns).AddRow("1", "a", "b", 1, 1))
		mock.ExpectQuery(q("detail_coverage_list")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("x"))
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorContains(t, err, "membaca baris coverage")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("coverage rows", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_get")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows(detailColumns).AddRow("1", "a", "b", 1, 1))
		mock.ExpectQuery(q("detail_coverage_list")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows(coverageColumns).AddRow("1", "p", "p", "c", "c").RowError(0, errDB))
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "menelusuri coverage detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func sequenceRows(n int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(n)
}

func TestInsertNewWritesDetailAndCoveragesInOneTransaction(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(12))
	mock.ExpectExec(q("detail_insert")).WithArgs("00012", "100006", "Surat", 1, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(13))
	mock.ExpectExec(q("detail_coverage_insert")).
		WithArgs("00013", "00012", "100006", "Surat", 1, "TP01", "Silver", "TC01", "Medis").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertNew(context.Background(), daftardetaildokumentravel.Input{
		DocumentID: "100006", DocumentName: "Surat", Mandatory: true, MinUpload: 3,
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: "TP01", PlanName: "Silver", CoverageID: "TC01", CoverageName: "Medis"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, daftardetaildokumentravel.Detail{
		ID: "00012", DocumentID: "100006", DocumentName: "Surat", Mandatory: true, MinUpload: 3,
		Coverages: []daftardetaildokumentravel.Coverage{
			{ID: "00013", PlanID: "TP01", PlanName: "Silver", CoverageID: "TC01", CoverageName: "Medis"},
		},
	}, saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewWithoutCoveragesAndLongSequence(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1234567))
	mock.ExpectExec(q("detail_insert")).WithArgs("1234567", "", "", 0, 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertNew(context.Background(), daftardetaildokumentravel.Input{})
	require.NoError(t, err)
	require.Equal(t, "1234567", saved.ID)
	require.Nil(t, saved.Coverages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewErrorsRollBack(t *testing.T) {
	input := daftardetaildokumentravel.Input{
		DocumentID: "D", Coverages: []daftardetaildokumentravel.CoverageInput{{PlanID: "P"}},
	}
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
		{"coverage sequence", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1))
			m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengambil nomor urut"},
		{"coverage insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1))
			m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(2))
			m.ExpectExec(q("detail_coverage_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `menyisipkan coverage detail "00001"`},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(1))
			m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(2))
			m.ExpectExec(q("detail_coverage_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
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

func TestUpdateReplacesCoverages(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("detail_update")).WithArgs("100004", "Bagasi", 0, 2, "00003").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("detail_coverage_delete_all")).WithArgs("00003").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sequenceRows(20))
	mock.ExpectExec(q("detail_coverage_insert")).
		WithArgs("00020", "00003", "100004", "Bagasi", 0, "TP02", "Gold", "TC02", "Bagasi").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).Update(context.Background(), " 00003 ", daftardetaildokumentravel.Input{
		DocumentID: "100004", DocumentName: "Bagasi", MinUpload: 2,
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: "TP02", PlanName: "Gold", CoverageID: "TC02", CoverageName: "Bagasi"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "00003", saved.ID)
	require.Len(t, saved.Coverages, 1)
	require.Equal(t, "00020", saved.Coverages[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	input := daftardetaildokumentravel.Input{Coverages: []daftardetaildokumentravel.CoverageInput{{PlanID: "P"}}}
	t.Run("not found", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "9", input)
		require.ErrorIs(t, err, daftardetaildokumentravel.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	cases := []struct {
		name  string
		setup func(m sqlmock.Sqlmock)
		want  string
	}{
		{"update", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("detail_update")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `memperbarui detail "9"`},
		{"delete", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("detail_coverage_delete_all")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `membuang coverage detail "9"`},
		{"coverage", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("detail_coverage_delete_all")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("detail_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengambil nomor urut"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.setup(mock)
			_, err := NewRepo(db).Update(context.Background(), "9", input)
			require.ErrorIs(t, err, errDB)
			require.ErrorContains(t, err, tc.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepoCheckTable(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_check_table")).WillReturnRows(sqlmock.NewRows(detailColumns))
		mock.ExpectQuery(q("detail_coverage_check_table")).WillReturnRows(sqlmock.NewRows(coverageColumns))
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_check_table")).WillReturnRows(sqlmock.NewRows(detailColumns))
		mock.ExpectQuery(q("detail_coverage_check_table")).WillReturnError(errDB)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "(detail_coverage_check_table)")
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

func TestPlanRepo(t *testing.T) {
	planCols := []string{"PLANID", "PLANNAME"}
	coverageCols := []string{"COVERAGEID", "COVERAGENAME", "PLANID"}
	t.Run("plans", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("plan_list")).WillReturnRows(sqlmock.NewRows(planCols).AddRow(" TP01 ", " Silver "))
		rows, err := NewPlanRepo(db).ListPlans(context.Background())
		require.NoError(t, err)
		require.Equal(t, []daftardetaildokumentravel.Plan{{ID: "TP01", Name: "Silver"}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("plans errors", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("plan_list")).WillReturnError(errDB)
		_, err := NewPlanRepo(db).ListPlans(context.Background())
		require.ErrorContains(t, err, "membaca master plan travel")
		mock.ExpectQuery(q("plan_list")).WillReturnRows(sqlmock.NewRows([]string{"PLANID"}).AddRow("1"))
		_, err = NewPlanRepo(db).ListPlans(context.Background())
		require.ErrorContains(t, err, "membaca baris plan travel")
		mock.ExpectQuery(q("plan_list")).WillReturnRows(sqlmock.NewRows(planCols).AddRow("1", "a").RowError(0, errDB))
		_, err = NewPlanRepo(db).ListPlans(context.Background())
		require.ErrorContains(t, err, "menelusuri master plan travel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("coverages", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("coverage_list")).
			WillReturnRows(sqlmock.NewRows(coverageCols).AddRow(" TC01 ", " Medis ", " TP01 "))
		rows, err := NewPlanRepo(db).ListCoverages(context.Background())
		require.NoError(t, err)
		require.Equal(t, []daftardetaildokumentravel.CoverageOption{{ID: "TC01", Name: "Medis", PlanID: "TP01"}}, rows)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("coverages errors", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("coverage_list")).WillReturnError(errDB)
		_, err := NewPlanRepo(db).ListCoverages(context.Background())
		require.ErrorContains(t, err, "membaca master jaminan travel")
		mock.ExpectQuery(q("coverage_list")).WillReturnRows(sqlmock.NewRows([]string{"COVERAGEID"}).AddRow("1"))
		_, err = NewPlanRepo(db).ListCoverages(context.Background())
		require.ErrorContains(t, err, "membaca baris jaminan travel")
		mock.ExpectQuery(q("coverage_list")).
			WillReturnRows(sqlmock.NewRows(coverageCols).AddRow("1", "a", "p").RowError(0, errDB))
		_, err = NewPlanRepo(db).ListCoverages(context.Background())
		require.ErrorContains(t, err, "menelusuri master jaminan travel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("check", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("plan_check_table")).WillReturnRows(sqlmock.NewRows(planCols))
		require.NoError(t, NewPlanRepo(db).CheckTable(context.Background()))
		mock.ExpectQuery(q("plan_check_table")).WillReturnError(errDB)
		err := NewPlanRepo(db).CheckTable(context.Background())
		require.ErrorContains(t, err, "POOLDATA.M_PLANTRAVEL tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
