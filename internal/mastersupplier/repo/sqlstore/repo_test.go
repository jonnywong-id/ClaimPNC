package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
)

var errBoom = errors.New("boom")

// supplierColumns adalah kedua puluh sembilan kolom pembaca, sesuai urutan scanRow.
var supplierColumns = []string{
	"ID", "OLD_ID", "NAMA", "ALAMAT", "KOTA", "NAMA_CABANG", "KODE_POS", "NEGARA", "TELEPON", "FAX",
	"EMAIL", "NPWP", "CONTACT_PERSON", "STS_REKANAN", "JENIS_STATUS", "SUPPLIER_HE", "TOP", "TOD",
	"KETERANGAN", "BANK", "ACCOUNT_NO", "ACCOUNT_NAME", "BANK_BRANCH", "JENIS_SUPPLIER",
	"STS_AKTIF_PROMLIST", "STS_AKTIF", "STS_AUTOPAYMENT", "USERKLAIMID", "TGL_INSERT",
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

// supplierRow menyusun satu baris berisi nilai dengan spasi, untuk membuktikan pemangkasan.
func supplierRow(id, name, heavyEquipment string) []driver.Value {
	values := make([]driver.Value, len(supplierColumns))
	for i := range values {
		values[i] = " v" + supplierColumns[i] + " "
	}
	values[0] = id + "  "
	values[1] = nil
	values[2] = name
	values[14] = "9" // JENIS_STATUS tersimpan diabaikan, diturunkan dari SUPPLIER_HE
	values[15] = heavyEquipment
	return values
}

func TestListWithoutKeywordMapsRows(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("supplier_list")).WillReturnRows(
		sqlmock.NewRows(supplierColumns).AddRow(supplierRow("01A", "Satu", "1")...))

	got, err := NewRepo(db).List(context.Background(), mastersupplier.Filter{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "01A", got[0].ID)
	require.Empty(t, got[0].OldID)
	require.Equal(t, "vALAMAT", got[0].Address)
	require.Equal(t, "vTGL_INSERT", got[0].UpdatedAt)
	require.Equal(t, "1", got[0].HeavyEquipment)
	require.Equal(t, mastersupplier.DeriveSupplyType("1"), got[0].SupplyType)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci memakai kueri pencarian dengan pola LIKE yang sudah diloloskan.
func TestListWithKeywordEscapesLikePattern(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("supplier_list_search")).WithArgs(`%50\%\_A\\B%`).
		WillReturnRows(sqlmock.NewRows(supplierColumns))

	got, err := NewRepo(db).List(context.Background(), mastersupplier.Filter{Keyword: ` 50%_a\b `})
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("supplier_list")).WillReturnError(errBoom)
		_, err := NewRepo(db).List(context.Background(), mastersupplier.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar")
	})
	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("supplier_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := NewRepo(db).List(context.Background(), mastersupplier.Filter{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris daftar")
	})
	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("supplier_list")).WillReturnRows(
			sqlmock.NewRows(supplierColumns).AddRow(supplierRow("1", "a", "0")...).RowError(0, errBoom))
		_, err := NewRepo(db).List(context.Background(), mastersupplier.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar")
	})
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("supplier_get")).WithArgs("01A").WillReturnRows(
		sqlmock.NewRows(supplierColumns).AddRow(supplierRow("01A", "Satu", "0")...))
	got, err := NewRepo(db).Get(context.Background(), " 01A ")
	require.NoError(t, err)
	require.Equal(t, "Satu", got.Name)

	mock.ExpectQuery(q("supplier_get")).WithArgs("X").WillReturnRows(sqlmock.NewRows(supplierColumns))
	_, err = NewRepo(db).Get(context.Background(), "X")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)

	mock.ExpectQuery(q("supplier_get")).WithArgs("Y").WillReturnError(errBoom)
	_, err = NewRepo(db).Get(context.Background(), "Y")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), `membaca "Y"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nama dicari dalam huruf besar; nama kosong tidak menyentuh basis data.
func TestFindByName(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	_, err := repo.FindByName(context.Background(), "  ")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)

	mock.ExpectQuery(q("supplier_find_by_name")).WithArgs("SATU").WillReturnRows(
		sqlmock.NewRows(supplierColumns).AddRow(supplierRow("01A", "Satu", "0")...))
	got, err := repo.FindByName(context.Background(), " satu ")
	require.NoError(t, err)
	require.Equal(t, "01A", got.ID)

	mock.ExpectQuery(q("supplier_find_by_name")).WithArgs("DUA").WillReturnRows(sqlmock.NewRows(supplierColumns))
	_, err = repo.FindByName(context.Background(), "dua")
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)

	mock.ExpectQuery(q("supplier_find_by_name")).WithArgs("TIGA").WillReturnError(errBoom)
	_, err = repo.FindByName(context.Background(), "tiga")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), `mencari nama "tiga"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// decodedDocument mencocokkan argumen dokumen JSON menurut isinya, bukan urutan kuncinya.
