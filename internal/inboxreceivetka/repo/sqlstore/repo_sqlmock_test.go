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

	"claim-pnc/internal/inboxreceivetka"
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

var (
	taskColumns = []string{"REF", "CLAIMKEY", "NOKLAIM", "NOPOLIS", "TERTANGGUNG", "PESERTA", "DOL", "REGISTER"}
	dol         = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	// registered meniru DATE Oracle yang dikembalikan go-ora: berjam dan berzona sesi
	// (WIB). Pemindai wajib menurunkannya menjadi tanggal kalender pada tengah malam UTC.
	registered = time.Date(2026, 9, 1, 0, 0, 0, 0, time.FixedZone("WIB", 7*60*60))
)

func taskRows() *sqlmock.Rows { return sqlmock.NewRows(taskColumns) }

func TestListWithoutKeyword(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("tka_inbox_list")).WithArgs(inboxreceivetka.ResolvedWorkStatus, inboxreceivetka.MaxRows+1).
		WillReturnRows(taskRows().
			AddRow(" R1 ", " K1 ", " PNC-1 ", " POL ", " PT A ", " Peserta ", dol, registered).
			AddRow("R2", nil, "PNC-2", nil, nil, nil, nil, nil).
			AddRow("R3", nil, "PNC-3", nil, nil, nil, nil, nil))
	page, err := repo.List(context.Background(), inboxreceivetka.Filter{})
	require.NoError(t, err)
	require.False(t, page.Truncated)
	require.Len(t, page.Tasks, 3)
	first := page.Tasks[0]
	require.Equal(t, "R1", first.Reference)
	require.Equal(t, "K1", first.ClaimKey)
	require.Equal(t, "PNC-1", first.ClaimNumber)
	require.Equal(t, "Peserta", first.ParticipantName)
	require.Equal(t, dol, *first.DateOfLoss)
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *first.RegisteredOn)
	require.Nil(t, page.Tasks[1].RegisteredOn, "tanggal registrasi kosong tetap kosong")
	require.Nil(t, page.Tasks[1].DateOfLoss)
	require.Nil(t, page.Tasks[2].RegisteredOn)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSearchEscapesKeywordAndTruncates(t *testing.T) {
	repo, mock := newMock(t)
	rows := taskRows()
	for i := 0; i < inboxreceivetka.MaxRows+1; i++ {
		rows.AddRow("R", "K", "PNC", "P", "I", "N", nil, nil)
	}
	pattern := `%PT\_A\%%`
	mock.ExpectQuery(q("tka_inbox_search")).
		WithArgs(inboxreceivetka.ResolvedWorkStatus, pattern, pattern, pattern, pattern, inboxreceivetka.MaxRows+1).
		WillReturnRows(rows)
	page, err := repo.List(context.Background(), inboxreceivetka.Filter{Keyword: " pt_a% "})
	require.NoError(t, err)
	require.True(t, page.Truncated)
	require.Len(t, page.Tasks, inboxreceivetka.MaxRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("tka_inbox_list")).WillReturnError(errDB)
	_, err := repo.List(ctx, inboxreceivetka.Filter{})
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("tka_inbox_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(ctx, inboxreceivetka.Filter{})
	require.ErrorContains(t, err, "membaca baris daftar")

	mock.ExpectQuery(q("tka_inbox_list")).WillReturnRows(taskRows().
		AddRow("R", "K", "N", "P", "I", "N", nil, nil).RowError(0, errDB))
	_, err = repo.List(ctx, inboxreceivetka.Filter{})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func completion() inboxreceivetka.Completion {
	return inboxreceivetka.Completion{ClaimNumber: " PNC-1 ", CompletedAt: time.Date(2026, 9, 24, 15, 30, 0, 0, time.UTC)}
}

func TestCompleteWritesDocumentDate(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("tka_inbox_find_one")).WithArgs(inboxreceivetka.ResolvedWorkStatus, "PNC-1").
		WillReturnRows(taskRows().AddRow("R1", "K1", "PNC-1", "POL", "PT", "Peserta", dol, nil))
	mock.ExpectExec(q("tka_claim_set_document_date")).
		WithArgs(time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), "K1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	task, err := repo.Complete(context.Background(), completion())
	require.NoError(t, err)
	require.Equal(t, "PNC-1", task.ClaimNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteFailures(t *testing.T) {
	ctx := context.Background()
	found := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows().AddRow("R1", "K1", "PNC-1", "", "", "", nil, nil))
	}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  error
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "memulai transaksi"},
		{"find error", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "membaca pekerjaan"},
		{"find scan", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
			m.ExpectRollback()
		}, nil, "membaca baris daftar"},
		{"find rows", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows().
				AddRow("R", "K", "N", "", "", "", nil, nil).RowError(0, errDB))
			m.ExpectRollback()
		}, errDB, "menelusuri pekerjaan"},
		{"not found", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows())
			m.ExpectRollback()
		}, inboxreceivetka.ErrTaskNotFound, ""},
		{"ambiguous", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows().
				AddRow("R1", "K1", "PNC-1", "", "", "", nil, nil).AddRow("R2", "K2", "PNC-1", "", "", "", nil, nil))
			m.ExpectRollback()
		}, inboxreceivetka.ErrClaimAmbiguous, ""},
		{"claim missing", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows().AddRow("R1", nil, "PNC-1", "", "", "", nil, nil))
			m.ExpectRollback()
		}, inboxreceivetka.ErrClaimMissing, ""},
		{"update error", func(m sqlmock.Sqlmock) {
			found(m)
			m.ExpectExec(q("tka_claim_set_document_date")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, `memperbarui klaim "PNC-1"`},
		{"race", func(m sqlmock.Sqlmock) {
			found(m)
			m.ExpectExec(q("tka_claim_set_document_date")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, inboxreceivetka.ErrTaskNotFound, "tidak tersentuh pembaruan"},
		{"many rows", func(m sqlmock.Sqlmock) {
			found(m)
			m.ExpectExec(q("tka_claim_set_document_date")).WillReturnResult(sqlmock.NewResult(0, 2))
			m.ExpectRollback()
		}, inboxreceivetka.ErrClaimAmbiguous, "tersentuh 2 baris"},
		{"commit", func(m sqlmock.Sqlmock) {
			found(m)
			m.ExpectExec(q("tka_claim_set_document_date")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menutup transaksi pengisian"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.Complete(ctx, completion())
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

// RowsAffected yang tidak didukung driver tidak membatalkan pengisian yang sudah tersimpan.
func TestCompleteToleratesUnreadableRowsAffected(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("tka_inbox_find_one")).WillReturnRows(taskRows().AddRow("R1", "K1", "PNC-1", "", "", "", nil, nil))
	mock.ExpectExec(q("tka_claim_set_document_date")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	mock.ExpectCommit()
	_, err := repo.Complete(context.Background(), completion())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChecks(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("tka_inbox_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	require.NoError(t, repo.CheckTable(ctx))
	mock.ExpectQuery(q("tka_inbox_check_table")).WillReturnError(errDB)
	err := repo.CheckTable(ctx)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "daftar Inbox Receive TKA")

	mock.ExpectQuery(q("tka_inbox_check_claim_column")).WillReturnError(errDB)
	err = repo.CheckClaimColumn(ctx)
	require.ErrorContains(t, err, "TGLDOKLENGKAP")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCounts(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	counters := []struct {
		name string
		call func(context.Context) (int, error)
	}{
		{"tka_inbox_count_waiting", repo.CountWaiting},
		{"tka_inbox_count_pega_only", repo.CountPegaOnly},
		{"tka_inbox_count_orphan_claim", repo.CountOrphanClaim},
		{"tka_inbox_count_missing_participant", repo.CountMissingParticipant},
	}
	for i, c := range counters {
		mock.ExpectQuery(q(c.name)).WithArgs(inboxreceivetka.ResolvedWorkStatus).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(i + 3))
		total, err := c.call(ctx)
		require.NoError(t, err)
		require.Equal(t, i+3, total)

		mock.ExpectQuery(q(c.name)).WillReturnError(errDB)
		_, err = c.call(ctx)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, c.name)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSampleRegisteredOn(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("tka_inbox_sample_registered_on")).WillReturnRows(
		sqlmock.NewRows([]string{"V"}).AddRow(registered).AddRow(nil))
	got, err := repo.SampleRegisteredOn(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"20260901", ""}, got)

	mock.ExpectQuery(q("tka_inbox_sample_registered_on")).WillReturnError(errDB)
	_, err = repo.SampleRegisteredOn(ctx)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q("tka_inbox_sample_registered_on")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow(1, 2))
	_, err = repo.SampleRegisteredOn(ctx)
	require.ErrorContains(t, err, "membaca contoh tanggal registrasi")
	mock.ExpectQuery(q("tka_inbox_sample_registered_on")).WillReturnRows(
		sqlmock.NewRows([]string{"V"}).AddRow(registered).RowError(0, errDB))
	_, err = repo.SampleRegisteredOn(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCalendarDate(t *testing.T) {
	require.Nil(t, calendarDate(sql.NullTime{}))
	got := calendarDate(sql.NullTime{Time: time.Date(2026, 12, 31, 23, 30, 0, 0, time.FixedZone("WIB", 7*60*60)), Valid: true})
	require.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), *got,
		"tanggal kalender diambil dari zona sesi, bukan digeser ke UTC lebih dulu")
}
