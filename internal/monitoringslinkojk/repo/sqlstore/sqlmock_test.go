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

	"claim-pnc/internal/monitoringslinkojk"
)

// Uji di berkas ini menembak adapter lewat go-sqlmock: yang diperiksa adalah kueri BERNAMA
// yang benar-benar dikirim, urutan argumennya, dan pemetaan barisnya.

var errDB = errors.New("basis data mati")

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengembalikan pola regexp yang hanya cocok dengan teks kueri bernama itu.
func q(name string) string { return regexp.QuoteMeta(query(name)) }

func marchFilter() monitoringslinkojk.Filter {
	from := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
	return monitoringslinkojk.Filter{
		BusinessScope: monitoringslinkojk.ScopeSuretyBond,
		DateOfLoss:    &from, DateOfRequestDocument: &to,
		Page: 2, Size: 10,
	}
}

func TestSearchF06SendsFilterArgsAndMapsRows(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	filterArgs := []any{from, from, to, to, scopeExclude, scopeExclude,
		creditInsuranceNamePattern, scopeExclude, creditInsuranceNamePattern}

	mock.ExpectQuery(q("d01_count")).WithArgs(toDriver(filterArgs)...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(11)))
	mock.ExpectQuery(q("d01_rows")).WithArgs(toDriver(append(filterArgs, 10, 10))...).
		WillReturnRows(sqlmock.NewRows([]string{"no_klaim", "CONTRACT_NO", "TANGGAL_MACET", "TUNGGAKAN", "SUKU_BUNGA", "KETERANGAN"}).
			AddRow(" PNCN.26.0207 ", "KTR-1", time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), int64(45000000), 12.5, nil))

	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentF06, marchFilter())
	require.NoError(t, err)
	require.Equal(t, 11, page.Total)
	require.Len(t, page.Rows, 1)

	row := page.Rows[0]
	require.Equal(t, "PNCN.26.0207|KTR-1", row.Get(monitoringslinkojk.RowKeyColumn))
	require.Equal(t, "PNCN.26.0207", row.Get("no_klaim"))
	require.Equal(t, "31/03/2026", row.Get("tanggal_macet"))
	require.Equal(t, "45000000", row.Get("tunggakan"))
	require.Equal(t, "12.5", row.Get("suku_bunga"))
	require.Equal(t, "", row.Get("keterangan"))
	require.NoError(t, mock.ExpectationsWereMet())
}

// toDriver menyamakan bentuk argumen dengan yang dilihat sqlmock.
func toDriver(args []any) []driver.Value {
	out := make([]driver.Value, len(args))
	for i, a := range args {
		if n, ok := a.(int); ok {
			out[i] = int64(n)
			continue
		}
		out[i] = a
	}
	return out
}

