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

	"claim-pnc/internal/daftartipedokumen"
)

// newMockRepo menyiapkan repo di atas sqlmock dengan pencocok regexp.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exactQuery mencocokkan teks kueri persis seperti di berkas .sql.
func exactQuery(name string) string {
	return "^" + regexp.QuoteMeta(getQuery(name)) + "$"
}

var errBoom = errors.New("boom")

func TestListMapsRowsAndTrimsValues(t *testing.T) {
	repo, mock := newMockRepo(t)
	rows := sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES"}).
		AddRow("10001 ", " Dokumen Registrasi ", "Register").
		AddRow("10002", "Dokumen Survey", nil)
	mock.ExpectQuery(exactQuery("document_type_list")).WillReturnRows(rows)

	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumen.DocumentType{
		{ID: "10001", Type: "Dokumen Registrasi", ProcessStatus: "Register"},
		{ID: "10002", Type: "Dokumen Survey", ProcessStatus: ""},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("document_type_list")).WillReturnError(errBoom)

	_, err := repo.List(context.Background())
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca daftar tipe dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListScanError(t *testing.T) {
	repo, mock := newMockRepo(t)
	// Jumlah kolom tidak cocok membuat Scan gagal.
	rows := sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT"}).AddRow("1", "x")
	mock.ExpectQuery(exactQuery("document_type_list")).WillReturnRows(rows)

	_, err := repo.List(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris tipe dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRowsError(t *testing.T) {
	repo, mock := newMockRepo(t)
	rows := sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES"}).
		AddRow("10001", "A", "B").
		RowError(0, errBoom)
	mock.ExpectQuery(exactQuery("document_type_list")).WillReturnRows(rows)

	_, err := repo.List(context.Background())
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menelusuri daftar tipe dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTrimsIDAndMapsRow(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("document_type_get")).WithArgs("10003").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES"}).
			AddRow("10003", "Dokumen Komite", "Komite"))

	got, err := repo.Get(context.Background(), " 10003 ")
	require.NoError(t, err)
	require.Equal(t, daftartipedokumen.DocumentType{ID: "10003", Type: "Dokumen Komite", ProcessStatus: "Komite"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("document_type_get")).WithArgs("X").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES"}))

	_, err := repo.Get(context.Background(), "X")
	require.ErrorIs(t, err, daftartipedokumen.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("document_type_get")).WithArgs("X").WillReturnError(errBoom)

	_, err := repo.Get(context.Background(), "X")
	require.ErrorIs(t, err, errBoom)
	require.NotErrorIs(t, err, daftartipedokumen.ErrNotFound)
	require.Contains(t, err.Error(), `membaca tipe dokumen "X"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewIssuesIDAndCommits(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	mock.ExpectQuery(exactQuery("document_type_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(int64(7)))
	mock.ExpectExec(exactQuery("document_type_insert")).
		WithArgs("10007", "Dokumen Baru", "Catatan", "PNCADMIN", at).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := repo.InsertNew(context.Background(),
		daftartipedokumen.Input{Type: "Dokumen Baru", ProcessStatus: "Catatan"},
		daftartipedokumen.Editor{Identity: "PNCADMIN", At: at})
	require.NoError(t, err)
	require.Equal(t, daftartipedokumen.DocumentType{ID: "10007", Type: "Dokumen Baru", ProcessStatus: "Catatan"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewBeginError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin().WillReturnError(errBoom)

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "memulai transaksi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewSiteMissingRollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	mock.ExpectRollback()

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.EqualError(t, err, "daftartipedokumen/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewSiteQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).WillReturnError(errBoom)
	mock.ExpectRollback()

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca kode situs")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewSequenceError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	mock.ExpectQuery(exactQuery("document_type_next_sequence")).WillReturnError(errBoom)
	mock.ExpectRollback()

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "mengambil nomor urut")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewExecError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("2"))
	mock.ExpectQuery(exactQuery("document_type_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(int64(12)))
	mock.ExpectExec(exactQuery("document_type_insert")).WillReturnError(errBoom)
	mock.ExpectRollback()

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), `menyisipkan tipe dokumen "20012"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewCommitError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(exactQuery("document_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	mock.ExpectQuery(exactQuery("document_type_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(int64(1)))
	mock.ExpectExec(exactQuery("document_type_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errBoom)

	_, err := repo.InsertNew(context.Background(), daftartipedokumen.Input{}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menutup transaksi sisip")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSendsArgumentsInOrder(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(exactQuery("document_type_update")).
		WithArgs("Tipe", "Catatan", "PNCADMIN", at, "10001").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(context.Background(),
		daftartipedokumen.DocumentType{ID: "10001", Type: "Tipe", ProcessStatus: "Catatan"},
		daftartipedokumen.Editor{Identity: "PNCADMIN", At: at})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateNoRowsAffectedIsNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectExec(exactQuery("document_type_update")).WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(context.Background(), daftartipedokumen.DocumentType{ID: "X"}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, daftartipedokumen.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateExecError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectExec(exactQuery("document_type_update")).WillReturnError(errBoom)

	err := repo.Update(context.Background(), daftartipedokumen.DocumentType{ID: "X"}, daftartipedokumen.Editor{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), `memperbarui tipe dokumen "X"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Driver yang tidak dapat melaporkan jumlah baris terdampak dianggap berhasil.
func TestUpdateRowsAffectedErrorIsTreatedAsSuccess(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectExec(exactQuery("document_type_update")).
		WillReturnResult(sqlmock.NewErrorResult(errBoom))

	err := repo.Update(context.Background(), daftartipedokumen.DocumentType{ID: "X"}, daftartipedokumen.Editor{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	t.Run("readable", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactQuery("document_type_check_table")).
			WillReturnRows(sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES", "USER_EDIT", "TGL_EDIT"}))
		require.NoError(t, repo.CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("unreadable", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactQuery("document_type_check_table")).WillReturnError(errBoom)
		err := repo.CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "POOLDATA.LST_DOC_TYPE tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Pastikan sql.ErrNoRows tidak bocor dari Get ke pemanggil.
func TestGetNotFoundDoesNotLeakErrNoRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("document_type_get")).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "TYPE_DOCUMENT", "STS_PROSES"}))
	_, err := repo.Get(context.Background(), "Y")
	require.NotErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}
