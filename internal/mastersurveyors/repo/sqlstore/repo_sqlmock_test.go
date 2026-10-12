package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
)

var errBoom = errors.New("basis data mati")

var surveyorColumns = []string{
	"D_SURVEY_ID", "OLD_D_SURVEY_ID", "M_SURVEY_ID", "TYPE_DESC",
	"NAME", "ADDRESS", "KDPOS", "STATE",
	"TELEPHONE", "FAKSIMILE", "EMAIL", "OTHER_CONTACT",
	"BRANCH", "BRANCHNAME", "LOGIN_APLIKASI", "DOCID",
	"APPROVAL", "KOMITE", "TRFKOMITE",
	"TGL_APPROVE", "CATATAN", "USER_INPUT", "TGL_INPUT", "USER_UPDATE",
}

var (
	created = time.Date(2026, 9, 10, 2, 0, 0, 0, time.UTC)
	decided = time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
)

func surveyorRow(id string) []driver.Value {
	return []driver.Value{
		id, nil, "1001 ", "INTERNAL SURVEYOR",
		" Surveyor Satu ", "Jalan", "12345", "DKI",
		"021", nil, "a@b.c", nil,
		"001", "Pusat", "SURV1", nil,
		" 1 ", "KOMITE", "1",
		decided, "ok", "SISTEM", created, nil,
	}
}

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

