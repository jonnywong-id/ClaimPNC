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

	"claim-pnc/internal/dashboardclaim"
)

var (
	claimColumns = []string{
		"CLAIM_ID", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "BUSINESS_NAME",
		"BUSINESS_SOURCE", "BRANCH_NAME", "TECHNICAL_PIC", "ADMIN_PNC", "CLAIM_STATUS",
		"CLAIM_STATUS_LABEL", "PROCESS_STATUS", "LOSS_DATE", "REPORT_DATE", "REGISTERED_AT",
	}
	surveyColumns = []string{
		"SURVEY_ID", "SURVEY_NUMBER", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME",
		"REFERENCE_NUMBER", "SURVEYOR_NAME", "TECHNICAL_PIC", "SURVEY_LOCATION",
		"SURVEY_STATUS", "PROCESS_STATUS", "ADJUSTER_PIC", "SCHEDULED_AT", "ASSIGNED_AT",
	}
)

func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return regexp.QuoteMeta(query(name)) }

// Hitungan klaim berjalan mengirim penyaring dengan urutan bind kueri itu sendiri.
func TestCountOutstandingBindsFilters(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Kotak cari dikirim apa adanya (sudah dipangkas), polanya dibesarkan dan wildcard-nya
	// di-escape; paginasi tidak ikut pada kueri hitung.
	mock.ExpectQuery(exact("outstanding_count")).
		WithArgs("pol_1", `%POL\_1%`, `%POL\_1%`, "PA", "PA", "PA", "PA", "PA",
			// Kedua belas penanda panel penyaring kosong: ujinya hanya mengisi kotak cari.
			nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(12))

	total, err := repo.CountOutstanding(context.Background(),
		dashboardclaim.Filter{Business: dashboardclaim.BusinessPA, Search: " pol_1 ", Limit: 5, Offset: 10})
	require.NoError(t, err)
	require.Equal(t, 12, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountOutstandingError(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("outstanding_count")).
		WithArgs(nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
			nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.CountOutstanding(context.Background(), dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "dashboardclaim/sqlstore: menghitung klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Daftar klaim berjalan memetakan setiap kolom, termasuk NULL menjadi teks kosong dan
// tanggal nil.
func TestListOutstandingMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	loss := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	// Tanggal lapor kini T_CLAIM_PNC.RECEIVEDATE — kolom DATE, bukan teks bergaya Pega.
	report := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	registered := time.Date(2026, time.September, 2, 3, 0, 0, 0, time.UTC)

	// Delapan penyaring lama ditambah dua belas penanda panel — Outstanding sendiri yang
	// memilikinya; kueri survei tetap delapan.
	filters := []driver.Value{nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil}
	mock.ExpectQuery(exact("outstanding_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(30))
	mock.ExpectQuery(exact("outstanding_list")).WithArgs(append(filters, 25, 25)...).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID1", "PNCN.26.0001", "POL", "Tertanggung", "Bisnis", "Sumber", "Cabang",
				"PIC", "Admin", "1147", "Register", "Open", loss, report, registered).
			AddRow("ID2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := repo.ListOutstanding(context.Background(), dashboardclaim.Filter{Offset: 25})
	require.NoError(t, err)
	require.Equal(t, 30, page.Total)
	require.Equal(t, []dashboardclaim.ClaimRow{
		{
			ClaimID: "ID1", ClaimNumber: "PNCN.26.0001", PolicyNumber: "POL", InsuredName: "Tertanggung",
			BusinessName: "Bisnis", BusinessSource: "Sumber", BranchName: "Cabang", TechnicalPIC: "PIC",
			AdminPNC: "Admin", ClaimStatusCode: "1147", ClaimStatusLabel: "Register",
			ProcessStatus: "Open", LossDate: &loss, ReportDate: &report, RegisteredAt: registered,
		},
		{ClaimID: "ID2"},
	}, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOutstandingErrors(t *testing.T) {
	filters := []driver.Value{nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil}
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).WillReturnError(sql.ErrConnDone)
	_, err := repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "menghitung klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).WillReturnError(sql.ErrTxDone)
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrTxDone)
	require.Contains(t, err.Error(), "membaca daftar klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID", "N", "P", "I", "B", "S", "C", "T", "A", "K", "L", "O", "bukan-tanggal", nil, nil))
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris klaim")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID", "N", "P", "I", "B", "S", "C", "T", "A", "K", "L", "O", nil, nil, nil).
			RowError(0, boom))
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menutup daftar klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Hitungan survei memilih kueri menurut jenis surveyor, dengan urutan bind penyaring umum.
func TestCountSurveyPicksQueryPerKind(t *testing.T) {
	cases := map[dashboardclaim.SurveyorType]string{
		dashboardclaim.SurveyorAdjuster: "loss_adjuster_count",
		dashboardclaim.SurveyorInternal: "internal_surveyor_count",
	}
	for kind, name := range cases {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exact(name)).
			WithArgs("TRAVEL", "TRAVEL", "TRAVEL", "TRAVEL", "TRAVEL", nil, nil, nil).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(4))

		total, err := repo.CountSurvey(context.Background(), kind,
			dashboardclaim.Filter{Business: dashboardclaim.BusinessTravel})
		require.NoError(t, err)
		require.Equal(t, 4, total)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestSurveyUnknownKindTouchesNothing(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, err := repo.CountSurvey(context.Background(), "9", dashboardclaim.Filter{})
	require.EqualError(t, err, `dashboardclaim/sqlstore: jenis surveyor "9" tidak dikenal`)
	_, err = repo.ListSurvey(context.Background(), "9", dashboardclaim.Filter{})
	require.EqualError(t, err, `dashboardclaim/sqlstore: jenis surveyor "9" tidak dikenal`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountSurveyError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).WillReturnError(sql.ErrConnDone)

	_, err := repo.CountSurvey(context.Background(), dashboardclaim.SurveyorAdjuster, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), `menghitung survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSurveyMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	scheduled := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
	assigned := time.Date(2026, time.September, 10, 3, 0, 0, 0, time.UTC)

	filters := []driver.Value{"ALL", "ALL", "ALL", "ALL", "ALL", nil, nil, nil}
	mock.ExpectQuery(exact("internal_surveyor_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(2))
	mock.ExpectQuery(exact("internal_surveyor_list")).WithArgs(append(filters, 0, 10)...).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S1", "SRV-1", "PNCN.26.0001", "POL", "Tertanggung", "REF", "Surveyor",
				"PIC", "Jakarta", "Scheduled", "Open", "PIC Adj", scheduled, assigned).
			AddRow("S2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := repo.ListSurvey(context.Background(), dashboardclaim.SurveyorInternal,
		dashboardclaim.Filter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Equal(t, []dashboardclaim.SurveyRow{
		{
			SurveyID: "S1", SurveyNumber: "SRV-1", ClaimNumber: "PNCN.26.0001", PolicyNumber: "POL",
			InsuredName: "Tertanggung", ReferenceNumber: "REF", SurveyorName: "Surveyor",
			TechnicalPIC: "PIC", SurveyLocation: "Jakarta", SurveyStatus: "Scheduled",
			ProcessStatus: "Open", AdjusterPIC: "PIC Adj", ScheduledAt: &scheduled, AssignedAt: assigned,
		},
		{SurveyID: "S2"},
	}, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSurveyErrors(t *testing.T) {
	ctx := context.Background()
	kind := dashboardclaim.SurveyorAdjuster

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).WillReturnError(sql.ErrConnDone)
	_, err := repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).WillReturnError(sql.ErrTxDone)
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrTxDone)
	require.Contains(t, err.Error(), `membaca daftar survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S", "N", "C", "P", "I", "R", "S", "T", "L", "St", "O", "A", "bukan-tanggal", nil))
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris survei")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S", "N", "C", "P", "I", "R", "S", "T", "L", "St", "O", "A", nil, nil).
			RowError(0, boom))
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), `menutup daftar survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Dokumen yang KOSONG bukan galat — klaim yang belum punya baris di JSON_KLAIM tetap terbuka.
func TestFindClaimDetailTreatsMissingDocumentAsEmpty(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"NOMOR_KLAIM", "KLAIM_ID", "STATUS_PROSES", "STATUS_KLAIM",
			"PIC_TEKNIK", "ADMIN_PNC", "DIDAFTARKAN_PADA", "DOKUMEN",
		}).AddRow("PNCN.26.0001", "KUNCI-1", "Open", "1147", "PIC", "Admin", nil, nil))

	detail, err := repo.FindClaimDetail(context.Background(), "KUNCI-1")
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0001", detail.ClaimNumber)
	require.NotNil(t, detail.Document, "peta nil memaksa setiap pemanggil menjaganya")
	require.Empty(t, detail.Document)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Dokumen yang ADA tetapi rusak JUSTRU galat, dan galatnya tidak mengutip isinya.
func TestFindClaimDetailRejectsBrokenDocument(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Nilai di bawah sengaja BUKAN data nasabah: ia hanya perlu gagal diurai.
	const rusak = `{"ClaimData": `

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("KUNCI-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"NOMOR_KLAIM", "KLAIM_ID", "STATUS_PROSES", "STATUS_KLAIM",
			"PIC_TEKNIK", "ADMIN_PNC", "DIDAFTARKAN_PADA", "DOKUMEN",
		}).AddRow("PNCN.26.0002", "KUNCI-2", "Open", "1147", "PIC", "Admin", nil, rusak))

	_, err := repo.FindClaimDetail(context.Background(), "KUNCI-2")
	require.ErrorIs(t, err, dashboardclaim.ErrClaimDetailUnreadable)

	// Isi dokumen TIDAK boleh ikut ke pesan galat: ia memuat data nasabah (`D-69`), dan
	// galat berakhir di log.
	require.NotContains(t, err.Error(), "ClaimData")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Klaim yang tidak ada dijawab ErrClaimNotFound, bukan galat teknis.
func TestFindClaimDetailMissingClaim(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("TIDAK-ADA").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindClaimDetail(context.Background(), "TIDAK-ADA")
	require.ErrorIs(t, err, dashboardclaim.ErrClaimNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
