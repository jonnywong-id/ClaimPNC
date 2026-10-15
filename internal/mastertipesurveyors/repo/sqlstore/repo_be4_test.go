package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
)

func be4NewRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func be4Exact(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var be4Columns = []string{"M_SURVEY_ID", "DESCRIPTION", "OLD_M_SURVEY_ID"}

// TestListMapsAndTrimsRows memeriksa pemetaan baris, NULL, dan pemangkasan CHAR(4).
func TestListMapsAndTrimsRows(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("surveyor_type_list")).
		WillReturnRows(sqlmock.NewRows(be4Columns).
			AddRow("1001", "INTERNAL SURVEYOR ", nil).
			AddRow("1002", nil, "02  "))

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []mastertipesurveyors.SurveyorType{
		{Code: "1001", Description: "INTERNAL SURVEYOR"},
		{Code: "1002", LegacyCode: "02"},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListFailures memeriksa galat kueri, Scan, dan penelusuran.
func TestListFailures(t *testing.T) {
	boom := errors.New("boom")

	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("surveyor_type_list")).WillReturnError(boom)
	_, err := repo.List(context.Background())
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca daftar")

	mock.ExpectQuery(be4Exact("surveyor_type_list")).
		WillReturnRows(sqlmock.NewRows([]string{"M_SURVEY_ID"}).AddRow("1001"))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca baris")

	mock.ExpectQuery(be4Exact("surveyor_type_list")).
		WillReturnRows(sqlmock.NewRows(be4Columns).AddRow("1001", "A", nil).RowError(0, boom))
	_, err = repo.List(context.Background())
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menelusuri")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGetMapsNoRowsAndFailures memeriksa Get sukses, tidak ditemukan, dan galat.
func TestGetMapsNoRowsAndFailures(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("surveyor_type_get")).WithArgs("1003").
		WillReturnRows(sqlmock.NewRows(be4Columns).AddRow("1003", "EXPERT", nil))
	got, err := repo.Get(context.Background(), "1003")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1003", Description: "EXPERT"}, got)

	mock.ExpectQuery(be4Exact("surveyor_type_get")).WithArgs("9").
		WillReturnRows(sqlmock.NewRows(be4Columns))
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)

	boom := errors.New("putus")
	mock.ExpectQuery(be4Exact("surveyor_type_get")).WithArgs("8").WillReturnError(boom)
	_, err = repo.Get(context.Background(), "8")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca tipe surveyor")
	require.NoError(t, mock.ExpectationsWereMet())
}

// be4ExpectCode menyiapkan kedua kueri pembentuk kode di dalam transaksi.
func be4ExpectCode(mock sqlmock.Sqlmock, site string, sequence int64) {
	mock.ExpectQuery(be4Exact("surveyor_type_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(site))
	mock.ExpectQuery(be4Exact("surveyor_type_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(sequence))
}

// TestInsertIssuesCodeInsideTransaction memeriksa kode situs + tiga digit dan commit.
func TestInsertIssuesCodeInsideTransaction(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectBegin()
	be4ExpectCode(mock, "1 ", 12)
	mock.ExpectExec(be4Exact("surveyor_type_insert")).WithArgs("1012", "BARU").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	added, err := repo.Insert(context.Background(), "BARU")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1012", Description: "BARU"}, added)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestInsertFailurePaths memeriksa setiap langkah transaksi yang gagal beserta rollback.
func TestInsertFailurePaths(t *testing.T) {
	boom := errors.New("boom")
	ctx := context.Background()

	t.Run("begin", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("site missing", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery(be4Exact("surveyor_type_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		mock.ExpectRollback()
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, mastertipesurveyors.ErrNoSite)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("site error", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery(be4Exact("surveyor_type_site")).WillReturnError(boom)
		mock.ExpectRollback()
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "kode situs")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("sequence error", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin()
		mock.ExpectQuery(be4Exact("surveyor_type_site")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		mock.ExpectQuery(be4Exact("surveyor_type_next_sequence")).WillReturnError(boom)
		mock.ExpectRollback()
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "nomor urut")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert duplicate description", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin()
		be4ExpectCode(mock, "1", 5)
		mock.ExpectExec(be4Exact("surveyor_type_insert")).
			WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.ux_m_surveyors_desc) violated"))
		mock.ExpectRollback()
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock := be4NewRepo(t)
		mock.ExpectBegin()
		be4ExpectCode(mock, "1", 5)
		mock.ExpectExec(be4Exact("surveyor_type_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(boom)
		_, err := repo.Insert(ctx, "X")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menyimpan tipe surveyor baru")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestUpdateReadsBackRow memeriksa UPDATE lalu baca ulang.
func TestUpdateReadsBackRow(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectExec(be4Exact("surveyor_type_update")).WithArgs("BARU", "1001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(be4Exact("surveyor_type_get")).WithArgs("1001").
		WillReturnRows(sqlmock.NewRows(be4Columns).AddRow("1001", "BARU", "07"))

	updated, err := repo.Update(context.Background(), "1001", "BARU")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1001", Description: "BARU", LegacyCode: "07"}, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateFailurePaths memeriksa galat tulis, baris tak tersentuh, dan RowsAffected.
func TestUpdateFailurePaths(t *testing.T) {
	ctx := context.Background()

	repo, mock := be4NewRepo(t)
	mock.ExpectExec(be4Exact("surveyor_type_update")).
		WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.M_SURVEYORS_PK) violated"))
	_, err := repo.Update(ctx, "1001", "X")
	require.ErrorIs(t, err, mastertipesurveyors.ErrCodeTaken)

	boom := errors.New("lain")
	mock.ExpectExec(be4Exact("surveyor_type_update")).WillReturnError(boom)
	_, err = repo.Update(ctx, "1001", "X")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "mengubah tipe surveyor")

	mock.ExpectExec(be4Exact("surveyor_type_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(ctx, "1009", "X")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)

	mock.ExpectExec(be4Exact("surveyor_type_update")).
		WillReturnResult(sqlmock.NewErrorResult(boom))
	_, err = repo.Update(ctx, "1001", "X")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "jumlah baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckTable memeriksa jalur sukses dan gagal pemeriksaan tabel.
func TestCheckTable(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("surveyor_type_check_table")).WillReturnRows(sqlmock.NewRows(be4Columns))
	require.NoError(t, repo.CheckTable(context.Background()))

	boom := errors.New("ORA-00942")
	mock.ExpectQuery(be4Exact("surveyor_type_check_table")).WillReturnError(boom)
	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestThreeDigits meniru lpad tiga digit tanpa memotong.
func TestThreeDigits(t *testing.T) {
	require.Equal(t, "001", ThreeDigits(1))
	require.Equal(t, "999", ThreeDigits(999))
	require.Equal(t, "1000", ThreeDigits(1000))
}

// TestScanRowPassesNoRows membuktikan sql.ErrNoRows diteruskan apa adanya.
func TestScanRowPassesNoRows(t *testing.T) {
	_, err := scanRow(be4Scanner(func(...any) error { return sql.ErrNoRows }))
	require.Equal(t, sql.ErrNoRows, err)
}

type be4Scanner func(to ...any) error

func (f be4Scanner) Scan(to ...any) error { return f(to...) }
