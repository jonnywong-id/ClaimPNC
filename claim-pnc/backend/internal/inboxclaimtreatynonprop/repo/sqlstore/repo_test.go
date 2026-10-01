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

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// newMockRepo membentuk Repo di atas sqlmock yang mencocokkan kueri dengan regexp.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// pinNow mematok jam pemindai selama satu uji.
func pinNow(t *testing.T, at time.Time) {
	t.Helper()
	previous := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = previous })
}

// TestListScansRowsAndTotal memastikan kueri yang benar dipanggil dengan argumen yang benar,
// dan setiap kolom dipetakan ke isiannya.
func TestListScansRowsAndTotal(t *testing.T) {
	repo, mock := newMockRepo(t)
	pinNow(t, time.Date(2026, time.September, 28, 3, 0, 0, 0, time.UTC))

	created := time.Date(2026, time.September, 18, 3, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(resultColumns).
		AddRow("REF1", "CLMNP-1", "ADMIN", "M1", "JM1", "POL1", "2026-09-01",
			"Bisnis", "Sumber", "Ceding", "Tertanggung", created, "Pembuat", "Pengubah", 7).
		AddRow(nil, "CLMNP-2", nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil, 7)

	mock.ExpectQuery(regexp.QuoteMeta(query("list_admin"))).
		WithArgs("ADMIN", 25, 25).
		WillReturnRows(rows)

	q := sampleQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false)
	q.Caller.Login = "ADMIN"
	page, err := repo.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{Page: 2, Size: 25})
	require.NoError(t, err)

	require.Equal(t, 7, page.Total)
	require.Equal(t, inboxclaimtreatynonprop.Pagination{Page: 2, Size: 25}, page.Pagination)
	require.Len(t, page.Items, 2)
	require.Equal(t, inboxclaimtreatynonprop.WorkItem{
		Reference: "REF1", ClaimID: "CLMNP-1", AssignedOperator: "ADMIN",
		MasterID: "M1", JSONMasterID: "JM1", PolicyNumber: "POL1", LossDate: "2026-09-01",
		BusinessName: "Bisnis", BusinessSource: "Sumber", CedingCompany: "Ceding",
		InsuredName: "Tertanggung", CreatedAt: "20260918T030000.000 GMT", AgingDays: 10,
		CreateOperator: "Pembuat", LastUpdateOperator: "Pengubah",
	}, page.Items[0])
	require.Equal(t, inboxclaimtreatynonprop.WorkItem{ClaimID: "CLMNP-2"}, page.Items[1],
		"kolom NULL dari LEFT JOIN menjadi teks kosong dan umur nol")

	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListEmptyResultKeepsNonNilItems memastikan halaman kosong tetap senarai, bukan nil.
func TestListEmptyResultKeepsNonNilItems(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(query("list_technical"))).
		WithArgs(inboxclaimtreatynonprop.TechnicalWorkbasket, 0, 25).
		WillReturnRows(sqlmock.NewRows(resultColumns))

	q := sampleQuery(t, inboxclaimtreatynonprop.TabTechnical, false, false)
	page, err := repo.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{})
	require.NoError(t, err)
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Equal(t, 0, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListBlockedTabRejectedBeforeQuery memastikan tab tanpa kueri tidak menyentuh basis data.
func TestListBlockedTabRejectedBeforeQuery(t *testing.T) {
	repo, mock := newMockRepo(t)

	tab, found := inboxclaimtreatynonprop.FindTab(inboxclaimtreatynonprop.TabCommittee)
	require.True(t, found)

	_, err := repo.List(context.Background(), inboxclaimtreatynonprop.Query{Tab: tab},
		inboxclaimtreatynonprop.Pagination{})
	require.EqualError(t, err, `inboxclaimtreatynonprop/sqlstore: tab "3" belum punya kueri`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListQueryError membungkus galat kueri dengan nama kuerinya.
func TestListQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	boom := errors.New("ora-00942")

	mock.ExpectQuery(regexp.QuoteMeta(query("list_admin_all"))).
		WithArgs(0, 25).
		WillReturnError(boom)

	q := sampleQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false)
	_, err := repo.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menjalankan kueri list_admin_all")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListScanError membungkus galat pemindaian baris.
func TestListScanError(t *testing.T) {
	repo, mock := newMockRepo(t)

	// TOTAL_ROWS berisi teks yang tidak dapat dipindai ke bilangan.
	rows := sqlmock.NewRows(resultColumns).
		AddRow("R", "CLMNP-1", "A", "", "", "", "", "", "", "", "", nil, "", "", "bukan-angka")
	mock.ExpectQuery(regexp.QuoteMeta(query("list_admin_all_tba"))).
		WithArgs(0, 25).
		WillReturnRows(rows)

	q := sampleQuery(t, inboxclaimtreatynonprop.TabAdmin, true, true)
	_, err := repo.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris kueri list_admin_all_tba")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListRowsError membungkus galat penelusuran hasil.
func TestListRowsError(t *testing.T) {
	repo, mock := newMockRepo(t)
	boom := errors.New("koneksi putus")

	rows := sqlmock.NewRows(resultColumns).
		AddRow("R", "CLMNP-1", "A", "", "", "", "", "", "", "", "", nil, "", "", 1).
		RowError(0, boom)
	mock.ExpectQuery(regexp.QuoteMeta(query("list_admin_tba"))).
		WithArgs("ADMINNONPROP1", 0, 25).
		WillReturnRows(rows)

	q := sampleQuery(t, inboxclaimtreatynonprop.TabAdmin, false, true)
	_, err := repo.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menelusuri hasil kueri list_admin_tba")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckTableReadsBothQueues memastikan pemeriksaan membaca kedua kueri pemeriksa.
func TestCheckTableReadsBothQueues(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(query("check_admin"))).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(query("check_technical"))).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))

	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckTableAdminFailure menyebut tabel worklist saat kueri pertama gagal.
func TestCheckTableAdminFailure(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(query("check_admin"))).WillReturnError(sql.ErrConnDone)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "DATAPEGA.PC_ASSIGN_WORKLIST")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckTableTechnicalFailure menyebut tabel workbasket saat kueri kedua gagal.
func TestCheckTableTechnicalFailure(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta(query("check_admin"))).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(query("check_technical"))).WillReturnError(sql.ErrNoRows)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Contains(t, err.Error(), "DATAPEGA.PC_ASSIGN_WORKBASKET")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSplitByNameIgnoresCommentsAndEmptyBodies memastikan pemecah berkas .sql membuang
// komentar dan pernyataan kosong.
func TestSplitByNameIgnoresCommentsAndEmptyBodies(t *testing.T) {
	got := splitByName("-- kepala\nSELECT 0\n-- name: a\n-- penjelas\nSELECT 1\n" +
		"-- name: kosong\n-- hanya komentar\n-- name: b\nSELECT 2\n")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}
