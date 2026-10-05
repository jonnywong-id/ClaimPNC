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

// getColumns adalah bentuk hasil `document_object_get` — EMPAT kolom, dan yang keempat
// adalah dokumen JSON tempat pemetaan bisnis tinggal.
var getColumns = []string{"ID", "KETERANGAN", "OLD_ID", "JSON_DATA"}

func TestListMapsAndTrimsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("document_object_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "KETERANGAN", "OLD_ID"}).
			AddRow("100001 ", " KTP ", nil).
			AddRow("100002", "Polis", "0007  "))

	got, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.DocumentObject{
		{ID: "100001", Description: "KTP"},
		{ID: "100002", Description: "Polis", OldID: "0007"},
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

// Pemetaan bisnis dibongkar dari dokumen JSON, BUKAN dari kueri kedua.
//
// Bentuk dokumennya disalin apa adanya dari data produksi yang dibaca 2026-10-03.
func TestGetReadsBusinessesFromJSON(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("document_object_get")).WithArgs("100002").WillReturnRows(
		sqlmock.NewRows(getColumns).AddRow("100002", "Polis", nil,
			`{"ID":"100002","LIST_LBU_ID":[{"ID":"10027"},{"ID":"10045"}],"KET_DOC_OBJ":"Polis"}`))

	got, err := NewRepo(db).Get(context.Background(), "100002")
	require.NoError(t, err)
	require.Equal(t, daftarobjekdokumen.DocumentObject{
		ID: "100002", Description: "Polis",
		// Nama sengaja KOSONG: dokumen JSON memang hanya menyimpan ID. Pelengkapannya ada
		// di usecase, yang memang sudah membaca master bisnis.
		Businesses: []daftarobjekdokumen.Business{{ID: "10027"}, {ID: "10045"}},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Dokumen JSON yang rusak TIDAK menggagalkan pembacaan.
//
// Barisnya tetap dapat dibuka, hanya pemetaan bisnisnya yang kosong. Menggagalkan seluruh
// baris karena dokumennya cacat berarti satu baris rusak membuat layar tidak dapat dibuka
// sama sekali.
func TestGetToleratesBrokenJSON(t *testing.T) {
	for _, raw := range []string{"", "bukan json", `{"LIST_LBU_ID":"bukan senarai"}`} {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows(getColumns).AddRow("1", "a", nil, raw))

		got, err := NewRepo(db).Get(context.Background(), "1")
		require.NoError(t, err)
		require.Empty(t, got.Businesses)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestGetErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("x").
			WillReturnRows(sqlmock.NewRows(getColumns))
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
}

// expectReread menyiapkan pembacaan ulang setelah penyimpanan.
func expectReread(mock sqlmock.Sqlmock, id, description, raw string) {
	mock.ExpectQuery(q("document_object_get")).WithArgs(id).WillReturnRows(
		sqlmock.NewRows(getColumns).AddRow(id, description, nil, raw))
}

// Penambahan menerbitkan ID dari kode situs + LIMA digit urutan, lalu menyimpan dokumen
// JSON-nya — semuanya dalam satu transaksi.
func TestInsertIssuesIDAndWritesDocument(t *testing.T) {
	db, mock := newMock(t)

	const saved = `{"ID":"100007","KET_DOC_OBJ":"Polis","LIST_LBU_ID":[{"ID":"10013"}]}`

	mock.ExpectBegin()
	mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 1 "))
	mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	mock.ExpectExec(q("document_object_insert")).WithArgs("100007", "Polis", saved).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectReread(mock, "100007", "Polis", saved)

	got, err := NewRepo(db).Insert(context.Background(), daftarobjekdokumen.SaveData{
		Description: "Polis",
		Businesses: []daftarobjekdokumen.Business{
			{ID: "10013", Name: "ANEKA"},
			// ID kosong DIBUANG: dokumen JSON hanya menyimpan ID, sehingga pemetaan tanpa
			// ID tidak punya isi sama sekali.
			{ID: " ", Name: "BEBAS"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "100007", got.ID)
	require.Equal(t, []daftarobjekdokumen.Business{{ID: "10013"}}, got.Businesses)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrorsRollBack(t *testing.T) {
	data := daftarobjekdokumen.SaveData{
		Description: "Polis",
		Businesses:  []daftarobjekdokumen.Business{{ID: "10013", Name: "ANEKA"}},
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
		}, `menyisipkan "100001"`},
		{"commit error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("document_object_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
			mock.ExpectQuery(q("document_object_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			mock.ExpectExec(q("document_object_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
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

// Penyuntingan membaca dokumen LAMA lebih dulu, menyusun ulang, lalu menyimpannya.
func TestUpdateSavesAndRereads(t *testing.T) {
	db, mock := newMock(t)

	const sebelum = `{"ID":"100001","LIST_LBU_ID":[{"ID":"10002"}],"KET_DOC_OBJ":"Lama"}`
	const sesudah = `{"ID":"100001","KET_DOC_OBJ":"Baru","LIST_LBU_ID":[]}`

	mock.ExpectQuery(q("document_object_get")).WithArgs("100001").WillReturnRows(
		sqlmock.NewRows(getColumns).AddRow("100001", "Lama", nil, sebelum))
	mock.ExpectBegin()
	mock.ExpectExec(q("document_object_update")).WithArgs("Baru", sesudah, "100001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectReread(mock, "100001", "Baru", sesudah)

	got, err := NewRepo(db).Update(context.Background(), "100001", daftarobjekdokumen.SaveData{Description: "Baru"})
	require.NoError(t, err)
	require.Equal(t, "Baru", got.Description)
	require.Empty(t, got.Businesses, "pemetaan diganti seluruhnya, bukan digabung")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kunci yang TIDAK dikenal modul ini ikut terbawa saat baris disimpan ulang.
//
// Menyusun dokumen baru dari nol akan membuang apa pun yang pernah ditaruh sistem lama di
// sana — diam-diam, dan tanpa cara memulihkannya.
func TestUpdateKeepsUnknownJSONKeys(t *testing.T) {
	db, mock := newMock(t)

	const sebelum = `{"ID":"100001","LIST_LBU_ID":[],"KET_DOC_OBJ":"Lama","KUNCI_ASING":"jangan hilang"}`

	mock.ExpectQuery(q("document_object_get")).WithArgs("100001").WillReturnRows(
		sqlmock.NewRows(getColumns).AddRow("100001", "Lama", nil, sebelum))
	mock.ExpectBegin()
	mock.ExpectExec(q("document_object_update")).
		WithArgs("Baru", sqlmock.AnyArg(), "100001").
		WillReturnResult(sqlmock.NewResult(0, 1)).
		// Argumen ketiganya diperiksa lewat pencocokan terpisah supaya pesan gagalnya
		// menyebut kunci yang hilang, bukan sekadar "argumen tidak cocok".
		WillDelayFor(0)
	mock.ExpectCommit()
	expectReread(mock, "100001", "Baru", sebelum)

	_, err := NewRepo(db).Update(context.Background(), "100001", daftarobjekdokumen.SaveData{Description: "Baru"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	// Dokumen yang benar-benar disusun diperiksa langsung, bukan lewat tiruan.
	raw, err := buildDocument(parseDocument(sebelum), "100001", daftarobjekdokumen.SaveData{Description: "Baru"})
	require.NoError(t, err)
	require.Contains(t, raw, `"KUNCI_ASING":"jangan hilang"`)
	require.Contains(t, raw, `"KET_DOC_OBJ":"Baru"`)
}

func TestUpdateErrors(t *testing.T) {
	const ada = `{"ID":"1","LIST_LBU_ID":[],"KET_DOC_OBJ":"a"}`

	t.Run("baris tidak ada", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("9").
			WillReturnRows(sqlmock.NewRows(getColumns))
		_, err := NewRepo(db).Update(context.Background(), "9", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows(getColumns).AddRow("1", "a", nil, ada))
		mock.ExpectBegin()
		mock.ExpectExec(q("document_object_update")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "1", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), `memperbarui "1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("hilang di antara baca dan tulis", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_get")).WithArgs("1").WillReturnRows(
			sqlmock.NewRows(getColumns).AddRow("1", "a", nil, ada))
		mock.ExpectBegin()
		mock.ExpectExec(q("document_object_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		_, err := NewRepo(db).Update(context.Background(), "1", daftarobjekdokumen.SaveData{})
		require.ErrorIs(t, err, daftarobjekdokumen.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// CheckTable kini memeriksa SATU objek — tabel dasarnya — karena hanya itu yang disentuh
// modul ini. Dua pemeriksaan lain dibuang bersama tabel pemetaan yang tidak dipakai.
func TestCheckTable(t *testing.T) {
	t.Run("terbaca", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		require.NoError(t, NewRepo(db).CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("document_object_check_table")).WillReturnError(errBoom)
		err := NewRepo(db).CheckTable(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "POOLDATA.LST_DOC_OBJ tidak dapat dibaca")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBusinessRepoList(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"BISNISID", "BUSINESSNAME"}).AddRow("10002 ", " PA ").AddRow(nil, nil))
	got, err := NewBusinessRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.Business{{ID: "10002", Name: "PA"}, {}}, got)
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