func TestSearchD01SendsBranchCodeAndNoBusinessScope(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepoWithBranchCode(db, "007")

	mock.ExpectQuery(q("f06_count")).WithArgs(nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(1)))
	mock.ExpectQuery(q("f06_rows")).WithArgs("007", nil, nil, nil, nil, int64(0), int64(monitoringslinkojk.DefaultPageSize)).
		WillReturnRows(sqlmock.NewRows([]string{"NO_KLAIM", "CONTRACT_NO", "NO_CIF_DEBITUR", "JENIS_KELAMIN", "KODE_KANTOR_CABANG"}).
			AddRow("PNCN.26.0001", "K1", []byte("CIF9"), "P", "007"))

	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01,
		monitoringslinkojk.Filter{BusinessScope: monitoringslinkojk.ScopeCreditInsurance}.Normalize())
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "CIF9", page.Rows[0].Get("nomor_cif_debitur"))
	require.Equal(t, "P", page.Rows[0].Get("jenis_kelamin"))
	require.Equal(t, "007", page.Rows[0].Get("kode_kantor_cabang"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchEmptyOrOutOfRangeSkipsRowsQuery(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)

	// Hitungan NULL dibaca nol.
	mock.ExpectQuery(q("d01_count")).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(nil))
	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentF06, monitoringslinkojk.Filter{}.Normalize())
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.NotNil(t, page.Rows)
	require.Empty(t, page.Rows)

	// Halaman di luar jangkauan: total tetap, tanpa kueri baris.
	mock.ExpectQuery(q("d01_count")).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(int64(3)))
	page, err = repo.Search(context.Background(), monitoringslinkojk.SegmentF06,
		monitoringslinkojk.Filter{Page: 5, Size: 10}.Normalize())
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Empty(t, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchErrors(t *testing.T) {
	ctx := context.Background()
	filter := monitoringslinkojk.Filter{}.Normalize()

	t.Run("unknown segment", func(t *testing.T) {
		db, mock := newMockDB(t)
		_, err := NewRepo(db).Search(ctx, "X99", filter)
		require.ErrorIs(t, err, monitoringslinkojk.ErrUnknownSegment)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("d01_count")).WillReturnError(errDB)
		_, err := NewRepo(db).Search(ctx, monitoringslinkojk.SegmentF06, filter)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "d01_count")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows query", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("d01_count")).WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(int64(1)))
		mock.ExpectQuery(q("d01_rows")).WillReturnError(errDB)
		_, err := NewRepo(db).Search(ctx, monitoringslinkojk.SegmentF06, filter)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "d01_rows")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("d01_count")).WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(int64(1)))
		mock.ExpectQuery(q("d01_rows")).WillReturnRows(sqlmock.NewRows([]string{"NO_KLAIM"}).
			AddRow("A").RowError(0, errDB))
		_, err := NewRepo(db).Search(ctx, monitoringslinkojk.SegmentF06, filter)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "d01_rows")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStreamEmitsAllRowsAndStopsOnEmitError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	cols := []string{"NO_KLAIM", "CONTRACT_NO"}
	mock.ExpectQuery(q("f06_export")).WithArgs(DefaultBranchCode, nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows(cols).AddRow("A", "1").AddRow("B", "2"))

	var got []string
	err := repo.Stream(ctx, monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{},
		func(row monitoringslinkojk.Row) error {
			got = append(got, row.Get(monitoringslinkojk.RowKeyColumn))
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, []string{"A|1", "B|2"}, got)

	// Galat dari emit diteruskan apa adanya.
	errStop := errors.New("batas tercapai")
	mock.ExpectQuery(q("d01_export")).
		WillReturnRows(sqlmock.NewRows(cols).AddRow("A", "1").AddRow("B", "2"))
	calls := 0
	err = repo.Stream(ctx, monitoringslinkojk.SegmentF06, monitoringslinkojk.Filter{},
		func(monitoringslinkojk.Row) error {
			calls++
			return errStop
		})
	require.ErrorIs(t, err, errStop)
	require.Equal(t, 1, calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStreamErrors(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)
	emit := func(monitoringslinkojk.Row) error { return nil }

	require.ErrorIs(t, repo.Stream(context.Background(), "X99", monitoringslinkojk.Filter{}, emit),
		monitoringslinkojk.ErrUnknownSegment)

	mock.ExpectQuery(q("d01_export")).WillReturnError(errDB)
	err := repo.Stream(context.Background(), monitoringslinkojk.SegmentF06, monitoringslinkojk.Filter{}, emit)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "d01_export")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTextFormatsBooleansAndUnknownTypes(t *testing.T) {
	require.Equal(t, "1", text(true))
	require.Equal(t, "0", text(false))
	require.Equal(t, "42", text(int32(42)))
	require.Equal(t, "x", text(customNumber(" x ")))
}

type customNumber string

func (c customNumber) String() string { return string(c) }

// ── Aksi tulis ─────────────────────────────────────────────────────────────────

func TestCountReport(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	mock.ExpectQuery(q("report_count")).WithArgs("PNCN.26.0001").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(2)))
	mock.ExpectQuery(q("report_count")).WithArgs("B").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(nil))
	mock.ExpectQuery(q("report_count")).WithArgs("C").WillReturnError(errDB)

	n, err := repo.CountReport(ctx, "PNCN.26.0001")
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = repo.CountReport(ctx, "B")
	require.NoError(t, err)
	require.Zero(t, n)

	_, err = repo.CountReport(ctx, "C")
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "report_count")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertReportSendsTwentyEightArgsInSQLOrder(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)

	entry := monitoringslinkojk.ReportEntry{
		FacilityAccountNo: "1", DebtorCIF: "2", FacilityTypeCode: "3", FundSource: "4",
		PolicyStart: "5", PolicyEnd: "6", InterestRate: "7", CurrencyCode: "8", Obligation: "9",
		OriginalCurrencyValue: "10", CollectibilityCode: "11", DefaultDate: "12",
		DefaultReasonCode: "13", Arrears: "14", ArrearsDays: "15", ConditionDate: "16",
		ConditionCode: "17", BranchCode: "18", DataOperation: "19", Remark: "20",
		ReportMonth: "21", ClaimID: "22", ContractNo: "23", Recovery: "24", IDCardNo: "25",
		CompanyNPWP: "26", PolicyNo: "27", ClientID: "28",
	}
	args := make([]driver.Value, 0, 28)
	for i := 1; i <= 28; i++ {
		args = append(args, itoa(i))
	}

	mock.ExpectExec(q("report_insert")).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.InsertReport(context.Background(), entry))

	mock.ExpectExec(q("report_insert")).WillReturnError(errDB)
	err := repo.InsertReport(context.Background(), entry)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "report_insert")
	require.NoError(t, mock.ExpectationsWereMet())
}

func itoa(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return digits[i : i+1]
	}
	return itoa(i/10) + digits[i%10:i%10+1]
}

