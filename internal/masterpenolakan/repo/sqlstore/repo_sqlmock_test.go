package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
)

var errDB = errors.New("basis data mati")

// rowColumns adalah urutan kolom yang dibaca scanRow.
var rowColumns = []string{
	"ID_ND", "NOTE_ND", "ID_ST", "NOTE_ST", "STATUS", "USER_INPUT",
	"TANGGALKIRIM", "APPROVEBY", "TANGGAL_APPROVE", "NOTEAPPROVED",
}

// newMock membuka sqlmock dengan pencocok regexp.
func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah teks kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var (
	submittedAt = time.Date(2026, 9, 10, 3, 0, 0, 0, time.UTC)
	approvedAt  = time.Date(2026, 9, 12, 3, 0, 0, 0, time.UTC)
)

func TestListParentMapsRowsAndTrims(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("rejection_parent_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).
			AddRow("1 ", " POLIS ").
			AddRow(nil, nil))

	got, err := NewRepo(db).ListParent(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpenolakan.RejectionStatus{{ID: "1", Name: "POLIS"}, {}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListParentErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("rejection_parent_list")).WillReturnError(errDB)
	_, err := repo.ListParent(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca daftar status penolakan 1")

	// Galat Scan: jumlah kolom salah.
	mock.ExpectQuery(q("rejection_parent_list")).WillReturnRows(sqlmock.NewRows([]string{"ID_ST"}).AddRow("1"))
	_, err = repo.ListParent(context.Background())
	require.Error(t, err)

	mock.ExpectQuery(q("rejection_parent_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("1", "A").RowError(0, errDB))
	_, err = repo.ListParent(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menelusuri status penolakan 1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMapsFullRow(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("rejection_list")).WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow(" 1", "PREMI ", "1", "POLIS", "1 ", "adminpnc", submittedAt, "manager", approvedAt, " ok ").
		AddRow("2", "B", "1", "POLIS", "0", "adminpnc", nil, nil, nil, nil))

	got, err := NewRepo(db).List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "1", got[0].ID)
	require.Equal(t, "PREMI", got[0].Name)
	require.Equal(t, masterpenolakan.StatusApproved, got[0].Status)
	require.Equal(t, submittedAt, got[0].SubmittedAt)
	require.NotNil(t, got[0].ApprovedAt)
	require.Equal(t, approvedAt, *got[0].ApprovedAt)
	require.Equal(t, "ok", got[0].ApprovalNote)
	require.True(t, got[1].SubmittedAt.IsZero())
	require.Nil(t, got[1].ApprovedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("rejection_list")).WillReturnError(errDB)
	_, err := repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar")

	mock.ExpectQuery(q("rejection_list")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}).AddRow("1"))
	_, err = repo.List(context.Background())
	require.Error(t, err)

	mock.ExpectQuery(q("rejection_list")).WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow("1", "A", "1", "P", "0", "u", nil, nil, nil, nil).RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTrimsIDAndMapsErrors(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("rejection_get")).WithArgs("3").WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow("3", "A", "2", "P", "2", "u", submittedAt, nil, nil, nil))
	got, err := repo.Get(context.Background(), " 3 ")
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.StatusRejected, got.Status)

	mock.ExpectQuery(q("rejection_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(rowColumns))
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, masterpenolakan.ErrNotFound)

	mock.ExpectQuery(q("rejection_get")).WithArgs("9").WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, `membaca "9"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewWithExistingParent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rejection_parent_get")).WithArgs("2").
		WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "DIKECUALIKAN"))
	mock.ExpectQuery(q("rejection_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}).AddRow("1").AddRow(" 9").AddRow(nil))
	mock.ExpectExec(q("rejection_insert")).
		WithArgs("2", "DIKECUALIKAN", "10", "BARU", "adminpnc", submittedAt, "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	at := submittedAt.In(time.FixedZone("WIB", 7*3600))
	got, err := NewRepo(db).InsertNew(context.Background(), masterpenolakan.Submission{
		Input: masterpenolakan.Input{Name: "BARU", ParentID: "2"},
		By:    "adminpnc",
		At:    at,
	})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.RejectionStatus2{
		ID: "10", Name: "BARU", ParentID: "2", ParentName: "DIKECUALIKAN",
		Status: masterpenolakan.StatusPending, SubmittedBy: "adminpnc", SubmittedAt: submittedAt,
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewWithNewParent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rejection_parent_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"ID_ST"}).AddRow("1").AddRow("4"))
	mock.ExpectExec(q("rejection_parent_insert")).WithArgs("5", "INDUK BARU").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}))
	mock.ExpectExec(q("rejection_insert")).
		WithArgs("5", "INDUK BARU", "1", "ANAK", "u", submittedAt, "0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := NewRepo(db).InsertNew(context.Background(), masterpenolakan.Submission{
		Input: masterpenolakan.Input{Name: "ANAK", ParentName: "INDUK BARU"},
		By:    "u", At: submittedAt,
	})
	require.NoError(t, err)
	require.Equal(t, "1", got.ID)
	require.Equal(t, "5", got.ParentID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertNewFailures(t *testing.T) {
	existing := masterpenolakan.Submission{Input: masterpenolakan.Input{Name: "A", ParentID: "2"}, By: "u", At: submittedAt}
	fresh := masterpenolakan.Submission{Input: masterpenolakan.Input{Name: "A", ParentName: "P"}, By: "u", At: submittedAt}

	cases := []struct {
		name  string
		input masterpenolakan.Submission
		setup func(sqlmock.Sqlmock)
		check func(*testing.T, error)
	}{
		{
			name: "begin", input: existing,
			setup: func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) },
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") },
		},
		{
			name: "parent not found", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}))
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, masterpenolakan.ErrParentNotFound) },
		},
		{
			name: "parent read error", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, `membaca induk "2"`) },
		},
		{
			name: "parent lock error", input: fresh,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_list_id_locked")).WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "mengunci daftar ID (rejection_parent_list_id_locked)")
			},
		},
		{
			name: "parent insert error", input: fresh,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ST"}))
				m.ExpectExec(q("rejection_parent_insert")).WithArgs("1", "P").WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan induk "1"`) },
		},
		{
			name: "child lock scan error", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectQuery(q("rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND", "X"}).AddRow("1", "2"))
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "membaca ID (rejection_list_id_locked)") },
		},
		{
			name: "child lock rows error", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectQuery(q("rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}).AddRow("1").RowError(0, errDB))
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "menelusuri daftar ID (rejection_list_id_locked)")
			},
		},
		{
			name: "insert error", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectQuery(q("rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}))
				m.ExpectExec(q("rejection_insert")).WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan "1"`) },
		},
		{
			name: "commit error", input: existing,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectQuery(q("rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}))
				m.ExpectExec(q("rejection_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
				m.ExpectCommit().WillReturnError(errDB)
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.setup(mock)
			_, err := NewRepo(db).InsertNew(context.Background(), c.input)
			c.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateKeepsApprovalTrailAndResetsStatus(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnRows(sqlmock.NewRows(rowColumns).
		AddRow("1", "LAMA", "1", "POLIS", "1", "adminpnc", submittedAt, "manager", approvedAt, "ok"))
	mock.ExpectQuery(q("rejection_parent_get")).WithArgs("2").
		WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "DIKECUALIKAN"))
	mock.ExpectExec(q("rejection_update")).
		WithArgs("2", "DIKECUALIKAN", "BARU", "pic", submittedAt, "0", "1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := NewRepo(db).Update(context.Background(), " 1 ", masterpenolakan.Submission{
		Input: masterpenolakan.Input{Name: "BARU", ParentID: "2"}, By: "pic", At: submittedAt,
	})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.StatusPending, got.Status)
	require.Equal(t, "manager", got.ApprovedBy)
	require.Equal(t, approvedAt, *got.ApprovedAt)
	require.Equal(t, "ok", got.ApprovalNote)
	require.Equal(t, "BARU", got.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	input := masterpenolakan.Submission{Input: masterpenolakan.Input{Name: "A", ParentID: "2"}, By: "u", At: submittedAt}
	existingRow := func() *sqlmock.Rows {
		return sqlmock.NewRows(rowColumns).AddRow("1", "A", "1", "P", "0", "u", nil, nil, nil, nil)
	}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		check func(*testing.T, error)
	}{
		{
			name:  "begin",
			setup: func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) },
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") },
		},
		{
			name: "not found",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnRows(sqlmock.NewRows(rowColumns))
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, masterpenolakan.ErrNotFound) },
		},
		{
			name: "read error",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, `membaca "1"`) },
		},
		{
			name: "parent missing",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnRows(existingRow())
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}))
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorIs(t, err, masterpenolakan.ErrParentNotFound) },
		},
		{
			name: "exec error",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnRows(existingRow())
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectExec(q("rejection_update")).WillReturnError(errDB)
				m.ExpectRollback()
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, `memperbarui "1"`) },
		},
		{
			name: "commit error",
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectBegin()
				m.ExpectQuery(q("rejection_get")).WithArgs("1").WillReturnRows(existingRow())
				m.ExpectQuery(q("rejection_parent_get")).WithArgs("2").WillReturnRows(sqlmock.NewRows([]string{"ID_ST", "NOTE_ST"}).AddRow("2", "P"))
				m.ExpectExec(q("rejection_update")).WillReturnResult(sqlmock.NewResult(0, 1))
				m.ExpectCommit().WillReturnError(errDB)
			},
			check: func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi ubah") },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.setup(mock)
			_, err := NewRepo(db).Update(context.Background(), "1", input)
			c.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("rejection_parent_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID_ST"}))
	mock.ExpectQuery(q("rejection_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID_ND"}))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(q("rejection_parent_check_table")).WillReturnError(errDB)
	err := repo.CheckTable(context.Background())
	require.ErrorContains(t, err, "POOLDATA.MST_PENOLAKAN_KLAIM_1 tidak dapat dibaca")

	mock.ExpectQuery(q("rejection_parent_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID_ST"}))
	mock.ExpectQuery(q("rejection_check_table")).WillReturnError(errDB)
	err = repo.CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "POOLDATA.MST_PENOLAKAN_KLAIM_2 tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKomiteListAndGet(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepoKomite(db)

	mock.ExpectQuery(q("committee_rejection_list")).WillReturnRows(
		sqlmock.NewRows([]string{"IDMASTER", "NOTEMASTER"}).AddRow("111", " A ").AddRow("112", "B"))
	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpenolakan.CommitteeRejection{{ID: "111", Note: "A"}, {ID: "112", Note: "B"}}, list)

	mock.ExpectQuery(q("committee_rejection_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "membaca daftar penolakan komite")

	mock.ExpectQuery(q("committee_rejection_list")).WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}).AddRow("1"))
	_, err = repo.List(context.Background())
	require.Error(t, err)

	mock.ExpectQuery(q("committee_rejection_list")).WillReturnRows(
		sqlmock.NewRows([]string{"IDMASTER", "NOTEMASTER"}).AddRow("1", "A").RowError(0, errDB))
	_, err = repo.List(context.Background())
	require.ErrorContains(t, err, "menelusuri penolakan komite")

	mock.ExpectQuery(q("committee_rejection_get")).WithArgs("111").
		WillReturnRows(sqlmock.NewRows([]string{"IDMASTER", "NOTEMASTER"}).AddRow("111", "A"))
	got, err := repo.Get(context.Background(), " 111 ")
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "111", Note: "A"}, got)

	mock.ExpectQuery(q("committee_rejection_get")).WithArgs("9").
		WillReturnRows(sqlmock.NewRows([]string{"IDMASTER", "NOTEMASTER"}))
	_, err = repo.Get(context.Background(), "9")
	require.ErrorIs(t, err, masterpenolakan.ErrKomiteNotFound)

	mock.ExpectQuery(q("committee_rejection_get")).WithArgs("9").WillReturnError(errDB)
	_, err = repo.Get(context.Background(), "9")
	require.ErrorContains(t, err, `membaca penolakan komite "9"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKomiteInsertNew(t *testing.T) {
	// Tabel kosong: nomor pertama 111.
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("committee_rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}))
	mock.ExpectExec(q("committee_rejection_insert")).WithArgs("111", "CATATAN").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	got, err := NewRepoKomite(db).InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "CATATAN"})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "111", Note: "CATATAN"}, got)
	require.NoError(t, mock.ExpectationsWereMet())

	// Tabel berisi: nomor berikutnya dari yang tertinggi.
	db, mock = newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("committee_rejection_list_id_locked")).
		WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}).AddRow("111").AddRow("120"))
	mock.ExpectExec(q("committee_rejection_insert")).WithArgs("121", "X").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	got, err = NewRepoKomite(db).InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "X"})
	require.NoError(t, err)
	require.Equal(t, "121", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKomiteInsertNewFailures(t *testing.T) {
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, "memulai transaksi komite"},
		{"lock", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("committee_rejection_list_id_locked")).WillReturnError(errDB)
			m.ExpectRollback()
		}, "mengunci daftar ID"},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("committee_rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}))
			m.ExpectExec(q("committee_rejection_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, `menyisipkan penolakan komite "111"`},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("committee_rejection_list_id_locked")).WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}))
			m.ExpectExec(q("committee_rejection_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, "menutup transaksi sisip komite"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.setup(mock)
			_, err := NewRepoKomite(db).InsertNew(context.Background(), masterpenolakan.InputKomite{Note: "A"})
			require.ErrorIs(t, err, errDB)
			require.ErrorContains(t, err, c.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestKomiteUpdate(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepoKomite(db)

	mock.ExpectExec(q("committee_rejection_update")).WithArgs("BARU", "111").WillReturnResult(sqlmock.NewResult(0, 1))
	got, err := repo.Update(context.Background(), " 111 ", masterpenolakan.InputKomite{Note: "BARU"})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "111", Note: "BARU"}, got)

	mock.ExpectExec(q("committee_rejection_update")).WithArgs("BARU", "9").WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(context.Background(), "9", masterpenolakan.InputKomite{Note: "BARU"})
	require.ErrorIs(t, err, masterpenolakan.ErrKomiteNotFound)

	mock.ExpectExec(q("committee_rejection_update")).WillReturnError(errDB)
	_, err = repo.Update(context.Background(), "9", masterpenolakan.InputKomite{Note: "BARU"})
	require.ErrorContains(t, err, `memperbarui penolakan komite "9"`)

	// Pengemudi yang tidak dapat menghitung baris terdampak: dianggap berhasil.
	mock.ExpectExec(q("committee_rejection_update")).WithArgs("C", "5").
		WillReturnResult(sqlmock.NewErrorResult(errDB))
	got, err = repo.Update(context.Background(), "5", masterpenolakan.InputKomite{Note: "C"})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "5", Note: "C"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestKomiteCheckTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepoKomite(db)

	mock.ExpectQuery(q("committee_rejection_check_table")).WillReturnRows(sqlmock.NewRows([]string{"IDMASTER"}))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(q("committee_rejection_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTable(context.Background()), "POOLDATA.MST_REJECTED_KOMITE tidak dapat dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t, `masterpenolakan/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { _ = getQuery("tidak_ada") })
}
