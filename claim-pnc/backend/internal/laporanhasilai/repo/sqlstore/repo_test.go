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

	"claim-pnc/internal/laporanhasilai"
)

var listColumns = []string{
	"KOMITE_ID", "KOMITEKE", "NO_KLAIM", "STATUSAPPROVE", "TANGGALKOMITE",
	"OBJECTID", "COVERAGEID", "RESULTAI", "TGLAI",
}

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func sql_(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// filter memakai jam di tengah hari supaya pemangkasan ke tanggal terbukti.
var filter = laporanhasilai.Filter{
	From: time.Date(2026, time.September, 1, 13, 45, 0, 0, time.UTC),
	To:   time.Date(2026, time.September, 30, 22, 0, 0, 0, time.UTC),
}

var (
	wantFrom = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	// Batas atas eksklusif: sehari sesudah tanggal sampai.
	wantTo = time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
)

func TestListCountsThenReadsThePage(t *testing.T) {
	repo, mock := newMock(t)
	at := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(sql_("report_count")).WithArgs(wantFrom, wantTo).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(3))
	mock.ExpectQuery(sql_("report_list")).WithArgs(wantFrom, wantTo, 2, 2).
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow(" K-1 ", "1", " PNC-77 ", " 1 ", at, " O1 ", " C1 ", " DITERIMA ", at).
			AddRow("K-1", "2", "PNC-77", "2", at, "O1", "C1", "DITOLAK", nil))

	page, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Rows, 2)

	first := page.Rows[0]
	require.Equal(t, "K-1|1|O1|C1", first.ID)
	// Nomor klaim hanya tampil pada komite ke-1.
	require.Equal(t, "PNC-77", first.ClaimNumber)
	require.Equal(t, laporanhasilai.LabelAccepted, first.CommitteeStatus)
	require.Equal(t, "1", first.CommitteeStatusCode)
	require.Equal(t, at, first.CommitteeDate)
	require.Equal(t, "DITERIMA", first.AIStatus)
	require.Equal(t, at, first.AIDate)

	second := page.Rows[1]
	require.Equal(t, "K-1|2|O1|C1", second.ID)
	require.Empty(t, second.ClaimNumber)
	require.Equal(t, laporanhasilai.LabelRejected, second.CommitteeStatus)
	require.True(t, second.AIDate.IsZero())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBeyondTotalSkipsTheListQuery(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(sql_("report_count")).WithArgs(wantFrom, wantTo).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(5))

	page, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{Page: 3, Size: 50})
	require.NoError(t, err)
	require.Equal(t, 5, page.Total)
	require.Empty(t, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsCountError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("ORA-00942")
	mock.ExpectQuery(sql_("report_count")).WillReturnError(boom)

	_, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "report_count")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsListQueryError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("ORA-01013")
	mock.ExpectQuery(sql_("report_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(sql_("report_list")).WillReturnError(boom)

	_, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "membaca daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsScanError(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(sql_("report_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(sql_("report_list")).WillReturnRows(sqlmock.NewRows(listColumns).
		AddRow("K", "1", "N", "1", "bukan-tanggal", "O", "C", "DITERIMA", nil))

	_, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{})
	require.ErrorContains(t, err, "membaca baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsRowsError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("koneksi putus")
	mock.ExpectQuery(sql_("report_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(sql_("report_list")).WillReturnRows(sqlmock.NewRows(listColumns).
		AddRow("K", "1", "N", "1", nil, "O", "C", "DITERIMA", nil).RowError(0, boom))

	_, err := repo.List(context.Background(), filter, laporanhasilai.Pagination{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountAllUsesTheCountQuery(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(sql_("report_count")).WithArgs(wantFrom, wantTo).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))

	total, err := repo.CountAll(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 42, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableRunsTheProbe(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(sql_("report_check_table")).WillReturnRows(sqlmock.NewRows(listColumns))

	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableWrapsQueryError(t *testing.T) {
	repo, mock := newMock(t)
	boom := errors.New("ORA-01031")
	mock.ExpectQuery(sql_("report_check_table")).WillReturnError(boom)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "memeriksa tabel Laporan Hasil AI")
	require.NoError(t, mock.ExpectationsWereMet())
}

// noRowsScanner meniru *sql.Row yang tidak menemukan baris.
type noRowsScanner struct{}

func (noRowsScanner) Scan(...any) error { return sql.ErrNoRows }

func TestScanRowPassesNoRowsThroughUnwrapped(t *testing.T) {
	_, err := scanRow(noRowsScanner{})
	require.Equal(t, sql.ErrNoRows, err)
}