func TestListMembacaHalamanDenganFilter(t *testing.T) {
	repo, mock := newMock(t)

	filter := mastersurveyors.Filter{
		Status: mastersurveyors.StatusPending, Name: " budi ", TypeCode: "1002",
		MyCommitteeOnly: true, CommitteeIdentity: "KOMITE", Offset: -5, Limit: 9999,
	}
	filters := []driver.Value{"0", "0", "budi", "budi", nil, nil, "1002", "1002", "KOMITE", "KOMITE"}

	mock.ExpectQuery(exact("surveyor_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(12))
	mock.ExpectQuery(exact("surveyor_list")).
		WithArgs(append(append([]driver.Value{}, filters...), 0, 500)...).
		WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(surveyorRow("1000001")...))

	rows, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, 12, total)
	require.Len(t, rows, 1)

	got := rows[0]
	require.Equal(t, "1000001", got.ID)
	require.Equal(t, "1001", got.TypeCode)
	require.Equal(t, "Surveyor Satu", got.Name)
	require.Equal(t, mastersurveyors.StatusApproved, got.Status)
	require.Equal(t, decided, *got.DecidedAt)
	require.Equal(t, created, got.CreatedAt)
	require.Empty(t, got.Fax)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Komite tidak ikut menyaring bila MyCommitteeOnly mati; batas bawaan 50.
func TestListBatasBawaan(t *testing.T) {
	repo, mock := newMock(t)

	filters := []driver.Value{nil, nil, nil, nil, "x", "x", nil, nil, nil, nil}
	mock.ExpectQuery(exact("surveyor_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
	mock.ExpectQuery(exact("surveyor_list")).
		WithArgs(append(append([]driver.Value{}, filters...), 3, 50)...).
		WillReturnRows(sqlmock.NewRows(surveyorColumns))

	rows, total, err := repo.List(context.Background(), mastersurveyors.Filter{
		AppLogin: "x", CommitteeIdentity: "TIDAKDIPAKAI", Offset: 3,
	})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMeneruskanGalat(t *testing.T) {
	t.Run("hitung", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_count")).WillReturnError(errBoom)
		_, _, err := repo.List(context.Background(), mastersurveyors.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("daftar", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_count")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(exact("surveyor_list")).WillReturnError(errBoom)
		_, _, err := repo.List(context.Background(), mastersurveyors.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_count")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(exact("surveyor_list")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, _, err := repo.List(context.Background(), mastersurveyors.Filter{})
		require.ErrorContains(t, err, "memindai baris surveyor")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_count")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(exact("surveyor_list")).
			WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(surveyorRow("1")...).
				RowError(0, errBoom))
		_, _, err := repo.List(context.Background(), mastersurveyors.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGet(t *testing.T) {
	t.Run("ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_get")).WithArgs("1000001").
			WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(surveyorRow("1000001")...))
		got, err := repo.Get(context.Background(), " 1000001 ")
		require.NoError(t, err)
		require.Equal(t, "SURV1", got.AppLogin)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("tidak ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_get")).WillReturnRows(sqlmock.NewRows(surveyorColumns))
		_, err := repo.Get(context.Background(), "x")
		require.ErrorIs(t, err, mastersurveyors.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("galat", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_get")).WillReturnError(errBoom)
		_, err := repo.Get(context.Background(), "x")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindByNameKeyDanLogin(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("surveyor_by_name_key")).WithArgs("BUDISANTOSO").
		WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(surveyorRow("1")...))
	mock.ExpectQuery(exact("surveyor_by_login")).WithArgs("SURV1").
		WillReturnRows(sqlmock.NewRows(surveyorColumns))

	byName, err := repo.FindByNameKey(context.Background(), "budi santoso")
	require.NoError(t, err)
	require.Len(t, byName, 1)

	byLogin, err := repo.FindByAppLogin(context.Background(), " surv1 ")
	require.NoError(t, err)
	require.Empty(t, byLogin)

	empty, err := repo.FindByAppLogin(context.Background(), "  ")
	require.NoError(t, err)
	require.Nil(t, empty, "login kosong tidak menembak basis data")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryManyMeneruskanGalat(t *testing.T) {
	t.Run("kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_by_login")).WillReturnError(errBoom)
		_, err := repo.FindByAppLogin(context.Background(), "x")
		require.ErrorContains(t, err, "mencari surveyor menurut login aplikasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("pindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_by_name_key")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := repo.FindByNameKey(context.Background(), "x")
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("baris", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("surveyor_by_name_key")).
			WillReturnRows(sqlmock.NewRows(surveyorColumns).AddRow(surveyorRow("1")...).
				RowError(0, errBoom))
		_, err := repo.FindByNameKey(context.Background(), "x")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func sample() mastersurveyors.Surveyor {
	return mastersurveyors.Surveyor{
		TypeCode: "1002", Name: " Adjuster ", Email: "a@b.c", AppLogin: "",
		Status: mastersurveyors.StatusPending, Committee: "KOMITE",
	}
}

// Kode surveyor dibentuk dari kode situs ditambah enam digit urutan.
func TestInsertMembentukKodeDariSitusDanUrutan(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("surveyor_site")).
		WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow(" 1 "))
	mock.ExpectQuery(exact("surveyor_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(42)))
	mock.ExpectExec(exact("surveyor_insert")).
		WithArgs("1000042", "1002", "Adjuster", nil, nil, nil, nil, nil, "a@b.c", nil,
			nil, nil, nil, nil, "0", "KOMITE", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := repo.Insert(context.Background(), sample())
	require.NoError(t, err)
	require.Equal(t, "1000042", saved.ID)
	require.Equal(t, "Adjuster", saved.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertMeneruskanGalat(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := repo.Insert(context.Background(), sample())
		require.ErrorContains(t, err, "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("tanpa situs", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("surveyor_site")).WillReturnRows(sqlmock.NewRows([]string{"S"}))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), sample())
		require.ErrorIs(t, err, mastersurveyors.ErrNoSite)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("situs galat", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("surveyor_site")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), sample())
		require.ErrorContains(t, err, "membaca kode situs")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("urutan galat", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("surveyor_site")).
			WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow("1"))
		mock.ExpectQuery(exact("surveyor_next_sequence")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), sample())
		require.ErrorContains(t, err, "nomor urut surveyor")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("surveyor_site")).
			WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow("1"))
		mock.ExpectQuery(exact("surveyor_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1234567)))
		mock.ExpectExec(exact("surveyor_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := repo.Insert(context.Background(), sample())
		require.ErrorContains(t, err, "menyelesaikan transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Pelanggaran indeks unik diterjemahkan menjadi galat domain.
func TestTulisMenerjemahkanPelanggaranIndeks(t *testing.T) {
	cases := []struct {
		message string
		want    error
		text    string
	}{
		{"ORA-00001: unique constraint (POOLDATA.UX_D_SURVEYORS_NAME) violated",
			mastersurveyors.ErrNameTaken, ""},
		{"ora-00001: ux_d_surveyors_login", mastersurveyors.ErrLoginTaken, ""},
		{"ORA-00001: D_SURVEYORS_PK", nil, "kode surveyor bentrok"},
		{"lain", nil, "menyimpan surveyor: lain"},
	}

	for _, c := range cases {
		t.Run(c.message, func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectBegin()
			mock.ExpectQuery(exact("surveyor_site")).
				WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow("1"))
			mock.ExpectQuery(exact("surveyor_next_sequence")).
				WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(1)))
			mock.ExpectExec(exact("surveyor_insert")).WillReturnError(errors.New(c.message))
			mock.ExpectRollback()

			_, err := repo.Insert(context.Background(), sample())
			if c.want != nil {
				require.ErrorIs(t, err, c.want)
			} else {
				require.ErrorContains(t, err, c.text)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	changed := sample()
	changed.ID = "1000002"
	changed.Status = mastersurveyors.StatusApproved
	changed.NeedDirector = "1"

	t.Run("berhasil", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact("surveyor_update")).
			WithArgs("1002", "Adjuster", nil, nil, nil, nil, nil, "a@b.c", nil, nil, nil,
				nil, nil, "1", "1", "1000002").
			WillReturnResult(sqlmock.NewResult(0, 1))
		require.NoError(t, repo.Update(context.Background(), changed))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("tidak ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact("surveyor_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		require.ErrorIs(t, repo.Update(context.Background(), changed),
			mastersurveyors.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows affected tidak terbaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact("surveyor_update")).
			WillReturnResult(sqlmock.NewErrorResult(errBoom))
		require.NoError(t, repo.Update(context.Background(), changed))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("galat", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectExec(exact("surveyor_update")).
			WillReturnError(errors.New("UX_D_SURVEYORS_LOGIN"))
		require.ErrorIs(t, repo.Update(context.Background(), changed),
			mastersurveyors.ErrLoginTaken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("surveyor_check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"D_SURVEY_ID"}))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(exact("surveyor_check_table")).WillReturnError(errBoom)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "D_SURVEYORS")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPembantuNilai(t *testing.T) {
	require.Equal(t, "000042", pad6(42))
	require.Equal(t, "1234567", pad6(1234567))
	require.Nil(t, nullTime(nil))
	require.Equal(t, decided, nullTime(&decided))
}
