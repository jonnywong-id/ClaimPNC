package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
)

var (
	errOracle = errors.New("oracle menolak")
	single    = []string{"X"}
	pairCols  = []string{"A", "B"}
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// workshopColumns adalah 41 kolom pembaca bengkel.
func workshopColumns() []string {
	cols := make([]string, 41)
	for i := range cols {
		cols[i] = "C" + strconv.Itoa(i)
	}
	return cols
}

// workshopRow menyusun satu baris berisi ID, nama, dan status; kolom lain NULL.
func workshopRow(id, name, status string) *sqlmock.Rows {
	values := make([]driver.Value, 41)
	values[0] = id + " "
	values[1] = " " + name
	values[15] = " login "
	values[40] = status
	return sqlmock.NewRows(workshopColumns()).AddRow(values...)
}

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

func TestListMapsRowsWithAndWithoutKeyword(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("bengkel_list")).WithArgs("0").
		WillReturnRows(workshopRow("010000000001", "Bengkel", "0"))
	list, err := repo.List(ctx, masterbengkel.Filter{Status: masterbengkel.StatusPending})
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.Workshop{{
		ID: "010000000001", Name: "Bengkel", Login: "login", Status: masterbengkel.StatusPending,
	}}, list)

	mock.ExpectQuery(q("bengkel_list_search")).WithArgs("1", `%A\_B%`).
		WillReturnRows(sqlmock.NewRows(workshopColumns()))
	list, err = repo.List(ctx, masterbengkel.Filter{Status: masterbengkel.StatusApproved, Keyword: "a_b"})
	require.NoError(t, err)
	require.Empty(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	filter := masterbengkel.Filter{Status: masterbengkel.StatusPending}

	mock.ExpectQuery(q("bengkel_list")).WillReturnError(errOracle)
	_, err := repo.List(ctx, filter)
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(q("bengkel_list")).WillReturnRows(sqlmock.NewRows(single).AddRow("x"))
	_, err = repo.List(ctx, filter)
	require.ErrorContains(t, err, "membaca baris daftar")

	mock.ExpectQuery(q("bengkel_list")).
		WillReturnRows(workshopRow("1", "a", "0").RowError(0, errOracle))
	_, err = repo.List(ctx, filter)
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Ketiga pencari satu baris membedakan "tidak ada" dari galat.
func TestSingleRowLookups(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("bengkel_get")).WithArgs("1").WillReturnRows(workshopRow("1", "A", "0"))
	got, err := repo.Get(ctx, " 1 ")
	require.NoError(t, err)
	require.Equal(t, "A", got.Name)

	mock.ExpectQuery(q("bengkel_get")).WillReturnError(sql.ErrNoRows)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	mock.ExpectQuery(q("bengkel_get")).WillReturnError(errOracle)
	_, err = repo.Get(ctx, "1")
	require.ErrorContains(t, err, `membaca "1"`)

	_, err = repo.FindByName(ctx, " ")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	mock.ExpectQuery(q("bengkel_find_by_name")).WithArgs("BENGKEL").
		WillReturnRows(workshopRow("1", "Bengkel", "0"))
	byName, err := repo.FindByName(ctx, " bengkel ")
	require.NoError(t, err)
	require.Equal(t, "1", byName.ID)
	mock.ExpectQuery(q("bengkel_find_by_name")).WillReturnError(sql.ErrNoRows)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	mock.ExpectQuery(q("bengkel_find_by_name")).WillReturnError(errOracle)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorContains(t, err, "mencari nama")

	_, err = repo.FindByLogin(ctx, "")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	mock.ExpectQuery(q("bengkel_find_by_login")).WithArgs("LOGIN").
		WillReturnRows(workshopRow("2", "B", "1"))
	byLogin, err := repo.FindByLogin(ctx, "login")
	require.NoError(t, err)
	require.Equal(t, "2", byLogin.ID)
	mock.ExpectQuery(q("bengkel_find_by_login")).WillReturnError(sql.ErrNoRows)
	_, err = repo.FindByLogin(ctx, "x")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
	mock.ExpectQuery(q("bengkel_find_by_login")).WillReturnError(errOracle)
	_, err = repo.FindByLogin(ctx, "x")
	require.ErrorContains(t, err, "mencari login")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penambahan memeriksa nama dan login di dalam satu transaksi.
func TestInsertChecksNameAndLoginInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)
	workshop := sampleWorkshop()
	workshop.Login = "loginbaru"

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single))
	mock.ExpectQuery(q("bengkel_lock_by_login")).WithArgs("LOGINBARU").
		WillReturnRows(sqlmock.NewRows(single))
	mock.ExpectExec(q("bengkel_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Insert(context.Background(), workshop))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	withLogin := sampleWorkshop()
	withLogin.Login = "login"
	noLogin := sampleWorkshop()
	noLogin.Login = ""

	cases := []struct {
		name     string
		workshop masterbengkel.Workshop
		prepare  func(sqlmock.Sqlmock)
		check    func(*testing.T, error)
	}{
		{"mulai", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"nama dipakai", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single).AddRow(1))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterbengkel.ErrNameTaken) }},
		{"periksa nama", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memeriksa") }},
		{"telusur nama", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).
				WillReturnRows(sqlmock.NewRows(single).AddRow(1).RowError(0, errOracle))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menelusuri pemeriksaan") }},
		{"login dipakai", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single))
			m.ExpectQuery(q("bengkel_lock_by_login")).WillReturnRows(sqlmock.NewRows(single).AddRow(1))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterbengkel.ErrLoginTaken) }},
		{"periksa login", withLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single))
			m.ExpectQuery(q("bengkel_lock_by_login")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memeriksa") }},
		{"sisip", noLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single))
			m.ExpectExec(q("bengkel_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menyisipkan") }},
		{"commit", noLogin, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("bengkel_lock_by_name")).WillReturnRows(sqlmock.NewRows(single))
			m.ExpectExec(q("bengkel_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			c.check(t, repo.Insert(context.Background(), c.workshop))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	// Nama kosong tidak diperiksa ke basis data.
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("bengkel_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.Insert(context.Background(), masterbengkel.Workshop{ID: "1"}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectExec(q("bengkel_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Update(ctx, sampleWorkshop()))

	mock.ExpectExec(q("bengkel_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.Update(ctx, sampleWorkshop()), masterbengkel.ErrNotFound)

	// Driver tanpa RowsAffected tidak dianggap gagal.
	mock.ExpectExec(q("bengkel_update")).WillReturnResult(sqlmock.NewErrorResult(errOracle))
	require.NoError(t, repo.Update(ctx, sampleWorkshop()))

	mock.ExpectExec(q("bengkel_update")).WillReturnError(errOracle)
	require.ErrorContains(t, repo.Update(ctx, sampleWorkshop()), "memperbarui")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatus(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	changed, err := repo.SetStatus(ctx, nil, masterbengkel.StatusApproved)
	require.NoError(t, err)
	require.Zero(t, changed)

	mock.ExpectBegin()
	mock.ExpectExec(q("bengkel_set_status")).WithArgs("1", "A").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("bengkel_set_status")).WithArgs("1", "B").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("bengkel_set_status")).WithArgs("1", "C").
		WillReturnResult(sqlmock.NewErrorResult(errOracle))
	mock.ExpectCommit()
	changed, err = repo.SetStatus(ctx, []string{" A", "B", "C"}, masterbengkel.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	mock.ExpectBegin().WillReturnError(errOracle)
	_, err = repo.SetStatus(ctx, []string{"A"}, masterbengkel.StatusApproved)
	require.ErrorContains(t, err, "memulai transaksi keputusan")

	mock.ExpectBegin()
	mock.ExpectExec(q("bengkel_set_status")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.SetStatus(ctx, []string{"A"}, masterbengkel.StatusApproved)
	require.ErrorContains(t, err, `menetapkan status "A"`)

	mock.ExpectBegin()
	mock.ExpectExec(q("bengkel_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errOracle)
	_, err = repo.SetStatus(ctx, []string{"A"}, masterbengkel.StatusApproved)
	require.ErrorContains(t, err, "menutup transaksi keputusan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextID(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_site")).WillReturnRows(sqlmock.NewRows(single).AddRow(" 01 "))
	mock.ExpectQuery(q("bengkel_next_sequence")).WillReturnRows(sqlmock.NewRows(single).AddRow(9))
	mock.ExpectCommit()
	id, err := repo.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterbengkel.ComposeID("01", 9, sequenceWidth), id)

	mock.ExpectBegin().WillReturnError(errOracle)
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "memulai transaksi penomoran")

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_site")).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "tidak punya baris CURRENT_SITE='1'")

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_site")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "membaca kode situs")

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_site")).WillReturnRows(sqlmock.NewRows(single).AddRow("01"))
	mock.ExpectQuery(q("bengkel_next_sequence")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "mengambil nomor urut")

	mock.ExpectBegin()
	mock.ExpectQuery(q("bengkel_site")).WillReturnRows(sqlmock.NewRows(single).AddRow("01"))
	mock.ExpectQuery(q("bengkel_next_sequence")).WillReturnRows(sqlmock.NewRows(single).AddRow(1))
	mock.ExpectCommit().WillReturnError(errOracle)
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "menutup transaksi penomoran")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Daftar acuan melewati baris yang kode atau namanya kosong.
func TestReferenceListsSkipIncompleteRows(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("bengkel_branch_list")).WillReturnRows(sqlmock.NewRows(pairCols).
		AddRow(" Pusat ", " 001 ").AddRow(nil, "002").AddRow("Tanpa Kode", nil))
	branches, err := repo.ListBranches(ctx)
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.Branch{{ID: "001", Name: "Pusat"}}, branches)

	mock.ExpectQuery(q("bengkel_city_search")).WithArgs("%JAKARTA%", "JAKARTA").
		WillReturnRows(sqlmock.NewRows(pairCols).AddRow("3171", "Jakarta Pusat").AddRow(nil, "x"))
	cities, err := repo.SearchCities(ctx, " jakarta ")
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.City{{ID: "3171", Name: "Jakarta Pusat"}}, cities)

	mock.ExpectQuery(q("bengkel_bank_list")).WillReturnRows(sqlmock.NewRows(pairCols).
		AddRow("002", "Bank Satu").AddRow("003", nil))
	banks, err := repo.ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, []masterbengkel.Bank{{Code: "002", Name: "Bank Satu"}}, banks)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReferenceListErrors(t *testing.T) {
	ctx := context.Background()

	type call func(*Repo) error
	lists := []struct {
		name string
		call call
		what string
	}{
		{"bengkel_branch_list", func(r *Repo) error { _, err := r.ListBranches(ctx); return err }, "cabang"},
		{"bengkel_city_search", func(r *Repo) error { _, err := r.SearchCities(ctx, "x"); return err }, "kota"},
		{"bengkel_bank_list", func(r *Repo) error { _, err := r.ListBanks(ctx); return err }, "bank"},
	}

	for _, l := range lists {
		repo, mock := newMock(t)

		mock.ExpectQuery(q(l.name)).WillReturnError(errOracle)
		require.ErrorIs(t, l.call(repo), errOracle)

		mock.ExpectQuery(q(l.name)).WillReturnRows(sqlmock.NewRows(single).AddRow("x"))
		require.ErrorContains(t, l.call(repo), "membaca baris "+l.what)

		mock.ExpectQuery(q(l.name)).
			WillReturnRows(sqlmock.NewRows(pairCols).AddRow("a", "b").RowError(0, errOracle))
		require.ErrorContains(t, l.call(repo), "menelusuri daftar "+l.what)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestChecksAndCounters(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("bengkel_check_table")).WillReturnRows(sqlmock.NewRows(single))
	require.NoError(t, repo.CheckTable(ctx))
	mock.ExpectQuery(q("bengkel_check_table")).WillReturnError(errOracle)
	require.ErrorContains(t, repo.CheckTable(ctx), "POOLDATA.BENGKEL_HE tidak dapat dibaca")

	mock.ExpectQuery(q("bengkel_check_json_mirror")).WillReturnRows(sqlmock.NewRows(single))
	require.NoError(t, repo.CheckJSONMirror(ctx))
	mock.ExpectQuery(q("bengkel_check_json_mirror")).WillReturnError(errOracle)
	require.ErrorContains(t, repo.CheckJSONMirror(ctx), "POOLDATA.M_BENGKEL_HE tidak dapat dibaca")

	mock.ExpectQuery(q("bengkel_count_pending")).WithArgs("0").
		WillReturnRows(sqlmock.NewRows(single).AddRow(3))
	pending, err := repo.CountByStatus(ctx, masterbengkel.StatusPending)
	require.NoError(t, err)
	require.Equal(t, 3, pending)
	mock.ExpectQuery(q("bengkel_count_pending")).WillReturnError(errOracle)
	_, err = repo.CountByStatus(ctx, masterbengkel.StatusPending)
	require.ErrorContains(t, err, "menghitung baris status")

	mock.ExpectQuery(q("bengkel_count_all")).WillReturnRows(sqlmock.NewRows(single).AddRow(5))
	all, err := repo.CountAll(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, all)
	mock.ExpectQuery(q("bengkel_count_all")).WillReturnError(errOracle)
	_, err = repo.CountAll(ctx)
	require.ErrorContains(t, err, "menghitung baris")

	mock.ExpectQuery(q("bengkel_count_json_mirror")).WillReturnRows(sqlmock.NewRows(single).AddRow(6))
	mirror, err := repo.CountJSONMirror(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, mirror)
	mock.ExpectQuery(q("bengkel_count_json_mirror")).WillReturnError(errOracle)
	_, err = repo.CountJSONMirror(ctx)
	require.ErrorContains(t, err, "M_BENGKEL_HE")
	require.NoError(t, mock.ExpectationsWereMet())
}

// DATAID memakai dua digit tahun dari jam yang disetel dan nomor urut dari sequence.
func TestNextDocumentIDUsesTheClockYear(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	repo = repo.WithClock(fixedClock{at: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)})

	mock.ExpectQuery(q("bengkel_next_document_sequence")).
		WillReturnRows(sqlmock.NewRows(single).AddRow(12))
	id, err := repo.NextDocumentID(ctx)
	require.NoError(t, err)
	require.Equal(t, "270000000012", id)

	mock.ExpectQuery(q("bengkel_next_document_sequence")).WillReturnError(errOracle)
	_, err = repo.NextDocumentID(ctx)
	require.ErrorContains(t, err, "nomor urut lampiran")
	require.NoError(t, mock.ExpectationsWereMet())

	// Tanpa jam yang disetel, tahunnya diambil dari jam sistem.
	plain, plainMock := newMock(t)
	plainMock.ExpectQuery(q("bengkel_next_document_sequence")).
		WillReturnRows(sqlmock.NewRows(single).AddRow(1))
	id, err = plain.NextDocumentID(ctx)
	require.NoError(t, err)
	require.Len(t, id, 12)
	require.Equal(t, "0000000001", id[2:])
}

