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

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/platform/clock"
)

// fixedClock adalah jam tetap supaya nomor yang terbit tidak bergantung jam dinding.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// newMock membentuk repo di atas sqlmock dengan pencocokan kueri berbasis regex.
func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	return NewRepo(db, fixedClock{now: now}), mock, db
}

// pattern mengubah teks kueri bernama menjadi regex literal.
func pattern(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// listColumns adalah kedua puluh kolom yang dibaca scanRow, berurutan.
var listColumns = []string{
	"id", "claim_number", "assignment_ref", "policy_number", "insured_name",
	"reporter_name", "business_name", "reference_number", "date_of_loss", "created_at",
	"created_by", "branch_code", "branch_name", "aging_at", "reason", "email_subject",
	"position", "origin", "aging_value", "last_message",
}

// detailColumns adalah ketiga puluh kolom yang dibaca scanDetailRow, berurutan.
var detailColumns = []string{
	"id", "claim_number", "assignment_ref", "policy_number", "insured_name",
	"reporter_name", "business_name", "reference_number", "date_of_loss", "created_at",
	"created_by", "branch_code", "branch_name", "aging_at", "reason", "email_subject",
	"position", "origin", "aging_value", "received_date", "reporter_email",
	"reporter_phone", "courier_name", "estimate_value", "loss_location", "chronology",
	"damage_detail", "not_registered_note", "document_count", "last_message",
}

func toValues(argument []any) []driver.Value {
	result := make([]driver.Value, 0, len(argument))
	for _, a := range argument {
		if n, isInt := a.(int); isInt {
			result = append(result, int64(n))
			continue
		}
		result = append(result, a)
	}
	return result
}

func TestListReportCountsAndMapsRows(t *testing.T) {
	repo, mock, _ := newMock(t)
	filter := inboxlaporanklaim.Filter{
		Category:     inboxlaporanklaim.CategoryOutstanding,
		BranchCode:   " 1001 ",
		BusinessLine: inboxlaporanklaim.BusinessLinePA,
	}
	countArgument := append(scopeArguments(filter), categoryArguments(filter.Category)...)
	listArgument := append(append([]any(nil), countArgument...), 10, 10)

	mock.ExpectQuery(pattern("claim_report_count_body")).
		WithArgs(toValues(countArgument)...).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(12))

	createdAt := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	mock.ExpectQuery(pattern("claim_report_list_body")).
		WithArgs(toValues(listArgument)...).
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow(" RCV-1 ", "PNC-9", "LOCK1", "POL1", "Insured", "Pelapor", "Fire", "REF1",
				nil, createdAt, "OP1", "1001 ", "Jakarta", nil, "alasan", "subj",
				"Outstanding", "pega", int64(4), "pesan").
			AddRow("RCVN.26.1", nil, nil, nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil, nil, nil, nil,
				"Not Transferred", "claimpnc", nil, nil))

	page, err := repo.List(context.Background(), filter, inboxlaporanklaim.Pagination{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 12, page.Total)
	require.Equal(t, inboxlaporanklaim.Pagination{Page: 2, Size: 10}, page.Pagination)
	require.Len(t, page.Report, 2)

	first := page.Report[0]
	require.Equal(t, "RCV-1", first.ID)
	require.Equal(t, "PNC-9", first.ClaimNumber)
	require.Equal(t, "1001", first.BranchCode)
	require.Equal(t, "4", first.AgingValue)
	require.Equal(t, createdAt, first.CreatedAt)
	require.True(t, first.DateOfLoss.IsZero())
	require.True(t, first.Transferred)
	require.Equal(t, inboxlaporanklaim.OriginLegacy, first.Origin)
	require.Equal(t, "pesan", first.LastMessage)

	second := page.Report[1]
	require.Equal(t, "", second.AgingValue, "NULL harus menjadi teks kosong, bukan 0")
	require.False(t, second.Transferred)
	require.Equal(t, inboxlaporanklaim.OriginNew, second.Origin)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListReportCountFailureIsWrapped(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_count_body")).WillReturnError(errors.New("boom"))

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "menghitung baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListReportQueryFailureIsWrapped(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_count_body")).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	mock.ExpectQuery(pattern("claim_report_list_body")).WillReturnError(errors.New("boom"))

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "membaca daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListReportScanFailureIsWrapped(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_count_body")).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	// Satu kolom saja — Scan ke dua puluh tujuan pasti gagal.
	mock.ExpectQuery(pattern("claim_report_list_body")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("RCV-1"))

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "membaca baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListReportRowsErrIsWrapped(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_count_body")).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	rows := sqlmock.NewRows(listColumns).
		AddRow("RCV-1", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil).
		RowError(0, errors.New("putus"))
	mock.ExpectQuery(pattern("claim_report_list_body")).WillReturnRows(rows)

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tab komunikasi tanpa identitas pemanggil menjawab kosong tanpa menyentuh basis data.
func TestListMessageWithoutOperatorReturnsEmptyPage(t *testing.T) {
	repo, mock, _ := newMock(t)

	page, err := repo.List(context.Background(),
		inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryMessageWaiting},
		inboxlaporanklaim.Pagination{Page: 0, Size: 0})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Nil(t, page.Report)
	require.Equal(t, inboxlaporanklaim.Pagination{Page: 1, Size: inboxlaporanklaim.DefaultPageSize}, page.Pagination)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Bind percakapan dikirim dua kali, dan blok SELECT-nya muncul lebih dulu.
