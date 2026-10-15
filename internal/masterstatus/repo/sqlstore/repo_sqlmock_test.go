package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var statusColumns = []string{"LSC_ID", "LSC_NOTE", "LSC_ID_OLD"}

func TestList(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("claim_status_list")).WillReturnRows(sqlmock.NewRows(statusColumns).
		AddRow(" 1134 ", " Open ", " 01 ").AddRow("1147", nil, nil))
	got, err := repo.List(ctx)
	require.NoError(t, err)
	require.Equal(t, []masterstatus.ClaimStatus{{Code: "1134", Label: "Open", LegacyCode: "01"}, {Code: "1147"}}, got)

	mock.ExpectQuery(q("claim_status_list")).WillReturnError(errDB)
	_, err = repo.List(ctx)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q("claim_status_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca baris status")
	mock.ExpectQuery(q("claim_status_list")).WillReturnRows(sqlmock.NewRows(statusColumns).AddRow("1", "a", "b").RowError(0, errDB))
	_, err = repo.List(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("claim_status_get")).WithArgs("1134").WillReturnRows(sqlmock.NewRows(statusColumns).AddRow("1134", "Open", nil))
	got, err := repo.Get(ctx, "1134")
	require.NoError(t, err)
	require.Equal(t, masterstatus.ClaimStatus{Code: "1134", Label: "Open"}, got)

	mock.ExpectQuery(q("claim_status_get")).WillReturnRows(sqlmock.NewRows(statusColumns))
	_, err = repo.Get(ctx, "x")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
	mock.ExpectQuery(q("claim_status_get")).WillReturnError(errDB)
	_, err = repo.Get(ctx, "x")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertIssuesCodeFromSiteAndSequence(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("claim_status_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow(" 1 "))
	mock.ExpectQuery(q("claim_status_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(7))
	mock.ExpectExec(q("claim_status_insert")).WithArgs("1007", "Baru").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	got, err := repo.Insert(context.Background(), "Baru")
	require.NoError(t, err)
	require.Equal(t, masterstatus.ClaimStatus{Code: "1007", Label: "Baru"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	ctx := context.Background()
	issued := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(q("claim_status_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("1"))
		m.ExpectQuery(q("claim_status_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  error
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "memulai transaksi"},
		{"site missing", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("claim_status_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}))
			m.ExpectRollback()
		}, nil, "CURRENT_SITE aktif"},
		{"site error", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("claim_status_site")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "kode situs"},
		{"sequence", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("claim_status_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow("1"))
			m.ExpectQuery(q("claim_status_next_sequence")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "nomor urut"},
		{"label taken", func(m sqlmock.Sqlmock) {
			issued(m)
			m.ExpectExec(q("claim_status_insert")).WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.ux_m_sts_claim_label) violated"))
			m.ExpectRollback()
		}, masterstatus.ErrLabelTaken, ""},
		{"code taken", func(m sqlmock.Sqlmock) {
			issued(m)
			m.ExpectExec(q("claim_status_insert")).WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.M_STS_CLAIM_PK) violated"))
			m.ExpectRollback()
		}, masterstatus.ErrCodeTaken, ""},
		{"insert", func(m sqlmock.Sqlmock) {
			issued(m)
			m.ExpectExec(q("claim_status_insert")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan status"},
		{"commit", func(m sqlmock.Sqlmock) {
			issued(m)
			m.ExpectExec(q("claim_status_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menyimpan status baru"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.Insert(ctx, "Baru")
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			if tc.text != "" {
				require.ErrorContains(t, err, tc.text)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectExec(q("claim_status_update")).WithArgs("Baru", "1134").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("claim_status_get")).WithArgs("1134").WillReturnRows(sqlmock.NewRows(statusColumns).AddRow("1134", "Baru", "01"))
	got, err := repo.Update(ctx, "1134", "Baru")
	require.NoError(t, err)
	require.Equal(t, "Baru", got.Label)

	mock.ExpectExec(q("claim_status_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(ctx, "9999", "x")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)

	mock.ExpectExec(q("claim_status_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = repo.Update(ctx, "1134", "x")
	require.ErrorIs(t, err, errDB)

	mock.ExpectExec(q("claim_status_update")).WillReturnError(errors.New("UX_M_STS_CLAIM_LABEL"))
	_, err = repo.Update(ctx, "1134", "x")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("claim_status_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(q("claim_status_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckTable(context.Background()), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}