func TestStreamSourceMapsEntries(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(q("source_rows")).
		WithArgs(DefaultBranchCode, nil, nil, nil, nil,
			scopeInclude, scopeInclude, creditInsuranceNamePattern,
			scopeInclude, creditInsuranceNamePattern).
		WillReturnRows(sqlmock.NewRows([]string{
			"CLAIM_ID", "CONTRACT_NO", "NOREK_FASILITAS", "KODE_VALUTA", "JUMLAH_KEWAJIBAN",
			"TANGGAL_MACET", "KODE_KANTOR_CABANG", "NO_POLIS",
		}).
			AddRow("A", "K1", "R1", "IDR", float64(150000000), time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), "001", "P1").
			AddRow("B", "K2", nil, "", "500", nil, "001", nil).
			AddRow("C", "K3", nil, "USD", nil, nil, "001", nil))

	var entries []monitoringslinkojk.ReportEntry
	err := repo.StreamSource(context.Background(),
		monitoringslinkojk.Filter{BusinessScope: monitoringslinkojk.ScopeCreditInsurance},
		func(e monitoringslinkojk.ReportEntry) error {
			entries = append(entries, e)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, entries, 3)

	require.Equal(t, "A", entries[0].ClaimID)
	require.Equal(t, "R1", entries[0].FacilityAccountNo)
	require.Equal(t, "IDR 150000000", entries[0].OriginalCurrencyValue)
	// Tanggal kondisi direplikasi dari tanggal macet.
	require.Equal(t, "04/03/2026", entries[0].DefaultDate)
	require.Equal(t, "04/03/2026", entries[0].ConditionDate)
	require.Equal(t, "P1", entries[0].PolicyNo)
	require.Empty(t, entries[0].ArrearsDays)

	// Tanpa spasi menggantung bila salah satunya kosong.
	require.Equal(t, "500", entries[1].OriginalCurrencyValue)
	require.Equal(t, "USD", entries[2].OriginalCurrencyValue)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStreamSourceErrors(t *testing.T) {
	ctx := context.Background()
	emit := func(monitoringslinkojk.ReportEntry) error { return nil }

	t.Run("query", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("source_rows")).WillReturnError(errDB)
		err := NewRepo(db).StreamSource(ctx, monitoringslinkojk.Filter{}, emit)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "source_rows")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("emit", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("source_rows")).WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("A").AddRow("B"))
		errStop := errors.New("berhenti")
		err := NewRepo(db).StreamSource(ctx, monitoringslinkojk.Filter{},
			func(monitoringslinkojk.ReportEntry) error { return errStop })
		require.ErrorIs(t, err, errStop)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(q("source_rows")).WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("A").RowError(0, errDB))
		err := NewRepo(db).StreamSource(ctx, monitoringslinkojk.Filter{}, emit)
		require.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubmissionQueries(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewRepo(db)
	ctx := context.Background()
	sub := monitoringslinkojk.Submission{ID: 7, ClaimID: "A", ContractNo: "K"}

	mock.ExpectQuery(q("submission_next")).WillReturnRows(sqlmock.NewRows([]string{"CASEDB"}).AddRow(int64(8)))
	mock.ExpectQuery(q("submission_next")).WillReturnRows(sqlmock.NewRows([]string{"CASEDB"}).AddRow(nil))
	mock.ExpectQuery(q("submission_next")).WillReturnError(errDB)
	mock.ExpectExec(q("submission_insert")).WithArgs("A", "K", int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("submission_insert")).WillReturnError(errDB)
	mock.ExpectExec(q("submission_done")).WithArgs("C1", "T1", int64(7), "A").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("submission_done")).WillReturnError(errDB)

	id, err := repo.NextSubmissionID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(8), id)

	// NULL berarti belum ada pengiriman: nomor pertama 1.
	id, err = repo.NextSubmissionID(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	_, err = repo.NextSubmissionID(ctx)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "submission_next")

	require.NoError(t, repo.RecordSubmission(ctx, sub))
	err = repo.RecordSubmission(ctx, sub)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "submission_insert")

	result := monitoringslinkojk.SubmissionResult{ClientID: "C1", TransactionID: "T1"}
	require.NoError(t, repo.CompleteSubmission(ctx, sub, result))
	err = repo.CompleteSubmission(ctx, sub, result)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "submission_done")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ── Pemeriksa ──────────────────────────────────────────────────────────────────

func TestProbeRunsEveryCheckEvenAfterFailure(t *testing.T) {
	db, mock := newMockDB(t)

	names := []string{
		"probe_slik_table", "probe_objectlist_table", "probe_general_table",
		"probe_claim_pnc_table", "probe_adjustment_table", "probe_coverage_table",
		"probe_submission_table",
	}
	for i, name := range names {
		e := mock.ExpectQuery(q(name)).WithArgs(probeSentinel)
		if i == 0 {
			e.WillReturnError(errDB)
			continue
		}
		e.WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(int64(0)))
	}

	results := Probe(context.Background(), db)
	require.Len(t, results, len(names))
	require.False(t, results[0].OK())
	require.ErrorIs(t, results[0].Err, errDB)
	require.Contains(t, results[0].Err.Error(), "probe_slik_table")
	require.Equal(t, "Tabel SLIK OJK (segmen D01) terbaca", results[0].Name)
	for _, r := range results[1:] {
		require.True(t, r.OK(), r.Name)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.Panics(t, func() { _ = query("tidak_ada") })
}
