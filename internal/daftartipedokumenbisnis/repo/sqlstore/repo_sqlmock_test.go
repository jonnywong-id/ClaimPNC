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

	"claim-pnc/internal/daftartipedokumenbisnis"
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
func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

var errBoom = errors.New("boom")

var ruleColumns = []string{
	"ID", "BUSINESSID", "BUSINESS_NAME", "DOCUMENT_TYPE_ID", "TYPE_DOCUMENT",
	"OBJECT_DOC_ID", "KET_DOC_OBJ", "DOC_TYPE_DT_ID", "DETAIL_DOKUMEN", "STS_WAJIB", "MIN_DOC",
}

var editor = daftartipedokumenbisnis.Editor{
	Identity: "90000001",
	At:       time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC),
}

func TestListBusinessesMapsAndTrims(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NOTE"}).AddRow(" 001 ", " Fire ").AddRow("004", nil))

	got, err := NewRepo(db).ListBusinesses(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Business{{ID: "001", Name: "Fire"}, {ID: "004"}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBusinessesErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("business_list")).WillReturnError(errBoom)
		_, err := NewRepo(db).ListBusinesses(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		// Satu kolom saja membuat Scan dua tujuan gagal.
		mock.ExpectQuery(q("business_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).ListBusinesses(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("business_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID", "NOTE"}).AddRow("1", "a").RowError(0, errBoom))
		_, err := NewRepo(db).ListBusinesses(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListByBusinessMapsRowsAndMandatoryText(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("rule_list_by_business")).WithArgs("001").WillReturnRows(
		sqlmock.NewRows(ruleColumns).
			AddRow("10001 ", "001", "Fire", "20001", "REGISTER", "30001", "Bangunan", "40001",
				" Laporan ", "1", int64(2)).
			// STS_WAJIB selain "1" — termasuk NULL — dibaca sebagai tidak wajib.
			AddRow("10002", "001", "Fire", "20001", "REGISTER", nil, nil, "40002", "Foto", nil, nil).
			AddRow("10003", "001", "Fire", "20003", "SURVEY", nil, nil, "40003", "-", "Y", int64(0)))

	got, err := NewRepo(db).ListByBusiness(context.Background(), " 001 ")
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.DocumentRule{
		{
			ID: "10001", BusinessID: "001", BusinessName: "Fire", DocumentTypeID: "20001",
			DocumentTypeName: "REGISTER", ObjectDocID: "30001", ObjectDocName: "Bangunan",
			DetailTypeDocID: "40001", DetailDocument: "Laporan", Mandatory: true, MinDocument: 2,
		},
		{
			ID: "10002", BusinessID: "001", BusinessName: "Fire", DocumentTypeID: "20001",
			DocumentTypeName: "REGISTER", DetailTypeDocID: "40002", DetailDocument: "Foto",
		},
		{
			ID: "10003", BusinessID: "001", BusinessName: "Fire", DocumentTypeID: "20003",
			DocumentTypeName: "SURVEY", DetailTypeDocID: "40003", DetailDocument: "-",
		},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListByBusinessErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_list_by_business")).WithArgs("001").WillReturnError(errBoom)
		_, err := NewRepo(db).ListByBusiness(context.Background(), "001")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca aturan bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_list_by_business")).WithArgs("001").WillReturnRows(
			sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).ListByBusiness(context.Background(), "001")
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris aturan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_list_by_business")).WithArgs("001").WillReturnRows(
			sqlmock.NewRows(ruleColumns).
				AddRow("1", "001", "a", "b", "c", nil, nil, "d", "e", "1", int64(1)).
				RowError(0, errBoom))
		_, err := NewRepo(db).ListByBusiness(context.Background(), "001")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri aturan bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetReadsRuleAndCoverages(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("rule_get")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows(ruleColumns).
			AddRow("10001", "001", "Fire", "20001", "REGISTER", nil, nil, "40001", "Laporan", "1", int64(1)))
	mock.ExpectQuery(q("coverage_list")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows([]string{"COVERAGEID"}).AddRow(" 10009 ").AddRow(nil))

	got, err := NewRepo(db).Get(context.Background(), " 10001 ")
	require.NoError(t, err)
	require.Equal(t, "10001", got.ID)
	require.True(t, got.Mandatory)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10009"}, {ID: ""}}, got.Coverages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(ruleColumns))
		_, err := NewRepo(db).Get(context.Background(), "9")
		require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_get")).WithArgs("9").WillReturnError(errBoom)
		_, err := NewRepo(db).Get(context.Background(), "9")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca aturan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	coverageCase := func(t *testing.T, setup func(mock sqlmock.Sqlmock), message string) {
		t.Helper()
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows(ruleColumns).AddRow("1", "001", "a", "b", "c", nil, nil, "d", "e", "0", int64(0)))
		setup(mock)
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
		require.NoError(t, mock.ExpectationsWereMet())
	}
	t.Run("coverage query", func(t *testing.T) {
		coverageCase(t, func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(q("coverage_list")).WithArgs("1").WillReturnError(errBoom)
		}, "membaca jaminan aturan")
	})
	t.Run("coverage scan", func(t *testing.T) {
		coverageCase(t, func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(q("coverage_list")).WithArgs("1").WillReturnRows(
				sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
		}, "membaca baris jaminan")
	})
	t.Run("coverage rows err", func(t *testing.T) {
		coverageCase(t, func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(q("coverage_list")).WithArgs("1").WillReturnRows(
				sqlmock.NewRows([]string{"COVERAGEID"}).AddRow("1").RowError(0, errBoom))
		}, "menelusuri jaminan aturan")
	})
}

func TestInsertBatchWritesProductInOneTransaction(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow(" 1 "))
	mock.ExpectQuery(q("rule_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(7)))
	mock.ExpectExec(q("rule_insert")).
		WithArgs("10007", "001", "20001", nil, "40001", "Laporan", "1", 1, editor.At, editor.Identity).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("rule_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(8)))
	mock.ExpectExec(q("rule_insert")).
		WithArgs("10008", "004", "20001", nil, "40001", "Laporan", "1", 1, editor.At, editor.Identity).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertBatch(context.Background(), daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001", "004"},
		Rules: []daftartipedokumenbisnis.Input{{
			DocumentTypeID: "20001", DetailTypeDocID: "40001", DetailDocument: "Laporan",
			Mandatory: true, MinDocument: 1,
		}},
	}, editor)
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Equal(t, "10007", saved[0].ID)
	require.Equal(t, "001", saved[0].BusinessID)
	require.Equal(t, "10008", saved[1].ID)
	require.Equal(t, "004", saved[1].BusinessID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertBatchWritesObjectDocAndNotMandatory(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("2"))
	mock.ExpectQuery(q("rule_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(12345)))
	// Nomor yang melampaui empat digit tidak dipotong, sama dengan LPAD Oracle.
	mock.ExpectExec(q("rule_insert")).
		WithArgs("212345", "001", "20003", "30001", "40003", "Berita", "0", 0, editor.At, editor.Identity).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).InsertBatch(context.Background(), daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001"},
		Rules:       []daftartipedokumenbisnis.Input{{DocumentTypeID: "20003", ObjectDocID: "30001", DetailTypeDocID: "40003", DetailDocument: "Berita"}},
	}, editor)
	require.NoError(t, err)
	require.Equal(t, "30001", saved[0].ObjectDocID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertBatchErrorsRollBack(t *testing.T) {
	input := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"001"},
		Rules:       []daftartipedokumenbisnis.Input{{DocumentTypeID: "20001"}},
	}

	t.Run("begin", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("site missing", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}))
		mock.ExpectRollback()
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.Error(t, err)
		require.Contains(t, err.Error(), "M_SITE_DATABASE")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("site query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_site")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca kode situs")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("sequence", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("1"))
		mock.ExpectQuery(q("rule_next_sequence")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "mengambil nomor urut")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("1"))
		mock.ExpectQuery(q("rule_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1)))
		mock.ExpectExec(q("rule_insert")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), `menyisipkan aturan "10001"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("1"))
		mock.ExpectQuery(q("rule_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1)))
		mock.ExpectExec(q("rule_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := NewRepo(db).InsertBatch(context.Background(), input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menutup transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateWritesAndReReads(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(q("rule_update")).
		WithArgs("20004", "30003", "40005", "Kuitansi", "0", 2, editor.At, editor.Identity, "10001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("rule_get")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows(ruleColumns).
			AddRow("10001", "001", "Fire", "20004", "PAYMENT", "30003", "Orang", "40005", "Kuitansi", "0", int64(2)))
	mock.ExpectQuery(q("coverage_list")).WithArgs("10001").WillReturnRows(sqlmock.NewRows([]string{"COVERAGEID"}))

	got, err := NewRepo(db).Update(context.Background(), " 10001 ", daftartipedokumenbisnis.Input{
		DocumentTypeID: "20004", ObjectDocID: "30003", DetailTypeDocID: "40005",
		DetailDocument: "Kuitansi", MinDocument: 2,
	}, editor)
	require.NoError(t, err)
	require.Equal(t, "PAYMENT", got.DocumentTypeName)
	require.Equal(t, "Orang", got.ObjectDocName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	input := daftartipedokumenbisnis.Input{DocumentTypeID: "1"}

	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectExec(q("rule_update")).WillReturnError(errBoom)
		_, err := NewRepo(db).Update(context.Background(), "1", input, editor)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "memperbarui aturan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no row touched", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectExec(q("rule_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		_, err := NewRepo(db).Update(context.Background(), "1", input, editor)
		require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows affected unknown still re-reads", func(t *testing.T) {
		// Driver yang tidak dapat melaporkan baris tersentuh tidak dianggap baris hilang.
		db, mock := newMock(t)
		mock.ExpectExec(q("rule_update")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
		mock.ExpectQuery(q("rule_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows(ruleColumns).AddRow("1", "001", "a", "1", "c", nil, nil, "d", "e", "1", int64(1)))
		mock.ExpectQuery(q("coverage_list")).WithArgs("1").WillReturnRows(sqlmock.NewRows([]string{"COVERAGEID"}))
		got, err := NewRepo(db).Update(context.Background(), "1", input, editor)
		require.NoError(t, err)
		require.Equal(t, "1", got.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAddCoverageInsertsWithOwningBusiness(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rule_business_of")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows([]string{"BUSINESSID"}).AddRow(" 001 "))
	mock.ExpectQuery(q("coverage_exists")).WithArgs("10001", "10015").WillReturnRows(
		sqlmock.NewRows([]string{"N"}).AddRow(0))
	mock.ExpectExec(q("coverage_insert")).WithArgs("10001", "001", "10015").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("rule_get")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows(ruleColumns).AddRow("10001", "001", "Fire", "1", "R", nil, nil, "d", "e", "1", int64(1)))
	mock.ExpectQuery(q("coverage_list")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows([]string{"COVERAGEID"}).AddRow("10015"))

	got, err := NewRepo(db).AddCoverage(context.Background(), " 10001 ", " 10015 ")
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10015"}}, got.Coverages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAddCoverageExistingDoesNotInsert(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rule_business_of")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows([]string{"BUSINESSID"}).AddRow("001"))
	mock.ExpectQuery(q("coverage_exists")).WithArgs("10001", "10009").WillReturnRows(
		sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("rule_get")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows(ruleColumns).AddRow("10001", "001", "Fire", "1", "R", nil, nil, "d", "e", "1", int64(1)))
	mock.ExpectQuery(q("coverage_list")).WithArgs("10001").WillReturnRows(
		sqlmock.NewRows([]string{"COVERAGEID"}).AddRow("10009"))

	got, err := NewRepo(db).AddCoverage(context.Background(), "10001", "10009")
	require.NoError(t, err)
	require.Len(t, got.Coverages, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAddCoverageErrors(t *testing.T) {
	t.Run("rule missing", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_business_of")).WithArgs("9").WillReturnRows(sqlmock.NewRows([]string{"BUSINESSID"}))
		mock.ExpectRollback()
		_, err := NewRepo(db).AddCoverage(context.Background(), "9", "1")
		require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rule query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_business_of")).WithArgs("9").WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).AddCoverage(context.Background(), "9", "1")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca aturan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("exists query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_business_of")).WithArgs("9").WillReturnRows(
			sqlmock.NewRows([]string{"BUSINESSID"}).AddRow("001"))
		mock.ExpectQuery(q("coverage_exists")).WithArgs("9", "1").WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).AddCoverage(context.Background(), "9", "1")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "memeriksa jaminan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("rule_business_of")).WithArgs("9").WillReturnRows(
			sqlmock.NewRows([]string{"BUSINESSID"}).AddRow("001"))
		mock.ExpectQuery(q("coverage_exists")).WithArgs("9", "1").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
		mock.ExpectExec(q("coverage_insert")).WithArgs("9", "001", "1").WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).AddCoverage(context.Background(), "9", "1")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menambahkan jaminan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
		mock.ExpectQuery(q("coverage_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("rule_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
		mock.ExpectQuery(q("coverage_check_table")).WillReturnError(errBoom)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "coverage_check_table")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBusinessRepoList(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("business_choice_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NOTE"}).AddRow("001", " Fire "))
	got, err := NewBusinessRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Business{{ID: "001", Name: "Fire"}}, got)
	require.NoError(t, mock.ExpectationsWereMet())

	db2, mock2 := newMock(t)
	mock2.ExpectQuery(q("business_choice_list")).WillReturnError(errBoom)
	_, err = NewBusinessRepo(db2).List(context.Background())
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca master bisnis")

	db3, mock3 := newMock(t)
	mock3.ExpectQuery(q("business_choice_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = NewBusinessRepo(db3).List(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris master bisnis")
}

func TestDocumentTypeAndObjectDocRepoList(t *testing.T) {
	type lister interface {
		List(context.Context) ([]daftartipedokumenbisnis.Reference, error)
	}
	cases := []struct {
		name    string
		query   string
		build   func(*sql.DB) lister
		failure string
		scanErr string
	}{
		{"document type", "document_type_choice_list", func(db *sql.DB) lister { return NewDocumentTypeRepo(db) },
			"membaca master tipe dokumen", "membaca baris master tipe dokumen"},
		{"object doc", "object_doc_choice_list", func(db *sql.DB) lister { return NewObjectDocRepo(db) },
			"membaca master objek dokumen", "membaca baris master objek dokumen"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectQuery(q(c.query)).WillReturnRows(
				sqlmock.NewRows([]string{"ID", "NAME"}).AddRow(" 1 ", " A ").AddRow("2", nil))
			got, err := c.build(db).List(context.Background())
			require.NoError(t, err)
			require.Equal(t, []daftartipedokumenbisnis.Reference{{ID: "1", Name: "A"}, {ID: "2"}}, got)
			require.NoError(t, mock.ExpectationsWereMet())

			db2, mock2 := newMock(t)
			mock2.ExpectQuery(q(c.query)).WillReturnError(errBoom)
			_, err = c.build(db2).List(context.Background())
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), c.failure)

			db3, mock3 := newMock(t)
			mock3.ExpectQuery(q(c.query)).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			_, err = c.build(db3).List(context.Background())
			require.Error(t, err)
			require.Contains(t, err.Error(), c.scanErr)
		})
	}
}

func TestDetailTypeDocRepoListCarriesParent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("detail_type_doc_choice_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAME", "DOC_TYPE_ID"}).AddRow("40001", " Laporan ", " 20001 "))
	got, err := NewDetailTypeDocRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Reference{{ID: "40001", Name: "Laporan", ParentID: "20001"}}, got)
	require.NoError(t, mock.ExpectationsWereMet())

	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_type_doc_choice_list")).WillReturnError(errBoom)
		_, err := NewDetailTypeDocRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca master detail dokumen")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_type_doc_choice_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewDetailTypeDocRepo(db).List(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris master detail dokumen")
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("detail_type_doc_choice_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID", "NAME", "P"}).AddRow("1", "a", "b").RowError(0, errBoom))
		_, err := NewDetailTypeDocRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri master detail dokumen")
	})
}

func TestSplitByNameSkipsCommentsAndEmptyBodies(t *testing.T) {
	got := splitByName("-- komentar awal\n-- name: a\n-- catatan\nSELECT 1\n-- name: kosong\n-- hanya komentar\n-- name: b\nSELECT 2\n")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}