type decodedDocument map[string]any

func (d decodedDocument) Match(v driver.Value) bool {
	text, ok := v.(string)
	if !ok {
		return false
	}
	var got map[string]any
	if json.Unmarshal([]byte(text), &got) != nil {
		return false
	}
	for key, want := range d {
		if got[key] != want {
			return false
		}
	}
	return true
}

func TestInsertLocksNameThenInserts(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("supplier_lock_by_name")).WithArgs("BARU").WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	mock.ExpectExec(q("supplier_insert")).
		WithArgs("01B", decodedDocument{"ID": "01B ", "NAMA": "Baru", "KOTA": ""}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := NewRepo(db).Insert(context.Background(), mastersupplier.Supplier{ID: "01B ", Name: "Baru"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nama kosong tidak dikunci sama sekali; penyisipan tetap berjalan.
func TestInsertWithBlankNameSkipsLock(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("supplier_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, NewRepo(db).Insert(context.Background(), mastersupplier.Supplier{ID: "1"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	supplier := mastersupplier.Supplier{ID: "01B", Name: "Baru"}
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		check   func(t *testing.T, err error)
	}{
		{"begin", func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errBoom) },
			func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"name taken", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_lock_by_name")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("X"))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) {
			require.ErrorIs(t, err, mastersupplier.ErrNameTaken)
			require.Contains(t, err.Error(), `"Baru"`)
		}},
		{"lock query", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_lock_by_name")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `mengunci "Baru"`) }},
		{"lock rows err", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_lock_by_name")).WillReturnRows(
				sqlmock.NewRows([]string{"ID"}).AddRow("X").RowError(0, errBoom))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `menelusuri penguncian "Baru"`) }},
		{"insert", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_lock_by_name")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			mock.ExpectExec(q("supplier_insert")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan "01B"`) }},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_lock_by_name")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			mock.ExpectExec(q("supplier_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			tc.check(t, NewRepo(db).Insert(context.Background(), supplier))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Penyimpanan menulis kunci yang dikenal DI ATAS dokumen lama, dan kunci asing dibiarkan.
func TestUpdateMergesKnownKeysOverStoredDocument(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("supplier_read_document")).WithArgs("01A").WillReturnRows(
		sqlmock.NewRows([]string{"JSONDATA"}).AddRow(`{"NAMA":"Lama","KUNCI_LAIN":"tetap"}`))
	mock.ExpectExec(q("supplier_update")).
		WithArgs(decodedDocument{"NAMA": "Baru", "KUNCI_LAIN": "tetap"}, "01A").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := NewRepo(db).Update(context.Background(), mastersupplier.Supplier{ID: " 01A", Name: "Baru"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Dokumen lama yang rusak ditulis ulang dari nol, bukan menghentikan penyimpanan; driver yang
// tidak dapat melaporkan jumlah baris tidak dianggap gagal.
func TestUpdateWithCorruptDocumentAndUnknownRowsAffected(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("supplier_read_document")).WillReturnRows(
		sqlmock.NewRows([]string{"JSONDATA"}).AddRow(`bukan json`))
	mock.ExpectExec(q("supplier_update")).
		WithArgs(decodedDocument{"NAMA": "Baru"}, "01A").
		WillReturnResult(sqlmock.NewErrorResult(errBoom))
	mock.ExpectCommit()

	require.NoError(t, NewRepo(db).Update(context.Background(), mastersupplier.Supplier{ID: "01A", Name: "Baru"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(mock sqlmock.Sqlmock)
		check   func(t *testing.T, err error)
	}{
		{"begin", func(mock sqlmock.Sqlmock) { mock.ExpectBegin().WillReturnError(errBoom) },
			func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi simpan") }},
		{"not found", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_read_document")).WillReturnRows(sqlmock.NewRows([]string{"JSONDATA"}))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, mastersupplier.ErrNotFound) }},
		{"read error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_read_document")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `membaca dokumen "01A"`) }},
		{"exec", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_read_document")).WillReturnRows(sqlmock.NewRows([]string{"JSONDATA"}).AddRow(nil))
			mock.ExpectExec(q("supplier_update")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `memperbarui "01A"`) }},
		{"zero rows", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_read_document")).WillReturnRows(sqlmock.NewRows([]string{"JSONDATA"}).AddRow("{}"))
			mock.ExpectExec(q("supplier_update")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, mastersupplier.ErrNotFound) }},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_read_document")).WillReturnRows(sqlmock.NewRows([]string{"JSONDATA"}).AddRow("{}"))
			mock.ExpectExec(q("supplier_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit().WillReturnError(errBoom)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi simpan") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newMock(t)
			tc.prepare(mock)
			tc.check(t, NewRepo(db).Update(context.Background(), mastersupplier.Supplier{ID: "01A", Name: "Baru"}))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Permintaan persetujuan dikirim dengan waktu UTC dan salinan dokumen supplier.
func TestRequestApproval(t *testing.T) {
	db, mock := newMock(t)
	jakarta := time.FixedZone("WIB", 7*3600)
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, jakarta)
	mock.ExpectExec(q("approval_insert")).
		WithArgs("SUP.1", at.UTC(), "", "PETUGAS", "", "alasan", mastersupplier.PositionRequested,
			decodedDocument{"ID": "01A", "NAMA": "Satu"}, "01A").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := NewRepo(db).RequestApproval(context.Background(), mastersupplier.ApprovalRequest{
		ID: " SUP.1 ", SupplierID: " 01A ", RequestedBy: "PETUGAS", RequestedAt: at,
		Reason: "alasan", Position: mastersupplier.PositionRequested,
		Snapshot: mastersupplier.Supplier{ID: "01A", Name: "Satu"},
	})
	require.NoError(t, err)

	mock.ExpectExec(q("approval_insert")).WillReturnError(errBoom)
	err = NewRepo(db).RequestApproval(context.Background(), mastersupplier.ApprovalRequest{ID: "SUP.2"})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), `permintaan persetujuan "SUP.2"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextIDComposesSiteAndSequence(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("supplier_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 01 "))
	mock.ExpectQuery(q("supplier_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))
	mock.ExpectCommit()

	got, err := NewRepo(db).NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastersupplier.ComposeID("01", 42, mastersupplier.SequenceWidth), got)
	require.Equal(t, "0100000000042", got)
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
			mock.ExpectQuery(q("supplier_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			mock.ExpectRollback()
		}, "tidak punya baris CURRENT_SITE='1'"},
		{"site error", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_site")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "membaca kode situs"},
		{"sequence", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("01"))
			mock.ExpectQuery(q("supplier_next_sequence")).WillReturnError(errBoom)
			mock.ExpectRollback()
		}, "mengambil nomor urut"},
		{"commit", func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(q("supplier_site")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("01"))
			mock.ExpectQuery(q("supplier_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
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

// Keempat daftar acuan dua kolom berbagi bentuk yang sama: dipangkas, dan setiap jalur galat
// menyebut daftar yang gagal.
func TestTwoColumnLookups(t *testing.T) {
	type lookup struct {
		queryName string
		call      func(r *Repo) (any, error)
		want      any
		words     string
	}
	lookups := []lookup{
		{"supplier_branch_list", func(r *Repo) (any, error) { return r.ListBranches(context.Background()) },
			[]mastersupplier.Branch{{ID: "001", Name: "Pusat"}}, "cabang"},
		{"supplier_country_list", func(r *Repo) (any, error) { return r.ListCountries(context.Background()) },
			[]mastersupplier.Country{{ID: "001", Name: "Pusat"}}, "negara"},
		{"supplier_bank_list", func(r *Repo) (any, error) { return r.ListBanks(context.Background()) },
			[]mastersupplier.Bank{{Code: "001", Name: "Pusat"}}, "bank"},
		{"supplier_city_search", func(r *Repo) (any, error) { return r.SearchCities(context.Background(), " pu ") },
			[]mastersupplier.City{{ID: "001", Name: "Pusat"}}, "kota"},
	}
	for _, l := range lookups {
		t.Run(l.queryName, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectQuery(q(l.queryName)).WillReturnRows(
				sqlmock.NewRows([]string{"A", "B"}).AddRow(" 001 ", " Pusat "))
			got, err := l.call(NewRepo(db))
			require.NoError(t, err)
			require.Equal(t, l.want, got)

			mock.ExpectQuery(q(l.queryName)).WillReturnError(errBoom)
			_, err = l.call(NewRepo(db))
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), l.words)

			mock.ExpectQuery(q(l.queryName)).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1"))
			_, err = l.call(NewRepo(db))
			require.ErrorContains(t, err, "membaca baris "+l.words)

			mock.ExpectQuery(q(l.queryName)).WillReturnRows(
				sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "a").RowError(0, errBoom))
			_, err = l.call(NewRepo(db))
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), "menelusuri")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Pencarian kota mengirim pola LIKE dan ID persis dalam huruf besar; kata kunci pendek tidak
// menyentuh basis data.
func TestSearchCitiesArgumentsAndShortKeyword(t *testing.T) {
	db, mock := newMock(t)
	got, err := NewRepo(db).SearchCities(context.Background(), " j ")
	require.NoError(t, err)
	require.Nil(t, got)

	mock.ExpectQuery(q("supplier_city_search")).WithArgs("%BAND%", "BAND").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}))
	got, err = NewRepo(db).SearchCities(context.Background(), "band")
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Sandi dibagi ke kelompoknya; nilai kosong dan kelompok tak dikenal dilewati.
func TestListCodesGroupsValues(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("supplier_code_distinct")).WillReturnRows(sqlmock.NewRows([]string{"KELOMPOK", "NILAI"}).
		AddRow(keyPartnerStatus, " 1 ").
		AddRow(keySupplyType, "0").
		AddRow(keySupplierType, "2").
		AddRow(keyActiveRequested, "1").
		AddRow(keyAutoPayment, "0").
		AddRow(keyAutoPayment, " ").
		AddRow("LAIN", "9"))

	got, err := NewRepo(db).ListCodes(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastersupplier.CodeSet{
		PartnerStatus: []mastersupplier.CodeOption{{Value: "1", Label: "1"}},
		SupplyType:    []mastersupplier.CodeOption{{Value: "0", Label: "0"}},
		SupplierType:  []mastersupplier.CodeOption{{Value: "2", Label: "2"}},
		Active:        []mastersupplier.CodeOption{{Value: "1", Label: "1"}},
		AutoPayment:   []mastersupplier.CodeOption{{Value: "0", Label: "0"}},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListCodesErrors(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("supplier_code_distinct")).WillReturnError(errBoom)
	_, err := NewRepo(db).ListCodes(context.Background())
	require.ErrorContains(t, err, "membaca daftar sandi")

	mock.ExpectQuery(q("supplier_code_distinct")).WillReturnRows(sqlmock.NewRows([]string{"K"}).AddRow("x"))
	_, err = NewRepo(db).ListCodes(context.Background())
	require.ErrorContains(t, err, "membaca baris sandi")

	mock.ExpectQuery(q("supplier_code_distinct")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "V"}).AddRow("x", "y").RowError(0, errBoom))
	_, err = NewRepo(db).ListCodes(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar sandi")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan mode periksa: tabel terbaca, hitungan diteruskan, galat menyebut objeknya.
func TestCheckFunctions(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(q("supplier_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	require.NoError(t, repo.CheckTable(ctx))
	mock.ExpectQuery(q("supplier_check_table")).WillReturnError(errBoom)
	require.ErrorContains(t, repo.CheckTable(ctx), "M_SUPPLIER tidak dapat dibaca")

	mock.ExpectQuery(q("approval_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	require.NoError(t, repo.CheckApprovalTable(ctx))
	mock.ExpectQuery(q("approval_check_table")).WillReturnError(errBoom)
	require.ErrorContains(t, repo.CheckApprovalTable(ctx), "PROTEKSI_KLAIMMBU tidak dapat dibaca")

	mock.ExpectQuery(q("supplier_count_all")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(12))
	total, err := repo.CountAll(ctx)
	require.NoError(t, err)
	require.Equal(t, 12, total)
	mock.ExpectQuery(q("supplier_count_all")).WillReturnError(errBoom)
	_, err = repo.CountAll(ctx)
	require.ErrorContains(t, err, "menghitung baris")

	mock.ExpectQuery(q("supplier_check_document")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(10))
	readable, err := repo.CountReadable(ctx)
	require.NoError(t, err)
	require.Equal(t, 10, readable)
	mock.ExpectQuery(q("supplier_check_document")).WillReturnError(errBoom)
	_, err = repo.CountReadable(ctx)
	require.ErrorContains(t, err, "menghitung baris terbaca")

	mock.ExpectQuery(q("approval_count_pending")).WithArgs(mastersupplier.PositionRequested).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(3))
	pending, err := repo.CountApprovalPending(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, pending)
	mock.ExpectQuery(q("approval_count_pending")).WillReturnError(errBoom)
	_, err = repo.CountApprovalPending(ctx)
	require.ErrorContains(t, err, "menghitung permintaan menunggu")

	require.NoError(t, mock.ExpectationsWereMet())
}
