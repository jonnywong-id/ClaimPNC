package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftarobjekdokumen"
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

func TestListMapsAndTrimsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("document_object_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).
			AddRow("10001 ", " KTP ", nil).
			AddRow("10002", "Polis", "07"))

	got, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.DocumentObject{
		{ID: "10001", Description: "KTP"},
		{ID: "10002", Description: "Polis", OldID: "07"},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_list")).WillReturnError(errBoom)
		_, err := NewRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		// Satu kolom saja membuat Scan tiga tujuan gagal.
		mock.ExpectQuery(q("document_object_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).List(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).
				AddRow("1", "a", nil).RowError(0, errBoom))
		_, err := NewRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetReadsRowAndBusinesses(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("document_object_get")).WithArgs("10002").WillReturnRows(
		sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).AddRow("10002", "Polis", nil))
	mock.ExpectQuery(q("document_object_business_list")).WithArgs("10002").WillReturnRows(
		sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).
			AddRow("006 ", "FIRE").
			AddRow(nil, " KENDARAAN "))

	got, err := NewRepo(db).Get(context.Background(), "10002")
	require.NoError(t, err)
	require.Equal(t, daftarobjekdokumen.DocumentObject{
		ID: "10002", Description: "Polis",
		Businesses: []daftarobjekdokumen.Business{{ID: "006", Name: "FIRE"}, {ID: "", Name: "KENDARAAN"}},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("x").
			WillReturnRows(sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}))
		_, err := NewRepo(db).Get(context.Background(), "x")
		require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("x").WillReturnError(errBoom)
		_, err := NewRepo(db).Get(context.Background(), "x")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), `membaca "x"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("business query error", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).AddRow("1", "a", nil))
		mock.ExpectQuery(q("document_object_business_list")).WithArgs("1").WillReturnError(errBoom)
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca pemetaan bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("business scan error", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).AddRow("1", "a", nil))
		mock.ExpectQuery(q("document_object_business_list")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows([]string{"BISNISID"}).AddRow("1"))
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris pemetaan bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("business rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).AddRow("1", "a", nil))
		mock.ExpectQuery(q("document_object_business_list")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).AddRow("1", "a").RowError(0, errBoom))
		_, err := NewRepo(db).Get(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri pemetaan bisnis")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// expectReread menyiapkan pembacaan ulang setelah penyimpanan.
func expectReread(mock sqlmock.Sqlmock, id, description string) {
	mock.ExpectQuery(q("document_object_get")).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).AddRow(id, description, nil))
	mock.ExpectQuery(q("document_object_business_list")).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).AddRow("003", "ANEKA"))
}

// Penambahan menerbitkan ID dari kode situs + empat digit urutan, lalu memperbarui baris
// pemetaan yang sudah ada atau menyisipkan bila belum ada — semuanya dalam satu transaksi.
func TestInsertIssuesIDAndUpsertsBusinesses(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 1 "))
	mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	mock.ExpectExec(q("document_object_insert")).WithArgs("10007", "Polis").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("document_object_business_deactivate")).WithArgs("10007").WillReturnResult(sqlmock.NewResult(0, 0))
	// Baris pertama sudah ada: cukup dihidupkan kembali.
	mock.ExpectExec(q("document_object_business_activate")).
		WithArgs("003", "ANEKA", "10007", 1).WillReturnResult(sqlmock.NewResult(0, 1))
	// Baris kedua belum ada: ID kosong menjadi NULL lalu disisipkan.
	mock.ExpectExec(q("document_object_business_activate")).
		WithArgs(nil, "BEBAS", "10007", 2).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("document_object_business_insert")).
		WithArgs("10007", 2, nil, "BEBAS").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectReread(mock, "10007", "Polis")

	got, err := NewRepo(db).Insert(context.Background(), daftarobjekdokumen.SaveData{
		Description: "Polis",
		Businesses:  []daftarobjekdokumen.Business{{ID: "003", Name: "ANEKA"}, {ID: " ", Name: "BEBAS"}},
	})
	require.NoError(t, err)
	require.Equal(t, "10007", got.ID)
	require.Equal(t, []daftarobjekdokumen.Business{{ID: "003", Name: "ANEKA"}}, got.Businesses)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrorsRollBack(t *testing.T) {
	data := daftarobjekdokumen.SaveData{
		Description: "Polis",
		Businesses:  []daftarobjekdokumen.Business{{ID: "003", Name: "ANEKA"}},
	}

	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		message string
	}{
		{"begin", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin().WillReturnError(errBoom)
		}, "memulai transaksi"},
		{"no site row", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			mock.ExpectRollback()
		}, "tidak memuat baris CURRENT_SITE aktif"},
		{"site error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "membaca kode situs"},
		{"sequence error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "mengambil nomor urut"},
		{"insert error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, `menyisipkan "10001"`},
		{"deactivate error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(q("document_object_business_deactivate")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "menonaktifkan pemetaan bisnis"},
		{"activate error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(q("document_object_business_deactivate")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(q("document_object_business_activate")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "memperbarui pemetaan bisnis baris 1"},
		{"rows affected error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(q("document_object_business_deactivate")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(q("document_object_business_activate")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
			mock.ExpectRollback()
		}, "jumlah baris pemetaan bisnis tidak terbaca"},
		{"business insert error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(q("document_object_business_deactivate")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(q("document_object_business_activate")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(q("document_object_business_insert")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "menyisipkan pemetaan bisnis baris 1"},
		{"commit error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(q("document_object_business_deactivate")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec(q("document_object_business_activate")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, "menutup transaksi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			_, err := NewRepo(db).Insert(context.Background(), data)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.message)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateSavesAndRereads(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("document_object_update")).WithArgs("Baru", "10001").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("document_object_business_deactivate")).WithArgs("10001").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	expectReread(mock, "10001", "Baru")

	got, err := NewRepo(db).Update(context.Background(), "10001", daftarobjekdokumen.SaveData{Description: "Baru"})
	require.NoError(t, err)
	require.Equal(t, "Baru", got.Description)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("document_object_update")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "1", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), `memperbarui "1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no row touched", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("document_object_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "9", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("replace businesses", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("document_object_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(q("document_object_business_deactivate")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "1", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// CheckTable memeriksa ketiga objek terpisah, dan galat menyebut objek yang gagal.
func TestCheckTable(t *testing.T) {
	t.Run("all readable", func(t *testing.T) {
		db, mock := newMock(t)
		for _, name := range []string{"document_object_check_table", "document_object_write_check_table", "document_object_business_check_table"} {
			mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		}
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("second fails", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		mock.ExpectQuery(q("document_object_write_check_table")).WillReturnError(errBoom)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "POOLDATA.LST_DOC_OBJ tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBusinessRepoList(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).AddRow("002 ", " PA ").AddRow(nil, nil))
	got, err := NewBusinessRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.Business{{ID: "002", Name: "PA"}, {}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBusinessRepoListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("business_list")).WillReturnError(errBoom)
		_, err := NewBusinessRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar bisnis")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("business_list")).WillReturnRows(sqlmock.NewRows([]string{"BISNISID"}).AddRow("1"))
		_, err := NewBusinessRepo(db).List(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris bisnis")
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("business_list")).WillReturnRows(
			sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).AddRow("1", "a").RowError(0, errBoom))
		_, err := NewBusinessRepo(db).List(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar bisnis")
	})
}

func TestBusinessRepoCheckTable(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("business_check_table")).WillReturnRows(sqlmock.NewRows([]string{"BISNISID"}))
	require.NoError(t, NewBusinessRepo(db).CheckTable(context.Background()))

	mock.ExpectQuery(q("business_check_table")).WillReturnError(errBoom)
	err := NewBusinessRepo(db).CheckTable(context.Background())
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "POOLDATA.BUSINESS tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}
