package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
)

var errBoom = errors.New("boom")

// sparepartColumns adalah kedua puluh empat kolom pembaca, sesuai urutan scanRow.
var sparepartColumns = []string{
	"ID", "NAMA", "NO", "KODE", "HARGA", "KATEGORI", "TIPE", "BERAT", "PANJANG", "LEBAR",
	"TINGGI", "MIN", "MAX", "ORDER", "PROD", "SUBST", "JENIS", "SATUAN", "AKTIF", "STATUS_PART",
	"USER", "TGL_UPDATE_HARGA", "DOKUMEN", "APPROVAL",
}

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regex yang cocok persis.
func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// row menyusun satu baris dengan nilai berspasi; at boleh nil.
func row(id string, at any) []driver.Value {
	values := make([]driver.Value, len(sparepartColumns))
	for i := range values {
		values[i] = " v" + sparepartColumns[i] + " "
	}
	values[0] = id + " "
	values[21] = at
	values[23] = " 1 "
	return values
}

var jakarta = time.FixedZone("WIB", 7*3600)

func TestListMapsRowsAndPriceTime(t *testing.T) {
	db, mock := newMock(t)
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, jakarta)
	mock.ExpectQuery(q("sparepart_list")).WithArgs("1").WillReturnRows(sqlmock.NewRows(sparepartColumns).
		AddRow(row("SP1", at)...).
		AddRow(row("SP2", nil)...))

	got, err := NewRepo(db).List(context.Background(), mastersparepart.Filter{Status: mastersparepart.StatusApproved})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "SP1", got[0].ID)
	require.Equal(t, "vNAMA", got[0].Name)
	require.Equal(t, "vDOKUMEN", got[0].DocumentID)
	require.Equal(t, mastersparepart.StatusApproved, got[0].Status)
	require.NotNil(t, got[0].PriceUpdatedAt)
	require.Equal(t, at.UTC(), *got[0].PriceUpdatedAt)
	require.Equal(t, time.UTC, got[0].PriceUpdatedAt.Location())
	require.Nil(t, got[1].PriceUpdatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci memakai kueri pencarian dengan pola yang sama tiga kali.
func TestListWithKeywordSendsPatternThreeTimes(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("sparepart_list_search")).WithArgs("0", `%A\%B%`, `%A\%B%`, `%A\%B%`).
		WillReturnRows(sqlmock.NewRows(sparepartColumns))
	got, err := NewRepo(db).List(context.Background(), mastersparepart.Filter{Status: "0", Keyword: " a%b "})
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	filter := mastersparepart.Filter{Status: "1"}
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("sparepart_list")).WillReturnError(errBoom)
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorContains(t, err, "membaca daftar")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("sparepart_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorContains(t, err, "membaca baris daftar")
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("sparepart_list")).WillReturnRows(
			sqlmock.NewRows(sparepartColumns).AddRow(row("1", nil)...).RowError(0, errBoom))
		_, err := NewRepo(db).List(context.Background(), filter)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar")
	})
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("sparepart_get")).WithArgs("SP1").WillReturnRows(sqlmock.NewRows(sparepartColumns).AddRow(row("SP1", nil)...))
	got, err := repo.Get(context.Background(), " SP1 ")
	require.NoError(t, err)
	require.Equal(t, "SP1", got.ID)

	mock.ExpectQuery(q("sparepart_get")).WithArgs("X").WillReturnRows(sqlmock.NewRows(sparepartColumns))
	_, err = repo.Get(context.Background(), "X")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)

	mock.ExpectQuery(q("sparepart_get")).WithArgs("Y").WillReturnError(errBoom)
	_, err = repo.Get(context.Background(), "Y")
	require.ErrorContains(t, err, `membaca "Y"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Ketiga pencarian kunci alami memakai kuerinya sendiri, huruf besar, dan menolak nilai kosong.
func TestFindByNaturalKeys(t *testing.T) {
	type finder struct {
		queryName string
		label     string
		call      func(r *Repo, ctx context.Context, v string) (mastersparepart.Sparepart, error)
	}
	for _, f := range []finder{
		{"sparepart_find_by_name", "nama", (*Repo).FindByName},
		{"sparepart_find_by_number", "nomor", (*Repo).FindByNumber},
		{"sparepart_find_by_code", "kode", (*Repo).FindByCode},
	} {
		t.Run(f.queryName, func(t *testing.T) {
			db, mock := newMock(t)
			repo := NewRepo(db)

			_, err := f.call(repo, context.Background(), "  ")
			require.ErrorIs(t, err, mastersparepart.ErrNotFound)

			mock.ExpectQuery(q(f.queryName)).WithArgs("ABC").WillReturnRows(
				sqlmock.NewRows(sparepartColumns).AddRow(row("SP9", nil)...))
			got, err := f.call(repo, context.Background(), " abc ")
			require.NoError(t, err)
			require.Equal(t, "SP9", got.ID)

			mock.ExpectQuery(q(f.queryName)).WithArgs("ABC").WillReturnRows(sqlmock.NewRows(sparepartColumns))
			_, err = f.call(repo, context.Background(), "abc")
			require.ErrorIs(t, err, mastersparepart.ErrNotFound)

			mock.ExpectQuery(q(f.queryName)).WithArgs("ABC").WillReturnError(errBoom)
			_, err = f.call(repo, context.Background(), "abc")
			require.ErrorContains(t, err, "mencari "+f.label+` "abc"`)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Acuan kategori dan tipe dibaca dengan status disetujui dan dipotong pada MaxLookupRows.
func TestLookupsAreApprovedAndCapped(t *testing.T) {
	db, mock := newMock(t)
	categories := sqlmock.NewRows([]string{"ID", "NAMA"})
	types := sqlmock.NewRows([]string{"ID", "NAMA", "KATEGORI"})
	for i := 0; i < mastersparepart.MaxLookupRows+2; i++ {
		categories.AddRow(" 1 ", " ENGINE ")
		types.AddRow(" 7 ", " FILTER ", " 1 ")
	}
	mock.ExpectQuery(q("sparepart_category_list")).WithArgs(approvedLookup).WillReturnRows(categories)
	mock.ExpectQuery(q("sparepart_type_list")).WithArgs(approvedLookup).WillReturnRows(types)

	repo := NewRepo(db)
	gotCategories, err := repo.ListCategories(context.Background())
	require.NoError(t, err)
	require.Len(t, gotCategories, mastersparepart.MaxLookupRows)
	require.Equal(t, mastersparepart.Category{ID: "1", Name: "ENGINE"}, gotCategories[0])

	gotTypes, err := repo.ListTypes(context.Background())
	require.NoError(t, err)
	require.Len(t, gotTypes, mastersparepart.MaxLookupRows)
	require.Equal(t, mastersparepart.PartType{ID: "7", Name: "FILTER", CategoryID: "1"}, gotTypes[0])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLookupErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(q("sparepart_category_list")).WillReturnError(errBoom)
	_, err := repo.ListCategories(ctx)
	require.ErrorContains(t, err, "membaca kategori")
	mock.ExpectQuery(q("sparepart_category_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.ListCategories(ctx)
	require.ErrorContains(t, err, "membaca baris kategori")
	mock.ExpectQuery(q("sparepart_category_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "a").RowError(0, errBoom))
	_, err = repo.ListCategories(ctx)
	require.ErrorContains(t, err, "menelusuri kategori")

	mock.ExpectQuery(q("sparepart_type_list")).WillReturnError(errBoom)
	_, err = repo.ListTypes(ctx)
	require.ErrorContains(t, err, "membaca tipe")
	mock.ExpectQuery(q("sparepart_type_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.ListTypes(ctx)
	require.ErrorContains(t, err, "membaca baris tipe")
	mock.ExpectQuery(q("sparepart_type_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA", "KATEGORI"}).AddRow("1", "a", "1").RowError(0, errBoom))
	_, err = repo.ListTypes(ctx)
	require.ErrorContains(t, err, "menelusuri tipe")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penambahan memeriksa ketiga kunci lalu menyisipkan dengan waktu harga UTC.
func TestInsertChecksKeysThenInserts(t *testing.T) {
	db, mock := newMock(t)
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, jakarta)
	s := mastersparepart.Sparepart{ID: "SP1", Name: "Filter", Number: "n-1", Code: "", PriceUpdatedAt: &at, Status: "0"}

	arguments := make([]driver.Value, 0, 24)
	for _, a := range insertArguments(s) {
		arguments = append(arguments, a)
	}
	require.Equal(t, at.UTC(), arguments[21])

	mock.ExpectBegin()
	mock.ExpectQuery(q("sparepart_lock_by_keys")).WithArgs("FILTER", "N-1", "").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA", "NO", "KODE"}))
	mock.ExpectExec(q("sparepart_insert")).WithArgs(arguments...).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, NewRepo(db).Insert(context.Background(), s))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tanpa kunci alami sama sekali, pemeriksaan kunci dilewati.
func TestInsertWithoutKeysSkipsCheck(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("sparepart_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, NewRepo(db).Insert(context.Background(), mastersparepart.Sparepart{ID: "SP1"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Seluruh kunci yang bentrok dilaporkan sekaligus, masing-masing sekali, menyebut ID pemiliknya;
// kunci kosong tidak dianggap bentrok.
func TestInsertReportsEveryClashingKeyOnce(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA", "NO", "KODE"}).
			AddRow(" SP7 ", "filter", "N-1", "lain").
			AddRow("SP8", "FILTER", "x", "K-1").
			AddRow("SP9", "", "", ""))
	mock.ExpectRollback()

	err := NewRepo(db).Insert(context.Background(), mastersparepart.Sparepart{
		ID: "SP1", Name: "Filter", Number: "n-1", Code: "k-1",
	})
	var validation *mastersparepart.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []mastersparepart.Violation{
		{Field: "nomor_sparepart", Message: "Nomor tersebut telah digunakan sparepart SP7. Silakan ganti dengan yang lain."},
		{Field: "nama_sparepart", Message: "Nama tersebut telah digunakan sparepart SP7. Silakan ganti dengan yang lain."},
		{Field: "kode_sparepart", Message: "Kode tersebut telah digunakan sparepart SP8. Silakan ganti dengan yang lain."},
	}, validation.Violation)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	s := mastersparepart.Sparepart{ID: "SP1", Name: "Filter"}
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		message string
	}{
		{"begin", func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errBoom) }, "memulai transaksi"},
		{"lock query", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, `memeriksa kunci "SP1"`},
		{"lock scan", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectRollback()
		}, "membaca hasil pemeriksaan"},
		{"lock rows err", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnRows(
				sqlmock.NewRows([]string{"ID", "NAMA", "NO", "KODE"}).AddRow("1", "a", "b", "c").RowError(0, errBoom))
			mock.ExpectRollback()
		}, "menelusuri pemeriksaan kunci"},
		{"insert", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA", "NO", "KODE"}))
			mock.ExpectExec(q("sparepart_insert")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, `menyisipkan "SP1"`},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_lock_by_keys")).WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA", "NO", "KODE"}))
			mock.ExpectExec(q("sparepart_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, "menutup transaksi sisip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			require.ErrorContains(t, NewRepo(db).Insert(context.Background(), s), tc.message)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	s := mastersparepart.Sparepart{ID: " SP1 ", Name: "Filter"}

	mock.ExpectExec(q("sparepart_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(context.Background(), s))

	mock.ExpectExec(q("sparepart_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(context.Background(), s), mastersparepart.ErrNotFound)

	mock.ExpectExec(q("sparepart_update")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
	require.NoError(t, repo.Update(context.Background(), s))

	mock.ExpectExec(q("sparepart_update")).WillReturnError(errBoom)
	require.ErrorContains(t, repo.Update(context.Background(), s), `memperbarui " SP1 "`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatus(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	changed, err := repo.SetStatus(context.Background(), nil, "1")
	require.NoError(t, err)
	require.Zero(t, changed)

	mock.ExpectBegin()
	mock.ExpectExec(q("sparepart_set_status")).WithArgs("1", "A").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("sparepart_set_status")).WithArgs("1", "B").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("sparepart_set_status")).WithArgs("1", "C").WillReturnResult(sqlmock.NewErrorResult(errBoom))
	mock.ExpectCommit()
	changed, err = repo.SetStatus(context.Background(), []string{"A", " B ", "C"}, "1")
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	mock.ExpectBegin().WillReturnError(errBoom)
	_, err = repo.SetStatus(context.Background(), []string{"A"}, "1")
	require.ErrorContains(t, err, "memulai transaksi keputusan")

	mock.ExpectBegin()
	mock.ExpectExec(q("sparepart_set_status")).WillReturnError(errBoom)
	mock.ExpectRollback()
	_, err = repo.SetStatus(context.Background(), []string{"A"}, "1")
	require.ErrorContains(t, err, `menetapkan status "A"`)

	mock.ExpectBegin()
	mock.ExpectExec(q("sparepart_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errBoom)
	_, err = repo.SetStatus(context.Background(), []string{"A"}, "1")
	require.ErrorContains(t, err, "menutup transaksi keputusan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextID(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("sparepart_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 01 "))
	mock.ExpectQuery(q("sparepart_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(5))
	mock.ExpectCommit()
	got, err := NewRepo(db).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastersparepart.ComposeID("01", 5, sequenceWidth), got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextIDErrors(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		message string
	}{
		{"begin", func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errBoom) }, "memulai transaksi penomoran"},
		{"no site", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			mock.ExpectRollback()
		}, "tidak punya baris CURRENT_SITE='1'"},
		{"site error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_site")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "membaca kode situs"},
		{"sequence", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("01"))
			mock.ExpectQuery(q("sparepart_next_sequence")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "mengambil nomor urut"},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("sparepart_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("01"))
			mock.ExpectQuery(q("sparepart_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, "menutup transaksi penomoran"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			_, err := NewRepo(db).NextID(context.Background())
			require.ErrorContains(t, err, tc.message)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Keempat pemeriksaan tabel menyebut tabelnya saat gagal.
func TestCheckReadable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()
	checks := []struct {
		queryName string
		table     string
		call      func() error
	}{
		{"sparepart_check_table", "POOLDATA.SPAREPART_HE", func() error { return repo.CheckTable(ctx) }},
		{"sparepart_check_category_table", "POOLDATA.GCNM_M_SPAREPART_CATEGORY", func() error { return repo.CheckCategoryTable(ctx) }},
		{"sparepart_check_type_table", "POOLDATA.GCNM_M_SPAREPART_TYPE", func() error { return repo.CheckTypeTable(ctx) }},
		{"sparepart_check_json_mirror", "POOLDATA.M_SPAREPART_HE_BU", func() error { return repo.CheckJSONMirror(ctx) }},
	}
	for _, c := range checks {
		mock.ExpectQuery(q(c.queryName)).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		require.NoError(t, c.call())
		mock.ExpectQuery(q(c.queryName)).WillReturnError(errBoom)
		require.ErrorContains(t, c.call(), c.table+" tidak dapat dibaca")
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCounters(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()
	counters := []struct {
		queryName string
		call      func() (int, error)
	}{
		{"sparepart_count_all", func() (int, error) { return repo.CountAll(ctx) }},
		{"sparepart_count_json_mirror", func() (int, error) { return repo.CountJSONMirror(ctx) }},
		{"sparepart_count_orphan_category", func() (int, error) { return repo.CountOrphanCategory(ctx) }},
		{"sparepart_count_orphan_type", func() (int, error) { return repo.CountOrphanType(ctx) }},
	}
	for i, c := range counters {
		mock.ExpectQuery(q(c.queryName)).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(i + 3))
		total, err := c.call()
		require.NoError(t, err)
		require.Equal(t, i+3, total)
		mock.ExpectQuery(q(c.queryName)).WillReturnError(errBoom)
		_, err = c.call()
		require.ErrorContains(t, err, "menjalankan "+c.queryName)
	}
	mock.ExpectQuery(q("sparepart_count_by_status")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(9))
	total, err := repo.CountByStatus(ctx, "2")
	require.NoError(t, err)
	require.Equal(t, 9, total)
	require.NoError(t, mock.ExpectationsWereMet())
}
