package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesparepart"
)

var errBoom = errors.New("boom")

var (
	// SATU bentuk baris untuk keempat kueri pembaca. Sejak JOIN-nya dicabut (2026-10-04),
	// type_list, type_list_search, type_get, dan type_find_by_name mengembalikan kolom yang
	// sama persis — dan scanRow pun tinggal satu.
	rowColumns = []string{"ID", "NAMA", "KATEGORI_ID", "APPROVAL"}
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regex yang cocok persis.
func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// Tanpa kata kunci dipakai type_list; kunci berbentuk "8.0" dan "08" dirapikan menjadi "8".
func TestListWithoutKeywordTidiesKeys(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("type_list")).WithArgs("1").WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow("8.0", " Body ", "08", " 1 ").
		AddRow("1E3", "X", "abc", "1"))

	got, err := NewRepo(db).List(context.Background(), mastertipesparepart.Filter{Status: mastertipesparepart.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, []mastertipesparepart.PartType{
		{ID: "8", Name: "Body", CategoryID: "8", Status: mastertipesparepart.StatusApproved},
		{ID: "1E3", Name: "X", CategoryID: "abc", Status: mastertipesparepart.StatusApproved},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci diubah ke huruf besar dan karakter pola LIKE diloloskan.
func TestListWithKeywordEscapesPattern(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("type_list_search")).WithArgs("0", `%A\%\_\\B%`).WillReturnRows(sqlmock.NewRows(rowColumns))

	got, err := NewRepo(db).List(context.Background(), mastertipesparepart.Filter{
		Status: mastertipesparepart.StatusPending, Keyword: ` a%_\b `,
	})
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	filter := mastertipesparepart.Filter{Status: mastertipesparepart.StatusApproved}
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("type_list")).WillReturnError(errBoom)
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("type_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorContains(t, err, "membaca baris")
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("type_list")).WillReturnRows(
			sqlmock.NewRows(rowColumns).AddRow("1", "a", "1", "1").RowError(0, errBoom))
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar")
	})
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("type_get")).WithArgs("8").WillReturnRows(
		sqlmock.NewRows(rowColumns).AddRow("8", "Body", "3", "0"))
	got, err := repo.Get(context.Background(), " 8 ")
	require.NoError(t, err)
	require.Equal(t, "3", got.CategoryID)

	mock.ExpectQuery(q("type_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(rowColumns))
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)

	mock.ExpectQuery(q("type_get")).WithArgs("10").WillReturnError(errBoom)
	_, err = repo.Get(context.Background(), "10")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByName(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("type_find_by_name")).WithArgs("BODY").WillReturnRows(
		sqlmock.NewRows(rowColumns).AddRow("8.0", "Body", "3", "1"))
	got, err := repo.FindByName(context.Background(), " body ")
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.PartType{ID: "8", Name: "Body", CategoryID: "3", Status: "1"}, got)

	mock.ExpectQuery(q("type_find_by_name")).WithArgs("X").WillReturnRows(sqlmock.NewRows(rowColumns))
	_, err = repo.FindByName(context.Background(), "x")
	require.ErrorIs(t, err, mastertipesparepart.ErrNotFound)

	mock.ExpectQuery(q("type_find_by_name")).WithArgs("Y").WillReturnError(errBoom)
	_, err = repo.FindByName(context.Background(), "y")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kategori dibaca dengan status disetujui dan dibatasi MaxLookupRows.
