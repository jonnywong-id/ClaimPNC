package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterkategorisparepart"
)

var errDB = errors.New("db mati")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

var columns = []string{"PART_CATEGORY_ID", "PART_CATEGORY_NAME", "APPROVAL"}

func TestHelpers(t *testing.T) {
	require.Equal(t, "", tidyNumber(""))
	require.Equal(t, "12", tidyNumber("12.0"))
	require.Equal(t, "7", tidyNumber("007"))
	require.Equal(t, "A1", tidyNumber("A1"))

	require.Equal(t, `50\%\_A\\B`, escapeLike(`50%_A\B`))

	require.Equal(t, map[string]string{"a": "SELECT 1"},
		splitByName("x\n-- name: kosong\n-- name: a\n-- c\nSELECT 1\n"))
	require.Panics(t, func() { getQuery("tidak_ada") })
}

func TestListWithAndWithoutKeyword(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("category_list")).WithArgs("1").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("1.0", " ENGINE ", " 1 ").AddRow(nil, nil, nil))
	rows, err := repo.List(context.Background(), masterkategorisparepart.Filter{Status: "1", Keyword: "  "})
	require.NoError(t, err)
	require.Equal(t, []masterkategorisparepart.PartCategory{
		{ID: "1", Name: "ENGINE", Status: "1"},
		{},
	}, rows)

	// Kata kunci dibesarkan dan karakter khusus LIKE di-escape.
	mock.ExpectQuery(q("category_list_search")).WithArgs("0", `%50\%%`).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("4", "ELECTRICAL", "0"))
	rows, err = repo.List(context.Background(), masterkategorisparepart.Filter{Status: "0", Keyword: " 50% "})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("category_list")).WillReturnError(errDB)
	_, err := repo.List(context.Background(), masterkategorisparepart.Filter{Status: "1"})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(q("category_list")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow("1"))
	_, err = repo.List(context.Background(), masterkategorisparepart.Filter{Status: "1"})
	require.ErrorContains(t, err, "membaca baris")

	mock.ExpectQuery(q("category_list")).WillReturnRows(sqlmock.NewRows(columns).AddRow("1", "a", "1").RowError(0, errDB))
	_, err = repo.List(context.Background(), masterkategorisparepart.Filter{Status: "1"})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAndFindByName(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(q("category_get")).WithArgs("3").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("3", "UNDERCARRIAGE", "1"))
	got, err := repo.Get(ctx, " 3 ")
	require.NoError(t, err)
	require.Equal(t, masterkategorisparepart.PartCategory{ID: "3", Name: "UNDERCARRIAGE", Status: "1"}, got)

	mock.ExpectQuery(q("category_get")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)

	mock.ExpectQuery(q("category_get")).WillReturnError(errDB)
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("category_find_by_name")).WithArgs("ENGINE").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("1", "ENGINE", "1"))
	got, err = repo.FindByName(ctx, " engine ")
	require.NoError(t, err)
	require.Equal(t, "1", got.ID)

	mock.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)

	mock.ExpectQuery(q("category_find_by_name")).WillReturnError(errDB)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertLocksChecksNameAndIssuesID(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("category_find_by_name")).WithArgs("BRAKE").WillReturnRows(sqlmock.NewRows(columns))
	mock.ExpectQuery(q("category_all_ids")).WillReturnRows(
		sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).AddRow("6").AddRow("2").AddRow("abc"))
	mock.ExpectExec(q("category_insert")).WithArgs("7", "BRAKE", "0").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).Insert(context.Background(),
		masterkategorisparepart.PartCategory{Name: "BRAKE", Status: masterkategorisparepart.StatusPending})
	require.NoError(t, err)
	require.Equal(t, masterkategorisparepart.PartCategory{ID: "7", Name: "BRAKE", Status: "0"}, saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(m sqlmock.Sqlmock)
		check func(t *testing.T, err error)
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) },
			func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"lock", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnError(errDB)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "mengunci tabel kategori") }},
		{"name taken", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns).AddRow("1", "BRAKE", "1"))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken) }},
		{"name check fails", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnError(errDB)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, errDB) }},
		{"next id fails", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns))
			m.ExpectQuery(q("category_all_ids")).WillReturnError(errDB)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "membaca kunci kategori") }},
		// Kunci berikutnya melampaui lebar PART_CATEGORY_ID yang VARCHAR2(10).
		//
		// Menggantikan kasus "next id not positive" yang tidak lagi mungkin: maksimum
		// numerik tidak pernah negatif, sehingga berikutnya selalu >= 1.
		{"next id too wide", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns))
			m.ExpectQuery(q("category_all_ids")).WillReturnRows(
				sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).AddRow("9999999999"))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "melebihi 10 karakter") }},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns))
			m.ExpectQuery(q("category_all_ids")).WillReturnRows(
				sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).AddRow("1"))
			m.ExpectExec(q("category_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan "BRAKE"`) }},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("category_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectQuery(q("category_find_by_name")).WillReturnRows(sqlmock.NewRows(columns))
			m.ExpectQuery(q("category_all_ids")).WillReturnRows(
				sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).AddRow("1"))
			m.ExpectExec(q("category_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.setup(mock)
			saved, err := NewRepo(db).Insert(context.Background(), masterkategorisparepart.PartCategory{Name: "BRAKE"})
			require.Error(t, err)
			tc.check(t, err)
			require.Equal(t, masterkategorisparepart.PartCategory{}, saved)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Penomoran memakai maksimum NUMERIK, bukan leksikografis.
//
// Kasusnya dipilih supaya keduanya BERBEDA: atas {"9","10","2"} maksimum leksikografisnya
// "9" — yang akan menerbitkan 10 untuk kedua kalinya, persis cacat yang ada di Pega — dan
// maksimum numeriknya 10, yang menerbitkan 11.
//
// Baris berkunci bukan angka dilewati, tidak menggagalkan penambahan.
func TestNextIDUsesNumericMaximum(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("category_all_ids")).WillReturnRows(
		sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).
			AddRow("9").AddRow("10").AddRow("2").AddRow("bukan angka").AddRow(nil))

	id, err := NewRepo(db).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "11", id,
		"maksimum leksikografis akan menghasilkan 10 — kunci yang sudah dipakai")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tabel kosong menerbitkan kunci pertama "1", sama seperti `nvl(max(...),0)+1` milik Pega.
func TestNextIDOnEmptyTable(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("category_all_ids")).
		WillReturnRows(sqlmock.NewRows([]string{"PART_CATEGORY_ID"}))

	id, err := NewRepo(db).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "1", id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextIDErrors(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(q("category_all_ids")).WillReturnError(errDB)
	_, err := NewRepo(db).NextID(context.Background())
	require.ErrorIs(t, err, errDB)

	// Baris yang gagal dibaca di tengah penelusuran tidak boleh terbaca sebagai "tabel
	// kosong"; tanpa pemeriksaan rows.Err() ia akan menerbitkan "1" atas tabel yang berisi.
	mock.ExpectQuery(q("category_all_ids")).WillReturnRows(
		sqlmock.NewRows([]string{"PART_CATEGORY_ID"}).AddRow("3").RowError(0, errDB))
	_, err = NewRepo(db).NextID(context.Background())
	require.ErrorIs(t, err, errDB)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	c := masterkategorisparepart.PartCategory{ID: " 2 ", Name: "HYD", Status: "1"}

	mock.ExpectExec(q("category_update")).WithArgs("HYD", "1", "2").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), c))

	mock.ExpectExec(q("category_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), c), masterkategorisparepart.ErrNotFound)

	mock.ExpectExec(q("category_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.NoError(t, repo.Update(context.Background(), c))

	mock.ExpectExec(q("category_update")).WillReturnError(errDB)
	require.ErrorIs(t, repo.Update(context.Background(), c), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusCountsChangedRows(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	n, err := repo.SetStatus(context.Background(), nil, "1")
	require.NoError(t, err)
	require.Zero(t, n)

	mock.ExpectBegin()
	mock.ExpectExec(q("category_set_status")).WithArgs("1", "4").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("category_set_status")).WithArgs("1", "9").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("category_set_status")).WithArgs("1", "5").WillReturnResult(sqlmock.NewErrorResult(errDB))
	mock.ExpectCommit()
	n, err = repo.SetStatus(context.Background(), []string{"4", " 9 ", "5"}, "1")
	require.NoError(t, err)
	// Baris yang jumlahnya tidak terbaca tetap dihitung berubah.
	require.Equal(t, 2, n)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectBegin().WillReturnError(errDB)
	_, err := repo.SetStatus(context.Background(), []string{"1"}, "1")
	require.ErrorContains(t, err, "memulai transaksi keputusan")

	mock.ExpectBegin()
	mock.ExpectExec(q("category_set_status")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.SetStatus(context.Background(), []string{"1"}, "1")
	require.ErrorContains(t, err, `menetapkan status "1"`)

	mock.ExpectBegin()
	mock.ExpectExec(q("category_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.SetStatus(context.Background(), []string{"1"}, "1")
	require.ErrorContains(t, err, "menutup transaksi keputusan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCounts(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()
	num := func(n int) *sqlmock.Rows { return sqlmock.NewRows([]string{"C"}).AddRow(n) }

	mock.ExpectQuery(q("category_count_by_status")).WithArgs("0").WillReturnRows(num(2))
	mock.ExpectQuery(q("category_count_all")).WillReturnRows(num(6))
	mock.ExpectQuery(q("category_count_unknown_status")).WillReturnRows(num(1))
	mock.ExpectQuery(q("category_count_duplicate_name")).WillReturnRows(num(3))
	mock.ExpectQuery(q("category_count_orphan_sparepart")).WillReturnRows(num(4))

	n, err := repo.CountByStatus(ctx, "0")
	require.NoError(t, err)
	require.Equal(t, 2, n)
	n, err = repo.CountAll(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, n)
	n, err = repo.CountUnknownStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = repo.CountDuplicateName(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	n, err = repo.CountOrphanSparepart(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	mock.ExpectQuery(q("category_count_all")).WillReturnError(errDB)
	_, err = repo.CountAll(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "category_count_all")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("category_check_table")).WillReturnRows(sqlmock.NewRows(columns))
	require.NoError(t, NewRepo(db).CheckTable(context.Background()))

	mock.ExpectQuery(q("category_check_table")).WillReturnError(errDB)
	err := NewRepo(db).CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "memeriksa tabel kategori")
	require.NoError(t, mock.ExpectationsWereMet())
}