// Lampiran disisipkan dan ditautkan dalam satu transaksi.
func TestSaveDocument(t *testing.T) {
	ctx := context.Background()
	document := masterbengkel.Document{ID: " 26001 ", UploadedBy: " penguji ", Name: " a.pdf ",
		Note: " catatan ", MimeType: " application/pdf ", Content: []byte("isi")}

	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("bengkel_insert_document")).
		WithArgs("26001", "penguji", "a.pdf", "catatan", "application/pdf", []byte("isi")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("bengkel_set_document")).WithArgs("26001", "010000000001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.SaveDocument(ctx, " 010000000001 ", document))
	require.NoError(t, mock.ExpectationsWereMet())

	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"mulai", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errOracle) },
			func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi lampiran") }},
		{"sisip", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("bengkel_insert_document")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menyisipkan lampiran") }},
		{"tautkan", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("bengkel_insert_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("bengkel_set_document")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menautkan lampiran") }},
		{"bengkel tidak ada", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("bengkel_insert_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("bengkel_set_document")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterbengkel.ErrNotFound) }},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("bengkel_insert_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("bengkel_set_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi lampiran") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			c.check(t, repo.SaveDocument(ctx, "1", document))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindDocument(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	uploaded := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q("bengkel_get_document")).WithArgs("26001").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "N", "NOTE", "M", "OP", "AT", "C"}).
			AddRow(" 26001 ", " a.pdf ", nil, " application/pdf ", " penguji ", uploaded, []byte("isi")))
	document, err := repo.FindDocument(ctx, " 26001 ")
	require.NoError(t, err)
	require.Equal(t, masterbengkel.Document{
		ID: "26001", Name: "a.pdf", MimeType: "application/pdf", Content: []byte("isi"),
		UploadedBy: "penguji", UploadedAt: uploaded,
	}, document)

	mock.ExpectQuery(q("bengkel_get_document")).WillReturnError(sql.ErrNoRows)
	_, err = repo.FindDocument(ctx, "x")
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)

	mock.ExpectQuery(q("bengkel_get_document")).WillReturnError(errOracle)
	_, err = repo.FindDocument(ctx, "x")
	require.ErrorContains(t, err, "membaca lampiran")
	require.NoError(t, mock.ExpectationsWereMet())
}