func TestListMessageSendsConversationBindsFirst(t *testing.T) {
	repo, mock, _ := newMock(t)
	filter := inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageUnanswered,
		Operator: "OP1",
	}
	message := []any{"OP1", "OP1", "1", "1", nil, nil, "OP1", "OP1"}
	countArgument := append(scopeArguments(filter), message...)
	listArgument := append(append(append([]any(nil), message[2:]...), countArgument...), 0, 10)

	mock.ExpectQuery(pattern("claim_report_message_count_body")).
		WithArgs(toValues(countArgument)...).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(3))
	mock.ExpectQuery(pattern("claim_report_message_body")).
		WithArgs(toValues(listArgument)...).
		WillReturnRows(sqlmock.NewRows(listColumns))

	page, err := repo.List(context.Background(), filter, inboxlaporanklaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Empty(t, page.Report)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMessageFromSelfFiltersOnSenderEqual(t *testing.T) {
	require.Equal(t,
		[]any{"OP1", "OP1", "0", "0", "OP1", "OP1", nil, nil},
		messageArguments(" OP1 ", inboxlaporanklaim.MessageFilter{Status: "0", FromSelf: true}))
}

func TestListMessageCountFailure(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_message_count_body")).WillReturnError(errors.New("boom"))

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageReplied, Operator: "OP1",
	}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "menghitung baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMessageQueryFailure(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_message_count_body")).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	mock.ExpectQuery(pattern("claim_report_message_body")).WillReturnError(errors.New("boom"))

	_, err := repo.List(context.Background(), inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageReplied, Operator: "OP1",
	}, inboxlaporanklaim.Pagination{})
	require.ErrorContains(t, err, "membaca daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryArgumentsPerTab(t *testing.T) {
	cases := map[inboxlaporanklaim.Category][]any{
		inboxlaporanklaim.CategoryOutstanding:    {"1", "Outstanding", "Outstanding", nil, nil},
		inboxlaporanklaim.CategoryUnregistered:   {"1", "Not Registered", "Not Registered", nil, nil},
		inboxlaporanklaim.CategoryNotTransferred: {"1", "Not Transferred", "Not Transferred", nil, nil},
		inboxlaporanklaim.CategoryAccepted:       {"0", "Outstanding", "Outstanding", "1", nil},
		inboxlaporanklaim.CategoryRejected:       {"0", nil, nil, nil, "1"},
		inboxlaporanklaim.CategoryAll:            {"1", nil, nil, nil, nil},
	}
	for category, want := range cases {
		require.Equalf(t, want, categoryArguments(category), "tab %q", category)
	}
}

