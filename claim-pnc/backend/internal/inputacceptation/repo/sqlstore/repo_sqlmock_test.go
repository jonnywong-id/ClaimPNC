package sqlstore

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

var findColumns = []string{"CLAIM", "REF", "STATUS", "OPERATOR", "DOC"}

var claimQuery = inputacceptation.Query{
	ClaimID: "CLMNP-1001", Caller: inputacceptation.Caller{Login: "ADMIN"},
}

func TestFindReadsTheDocumentAndLogsItsStatistics(t *testing.T) {
	repo, mock := newMock(t)
	logs := &bytes.Buffer{}
	repo = repo.WithLogger(slog.New(slog.NewJSONHandler(logs, nil)))

	mock.ExpectQuery(exact("find_claim")).WithArgs("CLMNP-1001").
		WillReturnRows(sqlmock.NewRows(findColumns).AddRow(
			"CLMNP-1001", "REF-1", "New", "ADMIN",
			`{"NoClaim":"CLMNP-1001","AdjustmentList":[{"Type":"Final"}]}`))

	detail, err := repo.Find(context.Background(), claimQuery)
	require.NoError(t, err)
	require.Equal(t, "CLMNP-1001", detail.ClaimID)
	require.Equal(t, "REF-1", detail.Reference)
	require.Equal(t, "New", detail.StatusWork)
	require.Equal(t, "ADMIN", detail.LastUpdateOperator)
	require.Equal(t, "CLMNP-1001", detail.Get("claim_no"))
	require.Equal(t, "Final", detail.Rows(inputacceptation.GridAdjustment)[0]["type"])
	require.Contains(t, logs.String(), "dokumen akseptasi klaim treaty non-prop dibaca")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindWithoutADocumentStillOpens(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("find_claim")).
		WillReturnRows(sqlmock.NewRows(findColumns).AddRow("CLMNP-1002", "REF", "New", "ADMIN", nil))
	detail, err := repo.Find(context.Background(), claimQuery)
	require.NoError(t, err)
	require.Equal(t, "CLMNP-1002", detail.ClaimID)
	require.Empty(t, detail.Get("claim_no"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindFailures(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("find_claim")).WillReturnRows(sqlmock.NewRows(findColumns))
	_, err := repo.Find(context.Background(), claimQuery)
	require.ErrorIs(t, err, inputacceptation.ErrNotFound)

	mock.ExpectQuery(exact("find_claim")).WillReturnError(errDB)
	_, err = repo.Find(context.Background(), claimQuery)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca klaim CLMNP-1001")

	mock.ExpectQuery(exact("find_claim")).
		WillReturnRows(sqlmock.NewRows(findColumns).AddRow("C", "R", "S", "O", `{"NoClaim": `))
	_, err = repo.Find(context.Background(), claimQuery)
	require.ErrorContains(t, err, "mengurai dokumen klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveIsAlwaysRefusedAndLogged(t *testing.T) {
	repo, _ := newMock(t)
	cmd := inputacceptation.SaveCommand{Query: claimQuery, Values: map[string]string{"a": "b"}}

	require.ErrorIs(t, repo.Save(context.Background(), cmd), inputacceptation.ErrWriteNotOwned)

	logs := &bytes.Buffer{}
	repo.WithLogger(slog.New(slog.NewJSONHandler(logs, nil)))
	require.ErrorIs(t, repo.Save(context.Background(), cmd), inputacceptation.ErrWriteNotOwned)
	require.Contains(t, logs.String(), "submit akseptasi ditolak karena kepemilikan tabel")
	require.Contains(t, logs.String(), `"isian_diubah":1`)
}

func TestCheckTables(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("check_tables")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	require.NoError(t, repo.CheckTables(context.Background()))

	mock.ExpectQuery(exact("check_tables")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTables(context.Background()), "POOLDATA.JSON_KLAIM")
	require.NoError(t, mock.ExpectationsWereMet())
}
