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

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
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

var rowColumns = []string{
	"CLAIM", "POLICY", "INSURED", "BUSINESS", "PERIOD", "MONTH", "LOSS", "SOURCE", "REINS",
	"CAUSE", "TSI", "BRANCH", "STATUS", "CHRONO", "REMARK", "SHARE", "DEDUCTIBLE", "ASMVALUE",
	"CLAIMVALUE", "FEE", "LACK", "NET", "NETASM",
}

var lossDay = time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)

func rowValues() []driver.Value {
	return []driver.Value{
		" PNC-1 ", " POL-1 ", " PT A ", " FIRE ", " 2026 ", int64(3), lossDay, " Direct ", "1",
		" Api ", "10000", " Jakarta ", "1", " kronologi ", " catatan ", float64(40), int64(100),
		[]byte("200"), "300", nil, "", int64(400), int64(500),
	}
}

func filter() casestudyclaim.Filter {
	return casestudyclaim.Filter{FromYear: " 2025 ", ToYear: "2026", Limit: 10}
}

func TestListCountsThenReadsOnePage(t *testing.T) {
	repo, mock := newMock(t)
	args := filterArgs(filter().Normalize())
	values := make([]driver.Value, 0, len(args)+2)
	for _, a := range args {
		values = append(values, a)
	}

	mock.ExpectQuery(exact("case_study_count")).WithArgs(values...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(42))
	mock.ExpectQuery(exact("case_study_list")).WithArgs(append(values, 0, 10)...).
		WillReturnRows(sqlmock.NewRows(rowColumns).AddRow(rowValues()...))

	page, err := repo.List(context.Background(), filter())
	require.NoError(t, err)
	require.Equal(t, 42, page.Total)
	require.Len(t, page.Rows, 1)

	row := page.Rows[0]
	require.Equal(t, "PNC-1", row.ClaimNumber)
	require.Equal(t, "POL-1", row.PolicyNumber)
	require.Equal(t, "03", row.ClaimMonth)
	require.Equal(t, "Leader", row.ReinsurerRole)
	require.Equal(t, "Accepted", row.ClaimStatus)
	require.Equal(t, "kronologi", row.Chronology)
	require.Equal(t, "catatan", row.Remark)
	require.Equal(t, lossDay, *row.LossDate)
	require.Equal(t, money.FromMinorUnits(10000), *row.TSI)
	require.Equal(t, money.FromMinorUnits(200), *row.ASMShareValue)
	require.Equal(t, money.FromMinorUnits(300), *row.ClaimValue100)
	require.Nil(t, row.AdjusterFee, "NULL tetap kosong")
	require.Nil(t, row.LackOfDoc, "teks kosong tetap kosong")
	require.Equal(t, int64(40), *row.ASMSharePercent)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailures(t *testing.T) {
	repo, mock := newMock(t)
	count := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"TOTAL"}).AddRow(1) }

	_, err := repo.List(context.Background(), casestudyclaim.Filter{FromYear: "2026", ToYear: "2025"})
	require.ErrorIs(t, err, casestudyclaim.ErrPeriodReversed)

	mock.ExpectQuery(exact("case_study_count")).WillReturnError(errDB)
	_, err = repo.List(context.Background(), filter())
	require.ErrorContains(t, err, "menghitung klaim")

	mock.ExpectQuery(exact("case_study_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("case_study_list")).WillReturnError(errDB)
	_, err = repo.List(context.Background(), filter())
	require.ErrorContains(t, err, "membaca daftar klaim")

	mock.ExpectQuery(exact("case_study_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("case_study_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background(), filter())
	require.ErrorContains(t, err, "membaca baris klaim")

	mock.ExpectQuery(exact("case_study_count")).WillReturnRows(count())
	mock.ExpectQuery(exact("case_study_list")).
		WillReturnRows(sqlmock.NewRows(rowColumns).AddRow(rowValues()...).RowError(0, errDB))
	_, err = repo.List(context.Background(), filter())
	require.ErrorContains(t, err, "menelusuri daftar klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnreadableNumbersNameTheirColumn(t *testing.T) {
	cases := map[int]string{
		5:  "Bulan Klaim",
		10: "TSI (100%)",
		15: "ASM SHARE",
	}
	for index, column := range cases {
		repo, mock := newMock(t)
		values := rowValues()
		values[index] = "bukan angka"

		mock.ExpectQuery(exact("case_study_count")).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
		mock.ExpectQuery(exact("case_study_list")).
			WillReturnRows(sqlmock.NewRows(rowColumns).AddRow(values...))

		_, err := repo.List(context.Background(), filter())
		require.ErrorContains(t, err, column)
	}
}

func TestSaveRemarkTouchesExactlyOneRow(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec(exact("case_study_save_remark")).WithArgs("catatan", "PNC-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	saved, err := repo.SaveRemark(context.Background(), " PNC-1 ", " catatan ")
	require.NoError(t, err)
	require.True(t, saved)

	mock.ExpectExec(exact("case_study_save_remark")).WithArgs(nil, "PNC-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	saved, err = repo.SaveRemark(context.Background(), "PNC-1", "  ")
	require.NoError(t, err)
	require.False(t, saved, "klaim yang tidak ada tidak tersimpan")

	mock.ExpectExec(exact("case_study_save_remark")).WillReturnResult(sqlmock.NewResult(0, 2))
	_, err = repo.SaveRemark(context.Background(), "PNC-1", "x")
	require.ErrorContains(t, err, "CLAIMNO ternyata tidak unik")

	mock.ExpectExec(exact("case_study_save_remark")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = repo.SaveRemark(context.Background(), "PNC-1", "x")
	require.ErrorContains(t, err, "membaca jumlah baris terpengaruh")

	mock.ExpectExec(exact("case_study_save_remark")).
		WillReturnError(errors.New("ORA-12899: value too large for column"))
	_, err = repo.SaveRemark(context.Background(), "PNC-1", "x")
	require.ErrorIs(t, err, casestudyclaim.ErrRemarkRejectedByColumn)

	mock.ExpectExec(exact("case_study_save_remark")).WillReturnError(errDB)
	_, err = repo.SaveRemark(context.Background(), "PNC-1", "x")
	require.ErrorContains(t, err, "menyimpan catatan telaah")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("case_study_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(exact("case_study_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckTable(context.Background()), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

type stringer string

func (s stringer) String() string { return string(s) }

func TestReadIntegerHandlesEveryDriverShape(t *testing.T) {
	number, err := readInteger(7)
	require.NoError(t, err)
	require.Equal(t, int64(7), *number)

	number, err = readInteger(stringer(" 9 "))
	require.NoError(t, err)
	require.Equal(t, int64(9), *number)

	_, err = readInteger(struct{}{})
	require.ErrorContains(t, err, "tidak dikenali")

	for _, bad := range []float64{1.5, 1 << 54} {
		_, err = readInteger(bad)
		require.Error(t, err)
	}
	_, err = readInteger(float64(0) / zero())
	require.ErrorContains(t, err, "bukan bilangan")
}

// zero mengembalikan nol saat berjalan supaya pembagian menghasilkan NaN, bukan galat kompilasi.
func zero() float64 { return 0 }
