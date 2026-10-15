package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
)

// be4NewRepo membentuk repo di atas sqlmock dengan pencocok regex.
func be4NewRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// be4Exact mengubah teks kueri menjadi pola regex yang persis.
func be4Exact(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var be4Columns = []string{"WP_ID", "WP_PASAL", "WP_AYAT", "WP_KEJADIAN"}

// TestListWithoutKeywordCountsThenReads memeriksa urutan kueri, argumen, dan pemangkasan.
func TestListWithoutKeywordCountsThenReads(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(30))
	mock.ExpectQuery(be4Exact("clause_list")).
		WithArgs(25, masterpasalai.PageSize).
		WillReturnRows(sqlmock.NewRows(be4Columns).
			AddRow(" 026 ", "1 ", nil, "Banjir  ").
			AddRow("027", "2", "3", "Petir"))

	page, err := repo.List(context.Background(), masterpasalai.Filter{Page: 2})
	require.NoError(t, err)
	require.Equal(t, 30, page.Total)
	require.Equal(t, 2, page.Number)
	require.Equal(t, []masterpasalai.Clause{
		{ID: "026", Number: "1", Paragraph: "", Event: "Banjir"},
		{ID: "027", Number: "2", Paragraph: "3", Event: "Petir"},
	}, page.Clause)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListWithKeywordEscapesPattern memeriksa pola LIKE yang dilolos dan dikirim tiga kali.
func TestListWithKeywordEscapesPattern(t *testing.T) {
	repo, mock := be4NewRepo(t)
	pattern := `%A\%B\_C\\%`
	mock.ExpectQuery(be4Exact("clause_count_search")).
		WithArgs(pattern, pattern, pattern).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery(be4Exact("clause_list_search")).
		WithArgs(pattern, pattern, pattern, 0, masterpasalai.PageSize).
		WillReturnRows(sqlmock.NewRows(be4Columns).AddRow("1", "1", "1", "x"))

	page, err := repo.List(context.Background(), masterpasalai.Filter{Keyword: ` a%b_c\ `, Page: 1})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Len(t, page.Clause, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListBeyondRangeSkipsRead membuktikan halaman di luar jangkauan tidak membaca baris.
func TestListBeyondRangeSkipsRead(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(3))

	page, err := repo.List(context.Background(), masterpasalai.Filter{Page: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Nil(t, page.Clause)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListCountFailure membuktikan galat pencacah dibungkus dengan nama kuerinya.
func TestListCountFailure(t *testing.T) {
	repo, mock := be4NewRepo(t)
	boom := errors.New("ora-00942")
	mock.ExpectQuery(be4Exact("clause_count")).WillReturnError(boom)

	_, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "clause_count")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListQueryFailure membuktikan galat pembacaan daftar dibungkus.
func TestListQueryFailure(t *testing.T) {
	repo, mock := be4NewRepo(t)
	boom := errors.New("putus")
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(5))
	mock.ExpectQuery(be4Exact("clause_list")).WillReturnError(boom)

	_, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListScanFailure membuktikan galat Scan dibungkus "membaca baris".
func TestListScanFailure(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(5))
	mock.ExpectQuery(be4Exact("clause_list")).
		WillReturnRows(sqlmock.NewRows([]string{"WP_ID"}).AddRow("1"))

	_, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestListRowsErrFailure membuktikan galat penelusuran baris dibungkus.
func TestListRowsErrFailure(t *testing.T) {
	repo, mock := be4NewRepo(t)
	boom := errors.New("terputus di tengah")
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(5))
	mock.ExpectQuery(be4Exact("clause_list")).
		WillReturnRows(sqlmock.NewRows(be4Columns).AddRow("1", "1", "1", "x").RowError(0, boom))

	_, err := repo.List(context.Background(), masterpasalai.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menelusuri daftar")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCountAll membaca cacah seluruh baris.
func TestCountAll(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("clause_count")).
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(42))

	total, err := repo.CountAll(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckTable memeriksa jalur sukses dan gagal pemeriksaan tabel.
func TestCheckTable(t *testing.T) {
	repo, mock := be4NewRepo(t)
	mock.ExpectQuery(be4Exact("clause_check_table")).
		WillReturnRows(sqlmock.NewRows(be4Columns))
	require.NoError(t, repo.CheckTable(context.Background()))

	boom := errors.New("tabel tidak ada")
	mock.ExpectQuery(be4Exact("clause_check_table")).WillReturnError(boom)
	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "memeriksa tabel")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestScanRowPassesNoRowsThrough membuktikan sql.ErrNoRows tidak dibungkus.
func TestScanRowPassesNoRowsThrough(t *testing.T) {
	_, err := scanRow(be4ScannerFunc(func(...any) error { return sql.ErrNoRows }))
	require.Equal(t, sql.ErrNoRows, err)
}

type be4ScannerFunc func(target ...any) error

func (f be4ScannerFunc) Scan(target ...any) error { return f(target...) }

// TestGetQueryPanicsOnUnknownName membuktikan nama kueri salah ketik gagal keras.
func TestGetQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`masterpasalai/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { getQuery("tidak_ada") })
}
