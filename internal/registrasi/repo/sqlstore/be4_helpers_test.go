package sqlstore

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// be4Boom adalah galat basis data tiruan yang dipakai seluruh uji be4.
var be4Boom = errors.New("ora-tiruan: koneksi putus")

// be4DB membentuk koneksi sqlmock berpencocok regex.
func be4DB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// be4Q mengubah nama kueri menjadi pola regex yang mencocokkan teksnya persis.
func be4Q(name string) string { return "^" + regexp.QuoteMeta(loadQuery(name)) + "$" }

// be4Cols membuat n nama kolom tiruan.
func be4Cols(n int) []string {
	cols := make([]string, n)
	for i := range cols {
		cols[i] = "C" + strings.Repeat("X", i)
	}
	return cols
}

// be4Mode menentukan cara satu langkah kueri dijawab.
type be4Mode int

const (
	be4OK       be4Mode = iota
	be4Fail             // kueri/eksekusi mengembalikan be4Boom
	be4ScanFail         // baris dengan jumlah kolom salah sehingga Scan gagal
	be4RowErr           // baris pertama membawa be4Boom sebagai galat penelusuran
	be4Affected         // Exec berhasil tetapi RowsAffected gagal
)

// be4Step adalah satu pernyataan yang diharapkan berurutan.
type be4Step struct {
	name     string
	exec     bool
	affected int64
	cols     int
	rows     [][]driver.Value
	args     []driver.Value
}

// be4Expect mendaftarkan satu langkah pada mock menurut mode-nya.
func be4Expect(mock sqlmock.Sqlmock, s be4Step, mode be4Mode) {
	if s.exec {
		e := mock.ExpectExec(be4Q(s.name))
		if s.args != nil && mode == be4OK {
			e = e.WithArgs(s.args...)
		}
		switch mode {
		case be4Fail:
			e.WillReturnError(be4Boom)
		case be4Affected:
			e.WillReturnResult(sqlmock.NewErrorResult(be4Boom))
		default:
			e.WillReturnResult(sqlmock.NewResult(0, s.affected))
		}
		return
	}
	q := mock.ExpectQuery(be4Q(s.name))
	if s.args != nil && mode == be4OK {
		q = q.WithArgs(s.args...)
	}
	switch mode {
	case be4Fail:
		q.WillReturnError(be4Boom)
	case be4ScanFail:
		q.WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	case be4RowErr:
		rows := sqlmock.NewRows(be4Cols(s.cols))
		values := make([]driver.Value, s.cols)
		rows.AddRow(values...).RowError(0, be4Boom)
		q.WillReturnRows(rows)
	default:
		rows := sqlmock.NewRows(be4Cols(s.cols))
		for _, r := range s.rows {
			rows.AddRow(r...)
		}
		q.WillReturnRows(rows)
	}
}

// be4ExpectAll mendaftarkan seluruh langkah dalam mode OK.
func be4ExpectAll(mock sqlmock.Sqlmock, steps []be4Step) {
	for _, s := range steps {
		be4Expect(mock, s, be4OK)
	}
}

// be4AnyArgs membuat n argumen sembarang, lalu menimpa posisi tertentu dengan nilai pasti.
func be4AnyArgs(n int, exact map[int]driver.Value) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		if v, ok := exact[i]; ok {
			args[i] = v
			continue
		}
		args[i] = sqlmock.AnyArg()
	}
	return args
}
