package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// newMockRepo membentuk Repo di atas sqlmock yang mencocokkan kueri dengan regex.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exactly mengubah teks kueri bernama menjadi regex yang mencocokkannya persis.
func exactly(name string) string {
	return "^" + regexp.QuoteMeta(query(name)) + "$"
}

func listRows() *sqlmock.Rows {
	return sqlmock.NewRows(listColumns)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestListRunsTheTabQueryWithItsBindsAndMapsTheRows(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("list_cetak_surat")).
		WithArgs(
			inboxrclpucl.WorkClassClaim,
			inboxrclpucl.RCLPUCLWorkbasket,
			inboxrclpucl.WorkStatusCompleted,
			inboxrclpucl.ExpiryStatusActive,
			50, // halaman 2 × 50
			50,
		).
		WillReturnRows(listRows().
			AddRow("REF-1", "PNC-1", "POL-1", "PT Satu", "2025-06-13T14:41:01.532+07:00",
				"catatan", "2", nil, "2026-09-10", "0", "2026-09-01T08:00:00+07:00", 7).
			AddRow(nil, "PNC-2", nil, nil, nil, nil, "9", nil, nil, nil, nil, 7))

	page, err := repo.List(context.Background(), sampleQuery(t, inboxrclpucl.TabCetakSurat),
		inboxrclpucl.Pagination{Page: 2, Size: 50})
	require.NoError(t, err)

	require.Equal(t, 7, page.Total)
	require.Equal(t, inboxrclpucl.Pagination{Page: 2, Size: 50}, page.Pagination)
	require.Len(t, page.Items, 2)

	first := page.Items[0]
	require.Equal(t, "REF-1", first.Reference)
	require.Equal(t, "PNC-1", first.CaseID)
	require.Equal(t, "POL-1", first.PolicyNumber)
	require.Equal(t, "PT Satu", first.InsuredName)
	require.Equal(t, inboxrclpucl.DisplayTimeText("2025-06-13T14:41:01.532+07:00"), first.InboxEntryAt)
	require.Equal(t, "catatan", first.AnalystNote)
	require.Equal(t, inboxrclpucl.TrackPUCL, first.Track)
	require.Empty(t, first.LetterPrintedAt, "NULL dipindai sebagai teks kosong")
	require.Equal(t, inboxrclpucl.DisplayTimeText("2026-09-10"), first.ClaimAge)
	require.Equal(t, "0", first.ExpiryStatus)
	require.Equal(t, inboxrclpucl.DisplayTimeText("2026-09-01T08:00:00+07:00"), first.CreatedAt)

	// Kode jalur yang tidak dikenal menghasilkan teks kosong, meniru `CASE` tanpa `ELSE`.
	require.Empty(t, page.Items[1].Track)
	require.Empty(t, page.Items[1].Reference)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOfTheMSIGTabBindsTheMarker(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("list_klaim_msig")).
		WithArgs(
			inboxrclpucl.WorkClassClaim,
			inboxrclpucl.RCLPUCLWorkbasket,
			inboxrclpucl.WorkStatusCompleted,
			inboxrclpucl.PUCLReturnedToAnalyst,
			inboxrclpucl.MSIGMarker,
			0,
			inboxrclpucl.DefaultPageSize,
		).
		WillReturnRows(listRows())

	page, err := repo.List(context.Background(), sampleQuery(t, inboxrclpucl.TabKlaimMSIG),
		inboxrclpucl.Pagination{})
	require.NoError(t, err)

	// Halaman kosong: senarai kosong (bukan nil) dan Total nol.
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Zero(t, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOfTheKelengkapanDokumenTabRunsItsOwnQuery(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("list_kelengkapan_dokumen")).
		WithArgs(
			inboxrclpucl.WorkClassClaim,
			inboxrclpucl.RCLPUCLWorkbasket,
			inboxrclpucl.WorkStatusCompleted,
			inboxrclpucl.PUCLReturnedToAnalyst,
			0,
			inboxrclpucl.DefaultPageSize,
		).
		WillReturnRows(listRows().
			AddRow("REF", "PNC-9", "", "", "", "", "1", "", "", "", "", 1))

	page, err := repo.List(context.Background(),
		sampleQuery(t, inboxrclpucl.TabKelengkapanDokumen), inboxrclpucl.Pagination{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, inboxrclpucl.TrackRCL, page.Items[0].Track)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRejectsATabWithoutAQueryWithoutTouchingTheDatabase(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, err := repo.List(context.Background(),
		inboxrclpucl.Query{Tab: inboxrclpucl.Tab{Code: "9"}}, inboxrclpucl.Pagination{})
	require.ErrorContains(t, err, `tab "9" belum punya kueri`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsAQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00942")

	mock.ExpectQuery(exactly("list_cetak_surat")).WillReturnError(cause)

	_, err := repo.List(context.Background(), sampleQuery(t, inboxrclpucl.TabCetakSurat),
		inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menjalankan kueri list_cetak_surat")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsAScanFailure(t *testing.T) {
	repo, mock := newMockRepo(t)

	// TOTAL_ROWS bukan bilangan, sehingga pemindaiannya gagal.
	mock.ExpectQuery(exactly("list_cetak_surat")).
		WillReturnRows(listRows().
			AddRow("R", "C", "P", "N", "", "", "", "", "", "", "", "bukan-angka"))

	_, err := repo.List(context.Background(), sampleQuery(t, inboxrclpucl.TabCetakSurat),
		inboxrclpucl.Pagination{})
	require.ErrorContains(t, err, "membaca baris kueri list_cetak_surat")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsARowIterationFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("sambungan putus")

	mock.ExpectQuery(exactly("list_cetak_surat")).
		WillReturnRows(listRows().
			AddRow("R", "C", "P", "N", "", "", "", "", "", "", "", 1).
			RowError(0, cause))

	_, err := repo.List(context.Background(), sampleQuery(t, inboxrclpucl.TabCetakSurat),
		inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menelusuri hasil kueri list_cetak_surat")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// DailyReport
// ---------------------------------------------------------------------------

func TestDailyReportBindsTheRangeOnBothBranchesAndMapsTheRows(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("daily_report")).
		WithArgs(
			inboxrclpucl.WorkClassClaim,
			inboxrclpucl.RCLPUCLWorkbasket,
			"2026-09-01", "2026-09-30",
			inboxrclpucl.WorkClassClaim,
			inboxrclpucl.GroupPanelPA,
			"2026-09-01", "2026-09-30",
			0, inboxrclpucl.DefaultPageSize,
		).
		WillReturnRows(sqlmock.NewRows(reportColumns).
			AddRow("REF-1", "PNC-1", "POL-1", "PT Satu", "2026-09-10T09:30:00+07:00",
				"catatan", nil, "1", "Register", 4))

	rows, total, err := repo.DailyReport(context.Background(),
		inboxrclpucl.DateRange{From: "2026-09-01", To: "2026-09-30"}, inboxrclpucl.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 4, total)
	require.Equal(t, []inboxrclpucl.DailyReportRow{{
		Reference:       "REF-1",
		CaseID:          "PNC-1",
		PolicyNumber:    "POL-1",
		InsuredName:     "PT Satu",
		SentAt:          inboxrclpucl.DisplayTimeText("2026-09-10T09:30:00+07:00"),
		AnalystNote:     "catatan",
		LetterPrintedAt: "",
		Track:           inboxrclpucl.TrackRCL,
		ClaimStatus:     "Register",
	}}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyReportWithoutRowsReturnsAnEmptyList(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("daily_report")).WillReturnRows(sqlmock.NewRows(reportColumns))

	rows, total, err := repo.DailyReport(context.Background(),
		inboxrclpucl.DateRange{From: "2026-09-01", To: "2026-09-01"},
		inboxrclpucl.Pagination{Page: 3, Size: 10})
	require.NoError(t, err)
	require.NotNil(t, rows)
	require.Empty(t, rows)
	require.Zero(t, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyReportWrapsAQueryFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-01843")

	mock.ExpectQuery(exactly("daily_report")).WillReturnError(cause)

	rows, total, err := repo.DailyReport(context.Background(),
		inboxrclpucl.DateRange{From: "a", To: "b"}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menjalankan kueri daily_report")
	require.Nil(t, rows)
	require.Zero(t, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyReportWrapsAScanFailure(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("daily_report")).
		WillReturnRows(sqlmock.NewRows(reportColumns).
			AddRow("R", "C", "P", "N", "", "", "", "", "", "bukan-angka"))

	_, _, err := repo.DailyReport(context.Background(),
		inboxrclpucl.DateRange{From: "2026-09-01", To: "2026-09-30"}, inboxrclpucl.Pagination{})
	require.ErrorContains(t, err, "membaca baris kueri daily_report")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDailyReportWrapsARowIterationFailure(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("sambungan putus")

	mock.ExpectQuery(exactly("daily_report")).
		WillReturnRows(sqlmock.NewRows(reportColumns).
			AddRow("R", "C", "P", "N", "", "", "", "", "", 1).
			RowError(0, cause))

	_, _, err := repo.DailyReport(context.Background(),
		inboxrclpucl.DateRange{From: "2026-09-01", To: "2026-09-30"}, inboxrclpucl.Pagination{})
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "menelusuri hasil kueri daily_report")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Detail
// ---------------------------------------------------------------------------

func TestDetailReadsTheClaimByKeyAndWorkClass(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("detail")).
		WithArgs("ASM-FW-GCNMFW-WORK PNC-1", inboxrclpucl.WorkClassClaim).
		WillReturnRows(sqlmock.NewRows(detailColumns).
			AddRow("ASM-FW-GCNMFW-WORK PNC-1", "PNC-1", "2", "catatan", "POL-1",
				"2026-08-01T00:00:00+07:00", "komentar pucl", "Objek Pertama", "1500000"))

	detail, err := repo.Detail(context.Background(), "ASM-FW-GCNMFW-WORK PNC-1")
	require.NoError(t, err)

	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1", detail.Reference)
	require.Equal(t, "PNC-1", detail.ClaimNumber)
	require.Equal(t, inboxrclpucl.TrackPUCL, detail.Letter.Track)
	require.Equal(t, "2", detail.Letter.TrackCode)
	require.Equal(t, "catatan", detail.Letter.AnalystNote)
	require.Equal(t, "POL-1", detail.Letter.PolicyNumber)
	require.Equal(t, inboxrclpucl.DisplayTimeText("2026-08-01T00:00:00+07:00"), detail.Letter.LossDate)
	// "Nama Peserta" dan "UP" berasal dari satu sumber — nama objek pertama.
	require.Equal(t, "Objek Pertama", detail.Letter.InsuredName)
	require.Equal(t, "Objek Pertama", detail.Letter.SumInsured)
	require.Equal(t, "1500000", detail.Letter.BillAmount)
	require.Equal(t, "komentar pucl", detail.DocumentReceipt.PUCLNote)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailOfAnUnknownKeyIsNotFound(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("detail")).WillReturnRows(sqlmock.NewRows(detailColumns))

	_, err := repo.Detail(context.Background(), "TIDAK-ADA")
	require.Equal(t, inboxrclpucl.ErrClaimNotFound, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailWrapsOtherFailures(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-03113")

	mock.ExpectQuery(exactly("detail")).WillReturnError(cause)

	_, err := repo.Detail(context.Background(), "KUNCI")
	require.ErrorIs(t, err, cause)
	require.False(t, errors.Is(err, inboxrclpucl.ErrClaimNotFound))
	require.ErrorContains(t, err, "membaca layar kerja klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CheckTable
// ---------------------------------------------------------------------------

func expectProbe(mock sqlmock.Sqlmock, name string) {
	mock.ExpectQuery(exactly(name)).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
}

func TestCheckTableRunsTheThreeProbes(t *testing.T) {
	repo, mock := newMockRepo(t)

	expectProbe(mock, "check_rclpucl")
	expectProbe(mock, "check_columns")
	expectProbe(mock, "check_detail")

	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableNamesTheTablesWhenTheFirstProbeFails(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exactly("check_rclpucl")).WillReturnError(sql.ErrConnDone)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.ErrorContains(t, err, "DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableNamesTheColumnsWhenTheColumnProbeFails(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00904")

	expectProbe(mock, "check_rclpucl")
	mock.ExpectQuery(exactly("check_columns")).WillReturnError(cause)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "TANGGALCETAKDOKUMENPUCL_1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableNamesTheChildTablesWhenTheDetailProbeFails(t *testing.T) {
	repo, mock := newMockRepo(t)
	cause := errors.New("ORA-00942")

	expectProbe(mock, "check_rclpucl")
	expectProbe(mock, "check_columns")
	mock.ExpectQuery(exactly("check_detail")).WillReturnError(cause)

	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, cause)
	require.ErrorContains(t, err, "POOLDATA.T_CLAIM_OBJECTLIST")
	require.NoError(t, mock.ExpectationsWereMet())
}