func TestListCategories(t *testing.T) {
	db, mock := newMock(t)
	rows := sqlmock.NewRows([]string{"ID", "NAMA"})
	for i := 0; i < mastertipesparepart.MaxLookupRows+3; i++ {
		rows.AddRow("3.0", " Eksterior ")
	}
	mock.ExpectQuery(q("type_category_list")).WithArgs(string(mastertipesparepart.StatusApproved)).WillReturnRows(rows)

	got, err := NewRepo(db).ListCategories(context.Background())
	require.NoError(t, err)
	require.Len(t, got, mastertipesparepart.MaxLookupRows)
	require.Equal(t, mastertipesparepart.Category{ID: "3", Name: "Eksterior"}, got[0])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListCategoriesErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("type_category_list")).WillReturnError(errBoom)
	_, err := repo.ListCategories(context.Background())
	require.ErrorContains(t, err, "membaca kategori")

	mock.ExpectQuery(q("type_category_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.ListCategories(context.Background())
	require.ErrorContains(t, err, "membaca baris kategori")

	mock.ExpectQuery(q("type_category_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "a").RowError(0, errBoom))
	_, err = repo.ListCategories(context.Background())
	require.ErrorContains(t, err, "menelusuri kategori")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penambahan: kunci tabel, periksa nama, terbitkan ID, sisipkan — dalam satu transaksi.
func TestInsertLocksChecksIssuesAndInserts(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("type_find_by_name")).WithArgs("BODY").WillReturnRows(sqlmock.NewRows(rowColumns))
	mock.ExpectQuery(q("type_next_id")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(12))
	mock.ExpectExec(q("type_insert")).WithArgs("12", "Body", "3", "0").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := NewRepo(db).Insert(context.Background(), mastertipesparepart.PartType{
		ID: "abaikan", Name: "Body", CategoryID: "3", Status: mastertipesparepart.StatusPending,
	})
	require.NoError(t, err)
	require.Equal(t, mastertipesparepart.PartType{ID: "12", Name: "Body", CategoryID: "3", Status: "0"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		check   func(t *testing.T, err error)
	}{
		{"begin", func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errBoom) },
			func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"lock", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "mengunci tabel tipe") }},
		{"name taken", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnRows(sqlmock.NewRows(rowColumns).AddRow("1", "Body", "3", "1"))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, mastertipesparepart.ErrNameTaken) }},
		{"name check error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, errBoom) }},
		{"next id error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnRows(sqlmock.NewRows(rowColumns))
			mock.ExpectQuery(q("type_next_id")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menerbitkan ID tipe") }},
		{"next id not positive", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnRows(sqlmock.NewRows(rowColumns))
			mock.ExpectQuery(q("type_next_id")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "tidak masuk akal: 0") }},
		{"insert", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnRows(sqlmock.NewRows(rowColumns))
			mock.ExpectQuery(q("type_next_id")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("type_insert")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan "Body"`) }},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(q("type_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("type_find_by_name")).WillReturnRows(sqlmock.NewRows(rowColumns))
			mock.ExpectQuery(q("type_next_id")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("type_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			_, err := NewRepo(db).Insert(context.Background(), mastertipesparepart.PartType{Name: "Body"})
			tc.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// NextID di luar transaksi tidak menolak nilai nol — ia hanya dipakai pemeriksaan.
func TestNextID(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("type_next_id")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))
	got, err := NewRepo(db).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "42", got)

	mock.ExpectQuery(q("type_next_id")).WillReturnError(errBoom)
	_, err = NewRepo(db).NextID(context.Background())
	require.ErrorContains(t, err, "menerbitkan ID tipe")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	part := mastertipesparepart.PartType{ID: " 8 ", Name: "Body", CategoryID: "3", Status: "0"}

	mock.ExpectExec(q("type_update")).WithArgs("Body", "3", "0", "8").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), part))

	mock.ExpectExec(q("type_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), part), mastertipesparepart.ErrNotFound)

	// Driver tanpa RowsAffected tidak dianggap gagal.
	mock.ExpectExec(q("type_update")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
	require.NoError(t, repo.Update(context.Background(), part))

	mock.ExpectExec(q("type_update")).WillReturnError(errBoom)
	require.ErrorContains(t, repo.Update(context.Background(), part), `memperbarui " 8 "`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// SetStatus mencacah baris yang berubah; baris tanpa jumlah terpengaruh tetap dihitung.
func TestSetStatusCountsChangedRows(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	count, err := repo.SetStatus(context.Background(), nil, mastertipesparepart.StatusApproved)
	require.NoError(t, err)
	require.Zero(t, count)

	mock.ExpectBegin()
	mock.ExpectExec(q("type_set_status")).WithArgs("1", "1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("type_set_status")).WithArgs("1", "2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("type_set_status")).WithArgs("1", "3").WillReturnResult(sqlmock.NewErrorResult(errBoom))
	mock.ExpectCommit()
	count, err = repo.SetStatus(context.Background(), []string{"1", " 2 ", "3"}, mastertipesparepart.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusErrors(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := NewRepo(db).SetStatus(context.Background(), []string{"1"}, "1")
		require.ErrorContains(t, err, "memulai transaksi keputusan")
	})
	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("type_set_status")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).SetStatus(context.Background(), []string{"1"}, "1")
		require.ErrorContains(t, err, `menetapkan status "1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("type_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := NewRepo(db).SetStatus(context.Background(), []string{"1"}, "1")
		require.ErrorContains(t, err, "menutup transaksi keputusan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Keenam pencacah memakai kuerinya sendiri, dan galat menyebut nama kuerinya.
func TestCounters(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()

	counters := []struct {
		queryName string
		call      func() (int, error)
	}{
		{"type_count_all", func() (int, error) { return repo.CountAll(ctx) }},
		{"type_count_unknown_status", func() (int, error) { return repo.CountUnknownStatus(ctx) }},
		{"type_count_duplicate_name", func() (int, error) { return repo.CountDuplicateName(ctx) }},
		{"type_count_orphan_category", func() (int, error) { return repo.CountOrphanCategory(ctx) }},
		{"type_count_orphan_sparepart", func() (int, error) { return repo.CountOrphanSparepart(ctx) }},
	}
	for i, c := range counters {
		mock.ExpectQuery(q(c.queryName)).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(i + 1))
		total, err := c.call()
		require.NoError(t, err)
		require.Equal(t, i+1, total)

		mock.ExpectQuery(q(c.queryName)).WillReturnError(errBoom)
		_, err = c.call()
		require.ErrorContains(t, err, c.queryName)
	}

	mock.ExpectQuery(q("type_count_by_status")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	total, err := repo.CountByStatus(ctx, mastertipesparepart.StatusRejected)
	require.NoError(t, err)
	require.Equal(t, 7, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("type_check_table")).WillReturnRows(sqlmock.NewRows(rowColumns))
	require.NoError(t, NewRepo(db).CheckTable(context.Background()))

	mock.ExpectQuery(q("type_check_table")).WillReturnError(errBoom)
	require.ErrorContains(t, NewRepo(db).CheckTable(context.Background()), "memeriksa tabel tipe")
	require.NoError(t, mock.ExpectationsWereMet())
}

// tidyNumber hanya merapikan bentuk yang pasti bilangan bulat yang sama.
func TestTidyNumber(t *testing.T) {
	require.Equal(t, "", tidyNumber(""))
	require.Equal(t, "8", tidyNumber("8.0"))
	require.Equal(t, "8", tidyNumber("08"))
	require.Equal(t, "8.5", tidyNumber("8.5"))
	require.Equal(t, "1E3", tidyNumber("1E3"))
}