// Sembilan bind operator dikirim lebih dulu, lalu dua puluh satu bind penyaring.
func TestSummarizeBindsOperatorBeforeScopeAndReadsNulls(t *testing.T) {
	repo, mock, _ := newMock(t)
	filter := inboxlaporanklaim.Filter{Operator: "OP1", RegionCode: "K1"}
	argument := make([]any, 0, 30)
	for i := 0; i < 9; i++ {
		argument = append(argument, "OP1")
	}
	argument = append(argument, scopeArguments(filter)...)

	mock.ExpectQuery(pattern("claim_report_summary_body")).
		WithArgs(toValues(argument)...).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g", "h"}).
			AddRow(int64(10), int64(2), int64(3), nil, int64(4), int64(5), int64(6), int64(7)))

	summary, err := repo.Summarize(context.Background(), filter)
	require.NoError(t, err)
	require.Equal(t, inboxlaporanklaim.Summary{
		Total: 10, NotTransferred: 2, Unregistered: 3, Outstanding: 0, Accepted: 4,
		MessageUnanswered: 5, MessageWaiting: 6, MessageReplied: 7,
	}, summary)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSummarizeScanFailure(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_summary_body")).WillReturnError(errors.New("boom"))

	_, err := repo.Summarize(context.Background(), inboxlaporanklaim.Filter{})
	require.ErrorContains(t, err, "menghitung pencacah")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRegionsSkipsBlankCodes(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_region_list")).
		WillReturnRows(sqlmock.NewRows([]string{"basterritory"}).
			AddRow(" K1 ").AddRow("  ").AddRow(nil).AddRow("K2"))

	region, err := repo.ListRegions(context.Background())
	require.NoError(t, err)
	require.Equal(t, []inboxlaporanklaim.Region{
		{Code: "K1", Name: "K1"}, {Code: "K2", Name: "K2"},
	}, region)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRegionsFailures(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_region_list")).WillReturnError(errors.New("boom"))
		_, err := repo.ListRegions(context.Background())
		require.ErrorContains(t, err, "membaca daftar kanwil")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_region_list")).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("K1", "x"))
		_, err := repo.ListRegions(context.Background())
		require.ErrorContains(t, err, "membaca kanwil")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_region_list")).
			WillReturnRows(sqlmock.NewRows([]string{"basterritory"}).AddRow("K1").
				RowError(0, errors.New("putus")))
		_, err := repo.ListRegions(context.Background())
		require.ErrorContains(t, err, "menelusuri daftar kanwil")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func detailRow(id string) *sqlmock.Rows {
	received := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(detailColumns).AddRow(
		" "+id+" ", nil, nil, "POL1", "Insured", "Pelapor", "Fire", "REF1",
		nil, nil, "OP1", "1001", "Jakarta", nil, "alasan", "subj",
		"Not Transferred", "claimpnc", nil, received, "a@x", "0812", "kurir",
		int64(800000000), "lokasi", "kronologi", "rincian", "catatan", int64(3), "pesan")
}

// Nomor berawalan RCVN dibaca lewat jalur berkas sendiri, bukan jalur warisan.
func TestGetOwnReportUsesOwnQueryAndMapsDetail(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_get_own_body")).
		WithArgs("RCVN.26.5").
		WillReturnRows(detailRow("RCVN.26.5"))

	report, err := repo.Get(context.Background(), " RCVN.26.5 ")
	require.NoError(t, err)
	require.Equal(t, "RCVN.26.5", report.ID)
	require.Equal(t, "a@x", report.ReporterEmail)
	require.Equal(t, inboxlaporanklaim.Money(800000000), report.EstimateValue)
	require.Equal(t, 3, report.DocumentCount)
	require.Equal(t, "catatan", report.NotRegisteredNote)
	require.Equal(t, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), report.ReceivedDate)
	require.False(t, report.Transferred)
	require.Equal(t, "", report.AgingValue)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLegacyReportUsesSourcedQuery(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(sourced("claim_report_get_body"))).
		WithArgs("RCV-1").
		WillReturnRows(sqlmock.NewRows(detailColumns))

	_, err := repo.Get(context.Background(), "RCV-1")
	require.ErrorIs(t, err, inboxlaporanklaim.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tabel milik sendiri yang hilang diterjemahkan; tabel warisan yang hilang tidak.
func TestGetTranslatesMissingTableOnlyForOwnReports(t *testing.T) {
	missing := errors.New("ORA-00942: table or view does not exist")

	repo, mock, _ := newMock(t)
	mock.ExpectQuery(pattern("claim_report_get_own_body")).WillReturnError(missing)
	_, err := repo.Get(context.Background(), "RCVN-1")
	require.ErrorIs(t, err, inboxlaporanklaim.ErrStorageNotReady)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock, _ = newMock(t)
	mock.ExpectQuery(pattern("claim_report_get_body")).WillReturnError(missing)
	_, err = repo.Get(context.Background(), "RCV-1")
	require.Error(t, err)
	require.NotErrorIs(t, err, inboxlaporanklaim.ErrStorageNotReady)
	require.ErrorContains(t, err, "ORA-00942")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRowsErrAndScanFailure(t *testing.T) {
	t.Run("rows", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_get_own_body")).
			WillReturnRows(detailRow("RCVN-1").RowError(0, errors.New("putus")))
		_, err := repo.Get(context.Background(), "RCVN-1")
		require.ErrorContains(t, err, "putus")
		require.NotErrorIs(t, err, inboxlaporanklaim.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_get_own_body")).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("RCVN-1"))
		_, err := repo.Get(context.Background(), "RCVN-1")
		require.ErrorContains(t, err, "membaca baris detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Berkas Pega ditolak sebelum menyentuh basis data.
func TestUpdateRejectsLegacyOriginWithoutTouchingDB(t *testing.T) {
	repo, mock, _ := newMock(t)
	err := repo.Update(context.Background(), inboxlaporanklaim.ClaimReport{
		ID: "RCV-1", Origin: inboxlaporanklaim.OriginLegacy,
	})
	require.ErrorIs(t, err, inboxlaporanklaim.ErrReadOnlyOrigin)
	require.NoError(t, mock.ExpectationsWereMet())
}

func ownReport() inboxlaporanklaim.ClaimReport {
	return inboxlaporanklaim.ClaimReport{
		ID:            "RCVN.26.5",
		Origin:        inboxlaporanklaim.OriginNew,
		ReceivedDate:  time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC),
		DateOfLoss:    time.Date(2026, 8, 31, 18, 0, 0, 0, time.UTC),
		ReporterName:  " Budi ",
		PolicyNumber:  "POL1",
		EstimateValue: inboxlaporanklaim.Money(150000),
		BranchCode:    "1001",
		EmailSubject:  "subj",
	}
}

func TestUpdateWritesReportThenUpdatesClaimRow(t *testing.T) {
	repo, mock, _ := newMock(t)
	report := ownReport()

	mock.ExpectBegin()
	// Tanggal terima dikirim sebagai teks ISO; teks kosong menjadi NULL; estimasi int64.
	mock.ExpectExec(pattern("claim_report_update")).
		WithArgs("2026-09-02", report.DateOfLoss, "Budi", nil, nil, nil, "POL1", nil, nil,
			int64(150000), nil, "subj", nil, nil, nil, nil, "RCVN.26.5").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Tanggal kalender dipotong ke tengah malam WIB sebelum diikat.
	mock.ExpectExec(pattern("claim_report_pnc_update")).
		WithArgs("POL1", nil, clock.DateWIB(report.DateOfLoss), clock.DateWIB(report.ReceivedDate),
			"Budi", nil, nil, nil, "1001", "subj", "RCVN.26.5").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), report))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Baris T_CLAIM_PNC yang belum ada disisipkan pada penyimpanan berikutnya.
func TestUpdateInsertsClaimRowWhenMissing(t *testing.T) {
	repo, mock, _ := newMock(t)
	report := ownReport()
	report.ReceivedDate = time.Time{}
	report.DateOfLoss = time.Time{}

	mock.ExpectBegin()
	mock.ExpectExec(pattern("claim_report_update")).
		WithArgs(nil, nil, "Budi", nil, nil, nil, "POL1", nil, nil,
			int64(150000), nil, "subj", nil, nil, nil, nil, "RCVN.26.5").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(pattern("claim_report_pnc_insert")).
		WithArgs("POL1", nil, nil, nil, "Budi", nil, nil, nil, "1001", "subj", "RCVN.26.5").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), report))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Berkas bernomor bukan RCVN tidak pernah menulis baris T_CLAIM_PNC.
func TestUpdateSkipsClaimRowForForeignNumber(t *testing.T) {
	repo, mock, _ := newMock(t)
	report := ownReport()
	report.ID = "X-1"

	mock.ExpectBegin()
	mock.ExpectExec(pattern("claim_report_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), report))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin().WillReturnError(errors.New("boom"))
		require.ErrorContains(t, repo.Update(context.Background(), ownReport()), "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("missing table", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(pattern("claim_report_update")).
			WillReturnError(errors.New("ORA-00942: table or view does not exist"))
		mock.ExpectRollback()
		require.ErrorIs(t, repo.Update(context.Background(), ownReport()), inboxlaporanklaim.ErrStorageNotReady)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no rows", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(pattern("claim_report_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		require.ErrorIs(t, repo.Update(context.Background(), ownReport()), inboxlaporanklaim.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows affected unsupported", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		// Driver yang tidak dapat melaporkan jumlah baris tidak diartikan kegagalan.
		mock.ExpectExec(pattern("claim_report_update")).
			WillReturnResult(sqlmock.NewErrorResult(errors.New("tidak didukung")))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.Update(context.Background(), ownReport()))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("claim row update", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(pattern("claim_report_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()
		require.ErrorContains(t, repo.Update(context.Background(), ownReport()), "memperbarui baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("claim row insert", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(pattern("claim_report_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(pattern("claim_report_pnc_insert")).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()
		require.ErrorContains(t, repo.Update(context.Background(), ownReport()), "menyisipkan baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(pattern("claim_report_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errors.New("boom"))
		require.ErrorContains(t, repo.Update(context.Background(), ownReport()), "menutup transaksi simpan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newReport() inboxlaporanklaim.ClaimReport {
	return inboxlaporanklaim.ClaimReport{
		CreatedAt:  time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC),
		BranchCode: "1001",
		CreatedBy:  "OP1",
	}
}

func TestInsertIssuesNumberFromCurrentYear(t *testing.T) {
	repo, mock, _ := newMock(t)
	report := newReport()

	mock.ExpectBegin()
	mock.ExpectQuery(pattern("claim_report_next_sequence")).
		WithArgs("26").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
	mock.ExpectExec(pattern("claim_report_insert")).
		WithArgs("RCVN.26.7", report.CreatedAt, "1001", "OP1", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := repo.Insert(context.Background(), report)
	require.NoError(t, err)
	require.Equal(t, "RCVN.26.7", saved.ID)
	require.Equal(t, inboxlaporanklaim.OriginNew, saved.Origin)
	require.Equal(t, inboxlaporanklaim.PositionNotTransferred, saved.Position)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tabrakan nomor diselesaikan dengan mengambil nomor berikutnya.
func TestInsertRetriesOnDuplicateKey(t *testing.T) {
	repo, mock, _ := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(pattern("claim_report_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
	mock.ExpectExec(pattern("claim_report_insert")).
		WillReturnError(errors.New("ORA-00001: unique constraint violated"))
	mock.ExpectRollback()

	mock.ExpectBegin()
	mock.ExpectQuery(pattern("claim_report_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(8)))
	mock.ExpectExec(pattern("claim_report_insert")).
		WithArgs("RCVN.26.8", sqlmock.AnyArg(), "1001", "OP1", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := repo.Insert(context.Background(), newReport())
	require.NoError(t, err)
	require.Equal(t, "RCVN.26.8", saved.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertGivesUpAfterFiveCollisions(t *testing.T) {
	repo, mock, _ := newMock(t)
	for i := 0; i < 5; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery(pattern("claim_report_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
		mock.ExpectExec(pattern("claim_report_insert")).
			WillReturnError(errors.New("ORA-00001: unique constraint violated"))
		mock.ExpectRollback()
	}

	_, err := repo.Insert(context.Background(), newReport())
	require.ErrorContains(t, err, "bertabrakan 5 kali")
	require.ErrorContains(t, err, "ORA-00001")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	t.Run("begin", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin().WillReturnError(errors.New("boom"))
		_, err := repo.Insert(context.Background(), newReport())
		require.ErrorContains(t, err, "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("sequence missing", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(pattern("claim_report_next_sequence")).
			WillReturnError(errors.New("ORA-00942: table or view does not exist"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newReport())
		require.ErrorIs(t, err, inboxlaporanklaim.ErrStorageNotReady)
		require.ErrorContains(t, err, "mengambil nomor urut")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(pattern("claim_report_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
		mock.ExpectExec(pattern("claim_report_insert")).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newReport())
		require.ErrorContains(t, err, `menyisipkan "RCVN.26.7"`)
		require.NotErrorIs(t, err, inboxlaporanklaim.ErrStorageNotReady)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("claim row", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(pattern("claim_report_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
		mock.ExpectExec(pattern("claim_report_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()
		_, err := repo.Insert(context.Background(), newReport())
		require.ErrorContains(t, err, "memperbarui baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(pattern("claim_report_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(7)))
		mock.ExpectExec(pattern("claim_report_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(pattern("claim_report_pnc_update")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errors.New("boom"))
		_, err := repo.Insert(context.Background(), newReport())
		require.ErrorContains(t, err, "menutup transaksi sisip")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTableReadsAllThreeTables(t *testing.T) {
	repo, mock, _ := newMock(t)
	// Ketiga tabel diperiksa dari sebuah map, sehingga urutannya tidak tetap.
	mock.MatchExpectationsInOrder(false)
	for _, name := range []string{
		"claim_report_check_table", "claim_report_check_legacy_table", "claim_report_check_claim_row",
	} {
		mock.ExpectQuery(pattern(name)).WillReturnRows(sqlmock.NewRows([]string{"x"}))
	}

	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableNamesTheUnreadableTable(t *testing.T) {
	repo, mock, _ := newMock(t)
	mock.MatchExpectationsInOrder(false)
	for _, name := range []string{
		"claim_report_check_table", "claim_report_check_legacy_table", "claim_report_check_claim_row",
	} {
		mock.ExpectQuery(pattern(name)).WillReturnError(errors.New("ORA-01031"))
	}

	// Pemeriksaan berhenti pada tabel pertama yang gagal; tabel mana yang pertama
	// bergantung urutan map, sehingga yang diperiksa hanya bentuk pesannya.
	err := repo.CheckTable(context.Background())
	require.ErrorContains(t, err, "tidak dapat dibaca")
	require.ErrorContains(t, err, "ORA-01031")
	require.Regexp(t, `POOLDATA\.T_CLAIM`, err.Error())
}

func TestFindPolicy(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_policy_find")).
			WithArgs("POL1").
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).
				AddRow(" Insured ", "B1", "Fire", "REF", "006", "1"))
		policy, found, err := repo.FindPolicy(context.Background(), "POL1")
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, inboxlaporanklaim.Policy{
			Number: "POL1", InsuredName: "Insured", BusinessCode: "B1", BusinessName: "Fire",
			ReferenceNumber: "REF", GroupPanel: "006", Syariah: true,
		}, policy)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("not syariah when blank", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_policy_find")).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}).
				AddRow("I", nil, nil, nil, nil, nil))
		policy, found, err := repo.FindPolicy(context.Background(), "POL1")
		require.NoError(t, err)
		require.True(t, found)
		require.False(t, policy.Syariah)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("not found", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_policy_find")).
			WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f"}))
		_, found, err := repo.FindPolicy(context.Background(), "POL1")
		require.NoError(t, err)
		require.False(t, found)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("error", func(t *testing.T) {
		repo, mock, _ := newMock(t)
		mock.ExpectQuery(pattern("claim_report_policy_find")).WillReturnError(errors.New("boom"))
		_, found, err := repo.FindPolicy(context.Background(), "POL1")
		require.ErrorContains(t, err, "T_GENERAL")
		require.False(t, found)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newResolver(t *testing.T) (*BranchResolver, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewBranchResolver(db), mock
}

func TestBranchResolverResolve(t *testing.T) {
	t.Run("blank login skips query", func(t *testing.T) {
		resolver, mock := newResolver(t)
		code, resolved, err := resolver.Resolve(context.Background(), "  ")
		require.NoError(t, err)
		require.False(t, resolved)
		require.Empty(t, code)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("found is trimmed", func(t *testing.T) {
		resolver, mock := newResolver(t)
		mock.ExpectQuery(pattern("branch_of_login")).WithArgs("OP1").
			WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow(" 1001 "))
		code, resolved, err := resolver.Resolve(context.Background(), " OP1 ")
		require.NoError(t, err)
		require.True(t, resolved)
		require.Equal(t, "1001", code)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("blank code is unresolved", func(t *testing.T) {
		resolver, mock := newResolver(t)
		mock.ExpectQuery(pattern("branch_of_login")).
			WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow(nil))
		_, resolved, err := resolver.Resolve(context.Background(), "OP1")
		require.NoError(t, err)
		require.False(t, resolved)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("not registered", func(t *testing.T) {
		resolver, mock := newResolver(t)
		mock.ExpectQuery(pattern("branch_of_login")).
			WillReturnRows(sqlmock.NewRows([]string{"code"}))
		_, resolved, err := resolver.Resolve(context.Background(), "OP1")
		require.NoError(t, err)
		require.False(t, resolved)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("timeout", func(t *testing.T) {
		resolver, mock := newResolver(t)
		mock.ExpectQuery(pattern("branch_of_login")).WillReturnError(context.DeadlineExceeded)
		_, resolved, err := resolver.Resolve(context.Background(), "OP1")
		require.ErrorContains(t, err, "tidak dijawab dalam 5s")
		require.False(t, resolved)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("other error", func(t *testing.T) {
		resolver, mock := newResolver(t)
		mock.ExpectQuery(pattern("branch_of_login")).WillReturnError(errors.New("boom"))
		_, _, err := resolver.Resolve(context.Background(), "OP1")
		require.ErrorContains(t, err, `menerjemahkan cabang "OP1"`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBranchResolverCheckTable(t *testing.T) {
	resolver, mock := newResolver(t)
	mock.ExpectQuery(pattern("branch_of_login")).WithArgs("__periksa__").
		WillReturnRows(sqlmock.NewRows([]string{"code"}))
	require.NoError(t, resolver.CheckTable(context.Background()))

	mock.ExpectQuery(pattern("branch_of_login")).WillReturnError(errors.New("ORA-02019"))
	require.ErrorContains(t, resolver.CheckTable(context.Background()), "V_HRD_MST")
	require.NoError(t, mock.ExpectationsWereMet())
}
