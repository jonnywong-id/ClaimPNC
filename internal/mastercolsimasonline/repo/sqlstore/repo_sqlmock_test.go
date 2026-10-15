package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
)

var (
	errOracle = errors.New("oracle menolak")
	colCols   = []string{"M_COL_ID", "COL_DESC"}
	bizCols   = []string{"ID", "NOTE"}
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// expectGet menyiapkan pembacaan ulang satu baris beserta pemetaannya.
func expectGet(mock sqlmock.Sqlmock, code, description string, businesses ...[2]any) {
	mock.ExpectQuery(q("cause_of_loss_get")).WithArgs(code).
		WillReturnRows(sqlmock.NewRows(colCols).AddRow(code, description))
	rows := sqlmock.NewRows(bizCols)
	for _, b := range businesses {
		rows.AddRow(b[0], b[1])
	}
	mock.ExpectQuery(q("cause_of_loss_business_list")).WithArgs(code).WillReturnRows(rows)
}

func expectCode(m sqlmock.Sqlmock) {
	m.ExpectBegin()
	m.ExpectQuery(q("cause_of_loss_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	m.ExpectQuery(q("cause_of_loss_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
}

func TestListMapsAndTrimsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("cause_of_loss_list")).
		WillReturnRows(sqlmock.NewRows(colCols).AddRow("1001 ", " KEBAKARAN ").AddRow(nil, nil))

	list, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []mastercolsimasonline.CauseOfLoss{
		{Code: "1001", Description: "KEBAKARAN"}, {},
	}, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("cause_of_loss_list")).WillReturnError(errOracle)
	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(q("cause_of_loss_list")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca baris")

	mock.ExpectQuery(q("cause_of_loss_list")).
		WillReturnRows(sqlmock.NewRows(colCols).AddRow("1", "x").RowError(0, errOracle))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Get membaca baris lalu pemetaan bisnisnya, termasuk nama tanpa ID.
func TestGetReadsTheRowAndItsBusinesses(t *testing.T) {
	db, mock := newMock(t)
	expectGet(mock, "1001", "KEBAKARAN", [2]any{" 006 ", "FIRE"}, [2]any{nil, "BEBAS "})

	row, err := NewRepo(db).Get(context.Background(), "1001")
	require.NoError(t, err)
	require.Equal(t, mastercolsimasonline.CauseOfLoss{
		Code: "1001", Description: "KEBAKARAN",
		Businesses: []mastercolsimasonline.Business{{ID: "006", Name: "FIRE"}, {Name: "BEBAS"}},
	}, row)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("cause_of_loss_get")).WillReturnError(sql.ErrNoRows)
	_, err := repo.Get(ctx, "1")
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)

	mock.ExpectQuery(q("cause_of_loss_get")).WillReturnError(errOracle)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, `membaca "1"`)

	found := func() {
		mock.ExpectQuery(q("cause_of_loss_get")).
			WillReturnRows(sqlmock.NewRows(colCols).AddRow("1", "x"))
	}

	found()
	mock.ExpectQuery(q("cause_of_loss_business_list")).WillReturnError(errOracle)
	_, err = repo.Get(ctx, "1")
	require.ErrorContains(t, err, "membaca pemetaan bisnis")

	found()
	mock.ExpectQuery(q("cause_of_loss_business_list")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.Get(ctx, "1")
	require.ErrorContains(t, err, "membaca baris pemetaan bisnis")

	found()
	mock.ExpectQuery(q("cause_of_loss_business_list")).
		WillReturnRows(sqlmock.NewRows(bizCols).AddRow("a", "b").RowError(0, errOracle))
	_, err = repo.Get(ctx, "1")
	require.ErrorContains(t, err, "menelusuri pemetaan bisnis")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Insert menerbitkan kode dari situs + urutan, menulis deskripsi dua kali, menyimpan
// pemetaan (perbarui lebih dulu, sisip bila tidak ada), lalu membaca ulang.
func TestInsertIssuesTheCodeAndSavesInOneTransaction(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("cause_of_loss_site")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 1 "))
	mock.ExpectQuery(q("cause_of_loss_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	mock.ExpectExec(q("cause_of_loss_insert")).WithArgs("1007", "BANJIR", "BANJIR").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("cause_of_loss_business_update")).WithArgs("006", "1007", "FIRE").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("cause_of_loss_business_update")).WithArgs(nil, "1007", "BEBAS").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("cause_of_loss_business_insert")).WithArgs("1007", nil, "BEBAS").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectGet(mock, "1007", "BANJIR", [2]any{"006", "FIRE"}, [2]any{nil, "BEBAS"})

	saved, err := NewRepo(db).Insert(context.Background(), mastercolsimasonline.SaveData{
		Description: "BANJIR",
		Businesses: []mastercolsimasonline.Business{
			{ID: "006", Name: "FIRE"}, {ID: " ", Name: "BEBAS"}},
	})
	require.NoError(t, err)
	require.Equal(t, "1007", saved.Code)
	require.Len(t, saved.Businesses, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	ctx := context.Background()
	data := mastercolsimasonline.SaveData{
		Description: "X", Businesses: []mastercolsimasonline.Business{{Name: "B"}}}

	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		message string
	}{
		{"mulai transaksi", func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, "memulai transaksi"},
		{"situs tidak ada", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).WillReturnError(sql.ErrNoRows)
			m.ExpectRollback()
		}, "tidak memuat baris CURRENT_SITE aktif"},
		{"situs gagal", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, "membaca kode situs"},
		{"urutan gagal", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("cause_of_loss_site")).
				WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			m.ExpectQuery(q("cause_of_loss_next_sequence")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, "mengambil nomor urut"},
		{"sisip gagal", func(m sqlmock.Sqlmock) {
			expectCode(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, `menyisipkan "1001"`},
		{"pemetaan gagal", func(m sqlmock.Sqlmock) {
			expectCode(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("cause_of_loss_business_update")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, `memperbarui pemetaan bisnis "B"`},
		{"jumlah baris tak terbaca", func(m sqlmock.Sqlmock) {
			expectCode(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("cause_of_loss_business_update")).
				WillReturnResult(sqlmock.NewErrorResult(errOracle))
			m.ExpectRollback()
		}, "jumlah baris pemetaan bisnis tidak terbaca"},
		{"sisip pemetaan gagal", func(m sqlmock.Sqlmock) {
			expectCode(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("cause_of_loss_business_update")).
				WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectExec(q("cause_of_loss_business_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, `menyisipkan pemetaan bisnis "B"`},
		{"commit gagal", func(m sqlmock.Sqlmock) {
			expectCode(m)
			m.ExpectExec(q("cause_of_loss_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("cause_of_loss_business_update")).
				WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, "menutup transaksi"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.prepare(mock)
			_, err := NewRepo(db).Insert(ctx, data)
			require.ErrorContains(t, err, c.message)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Update menulis deskripsi dua kali, menyimpan pemetaan, lalu membaca ulang.
func TestUpdateSavesAndRereads(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(q("cause_of_loss_update")).WithArgs("BARU", "BARU", "1001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("cause_of_loss_business_update")).WithArgs("003", "1001", "ANEKA").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectGet(mock, "1001", "BARU", [2]any{"003", "ANEKA"})

	saved, err := NewRepo(db).Update(context.Background(), "1001", mastercolsimasonline.SaveData{
		Description: "BARU",
		Businesses:  []mastercolsimasonline.Business{{ID: "003", Name: "ANEKA"}},
	})
	require.NoError(t, err)
	require.Equal(t, "BARU", saved.Description)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("cause_of_loss_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()
	_, err := NewRepo(db).Update(ctx, "9", mastercolsimasonline.SaveData{})
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("cause_of_loss_update")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = NewRepo(db).Update(ctx, "9", mastercolsimasonline.SaveData{})
	require.ErrorContains(t, err, `memperbarui "9"`)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("cause_of_loss_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("cause_of_loss_business_update")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = NewRepo(db).Update(ctx, "9", mastercolsimasonline.SaveData{
		Businesses: []mastercolsimasonline.Business{{Name: "B"}}})
	require.ErrorContains(t, err, "memperbarui pemetaan bisnis")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan tabel menyentuh kedua tabel dan menyebut tabel yang gagal.
func TestCheckTable(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	mock.ExpectQuery(q("cause_of_loss_check_table")).WillReturnRows(sqlmock.NewRows(colCols))
	mock.ExpectQuery(q("cause_of_loss_business_check_table")).
		WillReturnRows(sqlmock.NewRows(bizCols))
	require.NoError(t, NewRepo(db).CheckTable(ctx))
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("cause_of_loss_check_table")).WillReturnRows(sqlmock.NewRows(colCols))
	mock.ExpectQuery(q("cause_of_loss_business_check_table")).WillReturnError(errOracle)
	err := NewRepo(db).CheckTable(ctx)
	require.ErrorContains(t, err, "POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Master bisnis milik GISFW dibaca dan diperiksa.
func TestBusinessRepo(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	repo := NewBusinessRepo(db)
	mock.ExpectQuery(q("business_list")).
		WillReturnRows(sqlmock.NewRows(bizCols).AddRow(" 003 ", " ANEKA ").AddRow(nil, nil))
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Equal(t, []mastercolsimasonline.Business{{ID: "003", Name: "ANEKA"}, {}}, list)

	mock.ExpectQuery(q("business_list")).WillReturnError(errOracle)
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca daftar bisnis")

	mock.ExpectQuery(q("business_list")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca baris bisnis")

	mock.ExpectQuery(q("business_list")).
		WillReturnRows(sqlmock.NewRows(bizCols).AddRow("a", "b").RowError(0, errOracle))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "menelusuri daftar bisnis")

	mock.ExpectQuery(q("business_check_table")).WillReturnRows(sqlmock.NewRows(bizCols))
	require.NoError(t, repo.CheckTable(ctx))

	mock.ExpectQuery(q("business_check_table")).WillReturnError(errOracle)
	require.ErrorContains(t, repo.CheckTable(ctx), "POOLDATA.BUSINESS tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnknownQueryNamePanics(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
