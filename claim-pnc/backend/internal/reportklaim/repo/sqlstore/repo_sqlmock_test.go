package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

var errDB = errors.New("basis data mati")

func newDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var (
	from = time.Date(2026, 9, 1, 0, 0, 0, 0, clock.ZoneWIB)
	to   = time.Date(2026, 9, 30, 0, 0, 0, 0, clock.ZoneWIB)
)

func toDriver(values []any) []driver.Value {
	result := make([]driver.Value, 0, len(values))
	for _, v := range values {
		result = append(result, v)
	}
	return result
}

func collect(rows *[]reportklaim.Row) func(reportklaim.Row) error {
	return func(row reportklaim.Row) error {
		*rows = append(*rows, row)
		return nil
	}
}

// Setiap rencana kueri dijalankan satu kali pada lini PA: tidak ada yang membaca master
// tambahan, sehingga satu-satunya kueri adalah kueri laporannya sendiri.
func TestStreamRunsEveryPlanWithItsArguments(t *testing.T) {
	codes := make([]string, 0, len(plans))
	for code := range plans {
		codes = append(codes, string(code))
	}
	sort.Strings(codes)

	filter := reportklaim.Filter{From: from, To: to, BusinessLine: reportklaim.BusinessLinePA,
		BusinessCode: "B01", FixedParam: map[string]string{"statusapprove": "1"}}
	for _, raw := range codes {
		code := reportklaim.Code(raw)
		t.Run(raw, func(t *testing.T) {
			db, mock := newDB(t)
			repo := NewRepo(db, nil)
			p := plans[code]
			mock.ExpectQuery(q(p.query(filter))).WithArgs(toDriver(p.args(filter))...).
				WillReturnRows(sqlmock.NewRows([]string{"KOLOM"}).AddRow("isi"))
			var rows []reportklaim.Row
			require.NoError(t, repo.Stream(context.Background(), reportklaim.Report{Code: code}, filter, collect(&rows)))
			require.Len(t, rows, 1)
			require.Equal(t, "isi", rows[0]["KOLOM"])
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStreamArgumentShapes(t *testing.T) {
	filter := reportklaim.Filter{From: from, To: to, BusinessLine: reportklaim.BusinessLinePA, BusinessCode: "B01",
		FixedParam: map[string]string{"statusapprove": "2"}}
	require.Equal(t, []any{from, to, "002"}, dateAndLine(filter))
	require.Equal(t, []any{from, to}, dateOnly(filter))
	require.Equal(t, []any{"002"}, lineOnly(filter))
	require.Nil(t, noArgs(filter))
	require.Equal(t, []any{"B01"}, businessCodeArgs(filter))
	require.Equal(t, []any{from, to, "2"}, komiteNonMBUArgs(filter))
	require.Equal(t, []any{from, to, "true"}, closeNonMBUArgs("true")(filter))
}

func TestStreamCloseNonMBUReadsMastersOnce(t *testing.T) {
	db, mock := newDB(t)
	repo := NewRepo(db, nil)
	filter := reportklaim.Filter{From: from, To: to, BusinessLine: reportklaim.BusinessLineNonMBU}

	mock.ExpectQuery(q("report_fee_scale")).WillReturnRows(
		sqlmock.NewRows([]string{"IDX", "AMOUNT", "FEE"}).AddRow(1, 1000.0, 10.0).AddRow(nil, 0.0, 0.0))
	mock.ExpectQuery(q("report_progress_names")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "Register").AddRow(nil, "x"))
	mock.ExpectQuery(q("report_dominant_factors")).WithArgs(from, to).WillReturnRows(
		sqlmock.NewRows([]string{"KLAIM", "NAMA"}).AddRow("C1", "Banjir"))
	mock.ExpectQuery(q("report_close_klaim_nonmbu")).WithArgs(from, to, "false").WillReturnRows(
		sqlmock.NewRows([]string{"StatusWork"}).AddRow("15/03/2026"))

	var rows []reportklaim.Row
	require.NoError(t, repo.Stream(context.Background(), reportklaim.Report{Code: reportklaim.CodeCloseKlaim}, filter, collect(&rows)))
	require.Len(t, rows, 1)
	require.Equal(t, "03", rows[0]["ClaimNo"], "bulan dua digit diturunkan dari kolom StatusWork")
	require.Equal(t, "2026", rows[0]["CloseClaimNote"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStreamKomiteNonMBUReadsHolidayFromSecondConnection(t *testing.T) {
	db, mock := newDB(t)
	aneka, anekaMock := newDB(t)
	repo := NewRepo(db, aneka)
	filter := reportklaim.Filter{From: from, To: to, BusinessLine: reportklaim.BusinessLineNonMBU,
		FixedParam: map[string]string{"statusapprove": "1"}}

	anekaMock.ExpectQuery(q("report_holiday_calendar")).WithArgs(from, to).WillReturnRows(
		sqlmock.NewRows([]string{"TGL"}).AddRow(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)).AddRow(nil))
	mock.ExpectQuery(q("report_progress_names")).WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}))
	mock.ExpectQuery(q("report_komite_nonmbu")).WithArgs(from, to, "1").WillReturnRows(
		sqlmock.NewRows([]string{"CountryID"}).AddRow("01/02/2026"))

	var rows []reportklaim.Row
	require.NoError(t, repo.Stream(context.Background(), reportklaim.Report{Code: reportklaim.CodeKomite}, filter, collect(&rows)))
	require.Equal(t, "02", rows[0]["AreaClaimId"])
	require.NoError(t, mock.ExpectationsWereMet())
	require.NoError(t, anekaMock.ExpectationsWereMet())
}

func TestStreamFailures(t *testing.T) {
	ctx := context.Background()
	filter := reportklaim.Filter{From: from, To: to, BusinessLine: reportklaim.BusinessLinePA}
	report := reportklaim.Report{Code: reportklaim.CodeKlaimHarian}

	db, mock := newDB(t)
	repo := NewRepo(db, nil)

	err := repo.Stream(ctx, reportklaim.Report{Code: "tidak-ada"}, filter, collect(new([]reportklaim.Row)))
	require.ErrorContains(t, err, "tidak punya kueri")

	mock.ExpectQuery(q("report_klaim_harian")).WillReturnError(errDB)
	err = repo.Stream(ctx, report, filter, collect(new([]reportklaim.Row)))
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menjalankan laporan")

	stop := errors.New("berhenti")
	mock.ExpectQuery(q("report_klaim_harian")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1").AddRow("2"))
	calls := 0
	err = repo.Stream(ctx, report, filter, func(reportklaim.Row) error { calls++; return stop })
	require.ErrorIs(t, err, stop)
	require.Equal(t, 1, calls, "penulisan yang gagal menghentikan pembacaan")

	mock.ExpectQuery(q("report_klaim_harian")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1").RowError(0, errDB))
	err = repo.Stream(ctx, report, filter, collect(new([]reportklaim.Row)))
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBusinessOptions(t *testing.T) {
	ctx := context.Background()
	db, mock := newDB(t)
	repo := NewRepo(db, nil)

	mock.ExpectQuery(q("report_business_options")).WillReturnRows(
		sqlmock.NewRows([]string{"KODE", "NAMA"}).AddRow(" 01 ", " Fire ").AddRow(nil, nil))
	got, err := repo.ListBusinessOptions(ctx)
	require.NoError(t, err)
	require.Equal(t, []reportklaim.BusinessOption{{Code: "01", Name: "Fire"}, {}}, got)

	mock.ExpectQuery(q("report_business_options")).WillReturnError(errDB)
	_, err = repo.ListBusinessOptions(ctx)
	require.ErrorIs(t, err, errDB)
	mock.ExpectQuery(q("report_business_options")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.ListBusinessOptions(ctx)
	require.ErrorContains(t, err, "memindai pilihan bisnis")
	mock.ExpectQuery(q("report_business_options")).WillReturnRows(
		sqlmock.NewRows([]string{"K", "N"}).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.ListBusinessOptions(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTextConvertsDriverValues(t *testing.T) {
	require.Equal(t, "", text(nil))
	require.Equal(t, "abc", text("abc"))
	require.Equal(t, "xyz", text([]byte("xyz")))
	require.Equal(t, "02/09/2026", text(time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)), "tanggal dibaca dalam WIB")
	require.Equal(t, "1", text(true))
	require.Equal(t, "0", text(false))
	require.Equal(t, "42", text(int64(42)))
	require.Equal(t, "1.5", text(1.5))
	require.Equal(t, "7", text(int32(7)))
}

// Ketiga master pelengkap turun ke keadaan "tidak tersedia" — bukan galat — pada setiap
// kegagalan pembacaannya, supaya laporannya tetap dapat diunduh.
func TestMastersFallBackToUnavailable(t *testing.T) {
	ctx := context.Background()

	empty := NewRepo(nil, nil)
	require.False(t, empty.feeScale(ctx).Available())
	require.False(t, empty.progressNames(ctx).Available())
	require.False(t, empty.dominantFactors(ctx, from, to).Available())
	require.False(t, empty.holidayCalendar(ctx, from, to).Available())

	db, mock := newDB(t)
	aneka, anekaMock := newDB(t)
	repo := NewRepo(db, aneka)
	require.False(t, repo.dominantFactors(ctx, time.Time{}, to).Available())
	require.False(t, repo.holidayCalendar(ctx, from, time.Time{}).Available())

	type failing struct {
		name    string
		columns []string
		read    func() bool
		mock    sqlmock.Sqlmock
	}
	cases := []failing{
		{"report_fee_scale", []string{"IDX", "AMOUNT", "FEE"}, func() bool { return repo.feeScale(ctx).Available() }, mock},
		{"report_progress_names", []string{"ID", "NAMA"}, func() bool { return repo.progressNames(ctx).Available() }, mock},
		{"report_dominant_factors", []string{"KLAIM", "NAMA"}, func() bool { return repo.dominantFactors(ctx, from, to).Available() }, mock},
		{"report_holiday_calendar", []string{"TGL"}, func() bool { return repo.holidayCalendar(ctx, from, to).Available() }, anekaMock},
	}
	for _, c := range cases {
		c.mock.ExpectQuery(q(c.name)).WillReturnError(errDB)
		require.False(t, c.read(), c.name+" galat kueri")

		c.mock.ExpectQuery(q(c.name)).WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D"}).AddRow(1, 2, 3, 4))
		require.False(t, c.read(), c.name+" galat pindai")

		values := make([]driver.Value, len(c.columns))
		for i := range values {
			values[i] = nil
		}
		c.mock.ExpectQuery(q(c.name)).WillReturnRows(sqlmock.NewRows(c.columns).AddRow(values...).RowError(0, errDB))
		require.False(t, c.read(), c.name+" galat penelusuran")
	}

	mock.ExpectQuery(q("report_dominant_factors")).WillReturnRows(sqlmock.NewRows([]string{"KLAIM", "NAMA"}).AddRow("C1", "Banjir"))
	factors := repo.dominantFactors(ctx, from, to)
	require.True(t, factors.Available())
	require.Equal(t, 1, factors.Count())
	require.NoError(t, mock.ExpectationsWereMet())
	require.NoError(t, anekaMock.ExpectationsWereMet())
}

func TestCheckSourceReadsHolidayCalendar(t *testing.T) {
	ctx := context.Background()
	db, _ := newDB(t)
	aneka, anekaMock := newDB(t)
	repo := NewRepo(db, aneka)

	anekaMock.ExpectQuery(q("report_holiday_calendar")).WillReturnRows(
		sqlmock.NewRows([]string{"TGL"}).AddRow(time.Now()).AddRow(time.Now()))
	state := repo.CheckSource(ctx)
	require.True(t, state.SecondConnection)
	require.True(t, state.HolidayReadable)
	require.Equal(t, 2, state.HolidayDays)
	require.NoError(t, state.HolidayError)

	anekaMock.ExpectQuery(q("report_holiday_calendar")).WillReturnError(errDB)
	state = repo.CheckSource(ctx)
	require.False(t, state.HolidayReadable)
	require.ErrorIs(t, state.HolidayError, errDB)

	anekaMock.ExpectQuery(q("report_holiday_calendar")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow(1, 2))
	state = repo.CheckSource(ctx)
	require.False(t, state.HolidayReadable)
	require.Error(t, state.HolidayError)

	anekaMock.ExpectQuery(q("report_holiday_calendar")).WillReturnRows(
		sqlmock.NewRows([]string{"TGL"}).AddRow(nil).RowError(0, errDB))
	state = repo.CheckSource(ctx)
	require.False(t, state.HolidayReadable)
	require.ErrorIs(t, state.HolidayError, errDB)
	require.NoError(t, anekaMock.ExpectationsWereMet())
}
