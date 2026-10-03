package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
)

var (
	errOracle  = errors.New("oracle menolak")
	causeCols  = []string{"M_COL_ID", "COL_DESC", "OLD_M_COL_ID"}
	singleCols = []string{"X"}
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// Daftar dirapikan dan diurutkan di Go, bukan di SQL.
func TestListTrimsAndSortsRows(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("cause_of_loss_list")).WillReturnRows(sqlmock.NewRows(causeCols).
		AddRow("1010", "Golongan J ", nil).
		AddRow("1002 ", nil, " 02 "))

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpenyebabkerugian.CauseOfLoss{
		{ID: "1002", LegacyID: "02"},
		{ID: "1010", Description: "Golongan J"},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("cause_of_loss_list")).WillReturnError(errOracle)
	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "membaca daftar penyebab kerugian")

	mock.ExpectQuery(q("cause_of_loss_list")).
		WillReturnRows(sqlmock.NewRows(singleCols).AddRow("x"))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca baris penyebab kerugian")

	mock.ExpectQuery(q("cause_of_loss_list")).
		WillReturnRows(sqlmock.NewRows(causeCols).AddRow("1", "a", nil).RowError(0, errOracle))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "menelusuri daftar penyebab kerugian")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("cause_of_loss_get")).WithArgs("1001").
		WillReturnRows(sqlmock.NewRows(causeCols).AddRow("1001", "A", "01"))
	cause, err := repo.Get(ctx, "1001")
	require.NoError(t, err)
	require.Equal(t, masterpenyebabkerugian.CauseOfLoss{
		ID: "1001", Description: "A", LegacyID: "01"}, cause)

	mock.ExpectQuery(q("cause_of_loss_get")).WillReturnError(sql.ErrNoRows)
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)

	mock.ExpectQuery(q("cause_of_loss_get")).WillReturnError(errOracle)
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "membaca penyebab kerugian")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Insert membentuk ID dari situs + urutan tiga digit di dalam satu transaksi.
func TestInsertIssuesTheIDInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("cause_of_loss_site")).
		WillReturnRows(sqlmock.NewRows(singleCols).AddRow(" 1 "))
	mock.ExpectQuery(q("cause_of_loss_next_sequence")).
		WillReturnRows(sqlmock.NewRows(singleCols).AddRow(12))
	mock.ExpectExec(q("cause_of_loss_insert")).WithArgs("1012", "Golongan").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	cause, err := repo.Insert(context.Background(), "Golongan")
	require.NoError(t, err)
	require.Equal(t, masterpenyebabkerugian.CauseOfLoss{ID: "1012", Description: "Golongan"}, cause)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	ctx := context.Background()

	expectID := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(q("cause_of_loss_site")).WillReturnRows(sqlmock.NewRows(singleCols).AddRow("1"))
		m.ExpectQuery(q("cause_of_loss_next_sequence")).
			WillReturnRows(sqlmock.NewRows(singleCols).AddRow(1))
	}

	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"mulai transaksi", func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"tanpa situs", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).WillReturnError(sql.ErrNoRows)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterpenyebabkerugian.ErrNoSite) }},
		{"situs gagal", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "membaca kode situs") }},
		{"urutan gagal", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).WillReturnRows(sqlmock.NewRows(singleCols).AddRow("1"))
			m.ExpectQuery(q("cause_of_loss_next_sequence")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "mengambil nomor urut") }},
		{"kunci utama bentrok", func(m sqlmock.Sqlmock) {
			expectID(m)
			m.ExpectExec(q("cause_of_loss_insert")).
				WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.m_cause_of_loss_pk) violated"))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterpenyebabkerugian.ErrIDTaken) }},
		{"sisip gagal", func(m sqlmock.Sqlmock) {
			expectID(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) {
			require.ErrorIs(t, err, errOracle)
			require.ErrorContains(t, err, "menyisipkan penyebab kerugian")
		}},
		{"commit gagal", func(m sqlmock.Sqlmock) {
			expectID(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) {
			require.ErrorContains(t, err, "menyimpan penyebab kerugian baru")
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			_, err := repo.Insert(ctx, "x")
			c.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Update memeriksa baris yang tersentuh lalu membaca ulang barisnya.
func TestUpdateChecksTouchedRowsAndRereads(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectExec(q("cause_of_loss_update")).WithArgs("Baru", "1001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("cause_of_loss_get")).WithArgs("1001").
		WillReturnRows(sqlmock.NewRows(causeCols).AddRow("1001", "Baru", "01"))
	cause, err := repo.Update(ctx, "1001", "Baru")
	require.NoError(t, err)
	require.Equal(t, "01", cause.LegacyID)

	mock.ExpectExec(q("cause_of_loss_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(ctx, "9", "x")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)

	mock.ExpectExec(q("cause_of_loss_update")).WillReturnError(errOracle)
	_, err = repo.Update(ctx, "9", "x")
	require.ErrorContains(t, err, "mengubah penyebab kerugian")

	mock.ExpectExec(q("cause_of_loss_update")).WillReturnResult(sqlmock.NewErrorResult(errOracle))
	_, err = repo.Update(ctx, "9", "x")
	require.ErrorContains(t, err, "membaca jumlah baris terubah")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan kesiapan tabel dan pencacah JSON yang belum dipindahkan.
func TestCheckTableAndCountPendingJSON(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("cause_of_loss_check_table")).WillReturnRows(sqlmock.NewRows(causeCols))
	require.NoError(t, repo.CheckTable(ctx))

	mock.ExpectQuery(q("cause_of_loss_check_table")).WillReturnError(errOracle)
	require.ErrorIs(t, repo.CheckTable(ctx), errOracle)

	mock.ExpectQuery(q("cause_of_loss_count_pending_json")).
		WillReturnRows(sqlmock.NewRows(singleCols).AddRow(3))
	total, err := repo.CountPendingJSON(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, total)

	mock.ExpectQuery(q("cause_of_loss_count_pending_json")).WillReturnError(errOracle)
	_, err = repo.CountPendingJSON(ctx)
	require.ErrorIs(t, err, errOracle)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestThreeDigitsNeverTruncates(t *testing.T) {
	require.Equal(t, "007", ThreeDigits(7))
	require.Equal(t, "1000", ThreeDigits(1000))
}

func TestUnknownQueryNamePanics(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
